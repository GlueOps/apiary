package jobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

// windowWith writes an enabled one-off window that opened a minute ago, with a
// budget. hive nil is the standalone pool.
func (f *mfleet) windowWith(hive *tenant.Hive, b tenant.MaintenanceBudget) tenant.MaintenanceSchedule {
	f.t.Helper()
	at := f.now.Add(-time.Minute)
	s := tenant.MaintenanceSchedule{
		Name: "budget", Kind: tenant.MaintenanceOnce, OnceAt: &at, DurationMinutes: 240,
		Reboot: tenant.RebootAlways, Enabled: true, MaintenanceBudget: b,
	}
	if hive != nil {
		s.HiveID = &hive.ID
	} else {
		s.Standalone = true
	}
	if err := tenant.ValidateMaintenanceSchedule(s); err != nil {
		f.t.Fatalf("invalid test window: %v", err)
	}
	if err := f.db.Create(&s).Error; err != nil {
		f.t.Fatalf("seed schedule: %v", err)
	}
	return s
}

// later moves the clock on past the run in progress, with every agent still
// reporting, so the next window is a new one.
func (f *mfleet) later() {
	f.now = f.now.Add(6 * time.Hour)
	f.heartbeatAll()
}

// inFlight is the node the run has upgrading, failing the test if there is none.
func (f *mfleet) inFlight() tenant.MaintenanceRunNode {
	f.t.Helper()
	r := f.run()
	for _, n := range r.Nodes {
		if n.Status == tenant.MaintenanceNodeUpgrading {
			return n
		}
	}
	f.t.Fatalf("nothing in flight: %s (%s, %q)", statuses(r), r.Status, r.Error)
	return tenant.MaintenanceRunNode{}
}

// completeInFlight finishes the node in flight with the given task status and
// ticks.
func (f *mfleet) completeInFlight(status string) {
	f.t.Helper()
	f.finishTask(f.inFlight(), status, "apt said no")
	f.heartbeatAll()
	f.tick()
}

func (f *mfleet) strikes(id uuid.UUID) repo.MaintenanceStrikes {
	f.t.Helper()
	s, err := repo.LoadMaintenanceStrikes(context.Background(), f.db, id)
	if err != nil {
		f.t.Fatalf("strikes: %v", err)
	}
	return s[id]
}

func five(f *mfleet) []tenant.Hypervisor {
	out := make([]tenant.Hypervisor, 0, 5)
	for _, n := range []string{"hv1", "hv2", "hv3", "hv4", "hv5"} {
		out = append(out, f.node(n))
	}
	return out
}

// THE DEFAULT IS THE OLD BEHAVIOUR. 0% stops at the first failure and says so
// exactly as before.
func TestFailureBudgetZeroStopsAtTheFirstFailure(t *testing.T) {
	f := newFleet(t, "dc1")
	five(f)
	f.windowWith(f.hive, tenant.MaintenanceBudget{})
	f.tick()
	f.completeInFlight(tenant.FleetTaskFailed)

	r := f.run()
	if r.Status != tenant.MaintenanceRunFailed ||
		r.Error != "hv1 did not complete: the upgrade failed: apt said no" {
		t.Fatalf("run: %s %q", r.Status, r.Error)
	}
	if got := statuses(r); got != "hv1=failed hv2=skipped hv3=skipped hv4=skipped hv5=skipped" {
		t.Fatalf("got %s", got)
	}
	if len(f.tasks()) != 1 {
		t.Fatal("nothing after the first failure may start")
	}
}

// Within budget a failure is recorded and the run carries on, one at a time. A
// run that ends with failures in it is still a FAILED run (R10).
func TestFailureBudgetCarriesOnPastAFailureWithinBudget(t *testing.T) {
	f := newFleet(t, "dc1")
	hvs := five(f)
	// 40% of 5 is 2: two failures carry on, the third would stop the run.
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaintenancePct: 40})
	f.tick()
	f.completeInFlight(tenant.FleetTaskFailed)

	if got := statuses(f.run()); got != "hv1=failed hv2=upgrading hv3=pending hv4=pending hv5=pending" {
		t.Fatalf("the run must carry on past a failure within budget: %s", got)
	}
	if f.hv(hvs[0].ID).Schedulable {
		t.Fatal("the failed node stays cordoned while the run carries on")
	}
	for range 4 {
		f.completeInFlight(tenant.FleetTaskSucceeded)
	}
	r := f.run()
	if got := statuses(r); got != "hv1=failed hv2=succeeded hv3=succeeded hv4=succeeded hv5=succeeded" {
		t.Fatalf("got %s", got)
	}
	if r.Status != tenant.MaintenanceRunFailed || !strings.Contains(r.Error, "1 of 5 node(s) failed, within") {
		t.Fatalf("a run with a failed node is not a success: %s %q", r.Status, r.Error)
	}
	var ev tenant.AuditEvent
	f.db.Where("action = ?", tenant.AuditMaintenanceFinish).First(&ev)
	if ev.Outcome != tenant.AuditOutcomeFailed {
		t.Fatalf("run_finish must be audited as failed: %q", ev.Outcome)
	}
}

