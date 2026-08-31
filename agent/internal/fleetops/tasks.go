package fleetops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Task kinds. They are strings on the wire, so they are constants here and the
// control plane's fleet handler documents the same values.
const (
	KindDistUpgrade = "fleet.host.dist-upgrade"
	KindJoin        = "fleet.datacenter.join"
	KindEvict       = "fleet.datacenter.evict"
)

// Phases. They are recorded before the step they name, and survive a reboot, so
// a resumed task can tell what it has already done.
const (
	phasePreflight = "preflight"
	phaseUpgrading = "upgrading"
	phaseRebooting = "rebooting"
	phaseVerifying = "verifying"
	phaseJoining   = "joining"
	phaseEvicting  = "evicting"
)

// Env is what fleetops needs from the agent runtime. It mirrors bee.Env, and is
// redeclared here so this package does not import the runtime that calls it.
type Env struct {
	Logf    func(stream, format string, args ...any)
	Phase   func(phase string)
	Declare func(ctx context.Context, d time.Duration, reason string) error
}

// Deps are the collaborators an executor needs.
type Deps struct {
	Cluster Cluster

	// ClusterErr is why Cluster is nil, kept so a refused task can say what
	// actually went wrong.
	//
	// It used to be logged at startup and discarded, and the refusal then had to
	// GUESS a cause — it blamed the token's permissions, which sent an operator to
	// check ACLs on a token that was fully privileged. The real reason was that no
	// token was configured at all. A message that confidently names the wrong
	// cause is worse than one that names none.
	ClusterErr error

	Runner Runner

	// Node is this host's Proxmox node name, used when pmxcfs cannot be read.
	// Prefer Cluster.NodeName(), which is the value the cluster itself keys on.
	Node   string
	Reboot func(ctx context.Context) error
}

// ErrNoCluster reports that this host has no Proxmox API client, so a task that
// needs one cannot run.
//
// The agent logs "fleet tasks will refuse rather than run" at startup when it
// cannot build a client — usually because the token has no ACLs. That promise
// was not kept: Deps.Cluster was left nil and every task that touched it
// dereferenced nil and panicked, which systemd turned into a crash loop.
//
// Refusing is what the startup message already claimed, and a refused task is
// visible in the control plane where a stack trace in a journal is not.
var ErrNoCluster = errors.New(
	"fleetops: this host has no Proxmox API client, so it cannot perform cluster work")

// requireCluster refuses before a task touches a nil client.
//
// Checked at the top of every task that needs one rather than defended inside
// each step: a nil dereference three calls deep reports as a segfault address,
// and the operator has to read a stack trace to learn that a credential is
// missing.
//
// The refusal carries the REAL reason where there is one. Guessing produced a
// message blaming a token's permissions on a host whose token was fully
// privileged and simply never configured — and an operator who trusts that goes
// and checks ACLs.
func requireCluster(d Deps) error {
	if d.Cluster != nil {
		return nil
	}
	if d.ClusterErr != nil {
		return fmt.Errorf("%w: %w", ErrNoCluster, d.ClusterErr)
	}
	return ErrNoCluster
}

// ---------------------------------------------------------------- dist-upgrade

// Reboot policies, named so the checks below cannot drift from the spec's
// documented values.
const (
	rebootIfNeeded = "if-needed"
	rebootAlways   = "always"
	rebootNever    = "never"
)

type DistUpgradeSpec struct {
	// AllowGuestDowntime permits the upgrade when guests are still running here.
	// Default false: an operator who has not said otherwise expects a rotation to
	// refuse rather than to drop VMs.
	AllowGuestDowntime bool `json:"allow_guest_downtime"`

	// Reboot: "if-needed" (default), "always", or "never". The control plane's
	// gate reads the same strings — see tenant.RebootNever — because it has to
	// know whether a reboot is possible before it can decide anything.
	Reboot string `json:"reboot"`

	// ExpectedDownSeconds is what the host declares before rebooting. It should
	// be generous — a declaration that expires while the host is still coming up
	// turns a normal reboot into an alert.
	ExpectedDownSeconds int `json:"expected_down_seconds"`
}

