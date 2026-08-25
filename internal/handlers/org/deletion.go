package org

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

type DeleteOrgRequest struct {
	Body struct {
		ConfirmName string `json:"confirm_name" required:"true" doc:"The organization's name, exactly. This call takes an entire tenant out of service; typing the name is the difference between meaning it and clicking the wrong row."`
	}
}

type DeleteOrgResponse struct {
	Body struct {
		Deleted bool   `json:"deleted"`
		Note    string `json:"note"`
	}
}

type SoleOwnershipsResponse struct {
	Body struct {
		Organizations []struct {
			ID   string `json:"id" format:"uuid"`
			Name string `json:"name"`
		} `json:"organizations"`
		BlocksAccountDeletion bool `json:"blocks_account_deletion"`
	}
}

type DeleteAccountRequest struct {
	Body struct {
		ConfirmEmail string `json:"confirm_email" required:"true" format:"email" doc:"Your own email address, exactly."`
	}
}

type DeleteAccountResponse struct {
	Body struct {
		Deleted bool `json:"deleted"`
	}
}

func now() time.Time { return time.Now().UTC() }

func deletionErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrConfirmationMismatch):
		return huma.Error422UnprocessableEntity("the confirmation does not match")
	case errors.Is(err, service.ErrSoleOwner):
		// 409, not 403: nothing is forbidden about the caller. The system is in a
		// state where the action would leave an organization with nobody able to
		// administer it, and the caller can resolve that themselves.
		return huma.Error409Conflict("this account is the only owner of an organization. " +
			"Promote another owner, or delete the organization, then try again.")
	case errors.Is(err, service.ErrOrgNotFound), errors.Is(err, service.ErrAccountNotFound):
		return huma.Error404NotFound("not found")
	default:
		slog.ErrorContext(ctx, "deletion failed", "error", err)
		return huma.Error500InternalServerError("deletion failed")
	}
}

// RegisterDeletion mounts the two destructive account operations.
//
// Both refuse impersonation. An operator acting as a customer must not be able
// to delete that customer's organization or account — the consent that makes
// impersonation acceptable is consent to be helped, not to be erased.
func RegisterDeletion(hapi huma.API, deps *app.Deps) {
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)
	svc := service.NewAccountDeletionService(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "org-delete",
		Method:      http.MethodDelete,
		Path:        "/orgs/current",
		Summary:     "Delete this organization",
		Description: "OWNER only, and impersonated sessions are refused. " +
			"The organization stops resolving immediately: its tenant database becomes unreachable, every session scoped to it is revoked, and its agents can no longer authenticate. " +
			"Key material is NOT destroyed by this call — the organization row holds the only key that can decrypt the tenant database, so erasing it would destroy the data irreversibly rather than merely retiring it. Purging is a separate operator step, which is what leaves a way back.",
		Tags:     []string{"Organizations"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *DeleteOrgRequest) (*DeleteOrgResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleOwner, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		if err := svc.DeleteOrganization(ctx, p.OrganizationID, in.Body.ConfirmName, now()); err != nil {
			return nil, deletionErr(ctx, err)
		}
		out := &DeleteOrgResponse{}
		out.Body.Deleted = true
		out.Body.Note = "The organization no longer resolves. Its data is retained, encrypted, until an operator purges it."
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "me-sole-ownerships",
		Method:      http.MethodGet,
		Path:        "/me/sole-ownerships",
		Summary:     "Organizations this account solely owns",
		Description: "Deleting an account is refused while it is the only owner of an organization, because that would leave a tenant nobody can administer. This names them, so the answer is actionable rather than just a refusal.",
		Tags:        []string{"Me"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *struct{}) (*SoleOwnershipsResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		orgs, err := svc.SoleOwnerships(ctx, *p.UserID)
		if err != nil {
			return nil, deletionErr(ctx, err)
		}
		out := &SoleOwnershipsResponse{}
		for _, o := range orgs {
			out.Body.Organizations = append(out.Body.Organizations, struct {
				ID   string `json:"id" format:"uuid"`
				Name string `json:"name"`
			}{ID: o.ID.String(), Name: o.Name})
		}
		out.Body.BlocksAccountDeletion = len(orgs) > 0
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "me-delete-account",
		Method:      http.MethodDelete,
		Path:        "/me",
		Summary:     "Delete your account",
		Description: "Refused while this account is the only owner of an organization — see /me/sole-ownerships. " +
			"Every session is revoked and every authentication factor removed. The record itself is retained: audit events, fleet tasks and invitations name the actor, and an audit trail that vanishes with the account is one the audited person can erase.",
		Tags:     []string{"Me"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *DeleteAccountRequest) (*DeleteAccountResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		// An impersonated session must not delete the account it is borrowing.
		if p.IsImpersonating() {
			return nil, huma.Error403Forbidden("an impersonated session may not delete the account")
		}
		users := repo.NewUserRepoGorm(deps.ControlDB)
		u, err := users.GetByID(ctx, *p.UserID)
		if err != nil || u == nil {
			return nil, huma.Error404NotFound("not found")
		}
		if !strings.EqualFold(strings.TrimSpace(in.Body.ConfirmEmail), u.Email) {
			return nil, huma.Error422UnprocessableEntity("the confirmation does not match")
		}
		if err := svc.DeleteAccount(ctx, *p.UserID, now()); err != nil {
			return nil, deletionErr(ctx, err)
		}
		out := &DeleteAccountResponse{}
		out.Body.Deleted = true
		return out, nil
	})
}
