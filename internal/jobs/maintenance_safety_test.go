package jobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
)

// windowAt writes an enabled one-off window opening at `at`.
func (f *mfleet) windowAt(hive *tenant.Hive, at time.Time) tenant.MaintenanceSchedule {
	f.t.Helper()
	s := tenant.MaintenanceSchedule{
		Name: "later", Kind: tenant.MaintenanceOnce, OnceAt: &at, DurationMinutes: 240,
		Reboot: tenant.RebootAlways, Enabled: true, HiveID: &hive.ID,
	}
	if err := f.db.Create(&s).Error; err != nil {
		f.t.Fatalf("seed schedule: %v", err)
	}
	return s
}

// heartbeatAll brings every machine in the fleet up to f.now.
func (f *mfleet) heartbeatAll() {
	var hvs []tenant.Hypervisor
	f.db.Where("bee_id IS NOT NULL").Find(&hvs)
	for _, h := range hvs {
		f.heartbeat(h)
	}
}

// SKIPPING AN OFFLINE NODE IS NOT A LICENCE TO REBOOT THE NEXT. Three nodes, one
// already down: rebooting a second leaves one of three, and the cluster goes
// read-only. The gate counted every member and let it through.
func TestMaintenanceDoesNotRebootANodeWhileAnotherIsDown(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1", offlineFor(time.Hour))
	b := f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()

	r := f.run()
	if got := statuses(r); got != "hv1=skipped hv2=failed hv3=skipped" {
		t.Fatalf("got %s", got)
	}
	if n := nodeByName(r, "hv2"); !strings.Contains(n.Reason, "quorum") ||
		!strings.Contains(n.Reason, "hv1") {
		t.Fatalf("the refusal must name quorum and the member that is down: %q", n.Reason)
	}
	if len(f.tasks()) != 0 || !f.hv(b.ID).Schedulable {
		t.Fatal("nothing may be rebooted while a member is already down")
	}
}

// Five nodes survive one down and one rebooting: 3 of 5 still vote.
func TestMaintenanceProceedsWhenQuorumSurvivesAMemberDown(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1", offlineFor(time.Hour))
	for _, n := range []string{"hv2", "hv3", "hv4", "hv5"} {
		f.node(n)
	}
	f.window(f.hive)
	f.tick()
	if got := statuses(f.run()); !strings.HasPrefix(got, "hv1=skipped hv2=upgrading") {
		t.Fatalf("got %s", got)
	}
}

// A member whose agent has never reported its membership is not known to be up,
// and unknown is not up.
func TestMaintenanceCountsAnUnreportedMemberAsDown(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3", func(_, bee map[string]any) { bee["observed_membership_at"] = nil })
	f.window(f.hive)
	f.tick()
	if n := nodeByName(f.run(), "hv1"); n.Status != tenant.MaintenanceNodeFailed ||
		!strings.Contains(n.Reason, "hv3") {
		t.Fatalf("hv1: %s %q", n.Status, n.Reason)
	}
}

// "BACK" NEEDS A REPORT FROM AFTER THE UPGRADE. The quorate flag a node sent
// before it rebooted says nothing about whether it is voting now.
func TestMaintenanceWaitsForAMembershipReportFromAfterTheUpgrade(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()

	f.now = f.now.Add(10 * time.Minute)
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	// Heartbeating, but the last membership report is from before the reboot.
	f.db.Model(&tenant.Bee{}).Where("id = ?", *a.BeeID).Updates(map[string]any{
		"last_seen_at": f.now, "observed_membership_at": f.now.Add(-9 * time.Minute)})
	f.tick()
	if got := statuses(f.run()); got != "hv1=upgrading hv2=pending hv3=pending" {
		t.Fatalf("a pre-reboot quorum report must not count as back: %s", got)
	}

	f.now = f.now.Add(time.Minute)
	f.heartbeatAll()
	f.tick()
	if n := nodeByName(f.run(), "hv1"); n.Status != tenant.MaintenanceNodeSucceeded {
		t.Fatalf("hv1 reported after the upgrade and is back: %s %q", n.Status, n.Reason)
	}
}

