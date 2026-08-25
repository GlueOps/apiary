package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

type TOTPSetupRequest struct{}

type TOTPSetupResponse struct {
	Body struct {
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	}
}

type TOTPVerifyRequest struct {
	Body struct {
		Code string `json:"code" required:"true" minLength:"6" maxLength:"6"`
	}
}

type TOTPVerifyResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

type TOTPDisableRequest struct {
	Body struct {
		Password string `json:"password,omitempty"`
		Code     string `json:"code,omitempty"`
	}
}

type TOTPDisableResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

type MFAVerifyRequest struct {
	UserAgent string `header:"User-Agent"`
	Body      struct {
		MFAToken string `json:"mfa_token" required:"true"`
		Code     string `json:"code" required:"true" minLength:"6" maxLength:"6"`
	}
}

type MFAVerifyResponse struct {
	Body struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		TokenType        string `json:"token_type"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
		OrganizationID   string `json:"organization_id"`
	}
}

func RegisterTOTPPublic(hapi huma.API, deps *app.Deps) {
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(deps.ControlDB)

	tokenSvc := deps.TokenSessions.Tokens()
	sessionSvc := deps.TokenSessions

	userRepo := repo.NewUserRepoGorm(deps.ControlDB)

	mfaSvc := service.NewMFAService(totpRepo, passkeyRepo, &userRepo, deps.Config.EncryptionMasterKey, "APIary")
	mfaSvc.SetAudit(repo.NewAuthAuditRepoGorm(deps.ControlDB))

	challenges := repo.NewAuthChallengeRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-mfa-verify",
		Method:      http.MethodPost,
		Path:        "/auth/mfa/verify",
		Summary:     "Complete MFA challenge",
		Description: "Validates an MFA pending token and TOTP code, then returns full access/refresh tokens.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *MFAVerifyRequest) (*MFAVerifyResponse, error) {
		claims, err := tokenSvc.Parse(input.Body.MFAToken, "mfa_pending")
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid or expired MFA token")
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid MFA token")
		}

		jti, err := uuid.Parse(claims.ID)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid MFA token")
		}
		challenge, err := challenges.GetLive(ctx, jti, time.Now().UTC())
		if err != nil {
			return nil, huma.Error500InternalServerError("mfa verification failed")
		}
		if challenge == nil {

			return nil, huma.Error401Unauthorized("invalid or expired MFA token")
		}

		if err := mfaSvc.ValidateTOTP(ctx, userID, input.Body.Code); err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidTOTPCode):

				if _, rerr := challenges.RecordFailure(ctx, jti); rerr != nil {
					slog.ErrorContext(ctx, "failed to record mfa attempt", "error", rerr)
				}
				return nil, huma.Error401Unauthorized("invalid TOTP code")
			case errors.Is(err, service.ErrTOTPNotSetup):
				return nil, huma.Error400BadRequest("TOTP not configured")
			default:
				slog.ErrorContext(ctx, "MFA verification failed", "error", err)
				return nil, huma.Error500InternalServerError("MFA verification failed")
			}
		}

		consumed, err := challenges.Consume(ctx, jti, time.Now().UTC())
		if err != nil {
			return nil, huma.Error500InternalServerError("mfa verification failed")
		}
		if !consumed {
			return nil, huma.Error401Unauthorized("invalid or expired MFA token")
		}

		_ = tokenSvc
		issued, err := sessionSvc.IssueTracked(ctx, service.IssueTrackedInput{
			SubjectID:   claims.Subject,
			SubjectKind: "user",
			OrgID:       claims.OrgID,
			Opts: service.IssueOpts{
				AMR:        []string{"pwd", "otp"},
				MFA:        true,
				MFAEnabled: true,
			},
			IPAddress: apimiddleware.ClientIPFromContext(ctx),
			UserAgent: input.UserAgent,
			EventType: "mfa_verify",
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to issue tokens", "error", err)
			return nil, huma.Error500InternalServerError("failed to issue tokens")
		}

		out := &MFAVerifyResponse{}
		out.Body.AccessToken = issued.AccessToken
		out.Body.RefreshToken = issued.RefreshToken
		out.Body.TokenType = issued.TokenType
		out.Body.ExpiresIn = issued.ExpiresIn
		out.Body.RefreshExpiresIn = issued.RefreshExpiresIn
		out.Body.OrganizationID = claims.OrgID
		return out, nil
	})
}

func RegisterTOTPProtected(hapi huma.API, deps *app.Deps) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(deps.ControlDB)

	mfaSvc := service.NewMFAService(totpRepo, passkeyRepo, &userRepo, deps.Config.EncryptionMasterKey, "APIary")
	mfaSvc.SetAudit(repo.NewAuthAuditRepoGorm(deps.ControlDB))

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-totp-setup",
		Method:      http.MethodPost,
		Path:        "/auth/totp/setup",
		Summary:     "Start TOTP setup",
		Description: "REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Generates a new TOTP secret and returns the otpauth URI for QR code rendering.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, _ *TOTPSetupRequest) (*TOTPSetupResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		user, err := userRepo.GetByID(ctx, *p.UserID)
		if err != nil || user == nil {
			return nil, huma.Error500InternalServerError("failed to load user")
		}

		result, err := mfaSvc.SetupTOTP(ctx, *p.UserID, user.Email)
		if err != nil {
			if errors.Is(err, service.ErrTOTPAlreadySetup) {
				return nil, huma.Error409Conflict("TOTP already configured")
			}
			return nil, huma.Error500InternalServerError("failed to setup TOTP")
		}

		out := &TOTPSetupResponse{}
		out.Body.Secret = result.Secret
		out.Body.URI = result.URI
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-totp-confirm",
		Method:      http.MethodPost,
		Path:        "/auth/totp/verify",
		Summary:     "Confirm TOTP setup",
		Description: "REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Confirms TOTP setup by validating a code. This activates MFA for the user.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, input *TOTPVerifyRequest) (*TOTPVerifyResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		if err := mfaSvc.VerifyAndActivateTOTP(ctx, *p.UserID, input.Body.Code); err != nil {
			switch {
			case errors.Is(err, service.ErrTOTPNotSetup):
				return nil, huma.Error400BadRequest("TOTP not set up — call setup first")
			case errors.Is(err, service.ErrTOTPAlreadySetup):
				return nil, huma.Error409Conflict("TOTP already verified")
			case errors.Is(err, service.ErrInvalidTOTPCode):
				return nil, huma.Error401Unauthorized("invalid TOTP code")
			default:
				return nil, huma.Error500InternalServerError("failed to verify TOTP")
			}
		}

		out := &TOTPVerifyResponse{}
		out.Body.Message = "TOTP activated"
		return out, nil
	})
}

func RegisterTOTPUserProtected(hapi huma.API, deps *app.Deps) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(deps.ControlDB)

	mfaSvc := service.NewMFAService(totpRepo, passkeyRepo, &userRepo, deps.Config.EncryptionMasterKey, "APIary")

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-totp-disable",
		Method:      http.MethodDelete,
		Path:        "/auth/totp",
		Summary:     "Disable TOTP",
		Description: "Disables TOTP MFA. Requires the user's password or a valid TOTP code for verification.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, input *TOTPDisableRequest) (*TOTPDisableResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		if deps.Policies != nil {

			ep, perr := deps.Policies.EffectivePolicy(ctx, p.OrganizationID, control.PolicyKindRequireMFA)
			if perr != nil {
				slog.ErrorContext(ctx, "require-mfa policy lookup failed", "error", perr)
				return nil, huma.Error500InternalServerError("failed to check MFA policy")
			}
			if ep != nil && ep.Enabled {
				pkCount, cerr := passkeyRepo.CountByUserID(ctx, *p.UserID)
				if cerr != nil {
					slog.ErrorContext(ctx, "failed to count passkeys", "error", cerr)
					return nil, huma.Error500InternalServerError("failed to check MFA policy")
				}
				if pkCount == 0 {
					return nil, huma.Error403Forbidden("cannot disable MFA — required by policy")
				}
			}
		}

		user, err := userRepo.GetByID(ctx, *p.UserID)
		if err != nil || user == nil {
			return nil, huma.Error500InternalServerError("failed to load user")
		}

		if input.Body.Password != "" {
			if !user.CheckPassword(input.Body.Password) {
				return nil, huma.Error403Forbidden("invalid password")
			}
		} else if input.Body.Code != "" {
			if err := mfaSvc.ValidateTOTP(ctx, *p.UserID, input.Body.Code); err != nil {
				return nil, huma.Error403Forbidden("invalid TOTP code")
			}
		} else {
			return nil, huma.Error400BadRequest("password or code required to disable TOTP")
		}

		if err := mfaSvc.DisableTOTP(ctx, *p.UserID); err != nil {
			return nil, huma.Error500InternalServerError("failed to disable TOTP")
		}

		out := &TOTPDisableResponse{}
		out.Body.Message = "TOTP disabled"
		return out, nil
	})
}
