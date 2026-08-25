package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type TokenSessionService struct {
	tokens      *TokenService
	sessions    repo.TokenSessionRepo
	audit       repo.AuthAuditRepo
	users       repo.UserRepo
	accessRules repo.AuthAccessRuleRepo
}

func (s *TokenSessionService) SetAccessRules(r repo.AuthAccessRuleRepo) {
	s.accessRules = r
}

func NewTokenSessionService(
	tokens *TokenService,
	sessions repo.TokenSessionRepo,
	audit repo.AuthAuditRepo,
	users repo.UserRepo,
) *TokenSessionService {
	return &TokenSessionService{
		tokens:   tokens,
		sessions: sessions,
		audit:    audit,
		users:    users,
	}
}

type IssueTrackedInput struct {
	SubjectID   string
	SubjectKind string
	OrgID       string
	FamilyID    *uuid.UUID
	Opts        IssueOpts
	IPAddress   string
	UserAgent   string
	EventType   string
	ActorID     *uuid.UUID
	Metadata    map[string]any
}

func (s *TokenSessionService) IssueTracked(ctx context.Context, in IssueTrackedInput) (*IssuedTokens, error) {
	now := time.Now().UTC()

	subjectID, err := uuid.Parse(in.SubjectID)
	if err != nil {
		return nil, err
	}

	if in.SubjectKind == "user" && s.accessRules != nil {
		if s.users == nil {
			return nil, ErrUnauthorized
		}
		user, err := s.users.GetByID(ctx, subjectID)
		if err != nil {
			return nil, fmt.Errorf("access rule check: load user: %w", err)
		}
		if user == nil {
			return nil, ErrUnauthorized
		}
		// Authenticate purpose ONLY. A create_org rule must not block a login:
		// a user invited into an existing tenant on a domain that may not mint
		// new tenants still has to be able to sign in.
		if err := s.accessRules.CheckAuthentication(ctx, user.Email); err != nil {
			return nil, err
		}
	}

	issued, err := s.tokens.Issue(in.SubjectID, in.OrgID, now, in.Opts)
	if err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(in.OrgID)
	if err != nil {
		return nil, err
	}

	familyID := uuid.New()
	if in.FamilyID != nil {
		familyID = *in.FamilyID
	}

	session := &control.TokenSession{
		SubjectID:        subjectID,
		SubjectKind:      in.SubjectKind,
		OrganizationID:   orgID,
		FamilyID:         familyID,
		AccessTokenJTI:   issued.AccessJTI,
		RefreshTokenJTI:  issued.RefreshJTI,
		AccessTokenHash:  HashToken(issued.AccessToken),
		RefreshTokenHash: HashToken(issued.RefreshToken),
		ActorID:          in.ActorID,
		AMR:              strings.Join(in.Opts.AMR, ","),
		IPAddress:        in.IPAddress,
		UserAgent:        truncateUA(in.UserAgent),
		AccessExpiresAt:  issued.AccessExpiresAt,
		RefreshExpiresAt: issued.RefreshExpiresAt,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	meta := in.Metadata
	if meta == nil {
		meta = map[string]any{}
	}

	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      subjectID,
		SubjectKind:    in.SubjectKind,
		OrganizationID: orgID,
		SessionID:      &session.ID,
		EventType:      in.EventType,
		IPAddress:      in.IPAddress,
		UserAgent:      truncateUA(in.UserAgent),
		Metadata:       datatypes.NewJSONType(meta),
	})

	return issued, nil
}

