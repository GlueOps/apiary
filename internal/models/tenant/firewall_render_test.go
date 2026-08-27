package tenant

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func aRule(name string, mut func(*FirewallRule)) FirewallRule {
	r := FirewallRule{
		Name: name, Enabled: true,
		Family: "inet", TableName: "filter", ChainName: "input",
		Protocol: TCP, Action: Accept,
		SrcCidr: "0.0.0.0/0", DstCidr: "0.0.0.0/0",
	}
	if mut != nil {
		mut(&r)
	}
	return r
}

func render(t *testing.T, in RenderInput) Rendered {
	t.Helper()
	out, err := Render(in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

// TestEveryInputChainAcceptsEstablishedBeforeAnyAuthoredRule.
//
// THE LOCKOUT TEST. A ruleset ending in a drop is the ordinary shape of a
// firewall, and applying one to a live host severs every established connection
// — including the SSH session applying it and, on a Proxmox node, the corosync
// ring that keeps the cluster quorate. Both are what "apply a default-deny
// policy" means on a machine currently being administered over the network.
//
// The position matters as much as the presence: nft evaluates in order and stops
// at the first terminal verdict, so an established-accept rendered AFTER an
// operator's drop never runs.
func TestEveryInputChainAcceptsEstablishedBeforeAnyAuthoredRule(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("default deny", func(r *FirewallRule) { r.Protocol = ProtocolAny; r.Action = Drop }),
	}})

	est := strings.Index(out.Body, acceptEstablished)
	lo := strings.Index(out.Body, acceptLoopback)
	deny := strings.Index(out.Body, "default deny")

	if est < 0 {
		t.Fatalf("no established-accept rendered; applying this severs the session that "+
			"applies it:\n%s", out.Body)
	}
	if lo < 0 {
		t.Fatalf("no loopback accept rendered; local services stop talking to themselves:\n%s",
			out.Body)
	}
	if est > deny || lo > deny {
		t.Fatalf("the preamble renders AFTER the operator's drop, so it never evaluates:\n%s",
			out.Body)
	}
}

// TestTheChainPolicyIsAcceptSoLoadingIsNotItselfADenial.
//
// A drop policy denies everything the instant the file loads, before any accept
// rule is evaluated — including the connection loading it. Deny is expressed as
// a RULE, which renders after the preamble and therefore cannot sever what is
// already established.
func TestTheChainPolicyIsAcceptSoLoadingIsNotItselfADenial(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{aRule("allow ssh", nil)}})
	if !strings.Contains(out.Body, "policy accept;") {
		t.Fatalf("chain policy is not accept:\n%s", out.Body)
	}
	if strings.Contains(out.Body, "policy drop") {
		t.Fatalf("a drop policy denies everything before the first accept is read:\n%s", out.Body)
	}
}

// TestOutputAndForwardDoNotGetTheInputPreamble. Accepting established on output
// is meaningless — the host originated it — and on forward it would accept
// transit the policy may exist to refuse.
func TestOutputAndForwardDoNotGetTheInputPreamble(t *testing.T) {
	t.Parallel()

	for _, chain := range []string{"output", "forward"} {
		out := render(t, RenderInput{Rules: []FirewallRule{
			aRule("x", func(r *FirewallRule) { r.ChainName = chain }),
		}})
		if strings.Contains(out.Body, acceptLoopback) {
			t.Fatalf("chain %s got the input preamble:\n%s", chain, out.Body)
		}
	}
}

// TestTheBodyFlushesFirst. The body is desired state: anything already loaded
// that this file does not declare must go, or a table removed from the policy
// keeps filtering forever, invisible to every tool that reads the policy from
// the control plane.
func TestTheBodyFlushesFirst(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{aRule("x", nil)}})
	flush := strings.Index(out.Body, "flush ruleset")
	table := strings.Index(out.Body, "table ")
	if flush < 0 || table < 0 || flush > table {
		t.Fatalf("the ruleset is not flushed before the tables are declared:\n%s", out.Body)
	}
}

