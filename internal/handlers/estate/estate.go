package estate

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

// EstateView is the wire shape. There is no field for secret material anywhere
// in this file: the service does not return it, and a field that does not exist
// cannot be serialised into a response by accident. Fingerprint is what an
// operator compares instead — it identifies the credential without disclosing it.
type EstateView struct {
	ID             string            `json:"id" format:"uuid"`
	Name           string            `json:"name"`
	Provider       string            `json:"provider" example:"aws"`
	Scope          map[string]string `json:"scope" doc:"Provider-specific target, e.g. {\"account\":\"111\",\"region\":\"eu-west-1\"}"`
	ScopeVersion   int               `json:"scope_version"`
	CredentialID   string            `json:"credential_id" format:"uuid"`
	CredentialName string            `json:"credential_name"`
	CredentialKind string            `json:"credential_kind" example:"static_secret"`
	Fingerprint    string            `json:"fingerprint" example:"SHA256:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU" doc:"Complete SHA-256 of the credential's identifying component, in ssh encoding. Compare it against your provider console to confirm which credential is installed."`
}

func toView(v service.EstateView) EstateView {
	return EstateView{
		ID:             v.ID.String(),
		Name:           v.Name,
		Provider:       v.Provider,
		Scope:          v.Scope,
		ScopeVersion:   v.ScopeVersion,
		CredentialID:   v.CredentialID.String(),
		CredentialName: v.CredentialName,
		CredentialKind: v.CredentialKind,
		Fingerprint:    v.Fingerprint,
	}
}

type ListRequest struct{}

type ListResponse struct {
	Body struct {
		Estates []EstateView `json:"estates"`
	}
}

type GetRequest struct {
	ID string `path:"id" format:"uuid"`
}

type GetResponse struct {
	Body EstateView
}

type CreateRequest struct {
	Body struct {
		Name           string            `json:"name" required:"true" minLength:"1" maxLength:"100"`
		Provider       string            `json:"provider" required:"true" enum:"aws,proxmox,hetzner"`
		Scope          map[string]string `json:"scope" required:"true" doc:"Provider-specific target. Keys are validated against the provider; unknown keys are refused."`
		CredentialName string            `json:"credential_name" maxLength:"100"`
		CredentialKind string            `json:"credential_kind,omitempty" enum:"static_secret,assumed_role" default:"static_secret"`
		Identity       string            `json:"identity" required:"true" doc:"The NON-SECRET identifying component — AWS access key id, Proxmox token id, or role ARN. This is what gets fingerprinted."`
		Secret         string            `json:"secret" required:"true" doc:"Credential material. Sealed at rest and never returned by any endpoint."`
	}
}

type CreateResponse struct {
	Body EstateView
}

type RotateRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Identity string `json:"identity" required:"true"`
		Secret   string `json:"secret" required:"true"`
	}
}

type RotateResponse struct {
	Body EstateView
}

type DeleteRequest struct {
	ID string `path:"id" format:"uuid"`
}

type DeleteResponse struct {
	Body struct {
		Deleted bool `json:"deleted"`
	}
}

// mapErr turns service and repository errors into the right status.
//
// Every refusal below is a deliberate decision, so none of them may surface as
// 500. A policy decision reported as a server error tells the caller the system
// is broken and puts an operator's own rules on their 5xx dashboard.
func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, repo.ErrEstateNotFound), errors.Is(err, repo.ErrCredentialNotFound):
		return huma.Error404NotFound("estate not found")

	case errors.Is(err, repo.ErrScopeKeyUnknown),
		errors.Is(err, repo.ErrScopeKeyMissing),
		errors.Is(err, repo.ErrScopeNotCanonical),
		errors.Is(err, repo.ErrUnknownProvider):
		// The message names the offending key, which is what the operator needs
		// in order to fix it — and reveals nothing about anyone else's data.
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, repo.ErrScopeDisagrees):
		// The credential is not for the target it was filed under. Worth a
		// distinct status: this is a mis-filing, not a malformed request.
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, repo.ErrEstateTargetTaken):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, repo.ErrProviderMismatch):
		return huma.Error409Conflict(err.Error())

	case errors.Is(err, repo.ErrIdentityMissing), errors.Is(err, repo.ErrSecretMissing):
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, service.ErrTenantUnavailable):
		// The organization exists but its tenant database is not provisioned or
		// not reachable. 503 rather than 500: it is expected to resolve.
		slog.ErrorContext(ctx, "tenant database unavailable", "error", err)
		return huma.NewError(http.StatusServiceUnavailable, "this organization's storage is not ready yet")

	default:
		slog.ErrorContext(ctx, "estate operation failed", "error", err)
		return huma.Error500InternalServerError("estate operation failed")
	}
}

func Register(hapi huma.API, deps *app.Deps) {
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "estates-list",
		Method:      http.MethodGet,
		Path:        "/estates",
		Summary:     "List estates",
		Description: "Every estate in the caller's organization, with the fingerprint of each bound credential. Secret material is never returned.",
		Tags:        []string{"Estates"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListRequest) (*ListResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		list, err := deps.Estates.List(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &ListResponse{}
		out.Body.Estates = make([]EstateView, 0, len(list))
		for _, e := range list {
			out.Body.Estates = append(out.Body.Estates, toView(e))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "estates-get",
		Method:      http.MethodGet,
		Path:        "/estates/{id}",
		Summary:     "Get an estate",
		Tags:        []string{"Estates"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *GetRequest) (*GetResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid estate id")
		}

		v, err := deps.Estates.Get(ctx, p.OrganizationID, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &GetResponse{Body: toView(*v)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "estates-create",
		Method:      http.MethodPost,
		Path:        "/estates",
		Summary:     "Register an estate and its credential",
		Description: "Creates the credential and the estate that binds it, atomically. " +
			"Requires admin. Impersonated sessions are refused: an operator acting as a user " +
			"must not be able to install a credential they could later use.",
		Tags:     []string{"Estates"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateRequest) (*CreateResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		v, err := deps.Estates.Register(ctx, service.RegisterEstateInput{
			OrgID:          p.OrganizationID,
			Name:           in.Body.Name,
			Provider:       in.Body.Provider,
			Scope:          in.Body.Scope,
			CredentialName: in.Body.CredentialName,
			CredentialKind: in.Body.CredentialKind,
			Identity:       in.Body.Identity,
			Secret:         repo.Secret(in.Body.Secret),
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &CreateResponse{Body: toView(*v)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "estates-rotate-credential",
		Method:      http.MethodPost,
		Path:        "/estates/{id}/rotate",
		Summary:     "Rotate an estate's credential",
		Description: "Replaces the sealed material in place, keeping the binding. " +
			"The fingerprint changes, which is how the rotation is confirmed.",
		Tags:     []string{"Estates"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RotateRequest) (*RotateResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid estate id")
		}

		v, err := deps.Estates.RotateCredential(ctx, p.OrganizationID, id, in.Body.Identity, repo.Secret(in.Body.Secret))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &RotateResponse{Body: toView(*v)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "estates-delete",
		Method:      http.MethodDelete,
		Path:        "/estates/{id}",
		Summary:     "Delete an estate and its credential",
		Description: "Removes both. Deleting only the estate would leave a sealed credential " +
			"nothing references — unreachable through the API and still live at the provider.",
		Tags:     []string{"Estates"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeleteRequest) (*DeleteResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid estate id")
		}

		if err := deps.Estates.Delete(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &DeleteResponse{}
		out.Body.Deleted = true
		return out, nil
	})
}
