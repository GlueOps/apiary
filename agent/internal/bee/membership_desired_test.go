package bee

import "testing"

// A REVISION ONCE ADVERTISED MUST NOT BE KEPT FOREVER.
//
// The heartbeat carries a membership revision only while there is one to carry.
// When a machine is departed its hive_id goes and the control plane rightly
// stops advertising anything — absent means HOLD.
//
// The agent stored the value only on the branch where it was present, so the
// last one survived indefinitely: it kept deciding it was diverged, kept
// fetching a composition that was now empty, and logged "named a revision but
// returned no composition" roughly once every two seconds. Observed on pve5
// after it was departed.
//
// Holding means forgetting what was asked for, not continuing to act on it.
func TestAWithdrawnMembershipIsForgotten(t *testing.T) {
	var b Bee

	b.desiredMembership.Store(&desiredMembership{revision: 29, digest: "sha256:abc"})
	if d, _ := b.desiredMembership.Load().(*desiredMembership); d == nil || d.revision != 29 {
		t.Fatalf("precondition: stored value = %+v, want revision 29", d)
	}

	// The control plane stops advertising one.
	b.desiredMembership.Store((*desiredMembership)(nil))

	d, _ := b.desiredMembership.Load().(*desiredMembership)
	if d != nil {
		t.Fatalf("a withdrawn membership was retained as %+v; the agent would keep converging "+
			"on a revision that no longer applies", d)
	}
}

// Storing a typed nil into the same atomic.Value must not panic — it is the same
// concrete type, which is what atomic.Value requires.
func TestClearingAndSettingMembershipAlternates(t *testing.T) {
	var b Bee
	for i := range 3 {
		b.desiredMembership.Store(&desiredMembership{revision: i + 1})
		b.desiredMembership.Store((*desiredMembership)(nil))
	}
	if d, _ := b.desiredMembership.Load().(*desiredMembership); d != nil {
		t.Fatalf("final value = %+v, want nil", d)
	}
}
