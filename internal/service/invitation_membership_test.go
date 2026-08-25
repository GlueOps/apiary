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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// This suite owns its own schema. Opening TEST_DATABASE_URL unscoped would
// migrate and TRUNCATE tables in "public" — which, pointed at a developer's own
// DATABASE_URL, destroys real rows. See testsupport.OpenSchema.
var invmSchema = testsupport.SchemaName("apiary_service_invm_test")

// invmDB returns a handle scoped to this suite's schema, emptied before and
// after each test. Invitation acceptance writes users, memberships, the
// invitation row and an audit event, and several tests below assert on COUNTS
// of those rows — a leftover row from a sibling test would turn "acceptance
// created exactly one membership" into a coin flip.
func invmDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, invmSchema,
		&control.User{},
		&control.Organization{},
		&control.Membership{},
		&control.Invitation{},
		&control.AuthAuditEvent{},
	)
	const truncate = `TRUNCATE invitations, auth_audit_events, memberships, organizations, users CASCADE`
	if err := db.Exec(truncate).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() { db.Exec(truncate) })
	return db
}

// invmInvitationService wires the REAL repositories against the test schema.
//
// Driving the real gorm repos rather than fakes is deliberate: half of what
// Accept promises (the compare-and-set on accepted_at, the transaction that
// rolls a half-built account back, the unique-violation branch) lives in SQL,
// and a fake repo would answer for all of it without executing any of it.
func invmInvitationService(db *gorm.DB, rules AccessRuleResolver, email *EmailService, baseURL string) *InvitationService {
	users := repo.NewUserRepoGorm(db)
	return NewInvitationService(
		db,
		repo.NewInvitationRepoGorm(db),
		&users,
		repo.NewMembershipRepoGorm(db),
		repo.NewOrganizationRepoGorm(db),
		nil, // no PolicyService: password policy is PolicyService's own subject
		rules,
		NewDirectEmailSender(email),
		repo.NewAuthAuditRepoGorm(db),
		baseURL,
	)
}

func invmMembershipService(db *gorm.DB) *MembershipService {
	return NewMembershipService(db, repo.NewMembershipRepoGorm(db), repo.NewAuthAuditRepoGorm(db))
}

func invmSeedOrg(t *testing.T, db *gorm.DB, name string) *control.Organization {
	t.Helper()
	org := &control.Organization{Name: name}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("seed organization %q: %v", name, err)
	}
	return org
}

// invmSeedUser creates an account whose password the test knows, so that a test
// asserting "the password was NOT rewritten" has something to compare against.
func invmSeedUser(t *testing.T, db *gorm.DB, email, password string, verified bool) *control.User {
	t.Helper()
	u := &control.User{
		Email:     strings.ToLower(email),
		FirstName: "Existing",
		LastName:  "Account",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := u.SetPassword(password); err != nil {
		t.Fatalf("seed user password: %v", err)
	}
	if verified {
		now := time.Now().UTC()
		u.EmailVerifiedAt = &now
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user %q: %v", email, err)
	}
	return u
}

func invmSeedMembership(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID, role control.MemberRole) *control.Membership {
	t.Helper()
	m := &control.Membership{UserID: userID, OrganizationID: orgID, Role: role}
	if err := db.Create(m).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	return m
}

// invmPlantInvitation writes an invitation whose RAW token the test knows.
//
// Planted rather than captured from Invite, because Invite deliberately never
// returns the raw token — the only copy outside the hash is the one in the
// email. TestInviteEmailsALinkThatActuallyRedeems below covers that seam; every
// other test here wants a token in hand without an SMTP round trip.
func invmPlantInvitation(t *testing.T, db *gorm.DB, orgID, inviterID uuid.UUID, email string, role control.MemberRole, rawToken string, expiresAt time.Time) *control.Invitation {
	t.Helper()
	sum := sha256.Sum256([]byte(rawToken))
	inv := &control.Invitation{
		OrganizationID: orgID,
		InvitedByID:    inviterID,
		Email:          strings.ToLower(email),
		Role:           role,
		TokenHash:      hex.EncodeToString(sum[:]),
		ExpiresAt:      expiresAt,
	}
	if err := db.Create(inv).Error; err != nil {
		t.Fatalf("plant invitation: %v", err)
	}
	return inv
}

func invmReloadInvitation(t *testing.T, db *gorm.DB, id uuid.UUID) control.Invitation {
	t.Helper()
	var inv control.Invitation
	if err := db.First(&inv, "id = ?", id).Error; err != nil {
		t.Fatalf("reload invitation: %v", err)
	}
	return inv
}

func invmReloadUser(t *testing.T, db *gorm.DB, email string) *control.User {
	t.Helper()
	var u control.User
	err := db.First(&u, "email = ?", strings.ToLower(email)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	return &u
}

func invmCount(t *testing.T, db *gorm.DB, model any, where string, args ...any) int64 {
	t.Helper()
	var n int64
	q := db.Model(model)
	if where != "" {
		q = q.Where(where, args...)
	}
	if err := q.Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// ---------------------------------------------------------------- Accept

// The single most important assertion in this file.
//
// An invitation can be addressed to ANY email, including one that already has
// an account — that is the ordinary case for adding a colleague who is already
// a customer elsewhere. Accepting proves control of a MAILBOX; it proves
// nothing about knowledge of that account's existing password. If Accept let
// the redeemer supply a password and wrote it, then inviting alice@corp.example
// into an organization the attacker owns would hand the attacker Alice's
// account across the whole platform. Refusal, not silent discard: a discarded
// password leaves the invitee believing they set one and unable to sign in.
func TestAcceptRefusesAPasswordWhenTheAddressAlreadyHasAnAccount(t *testing.T) {
	db := invmDB(t)
	rules := &purposeRules{}
	svc := invmInvitationService(db, rules, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const theirPassword = "the-victims-own-password"
	victim := invmSeedUser(t, db, "victim@corp.example", theirPassword, true)

	const raw = "invm-takeover-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, victim.Email, control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token:     raw,
		Password:  "attacker-chosen-password",
		FirstName: "Not",
		LastName:  "Alice",
	})
	if !errors.Is(err, ErrPasswordNotAccepted) {
		t.Fatalf("err = %v, want ErrPasswordNotAccepted — an invitation to an address that "+
			"already has an account was allowed to carry a new password, which makes every "+
			"invitation to a known address a full account takeover", err)
	}

	reloaded := invmReloadUser(t, db, victim.Email)
	if reloaded == nil {
		t.Fatal("the existing account vanished")
	}
	if !reloaded.CheckPassword(theirPassword) {
		t.Fatal("the existing account no longer accepts its own password — the invitation " +
			"rewrote the credential of an account it only ever proved mailbox control over")
	}
	if reloaded.CheckPassword("attacker-chosen-password") {
		t.Fatal("the existing account now accepts the password supplied with the invitation — " +
			"this is account takeover")
	}

	// A refusal must not consume the invitation either: the legitimate owner of
	// the mailbox still has to be able to sign in and accept it.
	if got := invmReloadInvitation(t, db, inv.ID); got.AcceptedAt != nil {
		t.Fatal("a refused acceptance still stamped accepted_at, burning an invitation the " +
			"real recipient can no longer redeem")
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
		t.Fatalf("memberships = %d, want 0 — a refused acceptance still granted access", n)
	}
}

// A brand-new invitee arrives through an emailed link, which is precisely the
// proof /auth/verify-email demands. If Accept does not stamp email_verified_at,
// the account it creates is born locked out by the login confirmation gate with
// no pending verification token to redeem — an invitee who can never sign in,
// and a support ticket rather than a security hole, but a total functional
// break of the invitation flow.
func TestAcceptCreatesAConfirmedAccountWithTheInvitedRole(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	const raw = "invm-newcomer-token"
	const password = "correct horse battery staple"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, "newcomer@corp.example",
		control.MemberRoleAdmin, raw, time.Now().UTC().Add(time.Hour))

	res, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: password, FirstName: "New", LastName: "Comer",
	})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if !res.AccountCreated {
		t.Fatal("AccountCreated = false for an address that had no account — the UI uses this " +
			"to decide whether to say 'your account is ready' or 'sign in as usual'")
	}
	if res.Organization != "Acme" {
		t.Fatalf("Organization = %q, want %q — the accept page names the wrong tenant", res.Organization, "Acme")
	}

	created := invmReloadUser(t, db, "newcomer@corp.example")
	if created == nil {
		t.Fatal("no account was created for the invitee")
	}
	if !created.EmailVerified() {
		t.Fatal("email_verified_at was not stamped — redeeming a token mailed to this address " +
			"IS proof of control of it, and without the stamp the new account is created already " +
			"locked out by the login confirmation gate with no verification token to redeem")
	}
	if !created.CheckPassword(password) {
		t.Fatal("the password chosen while accepting was not stored")
	}

	var m control.Membership
	if err := db.First(&m, "user_id = ? AND organization_id = ?", created.ID, org.ID).Error; err != nil {
		t.Fatalf("load membership: %v", err)
	}
	if m.Role != control.MemberRoleAdmin {
		t.Fatalf("membership role = %q, want %q — acceptance granted a role other than the one "+
			"the inviter chose", m.Role, control.MemberRoleAdmin)
	}

	if got := invmReloadInvitation(t, db, inv.ID); got.AcceptedAt == nil {
		t.Fatal("accepted_at was not stamped, so the token stays redeemable forever")
	}
	if n := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "invitation_accepted"); n != 1 {
		t.Fatalf("invitation_accepted audit events = %d, want 1 — 'who let this account in' "+
			"must remain answerable", n)
	}
}

