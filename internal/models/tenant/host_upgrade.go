package tenant

import (
	"fmt"
	"time"
)

// Deciding whether a host may be upgraded, in the control plane rather than on
// the agent.
//
// The agent used to decide this alone, by asking the Proxmox cluster API — which
// is why a dist-upgrade needed an API token for a job that is entirely local:
// apt-get, then a reboot. Nothing about the work is orchestrated through
// Proxmox; the token existed only to answer three questions before rebooting,
// and the control plane already knows all three.
//
// Moving it here also means ONE gate model rather than two. The departure gate
// already refuses on a stale observation rather than guessing, already reads
// cordoning as intent, and already reports refusals as codes with remedies. This
// is the same shape asking a different verb.

// Reboot policies a caller may ask for.
const (
	// RebootIfNeeded reboots only when the upgrade asks for one — a new kernel,
	// usually. The DEFAULT, and the reason a reboot has to be treated as possible
	// unless it is explicitly refused: an operator who never chose a policy has
	// still asked for something that may reboot.
	RebootIfNeeded = "if-needed"
	RebootAlways   = "always"
	// RebootNever upgrades packages and stops. The host keeps running the old
	// kernel, and nothing it is running is interrupted.
	RebootNever = "never"
)

// Upgrade refusal codes.
const (
	RefuseUpgradeNotCordoned   = "upgrade_not_cordoned"
	RefuseUpgradeGuestsPresent = "upgrade_guests_present"
	RefuseUpgradeQuorum        = "upgrade_quorum_at_risk"
	RefuseUpgradeStale         = "upgrade_observation_stale"
)

// UpgradeRequest is everything the gate decides on. Like the departure gate it
// reads nothing else — no clock, no database — so the policy is inspectable and
// the same inputs always give the same verdict.
type UpgradeRequest struct {
	Hypervisor Hypervisor
	// HiveNodeCount is the number of nodes in this host's cluster, and ZERO for a
	// host in none. Zero is a real answer, not missing data.
	HiveNodeCount int

	// Reboot is the requested policy. Empty means the default, which MAY reboot.
	Reboot string
	// AllowGuestDowntime is the operator saying, explicitly, that the guests on
	// this host may go down with it.
	AllowGuestDowntime bool

	// ObservationAge is how old the capacity observation is, and MaxAge is how
	// old it may be before the gate refuses to draw a conclusion from it.
	ObservationAge time.Duration
	MaxAge         time.Duration
}

// WillReboot reports whether this request might interrupt what the host is
// running.
//
// "Might", not "will": if-needed is decided on the host, after the packages are
// fetched, by whether a new kernel arrived. The gate cannot know that and must
// not assume the favourable answer — an operator who never chose a policy has
// still asked for something that can reboot.
func (r UpgradeRequest) WillReboot() bool {
	return r.Reboot != RebootNever
}

// CheckHostUpgrade returns every reason this host may not be upgraded.
//
// EVERY reason, not the first. An operator who fixes one and is then told about
// the next has been made to do the work twice, which is the same argument the
// departure gate makes.
func CheckHostUpgrade(req UpgradeRequest) []DepartureRefusal {
	var out []DepartureRefusal

	// NOTHING IS AT RISK WHEN NOTHING REBOOTS.
	//
	// This is the case that was refused outright before, and wrongly: the agent
	// ran a REBOOT preflight unconditionally, so `reboot: never` with guests
	// running was blocked by a check for a reboot that never happened. Upgrading
	// packages does not interrupt a running guest, cost quorum, or care whether
	// the host is cordoned.
	if !req.WillReboot() {
		return out
	}

	if req.MaxAge > 0 && req.ObservationAge > req.MaxAge {
		out = append(out, DepartureRefusal{
			Code:    RefuseUpgradeStale,
			Summary: "the last capacity sync is too stale to tell what is running here",
			Detail: fmt.Sprintf("the last capacity sync for this node is %s old",
				req.ObservationAge.Truncate(time.Second)),
			Remedy: "Run a sync and retry. Refusing rather than assuming: a host that looks " +
				"empty in a stale observation may be running something.",
		})
	}

	if req.Hypervisor.Schedulable {
		// A reboot is a window in which the host is gone. Placement landing in that
		// window is a guest created on a machine that is about to disappear, and
		// the cordon is how an operator says the window is open.
		out = append(out, DepartureRefusal{
			Code:    RefuseUpgradeNotCordoned,
			Summary: "the host still accepts placements",
			Detail:  "this host still accepts placements and the upgrade may reboot it",
			Remedy: "Cordon it first, so nothing new lands on a machine that is about to " +
				"restart.",
		})
	}

	if req.Hypervisor.ObservedGuests > 0 && !req.AllowGuestDowntime {
		out = append(out, DepartureRefusal{
			Code:    RefuseUpgradeGuestsPresent,
			Summary: "guests are still running on the host and a reboot would take them down",
			Detail: fmt.Sprintf("the provider reports %d guest(s) on this node",
				req.Hypervisor.ObservedGuests),
			Remedy: "Migrate them, or allow guest downtime deliberately. Upgrading without a " +
				"reboot is the other way out: the packages are installed and nothing is " +
				"interrupted, at the cost of running the old kernel until the next restart.",
		})
	}

	// QUORUM, NOT THE DEPARTURE MINIMUM. A reboot is temporary — the node comes
	// back — so the question is whether corosync keeps quorum while it is away,
	// not whether the cluster has enough nodes to run permanently without it.
	// Conflating the two is why a small cluster looks un-upgradeable: a three-node
	// cluster survives losing one and a two-node cluster does not, whatever the
	// departure minimum is set to.
	//
	// Checked only where there are SURVIVORS to strand. The point of the rule is
	// that the nodes left behind go read-only with their guests unmanageable —
	// so with zero nodes (no cluster) or one (nobody left), it has nothing to
	// protect and refusing would be arithmetic for its own sake. A single-node
	// cluster does technically lose quorum while it is down; there is no one to
	// notice.
	if req.HiveNodeCount > 1 && !QuorumSurvivesLosing(req.HiveNodeCount, 1) {
		out = append(out, DepartureRefusal{
			Code:    RefuseUpgradeQuorum,
			Summary: "rebooting this node would cost the cluster quorum",
			Detail: fmt.Sprintf("the cluster has %d node(s), and losing one leaves it without "+
				"quorum", req.HiveNodeCount),
			Remedy: "Add a node, or upgrade without a reboot. A cluster that loses quorum goes " +
				"read-only and its guests become unmanageable until it returns.",
		})
	}

	return out
}

// QuorumSurvivesLosing reports whether a cluster of n voting members still has
// quorum with `lost` of them gone.
//
// Corosync's rule: quorum is floor(n/2)+1 of the ORIGINAL member count, so the
// question is whether the survivors still reach it. Two nodes cannot survive
// losing one — quorum is 2 — which is the arithmetic behind every warning about
// two-node clusters.
func QuorumSurvivesLosing(members, lost int) bool {
	if members <= 0 {
		return false
	}
	return members-lost >= members/2+1
}
