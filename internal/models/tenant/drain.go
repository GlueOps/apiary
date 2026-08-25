package tenant

import "time"

// Drain states. DERIVED, never stored — same reason as Bee.LivenessAt and
// Hypervisor.Intended: a stored drain flag is a cache of a number that changes
// underneath it, and the direction it drifts is "reported drained while guests
// are still running". That is precisely the reading the departure gate must not
// accept, so there is no column for it to read.
const (
	// DrainStateActive: the node still accepts placements. Nothing has been
	// asked of it.
	DrainStateActive = "active"
	// DrainStateDraining: cordoned, but work is still on it. New placements stop
	// here; existing ones are moved by something else — Terraform, usually — and
	// this system only observes the result.
	DrainStateDraining = "draining"
	// DrainStateDrained: cordoned and empty. The only state a departure may
	// start from.
	DrainStateDrained = "drained"
	// DrainStateUnknown: cordoned, but the observations are too old to say
	// whether it is empty. NOT a synonym for draining — it is the absence of an
	// answer, and it is kept distinct so the gate refuses rather than guessing in
	// whichever direction happens to be convenient.
	DrainStateUnknown = "unknown"
)

// DrainThresholds is a struct rather than a bare duration argument so that
// adding a second window later cannot silently swap two positional durations —
// the same reasoning as ProvisioningThresholds.
type DrainThresholds struct {
	// ObservationMaxAge is how stale a sync may be and still support a decision
	// about emptiness. Past it the answer is DrainStateUnknown.
	ObservationMaxAge time.Duration
}

// DefaultObservationMaxAge is deliberately short. This window is the difference
// between "this node is empty" and "this node was empty a while ago", and a
// departure acts on the answer by dismantling the node's cluster membership.
const DefaultObservationMaxAge = 10 * time.Minute

// EffectiveObservationMaxAge treats a non-positive value as unset. Zero would
// otherwise mean every observation is stale the instant it is written, which
// makes every host permanently undrainable — a plausible reading of the number
// and a useless one.
func (t DrainThresholds) EffectiveObservationMaxAge() time.Duration {
	if t.ObservationMaxAge <= 0 {
		return DefaultObservationMaxAge
	}
	return t.ObservationMaxAge
}

// DrainStatus is the full answer, not just the verdict.
//
// The counts and the staleness come back alongside the state because the
// operator's next question after "why can this not leave" is always "what is
// still on it", and a bare state string sends them to the provider's UI to find
// out.
type DrainStatus struct {
	State string

	// Cordoned is the operator's stored intent: Hypervisor.Schedulable inverted.
	Cordoned bool

	// ObservedGuests is what the provider last reported. Counts STOPPED guests,
	// which is the non-obvious half — see Hypervisor.ObservedGuests.
	ObservedGuests int

	// HeldPlacements is what this system intended: placement rows still holding
	// capacity. Kept separate from ObservedGuests because the two disagree in
	// both directions and each disagreement means something different. Observed
	// above intended is a foreign guest; intended above observed is a placement
	// whose VM never got made or was deleted behind our back.
	HeldPlacements int

	// ObservationAge is how old the sync is, and Stale whether that exceeds the
	// threshold. Reported even when not stale so an operator can see they are
	// deciding on a five-second-old reading rather than a nine-minute-old one.
	ObservationAge time.Duration
	Stale          bool
}

// Empty reports whether nothing is left on the node, by either measure.
//
// Both must be zero. Either one alone is insufficient and they fail in opposite
// directions: placements alone miss a guest somebody created by hand, and
// observations alone are stale between syncs. This is the same pairing
// Hypervisor.Drift exists to expose, used here as a gate rather than a report.
func (s DrainStatus) Empty() bool { return s.ObservedGuests == 0 && s.HeldPlacements == 0 }

// DrainStateAt derives the drain state of one node at a moment.
//
// heldPlacements is passed in rather than read here for the same reason
// Hypervisor.Fits takes intended capacity: the caller must supply a number it
// counted in the transaction it is about to act in, and a signature satisfiable
// from a stale row would invite exactly the race this is meant to close.
func DrainStateAt(now time.Time, h Hypervisor, heldPlacements int, th DrainThresholds) DrainStatus {
	st := DrainStatus{
		Cordoned:       !h.Schedulable,
		ObservedGuests: h.ObservedGuests,
		HeldPlacements: heldPlacements,
	}

	// Cordon is checked first and answers without any observation at all. It is
	// stored operator intent, fully known, so a node that has not been cordoned
	// is definitively active however old the sync is. Reaching for staleness
	// first would report DrainStateUnknown for a host nobody has asked to drain.
	if h.Schedulable {
		st.State = DrainStateActive
		if h.LastSyncedAt != nil {
			st.ObservationAge = now.Sub(*h.LastSyncedAt)
			st.Stale = st.ObservationAge > th.EffectiveObservationMaxAge()
		} else {
			st.Stale = true
		}
		return st
	}

	// Cordoned. Emptiness is now a question about the world, and answering it
	// needs an observation recent enough to be worth acting on.
	if h.LastSyncedAt == nil {
		st.Stale = true
		st.State = DrainStateUnknown
		return st
	}
	st.ObservationAge = now.Sub(*h.LastSyncedAt)
	st.Stale = st.ObservationAge > th.EffectiveObservationMaxAge()
	if st.Stale {
		st.State = DrainStateUnknown
		return st
	}

	if !st.Empty() {
		st.State = DrainStateDraining
		return st
	}
	st.State = DrainStateDrained
	return st
}

