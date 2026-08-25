package service

import "testing"

func classifier() *NetworkClassifier {
	return NewNetworkClassifier(
		[]string{"3.0.0.0/8", "2600:1f00::/24"}, // stand-in for a provider range
		[]string{"185.220.101.0/24"},            // stand-in for a Tor exit range
		[]string{"3.9.9.0/24"},                  // SASE egress INSIDE the provider range
	)
}

// The interlock. This is the one that prevents the composite key from silently
// degrading back into the domain-only key it was introduced to replace.
func TestClassifyRefusesAKeyWhenAProxyIsUnparsed(t *testing.T) {
	c := classifier()

	got := c.Classify(SourceInput{
		ClientIP:       "10.42.0.7",   // the ingress pod, not the caller
		ForwardedFor:   "203.0.113.9", // proof a proxy is in front
		TrustedProxies: false,         // ...and we are not parsing it
	})

	if got.Trusted {
		t.Fatalf("an X-Forwarded-For with no trusted-proxy configuration means the client "+
			"address is the proxy's; classifying it as trusted collapses every caller "+
			"behind that ingress into one bucket. got %+v", got)
	}
	if got.Key != "" {
		t.Errorf("an untrusted source must produce no key at all, got %q", got.Key)
	}
}

// The same address IS usable once the proxy is configured, otherwise the
// interlock would make the feature permanently unusable behind any ingress.
func TestClassifyAcceptsTheAddressOnceProxiesAreTrusted(t *testing.T) {
	got := classifier().Classify(SourceInput{
		ClientIP:       "203.0.113.9",
		ForwardedFor:   "203.0.113.9",
		TrustedProxies: true,
	})
	if !got.Trusted {
		t.Fatalf("with TRUSTED_PROXY_CIDRS configured the derived address is usable: %+v", got)
	}
	if got.Key != "ip:203.0.113.9" {
		t.Errorf("key = %q, want per-address bucketing for a residential source", got.Key)
	}
}

// No proxy at all is the ordinary case and must not trip the interlock.
func TestClassifyTrustsADirectConnection(t *testing.T) {
	got := classifier().Classify(SourceInput{ClientIP: "198.51.100.4"})
	if !got.Trusted || got.Trust != SourceResidential {
		t.Fatalf("a direct connection with no XFF is usable and unclassifiable beyond "+
			"residential: %+v", got)
	}
}

// Datacenter addresses share one bucket per network. Bucketing them per address
// would hand an attacker on any cloud unlimited buckets, which is the whole
// reason the tier exists.
func TestDatacenterSourcesShareANetworkBucket(t *testing.T) {
	c := classifier()
	a := c.Classify(SourceInput{ClientIP: "3.5.140.1"})
	b := c.Classify(SourceInput{ClientIP: "3.250.9.99"})

	if a.Trust != SourceDatacenter || b.Trust != SourceDatacenter {
		t.Fatalf("addresses inside a configured provider range are datacenter: %+v / %+v", a, b)
	}
	if a.Key != b.Key {
		t.Fatalf("two addresses in one provider range must share a bucket, got %q and %q — "+
			"per-address bucketing inside a cloud is unlimited buckets", a.Key, b.Key)
	}
	if !a.Coarse {
		t.Error("a network-granularity bucket must report Coarse so a denial can be explained honestly")
	}
}

// A Tor exit inside a hosting range is still a Tor exit: the lower tier wins.
func TestAnonymisedBeatsDatacenter(t *testing.T) {
	c := NewNetworkClassifier([]string{"185.220.0.0/16"}, []string{"185.220.101.0/24"}, nil)
	got := c.Classify(SourceInput{ClientIP: "185.220.101.42"})
	if got.Trust != SourceAnonymised {
		t.Fatalf("overlapping ranges must resolve to the LOWER trust tier, got %v", got.Trust)
	}
}

