package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

var (
	// ErrMachineIDHeldElsewhere means another record already carries this
	// machine's id: the host is in the fleet twice, and binding the id here
	// would only move the duplicate.
	ErrMachineIDHeldElsewhere = errors.New("this machine's id already belongs to another hypervisor record")
	// ErrMachineIDMismatch means the record already belongs to a DIFFERENT
	// machine, so its address now reaches some other host.
	ErrMachineIDMismatch = errors.New("this hypervisor record belongs to a different machine")
)

// ClaimMachineID binds a machine id, read off the host over SSH, to the record
// onboarding is working on, so the agent's enrolment adopts that record.
//
// Idempotent: claiming the id a record already carries is a no-op, which is
// what a retried onboard does. It refuses rather than repairs in both other
// cases, because each means two records disagree about which host is which,
// and only an operator knows which one to keep.
func ClaimMachineID(ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID, machineID string) error {
	digest := tenant.MachineIDDigest(machineID)
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var other tenant.Hypervisor
		switch err := tx.Select("id", "address").
			Where("machine_id = ? AND id <> ?", digest, hypervisorID).First(&other).Error; {
		case err == nil:
			return fmt.Errorf("%w (record %s, address %q)", ErrMachineIDHeldElsewhere, other.ID, other.Address)
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		res := tx.Model(&tenant.Hypervisor{}).
			Where("id = ? AND (machine_id IS NULL OR machine_id = ?)", hypervisorID, digest).
			Updates(map[string]any{"machine_id": digest, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 1 {
			return nil
		}
		var exists int64
		if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", hypervisorID).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return fmt.Errorf("hypervisor record %s no longer exists", hypervisorID)
		}
		return ErrMachineIDMismatch
	})
}

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
		Updates(map[string]any{
			"hive_id": nil, "swarm_id": nil, "name": nil,
			// THIS IS THE MOMENT A DEPARTURE SUCCEEDED, so a stored departure
			// failure stops being true here.
			//
			// The capacity sync clears a stale JOIN outcome on the same
			// reasoning, and deliberately leaves a departure alone: a node that
			// failed to leave and is still a member has an accurate failure
			// beside it, and erasing it would delete a true statement about a
			// machine somebody is waiting on. That holds right up to the point
			// the node actually leaves — which is here, and which the sync never
			// sees, because a departed host is no longer in the cluster it
			// syncs.
			//
			// Nothing else can clear it. RecordOperationOutcome cleared it on the
			// success path, but it was called from the departure JOB, and the
			// departure moved to agent convergence and dispatches no job — so it
			// has no callers at all and the field is now written by nothing. A
			// failure recorded before that migration stayed on the row forever:
			// 192.168.240.12 sat in lab-a showing a 22-hour-old drain_unknown
			// refusal, having departed and rejoined twice since.
			"last_operation_error": gorm.Expr(
				"CASE WHEN last_operation = 'depart' THEN '' ELSE last_operation_error END"),
		}).Error; err != nil {
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
			// The join succeeded, so a stored join failure is spent. The capacity
			// sync clears this too, on its next tick and only for a node it can
			// see; doing it at the moment of truth means a machine is never
			// rendered with a contradiction it has already resolved.
			"last_operation_error": gorm.Expr(
				"CASE WHEN last_operation = 'join' THEN '' ELSE last_operation_error END"),
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
	// THE HOST IS THE PRIMARY OBSERVER, so it always records — there is no
	// condition on cluster membership any more.
	//
	// This was gated twice and both gates were wrong. First on `hive_id IS NULL`,
	// which assumed assigned meant joined: under convergence a machine is given a
	// hive_id as an INTENTION and joins later on its own, so in between it is in
	// a hive by the record and in no cluster in fact — invisible to the
	// per-cluster sync and skipped here. Then on what the agent observed, which
	// still left a CLUSTERED host depending on the capacity sync alone.
	//
	// That dependency is the problem. The sync needs a working hive API
	// credential, and when one is missing or stale it correctly refuses rather
	// than inventing membership — at which point last_synced_at freezes, the
	// drain gate reads "unknown", and no host in that hive can be departed. Every
	// node in a five-node lab sat two hours stale for exactly that reason, while
	// each of their agents was reporting every twenty-five seconds.
	//
	// The two writers do not fight. This records what a host counts on ITSELF
	// from /etc/pve; the sync records cluster-wide capacity the host does not
	// report. Where they overlap they should agree, and a disagreement is the
	// useful signal rather than a race — it is how a node that has silently
	// fallen out of its cluster becomes visible.
	_ = inACluster
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("bee_id = ?", beeID).
		Updates(map[string]any{
			"observed_guests": guests,
			"last_synced_at":  gorm.Expr("now()"),
		})
	if res.Error != nil {
		return res.Error
	}
	// MATCHING NOTHING IS NOT SUCCESS, and it is the failure this whole function
	// exists to prevent, wearing the other hat.
	//
	// The predicate is the agent's identity, but the row is the MACHINE, and the
	// link between them is set during onboarding. A machine whose bee_id was
	// never set — or was set to a bee that has since been re-enrolled, which a
	// rebuilt host does — matches zero rows. GORM reports no error for that: the
	// statement ran and updated nothing. So the agent reports every twenty-five
	// seconds, every write "succeeds", last_synced_at never moves, and the
	// departure gate refuses with drain_unknown citing an observation age that
	// only ever grows.
	//
	// Deliberately not an error return: failing the heartbeat over it would take
	// out convergence, membership and liveness for a host whose only real problem
	// is a missing link — and the gate already fails closed on the stale
	// observation, which is the safe direction. What was missing is any signal
	// saying WHY, which is what this logs.
	if res.RowsAffected == 0 {
		slog.ErrorContext(ctx, "an agent reported its guests and no machine claimed the "+
			"observation, so this host's drain freshness will never advance and every "+
			"departure from it refuses as drain_unknown",
			"bee_id", beeID, "observed_guests", guests)
	}
	return nil
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
	// AND ONLY FOR A MACHINE IN A HIVE. A node name means nothing outside a
	// cluster — a departure clears it for that reason — so recording one against
	// a fleetless row puts back exactly what the departure removed. The agent is
	// also guarded, but the condition belongs here too: this is the write.
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("bee_id = ? AND hive_id IS NOT NULL AND (name IS NULL OR name = '')", beeID).
		Update("name", node)
	if res.Error != nil {
		// A UNIQUE VIOLATION HERE IS A DUPLICATE MACHINE RECORD, not a transient
		// write failure, and it must not read as one. ux_hypervisor_hive_name is
		// unique on (hive_id, name), so this fails when some OTHER row in the
		// same hive already holds this node name — which happens when a sync
		// adopted the node as new while this row was still unnamed.
		//
		// It is unrecoverable from here and permanent: the real row can never be
		// given its name while the duplicate holds it, so it stays NULL forever
		// and the machine has two records. Observed on .12. The caller logs this
		// and continues, so the message has to say what is actually wrong rather
		// than looking like a failed write worth retrying.
		if isUniqueViolation(res.Error) {
			return fmt.Errorf("this host reports node name %q, but another hypervisor row in the "+
				"same hive already holds it — the machine has TWO records, and this one can never "+
				"be named while that is true. The duplicate is almost certainly a capacity sync "+
				"adoption of a machine that was assigned to the hive before it had a node name; "+
				"merge them: %w", node, res.Error)
		}
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

// isUniqueViolation reports whether an error is Postgres 23505.
//
// Matched on the SQLSTATE rather than on the message, which is localised and
// reworded between versions — the same trap that had isNotClustered grepping an
// API's prose for "no cluster".
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// MarkDeparting records that an operator has removed a machine from its hive,
// while the cluster still contains it.
//
// KEEPS hive_id AND name. That is what arms the eviction: a remaining member
// runs `pvecm delnode <node>`, and it can only do that if the composition NAMES
// the node. Clearing the row on the operator's click left nothing to name, which
// is why convergent eviction could be built and never armed.
//
// The machine leaves the member list and enters Departing in the same
// publication, so every remaining member learns in one step both that it should
// no longer be there and that removing it is authorised.
func MarkDeparting(ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID, disposition string) error {
	var row struct{ HiveID *uuid.UUID }
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Select("hive_id").Where("id = ?", hypervisorID).Scan(&row).Error; err != nil {
		return fmt.Errorf("read the hive this hypervisor is leaving: %w", err)
	}
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Updates(map[string]any{
			"departing_at": time.Now().UTC(),
			// Recorded now, acted on AFTER the cluster drops the node: disarming a
			// machine that is still a member destroys secrets it is still using.
			"credential_disposition": disposition,

			// Unschedulable immediately: a node on its way out must not be given
			// new work while it waits for a peer to remove it.
			"schedulable": false,
		}).Error; err != nil {
		return fmt.Errorf("mark %s departing: %w", hypervisorID, err)
	}
	// SNAPSHOT THE ROOT KEY THIS MACHINE HOLDS RIGHT NOW, so a rotation can be
	// checked afterwards. Without a before value there is nothing to compare the
	// after against, and the control plane would be taking the agent's word that
	// it replaced a key nobody else ever saw.
	//
	// Stored on the bee, which is where the observed key lives. Best effort: a
	// departure must not fail because a verification aid could not be recorded,
	// and its absence makes the rotation unjudged rather than judged wrong.
	if err := db.WithContext(ctx).Exec(`
		UPDATE bees SET departing_root_key = observed_root_key
		 WHERE id = (SELECT bee_id FROM hypervisors WHERE id = ?)`, hypervisorID).Error; err != nil {
		slog.WarnContext(ctx, "could not snapshot a departing host's root key, so a rotation "+
			"cannot be verified afterwards", "hypervisor_id", hypervisorID, "error", err)
	}

	publishMembership(ctx, db, row.HiveID)
	return nil
}

// FinaliseDepartures clears the machines a hive has actually dropped.
//
// EVIDENCE FROM A REMAINING MEMBER, not from the departing node. After `pvecm
// delnode` the removed host still believes it is a member — that is the whole
// reason it must not be powered back on — so asking it would get the wrong
// answer. A quorate peer that no longer lists it is the authority.
//
// Only then are hive_id, name and swarm_id cleared, which is the same
// MarkFleetless has always done. Doing it earlier is what let an agent read
// stale desired state and re-form the cluster it was being removed from; doing
// it never leaves the row claiming a membership the cluster has dropped.
func FinaliseDepartures(ctx context.Context, db *gorm.DB, hiveID uuid.UUID) (int, error) {
	// One quorate observation is enough, and it must come from a member that is
	// NOT itself departing — a node on its way out is not a witness to the
	// cluster's composition.
	var witness struct {
		Members datatypes.JSONType[[]string]
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT b.observed_members AS members
		  FROM hypervisors h
		  JOIN bees b ON b.id = h.bee_id
		 WHERE h.hive_id = ? AND h.departing_at IS NULL
		   AND b.observed_quorate = true
		   AND b.observed_membership_at > now() - interval '5 minutes'
		 LIMIT 1`, hiveID).Scan(&witness).Error; err != nil {
		return 0, fmt.Errorf("find a witness for departures from %s: %w", hiveID, err)
	}
	present := map[string]bool{}
	for _, n := range witness.Members.Data() {
		present[strings.ToLower(strings.TrimSpace(n))] = true
	}
	if len(present) == 0 {
		// No fresh quorate witness. Nothing is finalised rather than guessed:
		// clearing membership on no evidence is how a live node becomes a row
		// nobody owns.
		return 0, nil
	}

	var departing []tenant.Hypervisor
	if err := db.WithContext(ctx).
		Where("hive_id = ? AND departing_at IS NOT NULL", hiveID).
		Find(&departing).Error; err != nil {
		return 0, err
	}
	done := 0
	for _, h := range departing {
		if h.Name != nil && present[strings.ToLower(*h.Name)] {
			continue // still in the cluster; the eviction has not happened yet
		}
		// THE DISARM IS QUEUED BEFORE THE ROW IS CLEARED, and deliberately so.
		// The task names the peers whose configuration this host is still
		// carrying, and once hive_id is gone there is nothing left to derive them
		// from.
		queueDisarm(ctx, db, h, witness.Members.Data())

		if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Where("id = ?", h.ID).
			Updates(map[string]any{
				"hive_id": nil, "swarm_id": nil, "name": nil, "departing_at": nil,
				// credential_disposition is deliberately KEPT. It is the record of
				// what an operator asked for, and the disarm verdict needs it to
				// know whether a rotation was owed — clearing it here left every
				// departed machine unjudgeable on the one disposition that has an
				// observable consequence.
			}).Error; err != nil {
			return done, fmt.Errorf("finalise the departure of %s: %w", h.ID, err)
		}
		done++
	}
	if done > 0 {
		publishMembership(ctx, db, &hiveID)
	}
	return done, nil
}

// QueueDisarm asks a departed host to destroy the hive secrets it is carrying,
// on an operator's explicit request.
//
// The automatic path runs at finalisation and swallows its own failures, because
// a departure that has happened must not be reported as unfinished. This one is
// asked for, so a failure to queue is returned: the operator is waiting on an
// answer and "it did not work" is the answer.
func QueueDisarm(ctx context.Context, db *gorm.DB, h tenant.Hypervisor, disposition string) error {
	if h.BeeID == nil {
		return fmt.Errorf("this host has no agent, so nothing can destroy the secrets it holds")
	}
	spec, err := json.Marshal(map[string]any{"disposition": disposition, "peers": []string{}})
	if err != nil {
		return err
	}
	return NewFleetRepoGorm(db).CreateTask(ctx, &tenant.FleetTask{
		BeeID: *h.BeeID, Kind: KindDisarm,
		Spec: datatypes.JSON(spec), Status: tenant.FleetTaskPending,
	})
}

// queueDisarm asks a departed host to destroy the hive secrets it is carrying.
//
// A TASK rather than a convergence, because the disposition is an operator's
// decision that no observation implies — accept the exposure, or rotate this
// node's own identity. The WORK is local: writing files and restarting services
// on a host never needed the control plane to dial in, and running it over SSH
// made the control plane's own reachability a precondition of closing a
// credential exposure.
//
// Queued only once the cluster has actually dropped the node. Disarming a
// machine that is still a member destroys secrets it is still using.
//
// NEVER FAILS THE FINALISATION. The record is correct either way, and a
// departure that has happened must not be reported as unfinished because a
// follow-up could not be queued — that invites a retry of work already done. A
// host that never gets the task keeps the hive's secrets, which is what the
// error says.
func queueDisarm(ctx context.Context, db *gorm.DB, h tenant.Hypervisor, members []string) {
	if h.BeeID == nil {
		slog.WarnContext(ctx, "a departed host has no agent, so its copy of the hive's secrets "+
			"cannot be destroyed; it still authenticates against the cluster it left",
			"hypervisor_id", h.ID)
		return
	}
	peers := make([]string, 0, len(members))
	for _, n := range members {
		if n = strings.TrimSpace(n); n != "" {
			peers = append(peers, n)
		}
	}
	spec, err := json.Marshal(map[string]any{
		"disposition": h.CredentialDisposition,
		"peers":       peers,
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not build a disarm task", "hypervisor_id", h.ID, "error", err)
		return
	}
	task := tenant.FleetTask{
		BeeID:  *h.BeeID,
		Kind:   KindDisarm,
		Spec:   datatypes.JSON(spec),
		Status: tenant.FleetTaskPending,
	}
	if err := NewFleetRepoGorm(db).CreateTask(ctx, &task); err != nil {
		slog.WarnContext(ctx, "could not queue the disarm for a departed host; it still holds "+
			"the hive's secrets until one runs",
			"hypervisor_id", h.ID, "error", err)
	}
}

// KindDisarm matches the agent's fleetops.KindDisarm.
const KindDisarm = "fleet.datacenter.disarm"

// KindDistUpgrade matches the agent's fleetops.KindDistUpgrade. Named here
// because the outcome recorder has to label it for an operator, and a wire
// constant spelled out at each use site is one typo away from a task kind that
// silently matches nothing.
const KindDistUpgrade = "fleet.host.dist-upgrade"

// ObservedMembershipWindow is how recently an agent must have reported its
// cluster membership to be counted as a member.
const ObservedMembershipWindow = 5 * time.Minute

// HiveMember is one machine assigned to a hive, as the departure gate sees it.
type HiveMember struct {
	ID   uuid.UUID
	Name string
	// Departing: a departure has been requested and not yet finalised.
	Departing bool
	// DepartingFor is how long ago that departure was requested.
	DepartingFor time.Duration
	// InCluster: its agent reports being in this hive's cluster.
	InCluster bool
	// Fresh: that report is within ObservedMembershipWindow.
	Fresh bool
}

// Observed is whether the machine counts as a member at all: its own agent
// reported, recently, that it is in this cluster.
func (m HiveMember) Observed() bool { return m.InCluster && m.Fresh }

// HiveMembers is every machine assigned to a hive.
type HiveMembers []HiveMember

// LoadHiveMembers reads every machine assigned to a hive with what its agent
// last reported. The one query behind both counts the departure gate uses, so
// the node count and the remaining count cannot drift apart.
//
// ASSIGNMENT IS NOT MEMBERSHIP. Under convergence a machine gets a hive_id as an
// intention and joins afterwards, so rows include hosts that are not members
// yet; counting them let a departure take lab-b from two real members to one.
// A host whose agent has not reported recently is not counted either: that
// makes the gate refuse more readily, and a refused departure costs a retry
// where one allowed below the minimum costs quorum.
func LoadHiveMembers(ctx context.Context, db *gorm.DB, hiveID uuid.UUID, hiveName string) (HiveMembers, error) {
	var rows []struct {
		HiveMember
		DepartingSecs float64
	}
	err := db.WithContext(ctx).Raw(`
		SELECT h.id,
		       coalesce(h.name, '') AS name,
		       h.departing_at IS NOT NULL AS departing,
		       coalesce(extract(epoch FROM now() - h.departing_at), 0) AS departing_secs,
		       coalesce(lower(b.observed_cluster) = lower(?), false) AS in_cluster,
		       coalesce(b.observed_membership_at > now() - make_interval(secs => ?), false) AS fresh
		  FROM hypervisors h
		  LEFT JOIN bees b ON b.id = h.bee_id
		 WHERE h.hive_id = ?
		 ORDER BY h.name`,
		hiveName, ObservedMembershipWindow.Seconds(), hiveID).Scan(&rows).Error
	out := make(HiveMembers, 0, len(rows))
	for _, r := range rows {
		r.HiveMember.DepartingFor = time.Duration(r.DepartingSecs * float64(time.Second))
		out = append(out, r.HiveMember)
	}
	return out, err
}

// Present is how many nodes the hive has right now, the departing ones included.
func (ms HiveMembers) Present() int {
	n := 0
	for _, m := range ms {
		if m.Observed() {
			n++
		}
	}
	return n
}

// RemainingAfter is how many members the hive keeps once hypervisorID and every
// departure already requested have gone.
//
// The departure minimum is a floor on THIS number. A node stops counting the
// moment its departure is requested, whatever phase it has reached — still
// voting, already evicted, or disarming — because to the cluster all three are
// a node it is losing.
func (ms HiveMembers) RemainingAfter(hypervisorID uuid.UUID) int {
	n := 0
	for _, m := range ms {
		if m.ID != hypervisorID && m.Observed() && !m.Departing {
			n++
		}
	}
	return n
}

// NotCountedAfter names the machines, other than hypervisorID, that are
// assigned to the hive but not counted as remaining, each with the reason. It
// is what turns "would be left with 2" on a hive the UI shows with 4 nodes into
// something an operator can act on.
func (ms HiveMembers) NotCountedAfter(hypervisorID uuid.UUID) []string {
	var out []string
	for _, m := range ms {
		if m.ID == hypervisorID {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID.String()
		}
		switch {
		case m.Departing:
			// With its age, because a departure whose eviction has stalled keeps
			// the node out of the count indefinitely, and an old one here is the
			// first sign of that.
			out = append(out, fmt.Sprintf("%s (departure requested %s ago)",
				name, m.DepartingFor.Truncate(time.Minute)))
		case !m.InCluster:
			out = append(out, name+" (its agent does not report being in this cluster)")
		case !m.Fresh:
			out = append(out, fmt.Sprintf("%s (no membership report in the last %s)",
				name, ObservedMembershipWindow))
		}
	}
	return out
}

// ObservedHiveNodeCount is how many of a hive's machines are actually in its
// cluster right now, departing ones included. See LoadHiveMembers.
func ObservedHiveNodeCount(ctx context.Context, db *gorm.DB, hiveID uuid.UUID, hiveName string) (int, error) {
	ms, err := LoadHiveMembers(ctx, db, hiveID, hiveName)
	if err != nil {
		return 0, err
	}
	return ms.Present(), nil
}
