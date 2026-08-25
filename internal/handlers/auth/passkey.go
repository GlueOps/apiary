package auth

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

type PasskeyBeginRegisterRequest struct{}

type PasskeyBeginRegisterResponse struct {
	Body struct {
		Options   *protocol.CredentialCreation `json:"options"`
		Challenge string                       `json:"challenge"`
	}
}

type PasskeyFinishRegisterRequest struct {
	Body struct {
		Challenge  string          `json:"challenge" required:"true"`
		Name       string          `json:"name"`
		Credential json.RawMessage `json:"credential" required:"true"`
	}
}

type PasskeyFinishRegisterResponse struct {
	Body struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
	}
}

type PasskeyBeginLoginRequest struct{}

type PasskeyBeginLoginResponse struct {
	Body struct {
		Options   *protocol.CredentialAssertion `json:"options"`
		Challenge string                        `json:"challenge"`
	}
}

type PasskeyFinishLoginRequest struct {
	UserAgent string `header:"User-Agent"`
	Body      struct {
		Challenge  string          `json:"challenge" required:"true"`
		Credential json.RawMessage `json:"credential" required:"true"`
	}
}

type PasskeyFinishLoginResponse struct {
	Body struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		TokenType        string `json:"token_type"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
		OrganizationID   string `json:"organization_id"`
	}
}

type PasskeyListRequest struct{}

type PasskeyListResponse struct {
	Body struct {
		Passkeys []PasskeyEntry `json:"passkeys"`
	}
}

type PasskeyEntry struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  string     `json:"created_at"`
}

type PasskeyDeleteRequest struct {
	ID string `path:"id"`
}

type PasskeyDeleteResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func newPasskeySvc(deps *app.Deps) (*service.PasskeyService, repo.UserRepoGorm) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(deps.ControlDB)
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)

	passkeySvc, err := service.NewPasskeyService(
		passkeyRepo,
		&userRepo,
		membershipRepo,
		deps.TokenSessions.Tokens(),
		deps.TokenSessions,
		deps.Config.JWTSecret,
		deps.Config.WebAuthnRPID,
		deps.Config.WebAuthnRPOrigin,
		deps.Config.WebAuthnRPName,
	)
	if err != nil {
		panic(err)
	}

	passkeySvc.SetChallenges(repo.NewAuthChallengeRepoGorm(deps.ControlDB))
	passkeySvc.SetAudit(repo.NewAuthAuditRepoGorm(deps.ControlDB))

	return passkeySvc, userRepo
}

func RegisterPasskeyPublic(hapi huma.API, deps *app.Deps) {
	passkeySvc, _ := newPasskeySvc(deps)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkey-login-begin",
		Method:      http.MethodPost,
		Path:        "/auth/passkey/login/begin",
		Summary:     "Begin passkey login",
		Description: "Starts the WebAuthn discoverable login flow. Returns options and a signed challenge.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, _ *PasskeyBeginLoginRequest) (*PasskeyBeginLoginResponse, error) {
		result, err := passkeySvc.BeginLogin(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "failed to begin passkey login", "error", err)
			return nil, huma.Error500InternalServerError("failed to begin passkey login")
		}

		out := &PasskeyBeginLoginResponse{}
		out.Body.Options = result.Options
		out.Body.Challenge = result.Challenge
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkey-login-finish",
		Method:      http.MethodPost,
		Path:        "/auth/passkey/login/finish",
		Summary:     "Complete passkey login",
		Description: "Validates the WebAuthn assertion and returns access/refresh tokens.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *PasskeyFinishLoginRequest) (*PasskeyFinishLoginResponse, error) {
		parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Body.Credential))
		if err != nil {
			return nil, huma.Error400BadRequest("invalid WebAuthn response: " + err.Error())
		}

		tokens, orgID, err := passkeySvc.FinishLogin(ctx, service.FinishLoginInput{
			Challenge: input.Body.Challenge,
			Response:  parsedResponse,
			IP:        apimiddleware.ClientIPFromContext(ctx),
			UA:        input.UserAgent,
		})
		if err != nil {
			slog.ErrorContext(ctx, "passkey login failed", "error", err)
			return nil, huma.Error401Unauthorized("passkey login failed")
		}

		out := &PasskeyFinishLoginResponse{}
		out.Body.AccessToken = tokens.AccessToken
		out.Body.RefreshToken = tokens.RefreshToken
		out.Body.TokenType = tokens.TokenType
		out.Body.ExpiresIn = tokens.ExpiresIn
		out.Body.RefreshExpiresIn = tokens.RefreshExpiresIn
		out.Body.OrganizationID = orgID
		return out, nil
	})
}

func RegisterPasskeyProtected(hapi huma.API, deps *app.Deps) {
	passkeySvc, _ := newPasskeySvc(deps)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkey-register-begin",
		Method:      http.MethodPost,
		Path:        "/auth/passkey/register/begin",
		Summary:     "Begin passkey registration",
		Description: "REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Starts the WebAuthn registration flow. Returns creation options and a signed challenge.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, _ *PasskeyBeginRegisterRequest) (*PasskeyBeginRegisterResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		result, err := passkeySvc.BeginRegistration(ctx, *p.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to begin passkey registration", "error", err)
			return nil, huma.Error500InternalServerError("failed to begin passkey registration")
		}

		out := &PasskeyBeginRegisterResponse{}
		out.Body.Options = result.Options
		out.Body.Challenge = result.Challenge
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkey-register-finish",
		Method:      http.MethodPost,
		Path:        "/auth/passkey/register/finish",
		Summary:     "Complete passkey registration",
		Description: "REACHABLE BEFORE POLICY IS SATISFIED. A session that has authenticated but not yet met a policy such as require_mfa can call this — that is the point, since the endpoint exists to let someone comply. Authentication still applies in full; only authorization is relaxed, and only for accounts holding NO verified factor. An account that already has one must present it first, or a stolen password could add its own authenticator. Validates the WebAuthn attestation and stores the credential.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, input *PasskeyFinishRegisterRequest) (*PasskeyFinishRegisterResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(input.Body.Credential))
		if err != nil {
			return nil, huma.Error400BadRequest("invalid WebAuthn response: " + err.Error())
		}

		passkey, err := passkeySvc.FinishRegistration(ctx, *p.UserID, service.FinishRegistrationInput{
			Challenge: input.Body.Challenge,
			Response:  parsedResponse,
			Name:      input.Body.Name,
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to register passkey", "error", err)
			return nil, huma.Error500InternalServerError("failed to register passkey")
		}

		out := &PasskeyFinishRegisterResponse{}
		out.Body.ID = passkey.ID.String()
		out.Body.Name = passkey.Name
		out.Body.CreatedAt = passkey.CreatedAt.Format(time.RFC3339)
		return out, nil
	})
}

func RegisterPasskeyUserProtected(hapi huma.API, deps *app.Deps) {
	passkeySvc, _ := newPasskeySvc(deps)
	totpRepo := repo.NewUserTOTPRepoGorm(deps.ControlDB)
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkeys-list",
		Method:      http.MethodGet,
		Path:        "/auth/passkeys",
		Summary:     "List registered passkeys",
		Description: "Returns all passkeys registered for the authenticated user.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, _ *PasskeyListRequest) (*PasskeyListResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		keys, err := passkeySvc.List(ctx, *p.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to list passkeys", "error", err)
			return nil, huma.Error500InternalServerError("failed to list passkeys")
		}

		out := &PasskeyListResponse{}
		out.Body.Passkeys = make([]PasskeyEntry, len(keys))
		for i, k := range keys {
			out.Body.Passkeys[i] = PasskeyEntry{
				ID:         k.ID.String(),
				Name:       k.Name,
				LastUsedAt: k.LastUsedAt,
				CreatedAt:  k.CreatedAt.Format(time.RFC3339),
			}
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-passkeys-delete",
		Method:      http.MethodDelete,
		Path:        "/auth/passkeys/{id}",
		Summary:     "Remove a passkey",
		Description: "Deletes a registered passkey. If no MFA methods remain, MFA is disabled.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, input *PasskeyDeleteRequest) (*PasskeyDeleteResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("not allowed during impersonation")
		}

		passkeyID, err := uuid.Parse(input.ID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid passkey ID")
		}

		if deps.Policies != nil {

			currentCount, cerr := passkeyRepo.CountByUserID(ctx, *p.UserID)
			if cerr != nil {
				slog.ErrorContext(ctx, "failed to count passkeys", "error", cerr)
				return nil, huma.Error500InternalServerError("failed to check MFA policy")
			}
			if currentCount <= 1 {

				ep, perr := deps.Policies.EffectivePolicy(ctx, p.OrganizationID, control.PolicyKindRequirePasskey)
				if perr != nil {
					slog.ErrorContext(ctx, "require-passkey policy lookup failed", "error", perr)
					return nil, huma.Error500InternalServerError("failed to check MFA policy")
				}
				if ep != nil && ep.Enabled {
					return nil, huma.Error403Forbidden("cannot remove last passkey — required by policy")
				}

				ep2, perr2 := deps.Policies.EffectivePolicy(ctx, p.OrganizationID, control.PolicyKindRequireMFA)
				if perr2 != nil {
					slog.ErrorContext(ctx, "require-mfa policy lookup failed", "error", perr2)
					return nil, huma.Error500InternalServerError("failed to check MFA policy")
				}
				if ep2 != nil && ep2.Enabled {
					totp2, terr := totpRepo.GetByUserID(ctx, *p.UserID)
					if terr != nil {
						slog.ErrorContext(ctx, "failed to load TOTP factor", "error", terr)
						return nil, huma.Error500InternalServerError("failed to check MFA policy")
					}
					if totp2 == nil || !totp2.Verified {
						return nil, huma.Error403Forbidden("cannot remove last MFA method — required by policy")
					}
				}
			}
		}

		if err := passkeySvc.Delete(ctx, passkeyID, *p.UserID); err != nil {
			if errors.Is(err, service.ErrPasskeyNotFound) {
				return nil, huma.Error404NotFound("passkey not found")
			}
			slog.ErrorContext(ctx, "failed to delete passkey", "error", err)
			return nil, huma.Error500InternalServerError("failed to delete passkey")
		}

		remainingPasskeys, err := passkeyRepo.CountByUserID(ctx, *p.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check remaining passkeys", "error", err)
			return nil, huma.Error500InternalServerError("failed to check remaining passkeys")
		}

		totp, err := totpRepo.GetByUserID(ctx, *p.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check TOTP status", "error", err)
			return nil, huma.Error500InternalServerError("failed to check TOTP status")
		}

		hasTOTP := totp != nil && totp.Verified
		if remainingPasskeys == 0 && !hasTOTP {
			_ = userRepo.SetMFAEnabled(ctx, *p.UserID, false)
		}

		out := &PasskeyDeleteResponse{}
		out.Body.Message = "passkey removed"
		return out, nil
	})
}
