package control

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type UserPasskey struct {
	ID              uuid.UUID                    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID                    `gorm:"type:uuid;index;not null" json:"user_id"`
	CredentialID    string                       `gorm:"uniqueIndex;not null;type:text" json:"credential_id"`
	PublicKey       []byte                       `gorm:"not null;type:bytea" json:"-"`
	AttestationType string                       `gorm:"not null;type:varchar(64)" json:"attestation_type"`
	AAGUID          string                       `gorm:"type:varchar(36)" json:"aaguid"`
	SignCount       uint32                       `gorm:"not null;default:0" json:"sign_count"`
	BackupEligible  bool                         `gorm:"not null;default:false" json:"backup_eligible"`
	BackupState     bool                         `gorm:"not null;default:false" json:"backup_state"`
	Name            string                       `gorm:"not null;type:varchar(255)" json:"name"`
	Transports      datatypes.JSONType[[]string] `gorm:"type:jsonb" json:"transports"`
	LastUsedAt      *time.Time                   `json:"last_used_at,omitempty"`
	CreatedAt       time.Time                    `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
}
