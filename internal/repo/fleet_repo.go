package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrEnrolmentTokenInvalid = errors.New("fleet: enrolment token is invalid, expired, revoked or used up")
	ErrBeeNotFound           = errors.New("fleet: no such bee")
	ErrTaskNotFound          = errors.New("fleet: no such task for this bee")
	ErrBeeBusy               = errors.New("fleet: this bee already has an outstanding task")
	ErrTaskNotClaimable      = errors.New("fleet: task is not in a claimable state")
)

type FleetRepo interface {
	EnrolBee(ctx context.Context, in EnrolBeeInput) (*EnrolBeeResult, error)
	BeeByAPIKeyID(ctx context.Context, keyID uuid.UUID) (*tenant.Bee, error)
	RecordHeartbeat(ctx context.Context, in HeartbeatInput) error
	ClaimNextTask(ctx context.Context, beeID uuid.UUID, lease time.Duration) (*tenant.FleetTask, error)
	ReportTask(ctx context.Context, in ReportTaskInput) (*tenant.FleetTask, error)
	AppendTaskLogs(ctx context.Context, beeID, taskID uuid.UUID, lines []tenant.FleetTaskLog) (int, error)
	CreateTask(ctx context.Context, task *tenant.FleetTask) error
}

type FleetRepoGorm struct{ db *gorm.DB }

func NewFleetRepoGorm(db *gorm.DB) *FleetRepoGorm { return &FleetRepoGorm{db: db} }

type EnrolBeeInput struct {
	TokenHash string

	// MachineID is the DIGEST, never the raw /etc/machine-id — see
	// tenant.MachineIDDigest. Named for what it identifies rather than for the
	// hashing, because every caller must supply the same form and there is
	// exactly one.
	MachineID string

	// Address is what the AGENT calls itself — its hostname. Kept as the weakest
	// of the three candidates in preferredAddress, because an agent knows what it
	// calls itself and not how anybody else reaches it.
	Address string

	// PeerAddress is where this enrolment request actually came from, as resolved
	// by the trusted-proxy configuration. Reachable by demonstration rather than
	// by assumption, which is why it outranks the hostname.
	PeerAddress string

	AgentVersion string

	// APIKeyID is the agent-scoped key minted for this host during enrolment.
	APIKeyID uuid.UUID

	Now time.Time
}

// EnrolBee records a MACHINE and the agent running on it in this organization.
//
// It does NOT redeem the enrolment token: that row lives in the control
// database, so no transaction can cover both. Redemption happens first, in
// EnrolmentRepoGorm.RedeemToken, and this runs only once it has succeeded. The
// order is deliberate and fails closed — a crash between the two burns a use on
// a token that admitted nobody, which is recoverable by minting another, whereas
// the reverse leaves a spent token still redeemable, which is not.
//
// TWO ROWS, ONE TRANSACTION. The durable record is the Hypervisor: identity,
// address, cluster membership, onboarding history. The Bee is only the agent on
// it — credential, heartbeat, version. Enrolment therefore creates or ADOPTS a
// machine record and then creates or repoints the agent row against it, and both
// have to land together or a machine ends up with a credential nothing can find.
//
// Idempotent on MachineID: a hypervisor that is rebuilt, or that lost its state
// file, re-enrols onto the SAME machine record with a fresh credential rather
// than accumulating a second one. Without that, a rebuild during a rotation
// leaves a ghost that is permanently offline and an operator wondering which of
// two rows is the real host.
//
// AND idempotent against the capacity sync, which is the harder half. The sync
// discovers nodes in a cluster that nobody onboarded through this system and
// creates machine records for them keyed on (hive_id, name) — it has no machine
// id, because Proxmox's /cluster/resources does not report one. An agent later
// enrolling on such a node matches nothing by machine id, so without the
// address-to-node-name fallback below it would create a second record for a
// machine already in the table. See tenant.MatchMachineByAddress for why that
// fallback refuses rather than guesses when it is ambiguous.
//
// Re-enrolment deliberately leaves hive_id and name alone. A rebuilt host is
// still in whatever cluster the control plane last confirmed, and resetting
// membership here would silently un-join a running hypervisor because its agent
// restarted. Nothing below writes either column.
// EnrolBeeResult carries the enrolled agent and the credential it displaced.
//
// ReplacedKeyID exists so the credential a re-enrolment supersedes can be
// REVOKED rather than merely forgotten. Overwriting bees.api_key_id leaves the
// old key active in the control database forever: nothing points at it, so the
// agent surface refuses it once the bee lookup misses, but it still passes
// authentication, still counts as a live agent-scoped credential for the
// organization, and outlives the host it was minted for. A decommissioned or
// rebuilt machine should not leave a valid credential behind.
//
// It has to come out of the enrolment transaction. Reading the previous key with
// a separate SELECT beforehand races two enrolments of the same host against
// each other and can revoke the key the other one just installed.
//
// Nil when nothing was replaced — a first enrolment, or a fall-through that
// minted a new agent row.
type EnrolBeeResult struct {
	Bee           *tenant.Bee
	ReplacedKeyID *uuid.UUID
}

