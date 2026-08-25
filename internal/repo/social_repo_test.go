package repo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var socialTestSchema = testsupport.SchemaName("apiary_repo_social_test")

func socialTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testsupport.OpenSchema(t, socialTestSchema,
		&control.User{},
		&control.Organization{},
		&control.Account{},
		&control.Membership{},
		&control.AuthAccessRule{},
	)

	// EnsureAutoMembership used to consult a hardcoded Go map of public mailbox
	// domains. That map is gone; without these rows the public-domain refusal
	// would pass for the wrong reason — no rules, therefore allow.
	if _, err := SeedPublicEmailDomainRules(context.Background(), db); err != nil {
		t.Fatalf("seed public email domain rules: %v", err)
	}
	return db
}

func uniqueEmail(t *testing.T, local string) string {
	t.Helper()
	return local + "-" + strings.ReplaceAll(uuid.NewString(), "-", "") + "@example.test"
}

func mustCreateUser(t *testing.T, db *gorm.DB, email string) control.User {
	t.Helper()
	u := control.User{
		Email:     email,
		Password:  "x",
		FirstName: "T",
		LastName:  "U",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	t.Cleanup(func() {
		db.Where("user_id = ?", u.ID).Delete(&control.Membership{})
		db.Where("user_id = ?", u.ID).Delete(&control.Account{})
		db.Delete(&control.User{}, "id = ?", u.ID)
	})
	return u
}

func mustCreateOrg(t *testing.T, db *gorm.DB, domain string, verified bool) control.Organization {
	t.Helper()
	org := control.Organization{
		Name:   "org-" + domain,
		Domain: &domain,
		Status: "active",
	}
	if verified {
		at := time.Now().UTC()
		org.DomainVerifiedAt = &at
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create org for %s: %v", domain, err)
	}
	t.Cleanup(func() {
		db.Where("organization_id = ?", org.ID).Delete(&control.Membership{})
		db.Delete(&control.Organization{}, "id = ?", org.ID)
	})
	return org
}

func mustCreateAccount(t *testing.T, db *gorm.DB, userID uuid.UUID, provider, subject, email string, verified bool) control.Account {
	t.Helper()
	acct := control.Account{
		UserID:        userID,
		Provider:      provider,
		Subject:       subject,
		Email:         &email,
		EmailVerified: verified,
		Profile:       []byte(`{}`),
	}
	if err := db.Create(&acct).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	t.Cleanup(func() { db.Delete(&control.Account{}, "id = ?", acct.ID) })
	return acct
}

func newSocialRepo(db *gorm.DB) SocialAuthRepo { return NewSocialAuthRepoGorm(db) }

func TestUpsertFromIdentity_RefusesSilentMergeByEmail(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	email := uniqueEmail(t, "victim")
	victim := mustCreateUser(t, db, email)

	subject := "attacker-" + uuid.NewString()
	got, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       subject,
		Email:         email,
		EmailVerified: true,
	})
	if !errors.Is(err, ErrSocialLinkNotAllowed) {
		t.Fatalf("want ErrSocialLinkNotAllowed, got user=%v err=%v", got, err)
	}

	var accounts int64
	if err := db.Model(&control.Account{}).Where("subject = ?", subject).Count(&accounts).Error; err != nil {
		t.Fatalf("count accounts: %v", err)
	}
	if accounts != 0 {
		t.Fatalf("a provider account was linked to the victim despite the refusal")
	}
	var reloaded control.User
	if err := db.First(&reloaded, "id = ?", victim.ID).Error; err != nil {
		t.Fatalf("reload victim: %v", err)
	}
	if reloaded.Email != email {
		t.Fatalf("victim's email changed to %q", reloaded.Email)
	}
}

func TestUpsertFromIdentity_MergesWhenOperatorOptedIn(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	email := uniqueEmail(t, "existing")
	existing := mustCreateUser(t, db, email)

	got, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       "opted-in-" + uuid.NewString(),
		Email:         email,
		EmailVerified: true,
		LinkByEmail:   true,
	})
	if err != nil {
		t.Fatalf("UpsertFromIdentity with LinkByEmail: %v", err)
	}
	if got.ID != existing.ID {
		t.Fatalf("want the existing user %s, got %s", existing.ID, got.ID)
	}
}

func TestUpsertFromIdentity_RefusesUnverifiedEmail(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	email := uniqueEmail(t, "unverified")
	subject := "unverified-" + uuid.NewString()

	_, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       subject,
		Email:         email,
		EmailVerified: false,
	})
	if !errors.Is(err, ErrIdentityEmailNotVerified) {
		t.Fatalf("want ErrIdentityEmailNotVerified, got %v", err)
	}

	var users int64
	if err := db.Model(&control.User{}).Where("LOWER(email) = ?", email).Count(&users).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("a user row was created for an address the provider never verified")
	}
}

