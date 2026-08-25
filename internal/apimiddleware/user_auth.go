package apimiddleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// RequireUserAuth admits the three HUMAN-or-automation credential shapes: a
// session bearer token, a user API key, and an organization key/secret pair.
//
// It deliberately does not admit the fourth. An agent credential is refused here
// because Authenticate is asked for a specific scope, and the exclusion runs
// both ways: a compromised hypervisor must not be able to turn its credential on
// the rest of the tenant's API, and a human or an automation must not be able to
// forge an agent.
// RequireAuthenticated establishes WHO is calling and stops there.
//
// It is the pre-authz half of the split: the token must be a valid, live access
// token naming a real session, but the policy gate is NOT applied. A principal
// it admits may carry PolicySatisfied=false.
//
// Mount only remediation endpoints behind it — the ones whose whole purpose is
// to let someone comply with the policy refusing them. Anything else mounted
// here is reachable by a caller the policy has already declined, which is the
// bypass this exists to avoid creating.
//
// Everything except the gate still applies. The signature, the token type, the
// issuer, session liveness and the impersonation shape are all identity
// questions, and relaxing authorization must not relax authentication.
func RequireAuthenticated(api huma.API, deps *app.Deps) func(huma.Context, func(huma.Context)) {
	return authenticate(api, deps, false)
}

// RequireUserAuth is RequireAuthenticated plus the policy gate.
func RequireUserAuth(api huma.API, deps *app.Deps) func(huma.Context, func(huma.Context)) {
	return authenticate(api, deps, true)
}

func authenticate(api huma.API, deps *app.Deps, enforcePolicy bool) func(huma.Context, func(huma.Context)) {
	var memberships repo.MembershipRepo
	var keys *service.APIKeyService
	if deps != nil {
		if deps.ControlDB != nil {
			memberships = repo.NewMembershipRepoGorm(deps.ControlDB)
		}
		keys = deps.APIKeys
	}

	return func(hctx huma.Context, next func(huma.Context)) {
		presented, err := credentialFrom(hctx)
		if err != nil {
			writeKeyAuthError(api, hctx, err)
			return
		}

		// A key credential short-circuits: it is already a complete principal, and
		// none of the JWT handling below applies to it.
		ctx := withOrgIDHeader(hctx.Context(), hctx.Header(OrgIDHeader))
		if p, err := resolveAPIKeyPrincipal(ctx, keys, memberships, presented); err != nil {
			writeKeyAuthError(api, hctx, err)
			return
		} else if p != nil {
			p.PolicySatisfied = true
			next(huma.WithContext(hctx, WithPrincipal(hctx.Context(), p)))
			return
		}

		tokenStr := presented.bearer
		if tokenStr == "" {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "missing credential")
			return
		}

		claims := &service.AuthClaims{}
		token, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(deps.Config.JWTSecret), nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(deps.Config.JWTIssuer),
		)
		if err != nil || !token.Valid || claims.Type != "access" {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		// THE policy gate, and the only thing the pre-authz path relaxes.
		//
		// mfa_required is not "your credentials are wrong" — it is "you are who
		// you say, and policy will not let you act yet". Which is why the endpoints
		// that let someone comply cannot live behind it.
		if enforcePolicy && claims.MFAEnabled && !claims.MFA {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "mfa_required")
			return
		}

		userID, err := uuid.Parse(strings.TrimSpace(claims.Subject))
		if err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid bearer token")
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
			// Satisfied when the policy has nothing outstanding, whether because
			// none applies or because the second factor was presented. Derived from
			// the claims rather than from which middleware ran, so the two cannot
			// disagree.
			PolicySatisfied: !claims.MFAEnabled || claims.MFA,
		}

		// A token that CLAIMS impersonation but whose actor cannot be resolved is
		// refused outright. Swallowing the parse error left Impersonating=false, so
		// RefuseImpersonation() never fired and an impersonated session could
		// register a tenant cloud credential -- a fail-open on the guard protecting
		// the highest-value write in the system. The only issuer of imp=true sets
		// act_id from a uuid.UUID (impersonation_service.go), and impersonated
		// refresh is rejected outright, so no legitimate token can reach this arm.
		if claims.Impersonate {
			actorID, err := uuid.Parse(strings.TrimSpace(claims.ActorID))
			if err != nil {
				_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid bearer token")
				return
			}
			p.ActorID = &actorID
			p.Impersonating = true
		}

		hctx = huma.WithContext(hctx, WithPrincipal(hctx.Context(), p))
		next(hctx)
	}
}
