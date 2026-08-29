package fleetops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func authKeysFixture(t *testing.T, existing string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "authorized_keys")
	if existing != "" {
		if err := os.WriteFile(p, []byte(existing), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	old := ClusterAuthorizedKeys
	ClusterAuthorizedKeys = p
	t.Cleanup(func() { ClusterAuthorizedKeys = old })
	return p
}

// APPEND-ONLY. This file is PVE's, it carries every node's key and an operator's
// own, and rewriting it wholesale to admit one machine would lock people out of
// the cluster.
func TestAdmitKeysAppendsAndNeverRewrites(t *testing.T) {
	p := authKeysFixture(t, "ssh-ed25519 OPERATOR alice@laptop\nssh-ed25519 EXISTING root@pve\n")

	n, err := AdmitKeys(context.Background(), []string{"ssh-ed25519 NEWJOINER root@pve2"})
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if n != 1 {
		t.Fatalf("admitted = %d, want 1", n)
	}
	body, _ := os.ReadFile(p)
	for _, want := range []string{"OPERATOR alice@laptop", "EXISTING root@pve", "NEWJOINER root@pve2"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("authorized_keys = %q, missing %q", body, want)
		}
	}
}

// Idempotent, and compared on type+material rather than on the whole line: the
// same key reads as "root@pve2" on one node and "root@pve" after a rename, and
// treating those as different admits a duplicate on every single convergence.
func TestAdmitKeysIgnoresTheComment(t *testing.T) {
	p := authKeysFixture(t, "ssh-ed25519 SAMEKEY root@pve2\n")

	n, err := AdmitKeys(context.Background(), []string{"ssh-ed25519 SAMEKEY root@renamed"})
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if n != 0 {
		t.Fatalf("admitted = %d, want 0 — the key is already trusted", n)
	}
	body, _ := os.ReadFile(p)
	if strings.Count(string(body), "SAMEKEY") != 1 {
		t.Fatalf("the key was duplicated: %q", body)
	}
}

// A rotated node presents a key the cluster has never seen. Admitting it is the
// whole reason this exists: without it, `pvecm add --use_ssh` cannot
// authenticate and fails with "unable to copy ssh ID" and nothing else.
func TestAdmitKeysAdmitsARotatedIdentity(t *testing.T) {
	p := authKeysFixture(t, "ssh-ed25519 OLDKEY root@pve2\n")

	n, err := AdmitKeys(context.Background(), []string{"ssh-ed25519 ROTATED root@pve2"})
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if n != 1 {
		t.Fatalf("admitted = %d, want 1", n)
	}
	body, _ := os.ReadFile(p)
	if !strings.Contains(string(body), "ROTATED") {
		t.Fatalf("the rotated key was not admitted: %q", body)
	}
	// The old key is left alone: this file is not ours to prune, and removing a
	// key an operator or another node still relies on is not recoverable from
	// here.
	if !strings.Contains(string(body), "OLDKEY") {
		t.Fatalf("an existing key was removed: %q", body)
	}
}

// Only the deterministic owner admits, or five members append the same key to
// one replicated file at once.
func TestAdmissionHasOneOwner(t *testing.T) {
	d := Desired{Cluster: "lab-a", Members: []DesiredMember{
		{Node: "pve", HypervisorID: "a"},
		{Node: "pve2", HypervisorID: "b"},
	}}
	obs := Observation{Cluster: "lab-a", Members: []string{"pve", "pve2"}, Quorate: true}

	owner, ok := d.admissionOwner(obs)
	if !ok || owner != "pve" {
		t.Fatalf("owner = %q (%v), want the lowest-named present member", owner, ok)
	}
}
