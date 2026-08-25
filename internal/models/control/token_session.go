package control

import (
	"time"

	"github.com/google/uuid"
)

type TokenSession struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SubjectID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_token_sessions_subject,composite:subject" json:"subject_id"`
	SubjectKind      string     `gorm:"type:varchar(10);not null;index:idx_token_sessions_subject,composite:subject" json:"subject_kind"`
	OrganizationID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	FamilyID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"family_id"`
	AccessTokenJTI   string     `gorm:"type:varchar(36);uniqueIndex;not null" json:"-"`
	RefreshTokenJTI  string     `gorm:"type:varchar(36);uniqueIndex;not null" json:"-"`
	AccessTokenHash  string     `gorm:"type:varchar(64);not null" json:"-"`
	RefreshTokenHash string     `gorm:"type:varchar(64);not null" json:"-"`
	ActorID          *uuid.UUID `gorm:"type:uuid" json:"actor_id,omitempty"`
	AMR              string     `gorm:"type:varchar(100)" json:"amr,omitempty"`
	IPAddress        string     `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        string     `gorm:"type:text" json:"user_agent,omitempty"`
	AccessExpiresAt  time.Time  `gorm:"type:timestamptz;not null" json:"access_expires_at"`
	RefreshExpiresAt time.Time  `gorm:"type:timestamptz;not null" json:"refresh_expires_at"`
	RevokedAt        *time.Time `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (TokenSession) TableName() string { return "token_sessions" }
