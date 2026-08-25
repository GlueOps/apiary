package tenant

import (
	"time"

	"github.com/google/uuid"
)

// A Swarm is a named group of hypervisors within one hive.
//
// It exists because a cluster is not automatically one pool of metal. Thirty
// hypervisors in a cluster are routinely meant as `fleet-prod`, `fleet-nonprod`
// and spare capacity, and without a grouping the placement oracle will happily
// put a non-production workload on the metal reserved for production — a cluster
// is a failure domain and an administrative boundary, not a scheduling one.
//
// WITHIN one hive, deliberately. Membership of a swarm cannot span clusters,
// because the guarantees people expect of a group — live migration, HA failover
// between members — are exactly the guarantees corosync provides inside a
// cluster and nowhere else. A swarm spanning two hives would be a group whose
// members cannot take each other's work, which is the opposite of why the group
// was drawn.
type Swarm struct {
	// The uniqueIndex here looks redundant on a primary key and is not: paired
	// with HiveID it is the TARGET of the composite foreign key on Hypervisor,
	// and that is what makes "a hypervisor's swarm is in the hypervisor's hive"
	// a database rule rather than a hope.
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();uniqueIndex:ux_swarms_hive_id,priority:2" json:"id"`

	// HiveID is the cluster this swarm subdivides.
	//
	HiveID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_swarms_hive_id,priority:1;uniqueIndex:ux_swarms_hive_name,priority:1" json:"hive_id"`
	Hive   *Hive     `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`

	// Name is unique within the hive, not globally: two clusters may both
	// sensibly have a swarm called "prod".
	Name string `gorm:"size:255;not null;uniqueIndex:ux_swarms_hive_name,priority:2" json:"name"`

	Description string `gorm:"type:text;not null;default:''" json:"description,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at" format:"date-time"`
}

// SwarmMembership describes where one hypervisor sits.
//
// Derived for a view rather than stored: the fact lives in Hypervisor.SwarmID
// and deriving it keeps one answer to "is this spare".
const (
	// SwarmMembershipAssigned: the hypervisor belongs to a named swarm.
	SwarmMembershipAssigned = "assigned"
	// SwarmMembershipSpare: in a hive, in no swarm. Available to be added to one.
	//
	// NOT the same as "fleetless", which this codebase already uses for a host
	// belonging to no HIVE — a machine mid-migration between clusters. Spare
	// capacity is in a cluster and working; it simply has no owner yet. Two very
	// different situations, and using one word for both would put a host that is
	// halfway through a migration in the same list as idle metal.
	SwarmMembershipSpare = "spare"
)

// MembershipOf reports whether a hypervisor is assigned or spare.
func MembershipOf(h Hypervisor) string {
	if h.SwarmID != nil {
		return SwarmMembershipAssigned
	}
	return SwarmMembershipSpare
}
