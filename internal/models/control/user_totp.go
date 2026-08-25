package control

import (
	"time"

	"github.com/google/uuid"
)

type UserTOTP struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Secret       string    `gorm:"not null;type:text" json:"-"`
	Verified     bool      `gorm:"default:false" json:"verified"`
	LastUsedStep int64     `gorm:"not null;default:0" json:"-"`
	CreatedAt    time.Time `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;not null;default:now()"`
}
