package policy

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

// RegisterOrgProtected mounts the organization policy surface.
//
// The organization comes from the caller's session, never from the request —
// this is the same IDOR seam as everywhere else, and here it would let an admin
// of one tenant loosen another tenant's security posture.
func RegisterOrgProtected(hapi huma.API, deps *app.Deps) {
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "org-policies-list",
		Method:      http.MethodGet,
		Path:        "/organization/policies",
		Summary:     "List this organization's policies",
		Description: "Each kind shows this organization's own position AND the effective result, " +
			"because 'what applies to us' is the question being asked and neither level answers it alone. " +
			"A restriction merges toward strictness: the platform's floor cannot be lowered here, only raised.",
		Tags:     []string{"Organization"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *struct{}) (*ListResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		rows, err := deps.Policies.ListOrgPolicies(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		set := map[control.PolicyKind]control.OrgPolicy{}
		for _, r := range rows {
			set[r.Kind] = r
		}

		out := &ListResponse{}
		for _, k := range allKinds {
			row, present := set[k]
			eff, err := deps.Policies.EffectivePolicy(ctx, p.OrganizationID, k)
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
		OperationID: "org-policies-set",
		Method:      http.MethodPut,
		Path:        "/organization/policies/{kind}",
		Summary:     "Set a policy for this organization",
		Description: "Requires admin. Impersonated sessions are refused: an operator acting as a " +
			"user must not be able to change that organization's security posture under their name. " +
			"Note that disabling here does NOT switch off a platform policy — a restriction merges " +
			"toward strictness, so an organization can raise the floor but never lower it.",
		Tags:     []string{"Organization"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *SetRequest) (*SetResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		kind := control.PolicyKind(in.Kind)

		row, err := deps.Policies.SetOrgPolicy(ctx, *p.UserID, p.OrganizationID, kind,
			in.Body.State == "enabled", toConfig(in.Body.Config))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		eff, err := deps.Policies.EffectivePolicy(ctx, p.OrganizationID, kind)
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
		OperationID: "org-policies-clear",
		Method:      http.MethodDelete,
		Path:        "/organization/policies/{kind}",
		Summary:     "Return a policy to inherit",
		Tags:        []string{"Organization"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeleteRequest) (*DeleteResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		if err := deps.Policies.DeleteOrgPolicy(ctx, p.OrganizationID, control.PolicyKind(in.Kind)); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeleteResponse{}
		out.Body.State = "inherit"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "org-policies-impact",
		Method:      http.MethodGet,
		Path:        "/organization/policies/{kind}/impact",
		Summary:     "How many members this policy would lock out",
		Description: "Scoped to this organization. Ask before enabling — the alternative is finding out " +
			"from the people who can no longer sign in.",
		Tags:     []string{"Organization"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ImpactRequest) (*ImpactResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		orgID := p.OrganizationID
		n, err := deps.Policies.PreEnableCheck(ctx, &orgID, control.PolicyKind(in.Kind))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ImpactResponse{}
		out.Body.NonCompliantUsers = n
		if n == 0 {
			out.Body.Message = "Every member already satisfies this. Enabling it changes nothing today."
		} else {
			out.Body.Message = "These members do not satisfy this policy yet and will be refused once it applies."
		}
		return out, nil
	})
}
