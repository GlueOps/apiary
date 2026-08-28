package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// BootstrapHiveWorker creates a hive's first cluster on one standalone host.
type BootstrapHiveWorker struct {
	river.WorkerDefaults[BootstrapHiveArgs]
	deps BootstrapDeps
}

func NewBootstrapHiveWorker(d BootstrapDeps) *BootstrapHiveWorker {
	return &BootstrapHiveWorker{deps: d}
}

func (w *BootstrapHiveWorker) Work(ctx context.Context, job *river.Job[BootstrapHiveArgs]) error {
	orgID, err := uuid.Parse(job.Args.OrganizationID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("bootstrap: invalid organization id %q: %w",
			job.Args.OrganizationID, err))
	}
	hypID, err := uuid.Parse(job.Args.HypervisorID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("bootstrap: invalid hypervisor id %q: %w",
			job.Args.HypervisorID, err))
	}
	hiveID, err := uuid.Parse(job.Args.TargetHiveID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("bootstrap: invalid target hive id %q: %w",
			job.Args.TargetHiveID, err))
	}

	err = BootstrapHive(ctx, w.deps, BootstrapInput{
		OrganizationID: orgID,
		HypervisorID:   hypID,
		TargetHiveID:   hiveID,
		Links:          job.Args.Links,
	})

	// Recorded against the MACHINE, success or failure, for the arrival's reason:
	// a bootstrap has no state column of its own, so a failure is otherwise a
	// retryable row in a queue and nothing on screen — an operator watching a
	// host that simply never forms a cluster.
	//
	// Filed under "join" rather than a kind of its own: to the operator reading
	// the row it answers the same question, and a second verb here would mean a
	// machine's last outcome could be one of two columns depending on which side
	// of the chicken-and-egg it was on.
	if w.deps.RecordOutcome != nil {
		if recErr := w.deps.RecordOutcome(ctx, orgID, hypID, "join", err); recErr != nil {
			slog.ErrorContext(ctx, "could not record the bootstrap outcome",
				"hypervisor_id", hypID, "error", recErr)
		}
	}

	// Cancelled rather than retried: a refusal is a decision about the current
	// state of the host, and a backoff loop turns one clear "this node is already
	// clustered" into an hour of identical failures that bury it.
	if errors.Is(err, ErrBootstrapRefused) {
		return river.JobCancel(err)
	}
	return err
}
