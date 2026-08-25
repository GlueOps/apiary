package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// This file drives the handler packages — internal/handlers/{auth,invitation,
// org,admin,me} — through the REAL router that api.Build assembles, rather than
// calling their closures directly.
//
// That is deliberate and is the only way several of these properties are
// observable at all. The Register* functions are thin; all of their complexity
// lives in closures that are only reachable once huma has parsed and VALIDATED
// the request and the bearer middleware has resolved a principal into the
// context. A unit test that invoked the closure with a hand-built context would
// be asserting against a request the server can never actually produce, and
// would keep passing after a change to the group an endpoint is registered on —
// which is the single most consequential mistake available in this tree.

// hhtSchema is private to this test binary invocation. Opening the DSN unscoped
// migrates and truncates tables in "public", which against a developer's own
// DATABASE_URL destroys real data.
var hhtSchema = testsupport.SchemaName("apiary_api_handler_surface_test")

const (
	hhtSecret = "test-secret"
	hhtIssuer = "apiary-test"
	hhtBase   = "/api/v1"
)

// hhtClientIP hands every request its own source address.
//
// The router rate-limits /auth/* at 20 requests per minute per client IP. These
// tests care about status codes and bodies, not about the limiter, and a suite
// that starts answering 429 halfway through would fail with a message about the
// wrong thing entirely. Per-request addresses take the limiter out of the
// picture without weakening it — TestUnmatchedPathsAreRateLimited still proves
// it works.
var hhtIPCounter atomic.Uint32

func hhtNextIP() string {
	n := hhtIPCounter.Add(1)
	return fmt.Sprintf("198.51.100.%d:%d", 1+n%250, 20000+n%40000)
}

// hhtDB opens the schema-isolated control database with every model these
// handlers touch migrated into it.
func hhtDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, hhtSchema,
		&control.User{},
		&control.Organization{},
		&control.Membership{},
		&control.TokenSession{},
		&control.EmailVerificationToken{},
		&control.AuthAccessRule{},
		&control.AuthAuditEvent{},
		&control.Invitation{},
		&control.UserTOTP{},
		&control.UserPasskey{},
		&control.ImpersonationGrant{},
		&control.EmailChangeRequest{},
	)
}

// hhtApp builds the real API on a real control database.
//
// testApp(t) in router_test.go deliberately passes nil repositories, which is
// right for the reachability sweeps — they only need to know which group an
// operation is on. It cannot be used here: every handler body below reaches a
// repository, and with a nil *gorm.DB the first query panics, the Recoverer
// turns that into a 500, and a test asserting "not 500" would then pass for the
// wrong reason. Anything asserting on a handler's DECISION needs the database.
func hhtApp(t *testing.T, db *gorm.DB) *App {
	t.Helper()

	tokenSvc, err := service.NewTokenService(hhtSecret, hhtIssuer, 15*time.Minute, 168*time.Hour)
	if err != nil {
		t.Fatalf("token service: %v", err)
	}
	userRepo := repo.NewUserRepoGorm(db)
	accessRules := repo.NewAuthAccessRuleRepoGorm(db)

	cfg := config.Config{
		JWTSecret:          hhtSecret,
		JWTIssuer:          hhtIssuer,
		BasePath:           hhtBase,
		BaseURL:            "http://localhost:8080",
		Env:                "test",
		BindHost:           "127.0.0.1",
		BindPort:           "8080",
		FrontendMode:       config.FrontendModeNone,
		CORSAllowedOrigins: "http://localhost:8080",
	}

	deps := &app.Deps{
		Config:    &cfg,
		ControlDB: db,
		TokenSessions: service.NewTokenSessionService(
			tokenSvc,
			repo.NewTokenSessionRepoGorm(db),
			repo.NewAuthAuditRepoGorm(db),
			&userRepo,
		),
		// Generous limits: these tests are about handler behaviour, not about
		// the admission ceiling, which has its own suite in internal/service.
		Admission: service.NewAdmissionService(accessRules, service.OrgBootstrapLimits{
			MaxTotalOrgs:             100000,
			MaxNewOrgsPerHour:        100000,
			MaxUnprovenOrgsPerDomain: 100000,
		}),
	}

	built, err := Build(cfg, deps)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return built
}

