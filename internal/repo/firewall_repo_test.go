package repo

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var firewallTestSchema = testsupport.SchemaName("apiary_repo_firewall_test")

func firewallDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, firewallTestSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.FirewallRuleset{},
		&tenant.FirewallRule{}, &tenant.FirewallBinding{},
		// Sets are migrated here too, and the RESTRICT constraint from a rule to
		// its set only exists if both tables do — a fixture missing the set table
		// would let TestASetInUseCannotBeDeleted pass for the wrong reason.
		&tenant.FirewallSet{}, &tenant.FirewallSetElement{},
	)
}

func seedSwarm(t *testing.T, db *gorm.DB, hive tenant.Hive, name string) tenant.Swarm {
	t.Helper()
	s := tenant.Swarm{HiveID: hive.ID, Name: name + "-" + uuid.NewString()[:8]}
	if err := db.Create(&s).Error; err != nil {
		t.Fatalf("create swarm: %v", err)
	}
	return s
}

// seedBeeOnHypervisor creates the machine record and the agent on it, linked the
// only way they are linked: Hypervisor.BeeID. Bee deliberately carries no hive
// of its own, so this is the sole route from an agent to a scope.
func seedBeeOnHypervisor(t *testing.T, db *gorm.DB, hiveID, swarmID *uuid.UUID) tenant.Bee {
	t.Helper()
	bee := tenant.Bee{APIKeyID: uuid.New()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatalf("create bee: %v", err)
	}
	hv := tenant.Hypervisor{HiveID: hiveID, SwarmID: swarmID, BeeID: &bee.ID, Address: "10.0.0.1"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("create hypervisor: %v", err)
	}
	return bee
}

func seedRuleset(t *testing.T, r *FirewallRepoGorm, name string) tenant.FirewallRuleset {
	t.Helper()
	rs := tenant.FirewallRuleset{Name: name + "-" + uuid.NewString()[:8]}
	if err := r.CreateRuleset(context.Background(), &rs); err != nil {
		t.Fatalf("create ruleset: %v", err)
	}
	return rs
}

func seedHive(t *testing.T, db *gorm.DB, name string) tenant.Hive {
	t.Helper()
	h := tenant.Hive{Name: name + "-" + uuid.NewString()[:8], Provider: "proxmox"}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}
	return h
}

// translateFirewallConflict matches on INDEX NAMES inside a driver error
// string. That is fragile in a specific way: renaming an index in the model
// would not break the build, would not fail any unit test, and would silently
// turn every one of these refusals into a 500 — the operator gets "firewall
// operation failed" instead of "that name is taken". Only a real Postgres can
// catch it.
func TestConflictsAreTranslatedNotLeakedAsDriverErrors(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	t.Run("duplicate ruleset name", func(t *testing.T) {
		first := seedRuleset(t, r, "management")
		dup := tenant.FirewallRuleset{Name: first.Name}
		if err := r.CreateRuleset(ctx, &dup); !errors.Is(err, ErrFirewallRulesetTaken) {
			t.Fatalf("got %v, want ErrFirewallRulesetTaken", err)
		}
	})

	t.Run("duplicate org-wide binding", func(t *testing.T) {
		rs := seedRuleset(t, r, "base")
		if err := r.CreateBinding(ctx, &tenant.FirewallBinding{RulesetID: rs.ID, Priority: 10, Enabled: true}); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		err := r.CreateBinding(ctx, &tenant.FirewallBinding{RulesetID: rs.ID, Priority: 20, Enabled: true})
		if !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists — a nullable column makes this the "+
				"case a plain unique index does NOT catch", err)
		}
	})

	t.Run("duplicate hive binding", func(t *testing.T) {
		rs := seedRuleset(t, r, "dc")
		hive := seedHive(t, db, "dc1")
		b := tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 10, Enabled: true}
		if err := r.CreateBinding(ctx, &b); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		dup := tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 20, Enabled: true}
		if err := r.CreateBinding(ctx, &dup); !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists", err)
		}
	})

	// The swarm index is NEW, and the matcher is a substring switch: adding an
	// index without adding its name here turns every duplicate swarm binding into
	// a 500 rather than a 409.
	t.Run("duplicate swarm binding", func(t *testing.T) {
		rs := seedRuleset(t, r, "swarm-dup")
		hive := seedHive(t, db, "dc-swarm")
		swarm := seedSwarm(t, db, hive, "prod")
		b := tenant.FirewallBinding{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 10, Enabled: true}
		if err := r.CreateBinding(ctx, &b); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		dup := tenant.FirewallBinding{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 20, Enabled: true}
		if err := r.CreateBinding(ctx, &dup); !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists", err)
		}
	})

	t.Run("binding a swarm that does not exist", func(t *testing.T) {
		rs := seedRuleset(t, r, "ghost-swarm")
		hive := seedHive(t, db, "dc-ghost")
		missing := uuid.New()
		err := r.CreateBinding(ctx, &tenant.FirewallBinding{
			HiveID: &hive.ID, SwarmID: &missing, RulesetID: rs.ID, Priority: 10, Enabled: true,
		})
		if !errors.Is(err, ErrFirewallSwarmNotFound) {
			t.Fatalf("got %v, want ErrFirewallSwarmNotFound", err)
		}
	})

	t.Run("binding a fleet that does not exist", func(t *testing.T) {
		rs := seedRuleset(t, r, "ghost")
		missing := uuid.New()
		err := r.CreateBinding(ctx, &tenant.FirewallBinding{HiveID: &missing, RulesetID: rs.ID, Priority: 10, Enabled: true})
		if !errors.Is(err, ErrFirewallHiveNotFound) {
			t.Fatalf("got %v, want ErrFirewallHiveNotFound", err)
		}
	})
}

