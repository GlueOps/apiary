package me

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

type GetMeRequest struct{}

type GetMeResponse struct {
	Body struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		IsActive  bool   `json:"is_active"`
		MFA       struct {
			Enabled     bool `json:"enabled"`
			TOTPEnabled bool `json:"totp_enabled"`
		} `json:"mfa"`
		Organization struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"organization"`
		LastLoginAt *time.Time `json:"last_login_at,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
	}
}

type UpdateMeRequest struct {
	Body struct {
		FirstName *string `json:"first_name,omitempty"`
		LastName  *string `json:"last_name,omitempty"`
		Email     *string `json:"email,omitempty" format:"email"`
	}
}

type UpdateMeResponse struct {
	Body struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		IsActive  bool   `json:"is_active"`
	}
}

type ChangePasswordRequest struct {
	Body struct {
		CurrentPassword string `json:"current_password" required:"true" minLength:"1"`
		NewPassword     string `json:"new_password" required:"true" minLength:"8"`
	}
}

type ChangePasswordResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func Register(hapi huma.API, deps *app.Deps) {
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)
	orgRepo := repo.NewOrganizationRepoGorm(deps.ControlDB)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)

	profileSvc := service.NewProfileService(new(repo.NewUserRepoGorm(deps.ControlDB)), orgRepo, membershipRepo, totpRepo, deps.Policies, deps.TokenSessions, repo.NewAuthAuditRepoGorm(deps.ControlDB))
	profileSvc.SetEmailChange(repo.NewEmailChangeRepoGorm(deps.ControlDB), deps.MailSender(), deps.Config.BaseURL)

	huma.Register(hapi, huma.Operation{
		OperationID: "me-get",
		Method:      http.MethodGet,
		Path:        "/me",
		Summary:     "Get current user profile",
		Description: "Returns the authenticated user's profile, MFA status, and organization.",
		Tags:        []string{"Profile"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *GetMeRequest) (*GetMeResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		result, err := profileSvc.GetProfile(ctx, *p.UserID, p.OrganizationID)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				return nil, huma.Error404NotFound("user not found")
			}
			slog.ErrorContext(ctx, "failed to load profile", "error", err)
			return nil, huma.Error500InternalServerError("failed to load profile")
		}

		out := &GetMeResponse{}
		out.Body.ID = result.User.ID.String()
		out.Body.Email = result.User.Email
		out.Body.FirstName = result.User.FirstName
		out.Body.LastName = result.User.LastName
		out.Body.Role = string(result.User.Role)
		out.Body.IsActive = result.User.IsActive
		out.Body.MFA.Enabled = result.MFAEnabled
		out.Body.MFA.TOTPEnabled = result.TOTPEnabled
		out.Body.LastLoginAt = result.User.LastLoginAt
		out.Body.CreatedAt = result.User.CreatedAt
		if result.Organization != nil {
			out.Body.Organization.ID = result.Organization.ID.String()
			out.Body.Organization.Name = result.Organization.Name
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "me-update",
		Method:      http.MethodPatch,
		Path:        "/me",
		Summary:     "Update current user profile",
		Description: "Update the authenticated user's first name, last name, or email.",
		Tags:        []string{"Profile"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, input *UpdateMeRequest) (*UpdateMeResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		user, err := profileSvc.UpdateProfile(ctx, *p.UserID, service.UpdateProfileInput{
			FirstName: input.Body.FirstName,
			LastName:  input.Body.LastName,
			Email:     input.Body.Email,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrNothingToUpdate):
				return nil, huma.Error400BadRequest("no fields to update")
			case errors.Is(err, service.ErrEmailChangeRequiresVerification):

				return nil, huma.Error400BadRequest(
					"email cannot be changed here; POST /me/email to receive a confirmation link")
			}
			slog.ErrorContext(ctx, "failed to update profile", "error", err)
			return nil, huma.Error500InternalServerError("failed to update profile")
		}

		out := &UpdateMeResponse{}
		out.Body.ID = user.ID.String()
		out.Body.Email = user.Email
		out.Body.FirstName = user.FirstName
		out.Body.LastName = user.LastName
		out.Body.Role = string(user.Role)
		out.Body.IsActive = user.IsActive
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "me-change-password",
		Method:      http.MethodPut,
		Path:        "/me/password",
		Summary:     "Change password",
		Description: "Change the authenticated user's password. Requires the current password.",
		Tags:        []string{"Profile"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, input *ChangePasswordRequest) (*ChangePasswordResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		err := profileSvc.ChangePassword(ctx, *p.UserID, service.ChangePasswordInput{
			CurrentPassword: input.Body.CurrentPassword,
			NewPassword:     input.Body.NewPassword,
			OrgID:           p.OrganizationID,
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrWrongPassword):
				return nil, huma.Error403Forbidden("current password is incorrect")
			case errors.Is(err, service.ErrUserNotFound):
				return nil, huma.Error404NotFound("user not found")
			default:
				slog.ErrorContext(ctx, "failed to change password", "error", err)
				return nil, huma.Error500InternalServerError("failed to change password")
			}
		}

		out := &ChangePasswordResponse{}
		out.Body.Message = "password changed"
		return out, nil
	})
}

