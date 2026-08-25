package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailChangeRepo interface {
	Create(ctx context.Context, r *control.EmailChangeRequest) error

	Consume(ctx context.Context, hash string, now time.Time) (*control.EmailChangeRequest, error)

	InvalidateForUser(ctx context.Context, userID uuid.UUID) error
}

type EmailChangeRepoGorm struct{ db *gorm.DB }

func NewEmailChangeRepoGorm(db *gorm.DB) *EmailChangeRepoGorm { return &EmailChangeRepoGorm{db: db} }

var _ EmailChangeRepo = (*EmailChangeRepoGorm)(nil)

func (r *EmailChangeRepoGorm) Create(ctx context.Context, req *control.EmailChangeRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *EmailChangeRepoGorm) Consume(ctx context.Context, hash string, now time.Time) (*control.EmailChangeRequest, error) {
	var out control.EmailChangeRequest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&control.EmailChangeRequest{}).
			Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("token_hash = ?", hash).First(&out).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *EmailChangeRepoGorm) InvalidateForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&control.EmailChangeRequest{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}
