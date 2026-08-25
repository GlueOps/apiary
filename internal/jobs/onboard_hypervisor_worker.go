package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// OnboardHypervisorWorker installs the control plane's key and the agent on one
// host.
//
// River workers are trusted by design and run with the same environment as the
// API, which is what makes it acceptable for this one to unseal the fleet key's
// private half — something no handler may do.
type OnboardHypervisorWorker struct {
	river.WorkerDefaults[OnboardHypervisorArgs]
	deps OnboardDeps
}

func NewOnboardHypervisorWorker(d OnboardDeps) *OnboardHypervisorWorker {
	return &OnboardHypervisorWorker{deps: d}
}

func (w *OnboardHypervisorWorker) Work(ctx context.Context, job *river.Job[OnboardHypervisorArgs]) error {
	orgID, err := uuid.Parse(job.Args.OrganizationID)
	if err != nil {
		// Unparseable arguments will not become parseable on a retry, so this is
		// cancelled rather than left to exhaust the attempt budget and sit in the
		// queue looking like a transient failure.
		return river.JobCancel(fmt.Errorf("onboard: invalid organization id %q: %w",
			job.Args.OrganizationID, err))
	}
	if job.Args.Address == "" || job.Args.MachineID == "" {
		return river.JobCancel(fmt.Errorf("onboard: an address and a machine id are required"))
	}

	return Onboard(ctx, w.deps, OnboardInput{
		OrganizationID: orgID,
		Address:        job.Args.Address,
		User:           job.Args.User,
		MachineID:      job.Args.MachineID,
	})
}
