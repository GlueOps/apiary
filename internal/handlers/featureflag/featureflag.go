// Package featureflag mounts the feature flag surface.
//
// TWO SURFACES WITH DIFFERENT AUTHORITY, and the split is the point. Platform
// staff set flags at both levels; a tenant reads what applies to it and can
// change nothing. A tenant that could set its own flags could switch legacy
// authentication back on for itself, which is exactly the kind of decision a
// flag exists to take away from them.
package featureflag

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
)

// TYPES ARE NAMED DISTINCTIVELY ON PURPOSE. The generated SDKs flatten every
// handler package into one namespace, so a second `ListResponse` in this tree
// does not collide loudly — hey-api emits `ListResponseBody1` and the TypeScript
// client silently binds an operation to the wrong shape. The compiler caught it
// here only because the fields differed.
//
// FlagView is one resolved flag, with the reason it resolved that way.
//
// SOURCE TRAVELS WITH THE VALUE. "Off" and "off because the platform says so"
// and "off because this tenant was switched off" send an operator to three
// different places, and a bare boolean sends them to the wrong one twice.
type FlagView struct {
	Key     string `json:"key"`
	Value   bool   `json:"value"`
	Source  string `json:"source" doc:"default, platform or organization — which level decided."`
	Scope   string `json:"scope" doc:"platform: one value for the installation. tenant: a platform default that an organization row overrides."`
	Default bool   `json:"default" doc:"What applies when neither level has a row."`
	Why     string `json:"why,omitempty"`
}

func viewOf(e control.EffectiveFlag) FlagView {
	return FlagView{
		Key: e.Key, Value: e.Value, Source: string(e.Source),
		Scope: string(e.Scope), Default: e.Default, Why: e.Why,
	}
}

type FlagListResponse struct {
	Body struct {
		Flags []FlagView `json:"flags"`
	}
}

type SetFlagRequest struct {
	Key  string `path:"key" maxLength:"120"`
	Body struct {
		Value bool `json:"value"`

		// AcknowledgeLockout is the operator saying they intend to remove a way
		// of signing in with nothing else configured.
		//
		// NO DEFAULT THAT MEANS YES. Absent is absent, and the guard refuses —
		// the same shape as a departure's credential disposition, and for the
		// same reason: this is a judgement only a person can make, and picking
		// one silently would be this system deciding what an operator can afford.
		AcknowledgeLockout bool `json:"acknowledge_lockout,omitempty" doc:"Proceed even though disabling this would leave no way to sign in. Only consulted for a flag that gates sign-in."`
	}
}

type SetOrgFlagRequest struct {
	OrgID string `path:"org_id" format:"uuid"`
	Key   string `path:"key" maxLength:"120"`
	Body  struct {
		Value bool `json:"value"`
	}
}

type ClearFlagRequest struct {
	Key string `path:"key" maxLength:"120"`
}

type ClearOrgFlagRequest struct {
	OrgID string `path:"org_id" format:"uuid"`
	Key   string `path:"key" maxLength:"120"`
}

type FlagResponse struct {
	Body FlagView
}

type OverrideView struct {
	OrganizationID string `json:"organization_id"`
	// Name travels with the id because this list is read by a person deciding
	// whether to change an installation-wide default, and "who will this not
	// reach" is not a question a column of uuids answers.
	OrganizationName string `json:"organization_name,omitempty"`
	Value            bool   `json:"value"`
}

type FlagOverridesResponse struct {
	Body struct {
		Key       string         `json:"key"`
		Overrides []OverrideView `json:"overrides"`
	}
}

