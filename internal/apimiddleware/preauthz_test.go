package apimiddleware

import (
	"net/http"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
)

// The lockout.
//
// When an organization turns on require_mfa, login issues a session carrying
// MFAEnabled=true, MFA=false to every member — including the ones who never
// enrolled, because there is no factor to challenge them with. RequireUserAuth
// then answers 403 mfa_required to every request that session makes, and TOTP
// setup is behind RequireUserAuth. The user can authenticate, holds a valid
// token, and that token can do exactly one useful thing: nothing. Including the
// one action that would fix it.
//
// This is the conflation the pre-authz split exists to undo. Proving who you
// are (authn) and being allowed to act (authz) are different questions, and
// policy sits between them — so remediation has to be reachable from the first
// without requiring the second.
func TestRequireUserAuthLocksOutAUserWhoHasNotEnrolledYet(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, reached, _ := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
		mfaEnabled: true, mfa: false,
	}))

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected the policy gate to refuse, got %d", resp.Code)
	}
	if *reached {
		t.Fatal("the policy gate did not run")
	}
}

// The way out. RequireAuthenticated answers the authn question only: this token
// is valid, this session is live, this is who it names. It does NOT apply the
// policy gate, because policy is an authz concern and the endpoints behind this
// middleware exist precisely to let someone satisfy it.
func TestRequireAuthenticatedAdmitsAnMFAPendingSession(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, reached, seen := amwProbeAPI(t, RequireAuthenticated, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
		mfaEnabled: true, mfa: false,
	}))

	if resp.Code != http.StatusOK {
		t.Fatalf("a user who must enrol cannot reach enrolment, got %d: %s", resp.Code, resp.Body.String())
	}
	if !*reached {
		t.Fatal("the handler never ran")
	}
	p := *seen
	if p == nil || p.UserID == nil || *p.UserID != userID {
		t.Fatalf("pre-authz must still establish WHO: got %+v", p)
	}
	// The principal has to say it is not fully authorized, or a handler mounted
	// here by mistake would be indistinguishable from one behind the full gate.
	if p.PolicySatisfied {
		t.Fatal("an MFA-pending session was marked policy-satisfied; anything reading that flag " +
			"would treat a half-authenticated caller as a fully authorized one")
	}
}

// Pre-authz is NOT a weaker way in. It relaxes exactly one thing — the policy
// gate — and everything establishing identity still applies.
func TestRequireAuthenticatedStillEnforcesEverythingElse(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	cases := []struct {
		name string
		spec amwTokenSpec
		why  string
	}{
		{
			name: "a refresh token is not an access token",
			spec: amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti, typ: "refresh"},
			why:  "a refresh token reaching an enrolment endpoint would let a stolen refresh token add an authenticator",
		},
		{
			name: "a token signed with the wrong secret",
			spec: amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: jti, secret: "not-the-secret"},
			why:  "forgery must not become possible merely because the policy gate is relaxed",
		},
		{
			name: "a session that is not live",
			spec: amwTokenSpec{subject: userID.String(), orgID: orgID.String(), jti: uuid.NewString()},
			why:  "a logged-out session must not be able to enrol a factor",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			api, reached, _ := amwProbeAPI(t, RequireAuthenticated, amwDeps(db))
			resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, c.spec))
			if resp.Code == http.StatusOK || *reached {
				t.Fatalf("%s (got %d)", c.why, resp.Code)
			}
		})
	}
}

// A session that HAS satisfied policy is marked as such, so the flag means
// something rather than being permanently false.
func TestRequireAuthenticatedMarksASatisfiedSession(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, _, seen := amwProbeAPI(t, RequireAuthenticated, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
		mfaEnabled: true, mfa: true,
	}))
	if resp.Code != http.StatusOK {
		t.Fatalf("a fully satisfied session was refused: %d", resp.Code)
	}
	if p := *seen; p == nil || !p.PolicySatisfied {
		t.Fatalf("a session that completed its second factor is not marked satisfied: %+v", p)
	}
}

// RequireUserAuth must keep marking the principal satisfied — everything behind
// it has passed the gate by definition.
func TestRequireUserAuthMarksThePrincipalSatisfied(t *testing.T) {
	db := testDB(t)
	orgID := uuid.New()
	userID := amwSeedUser(t, db, control.RolePlatformCustomer, true)
	jti := amwSeedSession(t, db, userID, orgID)

	api, _, seen := amwProbeAPI(t, RequireUserAuth, amwDeps(db))
	resp := api.Get("/amw-probe", "Authorization: Bearer "+amwMint(t, amwTokenSpec{
		subject: userID.String(), orgID: orgID.String(), jti: jti,
	}))
	if resp.Code != http.StatusOK {
		t.Fatalf("valid session refused: %d", resp.Code)
	}
	if p := *seen; p == nil || !p.PolicySatisfied {
		t.Fatalf("a principal past the policy gate is not marked satisfied: %+v", p)
	}
}
