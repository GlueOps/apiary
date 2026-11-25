package fleetops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// A DetachedRunner runs one step of a task in a process the agent does not own,
// so the step outlives the agent.
//
// WHY THIS EXISTS. apt used to run as a child of the agent, under
// exec.CommandContext. Anything that stopped the agent mid-upgrade — the
// systemd watchdog, an OOM kill, a crash, the agent upgrading itself — killed
// dpkg with it, because systemd's default KillMode=control-group signals every
// process in the unit's cgroup, detached or not. What is left is a host with
// half-configured packages, and a resumed task that ran `apt-get dist-upgrade`
// straight into "dpkg was interrupted, you must manually run 'dpkg --configure
// -a'". A half-installed kernel or bootloader is how a hypervisor fails to boot.
//
// A step is keyed. The same key always means the same step of the same task, so
// RunDetached is idempotent: it starts the step, or re-attaches to it while it is
// still running, or returns the result it already recorded. It never starts a
// second copy, which for dpkg would be a second writer on the same database.
type DetachedRunner interface {
	// RunDetached runs name+args as the step key and waits for it, streaming its
	// output to log. Cancelling ctx stops the WAITING, never the step.
	RunDetached(ctx context.Context, log func(stream, format string, args ...any), key, name string, args ...string) error

	// State reports where the step key is without touching it.
	State(ctx context.Context, key string) (StepState, error)

	// Forget clears an interrupted step's record so it can be run again. Refused
	// for a step that is running or has a recorded result: those are facts, and
	// forgetting them is how a second copy starts.
	Forget(ctx context.Context, key string) error
}

// StepState is where a detached step is.
type StepState int

const (
	// StepNotStarted: nothing was ever started under this key.
	StepNotStarted StepState = iota
	// StepRunning: its unit is alive.
	StepRunning
	// StepFinished: it exited and recorded its status.
	StepFinished
	// StepInterrupted: it was started, its unit is gone, and it recorded no
	// status — it was killed. Distinct from NotStarted because what it was doing
	// may be half-done, and the caller has to repair before running anything.
	StepInterrupted
)

func (s StepState) String() string {
	switch s {
	case StepNotStarted:
		return "not started"
	case StepRunning:
		return "running"
	case StepFinished:
		return "finished"
	case StepInterrupted:
		return "interrupted"
	}
	return "unknown"
}

// ErrStepInterrupted reports a step that was started and killed before it could
// record a status.
var ErrStepInterrupted = errors.New("the step was killed before it recorded an exit status")

// ErrStepStateUnknown reports a step whose unit systemd could not be asked about
// for longer than the runner tolerates. It is NOT a result: the step may still
// be running, so a caller must not audit, repair or start anything on top of it.
var ErrStepStateUnknown = errors.New("systemd could not be asked whether the step is still running")

// systemdRunner runs each step as a transient systemd service.
//
// The unit is a sibling of the agent under PID 1, not a child, so stopping,
// restarting or killing the agent's own unit does not reach it. Its result is a
// FILE, written by the step's own shell after the command exits and renamed into
// place, because a file survives everything the agent does not: the agent
// restarting, the unit being garbage-collected, and the host rebooting.
type systemdRunner struct {
	// runner issues the systemd-run and systemctl calls themselves. Those are
	// short, and running them as the agent's children is fine.
	runner Runner
	dir    string
	poll   time.Duration

	// interruptedAfter is how many consecutive polls must see the unit gone with
	// no result before the step is called interrupted. More than one, because
	// the result is renamed into place a moment before the unit exits, and a
	// unit that is still activating reads as not active.
	interruptedAfter int

	// unreachableFor is how long systemd may fail to answer `systemctl show`
	// before the step's state is reported unknown. A failed query is NOT an
	// answer: during a daemon-reexec, which an upgrade of systemd itself
	// triggers, a share of calls fail while every unit carries on. Reading one
	// failure as "the step ended" started an audit and a repair alongside a
	// still-running apt.
	unreachableFor time.Duration
}

// NewSystemdRunner keeps step records under dir.
//
// dir must be on persistent storage — the agent's state directory — or a reboot
// during a step would erase the only evidence of how it ended.
func NewSystemdRunner(r Runner, dir string) (DetachedRunner, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("detached runner: create %s: %w", dir, err)
	}
	pruneStepRecords(dir, stepRecordMaxAge, time.Now())
	return &systemdRunner{
		runner: r, dir: dir, poll: time.Second, interruptedAfter: 3,
		unreachableFor: 2 * time.Minute,
	}, nil
}

// stepRecordMaxAge bounds how long step records are kept. Long enough to
// diagnose a failed upgrade after a weekend; short enough that the directory
// does not grow for the life of the host.
const stepRecordMaxAge = 30 * 24 * time.Hour

