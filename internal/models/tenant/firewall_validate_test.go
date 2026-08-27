package tenant

import (
	"errors"
	"github.com/google/uuid"
	"testing"
)

func validRule() FirewallRule {
	return FirewallRule{
		Name: "ssh from ops", Enabled: true,
		Family: FamilyInet, TableName: "filter", ChainName: "input", Position: 10,
		SrcCidr: "192.168.0.0/24", DstCidr: "0.0.0.0/0",
		Protocol: TCP, DstPort: 22, Action: Accept,
	}
}

func TestValidateAcceptsAWellFormedRule(t *testing.T) {
	r := validRule()
	if err := r.Validate(); err != nil {
		t.Fatalf("a valid rule was refused: %v", err)
	}
}

func TestValidateRefusesEachMalformedField(t *testing.T) {
	cases := []struct {
		name string
		bend func(*FirewallRule)
		want error
	}{
		{"unknown family", func(r *FirewallRule) { r.Family = "inet6" }, ErrFirewallFamily},
		{"unknown protocol", func(r *FirewallRule) { r.Protocol = "sctp" }, ErrFirewallProtocol},
		{"unknown action", func(r *FirewallRule) { r.Action = "ACCEPT" }, ErrFirewallAction},
		{"empty chain", func(r *FirewallRule) { r.ChainName = "  " }, ErrFirewallChainName},
		{"negative position", func(r *FirewallRule) { r.Position = -1 }, ErrFirewallPosition},
		{"source is not a prefix", func(r *FirewallRule) { r.SrcCidr = "192.168.0.1" }, ErrFirewallCIDR},
		{"destination is nonsense", func(r *FirewallRule) { r.DstCidr = "not-an-address" }, ErrFirewallCIDR},
		{"port above the range", func(r *FirewallRule) { r.DstPort = 70000 }, ErrFirewallPort},
		{"negative port", func(r *FirewallRule) { r.SrcPort = -1 }, ErrFirewallPort},
		{"range runs backwards", func(r *FirewallRule) { r.DstPort, r.DstPortEnd = 8000, 80 }, ErrFirewallPortRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := validRule()
			c.bend(&r)
			err := r.Validate()
			if !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

// The case worth having this function for at all.
//
// nft discards a port match on a protocol that has none, so the rule an
// operator reads back is not the rule being enforced — "allow icmp from here on
// port 22" silently becomes "allow ALL icmp from here", with nothing anywhere
// reporting a difference. Every other check here catches something that would
// fail loudly; this one catches something that would not.
func TestValidateRefusesPortsOnAProtocolThatHasNone(t *testing.T) {
	for _, proto := range []Protocol{ICMP, ProtocolAny} {
		t.Run(string(proto), func(t *testing.T) {
			r := validRule()
			r.Protocol = proto
			r.DstPort = 22
			if err := r.Validate(); !errors.Is(err, ErrFirewallPortless) {
				t.Fatalf("a %s rule carrying a port was accepted (%v); rendering drops the port "+
					"and the stored rule is wider than the one written", proto, err)
			}
		})
	}
}

func TestValidateAcceptsPortlessProtocolsWithoutPorts(t *testing.T) {
	for _, proto := range []Protocol{ICMP, ProtocolAny} {
		t.Run(string(proto), func(t *testing.T) {
			r := validRule()
			r.Protocol = proto
			r.DstPort, r.SrcPort = 0, 0
			if err := r.Validate(); err != nil {
				t.Fatalf("a portless %s rule was refused: %v", proto, err)
			}
		})
	}
}

// A v6 range in a rule declared ip renders into a table that cannot hold it, and
// nft rejects the WHOLE ruleset rather than the offending line — so one bad
// entry takes the host's entire firewall down with it.
func TestValidateRefusesAddressesFromTheWrongFamily(t *testing.T) {
	cases := []struct{ name, family, cidr string }{
		{"v6 range in an ip rule", FamilyIPv4, "2001:db8::/32"},
		{"v4 range in an ip6 rule", FamilyIPv6, "10.0.0.0/8"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := validRule()
			r.Family, r.SrcCidr = c.family, c.cidr
			if c.family == FamilyIPv6 {
				r.DstCidr = "::/0"
			}
			if err := r.Validate(); !errors.Is(err, ErrFirewallCIDRFamily) {
				t.Fatalf("got %v, want ErrFirewallCIDRFamily — nft rejects the entire ruleset "+
					"over one mismatched rule, so the host loses its whole firewall", err)
			}
		})
	}
}

// inet holds both, which is why it is the default.
func TestValidateAllowsEitherFamilyUnderInet(t *testing.T) {
	for _, cidr := range []string{"10.0.0.0/8", "2001:db8::/32"} {
		r := validRule()
		r.Family, r.SrcCidr = FamilyInet, cidr
		if err := r.Validate(); err != nil {
			t.Fatalf("inet refused %s: %v", cidr, err)
		}
	}
}

// Zero end means "just the start port" and must not read as a backwards range.
func TestValidateTreatsAnUnsetRangeEndAsASinglePort(t *testing.T) {
	r := validRule()
	r.DstPort, r.DstPortEnd = 30000, 0
	if err := r.Validate(); err != nil {
		t.Fatalf("a single-port rule was refused: %v", err)
	}
	if start, end := PortRange(r.DstPort, r.DstPortEnd); start != 30000 || end != 30000 {
		t.Fatalf("PortRange(30000, 0) = %d-%d, want 30000-30000", start, end)
	}
}

func TestRulesetNeedsAName(t *testing.T) {
	if err := (&FirewallRuleset{Name: "  "}).Validate(); !errors.Is(err, ErrFirewallRulesetName) {
		t.Fatalf("got %v, want ErrFirewallRulesetName", err)
	}
	if err := (&FirewallRuleset{Name: "management"}).Validate(); err != nil {
		t.Fatalf("a named ruleset was refused: %v", err)
	}
}

// A rule that names a set carries no literal on that side, and must not be
// asked for one.
//
// The any-address default is deliberately not applied to a side with a set,
// because a literal alongside a set is refused — nft renders one and ignores
// the other. Validating the resulting empty string anyway rejected every
// set-based rule an operator could write, with `malformed address range:
// source ""` — a true statement about a field they were right to leave blank.
// Caught against a live control plane, not in review.
func TestARuleNamingASetNeedsNoLiteralOnThatSide(t *testing.T) {
	setID := uuid.New()
	// With elements: an empty set is separately refused, because a set that
	// matches nothing silently turns an accept rule into a no-op.
	set := FirewallSet{
		ID: setID, Name: "allowed_ips", Family: FamilyIPv4,
		Elements: []FirewallSetElement{{SetID: setID, Value: "192.0.2.10"}},
	}

	r := FirewallRule{
		Name: "ssh from the whitelist", Enabled: true,
		Family: FamilyInet, TableName: "filter", ChainName: "input",
		Protocol: TCP, Action: Accept, DstPort: 22,
		SrcSetID: &setID,
		DstCidr:  AnyAddressFor(FamilyInet),
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("a source set with no source literal must validate: %v", err)
	}
	if err := r.validateAgainstSets(&set, nil); err != nil {
		t.Fatalf("resolved against its set it must still validate: %v", err)
	}

	// The other half of the contract: a literal AND a set is still refused,
	// because nft renders one and ignores the other.
	both := r
	both.SrcCidr = "10.0.0.0/8"
	if err := both.validateAgainstSets(&set, nil); err == nil {
		t.Fatal("naming both a set and a literal must be refused")
	}

	// And a malformed literal on a side WITHOUT a set is still caught.
	bad := r
	bad.SrcSetID = nil
	bad.SrcCidr = "not-an-address"
	if err := bad.Validate(); err == nil {
		t.Fatal("a malformed literal must still be refused when no set is named")
	}
}
