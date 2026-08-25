package control

import (
	"time"

	"github.com/google/uuid"
)

type AuthChallenge struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	JTI    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Kind string `gorm:"type:varchar(32);not null"`

	Attempts    int        `gorm:"not null;default:0"`
	MaxAttempts int        `gorm:"not null;default:5"`
	ConsumedAt  *time.Time `gorm:"type:timestamptz"`
	ExpiresAt   time.Time  `gorm:"type:timestamptz;not null;index"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

type AuthFailure struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	Identifier string `gorm:"type:varchar(320);not null;uniqueIndex:idx_auth_failures_identifier_kind"`
	Kind       string `gorm:"type:varchar(32);not null;uniqueIndex:idx_auth_failures_identifier_kind"`
	Count      int    `gorm:"not null;default:0"`

	LockedUntil *time.Time `gorm:"type:timestamptz"`
	WindowStart time.Time  `gorm:"type:timestamptz;not null;default:now()"`

	LastFailedAt time.Time `gorm:"type:timestamptz;not null;default:now();index"`
}
