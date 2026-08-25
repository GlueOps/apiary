package jobs

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// mutationOf matches a statement whose write TARGET is the given table. It
// anchors on the target rather than searching for the name anywhere, because
// the findings table's own column names contain "organizations" and a looser
// match would report the pass's one legitimate write as a violation.
//
// The cohort pass ships in SHADOW MODE, so the assertion is not "it did not
// block anyone this time" but "it cannot: it never issues a write that would".
func mutationOf(table string) *regexp.Regexp {
	return regexp.MustCompile(`(?is)^\s*(insert\s+into|update|delete\s+from)\s+"?` + regexp.QuoteMeta(table) + `"?\b`)
}

type sqlCapture struct {
	gormlogger.Interface
	statements []string
}

func (c *sqlCapture) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	c.statements = append(c.statements, sql)
}

func (c *sqlCapture) mutationsTouching(table string) []string {
	pattern := mutationOf(table)
	var hits []string
	for _, sql := range c.statements {
		if pattern.MatchString(sql) {
			hits = append(hits, sql)
		}
	}
	return hits
}

func TestCohortAuditWritesOnlyFindingsAndBlocksNothing(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	domain := uniqueRegistrableDomain("shadow")
	orgs := []control.Organization{
		mustCreateOrgForDomain(t, db, domain),
		mustCreateOrgForDomain(t, db, domain),
		mustCreateOrgForDomain(t, db, domain),
	}

	// Scoped to THIS test's unique domain, not a global COUNT(*). auth_access_rules
	// lives in the default search_path, which internal/repo's suite also writes to
	// concurrently, so a whole-table count measures the other package's inserts and
	// deletes rather than this worker. Scoping to the domain keeps exactly the
	// property that matters -- an auto-block would name this cohort's domain -- and
	// mutationsTouching below is the race-free primary check that the pass issued no
	// write to the table at all.
	var rulesBefore int64
	if err := db.Model(&control.AuthAccessRule{}).
		Where("value = ?", domain).
		Count(&rulesBefore).Error; err != nil {
		t.Fatalf("count access rules: %v", err)
	}
	if rulesBefore != 0 {
		t.Fatalf("a rule already names %s before the pass ran; the fixture is not clean", domain)
	}

	capture := &sqlCapture{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	observed := db.Session(&gorm.Session{Logger: capture})

	worker := NewDomainCohortAuditWorker(observed, CohortAuditOptions{MinSample: 3, ProvenRatio: 0.25})
	if err := worker.Work(ctx, nil); err != nil {
		t.Fatalf("work: %v", err)
	}

	if hits := capture.mutationsTouching("organizations"); len(hits) > 0 {
		t.Fatalf("the cohort pass wrote to organizations — shadow mode forbids it: %v", hits)
	}
	if hits := capture.mutationsTouching("auth_access_rules"); len(hits) > 0 {
		t.Fatalf("the cohort pass wrote to auth_access_rules — shadow mode forbids auto-blocking: %v", hits)
	}

	for _, org := range orgs {
		after := reloadOrg(t, db, org.ID)
		if OrganizationIsProven(&after) {
			t.Fatal("the cohort pass stamped proven_at; proving belongs to admission.org_proving alone")
		}
		if after.Status != org.Status {
			t.Fatalf("the cohort pass changed organization status from %q to %q", org.Status, after.Status)
		}
	}

	var rulesAfter int64
	if err := db.Model(&control.AuthAccessRule{}).
		Where("value = ?", domain).
		Count(&rulesAfter).Error; err != nil {
		t.Fatalf("count access rules after: %v", err)
	}
	if rulesAfter != rulesBefore {
		t.Fatalf("auth_access_rules rows naming %s moved from %d to %d — nothing may be auto-blocked", domain, rulesBefore, rulesAfter)
	}

	var finding control.DomainCohortFinding
	if err := db.First(&finding, "registrable_domain = ?", domain).Error; err != nil {
		t.Fatalf("expected a recorded finding for %s: %v", domain, err)
	}
	if finding.OrganizationsTotal != 3 || finding.OrganizationsProven != 0 {
		t.Fatalf("finding = %d/%d organizations, want 3/0", finding.OrganizationsProven, finding.OrganizationsTotal)
	}
	if !finding.Flagged {
		t.Fatal("a cohort of three organizations with none proven was not flagged")
	}
}

func TestCohortAuditUpsertsInsteadOfAccumulatingRows(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	domain := uniqueRegistrableDomain("upsert")
	org := mustCreateOrgForDomain(t, db, domain)
	mustCreateOrgForDomain(t, db, domain)

	worker := NewDomainCohortAuditWorker(db, CohortAuditOptions{MinSample: 2, ProvenRatio: 0.5})
	if err := worker.Work(ctx, nil); err != nil {
		t.Fatalf("first pass: %v", err)
	}

	var first control.DomainCohortFinding
	if err := db.First(&first, "registrable_domain = ?", domain).Error; err != nil {
		t.Fatalf("first finding: %v", err)
	}
	if !first.Flagged {
		t.Fatal("two organizations with none proven were not flagged at a 0.5 threshold")
	}

	proven := time.Now().UTC()
	if err := db.Model(&control.Organization{}).
		Where("id = ?", org.ID).
		Update("proven_at", proven).Error; err != nil {
		t.Fatalf("stamp proven_at: %v", err)
	}

	if err := worker.Work(ctx, nil); err != nil {
		t.Fatalf("second pass: %v", err)
	}

	var findings []control.DomainCohortFinding
	if err := db.Find(&findings, "registrable_domain = ?", domain).Error; err != nil {
		t.Fatalf("reload findings: %v", err)
	}
	// One row per domain, refreshed in place: an append-only history would make
	// the flagged set unreadable after a few days of runs.
	if len(findings) != 1 {
		t.Fatalf("findings rows for %s = %d, want exactly 1 (the pass upserts)", domain, len(findings))
	}
	if findings[0].OrganizationsProven != 1 {
		t.Fatalf("organizations_proven = %d after one was proven, want 1", findings[0].OrganizationsProven)
	}
	// 1/2 = 0.5 is NOT below a 0.5 threshold, so the flag must clear.
	if findings[0].Flagged {
		t.Fatal("the cohort stayed flagged after its proven ratio reached the threshold")
	}
}

func TestMutationMatcherWouldSeeAnEnforcementWrite(t *testing.T) {
	// Guards the guard: if this matcher stopped recognising the shape of an
	// enforcement write, the shadow-mode test above would pass while the pass
	// quietly suspended organizations.
	capture := &sqlCapture{statements: []string{
		`SELECT o.registrable_domain, count(*) FROM organizations o GROUP BY 1`,
		`INSERT INTO "domain_cohort_findings" ("registrable_domain","organizations_total") VALUES ('x.example',3)`,
		`UPDATE "organizations" SET "status"='suspended' WHERE registrable_domain = 'x.example'`,
		`INSERT INTO "auth_access_rules" ("kind","value","purpose","policy") VALUES ('domain','x.example','create_org','deny')`,
	}}

	if hits := capture.mutationsTouching("organizations"); len(hits) != 1 {
		t.Fatalf("matched %d organization writes, want exactly the UPDATE: %v", len(hits), hits)
	}
	if hits := capture.mutationsTouching("auth_access_rules"); len(hits) != 1 {
		t.Fatalf("matched %d access rule writes, want exactly the INSERT: %v", len(hits), hits)
	}
	if hits := capture.mutationsTouching("domain_cohort_findings"); len(hits) != 1 {
		t.Fatalf("matched %d findings writes, want exactly the upsert: %v", len(hits), hits)
	}
}
