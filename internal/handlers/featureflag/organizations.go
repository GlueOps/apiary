package featureflag

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
)

// The tenant directory lives in this package rather than under handlers/org
// because of who it is for. handlers/org is a customer acting inside THEIR
// organization; this is a platform operator looking across all of them, and the
// two belong on different sides of the role boundary. Putting it beside the
// endpoints that need it also keeps the reason it exists visible: without a way
// to name organizations, setting a flag for one means pasting a uuid.

type OrganizationSummaryView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
	Status string `json:"status"`
	// Deleted rather than a timestamp, because the only question a caller has is
	// whether this is a live tenant. The date is a separate concern and this list
	// is not where an operator investigates a deletion.
	Deleted   bool   `json:"deleted"`
	CreatedAt string `json:"created_at" format:"date-time"`
}

type ListOrganizationsRequest struct {
	Query          string `query:"q" maxLength:"200" doc:"Match the name or domain, case-insensitively."`
	IncludeDeleted bool   `query:"include_deleted" doc:"Include soft-deleted organizations. Off by default: a deleted tenant is not something to target."`
	Limit          int    `query:"limit" minimum:"1" maximum:"200" doc:"Default 50."`
	Offset         int    `query:"offset" minimum:"0"`
}

type ListOrganizationsResponse struct {
	Body struct {
		Organizations []OrganizationSummaryView `json:"organizations"`
	}
}

// RegisterOrganizationDirectory mounts the tenant directory on the PLATFORM
// surface.
//
// Must be handed a group carrying RequireAdminAuth. This is the single most
// sensitive read in this package: the names of every organization on the
// installation are not any one customer's to see, and a list of tenants is the
// first thing an attacker with a foothold in one of them would want. The group
// is the gate, which is why no role check appears below.
func RegisterOrganizationDirectory(hapi huma.API, deps *app.Deps) {
	huma.Register(hapi, huma.Operation{
		OperationID: "platform-organizations-list",
		Method:      http.MethodGet,
		Path:        "/platform/organizations",
		Summary:     "The organizations on this installation",
		Description: "Names and ids, so that anything targeted at one organization can be chosen " +
			"rather than typed as a uuid.\n\n" +
			"Soft-deleted organizations are EXCLUDED unless asked for. A deleted tenant is not " +
			"something to target, and its row survives only so a mistaken deletion can be seen " +
			"and undone.\n\n" +
			"Carries no credential material of any kind. An organization row also holds the " +
			"sealed connection string and tenant key that decrypt that customer's database; " +
			"those columns are never read by this query.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *ListOrganizationsRequest) (*ListOrganizationsResponse, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if deps == nil || deps.ControlDB == nil {
			// A check that cannot reach a decision must deny.
			return nil, huma.Error503ServiceUnavailable("the control database is not available")
		}

		rows, err := repo.ListOrganizations(ctx, deps.ControlDB, repo.OrganizationFilter{
			Query:          strings.TrimSpace(in.Query),
			IncludeDeleted: in.IncludeDeleted,
			Limit:          in.Limit,
			Offset:         in.Offset,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("could not list organizations", err)
		}

		out := &ListOrganizationsResponse{}
		out.Body.Organizations = make([]OrganizationSummaryView, 0, len(rows))
		for _, r := range rows {
			out.Body.Organizations = append(out.Body.Organizations, OrganizationSummaryView{
				ID: r.ID.String(), Name: r.Name, Domain: r.Domain, Status: r.Status,
				Deleted:   r.DeletedAt != nil,
				CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		return out, nil
	})
}
