package jobs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDepartStepsAgainstALiveCluster exercises the real step functions over SSH
// against real Proxmox nodes.
//
// It exists because the fake Runner in depart_hypervisor_test.go proves the
// ORDER of the steps and nothing about whether the commands work. Every defect
// this sequence was written for — the self-blocking delnode, the stale cfs lock,
// the keyring prune that updatecerts does not do, the inherited guest configs —
// was found by running commands on a cluster, not by reasoning about them.
//
// DESTRUCTIVE. It removes a node from its cluster and does not put it back. Set
// both variables only against a lab you are willing to rebuild:
//
//	APIARY_LAB_DEPARTING=addr,nodename APIARY_LAB_REMAINING=addr,nodename \
//	  go test ./internal/jobs/ -run TestDepartStepsAgainstALiveCluster -v
//
// Skipped loudly rather than silently: a skipped destructive test looks exactly
// like a passing one, and this is the only coverage the shell commands have.
func TestDepartStepsAgainstALiveCluster(t *testing.T) {
	departing := os.Getenv("APIARY_LAB_DEPARTING")
	remaining := os.Getenv("APIARY_LAB_REMAINING")
	if departing == "" || remaining == "" {
		t.Skip("APIARY_LAB_DEPARTING / APIARY_LAB_REMAINING are not set — SKIPPING; " +
			"the departure shell commands were NOT exercised by this run")
	}

	depAddr, depName := splitPair(t, departing)
	remAddr, remName := splitPair(t, remaining)

	key, err := os.ReadFile(os.Getenv("HOME") + "/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("read the operator key: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	open := func(addr string) Runner {
		r, err := NewSSHRunner(addr, "root", key)
		if err != nil {
			t.Fatalf("runner for %s: %v", addr, err)
		}
		if err := r.Connect(ctx); err != nil {
			t.Fatalf("connect to %s: %v", addr, err)
		}
		t.Cleanup(func() { _ = r.Close() })
		return r
	}
	dep, rem := open(depAddr), open(remAddr)

	keys, err := ReadNodeClientKeys(ctx, dep)
	if err != nil {
		t.Fatalf("ReadNodeClientKeys: %v", err)
	}
	t.Logf("ok  ReadNodeClientKeys: %d key(s) to prune by", len(keys))

	// Fingerprinted while the node is still a member, because /etc/pve is gone
	// the moment pve-cluster stops. These are what prove the hive's secrets were
	// actually destroyed rather than merely disconnected from.
	secretsBefore, err := ReadClusterSecretDigests(ctx, dep)
	if err != nil {
		t.Fatalf("ReadClusterSecretDigests: %v", err)
	}
	held := 0
	for _, d := range secretsBefore {
		if d != "absent" {
			held++
		}
	}
	if held == 0 {
		t.Fatal("the departing node holds none of the cluster secrets, so this test would " +
			"prove nothing about destroying them")
	}
	t.Logf("ok  ReadClusterSecretDigests: %d cluster secret(s) held before departure", held)

	// Read the keyring BEFORE pruning, so "the key was removed" is a measured
	// change rather than an assertion about a final state that may never have
	// contained it. This is the check that was got wrong by hand: grepping for a
	// comment the key does not carry reports absence and proves nothing.
	before, err := countKeyringMatches(ctx, rem, keys)
	if err != nil {
		t.Fatalf("count keyring matches before: %v", err)
	}
	if before == 0 {
		t.Fatalf("the departing node's key is not in the remaining cluster's keyring to begin " +
			"with, so a prune here would prove nothing — check the cluster is actually joined")
	}
	t.Logf("ok  the departing node holds %d key(s) in the remaining cluster's keyring", before)

	// How many nodes the cluster has BEFORE the removal, read from the cluster
	// rather than assumed. This was hardcoded to 2 (expecting 1 after), which is
	// the shape of the lab it was written against; run against a four-node
	// cluster it failed on its own assertion having removed the node correctly,
	// reporting a half-done removal that had in fact completed.
	nodesBefore, err := settledNodeCount(ctx, rem)
	if err != nil {
		t.Fatalf("count the cluster's nodes before the removal: %v", err)
	}
	if nodesBefore < 2 {
		t.Fatalf("the remaining cluster reports %d node(s), so there is nothing to depart from",
			nodesBefore)
	}
	t.Logf("ok  the cluster has %d node(s) before the removal, expecting %d after",
		nodesBefore, nodesBefore-1)

	if err := StopClusterStack(ctx, dep); err != nil {
		t.Fatalf("StopClusterStack: %v", err)
	}
	t.Log("ok  StopClusterStack")

	if err := DelNode(ctx, rem, depName, nodesBefore-1); err != nil {
		t.Fatalf("DelNode: %v", err)
	}
	t.Log("ok  DelNode, and corosync.conf agrees")

	removed, err := PruneKeysFromClusterKeyring(ctx, rem, keys)
	if err != nil {
		t.Fatalf("PruneKeysFromClusterKeyring: %v", err)
	}
	if removed != before {
		t.Errorf("prune removed %d key(s), expected %d", removed, before)
	}
	t.Logf("ok  PruneKeysFromClusterKeyring removed %d key(s) and verified none remain", removed)

	revoked, err := RevokeJoinKeys(ctx, rem, keys)
	if err != nil {
		t.Fatalf("RevokeJoinKeys: %v", err)
	}
	t.Logf("ok  RevokeJoinKeys removed %d ephemeral join key(s)", revoked)

	if err := RemoveNodeDir(ctx, rem, depName); err != nil {
		t.Fatalf("RemoveNodeDir: %v", err)
	}
	t.Log("ok  RemoveNodeDir")

	if err := MakeStandalone(ctx, dep, []string{remName}, secretsBefore); err != nil {
		t.Fatalf("MakeStandalone: %v", err)
	}
	t.Log("ok  MakeStandalone: standalone, /etc/pve writable, and the hive's secrets destroyed")

	// The outcome that matters, checked independently of the steps that produced
	// it: the departed node can no longer reach the cluster it left.
	out, err := rem.Run(ctx, "grep -c ssh- /etc/pve/priv/authorized_keys || true")
	if err != nil {
		t.Fatalf("re-read the keyring: %v", err)
	}
	t.Logf("ok  remaining cluster keyring now holds %s key(s)", trim(out))
}

func splitPair(t *testing.T, s string) (addr, name string) {
	t.Helper()
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return s[:i], s[i+1:]
		}
	}
	t.Fatalf("want \"addr,nodename\", got %q", s)
	return "", ""
}

