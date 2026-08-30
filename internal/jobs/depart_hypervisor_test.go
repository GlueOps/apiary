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
	// detached models a node that is already out of its cluster, which is the
	// state a retry of a part-finished departure starts from.
	detached bool
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
	// The cluster this node reports itself in. Empty means standalone, which is
	// how a retry recognises a departure that already detached — see
	// IsStandalone. The fake answers with a cluster while `detached` is false,
	// so an ordinary departure takes the full path.
	case strings.Contains(cmd, "APIARY_CLUSTER="):
		if f.detached {
			return "APIARY_CLUSTER=\n", nil
		}
		return "APIARY_CLUSTER=dc1\n", nil

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
		// Accept-exposure, because it is the disposition this control plane can
		// actually carry out. Rotate refuses at the gate while it is
		// unimplemented — see RefuseCredentialsRotateUnavailable.
		CredentialDisposition: tenant.CredentialsAcceptExposure,
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
	// NOT recorded fleetless here. The machine keeps its hive and name until a
	// remaining member observes the cluster has dropped it; clearing on this
	// job's success removed it from the composition before any peer could evict
	// it. See TestTheDepartureJobDoesNotClearMembership.
	if len(h.fleetless) != 0 {
		t.Fatalf("the job cleared membership, got %v", h.fleetless)
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
	if len(h.fleetless) != 0 {
		t.Fatalf("the machine must be recorded fleetless even with no agent enrolled, got %v",
			h.fleetless)
	}
}

