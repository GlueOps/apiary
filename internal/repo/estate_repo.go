package repo

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrUnknownProvider   = errors.New("estate: unknown provider")
	ErrScopeKeyUnknown   = errors.New("estate: scope contains a key this provider does not define")
	ErrScopeKeyMissing   = errors.New("estate: scope is missing a key this provider requires")
	ErrScopeNotCanonical = errors.New("estate: scope has a derivable key that was neither supplied nor derived")
	ErrScopeDisagrees    = errors.New("estate: asserted scope disagrees with the value derived from the credential")
	ErrProviderMismatch  = errors.New("estate: provider does not match the bound credential's provider")
	ErrEstateNotFound    = errors.New("estate: not found")
	// ErrEstateTargetTaken is the unique violation on (provider, scope)
	// surfaced as a sentinel. Without it the refusal reaches the handler as
	// an opaque wrapped driver error, falls through to the default arm, and a
	// deliberate "one estate per target" decision is reported as a server
	// fault — on the caller's screen and on the operator's 5xx dashboard.
	ErrEstateTargetTaken = errors.New("estate: another estate already claims this provider and scope")
)

// ProviderScopeSpec declares the shape of one provider's scope object.
//
// It exists because scope is jsonb, so the database cannot reject
// {"acount":"1"} — a typo is a perfectly valid, distinct scope, and it would
// silently create an estate that no request ever resolves to. Postgres cannot
// know the keys; the provider module does.
type ProviderScopeSpec struct {
	Provider string
	// Required keys must be present once canonicalisation has run.
	Required []string
	// Optional keys may be present.
	Optional []string
	// Derivable keys can be read back from the credential itself — an AWS
	// account falls out of sts:GetCallerIdentity. They are still REQUIRED to
	// end up in the stored scope: omitting the account would make two estates
	// in different accounts, same region, collapse to one scope. Derivation
	// moves who supplies the key, not whether it is there.
	Derivable []string
}

func (s ProviderScopeSpec) known() map[string]bool {
	out := map[string]bool{}
	for _, group := range [][]string{s.Required, s.Optional, s.Derivable} {
		for _, k := range group {
			out[k] = true
		}
	}
	return out
}

// ProviderScopeRegistry maps provider name to its scope shape. Provider modules
// register into it; the repo refuses a provider it has never heard of rather
// than storing an unvalidated scope.
type ProviderScopeRegistry map[string]ProviderScopeSpec

// DefaultProviderScopes is a starting point. A real provider module should own
// its own entry, including which keys it can actually derive.
var DefaultProviderScopes = ProviderScopeRegistry{
	"aws": {
		Provider:  "aws",
		Required:  []string{"region"},
		Optional:  []string{"vpc", "service"},
		Derivable: []string{"account"},
	},
	"proxmox": {
		Provider: "proxmox",
		Required: []string{"cluster"},
	},
	"hetzner": {
		Provider: "hetzner",
		Required: []string{"project"},
	},
}

type CreateEstateInput struct {
	Name string

	// AssertedScope is what the operator supplied.
	AssertedScope map[string]string

	// DerivedScope is what the provider read back from the credential itself.
	// Where a key appears in both they must agree — that check is the point: it
	// catches a production credential attached to a staging estate at
	// registration, rather than at the moment it is used against the wrong
	// account.
	DerivedScope map[string]string

	// CredentialID is the credential this estate is bound to, 1:1. It must
	// already exist, and its provider must match.
	CredentialID uuid.UUID
}

