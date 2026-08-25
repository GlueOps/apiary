package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Paths and files the departure sequence touches. The cluster keyring is the
// one that matters: it lives on pmxcfs, so a write to it needs quorum and
// replicates to every remaining node — which is exactly why pruning there is
// both possible and necessary.
const (
	clusterKeyring = "/etc/pve/priv/authorized_keys"
	corosyncConf   = "/etc/pve/corosync.conf"
	cfsLockDir     = "/etc/pve/priv/lock/file-corosync_conf"
)

// ReadNodeClientKeys returns the public keys a node authenticates to its peers
// with, as authorized_keys lines.
//
// Read from the node itself rather than reconstructed from a fingerprint,
// because the prune below matches on key MATERIAL. Matching on the comment field
// was tried against a real cluster and found the key absent when it was sitting
// right there — the worker-installed key carries no comment, so a grep for one
// reports success having done nothing. A fingerprint would work but has to be
// recomputed per key type; the material is what is actually in the file.
func ReadNodeClientKeys(ctx context.Context, r Runner) ([]string, error) {
	out, err := r.Run(ctx, "cat /root/.ssh/id_rsa.pub /root/.ssh/id_ed25519.pub 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("depart: read the node's client keys: %w (%s)", err, strings.TrimSpace(out))
	}
	var keys []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Keep only type+base64 and drop the comment. The same key is present in
		// the cluster keyring with a DIFFERENT comment — PVE rewrites it — so
		// including the comment in the match would miss it.
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		keys = append(keys, f[0]+" "+f[1])
	}
	if len(keys) == 0 {
		// Not benign. Without the departing node's keys there is nothing to prune
		// by, and a departure that cannot prune must not report success.
		return nil, fmt.Errorf("depart: the node has no root client key, so its access to the " +
			"cluster cannot be identified or removed")
	}
	return keys, nil
}

// PruneKeysFromClusterKeyring removes a departing node's keys from the cluster
// keyring on a REMAINING node, and proves they are gone.
//
// This closes a real hole, demonstrated on a live pair of clusters: `pvecm
// delnode` does not remove the departed node's public key from the remaining
// cluster's keyring, and `pvecm updatecerts` does not either — it only ever adds
// the local node's. So a host that leaves a hive keeps passwordless root on
// every machine still in it, indefinitely, and the operator who followed the
// documented removal has no signal that anything was left behind.
//
// The asymmetry is what makes it easy to miss: the JOINING side is cleaned
// automatically, because a join replaces the joiner's keyring wholesale. Only
// the side that stays is left holding the credential.
//
// Requires QUORUM on the remaining cluster: the keyring is on pmxcfs, which goes
// read-only without it. Call this after quorum has been restored, not between
// delnode and the recovery — see EnsureQuorum.
func PruneKeysFromClusterKeyring(ctx context.Context, r Runner, keys []string) (int, error) {
	if len(keys) == 0 {
		return 0, fmt.Errorf("depart: refusing to prune with no keys to match — a prune that " +
			"matches nothing succeeds while removing nothing")
	}

	before, err := countKeyringMatches(ctx, r, keys)
	if err != nil {
		return 0, err
	}

	// grep -F: fixed strings, never patterns. These are base64 blobs containing
	// + and / and they are read off a host, so treating them as regexes would be
	// both wrong and a way to make this command do something else.
	//
	// Written via a temp file and copied, not renamed: pmxcfs is a fuse
	// filesystem and does not honour a cross-filesystem rename, so the atomic
	// temp-then-mv idiom used elsewhere in this package does not apply here.
	var b strings.Builder
	b.WriteString("set -e; : > /tmp/.apiary-prune-pat; ")
	for _, k := range keys {
		fmt.Fprintf(&b, "printf '%%s\\n' %s >> /tmp/.apiary-prune-pat; ", shellQuote(k))
	}
	// `|| true` on grep: it exits 1 when nothing survives the filter, which is
	// the legitimate case of a keyring that held only the departing node's key.
	// Without it, set -e turns an empty result into a failure.
	fmt.Fprintf(&b,
		"grep -vF -f /tmp/.apiary-prune-pat %s > /tmp/.apiary-keyring.new || true; "+
			"cp /tmp/.apiary-keyring.new %s; rm -f /tmp/.apiary-prune-pat /tmp/.apiary-keyring.new",
		clusterKeyring, clusterKeyring)

	if out, err := r.Run(ctx, b.String()); err != nil {
		return 0, fmt.Errorf("depart: prune the cluster keyring: %w (%s)", err, strings.TrimSpace(out))
	}

	// The proof. A prune that reports success without checking is the same defect
	// as trusting updatecerts to have done it — and this one is silent, because
	// the access it leaves behind looks like nothing at all from the control
	// plane's side.
	after, err := countKeyringMatches(ctx, r, keys)
	if err != nil {
		return 0, err
	}
	if after != 0 {
		return before - after, fmt.Errorf("depart: %d of the departing node's key(s) are STILL in "+
			"%s after pruning; the node would keep root access to this hive after leaving it, so "+
			"the departure is refused rather than completed", after, clusterKeyring)
	}
	return before, nil
}

