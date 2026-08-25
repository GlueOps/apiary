package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// rsaStamp reads users.email_verified_at straight from the row rather than from
// whatever struct a repo method handed back. The property under test is what is
// PERSISTED — a method that returns a plausible timestamp while writing another
// one is exactly the failure these tests exist to catch.
func rsaStamp(t *testing.T, db *gorm.DB, userID uuid.UUID) *time.Time {
	t.Helper()
	var u control.User
	if err := db.First(&u, "id = ?", userID).Error; err != nil {
		t.Fatalf("reload user %s: %v", userID, err)
	}
	return u.EmailVerifiedAt
}

func rsaMembership(t *testing.T, db *gorm.DB, orgID, userID uuid.UUID) *control.Membership {
	t.Helper()
	var m control.Membership
	err := db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		t.Fatalf("load membership: %v", err)
	}
	return &m
}

// TestUpsertFromIdentity_KeepsTheFirstVerificationStamp is the reason both
// writes in UpsertFromIdentity use COALESCE(email_verified_at, now()) rather
// than a bare now().
//
// users.email_verified_at is evidence of how long an address has been proven —
// the admission ceiling reasons about that age when deciding whether an
// unproven organization may mint estate. A timestamp that moves forward every
// time the user signs in, or every time they connect another provider, proves
// nothing at all: an account created a year ago and one created this morning
// would be indistinguishable, and the ceiling that is supposed to slow down
// bulk registration would be trivially reset by a single OAuth round trip.
//
// Both COALESCE sites are exercised here on purpose: the second sign-in takes
// the "account row already exists" branch, and the second provider takes the
// "user exists, account does not" branch. They are separate SQL statements, so
// a regression can land in either one alone.
func TestUpsertFromIdentity_KeepsTheFirstVerificationStamp(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	email := uniqueEmail(t, "stamped")
	googleSubject := "google-" + uuid.NewString()

	created, err := r.UpsertFromIdentity(ctx, UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       googleSubject,
		Email:         email,
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	t.Cleanup(func() {
		db.Where("user_id = ?", created.ID).Delete(&control.Account{})
		db.Delete(&control.User{}, "id = ?", created.ID)
	})

	first := rsaStamp(t, db, created.ID)
	if first == nil {
		t.Fatalf("a user created from a provider-verified identity was left unconfirmed " +
			"(email_verified_at is NULL); they would meet the login confirmation gate " +
			"holding an address no confirmation email can ever be sent for")
	}
	if created.EmailVerifiedAt == nil || !created.EmailVerifiedAt.Equal(*first) {
		t.Fatalf("returned user's email_verified_at (%v) does not match the stored row (%v); "+
			"the method must re-read after stamping in SQL, or callers act on a stale struct",
			created.EmailVerifiedAt, first)
	}

	// Postgres now() is the transaction start time, so a fresh transaction is
	// needed for the clock to have moved at all. The sleep makes the difference
	// larger than timestamp resolution, so a restamp cannot be mistaken for
	// equality.
	time.Sleep(10 * time.Millisecond)

	again, err := r.UpsertFromIdentity(ctx, UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       googleSubject,
		Email:         email,
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("second sign-in with the same identity: %v", err)
	}
	if again.ID != created.ID {
		t.Fatalf("the same provider subject produced a second user (%s then %s); "+
			"the identity lookup is not matching on (provider, subject)", created.ID, again.ID)
	}
	if second := rsaStamp(t, db, created.ID); !second.Equal(*first) {
		t.Fatalf("signing in again moved email_verified_at from %v to %v; the stamp is "+
			"supposed to record when the address was FIRST proven, and the admission "+
			"ceiling that reads it can then be reset at will by re-authenticating",
			first, second)
	}

	time.Sleep(10 * time.Millisecond)

	linked, err := r.UpsertFromIdentity(ctx, UpsertSocialIdentityInput{
		Provider:      "github",
		Subject:       "github-" + uuid.NewString(),
		Email:         email,
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("linking a second provider to the same address: %v", err)
	}
	if linked.ID != created.ID {
		t.Fatalf("a second provider for the same verified address created user %s "+
			"instead of reusing %s", linked.ID, created.ID)
	}
	if third := rsaStamp(t, db, created.ID); !third.Equal(*first) {
		t.Fatalf("linking a second provider moved email_verified_at from %v to %v; "+
			"connecting an extra IdP is not new evidence about the address and must "+
			"not restamp the original proof", first, third)
	}
}

// TestUpsertFromIdentity_RequiresProviderAndSubject covers the guard that runs
// before any database work.
//
// (provider, subject) is the identity key. A blank subject would make every
// caller of that provider collapse onto one row: the first person to sign in
// wins the account and everyone after them is logged into it. Whitespace is
// trimmed first, so "   " must be refused exactly like "".
func TestUpsertFromIdentity_RequiresProviderAndSubject(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	cases := []struct {
		name     string
		provider string
		subject  string
	}{
		{"no provider", "", "subject-1"},
		{"no subject", "google", ""},
		{"whitespace subject", "google", "   "},
		{"whitespace provider", "  ", "subject-2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email := uniqueEmail(t, "keyless")
			_, err := r.UpsertFromIdentity(ctx, UpsertSocialIdentityInput{
				Provider:      tc.provider,
				Subject:       tc.subject,
				Email:         email,
				EmailVerified: true,
			})
			if err == nil {
				t.Fatalf("provider=%q subject=%q was accepted; an identity with no stable "+
					"key cannot be told apart from the next one that arrives", tc.provider, tc.subject)
			}
			if !strings.Contains(err.Error(), "provider and subject are required") {
				t.Fatalf("provider=%q subject=%q: got %v, want the provider/subject guard",
					tc.provider, tc.subject, err)
			}

			var users int64
			if err := db.Model(&control.User{}).Where("LOWER(email) = ?", email).Count(&users).Error; err != nil {
				t.Fatalf("count users: %v", err)
			}
			if users != 0 {
				t.Fatalf("a user row was created for an unkeyable identity")
			}
		})
	}
}

// TestUpsertFromIdentity_RefreshesTheKnownIdentity pins the "account row
// already exists" branch: the provider's current claims are written back, and
// the address is normalised on the way in.
//
// Normalisation is not cosmetic here. Every access rule, every conflict check
// and every auto-join keys on a lowercased address; a row stored as
// "Person@Example.TEST" is compared byte-for-byte against rules written in
// lower case and matches none of them, so an operator's deny silently stops
// applying to a user who happened to sign in from a provider that echoes the
// address back with different capitalisation.
func TestUpsertFromIdentity_RefreshesTheKnownIdentity(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	email := uniqueEmail(t, "known")
	user := mustCreateUser(t, db, email)
	subject := "known-" + uuid.NewString()
	acct := mustCreateAccount(t, db, user.ID, "google", subject, email, false)

	got, err := r.UpsertFromIdentity(ctx, UpsertSocialIdentityInput{
		Provider:      "  GOOGLE  ",
		Subject:       "  " + subject + "  ",
		Email:         "  " + strings.ToUpper(email) + " ",
		EmailVerified: true,
		Profile:       map[string]any{"name": "Ada", "hd": "example.test"},
	})
	if err != nil {
		t.Fatalf("re-authenticating a known identity: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("a known (provider, subject) resolved to user %s instead of %s; "+
			"the lookup is case- or whitespace-sensitive on the provider name", got.ID, user.ID)
	}
	if got.Email != email {
		t.Fatalf("stored email is %q, want the lower-cased %q; access rules and conflict "+
			"checks compare lower-cased addresses and would stop matching this user",
			got.Email, email)
	}

	var reloaded control.Account
	if err := db.First(&reloaded, "id = ?", acct.ID).Error; err != nil {
		t.Fatalf("reload account: %v", err)
	}
	if !reloaded.EmailVerified {
		t.Fatalf("the account row still says the provider has not verified this address; " +
			"EnsureAutoMembership only auto-joins on a verified account row, so the user " +
			"would be silently excluded from their own tenant")
	}
	if reloaded.Email == nil || *reloaded.Email != email {
		t.Fatalf("account email is %v, want %q", reloaded.Email, email)
	}
	// Decoded rather than string-matched: jsonb re-renders whitespace and key
	// order, so a textual comparison would assert Postgres' formatting instead
	// of the claims that were stored.
	var profile map[string]any
	if err := json.Unmarshal(reloaded.Profile, &profile); err != nil {
		t.Fatalf("stored profile is not valid JSON (%q): %v", string(reloaded.Profile), err)
	}
	if profile["name"] != "Ada" || profile["hd"] != "example.test" {
		t.Fatalf("the provider's profile claims were not stored (%v); a later sign-in must "+
			"refresh what the IdP now asserts about this identity", profile)
	}
}

// TestEnsureAutoMembership_GrantsPlainMembershipAndIsIdempotent covers the two
// things an auto-join must never get wrong.
//
// First, the role: auto-join is a convenience for someone who has proved an
// address at a domain the tenant has proved it owns. That earns "member", not
// control of the organization. A grant of owner or admin here would hand
// anybody who can receive mail at a verified domain the ability to change
// billing, invite users and delete estate.
//
// Second, repetition: EnsureAutoMembership runs on every social sign-in. If the
// existing-membership check stopped short-circuiting, each sign-in would either
// duplicate the row or — worse, for a founder who is the org's owner — write a
// second, lower-privileged row and leave authorisation reading whichever one it
// happens to load first.
func TestEnsureAutoMembership_GrantsPlainMembershipAndIsIdempotent(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "autorole")
	org := mustCreateOrg(t, db, domain, true)

	email := uuid.NewString() + "@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	if err := r.EnsureAutoMembership(ctx, user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	m := rsaMembership(t, db, org.ID, user.ID)
	if m == nil {
		t.Fatalf("no membership was granted for a verified address at a proven domain")
	}
	if m.Role != control.MemberRoleMember {
		t.Fatalf("auto-join granted role %q; anyone who can receive mail at a verified "+
			"domain would gain %q over the tenant. It must grant %q and nothing more",
			m.Role, m.Role, control.MemberRoleMember)
	}

	// A second sign-in must be a no-op.
	if err := r.EnsureAutoMembership(ctx, user.ID, email); err != nil {
		t.Fatalf("second EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 1 {
		t.Fatalf("a repeat sign-in produced %d membership rows; auto-join runs on every "+
			"login and must be idempotent", n)
	}

	// And it must never rewrite a role someone already holds.
	owner := mustCreateUser(t, db, uuid.NewString()+"@"+domain)
	mustCreateAccount(t, db, owner.ID, "google", "s-"+uuid.NewString(), owner.Email, true)
	if err := db.Create(&control.Membership{
		UserID:         owner.ID,
		OrganizationID: org.ID,
		Role:           control.MemberRoleOwner,
	}).Error; err != nil {
		t.Fatalf("seed owner membership: %v", err)
	}

	if err := r.EnsureAutoMembership(ctx, owner.ID, owner.Email); err != nil {
		t.Fatalf("EnsureAutoMembership for an existing owner: %v", err)
	}
	if got := rsaMembership(t, db, org.ID, owner.ID); got == nil || got.Role != control.MemberRoleOwner {
		t.Fatalf("the org owner's role after an auto-join pass is %v, want %q; a login "+
			"must not demote the person who owns the tenant", got, control.MemberRoleOwner)
	}
	if n := membershipCount(t, db, org.ID, owner.ID); n != 1 {
		t.Fatalf("the owner now holds %d membership rows", n)
	}
}

// TestEnsureAutoMembership_RequiresTheVerifiedAccountToCarryTheSameAddress
// pins the LOWER(email) = ? half of the verified-account check.
//
// The whole grant rests on one claim: THIS user has proved control of an
// address at the organization's verified domain. The evidence is an account row
// that is both email_verified and for that address. Counting only
// "email_verified = TRUE" would accept any verified provider identity the user
// holds — a personal Gmail, say — as proof of an address at acme.example that
// they merely typed in. That is unauthenticated entry into someone else's
// tenant, which is the most valuable thing this function can give away.
func TestEnsureAutoMembership_RequiresTheVerifiedAccountToCarryTheSameAddress(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "targettenant")
	org := mustCreateOrg(t, db, domain, true)

	// The user's only verified identity is at an unrelated domain.
	otherDomain := uniqueDomain(t, "personal")
	ownEmail := uuid.NewString() + "@" + otherDomain
	user := mustCreateUser(t, db, ownEmail)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), ownEmail, true)

	claimed := uuid.NewString() + "@" + domain
	if err := r.EnsureAutoMembership(ctx, user.ID, claimed); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("a user whose only verified identity is at %s was auto-joined into the "+
			"tenant that owns %s (%d rows) — an unproven address must never buy entry "+
			"into someone else's organization", otherDomain, domain, n)
	}

	// Case is the other half of the same predicate: the account row is compared
	// with LOWER(), so a provider that echoes the address back capitalised must
	// still count as proof.
	upperEmail := uuid.NewString() + "@" + domain
	upperUser := mustCreateUser(t, db, strings.ToLower(upperEmail))
	mustCreateAccount(t, db, upperUser.ID, "google", "s-"+uuid.NewString(), strings.ToUpper(upperEmail), true)

	if err := r.EnsureAutoMembership(ctx, upperUser.ID, upperEmail); err != nil {
		t.Fatalf("EnsureAutoMembership (mixed case): %v", err)
	}
	if n := membershipCount(t, db, org.ID, upperUser.ID); n != 1 {
		t.Fatalf("a verified account stored in upper case was not recognised as proof of "+
			"the same address (%d membership rows, want 1)", n)
	}
}

