package apimiddleware

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// This file covers the four functions that decide WHO the caller is:
// parseAccessClaims, isPlatformAdmin, RequireUserAuth and RequirePlatformAdmin,
// plus AssertPlatformRole which decides whether that caller is staff.
//
// Everything here is a security test. The properties being pinned down are:
//
//  1. A token that is unsigned, wrongly signed, expired, not yet valid, issued
//     by someone else, or of the wrong type is REJECTED. In particular alg=none
//     — the classic JWT forgery — must never authenticate anyone.
//  2. A rejection returns nil claims, never a zero-value *AuthClaims. A caller
//     that ignored the error and used the claims anyway would otherwise get a
//     principal with an empty subject and an empty org, and empty is a value
//     that compares equal to other empties.
//  3. The PLATFORM role ladder (control.UserRole: admin/support/sales/customer)
//     and the MEMBERSHIP ladder (owner/admin/member) are unrelated. Anyone is
//     the owner of an organization they created, so a staff check that admitted
//     a membership role would hand every customer the admin surface.
//  4. An impersonated session is not staff, whatever role the impersonated
//     subject holds.
//
// The DB-backed cases reuse this package's existing schema-scoped helpers
// (testDB, seedRevokedSession) rather than opening the raw DSN: an unscoped
// open migrates and truncates in "public", which against a developer's own
// DATABASE_URL is their real control database.

const (
	amwSecret = "apimiddleware-auth-claims-suite-secret"
	amwIssuer = "apiary-auth-claims-test"
)

// amwTokenSpec is every knob an attacker could turn on a token. Zero values are
// filled in with the values of a legitimate access token by amwMint, so each
// test names ONLY the thing it is subverting and the rest is known-good.
type amwTokenSpec struct {
	subject     string
	orgID       string
	jti         string
	typ         string
	issuer      string
	secret      string
	method      jwt.SigningMethod
	issuedAt    time.Time
	notBefore   time.Time
	expiresAt   time.Time
	mfaEnabled  bool
	mfa         bool
	impersonate bool
	actorID     string
	omitType    bool
}

// amwMint signs a token the way TokenService does, so the result is
// indistinguishable from production output except for the fields under test.
func amwMint(t *testing.T, spec amwTokenSpec) string {
	t.Helper()

	now := time.Now().UTC()
	if spec.subject == "" {
		spec.subject = uuid.NewString()
	}
	if spec.orgID == "" {
		spec.orgID = uuid.NewString()
	}
	if spec.jti == "" {
		spec.jti = uuid.NewString()
	}
	if spec.typ == "" && !spec.omitType {
		spec.typ = "access"
	}
	if spec.issuer == "" {
		spec.issuer = amwIssuer
	}
	if spec.secret == "" {
		spec.secret = amwSecret
	}
	if spec.method == nil {
		spec.method = jwt.SigningMethodHS256
	}
	if spec.issuedAt.IsZero() {
		spec.issuedAt = now
	}
	if spec.notBefore.IsZero() {
		spec.notBefore = now
	}
	if spec.expiresAt.IsZero() {
		spec.expiresAt = now.Add(15 * time.Minute)
	}

	claims := service.AuthClaims{
		OrgID:       spec.orgID,
		Type:        spec.typ,
		MFA:         spec.mfa,
		MFAEnabled:  spec.mfaEnabled,
		ActorID:     spec.actorID,
		Impersonate: spec.impersonate,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        spec.jti,
			Subject:   spec.subject,
			Issuer:    spec.issuer,
			IssuedAt:  jwt.NewNumericDate(spec.issuedAt),
			NotBefore: jwt.NewNumericDate(spec.notBefore),
			ExpiresAt: jwt.NewNumericDate(spec.expiresAt),
		},
	}

	tok := jwt.NewWithClaims(spec.method, claims)

	// alg=none is the forgery this suite most needs to be able to express, and
	// the library requires an explicit sentinel key to produce one at all.
	var key any = []byte(spec.secret)
	if spec.method == jwt.SigningMethodNone {
		key = jwt.UnsafeAllowNoneSignatureType
	}

	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return signed
}

func amwDeps(db *gorm.DB) *app.Deps {
	return &app.Deps{
		Config: &config.Config{
			JWTSecret: amwSecret,
			JWTIssuer: amwIssuer,
		},
		ControlDB: db,
	}
}

// amwSeedUser creates a user with an explicit platform role. Role is taken as a
// string-typed value rather than a control.UserRole constant so a test can seed
// a MEMBERSHIP role ("owner") into the platform-role column and prove the two
// ladders stay separate.
func amwSeedUser(t *testing.T, db *gorm.DB, role control.UserRole, active bool) uuid.UUID {
	t.Helper()

	user := control.User{
		Email:     "amw-" + uuid.NewString() + "@example.test",
		Password:  "x",
		FirstName: "Amw",
		LastName:  "Probe",
		Role:      role,
		IsActive:  active,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user with role %q: %v", role, err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&control.User{}, "id = ?", user.ID) })

	// IsActive carries `gorm:"default:true"`, so GORM omits the column on insert
	// when the value is the zero false and the row comes back ACTIVE. Seeding a
	// deactivated user by struct alone therefore silently seeds an active one,
	// and the "deactivated user is locked out" cases would pass against a
	// middleware with no is_active check at all. Force it with an explicit write
	// and verify, because a fixture that lies makes the test meaningless.
	if !active {
		if err := db.Model(&control.User{}).Where("id = ?", user.ID).
			Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate seeded user: %v", err)
		}
		var check control.User
		if err := db.Select("is_active").First(&check, "id = ?", user.ID).Error; err != nil {
			t.Fatalf("re-read seeded user: %v", err)
		}
		if check.IsActive {
			t.Fatal("fixture did not take: the user is still active, so the deactivation case would " +
				"pass vacuously")
		}
	}
	return user.ID
}

