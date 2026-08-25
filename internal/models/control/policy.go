package control

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PolicyKind string

const (
	PolicyKindRequireMFA         PolicyKind = "require_mfa"
	PolicyKindRequirePasskey     PolicyKind = "require_passkey"
	PolicyKindPasswordMinLength  PolicyKind = "password_min_length"
	PolicyKindPasswordComplexity PolicyKind = "password_complexity"
	PolicyKindSessionMaxLifetime PolicyKind = "session_max_lifetime"
)

var validPolicyKinds = map[PolicyKind]bool{
	PolicyKindRequireMFA:         true,
	PolicyKindRequirePasskey:     true,
	PolicyKindPasswordMinLength:  true,
	PolicyKindPasswordComplexity: true,
	PolicyKindSessionMaxLifetime: true,
}

func (k PolicyKind) IsValid() bool {
	return validPolicyKinds[k]
}

func (k PolicyKind) IsCheckable() bool {
	return k == PolicyKindRequireMFA || k == PolicyKindRequirePasskey
}

type PolicyConfig struct {
	MinLength          int  `json:"min_length,omitempty"`
	RequireUppercase   bool `json:"require_uppercase,omitempty"`
	RequireLowercase   bool `json:"require_lowercase,omitempty"`
	RequireDigit       bool `json:"require_digit,omitempty"`
	RequireSpecial     bool `json:"require_special,omitempty"`
	MaxLifetimeMinutes int  `json:"max_lifetime_minutes,omitempty"`
}

type PlatformPolicy struct {
	ID        uuid.UUID                        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Kind      PolicyKind                       `gorm:"not null;type:varchar(50);uniqueIndex" json:"kind"`
	Enabled   bool                             `gorm:"not null;default:false" json:"enabled"`
	Config    datatypes.JSONType[PolicyConfig] `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	CreatedBy uuid.UUID                        `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt time.Time                        `gorm:"type:timestamptz;column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time                        `gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" json:"updated_at"`
}

type OrgPolicy struct {
	ID             uuid.UUID                        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID                        `gorm:"type:uuid;not null;uniqueIndex:idx_org_policies_org_kind" json:"organization_id"`
	Organization   Organization                     `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"-"`
	Kind           PolicyKind                       `gorm:"not null;type:varchar(50);uniqueIndex:idx_org_policies_org_kind" json:"kind"`
	Enabled        bool                             `gorm:"not null;default:false" json:"enabled"`
	Config         datatypes.JSONType[PolicyConfig] `gorm:"type:jsonb;not null;default:'{}'" json:"config"`
	CreatedBy      uuid.UUID                        `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt      time.Time                        `gorm:"type:timestamptz;column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time                        `gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" json:"updated_at"`
}