// preferredAddress picks how the control plane should reach a machine.
//
// Order: the peer address, then what is already stored, then the hostname.
//
// THE PEER ADDRESS IS FIRST because it is the only candidate that is reachable
// by demonstration — the enrolment request arrived from it. That also keeps a
// machine that genuinely MOVED from being pinned to a stale address, which is
// what re-enrolment is for.
//
// THE HOSTNAME IS LAST, and this is the fix. The address used to be refreshed
// from it on every enrolment, on the reasoning that the agent which just spoke
// to us is the most current source. The reasoning is wrong in one specific way:
// an agent knows what it CALLS ITSELF and has no idea how anybody else reaches
// it. A Proxmox node calls itself "pve2", which resolves on the cluster network
// and nowhere else — so every departure failed with `lookup pve2: no such host`,
// retried on a backoff, and reported nothing. It is kept as a last resort for
// deployments where the hostname genuinely is the name the control plane uses,
// and it can only ever fill an address that is empty.
//
// The caveat worth stating: behind a NAT the peer address is the gateway rather
// than the host. That is not this fleet's shape — a control plane reaches
// hypervisors directly or it cannot manage them at all — and an operator who
// needs something else declares it, which the next enrolment from the same host
// then confirms rather than contradicts.
func preferredAddress(existing, peer, hostname string) string {
	if p := strings.TrimSpace(peer); p != "" {
		return p
	}
	if e := strings.TrimSpace(existing); e != "" {
		return e
	}
	return strings.TrimSpace(hostname)
}

func (r *FleetRepoGorm) EnrolBee(ctx context.Context, in EnrolBeeInput) (*EnrolBeeResult, error) {
	var out tenant.Bee
	var replaced *uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		machine, err := findOrCreateMachine(tx, in)
		if err != nil {
			return err
		}

		// The address is how the CONTROL PLANE reaches this machine, and only some
		// of what arrives at enrolment is evidence of that.
		//
		// It used to be refreshed from the agent's own hostname on every enrolment,
		// on the reasoning that the agent which just spoke to us is the most
		// current source. The reasoning is wrong in one specific way: an agent
		// knows what it calls itself and has no idea how anybody else reaches it.
		// A Proxmox node calls itself "pve2", which resolves on the cluster network
		// and nowhere else — so every departure and every reprovision failed with
		// `lookup pve2: no such host`, retried on a backoff, and reported nothing.
		//
		// So the order is: an address an operator DECLARED is authoritative and is
		// never overwritten; otherwise the peer address of this very request, which
		// is reachable by demonstration — the packet came from it; and the hostname
		// only as a last resort, for a deployment where it is genuinely the name
		// the control plane uses.
		if addr := preferredAddress(machine.Address, in.PeerAddress, in.Address); addr != machine.Address {
			if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", machine.ID).
				Updates(map[string]any{"address": addr, "updated_at": in.Now}).Error; err != nil {
				return fmt.Errorf("record the machine's address: %w", err)
			}
		}

		// Re-enrolment replaces the credential and clears any stale declared
		// unavailability: a host that has just come back and said hello is, by
		// definition, no longer away.
		if machine.BeeID != nil {
			// Read the credential we are about to displace, inside the transaction,
			// so the caller can revoke it. A key that is merely overwritten stays
			// valid in the control database with nothing pointing at it.
			var prev tenant.Bee
			switch err := tx.Select("api_key_id").First(&prev, "id = ?", *machine.BeeID).Error; {
			case err == nil:
				if prev.APIKeyID != uuid.Nil && prev.APIKeyID != in.APIKeyID {
					id := prev.APIKeyID
					replaced = &id
				}
			case errors.Is(err, gorm.ErrRecordNotFound):
				// The link points at a bee that is gone; the fall-through below mints
				// a new one and there is no credential to revoke.
			default:
				return err
			}

			res := tx.Model(&tenant.Bee{}).Where("id = ?", *machine.BeeID).Updates(map[string]any{
				"api_key_id":         in.APIKeyID,
				"agent_version":      in.AgentVersion,
				"status":             tenant.BeeStatusOnline,
				"last_seen_at":       in.Now,
				"unavailable_until":  nil,
				"unavailable_reason": "",
				"updated_at":         in.Now,
			})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				return tx.First(&out, "id = ?", *machine.BeeID).Error
			}
			// Nothing was updated, so nothing was displaced either.
			replaced = nil
			// The link pointed at a bee that is gone. Fall through and mint a new
			// agent row rather than failing the enrolment: the machine is real and
			// the host in front of us is asking to be admitted.
		}

		bee := tenant.Bee{
			APIKeyID:     in.APIKeyID,
			Status:       tenant.BeeStatusOnline,
			AgentVersion: in.AgentVersion,
			LastSeenAt:   &in.Now,
			EnrolledAt:   in.Now,
		}
		if err := tx.Create(&bee).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", machine.ID).
			Updates(map[string]any{"bee_id": bee.ID, "updated_at": in.Now}).Error; err != nil {
			return fmt.Errorf("link the agent to its machine: %w", err)
		}
		out = bee
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &EnrolBeeResult{Bee: &out, ReplacedKeyID: replaced}, nil
}