// An existing account that has never confirmed its address has, by clicking the
// invitation link, proven exactly what a verification token proves. Stamping it
// here is what stops an invited-but-unconfirmed colleague from being stuck.
func TestAcceptConfirmsAnExistingUnverifiedAccount(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	pending := invmSeedUser(t, db, "pending@corp.example", "their-own-password", false)

	const raw = "invm-unverified-token"
	invmPlantInvitation(t, db, org.ID, inviter.ID, pending.Email, control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	res, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: raw})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if res.AccountCreated {
		t.Fatal("AccountCreated = true for an address that already had an account")
	}
	reloaded := invmReloadUser(t, db, pending.Email)
	if !reloaded.EmailVerified() {
		t.Fatal("an existing unconfirmed account accepted an emailed invitation and was still " +
			"left unconfirmed, so it remains barred by the login gate despite having proven " +
			"control of the mailbox")
	}
}

// Replay. An invitation link lands in a mailbox and is clicked twice, prefetched
// by a scanner, or replayed deliberately. The second redemption must grant
// nothing and must not re-open the row.
func TestAcceptIsSingleUse(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	invitee := invmSeedUser(t, db, "member@corp.example", "their-own-password", true)

	const raw = "invm-replay-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, invitee.Email, control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	if _, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: raw}); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	firstStamp := invmReloadInvitation(t, db, inv.ID).AcceptedAt
	if firstStamp == nil {
		t.Fatal("first acceptance did not stamp accepted_at")
	}

	// The token no longer resolves to an OPEN invitation, and every one of the
	// four dead states answers with the same error so the unauthenticated
	// endpoint cannot be used to probe the table.
	if _, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: raw}); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("replayed token: err = %v, want ErrInvitationInvalid", err)
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 1 {
		t.Fatalf("memberships = %d, want 1 — a replayed invitation granted a second membership", n)
	}
	if got := invmReloadInvitation(t, db, inv.ID); !got.AcceptedAt.Equal(*firstStamp) {
		t.Fatal("a replayed acceptance moved accepted_at, re-dating a redemption that already happened")
	}
}

// Two open invitations for the same person into the same organization are
// possible by construction (the duplicate check in Invite is racy and there is
// no partial unique index — see repo.InvitationRepoGorm.Create). The second
// redemption must therefore be idempotent on the membership, and must not
// silently rewrite the role the person already holds.
func TestAcceptDoesNotGrantASecondMembershipOrRewriteAnExistingRole(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	invitee := invmSeedUser(t, db, "dup@corp.example", "their-own-password", true)

	const first = "invm-dup-token-one"
	const second = "invm-dup-token-two"
	invmPlantInvitation(t, db, org.ID, inviter.ID, invitee.Email, control.MemberRoleMember, first, time.Now().UTC().Add(time.Hour))
	// The second invitation names a HIGHER role on purpose: if acceptance were
	// not idempotent, a stale invitation would be a role-escalation primitive.
	invmPlantInvitation(t, db, org.ID, inviter.ID, invitee.Email, control.MemberRoleAdmin, second, time.Now().UTC().Add(time.Hour))

	if _, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: first}); err != nil {
		t.Fatalf("accept first: %v", err)
	}
	if _, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: second}); err != nil {
		t.Fatalf("accept second: %v", err)
	}

	var rows []control.Membership
	if err := db.Find(&rows, "user_id = ? AND organization_id = ?", invitee.ID, org.ID).Error; err != nil {
		t.Fatalf("load memberships: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("memberships = %d, want 1 — accepting a second invitation duplicated the "+
			"membership row, which makes owner counting and removal ambiguous", len(rows))
	}
	if rows[0].Role != control.MemberRoleMember {
		t.Fatalf("role = %q, want %q — a stale second invitation escalated an existing "+
			"member's role without any owner deciding to", rows[0].Role, control.MemberRoleMember)
	}
}

// An invitation is live for seven days. The join_org rule is re-evaluated at
// REDEMPTION, not only at issue, so a domain banned on day three cannot still
// walk in on day four.
func TestAcceptRechecksTheJoinOrgRuleAtRedemption(t *testing.T) {
	db := invmDB(t)
	rules := &purposeRules{}
	svc := invmInvitationService(db, rules, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	const raw = "invm-banned-later-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, "later-banned@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	// Day three: the platform bans the domain.
	rules.authenticate = repo.ErrAccessDenied

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: "correct horse battery staple", FirstName: "Later", LastName: "Banned",
	})
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied — an invitation issued before the domain "+
			"was banned was still redeemable after, so the ban is only advisory for anyone "+
			"holding a week-old link", err)
	}

	sawJoinOrg := false
	for _, p := range rules.seen {
		if p == repo.AccessPurposeJoinOrg {
			sawJoinOrg = true
		}
	}
	if !sawJoinOrg {
		t.Fatalf("purposes consulted = %v, want one of them to be %q — redemption asked the "+
			"wrong question, so a rule that bans joining organizations would not be applied here",
			rules.seen, repo.AccessPurposeJoinOrg)
	}

	if n := invmCount(t, db, &control.User{}, "email = ?", "later-banned@corp.example"); n != 0 {
		t.Fatalf("users = %d, want 0 — a refused redemption still created the account", n)
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
		t.Fatalf("memberships = %d, want 0 — a refused redemption still granted access", n)
	}
	if got := invmReloadInvitation(t, db, inv.ID); got.AcceptedAt != nil {
		t.Fatal("a refused redemption still burned the invitation")
	}
}

// An unwired resolver denies. It must never read as "no rules exist, let them
// in" — that turns a misconfiguration into an open door.
func TestAcceptWithNoAccessRuleResolverDenies(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, nil, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const raw = "invm-unwired-token"
	invmPlantInvitation(t, db, org.ID, inviter.ID, "someone@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: "correct horse battery staple", FirstName: "Some", LastName: "One",
	})
	if !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("err = %v, want ErrAdmissionNotConfigured — an unwired access-rule resolver "+
			"admitted an invitee, so a deployment that forgot to wire the rules has no ban list "+
			"at all", err)
	}
}

