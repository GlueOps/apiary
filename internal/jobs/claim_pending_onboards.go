package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
)

// ClaimPendingOnboards takes every claimable machine in one organization and
// enqueues an onboard job for each.
//
// CLAIM THEN ENQUEUE, never the other way round. The claim is the exclusivity
// token: it flips the row to `provisioning` and stamps a lease in ONE statement,
// so a row this tick took cannot also be taken by a concurrent tick or pushed
// back to pending by a redrive that read it a moment earlier. Enqueuing first
// and claiming after would leave a window in which two workers hold jobs for the
// same host, both installing keys and reloading sshd on it.
//
// An enqueue that fails after the claim leaves the row in `provisioning` with a
// live lease and no job behind it. That is deliberately not rolled back: the
// lease expires, the row becomes claimable again by its own rule, and the next
// tick picks it up. Rolling back would mean a second write that can itself fail,
// and the recovery path already exists for the worker-died case.
func ClaimPendingOnboards(
	ctx context.Context,
	db *gorm.DB,
	orgID uuid.UUID,
	insert func(ctx context.Context, args river.JobArgs) error,
) (claimed int, err error) {
	var rows []tenant.Hypervisor
	// Read first only to know WHICH rows to attempt. The read is not the
	// decision — every row is re-tested by the claim's own predicate, in the same
	// statement as the write, so a row that stopped being claimable in between is
	// simply not claimed.
	if err := db.WithContext(ctx).
		Where(tenant.ClaimablePredicate).
		Where(tenant.UnderAttemptCapPredicate, tenant.MaxOnboardAttempts).
		// A machine with no address has nothing to connect to, and claiming it
		// would burn an attempt per tick on a row that cannot progress until an
		// operator records an address. The redrive endpoint refuses for the same
		// reason and says so; here it is silent, because a tick is not a request.
		Where("address <> ''").
		Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("claim onboards: list claimable machines: %w", err)
	}

	for _, hv := range rows {
		ok, err := claimOne(ctx, db, hv.ID)
		if err != nil {
			// One row failing must not abandon the rest: the others are unrelated
			// machines, and a tick that stops at the first problem leaves a fleet
			// half-driven for as long as that problem lasts.
			slog.WarnContext(ctx, "claim onboards: could not claim machine",
				"organization_id", orgID, "hypervisor_id", hv.ID, "error", err)
			continue
		}
		if !ok {
			// Lost the race, or the rule said no at the instant of the write. Not
			// an error and not worth logging: a busy fleet loses races routinely.
			continue
		}

		if err := insert(ctx, OnboardHypervisorArgs{
			OrganizationID: orgID.String(),
			Address:        hv.Address,
			// Deliberately empty when the machine record has none. The stored column
			// holds a DIGEST and this job wants the raw value, so there is nothing to
			// convert; the job reads /etc/machine-id off the host instead, which is
			// the authority on it. See ReadMachineID.
			MachineID: "",
			// The row this tick claimed, so the worker can conclude it. Without it
			// the lease simply expires and the machine is retried, which is correct
			// but slow and tells an operator nothing about why.
			HypervisorID: hv.ID.String(),
		}); err != nil {
			if errors.Is(err, river.ErrNotFound) {
				continue
			}
			slog.WarnContext(ctx, "claim onboards: claimed a machine but could not enqueue it; "+
				"its lease will expire and the next tick will retry",
				"organization_id", orgID, "hypervisor_id", hv.ID, "error", err)
			continue
		}
		claimed++
	}
	return claimed, nil
}

// claimOne flips one machine to `provisioning` and stamps a lease, and reports
// whether it moved.
//
// ONE STATEMENT, for the reason redriveOnboarding is one statement: the rule and
// the write have to be the same instant or two workers end up on one host.
// `false` means the rule said no at the moment of the write, which is the only
// moment that counts.
//
// The attempt counter increments HERE, on the claim, not on failure. Counting
// failures would never reach the cap for the case the cap exists for — a worker
// that dies before it can record anything — and that is precisely the row that
// would otherwise be retried every tick forever.
func claimOne(ctx context.Context, db *gorm.DB, id uuid.UUID) (bool, error) {
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", id).
		Where(tenant.ClaimablePredicate).
		Where(tenant.UnderAttemptCapPredicate, tenant.MaxOnboardAttempts).
		Where("address <> ''").
		Updates(map[string]any{
			"provisioning_state": tenant.OnboardProvisioning,
			// Postgres now(), not a Go time: the lease is written by a worker and
			// read by an API replica, and the database is the clock they share.
			"provisioning_lease_expires_at": gorm.Expr("now() + ?::interval",
				tenant.LeaseExtension.String()),
			"provisioning_attempt":    gorm.Expr("provisioning_attempt + 1"),
			"provisioning_started_at": gorm.Expr("now()"),
		})
	return res.RowsAffected == 1, res.Error
}

// ClaimPendingOnboardsWorker fans the tick out across every active organization.
//
// Deliberately dumb, like the capacity sync fan-out it mirrors: it claims and
// enqueues and does no SSH work of its own, so a fleet with a hundred pending
// machines does not turn a scheduled job into a long-running process holding a
// connection while it talks to a hundred hosts.
type ClaimPendingOnboardsWorker struct {
	river.WorkerDefaults[ClaimPendingOnboardsArgs]
	controlDB *gorm.DB
	tenantDB  func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error)
	insert    func(ctx context.Context, args river.JobArgs) error
}

func NewClaimPendingOnboardsWorker(
	controlDB *gorm.DB,
	tenantDB func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error),
	insert func(ctx context.Context, args river.JobArgs) error,
) *ClaimPendingOnboardsWorker {
	return &ClaimPendingOnboardsWorker{controlDB: controlDB, tenantDB: tenantDB, insert: insert}
}

func (w *ClaimPendingOnboardsWorker) Work(
	ctx context.Context, _ *river.Job[ClaimPendingOnboardsArgs],
) error {
	var orgs []control.Organization
	if err := w.controlDB.WithContext(ctx).
		Where("status = ?", "active").
		Find(&orgs).Error; err != nil {
		return fmt.Errorf("claim onboards: list organizations: %w", err)
	}

	var claimed, skipped int
	for _, org := range orgs {
		db, err := w.tenantDB(ctx, org.ID)
		if err != nil {
			// Logged rather than returned. Returning retries the whole fan-out,
			// which would re-scan every organization already handled — and every row
			// already claimed is now leased, so the retry finds nothing to do while
			// burning a tick.
			slog.WarnContext(ctx, "claim onboards: tenant database unavailable, skipping",
				"organization_id", org.ID, "error", err)
			skipped++
			continue
		}
		n, err := ClaimPendingOnboards(ctx, db, org.ID, w.insert)
		if err != nil {
			slog.WarnContext(ctx, "claim onboards: organization failed, skipping",
				"organization_id", org.ID, "error", err)
			skipped++
			continue
		}
		claimed += n
	}

	if claimed > 0 || skipped > 0 {
		slog.InfoContext(ctx, "onboarding tick complete",
			"organizations", len(orgs), "claimed", claimed, "skipped", skipped)
	}
	return nil
}