// The same ruleset bound org-wide AND to a fleet is the base-plus-datacenter
// pattern the whole structure exists for. A fix for the nullable-unique gap that
// blocked this would be worse than the gap.
func TestARulesetMayBeBothOrgWideAndFleetScoped(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "shared")
	hive := seedHive(t, db, "dc-shared")

	if err := r.CreateBinding(ctx, &tenant.FirewallBinding{RulesetID: rs.ID, Priority: 10, Enabled: true}); err != nil {
		t.Fatalf("org-wide bind: %v", err)
	}
	if err := r.CreateBinding(ctx, &tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 20, Enabled: true}); err != nil {
		t.Fatalf("fleet bind was refused after an org-wide one: %v", err)
	}
}

// ListRules must return exactly what tenant.OrderedRules would, because the UI
// shows one and the host enforces the other. Two orderings written in two places
// do not stay in agreement.
func TestListRulesMatchesTheRenderOrder(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()
	rs := seedRuleset(t, r, "ordered")

	// Inserted out of order, and two share a position so the tiebreak is exercised.
	for _, spec := range []struct {
		name string
		pos  int
	}{{"third", 30}, {"first", 10}, {"second-a", 20}, {"second-b", 20}} {
		rule := tenant.FirewallRule{
			RulesetID: rs.ID, Name: spec.name, Enabled: true,
			Family: tenant.FamilyInet, TableName: "filter", ChainName: "input", Position: spec.pos,
			SrcCidr: "0.0.0.0/0", DstCidr: "0.0.0.0/0",
			Protocol: tenant.TCP, DstPort: 22, Action: tenant.Accept,
		}
		if err := r.CreateRule(ctx, &rule); err != nil {
			t.Fatalf("create rule %s: %v", spec.name, err)
		}
	}

	listed, err := r.ListRules(ctx, rs.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	bindings := []tenant.FirewallBinding{{RulesetID: rs.ID, Priority: 10, Enabled: true}}
	rendered := tenant.OrderedRules(bindings, map[uuid.UUID][]tenant.FirewallRule{rs.ID: listed})

	if len(rendered) != len(listed) {
		t.Fatalf("render dropped rules: %d listed, %d rendered", len(listed), len(rendered))
	}
	for i := range listed {
		if listed[i].ID != rendered[i].ID {
			t.Fatalf("the repository's ORDER BY and OrderedRules disagree at position %d "+
				"(%q vs %q); the operator would be reading a different policy from the one enforced",
				i, listed[i].Name, rendered[i].Name)
		}
	}
	if listed[0].Name != "first" {
		t.Fatalf("first rule is %q, want \"first\"", listed[0].Name)
	}
}

// POSTGRES TREATS NULL AS DISTINCT FROM NULL, so a plain unique index over the
// nullable scope columns constrains NOTHING — every org-wide row is unique
// against every other org-wide row, and the same ruleset binds any number of
// times. Each copy renders, so the ruleset appears repeatedly in the chain the
// host loads. This has been got wrong here before and only a real Postgres can
// catch it: the model tags look correct either way.
//
// One test per scope, because the three are enforced by three DIFFERENT partial
// indexes and a predicate that stops covering one of them leaves that scope
// silently unconstrained.
func TestARulesetCannotBeBoundTwiceAtTheSameScope(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	countBindings := func(rulesetID uuid.UUID) int64 {
		var n int64
		if err := db.Model(&tenant.FirewallBinding{}).Where("ruleset_id = ?", rulesetID).Count(&n).Error; err != nil {
			t.Fatalf("count bindings: %v", err)
		}
		return n
	}

	t.Run("org-wide", func(t *testing.T) {
		rs := seedRuleset(t, r, "dup-org")
		first := tenant.FirewallBinding{RulesetID: rs.ID, Priority: 10, Enabled: true}
		if err := r.CreateBinding(ctx, &first); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		dup := tenant.FirewallBinding{RulesetID: rs.ID, Priority: 20, Enabled: true}
		if err := r.CreateBinding(ctx, &dup); !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists", err)
		}
		if n := countBindings(rs.ID); n != 1 {
			t.Fatalf("%d org-wide bindings survived for one ruleset; each one renders, so the "+
				"ruleset would appear %d times in the chain the host loads", n, n)
		}
	})

	t.Run("one hive", func(t *testing.T) {
		rs := seedRuleset(t, r, "dup-hive")
		hive := seedHive(t, db, "dup-dc")
		first := tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 10, Enabled: true}
		if err := r.CreateBinding(ctx, &first); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		dup := tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 20, Enabled: true}
		if err := r.CreateBinding(ctx, &dup); !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists", err)
		}
		if n := countBindings(rs.ID); n != 1 {
			t.Fatalf("%d hive bindings survived for one ruleset in one fleet", n)
		}
	})

	t.Run("one swarm", func(t *testing.T) {
		rs := seedRuleset(t, r, "dup-swarm")
		hive := seedHive(t, db, "dup-swarm-dc")
		swarm := seedSwarm(t, db, hive, "fleet-prod")
		first := tenant.FirewallBinding{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 10, Enabled: true}
		if err := r.CreateBinding(ctx, &first); err != nil {
			t.Fatalf("first bind: %v", err)
		}
		dup := tenant.FirewallBinding{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 20, Enabled: true}
		if err := r.CreateBinding(ctx, &dup); !errors.Is(err, ErrFirewallBindingExists) {
			t.Fatalf("got %v, want ErrFirewallBindingExists", err)
		}
		if n := countBindings(rs.ID); n != 1 {
			t.Fatalf("%d swarm bindings survived for one ruleset in one swarm", n)
		}
	})
}