// Every dead invitation answers identically. The accept and preview endpoints
// are UNAUTHENTICATED: distinguishing "revoked" from "never existed" would let
// anyone holding a guessed token learn which ones are real.
func TestAcceptAndPreviewRefuseEveryDeadInvitationIdentically(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	now := time.Now().UTC()

	expired := invmPlantInvitation(t, db, org.ID, inviter.ID, "expired@corp.example",
		control.MemberRoleMember, "invm-dead-expired", now.Add(-time.Minute))
	revoked := invmPlantInvitation(t, db, org.ID, inviter.ID, "revoked@corp.example",
		control.MemberRoleMember, "invm-dead-revoked", now.Add(time.Hour))
	if err := db.Model(&control.Invitation{}).Where("id = ?", revoked.ID).
		Update("revoked_at", now).Error; err != nil {
		t.Fatalf("revoke: %v", err)
	}
	accepted := invmPlantInvitation(t, db, org.ID, inviter.ID, "accepted@corp.example",
		control.MemberRoleMember, "invm-dead-accepted", now.Add(time.Hour))
	if err := db.Model(&control.Invitation{}).Where("id = ?", accepted.ID).
		Update("accepted_at", now).Error; err != nil {
		t.Fatalf("mark accepted: %v", err)
	}
	_ = expired

	cases := []struct {
		name  string
		token string
	}{
		{"a token that names no invitation", "invm-token-that-never-existed"},
		{"an expired invitation", "invm-dead-expired"},
		{"a revoked invitation", "invm-dead-revoked"},
		{"an already accepted invitation", "invm-dead-accepted"},
		{"an empty token", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Preview(t.Context(), tc.token); !errors.Is(err, ErrInvitationInvalid) {
				t.Fatalf("Preview: err = %v, want ErrInvitationInvalid — %s is distinguishable "+
					"from an unknown token, which turns this unauthenticated endpoint into an "+
					"oracle for which tokens are real", err, tc.name)
			}
			_, err := svc.Accept(t.Context(), AcceptInvitationInput{
				Token: tc.token, Password: "correct horse battery staple", FirstName: "A", LastName: "B",
			})
			if !errors.Is(err, ErrInvitationInvalid) {
				t.Fatalf("Accept: err = %v, want ErrInvitationInvalid — %s was redeemable", err, tc.name)
			}
			if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
				t.Fatalf("memberships = %d, want 0 — %s granted access", n, tc.name)
			}
		})
	}
}

// A new invitee with no password has no account to fall back on; the refusal
// has to be its own error so the accept page can ask for one rather than
// reporting a dead link.
func TestAcceptRequiresAPasswordFromABrandNewInvitee(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const raw = "invm-nopassword-token"
	invmPlantInvitation(t, db, org.ID, inviter.ID, "brandnew@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{Token: raw, FirstName: "Brand", LastName: "New"})
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("err = %v, want ErrPasswordRequired — a passwordless account was created for a "+
			"brand-new invitee, which is an account nobody can ever sign in to but which now "+
			"occupies the address", err)
	}
	if n := invmCount(t, db, &control.User{}, ""); n != 1 { // only the inviter
		t.Fatalf("users = %d, want 1 — a refused acceptance created an account", n)
	}
}

// A blank name is refused and writes nothing.
//
// This is NOT the rollback test, though it reads like one: the name check fires
// before the first INSERT, so "nothing was written" holds whether or not the
// acceptance is transactional at all. TestAcceptRollsBackTheWholeAcceptance-
// WhenAStepFails below fails the LAST write instead, which is the only shape
// that actually exercises the rollback.
func TestAcceptRefusesABlankName(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const raw = "invm-noname-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, "noname@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: "correct horse battery staple", FirstName: "  ", LastName: "\t",
	})
	if !errors.Is(err, ErrRegistrationName) {
		t.Fatalf("err = %v, want ErrRegistrationName", err)
	}
	if n := invmCount(t, db, &control.User{}, "email = ?", "noname@corp.example"); n != 0 {
		t.Fatalf("users = %d, want 0 — the account survived a failed acceptance, so the address "+
			"is now taken by an account with no membership and no way in", n)
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
		t.Fatalf("memberships = %d, want 0", n)
	}
	if got := invmReloadInvitation(t, db, inv.ID); got.AcceptedAt != nil {
		t.Fatal("a failed acceptance burned the invitation, so the invitee can never retry")
	}
}

// Everything Accept writes lives in ONE transaction.
//
// The failure has to be injected at the LAST write — the compare-and-set on
// accepted_at — because that is the only point by which the user row and the
// membership row have both already been INSERTed. Any earlier failure (a blank
// name, a refused password) is raised before the first INSERT, so "no user, no
// membership" would hold with no transaction at all: deleting the
// db.Transaction wrapper entirely left this file green until this test existed.
//
// What it protects: a half-built acceptance leaves an account occupying the
// invitee's address with no membership and no way in, and the invitation still
// open — so the retry hits "this address already has an account" forever.
func TestAcceptRollsBackTheWholeAcceptanceWhenAStepFails(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const raw = "invm-rollback-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, "rollback@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	boom := errors.New("invm: injected failure on the invitation update")
	if err := db.Callback().Update().Before("gorm:update").
		Register("invm_fail_invitation_update", func(tx *gorm.DB) {
			if tx.Statement.Table == "invitations" {
				_ = tx.AddError(boom)
			}
		}); err != nil {
		t.Fatalf("register the failure injection: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Update().Remove("invm_fail_invitation_update") })

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: "correct horse battery staple", FirstName: "Roll", LastName: "Back",
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the injected failure — the last write in the acceptance did "+
			"not fail, so this test is not exercising what it claims to", err)
	}

	if n := invmCount(t, db, &control.User{}, "email = ?", "rollback@corp.example"); n != 0 {
		t.Fatalf("users = %d, want 0 — the account created earlier in the transaction survived "+
			"a failure later in it, so the invitee's address is now occupied by an account with "+
			"no membership, no session and no way to retry", n)
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
		t.Fatalf("memberships = %d, want 0 — a failed acceptance still granted access", n)
	}
	if got := invmReloadInvitation(t, db, inv.ID); got.AcceptedAt != nil {
		t.Fatal("a failed acceptance burned the invitation, so the invitee can never retry")
	}
}

