package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrTenantUnavailable   = errors.New("estate: this organization's tenant database is not available")
	ErrEstateHasCredential = errors.New("estate: a credential is still bound to this estate")
)

// TenantResolver is the one thing this service needs from the tenant manager:
// given an organization, hand back that organization's database handle and its
// unwrapped tenant key.
//
// It is declared here, on the CONSUMER side, deliberately. Depending on the
// concrete *database.TenantManager meant the service could not be reached
// without a provisioned tenant — a control-plane organization row carrying an
// encrypted DSN and a wrapped key — so the layer that keeps estate and
// credential consistent had no test at all, and the tests it could have had
// would have re-tested the tenant manager instead. *database.TenantManager
// satisfies this verbatim, so no caller changes.
type TenantResolver interface {
	GetTenantContext(ctx context.Context, orgID string) (*database.TenantContext, error)
}

// EstateService owns everything a request needs in order to touch an estate or
// a credential: resolving the caller's tenant database, deriving the key that
// seals credential material, and keeping the two records consistent with each
// other.
//
// It exists because the repositories are deliberately narrow. They enforce the
// rules that must hold for a single row — AAD binding, scope canonicalisation,
// fingerprint over the identifying component — and know nothing about which
// tenant is asking or where the key came from. That is this layer's job.
type EstateService struct {
	tenants   TenantResolver
	masterKey []byte
	scopes    repo.ProviderScopeRegistry
}

// isNilResolver catches a TYPED nil — a (*database.TenantManager)(nil) stored in
// the interface — which `tenants == nil` does not, because the interface value
// carries a type and is therefore non-nil. Without this the constructor would
// happily return a service that panics on the first request, inside
// TenantManager.GetTenantContext's mutex, rather than refusing at startup.
func isNilResolver(t TenantResolver) bool {
	if t == nil {
		return true
	}
	v := reflect.ValueOf(t)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

func NewEstateService(tenants TenantResolver, masterKey []byte, scopes repo.ProviderScopeRegistry) (*EstateService, error) {
	if isNilResolver(tenants) {
		return nil, errors.New("estate service: tenant manager is required")
	}
	if len(masterKey) != utils.MasterKeyLen {
		return nil, fmt.Errorf("estate service: master key must be %d bytes, got %d", utils.MasterKeyLen, len(masterKey))
	}
	if scopes == nil {
		scopes = repo.DefaultProviderScopes
	}
	return &EstateService{tenants: tenants, masterKey: masterKey, scopes: scopes}, nil
}

// bind resolves the tenant database and builds repositories bound to it.
//
// The tenant key is DERIVED per purpose rather than being the master key
// itself. Using one key for every purpose means a compromise anywhere is a
// compromise everywhere, and it makes rotating for one reason force
// re-encryption of everything.
func (s *EstateService) bind(ctx context.Context, orgID uuid.UUID) (*repo.CredentialRepoGorm, *repo.EstateRepoGorm, *gorm.DB, error) {
	if s == nil {
		return nil, nil, nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}

	dek, err := utils.DeriveKey(tCtx.Key, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("derive tenant data key: %w", err)
	}

	creds, err := repo.NewCredentialRepoGorm(tCtx.DB, orgID, utils.LegacyKeyID, dek, s.scopes)
	if err != nil {
		return nil, nil, nil, err
	}
	estates, err := repo.NewEstateRepoGorm(tCtx.DB, s.scopes)
	if err != nil {
		return nil, nil, nil, err
	}
	return creds, estates, tCtx.DB, nil
}

// EstateView is what leaves this service. It never carries sealed material:
// nothing above this layer has a reason to see a credential's plaintext, and a
// field that does not exist cannot be serialised into a response by accident.
type EstateView struct {
	ID             uuid.UUID
	Name           string
	Provider       string
	Scope          map[string]string
	ScopeVersion   int
	CredentialID   uuid.UUID
	CredentialName string
	CredentialKind string
	Fingerprint    string
}

func viewOf(e tenant.Estate, c *tenant.Credential) EstateView {
	v := EstateView{
		ID:           e.ID,
		Name:         e.Name,
		Provider:     e.Provider,
		Scope:        decodeScope(e.Scope),
		ScopeVersion: e.ScopeVersion,
		CredentialID: e.CredentialID,
	}
	if c != nil {
		v.CredentialName = c.Name
		v.CredentialKind = c.Kind
		v.Fingerprint = c.Fingerprint
	}
	return v
}

type RegisterEstateInput struct {
	OrgID uuid.UUID

	Name     string
	Provider string
	Scope    map[string]string

	CredentialName string
	CredentialKind string

	// Identity is the NON-SECRET identifying component — an access key id, a
	// token id, a role ARN. It is what gets fingerprinted.
	Identity string
	// Secret is sealed and never fingerprinted or logged.
	Secret repo.Secret
}

// Register creates the credential and the estate that binds it, in one
// transaction.
//
// Order is forced: Estate.CredentialID is NOT NULL, so the credential must
// exist first. That makes a partial failure possible — a credential with no
// estate is an orphaned secret nobody can see or reach — which is exactly why
// this runs in a transaction rather than as two calls from a handler.
func (s *EstateService) Register(ctx context.Context, in RegisterEstateInput) (*EstateView, error) {
	creds, estates, db, err := s.bind(ctx, in.OrgID)
	if err != nil {
		return nil, err
	}

	provider := strings.ToLower(strings.TrimSpace(in.Provider))

	var out EstateView
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCreds := creds.WithTx(tx)
		txEstates := estates.WithTx(tx)

		cred, err := txCreds.Create(ctx, provider, repo.CreateCredentialInput{
			Name: strings.TrimSpace(in.CredentialName),
			Kind: in.CredentialKind,
			Payload: repo.CredentialPayload{
				Identity: in.Identity,
				Secret:   in.Secret,
			},
		})
		if err != nil {
			return err
		}

		// DerivedScope is empty for now: reading an account back from a
		// credential means calling the provider, and no provider module exists
		// yet. The repo still refuses a scope missing a derivable key, so an
		// operator must supply the account explicitly until derivation lands —
		// which fails closed rather than storing an uncanonical scope.
		est, err := txEstates.Create(ctx, provider, repo.CreateEstateInput{
			Name:          strings.TrimSpace(in.Name),
			AssertedScope: in.Scope,
			CredentialID:  cred.ID,
		})
		if err != nil {
			return err
		}

		out = viewOf(*est, cred)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *EstateService) List(ctx context.Context, orgID uuid.UUID) ([]EstateView, error) {
	creds, estates, _, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}

	list, err := estates.List(ctx)
	if err != nil {
		return nil, err
	}

	byID := map[uuid.UUID]tenant.Credential{}
	all, err := creds.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range all {
		byID[c.ID] = c
	}

	out := make([]EstateView, 0, len(list))
	for _, e := range list {
		c := byID[e.CredentialID]
		out = append(out, viewOf(e, &c))
	}
	return out, nil
}

