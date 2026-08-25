package tenant

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func setOf(family string, values ...string) *FirewallSet {
	s := &FirewallSet{ID: uuid.New(), Name: "mgmt", Family: family}
	for _, v := range values {
		s.Elements = append(s.Elements, FirewallSetElement{Value: v})
	}
	return s
}

// A set is one address family. nft declares it ipv4_addr or ipv6_addr and
// cannot hold both — and nft rejects the WHOLE ruleset over one bad line, so
// this mistake does not cost you a set, it costs the host its firewall.
func TestSetFamilyMustBeOneFamilyNotInet(t *testing.T) {
	if err := setOf(FamilyInet, "10.0.0.1").Validate(); !errors.Is(err, ErrFirewallSetInet) {
		t.Fatalf("got %v, want ErrFirewallSetInet — inet is a table family, not an address type", err)
	}
	for _, f := range []string{FamilyIPv4, FamilyIPv6} {
		v := "10.0.0.1"
		if f == FamilyIPv6 {
			v = "2001:db8::1"
		}
		if err := setOf(f, v).Validate(); err != nil {
			t.Fatalf("a %s set was refused: %v", f, err)
		}
	}
}

// An operator types 192.168.10.60, not 192.168.10.60/32. Refusing that would be
// pedantry, and the renderer uses `flags interval` anyway, which accepts both.
func TestSetAcceptsBareAddressesAndPrefixes(t *testing.T) {
	cases := []struct{ family, value string }{
		{FamilyIPv4, "192.168.10.60"},
		{FamilyIPv4, "192.168.10.0/24"},
		{FamilyIPv6, "2001:db8::1"},
		{FamilyIPv6, "2001:db8::/32"},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			if err := setOf(c.family, c.value).Validate(); err != nil {
				t.Fatalf("refused %q in a %s set: %v", c.value, c.family, err)
			}
		})
	}
}

func TestSetRefusesElementsFromTheWrongFamily(t *testing.T) {
	if err := setOf(FamilyIPv4, "2001:db8::1").Validate(); !errors.Is(err, ErrFirewallSetElement) {
		t.Fatalf("a v6 address went into a v4 set: %v", err)
	}
	if err := setOf(FamilyIPv6, "10.0.0.1").Validate(); !errors.Is(err, ErrFirewallSetElement) {
		t.Fatalf("a v4 address went into a v6 set: %v", err)
	}
}

func setRule(family string) FirewallRule {
	r := validRule()
	r.Family = family
	r.SrcCidr = AnyAddressFor(family)
	r.DstCidr = AnyAddressFor(family)
	return r
}

// nft renders one and ignores the other, so the stored rule and the enforced
// rule would differ with nothing reporting it — the failure mode this whole
// codebase keeps refusing.
func TestRuleRefusesBothAnAddressAndASet(t *testing.T) {
	set := setOf(FamilyIPv4, "192.168.10.60")
	r := setRule(FamilyInet)
	r.SrcCidr = "10.0.0.0/8"
	r.SrcSetID = &set.ID

	if err := r.ValidateWithSets(set, nil); !errors.Is(err, ErrFirewallSetAndCIDR) {
		t.Fatalf("got %v, want ErrFirewallSetAndCIDR", err)
	}
}

// The wildcard is what "unset" looks like on the wire, so it must not read as a
// second, conflicting specification.
func TestRuleAcceptsASetAlongsideTheWildcard(t *testing.T) {
	set := setOf(FamilyIPv4, "192.168.10.60")
	r := setRule(FamilyInet)
	r.SrcSetID = &set.ID

	if err := r.ValidateWithSets(set, nil); err != nil {
		t.Fatalf("a set with the default wildcard source was refused: %v", err)
	}
}

// An empty set is not neutral. An ACCEPT matching nothing silently CLOSES
// access; a DROP matching nothing silently OPENS it. Either way the rule reads
// as policy and enforces the opposite of what it looks like.
func TestRuleRefusesAnEmptySet(t *testing.T) {
	empty := setOf(FamilyIPv4)
	r := setRule(FamilyInet)
	r.SrcSetID = &empty.ID

	if err := r.ValidateWithSets(empty, nil); !errors.Is(err, ErrFirewallSetEmpty) {
		t.Fatalf("got %v, want ErrFirewallSetEmpty", err)
	}
}

// An inet rule may reference either family; a family-specific rule may not
// reference the other, because the rendered set would never match its traffic.
func TestSetFamilyMustAgreeWithTheRule(t *testing.T) {
	v4 := setOf(FamilyIPv4, "10.0.0.1")
	v6 := setOf(FamilyIPv6, "2001:db8::1")

	cases := []struct {
		name       string
		ruleFamily string
		set        *FirewallSet
		wantRefuse bool
	}{
		{"inet takes v4", FamilyInet, v4, false},
		{"inet takes v6", FamilyInet, v6, false},
		{"ip takes v4", FamilyIPv4, v4, false},
		{"ip6 takes v6", FamilyIPv6, v6, false},
		{"ip refuses v6", FamilyIPv4, v6, true},
		{"ip6 refuses v4", FamilyIPv6, v4, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := setRule(c.ruleFamily)
			r.SrcSetID = &c.set.ID
			err := r.ValidateWithSets(c.set, nil)
			if got := errors.Is(err, ErrFirewallSetFamily); got != c.wantRefuse {
				t.Fatalf("refused=%v want=%v (err=%v)", got, c.wantRefuse, err)
			}
		})
	}
}

// A dangling reference must not render. It would produce `@name` for a set that
// is never declared, and nft rejects the whole ruleset.
func TestRuleRefusesAnUnresolvableSet(t *testing.T) {
	id := uuid.New()
	r := setRule(FamilyInet)
	r.SrcSetID = &id

	if err := r.ValidateWithSets(nil, nil); !errors.Is(err, ErrFirewallSetElement) {
		t.Fatalf("got %v, want a refusal for an unresolvable set", err)
	}
}

// Rules with no set at all must be unaffected — the literal path is still the
// right answer for 0.0.0.0/0 and for one host.
func TestRuleWithoutSetsIsUnchanged(t *testing.T) {
	r := validRule()
	if err := r.ValidateWithSets(nil, nil); err != nil {
		t.Fatalf("an ordinary literal rule was refused: %v", err)
	}
}
