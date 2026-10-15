package bee

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// listen stands in for systemd, so the protocol is tested against something that
// actually reads the socket rather than against our own assumptions about it.
func listen(t *testing.T) (*net.UnixConn, string) {
	t.Helper()
	// A short path on purpose: a unix socket address is capped near 104 bytes,
	// and t.TempDir() alone exceeds it on macOS with "bind: invalid argument".
	dir, err := os.MkdirTemp("", "bee")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "n.sock")
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, path
}

func recv(t *testing.T, c *net.UnixConn, within time.Duration) string {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(within))
	buf := make([]byte, 256)
	n, err := c.Read(buf)
	if err != nil {
		return ""
	}
	return string(buf[:n])
}

// Running the binary by hand must behave exactly as it did before systemd
// support existed. An agent that only worked under a supervisor would be
// undebuggable on the one host where you need to debug it.
func TestNotifierIsInertWithoutASocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	n := newNotifier()
	n.Ready()
	n.Status("anything")
	n.Stopping()
	n.Watchdog(context.Background()) // must return immediately, not block
}

func TestReadyIsSentWhenSupervised(t *testing.T) {
	c, path := listen(t)
	t.Setenv("NOTIFY_SOCKET", path)

	n := newNotifier()
	n.Ready()
	if got := recv(t, c, time.Second); got != "READY=1" {
		t.Fatalf("got %q, want READY=1 — a Type=notify unit stays in 'activating' until this "+
			"arrives, so anything ordered after the agent waits forever", got)
	}
}

// The watchdog reports healthy only while BOTH loops are running. Restart=always
// catches a process that exited; only this catches one that is still running and
// wedged — and the worse case is a hung report loop with a healthy execute loop,
// where the host is doing work nobody can see.
func TestWatchdogStopsPingingWhenALoopStalls(t *testing.T) {
	c, path := listen(t)
	t.Setenv("NOTIFY_SOCKET", path)
	t.Setenv("WATCHDOG_USEC", "200000") // 200ms, so it pings every 100ms

	n := newNotifier()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fresh := time.Now().UnixNano()
	n.reportAlive.Store(fresh)
	n.executeAlive.Store(fresh)
	go n.Watchdog(ctx)

	if got := recv(t, c, time.Second); got != "WATCHDOG=1" {
		t.Fatalf("got %q, want WATCHDOG=1 while both loops are alive", got)
	}

	// The execute loop wedges: its liveness stamp stops moving. Keep the report
	// loop healthy so the test proves BOTH are required, not just either.
	stale := time.Now().Add(-time.Hour).UnixNano()
	n.executeAlive.Store(stale)
	go func() {
		for i := 0; i < 40; i++ {
			n.reportAlive.Store(time.Now().UnixNano())
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Drain anything already in flight, then require silence.
	_ = recv(t, c, 150*time.Millisecond)
	if got := recv(t, c, 400*time.Millisecond); got != "" {
		t.Fatalf("still pinging (%q) with the execute loop wedged; systemd would keep a hung "+
			"agent alive, which is the state the watchdog exists to end", got)
	}
}

func TestWatchdogIsInertWithoutAnInterval(t *testing.T) {
	_, path := listen(t)
	t.Setenv("NOTIFY_SOCKET", path)
	os.Unsetenv("WATCHDOG_USEC")

	n := newNotifier()
	done := make(chan struct{})
	go func() { n.Watchdog(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Watchdog blocked with no WATCHDOG_USEC set; a unit without a watchdog would " +
			"leak this goroutine forever")
	}
}

// A long but bounded operation in the report loop keeps it marked alive, and
// only for as long as it is held.
//
// Live: a cluster join runs in the report loop and can wait four minutes for
// membership to settle. The stamp went stale at two, systemd's watchdog SIGABRTed
// the agent mid-join, and it restarted and tried again — every time.
func TestHoldReportAliveStampsUntilReleasedOrCapped(t *testing.T) {
	n := &notifier{}
	stale := time.Now().Add(-time.Hour).UnixNano()

	n.reportAlive.Store(stale)
	release := n.holdReportAlive(t.Context(), time.Minute)
	deadline := time.Now().Add(2 * time.Second)
	for n.reportAlive.Load() == stale {
		if time.Now().After(deadline) {
			t.Fatal("a held report loop was never stamped alive")
		}
		time.Sleep(10 * time.Millisecond)
	}
	release()
	release() // idempotent

	// Released: nothing stamps it any more, so a wedge after this is caught.
	time.Sleep(50 * time.Millisecond)
	n.reportAlive.Store(stale)
	time.Sleep(50 * time.Millisecond)
	if n.reportAlive.Load() != stale {
		t.Fatal("the report loop was still stamped after the hold was released")
	}

	// Capped: a hold that outlives max stops stamping on its own.
	n2 := &notifier{}
	_ = n2.holdReportAlive(t.Context(), 20*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	n2.reportAlive.Store(stale)
	time.Sleep(50 * time.Millisecond)
	if n2.reportAlive.Load() != stale {
		t.Fatal("a hold past its cap kept the report loop alive; a real wedge would never be restarted")
	}
}
