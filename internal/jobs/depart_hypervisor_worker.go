package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// DepartHypervisorWorker removes one host from its hive.
type DepartHypervisorWorker struct {
	river.WorkerDefaults[DepartHypervisorArgs]
	deps DepartDeps
}

func NewDepartHypervisorWorker(d DepartDeps) *DepartHypervisorWorker {
	return &DepartHypervisorWorker{deps: d}
}

func (w *DepartHypervisorWorker) Work(ctx context.Context, job *river.Job[DepartHypervisorArgs]) error {
	orgID, err := uuid.Parse(job.Args.OrganizationID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("depart: invalid organization id %q: %w",
			job.Args.OrganizationID, err))
	}
	hypID, err := uuid.Parse(job.Args.HypervisorID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("depart: invalid hypervisor id %q: %w",
			job.Args.HypervisorID, err))
	}

	err = Depart(ctx, w.deps, DepartInput{OrganizationID: orgID, HypervisorID: hypID})

	// A refusal is CANCELLED rather than retried. It is a decision about the
	// current state of the fleet, and retrying it on a backoff schedule turns one
	// clear "this host has four guests on it" into an hour of identical failures
	// that bury the message. The operator drains the host and asks again — which
	// is the workflow this gate exists to enforce, not an inconvenience to be
	// smoothed over by a retry loop.
	if errors.Is(err, ErrDepartureRefused) {
		return river.JobCancel(err)
	}
	return err
}
