package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"gorm.io/gorm"
)

// admitEveryDomain is the "no rules exist" resolver: it stands in for a rule
// table with nothing in it, which is the only case that legitimately allows.
type admitEveryDomain struct{}

func (admitEveryDomain) CheckAccessFor(context.Context, string, repo.AccessPurpose) error {
	return nil
}

// recordingRules answers with err and remembers exactly what it was asked. The
// purposes it records are the assertion that the gate never consults the
// authenticate (ban) rules — the conflation the Purpose split exists to remove.
type recordingRules struct {
	err      error
	emails   []string
	purposes []repo.AccessPurpose
}

func (r *recordingRules) CheckAccessFor(_ context.Context, email string, purpose repo.AccessPurpose) error {
	r.emails = append(r.emails, email)
	r.purposes = append(r.purposes, purpose)
	return r.err
}

func testLimits() OrgBootstrapLimits {
	return OrgBootstrapLimits{
		MaxTotalOrgs:             1000,
		MaxNewOrgsPerHour:        1000,
		MaxUnprovenOrgsPerDomain: 3,
	}
}

// A gate that was never wired must refuse. The dangerous failure is not a panic
// but the opposite: an unconfigured gate that behaves like an empty rule set.
func TestAdmitRefusesWhenTheGateIsNotConfigured(t *testing.T) {
	ctx := context.Background()
	in := AdmissionInput{Email: "founder@example.test"}

	var missing *AdmissionService
	if _, err := missing.Admit(ctx, nil, in); !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("nil gate: err = %v, want ErrAdmissionNotConfigured", err)
	}

	unwired := NewAdmissionService(nil, testLimits())
	if _, err := unwired.Admit(ctx, nil, in); !errors.Is(err, ErrAdmissionNotConfigured) {
		t.Fatalf("nil resolver: err = %v, want ErrAdmissionNotConfigured", err)
	}

	// A zero ceiling must mean "unconfigured", never "unlimited": a forgotten
	// config key must not silently open the gate it was meant to close.
	noCeiling := NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{
		MaxTotalOrgs:      10,
		MaxNewOrgsPerHour: 10,
	})
	_, err := noCeiling.Admit(ctx, nil, in)
	if err == nil {
		t.Fatal("a zero unproven-orgs-per-domain ceiling was treated as unlimited")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("zero ceiling: err = %v, want a not-configured error", err)
	}
}

// An address that cannot be reduced to an eTLD+1 has no bucket to be counted
// against and no registrar that could ever act on it. It is refused before any
// rule lookup and before any database work: the resolver must not be called.
func TestAdmitRefusesAnAddressWithNoRegistrableDomain(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"nodomain",
		"@example.test",
		"founder@localhost",
		"founder@co.uk",
		"founder@203.0.113.7",
		"founder@",
	}

	for _, email := range cases {
		rules := &recordingRules{}
		gate := NewAdmissionService(rules, testLimits())

		// tx is nil on purpose: reaching the database at all would panic here,
		// which proves the refusal happened before any I/O.
		_, err := gate.Admit(context.Background(), nil, AdmissionInput{Email: email})
		if !errors.Is(err, ErrUnresolvableDomain) {
			t.Fatalf("Admit(%q): err = %v, want ErrUnresolvableDomain", email, err)
		}
		if len(rules.purposes) != 0 {
			t.Fatalf("Admit(%q) consulted the rules for an address it cannot key", email)
		}
	}
}

// The gate evaluates create_org rules and ONLY create_org rules. Consulting the
// authenticate list here would re-create the defect the split removed: a user
// invited into an existing tenant on a blocked domain must still be able to log
// in, and a ban that blocks login must not be the thing that guards org
// creation.
func TestAdmitEvaluatesOnlyTheCreateOrgRules(t *testing.T) {
	rules := &recordingRules{err: repo.ErrAccessDenied}
	gate := NewAdmissionService(rules, testLimits())

	_, err := gate.Admit(context.Background(), nil, AdmissionInput{Email: "Founder+tag@Example.test"})
	if !errors.Is(err, repo.ErrAccessDenied) {
		t.Fatalf("err = %v, want repo.ErrAccessDenied", err)
	}
	if len(rules.purposes) != 1 {
		t.Fatalf("resolver was called %d times, want exactly 1", len(rules.purposes))
	}
	if rules.purposes[0] != repo.AccessPurposeCreateOrg {
		t.Fatalf("purpose = %q, want %q", rules.purposes[0], repo.AccessPurposeCreateOrg)
	}
}