// countKeyringMatches reports how many of these keys are in the cluster keyring.
func countKeyringMatches(ctx context.Context, r Runner, keys []string) (int, error) {
	return countMatches(ctx, r, clusterKeyring, keys)
}

// countMatches reports how many of these keys appear in one authorized-keys
// file. One definition for both prune paths, so "is this key still there" cannot
// be answered differently by the keyring check and the join-file check.
func countMatches(ctx context.Context, r Runner, path string, keys []string) (int, error) {
	var b strings.Builder
	b.WriteString(": > /tmp/.apiary-count-pat; ")
	for _, k := range keys {
		fmt.Fprintf(&b, "printf '%%s\\n' %s >> /tmp/.apiary-count-pat; ", shellQuote(k))
	}
	fmt.Fprintf(&b, "grep -cF -f /tmp/.apiary-count-pat %s || true; rm -f /tmp/.apiary-count-pat",
		path)

	out, err := r.Run(ctx, b.String())
	if err != nil {
		return 0, fmt.Errorf("depart: count keyring matches: %w (%s)", err, strings.TrimSpace(out))
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(lastLine(out)))
	if convErr != nil {
		// An unreadable count is not zero. Treating it as zero would report a
		// successful prune on the one path where nothing can be verified.
		return 0, fmt.Errorf("depart: could not read the keyring match count from %q", strings.TrimSpace(out))
	}
	return n, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}

// joinKeyFile is where an arrival installs the ephemeral key the joining node
// authenticates with. A plain local file, deliberately — the same reasoning as
// the managed key file, since a join replaces pmxcfs and a key that lived only
// there would vanish at the moment it was needed.
const joinKeyFile = managedDir + "/join"

// RevokeJoinKeys removes a departing node's keys from the join file on a
// remaining node.
//
// Pruning the pmxcfs keyring alone is NOT sufficient, and this was found by
// checking rather than reasoning: after a correct keyring prune the departed node
// still had a root shell on the hive it left, because the ephemeral key its
// arrival had installed was still sitting in the join file.
//
// The irony is worth stating, because it is the trap. authorized_keys.d survives
// a join, a leave, a quorum loss and updatecerts — which is precisely why the
// managed key belongs there, and precisely why a JOIN key must not be left there.
// The property that makes the location correct for a permanent credential makes
// it the worst available location for a temporary one.
//
// Runs from the departure side even though an arrival installed the key. A
// departure's obligation is that the node retains no access to the hive it left,
// and that is a statement about every path, not about the one file this system
// happens to think of first. Cleaning up after itself is the arrival's job too;
// depending on it is how the leak survives.
func RevokeJoinKeys(ctx context.Context, r Runner, keys []string) (int, error) {
	if len(keys) == 0 {
		return 0, fmt.Errorf("depart: refusing to revoke join keys with nothing to match")
	}

	// Absent is the normal case — most hives were not joined by this system, and
	// an arrival that already cleaned up leaves nothing here.
	out, err := r.Run(ctx, fmt.Sprintf("test -f %s && echo PRESENT || echo ABSENT", joinKeyFile))
	if err != nil {
		return 0, fmt.Errorf("depart: check for a join key file: %w (%s)", err, strings.TrimSpace(out))
	}
	if strings.TrimSpace(lastLine(out)) != "PRESENT" {
		return 0, nil
	}

	before, err := countMatches(ctx, r, joinKeyFile, keys)
	if err != nil {
		return 0, err
	}
	if before == 0 {
		return 0, nil
	}

	var b strings.Builder
	b.WriteString("set -e; : > /tmp/.apiary-join-pat; ")
	for _, k := range keys {
		fmt.Fprintf(&b, "printf '%%s\\n' %s >> /tmp/.apiary-join-pat; ", shellQuote(k))
	}
	fmt.Fprintf(&b,
		"grep -vF -f /tmp/.apiary-join-pat %s > /tmp/.apiary-join.new || true; "+
			"cat /tmp/.apiary-join.new > %s; chmod 0644 %s; "+
			"rm -f /tmp/.apiary-join-pat /tmp/.apiary-join.new",
		joinKeyFile, joinKeyFile, joinKeyFile)
	if out, err := r.Run(ctx, b.String()); err != nil {
		return 0, fmt.Errorf("depart: revoke join keys: %w (%s)", err, strings.TrimSpace(out))
	}

	after, err := countMatches(ctx, r, joinKeyFile, keys)
	if err != nil {
		return 0, err
	}
	if after != 0 {
		return before - after, fmt.Errorf("depart: %d of the departing node's key(s) are STILL "+
			"authorized in %s; the node would keep root access to this hive after leaving it",
			after, joinKeyFile)
	}
	return before, nil
}

