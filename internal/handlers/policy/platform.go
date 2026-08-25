package policy

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
)

// RegisterAdminProtected mounts the platform policy surface.
//
// It MUST be handed a group carrying RequireAdminAuth. There is no role check
// inside these handlers any more, deliberately: the group is the gate, so an
// endpoint added here is protected by existing rather than by its author
// remembering. TestPlatformPolicyIsAdminOnly pins that the group actually
// carries it, because the cost of getting this wrong is that any authenticated
// user can rewrite the security floor for every tenant.
func RegisterAdminProtected(hapi huma.API, deps *app.Deps) {
	// The principal is still read, for the audit trail — SetPlatformPolicy
	// records who made the change.
	principal := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		return p, nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-policies-list",
		Method:      http.MethodGet,
		Path:        "/platform/policies",
		Summary:     "List platform policies",
		Description: "Every policy kind, whether or not the platform has an opinion on it. A kind " +
			"the platform has never set reads as 'inherit', which is the identity element of the merge.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *struct{}) (*ListResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		rows, err := deps.Policies.ListPlatformPolicies(ctx)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		set := map[control.PolicyKind]control.PlatformPolicy{}
		for _, r := range rows {
			set[r.Kind] = r
		}

		out := &ListResponse{}
		for _, k := range allKinds {
			row, present := set[k]
			// The effective view at platform level uses uuid.Nil for the org, so
			// it shows what an organization with no opinion of its own would get.
			eff, err := deps.Policies.EffectivePolicy(ctx, uuid.Nil, k)
			if err != nil {
				return nil, mapErr(ctx, err)
			}
			v := PolicyView{
				Kind:             string(k),
				State:            stateOf(present, row.Enabled),
				EffectiveEnabled: eff.Enabled,
				EffectiveConfig:  fromConfig(eff.Config),
			}
			if present {
				v.Config = fromConfig(row.Config.Data())
			}
			out.Body.Policies = append(out.Body.Policies, v)
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-policies-set",
		Method:      http.MethodPut,
		Path:        "/platform/policies/{kind}",
		Summary:     "Set a platform policy",
		Description: "Impersonated sessions are refused: a platform policy applies to every tenant, " +
			"and an operator acting as somebody else should not be able to change it under their name.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *SetRequest) (*SetResponse, error) {
		p, err := principal(ctx)
		if err != nil {
			return nil, err
		}
		kind := control.PolicyKind(in.Kind)

		row, err := deps.Policies.SetPlatformPolicy(ctx, *p.UserID, kind,
			in.Body.State == "enabled", toConfig(in.Body.Config))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		eff, err := deps.Policies.EffectivePolicy(ctx, uuid.Nil, kind)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &SetResponse{Body: PolicyView{
			Kind:             string(kind),
			State:            stateOf(true, row.Enabled),
			Config:           fromConfig(row.Config.Data()),
			EffectiveEnabled: eff.Enabled,
			EffectiveConfig:  fromConfig(eff.Config),
		}}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-policies-clear",
		Method:      http.MethodDelete,
		Path:        "/platform/policies/{kind}",
		Summary:     "Return a platform policy to inherit",
		Description: "Deleting is how the platform withdraws its opinion. That is NOT the same as " +
			"disabling: a disabled policy contributes nothing to the merge either, but it is a " +
			"stated position, and the distinction is what an organization sees when deciding " +
			"whether it needs its own.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *DeleteRequest) (*DeleteResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		if err := deps.Policies.DeletePlatformPolicy(ctx, control.PolicyKind(in.Kind)); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeleteResponse{}
		out.Body.State = "inherit"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-policies-impact",
		Method:      http.MethodGet,
		Path:        "/platform/policies/{kind}/impact",
		Summary:     "How many users this policy would lock out",
		Description: "Ask BEFORE enabling. Turning on require_mfa across a platform where half the " +
			"users have never enrolled a factor locks them out at their next request, and the only " +
			"signal is a support queue.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ImpactRequest) (*ImpactResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		n, err := deps.Policies.PreEnableCheck(ctx, nil, control.PolicyKind(in.Kind))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ImpactResponse{}
		out.Body.NonCompliantUsers = n
		if n == 0 {
			out.Body.Message = "Every user already satisfies this. Enabling it changes nothing today."
		} else {
			out.Body.Message = "These users do not satisfy this policy yet and will be refused once it applies."
		}
		return out, nil
	})
}
