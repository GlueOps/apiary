package estate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/glueops/apiary/internal/testsupport"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Schemas dedicated to this suite.
//
// internal/repo's estate suite AutoMigrates credentials and estates into the
// PUBLIC schema of the same test database and drops them again on cleanup. Two
// packages test in parallel by default, so sharing public would have this suite
// failing for a reason that has nothing to do with handlers.
var (
	edgeControlSchema = testsupport.SchemaName("apiary_estate_edge_control")
	edgeTenantSchema  = testsupport.SchemaName("apiary_estate_edge_tenant")
	edgeEmptySchema   = testsupport.SchemaName("apiary_estate_edge_empty")
	// A SECOND tenant database. Estate isolation in this system is database-
	// level, not row-level -- EstateRepoGorm carries no organization and filters
	// by id alone -- so "another organization" only means anything if it resolves
	// to another database, which is what TenantManager does in production.
	edgeOtherTenantSchema = testsupport.SchemaName("apiary_estate_edge_tenant_b")
)

// theSecret is the sentinel this suite looks for in responses and logs. It is
// shaped like a real AWS secret access key so that a length- or charset-based
// redaction that only handles "obvious" secrets does not accidentally pass.
const theSecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

const theIdentity = "AKIAIOSFODNN7EXAMPLE"

func openScoped(t *testing.T, schema string) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the estate handler suite; " +
			"this test does NOT pass, it did not run")
	}

	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema).Error; err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"search_path="+schema), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open scoped test db (%s): %v", schema, err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// fixedTenant stands in for the tenant manager's LOOKUP, and nothing else.
//
// service.TenantResolver is the seam the service layer declares for exactly
// this, so nothing in this file reaches across into another package's code to
// make itself testable. Everything BELOW the seam stays real: the real
// EstateService, the real credential and estate repositories, real AES-GCM
// sealing with a real derived key, and real Postgres tables. What is replaced is
// the answer to "which database belongs to this organization" — the tenant
// manager's job, covered by internal/database's own suite.
//
// Provisioning a control-plane organization with a sealed DSN and a wrapped DEK
// was the alternative. It was rejected because it makes every handler test
// depend on tenant provisioning passing: this suite would then go red for
// reasons that say nothing about handlers, and the failure message would point
// at the wrong layer.
//
// It ROUTES BY ORGANIZATION rather than answering the same database for
// everyone. That is the production shape -- one database per tenant -- and it is
// also what gives this suite any grip on the brief's central invariant: with a
// single database behind every organization, all five `p.OrganizationID` reads
// in estate.go could be replaced by a freshly generated UUID and the entire
// suite stayed green. An organization with no entry gets an error, exactly as an
// unprovisioned organization does, so a handler that resolves the wrong
// organization fails loudly instead of silently reading the right rows.
type fixedTenant struct {
	byOrg map[uuid.UUID]*gorm.DB
	key   []byte
}

func (f fixedTenant) GetTenantContext(_ context.Context, orgID string) (*database.TenantContext, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return nil, fmt.Errorf("fixedTenant: malformed organization id %q: %w", orgID, err)
	}
	db, ok := f.byOrg[id]
	if !ok {
		return nil, fmt.Errorf("fixedTenant: no tenant database for organization %s", id)
	}
	return &database.TenantContext{DB: db, Key: f.key}, nil
}

type estatesMode int

const (
	// estatesWired is the normal case: a real service over real tenant tables.
	estatesWired estatesMode = iota
	// estatesNil leaves deps.Estates nil. Every *EstateService method reaches
	// bind first, and bind returns ErrTenantUnavailable for a nil receiver
	// rather than panicking, so this is how the 503 path is reached through the
	// real handler.
	estatesNil
	// estatesOnEmptySchema points the service at a schema with no tables, which
	// is the only way to drive a genuine internal error through the real call
	// path with a secret in the request.
	estatesOnEmptySchema
)

type harness struct {
	api    humatest.TestAPI
	ctl    *gorm.DB
	tdb    *gorm.DB
	orgID  uuid.UUID
	userID uuid.UUID

	// tenants is the live routing table the service resolves through, so a
	// second organization can be given its own database after construction.
	tenants map[uuid.UUID]*gorm.DB
	key     []byte
}

