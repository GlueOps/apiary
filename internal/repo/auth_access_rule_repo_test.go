package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var accessRuleTestSchema = testsupport.SchemaName("apiary_repo_access_rule_test")

func accessRuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, accessRuleTestSchema, &control.AuthAccessRule{})
}

func uniqueDomain(t *testing.T, prefix string) string {
	t.Helper()
	return prefix + "-" + strings.ReplaceAll(uuid.NewString(), "-", "") + ".test"
}

func mustRule(t *testing.T, db *gorm.DB, kind, value, purpose, policy string) control.AuthAccessRule {
	t.Helper()
	rule := control.AuthAccessRule{
		Kind:    kind,
		Value:   value,
		Purpose: purpose,
		Policy:  policy,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create rule (%s %s %s %s): %v", kind, value, purpose, policy, err)
	}
	t.Cleanup(func() { db.Delete(&control.AuthAccessRule{}, "id = ?", rule.ID) })
	return rule
}

func TestCheckAccess_EmailRuleBeatsDomainRuleWithinAPurpose(t *testing.T) {
	// The precedence that already existed must survive the purpose split: an
	// exact-email allow overrides a domain deny at the same purpose.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)

	domain := uniqueDomain(t, "prec")
	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)
	mustRule(t, db, control.AccessRuleKindEmail, "vip@"+domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyAllow)

	if err := r.CheckAuthentication(context.Background(), "vip@"+domain); err != nil {
		t.Fatalf("the exact-email allow lost to the domain deny: %v", err)
	}
	if err := r.CheckAuthentication(context.Background(), "other@"+domain); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("want ErrAccessDenied for an address with no email rule, got %v", err)
	}
}

func TestCheckAccess_PurposesAreIndependent(t *testing.T) {
	// The core claim of the purpose split. A create_org deny must not block
	// authentication (an invited user on a blocked domain has to be able to log
	// in), and an authenticate deny must not be what stops org creation.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	orgOnly := uniqueDomain(t, "orgonly")
	mustRule(t, db, control.AccessRuleKindDomain, orgOnly, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	if err := r.CheckAuthentication(ctx, "invited@"+orgOnly); err != nil {
		t.Fatalf("a create_org deny blocked authentication: %v — an invited user on a blocked domain must still be able to sign in", err)
	}
	if err := r.CheckOrgCreation(ctx, "invited@"+orgOnly); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("the create_org deny did not refuse org creation, got %v", err)
	}

	authOnly := uniqueDomain(t, "authonly")
	mustRule(t, db, control.AccessRuleKindDomain, authOnly, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)

	if err := r.CheckOrgCreation(ctx, "banned@"+authOnly); err != nil {
		t.Fatalf("an authenticate deny leaked into the org-creation decision: %v", err)
	}
	if err := r.CheckAuthentication(ctx, "banned@"+authOnly); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("the authenticate deny did not refuse the token, got %v", err)
	}
}

func TestCheckAccess_SameKindAndValueCoexistAcrossPurposes(t *testing.T) {
	// If the unique index were still (kind, value) the second row would be
	// rejected, and SeedPublicEmailDomainRules' ON CONFLICT DO NOTHING would
	// skip it silently — leaving org creation open on the very domain an
	// operator had already denied for authentication.
	db := accessRuleTestDB(t)
	domain := uniqueDomain(t, "bothpurposes")

	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)
	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	var n int64
	if err := db.Model(&control.AuthAccessRule{}).Where("value = ?", domain).Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Fatalf("want 2 rows for the same (kind, value) under different purposes, got %d", n)
	}
}

func TestCheckAccess_DomainDenyCoversSubdomainsViaRegistrableDomain(t *testing.T) {
	// Without the eTLD+1 tier a deny on evilcorp.com is defeated by
	// a@mail.evilcorp.com, which is the exact hole registrable-domain keying
	// exists to close.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	apex := uniqueDomain(t, "evilcorp")
	mustRule(t, db, control.AccessRuleKindDomain, apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	if err := r.CheckOrgCreation(ctx, "a@mail."+apex); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a deny on %s did not cover a@mail.%s, got %v", apex, apex, err)
	}
	if err := r.CheckOrgCreation(ctx, "a@"+apex); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a deny on %s did not cover the apex itself, got %v", apex, err)
	}
}

