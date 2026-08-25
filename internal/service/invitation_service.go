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
	"gorm.io/gorm"
)

var (
	// ErrInvitationInvalid covers unknown, expired, revoked and already
	// accepted. One error for all four: the endpoint is unauthenticated, so
	// distinguishing them would let anyone holding a guess probe the table.
	ErrInvitationInvalid = errors.New("invitation: this invitation is invalid or has expired")

	// ErrAlreadyMember is not an error the invitee sees — accepting when you are
	// already in is reported as success. It is returned to the INVITER, who is
	// asking for something that would do nothing.
	ErrAlreadyMember = errors.New("invitation: that address is already a member of this organization")

	// ErrPasswordRequired is returned when a brand-new invitee accepts without
	// choosing one. There is no account to fall back on.
	ErrPasswordRequired = errors.New("invitation: choose a password to finish setting up your account")

	// ErrPasswordNotAccepted is its inverse: an invitee who ALREADY has an
	// account may not set a password through this path. Accepting an invitation
	// proves control of a mailbox, not knowledge of an existing account's
	// password, and letting it rewrite one would make any invitation to a known
	// address a full account takeover.
	ErrPasswordNotAccepted = errors.New("invitation: this address already has an account — sign in to accept")

	ErrCannotInviteSelf = errors.New("invitation: you are already a member")
)

const invitationTTL = 7 * 24 * time.Hour

type InvitationService struct {
	db          *gorm.DB
	invitations repo.InvitationRepo
	users       repo.UserRepo
	memberships repo.MembershipRepo
	orgs        repo.OrganizationRepo
	policies    *PolicyService
	accessRules AccessRuleResolver
	email       EmailSender
	audit       repo.AuthAuditRepo
	baseURL     string
}

func NewInvitationService(
	db *gorm.DB,
	invitations repo.InvitationRepo,
	users repo.UserRepo,
	memberships repo.MembershipRepo,
	orgs repo.OrganizationRepo,
	policies *PolicyService,
	accessRules AccessRuleResolver,
	email EmailSender,
	audit repo.AuthAuditRepo,
	baseURL string,
) *InvitationService {
	return &InvitationService{
		db:          db,
		invitations: invitations,
		users:       users,
		memberships: memberships,
		orgs:        orgs,
		policies:    policies,
		accessRules: accessRules,
		email:       email,
		audit:       audit,
		baseURL:     baseURL,
	}
}

type InviteInput struct {
	OrgID     uuid.UUID
	InviterID uuid.UUID
	Email     string
	Role      control.MemberRole
}

type InvitationView struct {
	ID        uuid.UUID
	Email     string
	Role      control.MemberRole
	ExpiresAt time.Time
	CreatedAt time.Time
	// Status is derived rather than stored: the row carries accepted_at,
	// revoked_at and expires_at, and exactly one reading of those is correct.
	// Deriving it in one place stops the API and the UI disagreeing about what
	// an expired-but-unaccepted invitation is.
	Status string
}

func toInvitationView(inv control.Invitation) InvitationView {
	return InvitationView{
		ID:        inv.ID,
		Email:     inv.Email,
		Role:      inv.Role,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
		Status:    invitationStatus(inv),
	}
}

func invitationStatus(inv control.Invitation) string {
	switch {
	case inv.AcceptedAt != nil:
		return "accepted"
	case inv.RevokedAt != nil:
		return "revoked"
	case !inv.ExpiresAt.After(time.Now().UTC()):
		return "expired"
	default:
		return "pending"
	}
}

// Invite creates an invitation and mails the link.
//
// The organization comes from the caller's principal, never from the request —
// the handler resolves it — so this method cannot be used to invite someone into
// a tenant the caller has no membership in.
func (s *InvitationService) Invite(ctx context.Context, in InviteInput) (*InvitationView, error) {
	if s.invitations == nil || s.users == nil || s.memberships == nil {
		return nil, errors.New("invitation: service is not wired")
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return nil, ErrPasswordRequired
	}
	if !roleIsInvitable(in.Role) {
		return nil, fmt.Errorf("invitation: %q is not a role that can be invited", in.Role)
	}

	// purpose=join_org, checked here and again at acceptance. Twice on purpose:
	// this one gives the inviter an immediate, actionable refusal, and the one
	// in Accept is the enforcing check — an invitation issued a week ago must
	// not still be redeemable after the platform has banned the domain.
	//
	// An unwired resolver denies. It must never read as "no rules exist".
	if s.accessRules == nil {
		return nil, ErrAdmissionNotConfigured
	}
	if err := s.accessRules.CheckAccessFor(ctx, email, repo.AccessPurposeJoinOrg); err != nil {
		return nil, err
	}

	// Already a member? Nothing to offer.
	if existing, err := s.users.GetByEmail(ctx, email); err == nil && existing != nil {
		m, err := s.memberships.GetByUserAndOrg(ctx, existing.ID, in.OrgID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if m != nil {
			return nil, ErrAlreadyMember
		}
	}

	if open, err := s.invitations.FindOpenForEmail(ctx, in.OrgID, email); err != nil {
		return nil, err
	} else if open != nil {
		return nil, repo.ErrInvitationExists
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("invitation: generate token: %w", err)
	}
	rawToken := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(rawToken))

	inv := &control.Invitation{
		OrganizationID: in.OrgID,
		InvitedByID:    in.InviterID,
		Email:          email,
		Role:           in.Role,
		TokenHash:      hex.EncodeToString(sum[:]),
		ExpiresAt:      time.Now().UTC().Add(invitationTTL),
	}
	if err := s.invitations.Create(ctx, inv); err != nil {
		return nil, err
	}

	s.sendInvite(ctx, inv, rawToken)
	s.writeAudit(ctx, in.InviterID, in.OrgID, "invitation_created")

	view := toInvitationView(*inv)
	return &view, nil
}