// mapFlagErr turns the two registry refusals into distinguishable responses.
//
// An unknown key is 422: the request named something that does not exist and no
// amount of retrying helps. Setting a platform-scoped flag on one organization
// is 409: the flag is real and the request is coherent, it just conflicts with
// what that flag IS. Collapsing both into 422 would send an operator to check a
// spelling that was correct.
func mapFlagErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, control.ErrUnknownFlag):
		return huma.Error422UnprocessableEntity(err.Error() +
			" — flags are declared in code, so a key that is not in this build cannot be set")
	case errors.Is(err, control.ErrFlagNotTenantScoped):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, control.ErrWouldLockEveryoneOut):
		// 409, and the message carries the way forward. This is a refusal an
		// operator may legitimately override, so it has to say how — a bare
		// "conflict" would send them looking for a bug in their request.
		return huma.Error409Conflict(err.Error() +
			". Configure a sign-in provider or register a passkey first, or repeat this " +
			"request with acknowledge_lockout set if you mean to proceed anyway")
	default:
		return huma.Error500InternalServerError("could not apply the feature flag", err)
	}
}

// RegisterAdminProtected mounts the platform surface.
//
// It MUST be handed a group carrying RequireAdminAuth. No role check lives in
// these handlers, for the same reason as the policy surface: the group is the
// gate, so an endpoint added here is protected by existing rather than by its
// author remembering.
func RegisterAdminProtected(hapi huma.API, deps *app.Deps) {
	principal := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		return p, nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-flags-list",
		Method:      http.MethodGet,
		Path:        "/platform/flags",
		Summary:     "List every declared feature flag",
		Description: "Every flag this build declares, resolved as an organization with no " +
			"override of its own would see it.\n\n" +
			"Flags are declared IN CODE. The registry names the flag; the database supplies the " +
			"organization — so a key is never per-tenant, and adding a tenant adds no flags.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *struct{}) (*FlagListResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		// uuid.Nil: what a tenant with no opinion of its own gets.
		eff, err := deps.Flags.ListEffective(ctx, uuid.Nil)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		out := &FlagListResponse{}
		out.Body.Flags = make([]FlagView, 0, len(eff))
		for _, e := range eff {
			out.Body.Flags = append(out.Body.Flags, viewOf(e))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "platform-flag-set",
		Method:        http.MethodPut,
		Path:          "/platform/flags/{key}",
		Summary:       "Set a flag for the whole installation",
		DefaultStatus: http.StatusOK,
		Description: "Applies to every organization that has no override of its own.\n\n" +
			"A key this build does not declare is REFUSED. That is the reason flags are " +
			"registered: a free-form store accepts a mistyped key, reports success, and leaves " +
			"the real flag exactly as it was.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *SetFlagRequest) (*FlagResponse, error) {
		p, err := principal(ctx)
		if err != nil {
			return nil, err
		}
		if err := deps.Flags.SetPlatform(
			ctx, in.Key, in.Body.Value, *p.UserID, in.Body.AcknowledgeLockout,
		); err != nil {
			return nil, mapFlagErr(err)
		}
		eff, err := deps.Flags.Effective(ctx, uuid.Nil, in.Key)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		return &FlagResponse{Body: viewOf(*eff)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "platform-flag-clear",
		Method:        http.MethodDelete,
		Path:          "/platform/flags/{key}",
		Summary:       "Drop the installation-wide value, restoring the built-in default",
		DefaultStatus: http.StatusOK,
		Description: "Removing a row is not the same as setting it false: the flag falls back to " +
			"the default this build was written with, which is why an absent row never means off.\n\n" +
			"Clearing a flag that was not set SUCCEEDS. The requested state is reached either way.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ClearFlagRequest) (*FlagResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		if err := deps.Flags.ClearPlatform(ctx, in.Key); err != nil {
			return nil, mapFlagErr(err)
		}
		eff, err := deps.Flags.Effective(ctx, uuid.Nil, in.Key)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		return &FlagResponse{Body: viewOf(*eff)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "platform-org-flag-set",
		Method:        http.MethodPut,
		Path:          "/platform/organizations/{org_id}/flags/{key}",
		Summary:       "Set a flag for one organization",
		DefaultStatus: http.StatusOK,
		Description: "This is how a flag is targeted at one tenant. The organization is the ROW, " +
			"never part of the key — so a tenant rename rewrites nothing, and 'which tenants have " +
			"this on' is a query rather than a pattern match.\n\n" +
			"Refused for a platform-scoped flag: there is no per-tenant legacy login page.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *SetOrgFlagRequest) (*FlagResponse, error) {
		p, err := principal(ctx)
		if err != nil {
			return nil, err
		}
		orgID, perr := uuid.Parse(in.OrgID)
		if perr != nil {
			return nil, huma.Error422UnprocessableEntity("invalid organization id")
		}
		if err := deps.Flags.SetOrg(ctx, orgID, in.Key, in.Body.Value, *p.UserID); err != nil {
			return nil, mapFlagErr(err)
		}
		eff, err := deps.Flags.Effective(ctx, orgID, in.Key)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		return &FlagResponse{Body: viewOf(*eff)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "platform-org-flag-clear",
		Method:        http.MethodDelete,
		Path:          "/platform/organizations/{org_id}/flags/{key}",
		Summary:       "Drop one organization's override",
		DefaultStatus: http.StatusOK,
		Description:   "The organization falls back to the installation-wide value, or to the built-in default.",
		Tags:          []string{"Platform"},
		Security:      apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ClearOrgFlagRequest) (*FlagResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		orgID, perr := uuid.Parse(in.OrgID)
		if perr != nil {
			return nil, huma.Error422UnprocessableEntity("invalid organization id")
		}
		if err := deps.Flags.ClearOrg(ctx, orgID, in.Key); err != nil {
			return nil, mapFlagErr(err)
		}
		eff, err := deps.Flags.Effective(ctx, orgID, in.Key)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		return &FlagResponse{Body: viewOf(*eff)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-flag-overrides",
		Method:      http.MethodGet,
		Path:        "/platform/flags/{key}/overrides",
		Summary:     "Which organizations have set this flag themselves",
		Description: "The question to ask before changing an installation-wide default: these are " +
			"the organizations the change will NOT reach, because each has an opinion of its own.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ClearFlagRequest) (*FlagOverridesResponse, error) {
		if _, err := principal(ctx); err != nil {
			return nil, err
		}
		rows, err := deps.Flags.OverridesFor(ctx, in.Key)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		out := &FlagOverridesResponse{}
		out.Body.Key = in.Key
		out.Body.Overrides = make([]OverrideView, 0, len(rows))
		for _, r := range rows {
			out.Body.Overrides = append(out.Body.Overrides, OverrideView{
				OrganizationID:   r.OrganizationID.String(),
				OrganizationName: r.OrganizationName,
				Value:            r.Value,
			})
		}
		return out, nil
	})
}

// RegisterProtected mounts the tenant surface, which is READ ONLY.
//
// No write endpoint exists here at all — not one that checks a role and refuses.
// A route that does not exist cannot be reached by a caller who finds a hole in
// a role check, and there is no flag a tenant is currently allowed to set.
func RegisterProtected(hapi huma.API, deps *app.Deps) {
	huma.Register(hapi, huma.Operation{
		OperationID: "flags-list",
		Method:      http.MethodGet,
		Path:        "/flags",
		Summary:     "The feature flags in effect for this organization",
		Description: "Read only. Flags are set by platform staff; this says what applies here " +
			"and which level decided it.\n\n" +
			"Every declared flag appears, including the ones nobody has overridden — a flag " +
			"missing from this list would be indistinguishable from one that is switched off.",
		Tags:     []string{"Organization"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *struct{}) (*FlagListResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		eff, err := deps.Flags.ListEffective(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapFlagErr(err)
		}
		out := &FlagListResponse{}
		out.Body.Flags = make([]FlagView, 0, len(eff))
		for _, e := range eff {
			out.Body.Flags = append(out.Body.Flags, viewOf(e))
		}
		return out, nil
	})
}
