package fleet

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var departureSchema = testsupport.SchemaName("apiary_fleet_departure")

func departureDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, departureSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{}, &tenant.Placement{},
		&tenant.MembershipSnapshot{})
	if err := db.Exec(`TRUNCATE membership_snapshots, placements, hypervisors, bees, hives ` +
		`RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// seedDepartableHive is a hive of n observed members, every one cordoned,
// empty and freshly synced — so the minimum is the only thing that can refuse.
func seedDepartableHive(t *testing.T, db *gorm.DB, n, minimum int) (tenant.Hive, []uuid.UUID) {
	t.Helper()
	hive := tenant.Hive{Name: "dc1", MinimumNodes: minimum}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ids := make([]uuid.UUID, 0, n)
	for i := range n {
		bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: now}
		if err := db.Create(&bee).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&tenant.Bee{}).Where("id = ?", bee.ID).Updates(map[string]any{
			"observed_cluster": hive.Name, "observed_membership_at": now,
		}).Error; err != nil {
			t.Fatal(err)
		}
		name := "hv0" + string(rune('1'+i))
		hv := tenant.Hypervisor{BeeID: &bee.ID, HiveID: &hive.ID, Name: &name,
			Address: "192.0.2." + string(rune('1'+i))}
		if err := db.Create(&hv).Error; err != nil {
			t.Fatal(err)
		}
		// A map, because Schedulable carries a default tag and a false in the
		// struct would be dropped from the INSERT.
		if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).Updates(map[string]any{
			"schedulable": false, "last_synced_at": now, "observed_guests": 0,
		}).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, hv.ID)
	}
	return hive, ids
}

func hasCode(rs []tenant.DepartureRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

func departingCount(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var n int64
	if err := db.Model(&tenant.Hypervisor{}).Where("departing_at IS NOT NULL").Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return int(n)
}

// The incident, end to end through the code the handler runs: four nodes, a
// minimum of three, one departure already under way. A second must be refused,
// and the refusal must name the node it did not count.
func TestADepartureUnderWayIsNotCountedAsRemaining(t *testing.T) {
	db := departureDB(t)
	ctx := context.Background()
	_, ids := seedDepartableHive(t, db, 4, 3)

	if rs, err := requestDeparture(ctx, db, ids[3], tenant.CredentialsAcceptExposure, tenant.DrainThresholds{}); err != nil || len(rs) != 0 {
		t.Fatalf("first departure (4 -> 3, minimum 3) must pass: refusals %v, err %v", rs, err)
	}

	rs, err := requestDeparture(ctx, db, ids[0], tenant.CredentialsAcceptExposure, tenant.DrainThresholds{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(rs, tenant.RefuseHiveTooSmall) {
		t.Fatalf("second departure would leave 2 under a minimum of 3; got %v", rs)
	}
	var detail string
	for _, r := range rs {
		if r.Code == tenant.RefuseHiveTooSmall {
			detail = r.Detail
		}
	}
	if !strings.Contains(detail, "left with 2") || !strings.Contains(detail, "hv04 (departure requested 0s ago)") {
		t.Fatalf("refusal must give the count and name the node not counted, got %q", detail)
	}
	if got := departingCount(t, db); got != 1 {
		t.Fatalf("a refused departure must not be recorded: %d departing, want 1", got)
	}

	// The list view reads the same count.
	var hvs []tenant.Hypervisor
	if err := db.Order("name").Find(&hvs).Error; err != nil {
		t.Fatal(err)
	}
	views, err := buildHypervisorViews(ctx, db, hvs, tenant.DrainThresholds{})
	if err != nil {
		t.Fatal(err)
	}
	if v := views[0]; !viewHasCode(v.DepartureRefusals, tenant.RefuseHiveTooSmall) {
		t.Fatalf("the list must show %s as refused too, got %v", v.Name, v.DepartureRefusals)
	}
}

// The departing node itself is subtracted: three nodes under a minimum of three
// cannot lose one, in the request and in the list.
func TestTheDepartingNodeIsSubtracted(t *testing.T) {
	db := departureDB(t)
	ctx := context.Background()
	_, ids := seedDepartableHive(t, db, 3, 3)

	rs, err := requestDeparture(ctx, db, ids[0], tenant.CredentialsAcceptExposure, tenant.DrainThresholds{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(rs, tenant.RefuseHiveTooSmall) {
		t.Fatalf("3 -> 2 under a minimum of 3 must be refused, got %v", rs)
	}

	var hv tenant.Hypervisor
	if err := db.First(&hv, "id = ?", ids[0]).Error; err != nil {
		t.Fatal(err)
	}
	views, err := buildHypervisorViews(ctx, db, []tenant.Hypervisor{hv}, tenant.DrainThresholds{})
	if err != nil {
		t.Fatal(err)
	}
	if !viewHasCode(views[0].DepartureRefusals, tenant.RefuseHiveTooSmall) {
		t.Fatalf("the list must refuse 3 -> 2 as well, got %v", views[0].DepartureRefusals)
	}
}

// TWO DEPARTURES AT ONCE cannot both pass. Each, checked without the other,
// sees three remaining; under the hive lock the second sees the first.
//
// Repeated because a race that is only possible is not reliably hit once.
func TestConcurrentDeparturesCannotBothPassTheMinimum(t *testing.T) {
	for round := range 10 {
		db := departureDB(t)
		ctx := context.Background()
		_, ids := seedDepartableHive(t, db, 4, 3)

		var (
			wg       sync.WaitGroup
			start    = make(chan struct{})
			refusals [2][]tenant.DepartureRefusal
			errs     [2]error
		)
		for i := range 2 {
			wg.Go(func() {
				<-start
				refusals[i], errs[i] = requestDeparture(ctx, db, ids[i],
					tenant.CredentialsAcceptExposure, tenant.DrainThresholds{})
			})
		}
		close(start)
		wg.Wait()

		passed := 0
		for i := range 2 {
			if errs[i] != nil {
				t.Fatalf("round %d: departure %d: %v", round, i, errs[i])
			}
			if len(refusals[i]) == 0 {
				passed++
			} else if !hasCode(refusals[i], tenant.RefuseHiveTooSmall) {
				t.Fatalf("round %d: unexpected refusal %v", round, refusals[i])
			}
		}
		if passed != 1 || departingCount(t, db) != 1 {
			t.Fatalf("round %d: %d departures passed and %d recorded; exactly one may, "+
				"or the hive drops to 2 under a minimum of 3", round, passed, departingCount(t, db))
		}
	}
}

func viewHasCode(vs []RefusalView, code string) bool {
	for _, v := range vs {
		if v.Code == code {
			return true
		}
	}
	return false
}
