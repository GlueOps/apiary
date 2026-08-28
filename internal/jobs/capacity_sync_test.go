package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var capacitySchema = testsupport.SchemaName("apiary_capacity_sync")

func capacityDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, capacitySchema,
		&tenant.Hive{}, &tenant.Hypervisor{}, &tenant.Bee{},
		&tenant.Swarm{}, &tenant.Slot{}, &tenant.Pool{}, &tenant.Placement{})
}

// machine builds a Hypervisor in a hive under a node name. HiveID and Name are
// POINTERS now that a machine record outlives its cluster membership, and a
// literal per test would be address-taking that says nothing about the test.
func machine(hiveID uuid.UUID, name string) tenant.Hypervisor {
	return tenant.Hypervisor{HiveID: &hiveID, Name: &name}
}

func syncDeps(db *gorm.DB, rows []tenant.ResourceRow) CapacitySyncDeps {
	return CapacitySyncDeps{
		Inventory: func(context.Context, uuid.UUID, uuid.UUID) ([]tenant.ResourceRow, error) {
			return rows, nil
		},
		TenantDB: func(context.Context, uuid.UUID) (*gorm.DB, error) { return db, nil },
	}
}

func TestSyncWritesObservedGuestsAlongsideFreshness(t *testing.T) {
	// The specific failure this guards: a sync that stamps last_synced_at while
	// leaving observed_guests at its zero default turns every departure gate from
	// "refuses, the observation is stale" into "passes, the node looks empty".
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30, DiskBytes: 500 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", CPUCores: 1, MemBytes: 1 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", CPUCores: 4, MemBytes: 8 << 30},
	}
	n, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 node written, got %d", n)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND name = ?", hive.ID, "hv1").Error; err != nil {
		t.Fatal(err)
	}
	if hv.ObservedGuests != 2 {
		t.Errorf("observed_guests: want 2 (stopped guest included), got %d", hv.ObservedGuests)
	}
	if hv.LastSyncedAt == nil {
		t.Fatal("last_synced_at must be stamped")
	}
	if hv.ObservedCPUTotal != 8 || hv.ObservedRAMGBTotal != 32 {
		t.Errorf("totals wrong: %+v", hv)
	}
	// The foreign columns ride in the SAME statement, for the same reason. This
	// hive has no placements at all, so every guest on the node belongs to
	// somebody else; leaving these at zero behind a fresh last_synced_at says
	// "nothing here but ours", and the oracle keeps filling the node.
	if hv.ObservedForeignCPUUsed != 5 || hv.ObservedForeignRAMGBUsed != 9 {
		t.Errorf("foreign usage must be written alongside the freshness stamp: want cpu=5 "+
			"ram=9, got %d/%d", hv.ObservedForeignCPUUsed, hv.ObservedForeignRAMGBUsed)
	}
	// Cordoned, and the observation still lands. Those two go together on
	// purpose: an operator decides whether a newly found host may receive
	// guests, and needs the figures in front of them to decide. The full rule,
	// including that a later sync must not re-cordon a host somebody enabled,
	// is TestSyncCreatesNewlyDiscoveredNodesCordoned.
	if hv.Schedulable {
		t.Error("a newly discovered node must arrive cordoned; nothing has validated this " +
			"host, and arrival already treats the same column that way")
	}
}

func TestSyncGatesAgreeWithWhatItWrote(t *testing.T) {
	// End to end through the model: a node carrying only a STOPPED guest must
	// come out of the sync as draining, never drained. This is the join failure
	// from the lab expressed as a unit test.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8], MinimumNodes: 1}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", CPUCores: 1, MemBytes: 1 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ?", hive.ID).Error; err != nil {
		t.Fatal(err)
	}
	hv.Schedulable = false // cordon it

	drain := tenant.DrainStateAt(hv.LastSyncedAt.Add(time.Second), hv, 0, tenant.DrainThresholds{})
	if drain.State != tenant.DrainStateDraining {
		t.Fatalf("a node with a stopped guest must be draining, not %s — a stopped guest blocks "+
			"a cluster join exactly as a running one does", drain.State)
	}
	refusals := tenant.CheckDeparture(tenant.DepartureRequest{
		Hive: hive, Hypervisor: hv, Drain: drain, HiveNodeCount: 1,
	})
	if len(refusals) == 0 {
		t.Fatal("the departure must be refused while a stopped guest remains")
	}
}

func TestSyncPreservesOperatorIntent(t *testing.T) {
	// Reserved capacity, the overcommit override and the cordon are operator
	// intent. A sync that reset them would un-cordon a host somebody drained,
	// silently re-admitting it to the placement query mid-departure.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	seed := machine(hive.ID, "hv1")
	seed.CPUReserved, seed.RAMGBReserved, seed.DiskGBReserved = 2, 4, 50
	seed.CPUOvercommitRatio = 2.5
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}
	// Cordon with an UPDATE. Create cannot store false here — see the trap
	// documented on Hypervisor.Schedulable — and an earlier version of this test
	// set it in the struct literal, cordoned nothing, and then blamed the sync.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", seed.ID).
		Update("schedulable", false).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16, MemBytes: 64 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "id = ?", seed.ID).Error; err != nil {
		t.Fatal(err)
	}
	if hv.Schedulable {
		t.Error("the sync must NOT un-cordon a drained host")
	}
	if hv.CPUReserved != 2 || hv.RAMGBReserved != 4 || hv.DiskGBReserved != 50 {
		t.Errorf("reserved capacity must survive a sync, got %+v", hv)
	}
	if hv.CPUOvercommitRatio != 2.5 {
		t.Errorf("the per-host overcommit override must survive, got %v", hv.CPUOvercommitRatio)
	}
	if hv.ObservedCPUTotal != 16 {
		t.Errorf("observed capacity must be refreshed, got %d", hv.ObservedCPUTotal)
	}
}