func TestFailureBudgetStopsOnceFailuresExceedIt(t *testing.T) {
	f := newFleet(t, "dc1")
	five(f)
	// 20% of 5 is exactly 1: the first failure is within it, the second is not.
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaintenancePct: 20})
	f.tick()
	f.completeInFlight(tenant.FleetTaskFailed)
	f.completeInFlight(tenant.FleetTaskFailed)

	r := f.run()
	if got := statuses(r); got != "hv1=failed hv2=failed hv3=skipped hv4=skipped hv5=skipped" {
		t.Fatalf("got %s", got)
	}
	if r.Status != tenant.MaintenanceRunFailed ||
		!strings.Contains(r.Error, "2 of 5 node(s) failed, more than the 20% failure budget allows") ||
		!strings.Contains(r.Error, "hv2") {
		t.Fatalf("run: %s %q", r.Status, r.Error)
	}
	if len(f.tasks()) != 2 {
		t.Fatalf("no node may start once the budget is exceeded: %d tasks", len(f.tasks()))
	}
}

// A gate refusal spends the budget like any failure, but touches nothing.
func TestFailureBudgetCountsAGateRefusal(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1", func(hv, _ map[string]any) { hv["last_synced_at"] = time.Now().UTC().Add(-time.Hour) })
	for _, n := range []string{"hv2", "hv3", "hv4", "hv5"} {
		f.node(n)
	}
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaintenancePct: 20})
	f.tick()
	if got := statuses(f.run()); got != "hv1=failed hv2=upgrading hv3=pending hv4=pending hv5=pending" {
		t.Fatalf("a refusal within budget moves on in the same tick: %s", got)
	}
}

// C1. THE GATE ALONE IS NOT ENOUGH AFTER A FAILURE. Five nodes, one failed and
// now down: the gate allows a second reboot (3 of 5 still vote), but that
// leaves no vote for anything else to go wrong.
func TestFailureBudgetNeedsASpareVoteAfterAFailure(t *testing.T) {
	f := newFleet(t, "dc1")
	hvs := five(f)
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaintenancePct: 40})
	f.tick()
	f.finishTask(f.inFlight(), tenant.FleetTaskFailed, "kernel panic")
	f.now = f.now.Add(10 * time.Minute)
	for _, hv := range hvs[1:] {
		f.heartbeat(hv) // hv1 stays silent: it is down.
	}
	f.tick()

	r := f.run()
	if r.Status != tenant.MaintenanceRunFailed || !strings.Contains(r.Error, "no vote to spare") ||
		!strings.Contains(r.Error, "hv1") {
		t.Fatalf("run: %s %q (%s)", r.Status, r.Error, statuses(r))
	}
	if got := statuses(r); got != "hv1=failed hv2=skipped hv3=skipped hv4=skipped hv5=skipped" {
		t.Fatalf("got %s", got)
	}
	if len(f.tasks()) != 1 || !f.hv(hvs[1].ID).Schedulable {
		t.Fatal("no second node may be touched")
	}
}

// The standalone pool has no quorum to protect, so only the budget applies.
func TestFailureBudgetStandaloneNeedsNoSpareVote(t *testing.T) {
	f := newFleet(t, "")
	f.node("loose1")
	f.node("loose2")
	f.windowWith(nil, tenant.MaintenanceBudget{MaintenancePct: 50})
	f.tick()
	f.completeInFlight(tenant.FleetTaskFailed)
	if got := statuses(f.run()); got != "loose1=failed loose2=upgrading" {
		t.Fatalf("got %s", got)
	}
}

