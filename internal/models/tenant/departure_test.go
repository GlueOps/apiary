package tenant

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptime(t time.Time) *time.Time { return &t }

// hyp builds a cordoned, empty, freshly-synced node — the one shape that is
// allowed to depart. Each test spoils exactly one thing, so a test that passes
// for the wrong reason is visible.
func hyp(now time.Time) Hypervisor {
	hive := uuid.New()
	name := "hv5"
	return Hypervisor{
		ID:             uuid.New(),
		HiveID:         &hive,
		Name:           &name,
		Schedulable:    false,
		ObservedGuests: 0,
		LastSyncedAt:   ptime(now.Add(-30 * time.Second)),
	}
}

func codes(rs []DepartureRefusal) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Code)
	}
	return out
}

func hasCode(rs []DepartureRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

func TestDrainStateAt(t *testing.T) {
	now := time.Now()
	th := DrainThresholds{ObservationMaxAge: 10 * time.Minute}

	t.Run("uncordoned is active regardless of a stale sync", func(t *testing.T) {
		h := hyp(now)
		h.Schedulable = true
		h.LastSyncedAt = ptime(now.Add(-99 * time.Hour))
		got := DrainStateAt(now, h, 0, th)
		if got.State != DrainStateActive {
			t.Fatalf("want %s, got %s", DrainStateActive, got.State)
		}
		// Staleness is still reported — the operator should see they are looking
		// at an ancient reading even though the verdict did not need it.
		if !got.Stale {
			t.Error("a 99-hour-old sync should be reported stale even when active")
		}
	})

	t.Run("uncordoned with guests is still active, not draining", func(t *testing.T) {
		h := hyp(now)
		h.Schedulable = true
		h.ObservedGuests = 7
		if got := DrainStateAt(now, h, 0, th); got.State != DrainStateActive {
			t.Fatalf("a busy uncordoned node is active, got %s", got.State)
		}
	})

	t.Run("cordoned and empty and fresh is drained", func(t *testing.T) {
		if got := DrainStateAt(now, hyp(now), 0, th); got.State != DrainStateDrained {
			t.Fatalf("want %s, got %s", DrainStateDrained, got.State)
		}
	})

	t.Run("cordoned with observed guests is draining", func(t *testing.T) {
		h := hyp(now)
		h.ObservedGuests = 1
		if got := DrainStateAt(now, h, 0, th); got.State != DrainStateDraining {
			t.Fatalf("want %s, got %s", DrainStateDraining, got.State)
		}
	})

	t.Run("cordoned with held placements is draining even with zero observed guests", func(t *testing.T) {
		// The direction that catches a placement whose VM was deleted behind our
		// back: the provider says empty, the ledger says committed.
		if got := DrainStateAt(now, hyp(now), 2, th); got.State != DrainStateDraining {
			t.Fatalf("want %s, got %s", DrainStateDraining, got.State)
		}
	})

	t.Run("cordoned with a stale sync is unknown, not drained", func(t *testing.T) {
		h := hyp(now)
		h.LastSyncedAt = ptime(now.Add(-11 * time.Minute))
		got := DrainStateAt(now, h, 0, th)
		if got.State != DrainStateUnknown {
			t.Fatalf("want %s, got %s — a stale sync must not read as empty", DrainStateUnknown, got.State)
		}
	})

	t.Run("cordoned with no sync at all is unknown", func(t *testing.T) {
		h := hyp(now)
		h.LastSyncedAt = nil
		got := DrainStateAt(now, h, 0, th)
		if got.State != DrainStateUnknown {
			t.Fatalf("want %s, got %s", DrainStateUnknown, got.State)
		}
		if !got.Stale {
			t.Error("a node never synced must be reported stale")
		}
	})

	t.Run("the staleness boundary is inclusive of exactly-at-threshold", func(t *testing.T) {
		h := hyp(now)
		h.LastSyncedAt = ptime(now.Add(-10 * time.Minute))
		if got := DrainStateAt(now, h, 0, th); got.State != DrainStateDrained {
			t.Fatalf("exactly at the threshold is not yet stale, got %s", got.State)
		}
		h.LastSyncedAt = ptime(now.Add(-10*time.Minute - time.Nanosecond))
		if got := DrainStateAt(now, h, 0, th); got.State != DrainStateUnknown {
			t.Fatalf("one nanosecond past the threshold is stale, got %s", got.State)
		}
	})

	t.Run("the staleness boundary is the same on the uncordoned path", func(t *testing.T) {
		// Both branches compute staleness independently, so both boundaries need
		// covering — one shared test leaves the other free to drift.
		h := hyp(now)
		h.Schedulable = true
		h.LastSyncedAt = ptime(now.Add(-10 * time.Minute))
		if got := DrainStateAt(now, h, 0, th); got.Stale {
			t.Error("exactly at the threshold is not yet stale")
		}
		h.LastSyncedAt = ptime(now.Add(-10*time.Minute - time.Nanosecond))
		if got := DrainStateAt(now, h, 0, th); !got.Stale {
			t.Error("one nanosecond past the threshold is stale")
		}
	})

	t.Run("a zero threshold falls back to the default rather than staling everything", func(t *testing.T) {
		h := hyp(now)
		h.LastSyncedAt = ptime(now.Add(-time.Minute))
		if got := DrainStateAt(now, h, 0, DrainThresholds{}); got.State != DrainStateDrained {
			t.Fatalf("a zero threshold must mean unset, not zero tolerance; got %s", got.State)
		}
	})
}

func TestCheckDepartureAllowsTheDrainedCase(t *testing.T) {
	now := time.Now()
	h := hyp(now)
	req := DepartureRequest{
		Hive:           Hive{ID: *h.HiveID, Name: "dc1", MinimumNodes: 3},
		Hypervisor:     h,
		Drain:          DrainStateAt(now, h, 0, DrainThresholds{}),
		HiveNodeCount:  4,
		RemainingNodes: 3,
	}
	if rs := CheckDepartureState(req); len(rs) != 0 {
		t.Fatalf("a cordoned, empty, freshly-synced node in a 4-node hive must be allowed; refused with %v", codes(rs))
	}
}

func TestCheckDepartureRefusals(t *testing.T) {
	now := time.Now()
	base := func() DepartureRequest {
		h := hyp(now)
		return DepartureRequest{
			Hive:           Hive{ID: *h.HiveID, Name: "dc1", MinimumNodes: 3},
			Hypervisor:     h,
			Drain:          DrainStateAt(now, h, 0, DrainThresholds{}),
			HiveNodeCount:  4,
			RemainingNodes: 3,
		}
	}

	t.Run("uncordoned is refused", func(t *testing.T) {
		req := base()
		req.Hypervisor.Schedulable = true
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		if !hasCode(CheckDepartureState(req), RefuseNotCordoned) {
			t.Fatal("an uncordoned host must be refused")
		}
	})

	t.Run("observed guests are refused, and the message says stopped ones count", func(t *testing.T) {
		req := base()
		req.Hypervisor.ObservedGuests = 2
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		rs := CheckDepartureState(req)
		if !hasCode(rs, RefuseGuestsPresent) {
			t.Fatalf("guests must block departure; got %v", codes(rs))
		}
		for _, r := range rs {
			if r.Code == RefuseGuestsPresent {
				if !strings.Contains(r.Detail, "2") {
					t.Error("the message must name the count the operator has to clear")
				}
				// Checked across BOTH fields. That stopped guests count is the
				// non-obvious half, and which of the two halves carries it is a
				// presentation choice — losing it altogether is not. Asserting only
				// on Detail would have made moving it to Remedy look like a
				// regression, and asserting on neither is how it gets trimmed away
				// the next time somebody shortens a message for width.
				if !strings.Contains(r.Detail+" "+r.Remedy, "stopped") {
					t.Error("neither the message nor the remedy says stopped guests count — " +
						"that is the non-obvious half, and it is the half that gets trimmed")
				}
				if r.Remedy == "" {
					t.Error("no remedy: an operator told only that guests are present has to " +
						"guess that a sync must observe the result before retrying")
				}
			}
		}
	})

	t.Run("exactly one guest blocks departure", func(t *testing.T) {
		// The boundary, not a comfortable number. A gate written `> 1` passes
		// every test that uses two or more and lets a single live guest through.
		req := base()
		req.Hypervisor.ObservedGuests = 1
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		if !hasCode(CheckDepartureState(req), RefuseGuestsPresent) {
			t.Fatal("one guest is still a guest")
		}
	})

	t.Run("exactly one held placement blocks departure", func(t *testing.T) {
		req := base()
		req.Drain = DrainStateAt(now, req.Hypervisor, 1, DrainThresholds{})
		if !hasCode(CheckDepartureState(req), RefusePlacementsHeld) {
			t.Fatal("one held placement is still committed capacity")
		}
	})

	t.Run("an unset minimum defaults to three rather than to zero", func(t *testing.T) {
		// The distinction the fallback exists for. Reading unset as zero disables
		// the gate entirely on every hive that never set it — which is all of
		// them, since the column defaults to 0.
		req := base()
		req.Hive.MinimumNodes = 0
		req.HiveNodeCount, req.RemainingNodes = 3, 2
		if !hasCode(CheckDepartureState(req), RefuseHiveTooSmall) {
			t.Fatal("MinimumNodes=0 must mean the default of 3, so leaving 2 is refused")
		}
	})

	t.Run("held placements are refused separately from observed guests", func(t *testing.T) {
		req := base()
		req.Drain = DrainStateAt(now, req.Hypervisor, 3, DrainThresholds{})
		rs := CheckDepartureState(req)
		if !hasCode(rs, RefusePlacementsHeld) {
			t.Fatalf("held placements must block departure; got %v", codes(rs))
		}
		if hasCode(rs, RefuseGuestsPresent) {
			t.Error("zero observed guests must not also report guests_present — the two have different fixes")
		}
	})

	t.Run("a stale sync is refused as unknown, not as empty", func(t *testing.T) {
		req := base()
		req.Hypervisor.LastSyncedAt = ptime(now.Add(-time.Hour))
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		rs := CheckDepartureState(req)
		if !hasCode(rs, RefuseDrainUnknown) {
			t.Fatalf("a stale observation must refuse; got %v", codes(rs))
		}
	})

	t.Run("a departure that would leave fewer than the minimum is refused", func(t *testing.T) {
		// The minimum is a FLOOR on what remains, not a size to start from.
		req := base()
		req.HiveNodeCount, req.RemainingNodes = 3, 2
		rs := CheckDepartureState(req)
		if !hasCode(rs, RefuseHiveTooSmall) {
			t.Fatalf("leaving 2 under a minimum of 3 must refuse; got %v", codes(rs))
		}
	})

	t.Run("leaving exactly the minimum is allowed", func(t *testing.T) {
		req := base()
		req.Hive.MinimumNodes = 0 // unset -> default 3
		req.HiveNodeCount, req.RemainingNodes = 4, 3
		if rs := CheckDepartureState(req); len(rs) != 0 {
			t.Fatalf("leaving exactly the minimum must be allowed; refused with %v", codes(rs))
		}
	})

	t.Run("a departure already under way is not counted as remaining", func(t *testing.T) {
		// Four present, one already departing: this one would leave two.
		req := base()
		req.HiveNodeCount, req.RemainingNodes = 4, 2
		if !hasCode(CheckDepartureState(req), RefuseHiveTooSmall) {
			t.Fatal("a second departure while the first is running must count what remains")
		}
	})

	t.Run("the knob lets a smaller test fleet through", func(t *testing.T) {
		req := base()
		req.Hive.MinimumNodes = 2
		req.HiveNodeCount, req.RemainingNodes = 3, 2
		if rs := CheckDepartureState(req); len(rs) != 0 {
			t.Fatalf("an explicit minimum of 2 must permit leaving 2; refused with %v", codes(rs))
		}
	})

	t.Run("dismantling the last node is allowed only under a minimum of one", func(t *testing.T) {
		req := base()
		req.HiveNodeCount, req.RemainingNodes = 1, 0
		req.Hive.MinimumNodes = 1
		if rs := CheckDepartureState(req); len(rs) != 0 {
			t.Fatalf("a minimum of 1 must permit dismantling a one-node cluster; refused with %v", codes(rs))
		}
		req.Hive.MinimumNodes = 2
		if !hasCode(CheckDepartureState(req), RefuseHiveTooSmall) {
			t.Fatal("a minimum of 2 must refuse leaving nobody")
		}
	})

	t.Run("the hive-size refusal is reported even while still draining", func(t *testing.T) {
		// The point of reporting it early: it is the one refusal waiting cannot
		// fix, so discovering it after a full drain wastes the drain.
		req := base()
		req.HiveNodeCount, req.RemainingNodes = 2, 1
		req.Hypervisor.ObservedGuests = 5
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		rs := CheckDepartureState(req)
		if !hasCode(rs, RefuseHiveTooSmall) || !hasCode(rs, RefuseGuestsPresent) {
			t.Fatalf("both refusals must be reported together; got %v", codes(rs))
		}
	})

	t.Run("every reason is returned, not the first", func(t *testing.T) {
		req := base()
		req.HiveNodeCount, req.RemainingNodes = 1, 0
		req.Hypervisor.Schedulable = true
		req.Hypervisor.LastSyncedAt = ptime(now.Add(-time.Hour))
		req.Drain = DrainStateAt(now, req.Hypervisor, 0, DrainThresholds{})
		rs := CheckDepartureState(req)
		if len(rs) < 2 {
			t.Fatalf("a request failing several gates must report them all; got %v", codes(rs))
		}
	})

	t.Run("a fleetless host reports not_a_member and nothing else", func(t *testing.T) {
		req := base()
		req.HiveNodeCount = 0
		rs := CheckDepartureState(req)
		if len(rs) != 1 || rs[0].Code != RefuseNotAMember {
			t.Fatalf("want a single not_a_member refusal, got %v", codes(rs))
		}
	})

	t.Run("a machine in no hive reports not_a_member and nothing else", func(t *testing.T) {
		// The OTHER way a machine can be fleetless, and the one the nullable
		// hive_id introduced. HiveID nil with a node count that still looks
		// plausible: before the pointer this was uuid.Nil and the zero value did
		// the work by accident. Everything else about this request is departable,
		// so a gate that stopped checking membership would report no refusal at
		// all and a worker would go and dismantle a cluster the machine is not in.
		req := base()
		req.Hypervisor.HiveID = nil
		rs := CheckDepartureState(req)
		if len(rs) != 1 || rs[0].Code != RefuseNotAMember {
			t.Fatalf("want a single not_a_member refusal, got %v", codes(rs))
		}
	})
}

func TestDepartureRefusalsError(t *testing.T) {
	if err := DepartureRefusalsError(nil); err != nil {
		t.Fatalf("no refusals must be no error, got %v", err)
	}
	err := DepartureRefusalsError([]DepartureRefusal{
		{Code: RefuseNotCordoned, Detail: "a"},
		{Code: RefuseGuestsPresent, Detail: "b"},
	})
	// Both must survive into the message: a caller that drops the rest leaves the
	// operator fixing one gate at a time, one worker run each.
	if !strings.Contains(err.Error(), RefuseNotCordoned) || !strings.Contains(err.Error(), RefuseGuestsPresent) {
		t.Fatalf("every refusal must appear in the rendered error, got %q", err.Error())
	}
}

func TestCandidateForIsArithmeticOnly(t *testing.T) {
	hive := Hive{CPUOvercommitRatio: 1}
	h := Hypervisor{
		ID: uuid.New(), Schedulable: true,
		ObservedCPUTotal: 8, ObservedRAMGBTotal: 16, ObservedDiskGBTotal: 100,
	}

	c := CandidateFor(hive, h, Capacity{CPU: 2, RAMGB: 4, DiskGB: 10}, 3)
	if c.HypervisorID != h.ID {
		t.Error("the candidate must carry the hypervisor id")
	}
	if c.Remaining.CPU != 6 || c.Remaining.RAMGB != 12 || c.Remaining.DiskGB != 90 {
		t.Errorf("remaining: want 6/12/90, got %+v", c.Remaining)
	}

	// The fixture above has NO foreign load, which is precisely why it passed
	// against the version that never subtracted one. A node carrying guests
	// nobody placed through this system must lose that capacity too.
	f := h
	f.ObservedForeignCPUUsed, f.ObservedForeignRAMGBUsed, f.ObservedForeignDiskGBUsed = 3, 5, 20
	c = CandidateFor(hive, f, Capacity{CPU: 2, RAMGB: 4, DiskGB: 10}, 3)
	if c.Remaining.CPU != 3 || c.Remaining.RAMGB != 7 || c.Remaining.DiskGB != 70 {
		t.Errorf("remaining must be allocatable - foreign - intended (8-3-2 / 16-5-4 / "+
			"100-20-10 = 3/7/70), got %+v — a hypervisor carrying hand-created VMs otherwise "+
			"reports its full ceiling as free and the oracle keeps filling it", c.Remaining)
	}
	if c.PoolCount != 3 {
		t.Errorf("pool count: want 3, got %d", c.PoolCount)
	}

	// Deliberately NOT filtered here. The cordon is a query condition — see
	// SchedulablePredicate — because it has to be evaluated inside the same
	// SELECT ... FOR UPDATE as the insert that follows. A Go-level check would be
	// a second definition of one rule, and it would be the weaker of the two: it
	// runs at a different instant from the lock, so a host cordoned in between
	// would slip through it anyway.
	h.Schedulable = false
	if got := CandidateFor(hive, h, Capacity{}, 0); got.HypervisorID != h.ID {
		t.Error("CandidateFor must not silently drop a cordoned node; eligibility is the " +
			"query's job, and pretending otherwise hides where the rule actually lives")
	}
}

func TestSchedulablePredicateIsTheCordonRule(t *testing.T) {
	// A constant rather than a literal repeated at each call site, so the
	// placement query and any capacity report cannot disagree about what "this
	// node accepts work" means. Asserted because a silent edit here changes which
	// hosts Terraform places onto.
	if SchedulablePredicate != "schedulable IS TRUE" {
		t.Fatalf("the cordon predicate changed to %q — every placement query carries this, "+
			"so a change here withdraws or admits hosts fleet-wide", SchedulablePredicate)
	}
}

func TestCandidateForFloorsRemainingAtZero(t *testing.T) {
	// An overcommitted node must not report negative room: any caller comparing
	// with >= would read it as capacity.
	hive := Hive{CPUOvercommitRatio: 1}
	h := Hypervisor{ID: uuid.New(), Schedulable: true, ObservedCPUTotal: 4, ObservedRAMGBTotal: 8}
	c := CandidateFor(hive, h, Capacity{CPU: 99, RAMGB: 99, DiskGB: 99}, 0)
	if c.Remaining.CPU != 0 || c.Remaining.RAMGB != 0 || c.Remaining.DiskGB != 0 {
		t.Fatalf("remaining must floor at zero, got %+v", c.Remaining)
	}

	// The floor must hold when the overrun comes from the FOREIGN term alone,
	// with nothing intended at all. That is the realistic shape of it: a host
	// somebody loaded up by hand, which this system has placed nothing on.
	f := Hypervisor{ID: uuid.New(), Schedulable: true, ObservedCPUTotal: 4, ObservedRAMGBTotal: 8}
	f.ObservedForeignCPUUsed, f.ObservedForeignRAMGBUsed, f.ObservedForeignDiskGBUsed = 99, 99, 99
	c = CandidateFor(hive, f, Capacity{}, 0)
	if c.Remaining.CPU != 0 || c.Remaining.RAMGB != 0 || c.Remaining.DiskGB != 0 {
		t.Fatalf("a foreign-only overrun must floor at zero, got %+v — a negative figure reads "+
			"as capacity to every caller comparing with >=", c.Remaining)
	}
}

func TestBookableIsTheOneDefinitionFitsAndCandidateForShare(t *testing.T) {
	// Two spellings of "how much room is there" drift, and the direction they
	// drift in is overcommit — the same reasoning that makes
	// PlacementHoldsCapacity a single function. This asserts the two agree
	// on a node whose foreign load is the ONLY thing standing between the slot
	// and a fit.
	hive := Hive{CPUOvercommitRatio: 1}
	h := Hypervisor{
		ID: uuid.New(), Schedulable: true,
		ObservedCPUTotal: 8, ObservedRAMGBTotal: 16, ObservedDiskGBTotal: 100,
		ObservedForeignCPUUsed: 6, ObservedForeignRAMGBUsed: 12, ObservedForeignDiskGBUsed: 80,
	}
	b := h.Bookable(hive)
	if b.CPU != 2 || b.RAMGB != 4 || b.DiskGB != 20 {
		t.Fatalf("bookable = allocatable - foreign: want 2/4/20, got %+v", b)
	}

	slot := Slot{VCPU: 4, RAMGB: 4, DiskGB: 10}
	if h.Fits(hive, Capacity{}, slot) {
		t.Error("Fits must subtract the foreign load too: 4 vCPU cannot land on a node whose " +
			"8 cores are already 6 spoken for by guests this system did not place")
	}
	if got := CandidateFor(hive, h, Capacity{}, 0); got.Remaining.CPU != 2 {
		t.Errorf("CandidateFor must report the same 2 cores Bookable does, got %d — one "+
			"formula, so the planner and the fit check cannot disagree", got.Remaining.CPU)
	}

	// And the pair still agrees once ours is committed on top. Ours is counted
	// ONCE, through intended; a node with 8 cores, 2 foreign and 2 intended has
	// 4 left, not 8 and not 2.
	h.ObservedForeignCPUUsed = 2
	if got := CandidateFor(hive, h, Capacity{CPU: 2}, 0); got.Remaining.CPU != 4 {
		t.Errorf("want 4 cores left (8 - 2 foreign - 2 intended), got %d", got.Remaining.CPU)
	}
}

func TestClusteredPredicateIsPinnedLikeItsSiblings(t *testing.T) {
	// Pinned for the same reason as the other two, and pinned NOW precisely
	// because nothing consumes it yet: a literal with no caller is exactly the one
	// that drifts unnoticed, and by the time a placement query carries it the
	// change will look deliberate.
	//
	// It matters because a machine with no hive is not a placement candidate — it
	// is being onboarded, or it is between clusters mid-migration. Widening this
	// to something always-true would hand guests to metal that is not in a cluster
	// at all.
	if ClusteredPredicate != "hive_id IS NOT NULL" {
		t.Fatalf("the clustered predicate changed to %q — it decides whether unclustered "+
			"metal can receive guests", ClusteredPredicate)
	}
}

func TestProvisionedPredicateIsTheOnboardingGate(t *testing.T) {
	// A THIRD axis, not more of the cordon. Half-onboarded metal is reachable, in
	// the cluster, and schedulable by default — nothing else in the placement
	// query keeps guests off it. Asserted as a literal for the same reason as the
	// cordon predicate: an edit here silently changes which hosts Terraform places
	// onto, and the value has to keep agreeing with the OnboardX constants that
	// ck_hypervisors_provisioning_state enforces.
	if ProvisionedPredicate != "provisioning_state = 'provisioned'" {
		t.Fatalf("the onboarding predicate changed to %q — every placement query carries "+
			"this, so a change here admits half-installed metal fleet-wide", ProvisionedPredicate)
	}
	if !strings.Contains(ProvisionedPredicate, OnboardProvisioned) {
		t.Fatalf("the predicate must name OnboardProvisioned (%q), or the SQL and the Go "+
			"state machine can drift apart with nothing to notice", OnboardProvisioned)
	}
}

// The credential decision is the one gate an operator answers rather than fixes,
// so it is checked at execution and deliberately absent from the preview.
func TestCheckDepartureRequiresACredentialDecision(t *testing.T) {
	now := time.Now()
	h := hyp(now)
	req := DepartureRequest{
		Hive:           Hive{ID: *h.HiveID, Name: "dc1", MinimumNodes: 3},
		Hypervisor:     h,
		Drain:          DrainStateAt(now, h, 0, DrainThresholds{}),
		HiveNodeCount:  4,
		RemainingNodes: 3,
	}

	// The preview is about the cluster, and must not report an unanswered
	// question as a blocked cluster — a Depart control that can never enable.
	if rs := CheckDepartureState(req); len(rs) != 0 {
		t.Fatalf("the state preview must ignore the credential decision; refused with %v", codes(rs))
	}

	// Executing without one refuses. Not a default: rotating disrupts every API
	// client in the fleet and accepting leaves the departed host able to mint
	// tickets, so choosing silently would pick which of those the operator can
	// afford.
	if !hasCode(CheckDeparture(req), RefuseCredentialsUndecided) {
		t.Fatalf("an unset disposition must refuse; got %v", codes(CheckDeparture(req)))
	}
	for _, bad := range []string{"", "  ", "yes", "true", "ROTATE"} {
		req.CredentialDisposition = bad
		if !hasCode(CheckDeparture(req), RefuseCredentialsUndecided) {
			t.Fatalf("%q must not be accepted as a decision", bad)
		}
	}
	// Accepting the exposure is a decision this control plane can carry out.
	req.CredentialDisposition = CredentialsAcceptExposure
	if rs := CheckDeparture(req); len(rs) != 0 {
		t.Fatalf("accepting the exposure must be allowed; refused with %v", codes(rs))
	}

	// Rotation is a decision it CAN carry out, on the departing node: that host
	// leaves with a fresh root keypair, sharing nothing with the cluster. It does
	// NOT rotate the cluster's own secrets, which is a separate operator decision
	// with a fleet-wide blast radius.
	req.CredentialDisposition = CredentialsRotate
	if rs := CheckDeparture(req); len(rs) != 0 {
		t.Fatalf("rotate must be allowed; refused with %v", codes(rs))
	}
}

// A host that is not ready to leave must not also be told to answer a question
// about credentials — the fixable list is what the operator can act on, and
// stacking an unanswerable item onto it buries them.
func TestCheckDepartureDoesNotAskAboutCredentialsWhileStateRefuses(t *testing.T) {
	now := time.Now()
	h := hyp(now)
	h.Schedulable = true // uncordoned: a state refusal
	req := DepartureRequest{
		Hive:           Hive{ID: *h.HiveID, Name: "dc1", MinimumNodes: 3},
		Hypervisor:     h,
		Drain:          DrainStateAt(now, h, 0, DrainThresholds{}),
		HiveNodeCount:  4,
		RemainingNodes: 3,
	}

	rs := CheckDeparture(req)
	if !hasCode(rs, RefuseNotCordoned) {
		t.Fatalf("expected the cordon refusal; got %v", codes(rs))
	}
	if hasCode(rs, RefuseCredentialsUndecided) {
		t.Fatalf("the credential question must wait until the cluster is ready; got %v", codes(rs))
	}
}

// A HOST NOBODY IS WATCHING AND A HOST WHOSE WATCHER FELL BEHIND NEED DIFFERENT
// ADVICE.
//
// Both reach the gate as one stale timestamp, so the refusal used to report
// every silence as a stale capacity sync and tell the operator to run one. On a
// freshly rebuilt node that is unfollowable — there is no agent on it to sync,
// and the age quoted is simply how long the machine has stood there unobserved.
// Observed on a lab fleet rebuilt from scratch, where every node refused citing
// a sync it had never had.
func TestADepartureSaysWhichKindOfSilenceStoppedIt(t *testing.T) {
	base := DepartureRequest{
		Hive:                  Hive{Name: "lab-a", MinimumNodes: 1},
		HiveNodeCount:         3,
		CredentialDisposition: CredentialsRotate,
		Drain: DrainStatus{
			Cordoned: true, State: DrainStateUnknown, ObservationAge: 15*time.Minute + 22*time.Second,
		},
	}

	hiveID := uuid.New()
	beeID := uuid.New()

	// Never observed: no agent has ever enrolled on the machine.
	unwatched := base
	unwatched.Hypervisor = Hypervisor{HiveID: &hiveID}
	got := findRefusal(CheckDeparture(unwatched), RefuseDrainUnknown)
	if got == nil {
		t.Fatal("a host nothing observes was allowed to depart")
	}
	if !strings.Contains(got.Remedy, "Onboard") {
		t.Fatalf("remedy = %q, want the operator sent to onboard it — there is no agent on "+
			"this host to run a sync with", got.Remedy)
	}
	if strings.Contains(got.Detail, "15m22s") {
		t.Fatalf("detail = %q quotes an observation age for a host that has never been "+
			"observed, which reads as a lapsed agent rather than an absent one", got.Detail)
	}

	// Lapsed: an agent is linked, and it has stopped reporting.
	lapsed := base
	lapsed.Hypervisor = Hypervisor{HiveID: &hiveID, BeeID: &beeID}
	got = findRefusal(CheckDeparture(lapsed), RefuseDrainUnknown)
	if got == nil {
		t.Fatal("a host with a lapsed agent was allowed to depart")
	}
	if !strings.Contains(got.Detail, "15m22s") {
		t.Fatalf("detail = %q, want the actual age named so the operator can judge it",
			got.Detail)
	}
	if strings.Contains(got.Remedy, "Onboard") {
		t.Fatalf("remedy = %q tells the operator to onboard a host that is already onboarded",
			got.Remedy)
	}
}

func findRefusal(rs []DepartureRefusal, code string) *DepartureRefusal {
	for i := range rs {
		if rs[i].Code == code {
			return &rs[i]
		}
	}
	return nil
}