// The whole point of the change: fleet-prod and fleet-nonprod inside ONE cluster
// must be able to carry the same ruleset independently.
//
// This is what the hive index's predicate gaining `AND swarm_id IS NULL`
// protects. Without it, swarm rows fall into an index keyed (hive_id,
// ruleset_id), and the second swarm is refused with an error naming a different
// swarm — a correct operation reported as a conflict.
func TestOneRulesetMayBeBoundToTwoSwarmsInTheSameFleet(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "base-extra")
	hive := seedHive(t, db, "one-cluster")
	prod := seedSwarm(t, db, hive, "fleet-prod")
	nonprod := seedSwarm(t, db, hive, "fleet-nonprod")

	if err := r.CreateBinding(ctx, &tenant.FirewallBinding{
		HiveID: &hive.ID, SwarmID: &prod.ID, RulesetID: rs.ID, Priority: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("bind to fleet-prod: %v", err)
	}
	if err := r.CreateBinding(ctx, &tenant.FirewallBinding{
		HiveID: &hive.ID, SwarmID: &nonprod.ID, RulesetID: rs.ID, Priority: 10, Enabled: true,
	}); err != nil {
		t.Fatalf("binding the same ruleset to a SECOND swarm in the same cluster was refused: %v", err)
	}
}

// The three scopes are independent, so all three may carry one ruleset at once.
// A fix for the null-uniqueness gap that blocked this would be worse than the
// gap it closed.
func TestOneRulesetMayBeBoundAtAllThreeScopesAtOnce(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "everywhere")
	hive := seedHive(t, db, "three-scope")
	swarm := seedSwarm(t, db, hive, "prod")

	for _, b := range []tenant.FirewallBinding{
		{RulesetID: rs.ID, Priority: 10, Enabled: true},
		{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 20, Enabled: true},
		{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 30, Enabled: true},
	} {
		binding := b
		if err := r.CreateBinding(ctx, &binding); err != nil {
			t.Fatalf("bind at scope (hive=%v swarm=%v): %v", b.HiveID, b.SwarmID, err)
		}
	}
}

// A swarm without its fleet is the one combination the schema forbids, and the
// reason is not tidiness: hive_id is what the composite foreign key uses to
// prove the swarm is in the fleet the binding names. A (NULL, swarm) row also
// reads as ORG-WIDE to anything that tests hive_id first, which is how a
// one-swarm ruleset would silently become an organization-wide one.
func TestASwarmScopedBindingMustNameItsFleet(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "orphan-swarm")
	hive := seedHive(t, db, "orphan-dc")
	swarm := seedSwarm(t, db, hive, "prod")

	err := r.CreateBinding(ctx, &tenant.FirewallBinding{
		SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 10, Enabled: true,
	})
	if !errors.Is(err, ErrFirewallSwarmNeedsHive) {
		t.Fatalf("got %v, want ErrFirewallSwarmNeedsHive", err)
	}
}

// The composite foreign key makes "the binding's swarm is in the binding's
// fleet" a database rule. A binding that paired one cluster's hive with another
// cluster's swarm would resolve rules for hosts in neither.
func TestBindingASwarmFromADifferentFleetIsRefused(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "cross-fleet")
	dc1 := seedHive(t, db, "dc1")
	dc2 := seedHive(t, db, "dc2")
	inDC2 := seedSwarm(t, db, dc2, "prod")

	err := r.CreateBinding(ctx, &tenant.FirewallBinding{
		HiveID: &dc1.ID, SwarmID: &inDC2.ID, RulesetID: rs.ID, Priority: 10, Enabled: true,
	})
	if !errors.Is(err, ErrFirewallSwarmNotFound) {
		t.Fatalf("got %v, want ErrFirewallSwarmNotFound", err)
	}

	ok, err := r.SwarmExists(ctx, dc1.ID, inDC2.ID)
	if err != nil {
		t.Fatalf("SwarmExists: %v", err)
	}
	if ok {
		t.Fatal("SwarmExists accepted a swarm from another fleet; the pair is the identity, not the id alone")
	}
	if ok, err := r.SwarmExists(ctx, dc2.ID, inDC2.ID); err != nil || !ok {
		t.Fatalf("SwarmExists(dc2, its own swarm) = %v, %v; want true", ok, err)
	}
}

