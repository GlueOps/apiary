package control

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name                string     `gorm:"not null" json:"name"`
	Domain              *string    `gorm:"index" json:"domain"`
	DomainVerifiedAt    *time.Time `gorm:"column:domain_verified_at" json:"domain_verified_at,omitempty"`
	Status              string     `gorm:"not null;default:'pending'" json:"status"`
	ConnectionString    string     `gorm:"not null;default:''" json:"-"`
	EncryptedTenantKey  string     `gorm:"not null;default:''" json:"-"`
	TenantKeyIV         string     `gorm:"not null;default:''" json:"-"`
	TenantKeyTag        string     `gorm:"not null;default:''" json:"-"`
	ConnectionStringIV  string     `gorm:"not null;default:''" json:"-"`
	ConnectionStringTag string     `gorm:"not null;default:''" json:"-"`
	RegistrableDomain   *string    `gorm:"type:varchar(253);column:registrable_domain;index:idx_organizations_registrable_domain_proven,priority:1" json:"-"`
	ProvenAt            *time.Time `gorm:"type:timestamptz;column:proven_at;index:idx_organizations_registrable_domain_proven,priority:2" json:"-"`
	ProvingDaysSeen     int        `gorm:"column:proving_days_seen;not null;default:0" json:"-"`
	ProvingLastDay      *time.Time `gorm:"type:date;column:proving_last_day" json:"-"`

	// DeletedAt soft-deletes the organization, and soft is the only safe kind
	// here.
	//
	// This row holds EncryptedTenantKey and the sealed ConnectionString, which
	// are the ONLY things that can decrypt the tenant database. Deleting the row
	// destroys the customer's data whether or not the database is dropped: what
	// survives is ciphertext under a key that no longer exists anywhere. A hard
	// delete is therefore not "remove the record", it is "destroy everything,
	// irreversibly, from one API call".
	//
	// So deletion marks the row and stops it resolving. Purging the key material
	// and the database is a separate, deliberate operator step, which is what
	// gives a retention window and a way back from a mistaken deletion.
	DeletedAt *time.Time `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at;not null;default:now()" json:"updated_at"`
}
