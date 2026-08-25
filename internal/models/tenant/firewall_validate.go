package tenant

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/google/uuid"
	"strings"
)

// Validation failures. Named, because every one of them is something a person
// typed and must be told about precisely — "invalid rule" sends an operator
// back to guess which of eleven fields was wrong.
var (
	ErrFirewallFamily      = errors.New("firewall: unknown address family")
	ErrFirewallProtocol    = errors.New("firewall: unknown protocol")
	ErrFirewallAction      = errors.New("firewall: unknown action")
	ErrFirewallCIDR        = errors.New("firewall: malformed address range")
	ErrFirewallCIDRFamily  = errors.New("firewall: address range does not match the rule's family")
	ErrFirewallPort        = errors.New("firewall: port out of range")
	ErrFirewallPortRange   = errors.New("firewall: port range ends before it starts")
	ErrFirewallPortless    = errors.New("firewall: this protocol has no ports")
	ErrFirewallChainName   = errors.New("firewall: chain name is not usable")
	ErrFirewallPosition    = errors.New("firewall: position must not be negative")
	ErrFirewallRulesetName = errors.New("firewall: a ruleset needs a name")

	ErrFirewallSetAndCIDR = errors.New("firewall: a rule names an address OR a set, not both")
	ErrFirewallSetFamily  = errors.New("firewall: the set's address family does not match the rule's")
	ErrFirewallSetName    = errors.New("firewall: a set needs a name")
	ErrFirewallSetInet    = errors.New("firewall: a set is one address family; inet is not one")
	ErrFirewallSetElement = errors.New("firewall: malformed set element")
	ErrFirewallSetEmpty   = errors.New("firewall: a set with no elements matches nothing")
)

// Families this renders. inet covers both, which is the default and almost
// always what is wanted.
const (
	FamilyInet = "inet"
	FamilyIPv4 = "ip"
	FamilyIPv6 = "ip6"
)

// Validate refuses a rule that cannot mean what it says.
//
// It runs BEFORE anything is stored, because a firewall rule that is merely
// wrong is a normal mistake, while one that is nonsense gets silently
// reinterpreted: nft drops a port match on a protocol that has no ports, so
// "allow icmp from here on port 22" quietly becomes "allow all icmp from here".
// The rule an operator reads back is then not the rule being enforced, and no
// error was ever raised. Refusing is the only way that stays visible.
func (r *FirewallRule) Validate() error {
	switch r.Family {
	case FamilyInet, FamilyIPv4, FamilyIPv6:
	default:
		return fmt.Errorf("%w: %q (want inet, ip or ip6)", ErrFirewallFamily, r.Family)
	}

	switch r.Protocol {
	case TCP, UDP, ICMP, ProtocolAny:
	default:
		return fmt.Errorf("%w: %q", ErrFirewallProtocol, r.Protocol)
	}

	switch r.Action {
	case Accept, Drop, Reject:
	default:
		return fmt.Errorf("%w: %q (want accept, drop or reject)", ErrFirewallAction, r.Action)
	}

	if strings.TrimSpace(r.ChainName) == "" {
		return fmt.Errorf("%w: it is empty", ErrFirewallChainName)
	}
	if r.Position < 0 {
		return fmt.Errorf("%w: %d", ErrFirewallPosition, r.Position)
	}

	for label, cidr := range map[string]string{"source": r.SrcCidr, "destination": r.DstCidr} {
		if err := validateCIDR(label, cidr, r.Family); err != nil {
			return err
		}
	}

	// Ports only mean something for TCP and UDP. Carrying one on icmp or any is
	// not a harmless extra: it is discarded during rendering, widening the rule
	// far past what was written.
	hasPorts := r.Protocol == TCP || r.Protocol == UDP
	for label, pair := range map[string][2]int{
		"source":      {r.SrcPort, r.SrcPortEnd},
		"destination": {r.DstPort, r.DstPortEnd},
	} {
		start, end := pair[0], pair[1]
		if !hasPorts && (start != 0 || end != 0) {
			return fmt.Errorf("%w: %s carries a %s port, which is dropped when rendered — "+
				"the stored rule would be wider than the one written", ErrFirewallPortless, r.Protocol, label)
		}
		if err := validatePortPair(label, start, end); err != nil {
			return err
		}
	}
	return nil
}

func validateCIDR(label, cidr, family string) error {
	p, err := netip.ParsePrefix(strings.TrimSpace(cidr))
	if err != nil {
		return fmt.Errorf("%w: %s %q", ErrFirewallCIDR, label, cidr)
	}
	// A v6 range in a rule declared ip renders into a table that cannot hold it,
	// and nft rejects the whole ruleset rather than the one rule — so one bad
	// entry takes the host's entire firewall down with it. Catch it here.
	switch family {
	case FamilyIPv4:
		if !p.Addr().Is4() {
			return fmt.Errorf("%w: %s %q is IPv6 but the family is ip", ErrFirewallCIDRFamily, label, cidr)
		}
	case FamilyIPv6:
		if p.Addr().Is4() {
			return fmt.Errorf("%w: %s %q is IPv4 but the family is ip6", ErrFirewallCIDRFamily, label, cidr)
		}
	}
	return nil
}

