package tenant

import (
	"testing"

	"github.com/google/uuid"
)

func cap3(cpu, ram, disk int) Capacity { return Capacity{CPU: cpu, RAMGB: ram, DiskGB: disk} }

func candidates(n int, each Capacity) []Candidate {
	out := make([]Candidate, n)
	for i := range out {
		out[i] = Candidate{HypervisorID: uuid.New(), Remaining: each}
	}
	return out
}

func counts(ids []uuid.UUID) map[uuid.UUID]int {
	m := map[uuid.UUID]int{}
	for _, id := range ids {
		m[id]++
	}
	return m
}

// The reason to run an oracle at all.
//
// Packing one pool onto one node is efficient until that node reboots, and then
// every replica of one workload goes down together. Spread is the product.
func TestPlanSpreadsAPoolAcrossNodes(t *testing.T) {
	hvs := candidates(4, cap3(64, 256, 2000))
	ids, fit, ok := PlanPlacements(hvs, cap3(4, 8, 50), 4)

	if !ok || fit != 4 {
		t.Fatalf("plan did not place all four: fit=%d ok=%v", fit, ok)
	}
	per := counts(ids)
	if len(per) != 4 {
		t.Fatalf("four guests landed on %d nodes, want 4 — a pool packed onto fewer nodes "+
			"loses every replica together when one reboots: %v", len(per), per)
	}
}

// The bug this guards: if PoolCount is not incremented as the plan is built,
// every iteration sees identical counts, picks the same winner, and the entire
// batch lands on one node — silently, because capacity still fits.
//
// The node sizes are UNEVEN on purpose. With equal nodes the capacity decrement
// spreads the batch on its own — each assignment lowers that node's remaining
// CPU, so the CPU tie-break picks a different one next time — and the test
// passes even with the pool counting removed. It has to be a shape where
// capacity CANNOT do the spreading: one large node wins the CPU tie-break every
// round, so only the pool count can move the plan off it.
func TestPlanCountsItsOwnAssignmentsWhilePlanning(t *testing.T) {
	big := Candidate{HypervisorID: uuid.New(), Remaining: cap3(100, 1000, 10000)}
	small1 := Candidate{HypervisorID: uuid.New(), Remaining: cap3(10, 1000, 10000)}
	small2 := Candidate{HypervisorID: uuid.New(), Remaining: cap3(10, 1000, 10000)}

	ids, _, ok := PlanPlacements([]Candidate{big, small1, small2}, cap3(1, 1, 1), 3)
	if !ok {
		t.Fatal("plan failed with abundant capacity")
	}
	per := counts(ids)
	if len(per) != 3 {
		t.Fatalf("three guests used %d of 3 nodes: %v — the largest node wins every CPU "+
			"tie-break, so a planner that does not count its own assignments puts the whole "+
			"batch on it", len(per), per)
	}
}

// Existing occupancy steers the next placement. A pool already concentrated on
// one node should grow elsewhere.
func TestPlanPrefersNodesWithLeastOfThisPool(t *testing.T) {
	loaded := Candidate{HypervisorID: uuid.New(), Remaining: cap3(64, 256, 2000), PoolCount: 5}
	empty := Candidate{HypervisorID: uuid.New(), Remaining: cap3(64, 256, 2000), PoolCount: 0}

	ids, _, ok := PlanPlacements([]Candidate{loaded, empty}, cap3(4, 8, 50), 1)
	if !ok || len(ids) != 1 {
		t.Fatalf("no placement made: %v", ids)
	}
	if ids[0] != empty.HypervisorID {
		t.Fatal("placed onto the node already carrying five of this pool, ignoring spread")
	}
}

// Among nodes equally good for spread, prefer room to grow — so the fleet
// degrades evenly instead of filling one node to the brim.
func TestPlanBreaksSpreadTiesByRemainingCPU(t *testing.T) {
	small := Candidate{HypervisorID: uuid.New(), Remaining: cap3(8, 256, 2000)}
	large := Candidate{HypervisorID: uuid.New(), Remaining: cap3(64, 256, 2000)}

	ids, _, ok := PlanPlacements([]Candidate{small, large}, cap3(4, 8, 50), 1)
	if !ok {
		t.Fatal("no placement made")
	}
	if ids[0] != large.HypervisorID {
		t.Fatal("tie broken toward the fuller node")
	}
}

// Every dimension binds independently. A node with spare CPU and no RAM must not
// be chosen — the classic way an oracle overcommits memory, which kills guests
// rather than merely slowing them.
func TestPlanRespectsEveryDimension(t *testing.T) {
	cases := []struct {
		name string
		rem  Capacity
	}{
		{"no cpu", cap3(1, 256, 2000)},
		{"no ram", cap3(64, 4, 2000)},
		{"no disk", cap3(64, 256, 10)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hvs := []Candidate{{HypervisorID: uuid.New(), Remaining: c.rem}}
			_, fit, ok := PlanPlacements(hvs, cap3(4, 8, 50), 1)
			if ok || fit != 0 {
				t.Fatalf("placed a guest that does not fit on %s: fit=%d ok=%v", c.name, fit, ok)
			}
		})
	}
}

// A partial answer is more useful than a bare failure: "three of five fit" tells
// an operator whether to add capacity or shrink the request.
func TestPlanReportsHowManyFitWhenItCannotPlaceThemAll(t *testing.T) {
	hvs := []Candidate{{HypervisorID: uuid.New(), Remaining: cap3(12, 256, 2000)}}
	ids, fit, ok := PlanPlacements(hvs, cap3(4, 8, 50), 5)

	if ok {
		t.Fatal("claimed success with insufficient capacity")
	}
	if fit != 3 || len(ids) != 3 {
		t.Fatalf("12 cpu at 4 each should fit exactly 3, got fit=%d ids=%d", fit, len(ids))
	}
}

// A failed plan must not leave its partial effects in the caller's slice, or a
// rejected request would silently shrink the fleet's apparent capacity.
func TestPlanDoesNotMutateItsInput(t *testing.T) {
	hvs := candidates(2, cap3(8, 32, 200))
	before := []Capacity{hvs[0].Remaining, hvs[1].Remaining}

	_, _, _ = PlanPlacements(hvs, cap3(4, 8, 50), 10)

	for i := range hvs {
		if hvs[i].Remaining != before[i] {
			t.Fatalf("candidate %d was mutated: %+v -> %+v", i, before[i], hvs[i].Remaining)
		}
	}
}

func TestPlanHandlesNothingToPlaceAndNowhereToPlaceIt(t *testing.T) {
	if _, fit, ok := PlanPlacements(candidates(3, cap3(64, 256, 2000)), cap3(4, 8, 50), 0); !ok || fit != 0 {
		t.Fatalf("placing zero guests should trivially succeed: fit=%d ok=%v", fit, ok)
	}
	if _, fit, ok := PlanPlacements(nil, cap3(4, 8, 50), 1); ok || fit != 0 {
		t.Fatalf("placing onto no nodes should fail: fit=%d ok=%v", fit, ok)
	}
}
