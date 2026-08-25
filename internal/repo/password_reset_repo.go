package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasswordResetRepo interface {
	Create(ctx context.Context, token *control.PasswordResetToken) error
	FindByTokenHash(ctx context.Context, hash string) (*control.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpiredForUser(ctx context.Context, userID uuid.UUID) error

	InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error
}

type PasswordResetRepoGorm struct {
	db *gorm.DB
}

func NewPasswordResetRepoGorm(db *gorm.DB) *PasswordResetRepoGorm {
	return &PasswordResetRepoGorm{db: db}
}

func (r *PasswordResetRepoGorm) Create(ctx context.Context, token *control.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *PasswordResetRepoGorm) FindByTokenHash(ctx context.Context, hash string) (*control.PasswordResetToken, error) {
	var t control.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now().UTC()).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *PasswordResetRepoGorm) MarkUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&control.PasswordResetToken{}).Where("id = ?", id).Update("used_at", now).Error
}

func (r *PasswordResetRepoGorm) InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&control.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL AND expires_at > ?", userID, now).
		Update("used_at", now).Error
}

func (r *PasswordResetRepoGorm) DeleteExpiredForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND (used_at IS NOT NULL OR expires_at < ?)", userID, time.Now().UTC()).Delete(&control.PasswordResetToken{}).Error
}
