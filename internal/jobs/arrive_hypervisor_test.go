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

// arriveRunner answers the reads the arrival sequence makes and records every
// command, because the ordering — and in particular that the revoke runs even on
// failure — is what a return value cannot show.
type arriveRunner struct {
	name string
	log  *[]string

	clusterName string
	quorate     bool
	// blockers
	hasConf, hasAuthKey, coroRunning bool
	guestConfigs                     int
	nodeDirs                         string

	joinKeyMatches int
	revokeIsNoOp   bool
	knownHostsOK   bool
	nodeName       string
	failOn         string
	reconnectErr   error

	uploads *[]uploadedFile
}

func (a *arriveRunner) Run(_ context.Context, cmd string) (string, error) {
	*a.log = append(*a.log, a.name+": "+cmd)
	if a.failOn != "" && strings.Contains(cmd, a.failOn) {
		return "boom", errors.New("command failed")
	}

	switch {
	case strings.Contains(cmd, "APIARY_CONF"):
		yn := func(b bool) string {
			if b {
				return "yes"
			}
			return "no"
		}
		coro := "inactive"
		if a.coroRunning {
			coro = "active"
		}
		return fmt.Sprintf("APIARY_CONF=%s\nAPIARY_AKEY=%s\nAPIARY_CORO=%s\nAPIARY_GUESTS=%d\nAPIARY_DIRS=%s\n",
			yn(a.hasConf), yn(a.hasAuthKey), coro, a.guestConfigs, a.nodeDirs), nil
	case strings.Contains(cmd, "APIARY_NODE="):
		// The host answers with its own name. The fake must speak the same marker
		// contract, and a name here is what an IP-addressed machine cannot derive
		// from its address.
		if a.nodeName == "" {
			return "APIARY_NODE=\n", nil
		}
		return "APIARY_NODE=" + a.nodeName + "\n", nil
	case strings.Contains(cmd, "ssh_host_ed25519_key.pub"):
		return "ssh-ed25519 AAAAHOSTKEY root@target\n", nil
	case strings.Contains(cmd, "id_rsa.pub"):
		return "ssh-rsa AAAAJOINERRSA root@joiner\nssh-ed25519 AAAAJOINERED root@joiner\n", nil
	case strings.Contains(cmd, "ssh-keygen -F"):
		if a.knownHostsOK {
			return "FOUND\n", nil
		}
		return "MISSING\n", nil
	case strings.Contains(cmd, "APIARY_CLUSTER"):
		q := "no"
		if a.quorate {
			q = "yes"
		}
		return fmt.Sprintf("APIARY_CLUSTER=%s\nAPIARY_QUORATE=%s\n", a.clusterName, q), nil
	case strings.Contains(cmd, "test -f"):
		return "PRESENT\n", nil
	case strings.Contains(cmd, "grep -cF"):
		return fmt.Sprintf("%d\n", a.joinKeyMatches), nil
	case strings.Contains(cmd, "grep -vF"):
		if !a.revokeIsNoOp {
			a.joinKeyMatches = 0
		}
		return "", nil
	}
	return "", nil
}

// Upload records what was written as well as where. A test asserting that a
// credential file arrived has to check its CONTENT — "a file was written to that
// path" and "the token in it is the current one" are different claims, and only
// the second one matters after a join invalidates the old token.
func (a *arriveRunner) Upload(_ context.Context, path string, _ uint32, body []byte) error {
	if a.uploads != nil {
		*a.uploads = append(*a.uploads, uploadedFile{
			Host: a.name, Path: path, Body: append([]byte(nil), body...),
		})
	}
	return nil
}

// uploadedFile is one recorded write, with the host that received it.
type uploadedFile struct {
	Host string
	Path string
	Body []byte
}

func (a *arriveRunner) Reconnect(context.Context) error { return a.reconnectErr }

type arriveHarness struct {
	log          []string
	uploads      []uploadedFile
	joiner       *arriveRunner
	target       *arriveRunner
	snapshot     ArrivalSnapshot
	joined       []uuid.UUID
	joinedNames  []string
	syncs        []syncRequest
	delayedSyncs []delayedSync
	syncErr      error
	orgID        uuid.UUID
}

