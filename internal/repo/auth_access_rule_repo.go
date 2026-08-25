package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrAccessDenied = errors.New("access denied by policy")

var ErrInvalidRulePurpose = errors.New(`auth access rule: purpose must be "authenticate" or "create_org"`)

var ErrInvalidRuleValue = errors.New("access rule value is not usable")

var ErrInvalidRuleKind = errors.New(`auth access rule: kind must be "email" or "domain"`)

// AccessPurpose is typed rather than a bare string so a typo is a compile
// error instead of a rule that silently never matches.
type AccessPurpose string

const (
	AccessPurposeAuthenticate AccessPurpose = control.AccessRulePurposeAuthenticate
	AccessPurposeCreateOrg    AccessPurpose = control.AccessRulePurposeCreateOrg
	AccessPurposeJoinOrg      AccessPurpose = control.AccessRulePurposeJoinOrg
)

func AccessPurposeIsValid(p AccessPurpose) bool {
	return p == AccessPurposeAuthenticate || p == AccessPurposeCreateOrg || p == AccessPurposeJoinOrg
}

func AccessRuleKindIsValid(k string) bool {
	return k == control.AccessRuleKindEmail || k == control.AccessRuleKindDomain
}

type AuthAccessRuleRepo interface {
	// There is deliberately NO purposeless CheckAccess on this interface. A
	// caller that does not say which question it is asking is the exact defect
	// the Purpose split exists to remove: one rule set answering both "is this
	// identity banned?" and "may this domain mint tenants?" at a single call
	// site. Every caller must name its purpose.
	//
	// CheckAuthentication is the ban list and nothing else — a create_org rule
	// must never block a login, or a user invited into an existing tenant on a
	// blocked domain cannot reach the tenant they were invited into.
	CheckAuthentication(ctx context.Context, email string) error
	CheckOrgCreation(ctx context.Context, email string) error
	CheckAccessFor(ctx context.Context, email string, purpose AccessPurpose) error
	List(ctx context.Context) ([]control.AuthAccessRule, error)
	Create(ctx context.Context, rule *control.AuthAccessRule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AuthAccessRuleRepoGorm struct {
	db *gorm.DB
}

func NewAuthAccessRuleRepoGorm(db *gorm.DB) AuthAccessRuleRepo {
	return &AuthAccessRuleRepoGorm{db: db}
}

func (r *AuthAccessRuleRepoGorm) CheckAuthentication(ctx context.Context, email string) error {
	return r.CheckAccessFor(ctx, email, AccessPurposeAuthenticate)
}

func (r *AuthAccessRuleRepoGorm) CheckOrgCreation(ctx context.Context, email string) error {
	return r.CheckAccessFor(ctx, email, AccessPurposeCreateOrg)
}

func (r *AuthAccessRuleRepoGorm) CheckAccessFor(ctx context.Context, email string, purpose AccessPurpose) error {
	id, err := NormalizeEmailIdentity(email)
	if err != nil {
		// An address we cannot key cannot be matched against any rule, so no
		// security decision is reachable. House rule: DENY, do not fall through.
		return ErrAccessDenied
	}
	return EvaluateAccessRules(ctx, r.db, string(purpose), id)
}

// EvaluateAccessRules is the single implementation behind every resolver, so
// the email-beats-domain precedence cannot drift between purposes. It takes a
// *gorm.DB so a caller holding an open transaction (EnsureAutoMembership, the
// admission gate under its advisory lock) can decide on that same transaction.
func EvaluateAccessRules(ctx context.Context, db *gorm.DB, purpose string, id EmailIdentity) error {
	if db == nil {
		return ErrAccessDenied
	}
	if !AccessPurposeIsValid(AccessPurpose(purpose)) {
		// A purpose no resolver recognises would match no rows and therefore
		// allow everything. Refuse instead.
		return ErrAccessDenied
	}
	if id.Email == "" || id.Domain == "" {
		return ErrAccessDenied
	}

	var rule control.AuthAccessRule

	// The purpose predicate MUST stay inside the same Where string as the
	// parenthesised OR group. Chaining a second .Where relies on GORM's
	// buildExprs heuristic to parenthesise the OR; if that ever stops firing the
	// predicate becomes `purpose = X AND (email match) OR (domain match)`, which
	// lets a domain rule of ANY purpose decide authentication — exactly the
	// conflation the purpose split exists to remove.
	err := db.WithContext(ctx).
		Where("purpose = ? AND ((kind = ? AND value = ?) OR (kind = ? AND value IN ?))",
			purpose,
			control.AccessRuleKindEmail, id.Email,
			control.AccessRuleKindDomain, id.DomainCandidates()).
		// Both Order arguments must be plain strings. gorm v1.30.1's DB.Order
		// type-switches on clause.OrderBy / clause.OrderByColumn / string and
		// SILENTLY DISCARDS anything else, so a gorm.Expr here would drop the
		// ORDER BY with no error and collapse precedence to the row order.
		Order("CASE kind WHEN 'email' THEN 0 ELSE 1 END").
		// The domain candidates are {literal domain, eTLD+1} and the eTLD+1 is
		// always a suffix of the literal domain, hence never longer. Longest
		// first is therefore exactly most-specific first, with no bound
		// parameter needed.
		Order("LENGTH(value) DESC").
		First(&rule).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		// A database error is a hard failure, never an allow.
		return err
	}

	// Anything that is not exactly "allow" denies. A row whose policy column
	// holds "DENY", "block" or "" must not read as permission.
	if rule.Policy != control.AccessRulePolicyAllow {
		return ErrAccessDenied
	}
	return nil
}

func (r *AuthAccessRuleRepoGorm) List(ctx context.Context) ([]control.AuthAccessRule, error) {
	var rules []control.AuthAccessRule
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *AuthAccessRuleRepoGorm) Create(ctx context.Context, rule *control.AuthAccessRule) error {
	if rule == nil {
		return ErrInvalidRuleKind
	}
	// GORM omits a Go zero-value string for a column carrying a `default:` tag,
	// so a caller that forgets Purpose silently gets 'authenticate' and their
	// intended org-creation rule is a no-op. Refuse rather than guess.
	if !AccessPurposeIsValid(AccessPurpose(rule.Purpose)) {
		return ErrInvalidRulePurpose
	}
	if !AccessRuleKindIsValid(rule.Kind) {
		return ErrInvalidRuleKind
	}

	// Normalise Value the same way the lookup normalises the address it is
	// compared against. The evaluator binds a lowercased, plus-stripped
	// identity, so a rule stored as 'EvilCorp.example' or 'abuser+spam@x.test'
	// can never match anything — and it still appears in List() as active
	// policy, so an operator sees a deny they believe is in force while the
	// domain mints tenants. Validating Kind and Purpose but not Value left
	// exactly that gap.
	switch rule.Kind {
	case control.AccessRuleKindEmail:
		id, err := NormalizeEmailIdentity(rule.Value)
		if err != nil {
			return fmt.Errorf("%w: %q is not a usable email rule value", ErrInvalidRuleValue, rule.Value)
		}
		rule.Value = id.Email
	case control.AccessRuleKindDomain:
		domain := strings.ToLower(strings.TrimSpace(rule.Value))
		if domain == "" || strings.ContainsAny(domain, "@ ") {
			return fmt.Errorf("%w: %q is not a usable domain rule value", ErrInvalidRuleValue, rule.Value)
		}
		// Validated by round-tripping through the SAME normaliser the evaluator
		// uses, because "looks like a domain" and "can ever match" are different
		// questions and only the second one matters.
		//
		// EvaluateAccessRules compares against DomainCandidates(), which yields
		// the literal domain and its eTLD+1. A value that normalisation rejects —
		// "com", "notadomain", "..foo" — is not merely odd: it is a row that
		// stores cleanly, lists as active policy, and can never fire. An operator
		// reading it believes a domain is blocked while every address on it signs
		// in, which is worse than having written no rule at all.
		id, nerr := NormalizeEmailIdentity("probe@" + domain)
		if nerr != nil || id.Domain != domain {
			return fmt.Errorf("%w: %q can never match an address — a domain rule is compared "+
				"against a real hostname and its registrable domain, so this row would list as "+
				"policy while deciding nothing", ErrInvalidRuleValue, rule.Value)
		}
		rule.Value = domain
	}

	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *AuthAccessRuleRepoGorm) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&control.AuthAccessRule{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var ErrInvalidRulePolicy = errors.New(`auth access rule: policy must be "allow" or "deny"`)

// AuthAccessRuleAdminRepo is the AUTHORING surface, deliberately separate from
// AuthAccessRuleRepo above, which is the EVALUATION surface.
//
// The split is the point. Login and org creation hold the evaluation interface
// and can only ask questions; nothing on an authentication path is handed a
// handle that can rewrite the rules it is about to consult. Keeping Get and
// UpdatePolicy off that interface is what makes that true by construction
// rather than by convention.
type AuthAccessRuleAdminRepo interface {
	List(ctx context.Context) ([]control.AuthAccessRule, error)
	Get(ctx context.Context, id uuid.UUID) (*control.AuthAccessRule, error)
	Create(ctx context.Context, rule *control.AuthAccessRule) error
	UpdatePolicy(ctx context.Context, id uuid.UUID, policy string, reason *string) (*control.AuthAccessRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewAuthAccessRuleAdminRepoGorm(db *gorm.DB) AuthAccessRuleAdminRepo {
	return &AuthAccessRuleRepoGorm{db: db}
}

func (r *AuthAccessRuleRepoGorm) Get(ctx context.Context, id uuid.UUID) (*control.AuthAccessRule, error) {
	var rule control.AuthAccessRule
	if err := r.db.WithContext(ctx).First(&rule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdatePolicy changes the decision a rule carries, and nothing else.
//
// Kind, Value and Purpose are deliberately not updatable. They are the identity
// of the rule — the unique index is on that triple, and it is the seeder's
// conflict target — so editing one in place is not a correction, it is a
// different rule wearing the audit trail of the old one. Retargeting means
// deleting and creating, which leaves both facts visible.
//
// A nil reason CLEARS the column. That needs the map form: Updates on a struct
// skips zero values, so a struct-shaped clear would silently leave the old
// reason standing next to a policy that no longer matches it.
func (r *AuthAccessRuleRepoGorm) UpdatePolicy(ctx context.Context, id uuid.UUID, policy string, reason *string) (*control.AuthAccessRule, error) {
	if policy != control.AccessRulePolicyAllow && policy != control.AccessRulePolicyDeny {
		return nil, ErrInvalidRulePolicy
	}

	var rule control.AuthAccessRule
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Read first so a missing row is ErrRecordNotFound rather than a silent
		// zero-row update reported as success.
		if err := tx.First(&rule, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Model(&control.AuthAccessRule{}).Where("id = ?", id).
			Updates(map[string]any{"policy": policy, "reason": reason}).Error; err != nil {
			return err
		}
		return tx.First(&rule, "id = ?", id).Error
	})
	if err != nil {
		return nil, err
	}
	return &rule, nil
}
