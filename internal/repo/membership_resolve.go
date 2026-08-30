package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// DesiredMembershipForBee resolves the composition one bee's host should
// converge on.
//
// Returns (nil, nil) — HOLD — for a machine in no hive and for a hive that has
// published nothing. Neither is an instruction to leave a cluster, and treating
// them as one would dissolve an unpublished hive node by node.
func DesiredMembershipForBee(ctx context.Context, db *gorm.DB, beeID uuid.UUID) (*tenant.MembershipDesired, error) {
	var row struct {
		HiveID   *uuid.UUID
		ID       uuid.UUID
		Node     *string
		Revision int
		Digest   string
		Applied  int
		Issued   int
		IssuedAt *time.Time
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT h.hive_id            AS hive_id,
		       h.id                 AS id,
		       h.name               AS node,
		       COALESCE(v.membership_revision, 0)        AS revision,
		       COALESCE(v.membership_digest, '')         AS digest,
		       COALESCE(b.applied_membership_revision, 0) AS applied,
		       COALESCE(b.issued_membership_revision, 0)  AS issued,
		       b.issued_membership_at                     AS issued_at
		  FROM hypervisors h
		  LEFT JOIN hives v ON v.id = h.hive_id
		  LEFT JOIN bees  b ON b.id = h.bee_id
		 WHERE h.bee_id = ?`, beeID).Scan(&row).Error; err != nil {
		return nil, fmt.Errorf("membership: resolve the bee's hive: %w", err)
	}

	// NO HIVE is the only HOLD. A hive that has published nothing is NOT one:
	// publication is derived from the hypervisors assigned to it, so a hive with
	// members always has a composition, and the first publish has to happen
	// somewhere. Returning early on revision 0 made that impossible — the repair
	// below sat behind a check that bailed before reaching it.
	if row.HiveID == nil {
		return nil, nil
	}

	// Repair first: a rendered digest that disagrees with the published one means
	// either a write moved membership without publishing, or this hive has never
	// published at all. Idempotent, so the common case costs one locked read.
	name, members, departing, err := ResolveMembership(ctx, db, *row.HiveID)
	if err != nil {
		return nil, err
	}
	if _, digest := tenant.RenderMembership(name, members, departing); digest != row.Digest {
		res, perr := PublishHiveMembership(ctx, db, *row.HiveID)
		switch {
		case perr == nil, errors.Is(perr, ErrMembershipUnchanged):
			row.Revision, row.Digest = res.Revision, res.Digest
		default:
			return nil, perr
		}
	}

	// ONE STEP AT A TIME.
	//
	// The agent is issued exactly the revision after the one it has applied, and
	// nothing further until it acknowledges. Acknowledged means REACHED: the
	// agent re-observes corosync after acting and records a revision as applied
	// only when observed state equals desired state at that revision.
	//
	// Stepping is why every revision's composition is stored. Serving a step
	// means serving the body as published at that number — today's members under
	// an older number would be applied and reported as a digest nobody issued,
	// and the host would never be judged converged.
	next := row.Applied + 1
	if next > row.Revision {
		next = row.Revision
	}
	if next < 1 {
		next = 1
	}

	// While a step is in flight the SAME step is re-served, so a restarted agent
	// resumes the one it was given rather than being advanced past it. The stall
	// timeout stops a crashed agent wedging its own hive forever.
	if row.Issued > 0 && row.Issued != row.Applied &&
		row.IssuedAt != nil && time.Since(*row.IssuedAt) <= membershipStall {
		next = row.Issued
	}

	// A STEP THE HOST CANNOT REACH IS SKIPPED, not re-served forever.
	//
	// Stepping replays history, and history contains states that were withdrawn
	// before anyone acted on them. Observed live: lab-b went {pve3,pve4,pve5} at
	// 17, {pve3,pve4} at 18, and back to {pve3,pve4,pve5} by 21. pve5 applied 17,
	// was issued 18 — where it is desired-absent while sitting in the cluster
	// with peers — and Decide correctly answered VerbWait, because a departing
	// node cannot remove itself. That is not VerbNone, so it never acknowledged,
	// and 18 was re-served indefinitely.
	//
	// The agent was right and the step was impossible. So the control plane
	// checks reachability before issuing: it holds each host's own observation,
	// so it can run the same decision the agent will and advance past any step
	// whose answer is one only another actor could resolve.
	//
	// Only unreachable steps are skipped. A step that asks for real work — a
	// join, a create, an eviction — is issued and waited on, which is the whole
	// point of stepping.
	next = skipUnreachableSteps(ctx, db, *row.HiveID, next, row.Revision,
		tenant.Self{ID: row.ID.String(), Node: selfName(row.Node)}, observedOf(ctx, db, beeID))

	var doc tenant.Membership
	var digest string
	var snap tenant.MembershipSnapshot
	if err := db.WithContext(ctx).
		Where("hive_id = ? AND revision = ?", *row.HiveID, next).
		First(&snap).Error; err == nil {
		doc, digest = snap.Content.Data(), snap.Digest
	} else {
		// No stored body at that step — a hive published before snapshots
		// existed, or a gap in the sequence. Stepping cannot replay what was
		// never recorded, so serve the current composition at the current
		// revision rather than holding forever.
		next, digest = row.Revision, row.Digest
		doc, _ = tenant.RenderMembership(name, members, departing)
	}

	if next != row.Issued {
		if err := markMembershipIssued(ctx, db, beeID, next); err != nil {
			return nil, err
		}
	}

	return &tenant.MembershipDesired{
		Revision: next, Digest: digest, Membership: doc,
		Self: selfName(row.Node), SelfID: row.ID.String(),
	}, nil
}

// RecordObservedMembership stores a host's own report of the cluster it sees.
func RecordObservedMembership(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, obs tenant.Observed,
	fingerprint string, hostKeys []string, rootKey string,
) error {
	// Named columns, never a struct: an Updates over a Bee struct would write
	// every zero-valued field on the model and blank the agent version, the
	// applied firewall revision and the unavailability declaration along with it.
	fields := map[string]any{
		"observed_cluster":       obs.Cluster,
		"observed_members":       datatypes.NewJSONType(obs.Members),
		"observed_quorate":       obs.Quorate,
		"observed_membership_at": time.Now().UTC(),
	}
	// Only written when the host actually reported one. An agent that could not
	// read its own certificate must not blank the fingerprint every other node
	// joins through.
	if fingerprint != "" {
		fields["observed_fingerprint"] = fingerprint
	}
	// Same treatment: an agent that could not read its own host keys must not
	// blank the ones every other node pins in order to join through it.
	if len(hostKeys) > 0 {
		fields["observed_host_keys"] = datatypes.NewJSONType(hostKeys)
	}
	// Same treatment: an agent that could not read its own root key must not
	// blank the value a cluster needs in order to admit it.
	if rootKey != "" {
		fields["observed_root_key"] = rootKey
	}
	return db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", beeID).
		Updates(fields).Error
}

// RecordAppliedMembership stores the composition a bee has converged on.
func RecordAppliedMembership(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, revision int, digest string,
) error {
	return db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", beeID).
		Updates(map[string]any{
			"applied_membership_revision": revision,
			"applied_membership_digest":   digest,
			"applied_membership_at":       time.Now().UTC(),
		}).Error
}

// membershipStall bounds how long an unacknowledged issue is honoured.
//
// Generous: a join restarts pve-cluster and corosync and takes minutes, and
// reissuing underneath a host that is mid-convergence is the thing this gate
// exists to prevent. The cost of waiting too long is a slow hive; the cost of
// waiting too little is the behaviour that had an agent acting on a composition
// that had already been replaced.
const membershipStall = 10 * time.Minute

// markMembershipIssued records what this agent was just handed.
func markMembershipIssued(ctx context.Context, db *gorm.DB, beeID uuid.UUID, revision int) error {
	return db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", beeID).
		Updates(map[string]any{
			"issued_membership_revision": revision,
			"issued_membership_at":       time.Now().UTC(),
		}).Error
}

// selfName is this host's node name, or empty before it has one.
func selfName(name *string) string {
	if name == nil {
		return ""
	}
	return *name
}

// observedOf reads what this host last reported about its own cluster.
//
// Empty when it has never reported: an observation nobody made must not be
// invented, so a host that has said nothing is never judged unable to reach a
// step — it is simply issued it.
func observedOf(ctx context.Context, db *gorm.DB, beeID uuid.UUID) *tenant.Observed {
	var bee tenant.Bee
	if err := db.WithContext(ctx).First(&bee, "id = ?", beeID).Error; err != nil {
		return nil
	}
	if bee.ObservedMembershipAt == nil {
		return nil
	}
	return &tenant.Observed{
		Cluster: bee.ObservedCluster,
		Members: bee.ObservedMembers.Data(),
		Quorate: bee.ObservedQuorate,
	}
}

// skipUnreachableSteps advances past revisions this host could never acknowledge.
//
// A step is unreachable when the decision it produces is one only ANOTHER actor
// can resolve — waiting for a peer to evict this node, or blocked on an operator
// moving it between clusters. Re-serving such a step forever is a deadlock, and
// it is a deadlock in which the agent is behaving correctly.
//
// Bounded by the latest revision, so this always terminates: at worst it arrives
// at the newest composition, which is by definition the one the fleet is meant
// to be in.
func skipUnreachableSteps(
	ctx context.Context, db *gorm.DB, hiveID uuid.UUID, from, latest int,
	self tenant.Self, obs *tenant.Observed,
) int {
	if obs == nil {
		return from
	}
	for step := from; step < latest; step++ {
		var snap tenant.MembershipSnapshot
		if err := db.WithContext(ctx).
			Where("hive_id = ? AND revision = ?", hiveID, step).
			First(&snap).Error; err != nil {
			return step
		}
		verb, _, _ := snap.Content.Data().Decide(self, *obs)
		if verb != tenant.VerbWait && verb != tenant.VerbBlocked {
			return step
		}
	}
	return latest
}
