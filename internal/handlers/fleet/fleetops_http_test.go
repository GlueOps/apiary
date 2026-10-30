package fleet

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var fleetopsSchema = testsupport.SchemaName("apiary_fleetops_http")

// These tests drive the REAL registered routes over HTTP.
//
// Everything else in this package tests view assembly and refusal rendering,
// which is where the logic is — but not where the danger is. The danger is a
// route mounted with the wrong gate: a read open to a non-member, a mutation
// open to a plain member, or a mutation an operator can perform while
// impersonating a customer. None of those is visible from a unit test of the
// handler body, because the gate is a line inside the closure that a direct call
// never exercises.
//
// The tenant database is deliberately NOT provisioned. Every gate runs before
// the handler touches tenant storage, so an authorised request gets as far as
// "storage is not ready" — 503. That makes the two outcomes cleanly separable:
//
//	403 -> the gate refused
//	503 -> the gate passed and the request reached the tenant lookup
//
// A test that only ever asserted 403 could not tell a working gate from a broken
// route, so every deny case below is paired with the same request under a
// sufficient role.
func fleetopsAPI(t *testing.T) (humatest.TestAPI, *gorm.DB, *apimiddleware.Principal) {
	t.Helper()

	db := testsupport.OpenSchema(t, fleetopsSchema,
		&control.User{}, &control.Organization{}, &control.Membership{})

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate a master key: %v", err)
	}
	tenants, err := database.NewTenantManager(db, base64.StdEncoding.EncodeToString(key),
		// One entry is enough: no tenant is ever resolved here. The pool refuses
		// a zero size outright, so it has to be stated rather than left default.
		database.TenantPoolConfig{MaxSize: 1, MaxConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("tenant manager: %v", err)
	}

	// The principal is a POINTER shared with the test, so a case can flip
	// impersonation without rebuilding the API.
	principal := &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodBearer,
		OrganizationID: uuid.New(),
		SessionJTI:     uuid.NewString(),
	}
	uid := uuid.New()
	principal.UserID = &uid

	// Memberships carry foreign keys to both, so the rows have to exist before a
	// role can be granted. Seeded here rather than per test so every case starts
	// from the same shape.
	seedUser(t, db, uid)
	seedOrg(t, db, principal.OrganizationID)

	_, api := humatest.New(t, huma.DefaultConfig("fleetops-probe", "1.0.0"))
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(hctx, apimiddleware.WithPrincipal(hctx.Context(), principal)))
	})

	deps := &app.Deps{
		ControlDB: db,
		Tenants:   tenants,
		Jobs:      &workflow.MockEnqueuer{},
		Config:    &config.Config{},
	}
	if err := RegisterFleetOps(api, deps); err != nil {
		t.Fatalf("register: %v", err)
	}
	return api, db, principal
}

