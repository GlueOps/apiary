package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

// MaintenanceConfig is what the driver needs to know about the rest of the
// fleet's timing.
type MaintenanceConfig struct {
	// ObservationMaxAge is the upgrade gate's staleness bound.
	ObservationMaxAge time.Duration
	// OnlineGrace is how long after a heartbeat an agent still counts as online.
	OnlineGrace time.Duration
	// ExpectedDown is what each node declares before it reboots.
	ExpectedDown time.Duration
	// NodeDeadline is how long one node's turn may take, from queueing its
	// upgrade to being back, before the run stops on it.
	NodeDeadline time.Duration
}

// DefaultMaintenanceConfig fills what a caller left zero.
func (c MaintenanceConfig) withDefaults() MaintenanceConfig {
	if c.OnlineGrace <= 0 {
		c.OnlineGrace = 90 * time.Second
	}
	if c.ExpectedDown <= 0 {
		c.ExpectedDown = 15 * time.Minute
	}
	if c.NodeDeadline <= 0 {
		// The declared downtime plus room for apt-get and for the host to boot
		// and report. A deadline that merely equals the declaration reads every
		// slow boot as a failure.
		c.NodeDeadline = c.ExpectedDown + 45*time.Minute
	}
	return c
}

// AdvanceMaintenance is one tick of scheduled maintenance for one organization:
// start every run whose window has opened, then move every running run forward.
//
// Database work only, and bounded — a node's turn is a task handed to its agent,
// not something this waits on — so it runs inline in the tick rather than as a
// job per run.
func AdvanceMaintenance(ctx context.Context, db *gorm.DB, cfg MaintenanceConfig, now time.Time) error {
	cfg = cfg.withDefaults()
	var errs []error

	var schedules []tenant.MaintenanceSchedule
	if err := db.WithContext(ctx).Where("enabled").Find(&schedules).Error; err != nil {
		return fmt.Errorf("maintenance: list schedules: %w", err)
	}
	for _, s := range schedules {
		open, ok := s.WindowAt(now)
		if !ok || (s.LastWindowAt != nil && s.LastWindowAt.Equal(open)) {
			continue
		}
		if _, err := StartMaintenanceRun(ctx, db, s, open, now); err != nil &&
			!errors.Is(err, ErrMaintenanceScopeBusy) {
			errs = append(errs, fmt.Errorf("schedule %s: %w", s.ID, err))
		}
	}

	var runs []tenant.MaintenanceRun
	if err := db.WithContext(ctx).Where("status = ?", tenant.MaintenanceRunRunning).
		Find(&runs).Error; err != nil {
		return errors.Join(append(errs, fmt.Errorf("maintenance: list runs: %w", err))...)
	}
	for _, r := range runs {
		if err := advanceRun(ctx, db, cfg, r.ID, now); err != nil {
			// One run failing to advance must not hold up another scope's.
			errs = append(errs, fmt.Errorf("run %s: %w", r.ID, err))
		}
	}
	return errors.Join(errs...)
}

// ErrMaintenanceScopeBusy is a window opening while its scope already has a
// run in progress. The window is not consumed: the next tick tries again, so a
// run that finishes inside the window lets this one start.
var ErrMaintenanceScopeBusy = errors.New("this scope already has a maintenance run in progress")

