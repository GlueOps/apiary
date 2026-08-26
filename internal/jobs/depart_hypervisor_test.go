package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// departRunner records every command and answers the reads the sequence makes.
// Recording is the point: the ORDER of these steps is what can lock the control
// plane out of a host or leave a credential behind, and order is not observable
// from return values.
type departRunner struct {
	name string
	log  *[]string

	// keyringMatches is how many of the departing node's keys the cluster keyring
	// still holds. The prune sets it to zero unless pruneIsNoOp.
	keyringMatches int
	pruneIsNoOp    bool

	corosyncNodes int
	quorate       bool
	failOn        string

	// peer is the OTHER node in the pair. Stopping the departing node's cluster
	// stack is what costs the survivor its quorum on a small cluster, so the two
	// fakes are linked rather than each carrying an independent canned answer —
	// a survivor that simply starts non-quorate cannot distinguish "the departure
	// caused this" from "the cluster was already broken", and the preflight now
	// refuses the second.
	peer            *departRunner
	stopCostsQuorum bool
}

func (f *departRunner) Run(_ context.Context, cmd string) (string, error) {
	*f.log = append(*f.log, f.name+": "+cmd)

	if f.failOn != "" && strings.Contains(cmd, f.failOn) {
		return "boom", errors.New("command failed")
	}

	if f.stopCostsQuorum && strings.Contains(cmd, "systemctl stop pve-cluster") && f.peer != nil {
		f.peer.quorate = false
	}

	switch {
	case strings.Contains(cmd, "id_rsa.pub"):
		return "ssh-ed25519 AAAAKEYBODY root@hv5\n", nil
	case strings.Contains(cmd, "grep -cF -f"):
		return fmt.Sprintf("%d\n", f.keyringMatches), nil
	case strings.Contains(cmd, "grep -vF -f"):
		if !f.pruneIsNoOp {
			f.keyringMatches = 0
		}
		return "", nil
	case strings.Contains(cmd, "APIARY_QUORATE"):
		// The preflight's marker contract. A bare "Yes" is deliberately not
		// accepted, so the fake speaks the same protocol as the host.
		if f.quorate {
			return "APIARY_QUORATE=Yes\n", nil
		}
		return "APIARY_QUORATE=No\n", nil
	case strings.Contains(cmd, "Quorate"):
		if f.quorate {
			return "Yes\n", nil
		}
		return "No\n", nil
	case strings.Contains(cmd, "pvecm expected"):
		f.quorate = true
		return "", nil
	case strings.Contains(cmd, "APIARY_NODES"):
		// The marker contract the real command emits — a bare number is
		// deliberately not accepted, so the fake must speak the same protocol.
		return fmt.Sprintf("APIARY_NODES=%d\n", f.corosyncNodes), nil
	case strings.Contains(cmd, "pvecm delnode"):
		f.corosyncNodes--
		return "Killing node 2\n", nil
	}
	return "", nil
}

func (f *departRunner) Upload(context.Context, string, uint32, []byte) error { return nil }
func (f *departRunner) Reconnect(context.Context) error                      { return nil }

// syncRequest is one recorded call to EnqueueCapacitySync. Both ids are kept:
// the sync job is per (organization, hive), and a trigger that named the right
// hive in the wrong tenant's database would be indistinguishable from a correct
// one if only the hive were recorded.
type syncRequest struct {
	orgID  uuid.UUID
	hiveID uuid.UUID
}

