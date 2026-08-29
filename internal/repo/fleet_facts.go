package repo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// The three facts the onboarding job needs from the tenant database.
//
// These lived as closures inside cmd/worker.go, which put them somewhere no test
// could reach: cmd has no database fixture, so the whole suite stayed green with
// any of the three wrong. That is not a theoretical worry — each one turns on a
// detail the compiler cannot check. Heartbeated compares a DIGEST against a raw
// id, MarkFleetless names three columns as strings, and MarkJoined guards a
// field the type system says may legitimately be empty. Named here, they get the
// fixture that internal/repo already has, and cmd/worker.go keeps only the
// wiring: resolve the tenant database, call these.

// Heartbeated reports whether the agent on this machine has checked in since the
// given instant.
//
// TWO ROWS, ONE QUESTION. The machine id identifies the MACHINE and the
// heartbeat belongs to the AGENT on it, so the query has to cross the link
// between them — it once read bees.machine_id, a column that no longer exists,
// and a string column name is not something the compiler would have caught.
//
// The id is DIGESTED here. Enrolment stores the digest while the job carries the
// raw /etc/machine-id, so comparing them directly could never match: this would
// always answer false and every onboard would end in ErrNoHeartbeat, however
// healthy the host. See TestHeartbeatedDigestsTheMachineID, which is the only
// thing between that and a silent, fleet-wide onboarding failure.
//
// `since` is taken BEFORE the install, so an agent left running by a previous
// attempt is not accepted as proof that this one worked.
func Heartbeated(ctx context.Context, db *gorm.DB, machineID string, since time.Time) (bool, error) {
	var n int64
	err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Joins("JOIN bees ON bees.id = hypervisors.bee_id").
		Where("hypervisors.machine_id = ? AND bees.last_seen_at IS NOT NULL "+
			"AND bees.last_seen_at >= ?", tenant.MachineIDDigest(machineID), since).
		Count(&n).Error
	return n > 0, err
}

// MarkFleetless records that a hypervisor is no longer in any cluster.
//
// All three columns go together. The name is the node name INSIDE a cluster and
// means nothing outside one, and swarm_id is a grouping within a hive — leaving
// either behind would leave a row claiming membership of a cluster it has just
// left. ck_hypervisors_swarm_needs_hive refuses the swarm_id half outright, so
// dropping it from this map turns a departure into a constraint violation.
func MarkFleetless(ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID) error {
	// Read BEFORE the clear. Which hive this machine is leaving is the one thing
	// the update destroys, and it is exactly what has to be republished
	// afterwards so the remaining members learn the node is desired-absent.
	// Scanned into a struct rather than plucked: Pluck hands the driver value
	// straight to a scalar destination, and a uuid arrives as a string that a
	// *uuid.UUID cannot take.
	var leaving struct{ HiveID *uuid.UUID }
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Select("hive_id").Where("id = ?", hypervisorID).Scan(&leaving).Error; err != nil {
		return fmt.Errorf("read the hive this hypervisor is leaving: %w", err)
	}

	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Updates(map[string]any{"hive_id": nil, "swarm_id": nil, "name": nil}).Error; err != nil {
		return err
	}
	publishMembership(ctx, db, leaving.HiveID)
	return nil
}

// publishMembership republishes a hive's composition after its membership
// changed, and never fails the operation that changed it.
//
// Best effort ON PURPOSE, which is safe only because the resolve path repairs
// it: DesiredMembershipForBee re-renders on every heartbeat and publishes when
// the rendered digest disagrees with the stored one, so a publish missed here is
// picked up by the next heartbeat from any member of that hive. Doing it here as
// well is what makes convergence start in the same second as the write rather
// than on the next poll.
//
// Failing the caller instead would be worse than the delay it avoids: a
// departure that has already made a node standalone and then reports failure
// gets retried, and the retry re-runs a destructive step that has already
// happened.
func publishMembership(ctx context.Context, db *gorm.DB, hiveID *uuid.UUID) {
	if hiveID == nil || *hiveID == uuid.Nil {
		return
	}
	if _, err := PublishHiveMembership(ctx, db, *hiveID); err != nil &&
		!errors.Is(err, ErrMembershipUnchanged) {
		slog.WarnContext(ctx, "could not publish a hive's cluster composition after a membership "+
			"change; the next heartbeat from any of its members will republish it",
			"hive_id", *hiveID, "error", err)
	}
}

// MarkJoined records that a hypervisor has joined a hive under a node name, and
// leaves it UNSCHEDULABLE.
//
// A node that has just joined has not been validated in its new home. Its
// capacity has not been observed there — Proxmox does not populate
// /cluster/resources stats for a node the instant it joins, so the first sync
// after an arrival legitimately reports zero cores and zero memory — its
// firewall rules have not converged, and the agent may not yet have checked in.
// Landing a guest on it in that window is not something the control plane should
// decide on its own, so an operator makes it schedulable deliberately, having
// looked at what the sync eventually reported.
//
// Explicit false in a map, which is what makes it work. Schedulable carries a
// `default:true` tag, and GORM omits a zero value from an INSERT for any field
// that has one — so a struct with Schedulable:false stores TRUE. Updates() with
// an explicit map is not subject to that.
//
// The empty-name guard is the point. Membership without a node name leaves a row
// the next capacity sync cannot match to anything the Proxmox API reports, so
// the sync adopts the node as if it were new and the cluster ends up holding two
// rows for one machine — one real, one unmatchable. Refusing here costs a failed
// job and a retry; accepting costs a duplicate that only a human notices.
func MarkJoined(ctx context.Context, db *gorm.DB, hypervisorID, hiveID uuid.UUID, nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("the node joined %s but no node name was resolved for it, "+
			"so recording membership would leave a row the next capacity sync "+
			"cannot match and would duplicate", hiveID)
	}
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Updates(map[string]any{
			"hive_id":     hiveID,
			"name":        nodeName,
			"schedulable": false,
		}).Error; err != nil {
		return err
	}
	publishMembership(ctx, db, &hiveID)
	return nil
}

