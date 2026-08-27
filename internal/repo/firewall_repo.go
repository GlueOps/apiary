package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrFirewallRulesetNotFound = errors.New("firewall: no such ruleset")
	ErrFirewallRuleNotFound    = errors.New("firewall: no such rule")
	ErrFirewallBindingNotFound = errors.New("firewall: no such binding")
	ErrFirewallRulesetTaken    = errors.New("firewall: a ruleset with that name already exists")
	ErrFirewallBindingExists   = errors.New("firewall: that ruleset is already bound there")
	ErrFirewallHiveNotFound    = errors.New("firewall: no such fleet")
	ErrFirewallSwarmNotFound   = errors.New("firewall: no such swarm in that fleet")
	ErrFirewallSetNotFound     = errors.New("firewall: no such set")
	ErrFirewallSetElemNotFound = errors.New("firewall: no such element in that set")
	// ErrFirewallSetManaged refuses a hand-edit the next convergence would
	// silently revert. An edit that disappears without explanation is worse
	// than one that is refused.
	ErrFirewallSetManaged = errors.New("firewall: that set is maintained by the control plane")

	// ErrFirewallSwarmNeedsHive is the CHECK constraint, not a foreign key. A
	// swarm-scoped binding carries hive_id as well as swarm_id, and a row with
	// only swarm_id set is the one combination the schema forbids: it would be
	// indexed as swarm-scoped while reading as org-wide to anything that tests
	// hive_id first.
	ErrFirewallSwarmNeedsHive = errors.New("firewall: a swarm-scoped binding must name its fleet too")

	// ErrFirewallSetInUse is a RESTRICT violation, and Postgres raises those as
	// SQLSTATE 23001 (restrict_violation) — NOT 23503 (foreign_key_violation).
	// A handler matching only on 23503 would let this surface as a 500, telling
	// an operator the system is broken when it correctly refused to orphan the
	// rules pointing at the set they tried to delete.
	ErrFirewallSetInUse = errors.New("firewall: rules still reference this set")
)

// FirewallRepoGorm is scoped to ONE organization, because the handle it holds
// is that organization's database. There is no org column to filter on and no
// org argument to pass, which is what makes cross-tenant access unreachable
// rather than merely checked.
type FirewallRepoGorm struct{ db *gorm.DB }

func NewFirewallRepoGorm(db *gorm.DB) *FirewallRepoGorm { return &FirewallRepoGorm{db: db} }

// ---- rulesets --------------------------------------------------------------

func (r *FirewallRepoGorm) ListRulesets(ctx context.Context) ([]tenant.FirewallRuleset, error) {
	var out []tenant.FirewallRuleset
	err := r.db.WithContext(ctx).Order("name ASC").Find(&out).Error
	return out, err
}

func (r *FirewallRepoGorm) GetRuleset(ctx context.Context, id uuid.UUID) (*tenant.FirewallRuleset, error) {
	var rs tenant.FirewallRuleset
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFirewallRulesetNotFound
		}
		return nil, err
	}
	return &rs, nil
}

func (r *FirewallRepoGorm) CreateRuleset(ctx context.Context, rs *tenant.FirewallRuleset) error {
	if err := r.db.WithContext(ctx).Create(rs).Error; err != nil {
		return translateFirewallConflict(err)
	}
	return nil
}

func (r *FirewallRepoGorm) UpdateRuleset(ctx context.Context, id uuid.UUID, name, description string) (*tenant.FirewallRuleset, error) {
	res := r.db.WithContext(ctx).Model(&tenant.FirewallRuleset{}).
		Where("id = ?", id).
		Updates(map[string]any{"name": name, "description": description})
	if res.Error != nil {
		return nil, translateFirewallConflict(res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, ErrFirewallRulesetNotFound
	}
	return r.GetRuleset(ctx, id)
}

// DeleteRuleset removes a ruleset and, by cascade, its rules and bindings.
//
// The cascade is the point rather than an oversight: a ruleset with no rules is
// meaningless, and a binding to a ruleset that no longer exists would render
// nothing while still appearing in the list as though a fleet were covered.
func (r *FirewallRepoGorm) DeleteRuleset(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&tenant.FirewallRuleset{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFirewallRulesetNotFound
	}
	return nil
}

// ---- rules -----------------------------------------------------------------

// ListRules returns one ruleset's rules in the order they render within it.
//
// The ORDER BY matches tenant.OrderedRules exactly. Two orderings that are
// supposed to agree and are written in two places will not stay in agreement,
// so a reader looking at this list must be seeing what the renderer will do —
// otherwise the UI shows one policy and the host enforces another.
func (r *FirewallRepoGorm) ListRules(ctx context.Context, rulesetID uuid.UUID) ([]tenant.FirewallRule, error) {
	var out []tenant.FirewallRule
	err := r.db.WithContext(ctx).
		Where("ruleset_id = ?", rulesetID).
		Order("position ASC, created_at ASC, id ASC").
		Find(&out).Error
	return out, err
}

func (r *FirewallRepoGorm) GetRule(ctx context.Context, id uuid.UUID) (*tenant.FirewallRule, error) {
	var rule tenant.FirewallRule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFirewallRuleNotFound
		}
		return nil, err
	}
	return &rule, nil
}

