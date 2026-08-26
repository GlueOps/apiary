package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FirewallService owns authoring: rulesets, the rules inside them, and which
// fleets they are bound to.
//
// It does NOT publish. Turning authored rules into a revision a bee converges on
// is a separate step with a separate failure mode — an edit that is merely saved
// changes nothing on any host, which is what makes it safe to edit in the middle
// of the day.
type FirewallService struct {
	tenants TenantResolver
	newRepo func(db *gorm.DB) *repo.FirewallRepoGorm
}

func NewFirewallService(tenants TenantResolver) *FirewallService {
	return &FirewallService{
		tenants: tenants,
		newRepo: repo.NewFirewallRepoGorm,
	}
}

func (s *FirewallService) repoFor(ctx context.Context, orgID uuid.UUID) (*repo.FirewallRepoGorm, error) {
	// A check that cannot reach a decision must deny.
	if orgID == uuid.Nil || isNilResolver(s.tenants) {
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}
	return s.newRepo(tCtx.DB), nil
}

// ---- rulesets --------------------------------------------------------------

func (s *FirewallService) ListRulesets(ctx context.Context, orgID uuid.UUID) ([]tenant.FirewallRuleset, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.ListRulesets(ctx)
}

func (s *FirewallService) CreateRuleset(ctx context.Context, orgID uuid.UUID, name, description string) (*tenant.FirewallRuleset, error) {
	rs := &tenant.FirewallRuleset{Name: strings.TrimSpace(name), Description: strings.TrimSpace(description)}
	if err := rs.Validate(); err != nil {
		return nil, err
	}
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if err := r.CreateRuleset(ctx, rs); err != nil {
		return nil, err
	}
	return rs, nil
}

func (s *FirewallService) UpdateRuleset(ctx context.Context, orgID, id uuid.UUID, name, description string) (*tenant.FirewallRuleset, error) {
	probe := &tenant.FirewallRuleset{Name: strings.TrimSpace(name)}
	if err := probe.Validate(); err != nil {
		return nil, err
	}
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.UpdateRuleset(ctx, id, probe.Name, strings.TrimSpace(description))
}

func (s *FirewallService) DeleteRuleset(ctx context.Context, orgID, id uuid.UUID) error {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return err
	}
	return r.DeleteRuleset(ctx, id)
}

// ---- rules -----------------------------------------------------------------

func (s *FirewallService) ListRules(ctx context.Context, orgID, rulesetID uuid.UUID) ([]tenant.FirewallRule, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	// Confirms the ruleset is in THIS database before listing. Without it, an id
	// from another tenant returns an empty list rather than a refusal, which
	// reads as "that ruleset has no rules" instead of "that is not yours".
	if _, err := r.GetRuleset(ctx, rulesetID); err != nil {
		return nil, err
	}
	return r.ListRules(ctx, rulesetID)
}

// RuleInput is the whole rule. There is no partial update: applying half an
// edit — a new port against the old action — produces a rule nobody intended.
type RuleInput struct {
	Name     string
	Enabled  bool
	Family   string
	Table    string
	Chain    string
	Position int

	SrcCidr string
	DstCidr string

	Protocol   string
	SrcPort    int
	SrcPortEnd int
	DstPort    int
	DstPortEnd int

	InIface  string
	OutIface string

	Action  string
	Comment string
}

