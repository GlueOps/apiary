package repo

import (
	"context"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PublicEmailDomains is the corpus that used to live as a hardcoded Go map in
// social_repo.go, where only EnsureAutoMembership consulted it and neither
// org-creation path did. It is seeded into auth_access_rules so that adding a
// new disposable-mail provider is an INSERT, not a deploy.
var PublicEmailDomains = []string{
	"gmail.com", "googlemail.com", "yahoo.com", "yahoo.co.uk",
	"outlook.com", "hotmail.com", "hotmail.co.uk", "live.com",
	"msn.com", "icloud.com", "me.com", "mac.com", "aol.com",
	"proton.me", "protonmail.com", "pm.me", "gmx.com", "gmx.net",
	"mail.com", "yandex.com", "zoho.com", "fastmail.com", "hey.com",
	"qq.com", "163.com", "126.com", "duck.com", "tutanota.com",
	"tuta.io", "mailinator.com", "example.com", "example.org",
}

const publicEmailDomainSeedReason = "seeded: public/disposable mailbox provider; domain ownership implies no tenant"

// SeedPublicEmailDomainRules installs the corpus as create_org denies. It is
// idempotent and NEVER updates an existing row: an operator who wants to permit
// a seeded domain UPDATEs that row's policy to 'allow', which survives every
// subsequent boot. A DELETE does not — it is re-inserted on the next start.
//
// The conflict target is the full (kind, value, purpose) triple, which is why
// the unique index had to grow the purpose column: with the old two-column
// index an operator's existing (domain, gmail.com, authenticate) row would make
// the create_org row unwritable and DO NOTHING would skip it silently, leaving
// the gate open on exactly the domain the operator was worried about.
func SeedPublicEmailDomainRules(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, gorm.ErrInvalidDB
	}

	reason := publicEmailDomainSeedReason

	rows := make([]control.AuthAccessRule, 0, len(PublicEmailDomains))
	for _, domain := range PublicEmailDomains {
		rows = append(rows, control.AuthAccessRule{
			Kind:    control.AccessRuleKindDomain,
			Value:   domain,
			Purpose: control.AccessRulePurposeCreateOrg,
			Policy:  control.AccessRulePolicyDeny,
			Reason:  &reason,
		})
	}

	res := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "kind"},
			{Name: "value"},
			{Name: "purpose"},
		},
		DoNothing: true,
	}).CreateInBatches(rows, 100)

	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// publicEmailDomainSet is the corpus keyed for lookup. Built once at init from
// the slice above, which stays the single source of truth so adding a provider
// remains a one-line edit.
var publicEmailDomainSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(PublicEmailDomains))
	for _, d := range PublicEmailDomains {
		set[d] = struct{}{}
	}
	return set
}()

// IsSeededRule reports whether a rule carries the marker SeedPublicEmailDomainRules
// writes, and nothing else does.
//
// Exported so callers can ask the question without holding the string. A UI that
// string-matched the reason would silently start reporting every row as
// operator-authored the day the wording changed, and the reason column is free
// text an operator can overwrite — which is exactly why this is NOT the
// predicate for whether a delete survives. Use SeedCorpusCovers for that.
func IsSeededRule(rule control.AuthAccessRule) bool {
	return rule.Reason != nil && *rule.Reason == publicEmailDomainSeedReason
}

// SeedCorpusCovers reports whether the next boot's seeding pass would write this
// rule's (kind, value, purpose) triple.
//
// That triple is the seeder's conflict target, so this — not the reason string —
// decides what happens to an operator's edit at restart. A row it covers cannot
// be deleted durably: DO NOTHING skips the re-insert only while the row exists,
// so deleting one restores it as a create_org DENY on the next start and
// reverses the operator's intent with no signal. An UPDATE to 'allow' survives,
// because the seeder never updates.
func SeedCorpusCovers(rule control.AuthAccessRule) bool {
	if rule.Kind != control.AccessRuleKindDomain || rule.Purpose != control.AccessRulePurposeCreateOrg {
		return false
	}
	_, ok := publicEmailDomainSet[strings.ToLower(strings.TrimSpace(rule.Value))]
	return ok
}
