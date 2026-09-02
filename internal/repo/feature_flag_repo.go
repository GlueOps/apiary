package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/glueops/apiary/internal/models/control"
)

// FeatureFlagRepo stores the two levels a flag can be set at. It knows nothing
// about which keys are legal — that is the registry's job, checked above this.
type FeatureFlagRepo interface {
	GetPlatform(ctx context.Context, key string) (*control.PlatformFeatureFlag, error)
	ListPlatform(ctx context.Context) ([]control.PlatformFeatureFlag, error)
	SetPlatform(ctx context.Context, key string, value bool, by uuid.UUID) error
	ClearPlatform(ctx context.Context, key string) error

	GetOrg(ctx context.Context, orgID uuid.UUID, key string) (*control.OrgFeatureFlag, error)
	ListOrg(ctx context.Context, orgID uuid.UUID) ([]control.OrgFeatureFlag, error)
	// ListOrgOverridesFor answers "which organizations have this on" without a
	// LIKE over a composed key — the query the dotted notation would have made
	// awkward if the tenant lived in the key string.
	ListOrgOverridesFor(ctx context.Context, key string) ([]FlagOverride, error)
	SetOrg(ctx context.Context, orgID uuid.UUID, key string, value bool, by uuid.UUID) error
	ClearOrg(ctx context.Context, orgID uuid.UUID, key string) error
}

type FeatureFlagRepoGorm struct{ db *gorm.DB }

func NewFeatureFlagRepoGorm(db *gorm.DB) *FeatureFlagRepoGorm { return &FeatureFlagRepoGorm{db: db} }

var _ FeatureFlagRepo = (*FeatureFlagRepoGorm)(nil)

// A MISSING ROW IS (nil, nil), NEVER AN ERROR, at both levels.
//
// Absence is the ordinary case — most flags are never overridden anywhere — and
// it means "fall through to the next level", which resolution needs to be able
// to act on. Returning ErrRecordNotFound would make every caller unwrap it, and
// the caller that forgot would report an unset flag as a failure.
func (r *FeatureFlagRepoGorm) GetPlatform(ctx context.Context, key string) (*control.PlatformFeatureFlag, error) {
	var f control.PlatformFeatureFlag
	err := r.db.WithContext(ctx).First(&f, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FeatureFlagRepoGorm) ListPlatform(ctx context.Context) ([]control.PlatformFeatureFlag, error) {
	var out []control.PlatformFeatureFlag
	err := r.db.WithContext(ctx).Order("key ASC").Find(&out).Error
	return out, err
}

func (r *FeatureFlagRepoGorm) SetPlatform(ctx context.Context, key string, value bool, by uuid.UUID) error {
	// Columns named explicitly on conflict. Assigning the whole struct would
	// rewrite created_at with the new row's default and lose when the flag was
	// first set, which is the one piece of history this table carries.
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&control.PlatformFeatureFlag{Key: key, Value: value, UpdatedBy: by}).Error
}

// ClearPlatform removes the row so the flag falls back to its compiled default.
//
// Deleting nothing is SUCCESS. "Make sure this is not overridden" is satisfied
// by a row that was never there, and reporting not-found would make an operator
// investigate a state they had just asked for.
func (r *FeatureFlagRepoGorm) ClearPlatform(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Delete(&control.PlatformFeatureFlag{}, "key = ?", key).Error
}

func (r *FeatureFlagRepoGorm) GetOrg(ctx context.Context, orgID uuid.UUID, key string) (*control.OrgFeatureFlag, error) {
	if orgID == uuid.Nil {
		// No organization is not the nil organization. Querying for it would
		// match nothing, but only by accident.
		return nil, nil
	}
	var f control.OrgFeatureFlag
	err := r.db.WithContext(ctx).First(&f, "organization_id = ? AND key = ?", orgID, key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FeatureFlagRepoGorm) ListOrg(ctx context.Context, orgID uuid.UUID) ([]control.OrgFeatureFlag, error) {
	if orgID == uuid.Nil {
		return nil, nil
	}
	var out []control.OrgFeatureFlag
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("key ASC").Find(&out).Error
	return out, err
}

// FlagOverride is one organization's override WITH ITS NAME.
//
// The name is joined here rather than resolved by the caller. This list exists
// to be read by a person deciding whether to change an installation-wide
// default, and a column of bare uuids does not answer "who am I about to not
// affect" — while a per-row lookup from a UI would be one request per tenant
// against an endpoint that does not exist.
type FlagOverride struct {
	OrganizationID   uuid.UUID
	OrganizationName string
	Value            bool
}

// LEFT JOIN, so an override whose organization has been deleted still appears.
// The foreign key cascades, so that should be impossible — but a row that
// somehow outlived its organization is exactly the row worth seeing, and an
// inner join would hide it.
func (r *FeatureFlagRepoGorm) ListOrgOverridesFor(ctx context.Context, key string) ([]FlagOverride, error) {
	var out []FlagOverride
	err := r.db.WithContext(ctx).
		Table("org_feature_flags AS f").
		Select("f.organization_id, f.value, coalesce(o.name, '') AS organization_name").
		Joins("LEFT JOIN organizations o ON o.id = f.organization_id").
		Where("f.key = ?", key).
		Order("organization_name ASC, f.organization_id ASC").
		Scan(&out).Error
	return out, err
}

func (r *FeatureFlagRepoGorm) SetOrg(ctx context.Context, orgID uuid.UUID, key string, value bool, by uuid.UUID) error {
	if orgID == uuid.Nil {
		return errors.New("an organization override needs an organization")
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
	}).Create(&control.OrgFeatureFlag{
		OrganizationID: orgID, Key: key, Value: value, UpdatedBy: by,
	}).Error
}

func (r *FeatureFlagRepoGorm) ClearOrg(ctx context.Context, orgID uuid.UUID, key string) error {
	if orgID == uuid.Nil {
		return errors.New("an organization override needs an organization")
	}
	return r.db.WithContext(ctx).
		Delete(&control.OrgFeatureFlag{}, "organization_id = ? AND key = ?", orgID, key).Error
}