// DistUpgrade upgrades the host and, if the distribution asks for it, reboots.
//
// Resumable across its own reboot. The phase is written BEFORE the reboot and
// preserved by the store, so the agent that comes back up re-enters this
// function, sees phaseRebooting, and finishes rather than upgrading again. Get
// that wrong and a rotation loops forever on every host in the datacenter.
func DistUpgrade(ctx context.Context, d Deps, e *Env, phase string, rawSpec string) error {

	var spec DistUpgradeSpec
	if err := json.Unmarshal([]byte(orEmptyJSON(rawSpec)), &spec); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}
	if spec.Reboot == "" {
		spec.Reboot = rebootIfNeeded
	}

	// We are back from the reboot we caused.
	if phase == phaseRebooting {
		e.Phase(phaseVerifying)
		e.Logf("agent", "back after the reboot this task requested")
		return verifyAfterReboot(ctx, d, e)
	}

	// THE PREFLIGHT IS ABOUT THE REBOOT, so it runs only when one might happen.
	//
	// It used to run unconditionally, which refused `reboot: never` while guests
	// were running — a check for a reboot that never happens, blocking an upgrade
	// that cannot disturb them. Installing packages does not interrupt a guest.
	//
	// And it is now BEST EFFORT rather than required. The control plane gates this
	// before the task is created, from facts it already holds, so the decision has
	// been made by something that knows the whole fleet. This is a second opinion
	// with live data — worth having when the host can give one, and not worth
	// refusing a purely local job over when it cannot. That is why a dist-upgrade
	// no longer needs a Proxmox token at all: nothing about apt-get and a reboot
	// is orchestrated through the API.
	if spec.Reboot != rebootNever {
		if d.Cluster == nil {
			e.Logf("agent", "no Proxmox client here, so the local preflight is skipped; the "+
				"control plane gated this upgrade before it was queued")
		} else {
			e.Phase(phasePreflight)
			report, err := preflightReboot(ctx, d.Cluster, d.Node, spec.AllowGuestDowntime)
			if err != nil {
				// COULD NOT ASK is not the same as WAS TOLD NO, and treating them
				// alike made this preflight required in fact while the comment
				// above described it as best effort. A host whose local API could
				// not be read refused an upgrade the control plane had already
				// gated — observed as a TLS failure against a certificate the node
				// was serving from a CA it no longer had, which has nothing to do
				// with whether rebooting it is safe.
				//
				// The refusal below still stands: a preflight that RAN and said no
				// is a live fact about this host that the control plane's snapshot
				// cannot have.
				e.Logf("agent", "the local preflight could not be run (%v); continuing on the "+
					"control plane's gate, which was applied before this task was queued", err)
			} else {
				for _, n := range report.Notes {
					e.Logf("agent", "%s", n)
				}
				if rerr := report.err("dist-upgrade"); rerr != nil {
					return rerr
				}
			}
		}
	}
	e.Phase(phaseUpgrading)
	e.Logf("agent", "updating package lists")
	if err := d.Runner.Run(ctx, e.Logf, "apt-get", "update"); err != nil {
		return err
	}
	e.Logf("agent", "running dist-upgrade")
	if err := d.Runner.Run(ctx, e.Logf, "apt-get",
		"-o", "Dpkg::Options::=--force-confdef",
		"-o", "Dpkg::Options::=--force-confold",
		"-y", "dist-upgrade"); err != nil {
		return err
	}

	needs := rebootRequired()
	switch {
	case spec.Reboot == rebootNever:
		if needs {
			e.Logf("agent", "a reboot is required but reboot=never; this host is running an old kernel")
		}
		return nil
	case spec.Reboot == "if-needed" && !needs:
		e.Logf("agent", "no reboot required")
		return nil
	}

	down := time.Duration(spec.ExpectedDownSeconds) * time.Second
	if down <= 0 {
		down = 15 * time.Minute
	}

	// DECLARE, THEN ACT — and refuse to act if the declaration did not land.
	//
	// Without this the control plane sees a hypervisor stop answering and cannot
	// tell that from a failure, because nothing distinguishes them. The whole
	// unavailability model rests on the announcement arriving BEFORE the
	// silence, so a failure to announce has to stop the reboot.
	e.Phase(phaseRebooting)
	if err := e.Declare(ctx, down, "rebooting after dist-upgrade"); err != nil {
		return fmt.Errorf("could not tell the control plane this host is about to reboot, so it "+
			"would read the reboot as an outage; refusing to reboot: %w", err)
	}
	e.Logf("agent", "declared up to %s of expected downtime; rebooting now", down)

	return d.Reboot(ctx)
}