// The compare-and-set on accepted_at, which is the only thing standing between
// two simultaneous redemptions of one link and two memberships.
//
// The revocation is committed on a SECOND connection from inside the
// acceptance, in the window between "this invitation is open" (openInvitation,
// which reads outside the transaction) and the write that closes it. That is
// exactly the interleaving the guard exists for, and it is unreachable without
// injection: every other test in this file redeems an invitation nobody is
// touching, so both `AND accepted_at IS NULL AND revoked_at IS NULL` and the
// RowsAffected check could be deleted with the whole file still green.
func TestAcceptRefusesAnInvitationRevokedDuringAcceptance(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	const raw = "invm-race-token"
	inv := invmPlantInvitation(t, db, org.ID, inviter.ID, "racer@corp.example",
		control.MemberRoleMember, raw, time.Now().UTC().Add(time.Hour))

	// A separate handle, so the revocation commits independently rather than
	// joining the transaction it is meant to be racing.
	other := testsupport.OpenSchema(t, invmSchema)

	var once sync.Once
	var revokeErr error
	if err := db.Callback().Update().Before("gorm:update").
		Register("invm_revoke_mid_acceptance", func(tx *gorm.DB) {
			if tx.Statement.Table != "invitations" {
				return
			}
			once.Do(func() {
				revokeErr = other.Model(&control.Invitation{}).
					Where("id = ?", inv.ID).
					Update("revoked_at", time.Now().UTC()).Error
			})
		}); err != nil {
		t.Fatalf("register the interleaving: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Update().Remove("invm_revoke_mid_acceptance") })

	_, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: raw, Password: "correct horse battery staple", FirstName: "Ra", LastName: "Cer",
	})
	if revokeErr != nil {
		t.Fatalf("the injected revocation did not commit (%v), so this test proved nothing", revokeErr)
	}
	if !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("err = %v, want ErrInvitationInvalid — the invitation was revoked before the "+
			"acceptance closed it and the acceptance went through anyway: the update that "+
			"stamps accepted_at is not re-checking that the row is still open, so two "+
			"simultaneous redemptions of one link both succeed", err)
	}

	got := invmReloadInvitation(t, db, inv.ID)
	if got.AcceptedAt != nil {
		t.Fatal("a revoked invitation was stamped accepted — revocation is advisory for anyone " +
			"whose acceptance is already in flight")
	}
	if n := invmCount(t, db, &control.User{}, "email = ?", "racer@corp.example"); n != 0 {
		t.Fatalf("users = %d, want 0 — the account built earlier in the transaction outlived "+
			"the failed compare-and-set", n)
	}
	if n := invmCount(t, db, &control.Membership{}, ""); n != 0 {
		t.Fatalf("memberships = %d, want 0 — a revoked invitation still granted access", n)
	}
}

// ---------------------------------------------------------------- Preview

// Preview is what the accept page renders before the invitee commits to
// anything. Possession of the token IS the authorization — it was mailed to the
// address it names — and the page needs to know whether to ask for a password
// or send the visitor to sign in.
func TestPreviewReportsWhetherTheInvitedAddressAlreadyHasAnAccount(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Wayne Enterprises")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	invmSeedUser(t, db, "known@corp.example", "their-own-password", true)

	invmPlantInvitation(t, db, org.ID, inviter.ID, "known@corp.example",
		control.MemberRoleAdmin, "invm-preview-known", time.Now().UTC().Add(time.Hour))
	invmPlantInvitation(t, db, org.ID, inviter.ID, "stranger@corp.example",
		control.MemberRoleMember, "invm-preview-stranger", time.Now().UTC().Add(time.Hour))

	cases := []struct {
		name         string
		token        string
		wantEmail    string
		wantRole     control.MemberRole
		needsAccount bool
	}{
		{"address with an account", "invm-preview-known", "known@corp.example", control.MemberRoleAdmin, false},
		{"address with no account", "invm-preview-stranger", "stranger@corp.example", control.MemberRoleMember, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.Preview(t.Context(), tc.token)
			if err != nil {
				t.Fatalf("preview: %v", err)
			}
			if got.Email != tc.wantEmail {
				t.Fatalf("Email = %q, want %q — the page would ask the visitor to accept on "+
					"behalf of the wrong address", got.Email, tc.wantEmail)
			}
			if got.Role != tc.wantRole {
				t.Fatalf("Role = %q, want %q — the page promises a different level of access "+
					"than the invitation actually grants", got.Role, tc.wantRole)
			}
			if got.Organization != "Wayne Enterprises" {
				t.Fatalf("Organization = %q, want %q — the visitor is asked to join an "+
					"organization the page cannot name correctly", got.Organization, "Wayne Enterprises")
			}
			if got.NeedsAccount != tc.needsAccount {
				t.Fatalf("NeedsAccount = %v, want %v — the accept page would %s, and Accept "+
					"refuses that combination, so the invitation becomes un-acceptable through "+
					"the UI", got.NeedsAccount, tc.needsAccount,
					map[bool]string{true: "hide the password field from someone who must set one",
						false: "ask an existing account for a password"}[tc.needsAccount])
			}
			if got.NeedsAccountName != tc.needsAccount {
				t.Fatalf("NeedsAccountName = %v, want %v — Accept requires a name whenever it "+
					"creates an account", got.NeedsAccountName, tc.needsAccount)
			}
		})
	}
}

// Preview leading/trailing whitespace: links get wrapped by mail clients and
// pasted with a stray space. Trimming is what keeps a legitimate token working.
func TestPreviewTrimsWhitespaceAroundTheToken(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	invmPlantInvitation(t, db, org.ID, inviter.ID, "spaced@corp.example",
		control.MemberRoleMember, "invm-spaced-token", time.Now().UTC().Add(time.Hour))

	if _, err := svc.Preview(t.Context(), "  invm-spaced-token\n"); err != nil {
		t.Fatalf("preview with surrounding whitespace: %v — a token pasted out of a wrapped "+
			"email is reported as an invalid invitation", err)
	}
}

// ---------------------------------------------------------------- Invite

// Owner is deliberately not invitable. Ownership is transferred by an existing
// owner through UpdateRole, where the last-owner and owner-line guards live;
// emailing it to an address means an admin could mint an owner (and then have
// that owner promote them) without any owner ever deciding to.
func TestInviteRefusesRolesThatAreNotInvitable(t *testing.T) {
	cases := []struct {
		role      control.MemberRole
		invitable bool
	}{
		{control.MemberRoleMember, true},
		{control.MemberRoleAdmin, true},
		{control.MemberRoleOwner, false},
		{control.MemberRole("superuser"), false},
		{control.MemberRole(""), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.role)+"/"+fmt.Sprint(tc.invitable), func(t *testing.T) {
			db := invmDB(t)
			svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")
			org := invmSeedOrg(t, db, "Acme")
			inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

			_, err := svc.Invite(t.Context(), InviteInput{
				OrgID: org.ID, InviterID: inviter.ID, Email: "candidate@corp.example", Role: tc.role,
			})
			if tc.invitable {
				if err != nil {
					t.Fatalf("Invite(%q): %v, want success", tc.role, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Invite(%q) succeeded — a role that must never be handed out by email "+
					"was emailed to a stranger", tc.role)
			}
			if n := invmCount(t, db, &control.Invitation{}, ""); n != 0 {
				t.Fatalf("invitations = %d, want 0 — a refused role still wrote a redeemable "+
					"invitation row", n)
			}
		})
	}
}

// Inviting someone who is already in does nothing, so it is reported to the
// INVITER as an error rather than silently mailing a link that grants nothing.
// The case-folding half is the one that matters: without normalisation,
// "Alice@Corp.example" slips past the membership check and mails a live
// invitation to an existing member.
func TestInviteRefusesAnAddressThatIsAlreadyAMember(t *testing.T) {
	cases := []struct {
		name  string
		email string
	}{
		{"exact address", "alice@corp.example"},
		{"different case", "Alice@Corp.Example"},
		{"surrounding whitespace", "  alice@corp.example  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := invmDB(t)
			svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")
			org := invmSeedOrg(t, db, "Acme")
			inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
			alice := invmSeedUser(t, db, "alice@corp.example", "alice-password-1", true)
			invmSeedMembership(t, db, alice.ID, org.ID, control.MemberRoleMember)

			_, err := svc.Invite(t.Context(), InviteInput{
				OrgID: org.ID, InviterID: inviter.ID, Email: tc.email, Role: control.MemberRoleAdmin,
			})
			if !errors.Is(err, ErrAlreadyMember) {
				t.Fatalf("Invite(%q): err = %v, want ErrAlreadyMember — %q reached an existing "+
					"member as a fresh invitation", tc.email, err, tc.email)
			}
			if n := invmCount(t, db, &control.Invitation{}, ""); n != 0 {
				t.Fatalf("invitations = %d, want 0", n)
			}
		})
	}
}

