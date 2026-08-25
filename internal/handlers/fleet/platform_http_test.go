package fleet

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
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

var platformSchema = testsupport.SchemaName("apiary_fleet_platform_http")

// The same shape as fleetopsAPI, over the OTHER route group.
//
// The tenant database is again deliberately unprovisioned, so 403 means the gate
// refused and 503 means it passed and the request reached tenant storage. Minting
// an enrolment token is the exception and that is the point of this file: it
// opens no tenant database at all, so it completes with 201 and proves that a
// token can be minted with no cluster — and, before this change, with no cluster
// even being possible to name.
func platformAPI(t *testing.T) (humatest.TestAPI, *gorm.DB, *apimiddleware.Principal) {
	t.Helper()

	db := testsupport.OpenSchema(t, platformSchema,
		&control.User{}, &control.Organization{}, &control.Membership{},
		&control.BeeEnrolmentToken{})

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate a master key: %v", err)
	}
	tenants, err := database.NewTenantManager(db, base64.StdEncoding.EncodeToString(key),
		database.TenantPoolConfig{MaxSize: 1, MaxConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatalf("tenant manager: %v", err)
	}

	principal := &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodBearer,
		OrganizationID: uuid.New(),
		SessionJTI:     uuid.NewString(),
	}
	uid := uuid.New()
	principal.UserID = &uid
	seedUser(t, db, uid)
	seedOrg(t, db, principal.OrganizationID)

	_, api := humatest.New(t, huma.DefaultConfig("fleet-platform-probe", "1.0.0"))
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(hctx, apimiddleware.WithPrincipal(hctx.Context(), principal)))
	})

	deps := &app.Deps{
		ControlDB: db,
		Tenants:   tenants,
		Jobs:      &workflow.MockEnqueuer{},
		Config:    &config.Config{},
	}
	if err := RegisterUserProtected(api, deps); err != nil {
		t.Fatalf("register: %v", err)
	}
	return api, db, principal
}

// THE GAP THIS CLOSES. Minting used to require a hive id, validate it existed
// with a 404, and then never read it — so an organization with no cluster could
// not provision the first agent, which is the machine that would later become
// one of that cluster's nodes.
func TestMintingAnEnrolmentTokenNeedsNoCluster(t *testing.T) {
	api, db, p := platformAPI(t)
	grant(t, db, p, control.MemberRoleAdmin)

	res := api.Post("/fleet/enrolment-tokens", map[string]any{"ttl_minutes": 60})
	if res.Code != http.StatusCreated {
		t.Fatalf("minting with no cluster named must succeed: got %d, body %s", res.Code, res.Body)
	}

	// The row is what the enrolment path reads, so its absence would make a 201
	// meaningless.
	var n int64
	if err := db.Model(&control.BeeEnrolmentToken{}).
		Where("organization_id = ?", p.OrganizationID).Count(&n).Error; err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("%d tokens stored for this organization, want 1", n)
	}
}

// The deployed UI still sends hive_id. Huma rejects an unknown body field with
// 422 before the handler runs, so accepting-and-ignoring is what keeps that
// client working; this pins it, and pins that the id is not looked up — the
// value below names a hive that cannot exist, and the old handler answered 404.
func TestMintingStillAcceptsAndIgnoresAHiveID(t *testing.T) {
	api, db, p := platformAPI(t)
	grant(t, db, p, control.MemberRoleAdmin)

	res := api.Post("/fleet/enrolment-tokens", map[string]any{
		"hive_id":     uuid.NewString(),
		"ttl_minutes": 60,
	})
	if res.Code != http.StatusCreated {
		t.Fatalf("a hive id must be ignored rather than validated: got %d, body %s",
			res.Code, res.Body)
	}
	var n int64
	if err := db.Model(&control.BeeEnrolmentToken{}).
		Where("organization_id = ?", p.OrganizationID).Count(&n).Error; err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("%d tokens stored, want 1", n)
	}
}

// platformMutations are the write operations on this route group. Reads are
// covered by the tenant-lookup outcome the same way fleetops' are.
func platformMutations() []call {
	return []call{
		{"mint an enrolment token", http.MethodPost, "/fleet/enrolment-tokens",
			map[string]any{"ttl_minutes": 60}},
		{"create a hive", http.MethodPost, "/fleet/hives", map[string]any{"name": "dc1"}},
		{"queue a task", http.MethodPost, "/fleet/tasks",
			map[string]any{"bee_id": uuid.NewString(), "kind": "fleet.host.ping"}},
		{"remove an agent", http.MethodDelete, "/fleet/bees/" + uuid.NewString(), nil},
	}
}

