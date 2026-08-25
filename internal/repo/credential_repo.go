package repo

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCredentialRepoUnconfigured = errors.New("credential repo: not configured")
	ErrIdentityMissing            = errors.New("credential: an identifying component is required")
	ErrSecretMissing              = errors.New("credential: secret material is required")
	ErrCredentialNotFound         = errors.New("credential: not found")
)

// credentialsTable is the AAD domain separator. It is a constant rather than a
// parameter so a caller cannot accidentally seal a credential under another
// table's AAD and have it open on a row it does not belong to.
const credentialsTable = "credentials"

// Secret is credential material. It exists as a distinct type so that the
// fingerprint path cannot reach it: fingerprintIdentity takes a string and this
// is not one, so hashing the secret by mistake is a compile error rather than a
// silent security defect.
//
// String and GoString are overridden because the realistic way a secret escapes
// is not a deliberate log line but %v on a surrounding struct.
type Secret []byte

func (Secret) String() string   { return "[REDACTED]" }
func (Secret) GoString() string { return "[REDACTED]" }

// CredentialPayload separates what may be fingerprinted from what may not.
//
// Identity is the NON-SECRET identifying component — an AWS access key id, a
// Proxmox token id, a role ARN. It is what gets hashed, because ssh fingerprints
// the public half and comparing one is safe precisely because it discloses
// nothing.
//
// Secret is sealed and never hashed. A plain digest of secret material would
// hand anyone holding this database an offline verification oracle: they could
// confirm a guessed credential with no API call, no provider-side audit record,
// no rate limit and no detection.
type CredentialPayload struct {
	Identity string
	Secret   Secret
}

type CreateCredentialInput struct {
	Name string
	Kind string

	Payload CredentialPayload
}

type CredentialRepo interface {
	Create(ctx context.Context, provider string, in CreateCredentialInput) (*tenant.Credential, error)
	Get(ctx context.Context, id uuid.UUID) (*tenant.Credential, error)
	OpenSecret(ctx context.Context, cred *tenant.Credential) (Secret, error)
	List(ctx context.Context) ([]tenant.Credential, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CredentialRepoGorm is tenant-scoped: one instance per organization, because
// the AAD binds every ciphertext to (table, organization, row) and the tenant
// database does not carry an organization column to read it back from.
type CredentialRepoGorm struct {
	db     *gorm.DB
	orgID  uuid.UUID
	keyID  uuid.UUID
	key    []byte
	scopes ProviderScopeRegistry
}

var _ CredentialRepo = (*CredentialRepoGorm)(nil)

// NewCredentialRepoGorm fails closed. Every argument here is load-bearing for
// confidentiality, so a zero value is refused at construction rather than
// producing rows that look sealed and are not.
func NewCredentialRepoGorm(
	db *gorm.DB, orgID, keyID uuid.UUID, key []byte, scopes ProviderScopeRegistry,
) (*CredentialRepoGorm, error) {
	switch {
	case db == nil:
		return nil, fmt.Errorf("%w: database handle is nil", ErrCredentialRepoUnconfigured)
	case orgID == uuid.Nil:
		return nil, fmt.Errorf("%w: organization id is required for AAD binding", ErrCredentialRepoUnconfigured)
	case keyID == uuid.Nil:
		return nil, fmt.Errorf("%w: key id is required so a rotation can tell re-wrapped rows apart", ErrCredentialRepoUnconfigured)
	case len(key) != 32:
		return nil, fmt.Errorf("%w: tenant key must be 32 bytes, got %d", ErrCredentialRepoUnconfigured, len(key))
	}
	if scopes == nil {
		scopes = DefaultProviderScopes
	}
	return &CredentialRepoGorm{db: db, orgID: orgID, keyID: keyID, key: key, scopes: scopes}, nil
}

// fingerprintIdentity hashes the identifying component in ssh's encoding.
//
// It takes a string, never a Secret, and that is the enforcement: passing
// secret material here does not compile. Complete digest, never truncated — a
// comparison is only as strong as what is compared, and for an assumed role the
// identifying component is an ARN whose name the customer chooses, so a short
// prefix is grindable.
func fingerprintIdentity(identity string) string {
	sum := sha256.Sum256([]byte(identity))
	return "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(sum[:]), "=")
}

func (r *CredentialRepoGorm) Create(ctx context.Context, provider string, in CreateCredentialInput) (*tenant.Credential, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))

	if strings.TrimSpace(in.Payload.Identity) == "" {
		return nil, ErrIdentityMissing
	}
	if len(in.Payload.Secret) == 0 {
		return nil, ErrSecretMissing
	}

	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = tenant.CredentialKindStaticSecret
	}
	if kind != tenant.CredentialKindStaticSecret && kind != tenant.CredentialKindAssumedRole {
		return nil, fmt.Errorf("credential: unknown kind %q", kind)
	}

	// The id is generated here, not by the column default, because the AAD binds
	// the ciphertext to this row and the row must therefore be named before it
	// is sealed. Inserting first and updating after would leave a window in
	// which the row exists with no ciphertext.
	id := uuid.New()

	sealed, err := utils.Seal(
		in.Payload.Secret,
		r.keyID,
		r.key,
		utils.AAD(credentialsTable, r.orgID, id),
	)
	if err != nil {
		return nil, fmt.Errorf("seal credential: %w", err)
	}

	cred := &tenant.Credential{
		ID:          id,
		Name:        strings.TrimSpace(in.Name),
		Provider:    provider,
		Kind:        kind,
		Identity:    strings.TrimSpace(in.Payload.Identity),
		Fingerprint: fingerprintIdentity(in.Payload.Identity),
		Sealed:      sealed,
	}
	if err := r.db.WithContext(ctx).Create(cred).Error; err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}
	return cred, nil
}