// amwSeedSession gives a user a live (unrevoked, unexpired) session row, which
// is what checkSessionLiveness demands before any middleware will admit a
// token. Returns the access JTI to put in the token's jti claim.
func amwSeedSession(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID) string {
	t.Helper()

	now := time.Now().UTC()
	session := control.TokenSession{
		SubjectID:        userID,
		SubjectKind:      "user",
		OrganizationID:   orgID,
		FamilyID:         uuid.New(),
		AccessTokenJTI:   uuid.NewString(),
		RefreshTokenJTI:  uuid.NewString(),
		AccessTokenHash:  "h",
		RefreshTokenHash: "h",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshExpiresAt: now.Add(168 * time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&control.TokenSession{}, "id = ?", session.ID) })
	return session.AccessTokenJTI
}

// amwStubUsers is a repo.UserRepo whose only implemented method is GetByID.
//
// The embedded nil interface is deliberate: if AssertPlatformRole ever starts
// calling something else on the repository, the test panics with the method
// name instead of silently passing. calls counts lookups so a test can assert
// that a refusal happened BEFORE the database was consulted.
type amwStubUsers struct {
	repo.UserRepo
	user  *control.User
	err   error
	calls int
}

func (s *amwStubUsers) GetByID(context.Context, uuid.UUID) (*control.User, error) {
	s.calls++
	return s.user, s.err
}

// amwProbeOutput is the body of the handlers the middleware tests guard. It
// echoes the principal the middleware installed, because "the request was
// allowed" is a much weaker claim than "the request was allowed AS this user".
type amwProbeOutput struct {
	Body struct {
		UserID        string `json:"user_id"`
		OrgID         string `json:"org_id"`
		JTI           string `json:"jti"`
		Impersonating bool   `json:"impersonating"`
	}
}

// amwMiddleware is the shape of the two auth middlewares under test. They are
// passed as a constructor rather than an already-built middleware because both
// need the huma.API they will be mounted on in order to write an error response
// at all — handing them a nil API makes every refusal path panic instead of
// returning its status, which would turn a working guard into a 500.
type amwMiddleware func(huma.API, *app.Deps) func(huma.Context, func(huma.Context))

// amwProbeAPI registers a handler behind the supplied middleware. reached
// reports whether the handler body ran, and seen captures the principal.
func amwProbeAPI(t *testing.T, mw amwMiddleware, deps *app.Deps) (humatest.TestAPI, *bool, **Principal) {
	t.Helper()

	reached := false
	var seen *Principal

	_, api := humatest.New(t, huma.DefaultConfig("amw-probe", "1.0.0"))
	api.UseMiddleware(mw(api, deps))
	huma.Register(api, huma.Operation{
		OperationID: "amw-probe",
		Method:      http.MethodGet,
		Path:        "/amw-probe",
	}, func(ctx context.Context, _ *struct{}) (*amwProbeOutput, error) {
		reached = true
		if p, ok := PrincipalFromContext(ctx); ok {
			seen = p
		}
		out := &amwProbeOutput{}
		if seen != nil {
			if seen.UserID != nil {
				out.Body.UserID = seen.UserID.String()
			}
			out.Body.OrgID = seen.OrganizationID.String()
			out.Body.JTI = seen.SessionJTI
			out.Body.Impersonating = seen.IsImpersonating()
		}
		return out, nil
	})
	return api, &reached, &seen
}

// --- parseAccessClaims ------------------------------------------------------

func TestParseAccessClaims_AcceptsAWellFormedAccessToken(t *testing.T) {
	// The positive control. Without it every rejection case below would still
	// pass against a parseAccessClaims whose body is `return nil, ErrAccessDenied`,
	// and a function that denies everything is not an authentication check, it
	// is an outage.
	subject := uuid.NewString()
	jti := uuid.NewString()

	claims, err := parseAccessClaims(amwMint(t, amwTokenSpec{subject: subject, jti: jti}), amwDeps(nil))
	if err != nil {
		t.Fatalf("a correctly signed, unexpired access token from the configured issuer must be accepted: %v", err)
	}
	if claims == nil {
		t.Fatal("parseAccessClaims returned nil claims with a nil error; every caller dereferences them")
	}
	if claims.Subject != subject {
		t.Fatalf("claims must carry the token's subject through unchanged — the subject IS the identity; got %q want %q",
			claims.Subject, subject)
	}
	if claims.ID != jti {
		t.Fatalf("claims must carry the jti through: it is the handle checkSessionLiveness uses to find the "+
			"session row, so losing it would make revocation unenforceable; got %q want %q", claims.ID, jti)
	}
}