func newArriveHarness() *arriveHarness {
	h := &arriveHarness{orgID: uuid.New()}
	hiveID := uuid.New()
	credID := uuid.New()
	h.snapshot = ArrivalSnapshot{
		// A fully configured hive by default — an API URL and a bound credential
		// are what make a capacity sync worth enqueuing at all — so the tests that
		// care about the unconfigured case take them away deliberately.
		TargetHive: tenant.Hive{
			ID: hiveID, Name: "dc2",
			APIURL: "https://pve2.example:8006/api2/json", CredentialID: &credID,
		},
		CurrentHiveID:   nil,
		JoiningAddress:  "10.0.0.9",
		JoiningNodeName: "hv9",
		ObservedGuests:  0,
		TargetAddress:   "10.0.0.1",
		TargetNodeNames: []string{"hv1", "hv2"},
		TargetNodeCount: 2,
		Links:           []tenant.CorosyncLink{{Number: 0, Address: "10.1.0.9"}},
	}
	h.joiner = &arriveRunner{
		name: "joiner", log: &h.log, uploads: &h.uploads,
		clusterName: "dc2", quorate: true,
		nodeDirs: "hv9", knownHostsOK: true, nodeName: "hv9",
	}
	h.target = &arriveRunner{
		name: "target", log: &h.log, uploads: &h.uploads,
		clusterName: "dc2", quorate: true,
		joinKeyMatches: 1, knownHostsOK: true,
	}
	return h
}

func (h *arriveHarness) deps() ArriveDeps {
	return ArriveDeps{
		FleetKey: func(context.Context, uuid.UUID) (string, []byte, error) {
			return "ssh-ed25519 PUB", []byte("PRIV"), nil
		},
		Snapshot: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (ArrivalSnapshot, error) {
			return h.snapshot, nil
		},
		NewRunner: func(addr, _ string, _ []byte) (Runner, error) {
			if addr == h.snapshot.JoiningAddress {
				return h.joiner, nil
			}
			return h.target, nil
		},
		MarkJoined: func(_ context.Context, _, hypervisorID, hiveID uuid.UUID, nodeName string) error {
			h.joined = append(h.joined, hiveID)
			h.joinedNames = append(h.joinedNames, nodeName)
			return nil
		},
		EnqueueCapacitySync: func(_ context.Context, orgID, hiveID uuid.UUID) error {
			h.syncs = append(h.syncs, syncRequest{orgID: orgID, hiveID: hiveID})
			return h.syncErr
		},
		EnqueueCapacitySyncAfter: func(_ context.Context, orgID, hiveID uuid.UUID,
			after time.Duration) error {
			h.delayedSyncs = append(h.delayedSyncs, delayedSync{
				req: syncRequest{orgID: orgID, hiveID: hiveID}, after: after})
			return h.syncErr
		},
	}
}

func (h *arriveHarness) run() error {
	return Arrive(context.Background(), h.deps(),
		ArriveInput{OrganizationID: h.orgID, HypervisorID: uuid.New(), TargetHiveID: h.snapshot.TargetHive.ID})
}

// delayedSync is one recorded call to EnqueueCapacitySyncAfter. The delay is
// kept because it is the whole point of the call.
type delayedSync struct {
	req   syncRequest
	after time.Duration
}

// Arrival must ask for the hive to be re-observed LATER, and must not also ask
// for it immediately.
//
// Proxmox does not populate /cluster/resources stats for a node at the instant
// it joins, so an immediate sync records zero cores and zero memory and stamps
// last_synced_at — the figures then read as observed rather than missing, which
// is worse than having none. Measured on real clusters: cpu=0 at the moment of
// the join, cpu=4 moments later.
//
// The "and not also immediately" half is not tidiness. SyncHiveCapacityArgs is
// unique ByArgs across five states INCLUDING scheduled, and River reports a
// unique collision as a successful insert rather than an error — so enqueuing
// both would drop one silently, with nothing in a log to say which.
func TestArriveAsksForADelayedResyncAndOnlyThat(t *testing.T) {
	h := newArriveHarness()
	if err := h.run(); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if len(h.delayedSyncs) != 1 {
		t.Fatalf("want exactly one delayed sync, got %d", len(h.delayedSyncs))
	}
	got := h.delayedSyncs[0]
	if got.req.hiveID != h.snapshot.TargetHive.ID {
		t.Fatalf("the sync must be for the hive that gained the node (%s), got %s",
			h.snapshot.TargetHive.ID, got.req.hiveID)
	}
	if got.after != ArrivalResyncDelay {
		t.Fatalf("delay = %s, want ArrivalResyncDelay (%s)", got.after, ArrivalResyncDelay)
	}
	if got.after <= 0 {
		t.Fatal("a non-positive delay is an immediate sync wearing a delay's clothes, " +
			"which is the exact thing that recorded zero capacity")
	}
	if len(h.syncs) != 0 {
		t.Fatalf("arrival also enqueued %d immediate sync(s); one of the two would be "+
			"silently dropped as a unique duplicate", len(h.syncs))
	}
}

