package control

import (
	"time"

	"github.com/google/uuid"
)

// BeeEnrolmentToken admits a host into an organization once, within a window.
//
// Multi-use within a short expiry rather than one token per host: a rack is
// enrolled in a batch, and minting fifty tokens to bring up fifty hypervisors
// turns a security control into a reason to script around it. The expiry is what
// bounds the damage — the same trade kubeadm makes.
//
// Only the hash is stored. A leak of this table must not yield a token that can
// enrol a machine into the fleet.
type BeeEnrolmentToken struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// The organization the host is admitted INTO, and the only thing this token
	// resolves. It carries no hive: a host enrols into an organization and joins
	// a fleet afterwards, as a task.
	//
	// That is not a simplification for its own sake — it is the same path a
	// MIGRATING host takes. Moving hv5 from fleet1 to fleet2 already means
	// leaving, existing in no fleet, then joining, so "join a fleet" has to work
	// from the fleetless state regardless. Enrolling straight into that state
	// means a new host and a migrating host follow identical code, instead of a
	// second path that is only exercised on day one.
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organization_id"`
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"-"`

	TokenHash string `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`

	// Uses is compared against MaxUses inside the enrolment transaction. Zero
	// MaxUses means unlimited within the expiry.
	Uses    int `gorm:"not null;default:0" json:"uses"`
	MaxUses int `gorm:"not null;default:0" json:"max_uses"`

	ExpiresAt   time.Time  `gorm:"type:timestamptz;not null;index" json:"expires_at"`
	RevokedAt   *time.Time `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedByID uuid.UUID  `gorm:"type:uuid;not null" json:"created_by_id"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

// Open reports whether this token can still enrol a host.
func (t *BeeEnrolmentToken) Open(now time.Time) bool {
	if t == nil || t.RevokedAt != nil || !t.ExpiresAt.After(now) {
		return false
	}
	return t.MaxUses == 0 || t.Uses < t.MaxUses
}
