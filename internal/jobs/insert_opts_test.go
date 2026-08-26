package jobs

import (
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// TestEveryUniqueByStateCarriesRiversRequiredStates.
//
// River's UniqueOpts.validate refuses a ByState that omits any of available,
// pending, running or scheduled. validate is unexported and runs at INSERT, so
// an invalid set is not a compile error, not a startup error, and not a log line
// anyone sees — for a periodic job it presents as the tick never firing, with a
// registered worker, correct configuration and an empty river_job table.
//
// That is exactly how fleet.claim_pending_onboards shipped broken the first
// time: ByState listed available and running, being the only two states the rule
// cared about, and the onboarding tick silently never ran.
func TestEveryUniqueByStateCarriesRiversRequiredStates(t *testing.T) {
	t.Parallel()

	required := []rivertype.JobState{
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateRunning,
		rivertype.JobStateScheduled,
	}

	// Every args type in this package that customises its insert options. Listed
	// rather than reflected over, so adding one is a deliberate act that shows up
	// in review.
	for _, a := range []river.JobArgsWithInsertOpts{
		ClaimPendingOnboardsArgs{},
		DepartHypervisorArgs{},
		ArriveHypervisorArgs{},
		SyncHiveCapacityArgs{},
		SyncAllHivesArgs{},
	} {
		kind := a.(river.JobArgs).Kind()
		states := a.InsertOpts().UniqueOpts.ByState
		if len(states) == 0 {
			// An empty ByState means River's own defaults, which are valid.
			continue
		}
		have := make(map[rivertype.JobState]bool, len(states))
		for _, s := range states {
			have[s] = true
		}
		for _, want := range required {
			if !have[want] {
				t.Errorf("%s omits the required unique state %q: River rejects this at insert "+
					"time, so the job would silently never be enqueued", kind, want)
			}
		}
	}
}