// THE SECURITY ONE. ON DELETE SET NULL on this composite key would SUCCEED —
// Postgres nulls EVERY column of a composite foreign key, and both are nullable
// here — converting a one-swarm binding into an ORG-WIDE one. Deleting a swarm
// would widen its ruleset to every host in the organization with nothing
// reporting it.
//
// RESTRICT is what makes that unreachable, and the assertion is on the surviving
// row rather than only on the error: an implementation that deleted the swarm
// and left the binding intact-but-orphaned would also return no error here.
func TestDeletingASwarmCannotSilentlyWidenItsBindingToOrgWide(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := seedRuleset(t, r, "prod-only")
	hive := seedHive(t, db, "restrict-dc")
	swarm := seedSwarm(t, db, hive, "fleet-prod")

	binding := tenant.FirewallBinding{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: rs.ID, Priority: 10, Enabled: true}
	if err := r.CreateBinding(ctx, &binding); err != nil {
		t.Fatalf("bind: %v", err)
	}

	if err := db.Delete(&tenant.Swarm{}, "id = ?", swarm.ID).Error; err == nil {
		t.Fatal("deleting a swarm that still carries a firewall binding was ALLOWED; " +
			"under SET NULL that binding is now org-wide and this ruleset reaches every " +
			"host in the organization")
	}

	var after tenant.FirewallBinding
	if err := db.First(&after, "id = ?", binding.ID).Error; err != nil {
		t.Fatalf("reload binding: %v", err)
	}
	if after.SwarmID == nil || *after.SwarmID != swarm.ID {
		t.Fatalf("binding's swarm_id is now %v, want %v — the scope widened", after.SwarmID, swarm.ID)
	}
	if after.HiveID == nil {
		t.Fatal("binding's hive_id was nulled; the binding now reads as org-wide")
	}
}

// Resolution, and the failure the swarm scope exists to prevent: a host in
// fleet-nonprod must not receive fleet-prod's rules.
//
// The hive arm is what gets this wrong. `hive_id IS NULL OR hive_id = ?` looks
// right and now matches every swarm binding in the cluster too, because a
// swarm-scoped row carries hive_id as well.
func TestBindingsForScopeGivesOneSwarmNotItsSibling(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	orgRS := seedRuleset(t, r, "org-base")
	hiveRS := seedRuleset(t, r, "cluster-base")
	prodRS := seedRuleset(t, r, "prod-only")
	nonprodRS := seedRuleset(t, r, "nonprod-only")

	hive := seedHive(t, db, "resolve-dc")
	other := seedHive(t, db, "other-dc")
	prod := seedSwarm(t, db, hive, "fleet-prod")
	nonprod := seedSwarm(t, db, hive, "fleet-nonprod")
	elsewhere := seedSwarm(t, db, other, "fleet-prod")
	otherRS := seedRuleset(t, r, "other-cluster")

	for _, b := range []tenant.FirewallBinding{
		{RulesetID: orgRS.ID, Priority: 10, Enabled: true},
		{HiveID: &hive.ID, RulesetID: hiveRS.ID, Priority: 20, Enabled: true},
		{HiveID: &hive.ID, SwarmID: &prod.ID, RulesetID: prodRS.ID, Priority: 30, Enabled: true},
		{HiveID: &hive.ID, SwarmID: &nonprod.ID, RulesetID: nonprodRS.ID, Priority: 30, Enabled: true},
		{HiveID: &other.ID, SwarmID: &elsewhere.ID, RulesetID: otherRS.ID, Priority: 30, Enabled: true},
	} {
		binding := b
		if err := r.CreateBinding(ctx, &binding); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
	}

	// The schema is shared by every test in this package and nothing truncates
	// between them, so org-wide bindings from other tests are legitimately in the
	// table. Narrowing to this test's own rulesets keeps the exact-count
	// assertions meaningful — a leaked sibling swarm is still in this set and
	// still fails.
	mine := map[uuid.UUID]bool{
		orgRS.ID: true, hiveRS.ID: true, prodRS.ID: true, nonprodRS.ID: true, otherRS.ID: true,
	}
	rulesetsFor := func(scope tenant.BindingScope) []uuid.UUID {
		got, err := r.BindingsForScope(ctx, scope)
		if err != nil {
			t.Fatalf("BindingsForScope: %v", err)
		}
		ids := make([]uuid.UUID, 0, len(got))
		for _, b := range got {
			if mine[b.RulesetID] {
				ids = append(ids, b.RulesetID)
			}
		}
		return ids
	}

	t.Run("a host in fleet-prod", func(t *testing.T) {
		ids := rulesetsFor(tenant.BindingScope{HiveID: &hive.ID, SwarmID: &prod.ID})
		for _, want := range []uuid.UUID{orgRS.ID, hiveRS.ID, prodRS.ID} {
			if !slices.Contains(ids, want) {
				t.Fatalf("missing ruleset %v; got %v", want, ids)
			}
		}
		if slices.Contains(ids, nonprodRS.ID) {
			t.Fatalf("a host in fleet-prod received fleet-nonprod's ruleset; got %v", ids)
		}
		if slices.Contains(ids, otherRS.ID) {
			t.Fatalf("a host received another cluster's swarm ruleset; got %v", ids)
		}
		if len(ids) != 3 {
			t.Fatalf("got %d rulesets, want exactly 3: %v", len(ids), ids)
		}
	})

	t.Run("spare capacity, in the cluster but in no swarm", func(t *testing.T) {
		ids := rulesetsFor(tenant.BindingScope{HiveID: &hive.ID})
		for _, want := range []uuid.UUID{orgRS.ID, hiveRS.ID} {
			if !slices.Contains(ids, want) {
				t.Fatalf("missing ruleset %v; got %v", want, ids)
			}
		}
		if slices.Contains(ids, prodRS.ID) || slices.Contains(ids, nonprodRS.ID) {
			t.Fatalf("spare metal received a swarm's rules; got %v", ids)
		}
		if len(ids) != 2 {
			t.Fatalf("got %d rulesets, want exactly 2: %v", len(ids), ids)
		}
	})

	// A machine between clusters, or one still being onboarded. It still needs
	// management access, so the answer is the org-wide bindings — never nothing.
	t.Run("no cluster at all", func(t *testing.T) {
		ids := rulesetsFor(tenant.BindingScope{})
		if len(ids) != 1 || ids[0] != orgRS.ID {
			t.Fatalf("got %v, want exactly the org-wide ruleset %v; an empty answer means the "+
				"host holds whatever it last applied with nothing reporting why", ids, orgRS.ID)
		}
	})
}

