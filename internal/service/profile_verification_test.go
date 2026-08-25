package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// prftSchema isolates this file's fixtures. Every helper below goes through
// testsupport.OpenSchema rather than opening the DSN directly: an unscoped
// handle migrates and TRUNCATEs tables in "public", which against a developer's
// own DATABASE_URL destroys real rows. Registered with DropSchemas in TestMain.
var prftSchema = testsupport.SchemaName("apiary_service_profileverify_test")

// prftModels is both the migration list and the truncation list, so the two can
// never drift apart — a model migrated but not truncated leaks rows between
// tests in this file and turns a real failure into an unreproducible one.
var prftModels = []any{
	&control.User{},
	&control.Organization{},
	&control.Membership{},
	&control.AuthAuditEvent{},
	&control.EmailVerificationToken{},
	&control.EmailChangeRequest{},
	&control.TokenSession{},
	&control.UserTOTP{},
	&control.PlatformPolicy{},
	&control.OrgPolicy{},
}

func prftDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, prftSchema, prftModels...)
	prftTruncate(t, db)
	t.Cleanup(func() { prftTruncate(t, db) })
	return db
}

// prftTruncate resolves table names through gorm's own naming strategy rather
// than hard-coding them. A hand-written list silently stops truncating a table
// the day a model gains a TableName() method, and the symptom is a stale row
// from a previous test rather than an obvious error.
func prftTruncate(t *testing.T, db *gorm.DB) {
	t.Helper()
	names := make([]string, 0, len(prftModels))
	for _, m := range prftModels {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			t.Fatalf("parse model %T: %v", m, err)
		}
		names = append(names, stmt.Schema.Table)
	}
	if err := db.Exec("TRUNCATE " + strings.Join(names, ", ") + " CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// prftProfileService wires the real ProfileService against the test schema.
//
// The EmailService is deliberately nil: RequestEmailChange tolerates that (it
// stores the request and skips the send), which keeps these tests off SMTP
// without a fake mailer whose behaviour could drift from the real sender. It
// also means the raw token exists nowhere the test can reach it — which is the
// point, and why the confirmation tests plant their own rows instead.
func prftProfileService(db *gorm.DB, sessions *TokenSessionService, policies *PolicyService) *ProfileService {
	users := repo.NewUserRepoGorm(db)
	svc := NewProfileService(
		&users,
		repo.NewOrganizationRepoGorm(db),
		repo.NewMembershipRepoGorm(db),
		repo.NewUserTOTPRepoGorm(db),
		policies,
		sessions,
		repo.NewAuthAuditRepoGorm(db),
	)
	svc.SetEmailChange(repo.NewEmailChangeRepoGorm(db), nil, "https://apiary.test")
	return svc
}

func prftSessionService(db *gorm.DB) *TokenSessionService {
	users := repo.NewUserRepoGorm(db)
	return NewTokenSessionService(
		nil,
		repo.NewTokenSessionRepoGorm(db),
		repo.NewAuthAuditRepoGorm(db),
		&users,
	)
}

func prftPolicyService(db *gorm.DB) *PolicyService {
	users := repo.NewUserRepoGorm(db)
	return NewPolicyService(
		repo.NewPlatformPolicyRepoGorm(db),
		repo.NewOrgPolicyRepoGorm(db),
		&users,
		nil,
	)
}

// prftSeedSession plants a live session for a user. Uniqueness on the JTI and
// hash columns means every call has to differ, hence the uuid.
func prftSeedSession(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID) *control.TokenSession {
	t.Helper()
	// The JTI columns are varchar(36) — exactly a UUID — so they take a bare one.
	access, refresh := uuid.New().String(), uuid.New().String()
	s := &control.TokenSession{
		SubjectID:        userID,
		SubjectKind:      "user",
		OrganizationID:   orgID,
		FamilyID:         uuid.New(),
		AccessTokenJTI:   access,
		RefreshTokenJTI:  refresh,
		AccessTokenHash:  "ah-" + access,
		RefreshTokenHash: "rh-" + refresh,
		AccessExpiresAt:  time.Now().UTC().Add(time.Hour),
		RefreshExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	if err := db.Create(s).Error; err != nil {
		t.Fatalf("create token session: %v", err)
	}
	return s
}

// prftSeedEmailChange plants an email-change request whose RAW token the test
// knows, mirroring seedPendingRegistration in registration_split_test.go and for
// the same reason: RequestEmailChange does not return the raw token and must not
// start doing so, since the only copy that should exist outside the mailer is
// the one in the email.
func prftSeedEmailChange(t *testing.T, db *gorm.DB, userID uuid.UUID, newEmail, rawToken string, expiresAt time.Time) *control.EmailChangeRequest {
	t.Helper()
	sum := sha256.Sum256([]byte(rawToken))
	req := &control.EmailChangeRequest{
		UserID:    userID,
		NewEmail:  newEmail,
		TokenHash: hex.EncodeToString(sum[:]),
		ExpiresAt: expiresAt,
	}
	if err := db.Create(req).Error; err != nil {
		t.Fatalf("create email change request: %v", err)
	}
	return req
}

func prftReloadUser(t *testing.T, db *gorm.DB, id uuid.UUID) *control.User {
	t.Helper()
	var u control.User
	if err := db.First(&u, "id = ?", id).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	return &u
}

func prftCountAudit(t *testing.T, db *gorm.DB, userID uuid.UUID, eventType string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&control.AuthAuditEvent{}).
		Where("subject_id = ? AND event_type = ?", userID, eventType).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	return n
}

// prftRecordingEnqueuer records WHICH organizations were asked for, not just how
// many. countingEnqueuer answers "how many" and cannot distinguish "provisioned
// the org this user owns" from "provisioned every org this user belongs to".
type prftRecordingEnqueuer struct {
	// Inherits no-op implementations of the job kinds this fake has no opinion
	// about — the fleet lifecycle ones. Overridden methods below still win.
	workflow.MockEnqueuer

	mu   sync.Mutex
	orgs []string
}

func (e *prftRecordingEnqueuer) EnqueueProvisionTenant(_ context.Context, args *workflow.ProvisionTenantArgs) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.orgs = append(e.orgs, args.OrganizationID)
	return nil
}

func (e *prftRecordingEnqueuer) EnqueueDestroyTenant(context.Context, *workflow.DestroyTenantArgs) error {
	return nil
}

func (e *prftRecordingEnqueuer) seen() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.orgs...)
}

const prftPassword = "correct-horse-battery"

// ---------------------------------------------------------------------------
// ChangePassword
// ---------------------------------------------------------------------------

