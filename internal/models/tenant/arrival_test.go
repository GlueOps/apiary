package tenant

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func arrivalCodes(rs []ArrivalRefusal) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Code)
	}
	return out
}

func hasArrivalCode(rs []ArrivalRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

func TestCheckArrivalAllowsAFleetlessEmptyHost(t *testing.T) {
	req := ArrivalRequest{
		TargetHive:      Hive{ID: uuid.New(), Name: "dc2"},
		CurrentHiveID:   nil,
		ObservedGuests:  0,
		TargetNodeCount: 1,
	}
	if rs := CheckArrival(req); len(rs) != 0 {
		t.Fatalf("a fleetless, guestless host joining a live hive must be allowed; refused with %v",
			arrivalCodes(rs))
	}
}

func TestCheckArrivalRefusals(t *testing.T) {
	target := Hive{ID: uuid.New(), Name: "dc2"}
	base := func() ArrivalRequest {
		return ArrivalRequest{TargetHive: target, TargetNodeCount: 2}
	}

	t.Run("a host still in another hive is refused", func(t *testing.T) {
		other := uuid.New()
		req := base()
		req.CurrentHiveID = &other
		rs := CheckArrival(req)
		if !hasArrivalCode(rs, RefuseAlreadyInHive) {
			t.Fatalf("want already_in_hive, got %v", arrivalCodes(rs))
		}
	})

	t.Run("a host already in the TARGET hive gets its own code", func(t *testing.T) {
		// Distinct because the advice differs: "depart first" is wrong here, the
		// operator has simply asked twice.
		req := base()
		req.CurrentHiveID = &target.ID
		rs := CheckArrival(req)
		if !hasArrivalCode(rs, RefuseSameHive) {
			t.Fatalf("want same_hive, got %v", arrivalCodes(rs))
		}
		if hasArrivalCode(rs, RefuseAlreadyInHive) {
			t.Error("same_hive and already_in_hive must not both fire — they give opposite advice")
		}
	})

	t.Run("exactly one guest blocks the join", func(t *testing.T) {
		req := base()
		req.ObservedGuests = 1
		rs := CheckArrival(req)
		if !hasArrivalCode(rs, RefuseArrivalGuests) {
			t.Fatalf("want guests_present, got %v", arrivalCodes(rs))
		}
		for _, r := range rs {
			if r.Code != RefuseArrivalGuests {
				continue
			}
			// Checked across all three fields. That stopped guests count is the
			// non-obvious half — pvecm counts them and the operator is looking for
			// running VMs — and which field carries it is a presentation choice.
			// Losing it altogether is not, and that is what shortening a message for
			// width does first.
			whole := r.Summary + " " + r.Detail + " " + r.Remedy
			if !strings.Contains(strings.ToLower(whole), "stopped") {
				t.Error("nothing in this refusal says stopped guests count — pvecm counts them " +
					"and the operator will be looking for running VMs")
			}
			if r.Summary == "" {
				t.Error("no summary: the card groups refusals by code and would have to show " +
					"one host's guest count as if it applied to all of them")
			}
			if r.Remedy == "" {
				t.Error("no remedy: the operator is told what is wrong and not what to do")
			}
		}
	})

	t.Run("an empty target hive is ALLOWED: that is how a cluster is formed", func(t *testing.T) {
		// This used to refuse with target_hive_empty, and a separate bootstrap
		// operation existed solely to create the first node. They were the same
		// request all along, distinguished only by the control plane choosing a
		// verb in advance.
		//
		// Under convergence the host derives it: no other member is present, so
		// it creates. Refusing here made arrival depend on a precondition only
		// another operation could satisfy.
		req := base()
		req.TargetNodeCount = 0
		if rs := CheckArrival(req); len(rs) != 0 {
			t.Fatalf("arriving at an empty hive must be allowed; refused with %v", arrivalCodes(rs))
		}
	})

	t.Run("a one-node target is allowed", func(t *testing.T) {
		// Deliberately NOT gated on a minimum. Joining only grows a cluster, and
		// requiring three would make a three-node hive unreachable — you cannot
		// reach three without passing through two.
		req := base()
		req.TargetNodeCount = 1
		if rs := CheckArrival(req); len(rs) != 0 {
			t.Fatalf("joining a one-node hive must be allowed; refused with %v", arrivalCodes(rs))
		}
	})

	t.Run("every reason is returned, not the first", func(t *testing.T) {
		other := uuid.New()
		req := base()
		req.CurrentHiveID = &other
		req.ObservedGuests = 3
		req.TargetNodeCount = 0
		rs := CheckArrival(req)
		if len(rs) != 2 {
			t.Fatalf("want both remaining refusals, got %v", arrivalCodes(rs))
		}
	})
}

func TestArrivalRefusalsError(t *testing.T) {
	if err := ArrivalRefusalsError(nil); err != nil {
		t.Fatalf("no refusals must be no error, got %v", err)
	}
	err := ArrivalRefusalsError([]ArrivalRefusal{
		{Code: RefuseAlreadyInHive, Detail: "a"},
		{Code: RefuseArrivalGuests, Detail: "b"},
	})
	if !strings.Contains(err.Error(), RefuseAlreadyInHive) || !strings.Contains(err.Error(), RefuseArrivalGuests) {
		t.Fatalf("every refusal must appear, got %q", err.Error())
	}
}