type harness struct {
	log         []string
	departing   *departRunner
	remaining   *departRunner
	fleetless   []uuid.UUID
	syncs       []syncRequest
	syncErr     error
	orgID       uuid.UUID
	snapshot    DepartureSnapshot
	snapshotErr error
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{orgID: uuid.New()}
	hiveID, hypID, beeID := uuid.New(), uuid.New(), uuid.New()
	credID := uuid.New()
	now := time.Now().UTC()
	nodeName := "hv5"
	h.snapshot = DepartureSnapshot{
		// A hive the control plane can actually reach: an API URL and a bound
		// credential are what make a capacity sync worth enqueuing, so the default
		// harness describes a fully configured hive and the tests that care about
		// the unconfigured case take them away deliberately.
		Hive: tenant.Hive{
			ID: hiveID, Name: "dc1", MinimumNodes: 3,
			APIURL: "https://pve1.example:8006/api2/json", CredentialID: &credID,
		},
		Hypervisor: tenant.Hypervisor{
			ID: hypID, HiveID: &hiveID, Name: &nodeName,
			Address:     "10.0.0.5",
			Schedulable: false, ObservedGuests: 0,
			LastSyncedAt: &now, BeeID: &beeID,
		},
		BeeID:            &beeID,
		HeldPlacements:   0,
		HiveNodeCount:    3,
		DepartingAddress: "10.0.0.5",
		RemainingAddress: "10.0.0.1",
		PeerNames:        []string{"hv1", "hv2"},
	}
	h.departing = &departRunner{name: "departing", log: &h.log, quorate: true, corosyncNodes: 3}
	h.remaining = &departRunner{name: "remaining", log: &h.log, quorate: true, corosyncNodes: 3, keyringMatches: 1}
	return h
}

func (h *harness) deps() DepartDeps {
	return DepartDeps{
		FleetKey: func(context.Context, uuid.UUID) (string, []byte, error) {
			return "ssh-ed25519 PUB", []byte("PRIV"), nil
		},
		Snapshot: func(context.Context, uuid.UUID, uuid.UUID) (DepartureSnapshot, error) {
			return h.snapshot, h.snapshotErr
		},
		NewRunner: func(addr, _ string, _ []byte) (Runner, error) {
			if addr == h.snapshot.DepartingAddress {
				return h.departing, nil
			}
			return h.remaining, nil
		},
		MarkFleetless: func(_ context.Context, _, hypervisorID uuid.UUID) error {
			h.fleetless = append(h.fleetless, hypervisorID)
			return nil
		},
		EnqueueCapacitySync: func(_ context.Context, orgID, hiveID uuid.UUID) error {
			h.syncs = append(h.syncs, syncRequest{orgID: orgID, hiveID: hiveID})
			return h.syncErr
		},
	}
}

func (h *harness) run() error {
	return Depart(context.Background(), h.deps(), DepartInput{
		OrganizationID: h.orgID,
		HypervisorID:   h.snapshot.Hypervisor.ID,
		// These tests are about the removal SEQUENCE, so they answer the
		// credential question and move on. That the question must be answered at
		// all is covered by the gate's own tests; leaving it unset here would make
		// every one of them fail on the gate before reaching what it exercises.
		CredentialDisposition: tenant.CredentialsRotate,
	})
}

// The gate is wired into the job, not merely into the model — an unanswered
// disposition must stop a departure before any host is touched.
func TestDepartRefusesWithoutACredentialDecision(t *testing.T) {
	h := newHarness(t)
	err := Depart(context.Background(), h.deps(), DepartInput{
		OrganizationID: h.orgID,
		HypervisorID:   h.snapshot.Hypervisor.ID,
	})
	if err == nil {
		t.Fatal("a departure with no credential decision must be refused")
	}
	if !errors.Is(err, ErrDepartureRefused) {
		t.Fatalf("must be a policy refusal, not a failure: %v", err)
	}
	if !strings.Contains(err.Error(), tenant.RefuseCredentialsUndecided) {
		t.Fatalf("the refusal must name the code, got %v", err)
	}
	// The gate runs before either SSH session is opened, so a refused departure
	// must leave both hosts untouched — no stopped cluster stack on a node that
	// is then not going anywhere.
	if len(h.log) != 0 {
		t.Fatalf("nothing may reach either host before the gate passes; ran %v", h.log)
	}
}

