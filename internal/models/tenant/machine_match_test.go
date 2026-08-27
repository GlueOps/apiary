package tenant

import (
	"testing"

	"github.com/google/uuid"
)

func TestMachineIDDigestIsStableAndTrimmed(t *testing.T) {
	// Two callers store and look up by this value — enrolment writes it, the
	// onboarding job's proof-of-success reads it — and they used to hash
	// separately. The lookup could then never match, so every onboard ended in
	// "no heartbeat" however healthy the host.
	raw := "9f1c2d3e4b5a6789"
	// Two calls, held apart in variables: comparing the expression with itself
	// reads as a tautology to any reader and to static analysis, which is exactly
	// what it looked like. This asks the real question — twice over the same
	// input, same answer.
	first, second := MachineIDDigest(raw), MachineIDDigest(raw)
	if first != second {
		t.Fatal("the digest must be deterministic, or nothing can ever look a machine up")
	}
	if first == "" {
		t.Fatal("the digest is empty, so every machine would hash to the same key and " +
			"enrolment would adopt the first row it found")
	}
	if MachineIDDigest("  "+raw+"\n") != MachineIDDigest(raw) {
		t.Fatal("whitespace must be trimmed before hashing: an agent that reads /etc/machine-id " +
			"with a trailing newline would otherwise enrol as a different machine every time " +
			"the reader changed")
	}
	if got := MachineIDDigest(raw); len(got) != 64 {
		t.Fatalf("digest length = %d, want 64 hex characters — the column is varchar(64)", len(got))
	}
	if MachineIDDigest(raw) == raw {
		t.Fatal("the raw machine id was stored verbatim; a database is not the place to keep " +
			"an inventory of somebody's hosts in clear")
	}
}

func TestNodeNameFromAddressRefusesIPLiterals(t *testing.T) {
	// The dangerous case. A first label is only a node name when the address is a
	// NAME: taking one off 10.0.0.5 yields "10", which every other host on the
	// same /8 also yields — so two unrelated machines would look like one and the
	// merge would fold them into a single record.
	for _, addr := range []string{
		"10.0.0.5", "10.0.0.5:22", "192.0.2.1", "::1", "[2001:db8::1]:8006", "2001:db8::1",
	} {
		if got := NodeNameFromAddress(addr); got != "" {
			t.Errorf("NodeNameFromAddress(%q) = %q, want \"\": an address that is not a name "+
				"carries no node name, and guessing one merges unrelated machines", addr, got)
		}
	}
}

func TestNodeNameFromAddressTakesTheFirstLabel(t *testing.T) {
	cases := map[string]string{
		"hv5.dc1.example.com":      "hv5",
		"hv5.dc1.example.com:22":   "hv5",
		"hv5":                      "hv5",
		"  hv5.example.com  ":      "hv5",
		"":                         "",
		"HV5.example.com":          "HV5",
		"hv-05.dc1.example.com":    "hv-05",
		"hv5.dc1.example.com:8006": "hv5",
	}
	for addr, want := range cases {
		if got := NodeNameFromAddress(addr); got != want {
			t.Errorf("NodeNameFromAddress(%q) = %q, want %q", addr, got, want)
		}
	}
}

func unclustered(address string) Hypervisor {
	return Hypervisor{ID: uuid.New(), Address: address}
}

func TestMatchMachineByNodeNameRefusesAmbiguity(t *testing.T) {
	// Ambiguity is a REFUSAL, not a best guess. Two unclustered machines whose
	// addresses share a first label give no evidence which one joined the cluster,
	// and attaching a cluster's capacity — and later fleet operations — to the
	// wrong physical host is far worse than a duplicate record an operator can
	// merge deliberately.
	a := unclustered("hv5.dc1.example.com")
	b := unclustered("hv5.dc2.example.com")

	if m := MatchMachineByNodeName([]Hypervisor{a}, "hv5"); m == nil || m.ID != a.ID {
		t.Fatal("a single unambiguous candidate must be adopted")
	}
	if m := MatchMachineByNodeName([]Hypervisor{a, b}, "hv5"); m != nil {
		t.Fatalf("two equally plausible candidates must adopt NEITHER, got %s", m.ID)
	}
	if m := MatchMachineByNodeName([]Hypervisor{a}, "hv6"); m != nil {
		t.Fatal("a node name nothing matches must adopt nothing")
	}
	if m := MatchMachineByNodeName([]Hypervisor{a}, ""); m != nil {
		t.Fatal("an empty node name must match nothing: it is the value an IP address and an " +
			"absent address both produce, and it would otherwise adopt the first machine in " +
			"the list")
	}
}