// A member of ANOTHER organization is not a member of this one, so inviting them
// must work. Without this half, a bug that compared only on the user (and not on
// the organization) would leave the test above green while making it impossible
// to invite anyone who already uses the platform.
func TestInviteAllowsAMemberOfADifferentOrganization(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	acme := invmSeedOrg(t, db, "Acme")
	other := invmSeedOrg(t, db, "Other Co")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)
	bob := invmSeedUser(t, db, "bob@corp.example", "bob-password-11", true)
	invmSeedMembership(t, db, bob.ID, other.ID, control.MemberRoleOwner)

	view, err := svc.Invite(t.Context(), InviteInput{
		OrgID: acme.ID, InviterID: inviter.ID, Email: bob.Email, Role: control.MemberRoleMember,
	})
	if err != nil {
		t.Fatalf("Invite: %v — someone who belongs to a different organization could not be "+
			"invited into this one", err)
	}
	if view.Status != "pending" {
		t.Fatalf("Status = %q, want %q — a freshly created invitation is not shown as "+
			"outstanding, so nobody can tell it needs chasing or revoking", view.Status, "pending")
	}
	if !view.ExpiresAt.After(time.Now().UTC().Add(6*24*time.Hour)) ||
		view.ExpiresAt.After(time.Now().UTC().Add(8*24*time.Hour)) {
		t.Fatalf("ExpiresAt = %v, want roughly 7 days out — an invitation that never expires is "+
			"a permanent key to the organization sitting in a mailbox", view.ExpiresAt)
	}

	// The role the inviter asked for is the role the row carries. Without this,
	// Invite could hard-code any role it liked and every refusal test above
	// would still pass: they only ever assert which roles are REFUSED, never
	// that an accepted one is written through. Hard-coding owner here would
	// turn every invitation into an ownership grant, silently.
	stored := invmReloadInvitation(t, db, view.ID)
	if stored.Role != control.MemberRoleMember {
		t.Fatalf("stored invitation role = %q, want %q — Invite wrote a role other than the one "+
			"the inviter chose, so the level of access granted at redemption is decided by "+
			"nobody", stored.Role, control.MemberRoleMember)
	}
	if view.Role != control.MemberRoleMember {
		t.Fatalf("view.Role = %q, want %q — the members page reports a different role than the "+
			"invitation actually grants", view.Role, control.MemberRoleMember)
	}
	if stored.OrganizationID != acme.ID {
		t.Fatalf("stored organization = %v, want %v — the invitation grants membership of an "+
			"organization the inviter did not name", stored.OrganizationID, acme.ID)
	}
	if n := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "invitation_created"); n != 1 {
		t.Fatalf("invitation_created audit events = %d, want 1 — 'who invited this address' is "+
			"unanswerable, and the audit-count assertions on the refusal paths below are "+
			"satisfied by an audit write that never happens at all", n)
	}
}

// The join_org rule is consulted BEFORE any row is written, so a refused invite
// leaves nothing behind — and consulted at all, so a banned domain cannot be
// invited in by a member who happens to like them.
func TestInviteRefusesAnAddressTheJoinOrgRuleDenies(t *testing.T) {
	db := invmDB(t)
	rules := &purposeRules{authenticate: repo.ErrAccessDenied}
	svc := invmInvitationService(db, rules, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	_, err := svc.Invite(t.Context(), InviteInput{
		OrgID: org.ID, InviterID: inviter.ID, Email: "banned@corp.example", Role: control.MemberRoleMember,
	})
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied — a banned address was invited anyway", err)
	}
	if n := invmCount(t, db, &control.Invitation{}, ""); n != 0 {
		t.Fatalf("invitations = %d, want 0", n)
	}
	if len(rules.seen) == 0 || rules.seen[0] != repo.AccessPurposeJoinOrg {
		t.Fatalf("purposes consulted = %v, want %q first — the invite path asked the wrong "+
			"question of the rule set", rules.seen, repo.AccessPurposeJoinOrg)
	}

	// And unwired denies rather than reading as "no rules exist".
	unwired := invmInvitationService(db, nil, nil, "https://apiary.test")
	if _, err := unwired.Invite(t.Context(), InviteInput{
		OrgID: org.ID, InviterID: inviter.ID, Email: "anyone@corp.example", Role: control.MemberRoleMember,
	}); !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("err = %v, want ErrAdmissionNotConfigured — a deployment with no wired rule "+
			"resolver invites anybody", err)
	}
}

func TestInviteRefusesASecondOpenInvitationForTheSameAddress(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	in := InviteInput{OrgID: org.ID, InviterID: inviter.ID, Email: "twice@corp.example", Role: control.MemberRoleMember}
	if _, err := svc.Invite(t.Context(), in); err != nil {
		t.Fatalf("first invite: %v", err)
	}
	if _, err := svc.Invite(t.Context(), in); !errors.Is(err, repo.ErrInvitationExists) {
		t.Fatalf("second invite: err = %v, want repo.ErrInvitationExists — every re-click of "+
			"'invite' mints another live token for the same mailbox, so revoking one leaves the "+
			"others working", err)
	}
	if n := invmCount(t, db, &control.Invitation{}, ""); n != 1 {
		t.Fatalf("invitations = %d, want 1", n)
	}
}

// A blank address is refused and writes nothing.
//
// The sentinel is deliberately NOT pinned here: Invite currently answers a blank
// email with ErrPasswordRequired ("choose a password to finish setting up your
// account"), which is the wrong error for the wrong actor — the inviter is not
// setting a password. Asserting errors.Is(err, ErrPasswordRequired) would cement
// that; asserting the property leaves it free to be corrected. See the report.
func TestInviteRefusesABlankAddress(t *testing.T) {
	db := invmDB(t)
	svc := invmInvitationService(db, &purposeRules{}, nil, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	for _, email := range []string{"", "   ", "\t\n"} {
		if _, err := svc.Invite(t.Context(), InviteInput{
			OrgID: org.ID, InviterID: inviter.ID, Email: email, Role: control.MemberRoleMember,
		}); err == nil {
			t.Fatalf("Invite(%q) succeeded — an invitation addressed to nobody was created, and "+
				"its token would be redeemable by whoever ends up holding it", email)
		}
	}
	if n := invmCount(t, db, &control.Invitation{}, ""); n != 0 {
		t.Fatalf("invitations = %d, want 0", n)
	}
}

// ------------------------------------------------------- sendInvite / SMTP

// invmMailSink is a minimal SMTP server. It exists because the raw invitation
// token is never returned, never logged and never stored — the ONLY copy is the
// one in the email — so the property "the link we mailed actually redeems" is
// unobservable without intercepting the message.
//
// That property is worth the machinery: a regression that mailed the token HASH
// instead of the token, or that mailed the link to the inviter instead of the
// invitee, produces no error anywhere. The first locks every invitee out; the
// second mails a live credential to the wrong mailbox.
type invmMailSink struct {
	host string
	port int
	mail chan invmCapturedMail
}

type invmCapturedMail struct {
	rcpt string
	data string
}

func invmStartMailSink(t *testing.T) *invmMailSink {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	addr := ln.Addr().(*net.TCPAddr)
	sink := &invmMailSink{host: "127.0.0.1", port: addr.Port, mail: make(chan invmCapturedMail, 4)}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go sink.serve(conn)
		}
	}()
	return sink
}