func TestSyncClearsClusterFieldsWithoutDestroyingTheMachine(t *testing.T) {
	// The INVERSION of what this used to assert, and the change that made
	// Hypervisor the durable machine record.
	//
	// Deleting the row was what made fleet membership self-healing, and it also
	// destroyed the machine — its identity, its address, its onboarding history —
	// every time a host left a cluster. Clearing the cluster-derived columns gives
	// the same visible result: no hive, no node name, no capacity, so it cannot be
	// a placement candidate and reads as ours-but-unclustered.
	//
	// The zeroed observations are the half that actually bites. A machine left
	// carrying stale totals with no cluster is one DrainStateAt reports real
	// capacity for, and last_synced_at left fresh over stale numbers is the exact
	// fail-open the departure gate exists to avoid.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	prod := tenant.Swarm{HiveID: hive.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	machineID := uuid.NewString()
	gone := machine(hive.ID, "departed")
	gone.SwarmID = &prod.ID
	gone.MachineID = &machineID
	gone.Address = "departed.example.com"
	gone.ObservedCPUTotal, gone.ObservedRAMGBTotal, gone.ObservedGuests = 16, 64, 3
	gone.ObservedCPUUsed, gone.ObservedRAMGBUsed, gone.ObservedDiskGBUsed = 4, 8, 20
	gone.ObservedForeignCPUUsed, gone.ObservedForeignRAMGBUsed, gone.ObservedForeignDiskGBUsed = 3, 6, 15
	gone.ObservedDiskGBTotal = 500
	gone.CPUReserved = 2
	if err := db.Create(&gone).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "id = ?", gone.ID).Error; err != nil {
		t.Fatalf("the machine record was DELETED when its cluster stopped reporting it: %v — "+
			"its machine id, address and onboarding history went with it, and there is now no "+
			"row for a machine we still own", err)
	}

	if hv.HiveID != nil {
		t.Errorf("hive_id must be cleared, got %s", *hv.HiveID)
	}
	if hv.SwarmID != nil {
		t.Error("swarm_id must be cleared with hive_id: a swarm is a subdivision of a cluster " +
			"the machine has left, and ck_hypervisors_swarm_needs_hive forbids the combination")
	}
	if hv.Name != nil {
		t.Errorf("name must be cleared, got %q — another hive's sync could otherwise adopt "+
			"this machine by a name it no longer answers to", *hv.Name)
	}
	if hv.LastSyncedAt != nil {
		t.Error("last_synced_at must be cleared: there is no current observation of a node the " +
			"cluster is not reporting, and a fresh stamp over stale numbers is what makes the " +
			"departure gate pass on a host it cannot see")
	}
	if hv.ObservedCPUTotal != 0 || hv.ObservedRAMGBTotal != 0 || hv.ObservedDiskGBTotal != 0 ||
		hv.ObservedCPUUsed != 0 || hv.ObservedRAMGBUsed != 0 || hv.ObservedDiskGBUsed != 0 ||
		hv.ObservedGuests != 0 || hv.ObservedForeignCPUUsed != 0 ||
		hv.ObservedForeignRAMGBUsed != 0 || hv.ObservedForeignDiskGBUsed != 0 {
		t.Errorf("every observation must be zeroed, got %+v — a machine left carrying a "+
			"foreign load with no cluster reads as permanently oversubscribed, which is the "+
			"opposite mistake to the stale totals but still a wrong answer about a host "+
			"nothing can schedule onto", hv)
	}

	// What must SURVIVE. This is the whole reason the row is kept.
	if hv.MachineID == nil || *hv.MachineID != machineID {
		t.Error("the machine id must survive: it is what makes a re-enrolment find this record " +
			"instead of creating a second one")
	}
	if hv.Address != "departed.example.com" {
		t.Errorf("the address must survive, got %q — it is how anything reaches the box", hv.Address)
	}
	if hv.ProvisioningState != tenant.OnboardProvisioned {
		t.Errorf("provisioning_state must survive, got %q; resetting it would put the "+
			"onboarding ticker back onto a machine that is already installed", hv.ProvisioningState)
	}
	if hv.CPUReserved != 2 {
		t.Errorf("operator intent must survive, got cpu_reserved=%d", hv.CPUReserved)
	}
}

func TestSyncPreservesProvisioningState(t *testing.T) {
	// The onboarding axis is not the sync's to write. A sync that reset a working
	// host to 'pending' would hand it back to the onboarding ticker, which would
	// SSH into metal that is already running guests and reinstall the agent under
	// it.
	//
	// Named-column updates are what make this true by construction, so the test
	// is here to catch the day somebody replaces one with a struct save.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	seed := machine(hive.ID, "hv1")
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}
	// Through an UPDATE, not the struct: provisioning_state carries a `default:`
	// tag, so Create cannot write any value GORM considers zero — and 'failed' is
	// not zero, but 'pending' would be reachable only this way if it ever were.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", seed.ID).
		Updates(map[string]any{
			"provisioning_state":   tenant.OnboardFailed,
			"provisioning_attempt": 3,
			"provisioning_error":   "ssh: handshake failed",
		}).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "id = ?", seed.ID).Error; err != nil {
		t.Fatal(err)
	}
	if hv.ProvisioningState != tenant.OnboardFailed {
		t.Errorf("provisioning_state = %q, want it untouched at %q", hv.ProvisioningState, tenant.OnboardFailed)
	}
	if hv.ProvisioningAttempt != 3 {
		t.Errorf("provisioning_attempt = %d, want 3: resetting the counter would let an "+
			"unreachable host generate SSH attempts forever", hv.ProvisioningAttempt)
	}
	if hv.ProvisioningError == "" {
		t.Error("the last failure must survive, or 'failed' is not actionable without the logs")
	}
}

