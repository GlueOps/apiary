package fleetops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
)

// ErrDpkgInconsistent reports a host whose package database is half-configured
// and could not be repaired automatically.
//
// It ends the task WITHOUT a reboot, always. Rebooting with a half-installed
// kernel, initramfs or bootloader is how a hypervisor fails to come back, and a
// hypervisor that does not come back is the one failure a maintenance window has
// no way to undo remotely.
var ErrDpkgInconsistent = errors.New(
	"dpkg is left inconsistent and could not be repaired automatically; this host has NOT " +
		"been rebooted and needs a person")

// ErrInsufficientSpace reports a filesystem too full to upgrade into.
var ErrInsufficientSpace = errors.New("not enough free space to upgrade safely")

// Free space floors. Running out part-way is a reliable way to produce exactly
// the half-configured state the rest of this file exists to prevent, so it is
// checked before the first package is touched. Vars so a test can move them.
//
// Root: ONE new Proxmox kernel is about 945 MiB installed (its modules, mostly)
// plus a ~119 MiB .deb in /var/cache/apt/archives while it unpacks — already
// over the 1 GiB this used to be — and a dist-upgrade unpacks everything before
// it removes anything, the old kernel included. 3 GiB covers a kernel, its
// download, and a release's worth of other packages alongside it, with margin.
// Boot: a kernel plus its initramfs is around 120 MiB.
var (
	minFreeRoot uint64 = 3 << 30   // 3 GiB
	minFreeBoot uint64 = 200 << 20 // 200 MiB
)

// freeBytes reports the space available to root on the filesystem holding path.
// A var so tests do not depend on the disk of the machine running them.
var freeBytes = func(path string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Bsize), nil
}

// aptLockWait is how long apt waits for another dpkg user — unattended-upgrades,
// a person's apt — instead of failing on the dpkg lock at once.
//
// It covers the DPKG lock only. `apt-get update` takes /var/lib/apt/lists/lock,
// which no apt option waits on: with this set, apt 3.0.3 (trixie, which PVE 9
// is built on) still fails at once with exit 100 and "Could not get lock
// /var/lib/apt/lists/lock". Proxmox's daily pveupdate runs `apt-get update`, so
// that collision is real; aptUpdate retries it instead.
const aptLockWait = "DPkg::Lock::Timeout=300"

// The retry for a busy lists lock. Ten tries 30s apart is the same five minutes
// aptLockWait gives the dpkg lock. Vars so a test can shorten them.
var (
	aptListsLockRetries   = 10
	aptListsLockRetryWait = 30 * time.Second
)

// The options every dpkg-invoking step shares: keep a locally modified conffile
// and never stop to ask about one.
var dpkgConfOpts = []string{
	"-o", aptLockWait,
	"-o", "Dpkg::Options::=--force-confdef",
	"-o", "Dpkg::Options::=--force-confold",
}

// aptUpgrade is the package half of a dist-upgrade: everything between the
// reboot preflight and the reboot.
//
// Resumable. Each step is a detached step keyed by the task, so an agent that is
// restarted part-way — by the watchdog, a crash, its own upgrade — comes back
// here and finds the step still running (re-attach), finished (use the result),
// or killed (repair first). It never starts a second apt, and it never runs one
// on top of a dpkg somebody else interrupted.
func aptUpgrade(ctx context.Context, d Deps, e *Env) error {
	det, err := detachedFor(d, e)
	if err != nil {
		return err
	}
	key := func(step string) string { return e.TaskID + "-" + step }
	upgradeKey := key("apt-dist-upgrade")

	st, err := det.State(ctx, upgradeKey)
	if err != nil {
		return err
	}
	switch st {
	case StepNotStarted:
		// The first time through. Check the host is fit to upgrade before
		// touching anything — including repairing what somebody else left.
		if err := aptPreflight(ctx, d, e, det, key("pre")); err != nil {
			return err
		}
		e.Logf("agent", "updating package lists")
		if err := aptUpdate(ctx, det, e, key("apt-update")); err != nil {
			return err
		}
	case StepInterrupted:
		// This task's own upgrade was killed. dpkg is very likely mid-change, so
		// repair comes before anything else, and a repair that does not take
		// ends the task here.
		e.Logf("agent", "the dist-upgrade this task started was killed before it finished; "+
			"repairing the package database before going any further")
		if err := repairDpkg(ctx, d, e, det, key("interrupted")); err != nil {
			return err
		}
		if err := det.Forget(ctx, upgradeKey); err != nil {
			return err
		}
	case StepRunning:
		e.Logf("agent", "the dist-upgrade this task started is still running; waiting for it")
	case StepFinished:
		e.Logf("agent", "the dist-upgrade this task started has already finished")
	}

	e.Logf("agent", "running dist-upgrade")
	args := append(append([]string{}, dpkgConfOpts...), "-y", "dist-upgrade")
	upErr := det.RunDetached(ctx, e.Logf, upgradeKey, "apt-get", args...)
	if upErr != nil && ctx.Err() != nil {
		// The agent is stopping; the upgrade is not. Resume will re-attach.
		return ctx.Err()
	}
	if errors.Is(upErr, ErrStepStateUnknown) {
		// Not a result: apt may well still be running. Auditing now would read a
		// live dpkg as broken, and repairing would be a second writer on it.
		return fmt.Errorf("could not tell whether the dist-upgrade is still running, so the "+
			"package database was neither audited nor repaired and the host has not been "+
			"rebooted: %w", upErr)
	}

	// Audit whether or not apt said it succeeded. apt exiting non-zero is the
	// common way to be left half-configured, and apt exiting 0 is not proof of
	// the opposite.
	clean, detail, err := dpkgAudit(ctx, d.Runner)
	if err != nil {
		return err
	}
	if !clean {
		e.Logf("agent", "dpkg reports problems after the upgrade:\n%s", detail)
		if err := repairDpkg(ctx, d, e, det, key("post")); err != nil {
			if upErr != nil {
				return fmt.Errorf("%w (the upgrade itself failed: %w)", err, upErr)
			}
			return err
		}
	}
	if upErr != nil {
		// Repaired, or never broken, but the upgrade did not complete. No reboot:
		// whatever it was going to install is not all there.
		return fmt.Errorf("dist-upgrade failed; dpkg is consistent and the host has not been "+
			"rebooted: %w", upErr)
	}
	return nil
}

