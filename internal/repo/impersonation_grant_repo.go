package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImpersonationGrantRepo interface {
	Create(ctx context.Context, g *control.ImpersonationGrant) error
	GetActiveGrant(ctx context.Context, targetUserID, orgID uuid.UUID) (*control.ImpersonationGrant, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
}

type ImpersonationGrantRepoGorm struct {
	db *gorm.DB
}

func NewImpersonationGrantRepoGorm(db *gorm.DB) *ImpersonationGrantRepoGorm {
	return &ImpersonationGrantRepoGorm{db: db}
}

var _ ImpersonationGrantRepo = (*ImpersonationGrantRepoGorm)(nil)

func (r *ImpersonationGrantRepoGorm) Create(ctx context.Context, g *control.ImpersonationGrant) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *ImpersonationGrantRepoGorm) GetActiveGrant(ctx context.Context, targetUserID, orgID uuid.UUID) (*control.ImpersonationGrant, error) {
	var g control.ImpersonationGrant
	err := r.db.WithContext(ctx).
		Where("target_user_id = ? AND organization_id = ? AND revoked_at IS NULL AND expires_at > ?", targetUserID, orgID, time.Now().UTC()).
		First(&g).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &g, err
}

func (r *ImpersonationGrantRepoGorm) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.ImpersonationGrant{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", at).Error
}
