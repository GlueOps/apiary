package jobs

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
)

var maintenanceSchema = testsupport.SchemaName("apiary_maintenance")

// maintenanceDB opens the shared schema and empties it: AdvanceMaintenance
// works every schedule and run in the database, so one test's leftovers would
// be the next test's fleet.
func maintenanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, maintenanceSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{}, &tenant.FleetTask{},
		&tenant.AuditEvent{}, &tenant.MaintenanceSchedule{}, &tenant.MaintenanceRun{},
		&tenant.MaintenanceRunNode{})
	if err := db.Exec(`TRUNCATE maintenance_run_nodes, maintenance_runs, maintenance_schedules,
		audit_events, fleet_tasks, hypervisors, bees, hives CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

var maintCfg = MaintenanceConfig{
	ObservationMaxAge: 10 * time.Minute,
	OnlineGrace:       90 * time.Second,
	ExpectedDown:      15 * time.Minute,
	NodeDeadline:      time.Hour,
}

type mfleet struct {
	t    *testing.T
	db   *gorm.DB
	now  time.Time
	hive *tenant.Hive
}

func newFleet(t *testing.T, hiveName string) *mfleet {
	f := &mfleet{t: t, db: maintenanceDB(t), now: time.Now().UTC().Truncate(time.Second)}
	if hiveName != "" {
		f.hive = f.seedHive(hiveName)
	}
	return f
}

func (f *mfleet) seedHive(name string) *tenant.Hive {
	h := &tenant.Hive{Name: name}
	if err := f.db.Create(h).Error; err != nil {
		f.t.Fatalf("seed hive: %v", err)
	}
	return h
}

type nodeOpt func(hv map[string]any, bee map[string]any)

func noAgent() nodeOpt    { return func(hv, _ map[string]any) { hv["bee_id"] = nil } }
func departing() nodeOpt  { return func(hv, _ map[string]any) { hv["departing_at"] = time.Now().UTC() } }
func cordoned() nodeOpt   { return func(hv, _ map[string]any) { hv["schedulable"] = false } }
func standalone() nodeOpt { return func(hv, _ map[string]any) { hv["hive_id"] = nil } }
func offlineFor(d time.Duration) nodeOpt {
	return func(_, bee map[string]any) { bee["last_seen_at"] = time.Now().UTC().Add(-d) }
}

// node seeds a healthy, observed, schedulable member of the fleet's hive and
// applies opts. Every column is set by Updates after Create, because several of
// these carry default: tags that a Create would silently override.
func (f *mfleet) node(name string, opts ...nodeOpt) tenant.Hypervisor {
	f.t.Helper()
	bee := tenant.Bee{APIKeyID: uuid.New()}
	if err := f.db.Create(&bee).Error; err != nil {
		f.t.Fatalf("seed bee: %v", err)
	}
	beeUp := map[string]any{"last_seen_at": f.now, "observed_quorate": true, "status": "online"}
	if f.hive != nil {
		beeUp["observed_cluster"] = f.hive.Name
	}
	hv := tenant.Hypervisor{Address: name + ".example.test"}
	if err := f.db.Create(&hv).Error; err != nil {
		f.t.Fatalf("seed hypervisor: %v", err)
	}
	hvUp := map[string]any{
		"name": name, "bee_id": bee.ID, "schedulable": true, "last_synced_at": f.now,
		"provisioning_state": "provisioned",
	}
	if f.hive != nil {
		hvUp["hive_id"] = f.hive.ID
	} else {
		hvUp["hive_id"] = nil
	}
	for _, o := range opts {
		o(hvUp, beeUp)
	}
	if err := f.db.Model(&tenant.Bee{}).Where("id = ?", bee.ID).Updates(beeUp).Error; err != nil {
		f.t.Fatalf("update bee: %v", err)
	}
	if err := f.db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).Updates(hvUp).Error; err != nil {
		f.t.Fatalf("update hypervisor: %v", err)
	}
	return f.hv(hv.ID)
}

func (f *mfleet) hv(id uuid.UUID) tenant.Hypervisor {
	f.t.Helper()
	var hv tenant.Hypervisor
	if err := f.db.First(&hv, "id = ?", id).Error; err != nil {
		f.t.Fatalf("reload hypervisor: %v", err)
	}
	return hv
}

// window writes an enabled one-off window that opened a minute ago.
func (f *mfleet) window(hive *tenant.Hive) tenant.MaintenanceSchedule {
	f.t.Helper()
	at := f.now.Add(-time.Minute)
	s := tenant.MaintenanceSchedule{
		Name: "test", Kind: tenant.MaintenanceOnce, OnceAt: &at, DurationMinutes: 240,
		Reboot: tenant.RebootAlways, Enabled: true,
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

func (f *mfleet) tick() {
	f.t.Helper()
	if err := AdvanceMaintenance(context.Background(), f.db, maintCfg, f.now); err != nil {
		f.t.Fatalf("advance: %v", err)
	}
}

func (f *mfleet) run() tenant.MaintenanceRun {
	f.t.Helper()
	var r tenant.MaintenanceRun
	if err := f.db.Preload("Nodes", func(q *gorm.DB) *gorm.DB { return q.Order("position") }).
		Order("started_at DESC").First(&r).Error; err != nil {
		f.t.Fatalf("load run: %v", err)
	}
	return r
}

func (f *mfleet) runs() int64 {
	var n int64
	f.db.Model(&tenant.MaintenanceRun{}).Count(&n)
	return n
}

func (f *mfleet) tasks() []tenant.FleetTask {
	var ts []tenant.FleetTask
	f.db.Order("created_at").Find(&ts)
	return ts
}

// finishTask completes a node's upgrade as its agent would.
func (f *mfleet) finishTask(n tenant.MaintenanceRunNode, status, errText string) {
	f.t.Helper()
	if n.TaskID == nil {
		f.t.Fatalf("%s has no task to finish", n.Name)
	}
	if err := f.db.Model(&tenant.FleetTask{}).Where("id = ?", *n.TaskID).
		Updates(map[string]any{"status": status, "error": errText, "finished_at": f.now}).
		Error; err != nil {
		f.t.Fatalf("finish task: %v", err)
	}
}

// heartbeat brings a node's agent and its capacity observation up to f.now, as
// a live host's would be after the clock is moved on.
func (f *mfleet) heartbeat(hv tenant.Hypervisor) {
	f.db.Model(&tenant.Bee{}).Where("id = ?", *hv.BeeID).
		Updates(map[string]any{"last_seen_at": f.now, "unavailable_until": nil})
	f.db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).Update("last_synced_at", f.now)
}

func nodeByName(r tenant.MaintenanceRun, name string) tenant.MaintenanceRunNode {
	for _, n := range r.Nodes {
		if n.Name == name {
			return n
		}
	}
	return tenant.MaintenanceRunNode{}
}

func statuses(r tenant.MaintenanceRun) string {
	parts := make([]string, 0, len(r.Nodes))
	for _, n := range r.Nodes {
		parts = append(parts, n.Name+"="+n.Status)
	}
	return strings.Join(parts, " ")
}

// The rollout: one node at a time, cordoned while it is away, handed back when
// it is back, and the next node only then.
func TestMaintenanceUpgradesOneNodeAtATime(t *testing.T) {
	f := newFleet(t, "dc1")
	a, b, c := f.node("hv1"), f.node("hv2"), f.node("hv3")
	f.window(f.hive)

	f.tick()
	r := f.run()
	if got := statuses(r); got != "hv1=upgrading hv2=pending hv3=pending" {
		t.Fatalf("first tick: %s", got)
	}
	if f.hv(a.ID).Schedulable {
		t.Fatal("the node being upgraded must be cordoned")
	}
	if !f.hv(b.ID).Schedulable || !f.hv(c.ID).Schedulable {
		t.Fatal("nodes not yet reached must be left alone")
	}

	ts := f.tasks()
	if len(ts) != 1 || ts[0].Kind != repo.KindDistUpgrade {
		t.Fatalf("want exactly one dist-upgrade queued, got %+v", ts)
	}
	var spec map[string]any
	_ = json.Unmarshal(ts[0].Spec, &spec)
	if spec["allow_guest_downtime"] != true || spec["reboot"] != tenant.RebootAlways {
		t.Fatalf("the window grants guest downtime and carries its reboot policy: %v", spec)
	}

	// A tick while hv1 is still away starts nothing else.
	f.tick()
	if got := statuses(f.run()); got != "hv1=upgrading hv2=pending hv3=pending" {
		t.Fatalf("a second node started while the first was in flight: %s", got)
	}

	for i, want := range []string{
		"hv1=succeeded hv2=upgrading hv3=pending",
		"hv1=succeeded hv2=succeeded hv3=upgrading",
		"hv1=succeeded hv2=succeeded hv3=succeeded",
	} {
		r = f.run()
		var inFlight tenant.MaintenanceRunNode
		for _, n := range r.Nodes {
			if n.Status == tenant.MaintenanceNodeUpgrading {
				inFlight = n
			}
		}
		f.finishTask(inFlight, tenant.FleetTaskSucceeded, "")
		f.tick()
		if got := statuses(f.run()); got != want {
			t.Fatalf("step %d: want %s, got %s", i, want, got)
		}
	}
	if r = f.run(); r.Status != tenant.MaintenanceRunSucceeded {
		t.Fatalf("run: %s", r.Status)
	}
	for _, hv := range []tenant.Hypervisor{a, b, c} {
		if !f.hv(hv.ID).Schedulable {
			t.Fatalf("%s: the cordon the run took must be handed back", *hv.Name)
		}
	}
	if len(f.tasks()) != 3 {
		t.Fatalf("one upgrade per node: %d", len(f.tasks()))
	}
}

// A task that succeeded is not a node that is back. The run waits for the
// heartbeat and quorum, and stops when the deadline passes without them.
func TestMaintenanceStopsAtANodeThatDoesNotComeBack(t *testing.T) {
	f := newFleet(t, "dc1")
	a, b := f.node("hv1"), f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()

	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	// The agent reported and went silent.
	f.db.Model(&tenant.Bee{}).Where("id = ?", *a.BeeID).
		Update("last_seen_at", f.now.Add(-10*time.Minute))
	f.tick()
	if got := statuses(f.run()); got != "hv1=upgrading hv2=pending hv3=pending" {
		t.Fatalf("must wait for hv1 before its deadline: %s", got)
	}

	f.now = f.now.Add(2 * time.Hour)
	f.tick()
	r := f.run()
	if r.Status != tenant.MaintenanceRunFailed {
		t.Fatalf("run must fail: %s", r.Status)
	}
	if got := statuses(r); got != "hv1=failed hv2=skipped hv3=skipped" {
		t.Fatalf("got %s", got)
	}
	if n := nodeByName(r, "hv1"); !strings.Contains(n.Reason, "not back") {
		t.Fatalf("the failure must say why: %q", n.Reason)
	}
	if f.hv(a.ID).Schedulable {
		t.Fatal("a node that did not come back stays cordoned")
	}
	if !f.hv(b.ID).Schedulable || len(f.tasks()) != 1 {
		t.Fatal("nothing after the failed node may be touched")
	}
}

func TestMaintenanceStopsWhenAnUpgradeFails(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskFailed, "dpkg exploded")
	f.tick()

	r := f.run()
	if r.Status != tenant.MaintenanceRunFailed || !strings.Contains(r.Error, "dpkg exploded") {
		t.Fatalf("the run must fail and say why: %s %q", r.Status, r.Error)
	}
	if f.hv(a.ID).Schedulable {
		t.Fatal("a node whose upgrade failed stays cordoned")
	}
	if len(f.tasks()) != 1 {
		t.Fatal("no further node may start")
	}
}

// A node whose agent never collects the task: the task is withdrawn, so the
// agent cannot pick up a reboot hours later when it returns.
func TestMaintenanceWithdrawsAnUncollectedUpgrade(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.now = f.now.Add(2 * time.Hour)
	f.tick()
	if ts := f.tasks(); len(ts) != 1 || ts[0].Status != tenant.FleetTaskCancelled {
		t.Fatalf("the uncollected task must be withdrawn: %+v", ts)
	}
	if f.run().Status != tenant.MaintenanceRunFailed {
		t.Fatal("and the run stops on it")
	}
}

// Detached machines — no agent, offline agent, a departure under way, an agent
// that does not report the cluster — are skipped with a reason, and the run
// carries on with the rest.
func TestMaintenanceSkipsDetachedMachines(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1", noAgent())
	f.node("hv2", offlineFor(time.Hour))
	f.node("hv3", departing())
	f.node("hv4", func(_, bee map[string]any) { bee["observed_cluster"] = "other" })
	healthy := f.node("hv5")
	f.window(f.hive)
	f.tick()

	r := f.run()
	if got := statuses(r); got != "hv1=skipped hv2=skipped hv3=skipped hv4=skipped hv5=upgrading" {
		t.Fatalf("got %s", got)
	}
	for name, want := range map[string]string{
		"hv1": "no agent", "hv2": "offline since", "hv3": "departure",
		"hv4": "does not report being in this cluster",
	} {
		if n := nodeByName(r, name); !strings.Contains(n.Reason, want) {
			t.Errorf("%s: reason %q should mention %q", name, n.Reason, want)
		}
	}
	f.finishTask(nodeByName(r, "hv5"), tenant.FleetTaskSucceeded, "")
	f.tick()
	if r = f.run(); r.Status != tenant.MaintenanceRunSucceeded {
		t.Fatalf("skips do not fail the run: %s", r.Status)
	}
	if !f.hv(healthy.ID).Schedulable {
		t.Fatal("cordon handed back")
	}
	for _, hv := range []string{"hv1", "hv2", "hv3", "hv4"} {
		if n := nodeByName(r, hv); n.TaskID != nil || n.CordonedAt != nil {
			t.Fatalf("%s was touched", hv)
		}
	}
}

// Somebody cordons the node by hand while the run has it. The run cannot know
// why, so it leaves the cordon in place.
func TestMaintenanceLeavesAHandCordonedNodeCordoned(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()

	user := uuid.New()
	if err := repo.RecordAudit(context.Background(), f.db, &tenant.AuditEvent{
		PrincipalID: user, PrincipalKind: tenant.PrincipalKindUser,
		Action: tenant.AuditHypervisorCordon, TargetKind: tenant.AuditTargetHypervisor,
		TargetID: &a.ID, Outcome: tenant.AuditOutcomeOK,
		CreatedAt: f.now.Add(time.Second),
	}); err != nil {
		t.Fatalf("audit: %v", err)
	}
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()

	n := nodeByName(f.run(), "hv1")
	if n.Status != tenant.MaintenanceNodeSucceeded {
		t.Fatalf("hv1: %s", n.Status)
	}
	if f.hv(a.ID).Schedulable {
		t.Fatal("a node cordoned by hand during the run must stay cordoned")
	}
	if !strings.Contains(n.Reason, "by hand") {
		t.Fatalf("and say so: %q", n.Reason)
	}
}

// A node the run found already cordoned was somebody else's decision.
func TestMaintenanceDoesNotUncordonANodeItFoundCordoned(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1", cordoned())
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	if nodeByName(f.run(), "hv1").Status != tenant.MaintenanceNodeSucceeded {
		t.Fatal("it is still upgraded")
	}
	if f.hv(a.ID).Schedulable {
		t.Fatal("but its cordon is not the run's to remove")
	}
}

// A failed run is retried by the next window when the node is back online, and
// the retry inherits the cordon the failed run left — so success hands it back.
func TestMaintenanceRetryHandsBackTheCordonAFailedRunLeft(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskFailed, "boom")
	f.tick()
	if f.run().Status != tenant.MaintenanceRunFailed || f.hv(a.ID).Schedulable {
		t.Fatal("setup: run one fails and leaves hv1 cordoned")
	}

	f.now = f.now.Add(5 * time.Hour)
	f.heartbeat(a)
	for _, hv := range []string{"hv2", "hv3"} {
		var h tenant.Hypervisor
		f.db.First(&h, "name = ?", hv)
		f.heartbeat(h)
	}
	f.window(f.hive)
	f.tick()
	r := f.run()
	if got := statuses(r); !strings.HasPrefix(got, "hv1=upgrading") {
		t.Fatalf("the next window retries hv1: %s", got)
	}
	f.finishTask(nodeByName(r, "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	if !f.hv(a.ID).Schedulable {
		t.Fatal("the retry must hand back the cordon the failed run took")
	}
}

// Two windows on one cluster opening together are still one run.
func TestMaintenanceRunsOneAtATimePerCluster(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.window(f.hive)

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = AdvanceMaintenance(context.Background(), f.db, maintCfg, f.now)
		}()
	}
	wg.Wait()
	if n := f.runs(); n != 1 {
		t.Fatalf("one run per cluster, got %d", n)
	}
	var upgrading int64
	f.db.Model(&tenant.MaintenanceRunNode{}).Where("status = ?", "upgrading").Count(&upgrading)
	if upgrading != 1 || len(f.tasks()) != 1 {
		t.Fatalf("exactly one node in flight, got %d nodes and %d tasks", upgrading, len(f.tasks()))
	}
}

// The upgrade gate still applies at each node's turn: a two-node cluster cannot
// lose one to a reboot. The refusal fails the run and leaves the node as found.
func TestMaintenanceGateRefusalStopsTheRunAndTouchesNothing(t *testing.T) {
	f := newFleet(t, "dc1")
	a, b := f.node("hv1"), f.node("hv2")
	f.window(f.hive)
	f.tick()

	r := f.run()
	if r.Status != tenant.MaintenanceRunFailed {
		t.Fatalf("run: %s (%s)", r.Status, statuses(r))
	}
	if n := nodeByName(r, "hv1"); !strings.Contains(n.Reason, "quorum") {
		t.Fatalf("the refusal must be named: %q", n.Reason)
	}
	if !f.hv(a.ID).Schedulable || !f.hv(b.ID).Schedulable || len(f.tasks()) != 0 {
		t.Fatal("a refused node is left exactly as it was found")
	}
}

// A stale observation is not lifted by the window's guest-downtime grant.
func TestMaintenanceStillRefusesAStaleObservation(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1", func(hv, _ map[string]any) {
		hv["last_synced_at"] = time.Now().UTC().Add(-time.Hour)
	})
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	if n := nodeByName(f.run(), "hv1"); n.Status != tenant.MaintenanceNodeFailed ||
		!strings.Contains(n.Reason, "stale") {
		t.Fatalf("hv1: %s %q", n.Status, n.Reason)
	}
	if !f.hv(a.ID).Schedulable {
		t.Fatal("left as found")
	}
}

// The standalone pool is every provisioned machine in no cluster — and only
// those.
func TestMaintenanceStandalonePool(t *testing.T) {
	f := newFleet(t, "")
	loose := f.node("loose1", standalone())
	f.hive = f.seedHive("dc1")
	clustered := f.node("member1")
	f.hive = nil
	f.window(nil)
	f.tick()

	r := f.run()
	if !r.Standalone || len(r.Nodes) != 1 || r.Nodes[0].HypervisorID != loose.ID {
		t.Fatalf("the pool must hold only the unclustered machine: %s", statuses(r))
	}
	if r.Nodes[0].Status != tenant.MaintenanceNodeUpgrading {
		t.Fatalf("no quorum to protect for a standalone host: %s %q", r.Nodes[0].Status, r.Nodes[0].Reason)
	}
	if !f.hv(clustered.ID).Schedulable {
		t.Fatal("a clustered machine is not the pool's")
	}
}

// A machine that joins a cluster after the standalone run snapshotted it is
// that cluster's business now.
func TestMaintenanceStandaloneSkipsAMachineThatJoinedACluster(t *testing.T) {
	f := newFleet(t, "")
	f.node("loose1", standalone())
	second := f.node("loose2", standalone())
	f.window(nil)
	f.tick()
	h := f.seedHive("dc9")
	f.db.Model(&tenant.Hypervisor{}).Where("id = ?", second.ID).Update("hive_id", h.ID)
	f.finishTask(nodeByName(f.run(), "loose1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	if n := nodeByName(f.run(), "loose2"); n.Status != tenant.MaintenanceNodeSkipped ||
		!strings.Contains(n.Reason, "joined a cluster") {
		t.Fatalf("loose2: %s %q", n.Status, n.Reason)
	}
}

// A window that closes stops new nodes; the node in flight is still seen home.
func TestMaintenanceWindowCloseLetsTheNodeInFlightFinish(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()

	f.now = f.now.Add(4*time.Hour + time.Minute)
	f.heartbeat(a)
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	r := f.run()
	if r.Status != tenant.MaintenanceRunExpired {
		t.Fatalf("run: %s", r.Status)
	}
	if got := statuses(r); got != "hv1=succeeded hv2=skipped hv3=skipped" {
		t.Fatalf("got %s", got)
	}
	if !f.hv(a.ID).Schedulable {
		t.Fatal("the in-flight node is still handed back")
	}
}

func TestMaintenanceStopAfterTheNodeInFlight(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.db.Model(&tenant.MaintenanceRun{}).Where("1 = 1").Update("stop_requested", true)
	f.tick()
	if got := statuses(f.run()); got != "hv1=upgrading hv2=pending hv3=pending" {
		t.Fatalf("a stop does not abandon the node in flight: %s", got)
	}
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()
	r := f.run()
	if r.Status != tenant.MaintenanceRunStopped || statuses(r) != "hv1=succeeded hv2=skipped hv3=skipped" {
		t.Fatalf("got %s: %s", r.Status, statuses(r))
	}
}

// One window, one run, however many ticks land in it.
func TestMaintenanceOneRunPerWindow(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1", noAgent())
	f.window(f.hive)
	f.tick()
	if f.run().Status != tenant.MaintenanceRunSucceeded {
		t.Fatal("setup: a run with nothing to do finishes at once")
	}
	f.now = f.now.Add(time.Minute)
	f.tick()
	if n := f.runs(); n != 1 {
		t.Fatalf("the same window must not start a second run: %d", n)
	}
}

// A disabled window opens nothing.
func TestMaintenanceDisabledWindowStartsNothing(t *testing.T) {
	f := newFleet(t, "dc1")
	f.node("hv1")
	s := f.window(f.hive)
	f.db.Model(&tenant.MaintenanceSchedule{}).Where("id = ?", s.ID).Update("enabled", false)
	f.tick()
	if f.runs() != 0 {
		t.Fatal("a disabled window must not start a run")
	}
}

// The run's actions are on the audit trail, as the system and naming the run.
func TestMaintenanceAuditsItsCordons(t *testing.T) {
	f := newFleet(t, "dc1")
	a := f.node("hv1")
	f.node("hv2")
	f.node("hv3")
	f.window(f.hive)
	f.tick()
	f.finishTask(nodeByName(f.run(), "hv1"), tenant.FleetTaskSucceeded, "")
	f.tick()

	var evs []tenant.AuditEvent
	f.db.Where("target_id = ?", a.ID).Order("created_at").Find(&evs)
	if len(evs) != 2 || evs[0].Action != tenant.AuditHypervisorCordon ||
		evs[1].Action != tenant.AuditHypervisorUncordon {
		t.Fatalf("want cordon then uncordon on hv1, got %+v", evs)
	}
	for _, e := range evs {
		if e.PrincipalKind != tenant.PrincipalKindSystem || e.PrincipalID != f.run().ID {
			t.Fatalf("recorded as the run: %+v", e)
		}
	}
}