// The repository's ORDER BY and tenant.CompareBindingsForRender must agree, or
// the operator reads one policy from the list while the host enforces another.
// The scope CASE is written twice — once in SQL, once in Go — and two copies do
// not stay in agreement on their own.
func TestBindingsForScopeMatchesTheRenderComparator(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "order-dc")
	swarm := seedSwarm(t, db, hive, "fleet-prod")

	// All at the DEFAULT priority, which is the common case and the only one
	// where the scope tie-break shows. Inserted least-specific first, so a query
	// that ordered by age alone would return them backwards.
	orgRS := seedRuleset(t, r, "ord-org")
	hiveRS := seedRuleset(t, r, "ord-hive")
	swarmRS := seedRuleset(t, r, "ord-swarm")
	// And one with an explicit lower priority, to prove priority still outranks
	// the tie-break rather than being overtaken by it.
	firstRS := seedRuleset(t, r, "ord-first")

	for _, b := range []tenant.FirewallBinding{
		{RulesetID: orgRS.ID, Priority: 100, Enabled: true},
		{HiveID: &hive.ID, RulesetID: hiveRS.ID, Priority: 100, Enabled: true},
		{HiveID: &hive.ID, SwarmID: &swarm.ID, RulesetID: swarmRS.ID, Priority: 100, Enabled: true},
		{RulesetID: firstRS.ID, Priority: 10, Enabled: true},
	} {
		binding := b
		if err := r.CreateBinding(ctx, &binding); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
	}

	all, err := r.BindingsForScope(ctx, tenant.BindingScope{HiveID: &hive.ID, SwarmID: &swarm.ID})
	if err != nil {
		t.Fatalf("BindingsForScope: %v", err)
	}
	// Other tests in this package leave org-wide bindings behind in the shared
	// schema. Filtering preserves relative order, which is what is under test.
	mine := map[uuid.UUID]bool{orgRS.ID: true, hiveRS.ID: true, swarmRS.ID: true, firstRS.ID: true}
	got := make([]tenant.FirewallBinding, 0, 4)
	for _, b := range all {
		if mine[b.RulesetID] {
			got = append(got, b)
		}
	}
	if len(got) != 4 {
		t.Fatalf("got %d of this test's bindings, want 4", len(got))
	}

	wantOrder := []uuid.UUID{firstRS.ID, swarmRS.ID, hiveRS.ID, orgRS.ID}
	for i, want := range wantOrder {
		if got[i].RulesetID != want {
			t.Fatalf("SQL order position %d is ruleset %v, want %v; earlier renders FIRST and "+
				"nft stops at the first terminal verdict, so this is the difference between the "+
				"swarm's rules overriding the org-wide ones and being shadowed by them",
				i, got[i].RulesetID, want)
		}
	}

	resorted := slices.Clone(got)
	slices.SortStableFunc(resorted, tenant.CompareBindingsForRender)
	for i := range got {
		if got[i].ID != resorted[i].ID {
			t.Fatalf("the repository's ORDER BY and tenant.CompareBindingsForRender disagree at "+
				"position %d; the list and the rendered chain would be different policies", i)
		}
	}
}

