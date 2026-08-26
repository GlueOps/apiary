package tenant

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// Rendering an ordered rule list into an nft ruleset a host can load.
//
// The output is a COMPLETE ruleset, not a patch. `nft -f` on it replaces the
// tables it declares wholesale, which is what makes a host that missed three
// revisions converge on its next poll rather than replay a history it never saw
// — see FirewallRule's "desired state, not commands".
//
// It is deliberately pure: rules and sets in, text and digest out, no database
// and no clock. The snapshot stores the rendered text precisely so a change here
// cannot alter what an already-applied revision means, and a renderer that read
// anything else would break that guarantee the moment two agents disagreed.

// RenderInput is everything one hive's ruleset is rendered from.
type RenderInput struct {
	// Rules in the order they must evaluate — the order Preview returns, which
	// CompareBindingsForRender and OrderedRules already agree on. Rendering does
	// NOT re-sort: an ordering decided in two places stops agreeing, and here the
	// consequence is an operator reading a different policy from the one the host
	// enforces.
	Rules []FirewallRule

	// Sets referenced by those rules, by id. A rule naming a set that is absent
	// is an error rather than a skipped line: nft rejects a ruleset referencing
	// an undeclared set, and it rejects the WHOLE file, so quietly dropping the
	// rule would produce a host with no firewall at all instead of one missing a
	// line.
	Sets map[string]FirewallSet

	// Elements of those sets, by set id.
	Elements map[string][]FirewallSetElement
}

// Rendered is one hive's ruleset as a host will load it.
type Rendered struct {
	// Body is the nft script.
	Body string

	// Digest is sha256 of Body, hex, prefixed. A bee compares this against what
	// it applied to decide whether to converge, so it must depend on the BODY and
	// nothing else — a digest over the input rows would change when a comment
	// moved and stay still when the renderer changed.
	Digest string

	// Families is the set of address families the body declares tables for, so a
	// caller can say what was rendered without parsing it back.
	Families []string
}

// The chain preamble, emitted before any authored rule in every input chain.
//
// WITHOUT THESE, THE FIRST DROP RULE CUTS THE HOST OFF. A ruleset ending in a
// drop is the ordinary shape of a firewall, and applying one to a live host
// severs every established connection — including the SSH session applying it
// and, on a Proxmox node, the corosync ring that keeps the cluster quorate.
// Neither is a hypothetical: both are what "apply a default-deny policy" means
// on a machine that is currently being administered over the network.
//
// They are emitted by the renderer rather than left to an operator's first rule
// because an operator who forgets loses the host, and the failure is discovered
// only by walking to it. There is no rule an operator can write that makes these
// harmful: accepting established traffic accepts only what a rule already
// permitted to start, and accepting loopback accepts what never leaves the host.
const (
	acceptEstablished = "ct state established,related accept"
	acceptLoopback    = "iif lo accept"
	dropInvalid       = "ct state invalid drop"
)

