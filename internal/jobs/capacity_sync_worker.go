package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

// SyncHiveCapacityWorker refreshes one hive's capacity observations.
//
// One hive per job, fanned out by SyncAllHivesWorker, rather than one job that
// walks the fleet. A hive whose API is unreachable is the normal case during a
// datacenter incident, and in a single job it would take every other hive's sync
// down with it — including the observations the departure gate reads to decide
// whether a host is safe to remove.
type SyncHiveCapacityWorker struct {
	river.WorkerDefaults[SyncHiveCapacityArgs]
	deps CapacitySyncDeps
}

func NewSyncHiveCapacityWorker(d CapacitySyncDeps) *SyncHiveCapacityWorker {
	return &SyncHiveCapacityWorker{deps: d}
}

func (w *SyncHiveCapacityWorker) Work(ctx context.Context, job *river.Job[SyncHiveCapacityArgs]) error {
	orgID, err := uuid.Parse(job.Args.OrganizationID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("capacity sync: invalid organization id %q: %w",
			job.Args.OrganizationID, err))
	}
	hiveID, err := uuid.Parse(job.Args.HiveID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("capacity sync: invalid hive id %q: %w",
			job.Args.HiveID, err))
	}

	n, err := SyncHiveCapacity(ctx, w.deps, orgID, hiveID)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "capacity sync completed", "hive_id", hiveID, "nodes", n)
	return nil
}

// SyncAllHivesWorker enqueues one capacity sync per hive across every active
// organization.
//
// The fan-out is deliberately dumb: it enumerates and enqueues, and does no API
// work of its own. That keeps the periodic tick fast and bounded, so a fleet
// with a hundred hives does not turn the scheduled job into a long-running
// process holding a database connection while it talks to a hundred clusters.
type SyncAllHivesWorker struct {
	river.WorkerDefaults[SyncAllHivesArgs]
	controlDB *gorm.DB
	tenantDB  func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error)
	insert    func(ctx context.Context, args river.JobArgs) error
}

func NewSyncAllHivesWorker(
	controlDB *gorm.DB,
	tenantDB func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error),
	insert func(ctx context.Context, args river.JobArgs) error,
) *SyncAllHivesWorker {
	return &SyncAllHivesWorker{controlDB: controlDB, tenantDB: tenantDB, insert: insert}
}

func (w *SyncAllHivesWorker) Work(ctx context.Context, _ *river.Job[SyncAllHivesArgs]) error {
	var orgs []control.Organization
	if err := w.controlDB.WithContext(ctx).
		Where("status = ?", "active").
		Find(&orgs).Error; err != nil {
		return fmt.Errorf("capacity sync: list organizations: %w", err)
	}

	var enqueued, skipped int
	for _, org := range orgs {
		db, err := w.tenantDB(ctx, org.ID)
		if err != nil {
			// One organization's tenant database being unavailable must not stop
			// the others. Logged rather than returned, because returning would
			// retry the whole fan-out and re-enqueue every hive already handled.
			slog.WarnContext(ctx, "capacity sync: tenant database unavailable, skipping",
				"organization_id", org.ID, "error", err)
			skipped++
			continue
		}

		var hives []tenant.Hive
		if err := db.WithContext(ctx).
			Where("api_url <> '' AND credential_id IS NOT NULL").
			Find(&hives).Error; err != nil {
			slog.WarnContext(ctx, "capacity sync: could not list hives, skipping",
				"organization_id", org.ID, "error", err)
			skipped++
			continue
		}

		for _, h := range hives {
			if err := w.insert(ctx, SyncHiveCapacityArgs{
				OrganizationID: org.ID.String(),
				HiveID:         h.ID.String(),
			}); err != nil {
				if errors.Is(err, river.ErrNotFound) {
					continue
				}
				slog.WarnContext(ctx, "capacity sync: could not enqueue hive sync",
					"organization_id", org.ID, "hive_id", h.ID, "error", err)
				continue
			}
			enqueued++
		}
	}

	slog.InfoContext(ctx, "capacity sync fan-out complete",
		"organizations", len(orgs), "enqueued", enqueued, "skipped", skipped)
	return nil
}
