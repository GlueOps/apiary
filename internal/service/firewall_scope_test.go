package service

import (
	"context"
	"errors"
	"testing"

	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var fwScopeSchema = testsupport.SchemaName("apiary_fwsvc_scope_test")

// The swarm scope was fully built and reachable by nothing.
//
// Every layer below this one had it — the column, ScopeRank, the ORDER BY CASE,
// BindingsForScope, ScopeForBee, and their tests — but the service took a bare
// hive, so no caller could create a swarm binding or preview one. Tests at the
// repo layer all passed while the feature did not exist from outside. These are
// the tests that would have failed then.
func fwScopeSvc(t *testing.T) (*FirewallService, *gorm.DB, uuid.UUID) {
	t.Helper()
	db := testsupport.OpenSchema(t, fwScopeSchema,
		&tenant.Hive{}, &tenant.Swarm{},
		&tenant.FirewallRuleset{}, &tenant.FirewallRule{}, &tenant.FirewallBinding{},
	)
	orgID := uuid.New()
	svc := NewFirewallService(&fakeTenants{
		byOrg: map[uuid.UUID]*database.TenantContext{orgID: {DB: db}},
	})
	return svc, db, orgID
}

func fwScopeSeed(t *testing.T, db *gorm.DB) (hive tenant.Hive, swarm tenant.Swarm, rs tenant.FirewallRuleset) {
	t.Helper()
	hive = tenant.Hive{Name: "fleet-" + uuid.NewString()[:8], Provider: "proxmox"}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}
	swarm = tenant.Swarm{HiveID: hive.ID, Name: "prod-" + uuid.NewString()[:8]}
	if err := db.Create(&swarm).Error; err != nil {
		t.Fatalf("create swarm: %v", err)
	}
	rs = tenant.FirewallRuleset{Name: "rs-" + uuid.NewString()[:8]}
	if err := db.Create(&rs).Error; err != nil {
		t.Fatalf("create ruleset: %v", err)
	}
	return hive, swarm, rs
}

func TestBindCanScopeARulesetToASwarm(t *testing.T) {
	svc, db, orgID := fwScopeSvc(t)
	ctx := context.Background()
	hive, swarm, rs := fwScopeSeed(t, db)

	b, err := svc.Bind(ctx, orgID, &hive.ID, &swarm.ID, rs.ID, 0, true)
	if err != nil {
		t.Fatalf("Bind to a swarm: %v — the scope exists everywhere below this layer, "+
			"so a refusal here means it is still unreachable from outside", err)
	}
	if b.SwarmID == nil || *b.SwarmID != swarm.ID {
		t.Fatalf("the binding came back scoped to %v, not to swarm %s — the swarm was "+
			"dropped on the way to the row, so the binding silently applies to the whole "+
			"fleet instead of one group", b.SwarmID, swarm.ID)
	}
	if b.HiveID == nil || *b.HiveID != hive.ID {
		t.Fatalf("the swarm binding lost its fleet: %v", b.HiveID)
	}

	var stored tenant.FirewallBinding
	if err := db.First(&stored, "id = ?", b.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.SwarmID == nil || *stored.SwarmID != swarm.ID {
		t.Fatalf("the row stored swarm %v, not %s", stored.SwarmID, swarm.ID)
	}
}

// A swarm names a group INSIDE a hive, and BindingsForScope matches swarm
// bindings on the swarm id ALONE — which is only safe because the pair is
// validated before the row is written. The database CHECK is the backstop; this
// is the refusal that reaches the operator as a 422 with a sentence they can act
// on rather than a constraint-violation 500.
func TestBindRefusesASwarmWithoutItsFleet(t *testing.T) {
	svc, db, orgID := fwScopeSvc(t)
	ctx := context.Background()
	_, swarm, rs := fwScopeSeed(t, db)

	_, err := svc.Bind(ctx, orgID, nil, &swarm.ID, rs.ID, 0, true)
	if !errors.Is(err, repo.ErrFirewallSwarmNeedsHive) {
		t.Fatalf("got %v, want ErrFirewallSwarmNeedsHive — a binding naming a group but no "+
			"fleet does not identify a scope, because group names are unique only per fleet", err)
	}
}

// Checked as a PAIR. A swarm that exists in a DIFFERENT fleet must not bind
// here: it would be written with this hive and that swarm, and then resolve
// against hosts the operator never named.
//
// TWO LAYERS REFUSE THIS, and the test asserts the refusal rather than which
// layer produced it. The composite foreign key fk_firewall_bindings_swarm is the
// enforcing one — it fires whether or not anything checked first, which is what
// makes the rule true for every writer. The service's SwarmExists call is the
// fast path that answers before a write is attempted. Removing the service check
// leaves this test green precisely because the backstop returns the same error,
// and that agreement is the property worth having: the guarantee does not rest
// on a caller remembering to ask.
func TestBindRefusesASwarmFromAnotherFleet(t *testing.T) {
	svc, db, orgID := fwScopeSvc(t)
	ctx := context.Background()
	hive, _, rs := fwScopeSeed(t, db)
	_, otherSwarm, _ := fwScopeSeed(t, db)

	_, err := svc.Bind(ctx, orgID, &hive.ID, &otherSwarm.ID, rs.ID, 0, true)
	if !errors.Is(err, repo.ErrFirewallSwarmNotFound) {
		t.Fatalf("got %v, want ErrFirewallSwarmNotFound — a group from another fleet was "+
			"accepted, so the binding applies to hosts nobody named", err)
	}

	// Nothing was written on the way to that refusal. A binding row left behind by
	// a rejected call is the failure the refusal exists to prevent.
	var n int64
	// Scoped to THIS ruleset: the schema is shared across the tests in this file,
	// so an unscoped count answers for their rows too and would fail on their
	// success rather than on this call's refusal.
	if err := db.Model(&tenant.FirewallBinding{}).
		Where("ruleset_id = ?", rs.ID).Count(&n).Error; err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d binding rows exist after a refused bind — the cross-fleet scope was "+
			"written and then reported as an error", n)
	}
}