// toModel fills in the defaults a caller may leave out, so that "" does not
// reach Validate and fail with a message about a field the operator never saw.
func (in RuleInput) toModel() tenant.FirewallRule {
	family := strings.TrimSpace(in.Family)
	if family == "" {
		family = tenant.FamilyInet
	}
	table := strings.TrimSpace(in.Table)
	if table == "" {
		table = "filter"
	}
	chain := strings.TrimSpace(in.Chain)
	if chain == "" {
		chain = "input"
	}
	src := strings.TrimSpace(in.SrcCidr)
	dst := strings.TrimSpace(in.DstCidr)
	// The any-address default depends on the family: 0.0.0.0/0 in an ip6 rule is
	// a family mismatch, and defaulting to it would make every v6 rule fail
	// validation over a value nobody typed.
	if src == "" {
		src = tenant.AnyAddressFor(family)
	}
	if dst == "" {
		dst = tenant.AnyAddressFor(family)
	}
	protocol := tenant.Protocol(strings.TrimSpace(in.Protocol))
	if protocol == "" {
		protocol = tenant.TCP
	}
	action := tenant.Action(strings.TrimSpace(in.Action))
	if action == "" {
		action = tenant.Accept
	}
	position := in.Position
	if position == 0 {
		position = 100
	}
	return tenant.FirewallRule{
		Name: strings.TrimSpace(in.Name), Enabled: in.Enabled,
		Family: family, TableName: table, ChainName: chain, Position: position,
		SrcCidr: src, DstCidr: dst,
		Protocol: protocol,
		SrcPort:  in.SrcPort, SrcPortEnd: in.SrcPortEnd,
		DstPort: in.DstPort, DstPortEnd: in.DstPortEnd,
		InIface: strings.TrimSpace(in.InIface), OutIface: strings.TrimSpace(in.OutIface),
		Action: action, Comment: strings.TrimSpace(in.Comment),
	}
}
func (s *FirewallService) CreateRule(ctx context.Context, orgID, rulesetID uuid.UUID, in RuleInput) (*tenant.FirewallRule, error) {
	rule := in.toModel()
	rule.RulesetID = rulesetID
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if _, err := r.GetRuleset(ctx, rulesetID); err != nil {
		return nil, err
	}
	if err := r.CreateRule(ctx, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *FirewallService) UpdateRule(ctx context.Context, orgID, id uuid.UUID, in RuleInput) (*tenant.FirewallRule, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	// Read first: it proves the rule is in this database, and carries forward the
	// ruleset it belongs to. A rule cannot be moved between rulesets by an edit —
	// its position means something only within one.
	existing, err := r.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}
	rule := in.toModel()
	rule.ID = existing.ID
	rule.RulesetID = existing.RulesetID
	if err := rule.Validate(); err != nil {
		return nil, err
	}
	if err := r.UpdateRule(ctx, &rule); err != nil {
		return nil, err
	}
	return r.GetRule(ctx, id)
}

func (s *FirewallService) DeleteRule(ctx context.Context, orgID, id uuid.UUID) error {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return err
	}
	return r.DeleteRule(ctx, id)
}

// ---- bindings --------------------------------------------------------------

func (s *FirewallService) ListBindings(ctx context.Context, orgID uuid.UUID) ([]tenant.FirewallBinding, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.ListBindings(ctx)
}

// Bind attaches a ruleset to a fleet, or to the whole organization when hiveID
// is nil.
//
// The org-wide case is what covers a host between fleets. It is also how a base
// ruleset stops being copied per fleet, which is the duplication the whole
// structure exists to avoid.
func (s *FirewallService) Bind(
	ctx context.Context, orgID uuid.UUID, hiveID, swarmID *uuid.UUID,
	rulesetID uuid.UUID, priority int, enabled bool,
) (*tenant.FirewallBinding, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if _, err := r.GetRuleset(ctx, rulesetID); err != nil {
		return nil, err
	}
	// A SWARM WITHOUT ITS HIVE IS NOT A SCOPE. Swarm names are unique only within
	// a hive, so a binding carrying a bare swarm id names a group without saying
	// whose — and BindingsForScope matches swarm bindings on the swarm id ALONE,
	// which is safe precisely because the pair is validated here. Accepting the
	// half would let a binding written for one cluster's "prod" swarm resolve
	// against another cluster's.
	if swarmID != nil && hiveID == nil {
		return nil, repo.ErrFirewallSwarmNeedsHive
	}
	if hiveID != nil {
		ok, err := r.HiveExists(ctx, *hiveID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repo.ErrFirewallHiveNotFound
		}
	}
	if swarmID != nil {
		// Checked as a PAIR, not as "does this swarm exist anywhere". A swarm from
		// a different hive would otherwise bind happily and then apply to hosts
		// the operator never named.
		ok, err := r.SwarmExists(ctx, *hiveID, *swarmID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repo.ErrFirewallSwarmNotFound
		}
	}
	if priority == 0 {
		priority = 100
	}
	b := &tenant.FirewallBinding{
		HiveID: hiveID, SwarmID: swarmID, RulesetID: rulesetID,
		Priority: priority, Enabled: enabled,
	}
	if err := r.CreateBinding(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *FirewallService) UpdateBinding(ctx context.Context, orgID, id uuid.UUID, priority int, enabled bool) error {
	return s.UpdateBindingScope(ctx, orgID, id, priority, enabled, nil)
}

// UpdateBindingScope updates a binding and, when scope is non-nil, moves it.
//
// The scope is validated with the SAME rules the create path uses, because a
// binding moved into an invalid scope is indistinguishable from one created
// there. In particular a swarm without its hive is refused: swarm names are
// unique only within a hive, and BindingsForScope matches swarm bindings on the
// swarm id ALONE — safe only because the pair is checked here. Accepting the
// half would let a binding written for one cluster's "prod" group resolve
// against another cluster's.
func (s *FirewallService) UpdateBindingScope(
	ctx context.Context, orgID, id uuid.UUID, priority int, enabled bool,
	scope *repo.BindingScopeChange,
) error {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return err
	}
	if scope != nil {
		if scope.SwarmID != nil && scope.HiveID == nil {
			return repo.ErrFirewallSwarmNeedsHive
		}
		if scope.HiveID != nil {
			ok, err := r.HiveExists(ctx, *scope.HiveID)
			if err != nil {
				return err
			}
			if !ok {
				return repo.ErrFirewallHiveNotFound
			}
		}
		if scope.SwarmID != nil {
			// As a PAIR. A swarm from another hive would otherwise move happily and
			// then apply to hosts the operator never named.
			ok, err := r.SwarmExists(ctx, *scope.HiveID, *scope.SwarmID)
			if err != nil {
				return err
			}
			if !ok {
				return repo.ErrFirewallSwarmNotFound
			}
		}
	}
	if priority == 0 {
		priority = 100
	}
	return r.UpdateBindingWithScope(ctx, id, priority, enabled, scope)
}