// StartMaintenanceRun starts a run for a schedule's window.
//
// ONE RUN PER SCOPE, held by a transaction-scoped advisory lock on the scope
// rather than by an index: two schedules on one cluster opening at the same
// minute would otherwise each start a run, and two runs each taking "one node
// at a time" is two nodes at a time. The lock serialises the check and the
// insert, so the second waits and then sees the first.
//
// The node list is SNAPSHOTTED here, in a stable order, so what the run will
// touch is visible from its first moment. Whether each node is still eligible is
// decided again at its own turn.
func StartMaintenanceRun(
	ctx context.Context, db *gorm.DB, s tenant.MaintenanceSchedule, open, now time.Time,
) (*tenant.MaintenanceRun, error) {
	var run *tenant.MaintenanceRun
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))",
			maintenanceScopeKey(s.HiveID)).Error; err != nil {
			return err
		}
		// Re-read under the lock: another tick may have taken this window, or
		// the schedule may have been disabled or deleted, since it was listed.
		var cur tenant.MaintenanceSchedule
		if err := tx.First(&cur, "id = ?", s.ID).Error; err != nil {
			return err
		}
		if !cur.Enabled || (cur.LastWindowAt != nil && cur.LastWindowAt.Equal(open)) {
			return ErrMaintenanceScopeBusy
		}

		var active int64
		if err := scopeWhere(tx.Model(&tenant.MaintenanceRun{}), cur.HiveID).
			Where("status = ?", tenant.MaintenanceRunRunning).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return ErrMaintenanceScopeBusy
		}

		var hvs []tenant.Hypervisor
		q := tx.Model(&tenant.Hypervisor{})
		if cur.HiveID != nil {
			q = q.Where("hive_id = ?", *cur.HiveID)
		} else {
			// The standalone pool is machines that are FINISHED arriving and in no
			// cluster. One still being onboarded has no agent to hand work to.
			q = q.Where("hive_id IS NULL AND provisioning_state = ?", "provisioned")
		}
		if err := q.Order("coalesce(name, address), id").Find(&hvs).Error; err != nil {
			return err
		}

		run = &tenant.MaintenanceRun{
			ScheduleID: &cur.ID, HiveID: cur.HiveID, Standalone: cur.Standalone,
			Status: tenant.MaintenanceRunRunning, Reboot: cur.Reboot,
			WindowStart: open, WindowEnd: open.Add(cur.Duration()), StartedAt: now,
		}
		if err := tx.Create(run).Error; err != nil {
			return err
		}
		for i, hv := range hvs {
			n := tenant.MaintenanceRunNode{
				RunID: run.ID, HypervisorID: hv.ID, Name: hypervisorLabel(hv),
				Position: i, Status: tenant.MaintenanceNodePending,
			}
			if err := tx.Create(&n).Error; err != nil {
				return err
			}
			run.Nodes = append(run.Nodes, n)
		}
		updates := map[string]any{"last_window_at": open}
		// A one-off window has happened once it has started. Disabled rather
		// than deleted so the record of it stays beside its runs.
		if cur.Kind == tenant.MaintenanceOnce {
			updates["enabled"] = false
		}
		return tx.Model(&tenant.MaintenanceSchedule{}).Where("id = ?", cur.ID).
			Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	recordSystemAudit(ctx, db, run.ID, tenant.AuditMaintenanceStart, run.HiveID, map[string]any{
		"run_id": run.ID.String(), "schedule_id": s.ID.String(), "nodes": len(run.Nodes),
	})
	return run, nil
}

// advanceRun moves one run forward by as much as can be decided now.
//
// Under a row lock on the run, so a stop requested through the API and a tick
// cannot interleave half-way through a node's transition.
func advanceRun(ctx context.Context, db *gorm.DB, cfg MaintenanceConfig, runID uuid.UUID, now time.Time) error {
	d := &runDriver{db: db, cfg: cfg, now: now}
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d.tx = tx
		var run tenant.MaintenanceRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&run, "id = ?", runID).Error; err != nil {
			return err
		}
		if tenant.MaintenanceRunIsTerminal(run.Status) {
			return nil
		}
		d.run = &run
		var nodes []tenant.MaintenanceRunNode
		if err := tx.Where("run_id = ?", run.ID).Order("position").Find(&nodes).Error; err != nil {
			return err
		}
		if run.HiveID != nil {
			d.hive = &tenant.Hive{}
			if err := tx.First(d.hive, "id = ?", *run.HiveID).Error; err != nil {
				return err
			}
		}

		for i := range nodes {
			if nodes[i].Status != tenant.MaintenanceNodeUpgrading {
				continue
			}
			// ONE AT A TIME: while a node is in flight nothing else is started,
			// whatever the window or a stop request says.
			done, err := d.checkInFlight(ctx, &nodes[i])
			if err != nil || !done {
				return err
			}
			if nodes[i].Status == tenant.MaintenanceNodeFailed {
				return d.finish(nodes, tenant.MaintenanceRunFailed,
					fmt.Sprintf("%s did not complete: %s", nodes[i].Name, nodes[i].Reason),
					"the run stopped after "+nodes[i].Name+" failed")
			}
		}

		for i := range nodes {
			if nodes[i].Status != tenant.MaintenanceNodePending {
				continue
			}
			switch {
			case run.StopRequested:
				return d.finish(nodes, tenant.MaintenanceRunStopped, "stopped by an operator",
					"the run was stopped before its turn")
			case !now.Before(run.WindowEnd):
				// A node already in flight is allowed to finish past the window —
				// a reboot cannot be taken back — but no new one starts.
				return d.finish(nodes, tenant.MaintenanceRunExpired,
					"the window closed with nodes still to do",
					"the window closed before its turn")
			}
			started, err := d.startNode(ctx, &nodes[i])
			if err != nil {
				return err
			}
			if nodes[i].Status == tenant.MaintenanceNodeFailed {
				return d.finish(nodes, tenant.MaintenanceRunFailed,
					fmt.Sprintf("%s could not be upgraded: %s", nodes[i].Name, nodes[i].Reason),
					"the run stopped after "+nodes[i].Name+" failed")
			}
			if started {
				return nil
			}
			// Skipped: move straight on to the next node in this same tick.
		}
		return d.finish(nodes, tenant.MaintenanceRunSucceeded, "", "")
	})
	if err != nil {
		return err
	}
	// Written after the commit, so the trail never records a transition that
	// was rolled back.
	for _, a := range d.audits {
		a(ctx)
	}
	return nil
}

