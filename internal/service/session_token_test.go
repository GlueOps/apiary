package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// This suite covers the session/token lifecycle: what it takes to turn a
// refresh token into a new session, what it takes to be refused, and what
// revocation actually has to accomplish. Everything here is driven through the
// real gorm repositories against a schema private to this test binary — the
// session layer's whole job is agreeing with the database about which sessions
// are still alive, so a fake session store would test the agreement away.
var sestSchema = testsupport.SchemaName("apiary_service_sessiontoken_test")

const (
	sestSecret = "sest-jwt-secret-not-the-real-one"
	sestIssuer = "apiary-sest"
)

// sestHarness wires the production services exactly as
// handlers/auth/refresh.go wires them, so a change to that constructor chain
// shows up here rather than only in production.
type sestHarness struct {
	db          *gorm.DB
	users       repo.UserRepoGorm
	memberships repo.MembershipRepo
	orgs        repo.OrganizationRepo
	sessionRepo repo.TokenSessionRepo
	tokens      *TokenService
	tracker     *TokenSessionService
	svc         *SessionService
}

// sestNewHarness builds the stack. withPolicies selects whether a real
// PolicyService is wired: SessionService branches on `s.policies != nil`, and
// both sides of that branch ship (refresh.go passes deps.Policies, which may be
// nil in a partially configured deployment), so both are exercised.
func sestNewHarness(t *testing.T, withPolicies bool) *sestHarness {
	t.Helper()

	db := testsupport.OpenSchema(t, sestSchema,
		&control.User{},
		&control.Organization{},
		&control.Membership{},
		&control.TokenSession{},
		&control.AuthAuditEvent{},
		&control.PlatformPolicy{},
		&control.OrgPolicy{},
	)

	tokens, err := NewTokenService(sestSecret, sestIssuer, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	users := repo.NewUserRepoGorm(db)
	memberships := repo.NewMembershipRepoGorm(db)
	orgs := repo.NewOrganizationRepoGorm(db)
	sessionRepo := repo.NewTokenSessionRepoGorm(db)
	audit := repo.NewAuthAuditRepoGorm(db)

	tracker := NewTokenSessionService(tokens, sessionRepo, audit, &users)

	var policies *PolicyService
	if withPolicies {
		policies = NewPolicyService(
			repo.NewPlatformPolicyRepoGorm(db),
			repo.NewOrgPolicyRepoGorm(db),
			&users,
			nil,
		)
	}

	h := &sestHarness{
		db:          db,
		users:       users,
		memberships: memberships,
		orgs:        orgs,
		sessionRepo: sessionRepo,
		tokens:      tokens,
		tracker:     tracker,
	}
	h.svc = NewSessionService(users, memberships, orgs, tokens, tracker, policies)
	return h
}

// seedUser creates an active user with a unique email. Rows are never cleaned
// up between tests in this file and never counted globally: every assertion is
// scoped to a subject or an organization created by that test, so the suite
// stays correct however many other tests share the schema.
func (h *sestHarness) seedUser(t *testing.T) *control.User {
	t.Helper()
	u := &control.User{
		Email:     "sest-" + uuid.NewString() + "@example.test",
		Password:  "x",
		FirstName: "Sess",
		LastName:  "Tester",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := h.db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func (h *sestHarness) seedOrg(t *testing.T, name string) *control.Organization {
	t.Helper()
	o := &control.Organization{Name: name, Status: "active"}
	if err := h.db.Create(o).Error; err != nil {
		t.Fatalf("seed org: %v", err)
	}
	return o
}

func (h *sestHarness) seedMembership(t *testing.T, u *control.User, o *control.Organization, role control.MemberRole) {
	t.Helper()
	if err := h.db.Create(&control.Membership{UserID: u.ID, OrganizationID: o.ID, Role: role}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}
}

// seedActiveSession mints a real tracked session, the way login does. The
// returned tokens are the only copies that exist, exactly as in production.
func (h *sestHarness) seedActiveSession(t *testing.T, u *control.User, o *control.Organization, opts IssueOpts) *IssuedTokens {
	t.Helper()
	issued, err := h.tracker.IssueTracked(context.Background(), IssueTrackedInput{
		SubjectID:   u.ID.String(),
		SubjectKind: "user",
		OrgID:       o.ID.String(),
		Opts:        opts,
		IPAddress:   "203.0.113.10",
		UserAgent:   "sest/1.0",
		EventType:   "login",
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return issued
}

// sestUserAndOrg is the ordinary starting state: an active user who belongs to
// an organization and holds a live session in it.
func sestUserAndOrg(t *testing.T, h *sestHarness) (*control.User, *control.Organization, *IssuedTokens) {
	t.Helper()
	u := h.seedUser(t)
	o := h.seedOrg(t, "Sest Org")
	h.seedMembership(t, u, o, control.MemberRoleOwner)
	return u, o, h.seedActiveSession(t, u, o, IssueOpts{AMR: []string{"pwd"}})
}

// countAudit counts audit events of one type for one subject. Used to prove
// that a rejected call did not merely fail quietly but also recorded (or
// deliberately did not record) the event an operator would look for.
func (h *sestHarness) countAudit(t *testing.T, subjectID uuid.UUID, eventType string) int64 {
	t.Helper()
	var n int64
	if err := h.db.Model(&control.AuthAuditEvent{}).
		Where("subject_id = ? AND event_type = ?", subjectID, eventType).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	return n
}

func (h *sestHarness) countSessions(t *testing.T, subjectID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := h.db.Model(&control.TokenSession{}).
		Where("subject_id = ?", subjectID).
		Count(&n).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return n
}

// enableOrgMFAPolicy turns on require_mfa for one organization only.
//
// Deliberately an ORG policy rather than a platform one: PlatformPolicy has a
// unique index on kind, so a platform row is global to the schema and would
// leak into every other test sharing it.
func (h *sestHarness) enableOrgMFAPolicy(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	p := &control.OrgPolicy{
		OrganizationID: orgID,
		Kind:           control.PolicyKindRequireMFA,
		Enabled:        true,
		CreatedBy:      uuid.New(),
	}
	if err := h.db.Create(p).Error; err != nil {
		t.Fatalf("enable org mfa policy: %v", err)
	}
}

// ------------------------------------------------------- TokenService.Parse

// TestSestTokenParse_RejectsEveryUnacceptableToken is the signature/type gate
// in front of every authenticated path in this service. Each row is a token an
// attacker can actually produce: one signed with a key they chose, one whose
// algorithm they chose, one they were legitimately given for a different
// purpose, one that has expired, one minted by a different issuer sharing the
// secret, and assorted junk. Every one of them must come back as
// ErrInvalidToken and nothing else — a caller that gets claims back treats the
// subject as authenticated.
func TestSestTokenParse_RejectsEveryUnacceptableToken(t *testing.T) {
	svc, err := NewTokenService(sestSecret, sestIssuer, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	now := time.Now().UTC()

	good, err := svc.Issue(uuid.NewString(), uuid.NewString(), now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// A token from a service that shares nothing but the shape.
	other, err := NewTokenService("a-completely-different-secret", sestIssuer, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService(other secret): %v", err)
	}
	foreignSig, err := other.Issue(uuid.NewString(), uuid.NewString(), now)
	if err != nil {
		t.Fatalf("Issue(other secret): %v", err)
	}

	// Same secret, different issuer: the case that matters when one JWT secret
	// is reused across two deployments. Without issuer pinning, staging tokens
	// would authenticate against production.
	foreignIssuer, err := NewTokenService(sestSecret, "some-other-issuer", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService(other issuer): %v", err)
	}
	foreignIss, err := foreignIssuer.Issue(uuid.NewString(), uuid.NewString(), now)
	if err != nil {
		t.Fatalf("Issue(other issuer): %v", err)
	}

	expired, err := svc.Issue(uuid.NewString(), uuid.NewString(), now.Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("Issue(expired): %v", err)
	}

	// alg:none — the classic JWT forgery. The claims are whatever the attacker
	// wants; only the signing method check stands between them and an admin
	// session.
	noneClaims := AuthClaims{
		OrgID: uuid.NewString(),
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   uuid.NewString(),
			Issuer:    sestIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	algNone, err := jwt.NewWithClaims(jwt.SigningMethodNone, noneClaims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign alg=none: %v", err)
	}

	// HS512 with the same secret: a valid signature under a method the service
	// never issues. jwt.WithValidMethods plus the keyfunc check must refuse it,
	// otherwise the accepted algorithm set is whatever the attacker sends.
	algHS512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, noneClaims).SignedString([]byte(sestSecret))
	if err != nil {
		t.Fatalf("sign HS512: %v", err)
	}

	// A genuine token with one character of the signature flipped.
	tamperedSig := sestTamperSignature(t, good.AccessToken)

	// A genuine token whose PAYLOAD was rewritten to name a different subject,
	// keeping the original signature. This is the attack the signature exists
	// to stop: the claims are the attacker's, everything else is authentic.
	tamperedPayload := sestRewriteSubject(t, good.AccessToken, uuid.NewString())

	// mfa_pending is a real token type this service mints for a half-finished
	// login. It must never satisfy an access-token check: that would skip the
	// second factor entirely.
	mfaPending, err := svc.IssueMFAPending(uuid.NewString(), uuid.NewString(), now)
	if err != nil {
		t.Fatalf("IssueMFAPending: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		expectType string
		why        string
	}{
		{"access presented as refresh", good.AccessToken, "refresh",
			"an access token is handed to the browser and sent on every request; if it also refreshes, a leaked access token becomes an unlimited session"},
		{"refresh presented as access", good.RefreshToken, "access",
			"a refresh token is long-lived and is not subject to the access-token liveness checks; accepting it as an access token defeats them"},
		{"mfa_pending presented as access", mfaPending, "access",
			"a half-authenticated login must not produce API access"},
		{"mfa_pending presented as refresh", mfaPending, "refresh",
			"a half-authenticated login must not be exchangeable for a full session"},
		{"signature from another secret", foreignSig.AccessToken, "access",
			"anyone who can sign with a key of their own could otherwise mint any subject and org they like"},
		{"issuer mismatch", foreignIss.AccessToken, "access",
			"one JWT secret shared across environments must not let a staging token authenticate here"},
		{"expired", expired.AccessToken, "access",
			"expiry is the only bound on a stolen token's usefulness"},
		{"alg none", algNone, "access",
			"alg=none forgery: the signature check is skipped and the claims are attacker-chosen"},
		{"alg HS512", algHS512, "access",
			"the accepted algorithm must be the server's choice, not the token's"},
		{"tampered signature", tamperedSig, "access",
			"a token whose signature does not verify is a forgery attempt"},
		{"rewritten subject", tamperedPayload, "access",
			"an authentic signature over different claims: accepting it means anyone holding one valid token can become any user"},
		{"empty", "", "access", "an absent credential is not a credential"},
		{"whitespace", "   ", "access", "an absent credential is not a credential"},
		{"not a jwt", "definitely-not-a-token", "access", "malformed input must be refused, not panic or partially parse"},
		{"three junk segments", "aaa.bbb.ccc", "access", "malformed input must be refused, not panic or partially parse"},
		{"header only", strings.SplitN(good.AccessToken, ".", 2)[0], "access",
			"a truncated token must be refused"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := svc.Parse(tc.token, tc.expectType)
			if err == nil {
				t.Fatalf("Parse ACCEPTED a %s as a valid %q token (subject %q, org %q) — %s",
					tc.name, tc.expectType, claims.Subject, claims.OrgID, tc.why)
			}
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Parse rejected the %s with %v, want ErrInvalidToken; callers branch on the sentinel and would treat anything else as a server fault (500) rather than a refusal (401)", tc.name, err)
			}
			if claims != nil {
				t.Fatalf("Parse returned claims alongside an error for the %s; a caller that checks claims before err would authenticate a forgery", tc.name)
			}
		})
	}
}

// TestSestTokenParse_AcceptsItsOwnTokens is the other half of the gate: the
// rejections above are only meaningful if the accepted case really is accepted,
// with the claims the issuer put in. A Parse that refused everything would pass
// every test above.
func TestSestTokenParse_AcceptsItsOwnTokens(t *testing.T) {
	svc, err := NewTokenService(sestSecret, sestIssuer, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	userID := uuid.NewString()
	orgID := uuid.NewString()
	issued, err := svc.Issue(userID, orgID, time.Now().UTC(), IssueOpts{
		AMR:        []string{"pwd", "totp"},
		MFA:        true,
		MFAEnabled: true,
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	access, err := svc.Parse(issued.AccessToken, "access")
	if err != nil {
		t.Fatalf("Parse rejected a freshly issued access token: %v", err)
	}
	if access.Subject != userID || access.OrgID != orgID {
		t.Fatalf("access claims carry subject %q org %q, want %q / %q; the caller authorizes against these fields",
			access.Subject, access.OrgID, userID, orgID)
	}
	if access.ID != issued.AccessJTI {
		t.Fatalf("access jti in the token (%q) differs from the one recorded in the session row (%q); session revocation is looked up by that jti and would never match",
			access.ID, issued.AccessJTI)
	}
	if !access.MFA || !access.MFAEnabled {
		t.Fatalf("access claims lost the MFA state (mfa=%v mfa_enabled=%v); the middleware denies on mfa_enabled && !mfa, so dropping either silently changes who gets in",
			access.MFA, access.MFAEnabled)
	}
	if strings.Join(access.AMR, ",") != "pwd,totp" {
		t.Fatalf("access claims carry amr %v, want [pwd totp]; refresh copies amr forward and an audit of how a session was authenticated reads it", access.AMR)
	}

	refresh, err := svc.Parse(issued.RefreshToken, "refresh")
	if err != nil {
		t.Fatalf("Parse rejected a freshly issued refresh token: %v", err)
	}
	if refresh.ID != issued.RefreshJTI {
		t.Fatalf("refresh jti in the token (%q) differs from the session row's (%q); rotation looks the session up by it", refresh.ID, issued.RefreshJTI)
	}
}

// sestTamperSignature changes one character of a token's signature segment.
//
// The FIRST character, deliberately: an HS256 signature is 32 bytes encoded in
// 43 base64url characters, and the final character carries two bits that decode
// to nothing — four different last characters produce the same 32 bytes, so
// "flip the last character" is not reliably a tamper at all. That was this
// test's first version, and it "passed" by decoding to the identical signature.
func sestTamperSignature(t *testing.T, tok string) string {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || parts[2] == "" {
		t.Fatalf("sestTamperSignature: %q is not a signed jwt", tok)
	}
	first := parts[2][0]
	repl := byte('A')
	if first == 'A' {
		repl = 'B'
	}
	parts[2] = string(repl) + parts[2][1:]
	return strings.Join(parts, ".")
}

// sestRewriteSubject re-encodes the payload with a different subject and leaves
// the original signature in place.
func sestRewriteSubject(t *testing.T, tok, newSubject string) string {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("sestRewriteSubject: %q is not a jwt", tok)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("sestRewriteSubject: decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("sestRewriteSubject: unmarshal payload: %v", err)
	}
	claims["sub"] = newSubject
	rewritten, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("sestRewriteSubject: marshal payload: %v", err)
	}
	parts[1] = base64.RawURLEncoding.EncodeToString(rewritten)
	return strings.Join(parts, ".")
}

// --------------------------------------------------- SessionService.Refresh

// TestSestRefresh_RotatesAndKillsTheReplayedToken is the core refresh-token
// property. A refresh token is single-use: exchanging it must revoke it, and
// presenting it a second time is by definition either an attacker replaying a
// stolen token or a client replaying its own — the server cannot tell, so it
// must assume theft and kill the whole family, including the tokens the
// legitimate client is currently holding. Without that, a stolen refresh token
// grants an indefinite parallel session that the user can neither see nor end.
func TestSestRefresh_RotatesAndKillsTheReplayedToken(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	user, org, first := sestUserAndOrg(t, h)

	second, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: first.RefreshToken, IPAddress: "203.0.113.11", UserAgent: "sest/1.0"})
	if err != nil {
		t.Fatalf("first refresh of a live session failed: %v", err)
	}
	if second.Tokens.RefreshToken == first.RefreshToken {
		t.Fatalf("refresh returned the SAME refresh token; rotation is what makes a stolen token detectable, and reusing it means theft is never noticed")
	}
	if second.OrganizationID != org.ID.String() || second.Organization != org.Name {
		t.Fatalf("refresh reported org %q/%q, want %q/%q; the client renders and scopes on this", second.OrganizationID, second.Organization, org.ID, org.Name)
	}

	// The replay.
	_, err = h.svc.Refresh(ctx, RefreshInput{RefreshToken: first.RefreshToken, IPAddress: "198.51.100.9", UserAgent: "thief/1.0"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("replaying an already-spent refresh token returned %v, want ErrUnauthorized; a refresh token that works twice is a session an attacker can keep alive forever", err)
	}

	if n := h.countAudit(t, user.ID, "theft_detected"); n != 1 {
		t.Fatalf("replay recorded %d theft_detected events, want 1; this is the only signal an operator gets that a refresh token leaked", n)
	}

	// And the whole family dies with it — including the token the honest client
	// is holding. This is the point of family revocation: the user is forced to
	// log in again, which is exactly what should happen when their token leaked.
	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: second.Tokens.RefreshToken, IPAddress: "203.0.113.11", UserAgent: "sest/1.0"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("after a replay was detected the current refresh token still worked (%v); the attacker's parallel session survives the detection", err)
	}

	active, err := h.tracker.ListActive(ctx, user.ID, "user")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("after theft detection %d sessions are still listed active; the family was not revoked", len(active))
	}
}

// TestSestRefresh_RejectsAccessTokenAsRefresh pins the token-type separation at
// the session layer, not just in Parse. An access token is sent on every single
// request and lands in proxy logs and browser history; a refresh token is
// stored once and sent to one endpoint. If the refresh endpoint accepts an
// access token, that distinction — and the entire reason access tokens are
// short-lived — evaporates.
func TestSestRefresh_RejectsAccessTokenAsRefresh(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	_, _, issued := sestUserAndOrg(t, h)

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.AccessToken}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("the refresh endpoint accepted an ACCESS token (%v); a token that leaks through logs or Referer headers would then mint unlimited sessions", err)
	}

	// And the rejection is inert: presenting the wrong token type must not
	// disturb the live session, or an attacker could log a user out at will by
	// posting garbage.
	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); err != nil {
		t.Fatalf("the real refresh token stopped working after a wrong-type token was rejected: %v", err)
	}
}

// TestSestRefresh_RefusesRevokedAndUnknownSessions covers the ways a
// syntactically perfect refresh token can still be dead.
func TestSestRefresh_RefusesRevokedAndUnknownSessions(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	t.Run("session revoked by logout", func(t *testing.T) {
		user, _, issued := sestUserAndOrg(t, h)
		if err := h.tracker.Revoke(ctx, issued.AccessJTI, user.ID.String(), "203.0.113.1", "sest/1.0"); err != nil {
			t.Fatalf("Revoke: %v", err)
		}
		if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("a logged-out session's refresh token still minted a new session (%v); logout would not end anything", err)
		}
	})

	t.Run("token valid but no session row", func(t *testing.T) {
		user := h.seedUser(t)
		org := h.seedOrg(t, "Sest Org No Session")
		h.seedMembership(t, user, org, control.MemberRoleMember)
		// Signed by this very service, but never tracked — the shape of a token
		// minted by a leaked secret, or one whose session row was purged.
		orphan, err := h.tokens.Issue(user.ID.String(), org.ID.String(), time.Now().UTC())
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: orphan.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("a refresh token with no session row was accepted (%v); the session table would then not be the authority on which sessions exist", err)
		}
	})

	t.Run("empty and whitespace", func(t *testing.T) {
		for _, raw := range []string{"", "   ", "\t\n"} {
			if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: raw}); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("Refresh(%q) returned %v, want ErrUnauthorized", raw, err)
			}
		}
	})
}