func (s *invmMailSink) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))

	r := bufio.NewReader(conn)
	write := func(line string) bool {
		_, err := conn.Write([]byte(line + "\r\n"))
		return err == nil
	}
	if !write("220 invm.test ESMTP") {
		return
	}

	var captured invmCapturedMail
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		trimmed := strings.TrimSpace(line)
		// Verbs are matched case-insensitively, but the ARGUMENT is read from
		// the original line: an address is case-sensitive on the left of the @,
		// and upper-casing it here would have this sink report a recipient the
		// service never asked for.
		cmd := strings.ToUpper(trimmed)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			// A single-line 250 advertises no extensions: no STARTTLS to
			// negotiate and no AUTH, which is what keeps this sink small.
			write("250 invm.test")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO"):
			captured.rcpt = strings.Trim(strings.TrimSpace(trimmed[len("RCPT TO:"):]), "<>")
			write("250 OK")
		case strings.HasPrefix(cmd, "DATA"):
			write("354 End data with <CR><LF>.<CR><LF>")
			var body strings.Builder
			for {
				dl, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				body.WriteString(dl)
			}
			captured.data = body.String()
			select {
			case s.mail <- captured:
			default:
			}
			write("250 OK")
		case strings.HasPrefix(cmd, "QUIT"):
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (s *invmMailSink) await(t *testing.T) invmCapturedMail {
	t.Helper()
	select {
	case m := <-s.mail:
		return m
	case <-time.After(15 * time.Second):
		t.Fatal("no invitation email arrived — sendInvite delivered nothing, so an invitee " +
			"is never told they were invited and the token is lost with the request")
		return invmCapturedMail{}
	}
}

var invmAcceptURL = regexp.MustCompile(`/accept-invitation\?token=([0-9a-f]+)`)

// The end-to-end seam: Invite -> the mailed link -> Accept.
//
// It pins three things at once, each of which fails silently on its own:
// the mail goes to the INVITEE (not the inviter, whose address is the only other
// one in scope); the link carries the RAW token (mailing the stored hash would
// make every invitation permanently unredeemable); and that token is the one
// this invitation stored (mailing a token from a different invitation would let
// a link grant the wrong organization's membership).
func TestInviteEmailsALinkThatActuallyRedeems(t *testing.T) {
	db := invmDB(t)
	sink := invmStartMailSink(t)
	email := NewEmailService(sink.host, sink.port, "", "", "apiary@apiary.test")
	svc := invmInvitationService(db, &purposeRules{}, email, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	view, err := svc.Invite(t.Context(), InviteInput{
		OrgID: org.ID, InviterID: inviter.ID, Email: "invitee@corp.example", Role: control.MemberRoleAdmin,
	})
	if err != nil {
		t.Fatalf("invite: %v", err)
	}

	got := sink.await(t)
	if got.rcpt != "invitee@corp.example" {
		t.Fatalf("invitation was mailed to %q, want %q — a live invitation token was delivered "+
			"to the wrong mailbox", got.rcpt, "invitee@corp.example")
	}

	m := invmAcceptURL.FindStringSubmatch(got.data)
	if m == nil {
		t.Fatalf("no accept link in the invitation email; body was:\n%s", got.data)
	}
	rawToken := m[1]
	if !strings.Contains(got.data, "https://apiary.test/accept-invitation?token=") {
		t.Fatalf("the accept link does not point at the configured base URL; body was:\n%s", got.data)
	}

	sum := sha256.Sum256([]byte(rawToken))
	stored := invmReloadInvitation(t, db, view.ID)
	if hex.EncodeToString(sum[:]) != stored.TokenHash {
		t.Fatal("the token in the email does not hash to the token_hash stored for this " +
			"invitation — every invitee would find their link rejected as invalid")
	}
	// Belt and braces: redeem it for real. A hash comparison could still pass
	// while the link is unusable for some other reason.
	res, err := svc.Accept(t.Context(), AcceptInvitationInput{
		Token: rawToken, Password: "correct horse battery staple", FirstName: "In", LastName: "Vitee",
	})
	if err != nil {
		t.Fatalf("accepting the mailed link: %v — the link we sent does not work", err)
	}
	if !res.AccountCreated {
		t.Fatal("accepting the mailed link did not create the invitee's account")
	}

	// End to end, the role survives the whole round trip: invited as admin,
	// mailed, redeemed, and an admin membership is what exists at the end.
	created := invmReloadUser(t, db, "invitee@corp.example")
	if created == nil {
		t.Fatal("no account exists for the invitee after a successful acceptance")
	}
	var mem control.Membership
	if err := db.First(&mem, "user_id = ? AND organization_id = ?", created.ID, org.ID).Error; err != nil {
		t.Fatalf("load membership: %v", err)
	}
	if mem.Role != control.MemberRoleAdmin {
		t.Fatalf("membership role = %q, want %q — the invitation was issued for one role and "+
			"redeemed into another, so the access an inviter grants is not the access they chose",
			mem.Role, control.MemberRoleAdmin)
	}
}

// A dead mail server must not lose the invitation. The row is written and the
// send is fire-and-forget on purpose: an SMTP outage should mean "resend the
// link", not "the invitation never existed".
func TestInviteSurvivesAMailerThatCannotDeliver(t *testing.T) {
	db := invmDB(t)
	// Port 1 on loopback: nothing listens, so Send fails immediately.
	email := NewEmailService("127.0.0.1", 1, "", "", "apiary@apiary.test")
	svc := invmInvitationService(db, &purposeRules{}, email, "https://apiary.test")

	org := invmSeedOrg(t, db, "Acme")
	inviter := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	if _, err := svc.Invite(t.Context(), InviteInput{
		OrgID: org.ID, InviterID: inviter.ID, Email: "unreachable@corp.example", Role: control.MemberRoleMember,
	}); err != nil {
		t.Fatalf("invite: %v — a failed SMTP delivery propagated to the inviter, who would "+
			"retry and be told an invitation already exists", err)
	}
	if n := invmCount(t, db, &control.Invitation{}, ""); n != 1 {
		t.Fatalf("invitations = %d, want 1", n)
	}
}

// ------------------------------------------------- MembershipService.UpdateRole

// The owner line, in both directions, plus the floor of one owner.
//
// Each refusal asserts that the STORED role is unchanged, not merely that an
// error came back: a guard that returns an error after writing has guarded
// nothing.
func TestUpdateRoleGuardsTheOwnerLineAndTheLastOwner(t *testing.T) {
	cases := []struct {
		name string
		// roles of the memberships to seed; the first is the SUBJECT.
		subject   control.MemberRole
		coOwner   bool
		actorRole control.MemberRole
		newRole   control.MemberRole
		wantErr   error
		wantRole  control.MemberRole
	}{
		{
			name: "an admin may not demote an owner", subject: control.MemberRoleOwner, coOwner: true,
			actorRole: control.MemberRoleAdmin, newRole: control.MemberRoleMember,
			wantErr: ErrOwnerRoleRequiresOwner, wantRole: control.MemberRoleOwner,
		},
		{
			name: "an admin may not promote anyone to owner", subject: control.MemberRoleMember, coOwner: true,
			actorRole: control.MemberRoleAdmin, newRole: control.MemberRoleOwner,
			wantErr: ErrOwnerRoleRequiresOwner, wantRole: control.MemberRoleMember,
		},
		{
			name: "a member may not promote anyone to owner", subject: control.MemberRoleMember, coOwner: true,
			actorRole: control.MemberRoleMember, newRole: control.MemberRoleOwner,
			wantErr: ErrOwnerRoleRequiresOwner, wantRole: control.MemberRoleMember,
		},
		{
			name: "the last owner may not demote themselves", subject: control.MemberRoleOwner, coOwner: false,
			actorRole: control.MemberRoleOwner, newRole: control.MemberRoleAdmin,
			wantErr: ErrLastOwner, wantRole: control.MemberRoleOwner,
		},
		{
			name: "an owner may demote another owner when one remains", subject: control.MemberRoleOwner, coOwner: true,
			actorRole: control.MemberRoleOwner, newRole: control.MemberRoleAdmin,
			wantErr: nil, wantRole: control.MemberRoleAdmin,
		},
		{
			name: "an owner may promote a member to owner", subject: control.MemberRoleMember, coOwner: true,
			actorRole: control.MemberRoleOwner, newRole: control.MemberRoleOwner,
			wantErr: nil, wantRole: control.MemberRoleOwner,
		},
		{
			name: "an admin may promote a member to admin", subject: control.MemberRoleMember, coOwner: true,
			actorRole: control.MemberRoleAdmin, newRole: control.MemberRoleAdmin,
			wantErr: nil, wantRole: control.MemberRoleAdmin,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := invmDB(t)
			svc := invmMembershipService(db)
			org := invmSeedOrg(t, db, "Acme")

			subjectUser := invmSeedUser(t, db, "subject@corp.example", "subject-password", true)
			subject := invmSeedMembership(t, db, subjectUser.ID, org.ID, tc.subject)
			if tc.coOwner {
				co := invmSeedUser(t, db, "coowner@corp.example", "coowner-password", true)
				invmSeedMembership(t, db, co.ID, org.ID, control.MemberRoleOwner)
			}
			actor := invmSeedUser(t, db, "actor@corp.example", "actor-password-1", true)

			err := svc.UpdateRole(t.Context(), UpdateRoleInput{
				OrgID: org.ID, MembershipID: subject.ID, NewRole: tc.newRole,
				ActorID: actor.ID, ActorRole: tc.actorRole,
			})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}

			var after control.Membership
			if err := db.First(&after, "id = ?", subject.ID).Error; err != nil {
				t.Fatalf("reload membership: %v", err)
			}
			if after.Role != tc.wantRole {
				t.Fatalf("stored role = %q, want %q — %q: the guard reported a decision the "+
					"database does not agree with", after.Role, tc.wantRole, tc.name)
			}

			// Both directions. The refusal cases below assert zero audit rows,
			// which is trivially satisfied by never writing one — so a role
			// change that DID happen has to be shown in the log too, or the
			// audit assertions prove nothing.
			wantAudit := int64(0)
			if tc.wantErr == nil {
				wantAudit = 1
			}
			if n := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "membership_role_changed"); n != wantAudit {
				t.Fatalf("membership_role_changed audit events = %d, want %d — %q: the audit "+
					"trail disagrees with what actually happened to the membership", n, wantAudit, tc.name)
			}
		})
	}
}

