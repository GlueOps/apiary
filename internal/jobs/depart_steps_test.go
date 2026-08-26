package jobs

import (
	"context"
	"strings"
	"testing"
)

// stubRunner answers a fixed script. Separate from departRunner so these guard
// tests do not depend on the orchestration harness.
type stubRunner struct {
	cmds    []string
	replies map[string]string
	err     error

	// quorate models the cluster's actual state rather than a canned reply, so a
	// test can express "quorate until the node is killed" — which is the ordering
	// a removal performed while the departing node is still up produces.
	quorate                bool
	loseQuorumAfterDelnode bool
	delnodeRan             bool
}

func (s *stubRunner) Run(_ context.Context, cmd string) (string, error) {
	s.cmds = append(s.cmds, cmd)
	if s.err != nil {
		return "", s.err
	}

	// Stateful where the state is the point. A canned "Quorate: No" cannot
	// express a cluster that had quorum for the kill and lost it afterwards.
	if s.loseQuorumAfterDelnode {
		switch {
		case strings.Contains(cmd, "pvecm delnode"):
			s.delnodeRan = true
			s.quorate = false
			return "Killing node 2\n", nil
		case strings.Contains(cmd, "pvecm expected"):
			s.quorate = true
			return "", nil
		case strings.Contains(cmd, "Quorate"):
			if s.quorate {
				return "Yes\n", nil
			}
			return "No\n", nil
		}
	}

	for k, v := range s.replies {
		if strings.Contains(cmd, k) {
			return v, nil
		}
	}
	return "", nil
}
func (s *stubRunner) Upload(context.Context, string, uint32, []byte) error { return nil }
func (s *stubRunner) Reconnect(context.Context) error                      { return nil }

func TestPruneRefusesWithNoKeys(t *testing.T) {
	// Defence in depth behind ReadNodeClientKeys, and worth its own test because
	// the failure is silent: grep with an empty pattern file matches nothing, so
	// the prune "succeeds" having removed nothing and the verification then
	// agrees, leaving the departing node's access in place with a clean result.
	r := &stubRunner{}
	if _, err := PruneKeysFromClusterKeyring(context.Background(), r, nil); err == nil {
		t.Fatal("pruning with no keys must be refused, not treated as a no-op success")
	}
	if len(r.cmds) != 0 {
		t.Errorf("nothing should have been run on the host; ran %v", r.cmds)
	}
}

func TestReadNodeClientKeysRefusesWhenTheNodeHasNone(t *testing.T) {
	r := &stubRunner{replies: map[string]string{"id_rsa.pub": "\n"}}
	if _, err := ReadNodeClientKeys(context.Background(), r); err == nil {
		t.Fatal("a node with no root client key must fail: there is nothing to prune by")
	}
}

