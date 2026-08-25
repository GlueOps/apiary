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
// Ties resolve by age, then by id. Age is what a person would guess, and the id
// is there so that two rows created in the same transaction — which share a
// timestamp — still have exactly one order. A remaining tie would mean the
// digest changes between renders of unchanged data, and every bee in the hive
// would re-apply a ruleset that did not change.
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
	slices.SortStableFunc(active, func(a, b FirewallBinding) int {
		if c := cmp.Compare(a.Priority, b.Priority); c != 0 {
			return c
		}
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return cmp.Compare(a.ID.String(), b.ID.String())
	})

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
