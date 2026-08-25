package tenant

import (
	"strings"
	"testing"
)

// TestClaimAndRedriveAreDifferentSets is the reason these are two constants.
//
// They were nearly merged: both mention the same lease and the same states, and
// a single predicate reads like a simplification. It is not. Redrive must admit
// terminal rows — retrying a failure is its whole purpose — and the ticker must
// not, or every tick would re-onboard the entire working fleet. Re-enrolling a
// live host mints a credential the running agent never re-reads, so it 401s on
// every heartbeat afterwards and never recovers.
func TestClaimAndRedriveAreDifferentSets(t *testing.T) {
	t.Parallel()

	if strings.Contains(ClaimablePredicate, OnboardProvisioned) {
		t.Fatal("the ticker's predicate admits provisioned machines: it would re-onboard the " +
			"working fleet on every tick, breaking each agent's credential as it went")
	}
	if strings.Contains(ClaimablePredicate, OnboardFailed) {
		t.Fatal("the ticker's predicate admits failed machines: retrying past the attempt cap " +
			"is an operator's decision, made through redrive")
	}
	if !strings.Contains(ClaimablePredicate, OnboardPending) {
		t.Fatalf("the ticker's predicate does not mention %q, so nothing would ever be claimed",
			OnboardPending)
	}

	if !strings.Contains(RedrivablePredicate, OnboardFailed) {
		t.Fatal("redrive does not admit failed machines, which is the one case it exists for")
	}
	if strings.Contains(RedrivablePredicate, "'pending'") {
		t.Fatal("redrive admits pending machines: resetting a row already queued looks " +
			"identical to a successful redrive while changing nothing")
	}
}

// TestBothPredicatesTreatANullLeaseAsExpired. The claim stamps state and lease
// in one statement, so `provisioning` with a NULL lease cannot arise from
// correct code — and reading it as held would strand the row with SQL as the
// only way out, which is the predicament redrive exists to end.
func TestBothPredicatesTreatANullLeaseAsExpired(t *testing.T) {
	t.Parallel()

	for name, p := range map[string]string{
		"claim":   ClaimablePredicate,
		"redrive": RedrivablePredicate,
	} {
		if !strings.Contains(p, "provisioning_lease_expires_at IS NULL") {
			t.Fatalf("%s does not treat a NULL lease as expired, so a row that reached that "+
				"state is stuck permanently", name)
		}
		if !strings.Contains(p, "now()") {
			t.Fatalf("%s does not compare against the database clock; the lease is written by "+
				"a worker and read by an API replica, and a Go timestamp is not a clock they share",
				name)
		}
	}
}

// TestTheLeaseOutlastsAnOnboard. Onboarding waits for a heartbeat after
// installing the agent, so the slow case is minutes — a shorter lease would let
// a second worker claim a host the first is still inside.
func TestTheLeaseOutlastsAnOnboard(t *testing.T) {
	t.Parallel()
	if LeaseExtension.Minutes() < 5 {
		t.Fatalf("LeaseExtension is %s: shorter than an onboard's heartbeat wait, so a second "+
			"worker would claim a host the first is still installing keys on", LeaseExtension)
	}
}
