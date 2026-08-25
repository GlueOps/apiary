package tenant

import (
	"math/big"
	"net/netip"
	"sort"
	"strings"
)

// ConsolidateCIDRs reduces a list of addresses and prefixes to the fewest
// prefixes covering EXACTLY the same addresses.
//
// Consolidation, deliberately not aggregation. In networking, aggregating CIDRs
// means route summarisation, and summarisation WIDENS by design: a router given
// 203.0.113.16 and .17 is correct to advertise 203.0.113.0/27, because a route
// may cover more than it was told about. An access control may not. The same
// input here yields 203.0.113.16/31 — two addresses, not thirty-two — and the
// difference is thirty addresses nobody listed being able to reach SSH.
//
// Same coverage, fewer entries. That is the whole contract, and it is enforced
// by a test that expands every emitted prefix back to addresses and asserts set
// equality in BOTH directions — not by reading this function and believing it.
//
// It exists because real whitelists are enumerated per host. A production
// pve-ssh-whitelist observed in this project had 85 lines — two contiguous runs
// of 37 addresses plus a scattering of singles — which as a flat authorized_keys
// `from=` clause is 1062 characters. Consolidated it is 23 prefixes and 361.
// Length is the visible problem; the real one is that nobody can read an
// 85-entry list to answer "is this address permitted", so nobody audits it.
//
// Families are never merged: a v4 and a v6 prefix cannot summarise each other,
// and the sort keeps v4 first so the output is stable.
func ConsolidateCIDRs(sources []string) []string {
	var v4, v6 []netip.Prefix
	for _, s := range sources {
		p, ok := parsePrefix(s)
		if !ok {
			continue
		}
		if p.Addr().Is4() {
			v4 = append(v4, p)
		} else {
			v6 = append(v6, p)
		}
	}
	out := append(consolidate(v4, 32), consolidate(v6, 128)...)
	return out
}

// parsePrefix accepts a bare address as well as a prefix, because an operator
// writing a management address types 192.168.10.60, not 192.168.10.60/32.
func parsePrefix(s string) (netip.Prefix, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Prefix{}, false
	}
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return netip.Prefix{}, false
		}
		return p.Masked(), true
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, false
	}
	bits := 32
	if !a.Is4() {
		bits = 128
	}
	return netip.PrefixFrom(a, bits), true
}

// consolidate merges prefixes of one family into the fewest covering prefixes.
//
// Works on integer intervals rather than on prefixes directly: two prefixes that
// are adjacent but not siblings — 10.0.0.2/31 and 10.0.0.4/30 — cannot be merged
// by any prefix-level rule, yet together with their neighbours they form one run
// that decomposes into fewer blocks than it started with. Intervals see that;
// pairwise prefix merging does not.
func consolidate(ps []netip.Prefix, totalBits int) []string {
	if len(ps) == 0 {
		return nil
	}
	type span struct{ lo, hi *big.Int }
	spans := make([]span, 0, len(ps))
	for _, p := range ps {
		lo := addrToInt(p.Addr())
		size := new(big.Int).Lsh(big.NewInt(1), uint(totalBits-p.Bits()))
		hi := new(big.Int).Add(lo, size)
		hi.Sub(hi, big.NewInt(1))
		spans = append(spans, span{lo, hi})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].lo.Cmp(spans[j].lo) < 0 })

	merged := []span{spans[0]}
	for _, s := range spans[1:] {
		last := &merged[len(merged)-1]
		// Adjacent counts as overlapping: .2/31 ending at .3 and .4/30 starting at
		// .4 are one run, and treating them as separate is what leaves the output
		// as long as the input.
		next := new(big.Int).Add(last.hi, big.NewInt(1))
		if s.lo.Cmp(next) <= 0 {
			if s.hi.Cmp(last.hi) > 0 {
				last.hi = s.hi
			}
			continue
		}
		merged = append(merged, s)
	}

	var out []string
	for _, m := range merged {
		out = append(out, decompose(m.lo, m.hi, totalBits)...)
	}
	return out
}

// decompose renders one interval as the fewest prefixes that cover it exactly.
//
// At each step it takes the largest block that both STARTS at lo (bounded by
// lo's trailing zeros) and FITS inside the remaining interval. Ignoring either
// bound produces prefixes that are unaligned or that overrun the interval —
// which is how this silently grants addresses nobody listed.
func decompose(lo, hi *big.Int, totalBits int) []string {
	var out []string
	cur := new(big.Int).Set(lo)
	for cur.Cmp(hi) <= 0 {
		maxByAlign := totalBits
		if cur.Sign() != 0 {
			maxByAlign = trailingZeros(cur)
		}
		remaining := new(big.Int).Sub(hi, cur)
		remaining.Add(remaining, big.NewInt(1))
		maxBySize := remaining.BitLen() - 1

		size := maxByAlign
		if maxBySize < size {
			size = maxBySize
		}
		bits := totalBits - size
		addr := intToAddr(cur, totalBits)
		out = append(out, netip.PrefixFrom(addr, bits).String())

		cur = new(big.Int).Add(cur, new(big.Int).Lsh(big.NewInt(1), uint(size)))
	}
	return out
}

func trailingZeros(n *big.Int) int {
	for i := 0; i < n.BitLen(); i++ {
		if n.Bit(i) == 1 {
			return i
		}
	}
	return n.BitLen()
}

func addrToInt(a netip.Addr) *big.Int {
	if a.Is4() {
		b := a.As4()
		return new(big.Int).SetBytes(b[:])
	}
	b := a.As16()
	return new(big.Int).SetBytes(b[:])
}

func intToAddr(n *big.Int, totalBits int) netip.Addr {
	size := totalBits / 8
	b := make([]byte, size)
	n.FillBytes(b)
	a, _ := netip.AddrFromSlice(b)
	return a
}