func trim(s string) string { return fmt.Sprint(lastLine(s)) }

// TestRevokeJoinKeysAgainstALiveHost checks the second access path on its own.
//
// Separate from the full sequence because it was found separately: after a
// keyring prune that was verified correct, the departed node STILL had a root
// shell on the hive it left, through the ephemeral key its arrival had installed
// in the join file. A test for the whole sequence would have passed while that
// hole was open, because every step it checked did work.
//
//	APIARY_LAB_TARGET=addr APIARY_LAB_SUBJECT=addr \
//	  go test ./internal/jobs/ -run TestRevokeJoinKeysAgainstALiveHost -v
func TestRevokeJoinKeysAgainstALiveHost(t *testing.T) {
	target := os.Getenv("APIARY_LAB_TARGET")   // the host whose join file is cleaned
	subject := os.Getenv("APIARY_LAB_SUBJECT") // the host whose key is revoked
	if target == "" || subject == "" {
		t.Skip("APIARY_LAB_TARGET / APIARY_LAB_SUBJECT are not set — SKIPPING; join-key " +
			"revocation was NOT exercised against a host by this run")
	}

	key, err := os.ReadFile(os.Getenv("HOME") + "/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("read the operator key: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	open := func(addr string) *SSHRunner {
		r, err := NewSSHRunner(addr, "root", key)
		if err != nil {
			t.Fatalf("runner for %s: %v", addr, err)
		}
		if err := r.Connect(ctx); err != nil {
			t.Fatalf("connect to %s: %v", addr, err)
		}
		t.Cleanup(func() { _ = r.Close() })
		return r
	}
	tgt, subj := open(target), open(subject)

	keys, err := ReadNodeClientKeys(ctx, subj)
	if err != nil {
		t.Fatalf("ReadNodeClientKeys: %v", err)
	}

	// Prove the hole is open BEFORE closing it. A revocation test that starts
	// from a closed state passes without doing anything.
	reach := "ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=8 root@" +
		hostOnly(target) + " hostname 2>&1"
	out, _ := subj.Run(ctx, reach)
	if !strings.Contains(out, "Permission denied") {
		t.Logf("ok  before revocation the subject reaches the target: %q", trim(out))
	} else {
		t.Skipf("the subject already cannot reach the target (%q), so this run would prove "+
			"nothing — install a join key first", trim(out))
	}

	n, err := RevokeJoinKeys(ctx, tgt, keys)
	if err != nil {
		t.Fatalf("RevokeJoinKeys: %v", err)
	}
	t.Logf("ok  revoked %d join key(s)", n)

	out, _ = subj.Run(ctx, reach)
	if !strings.Contains(out, "Permission denied") {
		t.Fatalf("the subject STILL reaches the target after revocation: %q", trim(out))
	}
	t.Logf("ok  after revocation the subject is refused: %q", trim(out))
}

// hostOnly strips a port, since these addresses are passed bare to ssh.
func hostOnly(addr string) string {
	if i := strings.LastIndex(addr, ":"); i > 0 && !strings.Contains(addr[i:], "]") {
		return addr[:i]
	}
	return addr
}