func indexOf(log []string, substr string) int {
	for i, l := range log {
		if strings.Contains(l, substr) {
			return i
		}
	}
	return -1
}

func TestDepartHappyPath(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("a drained node in a 3-node hive must depart: %v", err)
	}
	// The MACHINE is recorded fleetless, not the agent. Membership lived on both
	// rows and they diverged during exactly this operation; there is one column
	// now and this is what writes it.
	if len(h.fleetless) != 1 || h.fleetless[0] != h.snapshot.Hypervisor.ID {
		t.Fatalf("the machine must be recorded fleetless exactly once, got %v", h.fleetless)
	}
}

func TestDepartRecordsAMachineWithNoAgentAsFleetless(t *testing.T) {
	// A cluster reports nodes nobody installed an agent on, and one of them can
	// still be departed. This used to be conditional on snap.BeeID — membership
	// lived on the bee, so a node with no bee had nowhere to record the change and
	// the departure silently left the fleet state wrong.
	h := newHarness(t)
	h.snapshot.BeeID = nil
	h.snapshot.Hypervisor.BeeID = nil

	if err := h.run(); err != nil {
		t.Fatalf("a drained agentless node must still depart: %v", err)
	}
	if len(h.fleetless) != 1 || h.fleetless[0] != h.snapshot.Hypervisor.ID {
		t.Fatalf("the machine must be recorded fleetless even with no agent enrolled, got %v",
			h.fleetless)
	}
}

func TestDepartOrderingIsLoadBearing(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	readKeys := indexOf(h.log, "id_rsa.pub")
	stopStack := indexOf(h.log, "systemctl stop pve-cluster")
	delnode := indexOf(h.log, "pvecm delnode")
	prune := indexOf(h.log, "grep -vF -f")
	standalone := indexOf(h.log, "pmxcfs -l")

	for name, i := range map[string]int{
		"read keys": readKeys, "stop stack": stopStack,
		"delnode": delnode, "prune": prune, "standalone": standalone,
	} {
		if i < 0 {
			t.Fatalf("%s never ran; log: %v", name, h.log)
		}
	}

	// Keys before the stack stops: it is the only window in which the material to
	// prune by can be read off the node.
	if readKeys > stopStack {
		t.Error("the departing node's keys must be read BEFORE its cluster stack is stopped")
	}
	// Stack stopped before delnode: otherwise the node can re-add its own key
	// through updatecerts after the prune, silently undoing it.
	if stopStack > delnode {
		t.Error("the departing node's stack must be stopped BEFORE delnode")
	}
	// Prune after delnode, because the prune writes to pmxcfs and needs the
	// quorum that delnode may have cost.
	if prune < delnode {
		t.Error("the keyring prune must come AFTER delnode so quorum has been restored")
	}
	// Standalone conversion last: it destroys the node's view of the cluster.
	if standalone < prune {
		t.Error("the standalone conversion must come after the prune")
	}
}

func TestDepartRefusalTouchesNothing(t *testing.T) {
	h := newHarness(t)
	h.snapshot.Hypervisor.ObservedGuests = 2 // not drained

	err := h.run()
	if !errors.Is(err, ErrDepartureRefused) {
		t.Fatalf("want ErrDepartureRefused, got %v", err)
	}
	if !strings.Contains(err.Error(), tenant.RefuseGuestsPresent) {
		t.Errorf("the refusal must name the failing gate, got %q", err.Error())
	}
	// The whole point of a gate: nothing on any host is touched.
	if len(h.log) != 0 {
		t.Fatalf("a refused departure must run NO commands on any host; ran %v", h.log)
	}
	if len(h.fleetless) != 0 {
		t.Error("a refused departure must not mark the bee fleetless")
	}
}