func pruneStepRecords(dir string, maxAge time.Duration, now time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// stepKey makes a key safe as both a file name and a systemd unit name.
//
// Unit names allow [A-Za-z0-9:_.\-]; anything else becomes '_'. Task ids are
// UUIDs, so in practice nothing is replaced — this is so a future caller cannot
// build a key that escapes the directory or names somebody else's unit.
func stepKey(key string) string {
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func (s *systemdRunner) unit(key string) string { return "apiary-step-" + stepKey(key) + ".service" }
func (s *systemdRunner) path(key, ext string) string {
	return filepath.Join(s.dir, stepKey(key)+"."+ext)
}

func (s *systemdRunner) State(ctx context.Context, key string) (StepState, error) {
	if _, ok, err := s.result(key); err != nil {
		return 0, err
	} else if ok {
		return StepFinished, nil
	}
	if _, err := os.Stat(s.path(key, "started")); errors.Is(err, os.ErrNotExist) {
		return StepNotStarted, nil
	} else if err != nil {
		return 0, err
	}
	alive, err := s.aliveTolerant(ctx, key)
	if err != nil {
		return 0, err
	}
	if alive {
		return StepRunning, nil
	}
	// Started, not alive, no result — but the result may be landing right now.
	// Look once more after a poll before calling it killed.
	if !sleepCtx(ctx, s.poll) {
		return 0, ctx.Err()
	}
	if _, ok, err := s.result(key); err != nil {
		return 0, err
	} else if ok {
		return StepFinished, nil
	}
	if alive, err := s.aliveTolerant(ctx, key); err != nil {
		return 0, err
	} else if alive {
		return StepRunning, nil
	}
	return StepInterrupted, nil
}

func (s *systemdRunner) Forget(ctx context.Context, key string) error {
	st, err := s.State(ctx, key)
	if err != nil {
		return err
	}
	if st == StepRunning || st == StepFinished {
		return fmt.Errorf("detached runner: refusing to forget step %q, which is %s", key, st)
	}
	for _, ext := range []string{"started", "log"} {
		if err := os.Remove(s.path(key, ext)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *systemdRunner) RunDetached(ctx context.Context, log func(stream, format string, args ...any), key, name string, args ...string) error {
	st, err := s.State(ctx, key)
	if err != nil {
		return err
	}
	switch st {
	case StepFinished:
		log("agent", "step %s already finished; using its recorded result", key)
		return s.finish(key, name, args)
	case StepInterrupted:
		return fmt.Errorf("step %s (%s): %w", key, name, ErrStepInterrupted)
	case StepRunning:
		log("agent", "step %s is still running from before; re-attaching", key)
	case StepNotStarted:
		if err := s.start(ctx, key, name, args); err != nil {
			return err
		}
	}
	return s.wait(ctx, log, key, name, args)
}

// start launches the step.
//
// The marker is written BEFORE the unit is started. The other order leaves a
// window where the step runs with no record that it was ever started, and a
// restarted agent would read it as never started and launch a second one.
func (s *systemdRunner) start(ctx context.Context, key, name string, args []string) error {
	if err := os.WriteFile(s.path(key, "started"), []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
		return fmt.Errorf("detached runner: record start of %s: %w", key, err)
	}
	rc, tmp, logPath := s.path(key, "rc"), s.path(key, "rc.tmp"), s.path(key, "log")

	// The result is written by the step's own shell, then renamed into place, so
	// a reader never sees a half-written status. "$@" carries the command
	// verbatim: nothing in name or args is ever interpreted by this shell.
	script := `"$@"; rc=$?; printf '%s\n' "$rc" > "$APIARY_STEP_RC_TMP" && mv -f "$APIARY_STEP_RC_TMP" "$APIARY_STEP_RC"; exit "$rc"`

	unit := s.unit(key)
	// A unit of the same name left loaded in a failed state would make
	// systemd-run refuse. --collect below normally prevents that; this covers
	// an older one. Failure here just means there was nothing to reset.
	_ = s.runner.Run(ctx, discard, "systemctl", "reset-failed", unit)

	cmd := []string{
		"--unit=" + unit,
		"--description=APIary agent step " + key,
		"--collect",
		"--property=Type=exec",
		"--property=StandardOutput=append:" + logPath,
		"--property=StandardError=append:" + logPath,
		// apt must never stop to ask a question: there is nobody at this
		// terminal, and a prompt would hang until the deadline.
		"--setenv=DEBIAN_FRONTEND=noninteractive",
		"--setenv=APT_LISTCHANGES_FRONTEND=none",
		"--setenv=APIARY_STEP_RC=" + rc,
		"--setenv=APIARY_STEP_RC_TMP=" + tmp,
		"--", "/bin/sh", "-c", script, "apiary-step", name,
	}
	cmd = append(cmd, args...)
	out := newOutputTail(5)
	if err := s.runner.Run(ctx, out.log, "systemd-run", cmd...); err != nil {
		// Not started, so not running: clear the marker so a retry does not
		// read this as a step that was killed.
		_ = os.Remove(s.path(key, "started"))
		return fmt.Errorf("detached runner: start %s: %w%s", key, err, out.suffix())
	}
	return nil
}

// wait follows the step's log until it records a result.
func (s *systemdRunner) wait(ctx context.Context, log func(stream, format string, args ...any), key, name string, args []string) error {
	f := &logFollower{path: s.path(key, "log"), log: log}
	gone := 0
	var failingSince time.Time
	for {
		f.drain()
		if _, ok, err := s.result(key); err != nil {
			return err
		} else if ok {
			f.drain()
			f.flush()
			return s.finish(key, name, args)
		}
		alive, err := s.alive(ctx, key)
		switch {
		case err != nil:
			// Could not ask. Not alive and not gone: neither counter moves, and
			// the loop keeps following the log and looking for the result.
			if failingSince.IsZero() {
				failingSince = time.Now()
				log("agent", "systemd did not answer about step %s (%v); still waiting", key, err)
			} else if time.Since(failingSince) >= s.unreachableFor {
				f.drain()
				f.flush()
				return fmt.Errorf("step %s (%s): %w for %s: %w", key, name, ErrStepStateUnknown, s.unreachableFor, err)
			}
		case alive:
			gone, failingSince = 0, time.Time{}
		default:
			failingSince = time.Time{}
			gone++
		}
		if gone >= s.interruptedAfter {
			f.drain()
			f.flush()
			return fmt.Errorf("step %s (%s): %w", key, name, ErrStepInterrupted)
		}
		if !sleepCtx(ctx, s.poll) {
			// The agent is stopping. The step is NOT: that is the point.
			return ctx.Err()
		}
	}
}

func (s *systemdRunner) finish(key, name string, args []string) error {
	code, _, err := s.result(key)
	if err != nil {
		return err
	}
	if code == 0 {
		return nil
	}
	return fmt.Errorf("%s %s: exit status %d%s", name, strings.Join(args, " "), code, s.tail(key))
}

// result reads the recorded exit status, if there is one.
func (s *systemdRunner) result(key string) (int, bool, error) {
	b, err := os.ReadFile(s.path(key, "rc"))
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, false, fmt.Errorf("detached runner: step %s recorded an unreadable status %q", key, strings.TrimSpace(string(b)))
	}
	return code, true, nil
}

// aliveTolerant is alive, retried while systemd fails to answer, for up to
// unreachableFor. See unreachableFor for why one failure is not an answer.
func (s *systemdRunner) aliveTolerant(ctx context.Context, key string) (bool, error) {
	var since time.Time
	for {
		alive, err := s.alive(ctx, key)
		if err == nil {
			return alive, nil
		}
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if since.IsZero() {
			since = time.Now()
		} else if time.Since(since) >= s.unreachableFor {
			return false, fmt.Errorf("step %s: %w for %s: %w", key, ErrStepStateUnknown, s.unreachableFor, err)
		}
		if !sleepCtx(ctx, s.poll) {
			return false, ctx.Err()
		}
	}
}

// alive asks systemd whether the step's unit is still doing something.
func (s *systemdRunner) alive(ctx context.Context, key string) (bool, error) {
	var state string
	capture := func(stream, format string, args ...any) {
		if stream == "stdout" {
			state = strings.TrimSpace(fmt.Sprintf(format, args...))
		}
	}
	// `systemctl show` on a unit that does not exist succeeds and reports
	// "inactive", so an error here is systemd itself being unreachable — not an
	// answer, and not to be read as "gone".
	if err := s.runner.Run(ctx, capture, "systemctl", "show", "--property=ActiveState", "--value", s.unit(key)); err != nil {
		return false, fmt.Errorf("detached runner: ask systemd about %s: %w", key, err)
	}
	switch state {
	case "active", "activating", "deactivating", "reloading", "refreshing":
		return true, nil
	}
	return false, nil
}

// tail renders the last lines of the step's log for an error.
func (s *systemdRunner) tail(key string) string {
	b, err := os.ReadFile(s.path(key, "log"))
	if err != nil {
		return ""
	}
	t := newOutputTail(5)
	for l := range strings.SplitSeq(string(b), "\n") {
		t.log("", "%s", l)
	}
	return t.suffix()
}

// logFollower streams a growing file line by line.
type logFollower struct {
	path    string
	log     func(stream, format string, args ...any)
	offset  int64
	partial string
}

func (f *logFollower) drain() {
	fh, err := os.Open(f.path)
	if err != nil {
		return
	}
	defer fh.Close()
	if _, err := fh.Seek(f.offset, io.SeekStart); err != nil {
		return
	}
	b, err := io.ReadAll(fh)
	if err != nil || len(b) == 0 {
		return
	}
	f.offset += int64(len(b))
	lines := strings.Split(f.partial+string(b), "\n")
	f.partial = lines[len(lines)-1]
	for _, l := range lines[:len(lines)-1] {
		// "stdout" because the control plane accepts only stdout, stderr and
		// agent, and the unit writes both of its streams into one file.
		f.log("stdout", "%s", l)
	}
}

func (f *logFollower) flush() {
	if f.partial != "" {
		f.log("stdout", "%s", f.partial)
		f.partial = ""
	}
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