func TestPlatformMutationsRefuseAPlainMember(t *testing.T) {
	api, db, p := platformAPI(t)
	grant(t, db, p, control.MemberRoleMember)

	for _, c := range platformMutations() {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("a plain member must not mint tokens, create hives, queue tasks or "+
					"remove agents: got %d", got)
			}
		})
	}

	grant(t, db, p, control.MemberRoleAdmin)
	for _, c := range platformMutations() {
		t.Run(c.name+" as admin", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("an admin must pass the gate: got %d", got)
			}
		})
	}
}

func TestPlatformMutationsRefuseAnImpersonatedAdmin(t *testing.T) {
	// Removing an agent revokes a credential and deletes the record naming it.
	// An operator acting as a customer holds admin in that customer's
	// organization, so a role check alone would admit them.
	api, db, p := platformAPI(t)
	grant(t, db, p, control.MemberRoleAdmin)

	actor := uuid.New()
	p.Impersonating = true
	p.ActorID = &actor
	if !p.IsImpersonating() {
		t.Fatal("the test principal must actually read as impersonating, or this proves nothing")
	}

	for _, c := range platformMutations() {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("an impersonated admin must be refused: got %d", got)
			}
		})
	}

	// The reads on this group stay open while impersonating, which is also what
	// makes the refusals above attributable to impersonation rather than to the
	// membership lookup failing.
	if got := api.Get("/fleet/bees").Code; got == http.StatusForbidden {
		t.Fatalf("listing agents must remain available while impersonating: got %d", got)
	}

	p.Impersonating = false
	p.ActorID = nil
	for _, c := range platformMutations() {
		t.Run(c.name+" not impersonating", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("a non-impersonated admin must pass: got %d", got)
			}
		})
	}
}

func TestPlatformReadsRefuseANonMember(t *testing.T) {
	api, db, p := platformAPI(t)
	revoke(t, db, p)

	reads := []call{
		{"list agents", http.MethodGet, "/fleet/bees", nil},
		{"read task logs", http.MethodGet, "/fleet/tasks/" + uuid.NewString() + "/logs", nil},
	}
	for _, c := range reads {
		t.Run(c.name, func(t *testing.T) {
			if got := c.do(api).Code; got != http.StatusForbidden {
				t.Fatalf("a non-member must be refused: got %d", got)
			}
		})
	}

	grant(t, db, p, control.MemberRoleMember)
	for _, c := range reads {
		t.Run(c.name+" as member", func(t *testing.T) {
			if got := c.do(api).Code; got == http.StatusForbidden {
				t.Fatalf("a member must pass the gate on a read: got %d", got)
			}
		})
	}
}

// Every fleet mutation resolves the tenant database from the principal, so an
// organization named in the request must not move which database is opened.
func TestPlatformResolvesTheOrganizationFromThePrincipalOnly(t *testing.T) {
	api, db, p := platformAPI(t)

	victim := uuid.New()
	seedOrg(t, db, victim)
	if err := db.Create(&control.Membership{
		UserID:         *p.UserID,
		OrganizationID: victim,
		Role:           control.MemberRoleAdmin,
	}).Error; err != nil {
		t.Fatalf("seed victim membership: %v", err)
	}

	for _, c := range platformMutations() {
		t.Run(c.name, func(t *testing.T) {
			path := c.path + "?organization_id=" + victim.String() + "&org=" + victim.String()
			if got := (call{c.name, c.method, path, c.body}).do(api).Code; got != http.StatusForbidden {
				t.Fatalf("naming another tenant's organization in the query must not admit the "+
					"request: got %d", got)
			}
		})
	}

	grant(t, db, p, control.MemberRoleAdmin)
	for _, c := range platformMutations() {
		t.Run(c.name+" with own-org admin", func(t *testing.T) {
			path := c.path + "?organization_id=" + victim.String()
			if got := (call{c.name, c.method, path, c.body}).do(api).Code; got == http.StatusForbidden {
				t.Fatalf("with admin in the principal's own organization the same request must "+
					"pass: got %d", got)
			}
		})
	}
}
