package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// DesiredMembershipForBee resolves the composition one bee's host should
// converge on.
//
// Returns (nil, nil) — HOLD — for a machine in no hive and for a hive that has
// published nothing. Neither is an instruction to leave a cluster, and treating
// them as one would dissolve an unpublished hive node by node.
func DesiredMembershipForBee(ctx context.Context, db *gorm.DB, beeID uuid.UUID) (*tenant.MembershipDesired, error) {
	var row struct {
		HiveID   *uuid.UUID
		Node     *string
		Revision int
		Digest   string
	}
	// One query across bee, hypervisor and hive. The bee is the authenticated
	// identity; the hypervisor is the machine it runs on and carries the node
	// name; the hive is what publishes.
	if err := db.WithContext(ctx).Raw(`
		SELECT h.hive_id            AS hive_id,
		       h.name               AS node,
		       COALESCE(v.membership_revision, 0) AS revision,
		       COALESCE(v.membership_digest, '')  AS digest
		  FROM hypervisors h
		  LEFT JOIN hives v ON v.id = h.hive_id
		 WHERE h.bee_id = ?`, beeID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("membership: resolve the bee's hive: %w", err)
	}
	// NO HIVE is the only HOLD. A hive that has published nothing is NOT one:
	// publication here is derived from the hypervisors assigned to it, so a hive
	// with members always has a composition to publish, and the first publish has
	// to happen somewhere.
	//
	// Returning early on revision 0 — which this did — made that impossible. The
	// repair below is what publishes, and it sat behind a check that bailed
	// before reaching it, so a hive that had never published never would: both
	// lab hives sat at revision 0 through a full restart with five agents
	// heartbeating at them.
	if row.HiveID == nil {
		return nil, nil
	}

	name, members, err := ResolveMembership(ctx, db, *row.HiveID)
	if err != nil {
		return nil, err
	}
	doc, digest := tenant.RenderMembership(name, members)

	// The published digest and the rendered one disagreeing means either that a
	// write moved membership without publishing, or that this hive has never
	// published at all — the stored digest is empty and matches nothing. Both are
	// repaired the same way, and both must be, because serving a fresh body under
	// a stale revision would make every agent converge and be judged diverged
	// again on the next heartbeat, forever.
	//
	// Idempotent, so the common case — everything already published — costs one
	// locked read and no revision.
	if digest != row.Digest {
		res, perr := PublishHiveMembership(ctx, db, *row.HiveID)
		switch {
		case perr == nil, errors.Is(perr, ErrMembershipUnchanged):
			row.Revision, row.Digest = res.Revision, res.Digest
		default:
			return nil, perr
		}
	}

	self := ""
	if row.Node != nil {
		self = *row.Node
	}
	return &tenant.MembershipDesired{
		Revision: row.Revision, Digest: row.Digest, Membership: doc, Self: self,
	}, nil
}

// RecordObservedMembership stores a host's own report of the cluster it sees.
func RecordObservedMembership(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, obs tenant.Observed, fingerprint string,
) error {
	// Named columns, never a struct: an Updates over a Bee struct would write
	// every zero-valued field on the model and blank the agent version, the
	// applied firewall revision and the unavailability declaration along with it.
	fields := map[string]any{
		"observed_cluster":       obs.Cluster,
		"observed_members":       datatypes.NewJSONType(obs.Members),
		"observed_quorate":       obs.Quorate,
		"observed_membership_at": time.Now().UTC(),
	}
	// Only written when the host actually reported one. An agent that could not
	// read its own certificate must not blank the fingerprint every other node
	// joins through.
	if fingerprint != "" {
		fields["observed_fingerprint"] = fingerprint
	}
	return db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", beeID).
		Updates(fields).Error
}

// RecordAppliedMembership stores the composition a bee has converged on.
func RecordAppliedMembership(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, revision int, digest string,
) error {
	return db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", beeID).
		Updates(map[string]any{
			"applied_membership_revision": revision,
			"applied_membership_digest":   digest,
			"applied_membership_at":       time.Now().UTC(),
		}).Error
}
