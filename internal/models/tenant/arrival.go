package tenant

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Arrival refusal codes. Stable strings, same contract as the departure ones.
const (
	RefuseAlreadyInHive     = "already_in_hive"
	RefuseArrivalGuests     = "guests_present"
	RefuseTargetHiveEmpty   = "target_hive_empty"
	RefuseTargetUnreachable = "target_unreachable"
	RefuseSameHive          = "same_hive"
)

// ArrivalRefusal is one reason a host may not join a hive.
type ArrivalRefusal struct {
	// Code identifies the rule. It is the stable half — like an HTTP status, it
	// is what a client switches on and what an operator learns to recognise.
	Code string

	// Summary states the rule GENERICALLY, with no numbers and no host in it.
	//
	// It exists because the same rule refuses many hosts with different figures.
	// Grouping refusals by code to avoid repeating them means picking one of
	// those details to display, and "the provider reports 2 guest(s)" shown for a
	// group that also contains a host with five is not a shorter message — it is
	// a wrong one. The summary is the half that is true of every host the rule
	// refused, so it can be shown once.
	Summary string

	// Detail states what is wrong ON THIS HOST, and names the offending number.
	// "3 guests are still on this node" is actionable; "not drained" sends
	// somebody to the provider's UI to work out why.
	//
	// Per host, so it cannot be grouped — see Summary.
	Detail string

	// Remedy states WHAT TO DO, separately.
	//
	// Separate because it is a different kind of sentence with a different
	// audience and a different place on screen. Nobody needs "401 means send
	// credentials" inline with every 401; they need it once, where they are
	// looking for what to do next. Folding the two together makes the common case
	// — an operator who already knows this rule — read the same instruction every
	// time, and makes the message too long to render beside the thing it is about.
	//
	// It is also the half that is easy to lose. Trimming a message for width
	// takes the remedy first, because the statement of the problem feels like the
	// important part — and then the non-obvious step goes with it. Giving it its
	// own field means shortening the message cannot silently delete it.
	Remedy string
}

func (r ArrivalRefusal) Error() string { return r.Code + ": " + r.Detail }

// ArrivalRequest is everything the arrival gate decides on.
type ArrivalRequest struct {
	// TargetHive is the hive being joined.
	TargetHive Hive

	// CurrentHiveID is what the joining host currently reports. Nil is the
	// expected state — a host arrives from nowhere, having departed first.
	CurrentHiveID *uuid.UUID

	// ObservedGuests on the JOINING host, stopped ones included. A guest here is
	// a hard refusal rather than a warning: `pvecm add` rejects the node outright
	// with "this host already contains virtual guests", and it counts stopped
	// guests too.
	ObservedGuests int

	// TargetNodeCount is how many nodes the target hive has. Zero means there is
	// nothing to join to.
	TargetNodeCount int
}

// CheckArrival returns every reason the host may not join, or nil.
//
// Every reason, for the same purpose as CheckDeparture: an operator fixing
// preconditions one job failure at a time is spending a worker run against a
// production cluster per round trip.
//
// Note what is NOT gated here. There is no minimum size for the TARGET hive:
// joining only ever grows a cluster, and requiring three nodes to join would
// make a three-node hive unreachable — you cannot get to three without passing
// through two. The minimum belongs on departure, which is where shrinking below
// quorum actually does damage.
func CheckArrival(req ArrivalRequest) []ArrivalRefusal {
	var out []ArrivalRefusal

	if req.CurrentHiveID != nil {
		if *req.CurrentHiveID == req.TargetHive.ID {
			// Worth its own code: this is an operator repeating themselves, not a
			// migration that skipped a step, and telling them to depart first would
			// be actively wrong advice.
			out = append(out, ArrivalRefusal{
				Code:    RefuseSameHive,
				Summary: "the host is already a member of the cluster it is being joined to",
				Detail:  fmt.Sprintf("this host is already a member of hive %q", req.TargetHive.Name),
				Remedy:  "Nothing to do — it is already where you are sending it.",
			})
		} else {
			out = append(out, ArrivalRefusal{
				Code:    RefuseAlreadyInHive,
				Summary: "the host still reports membership of another cluster",
				Detail:  "this host still reports membership of another cluster",
				Remedy: "Depart it first. A migration is a departure and then an arrival, and " +
					"the fleetless state between them is a real state — joining directly would " +
					"leave two clusters believing they own the same node.",
			})
		}
	}

	if req.ObservedGuests > 0 {
		out = append(out, ArrivalRefusal{
			Code:    RefuseArrivalGuests,
			Summary: "the joining host still has guests, stopped ones included",
			Detail: fmt.Sprintf("the joining host has %d guest(s), stopped ones included",
				req.ObservedGuests),
			Remedy: "Move or delete them before joining. `pvecm add` refuses a node with any " +
				"guests at all, and a stopped guest counts.",
		})
	}

	// AN EMPTY TARGET IS NO LONGER A REFUSAL.
	//
	// This used to refuse with target_hive_empty — "create the cluster on a first
	// node before adding a second" — and a whole separate bootstrap operation
	// existed solely to create that first node. The two were the same request all
	// along, distinguished only by the control plane having to choose a verb in
	// advance.
	//
	// Under convergence nothing chooses. The host observes what corosync reports
	// and derives it: to become a member of a cluster with no other members is
	// create, of one that has members is join. So arriving at an empty hive is
	// exactly how a cluster is formed, and refusing it was a precondition only
	// another operation could satisfy.
	//
	// RefuseTargetHiveEmpty is kept as a constant because it appears in stored
	// audit rows, which must stay readable.

	return out
}

// ArrivalRefusalsError renders a refusal list as one error.
func ArrivalRefusalsError(rs []ArrivalRefusal) error {
	if len(rs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		parts = append(parts, r.Code+" ("+r.Detail+")")
	}
	return fmt.Errorf("arrival refused: %s", strings.Join(parts, "; "))
}
