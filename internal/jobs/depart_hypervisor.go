package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// ErrDepartureRefused means a gate said no. It is NOT a system failure, and a
// caller that maps it to a 500 puts an operator's own drain policy on a 5xx
// dashboard.
var ErrDepartureRefused = errors.New("depart: refused by a precondition")

// DepartDeps is what departure needs from the rest of the system, declared on
// the consumer side so the sequence is testable without a tenant database or a
// cluster to take apart.
type DepartDeps struct {
	// RecordOutcome stores why this departure failed, or clears it on success.
	//
	// A dependency rather than a direct write, so this package still needs no
	// tenant database handle of its own and the whole sequence can be driven by a
	// test with no Postgres.
	RecordOutcome func(ctx context.Context, orgID, hypervisorID uuid.UUID, operation string, err error) error

	// FleetKey returns the control plane's key for an organization.
	FleetKey func(ctx context.Context, orgID uuid.UUID) (pub string, priv []byte, err error)

	// Snapshot reads everything the gate decides on, in ONE transaction.
	//
	// One call rather than four, because the gate's inputs have to be mutually
	// consistent: a node count read after a placement count can miss a host that
	// arrived in between, and the whole value of the gate is that its verdict
	// corresponds to a state that actually existed.
	Snapshot func(ctx context.Context, orgID, hypervisorID uuid.UUID) (DepartureSnapshot, error)

	// NewRunner opens a connection to a host.
	NewRunner func(addr, user string, priv []byte) (Runner, error)

	// MarkFleetless records that the MACHINE now belongs to no hive: it clears
	// hive_id, swarm_id and the node name together.
	//
	// Keyed on the hypervisor, not the bee. Hive membership lived on both rows
	// until this model, with nothing keeping the two in agreement, and a
	// migration was exactly where they diverged — departure cleared the bee's
	// while the capacity sync deleted the hypervisor. There is one column now and
	// this is what writes it.
	//
	// All three columns go together. A swarm is a subdivision of a cluster the
	// machine has left, and ck_hypervisors_swarm_needs_hive rejects the row
	// outright if swarm_id outlives hive_id; the node name is the cluster's word
	// for this node and means nothing outside it — worse, leaving it would let the
	// next sync of some other hive match this machine by a name it no longer has.
	//
	// A real state, not a transitional lie. Written after the cluster work
	// succeeds, so a failure part-way leaves the machine still claiming the hive
	// it is being removed from, which is the direction an operator can
	// investigate. The opposite ordering produces a host that reports belonging
	// nowhere while still being a voting cluster member.
	MarkFleetless func(ctx context.Context, orgID, hypervisorID uuid.UUID) error

	// EnqueueCapacitySync asks for a fresh observation of one hive.
	//
	// A dependency rather than something the WORKER does after Depart returns,
	// because the composition test that proves this fires — the fleet cycle in
	// internal/service — calls Depart directly with no River client anywhere. A
	// trigger reachable only through Work() would be untestable by the one test
	// that exercises the whole lifecycle against real clusters.
	//
	// Optional, like MarkFleetless: a harness that does not care about the
	// refresh leaves it nil rather than stubbing it.
	EnqueueCapacitySync func(ctx context.Context, orgID, hiveID uuid.UUID) error

	// Thresholds govern how stale an observation may be and still support a
	// decision about emptiness.
	Thresholds tenant.DrainThresholds
}