// TestSestRefresh_RefusesImpersonatedRefreshToken protects the time bound on
// support impersonation. An admin impersonating a customer holds a token marked
// imp=true with a short life; if that token could be refreshed, impersonation
// would renew itself indefinitely and outlive the support ticket that justified
// it. Re-authenticating as the admin is the only way to extend it.
func TestSestRefresh_RefusesImpersonatedRefreshToken(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	user := h.seedUser(t)
	org := h.seedOrg(t, "Sest Impersonated Org")
	h.seedMembership(t, user, org, control.MemberRoleMember)
	admin := h.seedUser(t)
	adminID := admin.ID

	issued := h.seedActiveSession(t, user, org, IssueOpts{
		AMR:         []string{"impersonation"},
		ActorID:     adminID.String(),
		Impersonate: true,
	})

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("an impersonation refresh token minted a new session (%v); support impersonation would then renew itself forever instead of expiring", err)
	}
}

// TestSestRefresh_RequiresLiveMembership is an authorization boundary, not a
// housekeeping check. The organization in a refresh token is a claim the holder
// presents; the only trustworthy source is the membership table. If refresh
// trusted the claim, someone removed from an organization would keep minting
// working tokens for it until their refresh token expired — up to 30 days by
// default.
func TestSestRefresh_RequiresLiveMembership(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	user, org, issued := sestUserAndOrg(t, h)

	// Offboarding: the membership row goes, the token in the user's hand stays.
	if err := h.db.Where("user_id = ? AND organization_id = ?", user.ID, org.ID).
		Delete(&control.Membership{}).Error; err != nil {
		t.Fatalf("delete membership: %v", err)
	}

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("a user removed from the organization refreshed a session for it anyway (%v); the org in the token was trusted over the membership table", err)
	}
}

