package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	// ErrVerificationTokenInvalid covers expired, already-used, unknown, and
	// issued-for-a-different-address. They are one error on purpose: telling a
	// caller which of those a token is turns the endpoint into an oracle over
	// tokens they do not hold.
	ErrVerificationTokenInvalid = errors.New("verification: link is invalid or has expired")

	// ErrEmailNotVerified is returned by the login path. It is separate from
	// ErrInvalidCredentials because the caller supplied a CORRECT password — the
	// UI has to be able to route them to "resend the link" rather than "you
	// typed it wrong". See the note in LoginService.Login on why that disclosure
	// is acceptable here and would not be at the identification step.
	ErrEmailNotVerified = errors.New("verification: this address has not been confirmed yet")
)

const (
	// Long enough to survive a link opened the next morning, short enough that a
	// mailbox compromised later does not hand over a live registration. Password
	// reset uses one hour because it grants a password; this grants only the
	// right to sign in with a password you already chose.
	verificationTokenTTL = 24 * time.Hour

	// The resend throttle. It counts ISSUED TOKENS in the window rather than
	// requests, so it cannot be reset by anything the caller controls, and it is
	// keyed on the user rather than the IP so that rotating source addresses
	// does not buy more mail. Its job is to stop this endpoint being used to
	// send mail at someone, not to stop registration abuse — the admission
	// ceiling does that.
	verificationResendWindow = 1 * time.Hour
	verificationResendMax    = 5
)

// EmailVerificationService issues and redeems the proof that a registrant can
// read the mailbox they signed up with.
//
// Redeeming is what releases tenant provisioning. Registration deliberately
// does not provision: a CREATE DATABASE and CREATE ROLE per unverified signup
// is a resource an unauthenticated caller can mint, and before this split every
// registration minted one before anyone had read a single email.
type EmailVerificationService struct {
	tokens      repo.EmailVerificationRepo
	users       repo.UserRepo
	memberships repo.MembershipRepo
	email       EmailSender
	jobs        workflow.Enqueuer
	audit       repo.AuthAuditRepo
	baseURL     string
}

func NewEmailVerificationService(
	tokens repo.EmailVerificationRepo,
	users repo.UserRepo,
	memberships repo.MembershipRepo,
	email EmailSender,
	jobs workflow.Enqueuer,
	audit repo.AuthAuditRepo,
	baseURL string,
) *EmailVerificationService {
	return &EmailVerificationService{
		tokens:      tokens,
		users:       users,
		memberships: memberships,
		email:       email,
		jobs:        jobs,
		audit:       audit,
		baseURL:     baseURL,
	}
}

// Issue mints a token for a user and mails the link.
//
// It returns an error, and registration propagates it. That is deliberate: if
// the token cannot be written, the account exists with no way to ever confirm
// it, and reporting success would leave the registrant permanently unable to
// sign in with no indication why.
func (s *EmailVerificationService) Issue(ctx context.Context, user *control.User) error {
	if s.tokens == nil {
		return errors.New("verification: token repository is not wired")
	}
	if user == nil {
		return errors.New("verification: no user supplied")
	}

	// Housekeeping, not correctness — a failure here must not stop a signup.
	_ = s.tokens.DeleteExpiredForUser(ctx, user.ID)

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("verification: generate token: %w", err)
	}
	rawToken := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawToken))

	// Only the hash is stored. A dump of this table must not yield a working
	// link for every pending registration.
	if err := s.tokens.Create(ctx, &control.EmailVerificationToken{
		UserID:    user.ID,
		Email:     user.Email,
		TokenHash: hex.EncodeToString(sum[:]),
		ExpiresAt: time.Now().UTC().Add(verificationTokenTTL),
	}); err != nil {
		return fmt.Errorf("verification: store token: %w", err)
	}

	s.send(ctx, user, rawToken)
	return nil
}

func (s *EmailVerificationService) send(ctx context.Context, user *control.User, rawToken string) {
	if s.email == nil {
		slog.WarnContext(ctx, "email service is not wired; verification link not sent", "user_id", user.ID)
		return
	}
	confirmURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, rawToken)
	body := fmt.Sprintf(`<h2>Confirm your email address</h2>
<p>Hi %s,</p>
<p>Confirm this address to finish setting up your account:</p>
<p><a href="%s">Confirm my email address</a></p>
<p>This link expires in 24 hours. If you did not create an account, you can ignore this email — nothing has been set up.</p>`,
		user.FirstName, confirmURL)

	// Detached from the request context so a client disconnect does not cancel a
	// send that has already been committed to, and given its own deadline so it
	// cannot outlive the process quietly. Same shape as the password reset path.
	// Enqueued, not sent inline: the queue owns retry and backoff. This used
	// to be a goroutine that logged and dropped, so any momentary SMTP fault
	// lost the message outright.
	if err := s.email.Send(ctx, user.Email, "Confirm your email address", body); err != nil {
		slog.ErrorContext(ctx, "verification email failed", "error", err)
	}
}

