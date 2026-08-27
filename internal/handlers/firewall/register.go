package firewall

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ListRulesetsRequest struct{}
type ListRulesetsResponse struct {
	Body struct {
		Rulesets []RulesetView `json:"rulesets"`
	}
}

type CreateRulesetRequest struct {
	Body struct {
		Name        string `json:"name" required:"true" minLength:"1" maxLength:"100"`
		Description string `json:"description,omitempty" maxLength:"500"`
	}
}
type RulesetResponse struct{ Body RulesetView }

type UpdateRulesetRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Name        string `json:"name" required:"true" minLength:"1" maxLength:"100"`
		Description string `json:"description,omitempty" maxLength:"500"`
	}
}

type IDRequest struct {
	ID string `path:"id" format:"uuid"`
}
type DeletedResponse struct {
	Body struct {
		Deleted bool `json:"deleted"`
	}
}

// ---- sets --------------------------------------------------------------------

type SetView struct {
	ID          string           `json:"id" format:"uuid"`
	Name        string           `json:"name"`
	Family      string           `json:"family" example:"ip"`
	Purpose     string           `json:"purpose" example:"general"`
	Description string           `json:"description,omitempty"`
	Managed     bool             `json:"managed" doc:"Contents maintained by the control plane. Edits are refused rather than silently reverted by the next convergence."`
	Elements    []SetElementView `json:"elements"`
}

type SetElementView struct {
	ID      string `json:"id" format:"uuid"`
	Value   string `json:"value" example:"10.10.10.0/24"`
	Comment string `json:"comment,omitempty"`
	Label   string `json:"label,omitempty"`
}

type ListSetsRequest struct{}
type ListSetsResponse struct {
	Body struct {
		Sets []SetView `json:"sets"`
	}
}

type CreateSetRequest struct {
	Body struct {
		Name string `json:"name" required:"true" minLength:"1" maxLength:"100"`
		// ip or ip6, never inet. An nft set is declared ipv4_addr or ipv6_addr
		// and holds one address family; inet is a TABLE family, and a set
		// declared with it renders a line nft rejects — taking the whole ruleset,
		// and the host's entire firewall, with it. The RULE stays inet.
		Family      string `json:"family,omitempty" enum:"ip,ip6" default:"ip" doc:"An nft set holds ONE address family. inet is a table family and is refused here; leave the rule inet and pick the family the addresses are."`
		Description string `json:"description,omitempty" maxLength:"500"`
		Purpose     string `json:"purpose,omitempty" enum:"general,management,peers" default:"general" doc:"management widens the from= restriction on the fleet SSH key, so it defaults to general — the harmless one."`
	}
}
type SetResponse struct{ Body SetView }

type UpdateSetRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Name        string `json:"name" required:"true" minLength:"1" maxLength:"100"`
		Description string `json:"description,omitempty" maxLength:"500"`
	}
}

type ListSetElementsRequest struct {
	ID string `path:"id" format:"uuid"`
}
type ListSetElementsResponse struct {
	Body struct {
		Elements []SetElementView `json:"elements"`
	}
}

type AddSetElementRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Value   string `json:"value" required:"true" minLength:"1" maxLength:"64" doc:"An address or a prefix. A bare 192.168.10.60 is accepted and treated as /32 — an operator writing a management address should not have to spell that out."`
		Comment string `json:"comment,omitempty" maxLength:"200"`
		Label   string `json:"label,omitempty" maxLength:"64"`
	}
}
type SetElementResponse struct{ Body SetElementView }

type DeleteSetElementRequest struct {
	ID        string `path:"id" format:"uuid"`
	ElementID string `path:"element_id" format:"uuid"`
}

func toSetView(s tenant.FirewallSet) SetView {
	out := SetView{
		ID: s.ID.String(), Name: s.Name, Family: s.Family,
		Purpose: s.Purpose, Description: s.Description, Managed: s.Managed,
	}
	out.Elements = make([]SetElementView, 0, len(s.Elements))
	for _, e := range s.Elements {
		out.Elements = append(out.Elements, toSetElementView(e))
	}
	return out
}

