package repo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// A join key's private half is written to a hypervisor. Storing one without a
// scope or an expiry is not a weaker join key — it is a fleet credential left on
// somebody else's disk, and nothing later would report it as wrong.
func TestMintRefusesAJoinKeyWithoutScopeOrExpiry(t *testing.T) {
	host, task := uuid.New(), uuid.New()
	exp := time.Now().Add(time.Minute)

	cases := []struct {
		name string
		opt  MintOptions
	}{
		{"no expiry", MintOptions{Purpose: tenant.SSHKeyPurposeJoin, ScopeHostID: &host, ScopeTaskID: &task}},
		{"no host", MintOptions{Purpose: tenant.SSHKeyPurposeJoin, ExpiresAt: &exp, ScopeTaskID: &task}},
		{"no task", MintOptions{Purpose: tenant.SSHKeyPurposeJoin, ExpiresAt: &exp, ScopeHostID: &host}},
		{"nothing at all", MintOptions{Purpose: tenant.SSHKeyPurposeJoin}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// A nil repo is enough: the shape is refused before any database work,
			// which is the point — the check is not a constraint you can be
			// missing a connection to.
			var r *SSHKeyRepoGorm
			_, err := r.MintWith(context.Background(), "probe", c.opt)
			if err == nil {
				t.Fatal("a contradictory join key was accepted")
			}
			if !strings.Contains(err.Error(), "scope") {
				t.Fatalf("the refusal does not explain what was missing: %v", err)
			}
		})
	}
}