// TestSestRefresh_RefusesDeactivatedUser: deactivation must take effect on the
// next refresh at the latest, whatever tokens the user still holds.
func TestSestRefresh_RefusesDeactivatedUser(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	user, _, issued := sestUserAndOrg(t, h)

	if err := h.db.Model(&control.User{}).Where("id = ?", user.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("a deactivated user refreshed their session (%v); disabling an account would not end its access", err)
	}
}

// TestSestRefresh_HonoursTokensRevokedAtCutoff pins the global "log me out
// everywhere" cutoff, in both directions. The one-directional version of this
// test would pass against an implementation that simply refused every refresh
// once tokens_revoked_at was set — which would log the user out again
// immediately after they signed back in.
func TestSestRefresh_HonoursTokensRevokedAtCutoff(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	t.Run("token issued before the cutoff is dead", func(t *testing.T) {
		user, _, issued := sestUserAndOrg(t, h)
		if err := h.users.UpdateTokensRevokedAt(ctx, user.ID, time.Now().UTC().Add(time.Hour)); err != nil {
			t.Fatalf("UpdateTokensRevokedAt: %v", err)
		}
		if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("a token issued before the global revocation cutoff still refreshed (%v); a password change or a compromised-account response would leave the attacker's session alive", err)
		}
	})

	t.Run("token issued after the cutoff still works", func(t *testing.T) {
		user, _, _ := sestUserAndOrg(t, h)
		if err := h.users.UpdateTokensRevokedAt(ctx, user.ID, time.Now().UTC().Add(-time.Hour)); err != nil {
			t.Fatalf("UpdateTokensRevokedAt: %v", err)
		}
		// A session minted after the cutoff — i.e. the user signed back in.
		org := h.seedOrg(t, "Sest Post Cutoff Org")
		h.seedMembership(t, user, org, control.MemberRoleOwner)
		fresh := h.seedActiveSession(t, user, org, IssueOpts{AMR: []string{"pwd"}})

		if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: fresh.RefreshToken}); err != nil {
			t.Fatalf("a session created AFTER the revocation cutoff was refused (%v); the cutoff is a point in time, not a permanent ban, and this would lock the user out of their own account", err)
		}
	})
}

