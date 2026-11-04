package fleetops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSystemd stands in for systemd-run and systemctl, and runs the step's
// command line FOR REAL with /bin/sh — so the wrapper script, the result file
// and the rename are exercised exactly as on a host. Only the "a sibling of the
// agent under PID 1" part is simulated: the step runs in a goroutine the test
// controls rather than a unit.
type fakeSystemd struct {
	mu      sync.Mutex
	running map[string]bool
	starts  int
	// noStart makes systemd-run "succeed" without running anything, which is
	// what a step killed the instant it began looks like afterwards.
	noStart bool
}

func newFakeSystemd() *fakeSystemd { return &fakeSystemd{running: map[string]bool{}} }

func (f *fakeSystemd) Run(ctx context.Context, log func(string, string, ...any), name string, args ...string) error {
	switch name {
	case "systemctl":
		if len(args) > 0 && args[0] == "show" {
			unit := args[len(args)-1]
			f.mu.Lock()
			alive := f.running[unit]
			f.mu.Unlock()
			state := "inactive"
			if alive {
				state = "active"
			}
			log("stdout", "%s", state)
		}
		return nil
	case "systemd-run":
		var unit string
		env := os.Environ()
		var cmd []string
		for i, a := range args {
			switch {
			case strings.HasPrefix(a, "--unit="):
				unit = strings.TrimPrefix(a, "--unit=")
			case strings.HasPrefix(a, "--setenv="):
				env = append(env, strings.TrimPrefix(a, "--setenv="))
			case strings.HasPrefix(a, "--property=StandardOutput=append:"):
				env = append(env, "FAKE_LOG="+strings.TrimPrefix(a, "--property=StandardOutput=append:"))
			case a == "--":
				cmd = args[i+1:]
			}
		}
		f.mu.Lock()
		f.starts++
		if f.noStart {
			f.mu.Unlock()
			return nil
		}
		f.running[unit] = true
		f.mu.Unlock()

		var logPath string
		for _, e := range env {
			if v, ok := strings.CutPrefix(e, "FAKE_LOG="); ok {
				logPath = v
			}
		}
		out, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		c := exec.Command(cmd[0], cmd[1:]...) // NOT tied to ctx: it must outlive the waiter
		c.Env, c.Stdout, c.Stderr = env, out, out
		if err := c.Start(); err != nil {
			return err
		}
		go func() {
			_ = c.Wait()
			_ = out.Close()
			f.mu.Lock()
			f.running[unit] = false
			f.mu.Unlock()
		}()
		return nil
	}
	return fmt.Errorf("unexpected command %s", name)
}

func (f *fakeSystemd) startCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.starts
}

func testSystemdRunner(t *testing.T, f *fakeSystemd) *systemdRunner {
	t.Helper()
	return &systemdRunner{runner: f, dir: t.TempDir(), poll: 10 * time.Millisecond, interruptedAfter: 3}
}

type lineLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *lineLog) log(stream, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, stream+": "+fmt.Sprintf(format, args...))
}

func (l *lineLog) has(s string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, x := range l.lines {
		if strings.Contains(x, s) {
			return true
		}
	}
	return false
}

func TestDetachedStepRecordsItsExitStatusAndStreamsItsOutput(t *testing.T) {
	r := testSystemdRunner(t, newFakeSystemd())
	var l lineLog

	err := r.RunDetached(context.Background(), l.log, "task-1-step", "/bin/sh", "-c", "echo hello; echo oops >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("err = %v, want the step's own exit status 3", err)
	}
	// stdout only: the control plane accepts stdout, stderr and agent, and the
	// unit writes both streams into one file.
	if !l.has("stdout: hello") || !l.has("stdout: oops") {
		t.Fatalf("step output was not streamed as stdout: %v", l.lines)
	}
}

