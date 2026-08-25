package service

import (
	"context"
	"errors"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var (
	ErrImpersonationNotAllowed = errors.New("impersonation not allowed")
	ErrTargetUserNotFound      = errors.New("target user not found")
	ErrTargetUserInactive      = errors.New("target user is inactive")
	ErrMembershipNotFound      = errors.New("membership not found in organization")
)

type ImpersonationService struct {
	users       repo.UserRepo
	memberships repo.MembershipRepo
	grants      repo.ImpersonationGrantRepo
	sessions    *TokenSessionService
	audit       repo.AuthAuditRepo
}

func NewImpersonationService(
	users repo.UserRepo,
	memberships repo.MembershipRepo,
	grants repo.ImpersonationGrantRepo,
	sessions *TokenSessionService,
	audit repo.AuthAuditRepo,
) *ImpersonationService {
	return &ImpersonationService{
		users:       users,
		memberships: memberships,
		grants:      grants,
		sessions:    sessions,
		audit:       audit,
	}
}

type StartImpersonationInput struct {
	AdminID        uuid.UUID
	TargetUserID   uuid.UUID
	OrganizationID uuid.UUID
	Reason         string
	IPAddress      string
	UserAgent      string
}

type StartImpersonationResult struct {
	AccessToken    string
	TokenType      string
	ExpiresIn      int
	OrganizationID string
	TargetUser     *control.User
}

func (s *ImpersonationService) Start(ctx context.Context, in StartImpersonationInput) (*StartImpersonationResult, error) {

	target, err := s.users.GetByID(ctx, in.TargetUserID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrTargetUserNotFound
	}
	if !target.IsActive {
		return nil, ErrTargetUserInactive
	}

	membership, err := s.memberships.GetByUserAndOrg(ctx, in.TargetUserID, in.OrganizationID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, ErrMembershipNotFound
	}

	if !membership.SupportEnabled {
		grant, err := s.grants.GetActiveGrant(ctx, in.TargetUserID, in.OrganizationID)
		if err != nil {
			return nil, err
		}
		if grant == nil {
			return nil, ErrImpersonationNotAllowed
		}
	}

	actorID := in.AdminID
	issued, err := s.sessions.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   in.TargetUserID.String(),
		SubjectKind: "user",
		OrgID:       in.OrganizationID.String(),
		Opts: IssueOpts{
			AMR:         []string{"impersonation"},
			Impersonate: true,
			ActorID:     in.AdminID.String(),
		},
		ActorID:   &actorID,
		IPAddress: in.IPAddress,
		UserAgent: in.UserAgent,
		EventType: "impersonation_start",
		Metadata: map[string]any{
			"actor_id": in.AdminID.String(),
			"org_id":   in.OrganizationID.String(),
			"reason":   in.Reason,
		},
	})
	if err != nil {
		return nil, err
	}

	return &StartImpersonationResult{
		AccessToken:    issued.AccessToken,
		TokenType:      issued.TokenType,
		ExpiresIn:      issued.ExpiresIn,
		OrganizationID: in.OrganizationID.String(),
		TargetUser:     target,
	}, nil
}

func (s *ImpersonationService) End(ctx context.Context, accessJTI string, targetUserID, actorID, orgID uuid.UUID, ip, ua string) error {
	if err := s.sessions.Revoke(ctx, accessJTI, targetUserID.String(), ip, ua); err != nil {
		return err
	}

	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      targetUserID,
		SubjectKind:    "user",
		OrganizationID: orgID,
		EventType:      "impersonation_end",
		IPAddress:      ip,
		UserAgent:      truncateUA(ua),
		Metadata: datatypes.NewJSONType(map[string]any{
			"actor_id": actorID.String(),
			"org_id":   orgID.String(),
		}),
	})

	return nil
}

type ToggleSupportInput struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Enabled        bool
	IPAddress      string
	UserAgent      string
}

func (s *ImpersonationService) ToggleSupport(ctx context.Context, in ToggleSupportInput) error {
	membership, err := s.memberships.GetByUserAndOrg(ctx, in.UserID, in.OrganizationID)
	if err != nil {
		return err
	}
	if membership == nil {
		return ErrMembershipNotFound
	}

	if err := s.memberships.UpdateSupportEnabled(ctx, membership.ID, in.Enabled); err != nil {
		return err
	}

	eventType := "support_enabled"
	if !in.Enabled {
		eventType = "support_disabled"
	}

	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      in.UserID,
		SubjectKind:    "user",
		OrganizationID: in.OrganizationID,
		EventType:      eventType,
		IPAddress:      in.IPAddress,
		UserAgent:      truncateUA(in.UserAgent),
		Metadata: datatypes.NewJSONType(map[string]any{
			"org_id":  in.OrganizationID.String(),
			"enabled": in.Enabled,
		}),
	})

	return nil
}