// aptPreflight refuses to start an upgrade on a host that cannot take one.
func aptPreflight(ctx context.Context, d Deps, e *Env, det DetachedRunner, key string) error {
	for _, c := range []struct {
		path  string
		floor uint64
	}{{"/", minFreeRoot}, {"/boot", minFreeBoot}} {
		free, err := freeBytes(c.path)
		if err != nil {
			// /boot may not exist as a path at all on some layouts; a missing
			// path is not a full disk. / always exists, so it cannot hide here.
			e.Logf("agent", "could not read free space on %s (%v); not checking it", c.path, err)
			continue
		}
		if free < c.floor {
			return fmt.Errorf("%w: %s has %s free and needs at least %s",
				ErrInsufficientSpace, c.path, humanBytes(free), humanBytes(c.floor))
		}
	}

	clean, detail, err := dpkgAudit(ctx, d.Runner)
	if err != nil {
		return err
	}
	if !clean {
		// Found broken, before this task changed anything. Repairing it is what a
		// person would do first anyway, and it is recorded so nobody later reads
		// this upgrade's log and wonders why dpkg --configure ran.
		e.Logf("agent", "the package database was already inconsistent before this upgrade "+
			"started; repairing it first:\n%s", detail)
		if err := repairDpkg(ctx, d, e, det, key); err != nil {
			return err
		}
	}
	return nil
}

// repairDpkg runs the standard repair and audits again.
//
// The standard repair, and only it: `dpkg --configure -a` finishes whatever was
// unpacked and not configured, and `apt-get -f install` resolves what that
// leaves. Anything cleverer is a decision about somebody's host that belongs to
// a person.
//
// --no-remove, because resolving can mean REMOVING. Without it, a package an
// operator installed by hand with `dpkg -i` and whose dependencies no repository
// satisfies was removed in testing, and the upgrade and reboot went ahead. With
// it apt aborts instead, the audit below still finds the problem, and the task
// stops at ErrDpkgInconsistent for a person — which is the right owner of the
// decision to remove somebody's package.
func repairDpkg(ctx context.Context, d Deps, e *Env, det DetachedRunner, key string) error {
	e.Logf("agent", "repairing: dpkg --configure -a")
	if err := runStep(ctx, det, e, key+"-dpkg-configure", true,
		"dpkg", "--force-confdef", "--force-confold", "--configure", "-a"); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, ErrStepStateUnknown) {
			return err
		}
		e.Logf("agent", "dpkg --configure -a failed: %v", err)
	}
	e.Logf("agent", "repairing: apt-get -f install --no-remove")
	fix := append(append([]string{}, dpkgConfOpts...), "-y", "--no-remove", "-f", "install")
	if err := runStep(ctx, det, e, key+"-apt-fix", true, "apt-get", fix...); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, ErrStepStateUnknown) {
			return err
		}
		e.Logf("agent", "apt-get -f install --no-remove failed: %v", err)
	}

	clean, detail, err := dpkgAudit(ctx, d.Runner)
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("%w:\n%s", ErrDpkgInconsistent, detail)
	}
	e.Logf("agent", "the package database is consistent again")
	return nil
}