func TestMatchMachineByNodeNameIgnoresClusteredMachines(t *testing.T) {
	// Adoption is only ever of a machine in NO cluster. A machine already in one
	// has its own (hive_id, name) identity, and moving it because another
	// cluster's node happens to share a first label would evict it from a fleet it
	// is a voting member of.
	hive := uuid.New()
	name := "hv5"
	clustered := Hypervisor{ID: uuid.New(), Address: "hv5.dc1.example.com", HiveID: &hive, Name: &name}
	if m := MatchMachineByNodeName([]Hypervisor{clustered}, "hv5"); m != nil {
		t.Fatal("a machine that is already in a cluster must never be adopted into another")
	}
}

func TestMatchMachineByAddressRefusesAmbiguityAndKnownMachines(t *testing.T) {
	// The enrolment side of the same merge, with the same refusal.
	named := func(name string, machineID *string) Hypervisor {
		n := name
		hive := uuid.New()
		return Hypervisor{ID: uuid.New(), HiveID: &hive, Name: &n, MachineID: machineID}
	}
	one := named("hv7", nil)
	two := named("hv7", nil)
	// Not yet in a cluster: there is no cluster for a domain to contradict, which
	// is what makes adoption from a fully qualified address safe here. The
	// clustered case is covered by TestTwoClustersWithTheSameNodeNameAdoptNeither.
	one.HiveID = nil
	two.HiveID = nil

	if m := MatchMachineByAddress([]Hypervisor{one}, "hv7.example.com", ""); m == nil || m.ID != one.ID {
		t.Fatal("a single discovered node matching the enrolling host's name must be adopted")
	}
	if m := MatchMachineByAddress([]Hypervisor{one, two}, "hv7.example.com", ""); m != nil {
		t.Fatalf("two discovered nodes with the same name must adopt NEITHER, got %s", m.ID)
	}

	// A machine that already carries a machine id is a DIFFERENT machine: it has
	// proved its own identity, and this fallback must never take a record away
	// from it on the strength of a name.
	other := "some-other-digest"
	claimed := named("hv7", &other)
	if m := MatchMachineByAddress([]Hypervisor{claimed}, "hv7.example.com", ""); m != nil {
		t.Fatal("a machine that already has a machine id must never be adopted by a different " +
			"one; the name is weaker evidence than the identity already on the row")
	}

	if m := MatchMachineByAddress([]Hypervisor{one}, "10.0.0.7", ""); m != nil {
		t.Fatal("an IP address yields no node name, so it must adopt nothing rather than " +
			"matching on the first octet")
	}
}

// The discovered-node adoption is the whole point of this fallback, and it has
// to work from a fully qualified address because that is what agents enrol with.
//
// A guard on "the enrolling address looks qualified" was tried here and
// reverted: it refused exactly this case — the common one — to narrow a rarer
// one, so every agent enrolling on a node the capacity sync had already found
// would have created a duplicate record.
func TestMatchMachineByAddressAdoptsADiscoveredNode(t *testing.T) {
	dc1 := uuid.New()
	name := "hv5"
	// What the capacity sync leaves behind: a hive and a node name, no machine
	// id, no address. Proxmox's /cluster/resources reports nothing else.
	candidates := []Hypervisor{{ID: uuid.New(), HiveID: &dc1, Name: &name}}

	if m := MatchMachineByAddress(candidates, "hv5.dc1.example.com", ""); m == nil {
		t.Fatal("an agent enrolling on a node the sync already discovered adopted nothing, " +
			"so enrolment creates a second record for one machine — one carrying the " +
			"cluster membership and capacity, the other the agent, and neither the machine")
	}
	if m := MatchMachineByAddress(candidates, "hv5", ""); m == nil {
		t.Fatal("a bare node name must adopt too")
	}
}

// The residual cross-cluster risk, pinned as the shape it actually has rather
// than as a guard that does not exist.
//
// `hv5.dc2.example.com` truncating to `hv5` and adopting dc1's `hv5` is possible
// ONLY while exactly one cluster holds that node name. The moment the second
// cluster is discovered there are two candidates and the ambiguity rule refuses
// both — which is the property worth asserting, because it is what bounds the
// window. Closing the rest needs evidence the model does not carry: the machine
// id the sync cannot report, or a hive recorded against a DNS domain.
func TestTwoClustersWithTheSameNodeNameAdoptNeither(t *testing.T) {
	dc1, dc2 := uuid.New(), uuid.New()
	n1, n2 := "hv5", "hv5"
	candidates := []Hypervisor{
		{ID: uuid.New(), HiveID: &dc1, Name: &n1},
		{ID: uuid.New(), HiveID: &dc2, Name: &n2},
	}
	if m := MatchMachineByAddress(candidates, "hv5.dc2.example.com", ""); m != nil {
		t.Fatalf("adopted %s — with the same node name in two clusters there is no evidence "+
			"which one enrolled, and attaching an agent's credential to the wrong physical "+
			"host is worse than a duplicate row an operator can merge", m.ID)
	}
}

