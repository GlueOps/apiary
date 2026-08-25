package auth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
)

type LogoutRequest struct {
	UserAgent string `header:"User-Agent"`
}

type LogoutResponse struct {
	Body struct {
		Message string `json:"message"`
	}
}

func RegisterLogout(hapi huma.API, deps *app.Deps) {
	sessionSvc := deps.TokenSessions

	huma.Register(hapi, huma.Operation{
		OperationID: "auth-logout",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Logout current session",
		Description: "Revokes the current token session.",
		Tags:        []string{"Auth"},
		Security:    apimiddleware.SecuritySessionOnly,
	}, func(ctx context.Context, input *LogoutRequest) (*LogoutResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		if p.SessionJTI != "" {
			_ = sessionSvc.Revoke(ctx, p.SessionJTI, p.UserID.String(), apimiddleware.ClientIPFromContext(ctx), input.UserAgent)
		}

		out := &LogoutResponse{}
		out.Body.Message = "logged out"
		return out, nil
	})
}