// The immediate variant is still honoured when the delayed one is not wired, so
// a caller that has not been updated degrades to the old behaviour rather than
// silently never syncing at all.
func TestArriveFallsBackToTheImmediateSyncWhenNoDelayedOneIsWired(t *testing.T) {
	h := newArriveHarness()
	d := h.deps()
	d.EnqueueCapacitySyncAfter = nil
	if err := Arrive(context.Background(), d, ArriveInput{
		OrganizationID: h.orgID, HypervisorID: uuid.New(),
		TargetHiveID: h.snapshot.TargetHive.ID}); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if len(h.syncs) != 1 {
		t.Fatalf("want one immediate sync as the fallback, got %d", len(h.syncs))
	}
}

func TestArriveHappyPath(t *testing.T) {
	h := newArriveHarness()
	if err := h.run(); err != nil {
		t.Fatalf("a fleetless empty host must join: %v", err)
	}
	if len(h.joined) != 1 || h.joined[0] != h.snapshot.TargetHive.ID {
		t.Fatalf("membership must be recorded once, got %v", h.joined)
	}
	// The NODE NAME goes with it. It is the capacity sync's half of this
	// machine's identity — /cluster/resources reports no machine id — so a
	// clustered row with no name is one the next sync cannot match and duplicates
	// on the spot, leaving the onboarding history on an orphan.
	if len(h.joinedNames) != 1 || h.joinedNames[0] != h.snapshot.JoiningNodeName {
		t.Fatalf("the node name must be recorded with the membership, got %v", h.joinedNames)
	}
}

func TestArriveOrdering(t *testing.T) {
	h := newArriveHarness()
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	seed := indexOf(h.log, "ssh-keygen -R")
	authorize := indexOf(h.log, "authorized_keys.d/join")
	join := indexOf(h.log, "pvecm add")
	revoke := indexOf(h.log, "grep -vF")

	for name, i := range map[string]int{
		"seed known_hosts": seed, "authorize join key": authorize,
		"pvecm add": join, "revoke": revoke,
	} {
		if i < 0 {
			t.Fatalf("%s never ran; log: %v", name, h.log)
		}
	}
	// known_hosts before the join: unseeded, pvecm fails with a message naming
	// none of this.
	if seed > join {
		t.Error("known_hosts must be seeded BEFORE pvecm add")
	}
	// The grant must exist before the join needs it, and must not outlive it.
	if authorize > join {
		t.Error("the join key must be authorized before pvecm add")
	}
	if revoke < join {
		t.Error("the join key must be revoked after the join, not before it")
	}
}

func TestArriveRevokesTheGrantEvenWhenTheJoinFails(t *testing.T) {
	// The one deliberate deviation from fail-fast: a failed join that leaves a
	// standing root grant on the target is worse than a failed join, and the file
	// it lives in survives cluster operations by design, so nothing else cleans it.
	h := newArriveHarness()
	h.joiner.failOn = "pvecm add"

	err := h.run()
	if err == nil {
		t.Fatal("a failed pvecm add must fail the arrival")
	}
	if indexOf(h.log, "grep -vF") < 0 {
		t.Fatal("the join key must be revoked even when the join fails")
	}
	if len(h.joined) != 0 {
		t.Error("a failed join must not record membership")
	}
}

func TestArriveFailsWhenTheGrantCannotBeRevoked(t *testing.T) {
	h := newArriveHarness()
	h.target.revokeIsNoOp = true

	err := h.run()
	if err == nil {
		t.Fatal("a revoke that leaves the grant in place must fail the arrival, even though " +
			"the join itself succeeded")
	}
	if !strings.Contains(err.Error(), "STILL") {
		t.Errorf("the error must say the grant is still authorized, got %q", err.Error())
	}
}