func TestReadNodeClientKeysStripsTheComment(t *testing.T) {
	// The same key sits in the cluster keyring with a DIFFERENT comment, because
	// PVE rewrites it. Matching on the whole line would therefore miss it — the
	// precise shape of a check that reports success having found nothing.
	r := &stubRunner{replies: map[string]string{
		"id_rsa.pub": "ssh-ed25519 AAAABODY root@hv5\nssh-rsa AAAAOTHER some other comment\n",
	}}
	keys, err := ReadNodeClientKeys(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	for _, k := range keys {
		if len(strings.Fields(k)) != 2 {
			t.Errorf("key %q must be type+body only, with the comment dropped", k)
		}
	}
	if len(keys) != 2 {
		t.Fatalf("want 2 keys, got %d: %v", len(keys), keys)
	}
}

func TestRemoveNodeDirRefusesUnsafeNames(t *testing.T) {
	// The name is interpolated into an rm -rf under /etc/pve. A slash lets it
	// escape the directory it is supposed to be naming.
	for _, name := range []string{"", "  ", "../../etc", "a/b", "has space"} {
		t.Run(name, func(t *testing.T) {
			r := &stubRunner{}
			if err := RemoveNodeDir(context.Background(), r, name); err == nil {
				t.Fatalf("name %q must be refused", name)
			}
			if len(r.cmds) != 0 {
				t.Errorf("nothing should reach the host for an unsafe name; ran %v", r.cmds)
			}
		})
	}
}

func TestMakeStandaloneRefusesUnsafePeerNames(t *testing.T) {
	r := &stubRunner{}
	if err := MakeStandalone(context.Background(), r, []string{"hv1", "../../etc"}, nil); err == nil {
		t.Fatal("an unsafe peer name must be refused")
	}
	// Refused BEFORE anything runs — a partially applied standalone conversion
	// leaves a node that is neither clustered nor standalone.
	if len(r.cmds) != 0 {
		t.Errorf("nothing should have run; ran %v", r.cmds)
	}
}

func TestEnsureQuorumRefusesNonsenseVoteCounts(t *testing.T) {
	for _, n := range []int{0, -1} {
		r := &stubRunner{}
		if err := EnsureQuorum(context.Background(), r, n); err == nil {
			t.Fatalf("expected votes of %d must be refused", n)
		}
	}
}

func TestEnsureQuorumIsANoOpWhenAlreadyQuorate(t *testing.T) {
	r := &stubRunner{replies: map[string]string{"Quorate": "Yes\n"}}
	if err := EnsureQuorum(context.Background(), r, 2); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	for _, c := range r.cmds {
		if strings.Contains(c, "pvecm expected") {
			t.Error("a quorate cluster must not have its expected vote count lowered")
		}
	}
}

func TestDelNodeRefusesNonsenseInput(t *testing.T) {
	r := &stubRunner{}
	if err := DelNode(context.Background(), r, "", 2); err == nil {
		t.Error("an empty node name must be refused")
	}
	if err := DelNode(context.Background(), r, "hv5", 0); err == nil {
		t.Error("expecting zero nodes after removal must be refused")
	}
}

func TestDelNodeFailsWhenCorosyncConfDisagrees(t *testing.T) {
	// The half-done removal: delnode reports "Killing node N" and then fails the
	// config rewrite, leaving the cluster expecting a vote from a node that is
	// never coming back. Its output alone cannot distinguish that from success.
	r := &stubRunner{replies: map[string]string{
		"Quorate":       "Yes\n",
		"APIARY_NODES=": "APIARY_NODES=3\n", // still three after removing one of three
	}}
	err := DelNode(context.Background(), r, "hv5", 2)
	if err == nil {
		t.Fatal("a corosync.conf that still lists the departed node must fail")
	}
	if !strings.Contains(err.Error(), "half-done") {
		t.Errorf("the error must name the actual condition, got %q", err.Error())
	}
}

func TestCountKeyringMatchesTreatsAnUnreadableCountAsAnError(t *testing.T) {
	// Not as zero. Zero is "the key is gone", which is exactly the answer that
	// lets a departure complete with the access still in place.
	r := &stubRunner{replies: map[string]string{"grep -cF": "not a number\n"}}
	if _, err := countKeyringMatches(context.Background(), r, []string{"ssh-ed25519 BODY"}); err == nil {
		t.Fatal("an unreadable count must be an error, never zero")
	}
}

func TestRevokeJoinKeysRefusesWithNoKeys(t *testing.T) {
	r := &stubRunner{}
	if _, err := RevokeJoinKeys(context.Background(), r, nil); err == nil {
		t.Fatal("revoking with no keys must be refused, not treated as a successful no-op")
	}
}

func TestRevokeJoinKeysIsANoOpWhenTheFileIsAbsent(t *testing.T) {
	// The normal case: most hives were not joined by this system.
	r := &stubRunner{replies: map[string]string{"test -f": "ABSENT\n"}}
	n, err := RevokeJoinKeys(context.Background(), r, []string{"ssh-rsa BODY"})
	if err != nil {
		t.Fatalf("an absent join file is not an error: %v", err)
	}
	if n != 0 {
		t.Errorf("want 0 revoked, got %d", n)
	}
	for _, c := range r.cmds {
		if strings.Contains(c, "grep -vF") {
			t.Error("nothing should be rewritten when the file does not exist")
		}
	}
}

func TestRevokeJoinKeysFailsWhenTheKeyRemains(t *testing.T) {
	// The same shape as a keyring prune that does not stick: the rewrite reports
	// success and the key is still authorized.
	r := &stubRunner{replies: map[string]string{
		"test -f":  "PRESENT\n",
		"grep -cF": "1\n", // still 1 after the rewrite
	}}
	if _, err := RevokeJoinKeys(context.Background(), r, []string{"ssh-rsa BODY"}); err == nil {
		t.Fatal("a revoke that leaves the key authorized must fail")
	}
}

func TestDelNodeRestoresQuorumBeforeAttemptingTheRemoval(t *testing.T) {
	// The departing node's stack is stopped before delnode runs, so on a small
	// cluster the survivor has ALREADY lost quorum when we get here — and pvecm
	// refuses with "cluster not ready - no quorum?" without doing anything.
	//
	// Found by running the whole cycle rather than by hand: a manual removal done
	// while the departing node was still up had quorum for the kill and lost it
	// afterwards. Both orderings occur, so quorum is restored on both sides of
	// the call.
	r := &stubRunner{replies: map[string]string{
		"Quorate":       "No\n",
		"APIARY_NODES=": "APIARY_NODES=1\n",
	}}
	if err := DelNode(context.Background(), r, "hv5", 1); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	var expectedAt, delnodeAt = -1, -1
	for i, c := range r.cmds {
		if strings.Contains(c, "pvecm expected") && expectedAt < 0 {
			expectedAt = i
		}
		if strings.Contains(c, "pvecm delnode") && delnodeAt < 0 {
			delnodeAt = i
		}
	}
	if expectedAt < 0 {
		t.Fatalf("quorum was never restored; ran %v", r.cmds)
	}
	if delnodeAt < 0 {
		t.Fatalf("delnode never ran; ran %v", r.cmds)
	}
	if expectedAt > delnodeAt {
		t.Error("quorum must be restored BEFORE delnode is attempted, or pvecm refuses outright")
	}
}

func TestDelNodeRestoresQuorumLostBYTheRemoval(t *testing.T) {
	// The other ordering, and the one found by hand on a live cluster: the
	// cluster HAS quorum when delnode runs, the kill costs it, and the very next
	// step of delnode — rewriting corosync.conf to lower the expected count —
	// then needs the writable /etc/pve it just lost.
	//
	// Distinct from the pre-delnode case: there, pvecm refuses outright and does
	// nothing. Here it half-succeeds and leaves the cluster read-only. Both
	// happen, so quorum is restored on both sides of the call and both need
	// covering — a fake that reports a fixed "not quorate" exercises only the
	// first, because the first call then fixes it and the second is a no-op.
	r := &stubRunner{
		quorate:                true,
		loseQuorumAfterDelnode: true,
		replies:                map[string]string{"APIARY_NODES=": "APIARY_NODES=1\n"},
	}
	if err := DelNode(context.Background(), r, "hv5", 1); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	delnodeAt := -1
	expectedAfter := false
	for i, c := range r.cmds {
		if strings.Contains(c, "pvecm delnode") && delnodeAt < 0 {
			delnodeAt = i
		}
		if delnodeAt >= 0 && i > delnodeAt && strings.Contains(c, "pvecm expected") {
			expectedAfter = true
		}
	}
	if delnodeAt < 0 {
		t.Fatalf("delnode never ran; ran %v", r.cmds)
	}
	if !expectedAfter {
		t.Fatal("quorum lost BY the removal must be restored after it, or /etc/pve stays " +
			"read-only and everything downstream fails obscurely")
	}
	if !r.quorate {
		t.Fatal("the cluster must be left quorate")
	}
}
