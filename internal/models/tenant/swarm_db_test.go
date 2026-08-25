package tenant_test

import (
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var swarmSchema = testsupport.SchemaName("apiary_swarm")

func swarmDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, swarmSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{})
}

func newHive(t *testing.T, db *gorm.DB, name string) tenant.Hive {
	t.Helper()
	h := tenant.Hive{Name: name + "-" + uuid.NewString()[:8]}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}
	return h
}

// machine builds a Hypervisor in a hive under a node name.
//
// A helper because HiveID and Name are POINTERS now that a machine record
// outlives its cluster membership, and a literal per test would be five lines of
// address-taking that says nothing about what the test is checking.
func machine(hiveID uuid.UUID, name string) tenant.Hypervisor {
	return tenant.Hypervisor{HiveID: &hiveID, Name: &name}
}

func TestSwarmMembershipCannotCrossHives(t *testing.T) {
	// The rule the composite foreign key exists for. A plain FK to swarms(id)
	// would happily let a hypervisor in dc1 join a swarm in dc2 — which reads as
	// a grouping and schedules as a lie, because those two machines cannot take
	// each other's work.
	db := swarmDB(t)
	dc1, dc2 := newHive(t, db, "dc1"), newHive(t, db, "dc2")

	prodInDC2 := tenant.Swarm{HiveID: dc2.ID, Name: "prod"}
	if err := db.Create(&prodInDC2).Error; err != nil {
		t.Fatalf("create swarm: %v", err)
	}

	crossHive := machine(dc1.ID, "hv1")
	crossHive.SwarmID = &prodInDC2.ID
	err := db.Create(&crossHive).Error
	if err == nil {
		t.Fatal("a hypervisor in dc1 must not be able to join a swarm in dc2")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Errorf("want a foreign key violation, got %v", err)
	}
}

func TestSwarmMembershipWithinOneHiveIsAllowed(t *testing.T) {
	// The positive control. Without it the test above could pass because the
	// insert is broken for some unrelated reason.
	db := swarmDB(t)
	dc1 := newHive(t, db, "dc1")

	prod := tenant.Swarm{HiveID: dc1.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create swarm: %v", err)
	}
	hv := machine(dc1.ID, "hv1")
	hv.SwarmID = &prod.ID
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("a hypervisor must be able to join a swarm in its own hive: %v", err)
	}
	if tenant.MembershipOf(hv) != tenant.SwarmMembershipAssigned {
		t.Error("a hypervisor with a swarm reads as assigned")
	}
}

func TestSpareCapacityIsAHypervisorWithNoSwarm(t *testing.T) {
	// "In the cluster, in no group" must be storable — it is the state metal sits
	// in between being racked and being promised to something.
	db := swarmDB(t)
	dc1 := newHive(t, db, "dc1")

	hv := machine(dc1.ID, "spare-1")
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("a hypervisor with no swarm must be storable: %v", err)
	}
	var back tenant.Hypervisor
	if err := db.First(&back, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if back.SwarmID != nil {
		t.Fatal("swarm membership must round-trip as absent, not as a zero uuid")
	}
	if tenant.MembershipOf(back) != tenant.SwarmMembershipSpare {
		t.Error("a hypervisor with no swarm reads as spare")
	}
}

func TestDeletingASwarmWithMembersIsRefused(t *testing.T) {
	// RESTRICT, deliberately. Deleting "fleet-prod" would otherwise turn every
	// member into spare capacity from one statement, with nothing recording that
	// anybody meant to.
	//
	// It is also forced: Postgres nulls EVERY column of a composite foreign key
	// on SET NULL, so `SET hive_id = NULL, swarm_id = NULL` would fail the NOT
	// NULL on hive_id and report a swarm deletion as an error about the wrong
	// column entirely.
	db := swarmDB(t)
	dc1 := newHive(t, db, "dc1")
	prod := tenant.Swarm{HiveID: dc1.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	member := machine(dc1.ID, "hv1")
	member.SwarmID = &prod.ID
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}

	err := db.Delete(&tenant.Swarm{}, "id = ?", prod.ID).Error
	if err == nil {
		t.Fatal("deleting a swarm that still has members must be refused")
	}
	if strings.Contains(strings.ToLower(err.Error()), "hive_id") {
		t.Errorf("the error must be about the swarm still being referenced, not about hive_id "+
			"being null — that is the SET NULL failure mode this avoids: %v", err)
	}

	// Unassign, then it deletes.
	if err := db.Model(&tenant.Hypervisor{}).Where("swarm_id = ?", prod.ID).
		Update("swarm_id", nil).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&tenant.Swarm{}, "id = ?", prod.ID).Error; err != nil {
		t.Fatalf("an empty swarm must delete: %v", err)
	}
}

