package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserTOTPRepo interface {
	Create(ctx context.Context, t *control.UserTOTP) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*control.UserTOTP, error)
	MarkVerified(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	SetLastUsedStep(ctx context.Context, id uuid.UUID, step int64) (bool, error)
}

type UserTOTPRepoGorm struct {
	db *gorm.DB
}

func NewUserTOTPRepoGorm(db *gorm.DB) *UserTOTPRepoGorm {
	return &UserTOTPRepoGorm{db: db}
}

func (r *UserTOTPRepoGorm) Create(ctx context.Context, t *control.UserTOTP) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *UserTOTPRepoGorm) GetByUserID(ctx context.Context, userID uuid.UUID) (*control.UserTOTP, error) {
	var t control.UserTOTP
	err := r.db.WithContext(ctx).First(&t, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *UserTOTPRepoGorm) MarkVerified(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&control.UserTOTP{}).
		Where("id = ?", id).
		Update("verified", true).
		Error
}

func (r *UserTOTPRepoGorm) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&control.UserTOTP{}).
		Error
}

func (r *UserTOTPRepoGorm) SetLastUsedStep(ctx context.Context, id uuid.UUID, step int64) (bool, error) {

	if step <= 0 {
		return false, fmt.Errorf("user_totp: refusing non-positive totp step %d", step)
	}

	res := r.db.WithContext(ctx).
		Model(&control.UserTOTP{}).
		Where("id = ? AND last_used_step < ?", id, step).
		Update("last_used_step", step)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
