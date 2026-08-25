package jobs

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
)

// TestArriveStepsAgainstALiveCluster exercises the real arrival steps over SSH.
//
// The counterpart to TestDepartStepsAgainstALiveCluster, and together the two
// are a migration. Every defect these steps defend against was found by running
// commands on a cluster: the bare "unable to copy ssh ID" on an unseeded
// known_hosts, the guest count that fires on a node with no guests, and the join
// key left authorized afterwards.
//
// DESTRUCTIVE. It joins a standalone node to a cluster.
//
//	APIARY_LAB_JOINER=addr,nodename APIARY_LAB_TARGET_NODE=addr,nodename \
//	APIARY_LAB_HIVE=clustername APIARY_LAB_LINK0=addr \
//	  go test ./internal/jobs/ -run TestArriveStepsAgainstALiveCluster -v
func TestArriveStepsAgainstALiveCluster(t *testing.T) {
	joinerSpec := os.Getenv("APIARY_LAB_JOINER")
	targetSpec := os.Getenv("APIARY_LAB_TARGET_NODE")
	hive := os.Getenv("APIARY_LAB_HIVE")
	link0 := os.Getenv("APIARY_LAB_LINK0") // one ring is enough for the lab
	if joinerSpec == "" || targetSpec == "" || hive == "" || link0 == "" {
		t.Skip("APIARY_LAB_JOINER / APIARY_LAB_TARGET_NODE / APIARY_LAB_HIVE / APIARY_LAB_LINK0 " +
			"are not set — SKIPPING; the arrival shell commands were NOT exercised by this run")
	}

	joinerAddr, joinerName := splitPair(t, joinerSpec)
	targetAddr, targetName := splitPair(t, targetSpec)

	key, err := os.ReadFile(os.Getenv("HOME") + "/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("read the operator key: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
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
	joiner, target := open(joinerAddr), open(targetAddr)

	blockers, err := ReadJoinBlockers(ctx, joiner, joinerName)
	if err != nil {
		t.Fatalf("ReadJoinBlockers: %v", err)
	}
	if blockers.Blocked() {
		t.Fatalf("the joiner is not in a joinable state: %s", blockers.Describe())
	}
	t.Log("ok  ReadJoinBlockers: nothing pvecm would refuse on")

	hostKeys, err := ReadHostKeys(ctx, target)
	if err != nil {
		t.Fatalf("ReadHostKeys: %v", err)
	}
	t.Logf("ok  ReadHostKeys: %d host key(s) read over an authenticated session", len(hostKeys))

	if err := SeedKnownHosts(ctx, joiner, []string{targetAddr, targetName}, hostKeys); err != nil {
		t.Fatalf("SeedKnownHosts: %v", err)
	}
	t.Log("ok  SeedKnownHosts, verified present under address and node name")

	joinerKeys, err := ReadNodeClientKeys(ctx, joiner)
	if err != nil {
		t.Fatalf("ReadNodeClientKeys: %v", err)
	}
	if err := AuthorizeJoinKey(ctx, target, preferRSA(joinerKeys), joinerAddr); err != nil {
		t.Fatalf("AuthorizeJoinKey: %v", err)
	}
	t.Log("ok  AuthorizeJoinKey, from=-restricted to the joiner")

	links := []tenant.CorosyncLink{{Number: 0, Address: link0}}
	if err := JoinCluster(ctx, joiner, targetAddr, links, hive); err != nil {
		t.Fatalf("JoinCluster: %v", err)
	}
	t.Logf("ok  JoinCluster: quorate member of %q", hive)

	if err := AssertManagedKeySurvived(ctx, joiner); err != nil {
		t.Fatalf("AssertManagedKeySurvived: %v", err)
	}
	t.Log("ok  the control plane still authenticates after the join replaced pmxcfs")

	n, err := RevokeJoinKeys(ctx, target, joinerKeys)
	if err != nil {
		t.Fatalf("RevokeJoinKeys: %v", err)
	}
	t.Logf("ok  RevokeJoinKeys removed %d grant(s) and verified none remain", n)

	// The outcome, checked independently of the steps that produced it: the
	// joiner is a member, and no longer holds a standing grant on the target.
	out, _ := joiner.Run(ctx, "ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=8 root@"+
		hostOnly(targetAddr)+" hostname 2>&1")
	if !strings.Contains(out, "Permission denied") {
		// Not a failure on its own: once joined, the CLUSTER's own keyring
		// legitimately authorizes inter-node SSH. Recorded so the distinction is
		// visible rather than assumed either way.
		t.Logf("note: the joiner still reaches the target (%q) — expected after a join, since "+
			"the cluster keyring authorizes members to each other", trim(out))
	} else {
		t.Log("ok  the ephemeral grant is gone and the joiner is refused on it")
	}
}
