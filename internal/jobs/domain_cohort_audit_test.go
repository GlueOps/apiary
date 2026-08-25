package jobs

import (
	"testing"
	"time"

	"github.com/glueops/apiary/internal/workflow"
	"gorm.io/gorm"
)

func TestEvaluateDomainCohortsNeverFlagsBelowTheMinimumSample(t *testing.T) {
	opts := CohortAuditOptions{MinSample: 5, ProvenRatio: 0.25}
	at := time.Now().UTC()

	rows := []DomainCohortRow{
		{RegistrableDomain: "small.example", OrganizationsTotal: 4, OrganizationsProven: 0},
		{RegistrableDomain: "atsample.example", OrganizationsTotal: 5, OrganizationsProven: 0},
	}

	findings := EvaluateDomainCohorts(rows, opts, at)
	if len(findings) != 2 {
		t.Fatalf("findings = %d, want 2 — every cohort is recorded, flagged or not", len(findings))
	}
	if findings[0].Flagged {
		t.Fatal("a cohort of 4 was flagged at a minimum sample of 5; below the sample the ratio is noise")
	}
	if !findings[1].Flagged {
		t.Fatal("a cohort of 5 with none proven was not flagged at a minimum sample of 5")
	}
	if findings[0].ProvenRatio != 0 || findings[1].ProvenRatio != 0 {
		t.Fatalf("proven ratios = %v/%v, want 0 for both", findings[0].ProvenRatio, findings[1].ProvenRatio)
	}
	if !findings[0].EvaluatedAt.Equal(at) {
		t.Fatal("evaluated_at was not stamped from the caller's clock")
	}
}

func TestEvaluateDomainCohortsFlagsStrictlyBelowTheRatio(t *testing.T) {
	opts := CohortAuditOptions{MinSample: 2, ProvenRatio: 0.5}

	cases := []struct {
		name        string
		row         DomainCohortRow
		wantRatio   float64
		wantFlagged bool
	}{
		{"none proven", DomainCohortRow{RegistrableDomain: "a.example", OrganizationsTotal: 4, OrganizationsProven: 0}, 0, true},
		{"below threshold", DomainCohortRow{RegistrableDomain: "b.example", OrganizationsTotal: 4, OrganizationsProven: 1}, 0.25, true},
		{"exactly at threshold", DomainCohortRow{RegistrableDomain: "c.example", OrganizationsTotal: 4, OrganizationsProven: 2}, 0.5, false},
		{"above threshold", DomainCohortRow{RegistrableDomain: "d.example", OrganizationsTotal: 4, OrganizationsProven: 3}, 0.75, false},
		{"empty cohort", DomainCohortRow{RegistrableDomain: "e.example", OrganizationsTotal: 0, OrganizationsProven: 0}, 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateDomainCohort(tc.row, opts, time.Now().UTC())
			if got.ProvenRatio != tc.wantRatio {
				t.Fatalf("proven ratio = %v, want %v", got.ProvenRatio, tc.wantRatio)
			}
			if got.Flagged != tc.wantFlagged {
				t.Fatalf("flagged = %v, want %v", got.Flagged, tc.wantFlagged)
			}
		})
	}
}

func TestRollupGroupsPrivateSuffixesOnly(t *testing.T) {
	opts := CohortAuditOptions{MinSample: 3, ProvenRatio: 0.25}

	rows := []DomainCohortRow{
		// Three free eTLD+1s under one private suffix: the per-domain ceiling
		// sees three separate paid-for buckets, the rollup sees one actor.
		{RegistrableDomain: "a1.blogspot.com", OrganizationsTotal: 2},
		{RegistrableDomain: "a2.blogspot.com", OrganizationsTotal: 2},
		{RegistrableDomain: "a3.blogspot.com", OrganizationsTotal: 2},
		// Ordinary registrable domains under an ICANN suffix must never be
		// rolled together; ".com" as a cluster is meaningless.
		{RegistrableDomain: "one.example", OrganizationsTotal: 2},
		{RegistrableDomain: "two.example", OrganizationsTotal: 2},
		{RegistrableDomain: "three.example", OrganizationsTotal: 2},
		{RegistrableDomain: "four.example", OrganizationsTotal: 2},
		// An unknown single-label TLD is not an operator handing out subdomains.
		{RegistrableDomain: "x.internal", OrganizationsTotal: 2},
		{RegistrableDomain: "y.internal", OrganizationsTotal: 2},
		{RegistrableDomain: "z.internal", OrganizationsTotal: 2},
	}

	rollups := rollupPrivateSuffixes(rows, opts)
	if len(rollups) != 1 {
		t.Fatalf("rollups = %+v, want exactly the blogspot.com cluster", rollups)
	}
	if rollups[0].Suffix != "blogspot.com" {
		t.Fatalf("rollup suffix = %q, want blogspot.com", rollups[0].Suffix)
	}
	if rollups[0].Domains != 3 || rollups[0].OrganizationsTotal != 6 {
		t.Fatalf("rollup = %d domains / %d organizations, want 3/6", rollups[0].Domains, rollups[0].OrganizationsTotal)
	}
}

func TestRollupStaysQuietWhenTheClusterIsProven(t *testing.T) {
	opts := CohortAuditOptions{MinSample: 3, ProvenRatio: 0.25}

	rows := []DomainCohortRow{
		{RegistrableDomain: "a1.blogspot.com", OrganizationsTotal: 2, OrganizationsProven: 2},
		{RegistrableDomain: "a2.blogspot.com", OrganizationsTotal: 2, OrganizationsProven: 2},
		{RegistrableDomain: "a3.blogspot.com", OrganizationsTotal: 2, OrganizationsProven: 1},
	}

	if rollups := rollupPrivateSuffixes(rows, opts); len(rollups) != 0 {
		t.Fatalf("rollups = %+v, want none — a cluster above the proven ratio is not a finding", rollups)
	}
}

func TestNewDomainCohortAuditWorkerFailsAtWiring(t *testing.T) {
	valid := CohortAuditOptions{MinSample: 5, ProvenRatio: 0.25}

	assertPanics(t, "nil database", func() { NewDomainCohortAuditWorker(nil, valid) })
	assertPanics(t, "zero minimum sample", func() {
		NewDomainCohortAuditWorker(&gorm.DB{}, CohortAuditOptions{MinSample: 0, ProvenRatio: 0.25})
	})
	assertPanics(t, "zero proven ratio", func() {
		NewDomainCohortAuditWorker(&gorm.DB{}, CohortAuditOptions{MinSample: 5, ProvenRatio: 0})
	})
	assertPanics(t, "proven ratio of one", func() {
		NewDomainCohortAuditWorker(&gorm.DB{}, CohortAuditOptions{MinSample: 5, ProvenRatio: 1})
	})
}

func TestAdmissionJobKindsAreStable(t *testing.T) {
	// River addresses periodic jobs by kind. A renamed kind silently stops the
	// job from ever running again while the worker registration still looks
	// correct.
	if kind := (workflow.OrgProvingArgs{}).Kind(); kind != "admission.org_proving" {
		t.Fatalf("OrgProvingArgs kind = %q, want admission.org_proving", kind)
	}
	if kind := (workflow.DomainCohortAuditArgs{}).Kind(); kind != "admission.domain_cohort_audit" {
		t.Fatalf("DomainCohortAuditArgs kind = %q, want admission.domain_cohort_audit", kind)
	}
}
