package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvitationNotFound = errors.New("invitation: not found")
	ErrInvitationExists   = errors.New("invitation: an open invitation for that address already exists")
)

type InvitationRepo interface {
	Create(ctx context.Context, inv *control.Invitation) error
	FindOpenByTokenHash(ctx context.Context, hash string) (*control.Invitation, error)
	FindOpenForEmail(ctx context.Context, orgID uuid.UUID, email string) (*control.Invitation, error)
	ListForOrg(ctx context.Context, orgID uuid.UUID) ([]control.Invitation, error)
	MarkAccepted(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, orgID, id uuid.UUID) error
}

type InvitationRepoGorm struct {
	db *gorm.DB
}

func NewInvitationRepoGorm(db *gorm.DB) *InvitationRepoGorm {
	return &InvitationRepoGorm{db: db}
}

// openPredicate is the single definition of "still redeemable".
//
// It exists as one string because the same condition is needed by the lookup,
// the duplicate check and the listing. Three hand-written copies is how one of
// them ends up missing revoked_at and quietly honouring a withdrawn invitation.
const openPredicate = `accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?`

// Create does NOT rely on a database constraint to prevent two open
// invitations for the same address; the caller's FindOpenForEmail check is
// racy by construction.
//
// That is a deliberate trade. The obvious fix is a partial unique index on
// (organization_id, email) WHERE accepted_at IS NULL AND revoked_at IS NULL,
// which GORM's tags cannot express — so it would have to be hand-written into a
// migration, and Atlas diffs the migrations against the models. An index the
// models do not describe gets DROPPED by the next generated migration, silently,
// which is a worse failure than the one it prevents.
//
// The residual harm is small: a duplicate open invitation lets the same person
// be invited twice, and Accept is idempotent on the membership and a
// compare-and-set on the invitation row, so the second redemption grants
// nothing extra.
func (r *InvitationRepoGorm) Create(ctx context.Context, inv *control.Invitation) error {
	inv.Email = strings.ToLower(strings.TrimSpace(inv.Email))
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *InvitationRepoGorm) FindOpenByTokenHash(ctx context.Context, hash string) (*control.Invitation, error) {
	var inv control.Invitation
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND "+openPredicate, hash, time.Now().UTC()).
		First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

func (r *InvitationRepoGorm) FindOpenForEmail(ctx context.Context, orgID uuid.UUID, email string) (*control.Invitation, error) {
	var inv control.Invitation
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND email = ? AND "+openPredicate,
			orgID, strings.ToLower(strings.TrimSpace(email)), time.Now().UTC()).
		First(&inv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

// ListForOrg is scoped by organization in the query, not by the caller.
//
// The org id comes from the principal at the handler; passing it into the
// WHERE clause here means there is no code path that returns another tenant's
// invitations even if a handler someday forgets to filter.
func (r *InvitationRepoGorm) ListForOrg(ctx context.Context, orgID uuid.UUID) ([]control.Invitation, error) {
	var out []control.Invitation
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&out).Error
	return out, err
}

// MarkAccepted is a compare-and-set on accepted_at.
//
// Two people opening the same emailed link at once both pass the lookup; only
// one may create a membership. Reporting ErrInvitationNotFound to the loser is
// correct — from its point of view the invitation is no longer open.
func (r *InvitationRepoGorm) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Model(&control.Invitation{}).
		Where("id = ? AND accepted_at IS NULL AND revoked_at IS NULL", id).
		Update("accepted_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

// Revoke takes the organization id as a predicate rather than trusting the
// caller to have checked ownership. Revoking is a mutation addressed by an id
// that came from a request, which is exactly the shape an IDOR takes.
func (r *InvitationRepoGorm) Revoke(ctx context.Context, orgID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Model(&control.Invitation{}).
		Where("id = ? AND organization_id = ? AND accepted_at IS NULL AND revoked_at IS NULL", id, orgID).
		Update("revoked_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrInvitationNotFound
	}
	return nil
}