// hhtReq issues one request through the full middleware stack.
func hhtReq(t *testing.T, a *App, method, path, bearer string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, hhtBase+path, reader)
	req.RemoteAddr = hhtNextIP()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	a.Router.ServeHTTP(rec, req)
	return rec
}

// hhtSeedUser creates an active user. role is the PLATFORM role.
func hhtSeedUser(t *testing.T, db *gorm.DB, email string, role control.UserRole) *control.User {
	t.Helper()

	u := &control.User{Email: email, FirstName: "Test", LastName: "User", Role: role, IsActive: true}
	if err := u.SetPassword("correct-horse-battery"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return u
}

func hhtSeedOrg(t *testing.T, db *gorm.DB, name string) *control.Organization {
	t.Helper()

	org := &control.Organization{Name: name}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("create organization %s: %v", name, err)
	}
	return org
}

func hhtSeedMembership(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID, role control.MemberRole) {
	t.Helper()

	if err := db.Create(&control.Membership{UserID: userID, OrganizationID: orgID, Role: role}).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
}

// hhtSession mints a bearer token that the real middleware will accept: a
// signed access JWT plus the live token_sessions row that checkSessionLiveness
// looks the JTI up in. Both halves are required — a token with no session row
// is refused, which is the point of that check.
func hhtSession(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID) string {
	t.Helper()
	return hhtSessionAs(t, db, userID, orgID, nil)
}

// hhtSessionAs is hhtSession with an optional impersonating actor. Passing a
// non-nil actor produces the same token shape ImpersonationService issues:
// subject is the TARGET user, act_id is the operator, imp is true.
func hhtSessionAs(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID, actorID *uuid.UUID) string {
	t.Helper()

	now := time.Now().UTC()
	jti := uuid.NewString()
	session := control.TokenSession{
		SubjectID:        userID,
		SubjectKind:      "user",
		OrganizationID:   orgID,
		FamilyID:         uuid.New(),
		AccessTokenJTI:   jti,
		RefreshTokenJTI:  uuid.NewString(),
		AccessTokenHash:  "hht",
		RefreshTokenHash: "hht",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshExpiresAt: now.Add(168 * time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create token session: %v", err)
	}

	claims := &service.AuthClaims{
		OrgID: orgID.String(),
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID.String(),
			Issuer:    hhtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}
	if actorID != nil {
		claims.Impersonate = true
		claims.ActorID = actorID.String()
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(hhtSecret))
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return signed
}

func hhtEmail(prefix string) string {
	return prefix + "-" + uuid.NewString() + "@hht-example.test"
}

// hhtJSON decodes a response body, failing with the raw body when it is not
// JSON — an HTML error page or an empty body otherwise surfaces as an opaque
// "unexpected end of input".
func hhtJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not a JSON object (status %d): %v\nbody: %s", rec.Code, err, rec.Body.String())
	}
	return out
}

// hhtWalkKeys collects every object key in a decoded JSON document, at any
// depth, lower-cased.
//
// Depth matters: a token smuggled back as {"session":{"access_token":...}} is
// exactly as leaked as a top-level one, and a check that only looked at the top
// level would call it clean.
func hhtWalkKeys(v any, out map[string]bool) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			out[strings.ToLower(k)] = true
			hhtWalkKeys(child, out)
		}
	case []any:
		for _, child := range t {
			hhtWalkKeys(child, out)
		}
	}
}

