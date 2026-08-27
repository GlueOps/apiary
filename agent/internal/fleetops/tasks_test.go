package fleetops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/pkg/proxmox"
)

type fakeCluster struct {
	statusErr error
	quorate   bool
	members   int
	names     []string
	guests    map[string][]Guest
	guestsErr error
	deleted   []string
	deleteErr error
	joined    *proxmox.ClusterJoinOptions
	joinErr   error

	// statusHook runs at the top of each Status call, with the number of calls
	// made so far including this one. It lets a test model a host whose API goes
	// away while pve-cluster restarts and comes back a member — which is the
	// whole of what a join looks like from here.
	statusHook  func(call int, f *fakeCluster)
	statusCalls int
}

func (f *fakeCluster) Status(context.Context) error {
	f.statusCalls++
	if f.statusHook != nil {
		f.statusHook(f.statusCalls, f)
	}
	return f.statusErr
}
func (f *fakeCluster) Quorate() bool       { return f.quorate }
func (f *fakeCluster) MemberCount() int    { return f.members }
func (f *fakeCluster) NodeNames() []string { return f.names }
func (f *fakeCluster) GuestsOn(_ context.Context, node string) ([]Guest, error) {
	return f.guests[node], f.guestsErr
}
func (f *fakeCluster) DeleteConfigNode(_ context.Context, node string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, node)
	return nil
}
func (f *fakeCluster) JoinCluster(_ context.Context, o *proxmox.ClusterJoinOptions) (string, error) {
	f.joined = o
	return "ok", f.joinErr
}

type recordingRunner struct{ cmds [][]string }

func (r *recordingRunner) Run(_ context.Context, _ func(string, string, ...any), name string, args ...string) error {
	r.cmds = append(r.cmds, append([]string{name}, args...))
	return nil
}

func testEnv(phases *[]string, declared *time.Duration, declareErr error) *Env {
	return &Env{
		Logf:  func(string, string, ...any) {},
		Phase: func(p string) { *phases = append(*phases, p) },
		Declare: func(_ context.Context, d time.Duration, _ string) error {
			*declared = d
			return declareErr
		},
	}
}

func healthyCluster() *fakeCluster {
	return &fakeCluster{quorate: true, members: 5, names: []string{"hv-01", "hv-02", "hv-03", "hv-04", "hv-05"}, guests: map[string][]Guest{}}
}

// Quorum arithmetic is the check that stops a rotation taking a datacenter
// read-only. It is computed rather than asked, because the API reports quorum
// for the CURRENT membership and the question is about a future one.
func TestQuorumSurvivesLosing(t *testing.T) {
	cases := []struct {
		members, losing int
		want            bool
		why             string
	}{
		{3, 1, true, "2 of 3 is a majority"},
		{2, 1, false, "1 of 2 is not a majority — the classic two-node trap"},
		{1, 1, false, "removing the only node leaves nothing"},
		{5, 1, true, "4 of 5"},
		{4, 1, true, "3 of 4 is a majority"},
		{4, 2, false, "2 of 4 is exactly half, and half is not a majority — the case that bites " +
			"when two nodes of a four-node cluster are rotated at once"},
		{5, 3, false, "2 of 5"},
	}
	for _, c := range cases {
		if got := quorumSurvivesLosing(c.members, c.losing); got != c.want {
			t.Errorf("quorumSurvivesLosing(%d, %d) = %v, want %v — %s",
				c.members, c.losing, got, c.want, c.why)
		}
	}
}

func TestDistUpgradeRefusesWhileGuestsAreRunning(t *testing.T) {
	cl := healthyCluster()
	cl.guests["hv-01"] = []Guest{{ID: "101", Name: "db", Type: "qemu", Status: "running"}}

	var phases []string
	var declared time.Duration
	runner := &recordingRunner{}

	err := DistUpgrade(context.Background(),
		Deps{Cluster: cl, Runner: runner, Node: "hv-01"},
		testEnv(&phases, &declared, nil), "", `{}`)

	if err == nil {
		t.Fatal("upgraded a host with a running guest; a rotation would drop it without warning")
	}
	if !strings.Contains(err.Error(), "still running") {
		t.Fatalf("error does not name the blocker: %v", err)
	}
	if len(runner.cmds) != 0 {
		t.Fatalf("ran %v despite refusing; the pre-flight is advisory, not a gate", runner.cmds)
	}
}

