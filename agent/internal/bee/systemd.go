package bee

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Systemd readiness and watchdog notification, spoken directly over the
// NOTIFY_SOCKET datagram protocol.
//
// Implemented here rather than pulled in as a dependency: it is thirty lines of
// writing short strings to a unix socket, and the agent's whole point is being a
// single static binary that can be dropped onto a hypervisor.
//
// Every function is a no-op when NOTIFY_SOCKET is unset, so running the binary
// by hand behaves exactly as it did before.

type notifier struct {
	conn *net.UnixConn
	// alive is stamped by each loop. The watchdog reports healthy only while
	// BOTH have run recently — see Watchdog.
	reportAlive  atomic.Int64
	executeAlive atomic.Int64
}

func newNotifier() *notifier {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return &notifier{}
	}
	// A leading '@' means an abstract socket, which Go expects as a NUL.
	if strings.HasPrefix(addr, "@") {
		addr = "\x00" + addr[1:]
	}
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: addr, Net: "unixgram"})
	if err != nil {
		return &notifier{}
	}
	return &notifier{conn: conn}
}

func (n *notifier) send(state string) {
	if n == nil || n.conn == nil {
		return
	}
	_, _ = n.conn.Write([]byte(state))
}

// Ready tells systemd the agent has started. Until this arrives a Type=notify
// unit is considered still starting, so anything ordered after it waits.
func (n *notifier) Ready() { n.send("READY=1") }

// Status sets the one-line description shown by `systemctl status`, which is the
// first thing anyone looks at on a hypervisor that is behaving oddly.
func (n *notifier) Status(format string, args ...any) {
	n.send("STATUS=" + fmt.Sprintf(format, args...))
}

func (n *notifier) Stopping() { n.send("STOPPING=1") }

// Watchdog feeds systemd for as long as BOTH loops are running.
//
// Both, deliberately. Restart=always catches a process that has exited; it
// cannot catch one that is still running but wedged — and the two loops wedge
// independently. A hung report loop with a healthy execute loop is the worse
// case: the host is doing work nobody can see, and the control plane has already
// decided it is offline.
func (n *notifier) Watchdog(ctx context.Context) {
	if n == nil || n.conn == nil {
		return
	}
	usec, err := strconv.ParseInt(os.Getenv("WATCHDOG_USEC"), 10, 64)
	if err != nil || usec <= 0 {
		return
	}
	// Half the interval, as systemd's own documentation recommends: it leaves
	// room for one missed tick without a restart.
	every := time.Duration(usec) * time.Microsecond / 2
	stale := time.Duration(usec) * time.Microsecond * 2

	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := time.Now()
			report := time.Unix(0, n.reportAlive.Load())
			execute := time.Unix(0, n.executeAlive.Load())
			if now.Sub(report) > stale || now.Sub(execute) > stale {
				// Deliberately silent: withholding the ping IS the report. Saying
				// "unhealthy" and continuing to ping would leave the agent wedged
				// and supervised, which is the state this exists to end.
				continue
			}
			n.send("WATCHDOG=1")
		}
	}
}
