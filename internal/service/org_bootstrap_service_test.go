package service

import (
	"context"
	"errors"
	"github.com/glueops/apiary/internal/testsupport"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var bootstrapTestSchema = testsupport.SchemaName("apiary_orgbootstrap_test")

type countingEnqueuer struct {
	// Inherits no-op implementations of the job kinds this fake has no opinion
	// about — the fleet lifecycle ones. Overridden methods below still win.
	workflow.MockEnqueuer

	emails     atomic.Int64
	provisions atomic.Int64
	fail       bool
}

func (c *countingEnqueuer) EnqueueProvisionTenant(context.Context, *workflow.ProvisionTenantArgs) error {
	c.provisions.Add(1)
	if c.fail {
		return errors.New("enqueue failed")
	}
	return nil
}

func (c *countingEnqueuer) EnqueueDestroyTenant(context.Context, *workflow.DestroyTenantArgs) error {
	return nil
}

func bootstrapTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the organization quota suite; " +
			"this test does NOT pass, it did not run")
	}

	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + bootstrapTestSchema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"search_path="+bootstrapTestSchema), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open scoped test db: %v", err)
	}
	if err := db.AutoMigrate(&control.User{}, &control.Organization{}, &control.Membership{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Exec(`TRUNCATE memberships, organizations, users CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`TRUNCATE memberships, organizations, users CASCADE`)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newTestUser(t *testing.T, db *gorm.DB) *control.User {
	t.Helper()
	user := &control.User{
		Email:     "bootstrap-" + uuid.NewString() + "@example.test",
		Password:  "x",
		FirstName: "Test",
		LastName:  "User",
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func countOrgs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&control.Organization{}).Count(&n).Error; err != nil {
		t.Fatalf("count orgs: %v", err)
	}
	return n
}

func TestEnsureUserHasWorkspaceRefusesUnconfiguredLimits(t *testing.T) {
	svc := NewOrgBootstrapService(&gorm.DB{}, &countingEnqueuer{}, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{}))
	_, err := svc.EnsureUserHasWorkspace(context.Background(), &control.User{ID: uuid.New()})
	if err == nil {
		t.Fatal("expected an error when quota limits are unset, got nil")
	}
	if !strings.Contains(err.Error(), "quota limits are not configured") {
		t.Fatalf("unexpected error: %v", err)
	}

	svc = NewOrgBootstrapService(&gorm.DB{}, &countingEnqueuer{}, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 5}))
	if _, err := svc.EnsureUserHasWorkspace(context.Background(), &control.User{ID: uuid.New()}); err == nil {
		t.Fatal("expected an error when only one of the two limits is set, got nil")
	}
}

func TestEnsureUserHasWorkspaceEnforcesTheTotalOrganizationQuota(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 2, MaxNewOrgsPerHour: 100, MaxUnprovenOrgsPerDomain: 100}))

	for i := range 2 {
		if _, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db)); err != nil {
			t.Fatalf("sign-in %d should have been allowed: %v", i, err)
		}
	}

	_, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if !errors.Is(err, ErrOrgQuotaExceeded) {
		t.Fatalf("third sign-in: err = %v, want ErrOrgQuotaExceeded", err)
	}
	if got := countOrgs(t, db); got != 2 {
		t.Fatalf("organizations = %d, want 2 — the quota did not stop the create", got)
	}
	if got := jobs.provisions.Load(); got != 2 {
		t.Fatalf("provisioning jobs = %d, want 2 — a denied sign-in still asked for a database", got)
	}
}

func TestEnsureUserHasWorkspaceEnforcesTheHourlyRate(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 1000, MaxNewOrgsPerHour: 1, MaxUnprovenOrgsPerDomain: 100}))

	if _, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db)); err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	_, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if !errors.Is(err, ErrOrgCreationRateExceeded) {
		t.Fatalf("second sign-in: err = %v, want ErrOrgCreationRateExceeded", err)
	}
	if got := countOrgs(t, db); got != 1 {
		t.Fatalf("organizations = %d, want 1", got)
	}
}

func TestConcurrentFirstSignInsCreateExactlyOneOrganization(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 1000, MaxNewOrgsPerHour: 1000, MaxUnprovenOrgsPerDomain: 100}))

	user := newTestUser(t, db)

	const goroutines = 8
	memberships := make([]*control.Membership, goroutines)
	errs := make([]error, goroutines)

	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	for i := range goroutines {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			memberships[i], errs[i] = svc.EnsureUserHasWorkspace(context.Background(), user)
		}()
	}
	start.Done()
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	if got := countOrgs(t, db); got != 1 {
		t.Fatalf("organizations = %d, want 1 — one account multiplied into %d databases", got, got)
	}
	if got := jobs.provisions.Load(); got != 1 {
		t.Fatalf("provisioning jobs = %d, want 1", got)
	}
	for i, m := range memberships {
		if m == nil || m.OrganizationID != memberships[0].OrganizationID {
			t.Fatalf("goroutine %d returned a different organization", i)
		}
	}

	var count int64
	if err := db.Model(&control.Membership{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if count != 1 {
		t.Fatalf("memberships = %d, want 1", count)
	}
}

func TestExistingMembershipIsReturnedEvenWhenTheQuotaIsFull(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 1, MaxNewOrgsPerHour: 1, MaxUnprovenOrgsPerDomain: 100}))

	user := newTestUser(t, db)
	first, err := svc.EnsureUserHasWorkspace(t.Context(), user)
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}

	again, err := svc.EnsureUserHasWorkspace(t.Context(), user)
	if err != nil {
		t.Fatalf("returning sign-in was denied by the quota: %v", err)
	}
	if again.OrganizationID != first.OrganizationID {
		t.Fatal("returning sign-in landed in a different organization")
	}
	if got := jobs.provisions.Load(); got != 1 {
		t.Fatalf("provisioning jobs = %d, want 1 — a returning user re-enqueued provisioning", got)
	}
}

func TestEnqueueFailureLeavesThePendingOrganizationAndDoesNotFailTheSignIn(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{fail: true}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{MaxTotalOrgs: 10, MaxNewOrgsPerHour: 10, MaxUnprovenOrgsPerDomain: 100}))

	m, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if err != nil {
		t.Fatalf("sign-in failed because the enqueue failed: %v", err)
	}

	var org control.Organization
	if err := db.First(&org, "id = ?", m.OrganizationID).Error; err != nil {
		t.Fatalf("load org: %v", err)
	}
	if org.Status != "pending" {
		t.Fatalf("org status = %q, want pending", org.Status)
	}
}

// EnqueueSendEmail satisfies workflow.Enqueuer. These tests are about tenant
// provisioning and never send mail, so it records the call and succeeds — a
// silent no-op would still compile if a future test started depending on it.
func (c *countingEnqueuer) EnqueueSendEmail(context.Context, *workflow.SendEmailArgs) error {
	c.emails.Add(1)
	return nil
}