func newHarness(t *testing.T, mode estatesMode) *harness {
	t.Helper()

	ctl := openScoped(t, edgeControlSchema)
	if err := ctl.AutoMigrate(&control.User{}, &control.Organization{}, &control.Membership{}); err != nil {
		t.Fatalf("migrate control schema: %v", err)
	}

	tdb := openScoped(t, edgeTenantSchema)
	if err := tdb.AutoMigrate(&tenant.Credential{}, &tenant.Estate{}); err != nil {
		t.Fatalf("migrate tenant schema: %v", err)
	}

	// Rows, not tables. estates carries UNIQUE (provider, scope), so a row left
	// behind by an earlier test would make the next one fail for the wrong
	// reason — an insert conflict reported as whatever status that maps to.
	if err := ctl.Exec(`TRUNCATE memberships, organizations, users RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate control tables: %v", err)
	}
	if err := tdb.Exec(`TRUNCATE estates, credentials RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate tenant tables: %v", err)
	}

	org := control.Organization{Name: "edge-" + uuid.NewString(), Status: "active"}
	if err := ctl.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	user := control.User{
		Email:     "edge-" + uuid.NewString() + "@example.test",
		Password:  "x",
		FirstName: "Edge",
		LastName:  "Tester",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := ctl.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	deps := &app.Deps{ControlDB: ctl}
	key := bytes.Repeat([]byte{0x2a}, utils.MasterKeyLen)
	tenants := map[uuid.UUID]*gorm.DB{}
	switch mode {
	case estatesNil:
		// deps.Estates stays a nil *service.EstateService.
	case estatesWired, estatesOnEmptySchema:
		db := tdb
		if mode == estatesOnEmptySchema {
			db = openScoped(t, edgeEmptySchema)
		}
		tenants[org.ID] = db
		svc, err := service.NewEstateService(fixedTenant{byOrg: tenants, key: key}, key, nil)
		if err != nil {
			t.Fatalf("estate service: %v", err)
		}
		deps.Estates = svc
	}

	_, hapi := humatest.New(t)
	Register(hapi, deps)

	return &harness{api: hapi, ctl: ctl, tdb: tdb, orgID: org.ID, userID: user.ID, tenants: tenants, key: key}
}

// grant seeds the caller's membership. AssertOrgRole reads the organization from
// the PRINCIPAL, so this row is what every authorisation decision below resolves
// against — there is no request parameter that can select a different one.
func (h *harness) grant(t *testing.T, role control.MemberRole) {
	t.Helper()
	m := control.Membership{UserID: h.userID, OrganizationID: h.orgID, Role: role}
	if err := h.ctl.Create(&m).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
}

func (h *harness) setRole(t *testing.T, role control.MemberRole) {
	t.Helper()
	err := h.ctl.Model(&control.Membership{}).
		Where("user_id = ? AND organization_id = ?", h.userID, h.orgID).
		Update("role", role).Error
	if err != nil {
		t.Fatalf("set role %s: %v", role, err)
	}
}

func (h *harness) revokeMembership(t *testing.T) {
	t.Helper()
	err := h.ctl.Unscoped().
		Where("user_id = ? AND organization_id = ?", h.userID, h.orgID).
		Delete(&control.Membership{}).Error
	if err != nil {
		t.Fatalf("delete membership: %v", err)
	}
}

func (h *harness) principal() *apimiddleware.Principal {
	uid := h.userID
	return &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodBearer,
		UserID:         &uid,
		OrganizationID: h.orgID,
		SessionJTI:     uuid.NewString(),
	}
}

// impersonated is a principal that would otherwise be fully authorised. Both
// fields are needed: Principal.IsImpersonating() requires Impersonating AND a
// non-nil ActorID, so setting only the flag would silently not be impersonation.
func (h *harness) impersonated() *apimiddleware.Principal {
	p := h.principal()
	actor := uuid.New()
	p.ActorID = &actor
	p.Impersonating = true
	return p
}

// do issues the request with p installed the way RequireUserAuth installs it.
//
// A nil p means NO principal in the context at all, which is the reachable
// shape: it is what a handler sees when it is registered on the public group by
// mistake. It is deliberately not a typed-nil *Principal in the context — that
// is a state the auth middleware cannot produce (it always builds a non-nil
// Principal), so asserting on it would be asserting about fiction.
func (h *harness) do(t *testing.T, p *apimiddleware.Principal, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	ctx := t.Context()
	if p != nil {
		ctx = apimiddleware.WithPrincipal(ctx, p)
	}
	if body == nil {
		return h.api.DoCtx(ctx, method, path)
	}
	return h.api.DoCtx(ctx, method, path, body)
}

func awsScope() map[string]string {
	return map[string]string{"account": "111111111111", "region": "eu-west-1"}
}

func createBody(name string, scope map[string]string, identity, secret string) map[string]any {
	return map[string]any{
		"name":            name,
		"provider":        "aws",
		"scope":           scope,
		"credential_name": name + "-key",
		"credential_kind": "static_secret",
		"identity":        identity,
		"secret":          secret,
	}
}

func detailOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return rec.Body.String()
	}
	return body.Detail
}

func idOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (%s)", err, rec.Body.String())
	}
	if body.ID == "" {
		t.Fatalf("response carries no estate id: %s", rec.Body.String())
	}
	return body.ID
}

// mentionsSecret looks for the sentinel in every encoding a careless log line or
// serialiser would plausibly produce.
func mentionsSecret(blob string) string {
	encodings := map[string]string{
		"raw":       theSecret,
		"base64":    base64.StdEncoding.EncodeToString([]byte(theSecret)),
		"base64url": base64.RawURLEncoding.EncodeToString([]byte(theSecret)),
		"hex":       hex.EncodeToString([]byte(theSecret)),
	}
	for kind, enc := range encodings {
		if strings.Contains(blob, enc) {
			return kind
		}
	}
	return ""
}

// ---------------------------------------------------------------- status mapping

// TestTenantUnavailableIs503OnEveryOperation drives all five operations with the
// service left nil.
//
// This is the one test that proves every handler actually routes its error
// through mapErr rather than returning it raw: huma renders a raw error as 500,
// so a handler that forgot the mapErr call shows up here as a 500 and names
// itself in the subtest name.
//
// 503 rather than 500 matters operationally: the organization exists and its
// storage is expected to become available, so this is not the operator's system
// being broken and must not land on their 5xx dashboard as though it were.
func TestTenantUnavailableIs503OnEveryOperation(t *testing.T) {
	h := newHarness(t, estatesNil)
	h.grant(t, control.MemberRoleOwner)

	id := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"estates-list", http.MethodGet, "/estates", nil},
		{"estates-get", http.MethodGet, "/estates/" + id, nil},
		{"estates-create", http.MethodPost, "/estates", createBody("prod", awsScope(), theIdentity, theSecret)},
		{"estates-rotate-credential", http.MethodPost, "/estates/" + id + "/rotate",
			map[string]any{"identity": theIdentity, "secret": theSecret}},
		{"estates-delete", http.MethodDelete, "/estates/" + id, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := h.do(t, h.principal(), tc.method, tc.path, tc.body)
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("%s %s answered %d, want 503: an unprovisioned tenant is a transient state, "+
					"not a server fault, and a 500 here also means this handler is not going through mapErr: %s",
					tc.method, tc.path, rec.Code, rec.Body.String())
			}
			if d := detailOf(t, rec); !strings.Contains(d, "storage is not ready") {
				t.Errorf("503 body does not say what is wrong: %q", d)
			}
		})
	}
}

// TestUnknownEstateIs404 checks all three id-bearing operations answer with the
// SAME flat body.
//
// Identical bodies are the point: a caller must not be able to tell "this estate
// belongs to another tenant" from "this estate does not exist". Different
// wording would turn these endpoints into an oracle for enumerating other
// organizations' estates.
func TestUnknownEstateIs404(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	missing := uuid.NewString()
	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"get", http.MethodGet, "/estates/" + missing, nil},
		{"rotate", http.MethodPost, "/estates/" + missing + "/rotate",
			map[string]any{"identity": theIdentity, "secret": theSecret}},
		{"delete", http.MethodDelete, "/estates/" + missing, nil},
	}

	bodies := map[string]string{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := h.do(t, h.principal(), tc.method, tc.path, tc.body)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s %s answered %d, want 404: %s", tc.method, tc.path, rec.Code, rec.Body.String())
			}
			d := detailOf(t, rec)
			if d != "estate not found" {
				t.Errorf("body is %q, want the flat \"estate not found\"", d)
			}
			bodies[tc.name] = d
		})
	}

	for name, d := range bodies {
		if d != bodies["get"] {
			t.Errorf("%s answers %q where get answers %q: the difference tells a caller which "+
				"unknown ids exist somewhere else", name, d, bodies["get"])
		}
	}
}

// TestUnknownScopeKeyIs422AndNamesTheOffendingKey pins both the status and the
// message.
//
// The status must be 422, not 400 and certainly not 500: the request parsed
// fine, the provider simply does not define that key. The message must name the
// key, which is the only part an operator can act on — and naming a scope key
// reveals nothing about anyone else's data.
func TestUnknownScopeKeyIs422AndNamesTheOffendingKey(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	scope := awsScope()
	scope["nonsense"] = "x"

	rec := h.do(t, h.principal(), http.MethodPost, "/estates",
		createBody("prod", scope, theIdentity, theSecret))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("answered %d, want 422: %s", rec.Code, rec.Body.String())
	}
	d := detailOf(t, rec)
	if !strings.Contains(d, "nonsense") {
		t.Errorf("the refusal does not name the offending key, so an operator cannot fix it: %q", d)
	}
	// The refusal is about a scope key. Nothing about the credential belongs in
	// it, and an error message is one of the easiest places for one to end up.
	if kind := mentionsSecret(rec.Body.String()); kind != "" {
		t.Errorf("the 422 body carries the submitted secret (%s encoding): %s", kind, rec.Body.String())
	}
}

