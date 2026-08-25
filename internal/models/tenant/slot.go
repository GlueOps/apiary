package tenant

import (
	"time"

	"github.com/google/uuid"
)

// A Slot is a named guest size — the unit a pool multiplies and the unit the
// placement oracle fits against a node.
//
// Sizes are named rather than written inline on each pool so that "what does a
// medium cost" has one answer. It also means resizing is a deliberate act with a
// visible blast radius: editing a slot changes what every pool using it consumes.
type Slot struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name string    `gorm:"size:255;uniqueIndex;not null" json:"name"`

	VCPU   int `gorm:"column:vcpu;not null" json:"vcpu"`
	RAMGB  int `gorm:"column:ram_gb;not null" json:"ram_gb"`
	DiskGB int `gorm:"column:disk_gb;not null" json:"disk_gb"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at" format:"date-time"`
}

// Capacity expresses a slot as the demand it places on a node.
func (s Slot) Capacity() Capacity {
	return Capacity{CPU: s.VCPU, RAMGB: s.RAMGB, DiskGB: s.DiskGB}
}
