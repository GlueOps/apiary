package firewall

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
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
	}
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
		Summary:     "Reprioritise or disable a binding",
		Tags:        []string{"Firewall"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateBindingRequest) (*DeletedResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid binding id")
		}
		if err := svc.UpdateBinding(ctx, p.OrganizationID, id, in.Body.Priority, in.Body.Enabled); err != nil {
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