// TestEvaluateAccessRules_DatabaseErrorIsNeverAnAllow is the single most
// important line in the resolver: an error from the lookup returns the error,
// while only gorm.ErrRecordNotFound means "no rule matched".
//
// If those two were collapsed — a naive `if err != nil { return nil }`, or a
// caller that treats any non-ErrAccessDenied result as permission — then a
// database blip, a cancelled request or a broken transaction would read as
// "policy says yes". The failure mode is that every deny in the table stops
// applying at exactly the moment the system is least healthy, which is also
// when a bulk-registration attempt is most likely to succeed.
func TestEvaluateAccessRules_DatabaseErrorIsNeverAnAllow(t *testing.T) {
	db := accessRuleTestDB(t)

	id, err := NormalizeEmailIdentity("a@" + uniqueDomain(t, "dberr"))
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}

	t.Run("dead transaction", func(t *testing.T) {
		tx := db.Begin()
		if tx.Error != nil {
			t.Fatalf("begin: %v", tx.Error)
		}
		if err := tx.Rollback().Error; err != nil {
			t.Fatalf("rollback: %v", err)
		}

		err := EvaluateAccessRules(context.Background(), tx, control.AccessRulePurposeCreateOrg, id)
		if err == nil {
			t.Fatalf("a query against a finished transaction was reported as \"no rule " +
				"matched, therefore allow\"; every deny in auth_access_rules would stop " +
				"applying the moment the database misbehaves")
		}
		if !errors.Is(err, sql.ErrTxDone) {
			t.Fatalf("want the underlying %v surfaced to the caller, got %v", sql.ErrTxDone, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("a transport error was reported as ErrRecordNotFound, which the " +
				"resolver treats as permission")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := EvaluateAccessRules(ctx, db, control.AccessRulePurposeAuthenticate, id)
		if err == nil {
			t.Fatalf("a lookup on a cancelled context was reported as allow; an aborted " +
				"request must never be the reason a ban-list entry stops applying")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled surfaced, got %v", err)
		}
	})
}

// TestEvaluateAccessRules_PrecedenceAndPurposeMatrix drives the resolver as the
// decision matrix it is.
//
// Three separate properties are entangled in one SQL statement — the purpose
// predicate, the email-before-domain ordering and the longest-value ordering —
// and each of them is a security decision:
//
//   - The purpose predicate is what keeps the three questions apart. A rule
//     written to stop a domain minting tenants must not decide a login, and a
//     join_org rule must not decide either of the other two.
//   - Email beats domain, in BOTH directions. The existing suite covers an
//     email allow overriding a domain deny (the operator's exception). This one
//     covers the mirror an operator relies on just as much: a single abusive
//     address denied inside a domain that is otherwise allowed. If ordering
//     collapsed, that deny would be silently outvoted by the surrounding allow.
func TestEvaluateAccessRules_PrecedenceAndPurposeMatrix(t *testing.T) {
	db := accessRuleTestDB(t)
	ctx := context.Background()

	domain := uniqueDomain(t, "matrix")
	abusive := "abuser@" + domain
	ordinary := "ordinary@" + domain

	// An allowed domain with one denied address inside it, per purpose.
	for _, purpose := range []string{
		control.AccessRulePurposeAuthenticate,
		control.AccessRulePurposeCreateOrg,
	} {
		mustRule(t, db, control.AccessRuleKindDomain, domain, purpose, control.AccessRulePolicyAllow)
		mustRule(t, db, control.AccessRuleKindEmail, abusive, purpose, control.AccessRulePolicyDeny)
	}
	// join_org is a third, independent purpose. Nothing above may decide it, and
	// its own deny may not leak into the other two.
	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeJoinOrg, control.AccessRulePolicyDeny)

	// A denied registrable domain carrying one carved-out subdomain, arranged so
	// that the exception sorts AFTER the deny alphabetically ("zzz.<apex>" vs
	// "<apex>"). Precedence here depends solely on ORDER BY LENGTH(value) DESC;
	// with the same fixture written the other way round, whichever row Postgres
	// happened to return first would be the "right" answer and the ordering
	// clause could be deleted without any test noticing.
	apex := uniqueDomain(t, "specificity")
	carveOut := "zzz." + apex
	mustRule(t, db, control.AccessRuleKindDomain, apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)
	mustRule(t, db, control.AccessRuleKindDomain, carveOut, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyAllow)

	cases := []struct {
		name    string
		email   string
		purpose string
		denied  bool
		why     string
	}{
		{
			name: "email deny beats domain allow (authenticate)", email: abusive,
			purpose: control.AccessRulePurposeAuthenticate, denied: true,
			why: "a single banned address inside an allowed domain was outvoted by the domain rule",
		},
		{
			name: "email deny beats domain allow (create_org)", email: abusive,
			purpose: control.AccessRulePurposeCreateOrg, denied: true,
			why: "the per-address deny lost to the domain allow when minting a tenant",
		},
		{
			name: "plus-addressing does not evade the email deny", email: "abuser+anything@" + domain,
			purpose: control.AccessRulePurposeAuthenticate, denied: true,
			why: "one '+' character was enough to escape an exact-address ban",
		},
		{
			name: "the rest of the domain is still allowed", email: ordinary,
			purpose: control.AccessRulePurposeAuthenticate, denied: false,
			why: "a deny written against one address was applied to the whole domain",
		},
		{
			name: "join_org deny does not decide authentication", email: ordinary,
			purpose: control.AccessRulePurposeAuthenticate, denied: false,
			why: "a join_org rule blocked a login — an invited user could not reach the tenant they were invited to",
		},
		{
			name: "join_org deny does not decide org creation", email: ordinary,
			purpose: control.AccessRulePurposeCreateOrg, denied: false,
			why: "a join_org rule was read as a create_org decision",
		},
		{
			name: "join_org is decided by its own rule", email: ordinary,
			purpose: control.AccessRulePurposeJoinOrg, denied: true,
			why: "the join_org deny did not fire for its own purpose",
		},
		{
			name: "the authenticate allow does not leak into join_org", email: abusive,
			purpose: control.AccessRulePurposeJoinOrg, denied: true,
			why: "an allow written for another purpose granted a join the operator denied",
		},
		{
			name: "the narrower subdomain allow beats the registrable-domain deny", email: "a@" + carveOut,
			purpose: control.AccessRulePurposeCreateOrg, denied: false,
			why: "an operator's carve-out for one host lost to the deny on its registrable domain; " +
				"rules stop meaning what they say as soon as the most-specific-first ordering is not enforced",
		},
		{
			name: "the registrable-domain deny still covers every other host", email: "a@other." + apex,
			purpose: control.AccessRulePurposeCreateOrg, denied: true,
			why: "a carve-out for one subdomain widened into an allow for the whole registrable domain",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := NormalizeEmailIdentity(tc.email)
			if err != nil {
				t.Fatalf("normalise %q: %v", tc.email, err)
			}
			err = EvaluateAccessRules(ctx, db, tc.purpose, id)
			denied := errors.Is(err, ErrAccessDenied)
			if err != nil && !denied {
				t.Fatalf("EvaluateAccessRules(%s, %s) returned an unexpected error: %v",
					tc.purpose, tc.email, err)
			}
			if denied != tc.denied {
				t.Fatalf("EvaluateAccessRules(%s, %s): denied=%v, want %v — %s",
					tc.purpose, tc.email, denied, tc.denied, tc.why)
			}
		})
	}
}

