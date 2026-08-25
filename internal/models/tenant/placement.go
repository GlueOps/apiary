package tenant

import (
	"time"

	"github.com/google/uuid"
)

// Placement states. A placement exists before the VM does — that is the point of
// the row: it is the reservation that stops two concurrent decisions choosing
// the same capacity.
const (
	// PlacementPending: capacity is committed, no VM exists yet. It counts
	// against the node from the moment it is written, because a reservation that
	// only counted once provisioning succeeded would let every concurrent
	// decision see the same free space.
	PlacementPending = "pending"
	// PlacementActive: the VM exists and carries VMID.
	PlacementActive = "active"
	// PlacementFailed: provisioning did not complete. Kept rather than deleted so
	// the capacity is released explicitly and the failure stays visible.
	PlacementFailed = "failed"
	// PlacementReleased: the VM is gone and the capacity is free.
	PlacementReleased = "released"
)

// PlacementHoldsCapacity reports whether a state still occupies the node.
//
// One definition, because the placement decision and any capacity report must
// agree about it. Two implementations of "does this count" drift, and the
// direction they drift in is overcommit.
func PlacementHoldsCapacity(state string) bool {
	return state == PlacementPending || state == PlacementActive
}

// A Placement is one pool member landed on one hypervisor.
type Placement struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	PoolID uuid.UUID `gorm:"type:uuid;not null;index" json:"pool_id"`
	Pool   *Pool     `gorm:"foreignKey:PoolID;constraint:OnDelete:CASCADE" json:"-"`

	// The node this occupies. Indexed with state, because the hot query is
	// "everything still holding capacity on this hypervisor" and it runs inside
	// the transaction that decides the next placement.
	HypervisorID uuid.UUID   `gorm:"type:uuid;not null;index:idx_placements_hypervisor_state,priority:1" json:"hypervisor_id"`
	Hypervisor   *Hypervisor `gorm:"foreignKey:HypervisorID;constraint:OnDelete:RESTRICT" json:"-"`

	State string `gorm:"size:16;not null;default:'pending';index:idx_placements_hypervisor_state,priority:2" json:"state"`

	// HiveID is denormalised from the pool so that VMID uniqueness can be
	// expressed on this row.
	//
	// A Proxmox VMID is unique within a CLUSTER, and the cluster was previously
	// only reachable through Pool — so the constraint that matters could not be
	// written where it needed to be, and two placements could claim one VMID with
	// nothing to stop them. Denormalising a foreign key purely to make a
	// constraint expressible is worth it here: the alternative is enforcing
	// cluster-wide uniqueness in application code, which is the thing databases
	// are for.
	HiveID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_placements_hive_vmid,priority:1,where:vmid IS NOT NULL" json:"hive_id"`

	// VMID is the provider's identifier, absent until the VM exists.
	//
	// The unique index above is PARTIAL. Postgres treats NULLs as distinct, so a
	// plain unique index over (hive_id, vmid) would constrain nothing while
	// placements are pending — every pending row would be unique against every
	// other by virtue of having no VMID. The same trap as the firewall bindings.
	VMID *int `gorm:"column:vmid;uniqueIndex:ux_placements_hive_vmid,priority:2,where:vmid IS NOT NULL" json:"vmid,omitempty"`

	// Why a placement failed, for the operator reading it later.
	Reason string `gorm:"type:text;not null;default:''" json:"reason,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at" format:"date-time"`
}

// HoldsCapacity reports whether this placement still occupies its node.
func (p Placement) HoldsCapacity() bool { return PlacementHoldsCapacity(p.State) }
