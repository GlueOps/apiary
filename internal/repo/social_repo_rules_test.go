package repo

import (
	"context"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
)

func TestEnsureAutoMembership_ConsultsCreateOrgRulesNotAHardcodedMap(t *testing.T) {
	// The old implementation compared the domain against a Go map, so a domain
	// added by an operator could not possibly change the outcome. Adding a
	// create_org deny row must now be enough — an INSERT, not a deploy.
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "disposable")
	org := mustCreateOrg(t, db, domain, true)

	email := uuid.NewString() + "@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	if err := r.EnsureAutoMembership(ctx, user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("auto-join fired for a domain denied org creation (%d rows)", n)
	}
}

func TestEnsureAutoMembership_SubdomainOfADeniedRegistrableDomain(t *testing.T) {
	// Same evasion the resolver closes for the gate: a deny on the registrable
	// domain must cover addresses at any host under it.
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	apex := uniqueDomain(t, "evilauto")
	sub := "mail." + apex
	org := mustCreateOrg(t, db, sub, true)

	email := uuid.NewString() + "@" + sub
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	mustRule(t, db, control.AccessRuleKindDomain, apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	if err := r.EnsureAutoMembership(ctx, user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("auto-join fired at mail.%s despite a deny on %s (%d rows)", apex, apex, n)
	}
}

func TestEnsureAutoMembership_IgnoresAuthenticatePurposeRules(t *testing.T) {
	// A ban-list entry is about tokens, not about which corporate domain a
	// verified account may be auto-joined into. Reading the wrong purpose here
	// would be the same conflation the split removes.
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "authrule")
	org := mustCreateOrg(t, db, domain, true)

	email := uuid.NewString() + "@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)

	if err := r.EnsureAutoMembership(ctx, user.ID, email); err != nil {
		t.Fatalf("EnsureAutoMembership: %v", err)
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 1 {
		t.Fatalf("an authenticate-purpose rule suppressed the auto-join (%d membership rows, want 1)", n)
	}
}

func TestEnsureAutoMembership_RefusesAddressesThatCannotBeKeyed(t *testing.T) {
	// No '@' at all, and a leading '@' — the latter previously reached the
	// domain lookup with an empty local part.
	db := socialTestDB(t)
	r := newSocialRepo(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "unkeyable")
	org := mustCreateOrg(t, db, domain, true)

	email := uuid.NewString() + "@" + domain
	user := mustCreateUser(t, db, email)
	mustCreateAccount(t, db, user.ID, "google", "s-"+uuid.NewString(), email, true)

	for _, bad := range []string{"", "nodomain", "@" + domain, "user@"} {
		if err := r.EnsureAutoMembership(ctx, user.ID, bad); err != nil {
			t.Fatalf("EnsureAutoMembership(%q): %v", bad, err)
		}
	}
	if n := membershipCount(t, db, org.ID, user.ID); n != 0 {
		t.Fatalf("membership granted from an address that cannot be keyed (%d rows)", n)
	}
}
