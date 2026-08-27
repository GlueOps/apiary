package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"testing"

	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var declareTestSchema = testsupport.SchemaName("apiary_repo_declare_test")

// declareNoMgmtTestSchema is deliberately separate: the shared schema is seeded
// with a management set so the other tests can declare at all, and emptying it
// mid-run would race them.
var declareNoMgmtTestSchema = testsupport.SchemaName("apiary_repo_declare_nomgmt_test")

func declareDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, declareTestSchema, &tenant.Hive{}, &tenant.Bee{},
		&tenant.Hypervisor{}, &tenant.FirewallSet{}, &tenant.FirewallSetElement{})
	seedManagementSet(t, db)
	return db
}

// seedManagementSet gives the fixture something for the fleet key's from=
// restriction to be derived from.
//
// Declaring refuses without one, because onboarding refuses without one — see
// ErrNoManagementSet. Every test here is about the declaration itself, so the
// precondition is satisfied once in the harness rather than repeated; the
// refusal has its own test on a schema that deliberately has none.
func seedManagementSet(t *testing.T, db *gorm.DB) {
	t.Helper()
	var n int64
	if err := db.Model(&tenant.FirewallSet{}).
		Where("purpose = ?", tenant.SetPurposeManagement).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		return
	}
	set := tenant.FirewallSet{
		Name: "management", Family: tenant.FamilyIPv4, Purpose: tenant.SetPurposeManagement,
	}
	if err := db.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&tenant.FirewallSetElement{
		SetID: set.ID, Value: "192.0.2.1/32",
	}).Error; err != nil {
		t.Fatal(err)
	}
}

// uniq makes an address that no other test in this package has used.
//
// The schema is shared by every test in the package, so a fixed address is a row
// left behind by whichever test ran first — and a count of "all pending rows"
// then measures the package's history rather than this test's work. That is what
// made these pass while skipping and fail against a real database.
func uniq(prefix string) string { return prefix + "-" + uuid.NewString()[:8] }

func hosts(addrs ...string) []tenant.DeclaredHost {
	out := make([]tenant.DeclaredHost, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, tenant.DeclaredHost{Address: a})
	}
	return out
}

// TestADeclaredHostIsQueuedAndNotSchedulable.
//
// Both columns carry a `default:` tag and GORM omits zero values for those on
// Create — so without an explicit write the row would store as 'provisioned',
// declaring a machine nobody has touched as already onboarded, AND as
// schedulable, admitting it to placement. The same trap that made a cordoned
// hypervisor store as schedulable.
func TestADeclaredHostIsQueuedAndNotSchedulable(t *testing.T) {
	db := declareDB(t)
	ctx := context.Background()

	addr := uniq("10.10.10.9")
	got, err := DeclareHosts(ctx, db, hosts(addr))
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if len(got) != 1 || !got[0].Created || !got[0].Queued {
		t.Fatalf("result = %+v", got)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "address = ?", addr).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if hv.ProvisioningState != tenant.OnboardPending {
		t.Fatalf("state = %q, want %q: a machine nobody has touched must not be recorded as "+
			"onboarded", hv.ProvisioningState, tenant.OnboardPending)
	}
	if hv.Schedulable {
		t.Fatal("a machine that has not been onboarded was admitted to placement")
	}
}

// TestARackIsDeclaredInOnePass, which is the case this exists for.
func TestARackIsDeclaredInOnePass(t *testing.T) {
	db := declareDB(t)
	ctx := context.Background()

	rack := []string{uniq("pve1"), uniq("pve2"), uniq("pve3"), uniq("pve4")}
	got, err := DeclareHosts(ctx, db, hosts(rack...))
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("%d results, want 4", len(got))
	}
	// Counted against THIS rack, not against every pending row in the schema —
	// the schema is shared, so the broader count measures the package's history.
	var n int64
	_ = db.Model(&tenant.Hypervisor{}).
		Where("address IN ? AND provisioning_state = ?", rack, tenant.OnboardPending).
		Count(&n).Error
	if n != 4 {
		t.Fatalf("%d rows queued, want 4", n)
	}
}