// SchedulablePredicate is the SQL a placement query MUST carry, and it is the
// ONE enforcement point for the cordon.
//
// It belongs in the query rather than in Go, and the reason is atomicity rather
// than tidiness: the placement decision reads candidate hypervisors with
// SELECT ... FOR UPDATE and inserts against that same read, so any condition that
// decides eligibility has to be evaluated inside that read. A filter applied to
// already-loaded rows is evaluated at a different instant from the lock that
// makes the decision safe, and a host cordoned in between is not excluded by it.
//
// waggle enforces it exactly here and nowhere else — `Where("schedulable IS
// TRUE")` alongside the locking clause — and the Terraform-driven placement path
// runs through that query, so cordoning a host is what withdraws it from
// Terraform's next apply. Adding a second check in Go would be a second
// definition of one rule, which is the shape that drifts.
//
// A cordoned host is therefore never a candidate, and CandidateFor below does
// NOT re-check it.
const SchedulablePredicate = "schedulable IS TRUE"

// SwarmScopePredicate is the OTHER half of what a placement query must carry,
// alongside SchedulablePredicate.
//
// A pool names the swarm it places onto, and the query restricts candidates to
// that swarm's members. Like the cordon, it belongs in the query rather than in
// Go, and for the same reason: it decides eligibility, so it has to be evaluated
// inside the same SELECT ... FOR UPDATE as the insert. A hypervisor moved between
// swarms in between would otherwise slip through a filter applied to rows already
// loaded.
//
// Together the placement query reads:
//
//	tx.Clauses(clause.Locking{Strength: "UPDATE"}).
//	   Where("hive_id = ?", pool.HiveID).
//	   Where(tenant.SwarmScopePredicate, pool.SwarmID).
//	   Where(tenant.SchedulablePredicate).
//	   Where(tenant.ProvisionedPredicate).
//
// Note what this excludes: SPARE capacity. A hypervisor in no swarm is not a
// candidate for anything, which is the point — it is available to be assigned,
// not available to be scheduled onto.
const SwarmScopePredicate = "swarm_id = ?"

// ProvisionedPredicate is the THIRD condition a placement query must carry, and
// it is a different axis from the other two rather than more of the same.
//
// Schedulability is an operator's intent about a working machine; this is whether
// the machine works at all yet. Half-onboarded metal — an SSH session that
// installed keys and then failed before the agent came up — is reachable, is in
// the cluster, and is `schedulable` by default, so nothing else in the query
// keeps guests off it.
//
// In the query for the same reason as the other two: eligibility is decided
// inside the SELECT ... FOR UPDATE that the insert is made against, and a machine
// whose onboarding was redriven between the load and the insert must be excluded
// by the read itself, not by a filter over rows already in memory.
//
// A literal rather than a placeholder so the whole rule is visible at every call
// site, matching SchedulablePredicate. The state names are the OnboardX constants
// above; provisioning_state is constrained to them by ck_hypervisors_provisioning_state.
const ProvisionedPredicate = "provisioning_state = 'provisioned'"

// ClusteredPredicate excludes machines that belong to no hive.
//
// Necessary now that hive_id is nullable and a machine record OUTLIVES its
// cluster membership: an onboarded-but-unjoined machine, and a departed one the
// sync has cleared, both sit in the table with zeroed observations. A placement
// query scoped by `hive_id = ?` already excludes them, but a query that forgot
// the hive would otherwise see rows that no provider can be asked to start a
// guest on.
const ClusteredPredicate = "hive_id IS NOT NULL"

// CandidateFor builds a planner candidate for one node:
//
//	remaining = allocatable - observed foreign - intended
//
// floored at zero.
//
// THREE inputs, not two, and the middle one is the correction. Computing
// `allocatable - intended` counted a guest nobody placed through this system
// ZERO times, so a hypervisor carrying twenty hand-created VMs reported its full
// ceiling as free and the oracle kept filling it. The foreign term is subtracted
// via Hypervisor.Bookable so that Fits cannot arrive at a different answer.
//
// Each guest is counted exactly once: ours through `intended` (the ledger, read
// under the same lock as the insert), everyone else's through the foreign
// observation. Subtracting the node's TOTAL observed usage instead would
// double-subtract our own.
//
// Arithmetic only. It does NOT decide eligibility — see SchedulablePredicate for
// why the cordon is a query condition and not a Go one. Passing it a cordoned
// hypervisor produces a candidate, because by the time a row reaches here the
// query that loaded it was required to have excluded cordoned nodes under the
// same lock as the insert; re-checking would only give the wrong impression that
// this is where the rule lives.
//
// The flooring is the part worth having in one place: a node committed beyond its
// allocatable ceiling must not report negative room, because every caller
// comparing with >= would read that as capacity.
func CandidateFor(hive Hive, h Hypervisor, intended Capacity, poolCount int) Candidate {
	b := h.Bookable(hive)
	return Candidate{
		HypervisorID: h.ID,
		Remaining: Capacity{
			CPU:    max(b.CPU-intended.CPU, 0),
			RAMGB:  max(b.RAMGB-intended.RAMGB, 0),
			DiskGB: max(b.DiskGB-intended.DiskGB, 0),
		},
		PoolCount: poolCount,
	}
}
