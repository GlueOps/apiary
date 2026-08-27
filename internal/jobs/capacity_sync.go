package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CapacitySyncDeps is what the sync needs, declared consumer-side so the upsert
// can be tested against a database without a Proxmox cluster.
type CapacitySyncDeps struct {
	// Inventory returns the hive's flat resource list, already mapped out of the
	// provider's wire format. Only this function knows what a provider is.
	Inventory func(ctx context.Context, orgID, hiveID uuid.UUID) ([]tenant.ResourceRow, error)

	// TenantDB opens the organization's database.
	TenantDB func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error)

	// NodeAddresses maps each cluster member's name to the address the CLUSTER
	// knows it by, from /cluster/status. Optional: nil, or an error, degrades to
	// the previous behaviour of recording a discovered node with no address.
	//
	// It exists because a node this system did not onboard is otherwise recorded
	// with a name and nothing to reach it by, and onboarding looks a host up by
	// ADDRESS — so the same machine ends up with two rows, one from each side.
	NodeAddresses func(ctx context.Context, orgID, hiveID uuid.UUID) (map[string]string, error)

	// ClusterIdentity reports whether the hive's API URL is a CLUSTER at all.
	//
	// A standalone Proxmox host answers /cluster/resources with itself, so
	// without this the sync records it as the hive's only member — and that one
	// row closes every route out: bootstrap refuses because the host is in a
	// hive and the hive is not empty, while arrival is permitted because the
	// target has a node, and then `pvecm add` is pointed at a machine that is
	// not a cluster. Both directions dead, and nothing saying why.
	//
	// Optional: nil skips the check, which is the pre-existing behaviour.
	ClusterIdentity func(ctx context.Context, orgID, hiveID uuid.UUID) (name string, clustered bool, err error)
}

// ErrHiveIsNotACluster reports that a hive's API URL answers as a standalone
// host rather than a cluster.
var ErrHiveIsNotACluster = errors.New("capacity sync: this hive's endpoint is a standalone host")