// TestSestRefresh_AppliesOrgMFAPolicyOnRefresh: an organization that turns on
// require_mfa must not have to wait for existing sessions to expire. Refresh
// recomputes the effective policy and stamps mfa_enabled into the new token;
// the middleware then denies any request whose token says mfa_enabled without
// mfa. Carrying the old mfa_enabled=false forward instead would let every
// already-signed-in user bypass the new policy for the life of their session.
func TestSestRefresh_AppliesOrgMFAPolicyOnRefresh(t *testing.T) {
	h := sestNewHarness(t, true)
	ctx := context.Background()
	_, org, issued := sestUserAndOrg(t, h)

	h.enableOrgMFAPolicy(t, org.ID)

	res, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken})
	if err != nil {
		t.Fatalf("refresh under a require_mfa policy failed: %v", err)
	}

	claims, err := h.tokens.Parse(res.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse refreshed access token: %v", err)
	}
	if !claims.MFAEnabled {
		t.Fatalf("the refreshed access token says mfa_enabled=false even though the organization requires MFA; every session that predates the policy would keep bypassing it")
	}
	if claims.MFA {
		t.Fatalf("the refreshed token claims mfa=true, but the session it came from was authenticated with a password alone; refresh must never upgrade the factors actually used")
	}
	if strings.Join(claims.AMR, ",") != "pwd" {
		t.Fatalf("refresh dropped or altered amr (%v, want [pwd]); the record of how the session was authenticated must survive rotation", claims.AMR)
	}

	// The converse, so the assertion above cannot be satisfied by an
	// implementation that simply stamps mfa_enabled on every refresh: a session
	// in an organization with NO require_mfa policy must come back with the flag
	// clear. Without this half, refresh could set it unconditionally and lock
	// every password-only user out of a tenant that never asked for a second
	// factor — the middleware denies on mfa_enabled && !mfa — and nothing here
	// would notice.
	relaxedUser := h.seedUser(t)
	relaxedOrg := h.seedOrg(t, "Sest Relaxed Refresh Org")
	h.seedMembership(t, relaxedUser, relaxedOrg, control.MemberRoleOwner)
	relaxedSession := h.seedActiveSession(t, relaxedUser, relaxedOrg, IssueOpts{AMR: []string{"pwd"}})

	relaxedRes, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: relaxedSession.RefreshToken})
	if err != nil {
		t.Fatalf("refresh in an organization with no MFA policy failed: %v", err)
	}
	relaxedClaims, err := h.tokens.Parse(relaxedRes.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse refreshed access token: %v", err)
	}
	if relaxedClaims.MFAEnabled {
		t.Fatalf("refreshing a session in an organization with no require_mfa policy still set mfa_enabled; the user would be locked out of a tenant that never asked for a second factor")
	}
}

