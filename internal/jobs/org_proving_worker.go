package jobs

import (
	"context"
	"fmt"
	"log"

	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type OrgProvingWorker struct {
	river.WorkerDefaults[workflow.OrgProvingArgs]
	db           *gorm.DB
	distinctDays int
}

// NewOrgProvingWorker fails at wiring time rather than at 3am, matching
// NewTokenSessionCleanupWorker. A threshold below two would mark every
// organization proven on its registration session and disable the ceiling
// silently, which is exactly the kind of failure that must not be deferred to
// the first run.
func NewOrgProvingWorker(db *gorm.DB, distinctDays int) *OrgProvingWorker {
	if db == nil {
		panic("jobs: OrgProvingWorker requires a control database handle")
	}
	if distinctDays < ProvenDistinctDaysDefault {
		panic(fmt.Sprintf(
			"jobs: OrgProvingWorker requires at least %d distinct UTC days, got %d",
			ProvenDistinctDaysDefault, distinctDays,
		))
	}
	return &OrgProvingWorker{db: db, distinctDays: distinctDays}
}

func (w *OrgProvingWorker) Work(
	ctx context.Context,
	_ *river.Job[workflow.OrgProvingArgs],
) error {
	result, err := SweepOrganizationProving(ctx, w.db, w.distinctDays)
	if err != nil {
		return fmt.Errorf("organization proving sweep: %w", err)
	}

	if result.OrganizationsAdvanced > 0 {
		log.Printf(
			"[admission.org_proving] advanced %d organizations, %d newly proven (threshold %d distinct UTC days)",
			result.OrganizationsAdvanced, result.OrganizationsProven, w.distinctDays,
		)
	}
	return nil
}