// TestOneTableWithTwoChainsIsOneBlock.
//
// Rendered flat by (table, chain), a table with two chains emits two `table`
// blocks and declares its sets twice. nft rejects that, and it rejects the WHOLE
// file — leaving the host with no firewall rather than a duplicated line.
func TestOneTableWithTwoChainsIsOneBlock(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("in", nil),
		aRule("out", func(r *FirewallRule) { r.ChainName = "output" }),
	}})

	if n := strings.Count(out.Body, "table inet filter {"); n != 1 {
		t.Fatalf("%d table blocks for one table:\n%s", n, out.Body)
	}
	if !strings.Contains(out.Body, "chain input {") || !strings.Contains(out.Body, "chain output {") {
		t.Fatalf("both chains should be inside the one table:\n%s", out.Body)
	}
}

// TestASetIsDeclaredOncePerTableThatUsesIt, and with interval flags, without
// which nft refuses every element carrying a prefix — which is most of them.
func TestASetIsDeclaredOncePerTableThatUsesIt(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	out := render(t, RenderInput{
		Rules: []FirewallRule{
			aRule("a", func(r *FirewallRule) { r.SrcSetID = &id }),
			aRule("b", func(r *FirewallRule) { r.SrcSetID = &id; r.ChainName = "output" }),
		},
		Sets:     map[string]FirewallSet{id.String(): {ID: id, Name: "management", Family: "ip"}},
		Elements: map[string][]FirewallSetElement{id.String(): {{Value: "10.0.0.0/8"}}},
	})

	if n := strings.Count(out.Body, "set management {"); n != 1 {
		t.Fatalf("set declared %d times in one table:\n%s", n, out.Body)
	}
	if !strings.Contains(out.Body, "flags interval") {
		t.Fatalf("no interval flag, so nft refuses every prefix element:\n%s", out.Body)
	}
	if !strings.Contains(out.Body, "@management") {
		t.Fatalf("the rule does not reference the set:\n%s", out.Body)
	}
}

// TestAMissingSetIsRefusedRatherThanSkipped. nft rejects a ruleset referencing
// an undeclared set, and rejects the whole file — so dropping the rule quietly
// would produce a host with NO firewall instead of one missing a line.
func TestAMissingSetIsRefusedRatherThanSkipped(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	_, err := Render(RenderInput{Rules: []FirewallRule{
		aRule("a", func(r *FirewallRule) { r.SrcSetID = &id }),
	}})
	if err == nil {
		t.Fatal("a rule referencing an absent set rendered without complaint")
	}
	if !strings.Contains(err.Error(), "no firewall") {
		t.Fatalf("error %q does not say what the consequence would be", err)
	}
}

// TestAnInetSetIsRefused. An nft set holds one address family; declaring one
// inet is a declaration nft rejects, taking the whole ruleset with it.
func TestAnInetSetIsRefused(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	_, err := Render(RenderInput{
		Rules: []FirewallRule{aRule("a", func(r *FirewallRule) { r.SrcSetID = &id })},
		Sets:  map[string]FirewallSet{id.String(): {ID: id, Name: "x", Family: "inet"}},
	})
	if err == nil {
		t.Fatal("an inet set was rendered")
	}
}

// TestRenderingIsStableForUnchangedInput.
//
// The digest decides whether a bee re-applies. An unstable render — set elements
// in database order, a map iterated — changes it between renders of unchanged
// data, and every host in the fleet re-applies a ruleset that did not change.
func TestRenderingIsStableForUnchangedInput(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	in := RenderInput{
		Rules: []FirewallRule{aRule("a", func(r *FirewallRule) { r.SrcSetID = &id })},
		Sets:  map[string]FirewallSet{id.String(): {ID: id, Name: "m", Family: "ip"}},
		Elements: map[string][]FirewallSetElement{id.String(): {
			{Value: "10.0.0.1"}, {Value: "10.0.0.2"}, {Value: "10.0.0.3"},
		}},
	}
	first := render(t, in)
	for i := 0; i < 20; i++ {
		if got := render(t, in); got.Digest != first.Digest {
			t.Fatalf("digest moved between renders of identical input: %s then %s",
				first.Digest, got.Digest)
		}
	}

	// And element ORDER must not matter, since the database returns rows in
	// whatever order it chose.
	in.Elements[id.String()] = []FirewallSetElement{
		{Value: "10.0.0.3"}, {Value: "10.0.0.1"}, {Value: "10.0.0.2"},
	}
	if got := render(t, in); got.Digest != first.Digest {
		t.Fatalf("reordering the same elements changed the digest, so every host would "+
			"re-apply an unchanged ruleset: %s then %s", first.Digest, got.Digest)
	}
}

