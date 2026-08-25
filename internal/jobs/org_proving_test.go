package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
)

func TestOrganizationIsProvenReadsTheStampedColumn(t *testing.T) {
	if OrganizationIsProven(nil) {
		t.Fatal("a nil organization was reported proven — an unknown organization must never be treated as proven")
	}

	org := &control.Organization{}
	if OrganizationIsProven(org) {
		t.Fatal("an organization with no proven_at was reported proven")
	}

	at := time.Now().UTC()
	org.ProvenAt = &at
	if !OrganizationIsProven(org) {
		t.Fatal("an organization with proven_at set was not reported proven")
	}
}

func TestSweepRefusesAThresholdBelowTwoDistinctDays(t *testing.T) {
	// A one-day threshold is satisfied by the session minted at registration,
	// so it would prove every organization the moment it is created and turn
	// the unproven-organization ceiling off without anyone noticing.
	for _, days := range []int{-1, 0, 1} {
		_, err := SweepOrganizationProving(context.Background(), nil, days)
		if !errors.Is(err, ErrProvingThresholdTooLow) {
			t.Fatalf("SweepOrganizationProving(days=%d) error = %v, want ErrProvingThresholdTooLow", days, err)
		}
	}
}

func TestNewOrgProvingWorkerFailsAtWiring(t *testing.T) {
	assertPanics(t, "nil database", func() { NewOrgProvingWorker(nil, 2) })
	assertPanics(t, "one-day threshold", func() { NewOrgProvingWorker(&gorm.DB{}, 1) })
}

func TestSweepProvesAfterTwoDistinctUTCDays(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	domain := uniqueRegistrableDomain("proving")
	org := mustCreateOrgForDomain(t, db, domain)

	base := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	mustCreateSessionAt(t, db, org.ID, base)
	mustCreateSessionAt(t, db, org.ID, base.AddDate(0, 0, 1))

	if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	after := reloadOrg(t, db, org.ID)
	if !OrganizationIsProven(&after) {
		t.Fatal("an organization with sessions on two distinct UTC days was not proven")
	}
	if after.ProvingDaysSeen != 2 {
		t.Fatalf("proving_days_seen = %d, want 2", after.ProvingDaysSeen)
	}
}

func TestSweepDoesNotProveABurstOnASingleDay(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	org := mustCreateOrgForDomain(t, db, uniqueRegistrableDomain("burst"))

	// The whole point of the definition: an attacker can mint any number of
	// sessions in one sitting. Only genuinely coming back on another calendar
	// day counts.
	base := time.Date(2026, 3, 2, 0, 30, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		mustCreateSessionAt(t, db, org.ID, base.Add(time.Duration(i)*time.Hour))
	}

	if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	after := reloadOrg(t, db, org.ID)
	if OrganizationIsProven(&after) {
		t.Fatal("twelve sessions on ONE UTC day proved an organization")
	}
	if after.ProvingDaysSeen != 1 {
		t.Fatalf("proving_days_seen = %d, want 1", after.ProvingDaysSeen)
	}
}

func TestSweepNeverRecountsADayAcrossRuns(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	org := mustCreateOrgForDomain(t, db, uniqueRegistrableDomain("idempotent"))

	day := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	mustCreateSessionAt(t, db, org.ID, day)

	for run := 1; run <= 3; run++ {
		if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
			t.Fatalf("sweep run %d: %v", run, err)
		}
		// A later session on the SAME day must not buy a second credit.
		mustCreateSessionAt(t, db, org.ID, day.Add(time.Duration(run)*time.Hour))
	}

	after := reloadOrg(t, db, org.ID)
	if OrganizationIsProven(&after) {
		t.Fatal("re-running the sweep over one day of sessions proved an organization")
	}
	if after.ProvingDaysSeen != 1 {
		t.Fatalf("proving_days_seen = %d after three sweeps of a single day, want 1", after.ProvingDaysSeen)
	}

	mustCreateSessionAt(t, db, org.ID, day.AddDate(0, 0, 1))
	if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
		t.Fatalf("sweep after second day: %v", err)
	}

	proven := reloadOrg(t, db, org.ID)
	if !OrganizationIsProven(&proven) {
		t.Fatal("a genuine return on a second UTC day did not prove the organization")
	}
}

func TestSweepTakesDaysInUTCNotTheServerTimeZone(t *testing.T) {
	// Both instants fall on the same calendar day in America/New_York and on
	// two different days in UTC. A bare created_at::date would resolve in the
	// connection's zone and refuse to prove this organization.
	db := admissionTestDBInTimeZone(t, "America/New_York")
	ctx := context.Background()

	org := mustCreateOrgForDomain(t, db, uniqueRegistrableDomain("utcdays"))

	mustCreateSessionAt(t, db, org.ID, time.Date(2026, 1, 1, 23, 30, 0, 0, time.UTC))
	mustCreateSessionAt(t, db, org.ID, time.Date(2026, 1, 2, 0, 30, 0, 0, time.UTC))

	if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	after := reloadOrg(t, db, org.ID)
	if after.ProvingDaysSeen != 2 || !OrganizationIsProven(&after) {
		t.Fatalf("proving_days_seen = %d, proven = %v; two sessions straddling UTC midnight must count as two UTC days",
			after.ProvingDaysSeen, OrganizationIsProven(&after))
	}
}