func TestMatchMachineByAddressRequiresARecordedAddressToAgree(t *testing.T) {
	// A machine onboarded by address keeps it, so when one is present it is the
	// strongest corroboration available.
	name := "hv5"
	candidates := []Hypervisor{
		{ID: uuid.New(), Name: &name, Address: "10.0.0.9"},
	}
	if m := MatchMachineByAddress(candidates, "hv5", ""); m != nil {
		t.Fatal("a candidate whose recorded address disagrees must not be adopted")
	}
	byAddr := []Hypervisor{{ID: uuid.New(), Name: &name, Address: "hv5"}}
	if m := MatchMachineByAddress(byAddr, "hv5", ""); m == nil {
		t.Fatal("a candidate whose recorded address agrees must be adopted")
	}
}

// A discovered node now carries the address the CLUSTER knows it by, which is
// an IP — while an agent calls itself by hostname. Matching those two against
// each other rejected every adoption this function exists to perform, and the
// second record appeared on a live cluster before this test existed.
//
// The peer address is what closes it: it is where the enrolment request
// demonstrably arrived from, so a recorded address agreeing with IT is evidence
// of the same machine just as much as one agreeing with the reported name.
func TestMatchAdoptsWhenTheRecordedAddressIsTheClusterIP(t *testing.T) {
	name := "pve4"
	discovered := Hypervisor{ID: uuid.New(), Name: &name, Address: "10.10.10.5"}

	// The agent calls itself pve4 and its request came from 10.10.10.5.
	if m := MatchMachineByAddress([]Hypervisor{discovered}, "pve4", "10.10.10.5"); m == nil {
		t.Fatal("an agent whose request came from the recorded address must adopt that record")
	}

	// Same host, no peer address known: the recorded IP cannot be corroborated,
	// so this must NOT adopt on the name alone — the address is evidence that has
	// to agree, not decoration to ignore.
	if m := MatchMachineByAddress([]Hypervisor{discovered}, "pve4", ""); m != nil {
		t.Fatal("a recorded address that agrees with nothing the agent is must not adopt")
	}

	// A different machine that happens to share the node name must not take it.
	if m := MatchMachineByAddress([]Hypervisor{discovered}, "pve4", "10.10.10.99"); m != nil {
		t.Fatal("a peer address disagreeing with the recorded one must not adopt")
	}
}

// A host DECLARED by an operator has an address and no name — nothing names it
// until a capacity sync sees it in a cluster. Requiring a name meant the agent
// that onboarding had just installed matched nothing and created a second
// record, every time. Reproduced from a clean tenant: five hosts declared,
// onboarded and enrolled produced ten rows.
func TestMatchAdoptsADeclaredHostThatHasNoNameYet(t *testing.T) {
	declared := Hypervisor{ID: uuid.New(), Address: "10.10.10.2"}

	// The agent calls itself "pve"; its request arrived from the declared address.
	if m := MatchMachineByAddress([]Hypervisor{declared}, "pve", "10.10.10.2"); m == nil {
		t.Fatal("a declared host must be adopted by the agent onboarding installed on it")
	}

	// A different host must not take it.
	if m := MatchMachineByAddress([]Hypervisor{declared}, "pve9", "10.10.10.99"); m != nil {
		t.Fatal("an agent from another address must not adopt this record")
	}

	// Nothing to match on at all is not a licence to match everything.
	empty := Hypervisor{ID: uuid.New()}
	if m := MatchMachineByAddress([]Hypervisor{empty}, "pve", "10.10.10.2"); m != nil {
		t.Fatal("a record with neither name nor address must never be adopted")
	}
}

// When a record carries BOTH, both have to agree. Adopting on half the evidence
// is how an agent's identity lands on the wrong host.
func TestMatchRefusesWhenOnlyHalfTheEvidenceAgrees(t *testing.T) {
	name := "pve4"
	both := Hypervisor{ID: uuid.New(), Name: &name, Address: "10.10.10.5"}

	if m := MatchMachineByAddress([]Hypervisor{both}, "pve4", "10.10.10.5"); m == nil {
		t.Fatal("both agreeing must adopt")
	}
	if m := MatchMachineByAddress([]Hypervisor{both}, "pve4", "10.10.10.99"); m != nil {
		t.Fatal("name agrees but address does not: must refuse")
	}
	if m := MatchMachineByAddress([]Hypervisor{both}, "pve9", "10.10.10.5"); m != nil {
		t.Fatal("address agrees but name does not: must refuse")
	}
}