// A STOP MEANS STOP. Another window waiting on the scope does not start the
// moment the stopped run ends — nor after a failure, nor after a run that
// already covered the window by starting inside it.
func TestMaintenanceAFinishedRunIsNotFollowedByAWaitingWindow(t *testing.T) {
	for _, how := range []string{
		tenant.MaintenanceRunStopped, tenant.MaintenanceRunFailed, tenant.MaintenanceRunSucceeded,
	} {
		t.Run(how, func(t *testing.T) {
			f := newFleet(t, "dc1")
			f.node("hv1")
			f.node("hv2")
			f.node("hv3")
			f.window(f.hive)
			f.window(f.hive)
			f.tick()
			if f.runs() != 1 {
				t.Fatalf("setup: one run, got %d", f.runs())
			}
			first := f.run()
			switch how {
			case tenant.MaintenanceRunStopped:
				f.db.Model(&tenant.MaintenanceRun{}).Where("id = ?", first.ID).
					Update("stop_requested", true)
				f.finishTask(nodeByName(first, "hv1"), tenant.FleetTaskSucceeded, "")
				f.tick()
			case tenant.MaintenanceRunFailed:
				f.finishTask(nodeByName(first, "hv1"), tenant.FleetTaskFailed, "boom")
				f.tick()
			case tenant.MaintenanceRunSucceeded:
				for range 3 {
					for _, n := range f.run().Nodes {
						if n.Status == tenant.MaintenanceNodeUpgrading {
							f.finishTask(n, tenant.FleetTaskSucceeded, "")
						}
					}
					f.tick()
				}
			}
			if got := f.run(); got.ID != first.ID || got.Status != how {
				t.Fatalf("setup: the first run should be %s, got %s (%s)", how, got.Status,
					statuses(got))
			}
			f.now = f.now.Add(time.Minute)
			f.heartbeatAll()
			f.tick()
			f.tick()
			if n := f.runs(); n != 1 {
				t.Fatalf("a window waiting behind a %s run started another: %d runs", how, n)
			}
			var waiting int64
			f.db.Model(&tenant.MaintenanceSchedule{}).Where("last_window_at IS NULL").
				Count(&waiting)
			if waiting != 0 {
				t.Fatal("the superseded window must be consumed, not left waiting")
			}
		})
	}
}

// A window that opens AFTER a stop is a new decision, and runs.
func TestMaintenanceAWindowOpeningAfterAStopStillRuns(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.db.Model(&tenant.MaintenanceRun{}).Where("1 = 1").Update("stop_requested", true)
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	if f.run().Status != tenant.MaintenanceRunStopped {
		t.Fatal("setup: stopped")
	}
	f.windowAt(f.hive, f.now.Add(10*time.Minute))
	f.now = f.now.Add(11 * time.Minute)
	f.heartbeatAll()
	f.tick()
	if n := f.runs(); n != 2 {
		t.Fatalf("a window that opened after the stop must run: %d runs", n)
	}
}

// A machine that departs or moves cluster during its turn is no longer this
// run's to hand back: an uncordon would offer placements on a host leaving.
func TestMaintenanceDoesNotUncordonANodeThatLeftDuringItsTurn(t *testing.T) {
	for _, how := range []string{"departing", "moved"} {
		t.Run(how, func(t *testing.T) {
			f := newFleet(t, "dc1")
			a := f.node("hv1")
			f.node("hv2")
			f.node("hv3")
			f.window(f.hive)
			f.tick()
			switch how {
			case "departing":
				f.db.Model(&tenant.Hypervisor{}).Where("id = ?", a.ID).
					Update("departing_at", f.now)
			case "moved":
				other := f.seedHive("dc2")
				f.db.Model(&tenant.Hypervisor{}).Where("id = ?", a.ID).
					Update("hive_id", other.ID)
			}
			f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
			f.tick()
			if f.hv(a.ID).Schedulable {
				t.Fatalf("a %s node was uncordoned", how)
			}
			if n := nodeByName(f.run(), "hv1"); !strings.Contains(n.Reason, "left cordoned") {
				t.Fatalf("and the node must say why: %q", n.Reason)
			}
		})
	}
}

// A run that failed is not an OK on the audit trail.
func TestMaintenanceAuditsAFailedRunAsFailed(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskFailed, "boom")
	f.tick()

	var ev tenant.AuditEvent
	if err := f.db.Where("action = ?", tenant.AuditMaintenanceFinish).First(&ev).Error; err != nil {
		t.Fatalf("no run_finish audit: %v", err)
	}
	if ev.Outcome != tenant.AuditOutcomeFailed {
		t.Fatalf("a failed run audited as %q", ev.Outcome)
	}
}

// The default deadline is the declared downtime plus 45 minutes. Every other
// test sets NodeDeadline, so this is the only one that sees the default.
func TestMaintenanceDefaultNodeDeadline(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	cfg := maintCfg
	cfg.NodeDeadline = 0
	if err := AdvanceMaintenance(context.Background(), f.db, cfg, f.now); err != nil {
		t.Fatalf("advance: %v", err)
	}
	n := nodeByName(f.run(), "hv1")
	want := f.now.Add(cfg.ExpectedDown + 45*time.Minute)
	if n.Deadline == nil || !n.Deadline.Equal(want) {
		t.Fatalf("deadline: want %s, got %v", want, n.Deadline)
	}
}
