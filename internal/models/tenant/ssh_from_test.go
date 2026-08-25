package tenant

import "testing"

func mgmtSet(values ...string) FirewallSet {
	s := FirewallSet{Name: "mgmt", Family: FamilyIPv4, Purpose: SetPurposeManagement}
	for _, v := range values {
		s.Elements = append(s.Elements, FirewallSetElement{Value: v})
	}
	return s
}

func peerSet(values ...string) FirewallSet {
	s := FirewallSet{Name: "peers", Family: FamilyIPv4, Purpose: SetPurposePeers, Managed: true}
	for _, v := range values {
		s.Elements = append(s.Elements, FirewallSetElement{Value: v})
	}
	return s
}

// THE reason purpose is declared rather than inferred.
//
// Corosync peers are permitted on port 22 so pvecm can distribute cluster
// configuration. Deriving `from=` by scanning rules that open 22 would therefore
// list every hypervisor in the fleet, making a stolen apiary private key usable
// from any one of them — one compromised node becoming a foothold against all
// its siblings, which is exactly what from= is bought to prevent.
func TestPeersNeverReachTheFromList(t *testing.T) {
	f := DeriveFromRestriction([]FirewallSet{
		mgmtSet("194.36.65.187"),
		peerSet("142.249.164.2", "142.249.164.3", "10.62.0.2", "10.62.0.3"),
	})

	if got := f.Render(); got != `from="194.36.65.187/32"` {
		t.Fatalf("got %q — a peer address in this list is a stolen key with somewhere to be used from", got)
	}
}

// A set with no stated purpose is not evidence that the control plane connects
// from it, so the safe default must be exclusion.
func TestGeneralSetsAreExcluded(t *testing.T) {
	general := mgmtSet("10.9.9.9")
	general.Purpose = SetPurposeGeneral

	f := DeriveFromRestriction([]FirewallSet{general, mgmtSet("194.36.65.187")})
	if got := f.Render(); got != `from="194.36.65.187/32"` {
		t.Fatalf("a general-purpose set leaked into the list: %q", got)
	}
}

// The output is consolidated, because real management lists are enumerated per
// host and per address, and a clause nobody can read is one nobody audits.
func TestFromListIsConsolidated(t *testing.T) {
	var addrs []string
	for i := 2; i <= 9; i++ {
		addrs = append(addrs, "10.62.0."+string(rune('0'+i)))
	}
	f := DeriveFromRestriction([]FirewallSet{mgmtSet(addrs...)})

	if len(f.Sources) >= len(addrs) {
		t.Fatalf("not consolidated: %d addresses -> %d entries", len(addrs), len(f.Sources))
	}
	if !f.Safe() {
		t.Fatal("a populated management set produced an unsafe restriction")
	}
}

// Stable output, so an unchanged management set renders an unchanged file. An
// unstable render rewrites authorized_keys on every convergence and makes "did
// this change" unanswerable.
func TestFromListIsStable(t *testing.T) {
	a := DeriveFromRestriction([]FirewallSet{mgmtSet("10.0.0.2", "10.0.0.1")})
	b := DeriveFromRestriction([]FirewallSet{mgmtSet("10.0.0.1", "10.0.0.2")})
	if a.Render() != b.Render() {
		t.Fatalf("input order changed the render: %q vs %q", a.Render(), b.Render())
	}
}

// A wildcard in a MANAGEMENT set means the control plane may connect from
// anywhere, so a clause would restrict nothing while appearing to.
func TestWildcardInManagementDisablesTheClause(t *testing.T) {
	for _, wild := range []string{"0.0.0.0/0", "::/0"} {
		t.Run(wild, func(t *testing.T) {
			f := DeriveFromRestriction([]FirewallSet{mgmtSet("10.0.0.1", wild)})
			if !f.Unrestricted || f.Render() != "" {
				t.Fatalf("wildcard did not disable the clause: %+v", f)
			}
			if !f.Safe() {
				t.Fatal("unrestricted must be safe — it means no clause, not no access")
			}
		})
	}
}

// The lockout guard. An empty list is not "allow nothing" — as a from= clause it
// is a key that works from nowhere, costing the control plane its only way to
// repair a broken agent.
func TestNoManagementSetIsUnsafe(t *testing.T) {
	f := DeriveFromRestriction([]FirewallSet{peerSet("142.249.164.2")})
	if f.Unrestricted || len(f.Sources) != 0 {
		t.Fatalf("expected an empty restricted result, got %+v", f)
	}
	if f.Safe() {
		t.Fatal("an empty from= list reported as safe; writing it leaves no way back in")
	}
}

// Several management sets union, because an operator may keep office and CI
// ranges apart without either being less legitimate.
func TestManagementSetsUnion(t *testing.T) {
	f := DeriveFromRestriction([]FirewallSet{
		mgmtSet("194.36.65.187"),
		mgmtSet("192.168.10.60"),
	})
	if len(f.Sources) != 2 {
		t.Fatalf("got %v, want both management sets", f.Sources)
	}
}