func toSetElementView(e tenant.FirewallSetElement) SetElementView {
	return SetElementView{
		ID: e.ID.String(), Value: e.Value, Comment: e.Comment, Label: e.Label,
	}
}

type ListRulesRequest struct {
	ID string `path:"id" format:"uuid"`
}
type ListRulesResponse struct {
	Body struct {
		Rules []RuleView `json:"rules"`
	}
}

type CreateRuleRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body RuleBody
}
type RuleResponse struct{ Body RuleView }

type UpdateRuleRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body RuleBody
}

type ListBindingsResponse struct {
	Body struct {
		Bindings []BindingView `json:"bindings"`
	}
}

type CreateBindingRequest struct {
	Body struct {
		RulesetID string `json:"ruleset_id" required:"true" format:"uuid"`
		HiveID    string `json:"hive_id,omitempty" format:"uuid" doc:"Omit to bind ORG-WIDE: every host in the organization, whatever fleet it is in. That is what covers a host between fleets, which has no hive-scoped bindings at all."`
		SwarmID   string `json:"swarm_id,omitempty" format:"uuid" doc:"Narrow the binding to one group of hypervisors inside the fleet. Requires hive_id: swarm names are unique only within a fleet, so a swarm alone does not identify a scope."`
		Priority  int    `json:"priority,omitempty" minimum:"0" default:"100" doc:"Lower renders first, and outranks rule positions inside the ruleset. Ties break by scope, most specific first."`
		Enabled   bool   `json:"enabled" default:"true"`
	}
}
type BindingResponse struct{ Body BindingView }

type UpdateBindingRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Priority int  `json:"priority,omitempty" minimum:"0"`
		Enabled  bool `json:"enabled"`

		// Scope moves this binding. ABSENT leaves it exactly where it is.
		//
		// A nested object rather than two sibling fields, because absence has to be
		// distinguishable from "both empty". Sibling ids would arrive as empty
		// strings from any client that predates this — every one of them — and the
		// binding would be silently re-scoped from its fleet to the whole
		// organization. Widening a binding to every host is the worst accident this
		// endpoint could have, so the absence of an instruction means absence.
		Scope *BindingScopeBody `json:"scope,omitempty"`
	}
}

// BindingScopeBody is where a binding should apply. An empty hive_id means the
// whole organization; a swarm_id requires the hive_id that contains it.
type BindingScopeBody struct {
	HiveID  string `json:"hive_id,omitempty" doc:"The fleet this binding applies to. Empty means every host in the organization, including hosts in no fleet."`
	SwarmID string `json:"swarm_id,omitempty" doc:"A group inside that fleet. Requires hive_id: swarm names are unique only within a fleet, so a bare swarm id names a group without saying whose."`
}

type PreviewRequest struct {
	HiveID  string `query:"hive_id" format:"uuid" doc:"Omit to preview a host that belongs to NO fleet — the state every migration passes through."`
	SwarmID string `query:"swarm_id" format:"uuid" doc:"Preview a host in one group inside the fleet. Requires hive_id. Omit to preview a host in no group, which receives fleet and organization scope only."`
}
type PreviewResponse struct {
	Body struct {
		Rules []RuleView `json:"rules"`
		Count int        `json:"count"`
	}
}