func seedUser(t *testing.T, db *gorm.DB, id uuid.UUID) {
	t.Helper()
	if err := db.Create(&control.User{
		ID:        id,
		Email:     id.String() + "@example.test",
		Password:  "x",
		FirstName: "Test",
		LastName:  "User",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedOrg(t *testing.T, db *gorm.DB, id uuid.UUID) {
	t.Helper()
	if err := db.Create(&control.Organization{
		ID:     id,
		Name:   "org-" + id.String()[:8],
		Status: "active",
	}).Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}
}

// grant writes a membership for the principal's own organization.
func grant(t *testing.T, db *gorm.DB, p *apimiddleware.Principal, role control.MemberRole) {
	t.Helper()
	if err := db.Where("user_id = ? AND organization_id = ?", *p.UserID, p.OrganizationID).
		Delete(&control.Membership{}).Error; err != nil {
		t.Fatalf("clear membership: %v", err)
	}
	if err := db.Create(&control.Membership{
		UserID:         *p.UserID,
		OrganizationID: p.OrganizationID,
		Role:           role,
	}).Error; err != nil {
		t.Fatalf("grant %s: %v", role, err)
	}
}

func revoke(t *testing.T, db *gorm.DB, p *apimiddleware.Principal) {
	t.Helper()
	if err := db.Where("user_id = ? AND organization_id = ?", *p.UserID, p.OrganizationID).
		Delete(&control.Membership{}).Error; err != nil {
		t.Fatalf("revoke: %v", err)
	}
}

// call is one request against a registered route.
type call struct {
	name   string
	method string
	path   string
	body   any
}

func (c call) do(api humatest.TestAPI) *httptest.ResponseRecorder {
	switch c.method {
	case http.MethodGet:
		return api.Get(c.path)
	case http.MethodPost:
		if c.body == nil {
			return api.Post(c.path)
		}
		return api.Post(c.path, c.body)
	case http.MethodPatch:
		return api.Patch(c.path, c.body)
	case http.MethodPut:
		return api.Put(c.path, c.body)
	case http.MethodDelete:
		return api.Delete(c.path)
	}
	panic("unsupported method " + c.method)
}

func reads() []call {
	hypID, hiveID := uuid.NewString(), uuid.NewString()
	return []call{
		{"list hives", http.MethodGet, "/fleet/hives", nil},
		{"list hypervisors", http.MethodGet, "/fleet/hypervisors", nil},
		{"get hypervisor", http.MethodGet, "/fleet/hypervisors/" + hypID, nil},
		{"list fleetless", http.MethodGet, "/fleet/fleetless", nil},
		{"unused hive id", http.MethodGet, "/fleet/hives?x=" + hiveID, nil},
		{"list maintenance windows", http.MethodGet, "/fleet/maintenance/schedules", nil},
		{"list maintenance runs", http.MethodGet, "/fleet/maintenance/runs", nil},
	}
}

func mutations() []call {
	hypID, hiveID := uuid.NewString(), uuid.NewString()
	return []call{
		{"cordon", http.MethodPost, "/fleet/hypervisors/" + hypID + "/cordon",
			map[string]any{"schedulable": false}},
		// A valid body, deliberately: departure now requires a credential decision,
		// and huma validates the body BEFORE this handler runs. Sending none would
		// make these authorization cases pass on a 422 from the schema rather than
		// on the 403 they exist to prove.
		{"depart", http.MethodPost, "/fleet/hypervisors/" + hypID + "/depart",
			map[string]any{"credential_disposition": "rotate"}},
		{"reprovision", http.MethodPost, "/fleet/hypervisors/" + hypID + "/reprovision", nil},
		{"sync", http.MethodPost, "/fleet/hives/" + hiveID + "/sync", nil},
		{"update hive", http.MethodPatch, "/fleet/hives/" + hiveID,
			map[string]any{"minimum_nodes": 3}},
		{"arrive", http.MethodPost, "/fleet/fleetless/" + hypID + "/arrive",
			map[string]any{
				"target_hive_id": hiveID,
				"links":          []map[string]any{{"number": 0, "address": "10.1.0.9"}},
			}},
		// Maintenance reboots hypervisors, so every write is admin-only and refused
		// while impersonating. Valid bodies for the same reason as depart's.
		{"schedule maintenance", http.MethodPost, "/fleet/maintenance/schedules",
			map[string]any{
				"hive_id": hiveID, "name": "weekly", "kind": "recurring", "weekdays": []int{0},
				"start_time": "02:00", "timezone": "Europe/Helsinki", "duration_minutes": 120,
				"reboot": "if-needed", "enabled": true,
			}},
		{"change maintenance", http.MethodPut, "/fleet/maintenance/schedules/" + hiveID,
			map[string]any{
				"name": "weekly", "kind": "once", "once_at": "2030-01-01T02:00:00Z",
				"duration_minutes": 60, "reboot": "always", "enabled": true,
			}},
		{"delete maintenance", http.MethodDelete, "/fleet/maintenance/schedules/" + hiveID, nil},
		{"stop maintenance run", http.MethodPost, "/fleet/maintenance/runs/" + hiveID + "/stop", nil},
	}
}

func TestFleetOpsReadsRefuseANonMember(t *testing.T) {
	api, db, p := fleetopsAPI(t)
	revoke(t, db, p)

	for _, c := range reads() {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("a non-member must be refused: got %d", got)
			}
		})
	}

	// Positive control on the identical routes: with membership the same
	// requests must get PAST the gate. Without this, the 403s above could equally
	// mean the routes are broken.
	grant(t, db, p, control.MemberRoleMember)
	for _, c := range reads() {
		t.Run(c.name+" as member", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("a member must pass the gate on a read: got %d", got)
			}
		})
	}
}