// The agent stopping must stop the WAITING, never the step. A restarted agent
// re-attaches to the same step and sees it through, without starting another.
func TestDetachedStepSurvivesTheAgentAndIsReattached(t *testing.T) {
	f := newFakeSystemd()
	r := testSystemdRunner(t, f)
	var l lineLog

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		errc <- r.RunDetached(ctx, l.log, "task-2-upgrade", "/bin/sh", "-c", "sleep 0.4; echo finished")
	}()
	time.Sleep(100 * time.Millisecond)
	cancel() // the agent is stopping
	if err := <-errc; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation, so the task is left to resume", err)
	}

	st, err := r.State(context.Background(), "task-2-upgrade")
	if err != nil || st != StepRunning {
		t.Fatalf("state = %v (%v) after the agent stopped; the step must still be running", st, err)
	}

	// The restarted agent.
	if err := r.RunDetached(context.Background(), l.log, "task-2-upgrade", "/bin/sh", "-c", "sleep 0.4; echo finished"); err != nil {
		t.Fatalf("re-attach failed: %v", err)
	}
	if n := f.startCount(); n != 1 {
		t.Fatalf("the step was started %d times; a second dpkg on the same database is the "+
			"failure this exists to prevent", n)
	}
	if !l.has("finished") {
		t.Fatalf("the re-attached waiter never saw the step's output: %v", l.lines)
	}
}

func TestDetachedFinishedStepIsNotRunAgain(t *testing.T) {
	f := newFakeSystemd()
	r := testSystemdRunner(t, f)

	for i := 0; i < 2; i++ {
		if err := r.RunDetached(context.Background(), discard, "task-3-update", "/bin/sh", "-c", "true"); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	if n := f.startCount(); n != 1 {
		t.Fatalf("a finished step was started %d times; its recorded result must be reused", n)
	}
}

// A step that was started and left no result was killed. That is reported as
// such — never as success, and never quietly restarted.
func TestDetachedKilledStepIsReportedInterrupted(t *testing.T) {
	f := newFakeSystemd()
	f.noStart = true
	r := testSystemdRunner(t, f)

	err := r.RunDetached(context.Background(), discard, "task-4-upgrade", "/bin/sh", "-c", "true")
	if !errors.Is(err, ErrStepInterrupted) {
		t.Fatalf("err = %v, want ErrStepInterrupted", err)
	}
	if st, _ := r.State(context.Background(), "task-4-upgrade"); st != StepInterrupted {
		t.Fatalf("state = %v, want interrupted", st)
	}
	if err := r.RunDetached(context.Background(), discard, "task-4-upgrade", "/bin/sh", "-c", "true"); !errors.Is(err, ErrStepInterrupted) {
		t.Fatalf("a killed step was silently restarted: %v", err)
	}

	if err := r.Forget(context.Background(), "task-4-upgrade"); err != nil {
		t.Fatalf("forget: %v", err)
	}
	if st, _ := r.State(context.Background(), "task-4-upgrade"); st != StepNotStarted {
		t.Fatalf("state after forget = %v, want not started", st)
	}
}

func TestDetachedForgetRefusesARecordedStep(t *testing.T) {
	r := testSystemdRunner(t, newFakeSystemd())
	if err := r.RunDetached(context.Background(), discard, "task-5", "/bin/sh", "-c", "true"); err != nil {
		t.Fatal(err)
	}
	if err := r.Forget(context.Background(), "task-5"); err == nil {
		t.Fatal("forgot a finished step; forgetting a fact is how a second copy starts")
	}
}

// Arguments are passed through "$@" and never interpreted by the wrapper shell.
func TestDetachedArgumentsAreNotShellInterpreted(t *testing.T) {
	r := testSystemdRunner(t, newFakeSystemd())
	canary := filepath.Join(t.TempDir(), "pwned")
	var l lineLog
	if err := r.RunDetached(context.Background(), l.log, "task-6", "echo", "$(touch "+canary+")"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(canary); err == nil {
		t.Fatal("an argument was executed by the wrapper shell")
	}
	if !l.has("$(touch ") {
		t.Fatalf("the argument did not arrive verbatim: %v", l.lines)
	}
}

func TestStepKeyCannotEscapeItsDirectory(t *testing.T) {
	k := stepKey("../../etc/cron.d/x y")
	if strings.ContainsAny(k, "/. ") {
		t.Fatalf("stepKey kept a path or unit-name separator: %q", k)
	}
}