func TestCheckAccess_SubdomainDenyDoesNotWidenToSiblings(t *testing.T) {
	// The tier only ever widens downward. A deny written against one subdomain
	// must not silently ban the whole registrable domain.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	apex := uniqueDomain(t, "sibling")
	mustRule(t, db, control.AccessRuleKindDomain, "mail."+apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)

	if err := r.CheckOrgCreation(ctx, "a@mail."+apex); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("the literal-subdomain deny did not fire, got %v", err)
	}
	if err := r.CheckOrgCreation(ctx, "a@other."+apex); err != nil {
		t.Fatalf("a deny on mail.%s wrongly denied other.%s: %v", apex, apex, err)
	}
	if err := r.CheckOrgCreation(ctx, "a@"+apex); err != nil {
		t.Fatalf("a deny on mail.%s wrongly denied the apex: %v", apex, err)
	}
}

func TestCheckAccess_LiteralDomainRuleBeatsRegistrableDomainRule(t *testing.T) {
	// Most specific wins among domain rules too: an allow on the exact host
	// must override a deny on its registrable domain.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	apex := uniqueDomain(t, "specific")
	mustRule(t, db, control.AccessRuleKindDomain, apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyDeny)
	mustRule(t, db, control.AccessRuleKindDomain, "eng."+apex, control.AccessRulePurposeCreateOrg, control.AccessRulePolicyAllow)

	if err := r.CheckOrgCreation(ctx, "a@eng."+apex); err != nil {
		t.Fatalf("the literal-host allow lost to the registrable-domain deny: %v", err)
	}
	if err := r.CheckOrgCreation(ctx, "a@sales."+apex); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("the registrable-domain deny stopped applying to other hosts, got %v", err)
	}
}

func TestCheckAccess_PlusAddressingIsDecidedByTheBareAddress(t *testing.T) {
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	domain := uniqueDomain(t, "plusaddr")
	mustRule(t, db, control.AccessRuleKindDomain, domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)
	mustRule(t, db, control.AccessRuleKindEmail, "a@"+domain, control.AccessRulePurposeAuthenticate, control.AccessRulePolicyAllow)

	// Without plus-stripping this address would miss the email rule and fall to
	// the domain deny; with it the operator's per-address decision holds.
	if err := r.CheckAuthentication(ctx, "A+newsletter@"+strings.ToUpper(domain)); err != nil {
		t.Fatalf("a+tag lost the exact-email rule written for a@: %v", err)
	}
}

func TestCheckAccess_DeniesAddressesThatCannotBeKeyed(t *testing.T) {
	// The old CheckAccess returned nil (allow) for all of these.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	for _, email := range []string{"", "nodomain", "user@", "@evilcorp.com", "a@localhost", "a@co.uk", "a@1.2.3.4"} {
		if err := r.CheckAuthentication(ctx, email); !errors.Is(err, ErrAccessDenied) {
			t.Errorf("CheckAuthentication(%q) = %v, want ErrAccessDenied — an address no rule can key must not be allowed", email, err)
		}
		if err := r.CheckOrgCreation(ctx, email); !errors.Is(err, ErrAccessDenied) {
			t.Errorf("CheckOrgCreation(%q) = %v, want ErrAccessDenied", email, err)
		}
	}
}

func TestEvaluateAccessRules_DeniesAnUnknownPurpose(t *testing.T) {
	// A purpose no resolver recognises matches no rows, so a naive
	// implementation would allow everything. It must refuse instead.
	db := accessRuleTestDB(t)

	id, err := NormalizeEmailIdentity("a@" + uniqueDomain(t, "unknownpurpose"))
	if err != nil {
		t.Fatalf("normalise: %v", err)
	}
	if err := EvaluateAccessRules(context.Background(), db, "creat_org", id); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a typo'd purpose was treated as allow: %v", err)
	}
	if err := EvaluateAccessRules(context.Background(), nil, control.AccessRulePurposeCreateOrg, id); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("a nil database handle was treated as allow: %v", err)
	}
}