// RecordObservedGuestsIfFleetless writes a host's self-reported guest count, and
// only while it belongs to no hive.
//
// The hive_id IS NULL predicate is the whole point. A clustered machine's
// observations belong to the capacity sync, which sees the whole cluster and can
// tell this host's guests from its peers'; a fleetless one is invisible to that
// sync entirely, because the sync runs per hive. Two writers on one column would
// otherwise take turns.
func RecordObservedGuestsIfFleetless(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, guests int, inACluster bool,
) error {
	// ASSIGNED IS NOT THE SAME AS JOINED, and keying on hive_id alone assumed it
	// was. Under convergence a machine is given a hive_id as an INTENTION and
	// joins later on its own, so between those two moments it is in a hive by
	// the record and in no cluster in fact. The capacity sync could not see it —
	// it is not in the cluster — and this writer skipped it because hive_id was
	// set. Neither wrote, last_synced_at froze, the drain gate read "unknown",
	// and the host could never be departed. Observed on .12, stuck at a
	// 16-minute-old sync that was never going to be superseded.
	//
	// The condition is therefore what the AGENT observes, not what the record
	// intends: a host that reports no cluster is one the per-cluster sync
	// structurally cannot cover, whatever its hive_id says.
	if inACluster {
		return nil
	}
	return db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("bee_id = ?", beeID).
		Updates(map[string]any{
			"observed_guests": guests,
			"last_synced_at":  gorm.Expr("now()"),
		}).Error
}

// RecordHostKeys stores a machine's own SSH host public keys.
//
// Captured during onboarding, over the control plane's authenticated session to
// that host, and used later by a DIFFERENT machine: a node joining a cluster
// must pin its peer's key before `pvecm add` runs ssh-copy-id, or the join fails
// with "unable to copy ssh ID: exit code 1" and nothing else.
//
// Named columns rather than a struct, so this cannot blank the rest of the row.
func RecordHostKeys(ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID, keys []string) error {
	if len(keys) == 0 {
		// Refused rather than stored empty. An empty list is indistinguishable
		// from "never captured" at the point it is read, and the reader's choice
		// between them is whether to attempt a join at all.
		return fmt.Errorf("refusing to record an empty set of host keys for %s", hypervisorID)
	}
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Update("ssh_host_keys", datatypes.NewJSONType(keys)).Error; err != nil {
		return err
	}
	// The composition carries these to the hosts that need them, so a capture
	// that does not republish leaves a joiner unable to pin a peer it can see.
	var hive struct{ HiveID *uuid.UUID }
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Select("hive_id").Where("id = ?", hypervisorID).Scan(&hive).Error; err == nil {
		publishMembership(ctx, db, hive.HiveID)
	}
	return nil
}

// AssignToHive records that a machine SHOULD be a member of a hive, without
// claiming it already is.
//
// The difference from MarkJoined is the whole point of convergence. MarkJoined
// records an accomplished fact and demands a node name, because a membership
// without one leaves a row the capacity sync cannot match. This records an
// INTENTION, and a machine that has never been in a cluster has no node name to
// give — the name is assigned when it joins one. Requiring it here would make
// the arrival impossible to express.
//
// The name is therefore left NULL and filled in when the host reports the name
// it actually got. That is sound because ux_hypervisor_hive_name is partial on
// `name IS NOT NULL`, so an unnamed member does not collide with anything.
//
// Publishing is what makes it happen: the hive's composition changes, every
// member sees the new revision, and the machine itself derives create-or-join
// from what corosync reports. Nothing is dispatched.
func AssignToHive(ctx context.Context, db *gorm.DB, hypervisorID, hiveID uuid.UUID) error {
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Updates(map[string]any{
			"hive_id": hiveID,
			// UNSCHEDULABLE until somebody looks. A node that has not joined yet
			// certainly cannot take work, and one that just has still has not been
			// validated in its new home — see MarkJoined for why that stays an
			// operator's decision.
			"schedulable": false,
		}).Error; err != nil {
		return fmt.Errorf("assign %s to hive %s: %w", hypervisorID, hiveID, err)
	}
	publishMembership(ctx, db, &hiveID)
	return nil
}

// RecordObservedNodeName fills in the Proxmox node name a host reports for
// itself.
//
// The name is not knowable before a machine is in a cluster, so an arrival
// assigns membership without one and this closes the loop afterwards. Written
// only when the row has none: once a machine has a name, the capacity sync and
// the composition are both keyed on it, and letting a heartbeat rename a row out
// from under them would strand every reference to it.
func RecordObservedNodeName(ctx context.Context, db *gorm.DB, beeID uuid.UUID, node string) error {
	if node == "" {
		return nil
	}
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("bee_id = ? AND (name IS NULL OR name = '')", beeID).
		Update("name", node)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		// The composition identifies members by node name to their peers, so a
		// name arriving changes what every other member should see.
		var hive struct{ HiveID *uuid.UUID }
		if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Select("hive_id").Where("bee_id = ?", beeID).Scan(&hive).Error; err == nil {
			publishMembership(ctx, db, hive.HiveID)
		}
	}
	return nil
}
