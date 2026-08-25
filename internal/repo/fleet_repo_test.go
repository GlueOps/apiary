package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var fleetTestSchema = testsupport.SchemaName("apiary_repo_fleet_test")

// fleetDB stands in for ONE organization's database.
//
// No control tables: the fleet lives entirely in a tenant database now, and the
// only fleet row still in control — the enrolment token — is exercised by
// enrolmentDB below. Keeping them apart in the fixture is not tidiness; a schema
// holding both would let a query join across the boundary and pass here while
// failing against real, separate databases.
func fleetDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, fleetTestSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.FleetTask{}, &tenant.FleetTaskLog{},
	)
}

func fleetSeedHive(t *testing.T, db *gorm.DB, name string) tenant.Hive {
	t.Helper()
	// No organization: the database is the organization.
	h := tenant.Hive{
		Name: name + "-" + uuid.NewString()[:8], Provider: "proxmox",
	}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}
	return h
}

// fleetSeedBee seeds BOTH rows one host is: the machine record and the agent on
// it. Two creates, because hive membership, address and machine id live on the
// machine now — a bee alone is an agent with nothing underneath it, which no
// enrolment can produce.
func fleetSeedBee(t *testing.T, db *gorm.DB, hiveID uuid.UUID, host string) tenant.Bee {
	t.Helper()
	now := time.Now().UTC()
	b := tenant.Bee{
		APIKeyID: uuid.New(),
		Status:   tenant.BeeStatusOnline, LastSeenAt: &now, EnrolledAt: now,
	}
	if err := db.Create(&b).Error; err != nil {
		t.Fatalf("create bee: %v", err)
	}
	machineID := uuid.NewString()
	name := host
	if err := db.Create(&tenant.Hypervisor{
		HiveID: &hiveID, Name: &name, Address: host,
		MachineID: &machineID, BeeID: &b.ID,
	}).Error; err != nil {
		t.Fatalf("create machine: %v", err)
	}
	return b
}

// machineFor reads the machine record behind an agent.
func machineFor(t *testing.T, db *gorm.DB, beeID uuid.UUID) tenant.Hypervisor {
	t.Helper()
	var m tenant.Hypervisor
	if err := db.First(&m, "bee_id = ?", beeID).Error; err != nil {
		t.Fatalf("no machine record is linked to bee %s: %v — enrolment must create both rows "+
			"in one transaction, or a machine ends up with a credential nothing can find",
			beeID, err)
	}
	return m
}

// A hypervisor that is rebuilt must land on the SAME bee. Otherwise a rebuild
// mid-rotation leaves a ghost row that is permanently offline, and an operator
// has two entries for one machine with no way to tell which is live.
func TestEnrolIsIdempotentOnTheMachineID(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	machine := uuid.NewString()

	first, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: machine, Address: "hv-01", APIKeyID: uuid.New(),
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first enrol: %v", err)
	}
	firstMachine := machineFor(t, db, first.ID)

	newKey := uuid.New()
	second, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: machine, Address: "hv-01-renamed", APIKeyID: newKey,
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("re-enrol: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("re-enrolling the same machine created a second bee (%s then %s); the first is "+
			"now a ghost that can never come online again", first.ID, second.ID)
	}
	if second.APIKeyID != newKey {
		t.Fatal("re-enrolment did not bind the new agent key, so the rebuilt host cannot authenticate")
	}

	secondMachine := machineFor(t, db, second.ID)
	if secondMachine.ID != firstMachine.ID {
		t.Fatalf("re-enrolling the same machine created a second MACHINE record (%s then %s); "+
			"its onboarding history, cluster membership and swarm assignment all stayed on the "+
			"first one", firstMachine.ID, secondMachine.ID)
	}
	if secondMachine.Address != "hv-01-renamed" {
		t.Fatalf("re-enrolment did not update the machine's address (got %q); the control plane "+
			"would keep reaching for the old one", secondMachine.Address)
	}
	var machines int64
	if err := db.Model(&tenant.Hypervisor{}).Where("machine_id = ?", machine).
		Count(&machines).Error; err != nil {
		t.Fatal(err)
	}
	if machines != 1 {
		t.Fatalf("%d machine records carry this machine id, want 1", machines)
	}
}