// The tiers must stay ordered, because CeilingFor compares them.
func TestTrustTiersAreOrdered(t *testing.T) {
	// Ascending by trust. CeilingFor compares tiers with < and >=, so inserting
	// a tier in the wrong place silently reassigns every allowance above it.
	ordered := []struct {
		name  string
		trust SourceTrust
	}{
		{"anonymised", SourceAnonymised},
		{"datacenter", SourceDatacenter},
		{"enterprise egress", SourceEnterpriseEgress},
		{"residential", SourceResidential},
		{"corporate", SourceCorporate},
		{"corroborated", SourceCorroborated},
	}
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].trust >= ordered[i].trust {
			t.Fatalf("%s (%d) must rank below %s (%d)",
				ordered[i-1].name, ordered[i-1].trust, ordered[i].name, ordered[i].trust)
		}
	}
}

func TestCeilingScalesWithTrust(t *testing.T) {
	const base = 3

	if got := CeilingFor(base, SourceAnonymised, true); got != 0 {
		t.Errorf("an anonymised source must get NO allowance, not a small one — nothing "+
			"distinguishes one requester from the next, so any allowance is a global "+
			"one. got %d", got)
	}
	if CeilingFor(base, SourceDatacenter, true) > CeilingFor(base, SourceResidential, false) {
		t.Error("a datacenter source must not out-rank a residential one")
	}
	if CeilingFor(base, SourceCorroborated, false) <= CeilingFor(base, SourceResidential, false) {
		t.Error("corroboration — mailbox domain and IP allocation agreeing — is the strongest " +
			"signal available short of verifying the mailbox, and must earn more than an " +
			"unclassifiable residential address")
	}
}

// An unparseable address is not a licence to skip the ceiling.
func TestUnparseableAddressIsNotTrusted(t *testing.T) {
	for _, bad := range []string{"", "   ", "not-an-ip", "999.1.1.1", "10.0.0.1:443"} {
		if got := classifier().Classify(SourceInput{ClientIP: bad}); got.Trusted {
			t.Errorf("ClientIP %q classified as trusted: %+v", bad, got)
		}
	}
}

// A typo in an operator's range list must not take registration down.
func TestBadPrefixesAreSkippedNotFatal(t *testing.T) {
	c := NewNetworkClassifier([]string{"not-a-cidr", "3.0.0.0/8", ""}, nil, nil)
	if got := c.Classify(SourceInput{ClientIP: "3.5.140.1"}); got.Trust != SourceDatacenter {
		t.Fatalf("a valid prefix alongside a malformed one must still match: %+v", got)
	}
}

// A SASE egress sits inside a cloud provider's space; here the HIGHER tier must
// win, the inverse of the anonymised case. Getting this backwards makes a bank's
// entire workforce look like untrusted cloud automation.
func TestEnterpriseEgressBeatsDatacenter(t *testing.T) {
	got := classifier().Classify(SourceInput{ClientIP: "3.9.9.10"})
	if got.Trust != SourceEnterpriseEgress {
		t.Fatalf("a SASE range inside a provider range must resolve to enterprise egress, got %v", got.Trust)
	}
	if !got.Coarse {
		t.Error("enterprise egress is shared by a workforce and must report Coarse")
	}
}

// The two axes are independent: coarse-and-trusted must out-rank
// coarse-and-untrusted, or the most security-mature tenants are punished hardest.
func TestCoarsenessAndTrustAreIndependent(t *testing.T) {
	const base = 3
	corp := CeilingFor(base, SourceEnterpriseEgress, true)
	cloud := CeilingFor(base, SourceDatacenter, true)
	home := CeilingFor(base, SourceResidential, false)

	if corp <= cloud {
		t.Errorf("a corporate tunnel (%d) must out-rank cloud hosting (%d): both are coarse, "+
			"but one carries a workforce and the other carries automation", corp, cloud)
	}
	if corp <= home {
		t.Errorf("a bucket shared by a whole workforce (%d) needs more room than one household (%d)", corp, home)
	}
}