// ORDERING OF WHAT THIS JOB STILL DOES.
//
// The cluster-side removal moved to the agent: a remaining member evicts the
// departing node from the published composition and restores its own quorum
// afterwards, because `pvecm expected` affects only the host it runs on. What is
// left here is the departing node's own disarm, where rotate versus
// accept-exposure is an operator decision no desired state carries.
//
// The keys are still read BEFORE anything is stopped — they are the evidence the
// standalone conversion proves itself against, and a stopped pmxcfs has nothing
// left to fingerprint.
func TestDepartOrderingIsLoadBearing(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	readKeys := indexOf(h.log, "id_rsa.pub")
	stopStack := indexOf(h.log, "systemctl stop pve-cluster")
	standalone := indexOf(h.log, "pmxcfs -l")

	for name, i := range map[string]int{
		"read keys": readKeys, "stop stack": stopStack, "standalone": standalone,
	} {
		if i < 0 {
			t.Fatalf("%s never ran: %v", name, h.log)
		}
	}
	if readKeys >= stopStack || stopStack >= standalone {
		t.Fatalf("out of order: keys=%d stop=%d standalone=%d", readKeys, stopStack, standalone)
	}

	// And the cluster-side removal is NOT here any more.
	if i := indexOf(h.log, "pvecm delnode"); i >= 0 {
		t.Fatalf("this job ran delnode at step %d; a remaining member owns that now, and "+
			"having both do it is what let the job win the race every time", i)
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
	if len(h.fleetless) != 0 {
		t.Errorf("the job cleared membership; FinaliseDepartures owns that now, got %v", h.fleetless)
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

// THE STRANDED DEPARTURE. A departure's middle step is irreversible: once
// delnode has run the node is out, and everything after it can still fail. One
// did — the disarm — and the job then retried from the top, hit the preflight,
// and refused with "the cluster config lists 1 node(s)", which was true and
// useless. Four attempts, none able to reach the one step that still needed
// doing, and the job cancelled leaving the node detached and armed.
//
// A retry must resume, not restart.
func TestARetriedDepartureResumesAfterTheDetach(t *testing.T) {
	h := newHarness(t)
	// The state a part-finished departure leaves: the node is already out.
	h.departing.detached = true
	// And the cluster is down to one node, so the preflight the first attempt
	// passed can no longer pass.
	h.remaining.corosyncNodes = 1

	if err := h.run(); err != nil {
		t.Fatalf("a retry of a part-finished departure must resume, not refuse: %v", err)
	}

	// delnode must NOT run again. The node is already out, and asking would be
	// asking to remove the last member of a cluster.
	for _, line := range h.log {
		if strings.Contains(line, "pvecm delnode") {
			t.Fatalf("delnode ran against an already-detached node: %q", line)
		}
	}
	// The steps that still needed doing must have run.
	var disarmed bool
	for _, line := range h.log {
		if strings.Contains(line, "pmxcfs -l") {
			disarmed = true
		}
	}
	if !disarmed {
		t.Fatal("the retry skipped the disarm, which is the work it existed to finish")
	}
	if len(h.fleetless) != 0 {
		t.Fatalf("the job cleared membership; FinaliseDepartures owns that now, got %v", h.fleetless)
	}
}

// And an ordinary departure must be unaffected: it is not a retry, so every step
// runs including the preflight and the removal.
// THE CLUSTER-SIDE REMOVAL IS NO LONGER THIS JOB'S, and these are the
// invariants that moved with it.
//
// A remaining member runs `pvecm delnode` from the published composition, so
// every check that used to guard it now guards it there, on the node that
// performs it:
//
//   - quorum and the majority a removal would break -> fleetops.Evict's preflight
//   - guests that a removal would orphan            -> fleetops.Evict's preflight
//   - the stale cfs lock a half-finished delnode leaves behind
//   - `pvecm expected` afterwards where the removal cost quorum, run on the
//     reconciling node, which is the only host it affects
//
// The last one is why the checks belong there rather than here: a central job
// had to choose a host to restore quorum on, and under per-cluster convergence
// the node doing the reconciling IS that host.
//
// This job keeps the departed node's own disarm, where rotate versus
// accept-exposure is an operator decision no desired state carries.
func TestTheDepartureJobNoLongerTouchesTheCluster(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("depart: %v", err)
	}
	for _, forbidden := range []string{"pvecm delnode", "pvecm expected", "file-corosync_conf"} {
		if i := indexOf(h.log, forbidden); i >= 0 {
			t.Fatalf("ran %q at step %d; a remaining member owns that now, and having both do "+
				"it is what let this job win the race every time", forbidden, i)
		}
	}
	// It still disarms the departing node, which is the half that stays here.
	if i := indexOf(h.log, "pmxcfs -l"); i < 0 {
		t.Fatalf("the departing node was not converted to standalone: %v", h.log)
	}
}

// A ONE-NODE CLUSTER WAS A DEAD END. Departing the last node was refused for
// want of a peer to run delnode from, and deleting a cluster is refused while it
// still holds a hypervisor — so the cluster could be neither dismantled nor
// emptied, and the message blamed DNS: "no remaining node has a resolvable
// address", for a cluster that simply had nobody else in it.
//
// Removing the last node is not a node removal. There is no delnode to run and
// no survivor to keep quorate, only the standalone conversion on the node
// itself — the mirror of creating a cluster on a first node.
func TestDepartingTheLastNodeDismantlesTheCluster(t *testing.T) {
	h := newHarness(t)
	h.snapshot.LastNode = true
	h.snapshot.RemainingAddress = ""
	h.snapshot.PeerNames = nil
	h.snapshot.HiveNodeCount = 1
	// The operator's minimum still governs. Lowered here because dismantling a
	// one-node cluster is only reachable on a fleet whose minimum permits it —
	// the gate refusing at the default of 3 is correct, and is what stops this
	// happening by accident on a production cluster.
	h.snapshot.Hive.MinimumNodes = 1

	if err := h.run(); err != nil {
		t.Fatalf("departing the only node must dismantle the cluster, not refuse: %v", err)
	}

	// Nothing may be attempted against a peer that does not exist.
	for _, line := range h.log {
		for _, forbidden := range []string{"pvecm delnode", "pvecm expected"} {
			if strings.Contains(line, forbidden) {
				t.Errorf("%q ran with no remaining node to run it on: %q", forbidden, line)
			}
		}
	}
	// The work that IS possible must still happen: the node becomes standalone.
	var converted bool
	for _, line := range h.log {
		if strings.Contains(line, "pmxcfs -l") {
			converted = true
		}
	}
	if !converted {
		t.Fatal("the last node was not converted to standalone, so nothing was actually done")
	}
	if len(h.fleetless) != 0 {
		t.Fatalf("the job cleared membership; FinaliseDepartures owns that now, got %v", h.fleetless)
	}
}

// And peers that EXIST but have no usable address is a different fault with a
// different remedy, so it keeps its own message rather than sharing one.
func TestPeersWithNoAddressAreNotReportedAsAnEmptyCluster(t *testing.T) {
	h := newHarness(t)
	h.snapshot.LastNode = false
	h.snapshot.RemainingAddress = ""

	err := h.run()
	if err == nil {
		t.Fatal("a departure with no reachable peer must refuse")
	}
	if strings.Contains(err.Error(), "last node") {
		t.Fatalf("an address problem was reported as an empty cluster: %v", err)
	}
}

// THE JOB MUST NOT CLEAR MEMBERSHIP, because doing so undoes the departure that
// started it.
//
// A departure MARKS the machine — departing_at set, hive_id and name kept — so
// the composition can name it and a remaining member can evict it. MarkFleetless
// knew nothing about that marker: it cleared hive_id and left departing_at set,
// producing a row that was fleetless AND departing at once, and removing the
// node from the composition before any peer had seen it.
//
// Observed on lab-a: pve was left alone below quorum with /etc/pve read-only,
// still listing a pve2 that nothing would ever remove — the node it needed to
// evict had already vanished from the document that authorises the eviction.
func TestTheDepartureJobDoesNotClearMembership(t *testing.T) {
	h := newHarness(t)
	if err := h.run(); err != nil {
		t.Fatalf("depart: %v", err)
	}
	// The harness records every MarkFleetless call; there must be none.
	if len(h.fleetless) > 0 {
		t.Fatal("the job cleared membership; that removes the node from the composition before " +
			"a peer can evict it, and leaves the row fleetless and departing at once")
	}
}