// The scope comes from ONE row — the hypervisor carrying the bee — because hive
// and swarm have to agree with each other. Bee carries no hive at all, so a
// missing hypervisor means org-wide, not an error and not a guess.
func TestScopeForBeeTakesTheHiveAndSwarmAsAPair(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "scope-dc")
	swarm := seedSwarm(t, db, hive, "fleet-prod")

	t.Run("assigned to a swarm", func(t *testing.T) {
		bee := seedBeeOnHypervisor(t, db, &hive.ID, &swarm.ID)
		scope, err := r.ScopeForBee(ctx, bee.ID)
		if err != nil {
			t.Fatalf("ScopeForBee: %v", err)
		}
		if scope.HiveID == nil || *scope.HiveID != hive.ID {
			t.Fatalf("hive = %v, want %v", scope.HiveID, hive.ID)
		}
		if scope.SwarmID == nil || *scope.SwarmID != swarm.ID {
			t.Fatalf("swarm = %v, want %v", scope.SwarmID, swarm.ID)
		}
	})

	// Spare capacity: in the cluster, in no swarm. Note that placement
	// deliberately excludes spare metal from candidacy; firewall resolution must
	// NOT copy that, because a spare host still needs management access.
	t.Run("spare capacity", func(t *testing.T) {
		bee := seedBeeOnHypervisor(t, db, &hive.ID, nil)
		scope, err := r.ScopeForBee(ctx, bee.ID)
		if err != nil {
			t.Fatalf("ScopeForBee: %v", err)
		}
		if scope.HiveID == nil || *scope.HiveID != hive.ID {
			t.Fatalf("hive = %v, want %v", scope.HiveID, hive.ID)
		}
		if scope.SwarmID != nil {
			t.Fatalf("swarm = %v, want nil", *scope.SwarmID)
		}
	})

	// An agent enrolled before its machine was discovered, or one whose cluster
	// fields were cleared on departure. Normal, and it must resolve to the
	// org-wide bindings rather than refusing.
	t.Run("no hypervisor row", func(t *testing.T) {
		bee := tenant.Bee{APIKeyID: uuid.New()}
		if err := db.Create(&bee).Error; err != nil {
			t.Fatalf("create bee: %v", err)
		}
		scope, err := r.ScopeForBee(ctx, bee.ID)
		if err != nil {
			t.Fatalf("ScopeForBee refused an agent with no machine record: %v; an absent answer "+
				"means the host holds whatever it last applied, with nothing reporting why", err)
		}
		if scope.HiveID != nil || scope.SwarmID != nil {
			t.Fatalf("scope = %+v, want empty", scope)
		}
	})
}

// The org arm's `AND swarm_id IS NULL` is strictly redundant while the CHECK
// stands. It is kept so the three predicates partition the table on their OWN
// face rather than depending on a constraint declared elsewhere — and this test
// is what stops that from being an untested claim in a comment, which is how
// such a clause gets "simplified away" later.
//
// With the CHECK gone, a (NULL, swarm) row is insertable: the composite foreign
// key is MATCH SIMPLE, so it is not checked at all when either column is null.
// Such a row must not be delivered to a host as org-wide.
//
// It drops and restores the constraint on the schema this whole package shares,
// so it must not be made parallel.
func TestTheOrgArmDoesNotLeanOnTheCheckConstraint(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "no-check-dc")
	swarm := seedSwarm(t, db, hive, "fleet-prod")
	rs := seedRuleset(t, r, "illegal-scope")

	const addCheck = `ALTER TABLE firewall_bindings ADD CONSTRAINT chk_firewall_bindings_swarm_needs_hive ` +
		`CHECK (swarm_id IS NULL OR hive_id IS NOT NULL)`
	if err := db.Exec(`ALTER TABLE firewall_bindings DROP CONSTRAINT chk_firewall_bindings_swarm_needs_hive`).Error; err != nil {
		t.Fatalf("drop check: %v (is the constraint still named this? the schema is what enforces "+
			"the scope partition, so a rename here is a silent change of behaviour)", err)
	}

	illegal := uuid.New()
	t.Cleanup(func() {
		db.Exec(`DELETE FROM firewall_bindings WHERE id = ?`, illegal)
		if err := db.Exec(addCheck).Error; err != nil {
			t.Errorf("could not restore the CHECK constraint: %v", err)
		}
	})
	if err := db.Exec(
		`INSERT INTO firewall_bindings (id, swarm_id, ruleset_id, priority, enabled) VALUES (?, ?, ?, ?, ?)`,
		illegal, swarm.ID, rs.ID, 10, true,
	).Error; err != nil {
		t.Fatalf("insert a (NULL hive, swarm) row with the CHECK removed: %v", err)
	}

	for name, scope := range map[string]tenant.BindingScope{
		"a host in no cluster":                           {},
		"a host in the swarm's own cluster, in no swarm": {HiveID: &hive.ID},
	} {
		got, err := r.BindingsForScope(ctx, scope)
		if err != nil {
			t.Fatalf("%s: BindingsForScope: %v", name, err)
		}
		for _, b := range got {
			if b.RulesetID == rs.ID {
				t.Fatalf("%s: a binding naming only a swarm was delivered as though it were "+
					"org-wide; the org arm is relying on the CHECK to exclude it", name)
			}
		}
	}
}

