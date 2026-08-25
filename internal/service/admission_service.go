package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glueops/apiary/internal/jobs"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"gorm.io/gorm"
)

var (
	// ErrUnprovenOrgCeiling is a 403, deliberately not a 429: the ceiling has no
	// time component, so "try again later" would be a lie. Waiting restores
	// nothing; only an organization becoming PROVEN frees capacity.
	ErrUnprovenOrgCeiling = errors.New("admission: this email domain already holds too many unproven organizations")

	// ErrUnresolvableDomain is a 422. The address cannot be reduced to an
	// eTLD+1, so there is no bucket to count it against and no domain a
	// registrar could ever act on. That is a refusal, never a fallthrough.
	ErrUnresolvableDomain = errors.New("admission: cannot determine a registrable domain for this address")

	// ErrAdmissionNotConfigured is a 500. It exists so that a gate that was
	// never wired denies instead of panicking or, far worse, waving everything
	// through: an unwired admission gate must not look like an empty rule set.
	ErrAdmissionNotConfigured = errors.New("admission: gate is not configured")
)

// AccessRuleResolver is the area boundary between the gate and the rules layer.
// The gate never reads auth_access_rules itself and never re-implements the
// email-beats-domain precedence; if it did, the rule an operator wrote and the
// rule the gate enforced could silently drift apart.
type AccessRuleResolver interface {
	CheckAccessFor(ctx context.Context, email string, purpose repo.AccessPurpose) error
}

// AdmissionInput is the identity asking to create an organization. Registration
// passes the submitted email; org bootstrap passes user.Email.
type AdmissionInput struct {
	Email string
}

// AdmissionDecision is populated on the allow path AND on the ErrUnprovenOrgCeiling
// path, so the denial can be logged with the numbers that produced it.
type AdmissionDecision struct {
	RegistrableDomain string
	UnprovenOrgs      int64
	Ceiling           int
}

// AdmissionService is the single gate in front of both org-creation paths.
// It holds no *gorm.DB: the transaction is handed to Admit, so the gate can be
// constructed at wiring time with dependencies that are not open yet.
type AdmissionService struct {
	rules  AccessRuleResolver
	limits OrgBootstrapLimits
}

func NewAdmissionService(rules AccessRuleResolver, limits OrgBootstrapLimits) *AdmissionService {
	return &AdmissionService{rules: rules, limits: limits}
}

// EnsureConfigured lets a caller refuse before it opens a transaction or reads
// anything. It is nil-receiver safe on purpose: api.Build constructs handlers
// with nil dependencies, and a nil gate must produce an error rather than a
// panic there — and a refusal rather than an allow everywhere else.
func (s *AdmissionService) EnsureConfigured() error {
	if s == nil {
		return ErrAdmissionNotConfigured
	}

	// Every limit is required. A zero that meant "unlimited" would turn a
	// forgotten config key into a silently open gate, which is the exact
	// failure this whole component exists to prevent.
	if s.limits.MaxTotalOrgs <= 0 || s.limits.MaxNewOrgsPerHour <= 0 || s.limits.MaxUnprovenOrgsPerDomain <= 0 {
		return fmt.Errorf("admission: organization quota limits are not configured")
	}
	if s.rules == nil {
		return ErrAdmissionNotConfigured
	}
	return nil
}