// TestARepeatedDeclarationLeavesTheExistingMachineAlone.
//
// Re-pasting a list is ordinary. Pushing an existing row back to pending because
// of it would put a second worker onto a host another may already be inside, and
// redriving one is a deliberate act with its own endpoint.
func TestARepeatedDeclarationLeavesTheExistingMachineAlone(t *testing.T) {
	db := declareDB(t)
	ctx := context.Background()

	addr := uniq("pve1")
	if _, err := DeclareHosts(ctx, db, hosts(addr)); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Pretend it finished onboarding.
	if err := db.Model(&tenant.Hypervisor{}).Where("address = ?", addr).
		Update("provisioning_state", tenant.OnboardProvisioned).Error; err != nil {
		t.Fatalf("advance: %v", err)
	}

	got, err := DeclareHosts(ctx, db, hosts(addr))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if got[0].Created {
		t.Fatal("a known address was reported as newly created")
	}
	if got[0].Detail == "" {
		t.Fatal("a repeat said nothing about why nothing happened")
	}

	var hv tenant.Hypervisor
	_ = db.First(&hv, "address = ?", addr).Error
	if hv.ProvisioningState != tenant.OnboardProvisioned {
		t.Fatalf("state = %q: a re-declaration reset a machine that was already onboarded",
			hv.ProvisioningState)
	}
}

// TestOneBadAddressDoesNotLoseTheRest.
//
// A rack declared all-or-nothing would roll back thirty-one good hosts over one
// repeat, and the operator would have to work out which, edit the list, and
// paste it again.
func TestOneBadAddressDoesNotLoseTheRest(t *testing.T) {
	db := declareDB(t)
	ctx := context.Background()

	repeated := uniq("pve2")
	batch := []string{uniq("pve1"), repeated, uniq("pve3")}

	if _, err := DeclareHosts(ctx, db, hosts(repeated)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := DeclareHosts(ctx, db, hosts(batch...))
	if err != nil {
		t.Fatalf("declare: %v", err)
	}

	created := 0
	for _, r := range got {
		if r.Created {
			created++
		}
	}
	if created != 2 {
		t.Fatalf("%d created, want 2: the repeat should not have cost the other two", created)
	}
	var n int64
	_ = db.Model(&tenant.Hypervisor{}).Where("address IN ?", batch).Count(&n).Error
	if n != 3 {
		t.Fatalf("%d machines, want 3", n)
	}
}

// Declaring refuses when nothing would authorise the fleet key.
//
// Onboarding installs that key with a from= restriction derived from the
// management sets and refuses when the derived list is empty, because the key
// would be installed permitting nothing. Without this check the API answered 201
// with queued=true for every host and the work failed afterwards, one job per
// machine, somewhere the operator was not looking: five hosts accepted, five
// jobs retried to exhaustion, five rows marked failed.
func TestDeclaringRefusesWithoutAManagementSet(t *testing.T) {
	db := testsupport.OpenSchema(t, declareNoMgmtTestSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.FirewallSet{}, &tenant.FirewallSetElement{})

	_, err := DeclareHosts(context.Background(), db,
		[]tenant.DeclaredHost{{Address: uniq("nomgmt"), User: "root"}})
	if !errors.Is(err, ErrNoManagementSet) {
		t.Fatalf("declaring without a management set must be refused; got %v", err)
	}

	// And nothing was recorded — a refusal that half-created rows would leave
	// hosts queued for work that cannot run.
	var n int64
	if err := db.Model(&tenant.Hypervisor{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("a refused declaration must create nothing; found %d rows", n)
	}

	// A set with no elements is the same refusal: it derives to an empty list.
	set := tenant.FirewallSet{
		Name: "empty", Family: tenant.FamilyIPv4, Purpose: tenant.SetPurposeManagement,
	}
	if err := db.Create(&set).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := DeclareHosts(context.Background(), db,
		[]tenant.DeclaredHost{{Address: uniq("nomgmt"), User: "root"}}); !errors.Is(err, ErrNoManagementSet) {
		t.Fatalf("an empty management set must still refuse; got %v", err)
	}
}