type EstateRepo interface {
	Create(ctx context.Context, provider string, in CreateEstateInput) (*tenant.Estate, error)
	Get(ctx context.Context, id uuid.UUID) (*tenant.Estate, error)
	List(ctx context.Context) ([]tenant.Estate, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type EstateRepoGorm struct {
	db     *gorm.DB
	scopes ProviderScopeRegistry
}

var _ EstateRepo = (*EstateRepoGorm)(nil)

func NewEstateRepoGorm(db *gorm.DB, scopes ProviderScopeRegistry) (*EstateRepoGorm, error) {
	if db == nil {
		return nil, errors.New("estate repo: database handle is nil")
	}
	if scopes == nil {
		scopes = DefaultProviderScopes
	}
	return &EstateRepoGorm{db: db, scopes: scopes}, nil
}

// canonicalScope validates the asserted scope against the provider spec and
// merges in derived values, refusing any disagreement.
func (r *EstateRepoGorm) canonicalScope(provider string, asserted, derived map[string]string) (map[string]string, error) {
	spec, ok := r.scopes[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, provider)
	}
	known := spec.known()

	out := map[string]string{}
	for k, v := range asserted {
		if !known[k] {
			// A typo is a valid distinct scope as far as jsonb is concerned, so
			// it would create an estate that nothing ever resolves to.
			return nil, fmt.Errorf("%w: provider %q, key %q", ErrScopeKeyUnknown, provider, k)
		}
		out[k] = v
	}

	for k, v := range derived {
		if !known[k] {
			return nil, fmt.Errorf("%w: provider %q, derived key %q", ErrScopeKeyUnknown, provider, k)
		}
		if got, had := out[k]; had {
			if subtle.ConstantTimeCompare([]byte(got), []byte(v)) != 1 {
				return nil, fmt.Errorf(
					"%w: provider %q asserted %s=%q but the credential resolves to %s=%q",
					ErrScopeDisagrees, provider, k, got, k, v)
			}
			continue
		}
		out[k] = v
	}

	for _, k := range spec.Required {
		if strings.TrimSpace(out[k]) == "" {
			return nil, fmt.Errorf("%w: provider %q requires %q", ErrScopeKeyMissing, provider, k)
		}
	}
	for _, k := range spec.Derivable {
		if strings.TrimSpace(out[k]) == "" {
			// Derivable is not optional. Without it two estates in two
			// different accounts collapse to one scope and collide on insert.
			return nil, fmt.Errorf("%w: provider %q, key %q", ErrScopeNotCanonical, provider, k)
		}
	}
	return out, nil
}

func (r *EstateRepoGorm) Create(ctx context.Context, provider string, in CreateEstateInput) (*tenant.Estate, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))

	scope, err := r.canonicalScope(provider, in.AssertedScope, in.DerivedScope)
	if err != nil {
		return nil, err
	}
	scopeJSON, err := marshalCanonicalScope(scope)
	if err != nil {
		return nil, err
	}

	// Provider is stored on both records because a credential is created before
	// the estate that binds it, so it cannot read the provider from here. That
	// makes disagreement representable, which makes checking it necessary: an
	// estate claiming to be AWS while holding a Proxmox token would fail later,
	// somewhere less obvious than this.
	var cred tenant.Credential
	if err := r.db.WithContext(ctx).First(&cred, "id = ?", in.CredentialID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	if !strings.EqualFold(cred.Provider, provider) {
		return nil, fmt.Errorf("%w: estate is %q, credential is %q", ErrProviderMismatch, provider, cred.Provider)
	}

	est := &tenant.Estate{
		ID:           uuid.New(),
		Name:         strings.TrimSpace(in.Name),
		Provider:     provider,
		Scope:        datatypes.JSON(scopeJSON),
		ScopeVersion: 1,
		CredentialID: in.CredentialID,
	}
	if err := r.db.WithContext(ctx).Omit("Credential").Create(est).Error; err != nil {
		if isUniqueViolationOn(err, "uniq_estates_provider_scope") {
			return nil, fmt.Errorf("%w: %s %v", ErrEstateTargetTaken, provider, scope)
		}
		if isUniqueViolationOn(err, "uniq_estates_credential") {
			return nil, fmt.Errorf("%w: that credential is already bound to another estate", ErrEstateTargetTaken)
		}
		return nil, fmt.Errorf("create estate: %w", err)
	}
	return est, nil
}

func (r *EstateRepoGorm) Get(ctx context.Context, id uuid.UUID) (*tenant.Estate, error) {
	var e tenant.Estate
	err := r.db.WithContext(ctx).Preload("Credential").First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrEstateNotFound
	}
	return &e, err
}

func (r *EstateRepoGorm) List(ctx context.Context) ([]tenant.Estate, error) {
	var out []tenant.Estate
	err := r.db.WithContext(ctx).Order("provider asc, name asc").Find(&out).Error
	return out, err
}

func (r *EstateRepoGorm) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&tenant.Estate{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrEstateNotFound
	}
	return nil
}

// marshalCanonicalScope emits object keys in sorted order. Postgres normalises
// jsonb on storage so this is not required for UNIQUE (provider, scope) to
// work, but it keeps what the application sends identical to what it reads
// back, which makes a diff of two scopes mean something.
func marshalCanonicalScope(m map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make(map[string]string, len(m))
	for _, k := range keys {
		ordered[k] = m[k]
	}
	return json.Marshal(ordered)
}

// WithTx returns a copy bound to tx.
func (r *EstateRepoGorm) WithTx(tx *gorm.DB) *EstateRepoGorm {
	c := *r
	c.db = tx
	return &c
}

// isUniqueViolationOn reports whether err is a Postgres unique violation on the
// named constraint. Matched by constraint name rather than by SQLSTATE alone,
// so two different unique rules on one table produce two different messages.
func isUniqueViolationOn(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
	}
	return false
}