func TestDepartRefusesUncordonedHost(t *testing.T) {
	h := newHarness(t)
	h.snapshot.Hypervisor.Schedulable = true

	err := h.run()
	if !errors.Is(err, ErrDepartureRefused) || !strings.Contains(err.Error(), tenant.RefuseNotCordoned) {
		t.Fatalf("an uncordoned host must be refused as not_cordoned, got %v", err)
	}
	if len(h.log) != 0 {
		t.Fatalf("nothing should have run; ran %v", h.log)
	}
}

func TestDepartRefusesTwoNodeHive(t *testing.T) {
	h := newHarness(t)
	h.snapshot.HiveNodeCount = 2

	err := h.run()
	if !errors.Is(err, ErrDepartureRefused) || !strings.Contains(err.Error(), tenant.RefuseHiveTooSmall) {
		t.Fatalf("a 2-node hive must be refused as hive_too_small, got %v", err)
	}
}

func TestDepartFailsWhenThePruneDoesNotStick(t *testing.T) {
	h := newHarness(t)
	// The prune runs and reports success but the key is still there — the exact
	// shape of the real defect, where delnode and updatecerts both "succeed"
	// having removed nothing.
	h.remaining.pruneIsNoOp = true

	err := h.run()
	if err == nil {
		t.Fatal("a prune that leaves the key behind must fail the departure")
	}
	if !strings.Contains(err.Error(), "STILL in") {
		t.Errorf("the error must say the key is still present, got %q", err.Error())
	}
	// The host must NOT be recorded as cleanly departed: that is the state an
	// operator would stop looking at.
	if len(h.fleetless) != 0 {
		t.Error("a failed prune must not mark the bee fleetless")
	}
	// And the standalone conversion must not have run — the node stays visibly
	// half-removed rather than looking finished.
	if indexOf(h.log, "pmxcfs -l") >= 0 {
		t.Error("the standalone conversion must not run after a failed prune")
	}
}

func TestDepartRestoresQuorumBeforePruning(t *testing.T) {
	h := newHarness(t)
	// The observed failure, modelled as it actually occurs: the cluster is
	// healthy when the departure begins, and STOPPING the departing node is what
	// costs the survivor its quorum — leaving pmxcfs read-only, so the prune,
	// which is a write to pmxcfs, cannot land until quorum is restored.
	h.departing.peer = h.remaining
	h.departing.stopCostsQuorum = true

	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	expected := indexOf(h.log, "pvecm expected")
	prune := indexOf(h.log, "grep -vF -f")
	if expected < 0 {
		t.Fatal("quorum must be restored when delnode costs it")
	}
	if expected > prune {
		t.Error("quorum must be restored BEFORE the prune, which writes to pmxcfs")
	}
}

func TestDepartClearsStaleCfsLockBeforeDelnode(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	lock := indexOf(h.log, "rmdir /etc/pve/priv/lock/file-corosync_conf")
	delnode := indexOf(h.log, "pvecm delnode")
	if lock < 0 {
		t.Fatal("the stale cfs lock must be cleared — a failed delnode leaves one that blocks every retry")
	}
	if lock > delnode {
		t.Error("the lock must be cleared before delnode is attempted")
	}
}

func TestDepartClearsPeerNodeDirsOnTheDepartingHost(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// The inherited replica of the peers' guest configs, which a later join reads
	// as this host's own guests and refuses on.
	for _, peer := range h.snapshot.PeerNames {
		if indexOf(h.log, "rm -rf '/etc/pve/nodes/"+peer+"'") < 0 {
			t.Errorf("peer directory %q must be removed from the departing host; log: %v", peer, h.log)
		}
	}
	// And its OWN directory must survive — removing it would leave the node with
	// no config of its own.
	if indexOf(h.log, "rm -rf '/etc/pve/nodes/hv5'") >= 0 &&
		strings.Contains(h.log[indexOf(h.log, "rm -rf '/etc/pve/nodes/hv5'")], "departing") {
		t.Error("the departing node must NOT delete its own node directory")
	}
}