// AssertClusterHealthy refuses to begin a departure against a cluster that is
// already degraded.
//
// This is the preflight that IS possible. Lowering the expected vote count
// beforehand is not — corosync rejects it while the departing node still votes,
// see EnsureQuorum — so the survivor's brief read-only window cannot be designed
// away. What can be avoided is ENTERING it from a bad starting position.
//
// Stopping a node in an already-non-quorate cluster makes a bad situation worse
// and delnode then refuses outright, leaving a stopped node, an unchanged
// cluster config, and an operator holding a message about quorum that says
// nothing about which of the two problems they now have.
//
// wantNodesBefore is what THIS system believes the cluster contains, including
// the node about to leave. Comparing it against the cluster's own config is a
// cheap consistency check between two independent views, and they disagree
// exactly when the capacity sync is stale — which is the same moment the node
// count feeding the hive-size gate is wrong.
func AssertClusterHealthy(ctx context.Context, r Runner, wantNodesBefore int) error {
	out, err := r.Run(ctx, `echo "APIARY_QUORATE=$(pvecm status 2>/dev/null | awk -F': +' '/^Quorate:/{print $2}')"`)
	if err != nil {
		return fmt.Errorf("depart: read the remaining cluster's quorum state: %w (%s)",
			err, strings.TrimSpace(out))
	}
	quorate := ""
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "APIARY_QUORATE="); ok {
			quorate = strings.TrimSpace(v)
		}
	}
	if !strings.EqualFold(quorate, "yes") {
		return fmt.Errorf("depart: the remaining cluster is NOT quorate before the departure has "+
			"begun (reported %q); removing another node from it would deepen the outage rather "+
			"than complete a migration. Restore quorum first", quorate)
	}

	if err := assertPmxcfsWritable(ctx, r); err != nil {
		return fmt.Errorf("depart: refusing to begin — %w", err)
	}

	n, err := countCorosyncNodes(ctx, r)
	if err != nil {
		return fmt.Errorf("depart: refusing to begin — %w", err)
	}
	if n < 2 {
		return fmt.Errorf("depart: the cluster config lists %d node(s), so there is no node to "+
			"remove without destroying the cluster", n)
	}
	if wantNodesBefore > 0 && n != wantNodesBefore {
		return fmt.Errorf("depart: this system believes the hive has %d node(s) but the cluster's "+
			"own config lists %d; the capacity observation is out of date, which also means the "+
			"node count the hive-size gate just decided on was wrong. Run a sync and retry",
			wantNodesBefore, n)
	}
	return nil
}