// A host enrols with NO fleet, and re-enrolling must not silently un-join one.
//
// Both halves matter. Enrolling fleetless is what lets a new host and a host
// migrating between fleets follow the same path. Leaving membership alone on
// re-enrolment is what stops an agent restart on a running hypervisor from
// quietly removing it from the fleet the control plane last confirmed.
func TestEnrolStartsFleetlessAndDoesNotResetMembership(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	machine := uuid.NewString()

	first, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: machine, Address: "hv-02", APIKeyID: uuid.New(),
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first enrol: %v", err)
	}
	if m := machineFor(t, db, first.ID); m.HiveID != nil {
		t.Fatalf("a freshly enrolled host was placed in fleet %s; joining one is a task, and "+
			"enrolling straight into a fleet means the migration path is never exercised", *m.HiveID)
	}

	// It joins a fleet, as the arrival worker would record: hive AND node name.
	hive := fleetSeedHive(t, db, "dc1")
	if err := db.Model(&tenant.Hypervisor{}).Where("bee_id = ?", first.ID).
		Updates(map[string]any{"hive_id": hive.ID, "name": "hv-02"}).Error; err != nil {
		t.Fatalf("join fleet: %v", err)
	}

	second, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: machine, Address: "hv-02", APIKeyID: uuid.New(),
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("re-enrol: %v", err)
	}
	m := machineFor(t, db, second.ID)
	if m.HiveID == nil || *m.HiveID != hive.ID {
		t.Fatalf("re-enrolment cleared fleet membership; a running hypervisor whose agent " +
			"restarted would silently leave the fleet it is still a member of")
	}
	if m.Name == nil || *m.Name != "hv-02" {
		t.Fatal("re-enrolment cleared the node name, which is the capacity sync's half of this " +
			"machine's identity — the next sync would not recognise the row and would duplicate it")
	}
}

// The other half of the merge, from the enrolment side: a node the capacity sync
// discovered in a cluster, that an agent is only afterwards installed on, must
// end as ONE row.
//
// Discovered rows carry (hive_id, name) and no machine id, because
// /cluster/resources reports none. Enrolment matches on machine id, finds
// nothing, and without the address-to-node-name fallback creates a second record
// for a machine already in the table — one carrying the cluster membership and
// the capacity, the other carrying the agent, and neither being the machine.
func TestDiscoveredFirstOnboardedSecondIsOneRow(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc-merge")

	name := "hv-07"
	discovered := tenant.Hypervisor{
		HiveID: &hive.ID, Name: &name,
		ObservedCPUTotal: 16, ObservedGuests: 3,
	}
	if err := db.Create(&discovered).Error; err != nil {
		t.Fatal(err)
	}

	bee, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: uuid.NewString(), Address: "hv-07.dc1.example.com",
		APIKeyID: uuid.New(), Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}

	// Scoped to this test's own hive and agent: the fixture schema is shared
	// across the suite, so an unqualified count would be reporting other tests.
	var all []tenant.Hypervisor
	if err := db.Where("hive_id = ? OR bee_id = ?", hive.ID, bee.ID).Find(&all).Error; err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("%d machine records exist for one physical machine, want 1: the cluster "+
			"membership and the agent are now on different rows and neither is the machine", len(all))
	}
	m := all[0]
	if m.ID != discovered.ID {
		t.Fatalf("enrolment created a new record (%s) instead of adopting the discovered one "+
			"(%s)", m.ID, discovered.ID)
	}
	if m.BeeID == nil || *m.BeeID != bee.ID {
		t.Fatal("the discovered record was not linked to the enrolling agent")
	}
	if m.MachineID == nil {
		t.Fatal("the adopted record did not learn the machine id, so the next re-enrolment " +
			"will not find it either")
	}
	if m.HiveID == nil || *m.HiveID != hive.ID || m.ObservedGuests != 3 {
		t.Fatalf("adoption overwrote what the sync had observed: %+v", m)
	}
}

