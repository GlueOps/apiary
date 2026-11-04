package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/control"
)

// MaintenanceTickWorker runs AdvanceMaintenance for every active organization.
type MaintenanceTickWorker struct {
	river.WorkerDefaults[MaintenanceTickArgs]
	controlDB *gorm.DB
	tenantDB  func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error)
	cfg       MaintenanceConfig
}

func NewMaintenanceTickWorker(
	controlDB *gorm.DB,
	tenantDB func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error),
	cfg MaintenanceConfig,
) *MaintenanceTickWorker {
	return &MaintenanceTickWorker{controlDB: controlDB, tenantDB: tenantDB, cfg: cfg}
}

func (w *MaintenanceTickWorker) Work(ctx context.Context, _ *river.Job[MaintenanceTickArgs]) error {
	var orgs []control.Organization
	if err := w.controlDB.WithContext(ctx).Where("status = ?", "active").
		Find(&orgs).Error; err != nil {
		return fmt.Errorf("maintenance tick: list organizations: %w", err)
	}
	now := time.Now().UTC()
	for _, org := range orgs {
		db, err := w.tenantDB(ctx, org.ID)
		if err != nil {
			// Logged, not returned: a retry would re-run every other
			// organization's tick as well, and the next tick is a minute away.
			slog.WarnContext(ctx, "maintenance tick: tenant database unavailable, skipping",
				"organization_id", org.ID, "error", err)
			continue
		}
		if err := AdvanceMaintenance(ctx, db, w.cfg, now); err != nil {
			slog.WarnContext(ctx, "maintenance tick: could not advance maintenance",
				"organization_id", org.ID, "error", err)
		}
	}
	return nil
}
