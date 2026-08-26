package firewall

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

type PublishRequest struct {
	HiveID string `path:"hive_id" format:"uuid"`
	Body   struct {
		Reason string `json:"reason,omitempty" maxLength:"500" doc:"Why this policy is being published. Read later by whoever is deciding whether to back it out, which is the moment it matters most."`
	}
}

type PublishResponse struct {
	Body struct {
		Revision  int    `json:"revision"`
		Digest    string `json:"digest"`
		Bytes     int    `json:"bytes"`
		Unchanged bool   `json:"unchanged" doc:"True when the rendered policy matched the published one and no revision was created. Not a failure: publishing twice is the ordinary way to reach it."`
	}
}

type RevertRequest struct {
	HiveID string `path:"hive_id" format:"uuid"`
	Body   struct {
		ToRevision int    `json:"to_revision" required:"true" minimum:"1" doc:"The revision whose BODY to republish. The new revision number is higher than the current one — a backout never moves the number down."`
		Reason     string `json:"reason,omitempty" maxLength:"500"`
	}
}

type SnapshotView struct {
	Revision  int    `json:"revision"`
	Digest    string `json:"digest"`
	Bytes     int    `json:"bytes"`
	RevertOf  *int   `json:"revert_of,omitempty"`
	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"created_at" format:"date-time"`
	Current   bool   `json:"current" doc:"Whether this is the revision the fleet is being told to converge on."`
}

type SnapshotsRequest struct {
	HiveID string `path:"hive_id" format:"uuid"`
}

type SnapshotsResponse struct {
	Body struct {
		Snapshots []SnapshotView `json:"snapshots"`
	}
}

type SnapshotBodyRequest struct {
	HiveID   string `path:"hive_id" format:"uuid"`
	Revision int    `path:"revision" minimum:"1"`
}

type SnapshotBodyResponse struct {
	Body struct {
		Revision int    `json:"revision"`
		Digest   string `json:"digest"`
		Content  string `json:"content"`
	}
}

// registerPublish mounts the operator half of convergence: turning authored
// rules into a revision the fleet is told to converge on.
//
// Publishing is DELIBERATELY an act rather than a consequence of editing. Every
// edit landing on metal the moment it is saved makes a half-finished policy a
// live one, and the window between "allow the new port" and "and here is the
// deny that makes it meaningful" is exactly when a mistake is unrecoverable
// remotely.
func registerPublish(
	hapi huma.API,
	admin func(context.Context) (*apimiddleware.Principal, error),
	tenantDB func(context.Context, uuid.UUID) (*gorm.DB, error),
	resolve func(context.Context, *gorm.DB, uuid.UUID) (tenant.RenderInput, error),
) {
	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-publish",
		Method:        http.MethodPost,
		Path:          "/firewall/hives/{hive_id}/publish",
		Summary:       "Publish this fleet's rules as the next revision",
		DefaultStatus: http.StatusCreated,
		Description: "Renders every binding that applies to this fleet into one nft ruleset, " +
			"stores it, and points the fleet at it. Bees converge on their next heartbeat.\n\n" +
			"Publishing an UNCHANGED policy does nothing and says so. Every revision makes the " +
			"whole fleet re-apply, and re-applying an identical ruleset flushes and reloads every " +
			"host's firewall to arrive at the state it was already in.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *PublishRequest) (*PublishResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		hiveID, err := uuid.Parse(in.HiveID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid fleet id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapPublishErr(err)
		}

		res, err := repo.PublishHiveFirewall(ctx, db, hiveID, p.UserID, in.Body.Reason, resolve)
		out := &PublishResponse{}
		if errors.Is(err, repo.ErrNothingToPublish) {
			// 201 with unchanged=true rather than an error. Nothing went wrong: the
			// operator asked for the published policy to match the authored one, and
			// it already does. A 409 here would train people to ignore 409s from
			// this endpoint, which is where the real refusals live.
			out.Body.Unchanged = true
			return out, nil
		}
		if err != nil {
			return nil, mapPublishErr(err)
		}
		out.Body.Revision = res.Revision
		out.Body.Digest = res.Digest
		out.Body.Bytes = res.Bytes
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "firewall-revert",
		Method:        http.MethodPost,
		Path:          "/firewall/hives/{hive_id}/revert",
		Summary:       "Republish an earlier revision's ruleset",
		DefaultStatus: http.StatusCreated,
		Description: "Publishes an OLD revision's body as a NEW revision. The number goes up, " +
			"never down: a bee decides whether to converge by comparing what it applied against " +
			"this number, and a number that moved backwards leaves a host pinned to the very " +
			"ruleset being backed out.\n\n" +
			"The old body verbatim, not a re-render of the old rules — the rules have moved on, " +
			"which is why this is a backout.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RevertRequest) (*PublishResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		hiveID, err := uuid.Parse(in.HiveID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid fleet id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapPublishErr(err)
		}

		res, err := repo.RevertHiveFirewall(ctx, db, hiveID, in.Body.ToRevision,
			p.UserID, in.Body.Reason)
		if err != nil {
			return nil, mapPublishErr(err)
		}
		out := &PublishResponse{}
		out.Body.Revision = res.Revision
		out.Body.Digest = res.Digest
		out.Body.Bytes = res.Bytes
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-snapshots-list",
		Method:      http.MethodGet,
		Path:        "/firewall/hives/{hive_id}/snapshots",
		Summary:     "List this fleet's published revisions",
		Description: "Newest first. This is what a backout is chosen from, so it carries the " +
			"reason each revision was published and which revision a backout reproduced.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *SnapshotsRequest) (*SnapshotsResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		hiveID, err := uuid.Parse(in.HiveID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid fleet id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapPublishErr(err)
		}

		var hive tenant.Hive
		if err := db.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
			return nil, mapPublishErr(err)
		}
		var snaps []tenant.FirewallSnapshot
		if err := db.WithContext(ctx).Where("hive_id = ?", hiveID).
			Order("revision DESC").Find(&snaps).Error; err != nil {
			return nil, mapPublishErr(err)
		}

		out := &SnapshotsResponse{}
		out.Body.Snapshots = make([]SnapshotView, 0, len(snaps))
		for _, s := range snaps {
			out.Body.Snapshots = append(out.Body.Snapshots, SnapshotView{
				Revision: s.Revision, Digest: s.Digest,
				// The body is deliberately not in the listing. It is tens of kilobytes
				// per revision and this endpoint exists to choose between them.
				Bytes:     len(s.Content),
				RevertOf:  s.RevertOf,
				Reason:    s.Reason,
				CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
				Current:   s.Revision == hive.FirewallRevision,
			})
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "firewall-snapshot-body",
		Method:      http.MethodGet,
		Path:        "/firewall/hives/{hive_id}/snapshots/{revision}",
		Summary:     "Read one published revision's ruleset",
		Description: "The exact text a host loads, so it can be compared against `nft list " +
			"ruleset` on the machine. That comparison is the only way to answer 'is this host " +
			"running what we think' from the host itself.",
		Tags:     []string{"Firewall"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *SnapshotBodyRequest) (*SnapshotBodyResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		hiveID, err := uuid.Parse(in.HiveID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid fleet id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapPublishErr(err)
		}
		var snap tenant.FirewallSnapshot
		if err := db.WithContext(ctx).
			First(&snap, "hive_id = ? AND revision = ?", hiveID, in.Revision).Error; err != nil {
			return nil, mapPublishErr(err)
		}
		out := &SnapshotBodyResponse{}
		out.Body.Revision = snap.Revision
		out.Body.Digest = snap.Digest
		out.Body.Content = snap.Content
		return out, nil
	})
}

// mapPublishErr keeps a missing fleet a 404 rather than a 500.
func mapPublishErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return huma.Error404NotFound("no such fleet or revision")
	}
	return huma.Error500InternalServerError("could not publish the firewall policy", err)
}
