package service

import (
	"context"
	"errors"
	"github.com/glueops/apiary/internal/testsupport"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var registrationTestSchema = testsupport.SchemaName("apiary_registration_test")

// purposeRules answers the two purposes independently. Registration consults
// both — the ban list before it writes anything, the create_org list inside the
// gate — and the tests below need to tell the two decisions apart.
type purposeRules struct {
	authenticate error
	createOrg    error
	seen         []repo.AccessPurpose
}

func (p *purposeRules) CheckAccessFor(_ context.Context, _ string, purpose repo.AccessPurpose) error {
	p.seen = append(p.seen, purpose)
	if purpose == repo.AccessPurposeCreateOrg {
		return p.createOrg
	}
	return p.authenticate
}

func registrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the registration admission suite; " +
			"this test does NOT pass, it did not run")
	}

	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + registrationTestSchema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"search_path="+registrationTestSchema), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open scoped test db: %v", err)
	}
	if err := db.AutoMigrate(
		&control.User{},
		&control.Organization{},
		&control.Membership{},
		&control.TokenSession{},
		&control.AuthAuditEvent{},
		&control.EmailVerificationToken{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	truncate := `TRUNCATE email_verification_tokens, auth_audit_events, token_sessions, memberships, organizations, users CASCADE`
	if err := db.Exec(truncate).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(truncate)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// jobs and sessions are still accepted, and still deliberately handed to the
// caller's assertions, even though registration no longer touches either: they
// moved to email confirmation. Keeping them in the signature is what lets
// assertNothingWasCreated go on asserting that a REFUSED registration enqueued
// no provisioning — an assertion that would quietly stop meaning anything if
// the enqueuer vanished from these tests entirely.
func newRegistrationService(db *gorm.DB, jobs *countingEnqueuer, rules AccessRuleResolver, gate *AdmissionService, sessions *TokenSessionService) *RegistrationService {
	users := repo.NewUserRepoGorm(db)
	_, _ = jobs, sessions
	return NewRegistrationService(db, &users, nil, rules, gate, nil)
}

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var n int64
	if err := db.Model(model).Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// assertNothingWasCreated is the assertion this whole change exists for. Before
// the gate, a refused registration still left behind a user, an organization, a
// membership and an enqueued CREATE DATABASE + CREATE ROLE — an orphan nothing
// ever cleaned up, because the refusal only happened at token issuance.
func assertNothingWasCreated(t *testing.T, db *gorm.DB, jobs *countingEnqueuer) {
	t.Helper()
	if n := countRows(t, db, &control.User{}); n != 0 {
		t.Fatalf("users = %d, want 0 — a refused registration created an account", n)
	}
	if n := countRows(t, db, &control.Organization{}); n != 0 {
		t.Fatalf("organizations = %d, want 0 — a refused registration created a tenant", n)
	}
	if n := countRows(t, db, &control.Membership{}); n != 0 {
		t.Fatalf("memberships = %d, want 0", n)
	}
	if n := jobs.provisions.Load(); n != 0 {
		t.Fatalf("provisioning jobs = %d, want 0 — a refused registration asked for a Postgres database", n)
	}
}

func registerInput(email string) RegisterInput {
	return RegisterInput{
		Email:     email,
		Password:  "correct horse battery staple",
		FirstName: "Test",
		LastName:  "User",
	}
}

func TestRegisterCreatesNothingWhenTheDomainMayNotCreateOrganizations(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{createOrg: repo.ErrAccessDenied}
	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, testLimits()), nil)

	_, err := svc.Register(t.Context(), registerInput("founder@evilcorp.com"))
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied", err)
	}
	assertNothingWasCreated(t, db, jobs)
}

// A create_org deny must NOT be reached through the authenticate list, and an
// authenticate deny must stop registration before a single row is written.
func TestRegisterCreatesNothingWhenTheBanListRefusesTheIdentity(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{authenticate: repo.ErrAccessDenied}
	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, testLimits()), nil)

	_, err := svc.Register(t.Context(), registerInput("banned@example.test"))
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied", err)
	}
	assertNothingWasCreated(t, db, jobs)

	if len(rules.seen) == 0 || rules.seen[0] != repo.AccessPurposeAuthenticate {
		t.Fatalf("purposes consulted = %v, want the authenticate list first", rules.seen)
	}
}

