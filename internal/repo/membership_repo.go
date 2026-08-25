package repo

import (
	"context"
	"errors"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MembershipRepo interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]control.Membership, error)
	ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]control.Membership, error)
	GetByID(ctx context.Context, id uuid.UUID) (*control.Membership, error)
	GetByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*control.Membership, error)
	Create(ctx context.Context, membership *control.Membership) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateRole(ctx context.Context, id uuid.UUID, role control.MemberRole) error
	UpdateSupportEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
}

type MembershipRepoGorm struct {
	db *gorm.DB
}

func NewMembershipRepoGorm(db *gorm.DB) MembershipRepo {
	return &MembershipRepoGorm{db: db}
}

func (r *MembershipRepoGorm) ListByUserID(ctx context.Context, userID uuid.UUID) ([]control.Membership, error) {
	var out []control.Membership
	err := r.db.WithContext(ctx).
		Preload("Organization").
		Where("user_id = ?", userID).
		Order("created_at asc").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MembershipRepoGorm) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]control.Membership, error) {
	var out []control.Membership
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("organization_id = ?", orgID).
		Order("created_at asc").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *MembershipRepoGorm) GetByID(ctx context.Context, id uuid.UUID) (*control.Membership, error) {
	var out control.Membership
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Organization").
		Where("id = ?", id).
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *MembershipRepoGorm) GetByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*control.Membership, error) {
	var out control.Membership
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &out, err
}

func (r *MembershipRepoGorm) Create(ctx context.Context, membership *control.Membership) error {
	return r.db.WithContext(ctx).Create(membership).Error
}

func (r *MembershipRepoGorm) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&control.Membership{}, "id = ?", id).Error
}

func (r *MembershipRepoGorm) UpdateRole(ctx context.Context, id uuid.UUID, role control.MemberRole) error {
	return r.db.WithContext(ctx).
		Model(&control.Membership{}).
		Where("id = ?", id).
		Update("role", role).Error
}

func (r *MembershipRepoGorm) UpdateSupportEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&control.Membership{}).
		Where("id = ?", id).
		Update("support_enabled", enabled).Error
}
