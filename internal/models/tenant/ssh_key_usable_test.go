package tenant

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func uuidNew() uuid.UUID { return uuid.New() }

func joinKey(over func(*SshKey)) *SshKey {
	host, task := uuidNew(), uuidNew()
	exp := time.Now().Add(10 * time.Minute)
	k := &SshKey{
		Name: "join", Purpose: SSHKeyPurposeJoin,
		ExpiresAt: &exp, ScopeHostID: &host, ScopeTaskID: &task,
	}
	if over != nil {
		over(k)
	}
	return k
}

// A join key's private half is written to a hypervisor, which is the whole
// reason it must be scoped and short-lived. An unscoped or non-expiring one is
// not a weaker join key — it is a fleet credential sitting on a host's disk.
func TestJoinKeyRefusedWithoutScopeOrExpiry(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name string
		key  *SshKey
	}{
		{"no expiry", joinKey(func(k *SshKey) { k.ExpiresAt = nil })},
		{"no host scope", joinKey(func(k *SshKey) { k.ScopeHostID = nil })},
		{"no task scope", joinKey(func(k *SshKey) { k.ScopeTaskID = nil })},
		{"expired", joinKey(func(k *SshKey) {
			past := now.Add(-time.Minute)
			k.ExpiresAt = &past
		})},
		{"revoked", joinKey(func(k *SshKey) { k.RevokedAt = &now })},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.key.Usable(now) {
				t.Fatal("a join key was usable in a shape that lets an ephemeral credential " +
					"become a permanent one")
			}
		})
	}

	if !joinKey(nil).Usable(now) {
		t.Fatal("a properly scoped, unexpired join key was refused")
	}
}

// A fleet key is deliberately unscoped, because its private half never leaves
// the control plane. Tightening it the same way would break the standing
// credential for no gain.
func TestFleetKeyMayBeUnscoped(t *testing.T) {
	k := &SshKey{Name: "fleet", Purpose: SSHKeyPurposeFleet}
	if !k.Usable(time.Now()) {
		t.Fatal("an unscoped fleet key was refused; its private half never leaves the control plane")
	}
	// Revocation and expiry still apply to it.
	now := time.Now()
	k.RevokedAt = &now
	if k.Usable(now) {
		t.Fatal("a revoked fleet key was usable")
	}
}

// Scope must be checked against the operation being performed, not merely
// present. The worker holds several keys, and picking the wrong one is a typo
// rather than a breach — so the check has to catch a typo.
func TestJoinKeyMustMatchTheOperationItWasMintedFor(t *testing.T) {
	now := time.Now()
	k := joinKey(nil)
	host, task := *k.ScopeHostID, *k.ScopeTaskID

	if !k.UsableForJoin(now, host, task) {
		t.Fatal("the key was refused for its own host and task")
	}
	if k.UsableForJoin(now, uuidNew(), task) {
		t.Fatal("a key scoped to another host was accepted")
	}
	if k.UsableForJoin(now, host, uuidNew()) {
		t.Fatal("a key scoped to another task was accepted")
	}
}

// A fleet key must never be accepted as a join key. It is the one credential
// that reaches every host, and using it for a join would write it to a
// hypervisor's disk — the exact thing the purpose split exists to prevent.
func TestFleetKeyIsNotAJoinKey(t *testing.T) {
	k := &SshKey{Name: "fleet", Purpose: SSHKeyPurposeFleet}
	if k.UsableForJoin(time.Now(), uuidNew(), uuidNew()) {
		t.Fatal("the standing fleet credential was accepted for a join")
	}
}
