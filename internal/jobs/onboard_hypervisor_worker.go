package jobs

import (
	"context"
	"fmt"
	"log/slog"

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
	if job.Args.Address == "" {
		return river.JobCancel(fmt.Errorf("onboard: an address is required: there is nothing to connect to"))
	}
	// A machine id is NOT required. The ticker claims rows discovered by the
	// capacity sync, which are identified by (hive, name) and carry no machine id
	// at all — and the stored column holds a digest where this job needs the raw
	// value, so there would be nothing to pass even when it is set. The job asks
	// the host, which is the authority on its own identity. See ReadMachineID.

	hvID, _ := uuid.Parse(job.Args.HypervisorID)

	err = Onboard(ctx, w.deps, OnboardInput{
		OrganizationID: orgID,
		HypervisorID:   hvID,
		Address:        job.Args.Address,
		User:           job.Args.User,
		MachineID:      job.Args.MachineID,
	})

	// Conclude the row the ticker claimed, on BOTH paths. Recording only success
	// would leave a failed machine sitting in `provisioning` until its lease
	// lapsed, which reads to an operator as "still working on it" for fifteen
	// minutes after it stopped.
	if hvID, parseErr := uuid.Parse(job.Args.HypervisorID); parseErr == nil {
		if w.deps.RecordOutcome == nil {
			slog.WarnContext(ctx, "onboard: no outcome recorder configured; the machine's "+
				"provisioning state will be resolved by its lease expiring",
				"hypervisor_id", hvID)
		} else if held, recErr := w.deps.RecordOutcome(ctx, orgID, hvID, err); recErr != nil {
			// Logged, never returned. Returning would retry the whole onboard for a
			// bookkeeping failure, and onboarding a host again is a far larger act
			// than failing to write one column.
			slog.ErrorContext(ctx, "onboard: could not record the outcome",
				"hypervisor_id", hvID, "error", recErr)
		} else if !held {
			slog.WarnContext(ctx, "onboard: finished after the provisioning lease was taken "+
				"by a later tick; the outcome was not recorded",
				"hypervisor_id", hvID)
		}
	}

	return err
}
