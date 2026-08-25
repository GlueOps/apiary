package control

import (
	"time"

	"github.com/google/uuid"
)

type ImpersonationGrant struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TargetUserID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"target_user_id"`
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null" json:"organization_id"`
	GrantedByID    uuid.UUID  `gorm:"type:uuid;not null" json:"granted_by_id"`
	Reason         string     `gorm:"type:text" json:"reason,omitempty"`
	ExpiresAt      time.Time  `gorm:"type:timestamptz;not null" json:"expires_at"`
	RevokedAt      *time.Time `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (ImpersonationGrant) TableName() string { return "impersonation_grants" }