func (s *FirewallService) Unbind(ctx context.Context, orgID, id uuid.UUID) error {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return err
	}
	return r.DeleteBinding(ctx, id)
}

// ManagementSets returns the sets that feed the authorized_keys `from=` clause.
//
// Purpose is read from the set rather than inferred from the ports a rule opens.
// Corosync peers are permitted on 22 so pvecm can distribute cluster
// configuration, so inferring would list every hypervisor in the fleet and make
// a stolen control-plane key usable from any of them.
func (s *FirewallService) ManagementSets(ctx context.Context, orgID uuid.UUID) ([]tenant.FirewallSet, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return r.SetsByPurpose(ctx, tenant.SetPurposeManagement)
}

// ---- preview ---------------------------------------------------------------

// Preview returns the flattened, ordered rules a fleet would receive.
//
// It exists because composition is invisible from any single screen: an
// operator editing "base" cannot see it interleaving with "dc1", and binding
// priority silently outranks the positions they were just looking at. Answering
// "what does hv5 actually get" is the difference between authoring a policy and
// hoping about one.
//
// A nil hiveID answers for a host that belongs to no fleet. A nil swarmID means
// "not in a swarm, or we do not know which" — both answer with hive and org
// scope, which is the same thing a real host in no swarm receives.
func (s *FirewallService) Preview(
	ctx context.Context, orgID uuid.UUID, hiveID, swarmID *uuid.UUID,
) ([]tenant.FirewallRule, error) {
	r, err := s.repoFor(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if hiveID != nil {
		ok, err := r.HiveExists(ctx, *hiveID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repo.ErrFirewallHiveNotFound
		}
	}
	if swarmID != nil {
		if hiveID == nil {
			return nil, repo.ErrFirewallSwarmNeedsHive
		}
		ok, err := r.SwarmExists(ctx, *hiveID, *swarmID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repo.ErrFirewallSwarmNotFound
		}
	}
	bindings, err := r.BindingsForScope(ctx, tenant.BindingScope{HiveID: hiveID, SwarmID: swarmID})
	if err != nil {
		return nil, err
	}
	byRuleset := make(map[uuid.UUID][]tenant.FirewallRule, len(bindings))
	for _, b := range bindings {
		if _, seen := byRuleset[b.RulesetID]; seen {
			continue
		}
		rules, err := r.ListRules(ctx, b.RulesetID)
		if err != nil {
			return nil, err
		}
		byRuleset[b.RulesetID] = rules
	}
	// Ordering is the model's, not this layer's: the renderer and the preview
	// must agree, and two orderings written in two places will not.
	return tenant.OrderedRules(bindings, byRuleset), nil
}

// IsFirewallValidation reports whether an error is something the operator typed,
// so a handler can answer 422 rather than 500. Every validation error is a
// deliberate refusal, and a refusal reported as a server fault puts an
// operator's own typo on a 5xx dashboard.
func IsFirewallValidation(err error) bool {
	for _, e := range []error{
		tenant.ErrFirewallFamily, tenant.ErrFirewallProtocol, tenant.ErrFirewallAction,
		tenant.ErrFirewallCIDR, tenant.ErrFirewallCIDRFamily, tenant.ErrFirewallPort,
		tenant.ErrFirewallPortRange, tenant.ErrFirewallPortless, tenant.ErrFirewallChainName,
		tenant.ErrFirewallPosition, tenant.ErrFirewallRulesetName,
	} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