// C8. A node limit leaves the rest for the next window, and the next window
// starts with them rather than with the same first nodes by name.
func TestMaxNodesPerRunLeavesTheRestForTheNextWindow(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaxNodesPerRun: 2})
	f.tick()
	f.completeInFlight(tenant.FleetTaskSucceeded)
	f.completeInFlight(tenant.FleetTaskSucceeded)

	r := f.run()
	if got := statuses(r); got != "hv1=succeeded hv2=succeeded hv3=skipped" {
		t.Fatalf("got %s", got)
	}
	if r.Status != tenant.MaintenanceRunSucceeded ||
		!strings.Contains(nodeByName(r, "hv3").Reason, "limit of 2") {
		t.Fatalf("run: %s, hv3: %q", r.Status, nodeByName(r, "hv3").Reason)
	}

	f.later()
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaxNodesPerRun: 2})
	f.tick()
	r = f.run()
	if got := statuses(r); got != "hv3=upgrading hv1=pending hv2=pending" {
		t.Fatalf("the next window must start with the node never maintained: %s", got)
	}
	f.completeInFlight(tenant.FleetTaskSucceeded)
	f.completeInFlight(tenant.FleetTaskSucceeded)
	if got := statuses(f.run()); got != "hv3=succeeded hv1=succeeded hv2=skipped" {
		t.Fatalf("then the least recently maintained: %s", got)
	}
}

// The budget is a share of N, the node limit, not of the whole scope.
func TestFailureBudgetIsAShareOfTheNodeLimit(t *testing.T) {
	f := newFleet(t, "dc1")
	five(f)
	// 50% of a 2-node run is 1. Of the 5-node scope it would have been 2.
	f.windowWith(f.hive, tenant.MaintenanceBudget{MaxNodesPerRun: 2, MaintenancePct: 50})
	f.tick()
	f.completeInFlight(tenant.FleetTaskFailed)
	f.completeInFlight(tenant.FleetTaskFailed)
	r := f.run()
	if !strings.Contains(r.Error, "2 of 2 node(s) failed, more than") {
		t.Fatalf("run: %s %q", r.Status, r.Error)
	}
}

// C3. Two runs in a row in which a node's upgrade started and failed, and every
// later run skips it until it is acknowledged. The rest of the scope is still
// maintained.
func TestTwoFailedWindowsFlagTheNode(t *testing.T) {
	f := newFleet(t, "")
	bad := f.node("loose1")
	good := f.node("loose2")

	for i := range 2 {
		f.windowWith(nil, tenant.MaintenanceBudget{MaintenancePct: 50})
		f.tick()
		if n := f.inFlight(); n.Name != "loose1" {
			t.Fatalf("run %d: expected loose1 first, got %s", i, n.Name)
		}
		f.completeInFlight(tenant.FleetTaskFailed)    // loose1
		f.completeInFlight(tenant.FleetTaskSucceeded) // loose2
		f.later()
	}
	if s := f.strikes(bad.ID); s.Count != 2 || !s.Flagged() {
		t.Fatalf("two failed runs in a row are two strikes: %+v", s)
	}

	f.windowWith(nil, tenant.MaintenanceBudget{MaintenancePct: 50})
	f.tick()
	r := f.run()
	n := nodeByName(r, "loose1")
	if n.Status != tenant.MaintenanceNodeSkipped || !strings.Contains(n.Reason, "acknowledge") ||
		!strings.Contains(n.Reason, "apt said no") {
		t.Fatalf("a flagged node is skipped, with its failure: %s %q", n.Status, n.Reason)
	}
	if got := nodeByName(r, "loose2").Status; got != tenant.MaintenanceNodeUpgrading {
		t.Fatalf("the rest of the scope is still maintained: %s", got)
	}
	f.completeInFlight(tenant.FleetTaskSucceeded)
	if r = f.run(); r.Status != tenant.MaintenanceRunSucceeded {
		t.Fatalf("a skipped flagged node is not a failure of this run: %s %q", r.Status, r.Error)
	}
	if f.hv(bad.ID).Schedulable {
		t.Fatal("a flagged node stays cordoned")
	}
	_ = good
}

