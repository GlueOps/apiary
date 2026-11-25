package fleetops

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// Every test in this package gets plenty of disk, so none depends on the disk of
// the machine running it. The tests about space set their own.
func init() {
	freeBytes = func(string) (uint64, error) { return 1 << 40, nil }
}

// fakeDetached is an in-memory DetachedRunner. Steps preset in `state` model an
// agent that was restarted part-way through a task.
type fakeDetached struct {
	state   map[string]StepState
	fail    map[string]error // what a step returns when it runs
	ran     []string         // steps actually run or re-attached, in order
	forgot  []string
	onRun   func(key string) // side effect, e.g. dpkg becoming clean after a repair
	blockOn string           // RunDetached of this key waits for ctx
	argv    map[string][]string
}

func newFakeDetached() *fakeDetached {
	return &fakeDetached{state: map[string]StepState{}, fail: map[string]error{}, argv: map[string][]string{}}
}

func (f *fakeDetached) State(_ context.Context, key string) (StepState, error) {
	return f.state[key], nil
}

func (f *fakeDetached) Forget(_ context.Context, key string) error {
	f.forgot = append(f.forgot, key)
	f.state[key] = StepNotStarted
	return nil
}

func (f *fakeDetached) RunDetached(ctx context.Context, _ func(string, string, ...any), key, name string, args ...string) error {
	f.argv[key] = append([]string{name}, args...)
	if f.state[key] == StepInterrupted {
		return ErrStepInterrupted
	}
	if key == f.blockOn {
		f.ran = append(f.ran, key)
		f.state[key] = StepRunning
		<-ctx.Done()
		return ctx.Err()
	}
	f.ran = append(f.ran, key)
	f.state[key] = StepFinished
	if f.onRun != nil {
		f.onRun(key)
	}
	return f.fail[key]
}

func (f *fakeDetached) ranStep(suffix string) bool {
	for _, k := range f.ran {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}

func (f *fakeDetached) index(suffix string) int {
	for i, k := range f.ran {
		if strings.HasSuffix(k, suffix) {
			return i
		}
	}
	return -1
}

// auditRunner answers `dpkg --audit` from a flag the test (or a repair) flips.
type auditRunner struct {
	dirty  bool
	audits int
}

func (a *auditRunner) Run(_ context.Context, log func(string, string, ...any), name string, args ...string) error {
	if name == "dpkg" && len(args) == 1 && args[0] == "--audit" {
		a.audits++
		if a.dirty {
			log("stdout", "The following packages are only half configured:")
			log("stdout", " proxmox-kernel-6.8.12-4-pve-signed")
			return errors.New("exit status 1")
		}
	}
	return nil
}

func aptDeps(r *auditRunner, det *fakeDetached, rebooted *bool) Deps {
	return Deps{
		Runner: r, Detached: det, Node: "hv-01",
		Reboot: func(context.Context) error { *rebooted = true; return nil },
	}
}

func aptEnv() *Env {
	var phases []string
	var declared time.Duration
	e := testEnv(&phases, &declared, nil)
	e.TaskID = "task-1"
	return e
}

// A dpkg that cannot be repaired ends the task with no reboot, and with no
// upgrade attempted on top of it.
func TestDpkgThatCannotBeRepairedFailsTheNodeUnrebooted(t *testing.T) {
	r := &auditRunner{dirty: true}
	det := newFakeDetached()
	rebooted := false

	err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "",
		`{"reboot":"always","allow_guest_downtime":true}`)

	if !errors.Is(err, ErrDpkgInconsistent) {
		t.Fatalf("err = %v, want ErrDpkgInconsistent", err)
	}
	if rebooted {
		t.Fatal("rebooted a host with a half-configured package database")
	}
	if det.ranStep("apt-dist-upgrade") {
		t.Fatal("ran dist-upgrade on top of a broken dpkg")
	}
	if !det.ranStep("pre-dpkg-configure") || !det.ranStep("pre-apt-fix") {
		t.Fatalf("the standard repair was not attempted: %v", det.ran)
	}
}

// Found broken before this task changed anything: repaired first, then upgraded.
func TestBrokenDpkgIsRepairedBeforeTheUpgrade(t *testing.T) {
	r := &auditRunner{dirty: true}
	det := newFakeDetached()
	det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-fix") {
			r.dirty = false
		}
	}
	rebooted := false

	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "",
		`{"reboot":"never"}`); err != nil {
		t.Fatalf("a repairable host failed: %v", err)
	}
	if fix, up := det.index("pre-apt-fix"), det.index("apt-dist-upgrade"); fix < 0 || up < 0 || fix > up {
		t.Fatalf("repair must come before the upgrade: %v", det.ran)
	}
}