func TestFleetOpsMutationsRefuseAPlainMember(t *testing.T) {
	api, db, p := fleetopsAPI(t)
	grant(t, db, p, control.MemberRoleMember)

	for _, c := range mutations() {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("a plain member must not cordon, depart, arrive, sync or reconfigure: got %d", got)
			}
		})
	}

	grant(t, db, p, control.MemberRoleAdmin)
	for _, c := range mutations() {
		t.Run(c.name+" as admin", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("an admin must pass the gate: got %d", got)
			}
		})
	}
}

func TestFleetOpsMutationsRefuseAnImpersonatedAdmin(t *testing.T) {
	// The sharp one. An operator acting AS a customer holds an admin membership
	// in that customer's organization, so a role check alone admits them — and
	// these operations cordon hosts, dismantle cluster membership and hand a
	// machine to another fleet, under the customer's name.
	api, db, p := fleetopsAPI(t)
	grant(t, db, p, control.MemberRoleAdmin)

	actor := uuid.New()
	p.Impersonating = true
	p.ActorID = &actor
	if !p.IsImpersonating() {
		t.Fatal("the test principal must actually read as impersonating, or this proves nothing")
	}

	for _, c := range mutations() {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("an impersonated admin must be refused: got %d", got)
			}
		})
	}

	// Reads stay open while impersonating: an operator investigating a ticket
	// needs to see the fleet. Asserted so that tightening one does not silently
	// tighten the other, and so the 403s above are attributable to impersonation
	// rather than to the membership lookup failing.
	for _, c := range reads() {
		t.Run("read "+c.name, func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("reads must remain available while impersonating: got %d", got)
			}
		})
	}

	// And with impersonation cleared, the same admin passes — which is what makes
	// the refusals above about impersonation and nothing else.
	p.Impersonating = false
	p.ActorID = nil
	for _, c := range mutations() {
		t.Run(c.name+" not impersonating", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("a non-impersonated admin must pass: got %d", got)
			}
		})
	}
}

func TestFleetOpsResolvesTheOrganizationFromThePrincipalOnly(t *testing.T) {
	// The IDOR that would matter most here: hypervisors and hives live in a
	// tenant database chosen by organization, so an organization taken from the
	// request would let one tenant cordon or depart another's metal.
	api, db, p := fleetopsAPI(t)

	victim := uuid.New()
	seedOrg(t, db, victim)
	// Admin in the VICTIM's organization only. If any handler resolved the
	// organization from the request, this membership would satisfy the gate.
	if err := db.Create(&control.Membership{
		UserID:         *p.UserID,
		OrganizationID: victim,
		Role:           control.MemberRoleAdmin,
	}).Error; err != nil {
		t.Fatalf("seed victim membership: %v", err)
	}

	for _, c := range mutations() {
		t.Run(c.name, func(t *testing.T) {
			path := c.path + "?organization_id=" + victim.String() + "&org=" + victim.String()
			if got := (call{c.name, c.method, path, c.body}).do(api).Code; got != http.StatusForbidden {
				t.Fatalf("naming another tenant's organization in the query must not admit the "+
					"request: got %d", got)
			}
		})
	}

	// Positive control: grant admin in the PRINCIPAL's own organization and the
	// identical victim-laden requests now pass the gate. Without this, the 403s
	// above could equally mean the routes are broken.
	grant(t, db, p, control.MemberRoleAdmin)
	for _, c := range mutations() {
		t.Run(c.name+" with own-org admin", func(t *testing.T) {
			path := c.path + "?organization_id=" + victim.String()
			if got := (call{c.name, c.method, path, c.body}).do(api).Code; got == http.StatusForbidden {
				t.Fatalf("with admin in the principal's own organization the same request must "+
					"pass: got %d", got)
			}
		})
	}
}

func TestFleetOpsRejectsAnOrganizationSmuggledInTheBody(t *testing.T) {
	// A stronger guarantee than the query-string case, and worth pinning
	// separately rather than folding into it: an unknown body field is rejected
	// with 422 before any handler code runs, so an organization named there never
	// reaches the point where it could be read. Asserting only "not admitted"
	// would let that protection be removed silently — the request would then
	// reach the handler and be refused for a different reason, and the test would
	// still pass while the surface had widened.
	api, db, p := fleetopsAPI(t)
	grant(t, db, p, control.MemberRoleAdmin)
	victim := uuid.New()

	for _, c := range mutations() {
		if c.body == nil {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			body := map[string]any{
				"organization_id": victim.String(),
				"org_id":          victim.String(),
			}
			for k, v := range c.body.(map[string]any) {
				body[k] = v
			}
			got := (call{c.name, c.method, c.path, body}).do(api).Code
			if got != http.StatusUnprocessableEntity {
				t.Fatalf("an unknown organization field in the body must be rejected outright: got %d", got)
			}
		})
	}
}

