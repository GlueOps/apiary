package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrSocialLinkNotAllowed = errors.New("an account with this email already exists; sign in with that account and link the provider from account settings")

var ErrIdentityEmailNotVerified = errors.New("social identity has no verified email")

type UpsertSocialIdentityInput struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	Profile       map[string]any
	LinkByEmail   bool
}

type SocialAuthRepo interface {
	UpsertFromIdentity(ctx context.Context, in UpsertSocialIdentityInput) (*control.User, error)
	EnsureAutoMembership(ctx context.Context, userID uuid.UUID, email string) error
}

type SocialAuthRepoGorm struct {
	db *gorm.DB
}

func NewSocialAuthRepoGorm(db *gorm.DB) SocialAuthRepo {
	return &SocialAuthRepoGorm{db: db}
}

func (r *SocialAuthRepoGorm) UpsertFromIdentity(ctx context.Context, in UpsertSocialIdentityInput) (*control.User, error) {
	in.Provider = strings.ToLower(strings.TrimSpace(in.Provider))
	in.Subject = strings.TrimSpace(in.Subject)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.AvatarURL = strings.TrimSpace(in.AvatarURL)

	if in.Provider == "" || in.Subject == "" {
		return nil, errors.New("provider and subject are required")
	}

	if in.Email == "" || !in.EmailVerified {
		return nil, ErrIdentityEmailNotVerified
	}

	profileJSON := []byte(`{}`)
	if len(in.Profile) > 0 {
		b, err := json.Marshal(in.Profile)
		if err != nil {
			return nil, err
		}
		profileJSON = b
	}

	var out control.User

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var acct control.Account

		err := tx.
			Preload("User").
			Where("provider = ? AND subject = ?", in.Provider, in.Subject).
			First(&acct).Error

		switch {
		case err == nil:

			accountUpdates := map[string]any{
				"email":          in.Email,
				"email_verified": in.EmailVerified,
				"profile":        datatypes.JSON(profileJSON),
			}

			var conflicts int64
			if err := tx.Model(&control.User{}).
				Where("LOWER(email) = ? AND id <> ?", in.Email, acct.UserID).
				Count(&conflicts).Error; err != nil {
				return err
			}
			if conflicts > 0 {
				return ErrSocialLinkNotAllowed
			}

			// email_verified_at is stamped here because reaching this point
			// REQUIRES in.EmailVerified — the guard at the top of this function
			// refuses an identity the provider has not verified. COALESCE keeps
			// the first confirmation: the timestamp is evidence of how long the
			// address has been proven, so a later sign-in must not push it
			// forward. Without this, a social user who signed up before the
			// confirmation gate existed would be locked out by it.
			if err := tx.Model(&control.User{}).
				Where("id = ?", acct.UserID).
				Updates(map[string]any{
					"email":             in.Email,
					"email_verified_at": gorm.Expr("COALESCE(email_verified_at, now())"),
				}).Error; err != nil {
				return err
			}

			if err := tx.Model(&control.Account{}).
				Where("id = ?", acct.ID).
				Updates(accountUpdates).Error; err != nil {
				return err
			}
			return tx.First(&out, "id = ?", acct.UserID).Error

		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var user control.User
		foundExistingUser := false

		err = tx.
			Model(&control.User{}).
			Select("users.*").
			Where("LOWER(users.email) = ?", in.Email).
			First(&user).Error

		switch {
		case err == nil:

			if strings.TrimSpace(user.Password) != "" && !in.LinkByEmail {
				return ErrSocialLinkNotAllowed
			}
			foundExistingUser = true
		case errors.Is(err, gorm.ErrRecordNotFound):

		default:
			return err
		}

		if !foundExistingUser {
			user = control.User{Email: in.Email}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		}

		// Same reasoning as the branch above: the provider has asserted this
		// address is verified, so the user is confirmed on arrival and never
		// meets the login confirmation gate. COALESCE so that linking a second
		// provider to an account does not restamp the original proof.
		if err := tx.Model(&control.User{}).
			Where("id = ?", user.ID).
			Update("email_verified_at", gorm.Expr("COALESCE(email_verified_at, now())")).Error; err != nil {
			return err
		}

		emailPts := (*string)(nil)
		if in.Email != "" {
			emailPts = &in.Email
		}

		acct = control.Account{
			UserID:        user.ID,
			Provider:      in.Provider,
			Subject:       in.Subject,
			Email:         emailPts,
			EmailVerified: in.EmailVerified,
			Profile:       datatypes.JSON(profileJSON),
		}
		if err := tx.Create(&acct).Error; err != nil {
			return err
		}
		// Re-read rather than returning the in-memory struct: the stamp above
		// was written in SQL, so `user` no longer matches the row.
		return tx.First(&out, "id = ?", user.ID).Error
	})

	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *SocialAuthRepoGorm) EnsureAutoMembership(ctx context.Context, userID uuid.UUID, email string) error {
	// The hardcoded publicEmailDomains map that used to gate this function is
	// gone; the same corpus now lives in auth_access_rules as create_org denies
	// (see SeedPublicEmailDomainRules). A domain we will not let mint a tenant
	// is a domain we will not silently auto-join someone into.
	id, err := NormalizeEmailIdentity(email)
	if err != nil {
		// This is a grant path, so an address we cannot decide gets nothing.
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// Decided on tx, not a second connection, so the rule that was true when
		// we looked is still true when the membership row is written.
		if err := EvaluateAccessRules(ctx, tx, control.AccessRulePurposeCreateOrg, id); err != nil {
			if errors.Is(err, ErrAccessDenied) {
				return nil
			}
			return err
		}

		var verified int64
		if err := tx.Model(&control.Account{}).
			Where("user_id = ? AND email_verified = TRUE AND LOWER(email) = ?", userID, id.Email).
			Count(&verified).Error; err != nil {
			return err
		}
		if verified == 0 {
			return nil
		}

		var orgs []control.Organization
		if err := tx.
			Where("LOWER(domain) = ? AND domain_verified_at IS NOT NULL", id.Domain).
			Limit(2).
			Find(&orgs).Error; err != nil {
			return err
		}
		if len(orgs) != 1 {

			return nil
		}
		org := orgs[0]

		var m control.Membership
		err := tx.Where("organization_id = ? AND user_id = ?", org.ID, userID).First(&m).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return tx.Create(&control.Membership{
			UserID:         userID,
			OrganizationID: org.ID,
			Role:           control.MemberRoleMember,
		}).Error
	})

}
