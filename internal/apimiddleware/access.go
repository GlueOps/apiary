package apimiddleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrAccessDenied = errors.New("access denied")

const accessJTIColumn = "access_token_jti"

func parseAccessClaims(tokenStr string, deps *app.Deps) (*service.AuthClaims, error) {
	if strings.TrimSpace(tokenStr) == "" {
		return nil, ErrAccessDenied
	}

	claims := &service.AuthClaims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(deps.Config.JWTSecret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(deps.Config.JWTIssuer),
	)
	if err != nil || !token.Valid || claims.Type != "access" {
		return nil, ErrAccessDenied
	}
	if claims.MFAEnabled && !claims.MFA {
		return nil, ErrAccessDenied
	}
	return claims, nil
}

func isPlatformAdmin(ctx context.Context, deps *app.Deps, claims *service.AuthClaims) (uuid.UUID, error) {
	if claims.Impersonate {
		return uuid.Nil, ErrAccessDenied
	}

	userID, err := uuid.Parse(strings.TrimSpace(claims.Subject))
	if err != nil {
		return uuid.Nil, ErrAccessDenied
	}

	if deps.ControlDB == nil {
		return uuid.Nil, ErrAccessDenied
	}

	var user control.User
	if err := deps.ControlDB.WithContext(ctx).
		Select("role", "is_active").
		First(&user, "id = ?", userID).Error; err != nil {
		return uuid.Nil, ErrAccessDenied
	}
	if !user.IsActive {
		return uuid.Nil, ErrAccessDenied
	}
	if user.Role != control.RolePlatformAdmin && user.Role != control.RolePlatformSupport {
		return uuid.Nil, ErrAccessDenied
	}
	return userID, nil
}

func checkSessionLiveness(ctx context.Context, deps *app.Deps, claims *service.AuthClaims, userID uuid.UUID) error {
	if deps.ControlDB == nil {
		slog.ErrorContext(ctx, "session liveness cannot be checked: control db is nil")
		return ErrAccessDenied
	}

	if strings.TrimSpace(claims.ID) == "" {
		return ErrAccessDenied
	}

	var session control.TokenSession
	err := deps.ControlDB.WithContext(ctx).
		Select("id", "revoked_at").
		First(&session, accessJTIColumn+" = ?", claims.ID).Error
	switch {
	case err == nil:
		if session.RevokedAt != nil {
			return ErrAccessDenied
		}
	case errors.Is(err, gorm.ErrRecordNotFound):

		return ErrAccessDenied
	default:
		slog.ErrorContext(ctx, "session liveness lookup failed", "error", err)
		return ErrAccessDenied
	}

	var user control.User
	if err := deps.ControlDB.WithContext(ctx).
		Select("is_active", "tokens_revoked_at").
		First(&user, "id = ?", userID).Error; err != nil {
		return ErrAccessDenied
	}
	if !user.IsActive {
		return ErrAccessDenied
	}
	if user.TokensRevokedAt != nil && claims.IssuedAt != nil &&
		claims.IssuedAt.Before(*user.TokensRevokedAt) {
		return ErrAccessDenied
	}

	return nil
}

func RequirePlatformAdminHTTP(deps *app.Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := parseAccessClaims(bearerTokenFromHTTP(r), deps)
			if err != nil {
				http.Error(w, "platform admin access required", http.StatusUnauthorized)
				return
			}
			if _, err := isPlatformAdmin(r.Context(), deps, claims); err != nil {
				http.Error(w, "platform admin access required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerTokenFromHTTP(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("Authorization")); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(after)
		}
		return ""
	}
	if c, err := r.Cookie("ag_jwt"); err == nil {
		return strings.TrimSpace(strings.TrimPrefix(c.Value, "Bearer "))
	}
	return ""
}
