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

type RefreshRequest struct {
	UserAgent string `header:"User-Agent"`
	Body      struct {
		RefreshToken string `json:"refresh_token" required:"true"`
	}
}

type RefreshResponse struct {
	Body struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		TokenType        string `json:"token_type"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
		OrganizationID   string `json:"organization_id"`
		Organization     string `json:"organization"`
	}
}

func RegisterRefresh(hapi huma.API, deps *app.Deps) {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)
	orgRepo := repo.NewOrganizationRepoGorm(deps.ControlDB)

	sessionSvc := service.NewSessionService(userRepo, membershipRepo, orgRepo, deps.TokenSessions.Tokens(), deps.TokenSessions, deps.Policies)

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-refresh",
		Method:      http.MethodPost,
		Path:        "/auth/refresh",
		Summary:     "Refresh auth tokens",
		Description: "Validates a refresh token and returns a new access and refresh token pair.",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *RefreshRequest) (*RefreshResponse, error) {
		result, err := sessionSvc.Refresh(ctx, service.RefreshInput{
			RefreshToken: input.Body.RefreshToken,
			IPAddress:    apimiddleware.ClientIPFromContext(ctx),
			UserAgent:    input.UserAgent,
		})
		if err != nil {
			if errors.Is(err, service.ErrUnauthorized) {
				return nil, huma.Error401Unauthorized("invalid refresh token")
			}
			slog.ErrorContext(ctx, "refresh failed", "error", err)
			return nil, huma.Error500InternalServerError("refresh failed")
		}
		out := &RefreshResponse{}
		out.Body.AccessToken = result.Tokens.AccessToken
		out.Body.RefreshToken = result.Tokens.RefreshToken
		out.Body.TokenType = result.Tokens.TokenType
		out.Body.ExpiresIn = result.Tokens.ExpiresIn
		out.Body.RefreshExpiresIn = result.Tokens.RefreshExpiresIn
		out.Body.OrganizationID = result.OrganizationID
		out.Body.Organization = result.Organization
		return out, nil
	})
}
