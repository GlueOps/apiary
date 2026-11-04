package fleetops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// withReboot swaps the reboot command and the wait for one test.
func withReboot(t *testing.T, wait time.Duration, cmd func(context.Context) ([]byte, error)) *int {
	t.Helper()
	calls := 0
	oldCmd, oldWait := rebootCommand, rebootWait
	rebootCommand = func(ctx context.Context) ([]byte, error) { calls++; return cmd(ctx) }
	rebootWait = wait
	t.Cleanup(func() { rebootCommand, rebootWait = oldCmd, oldWait })
	return &calls
}

// `systemctl reboot` returns once the job is queued. RebootHost used to
// return nil right then, and every real reboot was reported done before the
// host went down. The shutdown stopping the agent is the normal way out.
func TestRebootHostWaitsForTheShutdownToStopTheAgent(t *testing.T) {
	calls := withReboot(t, time.Hour, func(context.Context) ([]byte, error) { return nil, nil })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RebootHost(ctx) }()

	select {
	case err := <-done:
		t.Fatalf("returned %v before the shutdown stopped the agent", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel() // SIGTERM from the shutdown
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled so the task resumes after the boot", err)
		}
	case <-time.After(time.Second):
		t.Fatal("did not return when the agent was stopped")
	}
	if *calls != 1 {
		t.Fatalf("systemctl reboot ran %d times", *calls)
	}
}

// Still up rebootWait later: nil, and the caller's boot-id check says the
// reboot did not happen.
func TestRebootHostGivesUpWaitingAfterRebootWait(t *testing.T) {
	withReboot(t, 20*time.Millisecond, func(context.Context) ([]byte, error) { return nil, nil })
	start := time.Now()
	if err := RebootHost(context.Background()); err != nil {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) < 20*time.Millisecond {
		t.Fatal("returned before rebootWait")
	}
}

func TestRebootHostReportsARefusedReboot(t *testing.T) {
	withReboot(t, time.Hour, func(context.Context) ([]byte, error) {
		return []byte("Failed to reboot: Access denied\n"), errors.New("exit status 1")
	})
	err := RebootHost(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Access denied") {
		t.Fatalf("err = %v, want the refusal with systemctl's output", err)
	}
}
