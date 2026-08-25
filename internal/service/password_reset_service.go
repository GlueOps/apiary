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
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var (
	ErrResetTokenInvalid = errors.New("reset token is invalid or expired")
	ErrResetTokenUsed    = errors.New("reset token has already been used")
)

const resetTokenTTL = 1 * time.Hour

type PasswordResetService struct {
	resets      repo.PasswordResetRepo
	users       repo.UserRepo
	email       EmailSender
	baseURL     string
	sessions    *TokenSessionService
	policies    *PolicyService
	memberships repo.MembershipRepo
	audit       repo.AuthAuditRepo
}

func NewPasswordResetService(
	resets repo.PasswordResetRepo,
	users repo.UserRepo,
	email EmailSender,
	baseURL string,
	sessions *TokenSessionService,
	policies *PolicyService,
	memberships repo.MembershipRepo,
	audit repo.AuthAuditRepo,
) *PasswordResetService {
	return &PasswordResetService{
		resets:      resets,
		users:       users,
		email:       email,
		baseURL:     baseURL,
		sessions:    sessions,
		policies:    policies,
		memberships: memberships,
		audit:       audit,
	}
}

func (s *PasswordResetService) RequestReset(ctx context.Context, emailAddr string) error {

	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	user, err := s.users.GetByEmail(ctx, emailAddr)
	if err != nil {

		return nil
	}

	if user == nil || !user.IsActive {
		return nil
	}

	_ = s.resets.DeleteExpiredForUser(ctx, user.ID)

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	token := &control.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(resetTokenTTL),
	}
	if err := s.resets.Create(ctx, token); err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, rawToken)
	body := fmt.Sprintf(`<h2>Password Reset</h2>
<p>Hi %s,</p>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you didn't request this, you can safely ignore this email.</p>`,
		user.FirstName, resetURL)

	// Enqueued, not sent inline: the queue owns retry and backoff. This used
	// to be a goroutine that logged and dropped, so any momentary SMTP fault
	// lost the message outright.
	if err := s.email.Send(ctx, user.Email, "Password Reset", body); err != nil {
		slog.ErrorContext(ctx, "password reset email failed", "error", err)
	}

	return nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	token, err := s.resets.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}
	if token == nil {
		return ErrResetTokenInvalid
	}

	if s.policies != nil {
		orgID, err := s.primaryOrgID(ctx, token.UserID)
		if err == nil && orgID != uuid.Nil {
			if err := s.policies.ValidatePassword(ctx, orgID, newPassword); err != nil {
				return err
			}
		}
	}

	if err := s.resets.MarkUsed(ctx, token.ID); err != nil {
		return err
	}

	if err := s.users.UpdateProfile(ctx, token.UserID, repo.UpdateUserProfileInput{
		Password: &newPassword,
	}); err != nil {
		return err
	}

	if s.sessions != nil {
		if err := s.sessions.RevokeAll(ctx, token.UserID, "user", "", ""); err != nil {
			return err
		}
	}

	s.writeAudit(ctx, token.UserID, "password_reset")

	if err := s.resets.InvalidateAllForUser(ctx, token.UserID); err != nil {
		return err
	}

	return nil
}

func (s *PasswordResetService) primaryOrgID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	if s.memberships == nil {
		return uuid.Nil, nil
	}
	ms, err := s.memberships.ListByUserID(ctx, userID)
	if err != nil || len(ms) == 0 {
		return uuid.Nil, err
	}
	return ms[0].OrganizationID, nil
}

func (s *PasswordResetService) writeAudit(ctx context.Context, userID uuid.UUID, eventType string) {
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
