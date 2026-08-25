package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// A Pool is a desired count of VMs of one size, in one hive.
//
// The size is a Slot — a t-shirt size, "medium" meaning some fixed vCPU/RAM/disk
// — so a pool is a slot multiplied by a count, and a Placement is one of those
// VMs assigned to a hypervisor.
//
// DESIRED state, like everything else here: it says how many there should be,
// not how many there are. What there are is the count of placements holding
// capacity, and the difference is what a reconciler acts on.
type Pool struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// The hive, not a separate datacenter. See Hypervisor for why those were one
	// thing modelled twice.
	HiveID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_pools_hive_name,priority:1" json:"hive_id"`
	Hive   *Hive     `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`

	// SwarmID is which group of hypervisors this pool places onto. REQUIRED.
	//
	// Not nullable, and the alternative is worse in both directions. A nil
	// meaning "anywhere in the hive" would let a non-production pool land on the
	// metal somebody reserved for production, which is the exact thing swarms
	// exist to prevent. A nil meaning "only spare capacity" would be a second,
	// invisible rule that a reader has to know.
	//
	// So a pool names its swarm, and spare capacity is simply not a placement
	// target until an operator assigns it to one. That is the workflow anyway:
	// metal is racked, joins the cluster, sits spare, and is added to a swarm
	// when it should start carrying that swarm's work.
	//
	// Same composite foreign key as Hypervisor.SwarmID, for the same reason: the
	// pool's swarm must be in the pool's hive, and a plain reference to swarms(id)
	// would not say so.
	SwarmID uuid.UUID `gorm:"type:uuid;not null;index" json:"swarm_id"`
	Swarm   *Swarm    `gorm:"foreignKey:HiveID,SwarmID;references:HiveID,ID;constraint:OnDelete:RESTRICT" json:"-"`

	// RESTRICT, not CASCADE: deleting a slot that pools still reference would
	// silently destroy the pools and every placement under them. A slot in use
	// must be retired deliberately.
	SlotID uuid.UUID `gorm:"type:uuid;not null;index" json:"slot_id"`
	Slot   *Slot     `gorm:"foreignKey:SlotID;constraint:OnDelete:RESTRICT" json:"-"`

	// Unique within the hive rather than globally: two fleets may both sensibly
	// have a pool called "workers".
	Name string `gorm:"size:255;not null;uniqueIndex:ux_pools_hive_name,priority:2" json:"name"`

	DesiredCount int `gorm:"not null;default:0" json:"desired_count"`

	Metadata datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at" format:"date-time"`
}
