package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken       = errors.New("registration: that email is already registered")
	ErrRegistrationName = errors.New("registration: a name is required")
	ErrWeakPassword     = errors.New("registration: password does not meet policy")
)

type RegisterInput struct {
	Email            string
	Password         string
	FirstName        string
	LastName         string
	OrganizationName string
	IPAddress        string
	UserAgent        string
}

// RegisterResult carries no session and no tokens, by design.
//
// Registration used to sign the caller straight in. That made the first
// token_session for the new organization free — and token_sessions on distinct
// days are exactly what the proving sweep counts toward retiring the admission
// ceiling. An attacker got day one of proving for an address they had never
// read a single message at. Requiring a confirmed address before the first
// session is what makes the ceiling's premise true.
type RegisterResult struct {
	User           *control.User
	OrganizationID string
	Organization   string
	// VerificationSent reports whether the confirmation email was handed to the
	// mailer. The response says the same thing to the caller either way.
	VerificationSent bool
}

type RegistrationService struct {
	db           *gorm.DB
	users        repo.UserRepo
	policies     *PolicyService
	accessRules  AccessRuleResolver
	admission    *AdmissionService
	verification *EmailVerificationService
}

func NewRegistrationService(
	db *gorm.DB,
	users repo.UserRepo,
	policies *PolicyService,
	accessRules AccessRuleResolver,
	admission *AdmissionService,
	verification *EmailVerificationService,
) *RegistrationService {
	return &RegistrationService{
		db:           db,
		users:        users,
		policies:     policies,
		accessRules:  accessRules,
		admission:    admission,
		verification: verification,
	}
}

func (s *RegistrationService) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	first := strings.TrimSpace(in.FirstName)
	last := strings.TrimSpace(in.LastName)

	if first == "" && last == "" {
		return nil, ErrRegistrationName
	}

	// The abuse/ban list, purpose="authenticate", runs before anything is
	// written. Until now this check happened only inside IssueTracked, at the
	// very END of Register — by which point the user, the organization, the
	// membership AND the provisioning job (a real CREATE DATABASE + CREATE ROLE)
	// already existed. The caller got a 403 and the operator got an orphaned
	// database nothing ever cleans up.
	//
	// An unwired resolver denies: it must never read as "no rules exist".
	if s.accessRules == nil {
		return nil, ErrAdmissionNotConfigured
	}
	if err := s.accessRules.CheckAccessFor(ctx, email, repo.AccessPurposeAuthenticate); err != nil {
		return nil, err
	}

	if s.policies != nil {
		if err := s.policies.ValidatePassword(ctx, uuid.Nil, in.Password); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrWeakPassword, err)
		}
	}

	if existing, err := s.users.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrEmailTaken
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user := &control.User{
		Email:     email,
		FirstName: first,
		LastName:  last,
		Role:      control.RolePlatformCustomer,
		IsActive:  true,
	}
	if err := user.SetPassword(in.Password); err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	orgName := strings.TrimSpace(in.OrganizationName)
	if orgName == "" {
		orgName = workspaceName(user)
	}

	var org control.Organization
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// THE GATE, and it is the FIRST statement in the transaction that would
		// otherwise create the rows. Registration previously enforced no
		// create_org rules, no per-domain ceiling and none of the platform org
		// quotas — only the social bootstrap path did. Running here means a
		// refusal rolls back to nothing at all: no user, no organization, no
		// membership, and no provisioning job, because the enqueue below is only
		// reached once this transaction has committed.
		decision, err := s.admission.Admit(ctx, tx, AdmissionInput{Email: email})
		if err != nil {
			return err
		}

		if err := tx.Create(user).Error; err != nil {
			return err
		}
		// registrable_domain is stamped in the same transaction as the insert;
		// an unstamped organization is invisible to every later ceiling count.
		org = control.Organization{Name: orgName, RegistrableDomain: &decision.RegistrableDomain}
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		return tx.Create(&control.Membership{
			UserID:         user.ID,
			OrganizationID: org.ID,
			Role:           control.MemberRoleOwner,
		}).Error
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	// No provisioning job here. The organization row exists so that it occupies
	// a slot against the domain ceiling from this moment — that is what stops a
	// registrant opening a hundred pending organizations — but a tenant database
	// is a real, expensive resource and is not created until someone has proven
	// they can read the mailbox. See EmailVerificationService.Confirm.
	sent := true
	if s.verification == nil {
		// Not fatal, and deliberately not silent: an operator running without a
		// verification service configured has just created an account nobody can
		// confirm, and needs to see that in the log rather than discover it from
		// a support ticket.
		slog.ErrorContext(ctx, "registration completed with no verification service wired; account cannot be confirmed",
			"user_id", user.ID, "organization_id", org.ID)
		sent = false
	} else if err := s.verification.Issue(ctx, user); err != nil {
		// The account and organization are committed. Reporting a failure now
		// would be accurate but useless — the caller cannot retry registration,
		// the address is taken. Resend is the recovery path, so report success
		// and let them ask for another link.
		slog.ErrorContext(ctx, "could not issue verification token after registration",
			"user_id", user.ID, "error", err)
		sent = false
	}

	return &RegisterResult{
		User:             user,
		OrganizationID:   org.ID.String(),
		Organization:     org.Name,
		VerificationSent: sent,
	}, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