// CreateRule writes the rule, then forces Enabled if it was created disabled.
//
// The second statement is not belt-and-braces. GORM omits a zero value from the
// INSERT when the field carries a `default:` tag, and FirewallRule.Enabled is
// `default:true` — so `Create` with Enabled=false stores TRUE, and a rule an
// operator deliberately disabled is live from the moment it is written. Verified
// against Postgres; the same defect silently failed to cordon a hypervisor
// elsewhere in this codebase, and here it is a packet filter.
//
// Create-then-update rather than Create-with-Select: a `Select("*")` also forces
// the primary key, which fights the gen_random_uuid() default, and a field-list
// Select leaves GORM with no id to read back. Both were tried.
func (r *FirewallRepoGorm) CreateRule(ctx context.Context, rule *tenant.FirewallRule) error {
	// Captured BEFORE Create. GORM writes the INSERT's RETURNING values back into
	// the struct, so after Create `rule.Enabled` reads TRUE — the column default it
	// just applied — and a guard that consulted the struct would short-circuit and
	// never correct anything. The struct cannot be asked what the caller wanted.
	wanted := rule.Enabled

	// The address columns carry `default:'0.0.0.0/0'` and are the same trap.
	//
	// A rule that names a SET carries no literal on that side — the service layer
	// leaves it empty deliberately, because a literal alongside a set is refused.
	// Create then omits the empty string and Postgres writes 0.0.0.0/0, so the
	// stored row asserts a source constraint the renderer does not enforce: it
	// prefers the set and ignores the literal. Harmless to the host and wrong
	// everywhere it is read, which includes the API response a client edits from.
	wantedSrc, wantedDst := rule.SrcCidr, rule.DstCidr

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rule).Error; err != nil {
			return translateFirewallConflict(err)
		}

		// Only the fields whose wanted value is the zero one, because those are
		// exactly the fields Create dropped.
		fix := map[string]any{}
		if !wanted {
			fix["enabled"] = false
		}
		if wantedSrc == "" {
			fix["src_cidr"] = ""
		}
		if wantedDst == "" {
			fix["dst_cidr"] = ""
		}
		if len(fix) == 0 {
			return nil
		}
		if err := tx.Model(&tenant.FirewallRule{}).Where("id = ?", rule.ID).
			Updates(fix).Error; err != nil {
			return err
		}
		// Keep the returned struct honest: it would otherwise say enabled, or
		// carry an address the row does not, and the caller renders the struct.
		rule.Enabled = wanted
		rule.SrcCidr = wantedSrc
		rule.DstCidr = wantedDst
		return nil
	})
}

// UpdateRule replaces a rule wholesale.
//
// Whole-row rather than a patch: a firewall rule is one statement, and applying
// half an edit — the new port with the old action — produces a rule nobody
// intended and that no review would have approved.
func (r *FirewallRepoGorm) UpdateRule(ctx context.Context, rule *tenant.FirewallRule) error {
	res := r.db.WithContext(ctx).Model(&tenant.FirewallRule{}).
		Where("id = ?", rule.ID).
		Select("name", "enabled", "family", "table_name", "chain_name", "position",
			"src_cidr", "dst_cidr", "protocol", "src_port", "src_port_end",
			"dst_port", "dst_port_end", "in_iface", "out_iface", "action", "comment").
		Updates(rule)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFirewallRuleNotFound
	}
	return nil
}

func (r *FirewallRepoGorm) DeleteRule(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&tenant.FirewallRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFirewallRuleNotFound
	}
	return nil
}

// ---- bindings --------------------------------------------------------------