// TestScopeMissingARequiredKeyIs422 covers the other two scope arms.
//
// Distinct from the unknown-key case because they are different errors reaching
// the same status through different errors.Is arms — deleting either one from
// mapErr's 422 case drops that input to 500 while the other stays green.
func TestScopeMissingARequiredKeyIs422(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	cases := map[string]map[string]string{
		// region is REQUIRED for aws; nothing can supply it but the operator.
		"empty scope":  {},
		"account only": {"account": "111111111111"},
		// account is DERIVABLE, and derivable is not optional: without it two
		// estates in two accounts collapse to one scope.
		"region without account": {"region": "eu-west-1"},
	}

	for name, scope := range cases {
		t.Run(name, func(t *testing.T) {
			rec := h.do(t, h.principal(), http.MethodPost, "/estates",
				createBody("prod-"+strings.ReplaceAll(name, " ", "-"), scope, theIdentity, theSecret))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("scope %v answered %d, want 422: an uncanonical scope is a bad request, "+
					"not a server fault: %s", scope, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestEmptyIdentityOrSecretIs422 covers the gap huma's own validation leaves.
//
// Both fields are `required:"true"` and neither carries minLength, and huma's
// required only enforces that the KEY is present. So "" passes validation, reaches
// repo.CreateCredential and comes back as ErrIdentityMissing / ErrSecretMissing.
// Without the mapErr arm for those, submitting an empty string is a 500.
func TestEmptyIdentityOrSecretIs422(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	cases := map[string]struct{ identity, secret string }{
		"empty identity": {"", theSecret},
		"empty secret":   {theIdentity, ""},
		"both empty":     {"", ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := h.do(t, h.principal(), http.MethodPost, "/estates",
				createBody("prod-"+strings.ReplaceAll(name, " ", "-"), awsScope(), tc.identity, tc.secret))
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("answered %d, want 422: %s", rec.Code, rec.Body.String())
			}
			if kind := mentionsSecret(rec.Body.String()); kind != "" {
				t.Errorf("the 422 body echoes the request's secret (%s encoding): %s", kind, rec.Body.String())
			}
		})
	}
}

// TestNoRefusalEverSurfacesAs500 states the invariant directly instead of
// leaving it implied by a dozen separate expected-status checks.
//
// Every entry is a DELIBERATE refusal reached through the real call path. A
// policy decision reported as 500 tells the caller the system is broken, puts
// the operator's own rules on their 5xx dashboard, and — for the authorisation
// entries — makes a refusal indistinguishable from a crash.
func TestNoRefusalEverSurfacesAs500(t *testing.T) {
	missing := uuid.NewString()

	cases := []struct {
		name   string
		setup  func(t *testing.T) *harness
		who    func(h *harness) *apimiddleware.Principal
		method string
		path   string
		body   any
	}{
		{
			name: "unknown estate",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodGet,
			path:   "/estates/" + missing,
		},
		{
			name: "unknown scope key",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodPost,
			path:   "/estates",
			body: createBody("a", map[string]string{
				"account": "111111111111", "region": "eu-west-1", "nonsense": "x",
			}, theIdentity, theSecret),
		},
		{
			name: "missing required scope key",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodPost,
			path:   "/estates",
			body:   createBody("b", map[string]string{}, theIdentity, theSecret),
		},
		{
			name: "empty identity",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodPost,
			path:   "/estates",
			body:   createBody("c", awsScope(), "", theSecret),
		},
		{
			name: "empty secret",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodPost,
			path:   "/estates",
			body:   createBody("d", awsScope(), theIdentity, ""),
		},
		{
			name: "tenant unavailable",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesNil)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodGet,
			path:   "/estates",
		},
		{
			name:   "principal holds no membership",
			setup:  func(t *testing.T) *harness { return newHarness(t, estatesWired) },
			who:    (*harness).principal,
			method: http.MethodGet,
			path:   "/estates",
		},
		{
			name: "member role on a write",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleMember)
				return h
			},
			who:    (*harness).principal,
			method: http.MethodPost,
			path:   "/estates",
			body:   createBody("e", awsScope(), theIdentity, theSecret),
		},
		{
			name: "impersonated principal on a write",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    (*harness).impersonated,
			method: http.MethodDelete,
			path:   "/estates/" + missing,
		},
		{
			name: "no principal in context at all",
			setup: func(t *testing.T) *harness {
				h := newHarness(t, estatesWired)
				h.grant(t, control.MemberRoleOwner)
				return h
			},
			who:    func(*harness) *apimiddleware.Principal { return nil },
			method: http.MethodGet,
			path:   "/estates",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.setup(t)
			rec := h.do(t, tc.who(h), tc.method, tc.path, tc.body)
			if rec.Code == http.StatusInternalServerError {
				t.Fatalf("the refusal %q surfaced as 500 on %s %s: %s",
					tc.name, tc.method, tc.path, rec.Body.String())
			}
			if rec.Code < 400 {
				t.Fatalf("the refusal %q was ACCEPTED with %d on %s %s: %s",
					tc.name, rec.Code, tc.method, tc.path, rec.Body.String())
			}
		})
	}
}