// An unknown role name is refused outright rather than written through to a
// column whose value nothing in the authorization code recognises — a
// membership with role "supervisor" matches none of the owner/admin checks, so
// it would be silently powerless (or, if a check ever used != member, powerful).
func TestUpdateRoleRefusesAnUnknownRole(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)
	org := invmSeedOrg(t, db, "Acme")
	u := invmSeedUser(t, db, "subject@corp.example", "subject-password", true)
	m := invmSeedMembership(t, db, u.ID, org.ID, control.MemberRoleMember)

	err := svc.UpdateRole(t.Context(), UpdateRoleInput{
		OrgID: org.ID, MembershipID: m.ID, NewRole: control.MemberRole("superuser"),
		ActorID: u.ID, ActorRole: control.MemberRoleOwner,
	})
	if err == nil {
		t.Fatal("an unrecognised role was accepted — the membership now carries a role no " +
			"authorization check knows how to evaluate")
	}
	var after control.Membership
	if err := db.First(&after, "id = ?", m.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.Role != control.MemberRoleMember {
		t.Fatalf("stored role = %q, want %q", after.Role, control.MemberRoleMember)
	}
}

// The IDOR seam. The membership id arrives in the request path; the
// organization comes from the caller's principal. Without the comparison in
// resolve, an admin of ANY organization could rewrite roles in every other one
// by guessing ids — and the 404-shaped answer is what stops the endpoint being
// used to discover which ids are real.
func TestUpdateRoleRefusesAMembershipBelongingToAnotherOrganization(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)

	mine := invmSeedOrg(t, db, "Acme")
	theirs := invmSeedOrg(t, db, "Victim Co")
	attacker := invmSeedUser(t, db, "attacker@acme.example", "attacker-password", true)
	invmSeedMembership(t, db, attacker.ID, mine.ID, control.MemberRoleOwner)

	victimUser := invmSeedUser(t, db, "victim@victim.example", "victim-password-1", true)
	victim := invmSeedMembership(t, db, victimUser.ID, theirs.ID, control.MemberRoleMember)

	err := svc.UpdateRole(t.Context(), UpdateRoleInput{
		OrgID: mine.ID, MembershipID: victim.ID, NewRole: control.MemberRoleOwner,
		ActorID: attacker.ID, ActorRole: control.MemberRoleOwner,
	})
	if !errors.Is(err, ErrMembershipNotInOrg) {
		t.Fatalf("err = %v, want ErrMembershipNotInOrg — an owner of one organization addressed "+
			"a membership in another one and was not stopped at the tenancy check", err)
	}

	var after control.Membership
	if err := db.First(&after, "id = ?", victim.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.Role != control.MemberRoleMember {
		t.Fatalf("the other organization's membership is now %q, want %q — cross-tenant "+
			"privilege escalation", after.Role, control.MemberRoleMember)
	}
	if n := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "membership_role_changed"); n != 0 {
		t.Fatalf("membership_role_changed audit events = %d, want 0 — a refused change was "+
			"recorded as having happened", n)
	}
}

// A no-op is a no-op: setting the role someone already holds succeeds without
// touching the owner guards. This is what lets a UI submit the whole form
// without special-casing "unchanged" — and it must not become a back door,
// which the last-owner half below checks.
func TestUpdateRoleToTheSameRoleIsANoOp(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)
	org := invmSeedOrg(t, db, "Acme")
	u := invmSeedUser(t, db, "only-owner@corp.example", "owner-password-1", true)
	m := invmSeedMembership(t, db, u.ID, org.ID, control.MemberRoleOwner)

	// An admin re-asserting the sole owner's existing role crosses the owner
	// line on paper. It is allowed because nothing changes.
	if err := svc.UpdateRole(t.Context(), UpdateRoleInput{
		OrgID: org.ID, MembershipID: m.ID, NewRole: control.MemberRoleOwner,
		ActorID: u.ID, ActorRole: control.MemberRoleAdmin,
	}); err != nil {
		t.Fatalf("err = %v, want nil for a role change that changes nothing", err)
	}
	var after control.Membership
	if err := db.First(&after, "id = ?", m.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.Role != control.MemberRoleOwner {
		t.Fatalf("stored role = %q, want %q", after.Role, control.MemberRoleOwner)
	}
	if n := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "membership_role_changed"); n != 0 {
		t.Fatalf("membership_role_changed audit events = %d, want 0 — a change that did not "+
			"happen was written to the audit log, which is how an audit trail stops being "+
			"evidence of anything", n)
	}
}

// ---------------------------------------------------- MembershipService.Remove

