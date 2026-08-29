package fleetops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func knownHostsFixture(t *testing.T, existing string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "known_hosts")
	if existing != "" {
		if err := os.WriteFile(p, []byte(existing), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	old := KnownHostsPath
	KnownHostsPath = p
	t.Cleanup(func() { KnownHostsPath = old })
	return p
}

// Pinned under EVERY name the join may use. known_hosts is keyed on the string
// used to connect, and pvecm resolves node names internally as well as using
// the address it was given.
func TestPinPeerWritesEveryName(t *testing.T) {
	p := knownHostsFixture(t, "")
	if err := PinPeer(context.Background(),
		[]string{"10.0.0.1", "pve1"}, []string{"ssh-ed25519 AAAAC3Nz"}); err != nil {
		t.Fatalf("pin: %v", err)
	}
	body, _ := os.ReadFile(p)
	for _, want := range []string{"10.0.0.1 ssh-ed25519 AAAAC3Nz", "pve1 ssh-ed25519 AAAAC3Nz"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("known_hosts = %q, missing %q", body, want)
		}
	}
}

// A re-pin REPLACES. A stale entry left beside a fresh one makes ssh refuse on a
// conflict, which is the same failure this exists to prevent.
func TestPinPeerReplacesRatherThanAccumulates(t *testing.T) {
	p := knownHostsFixture(t, "pve1 ssh-ed25519 OLDKEY\nother ssh-ed25519 KEEPME\n")
	if err := PinPeer(context.Background(),
		[]string{"pve1"}, []string{"ssh-ed25519 NEWKEY"}); err != nil {
		t.Fatalf("pin: %v", err)
	}
	body, _ := os.ReadFile(p)
	if strings.Contains(string(body), "OLDKEY") {
		t.Fatalf("the stale entry survived: %q", body)
	}
	if !strings.Contains(string(body), "NEWKEY") {
		t.Fatalf("the new key was not written: %q", body)
	}
	// Another host's entry must not be collateral damage.
	if !strings.Contains(string(body), "other ssh-ed25519 KEEPME") {
		t.Fatalf("an unrelated entry was removed: %q", body)
	}
}

// Refused rather than silently skipped. Joining without pinning means trusting
// whatever answers on the address, with the corosync auth key as the prize.
func TestPinPeerRefusesWithNoKeys(t *testing.T) {
	knownHostsFixture(t, "")
	if err := PinPeer(context.Background(), []string{"pve1"}, nil); err == nil {
		t.Fatal("pinning with no host keys was accepted")
	}
}

// A member with no captured host keys is not a usable join target, for the same
// reason as one with no fingerprint.
func TestMembersWithoutHostKeysAreNotJoinTargets(t *testing.T) {
	d := Desired{Cluster: "lab-a", Self: "pve1", Members: []DesiredMember{
		{Node: "pve2", Address: "10.0.0.2", Fingerprint: "AA:BB", HostKeys: []string{"ssh-ed25519 K"}},
		{Node: "pve3", Address: "10.0.0.3", Fingerprint: "AA:BB"},
	}}
	got := d.joinTargets()
	if len(got) != 1 || got[0].Node != "pve2" {
		t.Fatalf("join targets = %+v, want just pve2", got)
	}
}

// PVE symlinks these files onto pmxcfs — observed live as
// /etc/ssh/ssh_known_hosts -> /etc/pve/priv/known_hosts. Writing through such a
// link is a cluster-wide, quorum-gated write to a store the next join replaces
// wholesale, and this runs on a host that is about to join.
//
// The danger is that every ordinary file call follows a symlink silently:
// os.WriteFile opens O_WRONLY|O_CREATE|O_TRUNC, so not checking does not produce
// a refused write, it rewrites every node's known_hosts at once.
func TestPinPeerRefusesToWriteThroughAPmxcfsSymlink(t *testing.T) {
	dir := t.TempDir()
	// Stand in for /etc/pve, which does not exist on the machine running tests.
	pve := filepath.Join(dir, "pve")
	if err := os.MkdirAll(pve, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(pve, "known_hosts")
	if err := os.WriteFile(target, []byte("original\n"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "known_hosts")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	old := KnownHostsPath
	KnownHostsPath = link
	t.Cleanup(func() { KnownHostsPath = old })

	// The guard keys on /etc/pve, so point it at the stand-in for this test.
	oldPrefix := pmxcfsPrefix
	// Resolved, because EvalSymlinks returns a fully-resolved path and the
	// test's temp dir is itself behind a symlink on some platforms (/var ->
	// /private/var on darwin). Comparing an unresolved prefix against a resolved
	// target silently never matches, which would make this test pass for the
	// wrong reason if the guard were removed.
	resolvedPVE, err := filepath.EvalSymlinks(pve)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	pmxcfsPrefix = resolvedPVE + string(filepath.Separator)
	t.Cleanup(func() { pmxcfsPrefix = oldPrefix })

	if perr := PinPeer(context.Background(), []string{"pve1"}, []string{"ssh-ed25519 K"}); perr == nil {
		t.Fatal("wrote through a symlink into the cluster's replicated store")
	}
	body, _ := os.ReadFile(target)
	if string(body) != "original\n" {
		t.Fatalf("the replicated file was modified: %q", body)
	}
}

// A plain local file is the ordinary case — a joiner is always standalone, and
// that is the state where this file is real and local.
func TestPinPeerAcceptsAPlainLocalFile(t *testing.T) {
	knownHostsFixture(t, "pve1 ssh-ed25519 OLD\n")
	if err := PinPeer(context.Background(),
		[]string{"pve1"}, []string{"ssh-ed25519 NEW"}); err != nil {
		t.Fatalf("pin: %v", err)
	}
}
