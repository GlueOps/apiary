package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// ResolveHiveRenderInput assembles everything one fleet's ruleset renders from.
//
// It takes a *gorm.DB rather than a repository because the publish path calls it
// INSIDE its transaction. Rules that changed between resolving and writing would
// make the stored snapshot a body that no set of rows ever produced, and a
// repository handle bound to the outer connection would resolve outside the
// transaction while looking as if it were inside it.
//
// FLEET-SCOPED, not host-scoped. The snapshot is one body per fleet, so the
// scope is (hive, no swarm): org-wide and hive-wide bindings, and no swarm ones.
// Rendering per swarm would mean a snapshot per swarm and a revision number that
// no longer identifies one artifact — and Hive.FirewallRevision is a single
// number by design.
func ResolveHiveRenderInput(
	ctx context.Context, tx *gorm.DB, hiveID uuid.UUID,
) (tenant.RenderInput, error) {
	var out tenant.RenderInput

	var bindings []tenant.FirewallBinding
	// Org-wide (hive NULL) and this hive's own, both with no swarm. A host in a
	// swarm receives more than this; what a FLEET publishes is the part every
	// host in it shares.
	if err := tx.WithContext(ctx).
		Where("swarm_id IS NULL").
		Where("hive_id IS NULL OR hive_id = ?", hiveID).
		Where("enabled IS TRUE").
		Find(&bindings).Error; err != nil {
		return out, fmt.Errorf("resolve: bindings: %w", err)
	}
	if len(bindings) == 0 {
		return out, nil
	}

	// Sorted with the ONE comparator, not a second ORDER BY. Two orderings meant
	// to agree and expressed in two places do not stay in agreement, and here the
	// consequence is an operator reading a different policy from the one the host
	// enforces.
	sortBindings(bindings)

	seen := make(map[uuid.UUID]bool, len(bindings))
	for _, b := range bindings {
		if seen[b.RulesetID] {
			// The same ruleset bound at two scopes renders once, at its earliest
			// position. Rendering it twice would duplicate every rule in it, and a
			// duplicated terminal verdict is dead weight at best and confusing at
			// worst when someone reads the host's ruleset back.
			continue
		}
		seen[b.RulesetID] = true

		var rules []tenant.FirewallRule
		if err := tx.WithContext(ctx).
			Where("ruleset_id = ? AND enabled IS TRUE", b.RulesetID).
			Order("position, created_at, id").
			Find(&rules).Error; err != nil {
			return out, fmt.Errorf("resolve: rules for ruleset %s: %w", b.RulesetID, err)
		}
		out.Rules = append(out.Rules, rules...)
	}

	// Sets are loaded for the rules that actually reference one, rather than
	// every set in the organization: an unreferenced set declared in the output
	// is a set nft has to parse for nothing, and one of the wrong family would
	// fail the whole file.
	needed := map[uuid.UUID]bool{}
	for _, r := range out.Rules {
		for _, id := range []*uuid.UUID{r.SrcSetID, r.DstSetID} {
			if id != nil {
				needed[*id] = true
			}
		}
	}
	if len(needed) == 0 {
		return out, nil
	}

	ids := make([]uuid.UUID, 0, len(needed))
	for id := range needed {
		ids = append(ids, id)
	}
	var sets []tenant.FirewallSet
	if err := tx.WithContext(ctx).Where("id IN ?", ids).Find(&sets).Error; err != nil {
		return out, fmt.Errorf("resolve: sets: %w", err)
	}
	out.Sets = make(map[string]tenant.FirewallSet, len(sets))
	for _, s := range sets {
		out.Sets[s.ID.String()] = s
	}

	var elements []tenant.FirewallSetElement
	if err := tx.WithContext(ctx).Where("set_id IN ?", ids).Find(&elements).Error; err != nil {
		return out, fmt.Errorf("resolve: set elements: %w", err)
	}
	out.Elements = make(map[string][]tenant.FirewallSetElement, len(sets))
	for _, e := range elements {
		k := e.SetID.String()
		out.Elements[k] = append(out.Elements[k], e)
	}
	return out, nil
}

// sortBindings orders bound rulesets for rendering.
func sortBindings(bs []tenant.FirewallBinding) {
	// A simple insertion sort keeps this using CompareBindingsForRender directly
	// rather than wrapping it, and the slice is a handful of rows.
	for i := 1; i < len(bs); i++ {
		for j := i; j > 0 && tenant.CompareBindingsForRender(bs[j], bs[j-1]) < 0; j-- {
			bs[j], bs[j-1] = bs[j-1], bs[j]
		}
	}
}
