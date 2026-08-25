package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailVerificationRepo interface {
	Create(ctx context.Context, token *control.EmailVerificationToken) error
	FindByTokenHash(ctx context.Context, hash string) (*control.EmailVerificationToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredForUser(ctx context.Context, userID uuid.UUID) error
	CountRecentForUser(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error)
}

type EmailVerificationRepoGorm struct {
	db *gorm.DB
}

func NewEmailVerificationRepoGorm(db *gorm.DB) *EmailVerificationRepoGorm {
	return &EmailVerificationRepoGorm{db: db}
}

func (r *EmailVerificationRepoGorm) Create(ctx context.Context, token *control.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindByTokenHash returns only a token that is live: unused and unexpired. The
// predicate is in the query rather than in the caller so that "is this token
// usable" has exactly one expression. A caller that fetched the row and checked
// the fields itself would be one forgotten condition away from accepting a used
// token, and that check is the whole security property here.
func (r *EmailVerificationRepoGorm) FindByTokenHash(ctx context.Context, hash string) (*control.EmailVerificationToken, error) {
	var t control.EmailVerificationToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now().UTC()).
		First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// MarkUsed burns the token, and reports whether it actually burned one.
//
// The guard on used_at makes this a compare-and-set: two requests racing with
// the same token both pass FindByTokenHash, but only one UPDATE matches a row.
// Without it both would proceed, which for a single confirmation is harmless
// and for anything later built on "this token was redeemed exactly once" is not.
func (r *EmailVerificationRepoGorm) MarkUsed(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Model(&control.EmailVerificationToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *EmailVerificationRepoGorm) InvalidateAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&control.EmailVerificationToken{}).
		Where("user_id = ? AND used_at IS NULL AND expires_at > ?", userID, now).
		Update("used_at", now).Error
}

func (r *EmailVerificationRepoGorm) DeleteExpiredForUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND (used_at IS NOT NULL OR expires_at < ?)", userID, time.Now().UTC()).
		Delete(&control.EmailVerificationToken{}).Error
}

// CountRecentForUser backs the resend throttle. It counts issued tokens rather
// than requests, so it cannot be reset by anything the caller controls.
func (r *EmailVerificationRepoGorm) CountRecentForUser(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Model(&control.EmailVerificationToken{}).
		Where("user_id = ? AND created_at > ?", userID, since).
		Count(&n).Error
	return n, err
}
