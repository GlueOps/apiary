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

// KindAgentUpgrade is the task that replaces an agent's own binary.
const KindAgentUpgrade = "fleet.host.agent-upgrade"

// UpgradeCandidate is one agent considered for an upgrade.
type UpgradeCandidate struct {
	BeeID   string `json:"bee_id"`
	Host    string `json:"host,omitempty"`
	Version string `json:"agent_version,omitempty"`
	Queued  bool   `json:"queued"`
	// Skipped says why this agent was left alone, and is empty when it was
	// queued. Reported per agent rather than as a count, because "14 of 20
	// queued" leaves the operator to work out which six and why.
	Skipped string `json:"skipped,omitempty"`
}

// QueueAgentUpgrades enqueues an upgrade task for every eligible agent.
//
// ROLLING BY CONSTRUCTION, not by a knob. Each task lands on its own host and
// the fleet already allows a bee at most one outstanding task, so a fleet-wide
// upgrade is one task per host and no host is doing two things at once. The
// hosts do proceed in parallel with each other, which is safe here for a reason
// specific to this task: an agent is not in the data path of anything it
// manages, so an agent that fails to come back costs observability and control
// of that host, not its guests.
//
// Three exclusions, each because queueing would be worse than skipping:
//
//   - an agent ALREADY on the target version has nothing to do, and reinstalling
//     it would restart every agent in the fleet to reach the state they were
//     already in;
//   - an agent that is OFFLINE cannot be handed a task it will act on, and one
//     queued now is one it collects whenever it returns — which may be during
//     the next incident, in the middle of it;
//   - an agent already HOLDING A TASK gets a 409 from the task path anyway, and
//     reporting that as a per-agent skip is more use than a failed batch.
func QueueAgentUpgrades(
	ctx context.Context,
	db *gorm.DB,
	hiveID *uuid.UUID,
	targetVersion string,
	// targetDigests are the sha256s of every binary this control plane serves.
	// An agent whose own digest is among them is running one of them and has
	// nothing to do, whatever any version says.
	targetDigests map[string]bool,
	byUser uuid.UUID,
	force bool,
	now time.Time,
	offlineAfter time.Duration,
) ([]UpgradeCandidate, error) {
	var bees []tenant.Bee
	q := db.WithContext(ctx).Model(&tenant.Bee{})
	if hiveID != nil {
		// A bee carries no hive — that column lives on the machine record, because
		// two columns answering "which cluster is this in" diverged during a
		// migration. So the fleet is reached through the hypervisor.
		q = q.Where("id IN (?)", db.Model(&tenant.Hypervisor{}).
			Select("bee_id").Where("hive_id = ? AND bee_id IS NOT NULL", *hiveID))
	}
	if err := q.Find(&bees).Error; err != nil {
		return nil, fmt.Errorf("agent upgrade: list agents: %w", err)
	}

	out := make([]UpgradeCandidate, 0, len(bees))
	for _, b := range bees {
		c := UpgradeCandidate{BeeID: b.ID.String(), Version: b.AgentVersion}

		switch {
		// DECIDED ON THE DIGEST, not the version.
		//
		// The catalogue advertises the CONTROL PLANE's version while the binaries
		// it serves carry whatever they were stamped with, and only a release
		// stamps both from one tag. Where they disagree, a version comparison never
		// converges: the agent upgrades, restarts, reports the same version it had,
		// is judged out of date again, and upgrades again — a loop that reinstalls
		// the fleet's agents forever and reads as the upgrade being broken.
		case b.AgentDigest != "" && targetDigests[b.AgentDigest] && !force:
			c.Skipped = "already running this build"
		// An agent too old to report a digest falls back to the version. It is the
		// only signal such an agent gives, and the first upgrade replaces it with
		// one that does report a digest — so the fallback is self-clearing.
		case b.AgentDigest == "" && b.AgentVersion == targetVersion && !force:
			c.Skipped = "already running " + targetVersion
		case b.LivenessAt(now, offlineAfter) != tenant.BeeStatusOnline:
			// Deliberately not queued for later. A task left waiting is collected
			// whenever the host returns, and "whenever" is often during the next
			// incident — an agent restarting itself in the middle of one is the
			// last thing anybody wants.
			c.Skipped = "offline; upgrade it once it is back"
		default:
			task := tenant.FleetTask{
				BeeID: b.ID, Kind: KindAgentUpgrade,
				Spec:   datatypes.JSON([]byte(`{"version":` + quoteJSON(targetVersion) + `}`)),
				Status: tenant.FleetTaskPending, CreatedByID: byUser,
			}
			err := NewFleetRepoGorm(db).CreateTask(ctx, &task)
			switch {
			case err == nil:
				c.Queued = true
			case errors.Is(err, ErrBeeBusy):
				c.Skipped = "already has an outstanding task"
			default:
				// Reported against the agent rather than failing the batch: the other
				// nineteen are unaffected and the operator can retry this one.
				c.Skipped = "could not be queued: " + err.Error()
			}
		}
		out = append(out, c)
	}
	return out, nil
}

// quoteJSON renders a version as a JSON string. Small enough to do here, and it
// keeps this file free of a marshal that can fail on a value that cannot.
func quoteJSON(s string) string {
	var b []byte
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b = append(b, '\\', byte(r))
		case '\n':
			b = append(b, '\\', 'n')
		default:
			if r < 0x20 {
				continue
			}
			b = append(b, string(r)...)
		}
	}
	return string(append(b, '"'))
}