// Admit is THE GATE. It MUST be called on the transaction that will create the
// organization, before any row is inserted and before any provisioning job is
// enqueued — creating a Postgres DATABASE and ROLE for a caller who is then
// refused leaves an orphan nothing ever cleans up.
//
// Evaluation order: registrable domain, create_org access rules, the
// unproven-organization ceiling, the total organization quota, the hourly
// creation rate. On success the caller MUST write decision.RegistrableDomain to
// organizations.registrable_domain in this same transaction, or the org it
// creates is invisible to every future ceiling check.
func (s *AdmissionService) Admit(ctx context.Context, tx *gorm.DB, in AdmissionInput) (AdmissionDecision, error) {
	var decision AdmissionDecision

	if err := s.EnsureConfigured(); err != nil {
		return decision, err
	}

	// Resolved first, before any I/O, and through the SAME normalisation the
	// rules resolver uses — otherwise the rule an operator wrote and the bucket
	// this gate counts could be keyed differently. The registrable domain is the
	// ceiling key, so an address that cannot be reduced to an eTLD+1 has no
	// bucket at all and must be refused rather than counted into whatever bucket
	// the empty string names.
	//
	// (identity.ICANNSuffix is false when the eTLD+1 came from the PSL's private
	// section, e.g. a.blogspot.com. Such hosts each get their own bucket, which
	// is cheap to multiply; surfacing that is the shadow-mode cohort pass's job,
	// not a decision this gate makes today.)
	identity, err := repo.NormalizeEmailIdentity(in.Email)
	if err != nil {
		return decision, fmt.Errorf("%w: %w", ErrUnresolvableDomain, err)
	}
	registrable := identity.RegistrableDomain
	decision.RegistrableDomain = registrable
	decision.Ceiling = s.limits.MaxUnprovenOrgsPerDomain

	// purpose=create_org ONLY. Consulting the authenticate rules here would
	// re-conflate the two questions the Purpose split exists to separate.
	// Anything non-nil — a deny or a database error — is a refusal; a database
	// error must never be read as "no rule matched".
	if err := s.rules.CheckAccessFor(ctx, in.Email, repo.AccessPurposeCreateOrg); err != nil {
		return decision, err
	}

	if tx == nil {
		return decision, fmt.Errorf("%w: no transaction was supplied to the gate", ErrAdmissionNotConfigured)
	}

	// Same advisory lock the org bootstrap path already used, taken by the gate
	// itself so BOTH paths serialise against each other. Postgres transaction
	// advisory locks are re-entrant within one transaction, so a caller that
	// already holds it is unaffected. Without it, N concurrent registrations
	// each read a count of K-1 and all N are admitted.
	if err := tx.WithContext(ctx).Exec(
		`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, orgBootstrapLockKey,
	).Error; err != nil {
		return decision, fmt.Errorf("admission lock: %w", err)
	}

	// The ceiling: at most K organizations per registrable domain that are not
	// yet proven. No time window — an attacker pacing one registration every
	// two minutes evades every sliding window, token bucket and decaying score
	// while still minting hundreds of databases a day.
	//
	// The count is NOT restated here. jobs.CountUnprovenOrganizations is the
	// single expression of "unproven", shared with the proving sweep and the
	// cohort audit; a second copy of `proven_at IS NULL` in this file is exactly
	// how the gate and the sweep would drift into disagreeing about which
	// organizations count against a domain. It is passed the open transaction so
	// it reads inside the advisory lock taken above.
	unproven, err := jobs.CountUnprovenOrganizations(ctx, tx, registrable)
	if err != nil {
		return decision, err
	}
	decision.UnprovenOrgs = unproven
	if decision.UnprovenOrgs >= int64(s.limits.MaxUnprovenOrgsPerDomain) {
		return decision, ErrUnprovenOrgCeiling
	}

	// The platform quotas, which until now guarded only the social bootstrap
	// path; registration bypassed them entirely.
	var total int64
	if err := tx.WithContext(ctx).Model(&control.Organization{}).Count(&total).Error; err != nil {
		return decision, fmt.Errorf("count organizations: %w", err)
	}
	if total >= int64(s.limits.MaxTotalOrgs) {
		return decision, ErrOrgQuotaExceeded
	}

	var recent int64
	if err := tx.WithContext(ctx).Model(&control.Organization{}).
		Where("created_at > ?", time.Now().UTC().Add(-time.Hour)).
		Count(&recent).Error; err != nil {
		return decision, fmt.Errorf("count recent organizations: %w", err)
	}
	if recent >= int64(s.limits.MaxNewOrgsPerHour) {
		return decision, ErrOrgCreationRateExceeded
	}

	return decision, nil
}
