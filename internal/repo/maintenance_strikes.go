package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// MaintenanceStrikes is a hypervisor's maintenance failures that still count.
type MaintenanceStrikes struct {
	Count int
	// The most recent of them.
	LatestAt     time.Time
	LatestReason string
	LatestRunID  uuid.UUID
}

// Flagged reports whether maintenance skips the node until it is acknowledged.
func (s MaintenanceStrikes) Flagged() bool { return s.Count >= tenant.MaintenanceStrikeLimit }

// LoadMaintenanceStrikes derives the strikes of the given hypervisors, or of
// every hypervisor when none are given. Hypervisors with none are absent.
//
// DERIVED, NOT STORED. A strike is a turn that ended failed WITH AN UPGRADE
// TASK QUEUED: the node was actually touched. A gate refusal, a busy agent and a
// skip queue nothing, so a healthy node refused because ANOTHER member was down
// is never flagged for it. Strikes count from the later of the machine's last
// successful maintenance and its last acknowledgement, so "two in a row" means
// two with no success between them, and an acknowledgement clears what came
// before it. A counter beside the rows would be one more thing to keep true.
func LoadMaintenanceStrikes(
	ctx context.Context, db *gorm.DB, hypervisorIDs ...uuid.UUID,
) (map[uuid.UUID]MaintenanceStrikes, error) {
	type row struct {
		HypervisorID uuid.UUID
		RunID        uuid.UUID
		Reason       string
		FinishedAt   time.Time
	}
	q := db.WithContext(ctx).Table("maintenance_run_nodes AS n").
		Select("n.hypervisor_id, n.run_id, n.reason, n.finished_at").
		Joins("JOIN hypervisors h ON h.id = n.hypervisor_id").
		Where("n.status = ? AND n.task_id IS NOT NULL AND n.finished_at IS NOT NULL",
			tenant.MaintenanceNodeFailed).
		Where("n.finished_at > coalesce(h.maintenance_acknowledged_at, '-infinity'::timestamptz)").
		Where(`n.finished_at > coalesce((SELECT max(s.finished_at) FROM maintenance_run_nodes s
			WHERE s.hypervisor_id = n.hypervisor_id AND s.status = ?), '-infinity'::timestamptz)`,
			tenant.MaintenanceNodeSucceeded).
		Order("n.finished_at DESC")
	if len(hypervisorIDs) > 0 {
		q = q.Where("n.hypervisor_id IN ?", hypervisorIDs)
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[uuid.UUID]MaintenanceStrikes{}
	for _, r := range rows {
		s, seen := out[r.HypervisorID]
		if !seen {
			// Newest first, so the first row seen is the latest failure.
			s = MaintenanceStrikes{LatestAt: r.FinishedAt, LatestReason: r.Reason, LatestRunID: r.RunID}
		}
		s.Count++
		out[r.HypervisorID] = s
	}
	return out, nil
}

// HiveKeepsASpareVote reports whether a hive still has quorum with one vote to
// spare after losing `self` on top of every other member not observed up. The
// members counted as down are returned for the refusal.
//
// The upgrade gate asks whether quorum survives losing one more node. This asks
// for one vote more than that, which is what a run must have before it carries
// on past a failure: with a failed node already down, an unrelated failure
// during the next reboot must not cost the cluster quorum.
func HiveKeepsASpareVote(
	ctx context.Context, db *gorm.DB, hiveID, self uuid.UUID, now time.Time,
) (bool, []string, int, error) {
	down, nodes, err := hiveMembersDown(ctx, db, hiveID, self, now)
	if err != nil {
		return false, nil, 0, err
	}
	return tenant.QuorumSurvivesLosing(nodes, len(down)+2), down, nodes, nil
}
