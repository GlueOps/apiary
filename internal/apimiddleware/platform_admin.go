package apimiddleware

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/google/uuid"
)

func RequirePlatformAdmin(api huma.API, deps *app.Deps) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		tokenStr := bearerToken(hctx)
		if tokenStr == "" {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "missing bearer token")
			return
		}

		claims, err := parseAccessClaims(tokenStr, deps)
		if err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		if claims.Impersonate {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "impersonated sessions may not use admin routes")
			return
		}

		userID, err := isPlatformAdmin(hctx.Context(), deps, claims)
		if err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "platform admin access required")
			return
		}

		if err := checkSessionLiveness(hctx.Context(), deps, claims, userID); err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "session is no longer valid")
			return
		}

		orgID, err := uuid.Parse(strings.TrimSpace(claims.OrgID))
		if err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		p := &Principal{
			Method:         AuthMethodBearer,
			UserID:         &userID,
			OrganizationID: orgID,
			SessionJTI:     claims.ID,
		}

		hctx = huma.WithContext(hctx, WithPrincipal(hctx.Context(), p))
		next(hctx)
	}
}
