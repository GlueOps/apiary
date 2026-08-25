package repo

import (
	"context"
	"errors"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrgPolicyRepo interface {
	GetByOrgAndKind(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) (*control.OrgPolicy, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]control.OrgPolicy, error)
	Upsert(ctx context.Context, policy *control.OrgPolicy) error
	Delete(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) error
}

type OrgPolicyRepoGorm struct {
	db *gorm.DB
}

func NewOrgPolicyRepoGorm(db *gorm.DB) *OrgPolicyRepoGorm {
	return &OrgPolicyRepoGorm{db: db}
}

var _ OrgPolicyRepo = (*OrgPolicyRepoGorm)(nil)

func (r *OrgPolicyRepoGorm) GetByOrgAndKind(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) (*control.OrgPolicy, error) {
	var p control.OrgPolicy
	err := r.db.WithContext(ctx).First(&p, "organization_id = ? AND kind = ?", orgID, kind).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *OrgPolicyRepoGorm) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]control.OrgPolicy, error) {
	var policies []control.OrgPolicy
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("kind ASC").Find(&policies).Error
	return policies, err
}

func (r *OrgPolicyRepoGorm) Upsert(ctx context.Context, policy *control.OrgPolicy) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "organization_id"}, {Name: "kind"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "config", "created_by", "updated_at"}),
		}).
		Create(policy).Error
}

func (r *OrgPolicyRepoGorm) Delete(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) error {
	result := r.db.WithContext(ctx).Delete(&control.OrgPolicy{}, "organization_id = ? AND kind = ?", orgID, kind)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