func roleIsInvitable(r control.MemberRole) bool {
	// Owner is deliberately absent. Ownership is transferred deliberately by an
	// existing owner, not handed out by emailing a link to a stranger.
	return r == control.MemberRoleMember || r == control.MemberRoleAdmin
}

func (s *InvitationService) sendInvite(ctx context.Context, inv *control.Invitation, rawToken string) {
	if s.email == nil {
		slog.WarnContext(ctx, "email service is not wired; invitation link not sent", "invitation_id", inv.ID)
		return
	}

	orgName := "an organization"
	if s.orgs != nil {
		if org, err := s.orgs.GetByID(ctx, inv.OrganizationID); err == nil && org != nil {
			orgName = org.Name
		}
	}

	acceptURL := fmt.Sprintf("%s/accept-invitation?token=%s", s.baseURL, rawToken)
	body := fmt.Sprintf(`<h2>You have been invited to %s</h2>
<p>Someone at %s invited you to join them.</p>
<p><a href="%s">Accept the invitation</a></p>
<p>This link expires in 7 days. If you were not expecting this, you can ignore it — nothing happens until you accept.</p>`,
		orgName, orgName, acceptURL)

	// Enqueued, not sent inline: the queue owns retry and backoff. This used
	// to be a goroutine that logged and dropped, so any momentary SMTP fault
	// lost the message outright.
	if err := s.email.Send(ctx, inv.Email, "You have been invited to "+orgName, body); err != nil {
		slog.ErrorContext(ctx, "invitation email failed", "error", err)
	}
}

// InvitationPreview is what the accept page needs in order to render before the
// invitee commits to anything.
type InvitationPreview struct {
	Email            string
	Organization     string
	Role             control.MemberRole
	NeedsAccount     bool
	NeedsAccountName bool
}

