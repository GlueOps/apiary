package fleet

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var maintenanceAckSchema = testsupport.SchemaName("apiary_fleet_maintenance_ack")

// ackAPI registers the maintenance routes against one tenant schema with the
// role gates stubbed open: who may call them is proven by the route tables in
// fleetops_http_test.go, against the real gates. This is what they do.
func ackAPI(t *testing.T) (humatest.TestAPI, *gorm.DB, *apimiddleware.Principal) {
	t.Helper()
	db := testsupport.OpenSchema(t, maintenanceAckSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{}, &tenant.AuditEvent{},
		&tenant.MaintenanceSchedule{}, &tenant.MaintenanceRun{}, &tenant.MaintenanceRunNode{})
	if err := db.Exec(`TRUNCATE maintenance_run_nodes, maintenance_runs, maintenance_schedules,
		audit_events, hypervisors, bees, hives CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	uid := uuid.New()
	p := &apimiddleware.Principal{Method: apimiddleware.AuthMethodBearer,
		OrganizationID: uuid.New(), UserID: &uid}
	_, api := humatest.New(t, huma.DefaultConfig("maintenance-ack", "1.0.0"))
	role := func(context.Context) (*apimiddleware.Principal, error) { return p, nil }
	registerMaintenance(api, func(context.Context, uuid.UUID) (*gorm.DB, error) { return db, nil },
		role, role)
	return api, db, p
}

// seedFailures gives a cordoned hypervisor `n` failed maintenance turns, each
// with an upgrade task queued, an hour apart and ending an hour ago.
func seedFailures(t *testing.T, db *gorm.DB, n int) tenant.Hypervisor {
	t.Helper()
	name := "hv1"
	hv := tenant.Hypervisor{Name: &name, Address: "192.0.2.1"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("schedulable", false).Error; err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Duration(n+1) * time.Hour)
	for i := range n {
		at := base.Add(time.Duration(i) * time.Hour)
		run := tenant.MaintenanceRun{Standalone: true, Status: tenant.MaintenanceRunFailed,
			Reboot: tenant.RebootAlways, WindowStart: at, WindowEnd: at.Add(time.Hour), StartedAt: at,
			FinishedAt: &at}
		if err := db.Create(&run).Error; err != nil {
			t.Fatal(err)
		}
		task := uuid.New()
		node := tenant.MaintenanceRunNode{RunID: run.ID, HypervisorID: hv.ID, Name: name,
			Status: tenant.MaintenanceNodeFailed, Reason: "the upgrade failed: apt said no",
			TaskID: &task, CordonedAt: &at, StartedAt: &at, FinishedAt: &at}
		if err := db.Create(&node).Error; err != nil {
			t.Fatal(err)
		}
	}
	return hv
}

func TestAcknowledgeIsAuditedAndRefusedWithNoStrikes(t *testing.T) {
	api, db, p := ackAPI(t)
	hv := seedFailures(t, db, 2)

	var list struct {
		Hypervisors []MaintenanceStrikesView `json:"hypervisors"`
	}
	resp := api.Get("/fleet/maintenance/strikes")
	_ = json.Unmarshal(resp.Body.Bytes(), &list)
	if len(list.Hypervisors) != 1 || !list.Hypervisors[0].Flagged || list.Hypervisors[0].Strikes != 2 ||
		list.Hypervisors[0].Schedulable {
		t.Fatalf("the flagged node must be listed, still cordoned: %d %s", resp.Code, resp.Body)
	}

	path := "/fleet/hypervisors/" + hv.ID.String() + "/maintenance/acknowledge"
	if resp := api.Post(path); resp.Code != http.StatusOK {
		t.Fatalf("acknowledge: %d %s", resp.Code, resp.Body)
	}
	var ev tenant.AuditEvent
	if err := db.Where("action = ?", tenant.AuditMaintenanceAcknowledge).First(&ev).Error; err != nil {
		t.Fatalf("the acknowledgement must be audited: %v", err)
	}
	if ev.PrincipalID != *p.UserID || ev.TargetID == nil || *ev.TargetID != hv.ID {
		t.Fatalf("audited as the person, against the machine: %+v", ev)
	}
	if meta := ev.Metadata.Data(); meta["strikes_cleared"] != float64(2) {
		t.Fatalf("the audit records what was cleared: %v", ev.Metadata.Data())
	}

	// C11: an acknowledgement is not an uncordon.
	var after tenant.Hypervisor
	db.First(&after, "id = ?", hv.ID)
	if after.Schedulable || after.MaintenanceAcknowledgedAt == nil {
		t.Fatalf("acknowledged and still cordoned: schedulable=%v ack=%v",
			after.Schedulable, after.MaintenanceAcknowledgedAt)
	}

	if resp := api.Post(path); resp.Code != http.StatusConflict {
		t.Fatalf("nothing left to acknowledge must be a 409: %d %s", resp.Code, resp.Body)
	}
	resp = api.Get("/fleet/maintenance/strikes")
	_ = json.Unmarshal(resp.Body.Bytes(), &list)
	if len(list.Hypervisors) != 0 {
		t.Fatalf("an acknowledged node has no strikes left: %s", resp.Body)
	}
}

// C6: an id that is not in this organization's database is not found.
func TestAcknowledgeAForeignHypervisorIsNotFound(t *testing.T) {
	api, _, _ := ackAPI(t)
	resp := api.Post("/fleet/hypervisors/" + uuid.NewString() + "/maintenance/acknowledge")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("got %d %s", resp.Code, resp.Body)
	}
}

func TestMaintenanceBudgetIsValidatedByTheAPI(t *testing.T) {
	api, _, _ := ackAPI(t)
	for _, c := range []map[string]any{{"maintenance_pct": 51}, {"max_nodes_per_run": 1001},
		{"max_nodes_per_run": -1}} {
		body := map[string]any{"standalone": true, "name": "w", "kind": "once",
			"once_at":          time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			"duration_minutes": 60, "reboot": "always", "enabled": true}
		for k, v := range c {
			body[k] = v
		}
		if resp := api.Post("/fleet/maintenance/schedules", body); resp.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%v must be refused: %d %s", c, resp.Code, resp.Body)
		}
	}
}
