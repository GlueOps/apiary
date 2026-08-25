package control

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Account struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id" format:"uuid"`
	UserID        uuid.UUID      `gorm:"index;not null" json:"user_id" format:"uuid"`
	User          User           `gorm:"foreignKey:UserID" json:"-"`
	Provider      string         `gorm:"not null" json:"provider"`
	Subject       string         `gorm:"not null" json:"subject"`
	Email         *string        `json:"email,omitempty"`
	EmailVerified bool           `gorm:"not null;default:false" json:"email_verified"`
	Profile       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"profile"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;column:created_at;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" json:"updated_at" format:"date-time"`
}