func TestArriveRefusalsTouchNoHost(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func(*arriveHarness)
		want  string
	}{
		{"already in a hive", func(h *arriveHarness) {
			other := uuid.New()
			h.snapshot.CurrentHiveID = &other
		}, tenant.RefuseAlreadyInHive},
		{"guests present", func(h *arriveHarness) {
			h.snapshot.ObservedGuests = 2
		}, tenant.RefuseArrivalGuests},
		{"target hive empty", func(h *arriveHarness) {
			h.snapshot.TargetNodeCount = 0
		}, tenant.RefuseTargetHiveEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newArriveHarness()
			tc.spoil(h)
			err := h.run()
			if !errors.Is(err, ErrArrivalRefused) {
				t.Fatalf("want ErrArrivalRefused, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("want %s in %q", tc.want, err.Error())
			}
			if len(h.log) != 0 {
				t.Errorf("a refused arrival must run NO commands; ran %v", h.log)
			}
		})
	}
}

func TestArriveRefusesAHostPvecmWouldReject(t *testing.T) {
	h := newArriveHarness()
	h.joiner.hasConf = true
	h.joiner.guestConfigs = 2
	h.joiner.nodeDirs = "hv9 hv1" // hv1 is a former peer's inherited directory

	err := h.run()
	if !errors.Is(err, ErrArrivalRefused) {
		t.Fatalf("want ErrArrivalRefused, got %v", err)
	}
	// The message must name the inherited directory, because pvecm's own wording
	// sends the operator looking for VMs that are not there.
	if !strings.Contains(err.Error(), "hv1") {
		t.Errorf("the refusal must name the foreign node directory, got %q", err.Error())
	}
	if indexOf(h.log, "pvecm add") >= 0 {
		t.Error("pvecm add must not be attempted on a host known to be unacceptable")
	}
}

func TestArriveFailsWhenKnownHostsDoesNotTake(t *testing.T) {
	h := newArriveHarness()
	h.joiner.knownHostsOK = false

	err := h.run()
	if err == nil {
		t.Fatal("known_hosts that did not take must fail here, not later as an opaque " +
			"\"unable to copy ssh ID\"")
	}
	if indexOf(h.log, "pvecm add") >= 0 {
		t.Error("the join must not be attempted with unseeded known_hosts")
	}
}

func TestArriveRefusesAnUnrestrictedGrant(t *testing.T) {
	// The from= clause is what bounds the grant. This file survives the join by
	// design, so an omitted restriction is a permanent unrestricted credential.
	r := &arriveRunner{name: "t", log: &[]string{}}
	if err := AuthorizeJoinKey(context.Background(), r, "ssh-rsa BODY", ""); err == nil {
		t.Fatal("a join key with no from= restriction must be refused")
	}
	if err := AuthorizeJoinKey(context.Background(), r, "", "10.0.0.9"); err == nil {
		t.Fatal("an empty join key must be refused")
	}
}

func TestArriveRequiresAtLeastOneCorosyncLink(t *testing.T) {
	h := newArriveHarness()
	h.snapshot.Links = nil
	err := h.run()
	if err == nil {
		t.Fatal("a corosync link must be required, not guessed from the management address")
	}
	if !errors.Is(err, ErrArrivalRefused) {
		t.Errorf("a missing link set is a refusal, not a system failure: %v", err)
	}
	if len(h.log) != 0 {
		t.Errorf("nothing should have run; ran %v", h.log)
	}
}

func TestArriveCarriesEveryCorosyncLinkToPvecm(t *testing.T) {
	// Redundant rings are the normal production shape, and dropping one silently
	// halves the cluster network's redundancy — which shows up only when the
	// surviving ring fails and healthy nodes get fenced.
	h := newArriveHarness()
	h.snapshot.Links = nil

	prio := 20
	err := Arrive(context.Background(), h.deps(), ArriveInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   h.snapshot.TargetHive.ID,
		Links: []tenant.CorosyncLink{
			{Number: 2, Address: "10.3.0.9"},
			{Number: 0, Address: "10.1.0.9", Priority: &prio},
		},
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	var join string
	for _, l := range h.log {
		if strings.Contains(l, "pvecm add") {
			join = l
		}
	}
	if join == "" {
		t.Fatal("pvecm add never ran")
	}
	for _, want := range []string{"--link0 'address=10.1.0.9,priority=20'", "--link2 'address=10.3.0.9'"} {
		if !strings.Contains(join, want) {
			t.Errorf("join command must carry %s; got %s", want, join)
		}
	}
	// Sorted by ring, so two attempts produce the same command and a diff shows
	// what actually changed.
	if strings.Index(join, "--link0") > strings.Index(join, "--link2") {
		t.Errorf("links must be ordered by ring number; got %s", join)
	}
}

func TestArriveRefusesAMalformedLinkSetBeforeTouchingAHost(t *testing.T) {
	// Validated before the join key is authorized. Discovering it afterwards
	// would leave a standing root grant on the target for an operation that was
	// never going to run.
	h := newArriveHarness()
	h.snapshot.Links = nil
	err := Arrive(context.Background(), h.deps(), ArriveInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   h.snapshot.TargetHive.ID,
		Links: []tenant.CorosyncLink{
			{Number: 0, Address: "10.1.0.9"},
			{Number: 0, Address: "10.2.0.9"},
		},
	})
	if err == nil {
		t.Fatal("two values for one ring must be refused: pvecm takes one --linkN, so the second " +
			"would silently replace the first")
	}
	if len(h.log) != 0 {
		t.Errorf("nothing should have run; ran %v", h.log)
	}
}