// StopClusterStack takes a departing node out of the cluster's view.
//
// Run FIRST, before delnode, so the node cannot write to the cluster's pmxcfs
// while it is being removed — in particular it cannot re-add its own key through
// `pvecm updatecerts` after the prune, which would silently undo the one step
// that closes the access hole.
func StopClusterStack(ctx context.Context, r Runner) error {
	// No `set -e`: corosync may already be dead, and `stop` on a failed unit is
	// itself a failure that says nothing useful.
	if out, err := r.Run(ctx, "systemctl stop pve-cluster corosync 2>&1 || true; "+
		"systemctl is-active pve-cluster corosync 2>&1 || true"); err != nil {
		return fmt.Errorf("depart: stop the cluster stack: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

// EnsureQuorum restores a writable /etc/pve on a cluster that lost quorum by
// having a node removed.
//
// IT CANNOT BE RUN AS A PREFLIGHT. Lowering the expected vote count before the
// departing node is gone would be the obvious way to avoid the read-only window
// below, and corosync refuses it — measured on a live 2-node cluster:
//
//	total votes = 2
//	pvecm expected 1  ->  Unable to set expected votes: CS_ERR_INVALID_PARAM
//	pvecm expected 2  ->  ok
//	pvecm expected 3  ->  ok
//
// votequorum will not set expected_votes BELOW the number of votes currently
// present. Raising it or matching it is fine; lowering it past the live count is
// not. So the call only becomes valid once the departing node has actually
// stopped voting, which is why it sits after StopClusterStack rather than before
// it. The brief read-only window on the survivor is inherent to a small cluster,
// not an ordering mistake — what AssertClusterHealthy does instead is refuse to
// ENTER that window when the cluster is already degraded.
//
// Necessary because `pvecm delnode` on a small cluster is self-blocking, which
// was observed rather than predicted: killing the node drops the survivor below
// the expected vote count, and the very next step of delnode — rewriting
// corosync.conf to LOWER that count — needs the writable /etc/pve it just lost.
// It fails with a bare "Permission denied" on a temp file and leaves the cluster
// read-only, guests unmanageable, and the config still naming the departed node.
//
// `pvecm expected <n>` lowers the runtime expectation without needing to write
// anything, which is the only way back in.
func EnsureQuorum(ctx context.Context, r Runner, expectedVotes int) error {
	if expectedVotes < 1 {
		return fmt.Errorf("depart: refusing to set expected votes to %d", expectedVotes)
	}
	out, err := r.Run(ctx, "pvecm status 2>/dev/null | awk -F': +' '/^Quorate:/{print $2}'")
	if err != nil {
		return fmt.Errorf("depart: read quorum state: %w (%s)", err, strings.TrimSpace(out))
	}
	if strings.EqualFold(strings.TrimSpace(lastLine(out)), "yes") {
		return nil
	}

	if out, err := r.Run(ctx, fmt.Sprintf("pvecm expected %d", expectedVotes)); err != nil {
		return fmt.Errorf("depart: restore quorum with expected=%d: %w (%s)",
			expectedVotes, err, strings.TrimSpace(out))
	}
	// Prove it, rather than assuming the command worked. Everything after this
	// writes to pmxcfs and would fail obscurely on a still-read-only filesystem.
	if err := assertPmxcfsWritable(ctx, r); err != nil {
		return err
	}
	return nil
}

// assertPmxcfsWritable checks the one property the rest of the sequence needs.
func assertPmxcfsWritable(ctx context.Context, r Runner) error {
	const probe = "/etc/pve/.apiary-write-probe"
	if out, err := r.Run(ctx, fmt.Sprintf("touch %s && rm -f %s", probe, probe)); err != nil {
		return fmt.Errorf("depart: /etc/pve is still read-only, so the cluster has not regained "+
			"quorum and nothing further can be written: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

// ClearStaleCfsLock removes the lock a failed delnode leaves behind.
//
// A delnode that dies part-way leaves /etc/pve/priv/lock/file-corosync_conf in
// place, and every retry then fails with "cfs-lock 'file-corosync_conf' error:
// got lock request timeout" — a message that points at contention rather than at
// a leftover directory. Without clearing it, the removal can never be completed
// and the cluster stays wedged.
func ClearStaleCfsLock(ctx context.Context, r Runner) error {
	if out, err := r.Run(ctx, fmt.Sprintf("rmdir %s 2>/dev/null || true", cfsLockDir)); err != nil {
		return fmt.Errorf("depart: clear the stale cfs lock: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

// DelNode removes a node from the cluster and verifies corosync.conf agrees.
//
// The verification is the point. delnode reports "Killing node N" and then may
// fail at the config rewrite, so its output alone does not distinguish a
// completed removal from a half-done one — and a half-done one leaves the
// cluster expecting a vote from a node that is never coming back.
func DelNode(ctx context.Context, r Runner, nodeName string, wantNodes int) error {
	if strings.TrimSpace(nodeName) == "" {
		return fmt.Errorf("depart: refusing to delnode an empty node name")
	}
	if wantNodes < 1 {
		return fmt.Errorf("depart: refusing to expect %d node(s) after removal", wantNodes)
	}

	if err := ClearStaleCfsLock(ctx, r); err != nil {
		return err
	}

	// Quorum FIRST, before delnode is even attempted.
	//
	// The departing node's cluster stack is stopped before this, so on a small
	// cluster the survivor has already lost quorum by the time we get here — and
	// `pvecm delnode` then refuses outright with "cluster not ready - no quorum?"
	// without doing anything at all.
	//
	// That is a different failure from the one after a successful delnode, and it
	// was found by running the sequence end to end rather than by hand: a manual
	// removal performed while the departing node was still UP had quorum for the
	// kill and lost it afterwards, which is the case the second call below covers.
	// Both orderings happen, so both need handling.
	if err := EnsureQuorum(ctx, r, wantNodes); err != nil {
		return err
	}

	// A failure here is not returned immediately: delnode routinely reports
	// "Could not kill node (error = CS_ERR_NOT_EXIST)" when the node is already
	// gone, which is success for our purposes. The assertion below is what
	// decides.
	delOut, delErr := r.Run(ctx, fmt.Sprintf("pvecm delnode %s 2>&1 || true", shellQuote(nodeName)))

	// And again afterwards: killing the node can cost the quorum that the call
	// above restored.
	if err := EnsureQuorum(ctx, r, wantNodes); err != nil {
		return err
	}

	// Retry, now that quorum is back and the lock is clear. The FIRST attempt is
	// the one that typically cannot write the config, so this is the normal path
	// rather than an exceptional one — observed on a live cluster where corosync
	// had already dropped to one node while corosync.conf still listed two.
	//
	// The count is polled rather than read once, and a failure to read it is NOT
	// treated as "no retry needed". An earlier version gated the retry on the
	// count succeeding, which meant the one situation needing a retry — pmxcfs
	// still restarting, count unreadable — was the one that skipped it.
	n, err := settledNodeCount(ctx, r)
	if err != nil {
		return err
	}
	if n != wantNodes {
		if err := ClearStaleCfsLock(ctx, r); err != nil {
			return err
		}
		if out, err := r.Run(ctx, fmt.Sprintf("pvecm delnode %s 2>&1 || true", shellQuote(nodeName))); err != nil {
			return fmt.Errorf("depart: delnode retry: %w (%s)", err, strings.TrimSpace(out))
		}
		if err := EnsureQuorum(ctx, r, wantNodes); err != nil {
			return err
		}
		n, err = settledNodeCount(ctx, r)
		if err != nil {
			return err
		}
	}
	if n != wantNodes {
		return fmt.Errorf("depart: %s still lists %d node(s) after removing %q, expected %d — the "+
			"removal is half-done and the cluster expects a vote from a node that is not coming "+
			"back (delnode said: %s; err: %v)",
			corosyncConf, n, nodeName, wantNodes, strings.TrimSpace(delOut), delErr)
	}
	return nil
}

// countCorosyncNodes counts node entries in the cluster config.
//
// Emits a MARKER rather than a bare number. The first version relied on
// `grep -c ... || echo -1` and was observed returning an empty string against a
// real cluster moments after a delnode: removing a node restarts pmxcfs, and
// while /etc/pve is being remounted the command can produce nothing at all. A
// bare-number contract cannot tell that apart from a malformed reply, and the
// resulting error blamed the parse instead of naming the condition.
func countCorosyncNodes(ctx context.Context, r Runner) (int, error) {
	// Assigned first, then echoed with a default, so the marker line is emitted
	// on every path including a missing file or an unreadable mount.
	cmd := fmt.Sprintf(`n=$(grep -c '^[[:space:]]*name:' %s 2>/dev/null); echo "APIARY_NODES=${n:--1}"`,
		corosyncConf)
	out, err := r.Run(ctx, cmd)
	if err != nil {
		return 0, fmt.Errorf("depart: count nodes in %s: %w (%s)", corosyncConf, err, strings.TrimSpace(out))
	}

	field := ""
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "APIARY_NODES="); ok {
			field = v
		}
	}
	if field == "" {
		return 0, fmt.Errorf("depart: %s could not be counted — the cluster filesystem is not "+
			"answering, which it does while pmxcfs restarts after a node removal (output: %q)",
			corosyncConf, strings.TrimSpace(out))
	}
	n, convErr := strconv.Atoi(field)
	if convErr != nil {
		return 0, fmt.Errorf("depart: could not read a node count from %q", strings.TrimSpace(out))
	}
	if n < 0 {
		return 0, fmt.Errorf("depart: %s is unreadable or absent", corosyncConf)
	}
	return n, nil
}

// settledNodeCount polls the node count until it can be read.
//
// Necessary because a node removal restarts pmxcfs, so the count is briefly
// unreadable exactly when it is being checked. Without this the assertion fails
// on a transient and reports it as a broken cluster.
func settledNodeCount(ctx context.Context, r Runner) (int, error) {
	const attempts = 6
	var lastErr error
	for i := 0; i < attempts; i++ {
		n, err := countCorosyncNodes(ctx, r)
		if err == nil {
			return n, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return 0, lastErr
}

// RemoveNodeDir deletes a departed node's directory from the cluster.
//
// `pvecm delnode` does not do this, and what is left behind is not merely
// untidy: the directory holds the departed node's guest configuration, and on a
// node that later tries to JOIN a cluster the join precheck reads those configs
// as local guests and refuses with "this host already contains virtual guests" —
// on a host that has none of its own. Observed exactly that way in the lab.
func RemoveNodeDir(ctx context.Context, r Runner, nodeName string) error {
	if strings.TrimSpace(nodeName) == "" || strings.ContainsAny(nodeName, "/ ") {
		// The name is interpolated into an rm -rf under /etc/pve. A slash would
		// let it escape the directory it is supposed to be naming.
		return fmt.Errorf("depart: refusing to remove a node directory for the unsafe name %q", nodeName)
	}
	path := "/etc/pve/nodes/" + nodeName
	if out, err := r.Run(ctx, fmt.Sprintf("rm -rf %s", shellQuote(path))); err != nil {
		return fmt.Errorf("depart: remove %s: %w (%s)", path, err, strings.TrimSpace(out))
	}
	return nil
}

// MakeStandalone converts a removed node back into a cluster of one.
//
// The Proxmox documentation says a removed node must be REINSTALLED before it
// can join another cluster. That is caution, not a technical barrier — proven by
// moving a node between two clusters with this sequence and nothing else. What
// the advice is protecting against is the four specific things `pvecm add`
// refuses on, all of which this clears:
//
//	authentication key '/etc/corosync/authkey' already exists
//	cluster config '/etc/pve/corosync.conf' already exists
//	this host already contains virtual guests      <- the inherited replica
//	corosync is already running
//
// `pmxcfs -l` is the load-bearing step: it runs the cluster filesystem in LOCAL
// mode, which is the only way to delete the cluster config on a node that has no
// quorum — and it has no quorum precisely because it was just removed.
func MakeStandalone(ctx context.Context, r Runner, peerNames []string) error {
	for _, p := range peerNames {
		if strings.TrimSpace(p) == "" || strings.ContainsAny(p, "/ ") {
			return fmt.Errorf("depart: refusing to remove a peer directory for the unsafe name %q", p)
		}
	}

	steps := []string{
		"systemctl stop pve-cluster corosync 2>&1 || true",
		"killall pmxcfs 2>/dev/null || true",
		"sleep 2",
		// Local mode. Without this every rm below fails read-only.
		"pmxcfs -l",
		"sleep 2",
		fmt.Sprintf("rm -f %s", shellQuote(corosyncConf)),
		"rm -rf /etc/corosync/*",
	}
	// The inherited replica of the former peers' guest configs — the third
	// precheck. Removed by name rather than by wiping /etc/pve/nodes, because
	// this node's OWN directory lives there too.
	for _, p := range peerNames {
		steps = append(steps, fmt.Sprintf("rm -rf %s", shellQuote("/etc/pve/nodes/"+p)))
	}
	steps = append(steps,
		"killall pmxcfs 2>/dev/null || true",
		"sleep 2",
		"systemctl start pve-cluster",
		"sleep 3",
	)

	for _, cmd := range steps {
		if out, err := r.Run(ctx, cmd); err != nil {
			return fmt.Errorf("depart: standalone conversion at %q: %w (%s)", cmd, err, strings.TrimSpace(out))
		}
	}

	// Prove the node is genuinely standalone and writable, rather than trusting a
	// sequence of commands that each individually reported success.
	if out, err := r.Run(ctx, fmt.Sprintf("test ! -f %s", shellQuote(corosyncConf))); err != nil {
		return fmt.Errorf("depart: %s still exists, so this node is not standalone: %w (%s)",
			corosyncConf, err, strings.TrimSpace(out))
	}
	if err := assertPmxcfsWritable(ctx, r); err != nil {
		return fmt.Errorf("depart: standalone conversion left /etc/pve unwritable: %w", err)
	}
	return nil
}