// TestDisabledRulesDoNotRender, which is the whole meaning of the flag.
func TestDisabledRulesDoNotRender(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("on", nil),
		aRule("off", func(r *FirewallRule) { r.Enabled = false }),
	}})
	if strings.Contains(out.Body, `"off"`) {
		t.Fatalf("a disabled rule rendered:\n%s", out.Body)
	}
	if !strings.Contains(out.Body, `"on"`) {
		t.Fatalf("an enabled rule did not render:\n%s", out.Body)
	}
}

// TestRuleOrderIsPreservedExactly. Preview already ordered these, and rendering
// must not re-sort: an order decided in two places stops agreeing, and here the
// consequence is an operator reading a different policy from the one enforced.
func TestRuleOrderIsPreservedExactly(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("zebra", nil), aRule("alpha", nil), aRule("middle", nil),
	}})
	z, a, m := strings.Index(out.Body, "zebra"), strings.Index(out.Body, "alpha"),
		strings.Index(out.Body, "middle")
	if z >= a || a >= m {
		t.Fatalf("rules were re-sorted:\n%s", out.Body)
	}
}

// TestProtocolAnyMatchesEverything. `meta l4proto { tcp, udp }` would EXCLUDE
// icmp, which is the opposite of "any" and a near-miss nobody reads twice.
func TestProtocolAnyMatchesEverything(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("all from mgmt", func(r *FirewallRule) {
			r.Protocol = ProtocolAny
			r.SrcCidr = "10.0.0.0/8"
		}),
	}})
	if strings.Contains(out.Body, "l4proto {") {
		t.Fatalf("`any` rendered as a protocol list, which excludes everything not in it:\n%s",
			out.Body)
	}
	if !strings.Contains(out.Body, "ip saddr 10.0.0.0/8 accept") {
		t.Fatalf("`any` did not render as an unqualified match:\n%s", out.Body)
	}
}

// TestIcmpUnderInetCoversV6. A rule whose whole purpose is to allow ping would
// otherwise silently not match v6 ping.
func TestIcmpUnderInetCoversV6(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("ping", func(r *FirewallRule) { r.Protocol = ICMP }),
	}})
	if !strings.Contains(out.Body, "icmpv6") {
		t.Fatalf("icmp under inet does not cover v6:\n%s", out.Body)
	}
}

// TestPortRangesRender, since one rule per port is how a chain grows to hundreds
// of lines nobody can read.
func TestPortRangesRender(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{
		aRule("corosync", func(r *FirewallRule) {
			r.Protocol = UDP
			r.DstPort, r.DstPortEnd = 5405, 5412
		}),
		aRule("ssh", func(r *FirewallRule) { r.DstPort = 22 }),
	}})
	if !strings.Contains(out.Body, "dport 5405-5412") {
		t.Fatalf("no port range:\n%s", out.Body)
	}
	if !strings.Contains(out.Body, "dport 22") {
		t.Fatalf("no single port:\n%s", out.Body)
	}
}

// TestAnAnyAddressIsOmittedRatherThanRendered, so the common rule stays readable
// instead of padding every line with 0.0.0.0/0.
func TestAnAnyAddressIsOmittedRatherThanRendered(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{Rules: []FirewallRule{aRule("ssh", func(r *FirewallRule) {
		r.DstPort = 22
	})}})
	if strings.Contains(out.Body, "0.0.0.0/0") {
		t.Fatalf("an any-address was rendered explicitly:\n%s", out.Body)
	}
}

