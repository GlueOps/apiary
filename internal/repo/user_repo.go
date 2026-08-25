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

type UpdateUserProfileInput struct {
	Email     *string
	FirstName *string
	LastName  *string
	Password  *string
}

type UserRepo interface {
	Create(ctx context.Context, u *control.User) error
	GetByEmail(ctx context.Context, email string) (*control.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*control.User, error)
	List(ctx context.Context) ([]control.User, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID, at time.Time) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateUserProfileInput) error
	SetMFAEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error
	UpdateTokensRevokedAt(ctx context.Context, userID uuid.UUID, at time.Time) error
	MarkEmailVerified(ctx context.Context, userID uuid.UUID, at time.Time) error
	CountUsersWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error)
	CountUsersWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error)
	BulkRevokeTokensWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error)
	BulkRevokeTokensWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error)
}

type UserRepoGorm struct {
	db *gorm.DB
}

func NewUserRepoGorm(db *gorm.DB) UserRepoGorm {
	return UserRepoGorm{db: db}
}

func (r *UserRepoGorm) Create(ctx context.Context, u *control.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByEmail excludes deleted accounts.
//
// A deleted account that can still be found by email can still be logged into,
// which makes the deletion cosmetic. Excluding it here also means the email
// becomes free again for a genuine re-registration, rather than colliding with
// a row nobody can reach.
func (r *UserRepoGorm) GetByEmail(ctx context.Context, email string) (*control.User, error) {
	var u control.User
	err := r.db.WithContext(ctx).First(&u, "email = ? AND deleted_at IS NULL", email).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// GetByID excludes deleted accounts, so that every middleware resolving a
// principal — platform role, admin gate — refuses one.
func (r *UserRepoGorm) GetByID(ctx context.Context, id uuid.UUID) (*control.User, error) {
	var u control.User
	err := r.db.WithContext(ctx).
		First(&u, "id = ? AND deleted_at IS NULL", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepoGorm) List(ctx context.Context) ([]control.User, error) {
	var users []control.User
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&users).Error
	return users, err
}

func (r *UserRepoGorm) UpdateLastLogin(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.User{}).
		Where("id = ?", userID).
		Update("last_login_at", at).
		Error
}

func (r *UserRepoGorm) SetMFAEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&control.User{}).
		Where("id = ?", userID).
		Update("mfa_enabled", enabled).
		Error
}

func (r *UserRepoGorm) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateUserProfileInput) error {
	updates := map[string]any{}

	if in.Email != nil {
		updates["email"] = strings.ToLower(strings.TrimSpace(*in.Email))
	}
	if in.FirstName != nil {
		updates["first_name"] = strings.TrimSpace(*in.FirstName)
	}
	if in.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*in.LastName)
	}
	if in.Password != nil {
		u := &control.User{}
		if err := u.SetPassword(*in.Password); err != nil {
			return err
		}
		updates["password"] = u.Password
	}

	if len(updates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&control.User{}).
		Where("id = ?", userID).
		Updates(updates).
		Error
}

func (r *UserRepoGorm) UpdateTokensRevokedAt(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.User{}).
		Where("id = ?", userID).
		Update("tokens_revoked_at", at).
		Error
}

func (r *UserRepoGorm) CountUsersWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&control.User{}).Where("mfa_enabled = false AND is_active = true")
	if orgID != nil {
		q = q.Where("id IN (SELECT user_id FROM memberships WHERE organization_id = ?)", *orgID)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *UserRepoGorm) CountUsersWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&control.User{}).
		Where("is_active = true").
		Where("id NOT IN (SELECT user_id FROM user_passkeys)")
	if orgID != nil {
		q = q.Where("id IN (SELECT user_id FROM memberships WHERE organization_id = ?)", *orgID)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *UserRepoGorm) BulkRevokeTokensWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	q := r.db.WithContext(ctx).Model(&control.User{}).
		Where("mfa_enabled = false AND is_active = true")
	if orgID != nil {
		q = q.Where("id IN (SELECT user_id FROM memberships WHERE organization_id = ?)", *orgID)
	}
	result := q.Update("tokens_revoked_at", time.Now().UTC())
	return result.RowsAffected, result.Error
}

func (r *UserRepoGorm) BulkRevokeTokensWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	q := r.db.WithContext(ctx).Model(&control.User{}).
		Where("is_active = true").
		Where("id NOT IN (SELECT user_id FROM user_passkeys)")
	if orgID != nil {
		q = q.Where("id IN (SELECT user_id FROM memberships WHERE organization_id = ?)", *orgID)
	}
	result := q.Update("tokens_revoked_at", time.Now().UTC())
	return result.RowsAffected, result.Error
}

// MarkEmailVerified stamps the moment this user proved control of their address.
//
// The email_verified_at IS NULL guard makes the first confirmation win and
// every later one a no-op, so a replayed link cannot move the timestamp
// forward. That matters because the admission ceiling reasons about how long an
// organization has been real; a timestamp a caller can refresh at will is not
// evidence of anything.
//
// A no-op is reported as success, not as an error. Confirming an already
// confirmed address is the state the caller wanted, and callers that treat it
// as a failure end up showing an error to someone who simply clicked twice.
func (r *UserRepoGorm) MarkEmailVerified(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&control.User{}).
		Where("id = ? AND email_verified_at IS NULL", userID).
		Update("email_verified_at", at.UTC()).Error
}
