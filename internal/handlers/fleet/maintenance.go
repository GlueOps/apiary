package fleet

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/models/tenant"
)

// MaintenanceScheduleInput is the editable part of a schedule.
type MaintenanceScheduleInput struct {
	Name string `json:"name" maxLength:"100"`
	Kind string `json:"kind" enum:"recurring,once"`
	// Recurring only.
	Weekdays  []int  `json:"weekdays,omitempty" doc:"Days the window opens on, 0 = Sunday … 6 = Saturday, in timezone."`
	StartTime string `json:"start_time,omitempty" doc:"Local time the window opens, HH:MM 24-hour, in timezone." example:"02:00"`
	Timezone  string `json:"timezone,omitempty" doc:"IANA zone name. Required for a recurring window: 02:00 means 02:00 local in summer and in winter." example:"Europe/Helsinki"`
	// Once only.
	OnceAt *time.Time `json:"once_at,omitempty" doc:"When a one-off window opens. A time in the past still opens the window now, provided the window has not already closed."`

	DurationMinutes int    `json:"duration_minutes" minimum:"15" maximum:"720" doc:"How long the window stays open. A node already rebooting when it closes is allowed to finish; no new node starts."`
	Reboot          string `json:"reboot" enum:"if-needed,always,never" doc:"Reboot policy each node's dist-upgrade runs with."`
	Enabled         bool   `json:"enabled"`
}

// CreateMaintenanceScheduleRequest names the scope as well as the window.
type CreateMaintenanceScheduleRequest struct {
	Body struct {
		MaintenanceScheduleInput
		// EXACTLY ONE of these. A schedule with neither is refused rather than read
		// as standalone: that default would reboot every unclustered machine in
		// the organization on behalf of somebody who forgot a field.
		HiveID     string `json:"hive_id,omitempty" format:"uuid" doc:"The cluster this window maintains."`
		Standalone bool   `json:"standalone,omitempty" doc:"True maintains the standalone pool instead: every hypervisor in no cluster, resolved when each run starts."`
	}
}

type UpdateMaintenanceScheduleRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body MaintenanceScheduleInput
}

type MaintenanceScheduleIDRequest struct {
	ID string `path:"id" format:"uuid"`
}

// MaintenanceScheduleView is a schedule with its next openings, so a wrong zone
// or weekday is visible before it reboots anything.
type MaintenanceScheduleView struct {
	ID              string   `json:"id" format:"uuid"`
	HiveID          string   `json:"hive_id,omitempty" format:"uuid"`
	HiveName        string   `json:"hive_name,omitempty"`
	Standalone      bool     `json:"standalone"`
	Name            string   `json:"name"`
	Kind            string   `json:"kind" enum:"recurring,once"`
	Weekdays        []int    `json:"weekdays"`
	StartTime       string   `json:"start_time,omitempty"`
	Timezone        string   `json:"timezone,omitempty"`
	OnceAt          string   `json:"once_at,omitempty" format:"date-time"`
	DurationMinutes int      `json:"duration_minutes"`
	Reboot          string   `json:"reboot"`
	Enabled         bool     `json:"enabled"`
	OpenNow         bool     `json:"open_now" doc:"The window is open at this moment."`
	NextOpenings    []string `json:"next_openings" doc:"The next three times the window opens, UTC."`
	LastWindowAt    string   `json:"last_window_at,omitempty" format:"date-time"`
}

type MaintenanceScheduleResponse struct{ Body MaintenanceScheduleView }

type ListMaintenanceSchedulesResponse struct {
	Body struct {
		Schedules []MaintenanceScheduleView `json:"schedules"`
	}
}

type MaintenanceRunNodeView struct {
	HypervisorID string `json:"hypervisor_id" format:"uuid"`
	Name         string `json:"name"`
	Status       string `json:"status" enum:"pending,upgrading,succeeded,failed,skipped"`
	Reason       string `json:"reason,omitempty" doc:"Why it was skipped or failed, or why it was left cordoned."`
	TaskID       string `json:"task_id,omitempty" format:"uuid"`
	LeftCordoned bool   `json:"left_cordoned" doc:"The run cordoned this node and has not handed the cordon back: it failed, or somebody cordoned it by hand meanwhile."`
	StartedAt    string `json:"started_at,omitempty" format:"date-time"`
	Deadline     string `json:"deadline,omitempty" format:"date-time"`
	FinishedAt   string `json:"finished_at,omitempty" format:"date-time"`
}