// Render turns an ordered rule list into an nft ruleset.
func Render(in RenderInput) (Rendered, error) {
	// Grouped by TABLE first and chains within it, not by (table, chain) flat.
	// A table with two chains rendered flat emits two `table` blocks, and the
	// sets scoped to that table are then declared twice — which nft rejects, and
	// it rejects the whole file, leaving the host with no firewall at all.
	type tableKey struct{ family, table string }

	tableOrder := make([]tableKey, 0, 2)
	chainOrder := make(map[tableKey][]string)
	byChain := make(map[tableKey]map[string][]FirewallRule)
	// Sets are tracked PER TABLE, because an nft set is scoped to its table and
	// a set of one address family declared inside a table of another is a
	// declaration nft refuses.
	setsPerTable := make(map[tableKey]map[string]bool)

	for i := range in.Rules {
		r := in.Rules[i]
		if !r.Enabled {
			continue
		}
		tk := tableKey{family: r.Family, table: r.TableName}
		if _, seen := byChain[tk]; !seen {
			// First appearance decides order, so the rendered file follows the rule
			// order the operator sees rather than a map's iteration.
			tableOrder = append(tableOrder, tk)
			byChain[tk] = make(map[string][]FirewallRule)
			setsPerTable[tk] = make(map[string]bool)
		}
		if _, seen := byChain[tk][r.ChainName]; !seen {
			chainOrder[tk] = append(chainOrder[tk], r.ChainName)
		}
		byChain[tk][r.ChainName] = append(byChain[tk][r.ChainName], r)

		for _, id := range []*uuid.UUID{r.SrcSetID, r.DstSetID} {
			if id != nil {
				setsPerTable[tk][id.String()] = true
			}
		}
	}

	var b strings.Builder
	// flush ruleset first: the body is desired state, so anything already on the
	// host that this file does not declare must go. Without it a table removed
	// from the policy stays loaded forever, still filtering, invisible to every
	// tool that reads the policy from here.
	b.WriteString("#!/usr/sbin/nft -f\n")
	b.WriteString("# Rendered by APIary. Do not edit: the next convergence overwrites it.\n")
	b.WriteString("flush ruleset\n")

	families := map[string]bool{}
	for _, tk := range tableOrder {
		families[tk.family] = true

		b.WriteString("\n")
		fmt.Fprintf(&b, "table %s %s {\n", tk.family, tk.table)

		for _, sid := range sortedKeys(setsPerTable[tk]) {
			set, ok := in.Sets[sid]
			if !ok {
				return Rendered{}, fmt.Errorf("render: a rule references set %s, which was not "+
					"supplied; nft rejects the whole ruleset over an undeclared set, so this "+
					"would leave the host with no firewall rather than one missing a line", sid)
			}
			if err := writeSet(&b, set, in.Elements[sid]); err != nil {
				return Rendered{}, err
			}
		}

		for _, chain := range chainOrder[tk] {
			fmt.Fprintf(&b, "  chain %s {\n", chain)
			// policy accept, deliberately. A drop policy denies everything the
			// moment the file loads, before any accept rule is evaluated; an
			// operator expresses deny by writing a rule, which renders after the
			// preamble below and so cannot sever what is already established.
			fmt.Fprintf(&b, "    type filter hook %s priority 0; policy accept;\n", hookFor(chain))
			if isInputLike(chain) {
				fmt.Fprintf(&b, "    %s\n", acceptEstablished)
				fmt.Fprintf(&b, "    %s\n", acceptLoopback)
				fmt.Fprintf(&b, "    %s\n", dropInvalid)
			}
			for _, r := range byChain[tk][chain] {
				line, err := renderRule(r, in.Sets)
				if err != nil {
					return Rendered{}, err
				}
				fmt.Fprintf(&b, "    %s\n", line)
			}
			b.WriteString("  }\n")
		}
		b.WriteString("}\n")
	}

	body := b.String()
	sum := sha256.Sum256([]byte(body))
	return Rendered{
		Body:     body,
		Digest:   "sha256:" + hex.EncodeToString(sum[:]),
		Families: sortedBoolKeys(families),
	}, nil
}

// renderRule turns one rule into one nft line.
func renderRule(r FirewallRule, sets map[string]FirewallSet) (string, error) {
	var parts []string

	if r.InIface != "" {
		parts = append(parts, "iifname "+quote(r.InIface))
	}
	if r.OutIface != "" {
		parts = append(parts, "oifname "+quote(r.OutIface))
	}

	src, err := addrMatch(r.Family, "saddr", r.SrcCidr, r.SrcSetID, sets)
	if err != nil {
		return "", err
	}
	parts = append(parts, src...)

	dst, err := addrMatch(r.Family, "daddr", r.DstCidr, r.DstSetID, sets)
	if err != nil {
		return "", err
	}
	parts = append(parts, dst...)

	switch r.Protocol {
	case ProtocolAny:
		// No protocol match at all. Emitting `meta l4proto { tcp, udp }` would
		// EXCLUDE icmp and everything else, which is the opposite of "any" and the
		// kind of near-miss nobody reads twice.
	case ICMP:
		// icmp under inet must name both, or v6 ping is silently unmatched by a
		// rule whose whole purpose was to allow ping.
		if r.Family == "inet" {
			parts = append(parts, "meta l4proto { icmp, icmpv6 }")
		} else {
			parts = append(parts, "ip protocol icmp")
		}
	default:
		parts = append(parts, "meta l4proto "+string(r.Protocol))
	}

	if portMatch := ports(r.Protocol, "sport", r.SrcPort, r.SrcPortEnd); portMatch != "" {
		parts = append(parts, string(r.Protocol)+" "+portMatch)
	}
	if portMatch := ports(r.Protocol, "dport", r.DstPort, r.DstPortEnd); portMatch != "" {
		parts = append(parts, string(r.Protocol)+" "+portMatch)
	}

	parts = append(parts, string(r.Action))

	// The rule's name travels into the ruleset as an nft comment. It is what
	// makes `nft list ruleset` on the host readable against the policy in the UI,
	// which is the only way to answer "is this host running what we think" from
	// the host itself.
	label := r.Name
	if r.Comment != "" {
		label = r.Name + " — " + r.Comment
	}
	if label != "" {
		parts = append(parts, "comment "+quote(label))
	}
	return strings.Join(parts, " "), nil
}