// The capacity sync's half of the merge: a machine onboarded by address and only
// LATER discovered in a cluster under a node name must end as ONE row.
//
// The sync knows only (hive, name) — /cluster/resources reports no machine id —
// so matching on that alone finds nothing for a machine whose hive_id is still
// NULL, and creates a second record beside the one holding all the onboarding
// history.
func TestOnboardedFirstDiscoveredSecondIsOneRow(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	machineID := uuid.NewString()
	onboarded := tenant.Hypervisor{
		MachineID: &machineID,
		Address:   "hv9.dc1.example.com",
	}
	if err := db.Create(&onboarded).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv9", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var members []tenant.Hypervisor
	if err := db.Where("hive_id = ?", hive.ID).Find(&members).Error; err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("%d rows in this hive for one physical machine, want 1: the onboarding "+
			"history is on one row and the capacity on another, and neither is the machine",
			len(members))
	}
	hv := members[0]
	if hv.ID != onboarded.ID {
		t.Fatalf("the sync created a new record (%s) instead of adopting the onboarded one "+
			"(%s)", hv.ID, onboarded.ID)
	}
	if hv.MachineID == nil || *hv.MachineID != machineID {
		t.Error("adoption lost the machine id, which is what makes re-enrolment idempotent")
	}
	if hv.Address != "hv9.dc1.example.com" {
		t.Errorf("adoption lost the address, got %q", hv.Address)
	}
	if hv.Name == nil || *hv.Name != "hv9" {
		t.Error("the adopted row must carry the cluster's node name, or the NEXT sync will " +
			"fail to match it and duplicate it after all")
	}
	if hv.ObservedCPUTotal != 8 {
		t.Errorf("the adopted row must also receive this sync's observations, got %d", hv.ObservedCPUTotal)
	}
}

// Adoption REFUSES when it cannot tell which machine it is looking at. Two
// unclustered machines whose addresses share a first label give no evidence which
// one joined, and attaching a cluster's capacity to the wrong physical host is
// worse than a duplicate an operator can merge.
func TestSyncRefusesToAdoptAnAmbiguousAddress(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	for _, addr := range []string{"hv10.dc1.example.com", "hv10.dc2.example.com"} {
		id := uuid.NewString()
		if err := db.Create(&tenant.Hypervisor{MachineID: &id, Address: addr}).Error; err != nil {
			t.Fatal(err)
		}
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv10", Status: "online", CPUCores: 8},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var adopted int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ? AND machine_id IS NOT NULL", hive.ID).Count(&adopted).Error; err != nil {
		t.Fatal(err)
	}
	if adopted != 0 {
		t.Fatal("the sync adopted one of two equally plausible machines; a fleet operation " +
			"would now act on the wrong physical host")
	}
}

func TestSyncDoesNotInventNodesFromGuestRowsAlone(t *testing.T) {
	// A guest naming an undescribed node is folded so it is not lost, but it must
	// not CREATE a hypervisor row — that row would carry zero capacity and read
	// as a full host to the oracle.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8},
		{Type: "qemu", Node: "hv-ghost", Status: "running", CPUCores: 4},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	// "No row with that name IN THIS HIVE" rather than "no row at all": the name
	// column is nullable now and shared across the fixture schema, so the
	// unqualified question is no longer the one being asked.
	var n int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ? AND name = ?", hive.ID, "hv-ghost").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("a guest row alone must not create a hypervisor row")
	}
}

func TestCordonCannotBeSetOnCreate(t *testing.T) {
	// Pins a GORM behaviour the cordon depends on, so that a future change to the
	// `default:true` tag surfaces here rather than as a host that quietly keeps
	// accepting placements after somebody cordoned it.
	//
	// If this test starts failing because Create now stores false, that is an
	// IMPROVEMENT — delete the test and the trap comment on Hypervisor.Schedulable
	// together. It is failing here to tell you the workaround is no longer needed,
	// not to tell you something broke.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	h := machine(hive.ID, "created-cordoned")
	h.Schedulable = false
	if err := db.Create(&h).Error; err != nil {
		t.Fatal(err)
	}
	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", h.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !got.Schedulable {
		t.Skip("Create now stores false — the trap is gone; remove this test and the comment " +
			"on Hypervisor.Schedulable")
	}

	// And the path the code actually uses.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", h.ID).
		Update("schedulable", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&got, "id = ?", h.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Schedulable {
		t.Fatal("an explicit Update must cordon the host")
	}
}

