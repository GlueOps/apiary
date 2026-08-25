// Package sshkey exposes the credential the control plane uses to reach
// hypervisors.
//
// Nothing here returns a private key, and no response type has a field one could
// occupy. The private half is unsealed only by River workers, which are trusted
// by design and run with the same environment as the API — reaching it through
// an HTTP response would put a fleet-wide credential into whatever logs,
// caches and error trackers sit between here and a browser.
//
// What IS returned is the public half in authorized_keys form and the SHA256
// fingerprint, which is everything an operator needs: one to paste onto a host,
// the other to compare against `ssh-keygen -lf` once it is there.
package sshkey

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
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

type KeyView struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`

	// PublicKey is in authorized_keys form so it pastes onto a host unchanged.
	PublicKey string `json:"public_key"`

	// Fingerprint is OpenSSH's own SHA256 form, so it compares directly against
	// `ssh-keygen -lf` on the hypervisor — which is how an operator confirms the
	// key they whitelisted is the key the control plane holds.
	Fingerprint string `json:"fingerprint"`

	Purpose   string `json:"purpose" enum:"fleet,join"`
	Usable    bool   `json:"usable" doc:"Whether this key may still be used: not revoked, not expired, and correctly scoped if it is a join key."`
	Revoked   bool   `json:"revoked"`
	ExpiresAt string `json:"expires_at,omitempty" format:"date-time"`
	CreatedAt string `json:"created_at" format:"date-time"`
}

func toView(k tenant.SshKey, now time.Time) KeyView {
	v := KeyView{
		ID: k.ID.String(), Name: k.Name,
		PublicKey: k.PublicKey, Fingerprint: k.Fingerprint,
		Purpose: k.Purpose, Usable: k.Usable(now),
		Revoked:   k.RevokedAt != nil,
		CreatedAt: k.CreatedAt.UTC().Format(time.RFC3339),
	}
	if k.ExpiresAt != nil {
		v.ExpiresAt = k.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return v
}

type ListRequest struct{}
type ListResponse struct {
	Body struct {
		Keys []KeyView `json:"keys"`
	}
}

type MintRequest struct {
	Body struct {
		Name string `json:"name" required:"true" minLength:"1" maxLength:"100" doc:"What this key is for, so it can be recognised in an authorized_keys file a year from now."`
	}
}
type MintResponse struct{ Body KeyView }

type RevokeRequest struct {
	ID string `path:"id" format:"uuid"`
}
type RevokeResponse struct {
	Body struct {
		Revoked bool   `json:"revoked"`
		Note    string `json:"note"`
	}
}

func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrSSHKeyName):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, service.ErrSSHKeyNotFound), errors.Is(err, repo.ErrSSHKeyNotFound):
		return huma.Error404NotFound("no such key")
	case errors.Is(err, service.ErrTenantUnavailable):
		slog.ErrorContext(ctx, "tenant database unavailable", "error", err)
		return huma.NewError(http.StatusServiceUnavailable, "this organization's storage is not ready yet")
	default:
		slog.ErrorContext(ctx, "ssh key operation failed", "error", err)
		return huma.Error500InternalServerError("ssh key operation failed")
	}
}

// RegisterUserProtected mounts key management. Membership-gated, because whose
// hypervisors this key opens is a question about the tenant, not about a job
// title.
func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	svc := service.NewSSHKeyService(deps.Tenants)
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	member := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}
	admin := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin, apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "ssh-keys-list",
		Method:      http.MethodGet,
		Path:        "/ssh-keys",
		Summary:     "List fleet SSH keys",
		Description: "Public halves and fingerprints. The private half is never returned by any endpoint — it is unsealed only inside a River worker, so a fleet-wide credential never passes through a response, a log or an error tracker.",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListRequest) (*ListResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		keys, err := svc.List(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		now := time.Now().UTC()
		out := &ListResponse{}
		out.Body.Keys = make([]KeyView, 0, len(keys))
		for _, k := range keys {
			out.Body.Keys = append(out.Body.Keys, toView(k, now))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "ssh-keys-mint",
		Method:        http.MethodPost,
		Path:          "/ssh-keys",
		Summary:       "Mint a fleet SSH key",
		DefaultStatus: http.StatusCreated,
		Description: "Creates an Ed25519 keypair. The private half is sealed with this organization's own key and never leaves the control plane; the public half is returned in authorized_keys form for whitelisting on hypervisors. " +
			"Requires admin and refuses impersonated sessions: this key opens root on every host that trusts it, and an operator acting as a customer must not be able to mint one they could later use.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *MintRequest) (*MintResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		k, err := svc.MintFleetKey(ctx, p.OrganizationID, in.Body.Name)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &MintResponse{Body: toView(*k, time.Now().UTC())}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "ssh-keys-revoke",
		Method:      http.MethodDelete,
		Path:        "/ssh-keys/{id}",
		Summary:     "Revoke a fleet SSH key",
		Description: "Marks the key unusable. It does NOT remove the public half from any hypervisor — that happens when the agents next converge, so a revoked key stays whitelisted on a host that is offline. " +
			"The row is kept rather than deleted, so a fingerprint found in an authorized_keys file later can still be attributed to something.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RevokeRequest) (*RevokeResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid key id")
		}
		if err := svc.Revoke(ctx, p.OrganizationID, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &RevokeResponse{}
		out.Body.Revoked = true
		out.Body.Note = "The control plane will no longer use this key. It remains in authorized_keys on any host that has not converged since — including every host that is currently offline."
		return out, nil
	})
}