// ------------------------------------------ SessionService.SwitchOrganization

// TestSestSwitchOrganization_RefusesOrgWithoutMembership is the authorization
// boundary this endpoint exists to enforce. The organization id arrives from
// the request; the only thing that may turn it into a token is a membership
// row. Getting this wrong is a complete cross-tenant break: any authenticated
// user could mint a token for any organization id they can guess or read.
func TestSestSwitchOrganization_RefusesOrgWithoutMembership(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	user := h.seedUser(t)
	home := h.seedOrg(t, "Sest Home Org")
	h.seedMembership(t, user, home, control.MemberRoleOwner)

	// A real organization belonging to somebody else.
	victimOwner := h.seedUser(t)
	victim := h.seedOrg(t, "Sest Victim Org")
	h.seedMembership(t, victimOwner, victim, control.MemberRoleOwner)

	before := h.countSessions(t, user.ID)

	res, err := h.svc.SwitchOrganization(ctx, SwitchOrganizationInput{
		UserID:         user.ID.String(),
		OrganizationID: victim.ID.String(),
	})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("a user with no membership switched into another tenant's organization: err=%v result=%+v — this is a full cross-tenant authorization break", err, res)
	}
	if res != nil {
		t.Fatalf("SwitchOrganization returned tokens alongside the refusal; a caller reading the result before the error would hand over a cross-tenant session")
	}
	if after := h.countSessions(t, user.ID); after != before {
		t.Fatalf("a session row was created for a refused organization switch (%d -> %d); the token exists in the database even though the caller was told no", before, after)
	}
}

// TestSestSwitchOrganization_IssuesTokenScopedToTheVerifiedOrg is the accepted
// path. The assertion that matters is on the CLAIMS: the new token must be
// scoped to the organization whose membership was verified, because everything
// downstream authorizes on org_id from the token.
func TestSestSwitchOrganization_IssuesTokenScopedToTheVerifiedOrg(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	user := h.seedUser(t)
	orgA := h.seedOrg(t, "Sest Org A")
	orgB := h.seedOrg(t, "Sest Org B")
	h.seedMembership(t, user, orgA, control.MemberRoleOwner)
	h.seedMembership(t, user, orgB, control.MemberRoleMember)
	h.seedActiveSession(t, user, orgA, IssueOpts{AMR: []string{"pwd"}})

	res, err := h.svc.SwitchOrganization(ctx, SwitchOrganizationInput{
		UserID:         user.ID.String(),
		OrganizationID: orgB.ID.String(),
		IPAddress:      "203.0.113.44",
		UserAgent:      "sest/1.0",
	})
	if err != nil {
		t.Fatalf("switching into an organization the user is a member of failed: %v", err)
	}
	if res.OrganizationID != orgB.ID.String() || res.Organization != orgB.Name {
		t.Fatalf("switch reported %q/%q, want %q/%q", res.OrganizationID, res.Organization, orgB.ID, orgB.Name)
	}

	claims, err := h.tokens.Parse(res.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse switched access token: %v", err)
	}
	if claims.OrgID != orgB.ID.String() {
		t.Fatalf("the new access token is scoped to org %q, want %q; every downstream authorization reads org_id from this claim", claims.OrgID, orgB.ID)
	}
	if claims.Subject != user.ID.String() {
		t.Fatalf("the new access token names subject %q, want %q", claims.Subject, user.ID)
	}

	// The switch is tracked: without a session row the token fails the liveness
	// check on its very first use.
	var row control.TokenSession
	if err := h.db.First(&row, "access_token_jti = ?", claims.ID).Error; err != nil {
		t.Fatalf("no session row was recorded for the switched token (%v); it would be rejected as unknown on first use", err)
	}
	if row.OrganizationID != orgB.ID {
		t.Fatalf("the session row records org %s, want %s; the session list a user sees would name the wrong tenant", row.OrganizationID, orgB.ID)
	}
	if n := h.countAudit(t, user.ID, "org_switch"); n != 1 {
		t.Fatalf("recorded %d org_switch audit events, want 1; a tenant change with no audit trail is invisible to an investigation", n)
	}
}

