// Package invitation carries two route groups that must never be mounted
// together: the organization-facing endpoints that create and withdraw
// invitations, which require an admin session, and the accept endpoints in
// accept.go, which are public by design because the invitee usually has no
// account yet.
//
// They live in one package because they share a service, and in separate files
// with separate Register functions because the only thing standing between the
// public group and the authed group is which huma.API they are handed. Mixing
// them into one Register call is how a public endpoint ends up behind auth, or
// far worse, an authed one ends up in front of it.
package invitation

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

func newService(deps *app.Deps) *service.InvitationService {
	userRepo := repo.NewUserRepoGorm(deps.ControlDB)
	return service.NewInvitationService(
		deps.ControlDB,
		repo.NewInvitationRepoGorm(deps.ControlDB),
		&userRepo,
		repo.NewMembershipRepoGorm(deps.ControlDB),
		repo.NewOrganizationRepoGorm(deps.ControlDB),
		deps.Policies,
		repo.NewAuthAccessRuleRepoGorm(deps.ControlDB),
		deps.MailSender(),
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
		deps.Config.BaseURL,
	)
}

type InvitationView struct {
	ID        string `json:"id" format:"uuid"`
	Email     string `json:"email"`
	Role      string `json:"role" example:"member"`
	ExpiresAt string `json:"expires_at" format:"date-time"`
	CreatedAt string `json:"created_at" format:"date-time"`
	Status    string `json:"status" enum:"pending,accepted,revoked,expired" doc:"An invitation is in exactly one of these states."`
}

func toView(v service.InvitationView) InvitationView {
	return InvitationView{
		ID:        v.ID.String(),
		Email:     v.Email,
		Role:      string(v.Role),
		ExpiresAt: v.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:    v.Status,
	}
}

type CreateRequest struct {
	Body struct {
		Email string `json:"email" required:"true" format:"email" maxLength:"255"`
		Role  string `json:"role,omitempty" enum:"member,admin" default:"member" doc:"Owner is deliberately not invitable; ownership is transferred by an existing owner."`
	}
}

type CreateResponse struct{ Body InvitationView }

type ListRequest struct{}

type ListResponse struct {
	Body struct {
		Invitations []InvitationView `json:"invitations"`
	}
}

type RevokeRequest struct {
	ID string `path:"id" format:"uuid"`
}

type RevokeResponse struct {
	Body struct {
		Revoked bool `json:"revoked"`
	}
}

// mapErr keeps every deliberate refusal off the 5xx dashboard.
func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrInvitationInvalid), errors.Is(err, repo.ErrInvitationNotFound):
		return huma.Error404NotFound("invitation not found")
	case errors.Is(err, service.ErrAlreadyMember):
		return huma.Error409Conflict("that address is already a member of this organization")
	case errors.Is(err, repo.ErrInvitationExists):
		return huma.Error409Conflict("an invitation for that address is already outstanding")
	case errors.Is(err, repo.ErrAccessDenied):
		// Deliberately vague: the response must not be usable to probe the
		// access-rule table for which domains the platform will admit.
		return huma.Error403Forbidden("that address may not be invited")
	case errors.Is(err, service.ErrAdmissionNotConfigured):
		slog.ErrorContext(ctx, "access rules are not wired; refusing to issue an invitation", "error", err)
		return huma.Error500InternalServerError("invitation failed")
	case errors.Is(err, service.ErrTenantNotProvisioned):
		return huma.NewError(http.StatusServiceUnavailable,
			"this organization's storage is not ready yet, so it cannot invite anyone")
	default:
		slog.ErrorContext(ctx, "invitation operation failed", "error", err)
		return huma.Error500InternalServerError("invitation failed")
	}
}

// requireProvisionedTenant refuses to let an organization with no tenant
// database invite anyone.
//
// An organization whose founder's domain was refused create_org, or whose
// confirmation has not happened yet, has no storage — so a member accepted into
// it would land somewhere with nothing behind it, and every tenant-scoped
// endpoint they touched would answer 503. Refusing at the invitation is the
// honest point to stop: the alternative is an invitee who accepts successfully
// and then finds a broken account.
func requireProvisionedTenant(ctx context.Context, deps *app.Deps, orgID uuid.UUID) error {
	if deps.Tenants == nil {
		return service.ErrTenantNotProvisioned
	}
	if _, err := deps.Tenants.GetTenantContext(ctx, orgID.String()); err != nil {
		return service.ErrTenantNotProvisioned
	}
	return nil
}

// RegisterOrgProtected mounts the endpoints that require an admin session.
func RegisterOrgProtected(hapi huma.API, deps *app.Deps) {
	svc := newService(deps)
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID:   "org-invitations-create",
		Method:        http.MethodPost,
		Path:          "/invitations",
		Summary:       "Invite someone into this organization",
		DefaultStatus: http.StatusCreated,
		Description: "Requires admin. Impersonated sessions are refused: an operator acting as a user must not be able to invite an account they control. " +
			"The organization comes from the session, never from the request.",
		Tags:     []string{"Invitations"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateRequest) (*CreateResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation(), apimiddleware.RequireActingUser())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		// An invitation names its inviter, and an org API key is a tenant rather
		// than a person, so there is nobody to name.
		//
		// Ahead of the provisioning check on purpose: the credential is wrong
		// whatever the tenant's storage is doing, and answering 503 first tells
		// the caller to wait for something that would not help when it arrived.
		inviter, err := p.ActingUser()
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		if err := requireProvisionedTenant(ctx, deps, p.OrganizationID); err != nil {
			return nil, mapErr(ctx, err)
		}

		role := control.MemberRole(in.Body.Role)
		if role == "" {
			role = control.MemberRoleMember
		}
		view, err := svc.Invite(ctx, service.InviteInput{
			OrgID:     p.OrganizationID,
			InviterID: inviter,
			Email:     in.Body.Email,
			Role:      role,
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &CreateResponse{Body: toView(*view)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "org-invitations-list",
		Method:      http.MethodGet,
		Path:        "/invitations",
		Summary:     "List this organization's invitations",
		Tags:        []string{"Invitations"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListRequest) (*ListResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		list, err := svc.List(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListResponse{}
		out.Body.Invitations = make([]InvitationView, 0, len(list))
		for _, v := range list {
			out.Body.Invitations = append(out.Body.Invitations, toView(v))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "org-invitations-revoke",
		Method:      http.MethodDelete,
		Path:        "/invitations/{id}",
		Summary:     "Withdraw an invitation",
		Description: "The organization is taken from the session and used as a query predicate, so an id belonging to another tenant answers 404 rather than revoking anything.",
		Tags:        []string{"Invitations"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RevokeRequest) (*RevokeResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation(), apimiddleware.RequireActingUser())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid invitation id")
		}
		actor, err := p.ActingUser()
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		if err := svc.Revoke(ctx, p.OrganizationID, id, actor); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &RevokeResponse{}
		out.Body.Revoked = true
		return out, nil
	})
}