// The current password is the only thing standing between a stolen access token
// and permanent control of the account: an attacker holding a session that has
// not yet been revoked would otherwise set a password of their own, and
// ChangePassword's own RevokeAll would then evict the legitimate owner.
//
// The test also asserts the two things a wrong guess must NOT do — change the
// password, and revoke sessions — because a refusal that still revoked would
// hand any thief a one-request logout of the real user.
func TestChangePasswordRefusesAWrongCurrentPassword(t *testing.T) {
	db := prftDB(t)
	user, org := seedPendingRegistration(t, db, "pw-wrong@example.com", "unused-token", control.MemberRoleOwner)
	session := prftSeedSession(t, db, user.ID, org.ID)
	svc := prftProfileService(db, prftSessionService(db), nil)

	err := svc.ChangePassword(t.Context(), user.ID, ChangePasswordInput{
		CurrentPassword: "not-the-current-password",
		NewPassword:     "a-brand-new-password",
		OrgID:           org.ID,
	})
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("err = %v, want ErrWrongPassword — a caller who did not know the current "+
			"password was allowed to set a new one, which turns a stolen access token into "+
			"permanent account takeover", err)
	}

	stored := prftReloadUser(t, db, user.ID)
	if !stored.CheckPassword(prftPassword) {
		t.Fatal("the original password no longer works after a REFUSED change — the refusal " +
			"wrote a new hash anyway")
	}
	if stored.CheckPassword("a-brand-new-password") {
		t.Fatal("the rejected new password now works — the current-password check did not gate the write")
	}

	var reloaded control.TokenSession
	if err := db.First(&reloaded, "id = ?", session.ID).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if reloaded.RevokedAt != nil {
		t.Fatal("a wrong-password attempt revoked the user's live sessions — that hands anyone " +
			"who can reach this endpoint a one-request logout of the real account owner")
	}
	if stored.TokensRevokedAt != nil {
		t.Fatal("a wrong-password attempt stamped tokens_revoked_at, invalidating every token the user holds")
	}
	if n := prftCountAudit(t, db, user.ID, "password_changed"); n != 0 {
		t.Fatalf("password_changed audit events = %d, want 0 — a refused change was recorded as a "+
			"successful one, which is what an incident reconstruction reads", n)
	}
}

// A successful change revokes EVERY session for the user, including the one that
// made the request. That is deliberate — the whole reason to change a password
// is usually that someone else may hold a session — and it is pinned here so it
// cannot be quietly relaxed into "all except mine" for convenience.
func TestChangePasswordRevokesEverySessionIncludingTheCallersOwn(t *testing.T) {
	db := prftDB(t)
	user, org := seedPendingRegistration(t, db, "pw-ok@example.com", "unused-token", control.MemberRoleOwner)
	callers := prftSeedSession(t, db, user.ID, org.ID)
	other := prftSeedSession(t, db, user.ID, org.ID)

	// A second user's session, to prove the revocation is scoped to the subject
	// and does not simply log out the whole table.
	bystander, bystanderOrg := seedPendingRegistration(t, db, "bystander@example.com", "unused-token-2", control.MemberRoleOwner)
	bystanderSession := prftSeedSession(t, db, bystander.ID, bystanderOrg.ID)

	svc := prftProfileService(db, prftSessionService(db), nil)

	before := time.Now().UTC().Add(-time.Second)
	if err := svc.ChangePassword(t.Context(), user.ID, ChangePasswordInput{
		CurrentPassword: prftPassword,
		NewPassword:     "an-entirely-different-password",
		OrgID:           org.ID,
	}); err != nil {
		t.Fatalf("change password: %v", err)
	}

	stored := prftReloadUser(t, db, user.ID)
	if !stored.CheckPassword("an-entirely-different-password") {
		t.Fatal("the new password does not work after a successful ChangePassword")
	}
	if stored.CheckPassword(prftPassword) {
		t.Fatal("the OLD password still works after a successful ChangePassword — a compromised " +
			"credential the user believes they have retired is still live")
	}

	for name, id := range map[string]uuid.UUID{"caller's own": callers.ID, "other device": other.ID} {
		var s control.TokenSession
		if err := db.First(&s, "id = ?", id).Error; err != nil {
			t.Fatalf("reload %s session: %v", name, err)
		}
		if s.RevokedAt == nil {
			t.Fatalf("the %s session survived a password change — whoever else holds a session "+
				"keeps their access despite the credential being rotated", name)
		}
	}

	if stored.TokensRevokedAt == nil {
		t.Fatal("tokens_revoked_at was not stamped — access tokens already minted stay valid " +
			"until they expire, because that column is what the token check compares against")
	}
	if stored.TokensRevokedAt.Before(before) {
		t.Fatalf("tokens_revoked_at = %v, want a time at or after the change (%v)", stored.TokensRevokedAt, before)
	}

	var untouched control.TokenSession
	if err := db.First(&untouched, "id = ?", bystanderSession.ID).Error; err != nil {
		t.Fatalf("reload bystander session: %v", err)
	}
	if untouched.RevokedAt != nil {
		t.Fatal("another user's session was revoked by this user's password change — the revocation " +
			"is not scoped to the subject")
	}

	if n := prftCountAudit(t, db, user.ID, "password_changed"); n != 1 {
		t.Fatalf("password_changed audit events = %d, want 1", n)
	}
}

// The organization's password policy is enforced on the NEW password, and a
// rejection must leave nothing changed. This pins the ordering inside
// ChangePassword: policy is consulted after the current-password check and
// before the write, so a policy failure cannot half-apply the change.
func TestChangePasswordEnforcesTheOrgPasswordPolicyAndChangesNothingWhenItFails(t *testing.T) {
	db := prftDB(t)
	user, org := seedPendingRegistration(t, db, "pw-policy@example.com", "unused-token", control.MemberRoleOwner)
	session := prftSeedSession(t, db, user.ID, org.ID)

	if err := db.Create(&control.OrgPolicy{
		OrganizationID: org.ID,
		Kind:           control.PolicyKindPasswordMinLength,
		Enabled:        true,
		Config:         datatypes.NewJSONType(control.PolicyConfig{MinLength: 24}),
		CreatedBy:      user.ID,
	}).Error; err != nil {
		t.Fatalf("create org policy: %v", err)
	}

	svc := prftProfileService(db, prftSessionService(db), prftPolicyService(db))

	const short = "twelve-chars"
	err := svc.ChangePassword(t.Context(), user.ID, ChangePasswordInput{
		CurrentPassword: prftPassword,
		NewPassword:     short,
		OrgID:           org.ID,
	})
	if !errors.Is(err, ErrPasswordPolicy) {
		t.Fatalf("err = %v, want ErrPasswordPolicy — a password below the organization's "+
			"minimum length was accepted, so the policy is advisory at the only endpoint "+
			"users actually change their password through", err)
	}

	stored := prftReloadUser(t, db, user.ID)
	if stored.CheckPassword(short) {
		t.Fatal("the policy-violating password was written anyway — the policy check runs after the write")
	}
	if !stored.CheckPassword(prftPassword) {
		t.Fatal("a policy rejection destroyed the existing password")
	}
	var reloaded control.TokenSession
	if err := db.First(&reloaded, "id = ?", session.ID).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if reloaded.RevokedAt != nil {
		t.Fatal("a policy rejection revoked the user's sessions — the user is logged out of an " +
			"account whose password did not change")
	}

	// And the same policy accepts a compliant password, so the test above cannot
	// be passing because ValidatePassword rejects everything.
	long := strings.Repeat("z", 30)
	if err := svc.ChangePassword(t.Context(), user.ID, ChangePasswordInput{
		CurrentPassword: prftPassword,
		NewPassword:     long,
		OrgID:           org.ID,
	}); err != nil {
		t.Fatalf("a policy-compliant password was rejected: %v", err)
	}
	if !prftReloadUser(t, db, user.ID).CheckPassword(long) {
		t.Fatal("a policy-compliant change did not take effect")
	}
}