// apiSettle bounds the wait for pmxcfs to come back after a reboot.
//
// Vars rather than consts so a test can shorten them.
var (
	apiSettleTimeout  = 3 * time.Minute
	apiSettleInterval = 5 * time.Second
)

func verifyAfterReboot(ctx context.Context, d Deps, e *Env) error {
	// THE HOST HAS JUST BOOTED, so pmxcfs may not be mounted yet. Reading once
	// and failing turned "pve-cluster has not started in the second after a
	// reboot" into a failed task, and a failed task is collected again —
	// observed as a dist-upgrade reaching attempt 39.
	//
	// The wait is much cheaper than it was. It used to be waiting for PVEPROXY,
	// because cluster state came from this node's own HTTP API — minutes of a
	// web server starting, for a fact that was on local disk the whole time.
	// Now it waits for pve-cluster to mount /etc/pve, which is quick. The bound
	// stays generous because being wrong in that direction costs a slow task and
	// being wrong in the other costs a failed one.
	//
	// Waited for, bounded, and then not fatal. See below.
	deadline := time.Now().Add(apiSettleTimeout)
	var err error
	for {
		err = d.Cluster.Status(ctx)
		if err == nil || isNotClustered(err) || time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(apiSettleInterval):
		}
	}

	if err != nil {
		if isNotClustered(err) {
			e.Logf("agent", "host is up; not a cluster member")
			return nil
		}
		// COULD NOT ASK, not WAS TOLD NO — the same distinction the preflight
		// makes. The upgrade has already happened and the host has already come
		// back; failing the task now cannot undo either, and the control plane
		// gated this before it was queued. A node that has left a cluster has no
		// local API credential at all, so it can never answer this.
		e.Logf("agent", "host is back; the cluster could not be read after %s (%v), so its "+
			"state is not confirmed here", apiSettleTimeout, err)
		return nil
	}

	if !d.Cluster.Quorate() {
		// Reported as a failure, not a note. A node that returns to a cluster
		// that has lost quorum is the moment a rotation must stop, before the
		// next host is taken down.
		return fmt.Errorf("host is back but the cluster is NOT quorate; stop the rotation before " +
			"taking another node down")
	}
	e.Logf("agent", "host is back and the cluster is quorate")
	return nil
}

// ------------------------------------------------------------------------ join

type JoinSpec struct {
	// Peer is an existing member to join through.
	Peer string `json:"peer"`
	// Fingerprint of that peer's certificate. Proxmox requires it, and it is
	// what stops a join being pointed at an impostor.
	Fingerprint string `json:"fingerprint"`
	Force       bool   `json:"force"`
}