type MaintenanceRunView struct {
	ID            string                   `json:"id" format:"uuid"`
	ScheduleID    string                   `json:"schedule_id,omitempty" format:"uuid"`
	ScheduleName  string                   `json:"schedule_name,omitempty"`
	HiveID        string                   `json:"hive_id,omitempty" format:"uuid"`
	Standalone    bool                     `json:"standalone"`
	Status        string                   `json:"status" enum:"running,succeeded,failed,stopped,expired"`
	Error         string                   `json:"error,omitempty"`
	StopRequested bool                     `json:"stop_requested"`
	Reboot        string                   `json:"reboot"`
	WindowStart   string                   `json:"window_start" format:"date-time"`
	WindowEnd     string                   `json:"window_end" format:"date-time"`
	StartedAt     string                   `json:"started_at" format:"date-time"`
	FinishedAt    string                   `json:"finished_at,omitempty" format:"date-time"`
	Nodes         []MaintenanceRunNodeView `json:"nodes"`
}

type ListMaintenanceRunsRequest struct {
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

type ListMaintenanceRunsResponse struct {
	Body struct {
		Runs []MaintenanceRunView `json:"runs"`
	}
}

type MaintenanceRunIDRequest struct {
	ID string `path:"id" format:"uuid"`
}

type MaintenanceRunResponse struct{ Body MaintenanceRunView }

func registerMaintenance(
	hapi huma.API,
	tenantDB func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error),
	member, admin func(ctx context.Context) (*apimiddleware.Principal, error),
) {
	// open resolves the caller and their tenant database in one step. Every
	// query below runs against that database, which IS the organization — a
	// schedule or run id from another organization is simply not found.
	open := func(ctx context.Context, role func(context.Context) (*apimiddleware.Principal, error)) (
		*apimiddleware.Principal, *gorm.DB, error) {
		p, err := role(ctx)
		if err != nil {
			return nil, nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, nil, err
		}
		return p, db.WithContext(ctx), nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-maintenance-schedules-list",
		Method:      http.MethodGet,
		Path:        "/fleet/maintenance/schedules",
		Summary:     "List maintenance windows",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *struct{}) (*ListMaintenanceSchedulesResponse, error) {
		_, db, err := open(ctx, member)
		if err != nil {
			return nil, err
		}
		var rows []tenant.MaintenanceSchedule
		if err := db.Order("standalone, name, created_at").Find(&rows).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		names, err := hiveNames(db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListMaintenanceSchedulesResponse{}
		out.Body.Schedules = make([]MaintenanceScheduleView, 0, len(rows))
		now := time.Now().UTC()
		for _, s := range rows {
			out.Body.Schedules = append(out.Body.Schedules, scheduleView(s, names, now))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-maintenance-schedules-create",
		Method:        http.MethodPost,
		Path:          "/fleet/maintenance/schedules",
		Summary:       "Schedule a maintenance window",
		DefaultStatus: http.StatusCreated,
		Description: "In each window the control plane dist-upgrades the scope's hypervisors ONE AT " +
			"A TIME: it cordons the node, queues the upgrade, waits for the node to come back " +
			"(agent heartbeating, and in a cluster, quorate), hands the cordon back, then moves on. " +
			"The first node that fails or does not come back in time stops the run and stays " +
			"cordoned.\n\n" +
			"A window GRANTS GUEST DOWNTIME: guests on a node reboot with it. It lifts no other " +
			"condition of the upgrade gate — a stale observation or a reboot that would cost the " +
			"cluster quorum still refuses, at that node's turn.\n\n" +
			"Machines with no agent, an offline agent, or a departure under way are skipped with " +
			"the reason, and do not stop the run.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateMaintenanceScheduleRequest) (*MaintenanceScheduleResponse, error) {
		p, db, err := open(ctx, admin)
		if err != nil {
			return nil, err
		}
		s := tenant.MaintenanceSchedule{Standalone: in.Body.Standalone, CreatedByID: p.UserID}
		if in.Body.HiveID != "" {
			id, err := uuid.Parse(in.Body.HiveID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid hive_id")
			}
			if err := db.First(&tenant.Hive{}, "id = ?", id).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, huma.Error404NotFound("no such cluster")
				}
				return nil, mapErr(ctx, err)
			}
			s.HiveID = &id
		}
		now := time.Now().UTC()
		if err := applyScheduleInput(&s, in.Body.MaintenanceScheduleInput, now); err != nil {
			return nil, err
		}
		if err := db.Create(&s).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		auditOK(ctx, db, p, tenant.AuditMaintenanceScheduleCreate, scheduleTargetKind(s),
			scheduleTargetID(s), scheduleAuditMeta(s))
		names, err := hiveNames(db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &MaintenanceScheduleResponse{Body: scheduleView(s, names, now)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-maintenance-schedules-update",
		Method:      http.MethodPut,
		Path:        "/fleet/maintenance/schedules/{id}",
		Summary:     "Change a maintenance window",
		Description: "Replaces the window. The scope cannot be changed — delete and recreate. A run " +
			"already in progress keeps the window it started with.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateMaintenanceScheduleRequest) (*MaintenanceScheduleResponse, error) {
		p, db, err := open(ctx, admin)
		if err != nil {
			return nil, err
		}
		s, err := findSchedule(ctx, db, in.ID)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		if err := applyScheduleInput(&s, in.Body, now); err != nil {
			return nil, err
		}
		// Select every editable column by name: Save would do, but naming them
		// keeps the scope columns out of any update this handler issues.
		if err := db.Model(&tenant.MaintenanceSchedule{}).Where("id = ?", s.ID).
			Select("name", "kind", "weekdays", "start_time", "timezone", "once_at",
				"duration_minutes", "reboot", "enabled").
			Updates(&s).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		auditOK(ctx, db, p, tenant.AuditMaintenanceScheduleUpdate, scheduleTargetKind(s),
			scheduleTargetID(s), scheduleAuditMeta(s))
		names, err := hiveNames(db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &MaintenanceScheduleResponse{Body: scheduleView(s, names, now)}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-maintenance-schedules-delete",
		Method:        http.MethodDelete,
		Path:          "/fleet/maintenance/schedules/{id}",
		Summary:       "Delete a maintenance window",
		Description:   "A run already in progress is not stopped by this — stop it separately. Past runs are kept.",
		DefaultStatus: http.StatusNoContent,
		Tags:          []string{"Fleet"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *MaintenanceScheduleIDRequest) (*struct{}, error) {
		p, db, err := open(ctx, admin)
		if err != nil {
			return nil, err
		}
		s, err := findSchedule(ctx, db, in.ID)
		if err != nil {
			return nil, err
		}
		if err := db.Delete(&tenant.MaintenanceSchedule{}, "id = ?", s.ID).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		auditOK(ctx, db, p, tenant.AuditMaintenanceScheduleDelete, scheduleTargetKind(s),
			scheduleTargetID(s), scheduleAuditMeta(s))
		return nil, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-maintenance-runs-list",
		Method:      http.MethodGet,
		Path:        "/fleet/maintenance/runs",
		Summary:     "List recent maintenance runs",
		Description: "Newest first, with every node's turn: what happened, and why a node was skipped, failed or left cordoned.",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ListMaintenanceRunsRequest) (*ListMaintenanceRunsResponse, error) {
		_, db, err := open(ctx, member)
		if err != nil {
			return nil, err
		}
		limit := in.Limit
		if limit <= 0 || limit > 100 {
			limit = 20
		}
		var runs []tenant.MaintenanceRun
		if err := db.Preload("Nodes", func(q *gorm.DB) *gorm.DB { return q.Order("position") }).
			Order("started_at DESC").Limit(limit).Find(&runs).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		names, err := scheduleNames(db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListMaintenanceRunsResponse{}
		out.Body.Runs = make([]MaintenanceRunView, 0, len(runs))
		for _, r := range runs {
			out.Body.Runs = append(out.Body.Runs, runView(r, names))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-maintenance-runs-stop",
		Method:      http.MethodPost,
		Path:        "/fleet/maintenance/runs/{id}/stop",
		Summary:     "Stop a maintenance run",
		Description: "Stops the run AFTER the node in flight. That node's upgrade is not aborted — a " +
			"host part-way through a reboot is not something the control plane can take back — " +
			"and it is still watched until it is back or its deadline passes. No further node starts.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *MaintenanceRunIDRequest) (*MaintenanceRunResponse, error) {
		p, db, err := open(ctx, admin)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid run id")
		}
		res := db.Model(&tenant.MaintenanceRun{}).
			Where("id = ? AND status = ?", id, tenant.MaintenanceRunRunning).
			Update("stop_requested", true)
		if res.Error != nil {
			return nil, mapErr(ctx, res.Error)
		}
		var run tenant.MaintenanceRun
		if err := db.Preload("Nodes", func(q *gorm.DB) *gorm.DB { return q.Order("position") }).
			First(&run, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, huma.Error404NotFound("no such maintenance run")
			}
			return nil, mapErr(ctx, err)
		}
		if res.RowsAffected == 0 {
			return nil, huma.Error409Conflict("this run has already finished (" + run.Status + ")")
		}
		auditOK(ctx, db, p, tenant.AuditMaintenanceRunStop, scheduleTargetKind(
			tenant.MaintenanceSchedule{HiveID: run.HiveID}), run.HiveID,
			map[string]any{"run_id": run.ID.String()})
		names, err := scheduleNames(db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &MaintenanceRunResponse{Body: runView(run, names)}, nil
	})
}

// applyScheduleInput copies the editable fields and validates the result.
func applyScheduleInput(s *tenant.MaintenanceSchedule, in MaintenanceScheduleInput, now time.Time) error {
	s.Name = strings.TrimSpace(in.Name)
	s.Kind = in.Kind
	s.DurationMinutes = in.DurationMinutes
	s.Reboot = in.Reboot
	if s.Reboot == "" {
		s.Reboot = tenant.RebootIfNeeded
	}
	s.Enabled = in.Enabled
	s.Weekdays, s.StartTime, s.Timezone, s.OnceAt = 0, "", "", nil
	switch in.Kind {
	case tenant.MaintenanceRecurring:
		for _, d := range in.Weekdays {
			if d < 0 || d > 6 {
				return huma.Error422UnprocessableEntity(tenant.ErrMaintenanceWeekdays.Error())
			}
			s.Weekdays |= 1 << d
		}
		s.StartTime, s.Timezone = in.StartTime, in.Timezone
	case tenant.MaintenanceOnce:
		if in.OnceAt != nil {
			at := in.OnceAt.UTC().Truncate(time.Minute)
			s.OnceAt = &at
		}
	}
	if err := tenant.ValidateMaintenanceSchedule(*s); err != nil {
		return huma.Error422UnprocessableEntity(err.Error())
	}
	if s.Kind == tenant.MaintenanceOnce && s.Enabled && !s.OnceAt.Add(s.Duration()).After(now) {
		return huma.Error422UnprocessableEntity("that window has already closed")
	}
	return nil
}

func findSchedule(ctx context.Context, db *gorm.DB, raw string) (tenant.MaintenanceSchedule, error) {
	var s tenant.MaintenanceSchedule
	id, err := uuid.Parse(raw)
	if err != nil {
		return s, huma.Error422UnprocessableEntity("invalid schedule id")
	}
	if err := db.First(&s, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s, huma.Error404NotFound("no such maintenance window")
		}
		return s, mapErr(ctx, err)
	}
	return s, nil
}

func hiveNames(db *gorm.DB) (map[uuid.UUID]string, error) {
	var hives []tenant.Hive
	if err := db.Select("id", "name").Find(&hives).Error; err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(hives))
	for _, h := range hives {
		out[h.ID] = h.Name
	}
	return out, nil
}

func scheduleNames(db *gorm.DB) (map[uuid.UUID]string, error) {
	var rows []tenant.MaintenanceSchedule
	if err := db.Select("id", "name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, s := range rows {
		out[s.ID] = s.Name
	}
	return out, nil
}

func scheduleTargetKind(s tenant.MaintenanceSchedule) string {
	if s.HiveID != nil {
		return tenant.AuditTargetHive
	}
	return tenant.AuditTargetFleet
}

func scheduleTargetID(s tenant.MaintenanceSchedule) *uuid.UUID { return s.HiveID }

func scheduleAuditMeta(s tenant.MaintenanceSchedule) map[string]any {
	m := map[string]any{
		"schedule_id": s.ID.String(), "name": s.Name, "kind": s.Kind,
		"duration_minutes": s.DurationMinutes, "reboot": s.Reboot, "enabled": s.Enabled,
		"standalone": s.Standalone,
	}
	if s.Kind == tenant.MaintenanceRecurring {
		m["weekdays"], m["start_time"], m["timezone"] = weekdayList(s.Weekdays), s.StartTime, s.Timezone
	} else if s.OnceAt != nil {
		m["once_at"] = s.OnceAt.UTC().Format(time.RFC3339)
	}
	return m
}

func weekdayList(mask int) []int {
	out := []int{}
	for d := range 7 {
		if mask&(1<<d) != 0 {
			out = append(out, d)
		}
	}
	return out
}

func scheduleView(s tenant.MaintenanceSchedule, hives map[uuid.UUID]string, now time.Time) MaintenanceScheduleView {
	v := MaintenanceScheduleView{
		ID: s.ID.String(), Standalone: s.Standalone, Name: s.Name, Kind: s.Kind,
		Weekdays: weekdayList(s.Weekdays), StartTime: s.StartTime, Timezone: s.Timezone,
		DurationMinutes: s.DurationMinutes, Reboot: s.Reboot, Enabled: s.Enabled,
		NextOpenings: []string{},
	}
	if s.HiveID != nil {
		v.HiveID, v.HiveName = s.HiveID.String(), hives[*s.HiveID]
	}
	if s.OnceAt != nil {
		v.OnceAt = s.OnceAt.UTC().Format(time.RFC3339)
	}
	if s.LastWindowAt != nil {
		v.LastWindowAt = s.LastWindowAt.UTC().Format(time.RFC3339)
	}
	if s.Enabled {
		_, v.OpenNow = s.WindowAt(now)
		for _, t := range s.NextOpenings(now, 3) {
			v.NextOpenings = append(v.NextOpenings, t.Format(time.RFC3339))
		}
	}
	return v
}

func runView(r tenant.MaintenanceRun, schedules map[uuid.UUID]string) MaintenanceRunView {
	ts := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	v := MaintenanceRunView{
		ID: r.ID.String(), Standalone: r.Standalone, Status: r.Status, Error: r.Error,
		StopRequested: r.StopRequested, Reboot: r.Reboot,
		WindowStart: ts(&r.WindowStart), WindowEnd: ts(&r.WindowEnd), StartedAt: ts(&r.StartedAt),
		FinishedAt: ts(r.FinishedAt), Nodes: make([]MaintenanceRunNodeView, 0, len(r.Nodes)),
	}
	if r.ScheduleID != nil {
		v.ScheduleID, v.ScheduleName = r.ScheduleID.String(), schedules[*r.ScheduleID]
	}
	if r.HiveID != nil {
		v.HiveID = r.HiveID.String()
	}
	for _, n := range r.Nodes {
		nv := MaintenanceRunNodeView{
			HypervisorID: n.HypervisorID.String(), Name: n.Name, Status: n.Status, Reason: n.Reason,
			LeftCordoned: n.CordonedAt != nil && !n.Restored && n.Status != tenant.MaintenanceNodeUpgrading,
			StartedAt:    ts(n.StartedAt), Deadline: ts(n.Deadline), FinishedAt: ts(n.FinishedAt),
		}
		if n.TaskID != nil {
			nv.TaskID = n.TaskID.String()
		}
		v.Nodes = append(v.Nodes, nv)
	}
	return v
}