// runStep runs one detached step. With rerunIfKilled, a step that was killed
// part-way is cleared and run again — only for steps that are safe to repeat
// (apt-get update, and the repair commands, which exist to be re-run).
func runStep(ctx context.Context, det DetachedRunner, e *Env, key string, rerunIfKilled bool, name string, args ...string) error {
	err := det.RunDetached(ctx, e.Logf, key, name, args...)
	if rerunIfKilled && errors.Is(err, ErrStepInterrupted) {
		e.Logf("agent", "step %s was killed part-way; it is safe to repeat, so running it again", key)
		if ferr := det.Forget(ctx, key); ferr != nil {
			return ferr
		}
		err = det.RunDetached(ctx, e.Logf, key, name, args...)
	}
	return err
}

// aptUpdate runs `apt-get update`, retrying while somebody else holds the
// lists lock. See aptLockWait for why apt cannot be told to wait itself.
//
// Each attempt is its own step key: a finished step's result is a recorded
// fact that is never forgotten, so a retry cannot reuse the key that failed.
// On resume the earlier attempts replay their recorded results and the loop
// carries on from the first one that has none.
func aptUpdate(ctx context.Context, det DetachedRunner, e *Env, key string) error {
	for attempt := 0; ; attempt++ {
		k := key
		if attempt > 0 {
			k = fmt.Sprintf("%s-retry%d", key, attempt)
		}
		err := runStep(ctx, det, e, k, true, "apt-get", "update")
		if err == nil || !listsLockBusy(err) || attempt >= aptListsLockRetries {
			return err
		}
		e.Logf("agent", "the package lists are locked by another apt-get update (Proxmox's "+
			"pveupdate, usually); trying again in %s", aptListsLockRetryWait)
		if !sleepCtx(ctx, aptListsLockRetryWait) {
			return ctx.Err()
		}
	}
}

func listsLockBusy(err error) bool {
	return strings.Contains(err.Error(), "/var/lib/apt/lists/lock")
}

// dpkgAudit reports whether dpkg considers its database consistent.
//
// `dpkg --audit` prints nothing for a healthy system. Depending on the dpkg
// version it exits 0 or 1 when it does find something, so the OUTPUT is the
// verdict, and an exit failure with no output is "could not ask" — which refuses
// rather than being read as healthy.
func dpkgAudit(ctx context.Context, r Runner) (clean bool, detail string, err error) {
	if r == nil {
		return false, "", errors.New("dpkg audit: no command runner")
	}
	var lines []string
	collect := func(_ string, format string, args ...any) {
		if l := strings.TrimRight(fmt.Sprintf(format, args...), " \t"); strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	runErr := r.Run(ctx, collect, "dpkg", "--audit")
	if len(lines) > 0 {
		return false, strings.Join(lines, "\n"), nil
	}
	if runErr != nil {
		return false, "", fmt.Errorf("could not audit the package database: %w", runErr)
	}
	return true, "", nil
}

// detachedFor picks the runner for package steps.
//
// The detached runner needs a task id, because the id is what makes a step's
// key mean "this step of THIS task". Without one, two tasks would share keys and
// the second would replay the first's recorded result as its own.
func detachedFor(d Deps, e *Env) (DetachedRunner, error) {
	if d.Detached == nil {
		return inlineRunner{r: d.Runner}, nil
	}
	if strings.TrimSpace(e.TaskID) == "" {
		return nil, errors.New("dist-upgrade: no task id, so its steps cannot be tracked " +
			"across an agent restart; refusing rather than risking a second apt")
	}
	return d.Detached, nil
}

// inlineRunner runs steps as the agent's own children, with no record.
//
// The behaviour before detached steps existed, used when Deps.Detached is nil:
// in tests, and in production ONLY when the agent could not create its steps
// directory (see fleetDeps). It is NOT a fallback for a host without systemd —
// nothing probes for one; there, systemd-run fails to start the step and the
// task fails before apt has run. It cannot survive an agent restart, which is
// exactly why production wires the systemd runner instead.
type inlineRunner struct{ r Runner }

func (i inlineRunner) RunDetached(ctx context.Context, log func(stream, format string, args ...any), _ string, name string, args ...string) error {
	if i.r == nil {
		return errors.New("no command runner")
	}
	return i.r.Run(ctx, log, name, args...)
}
func (inlineRunner) State(context.Context, string) (StepState, error) { return StepNotStarted, nil }
func (inlineRunner) Forget(context.Context, string) error             { return nil }

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