// Preview describes an invitation to whoever holds its token.
//
// Possession of the token IS the authorization: it was mailed to the address it
// names. It reveals whether that address already has an account, which the page
// needs in order to ask for a password or to send them to sign in — and which a
// token holder is entitled to know about their own address.
func (s *InvitationService) Preview(ctx context.Context, rawToken string) (*InvitationPreview, error) {
	inv, err := s.openInvitation(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	out := &InvitationPreview{Email: inv.Email, Role: inv.Role, Organization: "an organization"}
	if s.orgs != nil {
		if org, err := s.orgs.GetByID(ctx, inv.OrganizationID); err == nil && org != nil {
			out.Organization = org.Name
		}
	}

	user, err := s.users.GetByEmail(ctx, inv.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	out.NeedsAccount = user == nil
	out.NeedsAccountName = user == nil
	return out, nil
}

func (s *InvitationService) openInvitation(ctx context.Context, rawToken string) (*control.Invitation, error) {
	if s.invitations == nil {
		return nil, errors.New("invitation: service is not wired")
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	inv, err := s.invitations.FindOpenByTokenHash(ctx, hex.EncodeToString(sum[:]))
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvitationInvalid
	}
	return inv, nil
}

type AcceptInvitationInput struct {
	Token     string
	Password  string
	FirstName string
	LastName  string
}

type AcceptInvitationResult struct {
	Email        string
	Organization string
	// AccountCreated distinguishes the two endings the UI has to explain: a new
	// account that now has a password, and an existing account that just gained
	// a membership.
	AccountCreated bool
}

// Accept redeems an invitation. It issues NO session, on either branch.
//
// A new invitee sets a password here and then signs in through /auth/login like
// anyone else; an existing account gains the membership and signs in as it
// already could. Keeping session minting on the login path alone means the
// checks that guard it — lockout, MFA, the confirmation gate — are written once
// and cannot be skipped by arriving through this door.
func (s *InvitationService) Accept(ctx context.Context, in AcceptInvitationInput) (*AcceptInvitationResult, error) {
	inv, err := s.openInvitation(ctx, in.Token)
	if err != nil {
		return nil, err
	}

	// Enforced here, not just at issue time. An invitation is live for seven
	// days; a domain banned from joining on day three must not still be
	// redeemable on day four.
	if s.accessRules == nil {
		return nil, ErrAdmissionNotConfigured
	}
	if err := s.accessRules.CheckAccessFor(ctx, inv.Email, repo.AccessPurposeJoinOrg); err != nil {
		return nil, err
	}

	user, err := s.users.GetByEmail(ctx, inv.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	switch {
	case user != nil && in.Password != "":
		// Refused rather than ignored. Silently discarding the password would
		// leave the invitee believing they had set one.
		return nil, ErrPasswordNotAccepted
	case user == nil && in.Password == "":
		return nil, ErrPasswordRequired
	}

	created := false
	now := time.Now().UTC()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if user == nil {
			if s.policies != nil {
				if err := s.policies.ValidatePassword(ctx, inv.OrganizationID, in.Password); err != nil {
					return fmt.Errorf("%w: %w", ErrWeakPassword, err)
				}
			}
			first := strings.TrimSpace(in.FirstName)
			last := strings.TrimSpace(in.LastName)
			if first == "" && last == "" {
				return ErrRegistrationName
			}
			u := &control.User{
				Email:     inv.Email,
				FirstName: first,
				LastName:  last,
				Role:      control.RolePlatformCustomer,
				IsActive:  true,
				// Redeeming a token that was mailed to this address IS proof of
				// control over it — the same proof /auth/verify-email asks for.
				// Without this stamp the new account would be created already
				// locked out by the login confirmation gate, with no pending
				// verification token to redeem.
				EmailVerifiedAt: &now,
			}
			if err := u.SetPassword(in.Password); err != nil {
				return fmt.Errorf("hash password: %w", err)
			}
			if err := tx.Create(u).Error; err != nil {
				if isUniqueViolation(err) {
					// Someone registered this address between our lookup and
					// here. Their account, their password; the invitation is
					// still theirs to accept by signing in.
					return ErrPasswordNotAccepted
				}
				return err
			}
			user = u
			created = true
		} else if !user.EmailVerified() {
			// An existing but unconfirmed account that reaches an emailed link
			// has just proven the same thing a verification token proves.
			if err := tx.Model(&control.User{}).
				Where("id = ? AND email_verified_at IS NULL", user.ID).
				Update("email_verified_at", now).Error; err != nil {
				return err
			}
		}

		// Idempotent: a membership that already exists is the state we wanted.
		var existing int64
		if err := tx.Model(&control.Membership{}).
			Where("user_id = ? AND organization_id = ?", user.ID, inv.OrganizationID).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			if err := tx.Create(&control.Membership{
				UserID:         user.ID,
				OrganizationID: inv.OrganizationID,
				Role:           inv.Role,
			}).Error; err != nil {
				return err
			}
		}

		// Last, and inside the transaction: if this fails because another
		// request beat us to it, the membership we just wrote rolls back with
		// it rather than being created twice.
		res := tx.Model(&control.Invitation{}).
			Where("id = ? AND accepted_at IS NULL AND revoked_at IS NULL", inv.ID).
			Update("accepted_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvitationInvalid
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.writeAudit(ctx, user.ID, inv.OrganizationID, "invitation_accepted")

	out := &AcceptInvitationResult{Email: inv.Email, AccountCreated: created, Organization: "your organization"}
	if s.orgs != nil {
		if org, err := s.orgs.GetByID(ctx, inv.OrganizationID); err == nil && org != nil {
			out.Organization = org.Name
		}
	}
	return out, nil
}

func (s *InvitationService) List(ctx context.Context, orgID uuid.UUID) ([]InvitationView, error) {
	rows, err := s.invitations.ListForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toInvitationView(r))
	}
	return out, nil
}

func (s *InvitationService) Revoke(ctx context.Context, orgID, id, actorID uuid.UUID) error {
	if err := s.invitations.Revoke(ctx, orgID, id); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, orgID, "invitation_revoked")
	return nil
}

func (s *InvitationService) writeAudit(ctx context.Context, subjectID, orgID uuid.UUID, eventType string) {
	if s.audit == nil {
		return
	}
	if err := s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      subjectID,
		SubjectKind:    "user",
		OrganizationID: orgID,
		EventType:      eventType,
		Metadata:       datatypes.NewJSONType(map[string]any{}),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to write auth audit event", "event_type", eventType, "error", err)
	}
}

// ErrTenantNotProvisioned means the organization has no tenant database yet, so
// there is nothing for a new member to be admitted into. It is distinct from
// ErrTenantUnavailable, which means a provisioned tenant could not be reached:
// one is "not yet", the other is "not right now", and only the second is worth
// paging anybody about.
var ErrTenantNotProvisioned = errors.New("invitation: this organization has no tenant database yet")