// TestSestSwitchOrganization_RefusesBadInputAndDeadUsers sweeps the remaining
// refusals. They matter for the same reason: each one is a path that would
// otherwise end in a minted token.
func TestSestSwitchOrganization_RefusesBadInputAndDeadUsers(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	member := h.seedUser(t)
	org := h.seedOrg(t, "Sest Switch Guard Org")
	h.seedMembership(t, member, org, control.MemberRoleMember)

	inactive := h.seedUser(t)
	inactiveOrg := h.seedOrg(t, "Sest Inactive Org")
	h.seedMembership(t, inactive, inactiveOrg, control.MemberRoleOwner)
	if err := h.db.Model(&control.User{}).Where("id = ?", inactive.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	tests := []struct {
		name  string
		in    SwitchOrganizationInput
		why   string
		count *control.User
	}{
		{
			name: "unparsable user id",
			in:   SwitchOrganizationInput{UserID: "not-a-uuid", OrganizationID: org.ID.String()},
			why:  "a caller-supplied subject that is not a uuid must not reach the membership lookup",
		},
		{
			name: "unparsable org id",
			in:   SwitchOrganizationInput{UserID: member.ID.String(), OrganizationID: "../../admin"},
			why:  "a malformed organization id must be refused before it is used in a query",
		},
		{
			name: "empty org id",
			in:   SwitchOrganizationInput{UserID: member.ID.String(), OrganizationID: "   "},
			why:  "an empty organization must never resolve to a default one",
		},
		{
			name: "unknown user",
			in:   SwitchOrganizationInput{UserID: uuid.NewString(), OrganizationID: org.ID.String()},
			why:  "a subject with no user row is not a user",
		},
		{
			name: "org that does not exist",
			in:   SwitchOrganizationInput{UserID: member.ID.String(), OrganizationID: uuid.NewString()},
			why:  "there is no membership for an organization that was never created",
		},
		{
			name:  "deactivated user",
			in:    SwitchOrganizationInput{UserID: inactive.ID.String(), OrganizationID: inactiveOrg.ID.String()},
			why:   "deactivation must hold even where the membership row still exists",
			count: inactive,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var before int64
			if tc.count != nil {
				before = h.countSessions(t, tc.count.ID)
			}
			res, err := h.svc.SwitchOrganization(ctx, tc.in)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("SwitchOrganization(%s) returned %v (result %+v), want ErrUnauthorized — %s", tc.name, err, res, tc.why)
			}
			if res != nil {
				t.Fatalf("SwitchOrganization(%s) returned a result alongside the error", tc.name)
			}
			if tc.count != nil {
				if after := h.countSessions(t, tc.count.ID); after != before {
					t.Fatalf("SwitchOrganization(%s) created a session row despite refusing (%d -> %d)", tc.name, before, after)
				}
			}
		})
	}
}

// TestSestSwitchOrganization_DropsImpersonationMarkers documents a real gap
// rather than a guarantee.
//
// SwitchOrganizationInput carries no actor and no impersonation flag, so the
// service CANNOT preserve them: the token it mints is an ordinary first-party
// session for the target user. Today nothing routes to SwitchOrganization (no
// handler registers it), so this is latent. The moment one does, the caller —
// not this function — has to refuse an impersonated principal, because an
// impersonating admin who calls it walks away with a clean, non-impersonated,
// fully refreshable token for the customer, and every downstream check that
// keys on imp/act_id (refresh refusal, platform-admin denial, audit
// attribution) stops seeing the impersonation.
//
// The assertion pins the fact this conclusion rests on. If someone later
// teaches SwitchOrganization to carry impersonation through, this test fails
// and they get to read the reasoning instead of silently changing it.
func TestSestSwitchOrganization_DropsImpersonationMarkers(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	user := h.seedUser(t)
	orgA := h.seedOrg(t, "Sest Imp Org A")
	orgB := h.seedOrg(t, "Sest Imp Org B")
	h.seedMembership(t, user, orgA, control.MemberRoleMember)
	h.seedMembership(t, user, orgB, control.MemberRoleMember)

	admin := h.seedUser(t)
	h.seedActiveSession(t, user, orgA, IssueOpts{
		AMR:         []string{"impersonation"},
		ActorID:     admin.ID.String(),
		Impersonate: true,
	})

	res, err := h.svc.SwitchOrganization(ctx, SwitchOrganizationInput{
		UserID:         user.ID.String(),
		OrganizationID: orgB.ID.String(),
	})
	if err != nil {
		t.Fatalf("SwitchOrganization: %v", err)
	}
	claims, err := h.tokens.Parse(res.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse switched token: %v", err)
	}
	if claims.Impersonate || claims.ActorID != "" {
		t.Fatalf("SwitchOrganization now carries impersonation markers (imp=%v act_id=%q). If that is deliberate, the impersonation time bound and the refresh refusal in TokenSessionService.Refresh must be re-checked against it; if it is not, an admin can launder an impersonated session into a first-party one",
			claims.Impersonate, claims.ActorID)
	}

	var row control.TokenSession
	if err := h.db.First(&row, "access_token_jti = ?", claims.ID).Error; err != nil {
		t.Fatalf("load switched session row: %v", err)
	}
	if row.ActorID != nil {
		t.Fatalf("the switched session records actor %v; see the comment above — the input carries no actor, so this could only come from somewhere unexpected", row.ActorID)
	}
}

