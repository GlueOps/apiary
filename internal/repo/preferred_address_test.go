package repo

import "testing"

// TestPreferredAddressNeverLetsAnAgentOverwriteADeclaredOne.
//
// THE BUG THIS FIXES. The address used to be refreshed from the agent's own
// hostname on every enrolment, on the reasoning that the agent which just spoke
// to us is the most current source. An agent knows what it CALLS ITSELF and has
// no idea how anybody else reaches it: a Proxmox node calls itself "pve2", which
// resolves on the cluster network and nowhere else.
//
// The result was that every departure and reprovision failed with
// `lookup pve2: no such host`, retried on a backoff, and reported nothing — so a
// cordoned, drained node simply would not leave, with no reason on screen.
func TestPreferredAddressNeverLetsAnAgentOverwriteADeclaredOne(t *testing.T) {
	t.Parallel()

	if got := preferredAddress("10.10.10.3", "", "pve2"); got != "10.10.10.3" {
		t.Fatalf("got %q: a hostname overwrote a stored address, which is the bug — "+
			"the control plane then reaches for a name only the cluster network resolves", got)
	}
}

// TestAMachineThatMovedGetsItsNewAddress.
//
// The other half, and the reason the peer address outranks the stored one: a
// re-enrolment from somewhere new is how a machine that moved says so, and
// pinning it to the old address would leave the control plane reaching for a
// host that is no longer there.
func TestAMachineThatMovedGetsItsNewAddress(t *testing.T) {
	t.Parallel()

	if got := preferredAddress("10.10.10.3", "10.10.20.9", "pve2"); got != "10.10.20.9" {
		t.Fatalf("got %q, want the address the request actually came from", got)
	}
}

// TestThePeerAddressBeatsTheHostname, because it is reachable by demonstration
// rather than by assumption — the packet came from it.
func TestThePeerAddressBeatsTheHostname(t *testing.T) {
	t.Parallel()

	if got := preferredAddress("", "10.10.10.3", "pve2"); got != "10.10.10.3" {
		t.Fatalf("got %q, want the address the request actually came from", got)
	}
}

// TestTheHostnameIsTheLastResort, for a deployment where it genuinely is the
// name the control plane uses.
func TestTheHostnameIsTheLastResort(t *testing.T) {
	t.Parallel()

	if got := preferredAddress("", "", "pve2.example.com"); got != "pve2.example.com" {
		t.Fatalf("got %q, want the hostname when nothing better is known", got)
	}
	if got := preferredAddress("", "   ", "pve2"); got != "pve2" {
		t.Fatalf("got %q: whitespace is not a peer address", got)
	}
}

// TestAnExistingAddressSurvivesEvenWhenNothingElseIsKnown, so a re-enrolment
// that carries no useful candidate does not blank it.
func TestAnExistingAddressSurvivesEvenWhenNothingElseIsKnown(t *testing.T) {
	t.Parallel()

	if got := preferredAddress("10.10.10.3", "", ""); got != "10.10.10.3" {
		t.Fatalf("got %q, want the address left alone", got)
	}
}
