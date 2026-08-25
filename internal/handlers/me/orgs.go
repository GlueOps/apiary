package me

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
)

type GetMyOrganizationsRequest struct{}

type GetMyOrganizationsResponse struct {
	Body struct {
		Organizations []MyOrganizationView `json:"organizations"`
	}
}

type MyOrganizationView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	Role           string `json:"role"`
	IsDefault      bool   `json:"is_default"`
	SupportEnabled bool   `json:"support_enabled"`
}

func RegisterOrganizations(hapi huma.API, deps *app.Deps) {
	membershipRepo := repo.NewMembershipRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "me-orgs-list",
		Method:      http.MethodGet,
		Path:        "/me/orgs",
		Summary:     "List my organizations",
		Description: "Returns the organizations the authenticated user belongs to.",
		Tags:        []string{"Profile"},
		Security:    apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *GetMyOrganizationsRequest) (*GetMyOrganizationsResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		memberships, err := membershipRepo.ListByUserID(ctx, *p.UserID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to load organizations", "error", err)
			return nil, huma.Error500InternalServerError("failed to load organizations")
		}

		out := &GetMyOrganizationsResponse{}
		out.Body.Organizations = make([]MyOrganizationView, 0, len(memberships))

		for _, m := range memberships {
			item := MyOrganizationView{
				ID:             m.OrganizationID.String(),
				Role:           string(m.Role),
				IsDefault:      m.OrganizationID == p.OrganizationID,
				Name:           m.Organization.Name,
				Status:         m.Organization.Status,
				SupportEnabled: m.SupportEnabled,
			}
			out.Body.Organizations = append(out.Body.Organizations, item)
		}

		return out, nil
	})
}