func (r *CredentialRepoGorm) Get(ctx context.Context, id uuid.UUID) (*tenant.Credential, error) {
	var c tenant.Credential
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCredentialNotFound
	}
	return &c, err
}

// OpenSecret decrypts under the AAD for this exact row. A ciphertext moved onto
// another row fails here rather than opening cleanly, which is the property the
// AAD exists for.
func (r *CredentialRepoGorm) OpenSecret(ctx context.Context, cred *tenant.Credential) (Secret, error) {
	if cred == nil {
		return nil, ErrCredentialNotFound
	}
	plain, err := cred.Open(r.key, utils.AAD(credentialsTable, r.orgID, cred.ID))
	if err != nil {
		return nil, fmt.Errorf("open credential %s: %w", cred.ID, err)
	}
	return Secret(plain), nil
}

func (r *CredentialRepoGorm) List(ctx context.Context) ([]tenant.Credential, error) {
	var out []tenant.Credential
	err := r.db.WithContext(ctx).Order("provider asc, name asc").Find(&out).Error
	return out, err
}

func (r *CredentialRepoGorm) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&tenant.Credential{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

// WithTx returns a copy bound to tx. The AAD inputs (organization, key) travel
// with it, so a credential written inside a transaction is sealed exactly as it
// would be outside one.
func (r *CredentialRepoGorm) WithTx(tx *gorm.DB) *CredentialRepoGorm {
	c := *r
	c.db = tx
	return &c
}

// Rotate replaces the sealed material and the fingerprint on an existing row,
// keeping its identity and any binding to it.
//
// It re-seals under the AAD for THIS row, so a rotation cannot be used to smuggle
// material from another row in: the ciphertext written here only ever opens here.
func (r *CredentialRepoGorm) Rotate(ctx context.Context, id uuid.UUID, payload CredentialPayload) (*tenant.Credential, error) {
	if strings.TrimSpace(payload.Identity) == "" {
		return nil, ErrIdentityMissing
	}
	if len(payload.Secret) == 0 {
		return nil, ErrSecretMissing
	}

	cred, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	sealed, err := utils.Seal(payload.Secret, r.keyID, r.key, utils.AAD(credentialsTable, r.orgID, cred.ID))
	if err != nil {
		return nil, fmt.Errorf("seal rotated credential: %w", err)
	}

	cred.Sealed = sealed
	cred.Identity = strings.TrimSpace(payload.Identity)
	cred.Fingerprint = fingerprintIdentity(payload.Identity)

	// Identity moves in the SAME statement as the ciphertext it identifies. A
	// rotation that replaced the secret and left the old token id behind would
	// produce a credential whose two halves belong to different credentials —
	// authenticating as nothing, with a fingerprint that says it rotated fine.
	if err := r.db.WithContext(ctx).Model(&tenant.Credential{}).
		Where("id = ?", cred.ID).
		Updates(map[string]any{
			"key_id":      sealed.KeyID,
			"ciphertext":  sealed.Ciphertext,
			"iv":          sealed.IV,
			"tag":         sealed.Tag,
			"identity":    cred.Identity,
			"fingerprint": cred.Fingerprint,
		}).Error; err != nil {
		return nil, fmt.Errorf("rotate credential: %w", err)
	}
	return cred, nil
}
