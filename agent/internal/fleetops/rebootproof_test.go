package fleetops

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"
)

// fakeFacts is a host whose boot, kernel and packages the test controls.
type fakeFacts struct {
	boot, running, pin string
	kernels            []string
	pkgs               map[string]string
}

func (f *fakeFacts) BootID() (string, error)        { return f.boot, nil }
func (f *fakeFacts) RunningKernel() (string, error) { return f.running, nil }
func (f *fakeFacts) InstalledKernels() ([]string, error) {
	return append([]string{}, f.kernels...), nil
}
func (f *fakeFacts) PinnedKernel() (string, error) { return f.pin, nil }
func (f *fakeFacts) Packages(context.Context) (map[string]string, error) {
	return maps.Clone(f.pkgs), nil
}

const (
	oldKernel = "6.8.12-4-pve"
	newKernel = "6.8.12-10-pve"
)

// proofHost is a host before an upgrade that installs a new kernel and a new
// pve-manager. Nothing writes reboot-required: the new kernel alone has to be
// enough to reboot under if-needed.
type proofHost struct {
	facts    *fakeFacts
	det      *fakeDetached
	reboots  int
	declares int
	deps     Deps
	env      *Env
	logs     []string
	phase    string
	// rebootTakes: true models a real reboot — the shutdown stops the agent.
	// false models systemctl accepting the job and nothing happening.
	rebootTakes bool
}

func newProofHost(t *testing.T) *proofHost {
	t.Helper()
	h := &proofHost{
		facts: &fakeFacts{
			boot: "boot-1", running: oldKernel, kernels: []string{oldKernel},
			pkgs: map[string]string{"pve-manager": "8.2.4", "proxmox-kernel-" + oldKernel: "1", "curl": "7.88"},
		},
		det:         newFakeDetached(),
		rebootTakes: true,
	}
	h.det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			h.facts.pkgs["pve-manager"] = "8.2.7"
			h.facts.pkgs["proxmox-kernel-"+newKernel] = "1"
			h.facts.kernels = append(h.facts.kernels, newKernel)
		}
	}
	h.env = &Env{
		TaskID: "task-1",
		Logf: func(_ string, f string, a ...any) {
			h.logs = append(h.logs, fmt.Sprintf(f, a...))
		},
		Phase:   func(p string) { h.phase = p },
		Declare: func(context.Context, time.Duration, string) error { h.declares++; return nil },
	}
	h.deps = Deps{
		Runner: &auditRunner{}, Detached: h.det, Node: "hv-01",
		Facts: h.facts, RecordDir: t.TempDir(),
	}
	return h
}

// run starts the task, or resumes it in the recorded phase. A reboot that
// "takes" cancels the context exactly as SIGTERM from the shutdown does.
func (h *proofHost) run(spec string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.deps.Reboot = func(context.Context) error {
		h.reboots++
		if h.rebootTakes {
			cancel()
			return ctx.Err()
		}
		return nil
	}
	return DistUpgrade(ctx, h.deps, h.env, h.phase, spec)
}

// bootInto models the host coming back up.
func (h *proofHost) bootInto(kernel string) {
	h.facts.boot = "boot-2"
	h.facts.running = kernel
}

func (h *proofHost) logged(s string) bool {
	for _, l := range h.logs {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

const specAlways = `{"reboot":"always","allow_guest_downtime":true}`

// The whole cycle, end to end: upgrade, reboot, come back on the new kernel.
func TestARebootIsProvenByANewBootTheNewKernelAndTheNewPackages(t *testing.T) {
	h := newProofHost(t)

	if err := h.run(specAlways); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; a reboot under way must leave the task to resume, not finish it", err)
	}
	if h.phase != phaseRebooting {
		t.Fatalf("phase = %q, want rebooting", h.phase)
	}

	h.bootInto(newKernel)
	if err := h.run(specAlways); err != nil {
		t.Fatalf("a correctly upgraded and rebooted host failed verification: %v", err)
	}
	if h.reboots != 1 {
		t.Fatalf("rebooted %d times", h.reboots)
	}
	if !h.logged(`reboot-proof verdict: {`) || !h.logged(`"ok":true`) {
		t.Fatalf("no machine-readable verdict was logged: %v", h.logs)
	}
	if !h.logged(`"package":"pve-manager","old":"8.2.4","new":"8.2.7"`) {
		t.Fatalf("the changed packages were not recorded old → new: %v", h.logs)
	}
}

// THE BUG THIS SLICE ENDS. `systemctl reboot` returning used to finish the task
// as done — before the host went down, and whether or not it ever did. A host
// still up on the same boot must never be reported as upgraded.
func TestAHostThatNeverRebootsIsNeverReportedDone(t *testing.T) {
	h := newProofHost(t)
	h.rebootTakes = false

	err := h.run(specAlways)
	if !errors.Is(err, ErrDidNotReboot) {
		t.Fatalf("err = %v, want ErrDidNotReboot; the host never went down", err)
	}
	if h.reboots != maxRebootAttempts {
		t.Fatalf("asked for %d reboots, want %d before giving up", h.reboots, maxRebootAttempts)
	}
	if h.declares != h.reboots {
		t.Fatalf("declared %d times for %d reboots; every attempt must be announced first",
			h.declares, h.reboots)
	}
}

// The agent was restarted between declaring and rebooting (the watchdog, a
// crash). It resumes in phaseRebooting on the SAME boot, and must reboot rather
// than "verify" a reboot that never happened.
func TestResumingOnTheSameBootRebootsInsteadOfVerifying(t *testing.T) {
	h := newProofHost(t)
	if err := h.run(specAlways); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	// No bootInto: same boot id, as if the reboot never ran.
	if err := h.run(specAlways); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; the resumed agent should have asked for the reboot again", err)
	}
	if h.reboots != 2 {
		t.Fatalf("rebooted %d times, want a second request", h.reboots)
	}

	h.bootInto(newKernel)
	if err := h.run(specAlways); err != nil {
		t.Fatalf("the second reboot took and was not accepted: %v", err)
	}
}

