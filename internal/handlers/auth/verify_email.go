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

// newEmailVerificationService builds the service both this file and the
// registration handler need.
//
// It is a shared constructor rather than two independent ones so that the two
// cannot drift — registration issuing tokens with one TTL while confirmation
// reads another would be invisible until links started expiring early.
func newEmailVerificationService(deps *app.Deps) *service.EmailVerificationService {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	return service.NewEmailVerificationService(
		repo.NewEmailVerificationRepoGorm(deps.ControlDB),
		&userRepo,
		repo.NewMembershipRepoGorm(deps.ControlDB),
		deps.MailSender(),
		deps.Jobs,
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
		deps.Config.BaseURL,
	)
}

type VerifyEmailRequest struct {
	Body struct {
		Token string `json:"token" required:"true" minLength:"1" maxLength:"128" doc:"The token from the confirmation link."`
	}
}

type VerifyEmailResponse struct {
	Body struct {
		Verified           bool   `json:"verified"`
		AlreadyVerified    bool   `json:"already_verified" doc:"True when this address was already confirmed — a second click on the same link."`
		TenantProvisioning bool   `json:"tenant_provisioning" doc:"True when confirming released provisioning of the organization's database."`
		Message            string `json:"message"`
	}
}

type ResendVerificationRequest struct {
	Body struct {
		Email string `json:"email" required:"true" format:"email" maxLength:"255"`
	}
}

type ResendVerificationResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func RegisterEmailVerification(hapi huma.API, deps *app.Deps) {
	svc := newEmailVerificationService(deps)

	huma.Register(hapi, huma.Operation{
		OperationID:   "auth-verify-email",
		Method:        http.MethodPost,
		Path:          "/auth/verify-email",
		Summary:       "Confirm an email address",
		DefaultStatus: http.StatusOK,
		Description: "Redeems the token from a confirmation link. This is what releases provisioning of the organization's tenant database — registration deliberately creates nothing but rows until an address is confirmed. " +
			"No session is issued: sign in through /auth/login afterwards.",
		Tags: []string{"Auth"},
	}, func(ctx context.Context, in *VerifyEmailRequest) (*VerifyEmailResponse, error) {
		result, err := svc.Confirm(ctx, in.Body.Token)
		if err != nil {
			if errors.Is(err, service.ErrVerificationTokenInvalid) {
				// One answer for expired, used, unknown and issued-for-another-
				// address. Distinguishing them would let a caller probe tokens
				// they do not hold.
				return nil, huma.Error400BadRequest("this confirmation link is invalid or has expired")
			}
			slog.ErrorContext(ctx, "email confirmation failed", "error", err)
			return nil, huma.Error500InternalServerError("could not confirm this address")
		}

		out := &VerifyEmailResponse{}
		out.Body.Verified = true
		out.Body.AlreadyVerified = result.AlreadyVerified
		out.Body.TenantProvisioning = result.TenantProvisioning
		if result.AlreadyVerified {
			out.Body.Message = "This address was already confirmed. You can sign in."
		} else {
			out.Body.Message = "Your address is confirmed. You can sign in now."
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "auth-resend-verification",
		Method:        http.MethodPost,
		Path:          "/auth/resend-verification",
		Summary:       "Send a new confirmation link",
		DefaultStatus: http.StatusAccepted,
		Description: "Always answers 202, whatever the address. An unauthenticated endpoint that answered differently for a registered address than an unregistered one would be a membership oracle. " +
			"Sending is throttled per account.",
		Tags: []string{"Auth"},
	}, func(ctx context.Context, in *ResendVerificationRequest) (*ResendVerificationResponse, error) {
		// Resend swallows every non-wiring failure by design, so an error here
		// means the service itself is not configured — not that the address was
		// rejected.
		if err := svc.Resend(ctx, in.Body.Email); err != nil {
			slog.ErrorContext(ctx, "resend verification is not operational", "error", err)
		}
		out := &ResendVerificationResponse{}
		out.Body.Message = "If that address needs confirming, a new link is on its way."
		return out, nil
	})
}
