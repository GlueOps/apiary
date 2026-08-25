package apimiddleware

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

// RequireAdminAuth admits only platform administrators.
//
// It is a MIDDLEWARE rather than a check inside each handler because the route
// group is the honest place for an authorization boundary: an endpoint added to
// the admin group is protected by existing, and an endpoint that must not be
// admin-only cannot drift into being so by someone copying a neighbouring
// handler's first three lines. The previous arrangement — AssertPlatformRole
// called at the top of every handler — protects exactly the handlers whose
// author remembered.
//
// It must be layered AFTER RequireUserAuth: it reads the principal that
// establishes, and a group carrying only this middleware would admit anyone,
// because there would be no principal to reject.
//
// The three platform roles are not interchangeable, and this gate is the
// narrowest of them:
//
//	admin    the platform operator. Everything, including policy that binds
//	         every tenant. This gate.
//	support  may impersonate a member who has consented (membership
//	         support_enabled), and nothing else. NOT admitted here — a support
//	         engineer helping one customer must not be able to change the
//	         security floor for all of them.
//	customer the tenant's own people, authorized through their membership.
func RequireAdminAuth(api huma.API, deps *app.Deps) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		ctx := hctx.Context()

		p, ok := PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			// No principal means RequireUserAuth did not run, or ran and failed.
			// Either way this cannot reach a decision, so it denies.
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "not authenticated")
			return
		}

		// An impersonated session is refused whatever role it carries.
		// Impersonation issues a session whose SUBJECT is the target, so the role
		// read below would be the target's: a support engineer impersonating a
		// customer who happens to be platform staff would otherwise inherit god
		// mode from the person they are helping.
		if p.IsImpersonating() {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden,
				"an impersonated session may not perform platform administration")
			return
		}

		if deps == nil || deps.ControlDB == nil {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "insufficient permissions")
			return
		}
		users := repo.NewUserRepoGorm(deps.ControlDB)
		user, err := users.GetByID(ctx, *p.UserID)
		if err != nil || user == nil || !user.IsActive || user.Role != control.RolePlatformAdmin {
			// One answer for "not an admin", "inactive", and "the lookup failed".
			// Distinguishing them would let any authenticated user probe who the
			// platform administrators are.
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "insufficient permissions")
			return
		}

		next(hctx)
	}
}