// The empty string must survive schema validation, because it is how a caller
// unbinds a credential.
//
// UpdateHiveRequest.credential_id carried format:"uuid" while the handler
// implemented `"" means unbind`. Those cannot both be true: format validation
// runs before any handler, so the unbind branch was unreachable and — because
// the UI sends the field on every save — the ordinary case of saving a cluster
// that has no credential bound yet returned 422 with nothing in it that named
// the problem.
//
// Asserted against the generated schema rather than by sending a request,
// because that is where the constraint lives; a handler test would pass while
// the document that rejects the request still said uuid.
func TestUnbindingACredentialSurvivesSchemaValidation(t *testing.T) {
	api, _, _ := fleetopsAPI(t)

	// Straight off the registered API's document, which is the artifact that
	// actually rejects the request.
	doc, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatalf("render the openapi document: %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(doc, &spec); err != nil {
		t.Fatalf("parse the openapi document: %v", err)
	}
	comps, _ := spec["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	body, ok := schemas["UpdateHiveRequestBody"].(map[string]any)
	if !ok {
		t.Fatal("no UpdateHiveRequestBody schema was generated; this test is asserting nothing")
	}
	props, _ := body["properties"].(map[string]any)
	prop, ok := props["credential_id"].(map[string]any)
	if !ok {
		t.Fatal("UpdateHiveRequestBody has no credential_id property")
	}
	if f, has := prop["format"]; has {
		t.Fatalf("credential_id declares format %v, so the empty string is rejected during "+
			"schema validation and a cluster with no credential cannot be saved at all; the "+
			"handler validates the id itself and returns a 422 that names the field", f)
	}
}

// A DEPARTURE DISPATCHES NOTHING. The whole sequence is convergent.
//
// The SSH job raced it in three separate ways before this: it ran delnode
// itself, it cleared membership on success, and it converted the departing node
// to standalone immediately. The last is the subtlest and the most damaging — it
// stops the departing node's corosync BEFORE a peer has removed it, dropping the
// survivor below quorum, and a non-quorate peer refuses to evict. lab-b was left
// with pve3 alone and unable to remove a pve4 that had already made itself
// standalone.
func TestDepartingDispatchesNoJob(t *testing.T) {
	src, err := os.ReadFile("fleetops.go")
	if err != nil {
		t.Fatalf("read handler: %v", err)
	}
	if strings.Contains(string(src), "EnqueueDepartHypervisor") {
		t.Fatal("the departure handler still enqueues the SSH job; it races every convergent " +
			"step and its standalone conversion drops the survivor below quorum before a peer " +
			"can evict")
	}
}

// A DISARM CAN BE RETRIED, AND ONLY ON A MACHINE THAT HAS LEFT.
//
// A failed disarm was the one outcome the fleet view could show and an operator
// could not act on — the task kind was not even in the enum, so retrying it
// meant writing SQL against the tenant database.
//
// Refused for a machine still IN a hive, because the secrets it holds are the
// ones it is using: destroying them would break the node rather than detach it.
func TestDisarmIsRetryableOnlyForADepartedMachine(t *testing.T) {
	src, err := os.ReadFile("fleetops.go")
	if err != nil {
		t.Fatalf("read handler: %v", err)
	}
	body := string(src)
	if !strings.Contains(body, `Path:          "/fleet/hypervisors/{id}/disarm"`) {
		t.Fatal("no disarm endpoint; a failed disarm is only actionable through SQL")
	}
	if !strings.Contains(body, "still a member of a cluster") {
		t.Fatal("the endpoint does not refuse a machine that is still in a hive, whose secrets " +
			"are the ones it is using")
	}
	if !strings.Contains(body, "has no agent") {
		t.Fatal("the endpoint does not refuse a machine with no agent, and the disarm runs ON " +
			"the host")
	}
}