func TestAuthAccessRules_DatabaseRefusesUnenforceableRows(t *testing.T) {
	// A row with an unrecognised purpose or policy is a rule the operator
	// believes they wrote and that nothing ever enforces. Make it unwritable.
	db := accessRuleTestDB(t)
	domain := uniqueDomain(t, "constraint")

	if err := db.Exec(
		"INSERT INTO auth_access_rules (kind, value, purpose, policy) VALUES (?, ?, ?, ?)",
		control.AccessRuleKindDomain, domain, "creat_org", control.AccessRulePolicyDeny).Error; err == nil {
		db.Exec("DELETE FROM auth_access_rules WHERE value = ?", domain)
		t.Fatalf("a rule with purpose 'creat_org' was accepted; it would match no resolver and silently enforce nothing")
	}

	if err := db.Exec(
		"INSERT INTO auth_access_rules (kind, value, purpose, policy) VALUES (?, ?, ?, ?)",
		control.AccessRuleKindDomain, domain, control.AccessRulePurposeCreateOrg, "DENY").Error; err == nil {
		db.Exec("DELETE FROM auth_access_rules WHERE value = ?", domain)
		t.Fatalf("a rule with policy 'DENY' was accepted; policy is compared byte-for-byte")
	}
}

func TestEvaluateAccessRules_TreatsAnythingButAllowAsDeny(t *testing.T) {
	// Belt and braces behind the CHECK constraint: if a row ever reaches the
	// table with a policy that is not exactly "allow", it must not read as
	// permission. The constraint is dropped for the length of this test only.
	db := accessRuleTestDB(t)
	ctx := context.Background()

	// auth_access_rules lives in the default search_path, which internal/jobs
	// also AutoMigrates. Dropping the constraint on the shared connection left a
	// window in which a concurrent AutoMigrate re-added it under this test's
	// feet, and the insert below then failed on the very constraint this test
	// removed. Do the whole thing in one transaction instead: the ALTER takes an
	// ACCESS EXCLUSIVE lock, so no other session ever observes the table without
	// its constraint, and the rollback restores it atomically -- no cleanup that
	// can itself race with someone else's migrator.
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })

	if err := tx.Exec("ALTER TABLE auth_access_rules DROP CONSTRAINT IF EXISTS chk_auth_access_rules_policy").Error; err != nil {
		t.Fatalf("drop policy constraint: %v", err)
	}

	for _, policy := range []string{"DENY", "Allow", "block", ""} {
		domain := uniqueDomain(t, "sloppy")
		if err := tx.Exec(
			"INSERT INTO auth_access_rules (kind, value, purpose, policy) VALUES (?, ?, ?, ?)",
			control.AccessRuleKindDomain, domain, control.AccessRulePurposeAuthenticate, policy).Error; err != nil {
			t.Fatalf("insert policy %q: %v", policy, err)
		}

		id, err := NormalizeEmailIdentity("a@" + domain)
		if err != nil {
			t.Fatalf("normalise: %v", err)
		}
		err = EvaluateAccessRules(ctx, tx, control.AccessRulePurposeAuthenticate, id)
		tx.Exec("DELETE FROM auth_access_rules WHERE value = ?", domain)

		if !errors.Is(err, ErrAccessDenied) {
			t.Errorf("policy %q was treated as permission (%v)", policy, err)
		}
	}
}

func TestAuthAccessRuleRepo_CreateRefusesAnUnsetPurpose(t *testing.T) {
	// GORM omits a zero-value string for a column with a `default:` tag, so a
	// caller who forgets Purpose silently gets 'authenticate' and their
	// org-creation rule never fires.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	rule := &control.AuthAccessRule{
		Kind:   control.AccessRuleKindDomain,
		Value:  uniqueDomain(t, "nopurpose"),
		Policy: control.AccessRulePolicyDeny,
	}
	if err := r.Create(ctx, rule); !errors.Is(err, ErrInvalidRulePurpose) {
		t.Fatalf("Create with no purpose = %v, want ErrInvalidRulePurpose", err)
	}

	rule = &control.AuthAccessRule{
		Kind:    "e-mail",
		Value:   "a@" + uniqueDomain(t, "badkind"),
		Purpose: control.AccessRulePurposeCreateOrg,
		Policy:  control.AccessRulePolicyDeny,
	}
	if err := r.Create(ctx, rule); !errors.Is(err, ErrInvalidRuleKind) {
		t.Fatalf("Create with kind %q = %v, want ErrInvalidRuleKind", rule.Kind, err)
	}
}