func TestSyncUpdatesObservedGuestsOnAnExistingNode(t *testing.T) {
	// The UPDATE path, which is the one that runs on every sync after the first —
	// so in practice it is the only one that matters. An earlier version of these
	// tests covered only the create path, and a mutation zeroing the guest count
	// on update survived: every already-known host would have reported empty
	// forever, and every departure gate would have passed on it.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	// First sync discovers the node with no guests.
	first := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, first), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}
	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND name = ?", hive.ID, "hv1").Error; err != nil {
		t.Fatal(err)
	}
	if hv.ObservedGuests != 0 {
		t.Fatalf("the node starts empty, got %d", hv.ObservedGuests)
	}

	// Guests appear. The second sync must reflect that on the EXISTING row.
	second := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", CPUCores: 4, MemBytes: 8 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", CPUCores: 1, MemBytes: 1 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, second), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&hv, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if hv.ObservedGuests != 2 {
		t.Fatalf("the update path must write the guest count: want 2, got %d", hv.ObservedGuests)
	}
	if hv.ObservedCPUUsed != 5 {
		t.Errorf("allocated usage must update too: want 5, got %d", hv.ObservedCPUUsed)
	}
	// The UPDATE path's own copy of the foreign columns. Nothing is placed in
	// this hive, so both guests are strangers' — and a mutation that dropped
	// these three from the named-column update would leave every already-known
	// host reporting no foreign load forever, which is precisely the shape of the
	// bug being fixed.
	if hv.ObservedForeignCPUUsed != 5 || hv.ObservedForeignRAMGBUsed != 9 {
		t.Fatalf("the update path must write the foreign columns: want cpu=5 ram=9, got %d/%d",
			hv.ObservedForeignCPUUsed, hv.ObservedForeignRAMGBUsed)
	}

	// And back down again when they go away — a count that only ever grows would
	// pin a node as undrainable forever.
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, first), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&hv, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if hv.ObservedGuests != 0 {
		t.Fatalf("the guest count must fall as well as rise: want 0, got %d", hv.ObservedGuests)
	}
	if hv.ObservedForeignCPUUsed != 0 || hv.ObservedForeignRAMGBUsed != 0 {
		t.Fatalf("foreign usage must fall too, or a node reads as permanently oversubscribed "+
			"long after the guests left: got %d/%d",
			hv.ObservedForeignCPUUsed, hv.ObservedForeignRAMGBUsed)
	}
}

func TestSyncPreservesSwarmMembership(t *testing.T) {
	// Swarm assignment is operator intent, exactly like the cordon and reserved
	// capacity. A sync that dropped it would take a hypervisor out of fleet-prod
	// on the next tick, which the placement oracle would read as capacity
	// vanishing for no reason anybody could point at.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	prod := tenant.Swarm{HiveID: hive.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	seed := machine(hive.ID, "hv1")
	seed.SwarmID = &prod.ID
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16, MemBytes: 64 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "id = ?", seed.ID).Error; err != nil {
		t.Fatal(err)
	}
	if hv.SwarmID == nil || *hv.SwarmID != prod.ID {
		t.Fatal("the sync must not drop swarm membership")
	}
	if hv.ObservedCPUTotal != 16 {
		t.Errorf("observed capacity must still refresh, got %d", hv.ObservedCPUTotal)
	}
}

func TestSyncDiscoversNewHypervisorsAsSpare(t *testing.T) {
	// A newly discovered hypervisor belongs to no swarm. It is in the cluster and
	// healthy, but nobody has promised it to anything — which is the state spare
	// capacity is, and it must not be a placement candidate until assigned.
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	rows := []tenant.ResourceRow{
		{Type: "node", Node: "fresh", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND name = ?", hive.ID, "fresh").Error; err != nil {
		t.Fatal(err)
	}
	if hv.SwarmID != nil {
		t.Fatal("a newly discovered hypervisor must be spare, not silently placed in a swarm")
	}
	if tenant.MembershipOf(hv) != tenant.SwarmMembershipSpare {
		t.Error("it must read as spare")
	}
	// A node nobody onboarded through this system. It has no machine id — nothing
	// has ever read one off it — and it is 'provisioned' because it is plainly a
	// working member of a cluster, not something the onboarding ticker should go
	// and SSH into.
	if hv.MachineID != nil {
		t.Errorf("a discovered node must carry no machine id, got %q; inventing one would "+
			"make a later real enrolment look like a different machine", *hv.MachineID)
	}
	if hv.ProvisioningState != tenant.OnboardProvisioned {
		t.Errorf("provisioning_state = %q, want %q: defaulting a discovered node to pending "+
			"enqueues an SSH onboarding run against every hypervisor in every tenant on the "+
			"first tick after deploy", hv.ProvisioningState, tenant.OnboardProvisioned)
	}
}

// placedGuest writes one placement into the ledger at a given state and VMID.
//
// The state goes in through an UPDATE rather than the struct literal. State
// carries a `default:` tag, so GORM omits it at its zero value and the column
// default applies — the same trap documented at length on Hypervisor.Schedulable.
// Every state used here happens to be non-zero, but a test whose fixture depends
// on that is one rename away from silently seeding 'pending' everywhere and
// asserting nothing.
func placedGuest(t *testing.T, db *gorm.DB, hiveID, poolID uuid.UUID, hvID uuid.UUID, vmid int, state string) {
	t.Helper()
	p := tenant.Placement{PoolID: poolID, HypervisorID: hvID, HiveID: hiveID, VMID: &vmid}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Placement{}).Where("id = ?", p.ID).
		Update("state", state).Error; err != nil {
		t.Fatal(err)
	}
}