// Preview is the screen that answers "what does this host actually get". If it
// cannot take a swarm, it answers for the fleet and quietly omits every
// swarm-scoped rule the host really receives — which is worse than not having
// the screen, because the operator believes it.
func TestPreviewSeesSwarmScopedRules(t *testing.T) {
	svc, db, orgID := fwScopeSvc(t)
	ctx := context.Background()
	hive, swarm, rs := fwScopeSeed(t, db)

	if err := db.Create(&tenant.FirewallRule{
		RulesetID: rs.ID, Name: "swarm-only", Enabled: true, Position: 1,
		Family: "inet", Protocol: "tcp", Action: "accept", DstPort: 22, DstPortEnd: 22,
	}).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}
	if _, err := svc.Bind(ctx, orgID, &hive.ID, &swarm.ID, rs.ID, 0, true); err != nil {
		t.Fatalf("Bind: %v", err)
	}

	inSwarm, err := svc.Preview(ctx, orgID, &hive.ID, &swarm.ID)
	if err != nil {
		t.Fatalf("Preview in swarm: %v", err)
	}
	if len(inSwarm) != 1 || inSwarm[0].Name != "swarm-only" {
		t.Fatalf("a host in the group previewed %d rules, want the one bound to its group — "+
			"the preview is ignoring the group and answering for the whole fleet", len(inSwarm))
	}

	// And the same host WITHOUT the group must not see it, or the scope is
	// decorative: everything would resolve everywhere and the preview would
	// simply always agree with itself.
	outOfSwarm, err := svc.Preview(ctx, orgID, &hive.ID, nil)
	if err != nil {
		t.Fatalf("Preview outside swarm: %v", err)
	}
	if len(outOfSwarm) != 0 {
		t.Fatalf("a host in the fleet but not the group previewed %d rules, want none — "+
			"a group-scoped binding is leaking to the whole fleet", len(outOfSwarm))
	}
}

func TestPreviewRefusesASwarmWithoutItsFleet(t *testing.T) {
	svc, db, orgID := fwScopeSvc(t)
	ctx := context.Background()
	_, swarm, _ := fwScopeSeed(t, db)

	if _, err := svc.Preview(ctx, orgID, nil, &swarm.ID); !errors.Is(err, repo.ErrFirewallSwarmNeedsHive) {
		t.Fatalf("got %v, want ErrFirewallSwarmNeedsHive", err)
	}
}
