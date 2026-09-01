package app

import (
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
)

type Deps struct {
	Config        *config.Config
	ControlDB     *gorm.DB
	DBPool        *pgxpool.Pool
	River         *river.Client[pgx.Tx]
	Tenants       *database.TenantManager
	Jobs          workflow.Enqueuer
	TokenSessions *service.TokenSessionService
	Policies      *service.PolicyService

	// Flags is the feature flag registry's resolver. Distinct from Policies
	// because the two merge in opposite directions: a policy can only tighten as
	// it descends, and a flag's whole purpose is that the narrower level
	// overrides — in either direction.
	Flags   *service.FeatureFlagService
	Estates *service.EstateService
	Mail    service.EmailSender

	// APIKeys verifies every key/secret credential, on every surface. One
	// instance, because scope exclusion is only a boundary if the same code
	// enforces it for org keys and agent keys alike.
	APIKeys *service.APIKeyService

	// Admission is THE org-creation gate, constructed once so registration and
	// social bootstrap cannot end up enforcing different limits. It is a
	// pointer whose nil case denies rather than panics, because api.Build is
	// exercised with partially-populated Deps in the router tests.
	Admission *service.AdmissionService
}

// MailSender returns the configured sender, falling back to sending inline.
//
// The fallback is not laziness: api.Build is exercised in tests with
// partially-populated Deps, and a nil sender there would panic inside a handler
// rather than fail visibly at construction. In production serve.go sets Mail to
// the QUEUED sender, so mail gets retries and backoff; the direct path is a
// development convenience and has neither.
func (d *Deps) MailSender() service.EmailSender {
	if d != nil && d.Mail != nil {
		return d.Mail
	}
	var cfg config.Config
	if d != nil && d.Config != nil {
		cfg = *d.Config
	}
	return service.NewDirectEmailSender(service.NewEmailService(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom,
	))
}