// findOrCreateMachine resolves the machine record an enrolling agent is running
// on, walking the reconciliation ladder from the design note: machine id first,
// then the node name its address implies, then a new record.
func findOrCreateMachine(tx *gorm.DB, in EnrolBeeInput) (tenant.Hypervisor, error) {
	var machine tenant.Hypervisor

	// Locked, because the rest of enrolment writes this row and a concurrent
	// re-enrolment of the same machine must serialise behind it rather than both
	// minting agents against one host.
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("machine_id = ?", in.MachineID).First(&machine).Error
	switch {
	case err == nil:
		return machine, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return machine, err
	}

	// No record carries this machine id. Before creating one, look for a machine
	// the capacity sync discovered in a cluster and never learned an identity for.
	//
	// Deliberately NOT locked. The candidate set is every machine with no machine
	// id, and holding a lock over all of them for the length of an enrolment would
	// block the capacity sync on rows this enrolment will not touch — and lock
	// them in the opposite order from the sync, which is how two loops deadlock.
	// The adoption below is conditional instead, so losing the race is detected
	// rather than prevented.
	var candidates []tenant.Hypervisor
	if err := tx.Where("machine_id IS NULL").Find(&candidates).Error; err != nil {
		return machine, fmt.Errorf("look for a discovered machine to adopt: %w", err)
	}
	if m := tenant.MatchMachineByAddress(candidates, in.Address); m != nil {
		res := tx.Model(&tenant.Hypervisor{}).
			Where("id = ? AND machine_id IS NULL", m.ID).
			Updates(map[string]any{"machine_id": in.MachineID, "updated_at": in.Now})
		if res.Error != nil {
			return machine, res.Error
		}
		if res.RowsAffected == 1 {
			return machine, tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&machine, "id = ?", m.ID).Error
		}
		// Somebody claimed it between the scan and the update. Creating a fresh
		// record is the safe outcome: a duplicate an operator can merge, rather
		// than an agent's identity written onto a machine that is now known to be
		// a different one.
	}

	// A machine nobody has seen before.
	//
	// provisioning_state is left at its zero value ON PURPOSE, which the column
	// default turns into 'provisioned'. An agent that has enrolled itself IS
	// provisioned, whatever installed it. Note that this is also the only value a
	// struct create CAN produce here — GORM omits zero-value fields carrying a
	// `default:` tag — so a create path that meant anything else would have to say
	// so with an UPDATE.
	machine = tenant.Hypervisor{
		MachineID: &in.MachineID,
		Address:   in.Address,
	}
	if err := tx.Create(&machine).Error; err != nil {
		return machine, fmt.Errorf("create the machine record: %w", err)
	}
	return machine, nil
}

func (r *FleetRepoGorm) BeeByAPIKeyID(ctx context.Context, keyID uuid.UUID) (*tenant.Bee, error) {
	if keyID == uuid.Nil {
		// A zero key id would otherwise match any row whose column was never set.
		return nil, ErrBeeNotFound
	}
	var bee tenant.Bee
	err := r.db.WithContext(ctx).Where("api_key_id = ?", keyID).First(&bee).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBeeNotFound
		}
		return nil, err
	}
	return &bee, nil
}