// TestUserRepoUpdateProfile_NormalisesAndHashes covers the write path every
// profile edit, email change confirmation and password reset funnels through.
//
// The email is lower-cased and trimmed for the same reason UpsertFromIdentity
// does it: an address stored with stray case is invisible to every rule lookup
// and uniqueness check in the system, which compare lower-cased values. The
// password must be stored as a bcrypt hash — the field is written from a map,
// so a regression that assigns *in.Password directly still compiles, still
// passes any "the update succeeded" assertion, and silently puts plaintext
// credentials in the users table.
func TestUserRepoUpdateProfile_NormalisesAndHashes(t *testing.T) {
	db := socialTestDB(t)
	rsaUsers := NewUserRepoGorm(db)
	ctx := context.Background()

	user := mustCreateUser(t, db, uniqueEmail(t, "profile"))

	newEmail := uniqueEmail(t, "renamed")
	plaintext := "correct-horse-battery"
	mixed := strings.ToUpper(newEmail)
	first := "  Ada  "
	last := "\tLovelace\n"

	if err := rsaUsers.UpdateProfile(ctx, user.ID, UpdateUserProfileInput{
		Email:     &mixed,
		FirstName: &first,
		LastName:  &last,
		Password:  &plaintext,
	}); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}

	var got control.User
	if err := db.First(&got, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Email != newEmail {
		t.Fatalf("email stored as %q, want the trimmed lower-case %q; access rules, the "+
			"uniqueness index and every conflict check key on the lower-cased address",
			got.Email, newEmail)
	}
	if got.FirstName != "Ada" || got.LastName != "Lovelace" {
		t.Fatalf("names stored as %q/%q, want %q/%q — untrimmed input reaches display "+
			"names and invitation lookups verbatim", got.FirstName, got.LastName, "Ada", "Lovelace")
	}
	if got.Password == plaintext {
		t.Fatalf("the password was stored in plaintext")
	}
	if !got.CheckPassword(plaintext) {
		t.Fatalf("the stored password does not verify against the one that was set; the "+
			"user has just been locked out of their own account (stored: %q)", got.Password)
	}

	// An input that asks for nothing must issue no statement at all — that is
	// what the `len(updates) == 0` early return is for, and updated_at is the
	// only thing that proves it. GORM does NOT reject an Updates() with an empty
	// map: ConvertToAssignments still appends the autoUpdateTime column, so
	// dropping the guard silently emits `UPDATE users SET updated_at = now()`.
	// That returns no error and leaves every asserted column alone, so a test
	// that only compares email/name/password stays green while every no-op call
	// takes a row lock and bumps the timestamp other code reads. Asserting on
	// updated_at is what makes the guard's removal observable.
	before := got
	if err := rsaUsers.UpdateProfile(ctx, user.ID, UpdateUserProfileInput{}); err != nil {
		t.Fatalf("UpdateProfile with no fields returned %v; asking for no change is not a failure", err)
	}
	var after control.User
	if err := db.First(&after, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.Email != before.Email || after.FirstName != before.FirstName ||
		after.LastName != before.LastName || after.Password != before.Password {
		t.Fatalf("an empty UpdateUserProfileInput overwrote stored fields: %+v -> %+v", before, after)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("an empty UpdateUserProfileInput still issued an UPDATE: updated_at moved "+
			"from %v to %v. A request that asks for no change must not write, lock the row, "+
			"or advance the timestamp", before.UpdatedAt, after.UpdatedAt)
	}
}

// TestUserRepoUpdateProfile_RejectsAWeakPasswordWithoutWritingAnything pins the
// ordering inside UpdateProfile: the password is hashed BEFORE the UPDATE is
// issued, so a rejected password aborts the whole statement.
//
// ChangePassword and the password-reset flow both call this with the password
// alone, but ConfirmEmailChange calls it with an email — and a future caller
// combining the two is the case that matters. If the hash were computed after
// the write, a refused password would still have committed the other columns,
// and the caller's error message ("password too short") would be a lie about
// what happened to the account.
func TestUserRepoUpdateProfile_RejectsAWeakPasswordWithoutWritingAnything(t *testing.T) {
	db := socialTestDB(t)
	rsaUsers := NewUserRepoGorm(db)
	ctx := context.Background()

	originalEmail := uniqueEmail(t, "weakpw")
	user := mustCreateUser(t, db, originalEmail)

	attempted := uniqueEmail(t, "shouldnotland")
	short := "short"
	if err := rsaUsers.UpdateProfile(ctx, user.ID, UpdateUserProfileInput{
		Email:    &attempted,
		Password: &short,
	}); err == nil {
		t.Fatalf("a %d-character password was accepted", len(short))
	}

	var got control.User
	if err := db.First(&got, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Email != originalEmail {
		t.Fatalf("the email was changed to %q even though the update was refused; a "+
			"caller told \"password too short\" would have no idea their address had moved",
			got.Email)
	}

	var landed int64
	if err := db.Model(&control.User{}).Where("LOWER(email) = ?", attempted).Count(&landed).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if landed != 0 {
		t.Fatalf("the refused update still wrote %q into the users table", attempted)
	}
}

// TestUserRepoMarkEmailVerified_FirstStampWins covers the WHERE
// email_verified_at IS NULL guard.
//
// Same evidence argument as the social path: the stamp says when the address
// was first proved, and the admission ceiling reads it to decide how much an
// account may do. Without the guard, anyone holding a still-valid verification
// link — or simply clicking the same link twice — could push the timestamp
// forward whenever they liked, and "verified three weeks ago" would mean
// nothing. The second call must still report success: confirming an already
// confirmed address is the state the caller asked for, and returning an error
// shows a scary page to someone who double-clicked.
func TestUserRepoMarkEmailVerified_FirstStampWins(t *testing.T) {
	db := socialTestDB(t)
	rsaUsers := NewUserRepoGorm(db)
	ctx := context.Background()

	user := mustCreateUser(t, db, uniqueEmail(t, "verify"))
	if rsaStamp(t, db, user.ID) != nil {
		t.Fatalf("a freshly created password user was already stamped as verified")
	}

	firstAt := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Microsecond)
	if err := rsaUsers.MarkEmailVerified(ctx, user.ID, firstAt); err != nil {
		t.Fatalf("first MarkEmailVerified: %v", err)
	}
	stored := rsaStamp(t, db, user.ID)
	if stored == nil || !stored.UTC().Equal(firstAt) {
		t.Fatalf("stored stamp %v, want %v", stored, firstAt)
	}

	laterAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := rsaUsers.MarkEmailVerified(ctx, user.ID, laterAt); err != nil {
		t.Fatalf("re-confirming an already confirmed address returned %v; clicking the "+
			"link twice is not an error and must not surface as one", err)
	}
	again := rsaStamp(t, db, user.ID)
	if again == nil || !again.UTC().Equal(firstAt) {
		t.Fatalf("a replayed confirmation moved email_verified_at from %v to %v; the "+
			"timestamp is evidence of how long the address has been proven, and anything "+
			"that reasons about its age can then be reset on demand", firstAt, again)
	}

	// The guard is scoped to the user, not global: a second, still-unverified
	// user must still be stampable.
	other := mustCreateUser(t, db, uniqueEmail(t, "verify2"))
	if err := rsaUsers.MarkEmailVerified(ctx, other.ID, laterAt); err != nil {
		t.Fatalf("MarkEmailVerified for a different user: %v", err)
	}
	if s := rsaStamp(t, db, other.ID); s == nil || !s.UTC().Equal(laterAt) {
		t.Fatalf("a second user's confirmation was suppressed (stamp %v, want %v)", s, laterAt)
	}
}

// TestAuthAccessRuleRepoCreate_NormalisesValuesSoTheRuleActuallyMatches covers
// the half of Create that validates Value.
//
// The resolver binds a lower-cased, plus-stripped identity, so a rule stored as
// "EvilCorp.Example" or "abuser+spam@x.test" can never match anything. The
// dangerous part is that such a rule is not visibly broken: it is accepted, it
// appears in List() as active policy, and the operator believes the domain is
// blocked while it goes on minting tenants. A deny that silently enforces
// nothing is worse than no deny at all, because nobody goes looking for it.
//
// The assertions therefore do not stop at "the row was stored lower-cased" —
// each rule is put to the resolver afterwards, because storing a normalised
// value is only interesting if it makes the decision it was written for.
func TestAuthAccessRuleRepoCreate_NormalisesValuesSoTheRuleActuallyMatches(t *testing.T) {
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	base := uniqueDomain(t, "createnorm")

	domainRule := &control.AuthAccessRule{
		Kind:    control.AccessRuleKindDomain,
		Value:   "  " + strings.ToUpper(base) + "  ",
		Purpose: control.AccessRulePurposeCreateOrg,
		Policy:  control.AccessRulePolicyDeny,
	}
	if err := r.Create(ctx, domainRule); err != nil {
		t.Fatalf("Create domain rule: %v", err)
	}
	t.Cleanup(func() { db.Delete(&control.AuthAccessRule{}, "id = ?", domainRule.ID) })

	if domainRule.Value != base {
		t.Fatalf("domain rule stored as %q, want the trimmed lower-case %q", domainRule.Value, base)
	}
	if err := r.CheckOrgCreation(ctx, "someone@"+base); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a deny an operator wrote against %s did not stop org creation (%v); the "+
			"rule is listed as active policy while enforcing nothing", base, err)
	}

	emailRule := &control.AuthAccessRule{
		Kind:    control.AccessRuleKindEmail,
		Value:   " Abuser+Spam@" + strings.ToUpper(base) + " ",
		Purpose: control.AccessRulePurposeAuthenticate,
		Policy:  control.AccessRulePolicyDeny,
	}
	if err := r.Create(ctx, emailRule); err != nil {
		t.Fatalf("Create email rule: %v", err)
	}
	t.Cleanup(func() { db.Delete(&control.AuthAccessRule{}, "id = ?", emailRule.ID) })

	wantEmail := "abuser@" + base
	if emailRule.Value != wantEmail {
		t.Fatalf("email rule stored as %q, want %q — the resolver strips plus-addressing "+
			"and lower-cases before comparing, so anything else can never match",
			emailRule.Value, wantEmail)
	}
	for _, banned := range []string{wantEmail, "ABUSER@" + base, "abuser+anything@" + base} {
		if err := r.CheckAuthentication(ctx, banned); !errors.Is(err, ErrAccessDenied) {
			t.Fatalf("the ban on %s did not stop %s from authenticating (%v)", wantEmail, banned, err)
		}
	}

	// Values that cannot be keyed must be refused outright rather than stored as
	// policy that will never fire.
	unusable := []struct {
		kind  string
		value string
	}{
		{control.AccessRuleKindEmail, "not-an-email"},
		{control.AccessRuleKindEmail, "@" + base},
		{control.AccessRuleKindEmail, ""},
		{control.AccessRuleKindDomain, ""},
		{control.AccessRuleKindDomain, "   "},
		{control.AccessRuleKindDomain, "user@" + base},
		{control.AccessRuleKindDomain, "two words.test"},
	}
	for _, tc := range unusable {
		rule := &control.AuthAccessRule{
			Kind:    tc.kind,
			Value:   tc.value,
			Purpose: control.AccessRulePurposeCreateOrg,
			Policy:  control.AccessRulePolicyDeny,
		}
		err := r.Create(ctx, rule)
		if !errors.Is(err, ErrInvalidRuleValue) {
			db.Delete(&control.AuthAccessRule{}, "id = ?", rule.ID)
			t.Errorf("Create(kind=%s, value=%q) = %v, want ErrInvalidRuleValue — an "+
				"unmatchable rule accepted here reads as enforced policy forever after",
				tc.kind, tc.value, err)
		}
	}

	if err := r.Create(ctx, nil); err == nil {
		t.Fatalf("Create(nil) was accepted")
	}
}
