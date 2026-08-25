package tenant

import (
	"strings"
)

// SSHPort is the port an authorized_keys `from=` restriction is derived for.
const SSHPort = 22

// FromRestriction is the derived authorized_keys `from=` list.
type FromRestriction struct {
	// Sources, deduplicated and sorted so the rendered file is stable — an
	// unstable render would rewrite authorized_keys on every convergence and make
	// "did this change" unanswerable.
	Sources []string

	// Unrestricted means some enabled accept covering SSH permits ANY source, so
	// no `from=` should be written at all.
	//
	// Distinct from an empty Sources list, and the distinction is the whole
	// point: "everyone may connect" and "nobody may connect" are opposite facts
	// that both render to an empty-looking result if collapsed.
	Unrestricted bool
}

// Safe reports whether this restriction can be written.
//
// An empty list is NOT "allow nothing" — as a `from=` clause it is a key that
// works from nowhere, which costs the control plane its ability to repair a
// broken agent. The caller must refuse rather than write it.
func (f FromRestriction) Safe() bool {
	return f.Unrestricted || len(f.Sources) > 0
}

// Render produces the authorized_keys option, or "" when no restriction applies.
func (f FromRestriction) Render() string {
	if f.Unrestricted || len(f.Sources) == 0 {
		return ""
	}
	return `from="` + strings.Join(f.Sources, ",") + `"`
}

// DeriveFromRestriction computes the `from=` list from the MANAGEMENT sets.
//
// It reads declared purpose rather than scanning rules by port, and that change
// is not tidiness. The earlier version collected the sources of every accept
// covering port 22, which is correct until a cluster exists: corosync peers are
// permitted on 22 so pvecm can distribute cluster configuration, so the derived
// list would include every hypervisor in the fleet. A stolen apiary private key
// would then be usable from any one of them — one compromised node becoming a
// foothold against all its siblings, which is precisely what `from=` is bought
// to prevent.
//
// No port can tell a peer from an operator. The declaration can.
//
// Consolidated on the way out: real management lists are enumerated per host and
// per address, and a `from=` clause nobody can read is one nobody audits.
func DeriveFromRestriction(sets []FirewallSet) FromRestriction {
	seen := map[string]struct{}{}
	out := FromRestriction{}

	for i := range sets {
		set := &sets[i]
		if set.Purpose != SetPurposeManagement {
			continue
		}
		for _, e := range set.Elements {
			v := strings.TrimSpace(e.Value)
			if v == "" {
				continue
			}
			// A management set holding the wildcard means the control plane may
			// connect from anywhere, so a clause would restrict nothing while
			// looking like it does.
			if v == AnyAddressFor(FamilyIPv4) || v == AnyAddressFor(FamilyIPv6) {
				out.Unrestricted = true
				return out
			}
			seen[v] = struct{}{}
		}
	}

	sources := make([]string, 0, len(seen))
	for v := range seen {
		sources = append(sources, v)
	}
	out.Sources = ConsolidateCIDRs(sources)
	return out
}
