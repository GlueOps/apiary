package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrAPIKeyNotFound = errors.New("api key: not found")

type APIKeyRepo interface {
	Create(ctx context.Context, k *control.APIKey) error
	ByKeyHash(ctx context.Context, hash string) (*control.APIKey, error)
	TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	ListForOrg(ctx context.Context, orgID uuid.UUID) ([]control.APIKey, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]control.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type APIKeyRepoGorm struct{ db *gorm.DB }

func NewAPIKeyRepoGorm(db *gorm.DB) *APIKeyRepoGorm { return &APIKeyRepoGorm{db: db} }

func (r *APIKeyRepoGorm) Create(ctx context.Context, k *control.APIKey) error {
	return r.db.WithContext(ctx).Create(k).Error
}

// ByKeyHash finds a key by the hash of its identifying half.
//
// It deliberately does NOT filter on revoked or expired. Those are decided by
// the caller against control.APIKey.Usable, so that "no such key" and "a key
// that exists but may not be used" stay distinguishable in the logs — while the
// RESPONSE to the client remains identical either way.
func (r *APIKeyRepoGorm) ByKeyHash(ctx context.Context, hash string) (*control.APIKey, error) {
	if strings.TrimSpace(hash) == "" {
		return nil, ErrAPIKeyNotFound
	}
	var k control.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&k).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, err
	}
	return &k, nil
}

// TouchLastUsed records that a key was presented.
//
// Best-effort and non-blocking for the caller: a write failure here must never
// fail an authenticated request, because the audit value of last_used_at is
// lower than the cost of refusing a valid credential over it.
func (r *APIKeyRepoGorm) TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&control.APIKey{}).
		Where("id = ?", id).
		UpdateColumn("last_used_at", at.UTC()).Error
}

func (r *APIKeyRepoGorm) ListForOrg(ctx context.Context, orgID uuid.UUID) ([]control.APIKey, error) {
	var out []control.APIKey
	err := r.db.WithContext(ctx).
		Where("org_id = ?", orgID).Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *APIKeyRepoGorm) ListForUser(ctx context.Context, userID uuid.UUID) ([]control.APIKey, error) {
	var out []control.APIKey
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).Order("created_at DESC").Find(&out).Error
	return out, err
}

// Revoke is idempotent: revoking an already-revoked key is the state the caller
// wanted, and erroring would make a retried request look like a failure.
func (r *APIKeyRepoGorm) Revoke(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&control.APIKey{}).
		Where("id = ?", id).Update("revoked", true).Error
}