// ledgerFixture builds the minimum needed to write placements: a hive, a swarm,
// a slot, a pool and a discovered hypervisor.
func ledgerFixture(t *testing.T, db *gorm.DB) (tenant.Hive, tenant.Pool, tenant.Hypervisor) {
	t.Helper()
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	swarm := tenant.Swarm{HiveID: hive.ID, Name: "prod"}
	if err := db.Create(&swarm).Error; err != nil {
		t.Fatal(err)
	}
	slot := tenant.Slot{Name: "medium-" + uuid.NewString()[:8], VCPU: 4, RAMGB: 8, DiskGB: 50}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatal(err)
	}
	pool := tenant.Pool{HiveID: hive.ID, SwarmID: swarm.ID, SlotID: slot.ID, Name: "workers"}
	if err := db.Create(&pool).Error; err != nil {
		t.Fatal(err)
	}
	hv := machine(hive.ID, "hv1")
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}
	return hive, pool, hv
}

func TestSyncSubtractsGuestsThisSystemDidNotPlace(t *testing.T) {
	// THE BUG, end to end through the database. A node carrying one guest we
	// placed and one somebody made by hand must report the hand-made one as
	// foreign — and only that one. Before this, foreign guests were counted zero
	// times and a hypervisor full of them read as empty to the oracle.
	db := capacityDB(t)
	hive, pool, hv := ledgerFixture(t, db)
	placedGuest(t, db, hive.ID, pool.ID, hv.ID, 101, tenant.PlacementActive)

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 32, MemBytes: 128 << 30, DiskBytes: 1000 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 101, CPUCores: 4, MemBytes: 8 << 30, DiskBytes: 50 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 900, CPUCores: 8, MemBytes: 16 << 30, DiskBytes: 100 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedForeignCPUUsed != 8 || got.ObservedForeignRAMGBUsed != 16 || got.ObservedForeignDiskGBUsed != 100 {
		t.Fatalf("foreign usage must be the hand-made guest alone: want 8/16/100, got %d/%d/%d",
			got.ObservedForeignCPUUsed, got.ObservedForeignRAMGBUsed, got.ObservedForeignDiskGBUsed)
	}
	// The totals stay totals. Drift and the fleet view read them, and ObservedGuests
	// is what the departure and arrival gates count — all three need every guest.
	if got.ObservedCPUUsed != 12 || got.ObservedGuests != 2 {
		t.Errorf("the observed totals must still cover every guest, got cpu=%d guests=%d",
			got.ObservedCPUUsed, got.ObservedGuests)
	}

	// And the arithmetic the oracle will do: 32 cores, 8 held by a stranger, 4
	// committed by our own placement. Each guest counted ONCE.
	c := tenant.CandidateFor(hive, got, tenant.Capacity{CPU: 4, RAMGB: 8, DiskGB: 50}, 0)
	if c.Remaining.CPU != 20 {
		t.Fatalf("remaining cpu: want 20 (32 - 8 foreign - 4 intended), got %d — 24 means the "+
			"stranger's guest is invisible, 16 means ours was subtracted twice", c.Remaining.CPU)
	}
	if c.Remaining.RAMGB != 104 || c.Remaining.DiskGB != 850 {
		t.Errorf("remaining ram/disk: want 104/850, got %d/%d", c.Remaining.RAMGB, c.Remaining.DiskGB)
	}
}

func TestSyncExcludesOnlyPlacementsThatStillHoldCapacity(t *testing.T) {
	// THE ONE PLACE THIS MUST NOT COPY waggle, which excludes every placement row
	// carrying a vmid. Here a released or failed placement contributes nothing to
	// intended usage — see PlacementHoldsCapacity — so if its VM is still on the
	// node (leaked, or a deletion that failed) then excluding its VMID would
	// count that guest ZERO times and rebuild the exact bug this closes.
	db := capacityDB(t)
	hive, pool, hv := ledgerFixture(t, db)
	placedGuest(t, db, hive.ID, pool.ID, hv.ID, 101, tenant.PlacementActive)
	placedGuest(t, db, hive.ID, pool.ID, hv.ID, 102, tenant.PlacementPending)
	placedGuest(t, db, hive.ID, pool.ID, hv.ID, 800, tenant.PlacementReleased)
	placedGuest(t, db, hive.ID, pool.ID, hv.ID, 801, tenant.PlacementFailed)

	// Every one of those VMs is still on the node as far as the provider is
	// concerned: two of ours, and two whose placements were released or failed
	// without the guest actually going away.
	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 64, MemBytes: 256 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 101, CPUCores: 1},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 102, CPUCores: 2},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 800, CPUCores: 4},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 801, CPUCores: 8},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedForeignCPUUsed != 12 {
		t.Fatalf("foreign cpu: want 12 (the released VM's 4 plus the failed one's 8), got %d — "+
			"3 or 15 means the state filter is gone and a leaked guest is being counted "+
			"nowhere at all, because a released placement is not in intended either",
			got.ObservedForeignCPUUsed)
	}
}

func TestSyncTreatsAnotherHivesPlacementsAsForeign(t *testing.T) {
	// The ledger query is scoped to THIS hive. A VMID is unique inside a Proxmox
	// cluster and nowhere else, so an unscoped query would let a placement in
	// dc2 hide a hand-made VM that happens to share its id in dc1.
	db := capacityDB(t)
	hive, _, hv := ledgerFixture(t, db)
	other, otherPool, otherHV := ledgerFixture(t, db)
	placedGuest(t, db, other.ID, otherPool.ID, otherHV.ID, 700, tenant.PlacementActive)

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 32},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 700, CPUCores: 6},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedForeignCPUUsed != 6 {
		t.Fatalf("a VMID matching ANOTHER hive's placement is still a stranger here: want "+
			"foreign cpu=6, got %d", got.ObservedForeignCPUUsed)
	}
}