// Registration is now behind the same per-domain ceiling as the social path.
// Before this change it was behind nothing at all.
func TestRegisterEnforcesTheUnprovenOrganizationCeiling(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{}
	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, testLimits()), nil)

	// Three unproven organizations already exist for this eTLD+1, seeded under
	// a sub-domain so the test also proves the bucket is not the literal host.
	seedOrgs(t, db, "evilcorp.com", 3, false, time.Now().UTC())

	// Each seeded organization carries a confirmed owner — the ceiling counts
	// only those — so the baseline is not zero and the assertion is about what
	// the REFUSED registration added, which is nothing.
	usersBefore := countRows(t, db, &control.User{})

	_, err := svc.Register(t.Context(), registerInput("founder+tag@mail.evilcorp.com"))
	if !errors.Is(err, ErrUnprovenOrgCeiling) {
		t.Fatalf("err = %v, want ErrUnprovenOrgCeiling", err)
	}
	if n := countRows(t, db, &control.User{}); n != usersBefore {
		t.Fatalf("users = %d, want %d — the refused registration created an account", n, usersBefore)
	}
	if n := countRows(t, db, &control.Organization{}); n != 3 {
		t.Fatalf("organizations = %d, want the 3 seeded ones", n)
	}
	if n := jobs.provisions.Load(); n != 0 {
		t.Fatalf("provisioning jobs = %d, want 0", n)
	}
}

// The platform quotas existed only on the social bootstrap path; registration
// walked straight past them.
func TestRegisterEnforcesThePlatformOrganizationQuotas(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{}
	limits := OrgBootstrapLimits{
		MaxTotalOrgs:             1,
		MaxNewOrgsPerHour:        1000,
		MaxUnprovenOrgsPerDomain: 100,
	}
	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, limits), nil)

	seedOrgs(t, db, "someone.example", 1, false, time.Now().UTC())
	usersBefore := countRows(t, db, &control.User{})

	_, err := svc.Register(t.Context(), registerInput("founder@fresh.example"))
	if !errors.Is(err, ErrOrgQuotaExceeded) {
		t.Fatalf("err = %v, want ErrOrgQuotaExceeded", err)
	}
	if n := countRows(t, db, &control.User{}); n != usersBefore {
		t.Fatalf("users = %d, want %d — the quota did not stop the account", n, usersBefore)
	}
}

func TestRegisterRefusesAnAddressWithNoRegistrableDomain(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{}
	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, testLimits()), nil)

	_, err := svc.Register(t.Context(), registerInput("founder@localhost"))
	if !errors.Is(err, ErrUnresolvableDomain) {
		t.Fatalf("err = %v, want ErrUnresolvableDomain", err)
	}
	assertNothingWasCreated(t, db, jobs)
}

// An unwired resolver or an unwired gate must refuse, not admit.
func TestRegisterRefusesWhenAdmissionIsNotWired(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}

	noRules := newRegistrationService(db, jobs, nil, NewAdmissionService(&purposeRules{}, testLimits()), nil)
	if _, err := noRules.Register(t.Context(), registerInput("founder@example.test")); !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("unwired resolver: err = %v, want ErrAdmissionNotConfigured", err)
	}

	rules := &purposeRules{}
	noGate := newRegistrationService(db, jobs, rules, nil, nil)
	if _, err := noGate.Register(t.Context(), registerInput("founder@example.test")); !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("unwired gate: err = %v, want ErrAdmissionNotConfigured", err)
	}
	assertNothingWasCreated(t, db, jobs)
}

// The admitted organization carries its eTLD+1. An unstamped organization is
// invisible to every later ceiling count, which would make the ceiling leak one
// free slot per registration.
func TestRegisterStampsTheRegistrableDomainOnTheOrganization(t *testing.T) {
	db := registrationTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{}

	tokens, err := NewTokenService("test-jwt-secret-that-is-long-enough-1234", "apiary-test", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("token service: %v", err)
	}
	users := repo.NewUserRepoGorm(db)
	sessions := NewTokenSessionService(tokens, repo.NewTokenSessionRepoGorm(db), repo.NewAuthAuditRepoGorm(db), &users)

	svc := newRegistrationService(db, jobs, rules, NewAdmissionService(rules, testLimits()), sessions)

	result, err := svc.Register(t.Context(), registerInput("founder+tag@Mail.Acme.co.uk"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	var org control.Organization
	if err := db.First(&org, "id = ?", result.OrganizationID).Error; err != nil {
		t.Fatalf("load org: %v", err)
	}
	if org.RegistrableDomain == nil {
		t.Fatal("registrable_domain is NULL — the organization is invisible to the ceiling")
	}
	if *org.RegistrableDomain != "acme.co.uk" {
		t.Fatalf("registrable_domain = %q, want %q", *org.RegistrableDomain, "acme.co.uk")
	}
	if org.ProvenAt != nil {
		t.Fatal("a brand new organization must start unproven")
	}
}
