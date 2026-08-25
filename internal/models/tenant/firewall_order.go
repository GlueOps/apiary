package tenant

import (
	"cmp"
	"slices"

	"github.com/google/uuid"
)

// OrderedRules flattens a hive's bound rulesets into the exact sequence they
// render in.
//
// This is a function rather than an ORDER BY inside a repository because in a
// firewall the order IS the policy: an accept above a drop and the same two
// reversed are different rules about who may reach the host. It is worth being
// able to test that directly, without a database.
//
// Two levels, and both are needed. Binding priority orders rulesets against each
// other; rule position orders rules inside one. With only the second, two bound
// rulesets interleave arbitrarily and the same rows render as different
// policies on different runs.
//
// Binding order is CompareBindingsForRender's, in full — priority, then scope
// specificity, then age, then id. Rule order inside one ruleset resolves ties by
// age then id for the same reason: two rows created in the same transaction
// share a timestamp, and a remaining tie would change the rendered digest
// between renders of unchanged data, making every bee in the hive re-apply a
// ruleset that did not change.
//
// Disabled bindings and disabled rules are dropped here rather than filtered by
// the caller, so that "enabled" means the same thing everywhere it is asked.
func OrderedRules(bindings []FirewallBinding, rulesByRuleset map[uuid.UUID][]FirewallRule) []FirewallRule {
	active := make([]FirewallBinding, 0, len(bindings))
	for _, b := range bindings {
		if b.Enabled {
			active = append(active, b)
		}
	}
	// CompareBindingsForRender, not a comparator written out again here. This
	// function is what actually renders the policy, and it used to carry its own
	// copy of the ordering that was missing the scope-rank tie-break the
	// repository's ORDER BY and CompareBindingsForRender both apply. Since every
	// binding defaults to priority 100, that tie-break decides the common case:
	// the rendered policy put a year-old org-wide binding above a swarm binding
	// made this morning, while every list the operator reads ordered them the
	// other way round. Two orderings meant to agree, expressed twice.
	slices.SortStableFunc(active, CompareBindingsForRender)

	out := make([]FirewallRule, 0)
	for _, b := range active {
		rules := make([]FirewallRule, 0, len(rulesByRuleset[b.RulesetID]))
		for _, r := range rulesByRuleset[b.RulesetID] {
			if r.Enabled {
				rules = append(rules, r)
			}
		}
		slices.SortStableFunc(rules, func(x, y FirewallRule) int {
			if c := cmp.Compare(x.Position, y.Position); c != 0 {
				return c
			}
			if c := x.CreatedAt.Compare(y.CreatedAt); c != 0 {
				return c
			}
			return cmp.Compare(x.ID.String(), y.ID.String())
		})
		out = append(out, rules...)
	}
	return out
}