func TestParseAccessClaims_RejectsForgedUnusableAndWrongKindTokens(t *testing.T) {
	// Each row is a way a caller could try to authenticate as someone else, or
	// with a credential that should no longer work. All of them must land on
	// ErrAccessDenied specifically, and all of them must return NIL claims —
	// see the sentinel/nil assertions below for why the nil half matters.
	past := time.Now().UTC().Add(-2 * time.Hour)

	tests := map[string]struct {
		token string
		why   string
	}{
		"empty string": {
			token: "",
			why:   "an absent token is not an anonymous token; it must not parse to a zero-value identity",
		},
		"whitespace only": {
			token: "   \t ",
			why:   "a header of spaces trims to empty and must be treated as absent, not handed to the parser",
		},
		"not a jwt at all": {
			token: "hunter2",
			why:   "arbitrary strings must not authenticate",
		},
		"alg=none": {
			token: amwMint(t, amwTokenSpec{method: jwt.SigningMethodNone}),
			why: "alg=none is the classic JWT forgery: the attacker writes whatever subject and role they " +
				"like and attaches no signature. Accepting it is total authentication bypass",
		},
		"signed with the wrong secret": {
			token: amwMint(t, amwTokenSpec{secret: "the-attackers-own-secret"}),
			why:   "a signature that does not verify under the configured secret is a forgery",
		},
		"HS512 instead of HS256": {
			token: amwMint(t, amwTokenSpec{method: jwt.SigningMethodHS512}),
			why: "the algorithm is pinned. Letting the token choose its own algorithm is how algorithm-" +
				"confusion attacks start, even when the key material happens to be the same",
		},
		"HS384 instead of HS256": {
			token: amwMint(t, amwTokenSpec{method: jwt.SigningMethodHS384}),
			why:   "same pinning property as HS512, for the other HMAC variant",
		},
		"issued by a different issuer": {
			token: amwMint(t, amwTokenSpec{issuer: "some-other-apiary"}),
			why: "a token minted by another deployment sharing the secret (staging, a fork, a copied " +
				"config) must not be usable here",
		},
		"empty issuer": {
			token: amwMint(t, amwTokenSpec{issuer: " "}),
			why:   "an unclaimed issuer is not a match for the configured one",
		},
		"expired": {
			token: amwMint(t, amwTokenSpec{issuedAt: past, notBefore: past, expiresAt: past.Add(time.Minute)}),
			why:   "expiry is the only bound on a stolen access token; ignoring exp makes theft permanent",
		},
		"not yet valid": {
			token: amwMint(t, amwTokenSpec{notBefore: time.Now().UTC().Add(time.Hour)}),
			why:   "an nbf in the future means the token is not usable yet",
		},
		"refresh token presented as an access token": {
			token: amwMint(t, amwTokenSpec{typ: "refresh"}),
			why: "refresh tokens are long-lived and are stored where access tokens are not. Accepting one " +
				"on the API surface would extend a refresh token's reach to every endpoint",
		},
		"missing typ claim": {
			token: amwMint(t, amwTokenSpec{omitType: true}),
			why:   "an unlabelled token is not an access token; the check must be positive, not a deny-list",
		},
		"mfa enrolled but not satisfied": {
			token: amwMint(t, amwTokenSpec{mfaEnabled: true, mfa: false}),
			why: "a user who has enrolled in MFA has a half-authenticated token between password and code. " +
				"Honouring it would make enrolling in MFA a no-op",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			claims, err := parseAccessClaims(tc.token, amwDeps(nil))

			if !errors.Is(err, ErrAccessDenied) {
				t.Fatalf("%s\nwant ErrAccessDenied, got err=%v", tc.why, err)
			}
			// The nil half is not pedantry. Callers in this package do
			// `claims, err := parseAccessClaims(...)`, and a future caller that
			// mishandles err would, if a zero-value *AuthClaims came back, get
			// Subject=="" and OrgID=="" — an identity that silently matches
			// every other empty identity. Denying must mean handing back
			// nothing to fall through with.
			if claims != nil {
				t.Fatalf("parseAccessClaims denied but still returned claims %+v; a rejected token must "+
					"yield NO principal to fall through with, not a zero-value one", claims)
			}
		})
	}
}

func TestParseAccessClaims_AllowsMFASatisfiedToken(t *testing.T) {
	// The counterpart to the "mfa enrolled but not satisfied" row: the guard has
	// to be `enrolled AND NOT satisfied`, not `enrolled`. If it refused every
	// MFA user, MFA enrolment would lock people out and the fix would be to
	// remove the check.
	claims, err := parseAccessClaims(amwMint(t, amwTokenSpec{mfaEnabled: true, mfa: true}), amwDeps(nil))
	if err != nil {
		t.Fatalf("a token from a completed MFA login must be accepted, or enrolling in MFA locks the user out: %v", err)
	}
	if claims == nil || !claims.MFA {
		t.Fatalf("the mfa claim must survive parsing; got %+v", claims)
	}
}

// --- isPlatformAdmin --------------------------------------------------------

func TestIsPlatformAdmin_OnlyAdminAndSupportPass(t *testing.T) {
	// The decision matrix. The rows that matter most are the membership roles:
	// "owner" and "member" are values from control.MemberRole, an unrelated
	// ladder. Anyone who creates an organization is its owner, so if "owner"
	// satisfied this check every self-service signup would hold the platform
	// admin surface. ("admin" is intentionally absent from that group because
	// the two ladders share the literal string, so it proves nothing.)
	db := testDB(t)

	tests := map[string]struct {
		role  control.UserRole
		allow bool
		why   string
	}{
		"platform admin": {
			role: control.RolePlatformAdmin, allow: true,
			why: "admin is the role the platform surface exists for",
		},
		"platform support": {
			role: control.RolePlatformSupport, allow: true,
			why: "support staff are explicitly admitted by isPlatformAdmin",
		},
		"platform sales": {
			role: control.RolePlatformSales, allow: false,
			why: "sales is staff but NOT admin staff; the ladder is not ordered by seniority here",
		},
		"customer": {
			role: control.RolePlatformCustomer, allow: false,
			why: "the default role for every signup must never reach the platform surface",
		},
		"membership role owner": {
			role: control.UserRole(control.MemberRoleOwner), allow: false,
			why: "owner is a MEMBERSHIP role and anyone is owner of an org they created; admitting it " +
				"here would be self-service privilege escalation to platform admin",
		},
		"membership role member": {
			role: control.UserRole(control.MemberRoleMember), allow: false,
			why: "member is a MEMBERSHIP role and has no meaning on the platform ladder",
		},
		"empty role": {
			role: control.UserRole(""), allow: false,
			why: "an unset role must deny, not match by being falsy",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userID := amwSeedUser(t, db, tc.role, true)
			claims := claimsFor(userID, uuid.NewString())

			got, err := isPlatformAdmin(context.Background(), amwDeps(db), claims)

			if tc.allow {
				if err != nil {
					t.Fatalf("%s\nrole %q was refused the platform surface: %v", tc.why, tc.role, err)
				}
				if got != userID {
					t.Fatalf("isPlatformAdmin must return the ID it authorised — it is what the caller "+
						"builds the Principal from; got %s want %s", got, userID)
				}
				return
			}
			if !errors.Is(err, ErrAccessDenied) {
				t.Fatalf("%s\nrole %q reached the platform surface (err=%v)", tc.why, tc.role, err)
			}
			if got != uuid.Nil {
				t.Fatalf("a refusal must not hand back a usable user id; got %s", got)
			}
		})
	}
}

