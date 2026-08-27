// Package org carries the organization-management surface: who is in this
// organization and what they may do. Every endpoint takes its organization from
// the caller's session, never from the request.
package org

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

type MemberView struct {
	MembershipID   string `json:"membership_id" format:"uuid"`
	UserID         string `json:"user_id" format:"uuid"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Role           string `json:"role" example:"member"`
	SupportEnabled bool   `json:"support_enabled" doc:"Whether this member has granted platform support permission to act as them."`
}

type ListMembersRequest struct{}

type ListMembersResponse struct {
	Body struct {
		Members []MemberView `json:"members"`
	}
}

type UpdateRoleRequest struct {
	ID   string `path:"id" format:"uuid" doc:"The membership id, not the user id."`
	Body struct {
		Role string `json:"role" required:"true" enum:"owner,admin,member"`
	}
}

type UpdateRoleResponse struct {
	Body struct {
		Updated bool `json:"updated"`
	}
}

type RemoveMemberRequest struct {
	ID string `path:"id" format:"uuid"`
}

type RemoveMemberResponse struct {
	Body struct {
		Removed bool `json:"removed"`
	}
}

func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrMembershipNotInOrg):
		// Identical to "no such membership". A membership in another
		// organization must not be distinguishable from one that does not exist.
		return huma.Error404NotFound("no such member in this organization")
	case errors.Is(err, service.ErrLastOwner):
		return huma.Error409Conflict("an organization must keep at least one owner")
	case errors.Is(err, service.ErrOwnerRoleRequiresOwner):
		return huma.Error403Forbidden("only an owner may grant or revoke the owner role")
	default:
		slog.ErrorContext(ctx, "membership operation failed", "error", err)
		return huma.Error500InternalServerError("membership operation failed")
	}
}

// callerRole resolves the acting member: who they are, and what role they hold.
//
// AssertOrgRole has already proven the caller clears the minimum bar; this
// answers the different question of whether they are an OWNER, which the
// owner-line rules need. Resolved from the database rather than from anything in
// the request, for the same reason the organization is.
//
// It returns the id as well as the role because both callers write it into an
// audit row, and reading `*p.UserID` there was a nil dereference for an org API
// key — which every endpoint here admits, and which identifies a tenant rather
// than a person. Returning it from the one function both paths already go
// through means the refusal happens once and cannot be forgotten at a third
// call site.
func callerRole(
	ctx context.Context, memberships repo.MembershipRepo, p *apimiddleware.Principal,
) (uuid.UUID, control.MemberRole, error) {
	actor, err := p.ActingUser()
	if err != nil {
		return uuid.Nil, "", err
	}
	m, err := memberships.GetByUserAndOrg(ctx, actor, p.OrganizationID)
	if err != nil {
		return uuid.Nil, "", err
	}
	if m == nil {
		return uuid.Nil, "", apimiddleware.ErrNoMembership
	}
	return actor, m.Role, nil
}

func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	// Deleting the organization and deleting your own account. Both are
	// destructive, both refuse impersonation, and both require the caller to type
	// back what they are destroying.
	RegisterDeletion(hapi, deps)

	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)
	svc := service.NewMembershipService(deps.ControlDB, memberships, repo.NewAuthAuditRepoGorm(deps.ControlDB))

	huma.Register(hapi, huma.Operation{
		OperationID: "org-members-list",
		Method:      http.MethodGet,
		Path:        "/organization/members",
		Summary:     "List the members of this organization",
		Description: "Any member may see who else is in their organization. Roles are included because they determine who to ask for access.",
		Tags:        []string{"Organization"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListMembersRequest) (*ListMembersResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		list, err := svc.List(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListMembersResponse{}
		out.Body.Members = make([]MemberView, 0, len(list))
		for _, m := range list {
			out.Body.Members = append(out.Body.Members, MemberView{
				MembershipID:   m.MembershipID.String(),
				UserID:         m.UserID.String(),
				Email:          m.Email,
				FirstName:      m.FirstName,
				LastName:       m.LastName,
				Role:           string(m.Role),
				SupportEnabled: m.SupportEnabled,
			})
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "org-members-update-role",
		Method:      http.MethodPatch,
		Path:        "/organization/members/{id}",
		Summary:     "Change a member's role",
		Description: "Requires admin, and an owner for anything crossing the owner line in either direction. The last owner cannot be demoted. " +
			"Impersonated sessions are refused: an operator acting as a user must not be able to grant themselves standing access.",
		Tags:     []string{"Organization"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateRoleRequest) (*UpdateRoleResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation(), apimiddleware.RequireActingUser())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid membership id")
		}
		actorID, actorRole, err := callerRole(ctx, memberships, p)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		if err := svc.UpdateRole(ctx, service.UpdateRoleInput{
			OrgID:        p.OrganizationID,
			MembershipID: id,
			NewRole:      control.MemberRole(in.Body.Role),
			ActorID:      actorID,
			ActorRole:    actorRole,
		}); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &UpdateRoleResponse{}
		out.Body.Updated = true
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "org-members-remove",
		Method:      http.MethodDelete,
		Path:        "/organization/members/{id}",
		Summary:     "Remove a member from this organization",
		Description: "Requires admin, and an owner to remove an owner. The last owner cannot be removed.",
		Tags:        []string{"Organization"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RemoveMemberRequest) (*RemoveMemberResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation(), apimiddleware.RequireActingUser())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid membership id")
		}
		actorID, actorRole, err := callerRole(ctx, memberships, p)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		if err := svc.Remove(ctx, service.RemoveMemberInput{
			OrgID:        p.OrganizationID,
			MembershipID: id,
			ActorID:      actorID,
			ActorRole:    actorRole,
		}); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &RemoveMemberResponse{}
		out.Body.Removed = true
		return out, nil
	})
}
