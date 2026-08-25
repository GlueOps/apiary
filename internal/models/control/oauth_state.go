package control

import (
	"time"

	"github.com/google/uuid"
)

type OAuthState struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StateHash  string     `gorm:"type:varchar(64);not null;uniqueIndex"`
	Provider   string     `gorm:"type:varchar(32);not null"`
	Mode       string     `gorm:"type:varchar(32);not null;default:''"`
	Origin     string     `gorm:"type:text;not null;default:''"`
	CookieHash string     `gorm:"type:varchar(64);not null"`
	ExpiresAt  time.Time  `gorm:"type:timestamptz;not null;index"`
	ConsumedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (OAuthState) TableName() string { return "oauth_states" }
