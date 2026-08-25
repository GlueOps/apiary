package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenSessionRepo interface {
	Create(ctx context.Context, s *control.TokenSession) error
	GetByRefreshJTI(ctx context.Context, jti string) (*control.TokenSession, error)
	GetByAccessJTI(ctx context.Context, jti string) (*control.TokenSession, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeByFamilyID(ctx context.Context, familyID uuid.UUID, at time.Time) (int64, error)
	RevokeAllBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, at time.Time) (int64, error)
	ListActiveBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string) ([]control.TokenSession, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type TokenSessionRepoGorm struct {
	db *gorm.DB
}

func NewTokenSessionRepoGorm(db *gorm.DB) *TokenSessionRepoGorm {
	return &TokenSessionRepoGorm{db: db}
}

var _ TokenSessionRepo = (*TokenSessionRepoGorm)(nil)

func (r *TokenSessionRepoGorm) Create(ctx context.Context, s *control.TokenSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *TokenSessionRepoGorm) GetByRefreshJTI(ctx context.Context, jti string) (*control.TokenSession, error) {
	var s control.TokenSession
	err := r.db.WithContext(ctx).First(&s, "refresh_token_jti = ?", jti).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func (r *TokenSessionRepoGorm) GetByAccessJTI(ctx context.Context, jti string) (*control.TokenSession, error) {
	var s control.TokenSession
	err := r.db.WithContext(ctx).First(&s, "access_token_jti = ?", jti).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func (r *TokenSessionRepoGorm) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.TokenSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", at).Error
}

func (r *TokenSessionRepoGorm) RevokeByFamilyID(ctx context.Context, familyID uuid.UUID, at time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&control.TokenSession{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", at)
	return tx.RowsAffected, tx.Error
}

func (r *TokenSessionRepoGorm) RevokeAllBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, at time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&control.TokenSession{}).
		Where("subject_id = ? AND subject_kind = ? AND revoked_at IS NULL", subjectID, subjectKind).
		Update("revoked_at", at)
	return tx.RowsAffected, tx.Error
}

func (r *TokenSessionRepoGorm) ListActiveBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string) ([]control.TokenSession, error) {
	var sessions []control.TokenSession
	err := r.db.WithContext(ctx).
		Where("subject_id = ? AND subject_kind = ? AND revoked_at IS NULL AND refresh_expires_at > ?", subjectID, subjectKind, time.Now().UTC()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *TokenSessionRepoGorm) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).
		Where("refresh_expires_at < ? AND access_expires_at < ?", before, before).
		Delete(&control.TokenSession{})
	return tx.RowsAffected, tx.Error
}