func validatePortPair(label string, start, end int) error {
	for _, p := range []int{start, end} {
		if p < 0 || p > 65535 {
			return fmt.Errorf("%w: %s port %d", ErrFirewallPort, label, p)
		}
	}
	// end < start is rejected; end == 0 is "no end", which PortRange collapses.
	if end != 0 && end < start {
		return fmt.Errorf("%w: %s ports %d-%d", ErrFirewallPortRange, label, start, end)
	}
	return nil
}

// Validate refuses a ruleset that cannot be referred to.
func (s *FirewallRuleset) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return ErrFirewallRulesetName
	}
	return nil
}

// Validate refuses a set nft would reject or that would silently match nothing.
func (s *FirewallSet) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return ErrFirewallSetName
	}
	// inet is a TABLE family, not an address type. An nft set is declared
	// ipv4_addr or ipv6_addr and cannot hold both, so "inet" here would render a
	// declaration nft rejects — and nft rejects the whole ruleset over one bad
	// line, taking the host's entire firewall with it.
	switch s.Family {
	case FamilyIPv4, FamilyIPv6:
	case FamilyInet:
		return fmt.Errorf("%w: use %q or %q and let the rule stay inet", ErrFirewallSetInet, FamilyIPv4, FamilyIPv6)
	default:
		return fmt.Errorf("%w: %q", ErrFirewallFamily, s.Family)
	}
	for _, e := range s.Elements {
		if err := validateCIDR("element", elementPrefix(e.Value), s.Family); err != nil {
			return fmt.Errorf("%w: %w", ErrFirewallSetElement, err)
		}
	}
	return nil
}

// elementPrefix accepts a bare address as well as a prefix.
//
// An operator writing a management address types 192.168.10.60, not
// 192.168.10.60/32, and refusing that would be pedantry. nft accepts both in a
// set declared with `flags interval`, which is what the renderer must use — that
// flag is required for prefixes and costs nothing for single addresses, so
// always setting it removes a whole class of "why does this set reject a /24".
func elementPrefix(v string) string {
	v = strings.TrimSpace(v)
	if strings.Contains(v, "/") {
		return v
	}
	if strings.Contains(v, ":") {
		return v + "/128"
	}
	return v + "/32"
}

// validateAgainstSets refuses the two ways a set reference goes wrong.
//
// Called from Validate with the referenced sets resolved, because whether a
// reference is sound depends on the SET, which the rule row does not carry.
// Passing them in rather than loading here keeps this package free of a database.
func (r *FirewallRule) validateAgainstSets(src, dst *FirewallSet) error {
	check := func(label, cidr string, setID *uuid.UUID, set *FirewallSet) error {
		if setID == nil {
			return nil
		}
		// Both specified. nft renders one and ignores the other, so the stored
		// rule and the enforced rule would differ with nothing reporting it.
		if strings.TrimSpace(cidr) != "" && strings.TrimSpace(cidr) != AnyAddressFor(r.Family) {
			return fmt.Errorf("%w: %s names %q and a set", ErrFirewallSetAndCIDR, label, cidr)
		}
		if set == nil {
			return fmt.Errorf("%w: %s names a set that could not be resolved", ErrFirewallSetElement, label)
		}
		// A set with nothing in it renders to a rule matching no traffic. That is
		// not neutral: an ACCEPT that matches nothing silently closes access, and
		// a DROP that matches nothing silently opens it.
		if len(set.Elements) == 0 {
			return fmt.Errorf("%w: %s references %q", ErrFirewallSetEmpty, label, set.Name)
		}
		// inet rules take either; a family-specific rule takes only its own.
		if r.Family != FamilyInet && r.Family != set.Family {
			return fmt.Errorf("%w: rule is %s, set %q is %s", ErrFirewallSetFamily, r.Family, set.Name, set.Family)
		}
		return nil
	}
	if err := check("source", r.SrcCidr, r.SrcSetID, src); err != nil {
		return err
	}
	return check("destination", r.DstCidr, r.DstSetID, dst)
}

// ValidateWithSets is Validate plus the checks that need the referenced sets.
func (r *FirewallRule) ValidateWithSets(src, dst *FirewallSet) error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.validateAgainstSets(src, dst)
}

// AnyAddressFor is the wildcard for a family, so "unset" and "everything" are
// the same value rather than two.
//
// Exported and used by the service layer too: the wildcard appearing as two
// literals in two packages is how "" and 0.0.0.0/0 come to mean different things
// in different code paths.
func AnyAddressFor(family string) string {
	if family == FamilyIPv6 {
		return "::/0"
	}
	return "0.0.0.0/0"
}