// A password change for an id that has no user must be a clean, specific
// refusal. Reported as ErrUserNotFound rather than a nil-pointer panic, which is
// what the handler distinguishes from a 500.
func TestChangePasswordForAnUnknownUserIsErrUserNotFound(t *testing.T) {
	db := prftDB(t)
	svc := prftProfileService(db, prftSessionService(db), nil)

	err := svc.ChangePassword(t.Context(), uuid.New(), ChangePasswordInput{
		CurrentPassword: prftPassword,
		NewPassword:     "something-else-entirely",
	})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// GetProfile
// ---------------------------------------------------------------------------

// TOTPEnabled is what the UI renders as "two-factor is on". Reporting it for a
// half-finished enrolment — a secret generated but never confirmed with a code —
// tells a user they are protected by a factor they cannot actually produce, and
// they will then trust the account with it.
//
// A decision matrix over two independent flags, so table-driven.
func TestGetProfileReportsTOTPOnlyForACompletedEnrolment(t *testing.T) {
	cases := []struct {
		name        string
		mfaEnabled  bool
		totp        *bool // nil: no enrolment row at all; else UserTOTP.Verified
		wantMFA     bool
		wantTOTP    bool
		consequence string
	}{
		{
			name: "no mfa and no enrolment", mfaEnabled: false, totp: nil,
			wantMFA: false, wantTOTP: false,
			consequence: "a user with nothing set up was shown as protected",
		},
		{
			name: "enrolment verified but mfa switched off", mfaEnabled: false, totp: prftBool(true),
			wantMFA: false, wantTOTP: false,
			consequence: "a leftover verified secret was reported as an active factor even though " +
				"MFA is off for this account, so the profile page claims a protection the login path does not enforce",
		},
		{
			name: "mfa on but enrolment never started", mfaEnabled: true, totp: nil,
			wantMFA: true, wantTOTP: false,
			consequence: "MFA was reported as backed by TOTP with no enrolment row at all",
		},
		{
			name: "mfa on and enrolment started but not confirmed", mfaEnabled: true, totp: prftBool(false),
			wantMFA: true, wantTOTP: false,
			consequence: "an unconfirmed TOTP secret was reported as an enabled factor — the user " +
				"believes they have a second factor they have never successfully used",
		},
		{
			name: "mfa on and enrolment confirmed", mfaEnabled: true, totp: prftBool(true),
			wantMFA: true, wantTOTP: true,
			consequence: "a fully enrolled TOTP factor was reported as absent, which pushes the user " +
				"to re-enrol and replace a working secret",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := prftDB(t)
			user, org := seedPendingRegistration(t, db, "totp@example.com", "unused-token", control.MemberRoleOwner)
			if tc.mfaEnabled {
				if err := db.Model(&control.User{}).Where("id = ?", user.ID).
					Update("mfa_enabled", true).Error; err != nil {
					t.Fatalf("enable mfa: %v", err)
				}
			}
			if tc.totp != nil {
				if err := db.Create(&control.UserTOTP{
					UserID:   user.ID,
					Secret:   "JBSWY3DPEHPK3PXP",
					Verified: *tc.totp,
				}).Error; err != nil {
					t.Fatalf("create totp: %v", err)
				}
			}

			svc := prftProfileService(db, nil, nil)
			got, err := svc.GetProfile(t.Context(), user.ID, org.ID)
			if err != nil {
				t.Fatalf("get profile: %v", err)
			}
			if got.MFAEnabled != tc.wantMFA {
				t.Fatalf("MFAEnabled = %v, want %v", got.MFAEnabled, tc.wantMFA)
			}
			if got.TOTPEnabled != tc.wantTOTP {
				t.Fatalf("TOTPEnabled = %v, want %v — %s", got.TOTPEnabled, tc.wantTOTP, tc.consequence)
			}
			if got.User == nil || got.User.ID != user.ID {
				t.Fatal("GetProfile returned a different user than the one asked for")
			}
			if got.Organization == nil || got.Organization.ID != org.ID {
				t.Fatal("GetProfile returned a different organization than the one asked for")
			}
		})
	}
}

func prftBool(b bool) *bool { return &b }