func TestUpsertFromIdentity_RefusesEmptyEmail(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	_, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
		Provider:      "github",
		Subject:       "no-email-" + uuid.NewString(),
		EmailVerified: true,
	})
	if !errors.Is(err, ErrIdentityEmailNotVerified) {
		t.Fatalf("want ErrIdentityEmailNotVerified, got %v", err)
	}
}

func TestUpsertFromIdentity_RefusesProviderEmailChangeOntoAnotherUser(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	ownEmail := uniqueEmail(t, "linked")
	victimEmail := uniqueEmail(t, "target")
	linked := mustCreateUser(t, db, ownEmail)
	mustCreateUser(t, db, victimEmail)

	subject := "linked-" + uuid.NewString()
	mustCreateAccount(t, db, linked.ID, "google", subject, ownEmail, true)

	for _, optIn := range []bool{false, true} {
		_, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
			Provider:      "google",
			Subject:       subject,
			Email:         victimEmail,
			EmailVerified: true,
			LinkByEmail:   optIn,
		})
		if !errors.Is(err, ErrSocialLinkNotAllowed) {
			t.Fatalf("LinkByEmail=%v: want ErrSocialLinkNotAllowed, got %v", optIn, err)
		}
	}

	var reloaded control.User
	if err := db.First(&reloaded, "id = ?", linked.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Email != ownEmail {
		t.Fatalf("the linked user's email was re-pointed to %q", reloaded.Email)
	}
}

func TestUpsertFromIdentity_CreatesUserForNewIdentity(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	email := uniqueEmail(t, "newcomer")
	subject := "newcomer-" + uuid.NewString()

	got, err := r.UpsertFromIdentity(context.Background(), UpsertSocialIdentityInput{
		Provider:      "google",
		Subject:       subject,
		Email:         email,
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("UpsertFromIdentity: %v", err)
	}
	t.Cleanup(func() {
		db.Where("user_id = ?", got.ID).Delete(&control.Account{})
		db.Delete(&control.User{}, "id = ?", got.ID)
	})
	if got.Email != email {
		t.Fatalf("want email %q, got %q", email, got.Email)
	}

	var acct control.Account
	if err := db.First(&acct, "provider = ? AND subject = ?", "google", subject).Error; err != nil {
		t.Fatalf("account not created: %v", err)
	}
	if acct.UserID != got.ID {
		t.Fatalf("account linked to %s, want %s", acct.UserID, got.ID)
	}
}

func membershipCount(t *testing.T, db *gorm.DB, orgID, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&control.Membership{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Count(&n).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	return n
}

func TestEnsureAutoMembership_RefusesUnverifiedDomain(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	domain := "claimed-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
	org := mustCreateOrg(t, db, domain, false)

	email := "person@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	if err := r.EnsureAutoMembership(context.Background(), user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("membership of a tenant granted on an unproven domain claim (%d rows)", n)
	}
}

func TestEnsureAutoMembership_GrantsOnVerifiedDomain(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	domain := "proven-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
	org := mustCreateOrg(t, db, domain, true)

	email := "person@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	if err := r.EnsureAutoMembership(context.Background(), user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 1 {
		t.Fatalf("want 1 membership on a proven domain, got %d", n)
	}
}

func TestEnsureAutoMembership_RefusesWhenUserNeverVerifiedTheAddress(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	domain := "proven-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
	org := mustCreateOrg(t, db, domain, true)

	email := "person@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, false)

	if err := r.EnsureAutoMembership(context.Background(), user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("membership granted on an address the user never verified (%d rows)", n)
	}
}

func TestEnsureAutoMembership_RefusesPublicMailDomain(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	org := mustCreateOrg(t, db, "gmail.com", true)

	email := uuid.NewString() + "@gmail.com"
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	if err := r.EnsureAutoMembership(context.Background(), user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("auto-join fired for a consumer mailbox domain (%d rows)", n)
	}
}

func TestEnsureAutoMembership_RefusesAmbiguousDomain(t *testing.T) {
	db := socialTestDB(t)
	r := newSocialRepo(db)

	domain := "contested-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
	orgA := mustCreateOrg(t, db, domain, true)
	orgB := mustCreateOrg(t, db, domain, true)

	email := "person@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	if err := r.EnsureAutoMembership(context.Background(), user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, orgA.ID, user.ID) + membershipCount(t, db, orgB.ID, user.ID); n != 0 {
		t.Fatalf("membership granted despite two organizations claiming the domain (%d rows)", n)
	}
}
