package accessrule

import (
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

// seededRule is a row exactly as SeedPublicEmailDomainRules writes it.
//
// The reason is spelled out once, here, and every assertion below reaches it
// through repo.IsSeededRule rather than comparing strings itself. So a change to
// the seeder's wording fails this one line loudly instead of quietly turning
// every seeded row in the API into an operator-authored one.
func seededRule() control.AuthAccessRule {
	reason := "seeded: public/disposable mailbox provider; domain ownership implies no tenant"
	return control.AuthAccessRule{
		Kind:    control.AccessRuleKindDomain,
		Value:   repo.PublicEmailDomains[0],
		Purpose: control.AccessRulePurposeCreateOrg,
		Policy:  control.AccessRulePolicyDeny,
		Reason:  &reason,
	}
}

// The fixture above is only meaningful if repo agrees it is a seeded row.
func TestFixtureMatchesTheSeederThisPackageReportsOn(t *testing.T) {
	if !repo.IsSeededRule(seededRule()) {
		t.Fatal("repo.IsSeededRule rejects this fixture: the seeder's reason string has changed and every test below is now asserting nothing")
	}
	if !repo.SeedCorpusCovers(seededRule()) {
		t.Fatal("repo.SeedCorpusCovers rejects this fixture: the corpus or the triple the seeder writes has changed")
	}
}

func TestSeededRuleIsMarkedAndNotFlaggedAsOverridden(t *testing.T) {
	v := toView(seededRule(), true)

	if !v.Seeded {
		t.Fatal("a rule carrying the seed reason must be marked seeded, or the page cannot tell platform-shipped policy from an operator's own")
	}
	if v.SeedOverridden {
		t.Error("a seeded deny left as a deny has not been overridden")
	}
	if !v.SeedRestoresOnRestart {
		t.Error("a corpus row must warn that deleting it does not last")
	}
}

// The whole surface exists for this row: a deny the platform ships that somebody
// has turned into an allow, permanently, with nothing else on the installation
// mentioning it.
func TestOperatorOverrideOfASeededDenyIsSurfaced(t *testing.T) {
	rule := seededRule()
	rule.Policy = control.AccessRulePolicyAllow

	v := toView(rule, true)
	if !v.SeedOverridden {
		t.Fatal("a seeded domain now set to allow is the divergence this endpoint exists to report")
	}
}

// Editing the reason is the obvious way to lose the marking, and it must not
// lose the override: restoration keys on (kind, value, purpose), never on the
// free-text column.
func TestOverrideSurvivesAnEditedReason(t *testing.T) {
	rule := seededRule()
	rule.Policy = control.AccessRulePolicyAllow
	custom := "approved by legal, 2026-08"
	rule.Reason = &custom

	v := toView(rule, true)
	if v.Seeded {
		t.Error("the seed marker is gone, so seeded must read false")
	}
	if !v.SeedOverridden {
		t.Fatal("rewriting the reason must not hide that this is a seeded domain now set to allow")
	}
	if !v.SeedRestoresOnRestart {
		t.Fatal("the seeder re-inserts on (kind, value, purpose); an edited reason does not exempt the row")
	}
}

func TestOperatorAuthoredRuleIsNotMarked(t *testing.T) {
	reason := "customer contract"
	v := toView(control.AuthAccessRule{
		Kind:    control.AccessRuleKindDomain,
		Value:   "notinthecorpus.example",
		Purpose: control.AccessRulePurposeCreateOrg,
		Policy:  control.AccessRulePolicyDeny,
		Reason:  &reason,
	}, true)

	if v.Seeded || v.SeedOverridden || v.SeedRestoresOnRestart {
		t.Fatalf("an operator's own rule must carry no seed marking: %+v", v)
	}
}

// A corpus domain under a DIFFERENT purpose is a different rule. The seeder
// writes create_org only, so an authenticate rule on gmail.com is the operator's
// and deleting it sticks.
func TestCorpusDomainUnderAnotherPurposeIsNotSeeded(t *testing.T) {
	rule := seededRule()
	rule.Purpose = control.AccessRulePurposeAuthenticate
	rule.Reason = nil

	v := toView(rule, true)
	if v.SeedRestoresOnRestart {
		t.Fatal("the seed pass writes create_org rows only — warning about an authenticate row tells the operator something untrue")
	}
}

// With auth.seed_public_domain_rules off, nothing is re-inserted and a delete
// finally sticks. Warning anyway would train operators to ignore the warning.
func TestNoRestoreWarningWhenSeedingIsDisabled(t *testing.T) {
	v := toView(seededRule(), false)

	if v.SeedRestoresOnRestart {
		t.Fatal("seeding is switched off, so this delete survives the next boot")
	}
	if !v.Seeded {
		t.Error("the row was still written by the seed pass, whether or not the pass still runs")
	}
}
