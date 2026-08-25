package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

type StartAuthRequest struct {
	Provider string `path:"provider" example:"google"`
	Mode     string `query:"mode" example:"spa"`
	Origin   string `query:"origin" example:"http://localhost:5173"`
}

type StartAuthResponse struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		AuthURL string `json:"auth_url"`
	}
}

type CallbackAuthRequest struct {
	Provider  string `path:"provider" example:"google"`
	Code      string `query:"code" required:"true"`
	State     string `query:"state"`
	UserAgent string `header:"User-Agent"`

	Cookie string `header:"Cookie"`
}

type CallbackAuthPayload struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	OrganizationID   string `json:"organization_id"`
	Organization     string `json:"organization"`
}

type CallbackAuthResponse struct {
	ContentType string      `header:"Content-Type"`
	SetCookie   http.Cookie `header:"Set-Cookie"`
	Body        []byte
}

func RegisterSocialAuth(hapi huma.API, deps *app.Deps) {
	socialRepo := repo.NewSocialAuthRepoGorm(deps.ControlDB)
	accessRuleRepo := repo.NewAuthAccessRuleRepoGorm(deps.ControlDB)
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)

	tokenSvc, err := service.NewTokenService(
		deps.Config.JWTSecret,
		deps.Config.JWTIssuer,
		time.Duration(deps.Config.JWTAccessTTLMin)*time.Minute,
		time.Duration(deps.Config.JWTRefreshTTLHour)*time.Hour,
	)
	if err != nil {
		panic(err)
	}

	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	// The quota limits used to be assembled here, which is precisely how this
	// path and registration came to enforce different rules. Both now share the
	// single gate built in cmd/serve.go.
	bootstrapSvc := service.NewOrgBootstrapService(deps.ControlDB, deps.Jobs, deps.Admission)
	socialSvc := service.NewSocialAuthService(
		deps.Config,
		socialRepo,
		accessRuleRepo,
		membershipRepo,
		totpRepo,
		tokenSvc,
		bootstrapSvc,
		deps.TokenSessions,
		deps.Policies,
		repo.NewOAuthStateRepoGorm(deps.ControlDB),
	)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-social-start",
		Method:      http.MethodPost,
		Path:        "/auth/{provider}/start",
		Summary:     "Begin social login",
		Description: "Returns the provider authorization URL for the frontend to redirect to.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *StartAuthRequest) (*StartAuthResponse, error) {
		authURL, stateCookie, err := socialSvc.StartURL(ctx, input.Provider, input.Mode, input.Origin)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrUnsupportedProvider),
				errors.Is(err, service.ErrProviderNotConfig):
				return nil, huma.Error400BadRequest("provider not configured")
			default:
				slog.ErrorContext(ctx, "failed to create auth url", "error", err)
				return nil, huma.Error500InternalServerError("failed to create auth url")
			}
		}

		out := &StartAuthResponse{}
		out.Body.AuthURL = authURL

		out.SetCookie = http.Cookie{
			Name:     service.OAuthStateCookieName,
			Value:    stateCookie,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   600,
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-social-callback",
		Method:      http.MethodGet,
		Path:        "/auth/{provider}/callback",
		Summary:     "Handle social login callback",
		Description: "Exchanges the provider code, links or creates the user account, auto-joins by email domain, and returns tokens.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *CallbackAuthRequest) (*CallbackAuthResponse, error) {
		result, err := socialSvc.Callback(ctx, input.Provider, input.Code, input.State, oauthStateCookie(input.Cookie), apimiddleware.ClientIPFromContext(ctx), input.UserAgent)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrUnsupportedProvider),
				errors.Is(err, service.ErrProviderNotConfig),
				errors.Is(err, service.ErrMissingCode):
				return nil, huma.Error400BadRequest(err.Error())
			case errors.Is(err, service.ErrInvalidOAuthState):
				return nil, huma.Error400BadRequest("invalid or expired oauth state")
			case errors.Is(err, service.ErrEmailRequired):
				return nil, huma.Error403Forbidden("a verified email is required for social sign-in")
			case errors.Is(err, repo.ErrAccessDenied):
				return nil, huma.Error403Forbidden("access denied by policy")
			case errors.Is(err, repo.ErrIdentityEmailNotVerified):
				return nil, huma.Error403Forbidden("the provider did not assert a verified email address")
			case errors.Is(err, repo.ErrSocialLinkNotAllowed):

				return nil, huma.Error409Conflict("an account already exists for this email address; sign in with your password and link the provider from your account settings")
			case errors.Is(err, service.ErrOrgCreationRateExceeded):

				return nil, huma.Error429TooManyRequests("too many new organizations were created recently; try again later")
			case errors.Is(err, service.ErrOrgQuotaExceeded):

				return nil, huma.Error403Forbidden("this deployment has reached its organization limit; contact an administrator")
			case errors.Is(err, service.ErrNoOrganization):
				return nil, huma.Error403Forbidden("user is not a member of any organization")
			case errors.Is(err, service.ErrUnprovenOrgCeiling):
				// One admission gate must not produce two HTTP contracts. The
				// register handler answers 403 here; falling through to the
				// default made the same decision a 500 on this path, which
				// tells the caller the server is broken and puts a deliberate
				// policy refusal on the operator's 5xx dashboard.
				return nil, huma.Error403Forbidden("this email domain already has too many organizations pending first use")
			case errors.Is(err, service.ErrUnresolvableDomain):
				return nil, huma.Error422UnprocessableEntity("that email address has no registrable domain")
			case errors.Is(err, service.ErrAdmissionNotConfigured):
				slog.ErrorContext(ctx, "admission gate is not configured", "error", err)
				return nil, huma.Error500InternalServerError("social login failed")
			default:
				slog.ErrorContext(ctx, "social login callback failed", "provider", input.Provider, "error", err)
				return nil, huma.Error500InternalServerError("social login failed")
			}
		}

		if result.MFARequired {
			mfaPayload := map[string]any{
				"mfa_required":    true,
				"mfa_token":       result.MFAToken,
				"organization_id": result.OrganizationID,
				"organization":    result.Organization,
			}

			mode, origin := result.Mode, result.Origin
			if mode == "spa" {
				origin = allowedOrigin(origin, deps.Config)
				if origin == "" {
					origin = canonicalOrigin(deps.Config.BaseURL)
				}
				b, err := json.Marshal(mfaPayload)
				if err != nil {
					slog.ErrorContext(ctx, "failed to marshal mfa payload", "error", err)
					return nil, huma.Error500InternalServerError("failed to encode callback response")
				}
				return &CallbackAuthResponse{
					ContentType: "text/html; charset=utf-8",
					Body:        writePostMessageHTMLRaw(origin, b),
				}, nil
			}

			b, err := json.Marshal(mfaPayload)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to encode callback response")
			}
			return &CallbackAuthResponse{
				ContentType: "application/json",
				Body:        b,
			}, nil
		}

		payload := &CallbackAuthPayload{
			AccessToken:      result.Tokens.AccessToken,
			RefreshToken:     result.Tokens.RefreshToken,
			TokenType:        result.Tokens.TokenType,
			ExpiresIn:        result.Tokens.ExpiresIn,
			RefreshExpiresIn: result.Tokens.RefreshExpiresIn,
			OrganizationID:   result.OrganizationID,
			Organization:     result.Organization,
		}

		// Secure is derived from the ENVIRONMENT, not from whether a configured
		// URL happens to start with https. The old test read BaseURL and
		// OAuthRedirectBase, so a production deployment with either mistyped as
		// http:// shipped a bearer-token cookie over plaintext — a
		// misconfiguration silently downgrading a security attribute, which is
		// the shape of failure this codebase keeps finding. Only a development
		// environment may omit it, so an unset or misspelt ENV stays secure.
		secure := !config.IsDevEnv(deps.Config.Env)

		out := &CallbackAuthResponse{

			// nolint:gosec // G124 wants Secure as a literal true, which would
			// make the cookie undeliverable over plain http in development. It
			// IS set everywhere else — `secure` above is false only for
			// IsDevEnv — alongside HttpOnly and SameSite=Lax.
			//nolint:gosec
			SetCookie: http.Cookie{
				Name:     "ag_jwt",
				Value:    "Bearer " + result.Tokens.AccessToken,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   secure,
				MaxAge:   result.Tokens.ExpiresIn,
			},
		}

		mode, origin := result.Mode, result.Origin
		if mode == "spa" {
			origin = allowedOrigin(origin, deps.Config)
			if origin == "" {
				origin = canonicalOrigin(deps.Config.BaseURL)
			}
			out.ContentType = "text/html; charset=utf-8"
			htmlBody, err := writePostMessageHTML(origin, *payload)
			if err != nil {
				slog.ErrorContext(ctx, "failed to marshal callback payload", "error", err)
				return nil, huma.Error500InternalServerError("failed to encode callback response")
			}
			out.Body = htmlBody
			return out, nil
		}

		b, err := json.Marshal(payload)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to encode callback response")
		}

		out.ContentType = "application/json"
		out.Body = b
		return out, nil
	})
}