type RequestEmailChangeRequest struct {
	Body struct {
		Email string `json:"email" required:"true" format:"email"`
	}
}

type RequestEmailChangeResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

type ConfirmEmailChangeRequest struct {
	Body struct {
		Token string `json:"token" required:"true"`
	}
}

type ConfirmEmailChangeResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func RegisterEmailChange(hapi huma.API, deps *app.Deps) {
	orgRepo := repo.NewOrganizationRepoGorm(deps.ControlDB)
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)

	profileSvc := service.NewProfileService(
		new(repo.NewUserRepoGorm(deps.ControlDB)),
		orgRepo, membershipRepo, totpRepo,
		deps.Policies, deps.TokenSessions,
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
	)
	profileSvc.SetEmailChange(repo.NewEmailChangeRepoGorm(deps.ControlDB), deps.MailSender(), deps.Config.BaseURL)

	huma.Register(hapi, huma.Operation{
		OperationID: "me-request-email-change",
		Method:      http.MethodPost,
		Path:        "/me/email",
		Summary:     "Request an email address change",
		Description: "Sends a confirmation link to the new address. The current address keeps working until it is confirmed.",
		Tags:        []string{"Me"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, input *RequestEmailChangeRequest) (*RequestEmailChangeResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		if err := profileSvc.RequestEmailChange(ctx, *p.UserID, input.Body.Email); err != nil {
			if errors.Is(err, service.ErrNothingToUpdate) {
				return nil, huma.Error400BadRequest("an email address is required")
			}
			slog.ErrorContext(ctx, "failed to request email change", "error", err)
			return nil, huma.Error500InternalServerError("failed to request email change")
		}

		out := &RequestEmailChangeResponse{}
		out.Body.Message = "if that address can receive mail, a confirmation link has been sent to it"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "me-confirm-email-change",
		Method:      http.MethodPost,
		Path:        "/me/email/confirm",
		Summary:     "Confirm an email address change",
		Description: "Redeems the token mailed to the new address and applies the change.",
		Tags:        []string{"Me"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, input *ConfirmEmailChangeRequest) (*ConfirmEmailChangeResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		if err := profileSvc.ConfirmEmailChange(ctx, input.Body.Token); err != nil {
			if errors.Is(err, service.ErrEmailChangeInvalid) {
				return nil, huma.Error400BadRequest("this confirmation link is invalid or has expired")
			}
			slog.ErrorContext(ctx, "failed to confirm email change", "error", err)
			return nil, huma.Error500InternalServerError("failed to confirm email change")
		}

		out := &ConfirmEmailChangeResponse{}
		out.Body.Message = "email address updated"
		return out, nil
	})
}
