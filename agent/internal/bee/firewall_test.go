package bee

import (
	"testing"
)

// TestAppliedStartsEmptyAfterARestart.
//
// A restarted agent has applied nothing IN THIS PROCESS, so it reports nothing
// and reconverges. Asserting a revision this process never verified is how a
// host gets reported as compliant on the strength of a claim nobody checked —
// and re-applying an identical ruleset is cheap by comparison.
func TestAppliedStartsEmptyAfterARestart(t *testing.T) {
	t.Parallel()

	var a appliedFirewall
	if rev, digest := a.get(); rev != 0 || digest != "" {
		t.Fatalf("a fresh agent claims revision %d digest %q", rev, digest)
	}
}

func TestAppliedRoundTrips(t *testing.T) {
	t.Parallel()

	var a appliedFirewall
	a.set(7, "sha256:abc")
	if rev, digest := a.get(); rev != 7 || digest != "sha256:abc" {
		t.Fatalf("got %d %q, want 7 sha256:abc", rev, digest)
	}
}

// TestConvergenceIsSkippedWhenNothingChanged.
//
// The important case, and the common one: a fleet whose policy has not moved
// must not reload its firewall on every poll. Reloading is a flush and a reload
// of the host's entire ruleset to arrive at the state it was already in.
//
// Driven through the decision rather than through Apply, because the decision is
// what this test is about — there is no client here to fetch a body with, and
// reaching one would mean the check did not short-circuit.
func TestConvergenceIsSkippedWhenNothingChanged(t *testing.T) {
	t.Parallel()

	b := &Bee{}
	b.firewall.set(4, "sha256:same")

	// A nil client would panic on any fetch, so surviving this call IS the
	// assertion: nothing was fetched and nothing was applied.
	b.converge(t.Context(), 4, "sha256:same")
}

// TestHoldIsSkippedRatherThanTreatedAsAnEmptyRuleset.
//
// Revision zero, or an empty digest, means the machine is in no fleet, its fleet
// has published nothing, or the control plane could not tell which. None of them
// is an empty ruleset, and flushing on any of them strips a host of its
// management access at the moment it is least reachable.
func TestHoldIsSkippedRatherThanTreatedAsAnEmptyRuleset(t *testing.T) {
	t.Parallel()

	b := &Bee{}
	b.firewall.set(4, "sha256:running")

	// Again: a nil client means any fetch panics, so returning is the assertion.
	b.converge(t.Context(), 0, "")
	b.converge(t.Context(), 9, "")

	if rev, digest := b.firewall.get(); rev != 4 || digest != "sha256:running" {
		t.Fatalf("holding changed what the host reports: %d %q", rev, digest)
	}
}

// TestADigestChangeAloneCountsAsDivergence.
//
// A host that loaded the right revision from a corrupted or hand-edited file
// matches on the number and not on the content — and the content is what is
// being enforced. This pins that the comparison is not revision-only; it panics
// on the nil client precisely because it decided to fetch.
func TestADigestChangeAloneCountsAsDivergence(t *testing.T) {
	t.Parallel()

	b := &Bee{}
	b.firewall.set(4, "sha256:what-we-applied")

	defer func() {
		if recover() == nil {
			t.Fatal("a digest mismatch at the same revision did not trigger a fetch, so a host " +
				"running a hand-edited ruleset would never be corrected")
		}
	}()
	b.converge(t.Context(), 4, "sha256:what-it-should-be")
}
