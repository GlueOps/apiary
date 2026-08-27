package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

// dbFor opens the caller's tenant database.
//
// Separate from fleetFor because the firewall's publish and convergence paths
// want the handle itself: publishing is one transaction spanning two tables, and
// wrapping that in a repository interface would only hide the transaction the
// correctness argument rests on.
func (s *FleetService) dbFor(ctx context.Context, orgID string) (*gorm.DB, error) {
	if orgID == "" || isNilResolver(s.tenants) {
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}
	return tCtx.DB, nil
}

// DesiredFirewallFor returns what the host running this bee should be
// enforcing, or nil for HOLD.
//
// Nil is a real answer with a specific meaning, and callers must not flatten it
// into an empty ruleset: a bee whose machine is between fleets, or whose hive
// has published nothing, is told to keep what it has. Flushing there would strip
// a mid-migration host of its management access at the moment it is least
// reachable.
func (s *FleetService) DesiredFirewallFor(
	ctx context.Context, sess *BeeSession,
) (*tenant.FirewallSnapshot, error) {
	if sess == nil || sess.Bee == nil {
		return nil, repo.ErrBeeNotFound
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return nil, err
	}
	return repo.DesiredForBee(ctx, db, sess.Bee.ID)
}

// RecordAppliedFirewall stores what a bee reports it is enforcing.
//
// Reported rather than assumed, because a publish only says what should be
// running. Marking the fleet converged when an operator clicks Publish would
// report a policy as enforced on every host that happened to be switched off.
func (s *FleetService) RecordAppliedFirewall(
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
	return repo.RecordAppliedFirewall(ctx, db, sess.Bee.ID, revision, digest)
}

// RecordObservedGuests stores what a host says it is running.
//
// Written ONLY for a machine in no cluster, which is the gap: the capacity sync
// observes per cluster, so a fleetless machine is never observed by it at all,
// and fleetless is exactly the state a machine is in when it is about to join
// one. Restricting it to that case keeps a single writer per row rather than
// letting an agent report and a sync overwrite in turn, which would flap on
// every tick where the two disagree.
//
// The condition lives in the UPDATE rather than in a read-then-write, so a
// machine that joined a cluster between the two cannot have its swept-up
// observation land afterwards.
func (s *FleetService) RecordObservedGuests(
	ctx context.Context, sess *BeeSession, guests *int,
) error {
	if sess == nil || sess.Bee == nil {
		return repo.ErrBeeNotFound
	}
	// nil means the agent did not say. Zero means it said none, and those must
	// not collapse: an older agent omits the field, and treating silence as
	// "no guests" would invent the very observation this exists to stop.
	if guests == nil {
		return nil
	}
	db, err := s.dbFor(ctx, sess.OrganizationID.String())
	if err != nil {
		return err
	}
	return repo.RecordObservedGuestsIfFleetless(ctx, db, sess.Bee.ID, *guests)
}