func TestDepartRefusesWhenAKeyCannotBeIdentified(t *testing.T) {
	h := newHarness(t)
	h.departing.failOn = "id_rsa.pub"

	err := h.run()
	if err == nil {
		t.Fatal("unable to read the node's keys must fail: there is nothing to prune by")
	}
	if indexOf(h.log, "pvecm delnode") >= 0 {
		t.Error("delnode must not run when the prune cannot be identified — the node would " +
			"leave the cluster keeping root access to it")
	}
}

func TestDepartRequiresBothAddresses(t *testing.T) {
	for _, tc := range []struct{ name, departing, remaining string }{
		{"no departing address", "", "10.0.0.1"},
		{"no remaining address", "10.0.0.5", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.snapshot.DepartingAddress = tc.departing
			h.snapshot.RemainingAddress = tc.remaining
			if err := h.run(); err == nil {
				t.Fatal("both addresses are required")
			}
		})
	}
}

func TestDepartMarksFleetlessOnlyAfterTheClusterWork(t *testing.T) {
	h := newHarness(t)
	// The standalone conversion is the last cluster-side step; if it fails the
	// host is still a cluster member as far as anything can tell.
	h.departing.failOn = "pmxcfs -l"

	if err := h.run(); err == nil {
		t.Fatal("a failed standalone conversion must fail the departure")
	}
	if len(h.fleetless) != 0 {
		t.Error("the bee must not be recorded fleetless while the host is still a cluster member")
	}
}

func TestDepartRefusesAnAlreadyDegradedCluster(t *testing.T) {
	// The preflight. Stopping a node in a cluster that has already lost quorum
	// deepens the outage and delnode refuses anyway — leaving a stopped node, an
	// unchanged config, and an operator holding a quorum message that says
	// nothing about which of the two problems they now have.
	h := newHarness(t)
	h.remaining.quorate = false

	err := h.run()
	if err == nil {
		t.Fatal("a departure must not begin against a non-quorate cluster")
	}
	if !strings.Contains(err.Error(), "NOT quorate before the departure has begun") {
		t.Errorf("the error must say the cluster was already degraded, got %q", err.Error())
	}
	// Nothing may have been stopped: that is the whole point of a preflight.
	if indexOf(h.log, "systemctl stop pve-cluster") >= 0 {
		t.Error("the departing node's stack must not be stopped when the preflight fails")
	}
	if indexOf(h.log, "pvecm delnode") >= 0 {
		t.Error("delnode must not be attempted")
	}
}

func TestDepartRefusesWhenTheClusterAndThisSystemDisagreeOnNodeCount(t *testing.T) {
	// Two independent views of the same fact. They diverge exactly when the
	// capacity sync is stale — which is the same moment the node count the
	// hive-size gate just decided on was wrong, so proceeding would act on a
	// verdict reached from bad input.
	h := newHarness(t)
	h.snapshot.HiveNodeCount = 5 // this system believes 5; the cluster says 3

	err := h.run()
	if err == nil {
		t.Fatal("a disagreement about the node count must stop the departure")
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Errorf("the error must point at the stale observation, got %q", err.Error())
	}
	if indexOf(h.log, "systemctl stop pve-cluster") >= 0 {
		t.Error("nothing may be stopped on a disagreement")
	}
}

func TestDepartRefusesWhenTheClusterHasOnlyOneNodeLeft(t *testing.T) {
	h := newHarness(t)
	h.snapshot.HiveNodeCount = 1
	h.remaining.corosyncNodes = 1

	if err := h.run(); err == nil {
		t.Fatal("there is no node to remove from a single-node cluster without destroying it")
	}
}