type runDriver struct {
	// db is the connection the audit trail is written through once tx has
	// committed; tx is the run's own transaction.
	db     *gorm.DB
	tx     *gorm.DB
	cfg    MaintenanceConfig
	run    *tenant.MaintenanceRun
	hive   *tenant.Hive
	now    time.Time
	audits []func(context.Context)
}

func (d *runDriver) saveNode(n *tenant.MaintenanceRunNode) error {
	return d.tx.Model(&tenant.MaintenanceRunNode{}).Where("id = ?", n.ID).Updates(map[string]any{
		"status": n.Status, "reason": n.Reason, "task_id": n.TaskID,
		"cordoned_at": n.CordonedAt, "restored": n.Restored,
		"started_at": n.StartedAt, "deadline": n.Deadline, "finished_at": n.FinishedAt,
	}).Error
}

// finish ends the run, marking every node that never had its turn skipped.
func (d *runDriver) finish(nodes []tenant.MaintenanceRunNode, status, why, pendingReason string) error {
	for i := range nodes {
		if nodes[i].Status != tenant.MaintenanceNodePending {
			continue
		}
		nodes[i].Status = tenant.MaintenanceNodeSkipped
		nodes[i].Reason = pendingReason
		if err := d.saveNode(&nodes[i]); err != nil {
			return err
		}
	}
	now := d.now
	runID, hiveID := d.run.ID, d.run.HiveID
	d.audits = append(d.audits, func(ctx context.Context) {
		recordSystemAudit(ctx, d.db, runID, tenant.AuditMaintenanceFinish, hiveID,
			map[string]any{"run_id": runID.String(), "status": status, "error": why})
	})
	return d.tx.Model(&tenant.MaintenanceRun{}).Where("id = ?", d.run.ID).Updates(map[string]any{
		"status": status, "error": why, "finished_at": now,
	}).Error
}

