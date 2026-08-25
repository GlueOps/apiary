// Package admin carries the platform-staff surface. Everything in it is gated
// on the PLATFORM role held on the user record, never on a membership role.
package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

func newService(deps *app.Deps) *service.ImpersonationService {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	return service.NewImpersonationService(
		&userRepo,
		repo.NewMembershipRepoGorm(deps.ControlDB),
		repo.NewImpersonationGrantRepoGorm(deps.ControlDB),
		deps.TokenSessions,
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
	)
}

type ImpersonateStartRequest struct {
	XRealIP   string `header:"X-Real-Ip"`
	UserAgent string `header:"User-Agent"`
	Body      struct {
		UserID         string `json:"user_id" required:"true" format:"uuid"`
		OrganizationID string `json:"organization_id" required:"true" format:"uuid"`
		Reason         string `json:"reason" required:"true" minLength:"8" maxLength:"500" doc:"Recorded in the audit trail. Required: an impersonation with no stated reason is not reviewable."`
	}
}

type ImpersonateStartResponse struct {
	Body struct {
		AccessToken    string `json:"access_token"`
		TokenType      string `json:"token_type"`
		ExpiresIn      int    `json:"expires_in"`
		OrganizationID string `json:"organization_id"`
		TargetEmail    string `json:"target_email"`
	}
}

type ImpersonateEndRequest struct {
	XRealIP   string `header:"X-Real-Ip"`
	UserAgent string `header:"User-Agent"`
	Body      struct{}
}

type ImpersonateEndResponse struct {
	Body struct {
		Ended bool `json:"ended"`
	}
}

func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrTargetUserNotFound), errors.Is(err, service.ErrMembershipNotFound):
		// One answer for both. Distinguishing "no such user" from "that user is
		// not in that organization" would let staff enumerate memberships, and
		// staff endpoints are still worth keeping narrow.
		return huma.Error404NotFound("no such user in that organization")
	case errors.Is(err, service.ErrTargetUserInactive):
		return huma.Error409Conflict("that user's account is inactive")
	case errors.Is(err, service.ErrImpersonationNotAllowed):
		// The consent gate: the member has not enabled support access and no
		// active grant covers them. A deliberate refusal, so not a 5xx.
		return huma.Error403Forbidden("that user has not granted support access")
	default:
		slog.ErrorContext(ctx, "impersonation operation failed", "error", err)
		return huma.Error500InternalServerError("impersonation failed")
	}
}

// RegisterUserProtected mounts the impersonation surface behind the bearer
// middleware. The role checks inside each handler are what make it staff-only;
// being behind auth is necessary and nowhere near sufficient.
func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	svc := newService(deps)
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "admin-impersonate-start",
		Method:      http.MethodPost,
		Path:        "/admin/impersonate",
		Summary:     "Begin impersonating a user",
		Description: "Platform staff only. The target must have enabled support access or be covered by an active grant — the service refuses otherwise, and this endpoint cannot override that. " +
			"Returns an access token whose subject is the target user and whose actor is the caller; every action taken with it is attributed to both.",
		Tags:     []string{"Admin"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ImpersonateStartRequest) (*ImpersonateStartResponse, error) {
		p, _, err := apimiddleware.AssertPlatformRole(ctx, &userRepo, control.RolePlatformAdmin, control.RolePlatformSupport)
		if err != nil {
			return nil, apimiddleware.PlatformRoleError(err)
		}

		targetID, err := uuid.Parse(in.Body.UserID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid user id")
		}
		orgID, err := uuid.Parse(in.Body.OrganizationID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid organization id")
		}
		if targetID == *p.UserID {
			return nil, huma.Error422UnprocessableEntity("you cannot impersonate yourself")
		}

		result, err := svc.Start(ctx, service.StartImpersonationInput{
			AdminID:        *p.UserID,
			TargetUserID:   targetID,
			OrganizationID: orgID,
			Reason:         in.Body.Reason,
			IPAddress:      in.XRealIP,
			UserAgent:      in.UserAgent,
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &ImpersonateStartResponse{}
		out.Body.AccessToken = result.AccessToken
		out.Body.TokenType = result.TokenType
		out.Body.ExpiresIn = result.ExpiresIn
		out.Body.OrganizationID = result.OrganizationID
		if result.TargetUser != nil {
			out.Body.TargetEmail = result.TargetUser.Email
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "admin-impersonate-end",
		Method:      http.MethodPost,
		Path:        "/admin/impersonate/end",
		Summary:     "End the current impersonated session",
		Description: "Called BY the impersonated session, which is the only caller that holds the JTI being revoked. " +
			"It takes no parameters for that reason: everything it needs — which session, which target, which actor — comes from the token, so it cannot be pointed at somebody else's session.",
		Tags:     []string{"Admin"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ImpersonateEndRequest) (*ImpersonateEndResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		// Only an impersonated session has anything to end. A normal session
		// calling this would otherwise revoke ITSELF, turning a stray client
		// call into an unexplained logout.
		if !p.IsImpersonating() {
			return nil, huma.Error409Conflict("this session is not an impersonation")
		}
		if p.SessionJTI == "" {
			// Nothing to revoke and no way to name it; failing closed here is
			// better than reporting a revocation that did not happen.
			return nil, huma.Error409Conflict("this session cannot be ended")
		}

		if err := svc.End(ctx, p.SessionJTI, *p.UserID, *p.ActorID, p.OrganizationID, in.XRealIP, in.UserAgent); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ImpersonateEndResponse{}
		out.Body.Ended = true
		return out, nil
	})
}
