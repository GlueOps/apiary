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
	ErrUserNotFound    = errors.New("user not found")
	ErrWrongPassword   = errors.New("current password is incorrect")
	ErrNothingToUpdate = errors.New("nothing to update")
)

type ProfileResult struct {
	User         *control.User
	Organization *control.Organization
	MFAEnabled   bool
	TOTPEnabled  bool
}

type UpdateProfileInput struct {
	FirstName *string
	LastName  *string
	Email     *string
}

type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
	OrgID           uuid.UUID
}

type ProfileService struct {
	users         repo.UserRepo
	organizations repo.OrganizationRepo
	memberships   repo.MembershipRepo
	totps         repo.UserTOTPRepo
	policies      *PolicyService
	sessions      *TokenSessionService
	audit         repo.AuthAuditRepo
	emailChanges  repo.EmailChangeRepo
	email         EmailSender
	baseURL       string
}

var ErrEmailChangeRequiresVerification = errors.New("email changes must be confirmed via the verification link")

var ErrEmailChangeInvalid = errors.New("email change token is invalid or expired")

const emailChangeTTL = 1 * time.Hour

func (s *ProfileService) SetEmailChange(r repo.EmailChangeRepo, email EmailSender, baseURL string) {
	s.emailChanges = r
	s.email = email
	s.baseURL = baseURL
}

func NewProfileService(
	users repo.UserRepo,
	organizations repo.OrganizationRepo,
	memberships repo.MembershipRepo,
	totps repo.UserTOTPRepo,
	policies *PolicyService,
	sessions *TokenSessionService,
	audit repo.AuthAuditRepo,
) *ProfileService {
	return &ProfileService{
		users:         users,
		organizations: organizations,
		memberships:   memberships,
		totps:         totps,
		policies:      policies,
		sessions:      sessions,
		audit:         audit,
	}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID, orgID uuid.UUID) (*ProfileResult, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	org, err := s.organizations.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var totpEnabled bool
	if user.MFAEnabled {
		totp, err := s.totps.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		totpEnabled = totp != nil && totp.Verified
	}

	return &ProfileResult{
		User:         user,
		Organization: org,
		MFAEnabled:   user.MFAEnabled,
		TOTPEnabled:  totpEnabled,
	}, nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (*control.User, error) {
	if in.FirstName == nil && in.LastName == nil && in.Email == nil {
		return nil, ErrNothingToUpdate
	}

	if in.Email != nil {
		return nil, ErrEmailChangeRequiresVerification
	}

	err := s.users.UpdateProfile(ctx, userID, repo.UpdateUserProfileInput{
		FirstName: in.FirstName,
		LastName:  in.LastName,
	})
	if err != nil {
		return nil, err
	}

	return s.users.GetByID(ctx, userID)
}

func (s *ProfileService) RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error {
	newEmail = strings.ToLower(strings.TrimSpace(newEmail))
	if newEmail == "" {
		return ErrNothingToUpdate
	}
	if s.emailChanges == nil {
		return ErrEmailChangeRequiresVerification
	}

	if err := s.emailChanges.InvalidateForUser(ctx, userID); err != nil {
		return err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawToken))

	if err := s.emailChanges.Create(ctx, &control.EmailChangeRequest{
		UserID:    userID,
		NewEmail:  newEmail,
		TokenHash: hex.EncodeToString(sum[:]),
		ExpiresAt: time.Now().UTC().Add(emailChangeTTL),
	}); err != nil {
		return err
	}

	if s.email != nil {
		confirmURL := fmt.Sprintf("%s/confirm-email?token=%s", s.baseURL, rawToken)
		body := fmt.Sprintf(
			`<h2>Confirm your new email address</h2>`+
				`<p>Click the link below to confirm this address for your account:</p>`+
				`<p><a href="%s">Confirm email</a></p>`+
				`<p>This link expires in 1 hour. If you did not request this, ignore it — `+
				`your address has not changed.</p>`, confirmURL)

		// Enqueued, not sent inline: the queue owns retry and backoff. This used
		// to be a goroutine that logged and dropped, so any momentary SMTP fault
		// lost the message outright.
		if err := s.email.Send(ctx, newEmail, "Confirm your email address", body); err != nil {
			slog.ErrorContext(ctx, "email change confirmation failed", "error", err)
		}
	}

	return nil
}

func (s *ProfileService) ConfirmEmailChange(ctx context.Context, rawToken string) error {
	if s.emailChanges == nil {
		return ErrEmailChangeInvalid
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	req, err := s.emailChanges.Consume(ctx, hex.EncodeToString(sum[:]), time.Now().UTC())
	if err != nil {
		return err
	}
	if req == nil {
		return ErrEmailChangeInvalid
	}

	if err := s.users.UpdateProfile(ctx, req.UserID, repo.UpdateUserProfileInput{
		Email: &req.NewEmail,
	}); err != nil {
		return err
	}

	s.writeAudit(ctx, req.UserID, "email_changed")
	return nil
}

func (s *ProfileService) ChangePassword(ctx context.Context, userID uuid.UUID, in ChangePasswordInput) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if !user.CheckPassword(in.CurrentPassword) {
		return ErrWrongPassword
	}

	if s.policies != nil {
		if err := s.policies.ValidatePassword(ctx, in.OrgID, in.NewPassword); err != nil {
			return err
		}
	}

	if err := s.users.UpdateProfile(ctx, userID, repo.UpdateUserProfileInput{
		Password: &in.NewPassword,
	}); err != nil {
		return err
	}

	s.writeAudit(ctx, userID, "password_changed")

	if s.sessions != nil {
		if err := s.sessions.RevokeAll(ctx, userID, "user", "", ""); err != nil {
			return err
		}
	}

	return nil
}

func (s *ProfileService) writeAudit(ctx context.Context, userID uuid.UUID, eventType string) {
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