func TestDepartTriggersACapacitySyncForTheHiveItLeft(t *testing.T) {
	// Without this the fleet state describes a hive that still contains the
	// departed node until the scheduled tick — three minutes by default — and
	// placement decides against exactly that state.
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(h.syncs) != 1 {
		t.Fatalf("a departure must trigger exactly one capacity sync, got %v", h.syncs)
	}
	// The hive that LOST the node, in the tenant that owns it. The departing host
	// is fleetless now, so there is no second hive to observe.
	if h.syncs[0].hiveID != h.snapshot.Hive.ID {
		t.Errorf("the sync must name the hive the node left (%s), got %s",
			h.snapshot.Hive.ID, h.syncs[0].hiveID)
	}
	if h.syncs[0].orgID != h.orgID {
		t.Errorf("the sync must be enqueued for the departing organization (%s), got %s",
			h.orgID, h.syncs[0].orgID)
	}
}

func TestDepartTriggersTheSyncOnlyAfterTheClusterWorkSucceeded(t *testing.T) {
	// A sync enqueued before or despite failed cluster work would observe the
	// PRE-departure cluster and stamp last_synced_at on it, turning a stale
	// reading into a fresh and wrong one — after which DrainStateAt and the
	// node-count agreement check in AssertClusterHealthy stop protecting
	// anything, because both are built on staleness being visible.
	h := newHarness(t)
	h.departing.failOn = "pmxcfs -l" // the last cluster-side step

	if err := h.run(); err == nil {
		t.Fatal("a failed standalone conversion must fail the departure")
	}
	if len(h.syncs) != 0 {
		t.Fatalf("a departure that did not complete must not trigger a sync, got %v", h.syncs)
	}
}

func TestDepartRefusalTriggersNoCapacitySync(t *testing.T) {
	// Nothing was touched, so nothing observed changed. Enqueuing here would spend
	// a provider API call to re-learn what the database already says.
	h := newHarness(t)
	h.snapshot.Hypervisor.ObservedGuests = 2

	if err := h.run(); !errors.Is(err, ErrDepartureRefused) {
		t.Fatalf("want ErrDepartureRefused, got %v", err)
	}
	if len(h.syncs) != 0 {
		t.Fatalf("a refused departure must trigger no sync, got %v", h.syncs)
	}
}

func TestDepartSucceedsEvenWhenTheSyncCannotBeEnqueued(t *testing.T) {
	// The asymmetry with MarkFleetless, asserted rather than left to a comment.
	// The sync is a freshness optimisation the scheduled tick performs anyway;
	// returning this error would fail a departure that already succeeded, and
	// River would retry it against a cluster corosync has already changed —
	// AssertClusterHealthy then refuses forever, advising the very sync that
	// failed.
	h := newHarness(t)
	h.syncErr = errors.New("river is unreachable")

	if err := h.run(); err != nil {
		t.Fatalf("a failed sync enqueue must not fail a departure that succeeded: %v", err)
	}
	if len(h.fleetless) != 1 {
		t.Errorf("the machine must still be recorded fleetless, got %v", h.fleetless)
	}
	if len(h.syncs) != 1 {
		t.Errorf("the enqueue must have been attempted, got %v", h.syncs)
	}
}

func TestDepartTriggersNoSyncForAHiveThatCannotBeSynced(t *testing.T) {
	// A hive with no API URL or no bound credential fails the inventory read
	// before it opens a connection, so an enqueue here is a job that can only
	// fail — with River's whole retry schedule behind it — for a hive an operator
	// has not finished configuring.
	for _, tc := range []struct {
		name  string
		spoil func(*harness)
	}{
		{"no api url", func(h *harness) { h.snapshot.Hive.APIURL = "" }},
		{"blank api url", func(h *harness) { h.snapshot.Hive.APIURL = "   " }},
		{"no credential", func(h *harness) { h.snapshot.Hive.CredentialID = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			tc.spoil(h)
			if err := h.run(); err != nil {
				t.Fatalf("an unconfigured hive must not stop a departure: %v", err)
			}
			if len(h.syncs) != 0 {
				t.Fatalf("no sync may be enqueued for a hive that cannot be synced, got %v", h.syncs)
			}
		})
	}
}