// Adoption REFUSES when it cannot tell which machine it is looking at.
//
// Two discovered nodes named hv-08 in different clusters give no evidence which
// one the enrolling host is. Writing an agent's credential and machine id onto
// the wrong physical box is far worse than a duplicate record an operator can
// merge deliberately, so the ambiguous case creates a fresh row.
func TestEnrolRefusesToAdoptAnAmbiguousNodeName(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	dc1 := fleetSeedHive(t, db, "amb-dc1")
	dc2 := fleetSeedHive(t, db, "amb-dc2")

	name := "hv-08"
	for _, hiveID := range []uuid.UUID{dc1.ID, dc2.ID} {
		h := hiveID
		n := name
		if err := db.Create(&tenant.Hypervisor{HiveID: &h, Name: &n}).Error; err != nil {
			t.Fatal(err)
		}
	}

	bee, err := r.EnrolBee(context.Background(), EnrolBeeInput{
		MachineID: uuid.NewString(), Address: "hv-08.example.com",
		APIKeyID: uuid.New(), Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}
	m := machineFor(t, db, bee.ID)
	if m.HiveID != nil {
		t.Fatalf("enrolment attached this agent to a machine in hive %s on the strength of a "+
			"node name that two clusters both use; a fleet operation would now act on the "+
			"wrong physical host", *m.HiveID)
	}
}

// At most one outstanding task per bee. This is what makes "what is this
// hypervisor doing" answerable from one row during a rotation.
func TestABeeMayHoldOnlyOneOutstandingTask(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc2")
	bee := fleetSeedBee(t, db, hive.ID, "hv-a")

	first := tenant.FleetTask{BeeID: bee.ID, Kind: "fleet.host.drain", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &first); err != nil {
		t.Fatalf("first task: %v", err)
	}

	second := tenant.FleetTask{BeeID: bee.ID, Kind: "fleet.host.upgrade", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &second); !errors.Is(err, ErrBeeBusy) {
		t.Fatalf("queuing a second task: err = %v, want ErrBeeBusy", err)
	}

	// Once the first is terminal the bee is free again.
	if _, err := r.ReportTask(context.Background(), ReportTaskInput{
		BeeID: bee.ID, TaskID: first.ID, Status: tenant.FleetTaskSucceeded, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("report: %v", err)
	}
	if err := r.CreateTask(context.Background(), &second); err != nil {
		t.Fatalf("after the first finished, queuing another: %v", err)
	}
}

// A bee that crashed after claiming must be handed the SAME task back, not a
// new one and not nothing. The spec is desired state precisely so that
// re-running it is safe.
func TestClaimReturnsTheSameTaskToACrashedBee(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc3")
	bee := fleetSeedBee(t, db, hive.ID, "hv-b")

	task := tenant.FleetTask{BeeID: bee.ID, Kind: "fleet.host.drain", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &task); err != nil {
		t.Fatalf("create: %v", err)
	}

	first, err := r.ClaimNextTask(context.Background(), bee.ID, time.Minute)
	if err != nil || first == nil {
		t.Fatalf("first claim: %v (task=%v)", err, first)
	}
	again, err := r.ClaimNextTask(context.Background(), bee.ID, time.Minute)
	if err != nil || again == nil {
		t.Fatalf("second claim: %v (task=%v)", err, again)
	}
	if again.ID != first.ID {
		t.Fatalf("a re-claiming bee was handed a different task (%s then %s); the one it was "+
			"already running is now orphaned", first.ID, again.ID)
	}
	if again.Attempt < 2 {
		t.Fatalf("attempt = %d, want it to increment so a task that keeps being reclaimed is visible", again.Attempt)
	}
}

// A bee that finishes and dies before its report is acknowledged reports again
// on restart. Rejecting that would strand the task against a host that considers
// it done, and no further work could ever be issued to that host.
func TestTerminalReportsAreIdempotent(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc4")
	bee := fleetSeedBee(t, db, hive.ID, "hv-c")

	task := tenant.FleetTask{BeeID: bee.ID, Kind: "fleet.host.upgrade", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &task); err != nil {
		t.Fatalf("create: %v", err)
	}
	now := time.Now().UTC()

	if _, err := r.ReportTask(context.Background(), ReportTaskInput{
		BeeID: bee.ID, TaskID: task.ID, Status: tenant.FleetTaskSucceeded, Now: now,
	}); err != nil {
		t.Fatalf("first report: %v", err)
	}
	// The replay claims failure. The FIRST terminal outcome is the real one —
	// a late contradicting report must not rewrite history.
	out, err := r.ReportTask(context.Background(), ReportTaskInput{
		BeeID: bee.ID, TaskID: task.ID, Status: tenant.FleetTaskFailed, Error: "boom", Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("replayed report: %v, want it accepted so a retrying bee stops retrying", err)
	}
	if out.Status != tenant.FleetTaskSucceeded {
		t.Fatalf("status = %q, want it to stay %q; a replayed report rewrote a settled outcome",
			out.Status, tenant.FleetTaskSucceeded)
	}
}

// Under at-least-once delivery the same batch arrives twice. The second must be
// a no-op, and the count must say so — that is how a bee tells progress from
// retry.
func TestResentLogBatchesAreNoOps(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc5")
	bee := fleetSeedBee(t, db, hive.ID, "hv-d")

	task := tenant.FleetTask{BeeID: bee.ID, Kind: "fleet.host.drain", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &task); err != nil {
		t.Fatalf("create: %v", err)
	}

	now := time.Now().UTC()
	batch := []tenant.FleetTaskLog{
		{Seq: 1, Stream: "stdout", Message: "draining", LoggedAt: now, ReceivedAt: now},
		{Seq: 2, Stream: "stdout", Message: "draining", LoggedAt: now, ReceivedAt: now},
	}
	n, err := r.AppendTaskLogs(context.Background(), bee.ID, task.ID, append([]tenant.FleetTaskLog(nil), batch...))
	if err != nil || n != 2 {
		t.Fatalf("first batch: n=%d err=%v, want 2 new", n, err)
	}

	n, err = r.AppendTaskLogs(context.Background(), bee.ID, task.ID, append([]tenant.FleetTaskLog(nil), batch...))
	if err != nil {
		t.Fatalf("resent batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("resent batch accepted %d lines, want 0 — the stream now contains duplicates, "+
			"and an operator reading a failed rotation sees output that never happened twice", n)
	}

	// Two identical messages at different sequences are two real lines, not a
	// duplicate — which is why dedupe is on seq and not on content.
	var stored int64
	if err := db.Model(&tenant.FleetTaskLog{}).Where("task_id = ?", task.ID).Count(&stored).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if stored != 2 {
		t.Fatalf("stored %d lines, want 2", stored)
	}
}

// A bee may only ever touch its own task. The task id comes from the request, so
// this is the IDOR seam.
func TestABeeCannotReportOrLogAgainstAnotherBeesTask(t *testing.T) {
	db := fleetDB(t)
	r := NewFleetRepoGorm(db)
	hive := fleetSeedHive(t, db, "dc6")
	mine := fleetSeedBee(t, db, hive.ID, "hv-mine")
	theirs := fleetSeedBee(t, db, hive.ID, "hv-theirs")

	task := tenant.FleetTask{BeeID: theirs.ID, Kind: "fleet.host.drain", Status: tenant.FleetTaskPending, CreatedByID: uuid.New()}
	if err := r.CreateTask(context.Background(), &task); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := r.ReportTask(context.Background(), ReportTaskInput{
		BeeID: mine.ID, TaskID: task.ID, Status: tenant.FleetTaskFailed, Now: time.Now().UTC(),
	}); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("reporting on another bee's task: err = %v, want ErrTaskNotFound", err)
	}

	if _, err := r.AppendTaskLogs(context.Background(), mine.ID, task.ID, []tenant.FleetTaskLog{
		{Seq: 1, Message: "injected", LoggedAt: time.Now().UTC()},
	}); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("logging to another bee's task: err = %v, want ErrTaskNotFound", err)
	}
}
