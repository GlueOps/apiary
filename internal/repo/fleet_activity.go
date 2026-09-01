package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FleetActivityFilter narrows the trail. Every field is optional; the zero
// filter is "everything, newest first".
type FleetActivityFilter struct {
	Kind         string
	Status       string
	HypervisorID *uuid.UUID
	Limit        int
	Offset       int
}

// fleetActivityDefaultLimit and fleetActivityMaxLimit bound one page.
//
// A cap rather than a client's word: the error text on a task is unbounded prose
// from a remote host, so a page of five hundred is a response measured in
// megabytes for a screen that shows twenty rows.
const (
	fleetActivityDefaultLimit = 50
	fleetActivityMaxLimit     = 200
)

// FleetActivityRow is one thing an agent was asked to do, and what came of it.
//
// The machine's ADDRESS travels with the row. The trail is read by someone who
// has to go and look at a box, and an agent id names a database record rather
// than a host — resolving it per row on the client is a lookup that fails
// exactly when the agent has since been replaced, which is when the history
// matters most.
type FleetActivityRow struct {
	ID         uuid.UUID
	BeeID      uuid.UUID
	Address    string
	NodeName   string
	HiveName   string
	Kind       string
	Status     string
	Phase      string
	Error      string
	Attempt    int
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// ListFleetActivity returns the fleet's task history, newest first.
//
// LEFT JOINs throughout. A task outlives the machine it ran on — an agent can be
// deleted, a host can leave the hive it was acting against — and an inner join
// would silently drop exactly the rows an operator is looking for, which are the
// ones about work on a host that is no longer where it was.
func ListFleetActivity(
	ctx context.Context, db *gorm.DB, f FleetActivityFilter,
) ([]FleetActivityRow, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = fleetActivityDefaultLimit
	}
	if limit > fleetActivityMaxLimit {
		limit = fleetActivityMaxLimit
	}

	q := db.WithContext(ctx).
		Table("fleet_tasks AS t").
		Select(`t.id, t.bee_id, t.kind, t.status, coalesce(t.phase,'') AS phase,
		        coalesce(t.error,'') AS error, t.attempt, t.created_at,
		        t.started_at, t.finished_at,
		        coalesce(h.address,'') AS address, coalesce(h.name,'') AS node_name,
		        coalesce(v.name,'') AS hive_name`).
		Joins("LEFT JOIN hypervisors h ON h.bee_id = t.bee_id").
		Joins("LEFT JOIN hives v ON v.id = t.hive_id").
		Order("t.created_at DESC").
		Limit(limit).Offset(f.Offset)

	if f.Kind != "" {
		q = q.Where("t.kind = ?", f.Kind)
	}
	if f.Status != "" {
		q = q.Where("t.status = ?", f.Status)
	}
	if f.HypervisorID != nil {
		q = q.Where("h.id = ?", *f.HypervisorID)
	}

	var out []FleetActivityRow
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