func oauthStateCookie(raw string) string {
	header := http.Header{}
	header.Add("Cookie", raw)
	c, err := (&http.Request{Header: header}).Cookie(service.OAuthStateCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func canonicalOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func allowedOrigin(candidate string, cfg *config.Config) string {
	candidate = canonicalOrigin(candidate)
	if candidate == "" {
		return ""
	}

	allowed := map[string]struct{}{}
	for _, raw := range []string{cfg.BaseURL, cfg.ViteDevURL, cfg.OAuthRedirectBase} {
		if o := canonicalOrigin(raw); o != "" {
			allowed[o] = struct{}{}

			if alt := localhostAlternate(o); alt != "" {
				allowed[alt] = struct{}{}
			}
		}
	}

	if _, ok := allowed[candidate]; ok {
		return candidate
	}
	return ""
}

func localhostAlternate(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	port := u.Port()
	var altHost string
	switch host {
	case "localhost":
		altHost = "127.0.0.1"
	case "127.0.0.1":
		altHost = "localhost"
	default:
		return ""
	}
	alt := u.Scheme + "://" + altHost
	if port != "" {
		alt += ":" + port
	}
	return alt
}

func writePostMessageHTMLRaw(origin string, jsonPayload []byte) []byte {
	html := `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Authentication complete</title></head>
<body>
<script>
(function () {
  var payload = ` + string(jsonPayload) + `;
  var origin = ` + strconv.Quote(origin) + `;
  if (window.opener && origin) {
    window.opener.postMessage({ type: "` + OAuthCompleteMessageType + `", payload: payload }, origin);
    window.close();
    return;
  }
  document.body.innerText = "Authentication complete";
})();
</script>
</body>
</html>`
	return []byte(html)
}

func writePostMessageHTML(origin string, payload CallbackAuthPayload) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal callback payload: %w", err)
	}
	html := `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Authentication complete</title></head>
<body>
<script>
(function () {
  var payload = ` + string(b) + `;
  var origin = ` + strconv.Quote(origin) + `;
  if (window.opener && origin) {
    window.opener.postMessage({ type: "` + OAuthCompleteMessageType + `", payload: payload }, origin);
    window.close();
    return;
  }
  document.body.innerText = "Authentication complete";
})();
</script>
</body>
</html>`
	return []byte(html), nil
}

// OAuthCompleteMessageType is the postMessage `type` the social-auth popup posts
// to its opener when the flow finishes.
//
// It is a CONTRACT with the browser, not a label: ui/src/providers/auth-provider.tsx
// filters on this exact string, and a mismatch fails silently — the popup posts,
// nobody listens, the window closes, and the user is left on the login page with
// no error anywhere. That is precisely what happened when the product was
// renamed: the listener was updated to "apiary:oauth:complete" by a rename that
// swept .ts/.tsx files, while this string sat inside a JavaScript template
// literal in a .go file and was missed.
//
// TestOAuthCompleteMessageTypeMatchesTheUI pins the two together by reading the
// UI source, so the next rename cannot split them again.
const OAuthCompleteMessageType = "apiary:oauth:complete"
