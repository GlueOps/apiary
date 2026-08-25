package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserPasskeyRepo interface {
	Create(ctx context.Context, p *control.UserPasskey) error
	GetByCredentialID(ctx context.Context, credID string) (*control.UserPasskey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]control.UserPasskey, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	UpdateSignCount(ctx context.Context, id uuid.UUID, count uint32, usedAt time.Time) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type UserPasskeyRepoGorm struct {
	db *gorm.DB
}

func NewUserPasskeyRepoGorm(db *gorm.DB) *UserPasskeyRepoGorm {
	return &UserPasskeyRepoGorm{db: db}
}

func (r *UserPasskeyRepoGorm) Create(ctx context.Context, p *control.UserPasskey) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *UserPasskeyRepoGorm) GetByCredentialID(ctx context.Context, credID string) (*control.UserPasskey, error) {
	var p control.UserPasskey
	err := r.db.WithContext(ctx).First(&p, "credential_id = ?", credID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *UserPasskeyRepoGorm) ListByUserID(ctx context.Context, userID uuid.UUID) ([]control.UserPasskey, error) {
	var keys []control.UserPasskey
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&keys).Error
	return keys, err
}

func (r *UserPasskeyRepoGorm) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&control.UserPasskey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserPasskeyRepoGorm) UpdateSignCount(ctx context.Context, id uuid.UUID, count uint32, usedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.UserPasskey{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"sign_count":   count,
			"last_used_at": usedAt,
		}).Error
}

func (r *UserPasskeyRepoGorm) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&control.UserPasskey{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