// TestCreateStoresDisabledRatherThanSilentlyEnabling is a security test, not a
// tidiness one.
//
// GORM omits a zero value from the INSERT when the field carries a `default:`
// tag, so `Create(&FirewallRule{Enabled: false})` stores TRUE. A rule or binding
// an operator deliberately created DISABLED would therefore be live the moment it
// was written — a firewall rule applying when its author asked for it not to.
//
// This is the same defect already documented for Hypervisor.Schedulable, where it
// silently failed to cordon. Here the blast radius is a packet filter.
func TestCreateStoresDisabledRatherThanSilentlyEnabling(t *testing.T) {
	db := firewallDB(t)
	repo := NewFirewallRepoGorm(db)
	ctx := context.Background()

	rs := &tenant.FirewallRuleset{Name: "rs-" + uuid.NewString()[:8]}
	if err := repo.CreateRuleset(ctx, rs); err != nil {
		t.Fatalf("create ruleset: %v", err)
	}

	t.Run("rule", func(t *testing.T) {
		rule := &tenant.FirewallRule{
			RulesetID: rs.ID, Name: "drop-all", Enabled: false,
			Family: "inet", TableName: "filter", ChainName: "input", Position: 100,
			SrcCidr: "0.0.0.0/0", DstCidr: "0.0.0.0/0", Protocol: "any", Action: "drop",
		}
		if err := repo.CreateRule(ctx, rule); err != nil {
			t.Fatalf("create rule: %v", err)
		}
		var back tenant.FirewallRule
		if err := db.First(&back, "id = ?", rule.ID).Error; err != nil {
			t.Fatal(err)
		}
		if back.Enabled {
			t.Fatal("a rule created with Enabled=false was stored ENABLED — it is live " +
				"despite its author disabling it")
		}
	})

	t.Run("binding", func(t *testing.T) {
		b := &tenant.FirewallBinding{RulesetID: rs.ID, Priority: 100, Enabled: false}
		if err := repo.CreateBinding(ctx, b); err != nil {
			t.Fatalf("create binding: %v", err)
		}
		var back tenant.FirewallBinding
		if err := db.First(&back, "id = ?", b.ID).Error; err != nil {
			t.Fatal(err)
		}
		if back.Enabled {
			t.Fatal("a binding created with Enabled=false was stored ENABLED — the ruleset " +
				"applies immediately despite being bound disabled")
		}
	})

	t.Run("enabled still defaults to true when not specified", func(t *testing.T) {
		// The fix must not invert the default: a rule created without an opinion
		// is enabled, which is what the column default is for.
		rule := &tenant.FirewallRule{
			RulesetID: rs.ID, Name: "accept-ssh", Enabled: true,
			Family: "inet", TableName: "filter", ChainName: "input", Position: 200,
			SrcCidr: "0.0.0.0/0", DstCidr: "0.0.0.0/0", Protocol: "tcp", Action: "accept",
		}
		if err := repo.CreateRule(ctx, rule); err != nil {
			t.Fatalf("create rule: %v", err)
		}
		var back tenant.FirewallRule
		if err := db.First(&back, "id = ?", rule.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !back.Enabled {
			t.Fatal("a rule created enabled must stay enabled")
		}
	})
}

// TestAnUpdateWithNoScopeLeavesTheBindingWhereItIs.
//
// THE WIDENING TEST. Every client that predates the scope field sends only
// priority and enabled. If absence were read as "both ids empty", each of those
// requests would silently move a fleet-scoped binding to org-wide — pushing that
// ruleset onto every host in the organization, including hosts in no fleet. It
// is the worst accident this endpoint could have, and it would look like a
// successful save.
func TestAnUpdateWithNoScopeLeavesTheBindingWhereItIs(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "prod")
	rs := seedRuleset(t, r, "base")
	b := &tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 100, Enabled: true}
	if err := r.CreateBinding(ctx, b); err != nil {
		t.Fatalf("bind: %v", err)
	}

	// Exactly what an older client sends.
	if err := r.UpdateBinding(ctx, b.ID, 50, false); err != nil {
		t.Fatalf("update: %v", err)
	}

	var after tenant.FirewallBinding
	if err := db.First(&after, "id = ?", b.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.HiveID == nil {
		t.Fatal("an update that said nothing about scope widened the binding from one fleet " +
			"to the whole organization")
	}
	if *after.HiveID != hive.ID {
		t.Fatalf("scope moved to %s, want %s", *after.HiveID, hive.ID)
	}
	if after.Priority != 50 || after.Enabled {
		t.Fatalf("the fields that WERE named did not change: priority=%d enabled=%v",
			after.Priority, after.Enabled)
	}
}

// TestAnExplicitScopeMovesTheBinding, which is the whole point — and an empty
// hive id is how org-wide is spelled, so it must clear rather than be ignored.
func TestAnExplicitScopeMovesTheBinding(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "prod")
	rs := seedRuleset(t, r, "base")
	b := &tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 100, Enabled: true}
	if err := r.CreateBinding(ctx, b); err != nil {
		t.Fatalf("bind: %v", err)
	}

	// Fleet -> org-wide.
	if err := r.UpdateBindingWithScope(ctx, b.ID, 100, true, &BindingScopeChange{}); err != nil {
		t.Fatalf("re-scope to org: %v", err)
	}
	var after tenant.FirewallBinding
	_ = db.First(&after, "id = ?", b.ID).Error
	if after.HiveID != nil {
		t.Fatalf("still scoped to fleet %s after being moved org-wide", *after.HiveID)
	}

	// And back again.
	if err := r.UpdateBindingWithScope(ctx, b.ID, 100, true,
		&BindingScopeChange{HiveID: &hive.ID}); err != nil {
		t.Fatalf("re-scope to fleet: %v", err)
	}
	_ = db.First(&after, "id = ?", b.ID).Error
	if after.HiveID == nil || *after.HiveID != hive.ID {
		t.Fatal("moving back to a fleet did not take")
	}
}