// C4. A refusal touches nothing, so it is never a strike: a healthy node
// refused because of the cluster around it must not end up flagged.
func TestGateRefusalIsNotAStrike(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	for range 3 {
		// Two nodes: every reboot would cost quorum, so the gate refuses hv1.
		f.windowWith(f.hive, tenant.MaintenanceBudget{})
		f.tick()
		if n := nodeByName(f.run(), "hv1"); n.Status != tenant.MaintenanceNodeFailed {
			t.Fatalf("hv1 must be refused: %s %q", n.Status, n.Reason)
		}
		f.later()
	}
	if s := f.strikes(a.ID); s.Count != 0 {
		t.Fatalf("a refusal is not a strike: %+v", s)
	}
}

// Strikes are "in a row": a successful run between two failures resets them.
func TestSuccessBetweenFailuresResetsStrikes(t *testing.T) {
	f := newFleet(t, "")
	a := f.node("loose1")
	for _, outcome := range []string{tenant.FleetTaskFailed, tenant.FleetTaskSucceeded,
		tenant.FleetTaskFailed} {
		f.windowWith(nil, tenant.MaintenanceBudget{})
		f.tick()
		f.completeInFlight(outcome)
		f.later()
	}
	if s := f.strikes(a.ID); s.Count != 1 || s.Flagged() {
		t.Fatalf("only the failure after the success counts: %+v", s)
	}
}

// An acknowledgement clears the strikes before it, and the next window tries
// the node again; a later success hands back the cordon the failures left.
func TestAcknowledgedNodeIsTriedAgain(t *testing.T) {
	f := newFleet(t, "")
	a := f.node("loose1")
	for range 2 {
		f.windowWith(nil, tenant.MaintenanceBudget{})
		f.tick()
		f.completeInFlight(tenant.FleetTaskFailed)
		f.later()
	}
	if !f.strikes(a.ID).Flagged() {
		t.Fatal("setup: loose1 should be flagged")
	}
	f.db.Model(&tenant.Hypervisor{}).Where("id = ?", a.ID).
		Update("maintenance_acknowledged_at", f.now)
	if s := f.strikes(a.ID); s.Count != 0 {
		t.Fatalf("an acknowledgement clears the strikes before it: %+v", s)
	}
	f.now = f.now.Add(time.Minute)
	f.heartbeatAll()
	f.windowWith(nil, tenant.MaintenanceBudget{})
	f.tick()
	if n := f.inFlight(); n.Name != "loose1" {
		t.Fatalf("an acknowledged node is tried again: %s", n.Name)
	}
	f.completeInFlight(tenant.FleetTaskSucceeded)
	if !f.hv(a.ID).Schedulable {
		t.Fatal("success hands back the cordon the failed runs left")
	}
}

// Zeros must be stored as zeros: these columns carry a default, and GORM drops
// a zero on Create in favour of it. Safe only while that default is also zero.
func TestMaintenanceBudgetZeroRoundTrips(t *testing.T) {
	f := newFleet(t, "dc1")
	s := f.windowWith(f.hive, tenant.MaintenanceBudget{})
	var got tenant.MaintenanceSchedule
	f.db.First(&got, "id = ?", s.ID)
	if got.MaxNodesPerRun != 0 || got.MaintenancePct != 0 {
		t.Fatalf("zero must round-trip: %+v", got.MaintenanceBudget)
	}
	s2 := f.windowWith(f.hive, tenant.MaintenanceBudget{MaxNodesPerRun: 7, MaintenancePct: 30})
	// A fresh struct: First on one that already has a primary key adds it to
	// the WHERE clause.
	var got2 tenant.MaintenanceSchedule
	f.db.First(&got2, "id = ?", s2.ID)
	if got2.MaxNodesPerRun != 7 || got2.MaintenancePct != 30 {
		t.Fatalf("set values must round-trip: %+v", got2.MaintenanceBudget)
	}
}

// The database refuses what validation would, for a write that went around it.
func TestMaintenanceBudgetCheckConstraints(t *testing.T) {
	f := newFleet(t, "dc1")
	s := f.windowWith(f.hive, tenant.MaintenanceBudget{})
	for col, v := range map[string]int{"maintenance_pct": 51, "max_nodes_per_run": -1} {
		if err := f.db.Model(&tenant.MaintenanceSchedule{}).Where("id = ?", s.ID).
			Update(col, v).Error; err == nil {
			t.Fatalf("%s = %d must be refused by the database", col, v)
		}
	}
}