// Left broken BY the upgrade and not repairable: no reboot.
func TestDpkgBrokenAfterTheUpgradeIsNeverRebooted(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			r.dirty = true // and nothing makes it clean again
		}
	}
	rebooted := false

	err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "",
		`{"reboot":"always","allow_guest_downtime":true}`)
	if !errors.Is(err, ErrDpkgInconsistent) {
		t.Fatalf("err = %v, want ErrDpkgInconsistent", err)
	}
	if rebooted {
		t.Fatal("rebooted after the upgrade left dpkg half-configured")
	}
	if !det.ranStep("post-dpkg-configure") {
		t.Fatalf("no repair was attempted after the upgrade: %v", det.ran)
	}
}

// apt failing is a failed upgrade even when dpkg is consistent: whatever it was
// going to install is not all there, so no reboot.
func TestAFailedUpgradeNeverReboots(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.fail["task-1-apt-dist-upgrade"] = errors.New("apt-get: exit status 100")
	rebooted := false

	err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "",
		`{"reboot":"always","allow_guest_downtime":true}`)
	if err == nil || !strings.Contains(err.Error(), "exit status 100") {
		t.Fatalf("err = %v, want the upgrade's own failure", err)
	}
	if rebooted {
		t.Fatal("rebooted after a failed upgrade")
	}
}

// The agent was killed mid-upgrade. Resume repairs dpkg FIRST, then runs the
// upgrade again — never straight into "dpkg was interrupted".
func TestAKilledUpgradeIsRepairedThenRunAgain(t *testing.T) {
	r := &auditRunner{dirty: true}
	det := newFakeDetached()
	det.state["task-1-apt-dist-upgrade"] = StepInterrupted
	det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-fix") {
			r.dirty = false
		}
	}
	rebooted := false

	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), phaseUpgrading,
		`{"reboot":"never"}`); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if fix, up := det.index("interrupted-apt-fix"), det.index("apt-dist-upgrade"); fix < 0 || up < 0 || fix > up {
		t.Fatalf("the repair must run before the upgrade is retried: %v", det.ran)
	}
	if len(det.forgot) != 1 || det.forgot[0] != "task-1-apt-dist-upgrade" {
		t.Fatalf("forgot %v; exactly the killed upgrade step should be cleared", det.forgot)
	}
}

// Still running from before the agent restarted: re-attach, no preflight, no
// second apt-get update.
func TestARunningUpgradeIsReattachedNotRestarted(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.state["task-1-apt-dist-upgrade"] = StepRunning
	rebooted := false

	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), phaseUpgrading,
		`{"reboot":"never"}`); err != nil {
		t.Fatal(err)
	}
	if det.ranStep("apt-update") || det.ranStep("pre-dpkg-configure") {
		t.Fatalf("restarted work that was already under way: %v", det.ran)
	}
	if r.audits != 1 {
		t.Fatalf("audited %d times; only the post-upgrade audit may run while dpkg is busy", r.audits)
	}
}