func TestIsPlatformAdmin_RefusesEvenAValidAdminInEveryDisqualifyingCase(t *testing.T) {
	db := testDB(t)

	tests := map[string]struct {
		claims *service.AuthClaims
		deps   *app.Deps
		why    string
	}{
		"impersonated session belonging to an admin": {
			claims: func() *service.AuthClaims {
				c := claimsFor(amwSeedUser(t, db, control.RolePlatformAdmin, true), uuid.NewString())
				c.Impersonate = true
				return c
			}(),
			why: "an impersonation session's SUBJECT is the target user. If the target happens to be " +
				"staff, honouring it would let one impersonation start another and the audit trail " +
				"would name the wrong actor",
		},
		"deactivated admin": {
			claims: claimsFor(amwSeedUser(t, db, control.RolePlatformAdmin, false), uuid.NewString()),
			why: "deactivating an account is the offboarding lever; a still-valid token must stop " +
				"working the moment is_active goes false",
		},
		"subject is not a uuid": {
			claims: func() *service.AuthClaims {
				c := claimsFor(uuid.New(), uuid.NewString())
				c.Subject = "admin"
				return c
			}(),
			why: "a non-uuid subject must deny rather than be coerced or passed to the query",
		},
		"subject is empty": {
			claims: func() *service.AuthClaims {
				c := claimsFor(uuid.New(), uuid.NewString())
				c.Subject = ""
				return c
			}(),
			why: "an empty subject identifies nobody",
		},
		"subject names a user that does not exist": {
			claims: claimsFor(uuid.New(), uuid.NewString()),
			why:    "a token for a deleted user must not authorise; the lookup missing is a denial, not a pass",
		},
		"control db is not wired": {
			claims: claimsFor(amwSeedUser(t, db, control.RolePlatformAdmin, true), uuid.NewString()),
			deps:   &app.Deps{Config: &config.Config{}},
			why: "a check that cannot reach the database cannot reach a decision, and a check that " +
				"cannot decide must deny. Failing open here would make a DB outage an open admin API",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deps := tc.deps
			if deps == nil {
				deps = amwDeps(db)
			}

			got, err := isPlatformAdmin(context.Background(), deps, tc.claims)

			if !errors.Is(err, ErrAccessDenied) {
				t.Fatalf("%s\nwant ErrAccessDenied, got err=%v", tc.why, err)
			}
			if got != uuid.Nil {
				t.Fatalf("%s\na refusal handed back user id %s", tc.why, got)
			}
		})
	}
}

// --- RequireUserAuth --------------------------------------------------------

func TestRequireUserAuth_AdmitsAValidSessionAsTheRightPrincipal(t *testing.T) {
	// The positive control for every 401 below, and the assertion that the
	// middleware installs the identity the TOKEN names — not merely that it let
	// the request through.
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, reached, seen := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
	}))

	if resp.Code != http.StatusOK {
		t.Fatalf("a valid, live, non-impersonated session must be admitted, got %d: %s", resp.Code, resp.Body.String())
	}
	if !*reached {
		t.Fatal("the handler never ran for a valid session")
	}
	p := *seen
	if p == nil {
		t.Fatal("RequireUserAuth admitted the request without installing a Principal; every downstream " +
			"authorization check reads the principal, and a missing one is an unauthenticated request " +
			"wearing an authenticated one's clothes")
	}
	if p.UserID == nil || *p.UserID != userID {
		t.Fatalf("the principal must name the token's subject, or one user acts as another; got %v want %s",
			p.UserID, userID)
	}
	if p.OrganizationID != orgID {
		t.Fatalf("the principal's organization must come from the token, since it is what scopes every "+
			"tenant query; got %s want %s", p.OrganizationID, orgID)
	}
	if p.SessionJTI != jti {
		t.Fatalf("the principal must carry the session jti so the session can be revoked; got %q want %q",
			p.SessionJTI, jti)
	}
	if p.IsImpersonating() {
		t.Fatal("an ordinary token produced an impersonating principal")
	}
}

