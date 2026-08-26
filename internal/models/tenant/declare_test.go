package tenant

import (
	"strings"
	"testing"
)

// TestAPastedHostListIsAcceptedAsPeopleActuallyHaveIt.
//
// The list comes out of a spreadsheet, an inventory file or a terminal. Refusing
// anything but a strict format means the operator reformats it by hand, and that
// is where a typo enters a list of machines about to be connected to.
func TestAPastedHostListIsAcceptedAsPeopleActuallyHaveIt(t *testing.T) {
	t.Parallel()

	got, err := ParseHostList(`
# the new rack
pve1
pve2, pve3
10.10.10.4 10.10.10.5

pve6   # spare
`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"pve1", "pve2", "pve3", "10.10.10.4", "10.10.10.5", "pve6"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestARepeatedHostIsCollapsedRatherThanDuplicated.
//
// Two rows for one address are two workers racing to onboard one machine, and
// the unique index is not what should be reporting that to somebody who pasted a
// list with a repeat in it.
func TestARepeatedHostIsCollapsedRatherThanDuplicated(t *testing.T) {
	t.Parallel()

	got, err := ParseHostList("pve1\npve2\nPVE1\npve1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want two addresses", got)
	}
}

// TestTheShapesThatAreWrongOnTheirFaceAreRefusedWithTheFix.
//
// Each of these fails much later and much less clearly: a URL becomes an ssh
// host called "https", and host:port silently drops the port. The message names
// the corrected form, because an operator with thirty addresses needs to know
// what to change rather than that something was invalid.
func TestTheShapesThatAreWrongOnTheirFaceAreRefusedWithTheFix(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		addr, wants string
	}{
		{"https://pve1.example.com", "pve1.example.com"},
		{"http://10.10.10.2:8006", "10.10.10.2:8006"},
		{"pve1:22", "pve1"},
		{"10.10.10.2:22", "10.10.10.2"},
	} {
		err := ValidateHostAddress(tc.addr)
		if err == nil {
			t.Errorf("%q was accepted", tc.addr)
			continue
		}
		if !strings.Contains(err.Error(), tc.wants) {
			t.Errorf("refusing %q said %q, which does not name the corrected form %q",
				tc.addr, err, tc.wants)
		}
	}
}

// TestOrdinaryAddressesAreAccepted, including a bare IPv6 — which must not be
// mistaken for host:port, since it is full of colons.
func TestOrdinaryAddressesAreAccepted(t *testing.T) {
	t.Parallel()

	for _, addr := range []string{
		"pve1", "pve1.example.com", "10.10.10.2", "node-01.rack-3.dc",
		"2001:db8::1", "::1",
	} {
		if err := ValidateHostAddress(addr); err != nil {
			t.Errorf("%q was refused: %v", addr, err)
		}
	}
}

// TestAPasteThatIsNotAHostListIsRefusedRatherThanDialled.
//
// Every accepted address becomes an SSH connection attempt on the next tick, so
// a log file pasted into the box should be refused rather than turned into a
// thousand connections.
func TestAPasteThatIsNotAHostListIsRefusedRatherThanDialled(t *testing.T) {
	t.Parallel()

	if _, err := ParseHostList(""); err == nil {
		t.Fatal("empty input produced addresses")
	}
	if _, err := ParseHostList("# only a comment\n\n  \n"); err == nil {
		t.Fatal("a comment-only paste produced addresses")
	}

	var b strings.Builder
	for i := 0; i < MaxDeclaredHosts+1; i++ {
		b.WriteString("host")
		b.WriteString(strings.Repeat("x", 1))
		b.WriteString(itoa(i))
		b.WriteString("\n")
	}
	_, err := ParseHostList(b.String())
	if err == nil {
		t.Fatalf("more than %d addresses was accepted", MaxDeclaredHosts)
	}
	if !strings.Contains(err.Error(), "SSH connection attempt") {
		t.Fatalf("error %q does not say why the cap exists", err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var d []byte
	for i > 0 {
		d = append([]byte{byte('0' + i%10)}, d...)
		i /= 10
	}
	return string(d)
}
