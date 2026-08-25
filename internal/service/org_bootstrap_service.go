package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/workflow"
	"gorm.io/gorm"
)

type OrgBootstrapLimits struct {
	MaxTotalOrgs             int
	MaxNewOrgsPerHour        int
	MaxUnprovenOrgsPerDomain int
}

var (
	ErrOrgQuotaExceeded        = errors.New("organization quota exceeded")
	ErrOrgCreationRateExceeded = errors.New("organization creation rate exceeded")
)

const orgBootstrapLockKey = "apiary:org_bootstrap"

type OrgBootstrapService struct {
	db        *gorm.DB
	jobs      workflow.Enqueuer
	admission *AdmissionService
}

func NewOrgBootstrapService(db *gorm.DB, jobs workflow.Enqueuer, admission *AdmissionService) *OrgBootstrapService {
	return &OrgBootstrapService{db: db, jobs: jobs, admission: admission}
}

func (s *OrgBootstrapService) EnsureUserHasWorkspace(ctx context.Context, user *control.User) (*control.Membership, error) {

	// Refuse before opening a transaction, and refuse rather than panic when the
	// gate was never wired: an unconfigured admission gate must never read as an
	// empty rule set that admits everyone.
	if err := s.admission.EnsureConfigured(); err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("org bootstrap: user is nil")
	}

	var existing []control.Membership
	if err := s.db.WithContext(ctx).
		Preload("Organization").
		Where("user_id = ?", user.ID).
		Find(&existing).Error; err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return &existing[0], nil
	}

	orgName := workspaceName(user)

	var org control.Organization
	var membership control.Membership
	created := false

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, orgBootstrapLockKey,
		).Error; err != nil {
			return fmt.Errorf("org bootstrap lock: %w", err)
		}

		var underLock []control.Membership
		if err := tx.Preload("Organization").
			Where("user_id = ?", user.ID).
			Find(&underLock).Error; err != nil {
			return err
		}
		if len(underLock) > 0 {
			membership = underLock[0]
			org = membership.Organization
			return nil
		}

		// THE GATE, on the transaction that will create the organization and
		// before any row is written. It runs after the membership recheck above
		// so a returning user is never refused by a quota they already passed,
		// and it carries the access rules and the ceiling that this path never
		// had, alongside the two quotas that used to be inlined here.
		decision, err := s.admission.Admit(ctx, tx, AdmissionInput{Email: user.Email})
		if err != nil {
			return err
		}

		// The domain is stamped in the same transaction as the insert. An
		// organization row with a NULL registrable_domain is invisible to every
		// later ceiling count, so an unstamped org is a free extra slot.
		org = control.Organization{
			Name:              orgName,
			RegistrableDomain: &decision.RegistrableDomain,
		}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}

		membership = control.Membership{
			UserID:         user.ID,
			OrganizationID: org.ID,
			Role:           "owner",
		}
		if err := tx.Create(&membership).Error; err != nil {
			return err
		}

		created = true
		return nil
	}); err != nil {
		return nil, err
	}

	membership.Organization = org

	if created && s.jobs != nil {
		if err := s.jobs.EnqueueProvisionTenant(ctx, &workflow.ProvisionTenantArgs{
			OrganizationID: org.ID.String(),
		}); err != nil {

			slog.ErrorContext(ctx, "enqueue tenant provisioning failed; organization left pending",
				"organization_id", org.ID.String(), "error", err)
		}
	}

	return &membership, nil
}

func workspaceName(user *control.User) string {
	if user.Email != "" && strings.TrimSpace(user.Email) != "" {
		email := strings.TrimSpace(user.Email)
		if i := strings.Index(email, "@"); i > 0 {
			return email[:i] + "'s Workspace"
		}
	}
	return "My Workspace"
}