func TestArriveVerifiesQuorumNotJustMembership(t *testing.T) {
	h := newArriveHarness()
	h.joiner.quorate = false

	err := h.run()
	if err == nil {
		t.Fatal("a node that joined but is not quorate has not usefully joined")
	}
	if !strings.Contains(err.Error(), "quorate") {
		t.Errorf("the error must name quorum, got %q", err.Error())
	}
	if len(h.joined) != 0 {
		t.Error("a non-quorate node must not be recorded as a member")
	}
}

func TestArriveRefusesWhenTheManagedKeyDidNotSurvive(t *testing.T) {
	// The design premise, checked rather than assumed: the join replaces pmxcfs
	// and destroys every key in it.
	h := newArriveHarness()
	h.joiner.reconnectErr = errors.New("permission denied")

	err := h.run()
	if err == nil {
		t.Fatal("losing SSH to the host after a join must fail the arrival")
	}
	if !strings.Contains(err.Error(), "no longer authenticate") {
		t.Errorf("the error must say access was lost, got %q", err.Error())
	}
	if len(h.joined) != 0 {
		t.Error("membership must not be recorded for a host we can no longer reach")
	}
}

func TestArriveRefusesOnInheritedGuestConfigsAlone(t *testing.T) {
	// ONLY the guest configs, with no cluster config and no running corosync.
	// This is the exact real-world shape: a node with none of its own guests,
	// carrying its former cluster's replica of a PEER's guest configs, which
	// pvecm counts as "this host already contains virtual guests".
	//
	// Setting several blockers at once would let the guest term be deleted
	// entirely and the test still pass on one of the others.
	h := newArriveHarness()
	h.joiner.hasConf = false
	h.joiner.hasAuthKey = false
	h.joiner.coroRunning = false
	h.joiner.guestConfigs = 2
	h.joiner.nodeDirs = "hv9 hv1"

	err := h.run()
	if !errors.Is(err, ErrArrivalRefused) {
		t.Fatalf("inherited guest configs alone must refuse the join, got %v", err)
	}
	if !strings.Contains(err.Error(), "former cluster peer") {
		t.Errorf("the message must explain these belong to a peer, not to this host — pvecm's "+
			"own wording sends the operator hunting for VMs that do not exist; got %q", err.Error())
	}
	if indexOf(h.log, "pvecm add") >= 0 {
		t.Error("pvecm add must not be attempted")
	}
}

func TestArriveRefusesWhenTheNodeJoinsTheWrongCluster(t *testing.T) {
	// pvecm reports success having added the node to whatever cluster answered on
	// that address. If it is not the hive we were asked for, the control plane's
	// record and the metal disagree, and every later fleet operation acts on the
	// wrong cluster.
	h := newArriveHarness()
	h.joiner.clusterName = "some-other-cluster"

	err := h.run()
	if err == nil {
		t.Fatal("joining a cluster other than the requested hive must fail")
	}
	if !strings.Contains(err.Error(), "some-other-cluster") || !strings.Contains(err.Error(), "dc2") {
		t.Errorf("the error must name both what was joined and what was asked for, got %q", err.Error())
	}
	if len(h.joined) != 0 {
		t.Error("membership must not be recorded when the wrong cluster was joined")
	}
}

func TestArriveStillRevokesAfterAWrongClusterJoin(t *testing.T) {
	// The grant must not outlive a join that went somewhere unexpected — that is
	// the case where a standing credential is least wanted.
	h := newArriveHarness()
	h.joiner.clusterName = "some-other-cluster"
	if err := h.run(); err == nil {
		t.Fatal("expected failure")
	}
	if indexOf(h.log, "grep -vF") < 0 {
		t.Fatal("the join key must be revoked even when the node joined the wrong cluster")
	}
}

