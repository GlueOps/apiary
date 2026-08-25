package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DepartureSnapshotFromDB reads every input the departure gate decides on, in
// ONE transaction.
//
// A single transaction rather than four queries because the gate's verdict is
// only meaningful if its inputs describe a state that actually existed at one
// moment. Counting placements and then counting nodes lets a host arrive or a
// guest be created in between, and the resulting verdict corresponds to no real
// state of the fleet — which is the worst kind of wrong for a check whose whole
// job is to be trusted before a cluster is taken apart.
func DepartureSnapshotFromDB(ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID) (DepartureSnapshot, error) {
	var snap DepartureSnapshot

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var hyp tenant.Hypervisor
		if err := tx.First(&hyp, "id = ?", hypervisorID).Error; err != nil {
			return fmt.Errorf("hypervisor %s: %w", hypervisorID, err)
		}
		snap.Hypervisor = hyp
		snap.BeeID = hyp.BeeID

		// Membership first. HiveID is nullable now that this row is the durable
		// machine record, so "in no cluster" is a normal state — and every read
		// below is scoped by hive_id, which against NULL matches nothing. Without
		// this the snapshot failed with "hive 00000000-...: record not found",
		// which a worker retries, instead of the gate's own not_a_member answer,
		// which a worker cancels.
		//
		// Returned as a snapshot rather than as a refusal from here: the gate lives
		// in CheckDeparture and refuses on exactly this — a nil HiveID or a zero
		// node count — so raising it here would be a second copy of one rule, and
		// the copies would have to agree forever.
		//
		// The snapshot is NOT empty. Hypervisor and BeeID are already set above and
		// are deliberately kept, because CheckDeparture needs the machine to answer
		// at all: it is what it reports not_a_member ABOUT. What is left unset is
		// everything read per-hive below — the hive, its node count, the placements
		// — every one of which is scoped by hive_id and would match nothing against
		// a NULL anyway.
		if hyp.HiveID == nil {
			return nil
		}

		if err := tx.First(&snap.Hive, "id = ?", *hyp.HiveID).Error; err != nil {
			return fmt.Errorf("hive %s: %w", *hyp.HiveID, err)
		}

		// Placements still holding capacity, using the shared definition rather
		// than an inline state list. Two definitions of "does this count" drift,
		// and the direction they drift in is a node that departs while a pending
		// placement still names it.
		var held int64
		if err := tx.Model(&tenant.Placement{}).
			Where("hypervisor_id = ? AND state IN ?", hyp.ID,
				[]string{tenant.PlacementPending, tenant.PlacementActive}).
			Count(&held).Error; err != nil {
			return fmt.Errorf("count held placements: %w", err)
		}
		snap.HeldPlacements = int(held)

		var nodes int64
		if err := tx.Model(&tenant.Hypervisor{}).
			Where("hive_id = ?", *hyp.HiveID).Count(&nodes).Error; err != nil {
			return fmt.Errorf("count hive nodes: %w", err)
		}
		snap.HiveNodeCount = int(nodes)

		// Peers: every other node in the hive. Their names are what has to be
		// cleared off the departing host, because it carries the cluster's replica
		// of their guest configs and a later join reads those as its own guests.
		var peers []tenant.Hypervisor
		if err := tx.Where("hive_id = ? AND id <> ?", *hyp.HiveID, hyp.ID).
			Order("name").Find(&peers).Error; err != nil {
			return fmt.Errorf("list peers: %w", err)
		}
		for _, p := range peers {
			// An unnamed peer contributes no directory to prune, and appending ""
			// would have the departing host clear a path that means "everything".
			if p.Name != nil && *p.Name != "" {
				snap.PeerNames = append(snap.PeerNames, *p.Name)
			}
		}

		// Addresses. Departure is the one fleet operation that needs to reach TWO
		// hosts, and neither is optional: the cluster-side removal cannot run on
		// the node being removed, and the standalone conversion cannot run
		// anywhere else.
		if hyp.Address == "" {
			return fmt.Errorf("the departing node has no reachable address: no address is " +
				"recorded for this machine")
		}
		snap.DepartingAddress = hyp.Address

		// Prefer a peer whose agent has been seen recently. A peer that is itself
		// offline may accept the connection and then fail mid-delnode, which is the
		// half-done removal this sequence works hardest to avoid.
		//
		// The address comes off the peer's MACHINE record and the liveness off its
		// agent. Two rows on purpose: where to connect is a fact about the box,
		// whether anyone is home is a fact about the software on it.
		for _, p := range peers {
			if p.Address == "" {
				continue
			}
			if snap.RemainingAddress == "" {
				snap.RemainingAddress = p.Address
			}
			if p.BeeID == nil {
				continue
			}
			var bee tenant.Bee
			if err := tx.First(&bee, "id = ?", *p.BeeID).Error; err != nil {
				continue
			}
			if bee.LastSeenAt != nil && time.Since(*bee.LastSeenAt) < 5*time.Minute {
				snap.RemainingAddress = p.Address
				break
			}
		}
		if snap.RemainingAddress == "" {
			return fmt.Errorf("no remaining node in hive %q has a resolvable address, so the "+
				"cluster-side removal cannot be performed", snap.Hive.Name)
		}
		return nil
	})
	if err != nil {
		return DepartureSnapshot{}, err
	}
	return snap, nil
}
