package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthChallengeRepo interface {
	Create(ctx context.Context, c *control.AuthChallenge) error

	GetLive(ctx context.Context, jti uuid.UUID, now time.Time) (*control.AuthChallenge, error)

	RecordFailure(ctx context.Context, jti uuid.UUID) (int, error)

	Consume(ctx context.Context, jti uuid.UUID, now time.Time) (bool, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type AuthChallengeRepoGorm struct{ db *gorm.DB }

func NewAuthChallengeRepoGorm(db *gorm.DB) *AuthChallengeRepoGorm {
	return &AuthChallengeRepoGorm{db: db}
}

var _ AuthChallengeRepo = (*AuthChallengeRepoGorm)(nil)

func (r *AuthChallengeRepoGorm) Create(ctx context.Context, c *control.AuthChallenge) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *AuthChallengeRepoGorm) GetLive(ctx context.Context, jti uuid.UUID, now time.Time) (*control.AuthChallenge, error) {
	var c control.AuthChallenge
	err := r.db.WithContext(ctx).
		Where("jti = ? AND consumed_at IS NULL AND expires_at > ? AND attempts < max_attempts", jti, now).
		First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *AuthChallengeRepoGorm) RecordFailure(ctx context.Context, jti uuid.UUID) (int, error) {
	if err := r.db.WithContext(ctx).
		Model(&control.AuthChallenge{}).
		Where("jti = ?", jti).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error; err != nil {
		return 0, err
	}
	var c control.AuthChallenge
	if err := r.db.WithContext(ctx).Where("jti = ?", jti).First(&c).Error; err != nil {
		return 0, err
	}
	return c.Attempts, nil
}

func (r *AuthChallengeRepoGorm) Consume(ctx context.Context, jti uuid.UUID, now time.Time) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&control.AuthChallenge{}).
		Where("jti = ? AND consumed_at IS NULL AND expires_at > ?", jti, now).
		Update("consumed_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *AuthChallengeRepoGorm) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).Where("expires_at < ?", before).Delete(&control.AuthChallenge{})
	return tx.RowsAffected, tx.Error
}

type AuthFailureRepo interface {
	IsLocked(ctx context.Context, identifier, kind string, now time.Time) (bool, error)

	RecordFailure(ctx context.Context, identifier, kind string, threshold int, lockFor time.Duration, now time.Time) (count int, locked bool, err error)

	Clear(ctx context.Context, identifier, kind string) error

	DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

type AuthFailureRepoGorm struct{ db *gorm.DB }

func NewAuthFailureRepoGorm(db *gorm.DB) *AuthFailureRepoGorm {
	return &AuthFailureRepoGorm{db: db}
}

var _ AuthFailureRepo = (*AuthFailureRepoGorm)(nil)

const failureWindow = 15 * time.Minute

func (r *AuthFailureRepoGorm) IsLocked(ctx context.Context, identifier, kind string, now time.Time) (bool, error) {
	var f control.AuthFailure
	err := r.db.WithContext(ctx).
		Where("identifier = ? AND kind = ?", identifier, kind).
		First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return f.LockedUntil != nil && f.LockedUntil.After(now), nil
}

func (r *AuthFailureRepoGorm) RecordFailure(
	ctx context.Context, identifier, kind string, threshold int, lockFor time.Duration, now time.Time,
) (int, bool, error) {
	var (
		count  int
		locked bool
	)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var f control.AuthFailure

		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("identifier = ? AND kind = ?", identifier, kind).
			First(&f).Error

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):

			f = control.AuthFailure{
				Identifier:   identifier,
				Kind:         kind,
				Count:        1,
				WindowStart:  now,
				LastFailedAt: now,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "identifier"}, {Name: "kind"}},
				DoUpdates: clause.Assignments(map[string]any{
					"count":          gorm.Expr("auth_failures.count + 1"),
					"last_failed_at": now,
				}),
			}).Create(&f).Error; err != nil {
				return err
			}

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("identifier = ? AND kind = ?", identifier, kind).
				First(&f).Error; err != nil {
				return err
			}

			if threshold > 0 && f.Count >= threshold && f.LockedUntil == nil {
				until := now.Add(lockFor)
				f.LockedUntil = &until
				locked = true
				if err := tx.Model(&control.AuthFailure{}).
					Where("id = ?", f.ID).
					Update("locked_until", until).Error; err != nil {
					return err
				}
			}
			count = f.Count
			return nil
		case err != nil:
			return err
		}

		if now.Sub(f.WindowStart) > failureWindow && (f.LockedUntil == nil || !f.LockedUntil.After(now)) {
			f.Count = 0
			f.WindowStart = now
			f.LockedUntil = nil
		}
		f.Count++
		f.LastFailedAt = now

		if threshold > 0 && f.Count >= threshold {
			until := now.Add(lockFor)
			f.LockedUntil = &until
			locked = true
		}
		count = f.Count
		return tx.Save(&f).Error
	})
	if err != nil {
		return 0, false, err
	}
	return count, locked, nil
}

func (r *AuthFailureRepoGorm) Clear(ctx context.Context, identifier, kind string) error {
	return r.db.WithContext(ctx).
		Where("identifier = ? AND kind = ?", identifier, kind).
		Delete(&control.AuthFailure{}).Error
}

func (r *AuthFailureRepoGorm) DeleteStale(ctx context.Context, before time.Time) (int64, error) {

	tx := r.db.WithContext(ctx).
		Where("last_failed_at < ? AND (locked_until IS NULL OR locked_until < ?)", before, before).
		Delete(&control.AuthFailure{})
	return tx.RowsAffected, tx.Error
}
