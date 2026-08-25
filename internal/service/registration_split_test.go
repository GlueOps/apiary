package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// newVerificationService builds the real thing against the test schema, with a
// counting enqueuer and no mailer. Issue tolerates a nil EmailService (it logs
// and skips the send), which is what keeps these tests off SMTP without
// introducing a fake whose behaviour could drift from the real sender.
func newVerificationService(db *gorm.DB, jobs *countingEnqueuer) *EmailVerificationService {
	users := repo.NewUserRepoGorm(db)
	return NewEmailVerificationService(
		repo.NewEmailVerificationRepoGorm(db),
		&users,
		repo.NewMembershipRepoGorm(db),
		nil,
		jobs,
		nil,
		"https://apiary.test",
	)
}

// seedPendingRegistration creates the rows a registration leaves behind, plus a
// verification token whose RAW value the test knows.
//
// The raw token is planted rather than captured from Issue because Issue does
// not return it and must not start doing so: the only copy that exists outside
// the mailer is the one in the email, and a method that hands it back is one
// careless handler away from putting it in a response body.
func seedPendingRegistration(t *testing.T, db *gorm.DB, email, rawToken string, role control.MemberRole) (*control.User, *control.Organization) {
	t.Helper()

	user := &control.User{Email: email, FirstName: "Test", LastName: "User", Role: control.RolePlatformCustomer, IsActive: true}
	if err := user.SetPassword("correct-horse-battery"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	org := &control.Organization{Name: "Test Org"}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	if err := db.Create(&control.Membership{UserID: user.ID, OrganizationID: org.ID, Role: role}).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}

	sum := sha256.Sum256([]byte(rawToken))
	if err := db.Create(&control.EmailVerificationToken{
		UserID:    user.ID,
		Email:     email,
		TokenHash: hex.EncodeToString(sum[:]),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	return user, org
}

// This is the whole point of the register/confirm split. Registration used to
// sign the caller in and enqueue a CREATE DATABASE before anyone had read a
// single email — which handed an attacker both a real Postgres database and,
// through the session it minted, day one of the proving that retires the
// admission ceiling. Both must now cost a confirmed mailbox.
func TestRegistrationMintsNoSessionAndNoTenantDatabase(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{}
	gate := NewAdmissionService(rules, testLimits())

	users := repo.NewUserRepoGorm(db)
	svc := NewRegistrationService(db, &users, nil, rules, gate, newVerificationService(db, jobs))

	result, err := svc.Register(t.Context(), registerInput("founder@example.com"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if result.OrganizationID == "" {
		t.Fatal("registration returned no organization id")
	}

	if n := countRows(t, db, &control.TokenSession{}); n != 0 {
		t.Fatalf("token_sessions = %d, want 0 — registration signed the caller in, which "+
			"hands an unconfirmed address its first day of organization proving for free", n)
	}
	if n := jobs.provisions.Load(); n != 0 {
		t.Fatalf("provisioning jobs = %d, want 0 — an unconfirmed registration asked for a "+
			"Postgres database, which is the resource this split exists to withhold", n)
	}
	if n := countRows(t, db, &control.EmailVerificationToken{}); n != 1 {
		t.Fatalf("email_verification_tokens = %d, want 1 — registration must leave a way to confirm", n)
	}

	var stored control.User
	if err := db.First(&stored, "email = ?", "founder@example.com").Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if stored.EmailVerified() {
		t.Fatal("a freshly registered user is already marked verified")
	}
	// The organization exists so that it occupies a slot against the domain
	// ceiling immediately; that is deliberate and separate from provisioning.
	if n := countRows(t, db, &control.Organization{}); n != 1 {
		t.Fatalf("organizations = %d, want 1 — the ceiling counts pending organizations, so the row must exist", n)
	}
}

func TestConfirmingReleasesProvisioningExactlyOnce(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	svc := newVerificationService(db, jobs)

	const raw = "a-known-raw-token"
	user, _ := seedPendingRegistration(t, db, "owner@example.com", raw, control.MemberRoleOwner)

	result, err := svc.Confirm(t.Context(), raw)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if result.AlreadyVerified {
		t.Fatal("first confirmation reported the address as already verified")
	}
	if !result.TenantProvisioning {
		t.Fatal("confirming did not report that provisioning was released")
	}
	if n := jobs.provisions.Load(); n != 1 {
		t.Fatalf("provisioning jobs = %d, want 1", n)
	}

	var stored control.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !stored.EmailVerified() {
		t.Fatal("confirming did not stamp email_verified_at")
	}

	// Replay. A confirmation link lands in a mailbox and gets clicked twice, is
	// prefetched by a scanner, or is replayed deliberately; none of those may
	// enqueue a second CREATE DATABASE for the same organization.
	if _, err := svc.Confirm(t.Context(), raw); !errors.Is(err, ErrVerificationTokenInvalid) {
		t.Fatalf("replayed token: err = %v, want ErrVerificationTokenInvalid", err)
	}
	if n := jobs.provisions.Load(); n != 1 {
		t.Fatalf("provisioning jobs after replay = %d, want 1 — the token was redeemed twice", n)
	}
}

// A token proves control of the address it was ISSUED for. If the account's
// address changes in between, redeeming it must not launder that proof onto the
// new one.
func TestConfirmRefusesATokenIssuedForADifferentAddress(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	svc := newVerificationService(db, jobs)

	const raw = "token-for-the-old-address"
	user, _ := seedPendingRegistration(t, db, "old@example.com", raw, control.MemberRoleOwner)

	if err := db.Model(&control.User{}).Where("id = ?", user.ID).
		Update("email", "attacker@example.com").Error; err != nil {
		t.Fatalf("change email: %v", err)
	}

	if _, err := svc.Confirm(t.Context(), raw); !errors.Is(err, ErrVerificationTokenInvalid) {
		t.Fatalf("err = %v, want ErrVerificationTokenInvalid — a token issued for one "+
			"address confirmed a different one", err)
	}
	if n := jobs.provisions.Load(); n != 0 {
		t.Fatalf("provisioning jobs = %d, want 0", n)
	}
}

// Only the owner's confirmation releases provisioning. A member invited into
// someone else's organization confirming their own address says nothing about
// whether that organization should have a database.
func TestConfirmingAsAPlainMemberDoesNotProvision(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	svc := newVerificationService(db, jobs)

	const raw = "member-token"
	seedPendingRegistration(t, db, "member@example.com", raw, control.MemberRoleMember)

	result, err := svc.Confirm(t.Context(), raw)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if result.TenantProvisioning {
		t.Fatal("a member's confirmation reported provisioning")
	}
	if n := jobs.provisions.Load(); n != 0 {
		t.Fatalf("provisioning jobs = %d, want 0 — a non-owner's confirmation built a tenant", n)
	}
}

// Resend is unauthenticated and takes an arbitrary address, so every outcome
// must be indistinguishable from outside. The assertion that carries the
// security property is that no token is minted for an address that has none.
func TestResendRevealsNothingAboutAnUnknownAddress(t *testing.T) {
	db := registrationTestDB(t)
	svc := newVerificationService(db, &countingEnqueuer{})

	if err := svc.Resend(t.Context(), "nobody@example.com"); err != nil {
		t.Fatalf("resend for an unknown address returned %v, want nil — the error itself "+
			"would tell a caller the address is not registered", err)
	}
	if n := countRows(t, db, &control.EmailVerificationToken{}); n != 0 {
		t.Fatalf("email_verification_tokens = %d, want 0", n)
	}
}

func TestResendDoesNotReissueForAConfirmedAddress(t *testing.T) {
	db := registrationTestDB(t)
	svc := newVerificationService(db, &countingEnqueuer{})

	const raw = "already-used"
	user, _ := seedPendingRegistration(t, db, "done@example.com", raw, control.MemberRoleOwner)
	now := time.Now().UTC()
	if err := db.Model(&control.User{}).Where("id = ?", user.ID).
		Update("email_verified_at", now).Error; err != nil {
		t.Fatalf("stamp verified: %v", err)
	}

	if err := svc.Resend(t.Context(), "done@example.com"); err != nil {
		t.Fatalf("resend: %v", err)
	}
	if n := countRows(t, db, &control.EmailVerificationToken{}); n != 1 {
		t.Fatalf("email_verification_tokens = %d, want 1 — resend minted a link for an "+
			"address that is already confirmed", n)
	}
}

// The login gate, at the seam it actually guards. A correct password is supplied
// on purpose: the point is that credentials alone are not enough, and that the
// refusal is distinguishable so the client can offer a resend.
func TestUnconfirmedAccountIsRefusedEvenWithTheCorrectPassword(t *testing.T) {
	user := &control.User{Email: "pending@example.com", IsActive: true}
	if err := user.SetPassword("correct-horse-battery"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	users := &loginFakeUserRepo{byEmail: map[string]*control.User{"pending@example.com": user}}
	// A memberships repo is supplied so that the CONFIRMED half below gets past
	// this gate and stops at the next real check rather than panicking on a nil
	// dependency — a panic there would look like the gate rejecting, which is
	// exactly the false pass this test must not have.
	svc := NewLoginService(users, emptyMemberships{}, nil, nil, nil, nil, nil, newLoginFakeFailureRepo())

	_, err := svc.Login(context.Background(), LoginInput{
		Email:     "pending@example.com",
		Password:  "correct-horse-battery",
		IPAddress: "203.0.113.10",
	})
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("err = %v, want ErrEmailNotVerified — an unconfirmed account signed in, "+
			"which mints the token_session the admission ceiling counts as proof of use", err)
	}

	// And the same account, once confirmed, gets past this gate. Without this
	// half, deleting the EmailVerified() check entirely would still leave the
	// test above passing for the wrong reason if it were ever inverted.
	now := time.Now().UTC()
	user.EmailVerifiedAt = &now
	_, err = svc.Login(context.Background(), LoginInput{
		Email:     "pending@example.com",
		Password:  "correct-horse-battery",
		IPAddress: "203.0.113.10",
	})
	if errors.Is(err, ErrEmailNotVerified) {
		t.Fatal("a confirmed account was still refused by the confirmation gate")
	}
}

// emptyMemberships answers "no organizations", which is enough to carry a login
// past the confirmation gate to the next check.
type emptyMemberships struct{ repo.MembershipRepo }

func (emptyMemberships) ListByUserID(context.Context, uuid.UUID) ([]control.Membership, error) {
	return nil, nil
}

// Two clicks arriving together is the ordinary case here: a mail client that
// prefetches links while the human also clicks one.
//
// This asserts the end-to-end property — exactly one confirmation wins, exactly
// one provisioning job — but it does NOT pin the compare-and-set in MarkUsed.
// Removing that guard leaves this test green, because FindByTokenHash's
// used_at IS NULL filter wins the race on timing alone at these intervals.
// TestMarkUsedIsCompareAndSet below covers that line deterministically, at the
// seam where the property actually lives.
func TestConcurrentConfirmationsProvisionExactlyOnce(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	svc := newVerificationService(db, jobs)

	const raw = "raced-token"
	seedPendingRegistration(t, db, "raced@example.com", raw, control.MemberRoleOwner)

	const racers = 8
	var wg sync.WaitGroup
	var succeeded atomic.Int64
	start := make(chan struct{})

	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := svc.Confirm(context.Background(), raw); err == nil {
				succeeded.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if n := succeeded.Load(); n != 1 {
		t.Fatalf("%d of %d concurrent confirmations succeeded, want exactly 1 — "+
			"the token was redeemed more than once", n, racers)
	}
	if n := jobs.provisions.Load(); n != 1 {
		t.Fatalf("provisioning jobs = %d, want 1 — a raced confirmation asked for the same "+
			"tenant database more than once", n)
	}
}

// MarkUsed must burn a token exactly once, and report it when it did not.
//
// Tested here rather than through Confirm because no wall-clock race reliably
// reaches it: Confirm's lookup already excludes used tokens, so the second
// caller almost always fails earlier and the guard never runs. Driving the repo
// directly is the only way this line is under test at all — and it is the line
// standing between a raced confirmation and two CREATE DATABASE jobs for one
// organization.
func TestMarkUsedIsCompareAndSet(t *testing.T) {
	db := registrationTestDB(t)
	tokens := repo.NewEmailVerificationRepoGorm(db)

	const raw = "cas-token"
	seedPendingRegistration(t, db, "cas@example.com", raw, control.MemberRoleOwner)

	sum := sha256.Sum256([]byte(raw))
	found, err := tokens.FindByTokenHash(t.Context(), hex.EncodeToString(sum[:]))
	if err != nil || found == nil {
		t.Fatalf("find token: %v (found=%v)", err, found)
	}

	if err := tokens.MarkUsed(t.Context(), found.ID); err != nil {
		t.Fatalf("first MarkUsed: %v, want nil", err)
	}
	// The second caller holds the same id — exactly the state two racing
	// confirmations are in after both passed the lookup.
	if err := tokens.MarkUsed(t.Context(), found.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second MarkUsed: err = %v, want gorm.ErrRecordNotFound — the token was "+
			"burned twice, so both callers would go on to provision", err)
	}
}
