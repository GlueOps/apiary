package tenant

import (
	"time"

	"github.com/google/uuid"
)

// SshKey is a tenant's onboarding keypair.
//
// It exists so a fleet can be brought up in bulk: the PUBLIC half is whitelisted
// on hypervisors (by image, cloud-init or config management), and River then
// reaches out over SSH to install and configure an agent on each one. That is a
// better bootstrap than an enrolment token, because the agent's own credential
// is minted and written during the install — it never transits a human, a
// runbook or a terminal history.
//
// The trade it makes is that the direction of trust reverses. An enrolment token
// lets a host prove it was authorised, and the control plane holds nothing
// dangerous. This private key is root on every hypervisor that has whitelisted
// it, so:
//
//   - it is TENANT scoped, so the blast radius is one customer's fleet;
//   - it is sealed at rest under that tenant's DEK, like every other secret here;
//   - reaching a host is expected to be constrained at the network layer
//     (nftables locked to the control plane's addresses), which is what keeps a
//     stolen key from being a root shell from anywhere;
//   - it is rotatable, and the replacement is minted through this same path so
//     River can still reprovision or restart an agent afterwards.
//
// The private half is never returned by any endpoint. Only PublicKey and
// Fingerprint leave the system.
type SshKey struct {
	ID   uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name string    `gorm:"not null" json:"name"`

	// Purpose separates the standing fleet credential from a key minted for one
	// operation.
	//
	// It exists because cluster join currently authenticates against the target's
	// PVE API with root's password, read from a file on the host that nothing in
	// this system mints, stores or rotates. That is a shared, permanent,
	// fleet-wide credential placed by hand — and it stops working entirely the
	// moment TOTP is enabled on root@pam, because TFA is a realm-level challenge
	// on the API. The more hardened the cluster, the less joinable it is.
	//
	// A per-join key replaces it with something scoped, expiring and revocable.
	Purpose string `gorm:"not null;default:'fleet';index" json:"purpose"`

	// ExpiresAt bounds a key's life independently of revocation.
	//
	// Revocation is an act somebody has to remember; expiry is what happens when
	// they do not. A join key outliving its join is a credential nobody is
	// watching, on a host nobody is thinking about.
	ExpiresAt *time.Time `gorm:"type:timestamptz;index" json:"expires_at,omitempty"`

	// ScopeHostID and ScopeTaskID narrow an ephemeral key to the one host and the
	// one operation it was minted for.
	//
	// Both nil for a fleet key, which is deliberately unscoped — it is the
	// standing credential the control plane uses to reach every host, and the
	// thing that makes an unscoped key acceptable is that its PRIVATE half never
	// leaves the control plane. A join key's private half must be written to a
	// hypervisor, so it may not also be fleet-wide.
	ScopeHostID *uuid.UUID `gorm:"type:uuid;index" json:"scope_host_id,omitempty"`
	ScopeTaskID *uuid.UUID `gorm:"type:uuid;index" json:"scope_task_id,omitempty"`

	// PublicKey in OpenSSH authorized_keys form, so it can be pasted or
	// templated onto a host without conversion.
	PublicKey string `gorm:"not null" json:"public_key"`

	EncryptedPrivateKey string `gorm:"not null" json:"-"`
	PrivateIV           string `gorm:"not null" json:"-"`
	PrivateTag          string `gorm:"not null" json:"-"`

	// PrivateKeyID names the key this row was sealed under.
	//
	// Without it a rotation cannot tell a re-wrapped row from one it has not
	// reached yet, and a partially-completed rotation is indistinguishable from a
	// finished one. Credentials carry the same field through utils.Sealed; this
	// model spells its sealed fields out, so it needs its own.
	PrivateKeyID uuid.UUID `gorm:"type:uuid" json:"-"`

	// Fingerprint in OpenSSH's own SHA256 form, over the PUBLIC key.
	//
	// It is what an operator compares against `ssh-keygen -lf` on a hypervisor to
	// confirm the right key was whitelisted. Complete, never truncated: a
	// fragment is not a fingerprint, it is a coincidence waiting to happen.
	Fingerprint string `gorm:"not null;index" json:"fingerprint"`

	// RevokedAt marks a key that must no longer be used to reach a host. The row
	// is kept rather than deleted so that "which key onboarded this hypervisor"
	// stays answerable after a rotation.
	RevokedAt *time.Time `gorm:"type:timestamptz" json:"revoked_at,omitempty"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}

// Active reports whether this key may still be used to reach a host.
func (k *SshKey) Active() bool { return k != nil && k.RevokedAt == nil }

// SSH key purposes.
const (
	// SSHKeyPurposeFleet is the standing credential the control plane uses to
	// reach hypervisors. Its private half never leaves the control plane, which
	// is what makes it safe to be unscoped.
	SSHKeyPurposeFleet = "fleet"

	// SSHKeyPurposeJoin is minted for ONE cluster join, on ONE host, and its
	// private half is written to that hypervisor so pvecm can authenticate over
	// SSH rather than against an API guarded by TOTP.
	//
	// Because the private half leaves the control plane, this purpose is only
	// acceptable while it is scoped and short-lived — which Usable() enforces
	// rather than leaving to whoever writes the worker.
	SSHKeyPurposeJoin = "join"
)

// Usable reports whether a key may still be used at the given moment.
//
// A join key with no expiry or no scope is refused outright rather than treated
// as a long-lived fleet key. That combination is how an ephemeral credential
// quietly becomes a permanent one: nothing fails, the key keeps working, and the
// private half stays on a hypervisor indefinitely.
func (k *SshKey) Usable(now time.Time) bool {
	if k == nil || k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.After(now) {
		return false
	}
	if k.Purpose == SSHKeyPurposeJoin {
		// Scoped AND expiring, both required. A check that cannot establish either
		// must refuse: the cost of refusing is one failed join, and the cost of
		// admitting is a fleet-wide private key on a hypervisor's disk.
		return k.ExpiresAt != nil && k.ScopeHostID != nil && k.ScopeTaskID != nil
	}
	return true
}

// UsableForJoin reports whether this key may authenticate a specific join.
//
// The scope is checked against the operation actually being performed, not
// merely present. A join key that is scoped to another host or another task is
// as wrong as an expired one, and far easier to reach by accident — the worker
// holds several keys and picking the wrong one is a typo, not a breach.
func (k *SshKey) UsableForJoin(now time.Time, hostID, taskID uuid.UUID) bool {
	if !k.Usable(now) || k.Purpose != SSHKeyPurposeJoin {
		return false
	}
	return *k.ScopeHostID == hostID && *k.ScopeTaskID == taskID
}