// Join adds this host to a cluster.
//
// The password is NOT in the spec, deliberately. Task specs are stored as jsonb
// in the control database and retained after the task completes; a cluster root
// password there would be a standing credential in a table many people can read,
// which is the same mistake the queued-email design exists to avoid. The bee
// reads it from local configuration placed on the host at provisioning time.
func Join(ctx context.Context, d Deps, e *Env, rawSpec string) error {
	if err := requireCluster(d); err != nil {
		return err
	}

	var spec JoinSpec
	if err := json.Unmarshal([]byte(orEmptyJSON(rawSpec)), &spec); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}
	if spec.Peer == "" {
		return fmt.Errorf("a peer to join through is required")
	}
	if spec.Fingerprint == "" {
		return fmt.Errorf("the peer's certificate fingerprint is required; without it the join " +
			"cannot tell the intended cluster from anything else answering on that address")
	}

	e.Phase(phasePreflight)
	// A host with guests must not join: Proxmox refuses, and the failure mode if
	// it did not is that the joining node's guest configuration is replaced by
	// the cluster's.
	guests, err := d.Cluster.GuestsOn(ctx, d.Node)
	if err == nil && len(guests) > 0 {
		return fmt.Errorf("this host has %d guest(s); joining a cluster discards local guest "+
			"configuration, so they must be moved or removed first", len(guests))
	}

	e.Phase(phaseJoining)
	e.Logf("agent", "joining the cluster through %s", spec.Peer)
	force := spec.Force
	status, err := d.Cluster.JoinCluster(ctx, &proxmoxJoinOptions{
		Hostname:    spec.Peer,
		Fingerprint: spec.Fingerprint,
		Force:       &force,
	})
	if err != nil {
		return fmt.Errorf("join: %w", err)
	}
	e.Logf("agent", "join accepted: %s", strings.TrimSpace(status))

	e.Phase(phaseVerifying)
	return verifyJoined(ctx, d, e)
}

// joinSettleTimeout bounds the wait for a join to take effect.
//
// The join restarts pve-cluster and corosync on this host, so its own API stops
// answering for a stretch in the middle. Minutes, not seconds — and a deadline
// rather than a fixed number of attempts, because what matters is how long an
// operator waits before being told something definite.
// Vars rather than consts so a test can shorten them; nothing in the agent
// writes to them at runtime.
var (
	joinSettleTimeout = 4 * time.Minute
	joinPollInterval  = 5 * time.Second
)

// verifyJoined waits until this host is actually a cluster member.
//
// POST /cluster/config/join returns once the request is ACCEPTED. The work runs
// as a task on the node afterwards, and it can fail there — a wrong password, a
// peer that is itself not quorate, corosync unable to reach the ring address.
// Returning nil on acceptance reported every one of those to the control plane
// as a completed join, so the task went green and the node never joined. The
// same mistake as trusting `pvecm add`'s exit code, one layer up.
//
// Membership is read from THIS host's own cluster status, which is the only
// thing that answers "am I in" rather than "was I told to try".
func verifyJoined(ctx context.Context, d Deps, e *Env) error {
	deadline := time.Now().Add(joinSettleTimeout)
	var last error

	for {
		err := d.Cluster.Status(ctx)
		switch {
		case err == nil && d.Cluster.MemberCount() > 0:
			// Quorum is NOT required here. A node joining a cluster whose other
			// members are down has joined correctly and has a real problem, and
			// those are different facts that deserve different messages.
			names := d.Cluster.NodeNames()
			if !d.Cluster.Quorate() {
				e.Logf("agent", "joined (members: %s) but the cluster is NOT quorate",
					strings.Join(names, ", "))
				return fmt.Errorf("this host joined the cluster, but the cluster is not quorate "+
					"(members: %s); the join itself succeeded, so do not retry it — find out why "+
					"the other members are not voting", strings.Join(names, ", "))
			}
			e.Logf("agent", "joined and quorate; members: %s", strings.Join(names, ", "))
			return nil

		case err == nil:
			// Reachable, reports a cluster, and claims no members. Not a state to
			// accept as success.
			last = fmt.Errorf("cluster status reports no members")

		case isNotClustered(err):
			// The expected state for most of this loop: the join has not taken
			// effect yet, or pve-cluster is mid-restart.
			last = err

		default:
			// The API is down while the services restart. Also expected.
			last = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("the join was accepted but this host is still not a cluster member "+
				"after %s; the last thing its API said was: %w", joinSettleTimeout, last)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(joinPollInterval):
		}
	}
}