type HeartbeatInput struct {
	BeeID uuid.UUID
	Now   time.Time

	AgentVersion string
	// AgentDigest is the sha256 of the binary this agent is running, and what the
	// upgrade decision is made on — see tenant.Bee.AgentDigest.
	AgentDigest string

	// UnavailableUntil is the bee declaring, IN ADVANCE, that it is about to be
	// away — a reboot, a firmware flash, a kernel upgrade. Nil leaves any
	// existing declaration alone; use ClearUnavailable to withdraw one.
	UnavailableUntil  *time.Time
	UnavailableReason string
	ClearUnavailable  bool
}

func (r *FleetRepoGorm) RecordHeartbeat(ctx context.Context, in HeartbeatInput) error {
	updates := map[string]any{"last_seen_at": in.Now, "updated_at": in.Now}
	if in.AgentVersion != "" {
		updates["agent_version"] = in.AgentVersion
	}
	// Empty is not written. An older agent reports no digest, and blanking the
	// column for it would make a host that has one look like a host that does
	// not — which the upgrade decision reads as "needs upgrading".
	if in.AgentDigest != "" {
		updates["agent_digest"] = in.AgentDigest
	}
	switch {
	case in.ClearUnavailable:
		// The bee is back and says so. Withdrawing the declaration is what turns
		// a returning host back into an ordinary one.
		updates["unavailable_until"] = nil
		updates["unavailable_reason"] = ""
	case in.UnavailableUntil != nil:
		updates["unavailable_until"] = *in.UnavailableUntil
		updates["unavailable_reason"] = in.UnavailableReason
	}

	res := r.db.WithContext(ctx).Model(&tenant.Bee{}).Where("id = ?", in.BeeID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrBeeNotFound
	}
	return nil
}

// ClaimNextTask hands this bee its one outstanding task, taking a lease.
//
// Two guarantees, both enforced in SQL rather than by the caller:
//
//   - AT MOST ONE. A bee that already holds a claimed or running task with a
//     live lease gets that same task back, not a second one. Re-delivering the
//     SAME task is deliberate: a bee that crashed after claiming and before
//     acting must be able to pick up where it left off, and the task spec is
//     desired state precisely so that re-running it is safe.
//   - NO DOUBLE ISSUE. The claim is a conditional UPDATE, so two API replicas
//     racing produce one winner; the loser sees no claimable row.
//
// An expired lease is reclaimable — that is the recovery path for a bee that died
// mid-task and never came back to report.
func (r *FleetRepoGorm) ClaimNextTask(ctx context.Context, beeID uuid.UUID, lease time.Duration) (*tenant.FleetTask, error) {
	now := time.Now().UTC()
	deadline := now.Add(lease)

	var task tenant.FleetTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Oldest first, and lock it: FOR UPDATE SKIP LOCKED would let a second
		// replica hand out the NEXT task to the same bee, which is exactly the
		// invariant this method exists to hold.
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("bee_id = ? AND status IN ?", beeID,
				[]string{tenant.FleetTaskPending, tenant.FleetTaskClaimed, tenant.FleetTaskRunning}).
			Order("created_at ASC").
			First(&task).Error
		if err != nil {
			return err
		}

		// Already running under a live lease held by this same bee: extend it and
		// hand back the same task rather than resetting its state.
		if task.Status == tenant.FleetTaskRunning &&
			task.LeaseExpiresAt != nil && task.LeaseExpiresAt.After(now) {
			return tx.Model(&tenant.FleetTask{}).Where("id = ?", task.ID).
				Update("lease_expires_at", deadline).Error
		}

		updates := map[string]any{
			"status":           tenant.FleetTaskClaimed,
			"lease_expires_at": deadline,
			"claimed_at":       now,
			"attempt":          gorm.Expr("attempt + 1"),
		}
		return tx.Model(&tenant.FleetTask{}).Where("id = ?", task.ID).Updates(updates).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // nothing to do; not an error
		}
		return nil, err
	}

	var out tenant.FleetTask
	if err := r.db.WithContext(ctx).First(&out, "id = ?", task.ID).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

type ReportTaskInput struct {
	BeeID  uuid.UUID
	TaskID uuid.UUID
	Status string
	Phase  string
	Result []byte
	Error  string
	Now    time.Time
}