func TestRequireUserAuth_RejectsBadAuthorizationHeaders(t *testing.T) {
	// bearerToken() is deliberately strict: exactly one "Bearer " prefix, case
	// sensitive. These rows pin that strictness in place, because the failure
	// mode of a lax parser is not "the request 401s" — it is that some other
	// scheme's credential gets handed to the JWT parser, or that an empty
	// remainder is treated as a token.
	db := testDB(t)
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	orgID := uuid.New()
	jti := amwSeedSession(t, db, userID, orgID)
	good := amwMint(t, amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti})

	tests := map[string]struct {
		headers []string
		why     string
	}{
		"no Authorization header at all": {
			headers: nil,
			why:     "an anonymous request must not become an authenticated one",
		},
		"empty Authorization header": {
			headers: []string{"Authorization: "},
			why:     "an empty header is an absent header",
		},
		"Basic scheme": {
			headers: []string{"Authorization: Basic dXNlcjpwYXNz"},
			why: "a Basic credential must never be forwarded to the JWT parser; only Bearer is " +
				"an accepted scheme here",
		},
		"bare token with no scheme": {
			headers: []string{"Authorization: " + good},
			why: "the scheme is not optional. Accepting a bare token widens what counts as a " +
				"credential presentation",
		},
		"lowercase bearer scheme": {
			headers: []string{"Authorization: bearer " + good},
			why: "the parser is case sensitive by construction; this row documents that as intended " +
				"behaviour so a future 'tidy-up' cannot silently loosen it without a failing test",
		},
		"Bearer with no token": {
			headers: []string{"Authorization: Bearer "},
			why:     "an empty credential must be refused before parsing, not parsed as an empty token",
		},
		"Bearer with only whitespace": {
			headers: []string{"Authorization: Bearer     "},
			why:     "whitespace trims to empty and must be refused",
		},
		"doubled scheme": {
			headers: []string{"Authorization: Bearer Bearer " + good},
			why:     "the remainder after one prefix is the token; a second prefix makes it malformed",
		},
		"garbage token": {
			headers: []string{"Authorization: Bearer not.a.jwt"},
			why:     "a malformed token must 401, not panic or fall through",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, reached, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))

			args := make([]any, 0, len(tc.headers))
			for _, h := range tc.headers {
				args = append(args, h)
			}
			resp := api.Get("/amw-probe", args...)

			if *reached {
				t.Fatalf("%s\nthe handler RAN for %v", tc.why, tc.headers)
			}
			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("%s\nwant 401, got %d: %s", tc.why, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestRequireUserAuth_RejectsUnusableTokensAndDeadSessions(t *testing.T) {
	db := testDB(t)
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	orgID := uuid.New()
	jti := amwSeedSession(t, db, userID, orgID)

	base := func(mut func(*amwTokenSpec)) string {
		spec := amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti}
		mut(&spec)
		return amwMint(t, spec)
	}
	past := time.Now().UTC().Add(-2 * time.Hour)

	tests := map[string]struct {
		token string
		want  int
		why   string
	}{
		"alg=none forgery": {
			token: base(func(s *amwTokenSpec) { s.method = jwt.SigningMethodNone }),
			want:  http.StatusUnauthorized,
			why: "an unsigned token names a real user with a real live session here — everything except " +
				"the signature is genuine. If alg=none were honoured, anyone could mint this",
		},
		"signed with the wrong secret": {
			token: base(func(s *amwTokenSpec) { s.secret = "not-the-servers-secret" }),
			want:  http.StatusUnauthorized,
			why:   "a forged signature over otherwise valid claims must not authenticate",
		},
		"wrong issuer": {
			token: base(func(s *amwTokenSpec) { s.issuer = "another-apiary" }),
			want:  http.StatusUnauthorized,
			why:   "a token from another deployment must not be usable here",
		},
		"expired": {
			token: base(func(s *amwTokenSpec) {
				s.issuedAt, s.notBefore, s.expiresAt = past, past, past.Add(time.Minute)
			}),
			want: http.StatusUnauthorized,
			why:  "a live session row must not resurrect an expired access token",
		},
		"refresh token on the API surface": {
			token: base(func(s *amwTokenSpec) { s.typ = "refresh" }),
			want:  http.StatusUnauthorized,
			why:   "refresh tokens are not access tokens and must not authorise API calls",
		},
		"unknown session jti": {
			token: base(func(s *amwTokenSpec) { s.jti = uuid.NewString() }),
			want:  http.StatusUnauthorized,
			why: "a perfectly signed token with no session row behind it has been revoked or never " +
				"existed; liveness is what makes logout mean anything",
		},
		"org id is not a uuid": {
			token: base(func(s *amwTokenSpec) { s.orgID = "acme" }),
			want:  http.StatusUnauthorized,
			why: "the org id scopes every tenant query. An unparseable one must refuse rather than " +
				"fall through to uuid.Nil, which would be a real, shared tenant scope",
		},
		"org id is missing": {
			token: base(func(s *amwTokenSpec) { s.orgID = " " }),
			want:  http.StatusUnauthorized,
			why:   "same as above: no org means no scope, and no scope must not mean every scope",
		},
		"mfa enrolled but not satisfied": {
			token: base(func(s *amwTokenSpec) { s.mfaEnabled = true }),
			want:  http.StatusForbidden,
			why: "the half-authenticated token issued between password and second factor must be " +
				"refused with mfa_required, not admitted",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, reached, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
			resp := api.Get("/amw-probe", "Authorization: Bearer "+tc.token)

			if *reached {
				t.Fatalf("%s\nthe handler RAN (status %d)", tc.why, resp.Code)
			}
			if resp.Code != tc.want {
				t.Fatalf("%s\nwant %d, got %d: %s", tc.why, tc.want, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestRequireUserAuth_DeactivatedUserIsLockedOutImmediately(t *testing.T) {
	// Offboarding: a token minted while the account was live must stop working
	// the moment the account is deactivated, without waiting for expiry. This is
	// the only lever an operator has between "revoke this person" and "their
	// 15-minute token runs out".
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)
	token := amwMint(t, amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti})

	// The before/after pair is what makes this test mean something: the SAME
	// token must be accepted first and refused second, so the only variable is
	// is_active. Without the precondition, a 401 afterwards could just as well
	// mean the fixture was broken all along.
	before, reachedBefore, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
	if resp := before.Get("/amw-probe", "Authorization: Bearer "+token); resp.Code != http.StatusOK {
		t.Fatalf("precondition: the token must work while the account is active, got %d: %s",
			resp.Code, resp.Body.String())
	}
	if !*reachedBefore {
		t.Fatal("precondition: the handler did not run for a live account, so the fixture is wrong")
	}

	if err := db.Model(&control.User{}).Where("id = ?", userID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	api2, reached2, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
	resp := api2.Get("/amw-probe", "Authorization: Bearer "+token)

	if *reached2 {
		t.Fatal("a deactivated user's still-unexpired token was accepted; deactivation would not lock anyone out")
	}
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 once the account is deactivated, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRequireUserAuth_HonoursTheGlobalTokensRevokedAtCutoff(t *testing.T) {
	// "Log me out everywhere" — the lever a user pulls after a password change
	// or a stolen laptop. It does not delete session rows; it stamps
	// users.tokens_revoked_at, and every access token ISSUED BEFORE that stamp
	// must stop working at once. Without this, the session row stays live and a
	// stolen 15-minute token keeps working for its full life after the user has
	// been told they are signed out everywhere.
	//
	// The last step is the other half of the property and is what pins the
	// DIRECTION of the comparison: a cutoff that predates the token must NOT
	// lock it out, or every revocation would be permanent and re-login would be
	// impossible.
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)
	token := amwMint(t, amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti})

	setCutoff := func(at time.Time) {
		t.Helper()
		if err := db.Model(&control.User{}).Where("id = ?", userID).
			Update("tokens_revoked_at", at).Error; err != nil {
			t.Fatalf("stamp tokens_revoked_at: %v", err)
		}
	}
	call := func() (int, bool) {
		t.Helper()
		api, reached, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
		resp := api.Get("/amw-probe", "Authorization: Bearer "+token)
		return resp.Code, *reached
	}

	if code, reached := call(); code != http.StatusOK || !reached {
		t.Fatalf("precondition: the token must work before any cutoff is stamped, got %d (handler ran: %v)",
			code, reached)
	}

	setCutoff(time.Now().UTC().Add(time.Hour))
	if code, reached := call(); reached || code != http.StatusUnauthorized {
		t.Fatalf("a token issued BEFORE the global revocation cutoff must be refused, or 'sign out of "+
			"every device' leaves every outstanding access token working; got %d (handler ran: %v)",
			code, reached)
	}

	setCutoff(time.Now().UTC().Add(-time.Hour))
	if code, reached := call(); code != http.StatusOK || !reached {
		t.Fatalf("a cutoff that PREDATES the token must not lock it out — otherwise one revocation is "+
			"permanent and the user could never sign in again; got %d (handler ran: %v)", code, reached)
	}
}

// --- RequirePlatformAdmin ---------------------------------------------------

func TestRequirePlatformAdmin_AdmitsStaffAndRefusesEveryoneElse(t *testing.T) {
	// The whole platform admin gate, end to end: real signing, real DB, real
	// session liveness. The refusal rows carry a genuine live session and a
	// genuine signature, so the ONLY thing being tested is the role decision.
	db := testDB(t)

	tests := map[string]struct {
		role control.UserRole
		want int
		why  string
	}{
		"platform admin": {control.RolePlatformAdmin, http.StatusOK,
			"admin must reach the platform surface, or the gate is just an outage"},
		"platform support": {control.RolePlatformSupport, http.StatusOK,
			"support is admitted by isPlatformAdmin and must reach the surface too"},
		"platform sales": {control.RolePlatformSales, http.StatusForbidden,
			"sales is staff but not admin staff"},
		"customer": {control.RolePlatformCustomer, http.StatusForbidden,
			"the default signup role must never reach the admin API"},
		"membership role owner": {control.UserRole(control.MemberRoleOwner), http.StatusForbidden,
			"owner is a MEMBERSHIP role. Anyone is owner of the org they created, so if it satisfied " +
				"this gate the admin API would be self-service"},
		"membership role member": {control.UserRole(control.MemberRoleMember), http.StatusForbidden,
			"member is a MEMBERSHIP role with no platform meaning"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			orgID := uuid.New()
			userID := amwSeedUser(t, db, tc.role, true)
			jti := amwSeedSession(t, db, userID, orgID)

			api, reached, seen := amwProbeAPI(t, RequirePlatformAdmin, amwDeps(db))
			resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
				subject: userID.String(), orgID: orgID.String(), jti: jti,
			}))

			if resp.Code != tc.want {
				t.Fatalf("%s\nrole %q: want %d, got %d: %s", tc.why, tc.role, tc.want, resp.Code, resp.Body.String())
			}
			if tc.want != http.StatusOK {
				if *reached {
					t.Fatalf("%s\nrole %q RAN the admin handler", tc.why, tc.role)
				}
				return
			}
			p := *seen
			if p == nil || p.UserID == nil || *p.UserID != userID {
				t.Fatalf("the admin principal must name the staff member who authenticated — the audit "+
					"trail for every admin action reads it; got %+v want user %s", p, userID)
			}
			if p.OrganizationID != orgID {
				t.Fatalf("the admin principal's org must come from the token; got %s want %s",
					p.OrganizationID, orgID)
			}
		})
	}
}

