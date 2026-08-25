package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrSoleOwner refuses to delete an account that is the only owner of an
	// organization. Not a formality: the organization would be left with nobody
	// who can add an owner, change a role, or delete it — an unadministrable
	// tenant that only platform staff can rescue.
	ErrSoleOwner = errors.New("account: this account is the only owner of an organization")

	// ErrConfirmationMismatch refuses a destructive action whose confirmation
	// does not match.
	ErrConfirmationMismatch = errors.New("account: the confirmation does not match")

	ErrOrgNotFound     = errors.New("account: no such organization")
	ErrAccountNotFound = errors.New("account: no such account")
)

// AccountDeletionService owns the two irreversible-looking operations: deleting
// a person, and deleting an organization.
//
// Both are SOFT deletes, for different reasons that happen to point the same
// way — see Organization.DeletedAt and User.DeletedAt. Neither destroys key
// material or a database; purging is a separate operator step, which is what
// leaves a way back from a mistake made at three in the morning.
type AccountDeletionService struct {
	db *gorm.DB
}

func NewAccountDeletionService(db *gorm.DB) *AccountDeletionService {
	return &AccountDeletionService{db: db}
}

// SoleOwnerships lists the organizations where this account is the ONLY owner.
//
// Returned rather than merely counted so the caller can name them: "you are the
// only owner of Acme and Foo" is actionable, "you cannot delete your account" is
// not.
func (s *AccountDeletionService) SoleOwnerships(ctx context.Context, userID uuid.UUID) ([]control.Organization, error) {
	var orgs []control.Organization
	err := s.db.WithContext(ctx).
		Raw(`
			SELECT o.*
			FROM organizations o
			JOIN memberships m ON m.organization_id = o.id
			WHERE m.user_id = ? AND m.role = ? AND o.deleted_at IS NULL
			  AND (
				SELECT COUNT(*) FROM memberships m2
				JOIN users u2 ON u2.id = m2.user_id
				WHERE m2.organization_id = o.id AND m2.role = ?
				  AND u2.deleted_at IS NULL
			  ) = 1
			ORDER BY o.name ASC`,
			userID, control.MemberRoleOwner, control.MemberRoleOwner).
		Scan(&orgs).Error
	return orgs, err
}

// DeleteAccount soft-deletes a user.
//
// It refuses while the account is the sole owner of any organization. The way
// out is to promote another owner or delete the organization — both deliberate
// acts, and both leaving the tenant administrable either way.
//
// One transaction: the checks and the write must not be separated, or an
// ownership change landing between them lets a sole owner delete themselves
// anyway.
func (s *AccountDeletionService) DeleteAccount(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user control.User
		if err := tx.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAccountNotFound
			}
			return err
		}

		// Counted inside the transaction, against rows locked for the duration.
		// Reading this outside would be a TOCTOU: the other owner resigns between
		// the check and the write, and the organization is left ownerless.
		var blocking int64
		if err := tx.Raw(`
			SELECT COUNT(*)
			FROM memberships m
			JOIN organizations o ON o.id = m.organization_id
			WHERE m.user_id = ? AND m.role = ? AND o.deleted_at IS NULL
			  AND (
				SELECT COUNT(*) FROM memberships m2
				JOIN users u2 ON u2.id = m2.user_id
				WHERE m2.organization_id = o.id AND m2.role = ?
				  AND u2.deleted_at IS NULL
			  ) = 1`,
			userID, control.MemberRoleOwner, control.MemberRoleOwner).
			Scan(&blocking).Error; err != nil {
			return err
		}
		if blocking > 0 {
			return fmt.Errorf("%w: %d organization(s)", ErrSoleOwner, blocking)
		}

		if err := tx.Model(&control.User{}).
			Where("id = ?", userID).
			Updates(map[string]any{"deleted_at": now, "is_active": false}).Error; err != nil {
			return err
		}

		// Every live session ends now. A soft delete that leaves a valid token
		// working is not a deletion — the account is gone from the UI and still
		// making authenticated requests.
		if err := tx.Model(&control.TokenSession{}).
			Where("subject_id = ? AND subject_kind = ? AND revoked_at IS NULL", userID, "user").
			Update("revoked_at", now).Error; err != nil {
			return err
		}

		// Authentication factors go with the account. Leaving a TOTP secret or a
		// passkey behind means the credentials outlive the thing they authenticate.
		if err := tx.Where("user_id = ?", userID).Delete(&control.UserTOTP{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&control.UserPasskey{}).Error
	})
}

// DeleteOrganization soft-deletes an organization.
//
// confirmName must equal the organization's name exactly. Not decoration: this
// is the single call that takes a customer's entire tenant out of service, and
// typing the name is the difference between meaning it and clicking the wrong
// row. Compared case-sensitively after trimming, because "acme" and "Acme" may
// be two different organizations.
func (s *AccountDeletionService) DeleteOrganization(ctx context.Context, orgID uuid.UUID, confirmName string, now time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var org control.Organization
		if err := tx.Where("id = ? AND deleted_at IS NULL", orgID).First(&org).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrgNotFound
			}
			return err
		}
		if strings.TrimSpace(confirmName) != org.Name {
			return ErrConfirmationMismatch
		}

		if err := tx.Model(&control.Organization{}).
			Where("id = ?", orgID).
			Updates(map[string]any{"deleted_at": now, "status": "deleted"}).Error; err != nil {
			return err
		}

		// Sessions scoped to this organization end. Memberships are left in place:
		// they are what a restore needs, and they are how the audit trail explains
		// who could reach the tenant while it existed.
		return tx.Model(&control.TokenSession{}).
			Where("organization_id = ? AND revoked_at IS NULL", orgID).
			Update("revoked_at", now).Error
	})
}
