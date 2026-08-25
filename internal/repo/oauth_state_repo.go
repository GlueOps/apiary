package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
)

type OAuthStateRepo interface {
	Create(ctx context.Context, s *control.OAuthState) error

	Consume(ctx context.Context, hash string, now time.Time) (*control.OAuthState, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type OAuthStateRepoGorm struct {
	db *gorm.DB
}

func NewOAuthStateRepoGorm(db *gorm.DB) *OAuthStateRepoGorm {
	return &OAuthStateRepoGorm{db: db}
}

var _ OAuthStateRepo = (*OAuthStateRepoGorm)(nil)

func (r *OAuthStateRepoGorm) Create(ctx context.Context, s *control.OAuthState) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *OAuthStateRepoGorm) Consume(ctx context.Context, hash string, now time.Time) (*control.OAuthState, error) {
	var out control.OAuthState

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		res := tx.Model(&control.OAuthState{}).
			Where("state_hash = ? AND consumed_at IS NULL AND expires_at > ?", hash, now).
			Update("consumed_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {

			return gorm.ErrRecordNotFound
		}
		return tx.Where("state_hash = ?", hash).First(&out).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *OAuthStateRepoGorm) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&control.OAuthState{})
	return tx.RowsAffected, tx.Error
}