// The agent stopping mid-upgrade returns the cancellation, which bee treats as
// "resume on restart". Nothing runs after it — in particular no repair, which
// would be a second dpkg on top of the running one.
func TestTheAgentStoppingLeavesTheUpgradeRunning(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.blockOn = "task-1-apt-dist-upgrade"
	rebooted := false

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	err := DistUpgrade(ctx, aptDeps(r, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
	if r.audits != 1 {
		t.Fatalf("audited %d times; nothing may touch dpkg after the agent was told to stop", r.audits)
	}
}

func TestTooLittleDiskRefusesBeforeTouchingAnything(t *testing.T) {
	prev := freeBytes
	freeBytes = func(p string) (uint64, error) {
		if p == "/boot" {
			return 50 << 20, nil
		}
		return 1 << 40, nil
	}
	t.Cleanup(func() { freeBytes = prev })

	r := &auditRunner{}
	det := newFakeDetached()
	rebooted := false
	err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`)
	if !errors.Is(err, ErrInsufficientSpace) || !strings.Contains(err.Error(), "/boot") {
		t.Fatalf("err = %v, want ErrInsufficientSpace naming /boot", err)
	}
	if len(det.ran) != 0 {
		t.Fatalf("ran %v on a host without room to upgrade", det.ran)
	}
}

// Without a task id, steps cannot be told apart across tasks; refuse rather than
// replay another task's recorded result.
func TestDetachedStepsNeedATaskID(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	rebooted := false
	e := aptEnv()
	e.TaskID = ""
	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), e, "", `{"reboot":"never"}`); err == nil {
		t.Fatal("ran detached steps with no task id")
	}
	if len(det.ran) != 0 {
		t.Fatalf("ran %v with no task id", det.ran)
	}
}

// The repair resolves, it never removes: `apt-get -f install` without
// --no-remove removed a package an operator had installed with dpkg -i, and the
// upgrade and reboot then went ahead.
func TestTheRepairNeverRemovesPackages(t *testing.T) {
	r := &auditRunner{dirty: true}
	det := newFakeDetached()
	rebooted := false

	_ = DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`)
	argv := det.argv["task-1-pre-apt-fix"]
	if len(argv) == 0 {
		t.Fatalf("the repair did not run: %v", det.ran)
	}
	if !slices.Contains(argv, "--no-remove") {
		t.Fatalf("repair argv = %v; without --no-remove apt may resolve by removing somebody's package", argv)
	}
}

// When systemd cannot say whether the upgrade is still running, nothing is
// audited, repaired or rebooted: any of those on top of a live apt is the
// failure the detached runner exists to prevent.
func TestAnUpgradeOfUnknownStateIsNeitherRepairedNorRebooted(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.fail["task-1-apt-dist-upgrade"] = fmt.Errorf("step x: %w", ErrStepStateUnknown)
	det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			r.dirty = true // what an audit of a dpkg mid-unpack reads as
		}
	}
	rebooted := false

	err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "",
		`{"reboot":"always","allow_guest_downtime":true}`)
	if !errors.Is(err, ErrStepStateUnknown) {
		t.Fatalf("err = %v, want ErrStepStateUnknown", err)
	}
	if rebooted || det.ranStep("post-dpkg-configure") || det.ranStep("post-apt-fix") {
		t.Fatalf("acted on an upgrade that may still be running (rebooted=%v): %v", rebooted, det.ran)
	}
	if r.audits != 1 {
		t.Fatalf("audited %d times; only the preflight audit may run", r.audits)
	}
}

// pveupdate holding the lists lock is waited out, not a failed upgrade.
func TestAptUpdateWaitsOutABusyListsLock(t *testing.T) {
	prev := aptListsLockRetryWait
	aptListsLockRetryWait = time.Millisecond
	t.Cleanup(func() { aptListsLockRetryWait = prev })

	r := &auditRunner{}
	det := newFakeDetached()
	busy := errors.New("apt-get update: exit status 100\n  E: Could not get lock /var/lib/apt/lists/lock. It is held by process 4242 (apt-get)")
	det.fail["task-1-apt-update"] = busy
	det.fail["task-1-apt-update-retry1"] = busy
	rebooted := false

	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`); err != nil {
		t.Fatalf("a busy lists lock failed the upgrade: %v", err)
	}
	if !det.ranStep("apt-update-retry2") || !det.ranStep("apt-dist-upgrade") {
		t.Fatalf("did not retry the update and carry on: %v", det.ran)
	}
}

// Any other update failure is not retried.
func TestAptUpdateFailingOtherwiseIsNotRetried(t *testing.T) {
	r := &auditRunner{}
	det := newFakeDetached()
	det.fail["task-1-apt-update"] = errors.New("apt-get update: exit status 100\n  E: The repository is not signed.")
	rebooted := false

	if err := DistUpgrade(context.Background(), aptDeps(r, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`); err == nil {
		t.Fatal("a failed update did not fail the upgrade")
	}
	if det.ranStep("retry1") || det.ranStep("apt-dist-upgrade") {
		t.Fatalf("ran on after a real update failure: %v", det.ran)
	}
}

// A root filesystem with 2 GiB free cannot take one Proxmox kernel (~945 MiB
// installed, plus its ~119 MiB download) alongside the rest of a dist-upgrade.
func TestTheRootFloorCoversAKernelUpgrade(t *testing.T) {
	prev := freeBytes
	freeBytes = func(p string) (uint64, error) {
		if p == "/" {
			return 2 << 30, nil
		}
		return 1 << 40, nil
	}
	t.Cleanup(func() { freeBytes = prev })

	det := newFakeDetached()
	rebooted := false
	err := DistUpgrade(context.Background(), aptDeps(&auditRunner{}, det, &rebooted), aptEnv(), "", `{"reboot":"never"}`)
	if !errors.Is(err, ErrInsufficientSpace) {
		t.Fatalf("err = %v, want ErrInsufficientSpace for 2 GiB free on /", err)
	}
}

func TestDpkgAuditThatCannotRunIsNotReadAsHealthy(t *testing.T) {
	failing := runnerFunc(func(context.Context, func(string, string, ...any), string, ...string) error {
		return errors.New("executable file not found")
	})
	clean, _, err := dpkgAudit(context.Background(), failing)
	if err == nil || clean {
		t.Fatalf("clean=%v err=%v; an audit that could not run must refuse, not pass", clean, err)
	}
}