// ------------------------------------------ SessionService.ListMyOrganizations

// TestSestListMyOrganizations_ReturnsOnlyTheCallersMemberships. This list drives
// the organization switcher, so it is both a UI feed and a disclosure surface:
// it must show every organization the caller belongs to and nothing else, and
// the "current" marker must follow the caller's actual session rather than
// defaulting to the first row.
func TestSestListMyOrganizations_ReturnsOnlyTheCallersMemberships(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	user := h.seedUser(t)
	orgA := h.seedOrg(t, "Sest List Org A")
	orgB := h.seedOrg(t, "Sest List Org B")
	h.seedMembership(t, user, orgA, control.MemberRoleOwner)
	h.seedMembership(t, user, orgB, control.MemberRoleMember)

	// A different tenant entirely.
	stranger := h.seedUser(t)
	strangerOrg := h.seedOrg(t, "Sest Stranger Org")
	h.seedMembership(t, stranger, strangerOrg, control.MemberRoleOwner)

	got, err := h.svc.ListMyOrganizations(ctx, user.ID.String(), orgB.ID.String())
	if err != nil {
		t.Fatalf("ListMyOrganizations: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d organizations, want 2 (%+v)", len(got), got)
	}

	byID := map[string]MyOrganizationResult{}
	for _, o := range got {
		if o.ID == strangerOrg.ID.String() {
			t.Fatalf("the list included an organization the caller is not a member of (%q); this endpoint would leak the existence and name of other tenants", o.Name)
		}
		byID[o.ID] = o
	}

	a, ok := byID[orgA.ID.String()]
	if !ok {
		t.Fatalf("org A is missing from the caller's own organization list; they would be unable to switch back into it")
	}
	if a.Role != string(control.MemberRoleOwner) {
		t.Fatalf("org A reports role %q, want %q; the UI hides administrative actions on this value", a.Role, control.MemberRoleOwner)
	}
	if a.Name != orgA.Name || a.Status != orgA.Status {
		t.Fatalf("org A reports name %q status %q, want %q / %q — the organization association was not loaded", a.Name, a.Status, orgA.Name, orgA.Status)
	}
	if a.IsDefault {
		t.Fatalf("org A is marked as the current organization, but the caller's session is in org B; the switcher would show the wrong tenant as active")
	}

	b := byID[orgB.ID.String()]
	if !b.IsDefault {
		t.Fatalf("org B is not marked current even though it is the session's organization; the switcher shows no active tenant")
	}
}

// TestSestListMyOrganizations_EdgeCases covers the shapes a handler depends on:
// an empty list must serialize as [] rather than null, an unknown or inactive
// caller must be refused, and a malformed id must not be treated as "no
// current org".
func TestSestListMyOrganizations_EdgeCases(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()

	loner := h.seedUser(t)

	t.Run("no memberships returns an empty non-nil slice", func(t *testing.T) {
		got, err := h.svc.ListMyOrganizations(ctx, loner.ID.String(), "")
		if err != nil {
			t.Fatalf("ListMyOrganizations: %v", err)
		}
		if got == nil {
			t.Fatalf("a user with no memberships got a nil slice; it serializes as JSON null and the client's list rendering breaks on it rather than showing an empty switcher")
		}
		if len(got) != 0 {
			t.Fatalf("a user with no memberships got %d organizations", len(got))
		}
	})

	t.Run("blank current org marks nothing as default", func(t *testing.T) {
		user := h.seedUser(t)
		org := h.seedOrg(t, "Sest Blank Current Org")
		h.seedMembership(t, user, org, control.MemberRoleMember)

		got, err := h.svc.ListMyOrganizations(ctx, user.ID.String(), "  ")
		if err != nil {
			t.Fatalf("ListMyOrganizations: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d organizations, want 1", len(got))
		}
		if got[0].IsDefault {
			t.Fatalf("an organization was marked current although the caller supplied no current organization; the switcher would claim a tenant is active that the session is not scoped to")
		}
	})

	t.Run("refusals", func(t *testing.T) {
		inactive := h.seedUser(t)
		if err := h.db.Model(&control.User{}).Where("id = ?", inactive.ID).Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate user: %v", err)
		}

		cases := []struct {
			name    string
			userID  string
			current string
		}{
			{"unparsable user id", "not-a-uuid", ""},
			{"blank user id", "   ", ""},
			{"unknown user", uuid.NewString(), ""},
			{"deactivated user", inactive.ID.String(), ""},
			{"unparsable current org", loner.ID.String(), "not-a-uuid"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := h.svc.ListMyOrganizations(ctx, tc.userID, tc.current)
				if !errors.Is(err, ErrUnauthorized) {
					t.Fatalf("ListMyOrganizations(%s) returned %v (%+v), want ErrUnauthorized", tc.name, err, got)
				}
				if got != nil {
					t.Fatalf("ListMyOrganizations(%s) returned data alongside the error", tc.name)
				}
			})
		}
	})
}

// -------------------------------------------- TokenSessionService.Revoke

// TestSestRevoke_MakesTheSessionUnusable. The point of logout is not that a
// column changes; it is that the credentials the client is holding stop
// working. Asserting revoked_at directly would pass even if nothing consulted
// it, so this drives the two paths that actually read the session's liveness
// from this package: refresh, and the active-session list a user is shown.
func TestSestRevoke_MakesTheSessionUnusable(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	user, _, issued := sestUserAndOrg(t, h)

	if err := h.tracker.Revoke(ctx, issued.AccessJTI, user.ID.String(), "203.0.113.7", "sest/1.0"); err != nil {
		t.Fatalf("Revoke on the caller's own session: %v", err)
	}

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("the revoked session's refresh token still minted a new session (%v); logging out would leave a way back in", err)
	}

	active, err := h.tracker.ListActive(ctx, user.ID, "user")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	for _, s := range active {
		if s.AccessTokenJTI == issued.AccessJTI {
			t.Fatalf("the revoked session is still listed as active; a user auditing their own sessions would be told a device is signed in that is not")
		}
	}

	if n := h.countAudit(t, user.ID, "logout"); n != 1 {
		t.Fatalf("recorded %d logout audit events, want 1; session termination must be attributable", n)
	}
}