// ConfirmResult tells the caller what happened without handing back a session.
type ConfirmResult struct {
	Email string
	// AlreadyVerified distinguishes a second click on the same link from the
	// first. Both are success; the UI says something different for each.
	AlreadyVerified bool
	// TenantProvisioning reports whether provisioning was actually enqueued.
	TenantProvisioning bool
}

// Confirm redeems a token: it marks the address verified and releases tenant
// provisioning. It issues no session — a confirmed registrant signs in through
// the ordinary login path, so that every session in the system is minted by one
// piece of code with one set of checks.
func (s *EmailVerificationService) Confirm(ctx context.Context, rawToken string) (*ConfirmResult, error) {
	if s.tokens == nil || s.users == nil {
		return nil, errors.New("verification: service is not wired")
	}

	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	token, err := s.tokens.FindByTokenHash(ctx, hex.EncodeToString(sum[:]))
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrVerificationTokenInvalid
	}

	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if user == nil {
		return nil, ErrVerificationTokenInvalid
	}

	// The address must still be the one the token was issued for. Without this,
	// a token issued for an address the registrant controlled would confirm
	// whatever address the account carries when the link is finally clicked.
	if !strings.EqualFold(user.Email, token.Email) {
		return nil, ErrVerificationTokenInvalid
	}

	alreadyVerified := user.EmailVerified()

	// Compare-and-set. Two clicks on the same link race to here; the loser gets
	// ErrRecordNotFound and is reported as an invalid link rather than being
	// allowed to enqueue a second provisioning job for the same organization.
	if err := s.tokens.MarkUsed(ctx, token.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVerificationTokenInvalid
		}
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.users.MarkEmailVerified(ctx, user.ID, now); err != nil {
		return nil, err
	}
	if err := s.tokens.InvalidateAllForUser(ctx, user.ID); err != nil {
		return nil, err
	}

	out := &ConfirmResult{Email: user.Email, AlreadyVerified: alreadyVerified}
	if !alreadyVerified {
		s.writeAudit(ctx, user.ID, "email_verified")
		out.TenantProvisioning = s.provision(ctx, user.ID)
	}
	return out, nil
}

// provision enqueues tenant provisioning for every organization this user owns
// that has been waiting on confirmation.
//
// A failure to enqueue is logged and reported, not returned: the address IS now
// verified and that fact is already committed. Failing the request would tell
// the registrant their link did not work while leaving them unable to use it
// again, which is strictly worse than a tenant that provisions late.
func (s *EmailVerificationService) provision(ctx context.Context, userID uuid.UUID) bool {
	if s.jobs == nil || s.memberships == nil {
		return false
	}
	ms, err := s.memberships.ListByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "could not list memberships to release provisioning", "error", err)
		return false
	}

	enqueued := false
	for _, m := range ms {
		// Only the owner's confirmation releases provisioning. A member who was
		// invited into someone else's organization confirming their own address
		// says nothing about whether that organization should exist.
		if m.Role != control.MemberRoleOwner {
			continue
		}
		if err := s.jobs.EnqueueProvisionTenant(ctx, &workflow.ProvisionTenantArgs{
			OrganizationID: m.OrganizationID.String(),
		}); err != nil {
			slog.ErrorContext(ctx, "could not enqueue tenant provisioning after verification",
				"organization_id", m.OrganizationID, "error", err)
			continue
		}
		enqueued = true
	}
	return enqueued
}

// Resend issues a fresh link. It reports success unconditionally.
//
// Every branch below returns nil: unknown address, inactive user, already
// verified, and throttled all look identical from outside. This endpoint is
// unauthenticated and takes an arbitrary email, so any branch that answered
// differently would be a membership oracle — "is this address registered here,
// and has it been confirmed" — answerable by anyone.
func (s *EmailVerificationService) Resend(ctx context.Context, emailAddr string) error {
	if s.users == nil || s.tokens == nil {
		return errors.New("verification: service is not wired")
	}

	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	user, err := s.users.GetByEmail(ctx, emailAddr)
	if err != nil || user == nil {
		// Including a genuine database error. The caller learns nothing either
		// way, and a 500 here would itself distinguish some addresses.
		return nil
	}
	if !user.IsActive || user.EmailVerified() {
		return nil
	}

	since := time.Now().UTC().Add(-verificationResendWindow)
	n, err := s.tokens.CountRecentForUser(ctx, user.ID, since)
	if err != nil {
		// Cannot establish that we are under the limit, so do not send.
		slog.ErrorContext(ctx, "could not count recent verification tokens", "error", err)
		return nil
	}
	if n >= verificationResendMax {
		slog.WarnContext(ctx, "verification resend throttled", "user_id", user.ID, "issued_in_window", n)
		return nil
	}

	if err := s.Issue(ctx, user); err != nil {
		slog.ErrorContext(ctx, "could not reissue verification token", "error", err)
	}
	return nil
}

func (s *EmailVerificationService) writeAudit(ctx context.Context, userID uuid.UUID, eventType string) {
	if s.audit == nil {
		return
	}
	if err := s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      userID,
		SubjectKind:    "user",
		OrganizationID: uuid.Nil,
		EventType:      eventType,
		Metadata:       datatypes.NewJSONType(map[string]any{}),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write auth audit event", "event_type", eventType, "error", err)
	}
}