// bindingRenderOrder is the SQL half of tenant.CompareBindingsForRender, and the
// CASE must stay identical to tenant.FirewallBinding.ScopeRank: swarm 0, hive 1,
// org 2, most specific rendering FIRST because nft stops at the first terminal
// verdict.
//
// It is a constant rather than a literal at each call site because the two
// binding queries below and the Go comparator all have to agree; three copies of
// an ORDER BY do not stay in agreement, and disagreement here means the operator
// reads one policy from the list while the host enforces another.
//
// swarm_id is tested before hive_id for the same reason ScopeRank does: a
// swarm-scoped row carries hive_id too, so `hive_id IS NOT NULL` no longer means
// "hive-scoped".
const bindingRenderOrder = "priority ASC, " +
	"(CASE WHEN swarm_id IS NOT NULL THEN 0 WHEN hive_id IS NOT NULL THEN 1 ELSE 2 END) ASC, " +
	"created_at ASC, id ASC"

// ListBindings returns every binding in RENDER order — org-wide, hive-scoped and
// swarm-scoped together, lowest priority first — because that is the order an
// operator needs to reason about, and any other order invites reading the policy
// backwards.
func (r *FirewallRepoGorm) ListBindings(ctx context.Context) ([]tenant.FirewallBinding, error) {
	var out []tenant.FirewallBinding
	err := r.db.WithContext(ctx).
		Preload("Ruleset").
		Order(bindingRenderOrder).
		Find(&out).Error
	return out, err
}

// BindingsForScope returns everything that applies to a host at one scope: the
// org-wide bindings, its fleet's, and its swarm's.
//
// The org-wide arm is unconditional, and that is not a convenience. A host that
// has left one fleet and not yet joined the next matches neither of the other
// two arms, and returning nothing would render it an empty ruleset at its most
// exposed moment.
//
// THE HIVE ARM MUST SAY `swarm_id IS NULL`. A swarm-scoped binding carries
// hive_id as well, so the older `hive_id IS NULL OR hive_id = ?` now matches
// every swarm binding in the cluster — a host in fleet-nonprod would receive
// fleet-prod's rules, which is the exact failure this scope was added to
// prevent. Leaning on `hive_id = NULL` never being true would also work, but it
// hides a security-relevant policy behind SQL trivia.
//
// The arms are built here rather than relying on that trivia so an absent scope
// is an absent clause: a nil SwarmID does not narrow, it simply contributes no
// arm.
func (r *FirewallRepoGorm) BindingsForScope(ctx context.Context, scope tenant.BindingScope) ([]tenant.FirewallBinding, error) {
	arms := []string{"(hive_id IS NULL AND swarm_id IS NULL)"}
	args := make([]any, 0, 2)
	if scope.HiveID != nil {
		arms = append(arms, "(hive_id = ? AND swarm_id IS NULL)")
		args = append(args, *scope.HiveID)
	}
	if scope.SwarmID != nil {
		arms = append(arms, "(swarm_id = ?)")
		args = append(args, *scope.SwarmID)
	}

	var out []tenant.FirewallBinding
	err := r.db.WithContext(ctx).
		Preload("Ruleset").
		Where(strings.Join(arms, " OR "), args...).
		Order(bindingRenderOrder).
		Find(&out).Error
	return out, err
}

// BindingsForHive answers for a host whose swarm is not known.
//
// A shim over BindingsForScope, kept only while the service layer still passes a
// bare hive. It is NOT the same as passing a swarm of nil by accident: a host
// genuinely in no swarm and a host whose swarm we failed to read both belong
// here, and both correctly get hive and org scope rather than nothing.
func (r *FirewallRepoGorm) BindingsForHive(ctx context.Context, hiveID *uuid.UUID) ([]tenant.FirewallBinding, error) {
	return r.BindingsForScope(ctx, tenant.BindingScope{HiveID: hiveID})
}

// ScopeForBee derives where an agent's machine sits, for firewall resolution.
//
// The pair comes from ONE row — the hypervisor carrying this bee — because hive
// and swarm must agree with each other. Bee no longer carries a hive at all
// (that was the two-answers-to-one-question bug the machine-record shape
// removed), so there is nothing to mix it with here; the rule is written down
// anyway because the tempting reintroduction is to fall back to some other
// source for the hive when only the swarm is missing.
//
// NO HYPERVISOR ROW IS NORMAL, NOT AN ERROR. A machine mid-onboard, or one whose
// agent enrolled before the cluster reported it, has none — and this returns the
// empty scope, which resolves to the org-wide bindings. Returning an error
// instead would deny the host any rules, and an absent ruleset means HOLD rather
// than flush, so the host would sit on whatever it last applied with nothing
// reporting why.
func (r *FirewallRepoGorm) ScopeForBee(ctx context.Context, beeID uuid.UUID) (tenant.BindingScope, error) {
	var row struct {
		HiveID  *uuid.UUID
		SwarmID *uuid.UUID
	}
	res := r.db.WithContext(ctx).
		Model(&tenant.Hypervisor{}).
		Select("hive_id", "swarm_id").
		Where("bee_id = ?", beeID).
		Limit(1).
		Scan(&row)
	if res.Error != nil {
		return tenant.BindingScope{}, res.Error
	}
	return tenant.BindingScope{HiveID: row.HiveID, SwarmID: row.SwarmID}, nil
}