func TestSwarmNamesAreUniquePerHiveNotGlobally(t *testing.T) {
	// Two clusters may both sensibly have a swarm called "prod".
	db := swarmDB(t)
	dc1, dc2 := newHive(t, db, "dc1"), newHive(t, db, "dc2")

	if err := db.Create(&tenant.Swarm{HiveID: dc1.ID, Name: "prod"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&tenant.Swarm{HiveID: dc2.ID, Name: "prod"}).Error; err != nil {
		t.Fatalf("the same swarm name in another hive must be allowed: %v", err)
	}
	if err := db.Create(&tenant.Swarm{HiveID: dc1.ID, Name: "prod"}).Error; err == nil {
		t.Fatal("a duplicate swarm name within one hive must be refused")
	}
}

func TestDeletingAHiveTakesItsSwarmsWithIt(t *testing.T) {
	// CASCADE from hive to swarm: a swarm has no meaning without its cluster.
	db := swarmDB(t)
	dc1 := newHive(t, db, "dc1")
	if err := db.Create(&tenant.Swarm{HiveID: dc1.ID, Name: "prod"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&tenant.Hive{}, "id = ?", dc1.ID).Error; err != nil {
		t.Fatalf("delete hive: %v", err)
	}
	var n int64
	if err := db.Model(&tenant.Swarm{}).Where("hive_id = ?", dc1.ID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("swarms must go with their hive, %d left", n)
	}
}

func TestAPoolCannotPlaceIntoAnotherHivesSwarm(t *testing.T) {
	// Same rule as hypervisor membership, on the other side of the placement
	// decision: a pool in dc1 must not target a swarm in dc2, or the query would
	// look for candidates that cannot exist and the pool would silently never
	// place anything.
	db := testsupport.OpenSchema(t, swarmSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.Slot{}, &tenant.Pool{})

	dc1, dc2 := newHive(t, db, "dc1"), newHive(t, db, "dc2")
	swarmInDC2 := tenant.Swarm{HiveID: dc2.ID, Name: "prod"}
	if err := db.Create(&swarmInDC2).Error; err != nil {
		t.Fatal(err)
	}
	slot := tenant.Slot{Name: "large-" + uuid.NewString()[:8], VCPU: 8, RAMGB: 32, DiskGB: 200}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatal(err)
	}

	err := db.Create(&tenant.Pool{
		HiveID: dc1.ID, SwarmID: swarmInDC2.ID, SlotID: slot.ID, Name: "web", DesiredCount: 3,
	}).Error
	if err == nil {
		t.Fatal("a pool in dc1 must not target a swarm in dc2")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Errorf("want a foreign key violation, got %v", err)
	}
}

func TestAPoolInItsOwnHivesSwarmIsAllowed(t *testing.T) {
	db := testsupport.OpenSchema(t, swarmSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.Slot{}, &tenant.Pool{})

	dc1 := newHive(t, db, "dc1")
	prod := tenant.Swarm{HiveID: dc1.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	slot := tenant.Slot{Name: "large-" + uuid.NewString()[:8], VCPU: 8, RAMGB: 32, DiskGB: 200}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatal(err)
	}
	// "A pool of 3 VMs at slot large", targeted at prod metal.
	if err := db.Create(&tenant.Pool{
		HiveID: dc1.ID, SwarmID: prod.ID, SlotID: slot.ID, Name: "web", DesiredCount: 3,
	}).Error; err != nil {
		t.Fatalf("a pool must be able to target a swarm in its own hive: %v", err)
	}
}

func TestDeletingASwarmWithPoolsIsRefused(t *testing.T) {
	// The other RESTRICT. A swarm still being placed into is capacity somebody is
	// counting on; removing it must be deliberate.
	db := testsupport.OpenSchema(t, swarmSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.Slot{}, &tenant.Pool{})

	dc1 := newHive(t, db, "dc1")
	prod := tenant.Swarm{HiveID: dc1.ID, Name: "prod"}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	slot := tenant.Slot{Name: "large-" + uuid.NewString()[:8], VCPU: 8, RAMGB: 32, DiskGB: 200}
	if err := db.Create(&slot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&tenant.Pool{
		HiveID: dc1.ID, SwarmID: prod.ID, SlotID: slot.ID, Name: "web", DesiredCount: 3,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Delete(&tenant.Swarm{}, "id = ?", prod.ID).Error; err == nil {
		t.Fatal("deleting a swarm that pools still place into must be refused")
	}
}

func TestSwarmScopePredicateExcludesSpareCapacity(t *testing.T) {
	// The predicate is the enforcement point, so this checks it against real rows
	// rather than asserting on the string. Spare capacity must NOT be a candidate:
	// it is available to be ASSIGNED, not available to be scheduled onto.
	db := testsupport.OpenSchema(t, swarmSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{})

	dc1 := newHive(t, db, "dc1")
	prod := tenant.Swarm{HiveID: dc1.ID, Name: "prod"}
	nonprod := tenant.Swarm{HiveID: dc1.ID, Name: "nonprod"}
	for _, sw := range []*tenant.Swarm{&prod, &nonprod} {
		if err := db.Create(sw).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, spec := range []struct {
		name  string
		swarm *uuid.UUID
	}{
		{"p1", &prod.ID},
		{"p2", &prod.ID},
		{"n1", &nonprod.ID},
		{"spare1", nil},
		{"spare2", nil},
	} {
		hv := machine(dc1.ID, spec.name)
		hv.SwarmID = spec.swarm
		if err := db.Create(&hv).Error; err != nil {
			t.Fatal(err)
		}
	}

	var names []string
	if err := db.Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", dc1.ID).
		Where(tenant.SwarmScopePredicate, prod.ID).
		Where(tenant.SchedulablePredicate).
		Order("name").Pluck("name", &names).Error; err != nil {
		t.Fatal(err)
	}

	if len(names) != 2 || names[0] != "p1" || names[1] != "p2" {
		t.Fatalf("a pool targeting prod must see only prod's members, got %v", names)
	}
}
