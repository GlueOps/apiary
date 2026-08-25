package control

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AuthAuditEvent struct {
	ID             uuid.UUID                          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SubjectID      uuid.UUID                          `gorm:"type:uuid;not null;index" json:"subject_id"`
	SubjectKind    string                             `gorm:"type:varchar(10);not null" json:"subject_kind"`
	OrganizationID uuid.UUID                          `gorm:"type:uuid;not null" json:"organization_id"`
	SessionID      *uuid.UUID                         `gorm:"type:uuid" json:"session_id,omitempty"`
	EventType      string                             `gorm:"type:varchar(30);not null;index" json:"event_type"`
	IPAddress      string                             `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent      string                             `gorm:"type:text" json:"user_agent,omitempty"`
	Metadata       datatypes.JSONType[map[string]any] `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt      time.Time                          `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (AuthAuditEvent) TableName() string { return "auth_audit_events" }
