package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ArrivalSnapshotFromDB reads every input the arrival gate decides on, in ONE
// transaction — same reasoning as the departure snapshot: a verdict assembled
// from separate reads corresponds to no state that actually existed.
//
// Keyed on the MACHINE, like departure. It used to be keyed on the bee, because
// a Hypervisor row was whatever a capacity sync had observed in a cluster and a
// host outside one therefore had no such row. That is no longer true: the
// Hypervisor IS the machine record and exists from the moment somebody records
// the machine, cluster or no cluster. Keying on it removes the asymmetry with
// departure and, more importantly, gives the join somewhere to write the node
// name when it succeeds — which is what lets the next capacity sync recognise
// this machine instead of creating a second row for it.
func ArrivalSnapshotFromDB(ctx context.Context, db *gorm.DB, hypervisorID, targetHiveID uuid.UUID) (ArrivalSnapshot, error) {
	var snap ArrivalSnapshot

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var joining tenant.Hypervisor
		if err := tx.First(&joining, "id = ?", hypervisorID).Error; err != nil {
			return fmt.Errorf("machine %s: %w", hypervisorID, err)
		}
		snap.CurrentHiveID = joining.HiveID
		snap.JoiningAddress = joining.Address
		if snap.JoiningAddress == "" {
			return fmt.Errorf("no address is recorded for this machine, so it cannot be reached")
		}

		// The joining host's own guest count, straight off the machine record.
		// This used to be looked up through hypervisors.bee_id, which no
		// production path ever wrote — so it always missed, the count was always
		// zero, and the arrival gate's guests_present refusal was dead code
		// outside the lab. It is now read from the row the gate is about.
		//
		// It is NOT the safe direction in general, which is why ReadJoinBlockers
		// re-asks the host directly before the join. This value only decides
		// whether to bother connecting.
		snap.ObservedGuests = joining.ObservedGuests
		if joining.Name != nil {
			snap.JoiningNodeName = *joining.Name
		}
		if snap.JoiningNodeName == "" {
			// Falls back to the address's first label, which is what a Proxmox node
			// name is. Only used to tell this node's own directory from a former
			// peer's, so a wrong guess — or the empty string an IP literal yields —
			// over-reports foreign directories rather than under-reporting them,
			// which is the safe direction.
			snap.JoiningNodeName = tenant.NodeNameFromAddress(joining.Address)
		}

		if err := tx.First(&snap.TargetHive, "id = ?", targetHiveID).Error; err != nil {
			return fmt.Errorf("target hive %s: %w", targetHiveID, err)
		}

		var members []tenant.Hypervisor
		if err := tx.Where("hive_id = ?", targetHiveID).Order("name").Find(&members).Error; err != nil {
			return fmt.Errorf("list target hive members: %w", err)
		}
		snap.TargetNodeCount = len(members)
		for _, m := range members {
			// A member with no node name contributes nothing to seed known_hosts
			// with, and appending "" would seed an entry that matches nothing.
			if m.Name != nil && *m.Name != "" {
				snap.TargetNodeNames = append(snap.TargetNodeNames, *m.Name)
			}
		}

		// Point the join at a member whose agent has been seen recently: a member
		// that is itself offline will accept the connection and then fail
		// mid-join, which is the half-joined state worth avoiding.
		//
		// The address comes off the MACHINE record and the liveness off its agent.
		// Two rows on purpose — where to connect is a fact about the box, whether
		// anyone is home is a fact about the software on it.
		for _, m := range members {
			if m.Address == "" {
				continue
			}
			if snap.TargetAddress == "" {
				snap.TargetAddress = m.Address
			}
			if m.BeeID == nil {
				continue
			}
			var mb tenant.Bee
			if err := tx.First(&mb, "id = ?", *m.BeeID).Error; err != nil {
				continue
			}
			if mb.LastSeenAt != nil && time.Since(*mb.LastSeenAt) < 5*time.Minute {
				snap.TargetAddress = m.Address
				break
			}
		}
		if snap.TargetNodeCount > 0 && snap.TargetAddress == "" {
			return fmt.Errorf("no member of hive %q has a resolvable address, so there is "+
				"nothing to point the join at", snap.TargetHive.Name)
		}
		return nil
	})
	if err != nil {
		return ArrivalSnapshot{}, err
	}
	return snap, nil
}
