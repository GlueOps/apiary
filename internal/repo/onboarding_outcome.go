package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// onboardingErrorLimit caps what is stored in provisioning_error.
//
// The value is rendered in the UI beside the machine, and an SSH failure can
// carry a wall of remote output. Truncated rather than dropped: the first line
// is almost always the reason, and an operator reading a truncated message still
// learns more than one reading an empty column.
const onboardingErrorLimit = 1000

// RecordOnboardOutcome stamps the terminal provisioning state on a machine.
//
// ONLY IF THE LEASE IS STILL OURS. The predicate requires the row to be in
// `provisioning` with a live lease, which is exactly the state the claim left it
// in. A worker that overran its lease has already had the machine taken by a
// later tick, and letting it write the outcome anyway would stamp a stale
// verdict over a run in progress — reporting `failed` on a host another worker
// is at that moment successfully onboarding.
//
// So a false return is not an error. It means "this run no longer owns the row",
// which is information the caller logs rather than retries: retrying would not
// make the lease ours again.
func RecordOnboardOutcome(
	ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID, err error,
) (bool, error) {
	state := tenant.OnboardProvisioned
	detail := ""
	if err != nil {
		state = tenant.OnboardFailed
		detail = err.Error()
		if len(detail) > onboardingErrorLimit {
			detail = detail[:onboardingErrorLimit]
		}
	}

	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Where("provisioning_state = ?", tenant.OnboardProvisioning).
		Where("provisioning_lease_expires_at > now()").
		Updates(map[string]any{
			"provisioning_state": state,
			"provisioning_error": detail,
			// Released explicitly. A terminal row holding a lease is readable as
			// "a worker is inside this machine" by provisioningLeaseHeld, which
			// would refuse a redrive for as long as the lease had left to run —
			// on a machine that has already finished.
			"provisioning_lease_expires_at": nil,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ErrLeaseLost reports that a run finished after its lease had already been
// taken by another tick.
var ErrLeaseLost = errors.New("onboard: the provisioning lease was no longer held")
