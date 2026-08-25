package store

import (
	"context"
	"path/filepath"
	"testing"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// The control plane re-delivers the SAME task to a bee that already holds one —
// that is how a bee which crashed mid-upgrade picks up where it left off.
// Accepting it again must not roll a running task back to accepted, or the bee
// would start from the top having already done half the work: on a rotation
// that means re-running an upgrade and re-taking a reboot.
func TestReAcceptingATaskDoesNotRestartIt(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	if err := s.AcceptTask(ctx, Task{ID: "t1", Kind: "fleet.host.upgrade", Spec: "{}", Attempt: 1}); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if err := s.MarkRunning(ctx, "t1", "upgrading"); err != nil {
		t.Fatalf("mark running: %v", err)
	}

	// Re-delivered on the next heartbeat, with a bumped attempt.
	if err := s.AcceptTask(ctx, Task{ID: "t1", Kind: "fleet.host.upgrade", Spec: "{}", Attempt: 2}); err != nil {
		t.Fatalf("re-accept: %v", err)
	}

	got, err := s.CurrentTask(ctx)
	if err != nil || got == nil {
		t.Fatalf("current: %v (task=%v)", err, got)
	}
	if got.Status != TaskRunning {
		t.Fatalf("status = %q, want %q — re-delivery reset a task that was already under way",
			got.Status, TaskRunning)
	}
	if got.Phase != "upgrading" {
		t.Fatalf("phase = %q, want it preserved; the bee lost track of where it had got to", got.Phase)
	}
	if got.Attempt != 2 {
		t.Fatalf("attempt = %d, want 2 — a task being repeatedly reclaimed must be visible", got.Attempt)
	}
}

// A finished task is not forgotten until its outcome has been acknowledged. That
// gap is exactly the state a bee is in when it completes work and dies before
// the report lands.
func TestAFinishedTaskSurvivesUntilItHasBeenReported(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	_ = s.AcceptTask(ctx, Task{ID: "t1", Kind: "k", Spec: "{}"})
	_ = s.MarkRunning(ctx, "t1", "working")
	if err := s.Finish(ctx, "t1", TaskDone, ""); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, _ := s.CurrentTask(ctx)
	if got == nil || got.Reported {
		t.Fatal("a locally-finished task must persist as unreported, or its outcome is lost on restart")
	}

	if err := s.MarkReported(ctx, "t1"); err != nil {
		t.Fatalf("mark reported: %v", err)
	}
	if err := s.ClearTask(ctx, "t1"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got, _ := s.CurrentTask(ctx); got != nil {
		t.Fatal("an acknowledged task with no pending logs should be cleared")
	}
}

// Clearing a task while its output is still unshipped would discard the only
// evidence of why a rotation failed.
func TestATaskIsNotClearedWhileItsLogsAreUnshipped(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	_ = s.AcceptTask(ctx, Task{ID: "t1", Kind: "k", Spec: "{}"})
	if err := s.AppendLog(ctx, "t1", "stderr", "E: dpkg was interrupted"); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = s.Finish(ctx, "t1", TaskFailed, "upgrade failed")
	_ = s.MarkReported(ctx, "t1")

	if err := s.ClearTask(ctx, "t1"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got, _ := s.CurrentTask(ctx); got == nil {
		t.Fatal("the task was cleared with unshipped logs; the reason it failed is now unrecoverable")
	}

	lines, err := s.UnshippedLogs(ctx, 10)
	if err != nil || len(lines) != 1 {
		t.Fatalf("unshipped: %d lines, err=%v", len(lines), err)
	}
	if err := s.MarkShipped(ctx, lines[0].Seq); err != nil {
		t.Fatalf("mark shipped: %v", err)
	}
	if err := s.ClearTask(ctx, "t1"); err != nil {
		t.Fatalf("clear after shipping: %v", err)
	}
	if got, _ := s.CurrentTask(ctx); got != nil {
		t.Fatal("once every line has shipped the task should be cleared")
	}
}

// The outbox sequence is what the control plane deduplicates on, so it must be
// assigned locally, once, in the order lines were produced — never reconstructed
// later from arrival order that a retry would scramble.
func TestLogSequencesAreMonotonicAndLocal(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	_ = s.AcceptTask(ctx, Task{ID: "t1", Kind: "k", Spec: "{}"})

	for i := range 5 {
		if err := s.AppendLog(ctx, "t1", "stdout", string(rune('a'+i))); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	lines, err := s.UnshippedLogs(ctx, 10)
	if err != nil {
		t.Fatalf("unshipped: %v", err)
	}
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	for i := 1; i < len(lines); i++ {
		if lines[i].Seq <= lines[i-1].Seq {
			t.Fatalf("sequence is not monotonic: %d then %d", lines[i-1].Seq, lines[i].Seq)
		}
	}

	// Shipping a prefix must not re-offer it.
	_ = s.MarkShipped(ctx, lines[2].Seq)
	rest, _ := s.UnshippedLogs(ctx, 10)
	if len(rest) != 2 {
		t.Fatalf("after shipping 3 of 5, %d remain unshipped, want 2", len(rest))
	}
}

// A dist-upgrade records its phase, reboots the hypervisor, and the agent comes
// back into the same task. If starting a task reset the phase, the executor
// could not tell "beginning" from "already upgraded and rebooted" — and would
// upgrade and reboot again, forever, across every host in a rotation.
func TestStartingATaskPreservesAPhaseItAlreadyReached(t *testing.T) {
	ctx := context.Background()
	s := open(t)

	_ = s.AcceptTask(ctx, Task{ID: "t1", Kind: "fleet.host.dist-upgrade", Spec: "{}"})
	_ = s.MarkRunning(ctx, "t1", "starting")
	if err := s.SetPhase(ctx, "t1", "rebooting"); err != nil {
		t.Fatalf("set phase: %v", err)
	}

	// The reboot happens here. The agent restarts and re-enters the task.
	if err := s.MarkRunning(ctx, "t1", "starting"); err != nil {
		t.Fatalf("re-mark running: %v", err)
	}

	got, _ := s.CurrentTask(ctx)
	if got == nil || got.Phase != "rebooting" {
		t.Fatalf("phase = %q, want %q — the agent has forgotten it already rebooted and will "+
			"upgrade and reboot again", got.Phase, "rebooting")
	}
}