// DepartureSnapshot is one consistent read of everything the gate needs.
type DepartureSnapshot struct {
	Hive       tenant.Hive
	Hypervisor tenant.Hypervisor

	// BeeID is the agent on the departing node, if one is enrolled. Nil is
	// legitimate: a cluster reports nodes nobody has installed an agent on.
	BeeID *uuid.UUID

	// HeldPlacements is how many placements on this node still hold capacity,
	// counted in the same transaction as everything else here.
	HeldPlacements int

	// HiveNodeCount includes the node that is leaving.
	HiveNodeCount int

	// RemainingNodes is the observed members left once this node and every
	// departure already requested have gone; the minimum is a floor on it.
	RemainingNodes int
	NotCounted     []string

	// DepartingAddress is where to reach the node being removed.
	DepartingAddress string

	// RemainingAddress is a node that STAYS, and is where delnode and the keyring
	// prune run. Both are needed: the cluster-side work cannot be done from the
	// node being removed, and the standalone conversion cannot be done from
	// anywhere else.
	RemainingAddress string

	// LastNode is true when the departing machine is the hive's ONLY member, so
	// there is no peer to perform the cluster-side removal from.
	//
	// Not an error, and distinguishing it from one is the point. An empty peer
	// set and a peer set whose addresses are all missing produced the same
	// message — "no remaining node has a resolvable address" — which sent an
	// operator to check DNS for a cluster that simply had nobody else in it.
	//
	// It also closed a dead end. Departing the last node was refused for want of
	// a peer, and deleting the cluster is refused while it holds a hypervisor, so
	// a one-node cluster could be neither dismantled nor emptied. Removing the
	// last node is not a node removal at all: there is no delnode to run and no
	// survivor to keep quorate, only the standalone conversion on the node
	// itself. That is the mirror of creating a cluster on a first node.
	LastNode bool

	// PeerNames are the other nodes' names, whose inherited guest-config
	// directories have to be cleared off the departing node before it can join
	// anything. Excludes the departing node's own name.
	PeerNames []string

	// SSHUser is the account to connect as. Empty means root.
	SSHUser string
}

// DepartInput identifies one host to remove from its hive.
type DepartInput struct {
	OrganizationID uuid.UUID
	HypervisorID   uuid.UUID

	// CredentialDisposition is the operator's answer about the cluster secrets
	// this node has seen. Carried per departure rather than configured, because
	// it is a judgement about ONE host leaving ONE cluster on ONE occasion — the
	// same host may be trusted this week and not the next. Empty refuses.
	CredentialDisposition string
}

