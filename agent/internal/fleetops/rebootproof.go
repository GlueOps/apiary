package fleetops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Proof of reboot.
//
// The "back" check used to be: the task succeeded, the agent is heartbeating,
// the cluster is quorate. A node that never rebooted at all passes that, and so
// does one that rebooted into the kernel it was already running. Neither has
// the fix the window existed to apply.
//
// So the upgrade records what it EXPECTS before the reboot — this boot's id,
// the kernel it should come back on, the packages it changed — and the agent
// that comes back compares against that record, not against "newest available".
// A kernel published after the run started must not turn a correctly upgraded
// node into a failed one.
//
// Each mismatch is its own error, because each points somewhere different.

// ErrDidNotReboot: the boot id is the one recorded before the reboot.
var ErrDidNotReboot = errors.New("did not reboot")

// ErrBootedOldKernel: the host rebooted, but not into the kernel the upgrade
// installed (or the one pinned with proxmox-boot-tool).
var ErrBootedOldKernel = errors.New("booted the old kernel")

// ErrPackageNotUpgraded: a package the upgrade changed is not at the version it
// was changed to.
var ErrPackageNotUpgraded = errors.New("package not at its upgraded version")

// maxRebootAttempts bounds how many times one task asks for a reboot that does
// not happen. A second try covers the agent being killed between declaring and
// rebooting; more than that is a host refusing to reboot, which needs a person.
const maxRebootAttempts = 2

// HostFacts reads what proves a reboot. An interface so the proof can be tested
// without a Linux host, and so its absence is explicit: nil Deps.Facts means no
// proof, never a guess.
type HostFacts interface {
	// BootID is /proc/sys/kernel/random/boot_id: new on every boot.
	BootID() (string, error)
	// RunningKernel is `uname -r`.
	RunningKernel() (string, error)
	// InstalledKernels lists the kernel releases with an image in /boot.
	InstalledKernels() ([]string, error)
	// PinnedKernel is the kernel proxmox-boot-tool will boot next if one is
	// pinned, "" when none is.
	PinnedKernel() (string, error)
	// Packages is the installed version of every package.
	Packages(ctx context.Context) (map[string]string, error)
}

// PackageChange is one package the upgrade changed. Old "" is a new install
// (a new kernel is one), New "" a removal.
type PackageChange struct {
	Package string `json:"package"`
	Old     string `json:"old,omitempty"`
	New     string `json:"new,omitempty"`
}

// RebootExpectation is recorded before the reboot and checked after it.
type RebootExpectation struct {
	TaskID         string          `json:"task_id"`
	BootID         string          `json:"boot_id"`
	RunningKernel  string          `json:"running_kernel"`
	ExpectedKernel string          `json:"expected_kernel,omitempty"`
	KernelPinned   bool            `json:"kernel_pinned,omitempty"`
	Changed        []PackageChange `json:"changed"`
	// ChangedKnown is false when the before-snapshot was never taken (an agent
	// upgraded mid-task), so an empty Changed is not read as "nothing changed".
	ChangedKnown   bool      `json:"changed_known"`
	RebootReasons  []string  `json:"reboot_reasons,omitempty"`
	RebootAttempts int       `json:"reboot_attempts"`
	RecordedAt     time.Time `json:"recorded_at"`
}

// proofLogPrefix marks the machine-readable lines. The control plane does not
// parse them yet; they are stable so that it can, and so a person can grep.
const proofLogPrefix = "reboot-proof"

// proof is the per-task view of the record directory. Nil when proof is not
// possible here, and every method is nil-safe so the call sites read straight.
type proof struct {
	facts HostFacts
	dir   string
	key   string
}

func proofFor(d Deps, e *Env) *proof {
	if d.Facts == nil || d.RecordDir == "" || strings.TrimSpace(e.TaskID) == "" {
		return nil
	}
	return &proof{facts: d.Facts, dir: d.RecordDir, key: stepKey(e.TaskID)}
}