// Removal crosses the same owner line as demotion, and hits the same floor: an
// organization with no owner has nobody who can appoint one, including the
// person who emptied it.
func TestRemoveGuardsTheOwnerLineAndTheLastOwner(t *testing.T) {
	cases := []struct {
		name      string
		subject   control.MemberRole
		coOwner   bool
		actorRole control.MemberRole
		wantErr   error
		removed   bool
	}{
		{
			name: "an admin may not remove an owner", subject: control.MemberRoleOwner, coOwner: true,
			actorRole: control.MemberRoleAdmin, wantErr: ErrOwnerRoleRequiresOwner, removed: false,
		},
		{
			name: "the last owner may not be removed", subject: control.MemberRoleOwner, coOwner: false,
			actorRole: control.MemberRoleOwner, wantErr: ErrLastOwner, removed: false,
		},
		{
			name: "an owner may remove another owner when one remains", subject: control.MemberRoleOwner, coOwner: true,
			actorRole: control.MemberRoleOwner, wantErr: nil, removed: true,
		},
		{
			name: "an admin may remove a plain member", subject: control.MemberRoleMember, coOwner: true,
			actorRole: control.MemberRoleAdmin, wantErr: nil, removed: true,
		},
		{
			name: "an admin may remove another admin", subject: control.MemberRoleAdmin, coOwner: true,
			actorRole: control.MemberRoleAdmin, wantErr: nil, removed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := invmDB(t)
			svc := invmMembershipService(db)
			org := invmSeedOrg(t, db, "Acme")

			subjectUser := invmSeedUser(t, db, "subject@corp.example", "subject-password", true)
			subject := invmSeedMembership(t, db, subjectUser.ID, org.ID, tc.subject)
			if tc.coOwner {
				co := invmSeedUser(t, db, "coowner@corp.example", "coowner-password", true)
				invmSeedMembership(t, db, co.ID, org.ID, control.MemberRoleOwner)
			}
			actor := invmSeedUser(t, db, "actor@corp.example", "actor-password-1", true)

			err := svc.Remove(t.Context(), RemoveMemberInput{
				OrgID: org.ID, MembershipID: subject.ID, ActorID: actor.ID, ActorRole: tc.actorRole,
			})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}

			n := invmCount(t, db, &control.Membership{}, "id = ?", subject.ID)
			if tc.removed && n != 0 {
				t.Fatalf("the membership is still present after a permitted removal (%q)", tc.name)
			}
			if !tc.removed && n != 1 {
				t.Fatalf("the membership was deleted despite the refusal (%q) — the guard "+
					"reported an error after already removing the row", tc.name)
			}

			wantAudit := int64(0)
			if tc.wantErr == nil {
				wantAudit = 1
			}
			if got := invmCount(t, db, &control.AuthAuditEvent{}, "event_type = ?", "membership_removed"); got != wantAudit {
				t.Fatalf("membership_removed audit events = %d, want %d — %q: an eviction that "+
					"happened is not in the log, or one that did not is", got, wantAudit, tc.name)
			}
		})
	}
}

// The IDOR seam again, on the destructive verb. Answering not-found rather than
// mutating is what stops an owner of one tenant deleting memberships in another
// by walking ids.
func TestRemoveRefusesAMembershipBelongingToAnotherOrganization(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)

	mine := invmSeedOrg(t, db, "Acme")
	theirs := invmSeedOrg(t, db, "Victim Co")
	attacker := invmSeedUser(t, db, "attacker@acme.example", "attacker-password", true)
	invmSeedMembership(t, db, attacker.ID, mine.ID, control.MemberRoleOwner)

	victimUser := invmSeedUser(t, db, "victim@victim.example", "victim-password-1", true)
	victim := invmSeedMembership(t, db, victimUser.ID, theirs.ID, control.MemberRoleAdmin)

	err := svc.Remove(t.Context(), RemoveMemberInput{
		OrgID: mine.ID, MembershipID: victim.ID, ActorID: attacker.ID, ActorRole: control.MemberRoleOwner,
	})
	if !errors.Is(err, ErrMembershipNotInOrg) {
		t.Fatalf("err = %v, want ErrMembershipNotInOrg — a membership in another organization "+
			"was addressed without being stopped at the tenancy check", err)
	}
	if n := invmCount(t, db, &control.Membership{}, "id = ?", victim.ID); n != 1 {
		t.Fatal("the other organization's membership was deleted — any owner can evict any " +
			"other tenant's members by guessing ids")
	}
}

// A membership id that names nothing answers the same way as one that belongs
// elsewhere. Two different errors here would be a membership-id oracle.
func TestMembershipOperationsOnAnUnknownIDAnswerNotFound(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)
	org := invmSeedOrg(t, db, "Acme")
	actor := invmSeedUser(t, db, "owner@acme.example", "owner-password-1", true)

	ghost := uuid.New()
	if err := svc.UpdateRole(t.Context(), UpdateRoleInput{
		OrgID: org.ID, MembershipID: ghost, NewRole: control.MemberRoleAdmin,
		ActorID: actor.ID, ActorRole: control.MemberRoleOwner,
	}); !errors.Is(err, ErrMembershipNotInOrg) {
		t.Fatalf("UpdateRole on an unknown id: err = %v, want ErrMembershipNotInOrg — a "+
			"distinguishable answer tells a caller which membership ids exist", err)
	}
	if err := svc.Remove(t.Context(), RemoveMemberInput{
		OrgID: org.ID, MembershipID: ghost, ActorID: actor.ID, ActorRole: control.MemberRoleOwner,
	}); !errors.Is(err, ErrMembershipNotInOrg) {
		t.Fatalf("Remove on an unknown id: err = %v, want ErrMembershipNotInOrg", err)
	}
}

// TestInvmLastOwnerInvariantSurvivesConcurrentDemotions is the regression guard
// for a defect that was empirically reproduced, not merely reasoned about: two
// concurrent demotions of two DIFFERENT owners each read a count of 2, each
// passed the `n <= 1` guard, and left the organization with ZERO owners.
//
// Locking only the row being changed does not fix it — the two requests touch
// different rows. The invariant is over the SET, so the whole set has to be
// locked. That is what lockOrgMemberships does.
//
// Deterministic when the code is correct: the row locks serialise the
// transactions, so exactly one demotion can win and an owner always remains. It
// is only non-deterministic when the code is BROKEN, which is why it runs
// several rounds — one round reproduced the bug about 58% of the time, so a
// single round would let a regression through roughly two times in five.
func TestInvmLastOwnerInvariantSurvivesConcurrentDemotions(t *testing.T) {
	db := invmDB(t)
	svc := invmMembershipService(db)

	const rounds = 6
	for round := range rounds {
		org := invmSeedOrg(t, db, fmt.Sprintf("race-org-%d", round))

		// Two owners, so exactly one demotion may legitimately succeed.
		var ids []uuid.UUID
		for i := range 2 {
			u := invmSeedUser(t, db, fmt.Sprintf("owner%d-r%d@example.com", i, round), "correct-horse-battery", true)
			ids = append(ids, invmSeedMembership(t, db, u.ID, org.ID, control.MemberRoleOwner).ID)
		}

		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make([]error, len(ids))
		for i, id := range ids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs[i] = svc.UpdateRole(context.Background(), UpdateRoleInput{
					OrgID:        org.ID,
					MembershipID: id,
					NewRole:      control.MemberRoleMember,
					ActorID:      uuid.New(),
					ActorRole:    control.MemberRoleOwner,
				})
			}()
		}
		close(start)
		wg.Wait()

		var owners int64
		if err := db.Model(&control.Membership{}).
			Where("organization_id = ? AND role = ?", org.ID, control.MemberRoleOwner).
			Count(&owners).Error; err != nil {
			t.Fatalf("round %d: count owners: %v", round, err)
		}
		if owners == 0 {
			t.Fatalf("round %d: the organization has NO owners left. Two concurrent demotions "+
				"both passed the last-owner guard, which is the state ErrLastOwner exists to "+
				"make unreachable — and it is unrecoverable, because nobody left inside the "+
				"organization can promote an owner. errs=%v", round, errs)
		}

		// Exactly one demotion may succeed; the loser must be told why.
		refused := 0
		for _, err := range errs {
			if errors.Is(err, ErrLastOwner) {
				refused++
			}
		}
		if refused != 1 {
			t.Fatalf("round %d: %d of 2 demotions were refused with ErrLastOwner, want exactly 1 "+
				"(owners remaining=%d, errs=%v)", round, refused, owners, errs)
		}
	}
}
