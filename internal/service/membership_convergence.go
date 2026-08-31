package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

// DesiredMembershipFor returns the composition the host running this bee should
// converge on, or nil when there is nothing to converge on.
//
// NIL MEANS HOLD, exactly as an absent firewall does. It is returned for a
// machine in no hive and for a hive that has published nothing, and neither is
// "you should be in no cluster" — reading it that way would make an unpublished
// hive dissolve itself node by node the first time every agent polled.
//
// The composition is re-rendered here rather than read from a stored body, and
// the revision comes from the hive row. A published revision whose rendered
// digest no longer matches means a write changed membership without publishing;
// that is repaired by publishing rather than served, because serving a body
// whose digest disagrees with the one the heartbeat advertised would make the
// agent converge and immediately be judged diverged again — forever.
func (s *FleetService) DesiredMembershipFor(
	ctx context.Context, sess *BeeSession,
) (*tenant.MembershipDesired, error) {
	if sess == nil || sess.Bee == nil {
		return nil, repo.ErrBeeNotFound
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return nil, err
	}
	return repo.DesiredMembershipForBee(ctx, db, sess.Bee.ID)
}

// RecordObservedMembership stores what a host says about the cluster it can see.
//
// UNCONDITIONAL, unlike the observed guest count, which is written only for a
// fleetless machine to keep a single writer per row. There is no competing
// writer here: the capacity sync records what the Proxmox API says about a
// CLUSTER, and this records what one HOST says about itself. They are allowed to
// disagree, and a disagreement between them is the useful signal — it is how a
// node that has silently fallen out of corosync while still answering its API
// becomes visible.
//
// An agent that did not report is distinguished from one reporting a standalone
// host by observed_membership_at, which is why the caller passes a pointer: an
// older agent omits the field entirely, and reading that as "standalone" would
// invent an observation nobody made and hand it to a loop that acts on it.
func (s *FleetService) RecordObservedMembership(
	ctx context.Context, sess *BeeSession, obs *tenant.Observed,
	fingerprint string, hostKeys []string, rootKey string, secrets *int, carriesConfig *bool,
) error {
	if sess == nil || sess.Bee == nil {
		return repo.ErrBeeNotFound
	}
	if obs == nil {
		return nil
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return err
	}
	return repo.RecordObservedMembership(ctx, db, sess.Bee.ID, *obs, fingerprint, hostKeys, rootKey, secrets, carriesConfig)
}

// RecordAppliedMembership stores the composition a bee reports it has converged
// on.
func (s *FleetService) RecordAppliedMembership(
	ctx context.Context, sess *BeeSession, revision int, digest string,
) error {
	if sess == nil || sess.Bee == nil {
		return repo.ErrBeeNotFound
	}
	if revision <= 0 || digest == "" {
		return nil
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return err
	}
	return repo.RecordAppliedMembership(ctx, db, sess.Bee.ID, revision, digest)
}

// PublishMembership commits a hive's composition, and is safe to call when
// nothing changed.
//
// Exposed on the service because every write that changes membership intent has
// to call it, and those writes live in handlers and jobs that should not be
// reaching into repo directly.
func (s *FleetService) PublishMembership(ctx context.Context, db *gorm.DB, hiveID uuid.UUID) error {
	_, err := repo.PublishHiveMembership(ctx, db, hiveID)
	if errors.Is(err, repo.ErrMembershipUnchanged) {
		return nil
	}
	return err
}

// staleObservation is how long an agent's report stays usable.
//
// Not a liveness check — the bee's own status covers that. This bounds how old
// an observation may be before a gate refuses to reason from it, and it is
// deliberately generous relative to the poll interval: an agent that missed two
// heartbeats is not yet a host whose cluster state is unknown.
const staleObservation = 5 * time.Minute

// ObservationFresh reports whether a bee's membership observation is recent
// enough to gate on.
func ObservationFresh(b *tenant.Bee, now time.Time) bool {
	if b == nil || b.ObservedMembershipAt == nil {
		return false
	}
	return now.Sub(*b.ObservedMembershipAt) <= staleObservation
}

// RecordObservedNodeName stores the Proxmox node name a host reports for itself.
//
// A machine has no node name until it is in a cluster, so an arrival records
// membership without one and the host supplies it once it has joined. Written
// only when the row has none — see the repo function for why a rename must not
// come from a heartbeat.
func (s *FleetService) RecordObservedNodeName(
	ctx context.Context, sess *BeeSession, node string,
) error {
	if sess == nil || sess.Bee == nil {
		return repo.ErrBeeNotFound
	}
	if node == "" {
		return nil
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return err
	}
	return repo.RecordObservedNodeName(ctx, db, sess.Bee.ID, node)
}

// FinaliseDepartures clears the machines this bee's hive has actually dropped.
//
// Driven from the heartbeat because that is when the evidence arrives: a quorate
// member reporting a composition that no longer lists a departing node is the
// authority. The departing node itself is not — after `pvecm delnode` it still
// believes it is a member, which is precisely why it must not be powered back
// on.
func (s *FleetService) FinaliseDepartures(ctx context.Context, sess *BeeSession) error {
	if sess == nil || sess.Bee == nil {
		return repo.ErrBeeNotFound
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return err
	}
	var row struct{ HiveID *uuid.UUID }
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Select("hive_id").Where("bee_id = ?", sess.Bee.ID).Scan(&row).Error; err != nil {
		return err
	}
	if row.HiveID == nil {
		return nil
	}
	_, err = repo.FinaliseDepartures(ctx, db, *row.HiveID)
	return err
}