// TestMovingOntoAnOccupiedScopeIsRefusedAsAConflict.
//
// The same ruleset bound twice at one scope renders twice, duplicating every
// rule in it. The operator needs "already bound there", not a constraint name.
func TestMovingOntoAnOccupiedScopeIsRefusedAsAConflict(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	hive := seedHive(t, db, "prod")
	rs := seedRuleset(t, r, "base")

	occupied := &tenant.FirewallBinding{RulesetID: rs.ID, Priority: 100, Enabled: true}
	if err := r.CreateBinding(ctx, occupied); err != nil {
		t.Fatalf("bind org-wide: %v", err)
	}
	moving := &tenant.FirewallBinding{HiveID: &hive.ID, RulesetID: rs.ID, Priority: 100, Enabled: true}
	if err := r.CreateBinding(ctx, moving); err != nil {
		t.Fatalf("bind to fleet: %v", err)
	}

	err := r.UpdateBindingWithScope(ctx, moving.ID, 100, true, &BindingScopeChange{})
	if !errors.Is(err, ErrFirewallBindingExists) {
		t.Fatalf("moving onto an occupied scope returned %v, want ErrFirewallBindingExists", err)
	}
}

// Sets had no CRUD at all until this change: the model, the validation and the
// renderer all supported them, and the only way to create one was to write SQL.
// Onboarding REFUSES without a management set, so automated onboarding could not
// be completed by an operator through the API.
func TestSetsAndTheirElementsRoundTrip(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	set := &tenant.FirewallSet{
		Name: "allowed-" + uuid.NewString()[:8], Family: tenant.FamilyIPv4,
		Purpose: tenant.SetPurposeManagement,
	}
	if err := r.CreateSet(ctx, set); err != nil {
		t.Fatalf("create set: %v", err)
	}

	for _, v := range []string{"192.0.2.10", "10.10.10.0/24"} {
		if err := r.AddSetElement(ctx, &tenant.FirewallSetElement{SetID: set.ID, Value: v}); err != nil {
			t.Fatalf("add element %s: %v", v, err)
		}
	}

	// Elements come back with the set. A set without its contents cannot answer
	// any question worth asking, and an empty one would be indistinguishable
	// from an unloaded one.
	got, err := r.GetSet(ctx, set.ID)
	if err != nil {
		t.Fatalf("get set: %v", err)
	}
	if len(got.Elements) != 2 {
		t.Fatalf("want 2 elements loaded with the set, got %d", len(got.Elements))
	}

	// Scoped by set as well as by element id, so an id from one set cannot
	// remove a row from another.
	other := &tenant.FirewallSet{Name: "other-" + uuid.NewString()[:8], Family: tenant.FamilyIPv4}
	if err := r.CreateSet(ctx, other); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteSetElement(ctx, other.ID, got.Elements[0].ID); !errors.Is(err, ErrFirewallSetElemNotFound) {
		t.Fatalf("an element id from another set must not delete: %v", err)
	}
	if err := r.DeleteSetElement(ctx, set.ID, got.Elements[0].ID); err != nil {
		t.Fatalf("delete element: %v", err)
	}

	if err := r.DeleteSet(ctx, set.ID); err != nil {
		t.Fatalf("delete set: %v", err)
	}
	if _, err := r.GetSet(ctx, set.ID); !errors.Is(err, ErrFirewallSetNotFound) {
		t.Fatalf("want not-found after delete, got %v", err)
	}
}

// A set a rule still points at must not be deletable.
//
// nft rejects a ruleset naming a set it has never heard of, and it rejects the
// WHOLE file — so a dangling reference leaves the host with no firewall at all
// rather than one missing a line. The database enforces it via
// constraint:OnDelete:RESTRICT rather than a check that could be forgotten.
func TestASetInUseCannotBeDeleted(t *testing.T) {
	db := firewallDB(t)
	r := NewFirewallRepoGorm(db)
	ctx := context.Background()

	set := &tenant.FirewallSet{Name: "inuse-" + uuid.NewString()[:8], Family: tenant.FamilyIPv4}
	if err := r.CreateSet(ctx, set); err != nil {
		t.Fatal(err)
	}
	rs := &tenant.FirewallRuleset{Name: "rs-" + uuid.NewString()[:8]}
	if err := r.CreateRuleset(ctx, rs); err != nil {
		t.Fatal(err)
	}
	rule := &tenant.FirewallRule{
		RulesetID: rs.ID, Name: "ssh from the whitelist", Enabled: true,
		Family: "inet", TableName: "filter", ChainName: "input",
		Protocol: tenant.TCP, Action: tenant.Accept, DstPort: 22,
		SrcSetID: &set.ID,
	}
	if err := r.CreateRule(ctx, rule); err != nil {
		t.Fatalf("create rule referencing the set: %v", err)
	}

	if err := r.DeleteSet(ctx, set.ID); err == nil {
		t.Fatal("deleting a set a rule still references must be refused")
	}
}
