package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrSSHKeyName     = errors.New("ssh key: a name is required")
	ErrSSHKeyNotFound = errors.New("ssh key: no such key")

	// ErrNoUsableFleetKey is returned when nothing can reach the fleet. It is a
	// distinct error because the remedy is specific — mint one and whitelist it —
	// and reporting it as a generic failure sends an operator looking at hosts.
	ErrNoUsableFleetKey = errors.New("ssh key: this organization has no usable fleet key")
)

// SSHKeyService owns the credential the control plane uses to reach hypervisors.
//
// The private half never leaves this process. It is sealed with a key derived
// per-tenant, and the only method that unseals it is PrivateKey, which exists
// for River workers — trusted by design, running with the same environment as
// the API — and for nothing else. No handler returns it, and no response schema
// has a field it could occupy.
type SSHKeyService struct {
	tenants TenantResolver
}

func NewSSHKeyService(tenants TenantResolver) *SSHKeyService {
	return &SSHKeyService{tenants: tenants}
}

// bind resolves the tenant's database and the key that seals private halves.
//
// Every method goes through here rather than holding a handle, so there is no
// path on which one organization's keys are sealed with another's material.
func (s *SSHKeyService) bind(ctx context.Context, orgID uuid.UUID) (*repo.SSHKeyRepoGorm, error) {
	if s == nil || isNilResolver(s.tenants) || orgID == uuid.Nil {
		// A check that cannot reach a decision must deny.
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}
	dek, err := utils.DeriveKey(tCtx.Key, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, fmt.Errorf("derive tenant data key: %w", err)
	}
	return repo.NewSSHKeyRepoGorm(tCtx.DB, orgID, utils.LegacyKeyID, dek)
}

// MintFleetKey creates the standing credential for reaching hypervisors.
//
// Unscoped and non-expiring, which is safe only because its private half stays
// here — see tenant.SSHKeyPurposeFleet. A join key, whose private half must be
// written to a hypervisor, is minted by MintJoinKey and is neither.
func (s *SSHKeyService) MintFleetKey(ctx context.Context, orgID uuid.UUID, name string) (*tenant.SshKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrSSHKeyName
	}
	r, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.Mint(ctx, name)
}

// MintJoinKey creates a single-use credential for one cluster join.
//
// It exists because pvecm's SSH join path (`pvecm add --use_ssh`) still has to
// authenticate its FIRST step — ssh-copy-id — and normally does so with the
// target's root password, which nothing in this system owns and which TOTP on
// root@pam blocks outright. A scoped, expiring key replaces that.
//
// Genuinely single-use: after ssh-copy-id, PVE's own /root/.ssh/id_rsa is on the
// target and every later step of the join authenticates with it. This key
// bootstraps one step and is dead weight afterwards, which is why a short TTL is
// not a compromise.
func (s *SSHKeyService) MintJoinKey(
	ctx context.Context, orgID uuid.UUID, hostID, taskID uuid.UUID, ttl time.Duration,
) (*tenant.SshKey, error) {
	if hostID == uuid.Nil || taskID == uuid.Nil {
		return nil, fmt.Errorf("%w: a join key must name the host and the task it is for", ErrSSHKeyName)
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	r, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}
	exp := time.Now().UTC().Add(ttl)
	return r.MintWith(ctx, fmt.Sprintf("join-%s", taskID.String()[:8]), repo.MintOptions{
		Purpose:     tenant.SSHKeyPurposeJoin,
		ExpiresAt:   &exp,
		ScopeHostID: &hostID,
		ScopeTaskID: &taskID,
	})
}

// List returns the organization's keys. Public halves and fingerprints only —
// the model's private fields carry json:"-" so no caller can serialise one by
// accident.
func (s *SSHKeyService) List(ctx context.Context, orgID uuid.UUID) ([]tenant.SshKey, error) {
	r, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.List(ctx)
}

// Revoke marks a key unusable.
//
// The row is kept rather than deleted so that "which key was this host
// whitelisted with" stays answerable after the fact — a fingerprint found in an
// authorized_keys file has to be attributable to something.
func (s *SSHKeyService) Revoke(ctx context.Context, orgID, id uuid.UUID) error {
	r, err := s.bind(ctx, orgID)
	if err != nil {
		return err
	}
	return r.Revoke(ctx, id)
}

// PrivateKey unseals a key's private half.
//
// This is the ONLY way to obtain it, and it exists for River workers — trusted
// by design, running with the same environment as the API. No handler calls it
// and no response type has a field it could occupy, because a fleet-wide
// credential passing through an HTTP response would also pass through whatever
// logs, caches and error trackers sit between here and a browser.
func (s *SSHKeyService) PrivateKey(ctx context.Context, orgID, id uuid.UUID) ([]byte, error) {
	r, err := s.bind(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.PrivateKey(ctx, id)
}

// ActiveFleetKey returns the key an operator should whitelist, and the one a
// worker should authenticate with.
//
// It refuses rather than picking arbitrarily when several are usable. Two live
// fleet keys is not a state to paper over: the operator has whitelisted one of
// them on some hosts and the other elsewhere, and choosing silently means half
// the fleet becomes unreachable in a way that looks like a network fault.
func (s *SSHKeyService) ActiveFleetKey(ctx context.Context, orgID uuid.UUID) (*tenant.SshKey, error) {
	keys, err := s.List(ctx, orgID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	var usable []tenant.SshKey
	for _, k := range keys {
		if k.Purpose == tenant.SSHKeyPurposeFleet && k.Usable(now) {
			usable = append(usable, k)
		}
	}
	switch len(usable) {
	case 0:
		return nil, ErrNoUsableFleetKey
	case 1:
		return &usable[0], nil
	default:
		return nil, fmt.Errorf("%w: %d usable fleet keys — revoke the ones no longer whitelisted, "+
			"because picking one silently would leave every host holding the other unreachable",
			ErrSSHKeyName, len(usable))
	}
}