// TestSestRevoke_RefusesToRevokeSomeoneElsesSession. accessJTI is supplied by
// the caller. Without the ownership check, any authenticated user who learned
// or guessed another session's jti could sign that user out at will — a denial
// of service against a specific account, and against every account at once if
// jtis were ever enumerable.
func TestSestRevoke_RefusesToRevokeSomeoneElsesSession(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	victim, _, victimTokens := sestUserAndOrg(t, h)
	attacker := h.seedUser(t)

	err := h.tracker.Revoke(ctx, victimTokens.AccessJTI, attacker.ID.String(), "198.51.100.9", "thief/1.0")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Revoke let one user terminate another user's session (err=%v); this is an authenticated denial of service against any account whose jti leaks", err)
	}

	// The victim's session must be untouched, not merely "the call returned an
	// error": a refusal that revokes first and complains afterwards is the same
	// bug.
	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: victimTokens.RefreshToken}); err != nil {
		t.Fatalf("the victim's session stopped working after another user's revoke attempt was refused: %v", err)
	}
	if n := h.countAudit(t, victim.ID, "logout"); n != 0 {
		t.Fatalf("a refused cross-user revoke wrote %d logout events against the victim; the audit trail would show a logout that never happened", n)
	}
}

// TestSestRevoke_MalformedSubjectCannotRevoke. Revoke parses the subject id and
// discards the parse error. That is only safe because the zero uuid it falls
// back to cannot match a real session's subject — this test is what keeps that
// true, and would fail loudly if the comparison were ever loosened.
func TestSestRevoke_MalformedSubjectCannotRevoke(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	_, _, issued := sestUserAndOrg(t, h)

	for _, subject := range []string{"", "not-a-uuid", uuid.Nil.String()} {
		err := h.tracker.Revoke(ctx, issued.AccessJTI, subject, "198.51.100.9", "thief/1.0")
		if !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("Revoke(subject=%q) returned %v, want ErrUnauthorized; the ignored uuid.Parse error would otherwise let a garbage subject terminate a real session", subject, err)
		}
	}

	if _, err := h.svc.Refresh(ctx, RefreshInput{RefreshToken: issued.RefreshToken}); err != nil {
		t.Fatalf("the session was revoked by a malformed subject id: %v", err)
	}
}

// TestSestRevoke_UnknownSessionIsANoOp. Logout is called with whatever jti the
// presented token carried, including tokens whose session row has already been
// cleaned up. It must succeed quietly: returning an error there would surface
// as a failed logout to a user who is, in fact, logged out — and would also
// tell a prober which jtis exist.
func TestSestRevoke_UnknownSessionIsANoOp(t *testing.T) {
	h := sestNewHarness(t, false)
	ctx := context.Background()
	user := h.seedUser(t)

	if err := h.tracker.Revoke(ctx, uuid.NewString(), user.ID.String(), "203.0.113.7", "sest/1.0"); err != nil {
		t.Fatalf("Revoke of an unknown session returned %v, want nil; a logout for an already-expired session must not fail", err)
	}
	if n := h.countAudit(t, user.ID, "logout"); n != 0 {
		t.Fatalf("revoking a session that does not exist wrote %d logout audit events; the audit log would record terminations that never occurred", n)
	}
}

// TestSestSwitchOrganization_AppliesTargetOrgMFAPolicy: MFA policy is per
// organization, so the factor requirement that applies is the TARGET
// organization's, decided at switch time. A user whose home organization does
// not require MFA must not carry that leniency into an organization that does —
// the token they walk away with has to say mfa_enabled, which is what makes the
// middleware refuse it until they present a second factor.
func TestSestSwitchOrganization_AppliesTargetOrgMFAPolicy(t *testing.T) {
	h := sestNewHarness(t, true)
	ctx := context.Background()

	user := h.seedUser(t)
	relaxed := h.seedOrg(t, "Sest Relaxed Org")
	strict := h.seedOrg(t, "Sest Strict Org")
	h.seedMembership(t, user, relaxed, control.MemberRoleMember)
	h.seedMembership(t, user, strict, control.MemberRoleMember)
	h.enableOrgMFAPolicy(t, strict.ID)

	// Switching into the organization that requires MFA, with only a password
	// behind this session.
	res, err := h.svc.SwitchOrganization(ctx, SwitchOrganizationInput{
		UserID:         user.ID.String(),
		OrganizationID: strict.ID.String(),
		MFA:            false,
	})
	if err != nil {
		t.Fatalf("SwitchOrganization into an org requiring MFA: %v", err)
	}
	strictClaims, err := h.tokens.Parse(res.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !strictClaims.MFAEnabled {
		t.Fatalf("switching into an organization that requires MFA produced a token with mfa_enabled=false; the middleware denies on mfa_enabled && !mfa, so this token would sail past the organization's own MFA requirement")
	}
	if strictClaims.MFA {
		t.Fatalf("the token claims mfa=true although no second factor was presented to this call; a switch must never assert a factor the user did not use")
	}

	// The converse, so the assertion above cannot be satisfied by simply always
	// setting the flag: an organization with no policy leaves it clear.
	res, err = h.svc.SwitchOrganization(ctx, SwitchOrganizationInput{
		UserID:         user.ID.String(),
		OrganizationID: relaxed.ID.String(),
	})
	if err != nil {
		t.Fatalf("SwitchOrganization into an org with no policy: %v", err)
	}
	relaxedClaims, err := h.tokens.Parse(res.Tokens.AccessToken, "access")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if relaxedClaims.MFAEnabled {
		t.Fatalf("switching into an organization with no MFA policy still set mfa_enabled; the user would be locked out of a tenant that never asked for a second factor")
	}
}