func TestBootingTheOldKernelIsANamedFailureNotAnotherReboot(t *testing.T) {
	h := newProofHost(t)
	_ = h.run(specAlways)

	h.bootInto(oldKernel) // e.g. a grub default someone set by hand
	err := h.run(specAlways)
	if !errors.Is(err, ErrBootedOldKernel) {
		t.Fatalf("err = %v, want ErrBootedOldKernel", err)
	}
	if !strings.Contains(err.Error(), newKernel) {
		t.Fatalf("%q does not say which kernel was expected", err)
	}
	if h.reboots != 1 {
		t.Fatalf("rebooted again (%d); another reboot boots the same kernel", h.reboots)
	}
}

// A pinned kernel is the expectation, even when it is not the newest.
func TestAPinnedKernelIsWhatIsExpected(t *testing.T) {
	h := newProofHost(t)
	h.facts.pin = oldKernel
	h.facts.kernels = []string{oldKernel, "6.5.13-6-pve"}
	_ = h.run(specAlways)

	h.bootInto(oldKernel)
	if err := h.run(specAlways); err != nil {
		t.Fatalf("booting the pinned kernel was failed: %v", err)
	}
}

func TestAPackageRolledBackIsANamedFailure(t *testing.T) {
	h := newProofHost(t)
	_ = h.run(specAlways)

	h.bootInto(newKernel)
	h.facts.pkgs["pve-manager"] = "8.2.4"
	err := h.run(specAlways)
	if !errors.Is(err, ErrPackageNotUpgraded) || !strings.Contains(err.Error(), "pve-manager is 8.2.4, expected 8.2.7") {
		t.Fatalf("err = %v, want ErrPackageNotUpgraded naming pve-manager", err)
	}
}

// Comparing against the recorded expectation, never "newest available": a
// kernel installed after the reboot does not fail a correct upgrade.
func TestALaterKernelDoesNotFailACorrectUpgrade(t *testing.T) {
	h := newProofHost(t)
	_ = h.run(specAlways)

	h.bootInto(newKernel)
	h.facts.kernels = append(h.facts.kernels, "6.8.12-11-pve")
	if err := h.run(specAlways); err != nil {
		t.Fatalf("a kernel newer than the expectation failed the node: %v", err)
	}
}

// A new kernel needs a reboot even if nothing wrote /run/reboot-required.
func TestANewKernelRebootsUnderIfNeeded(t *testing.T) {
	h := newProofHost(t)
	if err := h.run(`{"reboot":"if-needed","allow_guest_downtime":true}`); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v; a new kernel was installed and the host did not reboot", err)
	}
	if h.reboots != 1 {
		t.Fatalf("reboots = %d", h.reboots)
	}
}

func TestNoKernelChangeNoRebootUnderIfNeeded(t *testing.T) {
	h := newProofHost(t)
	h.det.onRun = func(key string) {
		if strings.HasSuffix(key, "apt-dist-upgrade") {
			h.facts.pkgs["curl"] = "7.88.1"
		}
	}
	if err := h.run(`{"reboot":"if-needed"}`); err != nil {
		t.Fatal(err)
	}
	if h.reboots != 0 {
		t.Fatal("rebooted for a curl update")
	}
}

// A task that rebooted before records existed (an agent upgraded mid-task) is
// unproven, and says so — it is not failed for lacking a record nobody wrote.
func TestAMissingExpectationIsUnprovenNotFailed(t *testing.T) {
	h := newProofHost(t)
	h.phase = phaseRebooting
	h.bootInto(newKernel)
	if err := h.run(specAlways); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !h.logged("cannot be proven") {
		t.Fatalf("an unproven reboot was not called unproven: %v", h.logs)
	}
}

func TestKernelReleasesSortByVersionNotText(t *testing.T) {
	ks := []string{"6.8.12-10-pve", "6.14.8-2-pve", "6.8.12-4-pve", "6.5.13-6-pve"}
	got, _, err := expectedKernel(&fakeFacts{kernels: ks})
	if err != nil || got != "6.14.8-2-pve" {
		t.Fatalf("newest = %q (%v), want 6.14.8-2-pve", got, err)
	}
	if compareKernel("6.8.12-10-pve", "6.8.12-4-pve") <= 0 {
		t.Fatal("-10 sorted before -4")
	}
}

func TestPackageDiffSeesInstallsUpgradesAndRemovals(t *testing.T) {
	got := diffPackages(
		map[string]string{"a": "1", "gone": "3", "same": "x"},
		map[string]string{"a": "2", "new": "1", "same": "x"})
	want := []PackageChange{{"a", "1", "2"}, {"gone", "3", ""}, {"new", "", "1"}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