func TestDistUpgradeRefusesWhenQuorumWouldNotSurvive(t *testing.T) {
	cl := healthyCluster()
	cl.members = 2 // the two-node cluster: losing one leaves no majority
	cl.names = []string{"hv-01", "hv-02"}

	var phases []string
	var declared time.Duration
	runner := &recordingRunner{}

	err := DistUpgrade(context.Background(),
		Deps{Cluster: cl, Runner: runner, Node: "hv-01"},
		testEnv(&phases, &declared, nil), "", `{}`)
	if err == nil || !strings.Contains(err.Error(), "majority") {
		t.Fatalf("err = %v, want a refusal naming quorum", err)
	}
	if len(runner.cmds) != 0 {
		t.Fatal("upgraded anyway")
	}
}

// A cluster whose status cannot be READ is not a cluster that is fine.
func TestDistUpgradeRefusesWhenClusterStateIsUnknown(t *testing.T) {
	cl := healthyCluster()
	cl.statusErr = errors.New("connection refused")

	var phases []string
	var declared time.Duration
	runner := &recordingRunner{}

	if _, err := preflightReboot(context.Background(), cl, "hv-01", false); err == nil {
		t.Fatal("an unreadable cluster status was treated as safe")
	}
	err := DistUpgrade(context.Background(),
		Deps{Cluster: cl, Runner: runner, Node: "hv-01"},
		testEnv(&phases, &declared, nil), "", `{}`)
	if err == nil {
		t.Fatal("proceeded with unknown cluster state")
	}
	if len(runner.cmds) != 0 {
		t.Fatal("ran apt with unknown cluster state")
	}
}

// The declaration must land BEFORE the reboot, and a failure to declare must
// stop it — otherwise the control plane reads a planned reboot as an outage,
// which is the one thing this whole model exists to prevent.
func TestDistUpgradeWillNotRebootIfItCannotAnnounceIt(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration
	rebooted := false

	err := DistUpgrade(context.Background(),
		Deps{
			Cluster: cl, Runner: &recordingRunner{}, Node: "hv-01",
			Reboot: func(context.Context) error { rebooted = true; return nil },
		},
		testEnv(&phases, &declared, errors.New("control plane unreachable")),
		"", `{"reboot":"always","expected_down_seconds":600}`)

	if err == nil {
		t.Fatal("rebooted without announcing it")
	}
	if rebooted {
		t.Fatal("the host rebooted even though the control plane was never told; it will be " +
			"recorded as an unexplained outage mid-rotation")
	}
	if len(phases) == 0 || phases[len(phases)-1] != phaseRebooting {
		t.Fatalf("phases = %v; the rebooting phase must be recorded BEFORE the attempt, or a "+
			"host that dies mid-declaration cannot resume", phases)
	}
}

// Re-entering after the reboot must NOT upgrade again. Without this a rotation
// loops forever on every host it touches.
func TestDistUpgradeResumingAfterItsOwnRebootDoesNotUpgradeAgain(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration
	runner := &recordingRunner{}
	rebooted := false

	err := DistUpgrade(context.Background(),
		Deps{
			Cluster: cl, Runner: runner, Node: "hv-01",
			Reboot: func(context.Context) error { rebooted = true; return nil },
		},
		testEnv(&phases, &declared, nil), phaseRebooting, `{}`)

	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if len(runner.cmds) != 0 {
		t.Fatalf("ran %v on resume; the host would upgrade and reboot in a loop", runner.cmds)
	}
	if rebooted {
		t.Fatal("rebooted again on resume")
	}
}

func TestEvictRefusesToRemoveTheHostRunningIt(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration

	err := Evict(context.Background(),
		Deps{Cluster: cl, Node: "hv-01"},
		testEnv(&phases, &declared, nil), `{"node":"hv-01"}`)
	if err == nil || !strings.Contains(err.Error(), "cannot remove itself") {
		t.Fatalf("err = %v, want a refusal: delnode is run by a surviving member", err)
	}
	if len(cl.deleted) != 0 {
		t.Fatal("attempted the removal anyway")
	}
}