func (p *proof) path(name string) string { return filepath.Join(p.dir, p.key+"."+name+".json") }

// snapshotBefore records every installed package version before the upgrade
// touches anything. Kept if already there: a resumed task must diff against the
// state before ITS upgrade, not against a half-upgraded host.
func (p *proof) snapshotBefore(ctx context.Context, e *Env) error {
	if p == nil {
		return nil
	}
	if _, err := os.Stat(p.path("pkgs-before")); err == nil {
		return nil
	}
	pkgs, err := p.facts.Packages(ctx)
	if err != nil {
		return fmt.Errorf("could not record the installed packages before upgrading, so the "+
			"upgrade could not be verified afterwards: %w", err)
	}
	return writeJSONAtomic(p.path("pkgs-before"), pkgs)
}

// expect builds the expectation after the upgrade and before any reboot.
func (p *proof) expect(ctx context.Context, e *Env) (*RebootExpectation, error) {
	if p == nil {
		return nil, nil
	}
	x := &RebootExpectation{TaskID: e.TaskID, RecordedAt: time.Now().UTC()}
	var err error
	if x.BootID, err = p.facts.BootID(); err != nil {
		return nil, fmt.Errorf("could not read this boot's id: %w", err)
	}
	if x.RunningKernel, err = p.facts.RunningKernel(); err != nil {
		return nil, fmt.Errorf("could not read the running kernel: %w", err)
	}
	x.ExpectedKernel, x.KernelPinned, err = expectedKernel(p.facts)
	if err != nil {
		// Recorded as unknown rather than refused: the boot id still proves the
		// reboot, and a missing /boot glob is not a reason to leave a CVE open.
		e.Logf("agent", "could not work out which kernel the host will boot (%v); the kernel "+
			"will not be checked after the reboot", err)
	}

	after, err := p.facts.Packages(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read the installed packages after upgrading: %w", err)
	}
	var before map[string]string
	if err := readJSON(p.path("pkgs-before"), &before); err == nil {
		x.Changed, x.ChangedKnown = diffPackages(before, after), true
	} else {
		e.Logf("agent", "no record of the packages before this upgrade (%v); which packages it "+
			"changed is unknown, so they will not be checked after the reboot", err)
	}

	if x.ExpectedKernel != "" && x.ExpectedKernel != x.RunningKernel {
		x.RebootReasons = append(x.RebootReasons, "new kernel "+x.ExpectedKernel)
	}
	for _, c := range x.Changed {
		if c.Package == "pve-qemu-kvm" && c.New != "" {
			x.RebootReasons = append(x.RebootReasons, "new pve-qemu-kvm "+c.New)
		}
	}
	if rebootRequired() {
		x.RebootReasons = append(x.RebootReasons, "reboot-required")
	}
	return x, nil
}

func (p *proof) save(x *RebootExpectation) error {
	if p == nil || x == nil {
		return nil
	}
	return writeJSONAtomic(p.path("reboot"), x)
}

func (p *proof) load() (*RebootExpectation, error) {
	var x RebootExpectation
	if err := readJSON(p.path("reboot"), &x); err != nil {
		return nil, err
	}
	return &x, nil
}

// rebootVerdict is what the check after the reboot found.
type rebootVerdict struct {
	// Rebooted is false when the boot id did not change.
	Rebooted bool
	Err      error
	Notes    []string
}