// ----------------------------------------------------------------------- evict

type EvictSpec struct {
	// Node to remove. NOT this one — see below.
	Node string `json:"node"`
}

// Evict removes another node from the cluster.
//
// This task is issued to a node that will REMAIN, never to the one being
// removed: `pvecm delnode` is executed by a surviving member, and a node cannot
// delete itself from a cluster it is about to stop being part of.
//
// The removed host must then never be powered back on with its old
// configuration — it will try to rejoin a cluster that no longer expects it.
// That warning is logged into the task output because it is the part operators
// forget, and the consequence lands on the cluster rather than the host.
func Evict(ctx context.Context, d Deps, e *Env, rawSpec string) error {
	if err := requireCluster(d); err != nil {
		return err
	}

	var spec EvictSpec
	if err := json.Unmarshal([]byte(orEmptyJSON(rawSpec)), &spec); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}
	if spec.Node == "" {
		return fmt.Errorf("the node to remove is required")
	}
	if strings.EqualFold(spec.Node, d.Node) {
		return fmt.Errorf("this task removes %q, which is the host running it; a node cannot "+
			"remove itself from the cluster — issue this to a node that will remain", spec.Node)
	}

	e.Phase(phasePreflight)
	r := &PreflightReport{}

	if err := d.Cluster.Status(ctx); err != nil {
		return fmt.Errorf("read cluster status: %w", err)
	}
	if !d.Cluster.Quorate() {
		// QUORUM IS RESTORED BEFORE THE REMOVAL, NOT ONLY AFTER, and the ordering
		// is load-bearing.
		//
		// `pvecm delnode` rewrites corosync.conf, which needs a writable /etc/pve
		// — and a non-quorate node has neither. So a cluster that has ALREADY lost
		// quorum cannot be repaired by removing the node that cost it: the removal
		// is the thing that is blocked. `pvecm expected` lowers the runtime
		// expectation without writing anything, which is the only way back in.
		//
		// Only when every absent member is one the control plane has recorded as
		// DEPARTING. That is the whole safety argument: lowering the expectation
		// while a node that should still be there is merely unreachable would let
		// this side of a partition declare itself quorate and act. A departing
		// node is not coming back, so its vote is not one to keep waiting for.
		if err := restoreQuorumBeforeRemoval(ctx, d, e, spec.Node); err != nil {
			r.blocked("the cluster is not quorate and quorum could not be restored first, so "+
				"removing a node cannot be agreed on: %v", err)
		}
	}
	members := d.Cluster.MemberCount()
	if !quorumSurvivesLosing(members, 1) {
		// A NOTE, NOT A REFUSAL, because the remedy runs on this node right after
		// the removal.
		//
		// Removing one of two leaves a survivor below the expected vote count and
		// its /etc/pve read-only. `pvecm expected 1` lowers the runtime
		// expectation and brings it back, and it CANNOT be done in advance —
		// corosync refuses to set expected below the votes currently present, so
		// it is only valid once the removed node has stopped voting.
		//
		// This node is the right place for it: `pvecm expected` affects only the
		// host it runs on, and under per-cluster convergence the node doing the
		// reconciling is the one that needs it lowered. Refusing here instead
		// would make a two-node hive impossible to shrink, which the departure
		// path has always supported.
		r.note("removing a node leaves %d of %d, so quorum will be restored on this node "+
			"afterwards with expected=%d", members-1, members, members-1)
	}

	known := false
	for _, n := range d.Cluster.NodeNames() {
		if strings.EqualFold(n, spec.Node) {
			known = true
			break
		}
	}
	if !known {
		r.blocked("%q is not a member of this cluster (members: %s)",
			spec.Node, strings.Join(d.Cluster.NodeNames(), ", "))
	}

	if guests, err := d.Cluster.GuestsOn(ctx, spec.Node); err != nil {
		r.blocked("could not list guests on %q, so it is not known whether removing it strands "+
			"anything: %v", spec.Node, err)
	} else if len(guests) > 0 {
		names := make([]string, 0, len(guests))
		for _, g := range guests {
			names = append(names, fmt.Sprintf("%s/%s", g.Type, g.ID))
		}
		// No override, deliberately. There used to be an allow_guest_loss flag
		// and it was wrong, because what it caused is not downtime — it is
		// ORPHANED GUESTS that nothing can adopt.
		//
		// Removing a node drops its guest configuration from this cluster's
		// view. If that node then joins another datacenter, the join DISCARDS
		// its local guest configuration. So the disks survive on storage while
		// both clusters believe the guest is somebody else's problem: the one it
		// left has forgotten it, and the one it joined never learned about it.
		// There is no supported path back — recovery means hand-editing config
		// under /etc/pve, which is precisely the position a fleet tool exists to
		// keep operators out of.
		//
		// The alternative is always available, including for hardware that is
		// already dead: reassign the guests to a surviving node FIRST, then
		// evict. Adoption before eviction is recoverable at every step;
		// eviction before adoption is recoverable at none.
		r.blocked("%q still has %d guest(s) registered (%s).\n"+
			"    Removing it now would orphan them: this cluster drops their configuration when the "+
			"node leaves, and if that node later joins another datacenter the join discards its "+
			"local guest configuration too. The disks would remain on storage with no cluster "+
			"owning them, and no supported way to adopt them back.\n"+
			"    Reassign them to a node that is staying first — by live migration if the host is "+
			"up, or by moving their configuration to a surviving node if it is already dead — and "+
			"then evict.",
			spec.Node, len(guests), strings.Join(names, ", "))
	}

	for _, n := range r.Notes {
		e.Logf("agent", "%s", n)
	}
	if err := r.err("evict"); err != nil {
		return err
	}

	e.Phase(phaseEvicting)

	// CLEAR A STALE LOCK FIRST. A delnode that died part-way leaves
	// /etc/pve/priv/lock/file-corosync_conf behind, and every retry then fails
	// with "cfs-lock 'file-corosync_conf' error: got lock request timeout" — a
	// message that points at contention rather than at a leftover directory.
	// Without clearing it the removal can never complete and the cluster stays
	// wedged.
	// Nil-guarded rather than assumed. A Deps without a Runner is a real shape —
	// a host that is not a Proxmox node builds one — and a nil dereference three
	// calls deep reports as a segfault address, which is the failure requireCluster
	// exists to keep out of the journal.
	if d.Runner != nil {
		if err := d.Runner.Run(ctx, discard, "sh", "-c",
			"rmdir "+cfsLockDir+" 2>/dev/null || true"); err != nil {
			e.Logf("agent", "could not clear a stale cfs lock (%v); continuing", err)
		}
	}

	e.Logf("agent", "removing %s from the cluster", spec.Node)
	if err := d.Cluster.DeleteConfigNode(ctx, spec.Node); err != nil {
		return fmt.Errorf("remove %s: %w", spec.Node, err)
	}

	// RESTORE QUORUM, because delnode on a small cluster is SELF-BLOCKING.
	//
	// Removing the node drops this one below the expected vote count, and the
	// very next thing delnode does — rewriting corosync.conf to lower that count
	// — needs the writable /etc/pve it just lost. It fails with a bare
	// "Permission denied" on a temp file and leaves the cluster read-only, guests
	// unmanageable, and the config still naming the departed node.
	//
	// `pvecm expected <n>` lowers the runtime expectation without writing
	// anything, which is the only way back in. It CANNOT be done in advance:
	// corosync refuses to set expected below the votes currently present, so this
	// is only valid once the removed node has actually stopped voting.
	if err := restoreQuorumAfterRemoval(ctx, d, e, members-1); err != nil {
		return err
	}

	e.Logf("agent", "%s removed.", spec.Node)
	e.Logf("agent", "IMPORTANT: do not power %s back on with its existing configuration. A removed "+
		"node still believes it is a member and will try to rejoin, which can corrupt the "+
		"cluster's state. Reinstall it before it is used again.", spec.Node)
	return nil
}

func orEmptyJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}

// cfsLockDir is the lock a half-finished delnode leaves behind.
var cfsLockDir = "/etc/pve/priv/lock/file-corosync_conf"

// restoreQuorumAfterRemoval lowers this node's expected votes when the removal
// cost it quorum.
//
// Only this node's, which is exactly right: `pvecm expected` affects the node it
// runs on, and under per-cluster convergence the node doing the reconciling IS
// the one that needs it lowered. That was awkward when a central job had to
// choose a host to run it on; here it falls out.
//
// Proven rather than assumed: everything a cluster does next writes to pmxcfs
// and would fail obscurely on a still-read-only filesystem.
func restoreQuorumAfterRemoval(ctx context.Context, d Deps, e *Env, expect int) error {
	if expect < 1 {
		return nil
	}
	if err := d.Cluster.Status(ctx); err == nil && d.Cluster.Quorate() {
		return nil
	}
	e.Logf("agent", "the removal cost this node quorum; lowering expected votes to %d", expect)
	if err := d.Cluster.Expected(ctx, expect); err != nil {
		return fmt.Errorf("restore quorum with expected=%d after removing a node: %w", expect, err)
	}
	if err := d.Cluster.Status(ctx); err != nil || !d.Cluster.Quorate() {
		return fmt.Errorf("expected votes were lowered to %d but this node is still not quorate, "+
			"so /etc/pve stays read-only and the cluster cannot be managed", expect)
	}
	e.Logf("agent", "quorum restored")
	return nil
}

