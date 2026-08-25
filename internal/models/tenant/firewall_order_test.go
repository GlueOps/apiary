package tenant

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ids that sort in a known order, so a test can tell an id tiebreak from luck.
func idAt(n byte) uuid.UUID {
	var u uuid.UUID
	u[15] = n
	return u
}

func names(rules []FirewallRule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		out = append(out, r.Name)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestOrderedRulesIsTwoLevel(t *testing.T) {
	base := uuid.New()
	dc := uuid.New()
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	bindings := []FirewallBinding{
		// Deliberately supplied out of priority order.
		{ID: idAt(2), RulesetID: dc, Priority: 20, Enabled: true, CreatedAt: t0},
		{ID: idAt(1), RulesetID: base, Priority: 10, Enabled: true, CreatedAt: t0},
	}
	rules := map[uuid.UUID][]FirewallRule{
		base: {
			{ID: idAt(11), Name: "base-30", Position: 30, Enabled: true, CreatedAt: t0},
			{ID: idAt(10), Name: "base-10", Position: 10, Enabled: true, CreatedAt: t0},
		},
		dc: {
			{ID: idAt(20), Name: "dc-10", Position: 10, Enabled: true, CreatedAt: t0},
		},
	}

	// Every base rule precedes every dc rule, because binding priority outranks
	// rule position. A single-level sort would interleave base-30 after dc-10.
	eq(t, names(OrderedRules(bindings, rules)), []string{"base-10", "base-30", "dc-10"})
}

func TestOrderedRulesDropsWhatIsDisabled(t *testing.T) {
	on, off := uuid.New(), uuid.New()
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	bindings := []FirewallBinding{
		{ID: idAt(1), RulesetID: on, Priority: 10, Enabled: true, CreatedAt: t0},
		// A disabled binding takes its whole ruleset out, however the rules
		// inside it are flagged.
		{ID: idAt(2), RulesetID: off, Priority: 20, Enabled: false, CreatedAt: t0},
	}
	rules := map[uuid.UUID][]FirewallRule{
		on: {
			{ID: idAt(10), Name: "kept", Position: 10, Enabled: true, CreatedAt: t0},
			{ID: idAt(11), Name: "disabled-rule", Position: 20, Enabled: false, CreatedAt: t0},
		},
		off: {
			{ID: idAt(20), Name: "in-disabled-ruleset", Position: 10, Enabled: true, CreatedAt: t0},
		},
	}

	eq(t, names(OrderedRules(bindings, rules)), []string{"kept"})
}

// The property the whole design rests on: unchanged data must render to one
// sequence, whatever order the database handed the rows back in. If it does not,
// the digest changes on its own and every bee in the hive re-applies a ruleset
// that did not change.
func TestOrderedRulesIsDeterministicUnderInputPermutation(t *testing.T) {
	rs := uuid.New()
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	// Same position AND same timestamp — rows written in one transaction. Only
	// the id can separate them.
	rules := []FirewallRule{
		{ID: idAt(3), Name: "c", Position: 10, Enabled: true, CreatedAt: t0},
		{ID: idAt(1), Name: "a", Position: 10, Enabled: true, CreatedAt: t0},
		{ID: idAt(2), Name: "b", Position: 10, Enabled: true, CreatedAt: t0},
	}
	bindings := []FirewallBinding{{ID: idAt(1), RulesetID: rs, Priority: 10, Enabled: true, CreatedAt: t0}}

	want := names(OrderedRules(bindings, map[uuid.UUID][]FirewallRule{rs: rules}))
	eq(t, want, []string{"a", "b", "c"})

	// Every permutation of the same rows must produce that same sequence.
	perms := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for _, p := range perms {
		shuffled := []FirewallRule{rules[p[0]], rules[p[1]], rules[p[2]]}
		eq(t, names(OrderedRules(bindings, map[uuid.UUID][]FirewallRule{rs: shuffled})), want)
	}
}

// Age separates rules at the same position, before the id is consulted. Without
// it, inserting a rule at an occupied position would place it by id — which is
// random, so a new rule would land above or below its neighbour by chance.
func TestOrderedRulesBreaksPositionTiesByAge(t *testing.T) {
	rs := uuid.New()
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	rules := []FirewallRule{
		// Higher id, but written first.
		{ID: idAt(9), Name: "older", Position: 10, Enabled: true, CreatedAt: t0},
		{ID: idAt(1), Name: "newer", Position: 10, Enabled: true, CreatedAt: t0.Add(time.Hour)},
	}
	bindings := []FirewallBinding{{ID: idAt(1), RulesetID: rs, Priority: 10, Enabled: true, CreatedAt: t0}}

	eq(t, names(OrderedRules(bindings, map[uuid.UUID][]FirewallRule{rs: rules})), []string{"older", "newer"})
}

func TestOrderedRulesHandlesABindingWithNoRules(t *testing.T) {
	empty, full := uuid.New(), uuid.New()
	t0 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	bindings := []FirewallBinding{
		{ID: idAt(1), RulesetID: empty, Priority: 10, Enabled: true, CreatedAt: t0},
		{ID: idAt(2), RulesetID: full, Priority: 20, Enabled: true, CreatedAt: t0},
	}
	rules := map[uuid.UUID][]FirewallRule{
		full: {{ID: idAt(10), Name: "only", Position: 10, Enabled: true, CreatedAt: t0}},
	}

	eq(t, names(OrderedRules(bindings, rules)), []string{"only"})
}

// OrderedRules is what actually renders the policy a host enforces, and it has
// to agree with CompareBindingsForRender — the comparator every list the
// operator reads is ordered by. It once carried its own copy of the ordering
// that omitted the scope-rank tie-break, so a year-old org-wide binding rendered
// above a swarm binding made this morning while the UI showed the reverse. Every
// binding defaults to priority 100, so this tie is the common case, not an edge.
func TestOrderedRulesUsesTheOneBindingComparator(t *testing.T) {
	rsOrg, rsSwarm := uuid.New(), uuid.New()
	hive, swarm := uuid.New(), uuid.New()
	old := time.Now().UTC().Add(-365 * 24 * time.Hour)
	recent := time.Now().UTC()

	bindings := []FirewallBinding{
		// Same priority — the default — so only the tie-break separates them.
		{ID: uuid.New(), RulesetID: rsOrg, Priority: 100, Enabled: true, CreatedAt: old},
		{ID: uuid.New(), RulesetID: rsSwarm, Priority: 100, Enabled: true, CreatedAt: recent,
			HiveID: &hive, SwarmID: &swarm},
	}
	rules := map[uuid.UUID][]FirewallRule{
		rsOrg:   {{ID: uuid.New(), Name: "org", Enabled: true, Position: 1}},
		rsSwarm: {{ID: uuid.New(), Name: "swarm", Enabled: true, Position: 1}},
	}

	// The more specific binding renders first, exactly as CompareBindingsForRender
	// and the repository's ORDER BY CASE both rank it.
	eq(t, names(OrderedRules(bindings, rules)), []string{"swarm", "org"})

	// And stated as the invariant rather than one example: whatever the comparator
	// says about two bindings, the rendered order must say the same.
	sorted := slices.Clone(bindings)
	slices.SortStableFunc(sorted, CompareBindingsForRender)
	want := make([]string, 0, len(sorted))
	for _, b := range sorted {
		for _, r := range rules[b.RulesetID] {
			want = append(want, r.Name)
		}
	}
	eq(t, names(OrderedRules(bindings, rules)), want)
}

// Priority still outranks specificity. The tie-break must not become the primary
// key, or an org-wide "drop bogons" ruleset can no longer be put above
// everything — which is the reason priority is an explicit operator knob.
func TestOrderedRulesKeepsPriorityAboveSpecificity(t *testing.T) {
	rsOrg, rsSwarm := uuid.New(), uuid.New()
	hive, swarm := uuid.New(), uuid.New()
	now := time.Now().UTC()

	bindings := []FirewallBinding{
		{ID: uuid.New(), RulesetID: rsSwarm, Priority: 20, Enabled: true, CreatedAt: now,
			HiveID: &hive, SwarmID: &swarm},
		{ID: uuid.New(), RulesetID: rsOrg, Priority: 10, Enabled: true, CreatedAt: now},
	}
	rules := map[uuid.UUID][]FirewallRule{
		rsOrg:   {{ID: uuid.New(), Name: "org", Enabled: true, Position: 1}},
		rsSwarm: {{ID: uuid.New(), Name: "swarm", Enabled: true, Position: 1}},
	}
	eq(t, names(OrderedRules(bindings, rules)), []string{"org", "swarm"})
}
