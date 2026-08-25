package repo

import (
	"context"
	"errors"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrganizationRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*control.Organization, error)
	List(ctx context.Context) ([]control.Organization, error)
	Create(ctx context.Context, org *control.Organization) error
	Update(ctx context.Context, id uuid.UUID, updates map[string]any) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type OrganizationRepoGorm struct {
	db *gorm.DB
}

func NewOrganizationRepoGorm(db *gorm.DB) OrganizationRepo {
	return &OrganizationRepoGorm{db: db}
}

func (r *OrganizationRepoGorm) List(ctx context.Context) ([]control.Organization, error) {
	var orgs []control.Organization
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&orgs).Error
	return orgs, err
}

func (r *OrganizationRepoGorm) Create(ctx context.Context, org *control.Organization) error {
	return r.db.WithContext(ctx).Create(org).Error
}

func (r *OrganizationRepoGorm) GetByID(ctx context.Context, id uuid.UUID) (*control.Organization, error) {
	var org control.Organization
	err := r.db.WithContext(ctx).First(&org, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &org, err
}

func (r *OrganizationRepoGorm) Update(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&control.Organization{}).Where("id = ?", id).Updates(updates).Error
}

func (r *OrganizationRepoGorm) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&control.Organization{}, "id = ?", id).Error
}
