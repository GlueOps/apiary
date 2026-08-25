package tenant

import (
	"github.com/google/uuid"
)

// Candidate is one hypervisor as the planner sees it: what it can still take,
// and how much of the pool being placed already sits on it.
type Candidate struct {
	HypervisorID uuid.UUID

	// Remaining is allocatable minus intended, computed by the caller under the
	// same lock as the insert that follows. Passed in rather than derived here so
	// that a plan cannot be made from a stale read.
	Remaining Capacity

	// PoolCount is how many guests OF THIS POOL already sit on this node. It is
	// the anti-affinity signal: spreading one pool across nodes is what stops a
	// single hypervisor failure taking out every replica of one workload.
	PoolCount int
}

func (c Candidate) fits(cost Capacity) bool {
	return c.Remaining.CPU >= cost.CPU &&
		c.Remaining.RAMGB >= cost.RAMGB &&
		c.Remaining.DiskGB >= cost.DiskGB
}

// PlanPlacements assigns `count` guests of one slot across candidates.
//
// Greedy, and the ordering is the whole policy:
//
//  1. Fewest guests of THIS POOL first — anti-affinity. Packing a pool onto one
//     node is efficient right up to the moment that node reboots, and then every
//     replica of one workload goes down together. Spreading is the reason to run
//     an oracle rather than picking the emptiest host.
//  2. Ties broken by most remaining CPU — balance. Among nodes equally good for
//     spread, prefer the one with room to grow, so the fleet degrades evenly
//     rather than filling one node to the brim first.
//
// Pure and deterministic: no clock, no database, no randomness. The caller
// supplies capacity read under a lock and receives an ordered list of node ids.
// That separation is what makes the policy testable — waggle's version proved
// the same shape works, and the value here is that the rule is inspectable
// rather than buried in a query.
//
// A partial result is returned WITH the count that fit rather than an error
// alone, because "three of five would fit" is what an operator needs in order to
// decide whether to add capacity or shrink the request.
func PlanPlacements(candidates []Candidate, cost Capacity, count int) (ids []uuid.UUID, fit int, ok bool) {
	// Copied because the loop decrements remaining capacity as it assigns, and
	// mutating the caller's slice would make a failed plan leave its partial
	// effects behind in the caller's view of the fleet.
	work := make([]Candidate, len(candidates))
	copy(work, candidates)

	ids = make([]uuid.UUID, 0, count)
	for placed := 0; placed < count; placed++ {
		best := -1
		for i := range work {
			if !work[i].fits(cost) {
				continue
			}
			if best == -1 {
				best = i
				continue
			}
			if work[i].PoolCount < work[best].PoolCount ||
				(work[i].PoolCount == work[best].PoolCount &&
					work[i].Remaining.CPU > work[best].Remaining.CPU) {
				best = i
			}
		}
		if best == -1 {
			return ids, placed, false
		}
		ids = append(ids, work[best].HypervisorID)
		work[best].Remaining.CPU -= cost.CPU
		work[best].Remaining.RAMGB -= cost.RAMGB
		work[best].Remaining.DiskGB -= cost.DiskGB
		// Counted immediately, so the next guest of this pool prefers elsewhere.
		// Without it the whole batch lands on one node: every iteration would see
		// the same pool counts and pick the same winner.
		work[best].PoolCount++
	}
	return ids, count, true
}
