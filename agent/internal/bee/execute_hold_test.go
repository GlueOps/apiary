package bee

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glueops/apiary-agent/internal/store"
)

// TestALongTaskKeepsTheWatchdogFed.
//
// The execute loop stamped its liveness only BETWEEN tasks. A task running
// longer than the watchdog's stale bound — two minutes in production, and a
// dist-upgrade usually is — read as a wedged loop, the watchdog went silent, and
// systemd SIGABRTed the agent mid-task. Before package steps were detached that
// kill took dpkg with it.
//
// Driven through execute() itself, so the test fails if the hold is removed
// from the call site and not only if the helper breaks.
func TestALongTaskKeepsTheWatchdogFed(t *testing.T) {
	c, path := listen(t)
	t.Setenv("NOTIFY_SOCKET", path)
	t.Setenv("WATCHDOG_USEC", "200000") // stale after 400ms, pings every 100ms

	prev := holdInterval
	holdInterval = 20 * time.Millisecond
	t.Cleanup(func() { holdInterval = prev })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	task := store.Task{ID: "t-long", Kind: "test.long"}
	if err := st.AcceptTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	release := make(chan struct{})
	b := New(Options{Store: st}, map[string]Executor{
		"test.long": func(ctx context.Context, _ store.Task, _ *Env) error {
			<-release // a task far longer than the stale bound
			return nil
		},
	})

	// Report loop healthy throughout, so only the execute loop is in question.
	go func() {
		for ctx.Err() == nil {
			b.notify.reportAlive.Store(time.Now().UnixNano())
			time.Sleep(10 * time.Millisecond)
		}
	}()
	b.notify.executeAlive.Store(time.Now().UnixNano())
	go b.notify.Watchdog(ctx)

	done := make(chan struct{})
	go func() { b.execute(ctx, task); close(done) }()

	// Well past the stale bound, the watchdog must still be pinging.
	time.Sleep(600 * time.Millisecond)
	// Datagrams queue: drain everything sent so far, including pings from before
	// the stale bound, so the next one read is genuinely current.
	for recv(t, c, 20*time.Millisecond) != "" {
	}
	if got := recv(t, c, 300*time.Millisecond); got != "WATCHDOG=1" {
		t.Fatalf("got %q mid-task; the watchdog has gone silent and systemd would kill the "+
			"agent part-way through a running task", got)
	}

	close(release)
	<-done
}

// TestTheExecuteHoldEnds, so a finished task does not leave the loop looking
// alive forever — which would hide a loop that wedges afterwards.
func TestTheExecuteHoldEnds(t *testing.T) {
	prev := holdInterval
	holdInterval = 5 * time.Millisecond
	t.Cleanup(func() { holdInterval = prev })

	n := &notifier{}
	release := n.holdExecuteAlive(context.Background(), time.Hour)
	time.Sleep(20 * time.Millisecond)
	release()
	time.Sleep(20 * time.Millisecond)

	stale := time.Now().Add(-time.Hour).UnixNano()
	n.executeAlive.Store(stale)
	time.Sleep(30 * time.Millisecond)
	if n.executeAlive.Load() != stale {
		t.Fatal("the hold kept stamping after it was released; a loop that wedged after its " +
			"task finished would never be restarted")
	}
}