// CreateBinding writes the binding, then forces Enabled if it was created
// disabled. Same reason as CreateRule: `default:true` means GORM never sends
// false, so a binding created disabled would apply its ruleset immediately.
func (r *FirewallRepoGorm) CreateBinding(ctx context.Context, b *tenant.FirewallBinding) error {
	wanted := b.Enabled

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(b).Error; err != nil {
			return translateFirewallConflict(err)
		}
		if wanted {
			return nil
		}
		if err := tx.Model(&tenant.FirewallBinding{}).Where("id = ?", b.ID).
			Update("enabled", false).Error; err != nil {
			return err
		}
		b.Enabled = false
		return nil
	})
}

func (r *FirewallRepoGorm) UpdateBinding(ctx context.Context, id uuid.UUID, priority int, enabled bool) error {
	return r.UpdateBindingWithScope(ctx, id, priority, enabled, nil)
}

// BindingScopeChange moves a binding to a different scope.
//
// A distinct type, and passed as a POINTER, so "leave the scope alone" is
// expressible. Without it a caller sending only priority and enabled — which is
// every caller that predates this — would have its binding silently re-scoped to
// org-wide, because both ids would arrive as nil. Widening a binding from one
// fleet to the whole organization is the single worst accident this endpoint
// could have, so the absence of an instruction has to mean absence.
type BindingScopeChange struct {
	HiveID  *uuid.UUID
	SwarmID *uuid.UUID
}

// UpdateBindingWithScope updates a binding, optionally moving its scope.
func (r *FirewallRepoGorm) UpdateBindingWithScope(
	ctx context.Context, id uuid.UUID, priority int, enabled bool, scope *BindingScopeChange,
) error {
	fields := map[string]any{"priority": priority, "enabled": enabled}
	if scope != nil {
		// Written as explicit NULLs rather than omitted. A map value of nil is what
		// GORM turns into NULL; leaving the key out would keep the old id, which is
		// the difference between "move this to org-wide" and "do nothing".
		fields["hive_id"] = scope.HiveID
		fields["swarm_id"] = scope.SwarmID
	}
	res := r.db.WithContext(ctx).Model(&tenant.FirewallBinding{}).
		Where("id = ?", id).
		Updates(fields)
	if res.Error != nil {
		// Translated, because a re-scope collides on exactly the indexes a create
		// does: moving a binding onto a scope where the same ruleset is already
		// bound would render that ruleset twice, duplicating every rule in it. The
		// operator needs "already bound there", not a constraint name.
		return translateFirewallConflict(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrFirewallBindingNotFound
	}
	return nil
}

func (r *FirewallRepoGorm) DeleteBinding(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&tenant.FirewallBinding{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFirewallBindingNotFound
	}
	return nil
}

// HiveExists is how a binding's fleet is checked before the foreign key would
// reject it.
//
// The key is the real guarantee; this exists so the caller gets "no such fleet"
// instead of a constraint violation surfacing as a 500. It needs no
// organization predicate: a hive belonging to someone else is not in this
// database to be found.
func (r *FirewallRepoGorm) HiveExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&tenant.Hive{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

// SwarmExists checks the PAIR, not the swarm id alone, so that "no such swarm"
// and "that swarm is in a different fleet" both refuse here rather than reaching
// the composite foreign key and surfacing as a 500.
//
// Checking the id alone would pass for a swarm belonging to another cluster and
// then fail at the constraint — which is safe, but reports a server fault for
// something the operator typed.
func (r *FirewallRepoGorm) SwarmExists(ctx context.Context, hiveID, swarmID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&tenant.Swarm{}).
		Where("id = ? AND hive_id = ?", swarmID, hiveID).
		Count(&n).Error
	return n > 0, err
}

// translateFirewallConflict turns a unique-violation into the specific refusal
// that caused it.
//
// Matching on the index name rather than the error class: both conflicts here
// are unique violations, and reporting "already exists" for a duplicate binding
// when the real problem was a duplicate ruleset name sends the operator to fix
// the wrong thing.
func translateFirewallConflict(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "idx_firewall_rulesets_name"):
		return ErrFirewallRulesetTaken
	case strings.Contains(msg, "idx_firewall_bindings_hive_ruleset"),
		strings.Contains(msg, "idx_firewall_bindings_swarm_ruleset"),
		strings.Contains(msg, "idx_firewall_bindings_org_ruleset"):
		return ErrFirewallBindingExists
	case strings.Contains(msg, "fk_firewall_bindings_hive"):
		return ErrFirewallHiveNotFound
	// The swarm foreign key is composite, so it also fires when the swarm exists
	// but belongs to a different cluster. Both are "no such swarm in that fleet",
	// which is what SwarmExists checks up front; this is the backstop.
	//
	// It is matched ahead of the RESTRICT catch-all below only because nothing in
	// this repository deletes a swarm — the RESTRICT direction of this key is
	// raised by the swarm's own delete path, not here.
	case strings.Contains(msg, "fk_firewall_bindings_swarm"):
		return ErrFirewallSwarmNotFound
	case strings.Contains(msg, "chk_firewall_bindings_swarm_needs_hive"):
		return ErrFirewallSwarmNeedsHive
	case strings.Contains(msg, "fk_firewall_rules_src_set"),
		strings.Contains(msg, "fk_firewall_rules_dst_set"),
		strings.Contains(msg, "23001"),
		strings.Contains(msg, "restrict"):
		return ErrFirewallSetInUse
	default:
		return err
	}
}