// Evicting a node that still has guests ORPHANS them, and there is no override
// for it on purpose.
//
// The failure is worse than downtime and worse than it looks. Removing the node
// drops its guest configuration from this cluster. If that node then joins
// another datacenter, the join discards its local guest configuration too. The
// disks survive on storage while neither cluster owns the guest: the one it left
// forgot it, the one it joined never learned about it. Recovery means
// hand-editing /etc/pve.
//
// So the refusal is unconditional, and the test asserts there is no way past it.
func TestEvictRefusesWhileTheTargetStillHasGuestsAndOffersNoOverride(t *testing.T) {
	cl := healthyCluster()
	cl.guests["hv-03"] = []Guest{{ID: "505", Type: "qemu", Status: "stopped"}}
	var phases []string
	var declared time.Duration

	err := Evict(context.Background(),
		Deps{Cluster: cl, Node: "hv-01"},
		testEnv(&phases, &declared, nil), `{"node":"hv-03"}`)
	if err == nil {
		t.Fatal("removed a node that still had a guest registered; that guest is now orphaned")
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("the refusal does not explain the consequence, so an operator will look for a "+
			"way around it: %v", err)
	}
	if len(cl.deleted) != 0 {
		t.Fatal("attempted the removal anyway")
	}

	// The flag that used to force this is gone. Passing it must NOT re-enable
	// the behaviour — an unknown spec field is ignored by encoding/json, so a
	// runbook still carrying it silently gets the safe path rather than the old
	// one.
	if err := Evict(context.Background(),
		Deps{Cluster: cl, Node: "hv-01"},
		testEnv(&phases, &declared, nil), `{"node":"hv-03","allow_guest_loss":true}`); err == nil {
		t.Fatal("a stale allow_guest_loss in a spec forced the eviction; the override was removed " +
			"precisely because what it causes cannot be undone")
	}
	if len(cl.deleted) != 0 {
		t.Fatal("removed the node despite the guests")
	}

	// With the guests reassigned, the eviction proceeds — adoption first is the
	// supported path, and it has to actually work or operators will look for
	// another one.
	delete(cl.guests, "hv-03")
	if err := Evict(context.Background(),
		Deps{Cluster: cl, Node: "hv-01"},
		testEnv(&phases, &declared, nil), `{"node":"hv-03"}`); err != nil {
		t.Fatalf("eviction refused even with no guests on the target: %v", err)
	}
	if len(cl.deleted) != 1 || cl.deleted[0] != "hv-03" {
		t.Fatalf("deleted = %v, want [hv-03]", cl.deleted)
	}
}

func TestEvictRefusesANodeThatIsNotAMember(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration
	err := Evict(context.Background(),
		Deps{Cluster: cl, Node: "hv-01"},
		testEnv(&phases, &declared, nil), `{"node":"hv-99"}`)
	if err == nil || !strings.Contains(err.Error(), "not a member") {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

// Without the fingerprint a join cannot tell the intended cluster from anything
// else answering on that address.
func TestJoinRequiresAFingerprint(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration

	err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "pw", nil }},
		testEnv(&phases, &declared, nil), `{"peer":"10.0.0.1"}`)
	if err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("err = %v, want a refusal naming the fingerprint", err)
	}
	if cl.joined != nil {
		t.Fatal("attempted the join anyway")
	}
}

// The password must come from the host, never from the task spec: specs are
// stored as jsonb in the control database and retained after completion.
func TestJoinTakesThePasswordFromTheHostNotTheSpec(t *testing.T) {
	cl := healthyCluster()
	var phases []string
	var declared time.Duration

	spec := `{"peer":"10.0.0.1","fingerprint":"AA:BB","password":"should-be-ignored"}`
	err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "from-the-host", nil }},
		testEnv(&phases, &declared, nil), spec)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if cl.joined == nil {
		t.Fatal("no join attempted")
	}
	if cl.joined.Password != "from-the-host" {
		t.Fatalf("password = %q, want the one supplied by the host; a spec field must never be "+
			"able to carry a cluster root password into the job table", cl.joined.Password)
	}
}

func TestJoinRefusesWhenTheHostAlreadyHasGuests(t *testing.T) {
	cl := healthyCluster()
	cl.guests["hv-06"] = []Guest{{ID: "301", Type: "qemu", Status: "stopped"}}
	var phases []string
	var declared time.Duration

	err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "pw", nil }},
		testEnv(&phases, &declared, nil), `{"peer":"10.0.0.1","fingerprint":"AA:BB"}`)
	if err == nil || !strings.Contains(err.Error(), "guest") {
		t.Fatalf("err = %v, want a refusal: joining discards local guest configuration", err)
	}
}

