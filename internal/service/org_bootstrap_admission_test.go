package service

import (
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

// The social bootstrap path goes through the same gate as registration, so the
// create_org rules now apply to it too — and a refusal must leave no
// organization and no provisioning job behind.
func TestEnsureUserHasWorkspaceRefusesADeniedDomain(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	rules := &purposeRules{createOrg: repo.ErrAccessDenied}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(rules, testLimits()))

	_, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied", err)
	}
	if got := countOrgs(t, db); got != 0 {
		t.Fatalf("organizations = %d, want 0", got)
	}
	if got := jobs.provisions.Load(); got != 0 {
		t.Fatalf("provisioning jobs = %d, want 0 — a denied sign-in asked for a database", got)
	}
	if len(rules.seen) != 1 || rules.seen[0] != repo.AccessPurposeCreateOrg {
		t.Fatalf("purposes consulted = %v, want exactly [create_org]", rules.seen)
	}
}

func TestEnsureUserHasWorkspaceEnforcesTheUnprovenOrganizationCeiling(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, testLimits()))

	// newTestUser's addresses all live on example.test, so this fills that
	// domain's ceiling exactly.
	seedOrgs(t, db, "example.test", 3, false, time.Now().UTC())

	_, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if !errors.Is(err, ErrUnprovenOrgCeiling) {
		t.Fatalf("err = %v, want ErrUnprovenOrgCeiling", err)
	}
	if got := countOrgs(t, db); got != 3 {
		t.Fatalf("organizations = %d, want the 3 seeded ones", got)
	}
	if got := jobs.provisions.Load(); got != 0 {
		t.Fatalf("provisioning jobs = %d, want 0", got)
	}
}

func TestEnsureUserHasWorkspaceStampsTheRegistrableDomain(t *testing.T) {
	db := bootstrapTestDB(t)
	jobs := &countingEnqueuer{}
	svc := NewOrgBootstrapService(db, jobs, NewAdmissionService(admitEveryDomain{}, testLimits()))

	m, err := svc.EnsureUserHasWorkspace(t.Context(), newTestUser(t, db))
	if err != nil {
		t.Fatalf("sign-in: %v", err)
	}

	var org control.Organization
	if err := db.First(&org, "id = ?", m.OrganizationID).Error; err != nil {
		t.Fatalf("load org: %v", err)
	}
	if org.RegistrableDomain == nil || *org.RegistrableDomain != "example.test" {
		t.Fatalf("registrable_domain = %v, want example.test — an unstamped organization is a free ceiling slot", org.RegistrableDomain)
	}
}