func TestSeedPublicEmailDomainRules_IsIdempotentAndNeverOverwrites(t *testing.T) {
	db := accessRuleTestDB(t)
	ctx := context.Background()

	if _, err := SeedPublicEmailDomainRules(ctx, db); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	var seeded int64
	if err := db.Model(&control.AuthAccessRule{}).
		Where("purpose = ? AND kind = ? AND value IN ?",
			control.AccessRulePurposeCreateOrg, control.AccessRuleKindDomain, PublicEmailDomains).
		Count(&seeded).Error; err != nil {
		t.Fatalf("count seeded: %v", err)
	}
	if int(seeded) != len(PublicEmailDomains) {
		t.Fatalf("want %d seeded rows, got %d", len(PublicEmailDomains), seeded)
	}

	n, err := SeedPublicEmailDomainRules(ctx, db)
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if n != 0 {
		t.Fatalf("re-seeding inserted %d rows; the seed must be idempotent", n)
	}

	// The documented escape hatch: an operator who wants to permit a seeded
	// domain UPDATEs the row to 'allow'. DO NOTHING must never clobber that.
	// (A DELETE would be resurrected on the next boot; that is deliberate.)
	if err := db.Model(&control.AuthAccessRule{}).
		Where("kind = ? AND value = ? AND purpose = ?",
			control.AccessRuleKindDomain, "gmail.com", control.AccessRulePurposeCreateOrg).
		Update("policy", control.AccessRulePolicyAllow).Error; err != nil {
		t.Fatalf("hand-edit seeded row: %v", err)
	}
	t.Cleanup(func() {
		db.Model(&control.AuthAccessRule{}).
			Where("kind = ? AND value = ? AND purpose = ?",
				control.AccessRuleKindDomain, "gmail.com", control.AccessRulePurposeCreateOrg).
			Update("policy", control.AccessRulePolicyDeny)
	})

	if _, err := SeedPublicEmailDomainRules(ctx, db); err != nil {
		t.Fatalf("third seed: %v", err)
	}

	var reloaded control.AuthAccessRule
	if err := db.Where("kind = ? AND value = ? AND purpose = ?",
		control.AccessRuleKindDomain, "gmail.com", control.AccessRulePurposeCreateOrg).
		First(&reloaded).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Policy != control.AccessRulePolicyAllow {
		t.Fatalf("re-seeding overwrote an operator's hand-edited policy (now %q)", reloaded.Policy)
	}
}

func TestSeedPublicEmailDomainRules_DoesNotBlockAuthentication(t *testing.T) {
	// Finding 5 moves the corpus into the rules table, but only for create_org.
	// If it leaked into the authenticate purpose every consumer-mailbox user in
	// the system would be locked out on the next boot.
	db := accessRuleTestDB(t)
	r := NewAuthAccessRuleRepoGorm(db)
	ctx := context.Background()

	if _, err := SeedPublicEmailDomainRules(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := r.CheckAuthentication(ctx, "someone@gmail.com"); err != nil {
		t.Fatalf("the seeded corpus blocked authentication for gmail.com: %v", err)
	}
	if err := r.CheckOrgCreation(ctx, "someone@gmail.com"); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("gmail.com was allowed to mint a tenant, got %v", err)
	}
	if err := r.CheckOrgCreation(ctx, "someone+tag@GoogleMail.com"); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("case and plus-addressing evaded the seeded deny, got %v", err)
	}

	var authRows int64
	if err := db.Model(&control.AuthAccessRule{}).
		Where("purpose = ? AND value IN ?", control.AccessRulePurposeAuthenticate, PublicEmailDomains).
		Count(&authRows).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if authRows != 0 {
		t.Fatalf("the seed wrote %d authenticate-purpose rows; it must only ever write create_org", authRows)
	}
}
