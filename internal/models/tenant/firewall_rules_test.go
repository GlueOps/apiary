package tenant

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Own helper rather than sharing firewall_order_test.go's, so these cases do not
// silently change meaning when that file's fixtures are edited for another
// reason.
func bindingID(n byte) uuid.UUID {
	var u uuid.UUID
	u[15] = n
	return u
}

func ptrID(u uuid.UUID) *uuid.UUID { return &u }

// The trap this method exists to avoid: a swarm-scoped binding carries hive_id
// TOO, because the composite foreign key needs it. Anything testing hive_id
// first therefore reads every swarm binding as hive-scoped, and the specificity
// ordering silently stops working while every field still looks right.
func TestScopeRankReadsSwarmBeforeHive(t *testing.T) {
	hive := uuid.New()
	swarm := uuid.New()

	cases := []struct {
		name string
		b    FirewallBinding
		want int
	}{
		{"org-wide", FirewallBinding{}, ScopeRankOrg},
		{"hive", FirewallBinding{HiveID: &hive}, ScopeRankHive},
		{"swarm carries its hive as well", FirewallBinding{HiveID: &hive, SwarmID: &swarm}, ScopeRankSwarm},
	}
	for _, c := range cases {
		if got := c.b.ScopeRank(); got != c.want {
			t.Fatalf("%s: ScopeRank() = %d, want %d", c.name, got, c.want)
		}
	}

	if ScopeRankSwarm >= ScopeRankHive || ScopeRankHive >= ScopeRankOrg {
		t.Fatalf("ranks are ordered %d/%d/%d; most specific must sort FIRST because nft "+
			"stops at the first terminal verdict, so an org-wide drop ranked first would "+
			"shadow the swarm ruleset meant to override it",
			ScopeRankSwarm, ScopeRankHive, ScopeRankOrg)
	}
}

// sortedNames applies the render comparator and reports the resulting order by
// the bindings' priorities and scopes, which is what the assertions are about.
func sortedIDs(bindings []FirewallBinding) []uuid.UUID {
	out := slices.Clone(bindings)
	slices.SortStableFunc(out, CompareBindingsForRender)
	ids := make([]uuid.UUID, 0, len(out))
	for _, b := range out {
		ids = append(ids, b.ID)
	}
	return ids
}

func assertOrder(t *testing.T, got []uuid.UUID, want []uuid.UUID, why string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v — %s", got, want, why)
	}
}

// The operator's explicit knob must still win. Letting scope specificity outrank
// priority would remove the ability to put an org-wide "drop bogons" ruleset
// above everything, which is a legitimate thing to want.
func TestPriorityOutranksSpecificity(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	hive, swarm := uuid.New(), uuid.New()

	org := FirewallBinding{ID: bindingID(1), Priority: 10, Enabled: true, CreatedAt: t0}
	swarmBinding := FirewallBinding{
		ID: bindingID(2), HiveID: &hive, SwarmID: &swarm,
		Priority: 20, Enabled: true, CreatedAt: t0,
	}

	assertOrder(t, sortedIDs([]FirewallBinding{swarmBinding, org}),
		[]uuid.UUID{org.ID, swarmBinding.ID},
		"an org-wide binding at priority 10 must still render before a swarm binding at 20")
}

// Ties are the COMMON case: every binding defaults to priority 100. The previous
// tie-break was age alone, which let an org-wide binding created a year ago
// outrank a swarm binding made this morning purely by accident of when it was
// written.
func TestSpecificityBreaksPriorityTiesAgainstAge(t *testing.T) {
	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	hiveA, swarmA := uuid.New(), uuid.New()

	// The org binding is both OLDER and has the lower id, so age or id alone
	// would put it first.
	org := FirewallBinding{ID: bindingID(1), Priority: 100, Enabled: true, CreatedAt: old}
	hive := FirewallBinding{ID: bindingID(2), HiveID: &hiveA, Priority: 100, Enabled: true, CreatedAt: old}
	swarm := FirewallBinding{
		ID: bindingID(3), HiveID: &hiveA, SwarmID: &swarmA,
		Priority: 100, Enabled: true, CreatedAt: recent,
	}

	assertOrder(t, sortedIDs([]FirewallBinding{org, hive, swarm}),
		[]uuid.UUID{swarm.ID, hive.ID, org.ID},
		"at equal priority the most specific scope must render first, whatever the ages are")
}

// Unchanged data must render to one sequence whatever order the database handed
// the rows back in, or the snapshot digest changes on its own and every bee
// re-applies a ruleset that did not change.
func TestRenderComparatorIsTotalAndStable(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	hive := uuid.New()

	// Same priority, same scope, same timestamp — rows written in one
	// transaction. Only the id can separate them.
	bindings := []FirewallBinding{
		{ID: bindingID(3), HiveID: ptrID(hive), Priority: 100, CreatedAt: t0},
		{ID: bindingID(1), HiveID: ptrID(hive), Priority: 100, CreatedAt: t0},
		{ID: bindingID(2), HiveID: ptrID(hive), Priority: 100, CreatedAt: t0},
	}
	want := []uuid.UUID{bindingID(1), bindingID(2), bindingID(3)}

	for _, p := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		shuffled := []FirewallBinding{bindings[p[0]], bindings[p[1]], bindings[p[2]]}
		assertOrder(t, sortedIDs(shuffled), want,
			"every permutation of the same rows must produce the same sequence")
	}
}

// Age still separates two bindings of the SAME scope, before the id is
// consulted. Without it a newly bound ruleset would land above or below its
// peer by chance of uuid.
func TestRenderComparatorBreaksScopeTiesByAge(t *testing.T) {
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	hive := uuid.New()

	older := FirewallBinding{ID: bindingID(9), HiveID: &hive, Priority: 100, CreatedAt: t0}
	newer := FirewallBinding{ID: bindingID(1), HiveID: &hive, Priority: 100, CreatedAt: t0.Add(time.Hour)}

	assertOrder(t, sortedIDs([]FirewallBinding{newer, older}),
		[]uuid.UUID{older.ID, newer.ID},
		"same priority and same scope must fall back to age before id")
}
