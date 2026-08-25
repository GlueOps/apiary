package fleet

import (
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// A machine between fleets has a nil HiveID, and rendering it must not panic.
//
// uuid.UUID.String() has a VALUE receiver, so calling it through a nil
// *uuid.UUID dereferences nil. The listing endpoint did exactly that, which
// means it crashed on precisely the hosts an operator opens it to find: the ones
// mid-migration, which is the state the whole tenant-scoped fleet redesign
// exists to represent. A fleet view that 500s whenever a host is between fleets
// is worse than one that never showed them.
//
// The column MOVED — from the bee to the machine record — and the trap moved
// with it, so this covers both renderers: hiveIDOf for the agent listing, which
// now reads the machine behind the agent, and hiveIDOfMachine for the hypervisor
// listing, where nullable hive membership is new.
func TestMachineViewRendersAHostWithNoFleet(t *testing.T) {
	hiveID := uuid.New()

	cases := []struct {
		name    string
		machine tenant.Hypervisor
		want    string
	}{
		{
			name:    "between fleets",
			machine: tenant.Hypervisor{ID: uuid.New(), Address: "hv5"},
			want:    "",
		},
		{
			name:    "in a fleet",
			machine: tenant.Hypervisor{ID: uuid.New(), Address: "hv1", HiveID: &hiveID},
			want:    hiveID.String(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The same conversions the handlers perform. Kept as helpers so the
			// guard is exercised rather than reimplemented by the test.
			if got := hiveIDOf(&c.machine); got != c.want {
				t.Fatalf("agent listing rendered the hive id as %q, want %q", got, c.want)
			}
			if got := hiveIDOfMachine(c.machine); got != c.want {
				t.Fatalf("machine listing rendered the hive id as %q, want %q", got, c.want)
			}
		})
	}
}

// A machine with no node name must render as absent rather than panicking or
// inventing one. There is no such name until it joins a cluster, so this is the
// normal state for everything the fleetless list holds.
func TestHypervisorViewRendersAMachineWithNoNodeName(t *testing.T) {
	if got := nodeNameOf(tenant.Hypervisor{ID: uuid.New(), Address: "hv5"}); got != "" {
		t.Fatalf("an unnamed machine rendered as %q", got)
	}
	name := "hv5"
	if got := nodeNameOf(tenant.Hypervisor{ID: uuid.New(), Name: &name}); got != name {
		t.Fatalf("node name rendered as %q, want %q", got, name)
	}
}

// The node count handed to the departure gate must be ZERO for a machine in no
// hive, not whatever the map happens to hold under the nil uuid.
//
// The gate reads a zero count as not_a_member, so this is not merely avoiding a
// nil dereference: indexing with a zero uuid would hand it a count belonging to
// no hive at all, and the count that lands there is "how many machines are in no
// cluster" — which would then be compared against the hive's minimum node count.
func TestNodeCountForAMachineInNoHiveIsZero(t *testing.T) {
	hiveID := uuid.New()
	counts := map[uuid.UUID]int{hiveID: 4, uuid.Nil: 9}

	if got := nodeCountFor(counts, tenant.Hypervisor{}); got != 0 {
		t.Fatalf("a machine in no hive was given a node count of %d; the departure gate reads "+
			"a non-zero count as membership and would let a worker go and dismantle a cluster "+
			"this machine is not in", got)
	}
	if got := nodeCountFor(counts, tenant.Hypervisor{HiveID: &hiveID}); got != 4 {
		t.Fatalf("node count = %d, want 4", got)
	}
}

// An empty hive id must serialise as ABSENT, not as the nil uuid. A client that
// receives 00000000-0000-0000-0000-000000000000 will go looking for a fleet with
// that id and find nothing, which reads as data corruption rather than as "this
// host is between fleets".
func TestViewsOmitRatherThanZeroAnAbsentFleet(t *testing.T) {
	b := BeeView{ID: uuid.NewString(), Address: "hv5"}
	if b.HiveID == uuid.Nil.String() {
		t.Fatal("an absent fleet rendered as the nil uuid on the agent view")
	}
	if b.HiveID != "" {
		t.Fatalf("an absent fleet rendered as %q on the agent view", b.HiveID)
	}

	h := HypervisorView{ID: uuid.NewString(), Address: "hv5"}
	if h.HiveID == uuid.Nil.String() {
		t.Fatal("an absent fleet rendered as the nil uuid on the machine view")
	}
	if h.HiveID != "" {
		t.Fatalf("an absent fleet rendered as %q on the machine view", h.HiveID)
	}
}
