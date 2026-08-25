package service

import (
	"context"
	"errors"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

var ErrUnauthorized = errors.New("unauthorized")

type RefreshInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type RefreshResult struct {
	Tokens         *IssuedTokens
	OrganizationID string
	Organization   string
}

type MeResult struct {
	User struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		IsActive  bool   `json:"is_active"`
	}
	Organization struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"organization"`
}

type MyOrganizationResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Role      string `json:"role"`
	IsDefault bool   `json:"is_default"`
}

type SwitchOrganizationInput struct {
	UserID         string
	OrganizationID string
	IPAddress      string
	UserAgent      string

	MFA bool
}

type SwitchOrganizationResult struct {
	Tokens         *IssuedTokens
	OrganizationID string
	Organization   string
}

type SessionService struct {
	users         repo.UserRepoGorm
	memberships   repo.MembershipRepo
	organizations repo.OrganizationRepo
	tokens        *TokenService
	sessions      *TokenSessionService
	policies      *PolicyService
}

func NewSessionService(
	users repo.UserRepoGorm,
	memberships repo.MembershipRepo,
	organizations repo.OrganizationRepo,
	tokens *TokenService,
	sessions *TokenSessionService,
	policies *PolicyService,
) *SessionService {
	return &SessionService{
		users:         users,
		memberships:   memberships,
		organizations: organizations,
		tokens:        tokens,
		sessions:      sessions,
		policies:      policies,
	}
}

func (s *SessionService) Refresh(ctx context.Context, in RefreshInput) (*RefreshResult, error) {
	raw := strings.TrimSpace(in.RefreshToken)
	if raw == "" {
		return nil, ErrUnauthorized
	}

	claims, err := s.tokens.Parse(raw, "refresh")
	if err != nil {
		return nil, ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrUnauthorized
	}

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil || !user.IsActive {
		return nil, ErrUnauthorized
	}

	if user.TokensRevokedAt != nil && claims.IssuedAt != nil {
		if claims.IssuedAt.Before(*user.TokensRevokedAt) {
			return nil, ErrUnauthorized
		}
	}

	if err := requireMembershipFromRepo(ctx, s.memberships, userID, orgID); err != nil {
		return nil, ErrUnauthorized
	}

	org, err := s.organizations.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return nil, ErrUnauthorized
	}

	var refreshOverride []IssueOpts
	if s.policies != nil {

		ep, perr := s.policies.EffectivePolicy(ctx, orgID, control.PolicyKindRequireMFA)
		if perr != nil {
			return nil, perr
		}
		effectiveMFAEnabled := user.MFAEnabled || (ep != nil && ep.Enabled)
		refreshOverride = append(refreshOverride, IssueOpts{
			AMR: claims.AMR,

			MFA:        claims.MFA,
			MFAEnabled: effectiveMFAEnabled,
		})
	}

	issued, _, err := s.sessions.Refresh(ctx, in.RefreshToken, in.IPAddress, in.UserAgent, refreshOverride...)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return &RefreshResult{
		Tokens:         issued,
		OrganizationID: org.ID.String(),
		Organization:   org.Name,
	}, nil
}

func (s *SessionService) ListMyOrganizations(
	ctx context.Context,
	userID string,
	currentOrgID string,
) ([]MyOrganizationResult, error) {
	uid, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return nil, ErrUnauthorized
	}

	var current uuid.UUID
	if strings.TrimSpace(currentOrgID) != "" {
		current, err = uuid.Parse(currentOrgID)
		if err != nil {
			return nil, ErrUnauthorized
		}
	}

	user, err := s.users.GetByID(ctx, uid)
	if err != nil || user == nil || !user.IsActive {
		return nil, ErrUnauthorized
	}

	memberships, err := s.memberships.ListByUserID(ctx, uid)
	if err != nil {
		return nil, err
	}

	out := make([]MyOrganizationResult, 0, len(memberships))
	for _, m := range memberships {
		item := MyOrganizationResult{
			ID:        m.OrganizationID.String(),
			Role:      string(m.Role),
			IsDefault: m.OrganizationID == current,
			Name:      m.Organization.Name,
			Status:    m.Organization.Status,
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *SessionService) SwitchOrganization(
	ctx context.Context,
	in SwitchOrganizationInput,
) (*SwitchOrganizationResult, error) {
	userID, err := uuid.Parse(strings.TrimSpace(in.UserID))
	if err != nil {
		return nil, ErrUnauthorized
	}

	orgID, err := uuid.Parse(strings.TrimSpace(in.OrganizationID))
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil || !user.IsActive {
		return nil, ErrUnauthorized
	}

	membership, err := s.memberships.GetByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, ErrUnauthorized
	}

	org, err := s.organizations.GetByID(ctx, orgID)
	if err != nil || org == nil {
		return nil, ErrUnauthorized
	}

	effectiveMFAEnabled := user.MFAEnabled
	if s.policies != nil {
		ep, perr := s.policies.EffectivePolicy(ctx, orgID, control.PolicyKindRequireMFA)
		if perr != nil {
			return nil, perr
		}
		if ep != nil && ep.Enabled {
			effectiveMFAEnabled = true
		}
	}

	issued, err := s.sessions.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   user.ID.String(),
		SubjectKind: "user",
		OrgID:       org.ID.String(),
		Opts: IssueOpts{
			MFA:        in.MFA,
			MFAEnabled: effectiveMFAEnabled,
		},
		IPAddress: in.IPAddress,
		UserAgent: in.UserAgent,
		EventType: "org_switch",
	})
	if err != nil {
		return nil, err
	}

	return &SwitchOrganizationResult{
		Tokens:         issued,
		OrganizationID: org.ID.String(),
		Organization:   org.Name,
	}, nil
}

func requireMembershipFromRepo(ctx context.Context, memberships repo.MembershipRepo, userID, orgID uuid.UUID) error {
	list, err := memberships.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for _, m := range list {
		if m.OrganizationID == orgID {
			return nil
		}
	}
	return ErrUnauthorized
}