// shortJoinWait makes the settle loop finish inside a test.
func shortJoinWait(t *testing.T, timeout, interval time.Duration) {
	t.Helper()
	oldT, oldI := joinSettleTimeout, joinPollInterval
	joinSettleTimeout, joinPollInterval = timeout, interval
	t.Cleanup(func() { joinSettleTimeout, joinPollInterval = oldT, oldI })
}

// THE GAP THIS CLOSES. POST /cluster/config/join returns once the request is
// ACCEPTED; the work runs as a task on the node afterwards and can fail there —
// a wrong password, a peer that is itself not quorate, corosync unable to reach
// the ring address. Join returned nil on acceptance, so every one of those was
// reported to the control plane as a completed join: the task went green and the
// node never joined.
func TestJoinFailsWhenTheHostNeverBecomesAMember(t *testing.T) {
	shortJoinWait(t, 120*time.Millisecond, 10*time.Millisecond)

	// Accepted, and the host stays exactly as it was: not clustered.
	cl := &fakeCluster{guests: map[string][]Guest{}, statusErr: errors.New("no cluster")}
	var phases []string
	var declared time.Duration

	err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "pw", nil }},
		testEnv(&phases, &declared, nil), `{"peer":"10.0.0.1","fingerprint":"AA:BB"}`)

	if err == nil {
		t.Fatal("a join that never took effect was reported as success")
	}
	if !strings.Contains(err.Error(), "still not a cluster member") {
		t.Fatalf("err = %v, want it to say the host never became a member", err)
	}
	// The last thing the API said has to survive into the message, or the
	// operator is told it failed and not what failed.
	if !strings.Contains(err.Error(), "no cluster") {
		t.Fatalf("err = %v, want the API's own last words carried through", err)
	}
	if cl.statusCalls < 2 {
		t.Fatalf("status was read %d time(s); the join must be given time to take effect, "+
			"not judged on one look", cl.statusCalls)
	}
}

// The host's own API goes away while pve-cluster and corosync restart, which is
// the middle of every real join. Treating that as failure would fail every one.
func TestJoinWaitsThroughTheServiceRestart(t *testing.T) {
	shortJoinWait(t, 2*time.Second, 5*time.Millisecond)

	cl := &fakeCluster{guests: map[string][]Guest{}, statusErr: errors.New("connection refused")}
	cl.statusHook = func(call int, f *fakeCluster) {
		if call >= 4 {
			f.statusErr = nil
			f.quorate = true
			f.members = 3
			f.names = []string{"hv-01", "hv-02", "hv-06"}
		}
	}
	var phases []string
	var declared time.Duration

	if err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "pw", nil }},
		testEnv(&phases, &declared, nil), `{"peer":"10.0.0.1","fingerprint":"AA:BB"}`); err != nil {
		t.Fatalf("a join that succeeded after the services restarted was reported as failure: %v", err)
	}
	if len(phases) == 0 || phases[len(phases)-1] != phaseVerifying {
		t.Fatalf("phases = %v, want the last one to be %q — a resumed task has to be able to "+
			"tell that the join was issued and only the confirmation is outstanding",
			phases, phaseVerifying)
	}
}

// Joined, but the cluster has no quorum. That is a real problem and a DIFFERENT
// one: the join worked, and retrying it is the wrong response.
func TestJoinSaysSoWhenTheClusterItJoinedIsNotQuorate(t *testing.T) {
	shortJoinWait(t, time.Second, 5*time.Millisecond)

	cl := &fakeCluster{
		guests: map[string][]Guest{}, quorate: false, members: 3,
		names: []string{"hv-01", "hv-02", "hv-06"},
	}
	var phases []string
	var declared time.Duration

	err := Join(context.Background(),
		Deps{Cluster: cl, Node: "hv-06", JoinPass: func() (string, error) { return "pw", nil }},
		testEnv(&phases, &declared, nil), `{"peer":"10.0.0.1","fingerprint":"AA:BB"}`)
	if err == nil {
		t.Fatal("joining a cluster with no quorum was reported as a clean success")
	}
	if !strings.Contains(err.Error(), "do not retry") {
		t.Fatalf("err = %v, want it to say the join itself succeeded so retrying is wrong", err)
	}
}
