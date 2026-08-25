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
	Code   string
	Detail string
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
				Code:   RefuseSameHive,
				Detail: fmt.Sprintf("this host is already a member of hive %q", req.TargetHive.Name),
			})
		} else {
			out = append(out, ArrivalRefusal{
				Code: RefuseAlreadyInHive,
				Detail: fmt.Sprintf(
					"this host still reports membership of another hive; depart it first. A " +
						"migration is a departure and then an arrival, and the fleetless state " +
						"between them is a real state — joining directly would leave two clusters " +
						"believing they own the same node"),
			})
		}
	}

	if req.ObservedGuests > 0 {
		out = append(out, ArrivalRefusal{
			Code: RefuseArrivalGuests,
			Detail: fmt.Sprintf(
				"the joining host has %d guest(s), STOPPED ONES INCLUDED — `pvecm add` refuses a "+
					"node with any guests at all, and a stopped guest counts. Move or delete them "+
					"before joining",
				req.ObservedGuests),
		})
	}

	if req.TargetNodeCount < 1 {
		out = append(out, ArrivalRefusal{
			Code: RefuseTargetHiveEmpty,
			Detail: fmt.Sprintf(
				"hive %q has no nodes to join to; create the cluster on a first node before "+
					"adding a second", req.TargetHive.Name),
		})
	}

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