// SetsByPurpose returns sets of one purpose with their elements loaded.
//
// Elements are preloaded because every caller needs them — a set without its
// contents cannot answer any question worth asking — and fetching them
// separately would make an empty set indistinguishable from an unloaded one.
func (r *FirewallRepoGorm) SetsByPurpose(ctx context.Context, purpose string) ([]tenant.FirewallSet, error) {
	var out []tenant.FirewallSet
	err := r.db.WithContext(ctx).
		Preload("Elements").
		Where("purpose = ?", purpose).
		Order("name ASC").
		Find(&out).Error
	return out, err
}

// ---- sets --------------------------------------------------------------------

// ListSets returns every set with its elements loaded, for the same reason
// SetsByPurpose preloads them: a set without its contents cannot answer any
// question worth asking, and an empty one would be indistinguishable from an
// unloaded one.
func (r *FirewallRepoGorm) ListSets(ctx context.Context) ([]tenant.FirewallSet, error) {
	var out []tenant.FirewallSet
	err := r.db.WithContext(ctx).
		Preload("Elements").
		Order("name ASC").
		Find(&out).Error
	return out, err
}

func (r *FirewallRepoGorm) GetSet(ctx context.Context, id uuid.UUID) (*tenant.FirewallSet, error) {
	var s tenant.FirewallSet
	if err := r.db.WithContext(ctx).Preload("Elements").First(&s, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFirewallSetNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *FirewallRepoGorm) CreateSet(ctx context.Context, s *tenant.FirewallSet) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *FirewallRepoGorm) UpdateSet(
	ctx context.Context, id uuid.UUID, name, description string,
) (*tenant.FirewallSet, error) {
	// Name and description only. Family is not editable: a set's family decides
	// the nft type its elements are declared as, so changing it would reinterpret
	// every element already in it — and the rules referencing it are validated
	// against the family they were written for.
	res := r.db.WithContext(ctx).Model(&tenant.FirewallSet{}).
		Where("id = ?", id).
		Updates(map[string]any{"name": name, "description": description})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrFirewallSetNotFound
	}
	return r.GetSet(ctx, id)
}

// DeleteSet removes a set and its elements.
//
// A set still referenced by a rule is refused by the database, not by a check
// here: FirewallRule.SrcSet carries constraint:OnDelete:RESTRICT precisely so
// this cannot become a ruleset that references a set nft has never heard of —
// which nft rejects wholesale, taking the host's entire firewall with it.
func (r *FirewallRepoGorm) DeleteSet(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("set_id = ?", id).Delete(&tenant.FirewallSetElement{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&tenant.FirewallSet{}, "id = ?", id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrFirewallSetNotFound
		}
		return nil
	})
}

func (r *FirewallRepoGorm) AddSetElement(ctx context.Context, e *tenant.FirewallSetElement) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// DeleteSetElement is scoped by set as well as by element id, so an element id
// from one set cannot remove a row from another.
func (r *FirewallRepoGorm) DeleteSetElement(ctx context.Context, setID, elementID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND set_id = ?", elementID, setID).
		Delete(&tenant.FirewallSetElement{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrFirewallSetElemNotFound
	}
	return nil
}