// ------------------------------------------------------------- authorisation

// TestWritesRefuseImpersonatedSessions checks the read/write SPLIT, not a
// blanket refusal.
//
// A blanket 403 for an impersonated session would pass a weaker test and would
// be wrong: an operator acting as a user has to be able to LOOK at the account
// in order to help. What they must not be able to do is install or replace a
// credential they could later use themselves, or delete one to force a
// re-registration they control.
func TestWritesRefuseImpersonatedSessions(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	// Seeded by the same, fully authorised user acting normally — so the only
	// thing that differs below is impersonation.
	rec := h.do(t, h.principal(), http.MethodPost, "/estates",
		createBody("prod", awsScope(), theIdentity, theSecret))
	if rec.Code != http.StatusOK {
		t.Fatalf("seed create answered %d: %s", rec.Code, rec.Body.String())
	}
	id := idOf(t, rec)

	writes := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create", http.MethodPost, "/estates",
			createBody("staging", map[string]string{"account": "222222222222", "region": "eu-west-1"}, theIdentity, theSecret)},
		{"rotate", http.MethodPost, "/estates/" + id + "/rotate",
			map[string]any{"identity": "AKIAROTATED", "secret": theSecret}},
		{"delete", http.MethodDelete, "/estates/" + id, nil},
	}

	for _, w := range writes {
		t.Run("write/"+w.name, func(t *testing.T) {
			rec := h.do(t, h.impersonated(), w.method, w.path, w.body)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s answered %d for an impersonated OWNER, want 403 — RefuseImpersonation() is "+
					"missing from this operation: %s", w.name, rec.Code, rec.Body.String())
			}
			if d := detailOf(t, rec); d != "impersonated sessions may not perform this action" {
				t.Errorf("%s was refused for the wrong reason (%q): a role refusal here would mean the "+
					"impersonation guard is not what stopped it", w.name, d)
			}
		})
	}

	reads := []struct{ name, path string }{
		{"list", "/estates"},
		{"get", "/estates/" + id},
	}
	for _, r := range reads {
		t.Run("read/"+r.name, func(t *testing.T) {
			rec := h.do(t, h.impersonated(), http.MethodGet, r.path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s answered %d for an impersonated member, want 200: an operator helping a "+
					"customer must still be able to look: %s", r.name, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestWritesRequireAdminAndReadsRequireMember walks one caller through three
// membership states.
//
// The third state is the security-relevant one: a caller with NO membership must
// get the same body as a caller with the wrong role. Different bodies would let
// anyone holding a token probe which organizations they are a member of, and in
// this system a membership is a membership in a tenant's cloud credentials.
func TestWritesRequireAdminAndReadsRequireMember(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleAdmin)

	rec := h.do(t, h.principal(), http.MethodPost, "/estates",
		createBody("prod", awsScope(), theIdentity, theSecret))
	if rec.Code != http.StatusOK {
		t.Fatalf("admin create answered %d: %s", rec.Code, rec.Body.String())
	}
	id := idOf(t, rec)

	stagingScope := map[string]string{"account": "222222222222", "region": "eu-west-1"}
	writes := func(estateID string) []struct {
		name   string
		method string
		path   string
		body   any
	} {
		return []struct {
			name   string
			method string
			path   string
			body   any
		}{
			{"create", http.MethodPost, "/estates", createBody("staging", stagingScope, theIdentity, theSecret)},
			{"rotate", http.MethodPost, "/estates/" + estateID + "/rotate",
				map[string]any{"identity": "AKIAROTATED", "secret": theSecret}},
			{"delete", http.MethodDelete, "/estates/" + estateID, nil},
		}
	}
	reads := []struct{ name, path string }{
		{"list", "/estates"},
		{"get", "/estates/" + id},
	}

	memberBodies := map[string]string{}

	t.Run("role=member", func(t *testing.T) {
		h.setRole(t, control.MemberRoleMember)

		for _, r := range reads {
			rec := h.do(t, h.principal(), http.MethodGet, r.path, nil)
			if rec.Code != http.StatusOK {
				t.Errorf("%s answered %d for a member, want 200: reading is the member role's whole "+
					"purpose: %s", r.name, rec.Code, rec.Body.String())
			}
		}
		for _, w := range writes(id) {
			rec := h.do(t, h.principal(), w.method, w.path, w.body)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s answered %d for a member, want 403: installing or replacing a cloud "+
					"credential is an admin action: %s", w.name, rec.Code, rec.Body.String())
			}
			memberBodies[w.name] = detailOf(t, rec)
		}
	})

	t.Run("role=admin", func(t *testing.T) {
		h.setRole(t, control.MemberRoleAdmin)

		for _, r := range reads {
			if rec := h.do(t, h.principal(), http.MethodGet, r.path, nil); rec.Code != http.StatusOK {
				t.Errorf("%s answered %d for an admin, want 200: %s", r.name, rec.Code, rec.Body.String())
			}
		}
		// Ordered so delete runs last: rotate needs the estate to still exist.
		for _, w := range writes(id) {
			rec := h.do(t, h.principal(), w.method, w.path, w.body)
			if rec.Code != http.StatusOK {
				t.Errorf("%s answered %d for an admin, want 200 — the role gate is refusing a caller "+
					"who should be allowed through: %s", w.name, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("no membership at all", func(t *testing.T) {
		h.revokeMembership(t)

		for _, r := range reads {
			rec := h.do(t, h.principal(), http.MethodGet, r.path, nil)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s answered %d for a non-member, want 403: %s", r.name, rec.Code, rec.Body.String())
			}
		}
		for _, w := range writes(id) {
			rec := h.do(t, h.principal(), w.method, w.path, w.body)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s answered %d for a non-member, want 403: %s", w.name, rec.Code, rec.Body.String())
			}
			if got, want := detailOf(t, rec), memberBodies[w.name]; got != want {
				t.Errorf("%s answers %q with no membership but %q with the wrong role: the difference "+
					"lets a token holder enumerate which organizations they belong to", w.name, got, want)
			}
		}
	})
}

// ------------------------------------------------------------- no secrets out

// TestCreateReturnsTheWholeFingerprintAndNeverTheSecret is the only case that
// drives a 200 through the create handler.
//
// Three separate claims, because they fail independently:
//
//	(a) the response carries no secret material, in any encoding;
//	(b) the fingerprint is COMPLETE — byte-identical to the stored one. A
//	    truncated fingerprint makes every human comparison against a provider
//	    console weak, and for an assumed role the identifying component is an ARN
//	    whose name the customer chooses, so a short prefix is grindable;
//	(c) the response object has exactly the documented key set, so a field added
//	    to EstateView cannot ship unnoticed.
func TestCreateReturnsTheWholeFingerprintAndNeverTheSecret(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	rec := h.do(t, h.principal(), http.MethodPost, "/estates",
		createBody("prod", awsScope(), theIdentity, theSecret))
	if rec.Code != http.StatusOK {
		t.Fatalf("create answered %d: %s", rec.Code, rec.Body.String())
	}

	raw := rec.Body.String()
	if kind := mentionsSecret(raw); kind != "" {
		t.Fatalf("the create response carries the secret (%s encoding): %s", kind, raw)
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	fingerprint, _ := got["fingerprint"].(string)
	if fingerprint == "" {
		t.Fatal("the response carries no fingerprint: there is then no way for an operator to confirm " +
			"WHICH credential was installed")
	}

	// Read the stored row directly rather than through the service, so a
	// truncation applied on the way out cannot hide behind the same code path
	// that produced the value.
	var stored tenant.Credential
	if err := h.tdb.First(&stored, "id = ?", got["credential_id"]).Error; err != nil {
		t.Fatalf("read back the credential row: %v", err)
	}
	if fingerprint != stored.Fingerprint {
		t.Errorf("the response fingerprint %q is not the stored fingerprint %q — it has been truncated, "+
			"re-cased or re-encoded, and every comparison an operator makes against it is weakened",
			fingerprint, stored.Fingerprint)
	}
	if strings.Contains(fingerprint, "…") || strings.Contains(fingerprint, "...") {
		t.Errorf("the fingerprint is elided: %q", fingerprint)
	}

	want := []string{
		"credential_id", "credential_kind", "credential_name", "fingerprint",
		"id", "name", "provider", "scope", "scope_version",
	}
	keys := make([]string, 0, len(got))
	for k := range got {
		if k == "$schema" {
			continue // huma's own pointer to the response schema, never data.
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("EstateView's wire shape changed.\n got: %v\nwant: %v\n"+
			"A new field on a response from a credential broker has to be argued for, not noticed later.",
			keys, want)
	}
}

// TestFailedCreateLogsNoSecretMaterial captures slog for the duration of two
// failing creates: one refused with 503, one that fails inside the repository
// with a genuine internal error.
//
// Stated plainly: this is an ADDITION guard. No line's deletion turns it red. It
// fires on the change it exists to catch — a handler or repository that starts
// logging the request struct, or an error that begins embedding the payload.
// Logs from a credential broker fan out to wherever logs go, and unlike a
// response they are retained.
//
// It also asserts the log carries the ERROR text, so the test cannot be
// satisfied by simply not logging anything.
func TestFailedCreateLogsNoSecretMaterial(t *testing.T) {
	capture := func(t *testing.T, fn func()) string {
		t.Helper()
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
		t.Cleanup(func() { slog.SetDefault(prev) })
		fn()
		return buf.String()
	}

	t.Run("tenant unavailable", func(t *testing.T) {
		h := newHarness(t, estatesNil)
		h.grant(t, control.MemberRoleOwner)

		var body string
		logs := capture(t, func() {
			rec := h.do(t, h.principal(), http.MethodPost, "/estates",
				createBody("prod", awsScope(), theIdentity, theSecret))
			body = rec.Body.String()
		})

		if kind := mentionsSecret(logs); kind != "" {
			t.Errorf("the 503 log line carries the submitted secret (%s encoding): %s", kind, logs)
		}
		if kind := mentionsSecret(body); kind != "" {
			t.Errorf("the 503 response carries the submitted secret (%s encoding): %s", kind, body)
		}
		if !strings.Contains(logs, "tenant database unavailable") {
			t.Errorf("nothing was logged for an unreachable tenant, so this case would also pass by "+
				"logging nothing at all: %s", logs)
		}
	})

	t.Run("internal failure", func(t *testing.T) {
		h := newHarness(t, estatesOnEmptySchema)
		h.grant(t, control.MemberRoleOwner)

		var body string
		var code int
		logs := capture(t, func() {
			rec := h.do(t, h.principal(), http.MethodPost, "/estates",
				createBody("prod", awsScope(), theIdentity, theSecret))
			body, code = rec.Body.String(), rec.Code
		})

		if code != http.StatusInternalServerError {
			t.Fatalf("a create against a tenant with no tables answered %d; this case is meant to drive "+
				"mapErr's default branch: %s", code, body)
		}
		if kind := mentionsSecret(logs); kind != "" {
			t.Errorf("the internal-error log carries the submitted secret (%s encoding): %s", kind, logs)
		}
		if kind := mentionsSecret(body); kind != "" {
			t.Errorf("the 500 response carries the submitted secret (%s encoding): %s", kind, body)
		}
		// The 500 body must be the fixed string. Returning err.Error() here is
		// how the underlying driver's message — table names, DSN fragments, and
		// in a bad case the parameters of the failing statement — reaches a
		// caller who has no business seeing any of it.
		if !strings.Contains(body, "estate operation failed") {
			t.Errorf("the 500 body is not the fixed message: %s", body)
		}
		if !strings.Contains(logs, "estate operation failed") {
			t.Errorf("nothing was logged for an internal failure, so this case would also pass by "+
				"logging nothing at all: %s", logs)
		}
	})
}

// --- integrator addition: the organization actually reaches the tenant data ---

// otherOrg creates a SECOND organization with its OWN tenant database and seeds
// the caller's membership in it, returning a principal for it. Two databases,
// because that is how this system separates tenants: EstateRepoGorm filters by
// id alone, so isolation is the tenant database, not a WHERE clause.
func (h *harness) otherOrg(t *testing.T, role control.MemberRole) *apimiddleware.Principal {
	t.Helper()

	org := control.Organization{Name: "edge-other-" + uuid.NewString(), Status: "active"}
	if err := h.ctl.Create(&org).Error; err != nil {
		t.Fatalf("create second org: %v", err)
	}
	m := control.Membership{UserID: h.userID, OrganizationID: org.ID, Role: role}
	if err := h.ctl.Create(&m).Error; err != nil {
		t.Fatalf("create second membership: %v", err)
	}

	other := openScoped(t, edgeOtherTenantSchema)
	if err := other.AutoMigrate(&tenant.Credential{}, &tenant.Estate{}); err != nil {
		t.Fatalf("migrate second tenant schema: %v", err)
	}
	if err := other.Exec(`TRUNCATE estates, credentials RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate second tenant tables: %v", err)
	}
	h.tenants[org.ID] = other

	uid := h.userID
	return &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodBearer,
		UserID:         &uid,
		OrganizationID: org.ID,
		SessionJTI:     uuid.NewString(),
	}
}

// TestEstateOperationsResolveThePrincipalsOrganization closes the hole the rest
// of this suite left open.
//
// Every other case here involves ONE organization, so none of them can tell
// p.OrganizationID apart from any other UUID: replacing all five reads of it in
// estate.go with a freshly generated organization left the entire suite green.
// The brief's first invariant — the organization comes from the PRINCIPAL and
// must be the one that reaches the data — was therefore unpinned at this layer.
//
// Two organizations, two tenant databases, the same user holding OWNER in both.
// Authorisation cannot refuse either caller and both are fully provisioned, so
// the only thing that can decide what each one sees is the organization carried
// on the principal.
func TestEstateOperationsResolveThePrincipalsOrganization(t *testing.T) {
	h := newHarness(t, estatesWired)
	h.grant(t, control.MemberRoleOwner)

	owner := h.principal()
	other := h.otherOrg(t, control.MemberRoleOwner)

	rec := h.do(t, owner, http.MethodPost, "/estates",
		createBody("prod", awsScope(), theIdentity, theSecret))
	if rec.Code != http.StatusOK {
		t.Fatalf("create answered %d: %s", rec.Code, rec.Body.String())
	}
	id := idOf(t, rec)

	// The write must have landed in the CREATING organization's database, not
	// merely somewhere. This is the assertion that fails the moment create
	// resolves any organization other than the principal's.
	var here int64
	if err := h.tdb.Model(&tenant.Estate{}).Where("id = ?", id).Count(&here).Error; err != nil {
		t.Fatalf("count estates in the creating organization's database: %v", err)
	}
	if here != 1 {
		t.Fatalf("the estate was not written to the principal's own tenant database (found %d rows); "+
			"create resolved some other organization", here)
	}

	t.Run("reads resolve the principal's organization", func(t *testing.T) {
		rec := h.do(t, owner, http.MethodGet, "/estates/"+id, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("the creating organization cannot read its own estate: %d %s", rec.Code, rec.Body.String())
		}

		rec = h.do(t, owner, http.MethodGet, "/estates", nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), id) {
			t.Fatalf("the estate is missing from its own organization's list: %d %s", rec.Code, rec.Body.String())
		}

		// The other organization is fully provisioned and its caller is an
		// OWNER, so a 404 here can only mean the lookup went to the other
		// tenant database — which is the isolation this system relies on.
		rec = h.do(t, other, http.MethodGet, "/estates/"+id, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("an owner in another organization read estate %s: got %d, want 404 — this is a "+
				"cross-tenant read of a row bound to a sealed cloud credential: %s",
				id, rec.Code, rec.Body.String())
		}

		rec = h.do(t, other, http.MethodGet, "/estates", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list answered %d for the other organization: %s", rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), id) {
			t.Fatalf("another organization's list contains estate %s: %s", id, rec.Body.String())
		}
	})

	t.Run("writes resolve the principal's organization", func(t *testing.T) {
		// Status alone is too weak a claim for a destructive operation: a
		// handler could answer 404 having already done the damage. The row
		// counts are the assertion that matters.
		rec := h.do(t, other, http.MethodPost, "/estates/"+id+"/rotate",
			map[string]any{"identity": theIdentity, "secret": "rotated-by-another-organization"})
		if rec.Code != http.StatusNotFound {
			t.Errorf("an owner in another organization rotated the credential on estate %s: got %d, want 404: %s",
				id, rec.Code, rec.Body.String())
		}

		rec = h.do(t, other, http.MethodDelete, "/estates/"+id, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("an owner in another organization deleted estate %s: got %d, want 404: %s",
				id, rec.Code, rec.Body.String())
		}

		var estates, creds int64
		if err := h.tdb.Model(&tenant.Estate{}).Where("id = ?", id).Count(&estates).Error; err != nil {
			t.Fatalf("count estates: %v", err)
		}
		if estates != 1 {
			t.Errorf("estate %s is gone from its own tenant database after another organization's "+
				"delete (count=%d)", id, estates)
		}
		if err := h.tdb.Model(&tenant.Credential{}).Count(&creds).Error; err != nil {
			t.Fatalf("count credentials: %v", err)
		}
		if creds != 1 {
			t.Errorf("credential rows in the owning tenant database = %d, want the single original: "+
				"another organization's write reached the sealed credential", creds)
		}
	})
}