// TestRegisterReturnsNoSessionTokensAnywhereInItsResponse is the HTTP-level half
// of the register/confirm split.
//
// internal/service proves the SERVICE mints no session row. This proves the
// HANDLER hands nothing session-shaped back to the caller, which is a separate
// failure with the same consequence: a token in this body is a usable session
// for an address nobody has confirmed, and confirming the address is the entire
// cost this split imposes on an attacker. Registration is unauthenticated, so
// anyone in the world can reach it.
//
// The assertion is on KEYS rather than on the two names, and it recurses. A
// future response struct that nests a session under another field, or renames
// the field to "jwt" or "session_token", must not slip past a check that only
// looked for "access_token" at the top level.
func TestRegisterReturnsNoSessionTokensAnywhereInItsResponse(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	email := hhtEmail("founder")
	rec := hhtReq(t, a, http.MethodPost, "/auth/register", "", map[string]any{
		"email":      email,
		"password":   "correct-horse-battery",
		"first_name": "Ada",
		"last_name":  "Lovelace",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /auth/register answered %d, want 201 — the registration surface is broken, "+
			"so nothing below is being tested: %s", rec.Code, rec.Body.String())
	}

	body := hhtJSON(t, rec)
	if id, _ := body["organization_id"].(string); id == "" {
		t.Fatalf("registration returned no organization_id; body: %s", rec.Body.String())
	}

	keys := map[string]bool{}
	hhtWalkKeys(body, keys)
	for _, banned := range []string{"access_token", "refresh_token", "token", "jwt", "session_token"} {
		if keys[banned] {
			t.Fatalf("POST /auth/register returned a %q key: registration is not a sign-in, and a session "+
				"handed out here is a working session for an address nobody has confirmed — which is the "+
				"whole cost the register/confirm split exists to impose. Body: %s", banned, rec.Body.String())
		}
	}

	// A cookie is the other way a session leaves this handler, and it would be
	// invisible to a body-only assertion.
	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("POST /auth/register set a cookie (%q); registration must issue no session by any route", got)
	}

	// And the ground truth behind both: no session row exists for this user.
	var user control.User
	if err := db.First(&user, "email = ?", email).Error; err != nil {
		t.Fatalf("registered user was not persisted: %v", err)
	}
	var sessions int64
	if err := db.Model(&control.TokenSession{}).Where("subject_id = ?", user.ID).Count(&sessions).Error; err != nil {
		t.Fatalf("count token sessions: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("registration created %d token_sessions rows for a brand-new unconfirmed account, want 0", sessions)
	}
}

// TestResendVerificationIsNotAMembershipOracle pins the one property that makes
// this endpoint safe to expose unauthenticated.
//
// It takes an email address, is reachable by anyone, and is the obvious place to
// ask "does this person have an account here?". The defence is that the answer
// is a constant: same status, same bytes, for a registered address and for one
// nobody has ever heard of. Anything that varies — a 404, a different message, a
// different key set — turns it into a free membership oracle over a control
// plane whose customer list is itself commercially sensitive.
//
// Both halves are asserted together, in one test, because the property is a
// COMPARISON. A test that only checked "unknown address gets 202" would stay
// green if the known-address branch started answering 200.
func TestResendVerificationIsNotAMembershipOracle(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	known := hhtEmail("registered")
	hhtSeedUser(t, db, known, control.RolePlatformCustomer)

	unknown := hhtEmail("never-registered")

	knownRec := hhtReq(t, a, http.MethodPost, "/auth/resend-verification", "", map[string]any{"email": known})
	unknownRec := hhtReq(t, a, http.MethodPost, "/auth/resend-verification", "", map[string]any{"email": unknown})

	if unknownRec.Code != http.StatusAccepted {
		t.Fatalf("POST /auth/resend-verification answered %d for an address with no account, want 202 — "+
			"an unauthenticated endpoint that answers differently for a registered address is a membership "+
			"oracle: %s", unknownRec.Code, unknownRec.Body.String())
	}
	if knownRec.Code != unknownRec.Code {
		t.Fatalf("resend answered %d for a registered address and %d for an unregistered one: the status "+
			"code alone reveals whether an address has an account here",
			knownRec.Code, unknownRec.Code)
	}
	if knownRec.Body.String() != unknownRec.Body.String() {
		t.Fatalf("resend answered different bodies for a registered and an unregistered address:\n"+
			"  registered:   %s\n  unregistered: %s\n"+
			"Any difference at all makes this endpoint a membership oracle.",
			knownRec.Body.String(), unknownRec.Body.String())
	}

	// The response must not echo the submitted address either: that would make
	// a logged or cached response a record of who was probed, and it is the
	// easiest accidental way to reintroduce a difference between the two.
	if strings.Contains(strings.ToLower(knownRec.Body.String()), strings.ToLower(known)) {
		t.Fatalf("resend echoed the submitted address back: %s", knownRec.Body.String())
	}
}

