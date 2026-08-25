package control

import (
	"time"

	"github.com/google/uuid"
)

type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

type Membership struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID    `gorm:"index;not null" json:"user_id"`
	User           User         `gorm:"foreignKey:UserID" json:"-"`
	OrganizationID uuid.UUID    `gorm:"index;not null" json:"org_id"`
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"-"`
	Role           MemberRole   `gorm:"not null;default:'member'" json:"role"`
	SupportEnabled bool         `gorm:"default:false" json:"support_enabled"`
	CreatedAt      time.Time    `json:"created_at,omitempty" gorm:"type:timestamptz;column:created_at;not null;default:now()"`
	UpdatedAt      time.Time    `json:"updated_at,omitempty" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()"`
}