// TestASetNameBecomesAValidIdentifier. Operator-facing names are free text; nft
// identifiers are not, and an invalid one is rejected along with the whole file.
func TestASetNameBecomesAValidIdentifier(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	out := render(t, RenderInput{
		Rules: []FirewallRule{aRule("a", func(r *FirewallRule) { r.SrcSetID = &id })},
		Sets:  map[string]FirewallSet{id.String(): {ID: id, Name: "prod jump-hosts (eu)", Family: "ip"}},
	})
	if !strings.Contains(out.Body, "set prod_jump_hosts__eu_ {") {
		t.Fatalf("set name was not made into an identifier:\n%s", out.Body)
	}
}

// TestAnEmptyPolicyRendersAnEmptyBodyRatherThanFailing.
//
// Nothing bound is a legitimate state, and it must not render as a file that
// drops everything. The publish path treats an empty body as "nothing to
// enforce"; a renderer that refused would make the no-bindings case an error an
// operator has to work around.
func TestAnEmptyPolicyRendersAnEmptyBodyRatherThanFailing(t *testing.T) {
	t.Parallel()

	out := render(t, RenderInput{})
	if strings.Contains(out.Body, "drop") {
		t.Fatalf("an empty policy rendered a denial:\n%s", out.Body)
	}
	if out.Digest == "" {
		t.Fatal("an empty policy has no digest, so convergence cannot be compared")
	}
}

// The shape a set exists for: one list, referenced by many rules.
//
// Written from an operator's own worked example, because the point of sets is
// not the syntax — it is the arithmetic. Four management addresses across two
// ports is TWO rules and ONE set. Expressed with per-rule literals, which is all
// the API can currently produce, it is EIGHT rules, and every address added to
// the whitelist adds another two.
//
// This renders today. Sets, elements, @name references and the either/or
// validation are all present in the model; what is missing is any way for an
// operator to create a set, add an element to it, or point a rule at one. This
// test is the target the API has to be able to produce.
func TestRenderingASetReferencedByManyRules(t *testing.T) {
	setID := uuid.New()
	allowed := FirewallSet{
		ID: setID, Name: "allowed_ips", Family: "ip",
		Purpose: SetPurposeManagement,
	}
	elements := []FirewallSetElement{
		{SetID: setID, Value: "192.0.2.10"},
		{SetID: setID, Value: "192.0.2.20"},
		{SetID: setID, Value: "198.51.100.5"},
		{SetID: setID, Value: "10.10.10.0/24"},
	}

	rule := func(name string, port int, action Action, useSet bool) FirewallRule {
		r := FirewallRule{
			Name: name, Enabled: true,
			Family: "inet", TableName: "filter", ChainName: "input",
			Protocol: TCP, Action: action, DstPort: port,
		}
		if useSet {
			r.SrcSetID = &setID
		}
		return r
	}

	out := render(t, RenderInput{
		Rules: []FirewallRule{
			rule("ssh from the whitelist", 22, Accept, true),
			rule("api from the whitelist", 8006, Accept, true),
			rule("ssh otherwise", 22, Drop, false),
			rule("api otherwise", 8006, Drop, false),
		},
		Sets:     map[string]FirewallSet{setID.String(): allowed},
		Elements: map[string][]FirewallSetElement{setID.String(): elements},
	})

	// The set is declared once, inside the table that references it — nft scopes
	// a set to its table, so a set declared elsewhere is a load error.
	for _, want := range []string{
		"set allowed_ips",
		"192.0.2.10",
		"192.0.2.20",
		"198.51.100.5",
		"10.10.10.0/24",
	} {
		if !strings.Contains(out.Body, want) {
			t.Fatalf("the rendered set must contain %q:\n%s", want, out.Body)
		}
	}

	// Both accept rules reference it rather than repeating the addresses.
	if n := strings.Count(out.Body, "@allowed_ips"); n != 2 {
		t.Fatalf("both accept rules must reference the set, got %d references:\n%s", n, out.Body)
	}

	// The arithmetic: the addresses appear in the set declaration and nowhere
	// else. A literal per rule is what this exists to avoid.
	if n := strings.Count(out.Body, "198.51.100.5"); n != 1 {
		t.Fatalf("an address must appear once, in the set — found %d:\n%s", n, out.Body)
	}
}
