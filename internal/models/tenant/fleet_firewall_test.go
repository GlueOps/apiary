package tenant

import "testing"

// The digest half is the whole point of this predicate, so the cases that must
// FAIL are the ones carrying a correct revision number. A version of
// FirewallConverged that compared only the revision would pass a naive
// happy-path test and report a green fleet for every host that mis-loaded a
// ruleset.
func TestFirewallConvergedRequiresBothRevisionAndDigest(t *testing.T) {
	const (
		want   = 7
		digest = "sha256:aa"
	)

	cases := []struct {
		name string
		bee  *Bee
		rev  int
		dig  string
		conv bool
	}{
		{
			name: "revision and digest both match",
			bee:  &Bee{AppliedFirewallRevision: 7, AppliedFirewallDigest: digest},
			rev:  want, dig: digest, conv: true,
		},
		{
			// The case the digest exists for: the bee says 7, and 7 is indeed
			// desired, but what it loaded is not what was published.
			name: "right revision, wrong content",
			bee:  &Bee{AppliedFirewallRevision: 7, AppliedFirewallDigest: "sha256:bb"},
			rev:  want, dig: digest, conv: false,
		},
		{
			// A revert publishes an OLD body under a NEW number, so content
			// matching while the revision lags is a real, reachable state — and it
			// is still not converged, because the bee has not seen the revert.
			name: "content matches an earlier revision",
			bee:  &Bee{AppliedFirewallRevision: 6, AppliedFirewallDigest: digest},
			rev:  want, dig: digest, conv: false,
		},
		{
			name: "never applied anything",
			bee:  &Bee{},
			rev:  want, dig: digest, conv: false,
		},
		{
			// Nothing published yet: a freshly enrolled host is not "behind".
			name: "no ruleset published",
			bee:  &Bee{},
			rev:  0, dig: "", conv: true,
		},
		{
			// Guards the ordering inside the function. Reading b.Applied... before
			// the nil check panics; returning false before the zero check would
			// report a host as behind a ruleset that does not exist.
			name: "nil bee, nothing published",
			bee:  nil,
			rev:  0, dig: "", conv: true,
		},
		{
			name: "nil bee, ruleset published",
			bee:  nil,
			rev:  want, dig: digest, conv: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.bee.FirewallConverged(c.rev, c.dig); got != c.conv {
				t.Fatalf("FirewallConverged(%d, %q) = %v, want %v", c.rev, c.dig, got, c.conv)
			}
		})
	}
}