func TestArriveUsesTheLinksFromTheRequest(t *testing.T) {
	// The corosync rings come from the operator, not the database and not the
	// management address. Before this was threaded through, every arrival
	// refused for want of them — the API had no way to supply any.
	h := newArriveHarness()
	h.snapshot.Links = nil // nothing stored; the request must provide them

	err := Arrive(context.Background(), h.deps(), ArriveInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   h.snapshot.TargetHive.ID,
		Links:          []tenant.CorosyncLink{{Number: 0, Address: "10.1.0.9"}},
	})
	if err != nil {
		t.Fatalf("links on the request must be enough: %v", err)
	}
	found := false
	for _, l := range h.log {
		if strings.Contains(l, "pvecm add") && strings.Contains(l, "10.1.0.9") {
			found = true
		}
	}
	if !found {
		t.Fatalf("pvecm add must carry the requested link; log: %v", h.log)
	}
}

func TestArriveStillRefusesWithNoLinksAnywhere(t *testing.T) {
	h := newArriveHarness()
	h.snapshot.Links = nil
	err := Arrive(context.Background(), h.deps(), ArriveInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   h.snapshot.TargetHive.ID,
	})
	if err == nil {
		t.Fatal("with no links from either source the arrival must refuse rather than guess")
	}
	if len(h.log) != 0 {
		t.Errorf("nothing should have run; ran %v", h.log)
	}
}

func TestArriveRefusesWhenTheTargetIsNotInAClusterAtAll(t *testing.T) {
	// Pointing an arrival at a standalone host: pvecm would have nothing to add
	// the node to. Refused before the join key is authorized, so a mistyped
	// address does not leave a standing grant on an unrelated machine.
	h := newArriveHarness()
	h.target.clusterName = ""

	err := h.run()
	if !errors.Is(err, ErrArrivalRefused) {
		t.Fatalf("want ErrArrivalRefused, got %v", err)
	}
	if indexOf(h.log, "pvecm add") >= 0 {
		t.Error("the join must not be attempted against a host that is in no cluster")
	}
}

func TestArriveChecksTheJoinerAgainstTheTargetsClusterNotTheHiveName(t *testing.T) {
	// Nothing enforces that an apiary hive is named after the Proxmox cluster it
	// represents. Asserting on the hive's name would fail a perfectly correct
	// join whenever an operator named the hive something else — and would still
	// miss the case that matters, which is landing in a cluster that merely
	// answers on that address.
	h := newArriveHarness()
	h.snapshot.TargetHive.Name = "a-name-nobody-gave-the-cluster"
	h.target.clusterName = "real-cluster"
	h.joiner.clusterName = "real-cluster"

	if err := h.run(); err != nil {
		t.Fatalf("a join into the target's actual cluster must succeed however the hive is named: %v", err)
	}

	// And the check still bites when the joiner lands somewhere else.
	h2 := newArriveHarness()
	h2.snapshot.TargetHive.Name = "irrelevant"
	h2.target.clusterName = "intended-cluster"
	h2.joiner.clusterName = "some-other-cluster"

	err := h2.run()
	if err == nil {
		t.Fatal("joining a cluster other than the target's must fail")
	}
	if !strings.Contains(err.Error(), "some-other-cluster") ||
		!strings.Contains(err.Error(), "intended-cluster") {
		t.Errorf("the error must name what was joined and what was expected, got %q", err.Error())
	}
}