// SyncHiveCapacity refreshes the Hypervisor rows for one hive.
//
// What it writes, and what it deliberately does not:
//
//   - OBSERVED columns are overwritten wholesale, including ObservedGuests. They
//     describe the world, and the sync is the only writer.
//   - RESERVED capacity, the per-host overcommit override, Schedulable, SWARM
//     MEMBERSHIP and the whole ONBOARDING LIFECYCLE are PRESERVED on existing
//     rows. The first four are operator intent, and a sync that reset them would
//     un-cordon a host somebody drained — silently re-admitting it to the
//     placement query mid-departure — or drop a hypervisor out of the swarm it
//     was assigned to, which reads to the oracle as capacity vanishing for no
//     reason. provisioning_state is worse still: resetting a working host to
//     'pending' would put the onboarding ticker back onto metal that is already
//     running guests. Every update below names the columns it writes, so all of
//     this survives by construction rather than by remembering to exclude it.
//   - Rows for nodes the cluster stopped reporting are CLEARED, NOT DELETED. See
//     below; this is the change that made Hypervisor the durable machine record.
//
// The fold runs INSIDE the transaction rather than before it, which is a change
// from when this only counted guests. It now needs the hive's placement ledger
// to tell our guests from everybody else's, and reading that ledger apart from
// the write would leave a window in which a placement acquires its VMID and is
// then recorded as foreign — subtracted once as foreign and again as intended.
//
// ObservedGuests is not optional. The departure and arrival gates read it, and a
// sync that wrote LastSyncedAt while leaving it at its zero default would turn
// every gate from "refuses because the observation is stale" into "passes
// because the node looks empty" — a fail-open on the check that decides whether
// to dismantle a host's cluster membership. That is why the fold produces it and
// the upsert below writes it in the same statement as LastSyncedAt: the two
// cannot come apart.
func SyncHiveCapacity(ctx context.Context, d CapacitySyncDeps, orgID, hiveID uuid.UUID) (int, error) {
	// ASKED BEFORE THE INVENTORY, because a standalone host answers the inventory
	// perfectly well and the answer is the problem.
	if d.ClusterIdentity != nil {
		_, clustered, cerr := d.ClusterIdentity(ctx, orgID, hiveID)
		switch {
		case cerr != nil:
			// Not fatal on its own: the inventory read below will fail too if the
			// endpoint is genuinely unreachable, and with a better message.
			slog.WarnContext(ctx, "could not establish whether this hive's endpoint is a cluster",
				"organization_id", orgID, "hive_id", hiveID, "error", cerr)
		case !clustered:
			// WHAT A STANDALONE ANSWER ACTUALLY PROVES is only that the machine at
			// the API URL is in no cluster. It says nothing about the other
			// machines recorded in this hive, and conflating the two is a way to
			// destroy a live fleet's records.
			//
			// A hive's API URL names ONE node. When that node departs, the address
			// still points at it, and it now answers as standalone — while the
			// cluster it left is running, quorate, and full of members. Releasing
			// on that answer would mark every one of them fleetless.
			//
			// So the release is confined to the case it was written for: a hive
			// whose only recorded member is the standalone host itself, which is
			// membership that was never true. With more than one member the
			// endpoint is simply pointed at the wrong node, and the fix is to
			// repoint it — said, not guessed at.
			members, merr := countHiveMembers(ctx, d, orgID, hiveID)
			if merr != nil {
				return 0, fmt.Errorf("capacity sync: count members before deciding whether this "+
					"hive is a cluster: %w", merr)
			}
			if members > 1 {
				return 0, fmt.Errorf("%w: the API URL points at a machine that is in no cluster, "+
					"but %d hypervisors are recorded as members of this one — so the address is "+
					"pointed at a node that has left rather than at a cluster that is gone. "+
					"Nothing was changed. Repoint this cluster's API URL at a node that is still "+
					"in it", ErrHiveIsNotACluster, members)
			}
			if n, rerr := releaseStandaloneMembers(ctx, d, orgID, hiveID); rerr != nil {
				slog.ErrorContext(ctx, "could not release members of a hive that is not a cluster",
					"organization_id", orgID, "hive_id", hiveID, "error", rerr)
			} else if n > 0 {
				slog.WarnContext(ctx, "released hypervisors recorded as members of a hive whose "+
					"endpoint is a standalone host; they are fleetless again and can now be used "+
					"to create the cluster",
					"organization_id", orgID, "hive_id", hiveID, "released", n)
			}
			return 0, fmt.Errorf("%w: the API URL answers as a single machine with no corosync "+
				"configuration, so it has no nodes to observe. Create the cluster on that host "+
				"first — Hypervisors, then Create the cluster — or point this hive at a member "+
				"of a cluster that already exists", ErrHiveIsNotACluster)
		}
	}

	rows, err := d.Inventory(ctx, orgID, hiveID)
	if err != nil {
		return 0, fmt.Errorf("capacity sync: read inventory: %w", err)
	}

	// Best effort, and deliberately not fatal. An address makes a discovered
	// record USABLE; capacity is what the sync exists to refresh, and failing the
	// whole run because one optional field could not be filled would trade a
	// cosmetic gap for a real one.
	var addrs map[string]string
	if d.NodeAddresses != nil {
		a, aerr := d.NodeAddresses(ctx, orgID, hiveID)
		if aerr != nil {
			slog.WarnContext(ctx, "could not read node addresses for this hive, so any node "+
				"discovered by this sync is recorded without one; onboarding it by address will "+
				"create a second record for the same machine until a later sync fills it in",
				"organization_id", orgID, "hive_id", hiveID, "error", aerr)
		} else {
			addrs = a
		}
	}

	db, err := d.TenantDB(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("capacity sync: open tenant database: %w", err)
	}

	now := time.Now().UTC()
	var written int

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var members []tenant.Hypervisor
		if err := tx.Where("hive_id = ?", hiveID).Find(&members).Error; err != nil {
			return fmt.Errorf("load current hypervisors: %w", err)
		}
		byName := make(map[string]tenant.Hypervisor, len(members))
		for _, h := range members {
			// A member with no node name cannot be matched by one. It should not
			// exist — the CHECK and the partial unique index both assume a
			// clustered row is named — but indexing a map on "" would quietly make
			// every such row the same row.
			if h.Name != nil && *h.Name != "" {
				byName[*h.Name] = h
			}
		}

		// Machines belonging to no hive: onboarded but never joined, or departed
		// and cleared. These are the candidates for ADOPTION — see the merge
		// comment below.
		var unclustered []tenant.Hypervisor
		if err := tx.Where("hive_id IS NULL").Find(&unclustered).Error; err != nil {
			return fmt.Errorf("load unclustered machines: %w", err)
		}

		// THE LEDGER, READ IN THE SAME TRANSACTION AS THE WRITE, and the fold
		// happens here rather than before the database is opened because of it.
		//
		// These are the VMIDs of guests this system placed. Everything else the
		// cluster reports is somebody's hand-created VM, and the whole point of
		// the fold's Foreign* totals is that a placement decision subtracts it.
		// Reading the ledger and writing the observation apart from one another
		// would leave a window in which a placement lands, gets a VMID, and is
		// then recorded as foreign — counted twice, once as foreign and once as
		// intended.
		managed, err := managedVMIDs(tx, hiveID)
		if err != nil {
			return err
		}
		observations := tenant.FoldClusterResources(rows, managed)

		for name, obs := range observations {
			cur, known := byName[name]

			// THE MERGE. A machine onboarded through this system has a machine id
			// and an address but no hive and no node name, because a node name does
			// not exist until it joins a cluster. When it appears in one, the sync
			// knows only (hive, name) — /cluster/resources reports no machine id —
			// so matching on (hive_id, name) alone finds nothing and creates a
			// SECOND record for a machine already in the table, carrying none of its
			// onboarding history. Adopting the unclustered record instead is what
			// makes onboarded-then-discovered end as one row.
			//
			// Adoption is by named-column UPDATE and touches only the two columns
			// that ARE cluster membership. In particular machine_id, address,
			// bee_id and the whole provisioning lifecycle stay exactly as the
			// onboarding left them, which is the entire point of adopting rather
			// than creating.
			if !known {
				if m := tenant.MatchMachineByNodeName(unclustered, name, addrs[name]); m != nil {
					// CORDONED on adoption, for the reason MarkJoined cordons on a
					// join: nothing has observed this machine's capacity in this
					// cluster yet, and Proxmox reports zeros for a node the moment
					// it appears. Landing a guest on it in that window is not the
					// control plane's call to make.
					//
					// Adoption specifically means the membership was NOT recorded
					// through this system — a join that failed after `pvecm` had
					// already succeeded, or one performed out of band. Observed in
					// the lab: a bootstrap created the cluster, failed its own
					// post-create check, never reached MarkJoined, and the node
					// arrived in the fleet schedulable because this path set
					// membership and nothing cordoned it.
					//
					// A node in a pre-existing cluster that this system never
					// onboarded is unaffected: it has no unclustered record to
					// adopt, so it takes the create branch below and keeps the
					// schedulable default, which is right for a host nobody has
					// expressed intent about.
					if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", m.ID).
						Updates(map[string]any{
							"hive_id":     hiveID,
							"name":        name,
							"schedulable": false,
						}).Error; err != nil {
						return fmt.Errorf("adopt machine %q into this hive: %w", name, err)
					}
					cur, known = *m, true
					// Withdraw it from the candidate pool: two node names must not
					// both adopt one machine record within a single sync.
					unclustered = withoutMachine(unclustered, m.ID)
				}
			}

			// A guest row naming a node with no node row still produces an
			// observation, and folding it in is the point — but it must not create
			// a Hypervisor row for a node the cluster never described, because that
			// row would then carry zero capacity and read as a full host.
			if !obs.Online && !known {
				continue
			}

			fields := map[string]any{
				"observed_cpu_total":     obs.CPUTotal,
				"observed_ram_gb_total":  obs.RAMGBTotal,
				"observed_disk_gb_total": obs.DiskGBTotal,
				"observed_cpu_used":      obs.CPUUsed,
				"observed_ram_gb_used":   obs.RAMGBUsed,
				"observed_disk_gb_used":  obs.DiskGBUsed,
				// Written in the SAME statement as last_synced_at. A freshness
				// stamp without a guest count is what makes the gates fail open,
				// and the foreign columns are here for exactly the same reason:
				// left at zero behind a fresh timestamp they say "no guests but
				// ours on this node", which is what lets the oracle keep filling a
				// host somebody hand-loaded.
				"observed_guests":               obs.Guests,
				"observed_foreign_cpu_used":     obs.ForeignCPUUsed,
				"observed_foreign_ram_gb_used":  obs.ForeignRAMGBUsed,
				"observed_foreign_disk_gb_used": obs.ForeignDiskGBUsed,
				"last_synced_at":                now,
			}

			if known {
				// Backfill an address onto a record that has none — a node
				// discovered before this system could read cluster addresses, or
				// before the hive had a credential to read them with.
				//
				// ONLY when empty. An operator or an onboarding run may have
				// recorded the address this control plane actually reaches the
				// host on, and that is not always the address corosync knows it
				// by; overwriting it here would silently repoint every SSH the
				// fleet makes to that host.
				if strings.TrimSpace(cur.Address) == "" {
					if a := strings.TrimSpace(addrs[name]); a != "" {
						fields["address"] = a
					}
				}
				if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", cur.ID).
					Updates(fields).Error; err != nil {
					return fmt.Errorf("update hypervisor %q: %w", name, err)
				}
			} else {
				// A node in this cluster that nobody onboarded through this system.
				// Schedulable defaults true and reserved capacity zero, which is
				// correct for a host nobody has expressed intent about yet.
				//
				// machine_id is left NULL: we have never read this machine's own
				// identity, and there is no honest value to invent. That is what
				// tenant.MatchMachineByAddress exists to resolve if an agent is
				// later enrolled on it.
				nodeName := name
				h := tenant.Hypervisor{
					HiveID: &hiveID,
					Name:   &nodeName,
					// The address the CLUSTER knows this node by, when the status
					// call could supply it. Recording it here is what stops a
					// later declaration of the same host — which looks rows up by
					// address — from creating a second record for one machine.
					Address:             addrs[name],
					ObservedCPUTotal:    obs.CPUTotal,
					ObservedRAMGBTotal:  obs.RAMGBTotal,
					ObservedDiskGBTotal: obs.DiskGBTotal,
					ObservedCPUUsed:     obs.CPUUsed,
					ObservedRAMGBUsed:   obs.RAMGBUsed,
					ObservedDiskGBUsed:  obs.DiskGBUsed,
					ObservedGuests:      obs.Guests,
					// A node discovered rather than onboarded is the case most
					// likely to be carrying guests nobody here placed, so omitting
					// these on the create path would leave the newly found host
					// reading as empty until some later sync happened to update it.
					ObservedForeignCPUUsed:    obs.ForeignCPUUsed,
					ObservedForeignRAMGBUsed:  obs.ForeignRAMGBUsed,
					ObservedForeignDiskGBUsed: obs.ForeignDiskGBUsed,
					LastSyncedAt:              &now,
				}
				if err := tx.Create(&h).Error; err != nil {
					return fmt.Errorf("create hypervisor %q: %w", name, err)
				}
				// NEW CAPACITY IS NOT SCHEDULABLE CAPACITY.
				//
				// This function's job is to observe what a cluster has. Whether
				// that capacity may be USED is a different question with a
				// different owner, and discovering a host answers only the first.
				// Writing true here answered both, on the strength of having seen
				// the node exist.
				//
				// Same rule as arrival, and the same column: an operator enables a
				// host, having seen what was observed on it. Until this changed,
				// the identical machine ended up in opposite states depending on
				// whether the control plane met it by MOVING it or by FINDING it —
				// and finding is the weaker evidence of the two, since nothing has
				// validated that host at all.
				//
				// A SECOND STATEMENT, because the first cannot express it. GORM
				// omits a zero value from an INSERT for any field carrying a
				// `default:` tag, and Schedulable carries `default:true` — so
				// Schedulable:false in the struct above stores TRUE. Verified
				// against Postgres, not inferred.
				//
				// Only on create. The update path deliberately never writes this
				// column, so a host an operator has already enabled stays enabled
				// through every subsequent sync.
				if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", h.ID).
					Update("schedulable", false).Error; err != nil {
					return fmt.Errorf("cordon newly discovered hypervisor %q: %w", name, err)
				}
			}
			written++
		}

		// Nodes the cluster stopped reporting.
		//
		// CLEARED, NOT DELETED, and that reversal is the reason this whole model
		// exists. Deleting was what made membership self-healing, but it also
		// destroyed the machine record — its identity, its address, its onboarding
		// history — every time a host left a cluster, which left onboarding with
		// nowhere to live and made a departed machine simply vanish. Clearing the
		// cluster-derived columns achieves the same visible result: the row is no
		// longer in any hive, no longer named, carries no capacity, and so cannot
		// be a placement candidate — it just stays visible as ours-but-unclustered.
		//
		// ZEROING THE OBSERVATIONS IS NOT OPTIONAL. Leaving stale totals on a
		// machine in no cluster is what would make DrainStateAt report real
		// capacity for a host nothing can schedule onto. last_synced_at goes to
		// NULL for the same reason: there is no current observation of a node the
		// cluster is not reporting, and a fresh timestamp over stale numbers is the
		// exact fail-open the gates are written to avoid.
		//
		// swarm_id goes with hive_id. A swarm is a subdivision of a cluster the
		// machine has left, so keeping it would be meaningless — and
		// ck_hypervisors_swarm_needs_hive rejects the row outright, since a
		// composite foreign key on (hive_id, swarm_id) stops being checked at all
		// once hive_id is NULL.
		//
		// What this gives up, said plainly: the delete used to fail against
		// fk_placements_hypervisor's RESTRICT when the ledger still held
		// placements on a node that had disappeared, which was a loud signal that
		// this system believed guests were running on a host the cluster had
		// forgotten. Clearing cannot fail that way. The discrepancy is still
		// visible — the row now reads as unclustered with zero capacity while
		// placements name it — but nothing raises it, and surfacing it is
		// deliberately left to the drift report rather than to a sync that would
		// otherwise stop syncing the whole hive over one stale placement.
		//
		// A named-column update rather than a struct save: hive_id, swarm_id and
		// name are exactly the nullable-to-NULL case GORM drops from a struct save,
		// and the observation columns carry `default:` tags, which it also omits at
		// zero. A struct save here would silently do nothing.
		for name, cur := range byName {
			if _, still := observations[name]; still {
				continue
			}
			if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", cur.ID).
				Updates(map[string]any{
					"hive_id":                nil,
					"swarm_id":               nil,
					"name":                   nil,
					"observed_cpu_total":     0,
					"observed_ram_gb_total":  0,
					"observed_disk_gb_total": 0,
					"observed_cpu_used":      0,
					"observed_ram_gb_used":   0,
					"observed_disk_gb_used":  0,
					"observed_guests":        0,
					// Zeroed with the rest. A machine in no cluster that kept a
					// foreign load would read as permanently oversubscribed, which
					// is the opposite mistake but still a wrong answer about a host
					// nothing can schedule onto.
					"observed_foreign_cpu_used":     0,
					"observed_foreign_ram_gb_used":  0,
					"observed_foreign_disk_gb_used": 0,
					"last_synced_at":                nil,
				}).Error; err != nil {
				return fmt.Errorf("the cluster no longer reports node %q, but clearing its "+
					"cluster membership failed: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("capacity sync: %w", err)
	}
	return written, nil
}

// managedVMIDs is the set of provider guest ids this hive's placement ledger
// accounts for — the guests the fold must NOT count as foreign, because they are
// already counted through intended usage.
//
// Flat, not per-node, and deliberately: a Proxmox VMID is unique across a
// CLUSTER, so a guest Proxmox migrated to a different host is still recognised
// as ours wherever the inventory finds it. Keying by node would make a migrated
// guest foreign on its new home while still counting as intended there,
// subtracting it twice.
//
// THE STATE FILTER IS THE PART THAT MUST NOT BE COPIED FROM waggle, which
// excludes every placement row carrying a vmid. Here a released or failed
// placement contributes nothing to intended usage, so if its VM is still on the
// node — leaked, or a deletion that failed — excluding its VMID would count that
// guest ZERO times and reproduce the exact bug the foreign accounting closes.
// The set is therefore derived from PlacementHoldsCapacity rather than restating
// the two state names, so the exclude set and intended usage cannot come to
// disagree about which rows count.
//
// Pending placements normally have no vmid yet and so never appear, which is
// right: nothing is on the node to discover, and the reservation is already
// carried by intended.
func managedVMIDs(tx *gorm.DB, hiveID uuid.UUID) (map[int]struct{}, error) {
	var ids []int
	if err := tx.Model(&tenant.Placement{}).
		Where("hive_id = ?", hiveID).
		Where("vmid IS NOT NULL").
		// AND NOT ZERO. FoldClusterResources treats a guest whose VMID never
		// arrived (zero) as foreign, which is only true while nothing puts 0 in
		// this set — a single placement row carrying a zero vmid would make every
		// unidentified guest in the cluster look like ours and stop its capacity
		// being subtracted at all, which is the exact failure the foreign
		// accounting exists to close. IS NOT NULL does not exclude a stored zero,
		// so it is excluded here rather than assumed away.
		Where("vmid <> 0").
		Where("state IN ?", placementStatesHoldingCapacity()).
		Distinct().Pluck("vmid", &ids).Error; err != nil {
		return nil, fmt.Errorf("read the placement ledger for this hive: %w", err)
	}
	set := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set, nil
}

// placementStatesHoldingCapacity filters every known placement state through
// PlacementHoldsCapacity rather than naming the two that pass.
//
// Roundabout on purpose. This list becomes SQL, and SQL cannot call the Go
// predicate that defines the rule; deriving it is what stops a future state — or
// a change of mind about whether 'failed' still holds capacity — from being
// applied to intended usage and silently not to the exclude set.
func placementStatesHoldingCapacity() []string {
	all := []string{
		tenant.PlacementPending,
		tenant.PlacementActive,
		tenant.PlacementFailed,
		tenant.PlacementReleased,
	}
	out := make([]string, 0, len(all))
	for _, s := range all {
		if tenant.PlacementHoldsCapacity(s) {
			out = append(out, s)
		}
	}
	return out
}

// withoutMachine drops one machine from the adoption pool.
func withoutMachine(pool []tenant.Hypervisor, id uuid.UUID) []tenant.Hypervisor {
	out := pool[:0]
	for _, h := range pool {
		if h.ID != id {
			out = append(out, h)
		}
	}
	return out
}

// releaseStandaloneMembers un-records membership of a hive whose endpoint turned
// out to be a standalone host, and returns how many rows it freed.
//
// The rows were never true. A machine cannot be a member of a cluster that does
// not exist, and while the row stands it is the thing that makes bootstrap
// refuse — "already in a hive", "the hive is not empty" — for the one host that
// has to create the cluster.
//
// Clears only what membership means: hive, swarm and the node name the cluster
// would have given it. The machine record itself survives, with its identity,
// address, onboarding history and agent intact — the same shape a departure
// leaves behind, because it is the same fact: this machine is in no cluster.
//
// Deliberately does NOT touch schedulable. A host recorded as unschedulable was
// cordoned by someone, and a correction to the control plane's own bookkeeping
// is not a reason to re-admit it to the placement query.
func releaseStandaloneMembers(
	ctx context.Context, d CapacitySyncDeps, orgID, hiveID uuid.UUID,
) (int64, error) {
	db, err := d.TenantDB(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("open tenant database: %w", err)
	}
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", hiveID).
		Updates(map[string]any{"hive_id": nil, "swarm_id": nil, "name": nil})
	return res.RowsAffected, res.Error
}

// countHiveMembers is how many hypervisors this hive is recorded as holding.
func countHiveMembers(
	ctx context.Context, d CapacitySyncDeps, orgID, hiveID uuid.UUID,
) (int64, error) {
	db, err := d.TenantDB(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("open tenant database: %w", err)
	}
	var n int64
	err = db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("hive_id = ?", hiveID).Count(&n).Error
	return n, err
}
