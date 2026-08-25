package repo

import (
	"context"
	"errors"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlatformPolicyRepo interface {
	GetByKind(ctx context.Context, kind control.PolicyKind) (*control.PlatformPolicy, error)
	List(ctx context.Context) ([]control.PlatformPolicy, error)
	Upsert(ctx context.Context, policy *control.PlatformPolicy) error
	Delete(ctx context.Context, kind control.PolicyKind) error
}

type PlatformPolicyRepoGorm struct {
	db *gorm.DB
}

func NewPlatformPolicyRepoGorm(db *gorm.DB) *PlatformPolicyRepoGorm {
	return &PlatformPolicyRepoGorm{db: db}
}

var _ PlatformPolicyRepo = (*PlatformPolicyRepoGorm)(nil)

func (r *PlatformPolicyRepoGorm) GetByKind(ctx context.Context, kind control.PolicyKind) (*control.PlatformPolicy, error) {
	var p control.PlatformPolicy
	err := r.db.WithContext(ctx).First(&p, "kind = ?", kind).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *PlatformPolicyRepoGorm) List(ctx context.Context) ([]control.PlatformPolicy, error) {
	var policies []control.PlatformPolicy
	err := r.db.WithContext(ctx).Order("kind ASC").Find(&policies).Error
	return policies, err
}

func (r *PlatformPolicyRepoGorm) Upsert(ctx context.Context, policy *control.PlatformPolicy) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "kind"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "config", "created_by", "updated_at"}),
		}).
		Create(policy).Error
}

func (r *PlatformPolicyRepoGorm) Delete(ctx context.Context, kind control.PolicyKind) error {
	result := r.db.WithContext(ctx).Delete(&control.PlatformPolicy{}, "kind = ?", kind)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