// TestAcceptSurfaceIsReachableWithoutASessionAndOrgSurfaceIsNot checks
// reachability by RESULT, not by status class.
//
// TestInvitationSurfaceKeepsItsPublicPrivateSplit in router_test.go already
// proves the accept endpoints do not answer 401. This goes one step further and
// requires that they reached their handler and produced the handler's OWN
// refusal for a token that does not exist — 404 from mapAcceptErr. That
// distinction matters because 500 is also "not 401": the endpoint could be
// public, unreachable in practice, and still pass a 401-only check. A public
// endpoint that is public and BROKEN strands every invitee.
func TestAcceptSurfaceIsReachableWithoutASessionAndOrgSurfaceIsNot(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	// A syntactically fine token that was never issued.
	bogus := strings.ReplaceAll(uuid.NewString(), "-", "")

	t.Run("preview", func(t *testing.T) {
		rec := hhtReq(t, a, http.MethodGet, "/invitations/preview?token="+bogus, "", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET /invitations/preview with an unknown token answered %d, want 404. "+
				"401 means the accept surface moved behind the bearer middleware, which breaks the flow "+
				"for the people it exists for — an invitee has no account yet. 5xx means it is public but "+
				"broken, which strands them just the same: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("accept", func(t *testing.T) {
		rec := hhtReq(t, a, http.MethodPost, "/invitations/accept", "", map[string]any{"token": bogus})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("POST /invitations/accept with an unknown token answered %d, want 404 — see the "+
				"preview case above: %s", rec.Code, rec.Body.String())
		}
	})

	// The other side of the same boundary. These read or grant organization
	// membership and must be unreachable without a session, whatever the body
	// says. Asserted here alongside the public pair so that the split is one
	// test's subject rather than an invariant split across two files.
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"list invitations", http.MethodGet, "/invitations", nil},
		{"create invitation", http.MethodPost, "/invitations", map[string]any{"email": hhtEmail("invitee")}},
		{"list members", http.MethodGet, "/organization/members", nil},
		{"start impersonation", http.MethodPost, "/admin/impersonate", map[string]any{
			"user_id":         uuid.NewString(),
			"organization_id": uuid.NewString(),
			"reason":          "a stated reason",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := hhtReq(t, a, tc.method, tc.path, "", tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s answered an anonymous caller with %d, want 401 — it reads or grants "+
					"organization membership and has been registered on the public group: %s",
					tc.method, tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestForbiddenIsIdenticalForNonMemberAndWrongRole is the membership-oracle test
// for the authenticated surface.
//
// A caller holding a valid session for organization X, who is not in X at all,
// and a caller who IS in X but only as a member, must receive the same refusal
// byte for byte. If they differ, any authenticated user can point a token at any
// organization id and learn from the response whether they hold a membership
// there — and in this system a membership is a view onto another tenant's cloud
// credentials, so confirming one is worth having.
//
// AssertOrgRole returns two distinct sentinels here on purpose (ErrNoMembership
// and ErrInsufficientRole) so the server can log which happened; OrgRoleError
// collapses them for the client. This test is what holds that collapse in place,
// and it is checked on all three admin-gated organization endpoints because the
// mapping is applied per handler and can be broken in one without the others.
func TestForbiddenIsIdenticalForNonMemberAndWrongRole(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Oracle Test Org")

	// In the organization, but only a member: below the admin bar.
	insider := hhtSeedUser(t, db, hhtEmail("insider"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, insider.ID, org.ID, control.MemberRoleMember)
	insiderToken := hhtSession(t, db, insider.ID, org.ID)

	// Not in the organization at all, but their token names it — this is the
	// shape of a deliberate probe.
	outsider := hhtSeedUser(t, db, hhtEmail("outsider"), control.RolePlatformCustomer)
	outsiderToken := hhtSession(t, db, outsider.ID, org.ID)

	membershipID := uuid.NewString()

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"create invitation", http.MethodPost, "/invitations", map[string]any{"email": hhtEmail("invitee"), "role": "member"}},
		{"list invitations", http.MethodGet, "/invitations", nil},
		{"update member role", http.MethodPatch, "/organization/members/" + membershipID, map[string]any{"role": "admin"}},
		{"remove member", http.MethodDelete, "/organization/members/" + membershipID, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outsiderRec := hhtReq(t, a, tc.method, tc.path, outsiderToken, tc.body)
			insiderRec := hhtReq(t, a, tc.method, tc.path, insiderToken, tc.body)

			if outsiderRec.Code != http.StatusForbidden {
				t.Fatalf("%s %s answered %d for a caller holding NO membership in the organization their "+
					"token names, want 403. 401 would be wrong (their session is valid), 404 or 200 would "+
					"be a great deal worse: %s", tc.method, tc.path, outsiderRec.Code, outsiderRec.Body.String())
			}
			if insiderRec.Code != outsiderRec.Code {
				t.Fatalf("%s %s answered %d for a member whose role is too low and %d for someone with no "+
					"membership at all: the status code alone tells any authenticated caller whether they "+
					"hold a membership in an organization they named",
					tc.method, tc.path, insiderRec.Code, outsiderRec.Code)
			}
			if insiderRec.Body.String() != outsiderRec.Body.String() {
				t.Fatalf("%s %s answered different 403 bodies for wrong-role and no-membership:\n"+
					"  no membership: %s\n  wrong role:    %s\n"+
					"The bodies must be identical or the endpoint is a membership oracle.",
					tc.method, tc.path, outsiderRec.Body.String(), insiderRec.Body.String())
			}

			// The refusal must also not name the organization, which would
			// confirm the id resolves to something real.
			if strings.Contains(outsiderRec.Body.String(), org.Name) {
				t.Fatalf("%s %s named the organization in its refusal (%s): a caller with no membership "+
					"learns the organization exists and what it is called",
					tc.method, tc.path, outsiderRec.Body.String())
			}
		})
	}
}

// TestPlatformStaffSurfaceRefusesAnOrdinaryCustomer covers the other role
// ladder.
//
// /admin/impersonate is gated on the PLATFORM role held on the user record, not
// on a membership role. The two ladders are unrelated, and conflating them is
// the escalation this endpoint is one line away from: everyone is an OWNER of
// the organization they created at registration, and owner outranks admin on the
// membership ladder — so an admin check that read a membership role would hand
// every customer in the system the ability to mint a session as any user.
func TestPlatformStaffSurfaceRefusesAnOrdinaryCustomer(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Customer Org")
	// An OWNER — the top of the membership ladder — to make the point that the
	// membership ladder buys nothing at all here.
	customer := hhtSeedUser(t, db, hhtEmail("owner-customer"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, customer.ID, org.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, customer.ID, org.ID)

	target := hhtSeedUser(t, db, hhtEmail("target"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, target.ID, org.ID, control.MemberRoleMember)

	rec := hhtReq(t, a, http.MethodPost, "/admin/impersonate", token, map[string]any{
		"user_id":         target.ID.String(),
		"organization_id": org.ID.String(),
		"reason":          "investigating a support ticket",
	})

	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST /admin/impersonate answered %d for an ordinary customer who is an OWNER of their own "+
			"organization, want 403. A 2xx here means any customer can mint a session as any user in the "+
			"system: %s", rec.Code, rec.Body.String())
	}
	body := strings.ToLower(rec.Body.String())
	if strings.Contains(body, target.Email) {
		t.Fatalf("the refusal echoed the target's email address (%s), confirming to a non-staff caller that "+
			"the account exists: %s", target.Email, rec.Body.String())
	}
}

// TestMalformedPathUUIDIsRejectedAsUnprocessable pins the status for a path
// parameter that is not a uuid.
//
// Two independent layers produce it: huma validates the `format:"uuid"` on the
// parameter, and the handler parses it again and answers 422 itself. Either
// alone is enough, which is precisely why this is worth pinning — the belt can
// be removed without anyone noticing the braces are gone too. The failure mode
// it prevents is a 500: an unhandled parse error on a path segment an
// unauthenticated scanner controls turns every malformed URL into a logged
// stack trace and a page on someone's error dashboard.
//
// The caller here is an ADMIN of the organization. That is load-bearing: the
// role assertion runs before the id is parsed, so a lesser caller would be
// refused at 403 and this test would prove nothing about the parse.
func TestMalformedPathUUIDIsRejectedAsUnprocessable(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Malformed Path Org")
	admin := hhtSeedUser(t, db, hhtEmail("admin"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, admin.ID, org.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, admin.ID, org.ID)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"revoke invitation", http.MethodDelete, "/invitations/not-a-uuid", nil},
		{"update member role", http.MethodPatch, "/organization/members/not-a-uuid", map[string]any{"role": "admin"}},
		{"remove member", http.MethodDelete, "/organization/members/not-a-uuid", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := hhtReq(t, a, tc.method, tc.path, token, tc.body)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("%s %s answered %d, want 422. A 5xx means a path segment any caller controls "+
					"reaches the database layer unparsed: %s", tc.method, tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestGetMeReturnsTheCallersOwnProfileAndNoCredentialMaterial exercises the /me
// surface, which is the most-called authenticated endpoint in the product and
// had no request-level test at all.
//
// Two properties, and the second is the one that matters. The profile is the
// caller's OWN — resolved from the principal the middleware put in the context,
// never from anything in the request — and it carries no credential material.
// The service loads the full control.User row, so a handler that marshalled the
// model instead of copying fields across would put the whole row on the wire.
// TWO layers stop the bcrypt hash going with it: this handler copying named
// fields, and control.User tagging Password and TokensRevokedAt as json:"-".
// The denylist below only fires once BOTH are gone, so it is paired with an
// allowlist on the top-level key set, which fails the moment the response
// carries anything GetMeResponse does not declare — including a wholesale
// embedding of the model that the tags happen to render harmless today.
func TestGetMeReturnsTheCallersOwnProfileAndNoCredentialMaterial(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Profile Org")
	user := hhtSeedUser(t, db, hhtEmail("profile"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, user.ID, org.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, user.ID, org.ID)

	rec := hhtReq(t, a, http.MethodGet, "/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me answered %d for a valid session, want 200: %s", rec.Code, rec.Body.String())
	}

	body := hhtJSON(t, rec)
	if got, _ := body["id"].(string); got != user.ID.String() {
		t.Fatalf("GET /me returned id %q, want the caller's own id %q — the profile must come from the "+
			"session's principal", got, user.ID.String())
	}
	if got, _ := body["email"].(string); got != user.Email {
		t.Fatalf("GET /me returned email %q, want %q", got, user.Email)
	}

	keys := map[string]bool{}
	hhtWalkKeys(body, keys)
	for _, banned := range []string{"password", "password_hash", "tokens_revoked_at", "connection_string"} {
		if keys[banned] {
			t.Fatalf("GET /me exposed a %q key: the response is being built from the control.User model "+
				"rather than copied into the response struct, and credential material is now in the "+
				"browser. Body: %s", banned, rec.Body.String())
		}
	}
	// Belt and braces: the stored bcrypt hash must not appear under ANY key.
	if user.Password != "" && strings.Contains(rec.Body.String(), user.Password) {
		t.Fatalf("GET /me returned the caller's stored password hash in its body: %s", rec.Body.String())
	}

	// The allowlist. A denylist can only refuse the names someone thought of,
	// and the regression this test exists for — handing back the control.User
	// row rather than copying fields out of it — arrives under whatever name the
	// field was given. Pinning the top-level key set to exactly what
	// me.GetMeResponse declares means any extra key fails here, whether or not
	// its contents happen to be harmless. Adding a field to the response is then
	// a deliberate act that updates this list, which is the point.
	declared := map[string]bool{
		"$schema": true, "id": true, "email": true, "first_name": true, "last_name": true,
		"role": true, "is_active": true, "mfa": true, "organization": true,
		"last_login_at": true, "created_at": true,
	}
	for k := range body {
		if !declared[strings.ToLower(k)] {
			t.Fatalf("GET /me returned an undeclared top-level key %q. Every key here must be one "+
				"me.GetMeResponse names: an extra one means the response is carrying something the "+
				"handler did not copy in field by field — most likely the control.User row itself, which "+
				"is one dropped json:\"-\" tag away from being the caller's password hash. Body: %s",
				k, rec.Body.String())
		}
	}
}

// TestImpersonatedSessionMayNotRemoveTheUsersPasskey covers the guard on the
// highest-complexity handler in the auth package, and the one place an operator
// could quietly weaken an account they are meant to be helping.
//
// Impersonation issues a session whose SUBJECT is the customer, so every
// ordinary authorization check the customer would pass, the operator passes too.
// The only thing standing between "support engineer troubleshooting an account"
// and "support engineer deleting that account's second factor" is the explicit
// IsImpersonating() refusal in this handler. Removing an MFA factor is not
// troubleshooting; it is a durable reduction in the account's security that
// outlives the impersonated session.
//
// The two calls are the test. Asserting only that an impersonated caller gets
// 403 would pass just as well if the endpoint were broken for everyone, so the
// same request is made with an ordinary session and required to get PAST the
// impersonation gate — 404 for a passkey id that does not exist. It is the
// DIFFERENCE between the two that shows the gate is about impersonation.
//
// That second call found a real bug on its first run: it answered 500, not 404.
// PasskeyService.Delete passed the repository's gorm.ErrRecordNotFound straight
// through, while the handler tests for service.ErrPasskeyNotFound — a sentinel
// that nothing in the tree ever returned — so the 404 branch was unreachable and
// every delete of an id that does not resolve was reported as a server fault.
// Fixed in internal/service/passkey_service.go by translating at the boundary
// that owns the sentinel. Keep this assertion at 404: relaxing it to "not 403"
// would let the bug back in.
func TestImpersonatedSessionMayNotRemoveTheUsersPasskey(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Impersonation Org")
	target := hhtSeedUser(t, db, hhtEmail("target"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, target.ID, org.ID, control.MemberRoleOwner)
	operator := hhtSeedUser(t, db, hhtEmail("operator"), control.RolePlatformSupport)

	passkeyID := uuid.NewString()

	impersonated := hhtSessionAs(t, db, target.ID, org.ID, &operator.ID)
	rec := hhtReq(t, a, http.MethodDelete, "/auth/passkeys/"+passkeyID, impersonated, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("DELETE /auth/passkeys/{id} answered %d for an IMPERSONATED session, want 403. "+
			"An operator acting as a customer must not be able to remove that customer's second factor: "+
			"it is a durable weakening of the account that outlives the impersonated session, and the "+
			"customer has no way to see it happen: %s", rec.Code, rec.Body.String())
	}

	// The control: the same request from the customer's own session must clear
	// the impersonation gate and be refused for the ordinary reason instead.
	// Without this, an endpoint that 403'd everybody would pass the check above.
	ownSession := hhtSession(t, db, target.ID, org.ID)
	ownRec := hhtReq(t, a, http.MethodDelete, "/auth/passkeys/"+passkeyID, ownSession, nil)
	if ownRec.Code != http.StatusNotFound {
		t.Fatalf("DELETE /auth/passkeys/{id} answered %d for the user's OWN session on an id that does not "+
			"exist, want 404. If this is also 403 then the assertion above proves nothing about "+
			"impersonation — the endpoint simply refuses everyone: %s", ownRec.Code, ownRec.Body.String())
	}
}