func TestPlacementStatesHoldingCapacityFollowsThePredicate(t *testing.T) {
	// This list becomes SQL, and SQL cannot call the Go predicate that defines
	// the rule. Deriving it is what stops a change of mind about which states
	// hold capacity from reaching intended usage and silently not the exclude
	// set — which would count a guest twice, or not at all.
	states := placementStatesHoldingCapacity()
	if len(states) == 0 {
		t.Fatal("no state holds capacity, so nothing would ever be excluded and every guest " +
			"we placed would be subtracted twice")
	}
	for _, s := range states {
		if !tenant.PlacementHoldsCapacity(s) {
			t.Errorf("%q is in the exclude set but does not hold capacity: its guest would be "+
				"counted nowhere at all", s)
		}
	}
	for _, s := range []string{tenant.PlacementPending, tenant.PlacementActive,
		tenant.PlacementFailed, tenant.PlacementReleased} {
		var listed bool
		for _, got := range states {
			listed = listed || got == s
		}
		if listed != tenant.PlacementHoldsCapacity(s) {
			t.Errorf("state %q: listed=%v but PlacementHoldsCapacity=%v — the exclude set and "+
				"intended usage must be the same set of rows by the same definition",
				s, listed, tenant.PlacementHoldsCapacity(s))
		}
	}
}

// A node the control plane has just met is not schedulable, and the operator's
// later decision survives every sync after that.
//
// Same column and same rule as arrival: a host receives guests when somebody
// says it may, having seen what was observed on it. The create path used to
// write true, so the identical machine ended up in opposite states depending on
// whether the control plane met it by MOVING it or by FINDING it — and the
// discovered case is the weaker one, since nothing has validated that host at
// all.
//
// Both directions are checked because the change is only on the create path. If
// it ever spread to the update path it would re-cordon a host an operator had
// deliberately enabled, on the next tick, for the life of the cluster — a far
// worse failure than the one being fixed, and one that looks like the fleet
// randomly refusing work.
func TestSyncCreatesNewlyDiscoveredNodesCordoned(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "hv-new", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND name = ?", hive.ID, "hv-new").Error; err != nil {
		t.Fatalf("the sync did not create the discovered node: %v", err)
	}
	if hv.Schedulable {
		t.Fatal("a newly discovered node is schedulable; the placement oracle can land guests " +
			"on a host nobody has looked at, which is the opposite of what arrival does with " +
			"the same column")
	}
	// The observation still lands — cordoning must not cost the figures an
	// operator needs in order to decide.
	if hv.ObservedCPUTotal != 8 || hv.ObservedRAMGBTotal != 32 {
		t.Errorf("observed capacity must be recorded on create, got cpu=%d ram=%d",
			hv.ObservedCPUTotal, hv.ObservedRAMGBTotal)
	}

	// The operator enables it, and that decision must outlive every later sync.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("schedulable", true).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := SyncHiveCapacity(context.Background(), syncDeps(db, rows), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&hv, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !hv.Schedulable {
		t.Fatal("a later sync re-cordoned a host the operator had enabled; the cordon rule " +
			"belongs to the create path alone, or the fleet refuses work at every tick")
	}
}

// syncDepsWithAddresses is syncDeps plus the /cluster/status lookup.
func syncDepsWithAddresses(db *gorm.DB, rows []tenant.ResourceRow,
	addrs map[string]string) CapacitySyncDeps {
	d := syncDeps(db, rows)
	d.NodeAddresses = func(context.Context, uuid.UUID, uuid.UUID) (map[string]string, error) {
		return addrs, nil
	}
	return d
}

// A discovered node must be recorded with the address the cluster knows it by.
//
// Without it the row is a name and nothing to reach the host on, and onboarding
// — which looks rows up by ADDRESS — creates a second row for the same machine.
// Observed on a live four-node cluster: pve4 and pve5 each ended up with a
// name-only row from the sync and an address-only row from a later declaration.
func TestSyncRecordsTheAddressForADiscoveredNode(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "pve4", Status: "online", CPUCores: 4, MemBytes: 8 << 30},
	}
	addrs := map[string]string{"pve4": "10.10.10.5"}
	if _, err := SyncHiveCapacity(context.Background(),
		syncDepsWithAddresses(db, rows, addrs), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var got tenant.Hypervisor
	if err := db.Where("hive_id = ?", hive.ID).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Address != "10.10.10.5" {
		t.Fatalf("a discovered node must carry the address the cluster knows it by; got %q",
			got.Address)
	}
}

