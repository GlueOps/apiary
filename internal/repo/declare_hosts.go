package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// DeclaredResult is what one address became.
type DeclaredResult struct {
	Address string `json:"address"`
	// ID is the machine record, whether it was created now or already existed.
	ID string `json:"id"`
	// Created is false when this address was already known. Not an error:
	// declaring a rack twice is what happens when somebody re-pastes a list, and
	// refusing the whole batch over one repeat would make the operator edit the
	// list by hand — which is where a typo enters.
	Created bool `json:"created"`
	// Queued is whether this machine is now waiting for the onboarding ticker.
	// False for one that was already provisioned and left alone.
	Queued bool `json:"queued"`
	// Detail explains anything that is not a plain creation.
	Detail string `json:"detail,omitempty"`
}

// DeclareHosts records intent for a set of addresses and queues them.
//
// EACH ADDRESS IS ITS OWN TRANSACTION, deliberately. A rack declared in one
// transaction is all-or-nothing, so a single address that collides with an
// existing row would roll back the other thirty-one — and the operator would
// have to work out which one, remove it, and paste the rest again. Partial
// success with a per-address report is what an operator can act on.
//
// A machine already known is LEFT ALONE rather than reset. Its provisioning
// state is a workflow that may be in progress, and pushing it back to pending
// because somebody re-pasted a list would put a second worker onto a host
// another is already inside. Redriving one is a deliberate act with its own
// endpoint.
// ErrNoManagementSet is returned when nothing would authorise the fleet key.
//
// A precondition rather than a failure: the hosts are fine, the request is well
// formed, and there is simply nowhere for the control plane to be allowed to
// connect from.
// Worded for the operator, not the log. The handler returns this text verbatim
// as the 422 detail and the declare dialog puts it straight in a toast, so it
// carries no "declare:" prefix and names the next action rather than the
// internal condition.
var ErrNoManagementSet = errors.New(
	"No management address set has any addresses in it, so the fleet key would be " +
		"installed permitting connections from nowhere. Add at least one address to a " +
		"management set, then declare these hosts.")

func DeclareHosts(
	ctx context.Context, db *gorm.DB, hosts []tenant.DeclaredHost,
) ([]DeclaredResult, error) {
	// REFUSED HERE, not discovered by a worker minutes later.
	//
	// Onboarding installs the fleet key with a `from=` restriction derived from
	// the management sets, and refuses when that list is empty because the key
	// would be installed with a restriction permitting nothing. Without this
	// check the API answers 201 with queued=true for every host and the work
	// fails afterwards, one job per machine, in a place the operator is not
	// looking. Observed on a clean tenant: five hosts accepted, five onboarding
	// jobs retried to exhaustion, five rows marked failed.
	//
	// Derived and judged through the same DeriveFromRestriction and Safe the
	// onboarding step uses, rather than restating the predicate here — a second
	// copy of "is this list usable" drifts into refusing what onboarding would
	// have accepted, or accepting what it will not.
	var sets []tenant.FirewallSet
	if err := db.WithContext(ctx).Preload("Elements").
		Where("purpose = ?", tenant.SetPurposeManagement).Find(&sets).Error; err != nil {
		return nil, fmt.Errorf("declare: read the management sets: %w", err)
	}
	if !tenant.DeriveFromRestriction(sets).Safe() {
		return nil, ErrNoManagementSet
	}

	out := make([]DeclaredResult, 0, len(hosts))

	for _, h := range hosts {
		addr := strings.TrimSpace(h.Address)
		if addr == "" {
			continue
		}

		var existing tenant.Hypervisor
		err := db.WithContext(ctx).Where("address = ?", addr).First(&existing).Error
		switch {
		case err == nil:
			out = append(out, DeclaredResult{
				Address: addr, ID: existing.ID.String(), Created: false,
				Queued: existing.ProvisioningState == tenant.OnboardPending,
				Detail: alreadyKnown(existing),
			})
			continue
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return nil, fmt.Errorf("declare: look up %s: %w", addr, err)
		}

		hv := tenant.Hypervisor{ID: uuid.New(), Address: addr}
		if err := db.WithContext(ctx).Create(&hv).Error; err != nil {
			// Lost a race with a concurrent declaration of the same address, or hit
			// some other constraint. Reported against the address rather than
			// failing the batch.
			out = append(out, DeclaredResult{
				Address: addr, Created: false, Queued: false,
				Detail: "could not be recorded: " + err.Error(),
			})
			continue
		}

		// pending and NOT schedulable, in a second statement.
		//
		// Both columns carry a `default:` tag, and GORM omits zero values for such
		// fields on Create — so provisioning_state would take its column default of
		// 'provisioned', declaring a machine as already onboarded, and schedulable
		// would take TRUE, admitting a host nobody has touched to placement. Same
		// trap that made a cordoned hypervisor store as schedulable.
		if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Where("id = ?", hv.ID).
			Updates(map[string]any{
				"provisioning_state": tenant.OnboardPending,
				"schedulable":        false,
			}).Error; err != nil {
			return nil, fmt.Errorf("declare: queue %s: %w", addr, err)
		}

		out = append(out, DeclaredResult{
			Address: addr, ID: hv.ID.String(), Created: true, Queued: true,
		})
	}
	return out, nil
}

// alreadyKnown says what an existing row is doing, so a repeated paste explains
// itself rather than looking like a failure.
func alreadyKnown(hv tenant.Hypervisor) string {
	switch hv.ProvisioningState {
	case tenant.OnboardPending:
		return "already recorded and waiting to be claimed"
	case tenant.OnboardProvisioning:
		return "already recorded; a worker is onboarding it now"
	case tenant.OnboardFailed:
		return "already recorded, and its last onboarding failed — reprovision it to try again"
	default:
		return "already recorded and onboarded"
	}
}
