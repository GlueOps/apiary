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
	if MachineIDDigest(raw) != MachineIDDigest(raw) {
		t.Fatal("the digest must be deterministic, or nothing can ever look a machine up")
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
	// clustered case is covered by TestMatchMachineByAddressRefusesACrossClusterAdoption.
	one.HiveID = nil
	two.HiveID = nil

	if m := MatchMachineByAddress([]Hypervisor{one}, "hv7.example.com"); m == nil || m.ID != one.ID {
		t.Fatal("a single discovered node matching the enrolling host's name must be adopted")
	}
	if m := MatchMachineByAddress([]Hypervisor{one, two}, "hv7.example.com"); m != nil {
		t.Fatalf("two discovered nodes with the same name must adopt NEITHER, got %s", m.ID)
	}

	// A machine that already carries a machine id is a DIFFERENT machine: it has
	// proved its own identity, and this fallback must never take a record away
	// from it on the strength of a name.
	other := "some-other-digest"
	claimed := named("hv7", &other)
	if m := MatchMachineByAddress([]Hypervisor{claimed}, "hv7.example.com"); m != nil {
		t.Fatal("a machine that already has a machine id must never be adopted by a different " +
			"one; the name is weaker evidence than the identity already on the row")
	}

	if m := MatchMachineByAddress([]Hypervisor{one}, "10.0.0.7"); m != nil {
		t.Fatal("an IP address yields no node name, so it must adopt nothing rather than " +
			"matching on the first octet")
	}
}

func TestMatchMachineByAddressRefusesACrossClusterAdoption(t *testing.T) {
	// The hole the ambiguity rule does not close. `hv5.dc2.example.com` truncates
	// to `hv5`, and when the only discovered `hv5` is in dc1 that is UNAMBIGUOUS
	// by name — so it adopted, and the agent's credential landed on the record for
	// a physically different host in another cluster.
	//
	// The domain is evidence the candidate cannot corroborate. Refusing costs a
	// duplicate row an operator can merge, which this package prefers to a wrong
	// adoption.
	dc1 := uuid.New()
	name := "hv5"
	candidates := []Hypervisor{{ID: uuid.New(), HiveID: &dc1, Name: &name}}

	if m := MatchMachineByAddress(candidates, "hv5.dc2.example.com"); m != nil {
		t.Fatal("a fully qualified address must not adopt a machine matched on its first " +
			"label alone — that is how an agent lands on a host in another cluster")
	}
	// A bare node name still adopts: that carries no contradicting evidence.
	if m := MatchMachineByAddress(candidates, "hv5"); m == nil {
		t.Fatal("a bare node name must still adopt")
	}
}

func TestMatchMachineByAddressRequiresARecordedAddressToAgree(t *testing.T) {
	// A machine onboarded by address keeps it, so when one is present it is the
	// strongest corroboration available.
	name := "hv5"
	candidates := []Hypervisor{
		{ID: uuid.New(), Name: &name, Address: "10.0.0.9"},
	}
	if m := MatchMachineByAddress(candidates, "hv5"); m != nil {
		t.Fatal("a candidate whose recorded address disagrees must not be adopted")
	}
	byAddr := []Hypervisor{{ID: uuid.New(), Name: &name, Address: "hv5"}}
	if m := MatchMachineByAddress(byAddr, "hv5"); m == nil {
		t.Fatal("a candidate whose recorded address agrees must be adopted")
	}
}