func TestArriveTriggersACapacitySyncForTheHiveItJoined(t *testing.T) {
	// Until the sync runs, this machine has membership recorded and no capacity
	// figures — and a row with zero observed capacity reads to the placement
	// oracle as a FULL host rather than a new one.
	//
	// DELAYED, because Proxmox has not populated stats for a node at the instant
	// it joins: a sync run immediately writes those zeros AND stamps
	// last_synced_at, so they read as observed rather than absent.
	h := newArriveHarness()
	if err := h.run(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(h.delayedSyncs) != 1 {
		t.Fatalf("an arrival must trigger exactly one capacity sync, got %v", h.delayedSyncs)
	}
	// The hive that GAINED the node. There is no second hive: CheckArrival refuses
	// with already_in_hive whenever the joining machine still claims one, so a
	// host reaching this point was fleetless.
	if h.delayedSyncs[0].req.hiveID != h.snapshot.TargetHive.ID {
		t.Errorf("the sync must name the hive the node joined (%s), got %s",
			h.snapshot.TargetHive.ID, h.delayedSyncs[0].req.hiveID)
	}
	if h.delayedSyncs[0].req.orgID != h.orgID {
		t.Errorf("the sync must be enqueued for the arriving organization (%s), got %s",
			h.orgID, h.delayedSyncs[0].req.orgID)
	}
}

func TestArriveTriggersNoCapacitySyncWhenTheJoinFailed(t *testing.T) {
	// A sync of the target after a failed join would observe a cluster this
	// machine never entered and stamp last_synced_at on it — a stale reading
	// recorded as a fresh one, which is the reading every gate downstream trusts.
	h := newArriveHarness()
	h.joiner.failOn = "pvecm add"

	if err := h.run(); err == nil {
		t.Fatal("a failed pvecm add must fail the arrival")
	}
	if len(h.syncs)+len(h.delayedSyncs) != 0 {
		t.Fatalf("an arrival that did not complete must not trigger a sync, got %v / %v",
			h.syncs, h.delayedSyncs)
	}
}

func TestArriveRefusalTriggersNoCapacitySync(t *testing.T) {
	h := newArriveHarness()
	h.snapshot.ObservedGuests = 2

	if err := h.run(); !errors.Is(err, ErrArrivalRefused) {
		t.Fatalf("want ErrArrivalRefused, got %v", err)
	}
	if len(h.syncs)+len(h.delayedSyncs) != 0 {
		t.Fatalf("a refused arrival must trigger no sync, got %v / %v", h.syncs, h.delayedSyncs)
	}
}

func TestArriveSucceedsEvenWhenTheSyncCannotBeEnqueued(t *testing.T) {
	// Returning it would fail a migration that worked, and River's retry would
	// re-run AuthorizeJoinKey — re-creating a root grant on the target and
	// revoking it again. Real side effects to protect a refresh the scheduled tick
	// performs anyway.
	h := newArriveHarness()
	h.syncErr = errors.New("river is unreachable")

	if err := h.run(); err != nil {
		t.Fatalf("a failed sync enqueue must not fail an arrival that succeeded: %v", err)
	}
	if len(h.joined) != 1 {
		t.Errorf("membership must still be recorded, got %v", h.joined)
	}
	if len(h.delayedSyncs) != 1 {
		t.Errorf("the enqueue must have been attempted, got %v", h.delayedSyncs)
	}
}

func TestArriveTriggersNoSyncForAHiveThatCannotBeSynced(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spoil func(*arriveHarness)
	}{
		{"no api url", func(h *arriveHarness) { h.snapshot.TargetHive.APIURL = "" }},
		{"blank api url", func(h *arriveHarness) { h.snapshot.TargetHive.APIURL = "   " }},
		{"no credential", func(h *arriveHarness) { h.snapshot.TargetHive.CredentialID = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newArriveHarness()
			tc.spoil(h)
			if err := h.run(); err != nil {
				t.Fatalf("an unconfigured hive must not stop an arrival: %v", err)
			}
			if len(h.syncs)+len(h.delayedSyncs) != 0 {
				t.Fatalf("no sync may be enqueued for a hive that cannot be synced, got %v / %v",
					h.syncs, h.delayedSyncs)
			}
		})
	}
}

func TestArriveWorksForAnIPAddressedMachine(t *testing.T) {
	// The case that was broken. A machine recorded by IP has no node name to
	// derive from its address — NodeNameFromAddress returns "" for an IP literal
	// by design, so 10.0.0.9 does not become the node "10" — and Proxmox metal is
	// routinely recorded by IP.
	//
	// Two things failed on the empty name: ReadJoinBlockers filters
	// /etc/pve/nodes against it, so the node's OWN directory read as a former
	// peer's and the arrival was refused before it began; and MarkJoined refuses
	// to record membership without a name, so a join that did succeed could not be
	// written down. Asking the host is what fixes both.
	h := newArriveHarness()
	h.snapshot.JoiningAddress = "10.0.0.9"
	h.snapshot.JoiningNodeName = "" // nothing derivable from an IP
	h.joiner.nodeName = "pve2"      // what the machine actually calls itself
	h.joiner.nodeDirs = "pve2"

	if err := h.run(); err != nil {
		t.Fatalf("an IP-addressed machine must be able to arrive: %v", err)
	}
	if len(h.joined) != 1 {
		t.Fatal("membership must be recorded")
	}
	if len(h.joinedNames) != 1 || h.joinedNames[0] != "pve2" {
		t.Fatalf("membership must be recorded under the name the HOST reports, got %v",
			h.joinedNames)
	}
}

