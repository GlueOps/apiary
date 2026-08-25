package tenant

import (
	"testing"
)

func labelled(pairs ...[2]string) *FirewallSet {
	s := &FirewallSet{Name: "mgmt", Family: FamilyIPv4}
	for _, p := range pairs {
		s.Elements = append(s.Elements, FirewallSetElement{Value: p[0], Label: p[1]})
	}
	return s
}

// A host is reachable by different addresses depending on the path taken to it,
// so both belong in the set. The label is what makes them one thing to remove.
func TestElementsGroupByLabel(t *testing.T) {
	s := labelled(
		[2]string{"142.249.164.20", "hv20"},
		[2]string{"10.62.0.20", "hv20"},
		[2]string{"194.36.65.187", ""},
	)
	groups := s.ElementsByLabel()

	if len(groups["hv20"]) != 2 {
		t.Fatalf("the pair did not group: %v", groups["hv20"])
	}
	if len(groups) != 2 {
		t.Fatalf("expected the pair plus one standalone, got %d groups", len(groups))
	}
}

// Two unlabelled addresses must not collide into one group — a standalone
// address being swept into somebody else's group would make "remove that
// machine" delete an unrelated entry.
func TestUnlabelledElementsDoNotCollide(t *testing.T) {
	s := labelled([2]string{"10.0.0.1", ""}, [2]string{"10.0.0.2", ""})
	if got := len(s.ElementsByLabel()); got != 2 {
		t.Fatalf("two unlabelled addresses formed %d groups, want 2", got)
	}
}

// THE check the label exists for.
//
// A machine whose private address was removed and whose public one was not looks
// like a healthy set until somebody reads every entry — and the surviving half
// is on a public range that eventually gets reallocated to a stranger, who
// inherits SSH access nobody granted.
func TestPartialLabelsFindsAHalfRemovedMachine(t *testing.T) {
	s := labelled(
		[2]string{"142.249.164.20", "hv20"},
		[2]string{"10.62.0.20", "hv20"},
		[2]string{"142.249.164.21", "hv21"},
		[2]string{"10.62.0.21", "hv21"},
	)
	// hv20's private half was removed; its public half survives.
	present := map[string]bool{
		"142.249.164.20": true,
		"142.249.164.21": true,
		"10.62.0.21":     true,
	}

	got := s.PartialLabels(present)
	if len(got) != 1 || got[0] != "hv20" {
		t.Fatalf("got %v, want [hv20] — the half-removed machine", got)
	}
}

// A machine fully present and one fully removed are both healthy states. Only
// the in-between is a finding, or the warning fires constantly and gets ignored.
func TestPartialLabelsIgnoresFullyPresentAndFullyAbsent(t *testing.T) {
	s := labelled(
		[2]string{"142.249.164.20", "hv20"},
		[2]string{"10.62.0.20", "hv20"},
		[2]string{"142.249.164.21", "hv21"},
		[2]string{"10.62.0.21", "hv21"},
	)
	present := map[string]bool{"142.249.164.20": true, "10.62.0.20": true}

	if got := s.PartialLabels(present); len(got) != 0 {
		t.Fatalf("got %v, want none — hv20 is whole and hv21 is entirely gone", got)
	}
}

// Pairing is NOT guaranteed — a management address may have no private
// counterpart in a given fleet — so the standalone case is ordinary, not an
// edge. It can never be partial, whether present or absent, because a group of
// one is either whole or entirely gone.
func TestStandaloneAddressesAreNeverPartial(t *testing.T) {
	s := labelled(
		[2]string{"194.36.65.187", ""},
		[2]string{"142.249.164.20", "hv20"},
		[2]string{"10.62.0.20", "hv20"},
	)

	cases := []struct {
		name    string
		present map[string]bool
	}{
		{"standalone absent", map[string]bool{"142.249.164.20": true, "10.62.0.20": true}},
		{"standalone present", map[string]bool{
			"194.36.65.187": true, "142.249.164.20": true, "10.62.0.20": true,
		}},
		{"nothing present", map[string]bool{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, l := range s.PartialLabels(c.present) {
				if l != "hv20" {
					t.Fatalf("reported %q as partial; only a labelled group can be", l)
				}
			}
		})
	}

	// And the labelled pair IS reported when only half survives, so the check
	// above is not passing merely because nothing is ever reported.
	half := map[string]bool{"194.36.65.187": true, "142.249.164.20": true}
	if got := s.PartialLabels(half); len(got) != 1 || got[0] != "hv20" {
		t.Fatalf("got %v, want [hv20]", got)
	}
}