// A database error from the rules layer is a refusal, never an allow. If a
// lookup failure fell through, an unreachable control database would be a
// platform-wide open door.
func TestAdmitPropagatesARuleLookupFailureRatherThanAdmitting(t *testing.T) {
	boom := errors.New("control database is unreachable")
	gate := NewAdmissionService(&recordingRules{err: boom}, testLimits())

	_, err := gate.Admit(context.Background(), nil, AdmissionInput{Email: "founder@example.test"})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the lookup error to propagate", err)
	}
}

// The ceiling key is the eTLD+1, not the substring after '@'. Keying on the
// literal host would hand an attacker unlimited free buckets via
// a@x.evilcorp.com / a@y.evilcorp.com, and would mishandle multi-label suffixes
// like co.uk. Plus-addressing must not create a second identity either.
func TestAdmitKeysTheDecisionOnTheRegistrableDomain(t *testing.T) {
	cases := map[string]string{
		"founder@evilcorp.com":              "evilcorp.com",
		"founder@x.evilcorp.com":            "evilcorp.com",
		"founder+spam@Mail.EvilCorp.co.uk":  "evilcorp.co.uk",
		"  Founder@Deep.Sub.Evilcorp.com  ": "evilcorp.com",
	}

	for email, want := range cases {
		// Deny at the rules step so the decision comes back without needing a
		// transaction; the domain is resolved before the rules are consulted.
		gate := NewAdmissionService(&recordingRules{err: repo.ErrAccessDenied}, testLimits())
		decision, err := gate.Admit(context.Background(), nil, AdmissionInput{Email: email})
		if !errors.Is(err, repo.ErrAccessDenied) {
			t.Fatalf("Admit(%q): err = %v, want repo.ErrAccessDenied", email, err)
		}
		if decision.RegistrableDomain != want {
			t.Fatalf("Admit(%q): registrable domain = %q, want %q", email, decision.RegistrableDomain, want)
		}
	}
}

func admitInTx(t *testing.T, db *gorm.DB, gate *AdmissionService, email string) (AdmissionDecision, error) {
	t.Helper()
	var decision AdmissionDecision
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		decision, err = gate.Admit(context.Background(), tx, AdmissionInput{Email: email})
		return err
	})
	return decision, err
}

func seedOrgs(t *testing.T, db *gorm.DB, domain string, n int, proven bool, createdAt time.Time) {
	t.Helper()
	for i := range n {
		org := control.Organization{
			Name:              "seed",
			RegistrableDomain: &domain,
		}
		if proven {
			at := time.Now().UTC().Add(-time.Hour)
			org.ProvenAt = &at
		}
		if err := db.Create(&org).Error; err != nil {
			t.Fatalf("seed org %d: %v", i, err)
		}
		if err := db.Model(&control.Organization{}).
			Where("id = ?", org.ID).
			UpdateColumn("created_at", createdAt).Error; err != nil {
			t.Fatalf("backdate org %d: %v", i, err)
		}

		// Each seeded organization gets a CONFIRMED owner, because the ceiling
		// counts only those. An organization founded on a merely asserted
		// address is deliberately invisible to it — otherwise anyone could
		// exhaust a company's budget with addresses they cannot read. A seed
		// without this would produce organizations that count as nothing, and
		// every ceiling assertion below would pass vacuously.
		verified := time.Now().UTC()
		owner := control.User{
			Email:           fmt.Sprintf("seed-%s@%s", org.ID, domain),
			FirstName:       "Seed",
			IsActive:        true,
			EmailVerifiedAt: &verified,
		}
		if err := db.Create(&owner).Error; err != nil {
			t.Fatalf("seed owner %d: %v", i, err)
		}
		if err := db.Create(&control.Membership{
			UserID:         owner.ID,
			OrganizationID: org.ID,
			Role:           control.MemberRoleOwner,
		}).Error; err != nil {
			t.Fatalf("seed owner membership %d: %v", i, err)
		}
	}
}