func TestGetProfileForAnUnknownUserIsErrUserNotFound(t *testing.T) {
	db := prftDB(t)
	svc := prftProfileService(db, nil, nil)

	if _, err := svc.GetProfile(t.Context(), uuid.New(), uuid.New()); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound — a missing user must not surface as a nil "+
			"ProfileResult the handler then dereferences", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateProfile
// ---------------------------------------------------------------------------

// UpdateProfile is the ordinary "save my details" write, and it must never be a
// way to set an email address. The address is the account's recovery channel and
// the thing every verification token is bound to; a direct write here would let
// anyone holding a session repoint the account at a mailbox of their own without
// ever proving they can read it.
//
// The third case is the one that matters most: a request carrying BOTH names and
// an email must be refused whole. If the guard ran after the name write, an
// attacker would get a partial apply and, worse, a caller could discover that
// the email was the only rejected field.
func TestUpdateProfileRefusesToWriteAnEmailAddress(t *testing.T) {
	newEmail := "attacker@evil.example"
	first := "Changed"

	cases := []struct {
		name    string
		in      UpdateProfileInput
		wantErr error
	}{
		{
			name:    "nothing supplied",
			in:      UpdateProfileInput{},
			wantErr: ErrNothingToUpdate,
		},
		{
			name:    "email alone",
			in:      UpdateProfileInput{Email: &newEmail},
			wantErr: ErrEmailChangeRequiresVerification,
		},
		{
			name:    "email smuggled alongside a name",
			in:      UpdateProfileInput{FirstName: &first, Email: &newEmail},
			wantErr: ErrEmailChangeRequiresVerification,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := prftDB(t)
			user, _ := seedPendingRegistration(t, db, "update@example.com", "unused-token", control.MemberRoleOwner)
			svc := prftProfileService(db, nil, nil)

			got, err := svc.UpdateProfile(t.Context(), user.ID, tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if got != nil {
				t.Fatal("a refused UpdateProfile still returned a user")
			}

			stored := prftReloadUser(t, db, user.ID)
			if stored.Email != "update@example.com" {
				t.Fatalf("email = %q, want %q — UpdateProfile wrote an email address, bypassing "+
					"the confirmation link that is the only proof the new mailbox is readable "+
					"by the account holder", stored.Email, "update@example.com")
			}
			if stored.FirstName != "Test" {
				t.Fatalf("first_name = %q, want %q — the refusal was not atomic: the name write "+
					"landed before the email guard ran", stored.FirstName, "Test")
			}
		})
	}
}

func TestUpdateProfileWritesNamesAndReturnsTheStoredUser(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "names@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	first, last := "  Ada  ", "Lovelace"
	got, err := svc.UpdateProfile(t.Context(), user.ID, UpdateProfileInput{FirstName: &first, LastName: &last})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	// Trimmed by the repo. Asserted because the returned value is what the
	// handler serialises straight back to the client, so an untrimmed name would
	// be what every later comparison sees.
	if got.FirstName != "Ada" || got.LastName != "Lovelace" {
		t.Fatalf("returned user = %q %q, want %q %q", got.FirstName, got.LastName, "Ada", "Lovelace")
	}
	stored := prftReloadUser(t, db, user.ID)
	if stored.FirstName != "Ada" || stored.LastName != "Lovelace" {
		t.Fatalf("stored user = %q %q, want %q %q — UpdateProfile returned a value it never persisted",
			stored.FirstName, stored.LastName, "Ada", "Lovelace")
	}
	if stored.Email != "names@example.com" {
		t.Fatalf("email = %q — a name-only update moved the address", stored.Email)
	}
}

// ---------------------------------------------------------------------------
// RequestEmailChange
// ---------------------------------------------------------------------------

var prftHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// The request records where the address is going, stores a value of the right
// shape for the capability that authorises the move, and must not touch the live
// address: the current one keeps working until the new one is proven, which is
// what stops a mistyped address from locking a user out of their own account.
//
// The same honesty note as the Issue shape test applies to token_hash here. The
// raw token is 32 random bytes hex-encoded, so its SHA-256 has exactly the same
// shape and this check passes either way; the hash-only property itself is
// pinned in TestRequestEmailChangeMailsALinkWhoseTokenIsStoredOnlyAsAHash, which
// is the only test that sees the mailed token and the stored column at once.
func TestRequestEmailChangeStoresOnlyAHashAndLeavesTheLiveAddressAlone(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "current@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	before := time.Now().UTC()
	if err := svc.RequestEmailChange(t.Context(), user.ID, "  New.Address@EXAMPLE.com  "); err != nil {
		t.Fatalf("request email change: %v", err)
	}

	var reqs []control.EmailChangeRequest
	if err := db.Find(&reqs).Error; err != nil {
		t.Fatalf("load requests: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("email_change_requests = %d, want 1", len(reqs))
	}
	got := reqs[0]

	if got.NewEmail != "new.address@example.com" {
		t.Fatalf("new_email = %q, want %q — the target address is stored unnormalised, so the "+
			"same mailbox typed with different case becomes a different account address and the "+
			"unique index stops catching collisions", got.NewEmail, "new.address@example.com")
	}
	if !prftHex64.MatchString(got.TokenHash) {
		t.Fatalf("token_hash = %q, want 64 lower-case hex characters — anything else means the "+
			"column is holding something other than a SHA-256 digest, and the obvious candidate "+
			"is the raw token itself", got.TokenHash)
	}
	if got.ExpiresAt.Before(before.Add(emailChangeTTL - time.Minute)) {
		t.Fatalf("expires_at = %v, want roughly %v from now — a longer-lived link means a mailbox "+
			"compromised later still hands over the account", got.ExpiresAt, emailChangeTTL)
	}
	if got.ExpiresAt.After(before.Add(emailChangeTTL + time.Minute)) {
		t.Fatalf("expires_at = %v, want roughly %v from now", got.ExpiresAt, emailChangeTTL)
	}
	if got.UsedAt != nil {
		t.Fatal("a freshly created email change request is already marked used")
	}

	if stored := prftReloadUser(t, db, user.ID); stored.Email != "current@example.com" {
		t.Fatalf("email = %q, want %q — requesting a change moved the address before it was "+
			"confirmed, so a typo in the new address locks the user out irrecoverably",
			stored.Email, "current@example.com")
	}
}

var prftChangeLinkToken = regexp.MustCompile(`/confirm-email\?token=([0-9a-fA-F]+)`)

// Only the hash of the email-change token is stored, and the link that was
// actually mailed is the one that redeems it.
//
// This is the email-change twin of TestIssueMailsALinkWhoseTokenIsStoredOnlyAsAHash
// and exists for the same reason: every other test in this file plants its own
// request row, so nothing else compares what RequestEmailChange WROTE against
// what it SENT. Without this, a service that stored the raw token verbatim —
// read access to email_change_requests then being a live "point this account at
// my mailbox" link for every pending change — passes the whole file, and so does
// one that stored an unrelated digest no mailed link can ever match.
func TestRequestEmailChangeMailsALinkWhoseTokenIsStoredOnlyAsAHash(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "mailed-change@example.com", "unused-token", control.MemberRoleOwner)

	trap := prftStartMailTrap(t)
	users := repo.NewUserRepoGorm(db)
	svc := NewProfileService(&users, repo.NewOrganizationRepoGorm(db), repo.NewMembershipRepoGorm(db),
		repo.NewUserTOTPRepoGorm(db), nil, nil, repo.NewAuthAuditRepoGorm(db))
	svc.SetEmailChange(
		repo.NewEmailChangeRepoGorm(db),
		NewDirectEmailSender(NewEmailService(trap.host, trap.port, "", "", "noreply@apiary.test")),
		"https://apiary.test",
	)

	if err := svc.RequestEmailChange(t.Context(), user.ID, "elsewhere@example.com"); err != nil {
		t.Fatalf("request email change: %v", err)
	}

	msg := trap.waitForMessage(t)
	match := prftChangeLinkToken.FindStringSubmatch(msg)
	if match == nil {
		t.Fatalf("no /confirm-email?token=... link in the delivered message:\n%s", msg)
	}
	mailed := match[1]

	var req control.EmailChangeRequest
	if err := db.First(&req).Error; err != nil {
		t.Fatalf("load stored request: %v", err)
	}

	if req.TokenHash == mailed {
		t.Fatal("the token that was mailed to the new address is sitting verbatim in " +
			"email_change_requests.token_hash — read access to that table, or any backup of it, " +
			"is a working link for repointing every account with a change in flight")
	}
	sum := sha256.Sum256([]byte(mailed))
	if want := hex.EncodeToString(sum[:]); req.TokenHash != want {
		t.Fatalf("token_hash = %q, want the SHA-256 of the mailed token %q", req.TokenHash, want)
	}

	// And the link that went out actually works. Without this half, storing an
	// unrelated digest would satisfy both assertions above while leaving every
	// user who requested a change unable to complete one.
	if err := svc.ConfirmEmailChange(t.Context(), mailed); err != nil {
		t.Fatalf("the link that was mailed does not confirm the change: %v", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "elsewhere@example.com" {
		t.Fatalf("email = %q, want %q — the mailed link confirmed without applying the address",
			stored.Email, "elsewhere@example.com")
	}
}

// Requesting a change must retire any outstanding one. Without it, every address
// the user has ever typed keeps a live link pointed at it, so a mailbox the user
// changed their mind about — or an address entered by an attacker who briefly
// held a session — can still claim the account hours later.
func TestRequestEmailChangeInvalidatesAnOutstandingRequest(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "start@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	const stale = "the-first-link"
	prftSeedEmailChange(t, db, user.ID, "first@example.com", stale, time.Now().UTC().Add(time.Hour))

	if err := svc.RequestEmailChange(t.Context(), user.ID, "second@example.com"); err != nil {
		t.Fatalf("request email change: %v", err)
	}

	if err := svc.ConfirmEmailChange(t.Context(), stale); !errors.Is(err, ErrEmailChangeInvalid) {
		t.Fatalf("err = %v, want ErrEmailChangeInvalid — the superseded link still works, so "+
			"every address ever requested keeps a live claim on this account", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "start@example.com" {
		t.Fatalf("email = %q, want %q — a superseded link moved the address", stored.Email, "start@example.com")
	}
}

func TestRequestEmailChangeRejectsAnEmptyAddress(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "empty@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	if err := svc.RequestEmailChange(t.Context(), user.ID, "   "); !errors.Is(err, ErrNothingToUpdate) {
		t.Fatalf("err = %v, want ErrNothingToUpdate", err)
	}
	if n := countRows(t, db, &control.EmailChangeRequest{}); n != 0 {
		t.Fatalf("email_change_requests = %d, want 0 — a blank address minted a live token, and "+
			"confirming it would blank the account's address", n)
	}
}

// With no EmailChangeRepo wired there is no way to prove the new address, so the
// request must refuse rather than fall through to a direct write.
func TestRequestEmailChangeRefusesWhenTheRepositoryIsNotWired(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "unwired@example.com", "unused-token", control.MemberRoleOwner)

	users := repo.NewUserRepoGorm(db)
	svc := NewProfileService(&users, repo.NewOrganizationRepoGorm(db), repo.NewMembershipRepoGorm(db),
		repo.NewUserTOTPRepoGorm(db), nil, nil, repo.NewAuthAuditRepoGorm(db))

	if err := svc.RequestEmailChange(t.Context(), user.ID, "new@example.com"); !errors.Is(err, ErrEmailChangeRequiresVerification) {
		t.Fatalf("err = %v, want ErrEmailChangeRequiresVerification", err)
	}
	if err := svc.ConfirmEmailChange(t.Context(), "anything"); !errors.Is(err, ErrEmailChangeInvalid) {
		t.Fatalf("err = %v, want ErrEmailChangeInvalid", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "unwired@example.com" {
		t.Fatalf("email = %q — an unwired service still changed the address", stored.Email)
	}
}

// ---------------------------------------------------------------------------
// ConfirmEmailChange
// ---------------------------------------------------------------------------

// The token is bound to the address it was issued for, and it burns on use.
//
// The replay half is the security assertion. Consider a user who moves
// old -> a@ -> b@: if a@'s link stayed live, whoever controls a@ could replay it
// afterwards and drag the account back to a mailbox they read. Single-use is the
// only thing that closes that, because nothing else in this path checks that the
// request being confirmed is the most recent one.
func TestConfirmEmailChangeAppliesTheBoundAddressExactlyOnce(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "before@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	const tokenA = "link-mailed-to-a"
	const tokenB = "link-mailed-to-b"
	prftSeedEmailChange(t, db, user.ID, "a@example.com", tokenA, time.Now().UTC().Add(time.Hour))

	if err := svc.ConfirmEmailChange(t.Context(), tokenA); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "a@example.com" {
		t.Fatalf("email = %q, want %q — confirming the link did not apply the address it was "+
			"issued for", stored.Email, "a@example.com")
	}
	if n := prftCountAudit(t, db, user.ID, "email_changed"); n != 1 {
		t.Fatalf("email_changed audit events = %d, want 1 — an address change is exactly the event "+
			"an account-takeover investigation looks for", n)
	}

	// The user moves on to a second address.
	prftSeedEmailChange(t, db, user.ID, "b@example.com", tokenB, time.Now().UTC().Add(time.Hour))
	if err := svc.ConfirmEmailChange(t.Context(), tokenB); err != nil {
		t.Fatalf("confirm second change: %v", err)
	}

	// Replaying the first link must not drag the account back to a@.
	if err := svc.ConfirmEmailChange(t.Context(), tokenA); !errors.Is(err, ErrEmailChangeInvalid) {
		t.Fatalf("replayed token: err = %v, want ErrEmailChangeInvalid", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "b@example.com" {
		t.Fatalf("email = %q, want %q — replaying a spent confirmation link moved the account back "+
			"to an address the holder of that old mailbox controls", stored.Email, "b@example.com")
	}
	if n := prftCountAudit(t, db, user.ID, "email_changed"); n != 2 {
		t.Fatalf("email_changed audit events = %d, want 2 — a replay was recorded as a real change", n)
	}
}

// Expiry and unknown tokens are the same refusal on purpose: distinguishing them
// would turn this endpoint into an oracle over tokens the caller does not hold.
func TestConfirmEmailChangeRefusesExpiredAndUnknownTokens(t *testing.T) {
	cases := []struct {
		name    string
		seed    bool
		expires time.Duration
		token   string
	}{
		{name: "expired an hour ago", seed: true, expires: -time.Hour, token: "stale-link"},
		{name: "expired a second ago", seed: true, expires: -time.Second, token: "stale-link"},
		{name: "never issued", seed: false, token: "a-token-nobody-minted"},
		{name: "empty", seed: false, token: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := prftDB(t)
			user, _ := seedPendingRegistration(t, db, "expiry@example.com", "unused-token", control.MemberRoleOwner)
			svc := prftProfileService(db, nil, nil)
			if tc.seed {
				prftSeedEmailChange(t, db, user.ID, "late@example.com", tc.token, time.Now().UTC().Add(tc.expires))
			}

			if err := svc.ConfirmEmailChange(t.Context(), tc.token); !errors.Is(err, ErrEmailChangeInvalid) {
				t.Fatalf("err = %v, want ErrEmailChangeInvalid", err)
			}
			if stored := prftReloadUser(t, db, user.ID); stored.Email != "expiry@example.com" {
				t.Fatalf("email = %q, want %q — a link that should not have been redeemable moved "+
					"the account's address", stored.Email, "expiry@example.com")
			}
			if n := prftCountAudit(t, db, user.ID, "email_changed"); n != 0 {
				t.Fatalf("email_changed audit events = %d, want 0", n)
			}
		})
	}
}

// A token is trimmed before hashing, because mail clients and copy-paste add
// whitespace. Pinned so the trim is not removed as noise: without it a link the
// user pasted with a trailing newline reads as an invalid token, and the support
// answer is "request another one", which trains people to click more links.
func TestConfirmEmailChangeToleratesSurroundingWhitespace(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "ws@example.com", "unused-token", control.MemberRoleOwner)
	svc := prftProfileService(db, nil, nil)

	const raw = "token-with-whitespace"
	prftSeedEmailChange(t, db, user.ID, "trimmed@example.com", raw, time.Now().UTC().Add(time.Hour))

	if err := svc.ConfirmEmailChange(t.Context(), "  "+raw+"\n"); err != nil {
		t.Fatalf("confirm with surrounding whitespace: %v", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.Email != "trimmed@example.com" {
		t.Fatalf("email = %q, want %q", stored.Email, "trimmed@example.com")
	}
}

// What happens to email_verified_at when the address moves.
//
// This is the interaction with the login confirmation gate, so it is pinned in
// both directions:
//
//   - A user who was ALREADY verified stays verified. That is correct rather
//     than lax: the confirmation link was mailed to the NEW address and clicking
//     it is exactly the proof the stamp represents. Clearing it here would lock
//     a user out of their own account moments after they proved control of the
//     mailbox, and the only way back in would be a second link to the same
//     address.
//
//   - A user who was NOT verified does not become verified. Confirming an email
//     CHANGE must never be a way around the registration gate: the two flows
//     write different rows and only EmailVerificationService.Confirm is allowed
//     to stamp the column.
//
// If either half ever flips, the login gate silently changes meaning.
func TestConfirmEmailChangeDoesNotInventOrDestroyTheVerifiedStamp(t *testing.T) {
	for _, verified := range []bool{true, false} {
		name := "unverified user stays unverified"
		if verified {
			name = "verified user stays verified"
		}
		t.Run(name, func(t *testing.T) {
			db := prftDB(t)
			user, _ := seedPendingRegistration(t, db, "stamp@example.com", "unused-token", control.MemberRoleOwner)
			var stampedAt time.Time
			if verified {
				stampedAt = time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
				if err := db.Model(&control.User{}).Where("id = ?", user.ID).
					Update("email_verified_at", stampedAt).Error; err != nil {
					t.Fatalf("stamp verified: %v", err)
				}
			}

			svc := prftProfileService(db, nil, nil)
			const raw = "stamp-token"
			prftSeedEmailChange(t, db, user.ID, "moved@example.com", raw, time.Now().UTC().Add(time.Hour))
			if err := svc.ConfirmEmailChange(t.Context(), raw); err != nil {
				t.Fatalf("confirm: %v", err)
			}

			stored := prftReloadUser(t, db, user.ID)
			if stored.Email != "moved@example.com" {
				t.Fatalf("email = %q, want %q", stored.Email, "moved@example.com")
			}
			if stored.EmailVerified() != verified {
				if verified {
					t.Fatal("confirming an email change cleared email_verified_at — the user just " +
						"proved control of the new mailbox and is now refused by the login " +
						"confirmation gate on an account they can no longer sign in to")
				}
				t.Fatal("confirming an email change stamped email_verified_at on a user who had " +
					"never verified anything — the email-change flow became a second, unaudited " +
					"door past the login confirmation gate that EmailVerificationService.Confirm " +
					"is supposed to be the only key to")
			}
			if verified && !stored.EmailVerifiedAt.UTC().Truncate(time.Second).Equal(stampedAt) {
				t.Fatalf("email_verified_at = %v, want it left at %v — the stamp was rewritten, "+
					"so the audit record of WHEN the address was proven no longer matches the "+
					"address it was proven for", stored.EmailVerifiedAt.UTC(), stampedAt)
			}
		})
	}
}

// An old verification link must be dead after the address moves. This is the
// seam where the email-change flow and the registration flow meet: Confirm
// re-checks that the account still carries the address the token was issued for,
// and this proves that check survives a change made through ConfirmEmailChange.
func TestAVerificationLinkIsDeadOnceTheAddressHasBeenChanged(t *testing.T) {
	db := prftDB(t)
	const verifyToken = "the-registration-link"
	user, _ := seedPendingRegistration(t, db, "signup@example.com", verifyToken, control.MemberRoleOwner)

	profiles := prftProfileService(db, nil, nil)
	const changeToken = "the-change-link"
	prftSeedEmailChange(t, db, user.ID, "elsewhere@example.com", changeToken, time.Now().UTC().Add(time.Hour))
	if err := profiles.ConfirmEmailChange(t.Context(), changeToken); err != nil {
		t.Fatalf("confirm email change: %v", err)
	}

	jobs := &prftRecordingEnqueuer{}
	verify := prftVerificationService(db, jobs)
	if _, err := verify.Confirm(t.Context(), verifyToken); !errors.Is(err, ErrVerificationTokenInvalid) {
		t.Fatalf("err = %v, want ErrVerificationTokenInvalid — a link issued for the OLD address "+
			"confirmed the new one, laundering proof of one mailbox into proof of another", err)
	}
	if stored := prftReloadUser(t, db, user.ID); stored.EmailVerified() {
		t.Fatal("the account was marked verified by a link issued for a different address")
	}
	if orgs := jobs.seen(); len(orgs) != 0 {
		t.Fatalf("provisioning jobs = %v, want none", orgs)
	}
}

// ---------------------------------------------------------------------------
// EmailVerificationService.Issue
// ---------------------------------------------------------------------------

func prftVerificationService(db *gorm.DB, jobs workflow.Enqueuer) *EmailVerificationService {
	users := repo.NewUserRepoGorm(db)
	return NewEmailVerificationService(
		repo.NewEmailVerificationRepoGorm(db),
		&users,
		repo.NewMembershipRepoGorm(db),
		nil,
		jobs,
		repo.NewAuthAuditRepoGorm(db),
		"https://apiary.test",
	)
}

// The shape of an issued token: bound to the address, 64 hex characters, a
// 24-hour life, unused, and issuing it proves nothing on its own.
//
// Read the honesty note first: the checks here canNOT tell a stored digest apart
// from a stored raw token. The raw token is 32 random bytes hex-encoded and its
// SHA-256 is also 64 hex characters, so every assertion below passes either way.
// Feeding the stored column back to Confirm does not separate them either, since
// Confirm hashes whatever it is given. The property "only the hash is stored"
// is pinned in TestIssueMailsALinkWhoseTokenIsStoredOnlyAsAHash, which is the
// only test that can see both values at once.
func TestIssueStoresATokenBoundToTheAddressWithTheExpectedShape(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "issue@example.com", "seeded-token", control.MemberRoleOwner)
	// Drop the token the seed planted so the only row left is the one Issue mints.
	if err := db.Where("user_id = ?", user.ID).Delete(&control.EmailVerificationToken{}).Error; err != nil {
		t.Fatalf("clear seeded token: %v", err)
	}

	jobs := &prftRecordingEnqueuer{}
	svc := prftVerificationService(db, jobs)

	before := time.Now().UTC()
	if err := svc.Issue(t.Context(), user); err != nil {
		t.Fatalf("issue: %v", err)
	}

	var tokens []control.EmailVerificationToken
	if err := db.Find(&tokens).Error; err != nil {
		t.Fatalf("load tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("email_verification_tokens = %d, want 1", len(tokens))
	}
	got := tokens[0]

	if !prftHex64.MatchString(got.TokenHash) {
		t.Fatalf("token_hash = %q, want 64 lower-case hex characters", got.TokenHash)
	}
	if got.Email != user.Email {
		t.Fatalf("token email = %q, want %q — the token is not bound to the address it was "+
			"issued for, so Confirm's address check has nothing to compare against", got.Email, user.Email)
	}
	if got.ExpiresAt.Before(before.Add(verificationTokenTTL-time.Minute)) ||
		got.ExpiresAt.After(before.Add(verificationTokenTTL+time.Minute)) {
		t.Fatalf("expires_at = %v, want roughly %v from now", got.ExpiresAt, verificationTokenTTL)
	}
	if got.UsedAt != nil {
		t.Fatal("a freshly issued token is already marked used")
	}
	if stored := prftReloadUser(t, db, user.ID); stored.EmailVerified() {
		t.Fatal("issuing a link marked the address verified — issuing proves nothing, redeeming does")
	}

	if orgs := jobs.seen(); len(orgs) != 0 {
		t.Fatalf("provisioning jobs = %v, want none — issuing a link released a tenant database "+
			"before anyone had clicked it", orgs)
	}
}

// prftMailTrap is a minimal SMTP server that captures the message body.
//
// It exists because the raw verification token is deliberately unreachable from
// Go: Issue does not return it, and must not start doing so. The only place it
// legitimately appears is the email, so the only honest way to assert "what is
// in the database is not what was mailed" is to be the mail server. Everything
// short of this — a shape check, a round trip through Confirm — passes equally
// for a service that stores the raw token, as a mutation run confirmed.
type prftMailTrap struct {
	host string
	port int
	msgs chan string
}

func prftStartMailTrap(t *testing.T) *prftMailTrap {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	trap := &prftMailTrap{host: host, port: port, msgs: make(chan string, 4)}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go trap.handle(conn)
		}
	}()
	return trap
}

// handle speaks just enough ESMTP for net/smtp.SendMail. STARTTLS is not
// advertised and no AUTH is offered, which is why the EmailService under test is
// constructed with an empty username: SendMail then skips both.
func (m *prftMailTrap) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	br := bufio.NewReader(conn)
	say := func(s string) { fmt.Fprintf(conn, "%s\r\n", s) }
	say("220 mailtrap ESMTP")

	var body strings.Builder
	inData := false
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		if inData {
			if strings.TrimRight(line, "\r\n") == "." {
				inData = false
				select {
				case m.msgs <- body.String():
				default:
				}
				body.Reset()
				say("250 OK")
				continue
			}
			body.WriteString(line)
			continue
		}
		switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			say("250-mailtrap")
			say("250 SIZE 10240000")
		case cmd == "DATA":
			inData = true
			say("354 End data with <CR><LF>.<CR><LF>")
		case cmd == "QUIT":
			say("221 Bye")
			return
		default:
			say("250 OK")
		}
	}
}

func (m *prftMailTrap) waitForMessage(t *testing.T) string {
	t.Helper()
	select {
	case msg := <-m.msgs:
		return msg
	case <-time.After(15 * time.Second):
		t.Fatal("no verification email arrived — Issue reported success without sending a link, " +
			"which leaves the registrant with an account they can never confirm")
		return ""
	}
}

var prftLinkToken = regexp.MustCompile(`/verify-email\?token=([0-9a-fA-F]+)`)

// Only the hash is stored. A dump of this table must not yield a working
// confirmation link for every pending registration in it.
//
// This is the one test that sees both halves: the token as it goes out in the
// mail, and the value left behind in the database. The three assertions are
// deliberately overlapping — stored != mailed catches a service that stopped
// hashing, stored == sha256(mailed) catches one that hashed something else or
// truncated the digest, and Confirm(mailed) succeeding catches the reverse
// mistake where the two sides stop agreeing and no link works at all.
func TestIssueMailsALinkWhoseTokenIsStoredOnlyAsAHash(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "mailed@example.com", "seeded-token", control.MemberRoleOwner)
	if err := db.Where("user_id = ?", user.ID).Delete(&control.EmailVerificationToken{}).Error; err != nil {
		t.Fatalf("clear seeded token: %v", err)
	}

	trap := prftStartMailTrap(t)
	jobs := &prftRecordingEnqueuer{}
	users := repo.NewUserRepoGorm(db)
	svc := NewEmailVerificationService(
		repo.NewEmailVerificationRepoGorm(db),
		&users,
		repo.NewMembershipRepoGorm(db),
		NewDirectEmailSender(NewEmailService(trap.host, trap.port, "", "", "noreply@apiary.test")),
		jobs,
		repo.NewAuthAuditRepoGorm(db),
		"https://apiary.test",
	)

	if err := svc.Issue(t.Context(), user); err != nil {
		t.Fatalf("issue: %v", err)
	}

	msg := trap.waitForMessage(t)
	match := prftLinkToken.FindStringSubmatch(msg)
	if match == nil {
		t.Fatalf("no /verify-email?token=... link in the delivered message:\n%s", msg)
	}
	mailed := match[1]

	var tok control.EmailVerificationToken
	if err := db.First(&tok).Error; err != nil {
		t.Fatalf("load stored token: %v", err)
	}

	if tok.TokenHash == mailed {
		t.Fatal("the token that was mailed to the registrant is sitting verbatim in " +
			"email_verification_tokens.token_hash — read access to this table, or any backup of " +
			"it, is a working confirmation link for every pending registration")
	}
	sum := sha256.Sum256([]byte(mailed))
	if want := hex.EncodeToString(sum[:]); tok.TokenHash != want {
		t.Fatalf("token_hash = %q, want the SHA-256 of the mailed token %q — the stored value is "+
			"neither the digest of the link nor, therefore, something Confirm can match it against",
			tok.TokenHash, want)
	}

	// And the link that was actually mailed works. Without this half, a change
	// that made Issue store an unrelated digest would satisfy both assertions
	// above while leaving every registrant permanently unable to confirm.
	result, err := svc.Confirm(t.Context(), mailed)
	if err != nil {
		t.Fatalf("the link that was mailed to the registrant does not confirm the account: %v", err)
	}
	if result.AlreadyVerified {
		t.Fatal("a first confirmation reported the address as already verified")
	}
	if !prftReloadUser(t, db, user.ID).EmailVerified() {
		t.Fatal("the mailed link did not stamp email_verified_at")
	}
}

// Two links for one user must not be the same link. If the token were derived
// from anything stable — the user id, the address, a counter — then knowing a
// registrant's email would be enough to construct their confirmation link.
func TestIssueMintsADifferentTokenEveryTime(t *testing.T) {
	db := prftDB(t)
	user, _ := seedPendingRegistration(t, db, "unique@example.com", "seeded-token", control.MemberRoleOwner)
	if err := db.Where("user_id = ?", user.ID).Delete(&control.EmailVerificationToken{}).Error; err != nil {
		t.Fatalf("clear seeded token: %v", err)
	}
	svc := prftVerificationService(db, &prftRecordingEnqueuer{})

	for range 2 {
		if err := svc.Issue(t.Context(), user); err != nil {
			t.Fatalf("issue: %v", err)
		}
	}

	var hashes []string
	if err := db.Model(&control.EmailVerificationToken{}).Pluck("token_hash", &hashes).Error; err != nil {
		t.Fatalf("pluck hashes: %v", err)
	}
	if len(hashes) != 2 {
		t.Fatalf("email_verification_tokens = %d, want 2 — Issue must leave the previous link "+
			"alive, because the resend throttle counts issued tokens rather than replacing them", len(hashes))
	}
	if hashes[0] == hashes[1] {
		t.Fatal("two issued tokens have the same hash — the token is derived from something " +
			"stable rather than drawn at random, so anyone who can guess that input holds the link")
	}
}

// Issue reports its failures rather than swallowing them: registration
// propagates the error, because an account created with no way to ever confirm
// it is an account the registrant can never sign in to and cannot diagnose.
func TestIssueRefusesWhenItCannotPossiblyMintALink(t *testing.T) {
	db := prftDB(t)

	unwired := NewEmailVerificationService(nil, nil, nil, nil, nil, nil, "https://apiary.test")
	if err := unwired.Issue(t.Context(), &control.User{Email: "x@example.com"}); err == nil {
		t.Fatal("Issue with no token repository returned nil — registration would report success " +
			"for an account that can never be confirmed")
	}

	svc := prftVerificationService(db, &prftRecordingEnqueuer{})
	if err := svc.Issue(t.Context(), nil); err == nil {
		t.Fatal("Issue with no user returned nil")
	}
	if n := countRows(t, db, &control.EmailVerificationToken{}); n != 0 {
		t.Fatalf("email_verification_tokens = %d, want 0", n)
	}
}

// ---------------------------------------------------------------------------
// EmailVerificationService.provision
// ---------------------------------------------------------------------------

// Only an OWNER's confirmation releases a tenant database, and it releases only
// the organizations that user owns.
//
// The mixed case is the one that matters and the one a count-only assertion
// cannot see: this user owns two organizations and is a plain member of a third.
// A regression that reads "this user owns something, so provision everything
// they belong to" produces the right NUMBER of jobs for a single-org fixture and
// hands a stranger's organization a database here.
func TestProvisioningReleasesOnlyTheOrganizationsTheConfirmerOwns(t *testing.T) {
	db := prftDB(t)
	jobs := &prftRecordingEnqueuer{}
	svc := prftVerificationService(db, jobs)

	const raw = "mixed-membership-token"
	user, ownedA := seedPendingRegistration(t, db, "mixed@example.com", raw, control.MemberRoleOwner)

	ownedB := &control.Organization{Name: "Second Owned Org"}
	if err := db.Create(ownedB).Error; err != nil {
		t.Fatalf("create owned org: %v", err)
	}
	if err := db.Create(&control.Membership{UserID: user.ID, OrganizationID: ownedB.ID, Role: control.MemberRoleOwner}).Error; err != nil {
		t.Fatalf("create owner membership: %v", err)
	}

	// An organization the user was invited into. Their mailbox says nothing
	// about whether this one should exist.
	someoneElses := &control.Organization{Name: "Someone Else's Org"}
	if err := db.Create(someoneElses).Error; err != nil {
		t.Fatalf("create foreign org: %v", err)
	}
	if err := db.Create(&control.Membership{UserID: user.ID, OrganizationID: someoneElses.ID, Role: control.MemberRoleMember}).Error; err != nil {
		t.Fatalf("create member membership: %v", err)
	}

	// And an admin membership — admin is not owner, and the check is an equality
	// test against owner rather than a rank comparison.
	adminOrg := &control.Organization{Name: "Org Where They Are Admin"}
	if err := db.Create(adminOrg).Error; err != nil {
		t.Fatalf("create admin org: %v", err)
	}
	if err := db.Create(&control.Membership{UserID: user.ID, OrganizationID: adminOrg.ID, Role: control.MemberRoleAdmin}).Error; err != nil {
		t.Fatalf("create admin membership: %v", err)
	}

	result, err := svc.Confirm(t.Context(), raw)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if !result.TenantProvisioning {
		t.Fatal("an owner's confirmation did not report that provisioning was released")
	}

	want := map[string]bool{ownedA.ID.String(): true, ownedB.ID.String(): true}
	got := jobs.seen()
	if len(got) != len(want) {
		t.Fatalf("provisioning jobs = %v, want exactly the two owned organizations %v", got, want)
	}
	for _, id := range got {
		if !want[id] {
			switch id {
			case someoneElses.ID.String():
				t.Fatalf("provisioning was enqueued for %s — an organization this user is only a "+
					"MEMBER of got a Postgres database because an unrelated address was confirmed", id)
			case adminOrg.ID.String():
				t.Fatalf("provisioning was enqueued for %s — an ADMIN membership released a tenant, "+
					"but only the owner's confirmation may do that", id)
			default:
				t.Fatalf("provisioning was enqueued for an unexpected organization %s", id)
			}
		}
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("owned organizations that were never provisioned: %v", want)
	}
}

// A user who owns nothing gets no tenant, and the result says so. Distinct from
// the existing member-only test in registration_split_test.go: that user belongs
// to exactly one organization, so it cannot tell "skipped the non-owned org"
// apart from "skipped everything".
func TestProvisioningIsNotReleasedWhenTheConfirmerOwnsNothing(t *testing.T) {
	db := prftDB(t)
	jobs := &prftRecordingEnqueuer{}
	svc := prftVerificationService(db, jobs)

	const raw = "no-owned-orgs-token"
	user, _ := seedPendingRegistration(t, db, "member-only@example.com", raw, control.MemberRoleMember)

	second := &control.Organization{Name: "Another Org They Only Joined"}
	if err := db.Create(second).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	if err := db.Create(&control.Membership{UserID: user.ID, OrganizationID: second.ID, Role: control.MemberRoleAdmin}).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}

	result, err := svc.Confirm(t.Context(), raw)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if result.TenantProvisioning {
		t.Fatal("a confirmation from a user who owns no organization reported provisioning")
	}
	if orgs := jobs.seen(); len(orgs) != 0 {
		t.Fatalf("provisioning jobs = %v, want none — confirming a member's own address built a "+
			"tenant database for an organization they do not own", orgs)
	}

	// The address is still confirmed. Provisioning and verification are separate
	// outcomes and a member must not be left unverifiable because they own nothing.
	if stored := prftReloadUser(t, db, user.ID); !stored.EmailVerified() {
		t.Fatal("a member's confirmation did not mark their address verified, so they can never " +
			"sign in to the organization that invited them")
	}
}

// EnqueueSendEmail satisfies workflow.Enqueuer. Mail delivery is not this
// suite's subject; the call is accepted so the interface is satisfied without
// pretending to test anything about it.
func (e *prftRecordingEnqueuer) EnqueueSendEmail(context.Context, *workflow.SendEmailArgs) error {
	return nil
}
