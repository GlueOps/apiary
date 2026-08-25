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

type ForgotPasswordRequest struct {
	Body struct {
		Email string `json:"email" required:"true" format:"email"`
	}
}

type ForgotPasswordResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

type ResetPasswordRequest struct {
	Body struct {
		Token       string `json:"token" required:"true" minLength:"64" maxLength:"64"`
		NewPassword string `json:"new_password" required:"true" minLength:"8"`
	}
}

type ResetPasswordResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func RegisterPasswordReset(hapi huma.API, deps *app.Deps) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)

	resetSvc := service.NewPasswordResetService(
		repo.NewPasswordResetRepoGorm(deps.ControlDB),
		&userRepo,
		deps.MailSender(),
		deps.Config.BaseURL,
		deps.TokenSessions,
		deps.Policies,
		repo.NewMembershipRepoGorm(deps.ControlDB),
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
	)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-forgot-password",
		Method:      http.MethodPost,
		Path:        "/auth/forgot-password",
		Summary:     "Request a password reset link",
		Description: "Sends a password reset link if the address belongs to an active account. The response is identical either way.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {

		if err := resetSvc.RequestReset(ctx, input.Body.Email); err != nil {
			slog.ErrorContext(ctx, "password reset request failed", "error", err)
			return nil, huma.Error500InternalServerError("could not process the request")
		}

		out := &ForgotPasswordResponse{}
		out.Body.Message = "if that address belongs to an account, a reset link is on its way"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-reset-password",
		Method:      http.MethodPost,
		Path:        "/auth/reset-password",
		Summary:     "Set a new password using a reset token",
		Description: "Consumes a single-use reset token, sets the new password, and revokes every existing session for the account.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *ResetPasswordRequest) (*ResetPasswordResponse, error) {
		err := resetSvc.ResetPassword(ctx, input.Body.Token, input.Body.NewPassword)
		switch {
		case err == nil:
		case errors.Is(err, service.ErrResetTokenInvalid):
			return nil, huma.Error400BadRequest("reset token is invalid or expired")
		case errors.Is(err, service.ErrPasswordPolicy):

			return nil, huma.Error422UnprocessableEntity(err.Error())
		default:
			slog.ErrorContext(ctx, "password reset failed", "error", err)
			return nil, huma.Error500InternalServerError("could not reset the password")
		}

		out := &ResetPasswordResponse{}
		out.Body.Message = "password updated; all sessions have been signed out"
		return out, nil
	})
}