// addrMatch renders a source or destination match.
//
// A rule names EITHER a literal or a set — Validate refuses both — so this picks
// whichever is present and emits nothing for an any-address literal, which keeps
// the common rule readable rather than padding every line with 0.0.0.0/0.
func addrMatch(
	family, side, cidr string, setID *uuid.UUID, sets map[string]FirewallSet,
) ([]string, error) {
	if setID != nil {
		s, ok := sets[setID.String()]
		if !ok {
			return nil, fmt.Errorf("render: a rule references set %s, which was not supplied", *setID)
		}
		return []string{fmt.Sprintf("%s %s @%s", addrFamily(family, s.Family), side, setName(s))}, nil
	}
	if cidr == "" || cidr == "0.0.0.0/0" || cidr == "::/0" {
		return nil, nil
	}
	fam := "ip"
	if strings.Contains(cidr, ":") {
		fam = "ip6"
	}
	return []string{fmt.Sprintf("%s %s %s", fam, side, cidr)}, nil
}

// addrFamily picks the keyword a set match is written with.
//
// Under inet the rule must say which family it is matching, because the table
// handles both and the set holds only one. Under an ip or ip6 table the set's
// own family is the table's family and the rule's keyword follows it.
func addrFamily(ruleFamily, setFamily string) string {
	if ruleFamily == "inet" {
		return setFamily
	}
	return ruleFamily
}

// ports renders a port match, or "" when the protocol has none.
//
// Silently omitted rather than refused for a portless protocol: Validate already
// refuses that combination at authoring time, and a renderer that failed here
// would turn one bad stored row into a host with no firewall.
func ports(p Protocol, side string, start, end int) string {
	if p != TCP && p != UDP {
		return ""
	}
	lo, hi := PortRange(start, end)
	if lo == 0 {
		return ""
	}
	if hi > lo {
		return fmt.Sprintf("%s %d-%d", side, lo, hi)
	}
	return fmt.Sprintf("%s %d", side, lo)
}

// writeSet declares one nft set with its elements.
func writeSet(b *strings.Builder, s FirewallSet, els []FirewallSetElement) error {
	var typ string
	switch s.Family {
	case "ip":
		typ = "ipv4_addr"
	case "ip6":
		typ = "ipv6_addr"
	default:
		// inet is not a set family — see FirewallSet.Family. Refused rather than
		// guessed, because nft rejects the whole ruleset over one bad declaration.
		return fmt.Errorf("render: set %q has family %q; an nft set holds one address "+
			"family and cannot be inet", s.Name, s.Family)
	}

	fmt.Fprintf(b, "  set %s {\n", setName(s))
	fmt.Fprintf(b, "    type %s\n", typ)
	// `flags interval` is required for PREFIXES. Without it nft rejects any
	// element carrying a mask, which is most of them — a management set is
	// written as ranges far more often than as single hosts.
	fmt.Fprintf(b, "    flags interval\n")
	if len(els) > 0 {
		values := make([]string, 0, len(els))
		for _, e := range els {
			values = append(values, e.Value)
		}
		// Sorted, so an unchanged set renders identically every time. Element rows
		// come back in whatever order the database chose, and an unstable order
		// would change the digest between renders of unchanged data — every bee in
		// the hive re-applying a ruleset that did not change.
		sort.Strings(values)
		fmt.Fprintf(b, "    elements = { %s }\n", strings.Join(values, ", "))
	}
	b.WriteString("  }\n")
	return nil
}

// setName is the nft identifier for a set.
//
// Derived from the set's own name with everything nft will not accept replaced,
// because the operator-facing name is free text and an identifier is not.
func setName(s FirewallSet) string {
	var b strings.Builder
	for _, r := range s.Name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	name := b.String()
	if name == "" || (name[0] >= '0' && name[0] <= '9') {
		// nft identifiers cannot start with a digit, and an empty one is not an
		// identifier at all.
		name = "s_" + name
	}
	return name
}

// hookFor maps a chain name to its netfilter hook.
//
// Defaults to the chain's own name, which is right for the standard three, and
// keeps a custom chain from rendering as a hook nft does not know.
func hookFor(chain string) string {
	switch strings.ToLower(chain) {
	case "input", "output", "forward", "prerouting", "postrouting":
		return strings.ToLower(chain)
	default:
		return "input"
	}
}

// isInputLike reports whether a chain sees traffic arriving at this host, and so
// needs the established/loopback preamble.
//
// output and forward are excluded deliberately. Accepting established traffic on
// output is meaningless — the host originated it — and on forward it would
// accept transit the policy may exist to refuse.
func isInputLike(chain string) bool {
	c := strings.ToLower(chain)
	return c == "input" || c == "prerouting"
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedBoolKeys(m map[string]bool) []string { return sortedKeys(m) }
