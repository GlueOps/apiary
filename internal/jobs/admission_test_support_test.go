package jobs

import (
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// admissionTestSchema is private to this test binary invocation, so a run
// cannot migrate or truncate anything in public — including a developer's own
// control database, which is what TEST_DATABASE_URL has historically pointed at.
var admissionTestSchema = testsupport.SchemaName("apiary_jobs_admission_test")

func admissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return admissionTestDBInTimeZone(t, "")
}

// admissionTestDBInTimeZone opens the control database, optionally pinning the
// SESSION time zone of every connection in the pool. The time zone matters:
// proving days are UTC calendar days, and an implementation that used a bare
// created_at::date would silently take them in the server's zone instead.
func admissionTestDBInTimeZone(t *testing.T, tz string) *gorm.DB {
	t.Helper()

	var params map[string]string
	if tz != "" {
		params = map[string]string{"TimeZone": tz}
	}
	return testsupport.OpenSchemaWithParams(t, admissionTestSchema, params,
		&control.User{},
		&control.Membership{},
		&control.Organization{},
		&control.TokenSession{},
		&control.DomainCohortFinding{},
		&control.AuthAccessRule{},
	)
}

// uniqueRegistrableDomain keeps concurrent runs of this suite from sharing a
// cohort bucket, which would make the counts non-deterministic.
func uniqueRegistrableDomain(prefix string) string {
	return prefix + "-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
}

// mustCreateOrgForDomain builds an organization that COUNTS against the ceiling:
// one whose owner has confirmed their address. The ceiling deliberately ignores
// organizations founded on a merely asserted address, so a helper that omitted
// the owner would silently produce orgs that count as nothing and every ceiling
// assertion built on it would pass vacuously.
func mustCreateOrgForDomain(t *testing.T, db *gorm.DB, registrableDomain string) control.Organization {
	t.Helper()
	return createOrgForDomain(t, db, registrableDomain, true)
}

// mustCreateOrgForDomainWithUnconfirmedOwner is the attacker's shape: a
// registration at a domain whose mailboxes they cannot read.
func mustCreateOrgForDomainWithUnconfirmedOwner(t *testing.T, db *gorm.DB, registrableDomain string) control.Organization {
	t.Helper()
	return createOrgForDomain(t, db, registrableDomain, false)
}

func createOrgForDomain(t *testing.T, db *gorm.DB, registrableDomain string, ownerConfirmed bool) control.Organization {
	t.Helper()

	org := control.Organization{
		Name:              "cohort-" + registrableDomain,
		Status:            "active",
		RegistrableDomain: &registrableDomain,
	}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create organization for %s: %v", registrableDomain, err)
	}
	owner := control.User{
		Email:     "owner-" + org.ID.String() + "@" + registrableDomain,
		FirstName: "Owner",
		IsActive:  true,
	}
	if ownerConfirmed {
		now := time.Now().UTC()
		owner.EmailVerifiedAt = &now
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner for %s: %v", registrableDomain, err)
	}
	membership := control.Membership{
		UserID:         owner.ID,
		OrganizationID: org.ID,
		Role:           control.MemberRoleOwner,
	}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("create owner membership for %s: %v", registrableDomain, err)
	}

	t.Cleanup(func() {
		db.Where("organization_id = ?", org.ID).Delete(&control.TokenSession{})
		db.Delete(&control.Membership{}, "id = ?", membership.ID)
		db.Delete(&control.Organization{}, "id = ?", org.ID)
		db.Delete(&control.User{}, "id = ?", owner.ID)
		db.Where("registrable_domain = ?", registrableDomain).Delete(&control.DomainCohortFinding{})
	})
	return org
}

func mustCreateSessionAt(t *testing.T, db *gorm.DB, orgID uuid.UUID, at time.Time) {
	t.Helper()

	session := control.TokenSession{
		SubjectID:        uuid.New(),
		SubjectKind:      "user",
		OrganizationID:   orgID,
		FamilyID:         uuid.New(),
		AccessTokenJTI:   uuid.NewString(),
		RefreshTokenJTI:  uuid.NewString(),
		AccessTokenHash:  strings.Repeat("a", 64),
		RefreshTokenHash: strings.Repeat("b", 64),
		AccessExpiresAt:  at.Add(time.Hour),
		RefreshExpiresAt: at.Add(24 * time.Hour),
		CreatedAt:        at,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create token session at %s: %v", at, err)
	}
}

func reloadOrg(t *testing.T, db *gorm.DB, id uuid.UUID) control.Organization {
	t.Helper()

	var org control.Organization
	if err := db.First(&org, "id = ?", id).Error; err != nil {
		t.Fatalf("reload organization: %v", err)
	}
	return org
}