func (s *EstateService) Get(ctx context.Context, orgID, id uuid.UUID) (*EstateView, error) {
	_, estates, _, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}
	e, err := estates.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	v := viewOf(*e, &e.Credential)
	return &v, nil
}

// Delete removes the estate and the credential it was bound to.
//
// Both, deliberately. The binding is 1:1, so deleting only the estate would
// leave a sealed credential nothing references — unreachable through the API,
// invisible in any listing, and still a live secret for whatever provider it
// belongs to. The foreign key is ON DELETE RESTRICT, so the order is forced.
func (s *EstateService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	creds, estates, db, err := s.bind(ctx, orgID)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCreds := creds.WithTx(tx)
		txEstates := estates.WithTx(tx)

		est, err := txEstates.Get(ctx, id)
		if err != nil {
			return err
		}
		if err := txEstates.Delete(ctx, id); err != nil {
			return err
		}
		return txCreds.Delete(ctx, est.CredentialID)
	})
}

// RotateCredential replaces the sealed material on an estate's credential in
// place, keeping the binding and the estate untouched.
//
// In place rather than create-and-swap because the binding is 1:1: a second
// credential for one estate is unrepresentable, so a rotation that created one
// could not be committed. The fingerprint changes with the material, which is
// how an operator confirms the rotation actually took.
func (s *EstateService) RotateCredential(ctx context.Context, orgID, estateID uuid.UUID, identity string, secret repo.Secret) (*EstateView, error) {
	creds, estates, db, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var out EstateView
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCreds := creds.WithTx(tx)
		txEstates := estates.WithTx(tx)

		est, err := txEstates.Get(ctx, estateID)
		if err != nil {
			return err
		}
		cred, err := txCreds.Rotate(ctx, est.CredentialID, repo.CredentialPayload{
			Identity: identity,
			Secret:   secret,
		})
		if err != nil {
			return err
		}
		out = viewOf(*est, cred)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func decodeScope(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}