func TestArriveRefusesWhenTheHostReportsNoNodeName(t *testing.T) {
	// A host that cannot say what it is called cannot be recorded: a row with
	// hive_id set and name NULL is the shape the capacity sync cannot match and
	// duplicates on its next tick. Refusing leaves the machine visibly fleetless
	// and retryable.
	h := newArriveHarness()
	h.joiner.nodeName = ""

	err := h.run()
	if err == nil {
		t.Fatal("an unnameable host must not be joined")
	}
	if indexOf(h.log, "pvecm add") >= 0 {
		t.Error("the join must not be attempted before the name is known")
	}
}

// TestArrivalRewritesTheAgentsProxmoxCredential.
//
// THE TOKEN A JOIN INVALIDATES. Proxmox keeps API tokens in
// /etc/pve/priv/token.cfg, which is on pmxcfs — a cluster database rather than a
// node one — so joining REPLACES it with the target cluster's copy. Whatever
// credential the agent was given at onboarding belonged to a different cluster,
// or to none, and after the join it does not exist on that host.
//
// Left alone, every fleet task on a freshly joined node fails to authenticate
// against its own API, and it reads as the join having broken something.
func TestArrivalRewritesTheAgentsProxmoxCredential(t *testing.T) {
	h := newArriveHarness()
	d := h.deps()

	var askedFor uuid.UUID
	d.BaseURL = "http://cp:8282"
	d.ProxmoxCredential = func(_ context.Context, _, hiveID uuid.UUID) (ProxmoxCredential, error) {
		askedFor = hiveID
		return ProxmoxCredential{TokenID: "root@pam!target", Secret: "target-secret"}, nil
	}

	if err := Arrive(context.Background(), d, ArriveInput{
		OrganizationID: h.orgID, HypervisorID: uuid.New(),
		TargetHiveID: h.snapshot.TargetHive.ID,
	}); err != nil {
		t.Fatalf("arrive: %v", err)
	}

	// The TARGET fleet's credential, not the machine's own — at this moment the
	// machine's row still says it is fleetless.
	if askedFor != h.snapshot.TargetHive.ID {
		t.Fatalf("asked for the credential of %s, want the target hive %s",
			askedFor, h.snapshot.TargetHive.ID)
	}

	var written []byte
	for _, u := range h.uploads {
		if u.Path == AgentEnvPath && u.Host == "joiner" {
			written = u.Body
		}
	}
	if written == nil {
		t.Fatalf("the credential was never written to the JOINING host; uploads were %v",
			h.uploads)
	}
	if !strings.Contains(string(written), "AGENT_PROXMOX_TOKEN_ID=root@pam!target") {
		t.Fatalf("the target fleet's token is not in the file:\n%s", written)
	}

	// And the agent is restarted, because it builds its Proxmox client once at
	// startup: writing the file without a restart leaves the dead credential in
	// memory until something else happens to restart the service.
	if indexOf(h.log, "restart apiary-agent") == -1 {
		t.Fatalf("the agent was not restarted, so it keeps using the credential the join "+
			"invalidated; commands were %v", h.log)
	}
}

// TestArrivalSucceedsWhenTheCredentialCannotBeWritten.
//
// The migration WORKED. Returning an error would re-run AuthorizeJoinKey on a
// retry — re-creating a root grant on the target and revoking it again — and
// show the operator a failed job for a move that succeeded.
func TestArrivalSucceedsWhenTheCredentialCannotBeWritten(t *testing.T) {
	h := newArriveHarness()
	d := h.deps()
	d.BaseURL = "http://cp:8282"
	d.ProxmoxCredential = func(_ context.Context, _, _ uuid.UUID) (ProxmoxCredential, error) {
		return ProxmoxCredential{}, errors.New("no credential for that fleet")
	}

	if err := Arrive(context.Background(), d, ArriveInput{
		OrganizationID: h.orgID, HypervisorID: uuid.New(),
		TargetHiveID: h.snapshot.TargetHive.ID,
	}); err != nil {
		t.Fatalf("a credential failure failed the whole arrival: %v", err)
	}
	if len(h.joined) != 1 {
		t.Fatalf("the membership was not recorded: %v", h.joined)
	}
}