func (s *TokenSessionService) Refresh(ctx context.Context, refreshTokenRaw, ip, ua string, overrideOpts ...IssueOpts) (*IssuedTokens, string, error) {
	raw := strings.TrimSpace(refreshTokenRaw)
	if raw == "" {
		return nil, "", ErrUnauthorized
	}

	claims, err := s.tokens.Parse(raw, "refresh")
	if err != nil {
		return nil, "", ErrUnauthorized
	}

	if claims.Impersonate {
		return nil, "", ErrUnauthorized
	}

	jti := claims.ID
	if jti == "" {
		return nil, "", ErrUnauthorized
	}

	session, err := s.sessions.GetByRefreshJTI(ctx, jti)
	if err != nil {
		return nil, "", ErrUnauthorized
	}
	if session == nil {
		return nil, "", ErrUnauthorized
	}

	now := time.Now().UTC()

	if session.RevokedAt != nil {
		_, _ = s.sessions.RevokeByFamilyID(ctx, session.FamilyID, now)
		_ = s.audit.Create(ctx, &control.AuthAuditEvent{
			SubjectID:      session.SubjectID,
			SubjectKind:    session.SubjectKind,
			OrganizationID: session.OrganizationID,
			SessionID:      &session.ID,
			EventType:      "theft_detected",
			IPAddress:      ip,
			UserAgent:      truncateUA(ua),
			Metadata:       datatypes.NewJSONType(map[string]any{"family_id": session.FamilyID.String()}),
		})
		return nil, "", ErrUnauthorized
	}

	_ = s.sessions.Revoke(ctx, session.ID, now)

	opts := IssueOpts{
		AMR:        claims.AMR,
		MFA:        claims.MFA,
		MFAEnabled: claims.MFAEnabled,
	}
	if len(overrideOpts) > 0 {
		opts = overrideOpts[0]
	}
	issued, err := s.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   session.SubjectID.String(),
		SubjectKind: session.SubjectKind,
		OrgID:       session.OrganizationID.String(),
		FamilyID:    &session.FamilyID,
		Opts:        opts,
		IPAddress:   ip,
		UserAgent:   ua,
		EventType:   "refresh",
	})
	if err != nil {
		return nil, "", err
	}

	return issued, session.OrganizationID.String(), nil
}

func (s *TokenSessionService) Revoke(ctx context.Context, accessJTI, subjectID, ip, ua string) error {
	session, err := s.sessions.GetByAccessJTI(ctx, accessJTI)
	if err != nil || session == nil {
		return nil
	}

	subID, _ := uuid.Parse(subjectID)
	if session.SubjectID != subID {
		return ErrUnauthorized
	}

	now := time.Now().UTC()
	if err := s.sessions.Revoke(ctx, session.ID, now); err != nil {
		return err
	}

	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      session.SubjectID,
		SubjectKind:    session.SubjectKind,
		OrganizationID: session.OrganizationID,
		SessionID:      &session.ID,
		EventType:      "logout",
		IPAddress:      ip,
		UserAgent:      truncateUA(ua),
		Metadata:       datatypes.NewJSONType(map[string]any{}),
	})
	return nil
}

func (s *TokenSessionService) RevokeAll(ctx context.Context, subjectID uuid.UUID, subjectKind, ip, ua string) error {
	now := time.Now().UTC()
	_, err := s.sessions.RevokeAllBySubject(ctx, subjectID, subjectKind, now)
	if err != nil {
		return err
	}

	if subjectKind == "user" && s.users != nil {
		_ = s.users.UpdateTokensRevokedAt(ctx, subjectID, now)
	}

	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      subjectID,
		SubjectKind:    subjectKind,
		OrganizationID: uuid.Nil,
		EventType:      "revoke_all",
		IPAddress:      ip,
		UserAgent:      truncateUA(ua),
		Metadata:       datatypes.NewJSONType(map[string]any{}),
	})
	return nil
}

func (s *TokenSessionService) ListActive(ctx context.Context, subjectID uuid.UUID, subjectKind string) ([]control.TokenSession, error) {
	return s.sessions.ListActiveBySubject(ctx, subjectID, subjectKind)
}

func (s *TokenSessionService) DeleteExpired(ctx context.Context) (int64, error) {
	return s.sessions.DeleteExpired(ctx, time.Now().UTC())
}

func (s *TokenSessionService) Tokens() *TokenService {
	return s.tokens
}

func truncateUA(ua string) string {
	if len(ua) > 512 {
		return ua[:512]
	}
	return ua
}