// startNode decides whether a node may have its turn and, if so, cordons it and
// queues its upgrade. It reports whether an upgrade was queued; a node that was
// skipped or failed is saved with its reason.
func (d *runDriver) startNode(ctx context.Context, n *tenant.MaintenanceRunNode) (bool, error) {
	skip := func(reason string) (bool, error) {
		n.Status, n.Reason, n.FinishedAt = tenant.MaintenanceNodeSkipped, reason, &d.now
		return false, d.saveNode(n)
	}

	var hv tenant.Hypervisor
	if err := d.tx.First(&hv, "id = ?", n.HypervisorID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return skip("the machine record no longer exists")
		}
		return false, err
	}
	if reason := d.ineligible(ctx, hv); reason != "" {
		return skip(reason)
	}

	spec, _ := json.Marshal(map[string]any{
		// The schedule is the grant. See the package comment in
		// tenant/maintenance.go: this is the ONE condition of the gate lifted.
		"allow_guest_downtime":  true,
		"reboot":                d.run.Reboot,
		"expected_down_seconds": int(d.cfg.ExpectedDown / time.Second),
	})

	// Cordon, gate and queue in one SAVEPOINT, so a refusal or a busy agent
	// leaves the machine exactly as it was found.
	var queued *tenant.FleetTask
	var refusals []tenant.DepartureRefusal
	cordonedAt := n.CordonedAt
	err := d.tx.Transaction(func(tx *gorm.DB) error {
		if hv.Schedulable {
			if err := tx.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
				Update("schedulable", false).Error; err != nil {
				return err
			}
			cordonedAt = &d.now
		} else {
			// Already cordoned. Ours to restore only if a failed run before this
			// one cordoned it and never handed it back — the retry inherits that
			// obligation. Anybody else's cordon is left alone.
			//
			// Only this machine's LATEST turn counts. An older failure followed by
			// a success that handed the cordon back has nothing left to inherit.
			var prev tenant.MaintenanceRunNode
			err := tx.Where("hypervisor_id = ? AND run_id <> ? AND started_at IS NOT NULL",
				hv.ID, d.run.ID).Order("started_at DESC").Limit(1).Find(&prev).Error
			if err != nil {
				return err
			}
			if prev.ID != uuid.Nil && prev.Status == tenant.MaintenanceNodeFailed &&
				prev.CordonedAt != nil && !prev.Restored {
				cordonedAt = prev.CordonedAt
			}
		}

		var gerr error
		refusals, gerr = repo.CheckUpgradeForBee(ctx, tx, *hv.BeeID, spec,
			d.cfg.ObservationMaxAge, d.now)
		if gerr != nil {
			return gerr
		}
		if len(refusals) > 0 {
			return errRolledBack
		}
		task := tenant.FleetTask{
			BeeID: *hv.BeeID, Kind: repo.KindDistUpgrade, Spec: datatypes.JSON(spec),
			Status: tenant.FleetTaskPending,
		}
		if err := repo.NewFleetRepoGorm(tx).CreateTask(ctx, &task); err != nil {
			return err
		}
		queued = &task
		return nil
	})
	switch {
	case errors.Is(err, errRolledBack):
		parts := make([]string, 0, len(refusals))
		for _, r := range refusals {
			parts = append(parts, r.Summary)
		}
		n.Status, n.FinishedAt = tenant.MaintenanceNodeFailed, &d.now
		n.Reason = "the upgrade gate refused: " + strings.Join(parts, "; ")
		return false, d.saveNode(n)
	case errors.Is(err, repo.ErrBeeBusy):
		return skip("its agent is busy with another task")
	case err != nil:
		return false, err
	}

	deadline := d.now.Add(d.cfg.NodeDeadline)
	n.Status, n.Reason = tenant.MaintenanceNodeUpgrading, ""
	n.TaskID, n.StartedAt, n.Deadline, n.CordonedAt = &queued.ID, &d.now, &deadline, cordonedAt
	if err := d.saveNode(n); err != nil {
		return false, err
	}
	runID, hvID, hiveID := d.run.ID, hv.ID, d.run.HiveID
	wasSchedulable := hv.Schedulable
	d.audits = append(d.audits, func(ctx context.Context) {
		if wasSchedulable {
			recordSystemAudit(ctx, d.db, runID, tenant.AuditHypervisorCordon, nil,
				map[string]any{"run_id": runID.String(), "hypervisor_id": hvID.String(),
					"schedulable": false})
		}
		recordSystemAudit(ctx, d.db, runID, tenant.AuditMaintenanceNode, hiveID,
			map[string]any{"run_id": runID.String(), "hypervisor_id": hvID.String(),
				"task_id": queued.ID.String()})
	})
	return true, nil
}

var errRolledBack = errors.New("rolled back")