// restoreQuorumBeforeRemoval makes /etc/pve writable again so a removal can
// proceed.
//
// The node being removed is passed in because it is the one absence this is
// allowed to discount. Any OTHER missing member means the cluster is degraded
// for a reason a departure does not explain, and lowering the expectation there
// would let a minority act.
func restoreQuorumBeforeRemoval(ctx context.Context, d Deps, e *Env, departing string) error {
	// ONLINE members, not configured ones. A departing node's entry stays in
	// corosync.conf until it is removed — which is the very thing being decided —
	// so reading the configured list here answers the opposite question.
	present := d.Cluster.OnlineNodeNames()
	for _, n := range present {
		if strings.EqualFold(n, departing) {
			// It is still voting, so the cluster is non-quorate for some other
			// reason and this is not ours to override.
			return fmt.Errorf("%q is still a voting member, so the lost quorum is not explained "+
				"by its departure", departing)
		}
	}
	// CLUSTER CONFIG MINUS ONE, not the number of nodes currently online.
	//
	// The two agree when the departing node is the only absence, and diverge
	// exactly where it matters: with three configured, one departing and one
	// merely down, the online count is 1 and would let a single node declare
	// itself quorate. Config minus one is 2, which correctly still demands a
	// majority of what remains.
	votes := d.Cluster.MemberCount() - 1
	if votes < 1 {
		return fmt.Errorf("removing %q would leave no configured nodes", departing)
	}
	e.Logf("agent", "the cluster is not quorate and %s has stopped voting; lowering expected "+
		"votes to %d — the cluster's configured node count less the one leaving — so the "+
		"removal can be written", departing, votes)
	if err := d.Cluster.Expected(ctx, votes); err != nil {
		return fmt.Errorf("lower expected votes to %d: %w", votes, err)
	}
	if err := d.Cluster.Status(ctx); err != nil || !d.Cluster.Quorate() {
		return fmt.Errorf("expected votes were lowered to %d and this node is still not quorate",
			votes)
	}
	e.Logf("agent", "quorum restored; the removal can proceed")
	return nil
}