// Depart removes one host from its hive, leaving it standalone and fleetless.
//
// It does NOT join anything. Departure and arrival are separate operations with
// separate gates, and the state between them — a host in no hive — is a real
// state this model can express rather than a window to be hurried through. A
// join that fails then leaves a standalone host, not a half-migrated one.
//
// The sequence, where each step's ORDER is load-bearing:
//
//  1. Gate. Every precondition, all reported at once. Nothing touches a host
//     until the whole list is empty.
//  2. Read the departing node's client keys, WHILE it is still reachable and
//     still a member. This is the only window in which the material to prune by
//     can be read from the node itself.
//  3. PREFLIGHT the node that stays: quorate, writable, and its own config
//     agreeing with ours about the membership. Then stop the departing node's
//     cluster stack, so it cannot write to the cluster's pmxcfs — in particular
//     cannot re-add its own key after the prune.
//  4. delnode from a remaining node, restoring quorum if the removal cost it.
//  5. PRUNE the departing node's keys from the remaining cluster's keyring, and
//     verify. This needs quorum, which is why it comes after step 4 and not
//     between the kill and the recovery.
//  6. REVOKE any ephemeral join key naming the departing node. A separate file
//     from the keyring, and a separate leak: pruning only the keyring left a
//     departed node with a working root shell on the hive it left.
//  7. Remove the departed node's directory from the remaining cluster.
//  8. Convert the departing node to standalone.
//  9. Record it as fleetless.
//  10. Ask for a fresh capacity observation of the hive it left, so placement
//     does not spend the rest of the sync interval deciding against a fleet
//     that still contains this node.
//
// Steps 3 through 7 are individually idempotent, because River retries and a
// departure interrupted half-way must converge rather than accumulate.
func Depart(ctx context.Context, d DepartDeps, in DepartInput) error {
	snap, err := d.Snapshot(ctx, in.OrganizationID, in.HypervisorID)
	if err != nil {
		return fmt.Errorf("depart: read state: %w", err)
	}

	drain := tenant.DrainStateAt(time.Now().UTC(), snap.Hypervisor, snap.HeldPlacements, d.Thresholds)
	refusals := tenant.CheckDeparture(tenant.DepartureRequest{
		Hive:                  snap.Hive,
		Hypervisor:            snap.Hypervisor,
		Drain:                 drain,
		HiveNodeCount:         snap.HiveNodeCount,
		RemainingNodes:        snap.RemainingNodes,
		NotCounted:            snap.NotCounted,
		CredentialDisposition: in.CredentialDisposition,
	})
	if len(refusals) > 0 {
		// Wrapped so a caller can distinguish a policy refusal from a broken
		// cluster, and joined into one message so the operator sees the whole list
		// rather than fixing one gate per worker run.
		return fmt.Errorf("%w: %w", ErrDepartureRefused, tenant.DepartureRefusalsError(refusals))
	}

	if snap.DepartingAddress == "" {
		return fmt.Errorf("depart: the departing node has no address recorded, and the " +
			"standalone conversion cannot be run from anywhere else")
	}
	if !snap.LastNode && snap.RemainingAddress == "" {
		return fmt.Errorf("depart: a remaining node's address is required — the cluster-side " +
			"removal cannot be run from the node being removed")
	}

	_, priv, err := d.FleetKey(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("depart: fleet key: %w", err)
	}
	user := snap.SSHUser
	if user == "" {
		user = "root"
	}

	departing, closeDeparting, err := connect(ctx, d, snap.DepartingAddress, user, priv)
	if err != nil {
		return fmt.Errorf("depart: connect to the departing node: %w", err)
	}
	defer closeDeparting()

	// No peer to connect to when this is the last node: there is no cluster-side
	// work, so there is nothing to connect for.
	var remaining Runner
	if !snap.LastNode {
		r, closeRemaining, cerr := connect(ctx, d, snap.RemainingAddress, user, priv)
		if cerr != nil {
			return fmt.Errorf("depart: connect to a remaining node: %w", cerr)
		}
		defer closeRemaining()
		remaining = r
	}

	// BEFORE the stack stops. Once the node is out of the cluster its keys are
	// still on disk, but reading them is one more thing that can fail at a point
	// where the prune can no longer be identified.
	keys, err := ReadNodeClientKeys(ctx, departing)
	if err != nil {
		return err
	}

	// Also before the stack stops, and for the same reason: /etc/pve is a fuse
	// mount that pve-cluster provides, so once it is stopped there is nothing
	// there to fingerprint. These digests are what MakeStandalone proves its
	// destruction against.
	secretsBefore, err := ReadClusterSecretDigests(ctx, departing)
	if err != nil {
		return err
	}

	// ALREADY DETACHED? Then this is a RETRY of a departure that got part way,
	// and the steps that remain are the ones after the detach.
	//
	// A departure is not one action, it is a sequence with an irreversible step
	// in the middle: once delnode has run, the node is out and no amount of
	// retrying puts it back. Everything after that — destroying the cluster's
	// secrets on it, pruning its keys from the keyring, recording the outcome —
	// can still fail, and did: a disarm failure left a node detached and armed,
	// and every retry then re-ran from the top, hit the preflight below, and
	// refused with "the cluster config lists 1 node(s)". Four attempts, none of
	// which could reach the one step that still needed doing, and the job
	// cancelled with the node in exactly the state the check exists to prevent.
	//
	// Asked of the HOST, not of the record, for the same reason AssertStandalone
	// asks the host: the record says what we intended and corosync says what is.
	// The last node has no peer, so every cluster-side step below is skipped and
	// only the standalone conversion runs. Treated exactly like an already-
	// detached retry, because the work that remains is the same work.
	alreadyDetached, adErr := IsStandalone(ctx, departing)
	if adErr != nil {
		return fmt.Errorf("depart: establish whether this node is still a cluster member: %w", adErr)
	}

	// THE CLUSTER-SIDE REMOVAL IS NOT THIS JOB'S.
	//
	// A remaining member runs `pvecm delnode` itself: the departure publishes the
	// node into the composition's Departing list, every peer sees it, and the
	// deterministic owner among them removes it locally. The node that acts is
	// one that is STAYING, which is the only node that can — a departing node
	// cannot remove itself.
	//
	// Its preflight moves with it, onto the same remaining member: fleetops.Evict
	// refuses on lost quorum and on guests it would orphan, and where the removal
	// costs quorum it runs `pvecm expected` on ITSELF afterwards. That last part
	// is why the check belongs there rather than here — `pvecm expected` affects
	// only the host it runs on, and the reconciling node is the one that needs it
	// lowered.
	//
	// This job used to do all of it and won the race every time: it completed in
	// seconds while the peers were still on their poll interval, so convergent
	// eviction was armed and never once fired. What is left here is the part
	// convergence deliberately does NOT do — disarming the departed node, where
	// rotate versus accept-exposure is an operator's decision with a security
	// consequence rather than something derivable from desired state.
	skipClusterSide := true

	slog.InfoContext(ctx, "the cluster-side removal is left to a remaining member",
		"hypervisor_id", in.HypervisorID,
		"already_detached", alreadyDetached, "last_node", snap.LastNode)

	if !skipClusterSide {
		if err := StopClusterStack(ctx, departing); err != nil {
			return err
		}
	}

	// The cluster's own word for this node. A clustered machine record always has
	// one — the partial unique index on (hive_id, name) assumes it — but delnode
	// takes a name and a wrong or empty one removes the wrong member, so this is
	// checked rather than assumed.
	if snap.Hypervisor.Name == nil || *snap.Hypervisor.Name == "" {
		return fmt.Errorf("depart: this machine is recorded as a member of %q but has no node "+
			"name, so there is nothing to hand to delnode; run a capacity sync so the cluster's "+
			"own name for it is recorded", snap.Hive.Name)
	}
	nodeName := *snap.Hypervisor.Name

	// One fewer node than the hive has now. Skipped on a retry: the node is
	// already out, and DelNode would refuse with "no node to remove without
	// destroying the cluster" — which is true, and not what is being asked.
	if !skipClusterSide {
		if err := DelNode(ctx, remaining, nodeName, snap.HiveNodeCount-1); err != nil {
			return err
		}
	}

	// The step that closes the access hole. A failure here fails the whole
	// departure: leaving the key behind is worse than leaving the host in a
	// half-removed state, because the half-removed state is visible in
	// `pvecm status` and a stale credential is visible nowhere.
	//
	// Skipped for the last node: there is no keyring to prune it from, because
	// there is no cluster left to hold one.
	if !snap.LastNode {
		if _, err := PruneKeysFromClusterKeyring(ctx, remaining, keys); err != nil {
			return err
		}
	}

	// The other path to the same access. A keyring prune alone left a departed
	// node with a root shell on the hive it left, via the ephemeral key its
	// arrival had installed in the join file — verified on a live pair of
	// clusters, after the keyring prune had been confirmed correct.
	if !snap.LastNode {
		if _, err := RevokeJoinKeys(ctx, remaining, keys); err != nil {
			return err
		}
	}

	if !snap.LastNode {
		if err := RemoveNodeDir(ctx, remaining, nodeName); err != nil {
			return err
		}
	}

	// The keys of a node that STAYS, so the disarm can be proved on the property
	// that matters — that the departed host no longer trusts the cluster —
	// rather than on whether some bytes changed.
	//
	// Best effort: a node whose keys cannot be read yields an empty list, and an
	// empty list makes the check pass rather than fail. That is the right
	// direction here only because the check is one of several, and failing a
	// departure that has already detached the node strands it. The failure to
	// read is logged.
	var peerKeys []string
	var pkErr error
	if !snap.LastNode {
		peerKeys, pkErr = ReadNodeClientKeys(ctx, remaining)
	}
	if pkErr != nil {
		slog.WarnContext(ctx, "could not read a remaining node's keys, so the departed host's "+
			"trust in the cluster could not be verified", "error", pkErr)
	}

	if err := MakeStandalone(ctx, departing, snap.PeerNames, secretsBefore,
		DisarmCheck{
			PeerKeys: peerKeys,
			OwnKeys:  keys,
			// The operator's answer, finally acted on. It was carried into this
			// job's arguments and never read, so rotate and accept-exposure did
			// the same thing and only one of them said so.
			RotateIdentity: in.CredentialDisposition == tenant.CredentialsRotate,
		}); err != nil {
		return err
	}

	// THE MEMBERSHIP IS NOT CLEARED HERE. Doing so undid the departure that
	// started it.
	//
	// A departure MARKS the machine — departing_at set, hive_id and name kept —
	// so the composition can name it and a remaining member can evict it.
	// MarkFleetless knows nothing about that marker: it cleared hive_id and left
	// departing_at set, producing a row that was fleetless AND departing at once,
	// and removing the node from the composition before any peer had seen it.
	//
	// The consequence was not cosmetic. lab-a was left with pve alone below
	// quorum and /etc/pve read-only, still listing a pve2 that nothing would ever
	// remove — because the node it needed to evict had already vanished from the
	// document that authorises the eviction.
	//
	// FinaliseDepartures clears the row instead, on evidence from a remaining
	// member that the cluster has actually dropped the node. This job's own
	// success says only that the departing host was disarmed, which is a
	// different fact.

	// The Hypervisor row is deliberately NOT deleted here, and no longer because
	// the capacity sync owns it — it does not. THIS ROW IS THE MACHINE. Deleting
	// it would throw away the machine's identity, its address and its onboarding
	// history because it left a cluster, which is the failure the whole model
	// exists to remove. What departure clears is cluster membership, which is what
	// MarkFleetless above writes; the sync clears the same columns when it notices
	// the node has gone, and the two agree because they write the same thing.
	//
	// Leaving it is safe in the window before the next sync because the row is
	// still cordoned, and the placement query excludes cordoned rows under the
	// same lock it decides against — see tenant.SchedulablePredicate. That is also
	// why the gate insists on a cordon BEFORE any of this runs: the cordon, not
	// the departure, is what withdraws the host from Terraform's next apply.

	// The hive that just lost a node is re-observed NOW rather than at the next
	// tick. The capacity sync is the only writer of the observed columns and it
	// runs on a timer — three minutes by default — so without this the fleet
	// state describes a hive that still contains this machine for up to that
	// long, and everything downstream reasons from it.
	//
	// Enqueued AFTER the cluster work, never before. The sync writes
	// observed_guests and last_synced_at in the same statement, so a sync that
	// observed the pre-departure cluster would not merely be stale — it would
	// stamp a stale reading as FRESH, which is worse, because DrainStateAt and
	// AssertClusterHealthy's node-count agreement check are both built on
	// staleness being visible.
	//
	// One hive, not two. This host is fleetless now; it belongs to nothing else
	// there is anything to observe about. A migration syncs two hives by being
	// two operations — departure refreshes the source, arrival the destination.
	//
	// A failure here is LOGGED, NOT RETURNED, and the asymmetry with MarkFleetless
	// directly above is the point rather than an oversight. MarkFleetless writes
	// durable state nothing else will ever repair, so losing it has to fail the
	// operation. This is a freshness optimisation over state the scheduled sync
	// writes anyway, and correctness in the window is already held elsewhere — by
	// the cordon the placement query evaluates under its own lock, and by
	// DrainStateAt refusing on a stale observation. Returning the error would
	// fail a departure that ALREADY SUCCEEDED, and River would then retry it
	// forever: the gate still passes on the unchanged snapshot, and
	// AssertClusterHealthy refuses against a node count corosync has already
	// decremented — telling the operator to run a sync, which is precisely the
	// thing that failed. Failing the operation to protect an optimisation inverts
	// the priority.
	if d.EnqueueCapacitySync != nil && hiveIsSyncable(snap.Hive) {
		if err := d.EnqueueCapacitySync(ctx, in.OrganizationID, snap.Hive.ID); err != nil {
			slog.WarnContext(ctx, "depart: the host left its hive but a capacity sync could not "+
				"be triggered; the fleet view of that hive may be up to one sync interval out of "+
				"date until the scheduled sync runs, and a manual sync clears it immediately",
				"organization_id", in.OrganizationID, "hive_id", snap.Hive.ID, "error", err)
		}
	}
	return nil
}