// ineligible says why a node may not take its turn, or "" when it may.
//
// DETACHED MACHINES ARE SKIPPED, NOT FAILED. A machine with no agent, an
// offline agent, or one part-way through leaving cannot be handed an upgrade,
// and failing the run over it would block every other node's maintenance for as
// long as that one machine stays broken. The reason is on the node, every run,
// so it never goes quiet.
func (d *runDriver) ineligible(_ context.Context, hv tenant.Hypervisor) string {
	if d.hive != nil {
		if hv.HiveID == nil || *hv.HiveID != d.hive.ID {
			return "it has left this cluster since the run started"
		}
	} else if hv.HiveID != nil {
		return "it has joined a cluster since the run started"
	}
	if hv.DepartingAt != nil {
		return "a departure from its cluster is under way"
	}
	if hv.BeeID == nil {
		return "no agent is attached to this machine"
	}
	var bee tenant.Bee
	if err := d.tx.First(&bee, "id = ?", *hv.BeeID).Error; err != nil {
		return "no agent is attached to this machine"
	}
	switch bee.LivenessAt(d.now, d.cfg.OnlineGrace) {
	case tenant.BeeStatusOnline:
	case tenant.BeeStatusUnavailable:
		return "its agent has declared itself unavailable: " + bee.UnavailableReason
	default:
		if bee.LastSeenAt == nil {
			return "its agent has never reported"
		}
		return "its agent has been offline since " + bee.LastSeenAt.UTC().Format(time.RFC3339)
	}
	if d.hive != nil && !strings.EqualFold(bee.ObservedCluster, d.hive.Name) {
		return "its agent does not report being in this cluster"
	}
	return ""
}

// checkInFlight decides whether the node in flight has finished. It reports
// done once the node has succeeded or failed.
//
// "BACK" IS NOT "THE TASK SUCCEEDED". The task succeeding means the agent ran
// its post-reboot verification; the node is back when its agent is also
// heartbeating with no declared absence and — in a cluster — reports the
// cluster quorate. Only then does the next node go, because the next node's
// reboot is safe only if this one is counting again.
func (d *runDriver) checkInFlight(_ context.Context, n *tenant.MaintenanceRunNode) (bool, error) {
	fail := func(reason string) (bool, error) {
		n.Status, n.Reason, n.FinishedAt = tenant.MaintenanceNodeFailed, reason, &d.now
		// Left cordoned on purpose: a node that did not come back must not take
		// placements until somebody has looked at it.
		return true, d.saveNode(n)
	}
	late := n.Deadline != nil && d.now.After(*n.Deadline)

	var task tenant.FleetTask
	if n.TaskID == nil {
		return fail("its upgrade task was never recorded")
	}
	if err := d.tx.First(&task, "id = ?", *n.TaskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail("its upgrade task no longer exists")
		}
		return false, err
	}

	switch task.Status {
	case tenant.FleetTaskFailed, tenant.FleetTaskCancelled:
		why := task.Error
		if why == "" {
			why = "the task was " + task.Status
		}
		return fail("the upgrade failed: " + why)
	case tenant.FleetTaskSucceeded:
	default:
		if !late {
			return false, nil
		}
		// An agent that never collected its task. Withdrawn so it cannot pick up
		// a reboot hours later, when it returns in the middle of something else.
		// One already claimed is left — it may be mid-reboot.
		if task.Status == tenant.FleetTaskPending {
			if err := d.tx.Model(&tenant.FleetTask{}).
				Where("id = ? AND status = ?", task.ID, tenant.FleetTaskPending).
				Updates(map[string]any{"status": tenant.FleetTaskCancelled, "finished_at": d.now,
					"error": "withdrawn: the maintenance window's deadline for this node passed"}).
				Error; err != nil {
				return false, err
			}
		}
		return fail(fmt.Sprintf("its upgrade did not finish within %s (the task is %s)",
			d.cfg.NodeDeadline, task.Status))
	}

	var hv tenant.Hypervisor
	var bee tenant.Bee
	if err := d.tx.First(&hv, "id = ?", n.HypervisorID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail("the machine record disappeared during its upgrade")
		}
		return false, err
	}
	if hv.BeeID == nil || d.tx.First(&bee, "id = ?", *hv.BeeID).Error != nil {
		return fail("its agent was detached during its upgrade")
	}
	notBack := ""
	switch {
	case bee.LivenessAt(d.now, d.cfg.OnlineGrace) != tenant.BeeStatusOnline:
		notBack = "its agent is not heartbeating"
	case d.hive != nil && !bee.ObservedQuorate:
		notBack = "it does not report the cluster quorate"
	}
	if notBack != "" {
		if !late {
			return false, nil
		}
		return fail(fmt.Sprintf("it upgraded but was not back within %s: %s",
			d.cfg.NodeDeadline, notBack))
	}

	n.Status, n.FinishedAt = tenant.MaintenanceNodeSucceeded, &d.now
	if err := d.restoreCordon(n, hv); err != nil {
		return false, err
	}
	return true, d.saveNode(n)
}