// The address is BACKFILLED onto a record that has none, and never overwritten.
//
// Backfilled because rows discovered before this lookup existed — or before the
// hive had a credential to read it with — are otherwise stuck without one
// forever. Never overwritten because an onboarding run records the address this
// control plane actually reaches the host on, which is not always the address
// corosync knows it by; replacing it would silently repoint every SSH the fleet
// makes to that host.
func TestSyncBackfillsAnAddressButNeverReplacesOne(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	blank := machine(hive.ID, "pve4")
	operatorSet := machine(hive.ID, "pve5")
	operatorSet.Address = "mgmt-pve5.example.com"
	if err := db.Create(&blank).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&operatorSet).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "pve4", Status: "online", CPUCores: 4, MemBytes: 8 << 30},
		{Type: "node", Node: "pve5", Status: "online", CPUCores: 4, MemBytes: 8 << 30},
	}
	addrs := map[string]string{"pve4": "10.10.10.5", "pve5": "10.10.10.6"}
	if _, err := SyncHiveCapacity(context.Background(),
		syncDepsWithAddresses(db, rows, addrs), uuid.New(), hive.ID); err != nil {
		t.Fatal(err)
	}

	var filled, kept tenant.Hypervisor
	if err := db.First(&filled, "id = ?", blank.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&kept, "id = ?", operatorSet.ID).Error; err != nil {
		t.Fatal(err)
	}
	if filled.Address != "10.10.10.5" {
		t.Fatalf("an empty address must be backfilled; got %q", filled.Address)
	}
	if kept.Address != "mgmt-pve5.example.com" {
		t.Fatalf("an address already recorded must survive the sync; got %q", kept.Address)
	}
}

// A hive whose status call fails still syncs. Capacity is what this job exists
// to refresh; failing the whole run because an optional field could not be
// filled would trade a cosmetic gap for a real one.
func TestSyncStillWritesCapacityWhenAddressesCannotBeRead(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}

	rows := []tenant.ResourceRow{
		{Type: "node", Node: "pve4", Status: "online", CPUCores: 4, MemBytes: 8 << 30},
	}
	d := syncDeps(db, rows)
	d.NodeAddresses = func(context.Context, uuid.UUID, uuid.UUID) (map[string]string, error) {
		return nil, fmt.Errorf("the hive API is unreachable")
	}
	n, err := SyncHiveCapacity(context.Background(), d, uuid.New(), hive.ID)
	if err != nil {
		t.Fatalf("a failed address lookup must not fail the sync: %v", err)
	}
	if n != 1 {
		t.Fatalf("the node must still be written; wrote %d", n)
	}
}

// THE CLOSED LOOP THIS OPENS.
//
// A standalone Proxmox host answers /cluster/resources with itself, so the sync
// recorded it as the hive's only member. That one row then closed every route
// out: bootstrap refused because the host was in a hive AND the hive was not
// empty, while arrival was permitted because the target had a node — right up to
// the point where `pvecm add` was pointed at a machine that is not a cluster.
//
// Observed on a five-node lab where nothing was clustered: two hosts recorded as
// members of two clusters that did not exist, and no way to create either.
func TestSyncRefusesAHiveWhoseEndpointIsAStandaloneHost(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	// The false membership a previous sync left behind.
	hv := machine(hive.ID, "pve")
	hv.Address = "192.0.2.11"
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "pve", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	deps.ClusterIdentity = func(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
		// What a standalone host reports: a successful read, and no cluster.
		return "", false, nil
	}

	_, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID)
	if !errors.Is(err, ErrHiveIsNotACluster) {
		t.Fatalf("err = %v, want ErrHiveIsNotACluster", err)
	}
	// And it must say what to do, or an operator is told no and nothing else.
	if !strings.Contains(err.Error(), "Create the cluster") {
		t.Fatalf("the refusal names no way forward: %v", err)
	}

	// The false membership is gone, so the host can now be used to CREATE the
	// cluster — which is the whole point of noticing.
	var after tenant.Hypervisor
	if err := db.First(&after, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.HiveID != nil {
		t.Fatalf("the host is still recorded in a cluster that does not exist (hive %v)", *after.HiveID)
	}
	if after.Name != nil {
		t.Fatalf("the node name survived, and it is a name only a cluster could have given: %q", *after.Name)
	}
	// The MACHINE survives — this corrects bookkeeping, it does not forget a box.
	if after.Address != "192.0.2.11" {
		t.Fatalf("address = %q, want the machine record left intact", after.Address)
	}
}

// A real cluster must be entirely unaffected, including one that has lost
// quorum: a clustered node still reports its cluster entry when it cannot vote,
// and treating that as standalone would release every member of a cluster during
// exactly the incident where the records matter most.
func TestSyncLeavesARealClusterAlone(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]tenant.Hypervisor{machine(hive.ID, "hv1")}[0]).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	deps.ClusterIdentity = func(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
		return "dc1", true, nil
	}

	n, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID)
	if err != nil {
		t.Fatalf("a real cluster was refused: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 node written, got %d", n)
	}

	var count int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", hive.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("%d members after a sync of a real cluster, want 1", count)
	}
}

// An endpoint that cannot be reached is NOT a standalone host. Mistaking the two
// would release a whole cluster's membership on a network blip.
func TestSyncDoesNotReleaseMembersWhenTheClusterCheckFails(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]tenant.Hypervisor{machine(hive.ID, "hv1")}[0]).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	deps.ClusterIdentity = func(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
		return "", false, errors.New("dial tcp 192.0.2.1:8006: i/o timeout")
	}

	if _, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID); err != nil {
		t.Fatalf("an unreachable check must not fail the sync on its own: %v", err)
	}
	var count int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", hive.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("membership was released on a failed check; %d members remain, want 1", count)
	}
}