// check compares this boot against the recorded expectation.
func (p *proof) check(ctx context.Context, x *RebootExpectation) rebootVerdict {
	now, err := p.facts.BootID()
	if err != nil {
		return rebootVerdict{Err: fmt.Errorf("could not read the boot id, so the reboot cannot "+
			"be proven: %w", err)}
	}
	if now == x.BootID {
		return rebootVerdict{Err: fmt.Errorf("%w: the boot id is still %s, the one recorded "+
			"before the reboot was requested", ErrDidNotReboot, now)}
	}
	v := rebootVerdict{Rebooted: true}

	if x.ExpectedKernel != "" {
		running, err := p.facts.RunningKernel()
		switch {
		case err != nil:
			v.Err = fmt.Errorf("could not read the running kernel: %w", err)
			return v
		case running != x.ExpectedKernel:
			why := "the newest installed"
			if x.KernelPinned {
				why = "the pinned kernel"
			}
			v.Err = fmt.Errorf("%w: running %s, expected %s (%s); the host rebooted but the "+
				"upgrade's kernel is not in effect", ErrBootedOldKernel, running, x.ExpectedKernel, why)
			return v
		}
	}

	if x.ChangedKnown && len(x.Changed) > 0 {
		pkgs, err := p.facts.Packages(ctx)
		if err != nil {
			v.Err = fmt.Errorf("could not read the installed packages: %w", err)
			return v
		}
		var wrong []string
		for _, c := range x.Changed {
			got, installed := pkgs[c.Package]
			switch {
			case c.New == "" && installed:
				wrong = append(wrong, fmt.Sprintf("%s is installed (%s) but the upgrade removed it", c.Package, got))
			case c.New != "" && got != c.New:
				if !installed {
					got = "not installed"
				}
				wrong = append(wrong, fmt.Sprintf("%s is %s, expected %s", c.Package, got, c.New))
			}
		}
		if len(wrong) > 0 {
			v.Err = fmt.Errorf("%w: %s", ErrPackageNotUpgraded, strings.Join(wrong, "; "))
			return v
		}
	}

	// Noted, not failed: /run is tmpfs, so this is something installed SINCE
	// the boot — unattended-upgrades, a person — not this upgrade left undone.
	if rebootRequired() {
		v.Notes = append(v.Notes, "reboot-required is set again; something was installed after this boot")
	}
	return v
}

// logExpectation emits the record as one stable, machine-readable line.
func logProof(e *Env, what string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	e.Logf("agent", "%s %s: %s", proofLogPrefix, what, b)
}

// expectedKernel is the kernel the next boot should run: the pinned one if
// proxmox-boot-tool has one pinned, otherwise the newest image in /boot.
func expectedKernel(f HostFacts) (kernel string, pinned bool, err error) {
	if pin, err := f.PinnedKernel(); err == nil && pin != "" {
		return pin, true, nil
	}
	ks, err := f.InstalledKernels()
	if err != nil {
		return "", false, err
	}
	if len(ks) == 0 {
		return "", false, errors.New("no kernel images found in /boot")
	}
	sort.Slice(ks, func(i, j int) bool { return compareKernel(ks[i], ks[j]) < 0 })
	return ks[len(ks)-1], false, nil
}

// compareKernel orders kernel releases: digit runs numerically, everything else
// as text. 6.8.12-10-pve after 6.8.12-4-pve, and 6.14.8-2-pve after both.
func compareKernel(a, b string) int {
	for a != "" && b != "" {
		ca, ra := kernelChunk(a)
		cb, rb := kernelChunk(b)
		na, ea := strconv.ParseUint(ca, 10, 64)
		nb, eb := strconv.ParseUint(cb, 10, 64)
		switch {
		case ea == nil && eb == nil:
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
		case ca != cb:
			return strings.Compare(ca, cb)
		}
		a, b = ra, rb
	}
	return strings.Compare(a, b)
}

func kernelChunk(s string) (chunk, rest string) {
	digit := s[0] >= '0' && s[0] <= '9'
	i := 1
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') == digit {
		i++
	}
	return s[:i], s[i:]
}