// ReportTask records progress or a terminal outcome.
//
// Terminal reports are IDEMPOTENT, and that is a requirement rather than a
// courtesy: a bee that finishes a task and dies before its report is
// acknowledged will report again on restart. Rejecting the second one would
// leave the task stuck against a bee that considers it finished, and no further
// task could ever be issued to that host.
//
// The bee id is a query predicate, not a trusted parameter — a bee may only
// report on its own task.
func (r *FleetRepoGorm) ReportTask(ctx context.Context, in ReportTaskInput) (*tenant.FleetTask, error) {
	var out tenant.FleetTask
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task tenant.FleetTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bee_id = ?", in.TaskID, in.BeeID).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTaskNotFound
			}
			return err
		}

		if tenant.FleetTaskIsTerminal(task.Status) {
			// Already finished. Report success to the caller so a retrying bee
			// stops retrying, and change nothing — the first terminal outcome is
			// the real one.
			out = task
			return nil
		}

		updates := map[string]any{"status": in.Status, "phase": in.Phase}
		if len(in.Result) > 0 {
			updates["result"] = in.Result
		}
		if in.Error != "" {
			updates["error"] = in.Error
		}
		if in.Status == tenant.FleetTaskRunning && task.StartedAt == nil {
			updates["started_at"] = in.Now
		}
		if tenant.FleetTaskIsTerminal(in.Status) {
			updates["finished_at"] = in.Now
			// The lease is released with the task; leaving it set would make a
			// finished task look claimed to anything reading leases.
			updates["lease_expires_at"] = nil
		}
		if err := tx.Model(&tenant.FleetTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&out, "id = ?", task.ID).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AppendTaskLogs stores streamed output, ignoring lines already stored.
//
// ON CONFLICT DO NOTHING against the (task_id, seq) unique index is what makes
// the stream correct under at-least-once delivery: a bee that could not confirm
// a batch resends it, and the resend must be a no-op rather than a duplicate.
// The count returned is how many were NEW, which is what tells an operator
// whether a bee is actually making progress or just retrying.
func (r *FleetRepoGorm) AppendTaskLogs(ctx context.Context, beeID, taskID uuid.UUID, lines []tenant.FleetTaskLog) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}

	// Ownership is checked once, here, rather than trusted from the request: a
	// bee may only write to its own task's stream.
	var n int64
	if err := r.db.WithContext(ctx).Model(&tenant.FleetTask{}).
		Where("id = ? AND bee_id = ?", taskID, beeID).Count(&n).Error; err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, ErrTaskNotFound
	}

	for i := range lines {
		lines[i].TaskID = taskID
	}
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_id"}, {Name: "seq"}},
			DoNothing: true,
		}).
		CreateInBatches(lines, 200)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// CreateTask refuses to queue work for a bee that is already busy.
//
// Enforced with a lock rather than a unique index because "at most one
// non-terminal task per bee" is a PARTIAL uniqueness constraint, which GORM's
// tags cannot express — and a hand-written partial index would be dropped by the
// next generated migration, since Atlas diffs against the models. See the same
// reasoning in InvitationRepoGorm.Create.
func (r *FleetRepoGorm) CreateTask(ctx context.Context, task *tenant.FleetTask) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var bee tenant.Bee
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", task.BeeID).First(&bee).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBeeNotFound
			}
			return err
		}

		var outstanding int64
		if err := tx.Model(&tenant.FleetTask{}).
			Where("bee_id = ? AND status NOT IN ?", task.BeeID,
				[]string{tenant.FleetTaskSucceeded, tenant.FleetTaskFailed, tenant.FleetTaskCancelled}).
			Count(&outstanding).Error; err != nil {
			return err
		}
		if outstanding > 0 {
			// Named by the MACHINE's address, not by the agent row's id: the
			// operator reading this has to go and look at a box.
			return fmt.Errorf("%w: %s", ErrBeeBusy, machineAddressForBee(tx, bee.ID))
		}
		return tx.Create(task).Error
	})
}

// machineAddressForBee names the host an agent runs on, for an error message.
//
// Best effort by construction: it is only ever used to make a refusal legible,
// so a machine record that has gone missing degrades to the agent's id rather
// than turning a clear "this host is busy" into a database error.
func machineAddressForBee(tx *gorm.DB, beeID uuid.UUID) string {
	var m tenant.Hypervisor
	if err := tx.Where("bee_id = ?", beeID).First(&m).Error; err != nil || m.Address == "" {
		return beeID.String()
	}
	return m.Address
}
