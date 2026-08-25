package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

type RegisterRequest struct {
	XRealIP   string `header:"X-Real-Ip"`
	UserAgent string `header:"User-Agent"`
	Body      struct {
		Email            string `json:"email" required:"true" format:"email" maxLength:"255"`
		Password         string `json:"password" required:"true" minLength:"8" maxLength:"128"`
		FirstName        string `json:"first_name" required:"true" minLength:"1" maxLength:"100"`
		LastName         string `json:"last_name,omitempty" maxLength:"100"`
		OrganizationName string `json:"organization_name,omitempty" maxLength:"255" doc:"Blank derives a workspace name from the email."`
	}
}

// RegisterResponse carries no tokens. Registration is not a sign-in.
//
// The address has to be confirmed first, and confirmation issues no session
// either — the registrant then signs in through /auth/login like anyone else.
// Keeping session minting on exactly one path means the checks that guard it
// are written once.
type RegisterResponse struct {
	Body struct {
		OrganizationID   string `json:"organization_id"`
		Organization     string `json:"organization"`
		VerificationSent bool   `json:"verification_sent" doc:"False means the confirmation email could not be handed to the mailer; the account exists and /auth/resend-verification is the recovery path."`
		Message          string `json:"message"`
	}
}

func RegisterSignup(hapi huma.API, deps *app.Deps) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)

	svc := service.NewRegistrationService(
		deps.ControlDB,
		&userRepo,
		deps.Policies,
		repo.NewAuthAccessRuleRepoGorm(deps.ControlDB),
		deps.Admission,
		newEmailVerificationService(deps),
	)

	huma.Register(hapi, huma.Operation{
		OperationID:   "auth-register",
		Method:        http.MethodPost,
		Path:          "/auth/register",
		Summary:       "Create an account and its first organization",
		DefaultStatus: http.StatusCreated,
		Description: "Creates the user, an organization, and the owner membership binding them — all in one transaction, because a user with no organization cannot sign in at all. " +
			"Returns NO tokens: the address must be confirmed first, and confirming issues no session either — sign in through /auth/login afterwards. " +
			"The organization's tenant database is not provisioned until the address is confirmed, so an unconfirmed signup costs a row and nothing else.",
		Tags: []string{"Auth"},
	}, func(ctx context.Context, input *RegisterRequest) (*RegisterResponse, error) {
		result, err := svc.Register(ctx, service.RegisterInput{
			Email:            input.Body.Email,
			Password:         input.Body.Password,
			FirstName:        input.Body.FirstName,
			LastName:         input.Body.LastName,
			OrganizationName: input.Body.OrganizationName,
			IPAddress:        input.XRealIP,
			UserAgent:        input.UserAgent,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmailTaken):
				return nil, huma.Error409Conflict("that email is already registered")
			case errors.Is(err, service.ErrWeakPassword):
				// The policy's own wording says which rule failed, which is the
				// whole point of surfacing it.
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, service.ErrRegistrationName):
				return nil, huma.Error422UnprocessableEntity("a name is required")
			case errors.Is(err, repo.ErrAccessDenied):
				// Deliberately does not name the rule, its kind or its value:
				// the response must not be usable as an oracle for probing the
				// access-rule table.
				return nil, huma.Error403Forbidden("registration is not permitted for this address")
			case errors.Is(err, service.ErrUnprovenOrgCeiling):
				// 403, not 429. The ceiling has no time component, so a
				// Retry-After semantic would be a lie: waiting frees nothing,
				// only an organization becoming proven does.
				return nil, huma.Error403Forbidden("this email domain already has too many organizations pending first use")
			case errors.Is(err, service.ErrUnresolvableDomain):
				return nil, huma.Error422UnprocessableEntity("that email address has no registrable domain")
			case errors.Is(err, service.ErrOrgQuotaExceeded),
				errors.Is(err, service.ErrOrgCreationRateExceeded):
				return nil, huma.Error429TooManyRequests("organization creation is temporarily unavailable")
			case errors.Is(err, service.ErrAdmissionNotConfigured):
				slog.ErrorContext(ctx, "admission gate is not wired; refusing registration", "error", err)
				return nil, huma.Error500InternalServerError("registration failed")
			default:
				slog.ErrorContext(ctx, "registration failed", "error", err)
				return nil, huma.Error500InternalServerError("registration failed")
			}
		}

		out := &RegisterResponse{}
		out.Body.OrganizationID = result.OrganizationID
		out.Body.Organization = result.Organization
		out.Body.VerificationSent = result.VerificationSent
		out.Body.Message = "Check your email for a confirmation link, then sign in."
		return out, nil
	})
}