func TestRequirePlatformAdmin_RefusesImpersonationEvenWhenTheSubjectIsAnAdmin(t *testing.T) {
	// The escalation this guard exists for. An impersonation session's subject
	// is the TARGET user; if that target happens to be staff, an operator
	// impersonating them would inherit the admin API while the audit trail
	// recorded the target as the actor. It must be a 403 (the token is valid,
	// the action is not), and the handler must not run.
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformAdmin, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, reached, _ := amwProbeAPI(t, RequirePlatformAdmin, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
		impersonate: true, actorID: uuid.NewString(),
	}))

	if *reached {
		t.Fatal("an impersonated session reached the platform admin surface; the actor could act as " +
			"admin under someone else's name")
	}
	if resp.Code != http.StatusForbidden {
		t.Fatalf("want 403 for an impersonated admin session, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRequirePlatformAdmin_RefusesUnusableCredentials(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformAdmin, true)
	jti := amwSeedSession(t, db, userID, orgID)

	base := func(mut func(*amwTokenSpec)) string {
		spec := amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti}
		mut(&spec)
		return amwMint(t, spec)
	}

	tests := map[string]struct {
		header string
		want   int
		why    string
	}{
		"no header": {"", http.StatusUnauthorized,
			"an anonymous request must not reach the admin surface"},
		"non-Bearer scheme": {"Authorization: Basic YWRtaW46YWRtaW4=", http.StatusUnauthorized,
			"only Bearer is an accepted presentation"},
		"alg=none forgery of a real admin": {
			"Authorization: Bearer " + base(func(s *amwTokenSpec) { s.method = jwt.SigningMethodNone }),
			http.StatusUnauthorized,
			"every claim here is genuine except the signature; honouring alg=none would hand the admin " +
				"API to anyone who can read a user id"},
		"wrong secret": {
			"Authorization: Bearer " + base(func(s *amwTokenSpec) { s.secret = "attacker-secret" }),
			http.StatusUnauthorized,
			"a forged signature must not authenticate an admin"},
		"expired admin token": {
			"Authorization: Bearer " + base(func(s *amwTokenSpec) {
				p := time.Now().UTC().Add(-2 * time.Hour)
				s.issuedAt, s.notBefore, s.expiresAt = p, p, p.Add(time.Minute)
			}),
			http.StatusUnauthorized,
			"expiry bounds a stolen admin token; a live session row must not extend it"},
		"revoked session": {
			"Authorization: Bearer " + base(func(s *amwTokenSpec) { s.jti = uuid.NewString() }),
			http.StatusUnauthorized,
			"an admin whose session has no row has logged out or been revoked; liveness is checked " +
				"AFTER the role, so this row also proves the role check is not the last word"},
		"unparseable org id": {
			"Authorization: Bearer " + base(func(s *amwTokenSpec) { s.orgID = "not-a-uuid" }),
			http.StatusUnauthorized,
			"an admin principal with uuid.Nil for an organization would be scoped to a real, shared " +
				"tenant id rather than to none"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, reached, _ := amwProbeAPI(t, RequirePlatformAdmin, amwDeps(db))

			args := []any{}
			if tc.header != "" {
				args = append(args, tc.header)
			}
			resp := api.Get("/amw-probe", args...)

			if *reached {
				t.Fatalf("%s\nthe admin handler RAN (status %d)", tc.why, resp.Code)
			}
			if resp.Code != tc.want {
				t.Fatalf("%s\nwant %d, got %d: %s", tc.why, tc.want, resp.Code, resp.Body.String())
			}
		})
	}
}