// hiveIsSyncable answers whether a capacity sync for this hive could achieve
// anything at all.
//
// The same predicate as the fan-out's WHERE clause and the "sync now" endpoint's
// precondition, because it is the same rule: without an API URL and a bound
// credential the inventory read refuses before it opens a connection (see
// service.ProxmoxInventoryService.Inventory). Enqueuing anyway would queue a job
// that can only fail — with River's full retry schedule behind it — for a hive an
// operator has not finished configuring, which is noise generated by a
// bookkeeping refresh nobody asked for.
func hiveIsSyncable(h tenant.Hive) bool {
	return strings.TrimSpace(h.APIURL) != "" && h.CredentialID != nil
}

// connect opens a runner and returns a closer, handling the optional Connect and
// Close that the ssh implementation provides but the interface does not require.
func connect(ctx context.Context, d DepartDeps, addr, user string, priv []byte) (Runner, func(), error) {
	r, err := d.NewRunner(addr, user, priv)
	if err != nil {
		return nil, func() {}, err
	}
	if c, ok := r.(interface{ Connect(context.Context) error }); ok {
		if err := c.Connect(ctx); err != nil {
			return nil, func() {}, err
		}
	}
	closer := func() {}
	if c, ok := r.(interface{ Close() error }); ok {
		closer = func() { _ = c.Close() }
	}
	return r, closer, nil
}