func TestSweepLeavesAlreadyProvenOrganizationsAlone(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	org := mustCreateOrgForDomain(t, db, uniqueRegistrableDomain("stable"))

	stamped := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Model(&control.Organization{}).
		Where("id = ?", org.ID).
		Updates(map[string]any{"proven_at": stamped, "proving_days_seen": 2}).Error; err != nil {
		t.Fatalf("stamp proven_at: %v", err)
	}

	mustCreateSessionAt(t, db, org.ID, time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC))
	if _, err := SweepOrganizationProving(ctx, db, 2); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	after := reloadOrg(t, db, org.ID)
	if after.ProvenAt == nil || !after.ProvenAt.UTC().Equal(stamped) {
		t.Fatalf("proven_at = %v, want it left at %v — proof is recorded once, not re-dated on every sweep",
			after.ProvenAt, stamped)
	}
	if after.ProvingDaysSeen != 2 {
		t.Fatalf("proving_days_seen = %d, want it left at 2", after.ProvingDaysSeen)
	}
}

func TestCountUnprovenOrganizationsIsScopedAndDeniesAnUnknownBucket(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	domain := uniqueRegistrableDomain("ceiling")
	other := uniqueRegistrableDomain("ceiling-other")

	first := mustCreateOrgForDomain(t, db, domain)
	mustCreateOrgForDomain(t, db, domain)
	mustCreateOrgForDomain(t, db, other)

	count, err := CountUnprovenOrganizations(ctx, db, domain)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("unproven organizations for %s = %d, want 2 (the other domain must not be counted)", domain, count)
	}

	proven := time.Now().UTC()
	if err := db.Model(&control.Organization{}).
		Where("id = ?", first.ID).
		Update("proven_at", proven).Error; err != nil {
		t.Fatalf("stamp proven_at: %v", err)
	}

	count, err = CountUnprovenOrganizations(ctx, db, domain)
	if err != nil {
		t.Fatalf("count after proving: %v", err)
	}
	if count != 1 {
		t.Fatalf("unproven organizations after proving one = %d, want 1", count)
	}

	// An empty bucket key means the caller could not identify the domain. That
	// must be an error the gate turns into a denial, never a zero count that
	// reads as free capacity.
	if _, err := CountUnprovenOrganizations(ctx, db, ""); err == nil {
		t.Fatal("counting an empty registrable domain returned no error — an unidentifiable bucket must not look empty")
	}
}

func assertPanics(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s did not panic — misconfiguration must fail at wiring, not silently at run time", what)
		}
	}()
	fn()
}

// TestUnconfirmedRegistrationsDoNotConsumeTheCeiling is the assertion that keeps
// the ceiling a control rather than a weapon.
//
// The attack it rules out: register three times as a@victim-corp.com,
// b@victim-corp.com and c@victim-corp.com — addresses the attacker cannot read —
// and the domain's budget is exhausted. The ceiling has no time component and
// nothing clears it, so every real employee of that company is refused
// permanently, by a control the platform runs against itself.
//
// Confirmation is what makes a slot cost something. An asserted address costs
// nothing, so it must buy nothing.
func TestUnconfirmedRegistrationsDoNotConsumeTheCeiling(t *testing.T) {
	db := admissionTestDB(t)
	ctx := context.Background()

	domain := uniqueRegistrableDomain("ceiling-unconfirmed")

	// Three registrations at a domain nobody here can receive mail at.
	mustCreateOrgForDomainWithUnconfirmedOwner(t, db, domain)
	mustCreateOrgForDomainWithUnconfirmedOwner(t, db, domain)
	mustCreateOrgForDomainWithUnconfirmedOwner(t, db, domain)

	count, err := CountUnprovenOrganizations(ctx, db, domain)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("unproven organizations for %s = %d, want 0 — unconfirmed registrations "+
			"consumed the domain's ceiling, which locks every real employee of that "+
			"company out permanently", domain, count)
	}

	// And a real one at the same domain does count, so the ceiling still works.
	mustCreateOrgForDomain(t, db, domain)
	count, err = CountUnprovenOrganizations(ctx, db, domain)
	if err != nil {
		t.Fatalf("count after a confirmed registration: %v", err)
	}
	if count != 1 {
		t.Fatalf("unproven organizations for %s = %d, want 1 — a CONFIRMED registration "+
			"must occupy a slot, or the ceiling counts nothing at all", domain, count)
	}
}