// RegisterUserProtected mounts rule authoring. It must be handed a group
// carrying RequireUserAuth.
func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	svc := service.NewFirewallService(deps.Tenants)
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	// Reading is open to any member. Changing is admin-only and refuses
	// impersonation: a support engineer acting as a customer must not be able to
	// open a path into that customer's metal.
	member := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}
	admin := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}

	// Publishing, backing out, and reading what was published. Mounted here so it
	// shares this file's admin gate: authoring a rule and putting it on metal are
	// both changes to what the fleet enforces.
	registerPublish(hapi, admin,
		func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error) {
			tCtx, err := deps.Tenants.GetTenantContext(ctx, orgID.String())
			if err != nil {
				return nil, err
			}
			return tCtx.DB, nil
		},
		repo.ResolveHiveRenderInput,
	)

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rulesets-list",
		Method:      http.MethodGet,
		Path:        "/firewall/rulesets",
		Summary:     "List firewall rulesets",
		Description: "Named groups of rules. Fleets are bound to these rather than owning rules directly, so a rule an organization wants everywhere is written once instead of copied per fleet.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListRulesetsRequest) (*ListRulesetsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		list, err := svc.ListRulesets(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListRulesetsResponse{}
		out.Body.Rulesets = make([]RulesetView, 0, len(list))
		for _, rs := range list {
			out.Body.Rulesets = append(out.Body.Rulesets, toRulesetView(rs))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-rulesets-create",
		Method:        http.MethodPost,
		Path:          "/firewall/rulesets",
		Summary:       "Create a firewall ruleset",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Firewall"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateRulesetRequest) (*RulesetResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		rs, err := svc.CreateRuleset(ctx, p.OrganizationID, in.Body.Name, in.Body.Description)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &RulesetResponse{Body: toRulesetView(*rs)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rulesets-update",
		Method:      http.MethodPut,
		Path:        "/firewall/rulesets/{id}",
		Summary:     "Rename a firewall ruleset",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateRulesetRequest) (*RulesetResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid ruleset id")
		}
		rs, err := svc.UpdateRuleset(ctx, p.OrganizationID, id, in.Body.Name, in.Body.Description)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &RulesetResponse{Body: toRulesetView(*rs)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rulesets-delete",
		Method:      http.MethodDelete,
		Path:        "/firewall/rulesets/{id}",
		Summary:     "Delete a firewall ruleset",
		Description: "Removes its rules and every binding to it. A binding to a ruleset that no longer exists would render nothing while still listing as though a fleet were covered.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *IDRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid ruleset id")
		}
		if err := svc.DeleteRuleset(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	// ---- sets ----------------------------------------------------------------
	//
	// A named list a rule points at instead of repeating a literal. Missing until
	// now, which meant the model, the validation and the renderer all supported
	// sets while the only way to create one was to write SQL — and onboarding
	// REFUSES without a management set, so automated onboarding could not be
	// completed by an operator at all.

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-sets-list",
		Method:      http.MethodGet,
		Path:        "/firewall/sets",
		Summary:     "List firewall sets",
		Description: "Named address lists. A rule referencing a set matches every element in it, so four management addresses across two ports is two rules and one set rather than eight rules — and adding an address is one element, not two more rules.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListSetsRequest) (*ListSetsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		list, err := svc.ListSets(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListSetsResponse{}
		out.Body.Sets = make([]SetView, 0, len(list))
		for _, set := range list {
			out.Body.Sets = append(out.Body.Sets, toSetView(set))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-sets-create",
		Method:        http.MethodPost,
		Path:          "/firewall/sets",
		Summary:       "Create a firewall set",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Firewall"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateSetRequest) (*SetResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		set, err := svc.CreateSet(ctx, p.OrganizationID,
			in.Body.Name, in.Body.Family, in.Body.Description, in.Body.Purpose)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &SetResponse{Body: toSetView(*set)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-sets-update",
		Method:      http.MethodPut,
		Path:        "/firewall/sets/{id}",
		Summary:     "Rename a firewall set",
		Description: "Name and description only. A set's family decides the nft type its elements are declared as, so changing it would reinterpret every element already in it and every rule written against it.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateSetRequest) (*SetResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid set id")
		}
		set, err := svc.UpdateSet(ctx, p.OrganizationID, id, in.Body.Name, in.Body.Description)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &SetResponse{Body: toSetView(*set)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-sets-delete",
		Method:      http.MethodDelete,
		Path:        "/firewall/sets/{id}",
		Summary:     "Delete a firewall set",
		Description: "Refused while a rule still references it. nft rejects a ruleset naming a set it has never heard of, and it rejects the WHOLE file — so a dangling reference would leave the host with no firewall at all rather than one missing a line.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *IDRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid set id")
		}
		if err := svc.DeleteSet(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-set-elements-list",
		Method:      http.MethodGet,
		Path:        "/firewall/sets/{id}/elements",
		Summary:     "List a set's elements",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ListSetElementsRequest) (*ListSetElementsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid set id")
		}
		set, err := svc.GetSet(ctx, p.OrganizationID, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListSetElementsResponse{}
		out.Body.Elements = make([]SetElementView, 0, len(set.Elements))
		for _, e := range set.Elements {
			out.Body.Elements = append(out.Body.Elements, toSetElementView(e))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-set-elements-add",
		Method:        http.MethodPost,
		Path:          "/firewall/sets/{id}/elements",
		Summary:       "Add an address to a set",
		DefaultStatus: http.StatusCreated,
		Description:   "Validated against the SET's family, not on its own: an ipv6 element in an ipv4 set renders a declaration nft rejects, and nft rejects the whole ruleset over one bad line.",
		Tags:          []string{"Firewall"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *AddSetElementRequest) (*SetElementResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid set id")
		}
		el, err := svc.AddSetElement(ctx, p.OrganizationID, id,
			in.Body.Value, in.Body.Comment, in.Body.Label)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &SetElementResponse{Body: toSetElementView(*el)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-set-elements-delete",
		Method:      http.MethodDelete,
		Path:        "/firewall/sets/{id}/elements/{element_id}",
		Summary:     "Remove an address from a set",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeleteSetElementRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		setID, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid set id")
		}
		elemID, err := uuid.Parse(in.ElementID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid element id")
		}
		if err := svc.RemoveSetElement(ctx, p.OrganizationID, setID, elemID); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rules-list",
		Method:      http.MethodGet,
		Path:        "/firewall/rulesets/{id}/rules",
		Summary:     "List a ruleset's rules",
		Description: "Returned in the order they render within this ruleset. Order is the policy — an accept above a drop is a different rule from the reverse.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ListRulesRequest) (*ListRulesResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid ruleset id")
		}
		rules, err := svc.ListRules(ctx, p.OrganizationID, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListRulesResponse{}
		out.Body.Rules = make([]RuleView, 0, len(rules))
		for _, r := range rules {
			out.Body.Rules = append(out.Body.Rules, toRuleView(r))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-rules-create",
		Method:        http.MethodPost,
		Path:          "/firewall/rulesets/{id}/rules",
		Summary:       "Add a rule to a ruleset",
		DefaultStatus: http.StatusCreated,
		Description:   "Refused if the rule cannot mean what it says — a port on a protocol that has none is dropped when rendered, so the stored rule would be wider than the one written.",
		Tags:          []string{"Firewall"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateRuleRequest) (*RuleResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid ruleset id")
		}
		rule, err := svc.CreateRule(ctx, p.OrganizationID, id, in.Body.toInput())
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &RuleResponse{Body: toRuleView(*rule)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rules-update",
		Method:      http.MethodPut,
		Path:        "/firewall/rules/{id}",
		Summary:     "Replace a rule",
		Description: "The whole rule, every time. A partial update could apply a new port against the old action, producing a rule nobody intended.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateRuleRequest) (*RuleResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid rule id")
		}
		rule, err := svc.UpdateRule(ctx, p.OrganizationID, id, in.Body.toInput())
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &RuleResponse{Body: toRuleView(*rule)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-rules-delete",
		Method:      http.MethodDelete,
		Path:        "/firewall/rules/{id}",
		Summary:     "Delete a rule",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *IDRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid rule id")
		}
		if err := svc.DeleteRule(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-bindings-list",
		Method:      http.MethodGet,
		Path:        "/firewall/bindings",
		Summary:     "List ruleset bindings",
		Description: "Which rulesets apply where, in render order — lowest priority first. Bindings without a fleet are org-wide.",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListRulesetsRequest) (*ListBindingsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		list, err := svc.ListBindings(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListBindingsResponse{}
		out.Body.Bindings = make([]BindingView, 0, len(list))
		for _, b := range list {
			out.Body.Bindings = append(out.Body.Bindings, toBindingView(b))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-bindings-create",
		Method:        http.MethodPost,
		Path:          "/firewall/bindings",
		Summary:       "Bind a ruleset to a fleet, or to the whole organization",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Firewall"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateBindingRequest) (*BindingResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		rulesetID, err := uuid.Parse(in.Body.RulesetID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid ruleset id")
		}
		var hiveID, swarmID *uuid.UUID
		if in.Body.HiveID != "" {
			id, err := uuid.Parse(in.Body.HiveID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid fleet id")
			}
			hiveID = &id
		}
		if in.Body.SwarmID != "" {
			id, err := uuid.Parse(in.Body.SwarmID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid group id")
			}
			swarmID = &id
		}
		// The swarm-needs-hive refusal and the swarm-belongs-to-this-hive check
		// both live in the service, not here: a second copy of a scope rule in the
		// handler is a second thing to keep in step, and the service is what the
		// UI and any future caller share.
		b, err := svc.Bind(ctx, p.OrganizationID, hiveID, swarmID, rulesetID, in.Body.Priority, in.Body.Enabled)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &BindingResponse{Body: toBindingView(*b)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-bindings-update",
		Method:      http.MethodPut,
		Path:        "/firewall/bindings/{id}",
		Summary:     "Reprioritise, disable, or re-scope a binding",
		Description: "Priority and enabled are always applied. Scope is applied only when the " +
			"request carries one — omitting it leaves the binding where it is, so a client that " +
			"knows nothing about scope cannot widen a fleet binding to the whole organization by " +
			"accident.\n\n" +
			"Moving a binding onto a scope where the same ruleset is already bound is refused: " +
			"the ruleset would render twice, duplicating every rule in it.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateBindingRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid binding id")
		}
		var scope *repo.BindingScopeChange
		if in.Body.Scope != nil {
			scope = &repo.BindingScopeChange{}
			if in.Body.Scope.HiveID != "" {
				hid, err := uuid.Parse(in.Body.Scope.HiveID)
				if err != nil {
					return nil, huma.Error422UnprocessableEntity("invalid fleet id")
				}
				scope.HiveID = &hid
			}
			if in.Body.Scope.SwarmID != "" {
				sid, err := uuid.Parse(in.Body.Scope.SwarmID)
				if err != nil {
					return nil, huma.Error422UnprocessableEntity("invalid group id")
				}
				scope.SwarmID = &sid
			}
		}
		if err := svc.UpdateBindingScope(ctx, p.OrganizationID, id,
			in.Body.Priority, in.Body.Enabled, scope); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-bindings-delete",
		Method:      http.MethodDelete,
		Path:        "/firewall/bindings/{id}",
		Summary:     "Remove a binding",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *IDRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid binding id")
		}
		if err := svc.Unbind(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeletedResponse{}
		out.Body.Deleted = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-preview",
		Method:      http.MethodGet,
		Path:        "/firewall/preview",
		Summary:     "Preview the rules a fleet would receive",
		Description: "The flattened, ordered result of every binding that applies — org-wide and fleet-specific together. " +
			"Composition is invisible from any single screen: editing one ruleset cannot show it interleaving with another, and binding priority outranks the positions being edited. " +
			"Omit hive_id to see what a host belonging to NO fleet receives.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *PreviewRequest) (*PreviewResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		var hiveID, swarmID *uuid.UUID
		if in.HiveID != "" {
			id, err := uuid.Parse(in.HiveID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid fleet id")
			}
			hiveID = &id
		}
		if in.SwarmID != "" {
			id, err := uuid.Parse(in.SwarmID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid group id")
			}
			swarmID = &id
		}
		rules, err := svc.Preview(ctx, p.OrganizationID, hiveID, swarmID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &PreviewResponse{}
		out.Body.Rules = make([]RuleView, 0, len(rules))
		for _, r := range rules {
			out.Body.Rules = append(out.Body.Rules, toRuleView(r))
		}
		out.Body.Count = len(rules)
		return out, nil
	})
}
