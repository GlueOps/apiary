package fleetops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cmapRunner answers corosync-cmapctl with a fixed running version and records
// every reload.
type cmapRunner struct {
	running string
	reloads int
}

func (r *cmapRunner) Run(_ context.Context, log func(string, string, ...any), name string, args ...string) error {
	switch name {
	case "corosync-cmapctl":
		log("stdout", "%s", "totem.config_version (u64) = "+r.running)
	case "corosync-cfgtool":
		r.reloads++
	}
	return nil
}

func withCorosyncConf(t *testing.T, version string) {
	t.Helper()
	oc := CorosyncConf
	CorosyncConf = filepath.Join(t.TempDir(), "corosync.conf")
	t.Cleanup(func() { CorosyncConf = oc })
	if version == "" {
		return
	}
	conf := "totem {\n  cluster_name: dc1\n  config_version: " + version + "\n  version: 2\n}\n"
	if err := os.WriteFile(CorosyncConf, []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }
func quiet(string, ...any)               {}

// The live failure: the file says 14, corosync runs 13. Reloaded once the drift
// has outlasted the grace window, and not before — pmxcfs writes the file a
// moment ahead of its own reload, and a check in that window must not race it.
func TestARuntimeBehindItsFileIsReloadedAfterTheGrace(t *testing.T) {
	withCorosyncConf(t, "14")
	r := &cmapRunner{running: "13"}
	c := &clock{t: time.Unix(1_000_000, 0)}
	cr := &CorosyncReload{Runner: r, now: c.now}

	cr.Check(context.Background(), quiet)
	c.advance(driftGrace / 2)
	cr.Check(context.Background(), quiet)
	if r.reloads != 0 {
		t.Fatalf("reloaded %d time(s) inside the grace window", r.reloads)
	}

	c.advance(driftGrace)
	var logged []string
	cr.Check(context.Background(), func(f string, a ...any) { logged = append(logged, f) })
	if r.reloads != 1 {
		t.Fatalf("reloads = %d after the drift persisted, want 1", r.reloads)
	}
	if len(logged) == 0 || !strings.Contains(logged[0], "reloading") {
		t.Fatalf("the reload was not logged: %v", logged)
	}

	// A drift that did not clear is retried, but spaced out.
	c.advance(reloadInterval / 2)
	cr.Check(context.Background(), quiet)
	if r.reloads != 1 {
		t.Fatalf("reloads = %d inside the retry interval, want 1", r.reloads)
	}
	c.advance(reloadInterval)
	cr.Check(context.Background(), quiet)
	if r.reloads != 2 {
		t.Fatalf("reloads = %d once the retry interval passed, want 2", r.reloads)
	}
}

// Equal is converged, and a runtime AHEAD of the file is a file pmxcfs has not
// rewritten yet — reloading would load the older configuration.
func TestOnlyARuntimeBehindTheFileIsReloaded(t *testing.T) {
	for _, running := range []string{"14", "15"} {
		withCorosyncConf(t, "14")
		r := &cmapRunner{running: running}
		c := &clock{t: time.Unix(1_000_000, 0)}
		cr := &CorosyncReload{Runner: r, now: c.now}
		for i := 0; i < 5; i++ {
			cr.Check(context.Background(), quiet)
			c.advance(driftGrace + reloadInterval)
		}
		if r.reloads != 0 {
			t.Fatalf("running %s against file 14: reloaded %d time(s)", running, r.reloads)
		}
	}
}

// A drift that clears on its own restarts the grace window, so a later drift is
// judged from when IT began rather than from the first one.
func TestADriftThatClearsResetsTheGrace(t *testing.T) {
	withCorosyncConf(t, "14")
	r := &cmapRunner{running: "13"}
	c := &clock{t: time.Unix(1_000_000, 0)}
	cr := &CorosyncReload{Runner: r, now: c.now}

	cr.Check(context.Background(), quiet)
	c.advance(driftGrace - time.Second)
	r.running = "14"
	cr.Check(context.Background(), quiet)
	r.running = "13"
	c.advance(2 * time.Second)
	cr.Check(context.Background(), quiet)
	if r.reloads != 0 {
		t.Fatalf("reloaded on a drift %s old", 2*time.Second)
	}
}

// A standalone host has no corosync.conf and nothing to repair; corosync is not
// even asked.
func TestAStandaloneHostIsLeftAlone(t *testing.T) {
	withCorosyncConf(t, "")
	r := &recordingRunner{}
	cr := &CorosyncReload{Runner: r}
	cr.Check(context.Background(), quiet)
	if len(r.cmds) != 0 {
		t.Fatalf("ran %v on a standalone host", r.cmds)
	}
}

// A nil CorosyncReload is what a host with no runner has; Check must be safe.
func TestANilCorosyncReloadIsANoop(t *testing.T) {
	var cr *CorosyncReload
	cr.Check(context.Background(), quiet)
}

// hangingJoinRunner blocks until its context ends, as pvecm add did while it
// waited for quorum that could not form.
type hangingJoinRunner struct{}

func (hangingJoinRunner) Run(ctx context.Context, _ func(string, string, ...any), _ string, _ ...string) error {
	<-ctx.Done()
	return ctx.Err()
}

// A join that cannot complete fails on its own deadline, with a message that
// says so, instead of running until the watchdog restarts the agent.
func TestAHungJoinFailsOnItsOwnDeadline(t *testing.T) {
	withCorosyncConf(t, "")
	ot := JoinTimeout
	JoinTimeout = 50 * time.Millisecond
	t.Cleanup(func() { JoinTimeout = ot })

	cl := NewLocalCluster(hangingJoinRunner{})
	start := time.Now()
	_, err := cl.JoinCluster(context.Background(), &proxmoxJoinOptions{
		Hostname: "10.0.0.1", Fingerprint: "AA:BB"})
	if err == nil {
		t.Fatal("a hung join was reported as accepted")
	}
	if !strings.Contains(err.Error(), "did not finish within") {
		t.Fatalf("err = %v, want it to name the deadline", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("the join ran %s past a %s deadline", time.Since(start), JoinTimeout)
	}
}