// restoreCordon hands the cordon back, when it was this run's to hand back.
//
// A node cordoned BY HAND after the run cordoned it stays cordoned: somebody
// decided something about that machine during the window, and the run cannot
// tell what. Read from the audit trail, which records every cordon and uncordon
// with who did it; the run's own are written as the system and do not count.
func (d *runDriver) restoreCordon(n *tenant.MaintenanceRunNode, hv tenant.Hypervisor) error {
	if n.CordonedAt == nil || n.Restored {
		return nil
	}
	if hv.Schedulable {
		// Somebody already uncordoned it. Nothing to hand back.
		n.Restored = true
		return nil
	}
	var touched int64
	if err := d.tx.Model(&tenant.AuditEvent{}).
		Where("target_kind = ? AND target_id = ? AND action IN ? AND principal_kind <> ? "+
			"AND created_at > ?", tenant.AuditTargetHypervisor, hv.ID,
			[]string{tenant.AuditHypervisorCordon, tenant.AuditHypervisorUncordon},
			tenant.PrincipalKindSystem, *n.CordonedAt).
		Count(&touched).Error; err != nil {
		return err
	}
	if touched > 0 {
		n.Reason = "left cordoned: it was cordoned by hand during maintenance"
		return nil
	}
	if err := d.tx.Model(&tenant.Hypervisor{}).Where("id = ? AND NOT schedulable", hv.ID).
		Update("schedulable", true).Error; err != nil {
		return err
	}
	n.Restored = true
	runID, hvID := d.run.ID, hv.ID
	d.audits = append(d.audits, func(ctx context.Context) {
		recordSystemAudit(ctx, d.db, runID, tenant.AuditHypervisorUncordon, nil,
			map[string]any{"run_id": runID.String(), "hypervisor_id": hvID.String(),
				"schedulable": true})
	})
	return nil
}

// recordSystemAudit writes an action no person took. The principal is the run
// itself, so "which run cordoned this" is one lookup.
func recordSystemAudit(ctx context.Context, db *gorm.DB, runID uuid.UUID, action string,
	hiveID *uuid.UUID, meta map[string]any) {
	target, targetID := tenant.AuditTargetFleet, (*uuid.UUID)(nil)
	if hiveID != nil {
		target, targetID = tenant.AuditTargetHive, hiveID
	}
	if hv, ok := meta["hypervisor_id"].(string); ok &&
		(action == tenant.AuditHypervisorCordon || action == tenant.AuditHypervisorUncordon) {
		if id, err := uuid.Parse(hv); err == nil {
			target, targetID = tenant.AuditTargetHypervisor, &id
		}
	}
	ev := &tenant.AuditEvent{
		PrincipalID: runID, PrincipalKind: tenant.PrincipalKindSystem,
		Action: action, TargetKind: target, TargetID: targetID,
		Outcome: tenant.AuditOutcomeOK, Metadata: repo.AuditMetadata(meta),
	}
	if err := repo.RecordAudit(ctx, db.Session(&gorm.Session{NewDB: true}), ev); err != nil {
		slog.ErrorContext(ctx, "a maintenance action was NOT written to the audit trail",
			"action", action, "run_id", runID, "error", err)
	}
}

func scopeWhere(q *gorm.DB, hiveID *uuid.UUID) *gorm.DB {
	if hiveID == nil {
		return q.Where("hive_id IS NULL AND standalone")
	}
	return q.Where("hive_id = ?", *hiveID)
}

func maintenanceScopeKey(hiveID *uuid.UUID) string {
	if hiveID == nil {
		return "maintenance:standalone"
	}
	return "maintenance:" + hiveID.String()
}

func hypervisorLabel(hv tenant.Hypervisor) string {
	if hv.Name != nil && *hv.Name != "" {
		return *hv.Name
	}
	if hv.Address != "" {
		return hv.Address
	}
	return hv.ID.String()
}
