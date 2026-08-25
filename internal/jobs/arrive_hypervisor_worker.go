package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// ArriveHypervisorWorker adds one standalone host to a hive.
type ArriveHypervisorWorker struct {
	river.WorkerDefaults[ArriveHypervisorArgs]
	deps ArriveDeps
}

func NewArriveHypervisorWorker(d ArriveDeps) *ArriveHypervisorWorker {
	return &ArriveHypervisorWorker{deps: d}
}

func (w *ArriveHypervisorWorker) Work(ctx context.Context, job *river.Job[ArriveHypervisorArgs]) error {
	orgID, err := uuid.Parse(job.Args.OrganizationID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("arrive: invalid organization id %q: %w",
			job.Args.OrganizationID, err))
	}
	hypID, err := uuid.Parse(job.Args.HypervisorID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("arrive: invalid hypervisor id %q: %w",
			job.Args.HypervisorID, err))
	}
	hiveID, err := uuid.Parse(job.Args.TargetHiveID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("arrive: invalid target hive id %q: %w",
			job.Args.TargetHiveID, err))
	}

	err = Arrive(ctx, w.deps, ArriveInput{
		OrganizationID: orgID,
		HypervisorID:   hypID,
		TargetHiveID:   hiveID,
		Links:          job.Args.Links,
	})

	// Cancelled rather than retried, for the same reason as a departure refusal:
	// it is a decision about the current state of the host, and a backoff loop
	// turns one clear "this node still has a cluster config" into an hour of
	// identical failures that bury it.
	if errors.Is(err, ErrArrivalRefused) {
		return river.JobCancel(err)
	}
	return err
}
