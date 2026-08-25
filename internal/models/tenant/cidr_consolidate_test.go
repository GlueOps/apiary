package tenant

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"testing"
)

// expand enumerates every address a prefix matches. Only sane for v4 test data,
// which is the point: the round-trip below is what proves consolidation exact,
// and a proof you cannot run is not one.
func expand(t *testing.T, prefixes []string) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, s := range prefixes {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			t.Fatalf("consolidation emitted an unparseable prefix %q: %v", s, err)
		}
		if !p.Addr().Is4() {
			t.Fatalf("expand only handles v4, got %q", s)
		}
		if p.Bits() < 16 {
			t.Fatalf("consolidation emitted %q — wider than a /16, which for a list of "+
				"individual hosts means it has grown the permitted set", s)
		}
		for a := p.Addr(); p.Contains(a); a = a.Next() {
			out[a.String()] = struct{}{}
		}
	}
	return out
}

// THE property. Consolidation must match exactly the addresses it was given —
// not one more.
//
// This is the question a CIDR in a firewall rule invites and the answer is not
// obvious: a prefix here is a MATCH PATTERN, not a network declaration. Emitting
// 10.62.0.16/28 says nothing about netmasks or about the /24 those hosts
// actually live on; it says "these sixteen addresses". Growing the set by even
// one address would mean this function silently granting access nobody wrote,
// which is the failure that would matter.
func TestConsolidationMatchesExactlyTheAddressesItWasGiven(t *testing.T) {
	var input []string
	for i := 2; i <= 38; i++ {
		input = append(input, fmt.Sprintf("142.249.164.%d", i), fmt.Sprintf("10.62.0.%d", i))
	}
	input = append(input, "194.36.65.187", "3.6.130.16", "3.79.224.50", "34.208.196.7",
		"13.50.141.55", "46.224.82.187", "157.180.6.14", "157.180.6.73", "95.217.36.236",
		// duplicated in the source, as real whitelists are
		"142.249.164.2", "142.249.164.3")

	got := ConsolidateCIDRs(input)

	want := map[string]struct{}{}
	for _, a := range input {
		want[a] = struct{}{}
	}
	have := expand(t, got)

	for a := range want {
		if _, ok := have[a]; !ok {
			t.Fatalf("%s was permitted before consolidation and is not after — narrowing a "+
				"restriction silently is how the control plane loses its way back in", a)
		}
	}
	for a := range have {
		if _, ok := want[a]; !ok {
			t.Fatalf("%s is permitted after consolidation and was not before — this function "+
				"granted access nobody wrote", a)
		}
	}
	if len(got) >= len(input) {
		t.Fatalf("no reduction: %d in, %d out", len(input), len(got))
	}
	t.Logf("%d addresses -> %d prefixes, exactly equivalent", len(want), len(got))
}

// A run that does not start or end on a power-of-two boundary is the case that
// tempts an implementation into rounding outward. 10.62.0.2-38 is exactly that:
// the enclosing single prefix would be a /26, which also admits .0, .1 and
// .39-.63.
func TestConsolidationDoesNotRoundOutToACoveringPrefix(t *testing.T) {
	var input []string
	for i := 2; i <= 38; i++ {
		input = append(input, fmt.Sprintf("10.62.0.%d", i))
	}
	got := ConsolidateCIDRs(input)

	for _, p := range got {
		if p == "10.62.0.0/26" || p == "10.62.0.0/24" {
			t.Fatalf("rounded out to %s, which permits addresses that were never listed", p)
		}
	}
	have := expand(t, got)
	for _, unlisted := range []string{"10.62.0.0", "10.62.0.1", "10.62.0.39", "10.62.0.63"} {
		if _, ok := have[unlisted]; ok {
			t.Fatalf("%s was not in the input and is permitted after consolidation", unlisted)
		}
	}
}

// A prefix given as input is preserved as itself rather than exploded.
// Consolidation must not turn 10.62.0.0/24 into 256 entries — the input's own
// summarisation is information, not something to undo.
func TestConsolidationPreservesAPrefixItWasGiven(t *testing.T) {
	got := ConsolidateCIDRs([]string{"10.62.0.0/24", "192.168.1.5"})
	sort.Strings(got)
	if want := "10.62.0.0/24,192.168.1.5/32"; strings.Join(got, ",") != want {
		t.Fatalf("got %q, want %q", strings.Join(got, ","), want)
	}
}

// A host inside a prefix already listed adds nothing.
func TestConsolidationAbsorbsAHostCoveredByAPrefix(t *testing.T) {
	got := ConsolidateCIDRs([]string{"10.62.0.0/24", "10.62.0.7"})
	if len(got) != 1 || got[0] != "10.62.0.0/24" {
		t.Fatalf("got %v, want just the covering prefix", got)
	}
}

// Families never merge, and the output stays stable so an unchanged firewall
// renders an unchanged authorized_keys file.
func TestConsolidationKeepsFamiliesApartAndIsStable(t *testing.T) {
	in := []string{"2001:db8::2", "10.0.0.1", "2001:db8::3", "10.0.0.2"}
	a := ConsolidateCIDRs(in)
	b := ConsolidateCIDRs([]string{"10.0.0.2", "2001:db8::3", "10.0.0.1", "2001:db8::2"})

	if strings.Join(a, ",") != strings.Join(b, ",") {
		t.Fatalf("input order changed the output: %v vs %v", a, b)
	}
	for _, p := range a {
		if strings.Contains(p, ":") && strings.Contains(p, ".") {
			t.Fatalf("families merged into %q", p)
		}
	}
}

func TestConsolidationIgnoresUnparseableInput(t *testing.T) {
	got := ConsolidateCIDRs([]string{"10.0.0.1", "not-an-address", "", "10.0.0.2"})
	// Two /32s, NOT 10.0.0.1/31 — which is not a prefix at all. A /31 starts on
	// an even address, so the real boundaries are .0-.1 and .2-.3, and .1 with .2
	// straddles them. Adjacency does not imply alignment, and a consolidator that
	// assumed it would emit prefixes covering addresses nobody listed.
	if strings.Join(got, ",") != "10.0.0.1/32,10.0.0.2/32" {
		t.Fatalf("got %v, want the two real addresses and neither of the junk entries", got)
	}
}

// Adjacency alone is not enough; the pair must also be ALIGNED. This is the
// property that stops consolidation reaching outside what it was given.
func TestConsolidationMergesOnlyAlignedNeighbours(t *testing.T) {
	aligned := ConsolidateCIDRs([]string{"10.0.0.2", "10.0.0.3"})
	if len(aligned) != 1 || aligned[0] != "10.0.0.2/31" {
		t.Fatalf("an aligned pair did not merge: %v", aligned)
	}
	straddling := ConsolidateCIDRs([]string{"10.0.0.1", "10.0.0.2"})
	if len(straddling) != 2 {
		t.Fatalf("a straddling pair merged into %v, which would cover .0 or .3", straddling)
	}
}