func diffPackages(before, after map[string]string) []PackageChange {
	var out []PackageChange
	for p, nv := range after {
		if ov := before[p]; ov != nv {
			out = append(out, PackageChange{Package: p, Old: ov, New: nv})
		}
	}
	for p, ov := range before {
		if _, ok := after[p]; !ok {
			out = append(out, PackageChange{Package: p, Old: ov})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Package < out[j].Package })
	return out
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ------------------------------------------------------------ the local host

// LocalFacts reads the facts from this host.
func LocalFacts(r Runner) HostFacts { return localFacts{r: r} }

type localFacts struct{ r Runner }

func readTrimmed(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return s, nil
}

func (localFacts) BootID() (string, error) { return readTrimmed("/proc/sys/kernel/random/boot_id") }

// RunningKernel reads /proc rather than calling uname(2), which has no
// portable Go binding and is the same value.
func (localFacts) RunningKernel() (string, error) {
	return readTrimmed("/proc/sys/kernel/osrelease")
}

func (localFacts) InstalledKernels() ([]string, error) {
	m, err := filepath.Glob("/boot/vmlinuz-*")
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(m))
	for _, p := range m {
		out = append(out, strings.TrimPrefix(filepath.Base(p), "vmlinuz-"))
	}
	return out, nil
}

// PinnedKernel: a one-off `kernel pin --next-boot` wins over a standing pin.
func (localFacts) PinnedKernel() (string, error) {
	for _, p := range []string{"/etc/kernel/next-boot-pin", "/etc/kernel/proxmox-boot-pin"} {
		if s, err := readTrimmed(p); err == nil {
			return s, nil
		}
	}
	return "", nil
}

func (f localFacts) Packages(ctx context.Context) (map[string]string, error) {
	if f.r == nil {
		return nil, errors.New("no command runner")
	}
	pkgs := map[string]string{}
	collect := func(_ string, format string, args ...any) {
		// "<package> <version> <status>": only installed packages count; a
		// removed-not-purged one still has a version here.
		fs := strings.Fields(fmt.Sprintf(format, args...))
		if len(fs) >= 3 && fs[2] == "installed" {
			pkgs[fs[0]] = fs[1]
		}
	}
	if err := f.r.Run(ctx, collect, "dpkg-query", "-W",
		"-f=${binary:Package} ${Version} ${db:Status-Status}\n"); err != nil {
		return nil, fmt.Errorf("dpkg-query: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, errors.New("dpkg-query listed no installed packages")
	}
	return pkgs, nil
}

// proveReboot is the first thing the agent does after the reboot it asked for.
//
// Same boot id: the reboot did not happen — the agent was restarted between
// declaring and rebooting, or the reboot was refused. Asked for again, up to
// maxRebootAttempts, and then a named failure. Any other mismatch is a named
// failure straight away: rebooting again will not change which kernel boots.
func proveReboot(ctx context.Context, d Deps, e *Env, spec DistUpgradeSpec) error {
	p := proofFor(d, e)
	if p == nil {
		e.Logf("agent", "no host facts or record directory here; the reboot is not proven")
		return nil
	}
	x, err := p.load()
	if err != nil {
		// An agent upgraded mid-task, or a reboot requested before records
		// existed. Unproven is said, not failed: nothing here can tell what the
		// upgrade was meant to change.
		e.Logf("agent", "no expectation was recorded before this reboot (%v); it cannot be proven", err)
		return nil
	}

	v := p.check(ctx, x)
	for _, n := range v.Notes {
		e.Logf("agent", "%s", n)
	}
	if !v.Rebooted && errors.Is(v.Err, ErrDidNotReboot) && x.RebootAttempts < maxRebootAttempts {
		e.Logf("agent", "%v; asking again (attempt %d of %d)", v.Err, x.RebootAttempts+1, maxRebootAttempts)
		e.Phase(phaseRebooting)
		return declareAndReboot(ctx, d, e, spec, p, x)
	}
	result := map[string]any{"boot_id_before": x.BootID, "expected_kernel": x.ExpectedKernel,
		"attempts": x.RebootAttempts, "ok": v.Err == nil}
	if v.Err != nil {
		result["failure"] = v.Err.Error()
		logProof(e, "verdict", result)
		return v.Err
	}
	logProof(e, "verdict", result)
	e.Logf("agent", "reboot proven: new boot, running %s, every changed package at its new version",
		orUnknown(x.ExpectedKernel))
	return nil
}

func orUnknown(s string) string {
	if s == "" {
		return "an unchecked kernel"
	}
	return s
}
