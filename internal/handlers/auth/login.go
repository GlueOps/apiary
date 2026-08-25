package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

type LoginRequest struct {
	UserAgent string `header:"User-Agent"`
	Body      struct {
		Email    string `json:"email" required:"true" format:"email"`
		Password string `json:"password" required:"true"`
	}
}

type LoginResponse struct {
	Body struct {
		AccessToken      string `json:"access_token,omitempty"`
		RefreshToken     string `json:"refresh_token,omitempty"`
		TokenType        string `json:"token_type,omitempty"`
		ExpiresIn        int    `json:"expires_in,omitempty"`
		RefreshExpiresIn int    `json:"refresh_expires_in,omitempty"`
		OrganizationID   string `json:"organization_id"`
		Organization     string `json:"organization"`
		MFARequired      bool   `json:"mfa_required,omitempty"`
		MFAToken         string `json:"mfa_token,omitempty"`
	}
}

func RegisterLogin(hapi huma.API, deps *app.Deps) {
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)

	loginSvc := service.NewLoginService(&userRepo, membershipRepo, totpRepo, deps.TokenSessions.Tokens(), deps.TokenSessions, deps.Policies, repo.NewAuthChallengeRepoGorm(deps.ControlDB), repo.NewAuthFailureRepoGorm(deps.ControlDB))

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Authenticate a user",
		Description: "Validates email and password, selects the user's organization, and returns JWT access and refresh tokens.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *LoginRequest) (*LoginResponse, error) {
		result, err := loginSvc.Login(ctx, service.LoginInput{
			Email:     input.Body.Email,
			Password:  input.Body.Password,
			IPAddress: apimiddleware.ClientIPFromContext(ctx),
			UserAgent: input.UserAgent,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrAccountLocked):

				return nil, huma.Error429TooManyRequests("too many failed attempts; try again later")
			case errors.Is(err, service.ErrInvalidCredentials):
				return nil, huma.Error401Unauthorized("invalid email or password")
			case errors.Is(err, service.ErrUserInactive):
				return nil, huma.Error403Forbidden("user account is inactive")
			case errors.Is(err, service.ErrEmailNotVerified):
				// Its own status, its own wording, and its own machine-readable
				// TYPE, so the client can offer a resend instead of "check your
				// password". Only a caller who has already supplied the correct
				// password ever sees it — the gate sits after the password check
				// for exactly that reason — so it discloses nothing to an
				// anonymous prober.
				//
				// The type URI is what the UI branches on. Matching the prose
				// instead would work right up until someone rewords the message,
				// and then the resend affordance would silently disappear with
				// nothing failing anywhere. RFC 7807 exists for this.
				return nil, emailNotVerifiedError()
			case errors.Is(err, service.ErrNoOrganization):
				return nil, huma.Error403Forbidden("user is not a member of any organization")
			case errors.Is(err, repo.ErrAccessDenied):
				// The authenticate-purpose ban list is the whole point of the
				// purpose split; unmapped it reported every banned login as a
				// server error. Deliberately vague, so the response cannot be
				// used to probe the rule table.
				return nil, huma.Error403Forbidden("access denied by policy")
			default:
				slog.ErrorContext(ctx, "login failed", "error", err)
				return nil, huma.Error500InternalServerError("login failed")
			}
		}

		out := &LoginResponse{}
		out.Body.OrganizationID = result.OrganizationID
		out.Body.Organization = result.Organization

		if result.MFARequired {
			out.Body.MFARequired = true
			out.Body.MFAToken = result.MFAToken
			return out, nil
		}

		out.Body.AccessToken = result.Tokens.AccessToken
		out.Body.RefreshToken = result.Tokens.RefreshToken
		out.Body.TokenType = result.Tokens.TokenType
		out.Body.ExpiresIn = result.Tokens.ExpiresIn
		out.Body.RefreshExpiresIn = result.Tokens.RefreshExpiresIn
		return out, nil
	})
}

// ErrTypeEmailNotVerified identifies the "confirm your address first" refusal in
// a way that survives rewording. It is part of the API contract: the UI routes
// on it, so changing the string is a breaking change.
const ErrTypeEmailNotVerified = "https://apiary.glueops.dev/errors/email-not-verified"

func emailNotVerifiedError() error {
	err := huma.Error403Forbidden("confirm your email address before signing in")
	var em *huma.ErrorModel
	if errors.As(err, &em) {
		em.Type = ErrTypeEmailNotVerified
	}
	return err
}
