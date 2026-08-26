package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// UpgradeGateInput is the spec fields the gate reads. Decoded leniently: a spec
// this does not understand is not a reason to refuse, and the agent validates
// its own arguments.
type UpgradeGateInput struct {
	AllowGuestDowntime bool   `json:"allow_guest_downtime"`
	Reboot             string `json:"reboot"`
}

// CheckUpgradeForBee runs the host-upgrade gate for the machine an agent is
// running on.
//
// GATED HERE, NOT ON THE AGENT. The agent used to decide alone by asking the
// Proxmox cluster API, which is why a dist-upgrade needed an API token for a job
// that is entirely local. Every fact the decision needs is already in this
// database, and deciding here means the refusal reaches the operator at the
// moment they ask rather than as a failed task minutes later.
//
// A bee with no machine record is NOT refused. That is a host mid-enrolment, and
// there is nothing to decide against — the agent still runs its own preflight
// when it has a client, so the check is not lost, only deferred.
func CheckUpgradeForBee(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, rawSpec []byte, maxAge time.Duration,
	now time.Time,
) ([]tenant.DepartureRefusal, error) {
	var spec UpgradeGateInput
	if len(rawSpec) > 0 {
		// Ignored on purpose. A malformed spec is the agent's to reject, and
		// refusing here would turn a typo into a gate refusal that names the wrong
		// problem.
		_ = json.Unmarshal(rawSpec, &spec)
	}

	var hv tenant.Hypervisor
	if err := db.WithContext(ctx).First(&hv, "bee_id = ?", beeID).Error; err != nil {
		//nolint:nilerr // A bee with no machine record has nothing to decide against.
		return nil, nil
	}

	req := tenant.UpgradeRequest{
		Hypervisor:         hv,
		Reboot:             spec.Reboot,
		AllowGuestDowntime: spec.AllowGuestDowntime,
		MaxAge:             maxAge,
	}
	if hv.LastSyncedAt != nil {
		req.ObservationAge = now.Sub(hv.LastSyncedAt.UTC())
	} else if hv.HiveID != nil {
		// Never observed, and in a cluster where it should have been. Reported as
		// maximally stale rather than as zero — an unobserved host is the case the
		// staleness rule exists for, and treating "no observation" as "fresh"
		// inverts it.
		req.ObservationAge = maxAge + time.Second
	}

	if hv.HiveID != nil {
		var nodes int64
		if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Where("hive_id = ?", *hv.HiveID).Count(&nodes).Error; err != nil {
			return nil, err
		}
		req.HiveNodeCount = int(nodes)
	}

	return tenant.CheckHostUpgrade(req), nil
}