// --- AssertPlatformRole -----------------------------------------------------

func amwUser(role control.UserRole, active bool) *control.User {
	return &control.User{ID: uuid.New(), Role: role, IsActive: active}
}

func TestAssertPlatformRole_MatchesOnlyTheNamedPlatformRoles(t *testing.T) {
	// The decision matrix. The membership rows are the point: control.MemberRole
	// values are a different ladder, and "owner" outranks "admin" on it. A check
	// that compared against the wrong ladder would let anyone who created an
	// organization pass a staff-only assertion.
	userID := uuid.New()
	orgID := uuid.New()

	tests := map[string]struct {
		role    control.UserRole
		allowed []control.UserRole
		ok      bool
		why     string
	}{
		"admin where admin is allowed": {
			control.RolePlatformAdmin, []control.UserRole{control.RolePlatformAdmin}, true,
			"the named role must pass, or the assertion is an outage",
		},
		"support where admin or support is allowed": {
			control.RolePlatformSupport, []control.UserRole{control.RolePlatformAdmin, control.RolePlatformSupport}, true,
			"any member of the allowed set passes, not just the first",
		},
		"support where only admin is allowed": {
			control.RolePlatformSupport, []control.UserRole{control.RolePlatformAdmin}, false,
			"the allowed set is an exact membership test, not a seniority floor; support is not admin",
		},
		"admin where only support is allowed": {
			control.RolePlatformAdmin, []control.UserRole{control.RolePlatformSupport}, false,
			"and it does not work upward either — a role ladder that let admin pass a support-only " +
				"gate would make every allowed-set an approximation",
		},
		"customer where admin is allowed": {
			control.RolePlatformCustomer, []control.UserRole{control.RolePlatformAdmin}, false,
			"the default signup role is not staff",
		},
		"membership role owner where admin is allowed": {
			control.UserRole(control.MemberRoleOwner), []control.UserRole{control.RolePlatformAdmin}, false,
			"owner is a MEMBERSHIP role. Anyone is owner of an org they created, and owner outranks " +
				"admin on that ladder — conflating the two is privilege escalation by signup",
		},
		"membership role member where admin is allowed": {
			control.UserRole(control.MemberRoleMember), []control.UserRole{control.RolePlatformAdmin}, false,
			"member is a MEMBERSHIP role with no platform meaning",
		},
		"admin with an EMPTY allowed set": {
			control.RolePlatformAdmin, nil, false,
			"an empty allowed set must deny rather than admit everyone. A caller that forgets to name " +
				"a role would otherwise silently open the endpoint to every authenticated user — the " +
				"single most dangerous default this function could have",
		},
		"customer with an empty allowed set": {
			control.RolePlatformCustomer, nil, false,
			"same, from the other side",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			users := &amwStubUsers{user: amwUser(tc.role, true)}
			ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

			gotP, gotU, err := AssertPlatformRole(ctx, users, tc.allowed...)

			if tc.ok {
				if err != nil {
					t.Fatalf("%s\nrole %q with allowed=%v was refused: %v", tc.why, tc.role, tc.allowed, err)
				}
				if gotP == nil || gotU == nil {
					t.Fatalf("a successful assertion must return the principal AND the user record — "+
						"handlers use the returned user to decide what to show; got p=%v u=%v", gotP, gotU)
				}
				if gotU.Role != tc.role {
					t.Fatalf("the returned user must be the one that was checked; got role %q want %q",
						gotU.Role, tc.role)
				}
				return
			}
			if !errors.Is(err, ErrNotPlatformStaff) {
				t.Fatalf("%s\nrole %q with allowed=%v: want ErrNotPlatformStaff, got %v",
					tc.why, tc.role, tc.allowed, err)
			}
			if gotP != nil || gotU != nil {
				t.Fatalf("%s\na refusal must return nothing usable; got p=%v u=%v", tc.why, gotP, gotU)
			}
			if statusOf(t, PlatformRoleError(err)) != http.StatusForbidden {
				t.Fatalf("%s\nthe refusal must map to 403", tc.why)
			}
		})
	}
}