// THE CREDENTIAL BREAK PATH. A hive's API URL names ONE node, and the credential
// is bound to it. When that node departs, the address still points at it and it
// now answers as standalone — while the cluster it left is running, quorate and
// full of members.
//
// Releasing on that answer would mark every remaining member fleetless: a live
// fleet erased from the control plane's records because one address went stale.
// The standalone answer only ever proved that the machine at the API URL is in
// no cluster; it never proved anything about the others.
func TestSyncDoesNotReleaseALiveClusterWhenItsApiUrlPointsAtADepartedNode(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	// Three members: the departed node's address is what the hive still points
	// at, and two nodes that are genuinely still in the cluster.
	for _, n := range []string{"hv1", "hv2", "hv3"} {
		m := machine(hive.ID, n)
		if err := db.Create(&m).Error; err != nil {
			t.Fatal(err)
		}
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "hv2", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	deps.ClusterIdentity = func(context.Context, uuid.UUID, uuid.UUID) (string, bool, error) {
		// What the departed node reports: a clean read, and no cluster.
		return "", false, nil
	}

	_, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID)
	if !errors.Is(err, ErrHiveIsNotACluster) {
		t.Fatalf("err = %v, want ErrHiveIsNotACluster", err)
	}
	// It must name the actual problem — a stale address, not a vanished cluster —
	// because the remedy is to repoint it, and the other message sends an
	// operator to create a cluster that already exists.
	if !strings.Contains(err.Error(), "Repoint") {
		t.Fatalf("the refusal does not say to repoint the API URL: %v", err)
	}

	var still int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", hive.ID).Count(&still).Error; err != nil {
		t.Fatal(err)
	}
	if still != 3 {
		t.Fatalf("%d members survived, want all 3 — a stale API URL must never erase a live "+
			"cluster's records", still)
	}
}

// A RECORDED FAILURE MUST NOT OUTLIVE ITS GOAL.
//
// last_operation_error stores the outcome of the last attempt, and an operator
// reads it as the state of the machine. When membership is resolved by a path
// that records no outcome of its own — adoption, or the sync clearing a node the
// cluster stopped reporting — the failure stands unchallenged forever. Three
// healthy nodes carried an eight-hour-old join failure that way, and surfacing
// last_operation_error in the UI turned a dormant inaccuracy into a visible one.
func TestAdoptionClearsAStaleJoinFailure(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	// A fleetless machine carrying the failure of a join that later happened
	// anyway, by a route that recorded nothing.
	hv := tenant.Hypervisor{
		Address:            "192.0.2.30",
		LastOperation:      "join",
		LastOperationError: "bootstrap: refused by a precondition: already_in_hive",
	}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "pve9", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	deps.NodeAddresses = func(context.Context, uuid.UUID, uuid.UUID) (map[string]string, error) {
		return map[string]string{"pve9": "192.0.2.30"}, nil
	}
	if _, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID); err != nil {
		t.Fatalf("sync: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.HiveID == nil {
		t.Fatal("the machine was not adopted, so the rest of this test proves nothing")
	}
	if got.LastOperationError != "" {
		t.Fatalf("a join failure survived the join succeeding: %q", got.LastOperationError)
	}
}

// The same in the other direction: a machine the cluster no longer reports is
// out, which is what a departure was for, so a stored departure failure now
// contradicts the world.
func TestClearingMembershipClearsAStaleDepartureFailure(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	hv := machine(hive.ID, "pve8")
	hv.LastOperation = "depart"
	hv.LastOperationError = "depart: refused by a precondition: not_a_member"
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	// The cluster reports a different node entirely: pve8 is gone.
	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "pve7", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	if _, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID); err != nil {
		t.Fatalf("sync: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.HiveID != nil {
		t.Fatal("membership was not cleared, so the rest of this test proves nothing")
	}
	if got.LastOperationError != "" {
		t.Fatalf("a departure failure survived the machine leaving: %q", got.LastOperationError)
	}
}

// The STEADY-STATE case. Adoption covers a machine arriving and the
// clear-not-delete path covers one leaving; a node that was already a member
// when its failure was recorded is neither, and nothing ever contradicts the
// stored outcome. One sat in a cluster for eight hours carrying the refusal of a
// bootstrap it had supposedly failed.
func TestAnObservedMemberLosesAStaleJoinFailure(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	hv := machine(hive.ID, "pve6")
	hv.LastOperation = "join"
	hv.LastOperationError = "bootstrap: refused by a precondition: already_in_hive"
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "pve6", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	if _, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID); err != nil {
		t.Fatalf("sync: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.LastOperationError != "" {
		t.Fatalf("a join failure survived on a node observed in the cluster: %q",
			got.LastOperationError)
	}
}

// A failed DEPARTURE on a node that is still a member is accurate: it tried to
// leave and did not. Clearing it would erase a true statement about a machine an
// operator is probably still waiting on.
func TestAnObservedMemberKeepsAFailedDeparture(t *testing.T) {
	db := capacityDB(t)
	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	hv := machine(hive.ID, "pve5")
	hv.LastOperation = "depart"
	hv.LastOperationError = "depart: refused by a precondition: drain_unknown"
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	deps := syncDeps(db, []tenant.ResourceRow{
		{Type: "node", Node: "pve5", Status: "online", CPUCores: 8, MemBytes: 32 << 30},
	})
	if _, err := SyncHiveCapacity(context.Background(), deps, uuid.New(), hive.ID); err != nil {
		t.Fatalf("sync: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.LastOperationError == "" {
		t.Fatal("a departure that genuinely failed was erased while the node is still a member")
	}
}