// K unproven organizations from one registrable domain, and the K+1th is
// refused. This is the whole point of the ceiling.
func TestAdmitRefusesTheOrganizationAfterTheCeilingIsReached(t *testing.T) {
	db := bootstrapTestDB(t)
	gate := NewAdmissionService(admitEveryDomain{}, testLimits())

	seedOrgs(t, db, "evilcorp.com", 2, false, time.Now().UTC())
	if _, err := admitInTx(t, db, gate, "third@evilcorp.com"); err != nil {
		t.Fatalf("third organization should still be admitted: %v", err)
	}

	seedOrgs(t, db, "evilcorp.com", 1, false, time.Now().UTC())
	decision, err := admitInTx(t, db, gate, "fourth@evilcorp.com")
	if !errors.Is(err, ErrUnprovenOrgCeiling) {
		t.Fatalf("fourth organization: err = %v, want ErrUnprovenOrgCeiling", err)
	}
	// The denial log line needs the numbers that produced it.
	if decision.UnprovenOrgs != 3 || decision.Ceiling != 3 {
		t.Fatalf("decision = %+v, want UnprovenOrgs 3 and Ceiling 3", decision)
	}
}

// Sub-domains and plus-addressing share the parent's bucket. Under
// substring-after-'@' keying every one of these would get its own free ceiling,
// which is exactly the evasion eTLD+1 keying exists to close.
func TestTheCeilingIsSharedAcrossSubdomainsAndPlusAddresses(t *testing.T) {
	db := bootstrapTestDB(t)
	gate := NewAdmissionService(admitEveryDomain{}, testLimits())

	seedOrgs(t, db, "evilcorp.com", 3, false, time.Now().UTC())

	for _, email := range []string{
		"a@evilcorp.com",
		"a@mail.evilcorp.com",
		"a+one@deep.sub.evilcorp.com",
		"A@EvilCorp.com",
	} {
		if _, err := admitInTx(t, db, gate, email); !errors.Is(err, ErrUnprovenOrgCeiling) {
			t.Fatalf("Admit(%q): err = %v, want ErrUnprovenOrgCeiling", email, err)
		}
	}

	// A genuinely different registrable domain keeps its own capacity.
	if _, err := admitInTx(t, db, gate, "a@goodcorp.example"); err != nil {
		t.Fatalf("an unrelated domain was refused: %v", err)
	}
}

// Proof, not patience, is what frees capacity. An organization with proven_at
// set does not occupy a slot.
func TestProvenOrganizationsDoNotConsumeTheCeiling(t *testing.T) {
	db := bootstrapTestDB(t)
	gate := NewAdmissionService(admitEveryDomain{}, testLimits())

	seedOrgs(t, db, "realcorp.example", 5, true, time.Now().UTC())
	if _, err := admitInTx(t, db, gate, "a@realcorp.example"); err != nil {
		t.Fatalf("proven organizations consumed the ceiling: %v", err)
	}
}

// The ceiling has NO time component. An attacker pacing one registration every
// two minutes defeats every fixed window, sliding window and token bucket; here
// year-old organizations still hold their slots.
func TestTheCeilingHasNoTimeComponent(t *testing.T) {
	db := bootstrapTestDB(t)
	gate := NewAdmissionService(admitEveryDomain{}, testLimits())

	longAgo := time.Now().UTC().AddDate(-2, 0, 0)
	seedOrgs(t, db, "patient.example", 3, false, longAgo)

	if _, err := admitInTx(t, db, gate, "a@patient.example"); !errors.Is(err, ErrUnprovenOrgCeiling) {
		t.Fatalf("err = %v, want ErrUnprovenOrgCeiling — waiting restored capacity", err)
	}
}

// The platform quotas that previously guarded only the social bootstrap path
// are now enforced by the gate itself, so both org-creation paths get them.
func TestAdmitEnforcesThePlatformQuotas(t *testing.T) {
	db := bootstrapTestDB(t)

	seedOrgs(t, db, "other.example", 2, false, time.Now().UTC())

	full := NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{
		MaxTotalOrgs:             2,
		MaxNewOrgsPerHour:        1000,
		MaxUnprovenOrgsPerDomain: 100,
	})
	if _, err := admitInTx(t, db, full, "a@fresh.example"); !errors.Is(err, ErrOrgQuotaExceeded) {
		t.Fatalf("err = %v, want ErrOrgQuotaExceeded", err)
	}

	hourly := NewAdmissionService(admitEveryDomain{}, OrgBootstrapLimits{
		MaxTotalOrgs:             1000,
		MaxNewOrgsPerHour:        2,
		MaxUnprovenOrgsPerDomain: 100,
	})
	if _, err := admitInTx(t, db, hourly, "a@fresh.example"); !errors.Is(err, ErrOrgCreationRateExceeded) {
		t.Fatalf("err = %v, want ErrOrgCreationRateExceeded", err)
	}
}