func TestAssertPlatformRole_ImpersonatedSessionIsRefusedUnconditionallyAndWithoutALookup(t *testing.T) {
	// Unconditional means: even when the impersonated SUBJECT is a platform
	// admin, and even when admin is in the allowed set. An impersonation session
	// names the target as its subject, so honouring the target's role would let
	// a support engineer impersonating a staff customer start a second
	// impersonation from inside the first — with the audit trail naming the
	// wrong actor.
	//
	// The lookup assertion pins the ORDER: refusing before touching the
	// repository means the decision cannot depend on what the database says,
	// so a DB that returned a stale or attacker-influenced role could not
	// change the answer.
	users := &amwStubUsers{user: amwUser(control.RolePlatformAdmin, true)}
	ctx := WithPrincipal(context.Background(), impersonatingPrincipal(uuid.New(), uuid.New()))

	p, u, err := AssertPlatformRole(ctx, users, control.RolePlatformAdmin, control.RolePlatformSupport)

	if !errors.Is(err, ErrNestedImpersonate) {
		t.Fatalf("an impersonated session whose subject IS an admin must be refused with "+
			"ErrNestedImpersonate, got %v", err)
	}
	if errors.Is(err, ErrNotPlatformStaff) {
		t.Fatal("nested impersonation must stay distinguishable from 'not staff': they are different " +
			"refusals with different messages, and collapsing them hides the escalation attempt from logs")
	}
	if p != nil || u != nil {
		t.Fatalf("a refused impersonation must return nothing usable; got p=%v u=%v", p, u)
	}
	if users.calls != 0 {
		t.Fatalf("the impersonation refusal must happen BEFORE the user lookup, so the database cannot "+
			"influence it; the repository was consulted %d time(s)", users.calls)
	}
	if got := statusOf(t, PlatformRoleError(err)); got != http.StatusForbidden {
		t.Fatalf("nested impersonation must map to 403, got %d", got)
	}
}

func TestAssertPlatformRole_DeniesWheneverItCannotReachADecision(t *testing.T) {
	// Every arm where the answer is unknown must be a denial. The nil-repo row
	// is the load-bearing one: a wiring mistake that left users nil would, under
	// a fail-open, turn every staff-only endpoint into a public one — and it
	// would pass every happy-path test, because the happy path never notices a
	// check that always says yes.
	userID := uuid.New()
	orgID := uuid.New()
	repoErr := errors.New("connection refused")

	tests := map[string]struct {
		ctx        context.Context
		users      repo.UserRepo
		wantStatus int
		sentinel   error
		why        string
	}{
		"no principal in context": {
			ctx:        context.Background(),
			users:      &amwStubUsers{user: amwUser(control.RolePlatformAdmin, true)},
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "an unauthenticated request has no principal. If that were treated as anything but a " +
				"denial, forgetting the auth middleware on a route would open it entirely",
		},
		"principal with no user id": {
			ctx: WithPrincipal(context.Background(), &Principal{
				Method: AuthMethodOrgSecret, OrganizationID: orgID,
			}),
			users:      &amwStubUsers{user: amwUser(control.RolePlatformAdmin, true)},
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "an org-secret principal is a machine, not a person, and has no platform role. It must " +
				"not be resolved to some default user",
		},
		"typed nil principal": {
			ctx:        WithPrincipal(context.Background(), (*Principal)(nil)),
			users:      &amwStubUsers{user: amwUser(control.RolePlatformAdmin, true)},
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "a typed nil satisfies the context type assertion, so it must be caught explicitly or " +
				"the assertion panics — and a panic in an auth check is a 500 on a route that should 403",
		},
		"nil user repository": {
			ctx:        WithPrincipal(context.Background(), bearerPrincipal(userID, orgID)),
			users:      nil,
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "a check that cannot reach a decision must deny. A nil repo means the wiring is wrong, " +
				"and failing open on bad wiring makes every staff endpoint public",
		},
		"user record not found": {
			ctx:        WithPrincipal(context.Background(), bearerPrincipal(userID, orgID)),
			users:      &amwStubUsers{user: nil},
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "the repo returns (nil, nil) for a missing row. A deleted user's token must not " +
				"authorise, and nil must not be dereferenced",
		},
		"deactivated admin": {
			ctx:        WithPrincipal(context.Background(), bearerPrincipal(userID, orgID)),
			users:      &amwStubUsers{user: amwUser(control.RolePlatformAdmin, false)},
			wantStatus: http.StatusForbidden,
			sentinel:   ErrNotPlatformStaff,
			why: "deactivation is the offboarding lever; a deactivated admin must lose staff powers " +
				"immediately, not when their token expires",
		},
		"repository error": {
			ctx:        WithPrincipal(context.Background(), bearerPrincipal(userID, orgID)),
			users:      &amwStubUsers{err: repoErr},
			wantStatus: http.StatusInternalServerError,
			sentinel:   repoErr,
			why: "a database failure is not a policy decision. It must deny, but as a 500 — reporting it " +
				"as 403 would make an outage look like a permissions problem and send the operator " +
				"hunting for a role that was never wrong",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p, u, err := AssertPlatformRole(tc.ctx, tc.users, control.RolePlatformAdmin, control.RolePlatformSupport)

			if err == nil {
				t.Fatalf("%s\nthe assertion PASSED", tc.why)
			}
			if !errors.Is(err, tc.sentinel) {
				t.Fatalf("%s\nwant an error wrapping %v, got %v", tc.why, tc.sentinel, err)
			}
			if p != nil || u != nil {
				t.Fatalf("%s\na denial must return nothing usable; got p=%v u=%v", tc.why, p, u)
			}
			if got := statusOf(t, PlatformRoleError(err)); got != tc.wantStatus {
				t.Fatalf("%s\nwant status %d, got %d", tc.why, tc.wantStatus, got)
			}
		})
	}
}

func TestPlatformRoleError_PassesNilThroughSoASuccessIsNotTurnedIntoAFailure(t *testing.T) {
	if err := PlatformRoleError(nil); err != nil {
		t.Fatalf("PlatformRoleError(nil) must be nil — handlers call it unconditionally on the "+
			"assertion's error, so a non-nil return here would refuse every authorised staff request; got %v", err)
	}
}
