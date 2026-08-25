package control

import (
	"time"

	"github.com/google/uuid"
)

type EmailChangeRequest struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	NewEmail  string    `gorm:"type:varchar(255);not null"`
	TokenHash string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null;index"`
	UsedAt    *time.Time
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
}
