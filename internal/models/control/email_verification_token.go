package control

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationToken proves that whoever registered an account can read the
// mailbox they registered with.
//
// It is deliberately a separate table from PasswordResetToken rather than a
// "kind" column on a shared one. The two have different lifetimes, different
// blast radius when leaked, and — most importantly — different invalidation
// rules: resetting a password revokes every session, while confirming an email
// revokes nothing. A shared table invites one path's rule to be applied to the
// other by whoever touches it next.
//
// Email is stored alongside UserID because the two can diverge. A token issued
// for alice@old.example and confirmed after the address has been changed must
// not be read as proof of control over the new one; Confirm compares this
// column against the user's current address and refuses on a mismatch.
type EmailVerificationToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Email     string     `gorm:"type:varchar(255);not null"`
	TokenHash string     `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null;index"`
	UsedAt    *time.Time `gorm:"type:timestamptz"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}
