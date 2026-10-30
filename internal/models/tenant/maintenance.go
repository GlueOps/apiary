package tenant

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Scheduled maintenance: a window in which the control plane upgrades and
// reboots the hypervisors of one scope, ONE AT A TIME, stopping at the first one
// that does not come back.
//
// A SCOPE is either one cluster or the STANDALONE POOL — every hypervisor in no
// cluster. The pool exists so that a machine outside a cluster is not a machine
// nobody ever patches; it is resolved when a run starts, so a host that joins a
// cluster moves onto that cluster's schedule without anybody editing either.
//
// The schedule GRANTS GUEST DOWNTIME. That is the operator's decision, made once
// when the window is created, and it is the only condition of the upgrade gate a
// run lifts: staleness and quorum are still checked for every node, at its turn.

// Schedule kinds.
const (
	MaintenanceRecurring = "recurring"
	MaintenanceOnce      = "once"
)

// Bounds on a window. Short enough below that a window is not a typo for a
// minute; bounded above so a window cannot quietly become "always".
const (
	MaintenanceMinDuration = 15 * time.Minute
	MaintenanceMaxDuration = 12 * time.Hour
)

// Run statuses. A run is terminal in every state but running.
const (
	MaintenanceRunRunning   = "running"
	MaintenanceRunSucceeded = "succeeded"
	MaintenanceRunFailed    = "failed"
	MaintenanceRunStopped   = "stopped"
	// MaintenanceRunExpired is a run whose window closed with nodes still to do.
	// Not a failure — nothing went wrong — but not a success either, and the
	// nodes it never reached say so.
	MaintenanceRunExpired = "expired"
)

// Run-node statuses.
const (
	MaintenanceNodePending   = "pending"
	MaintenanceNodeUpgrading = "upgrading"
	MaintenanceNodeSucceeded = "succeeded"
	MaintenanceNodeFailed    = "failed"
	MaintenanceNodeSkipped   = "skipped"
)

// A MaintenanceSchedule is one window, recurring or once.
type MaintenanceSchedule struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// HiveID is the cluster, and NULL is the standalone pool. Standalone is
	// stored as its own flag as well so that "no cluster" is something a caller
	// said rather than something they forgot to say: a schedule created without
	// a hive id would otherwise reboot every unclustered machine in the fleet.
	HiveID     *uuid.UUID `gorm:"type:uuid;index;check:ck_maintenance_schedules_scope,(hive_id IS NULL) = standalone" json:"hive_id,omitempty"`
	Hive       *Hive      `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`
	Standalone bool       `gorm:"not null" json:"standalone"`

	Name string `gorm:"type:varchar(100);not null" json:"name"`
	Kind string `gorm:"type:varchar(16);not null;check:ck_maintenance_schedules_kind,kind IN ('recurring','once')" json:"kind"`

	// Recurring: the weekdays (bit 0 is Sunday, as time.Weekday counts) and the
	// local time the window opens, in Timezone. Stored as wall-clock time and a
	// zone rather than as UTC, because "Sunday at 02:00" means 02:00 in winter
	// AND in summer, and a UTC instant means one of them.
	Weekdays  int    `gorm:"not null" json:"weekdays"`
	StartTime string `gorm:"type:varchar(5);not null" json:"start_time"`
	Timezone  string `gorm:"type:varchar(64);not null" json:"timezone"`

	// Once: the instant the window opens.
	OnceAt *time.Time `gorm:"type:timestamptz" json:"once_at,omitempty"`

	DurationMinutes int `gorm:"not null" json:"duration_minutes"`

	// Reboot is the policy handed to each node's dist-upgrade.
	Reboot string `gorm:"type:varchar(16);not null" json:"reboot"`

	// No default: tag, deliberately — GORM's Create drops a false on a field that
	// has one and stores the column default instead.
	Enabled bool `gorm:"not null" json:"enabled"`

	// LastWindowAt is the opening of the window that last started a run, so one
	// window starts one run however many ticks land inside it.
	LastWindowAt *time.Time `gorm:"type:timestamptz" json:"last_window_at,omitempty"`

	CreatedByID *uuid.UUID `gorm:"type:uuid" json:"created_by_id,omitempty"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

// A MaintenanceRun is one window being worked.
type MaintenanceRun struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// SET NULL rather than CASCADE: deleting a schedule must not delete the
	// record of what it did to the fleet.
	ScheduleID *uuid.UUID           `gorm:"type:uuid;index" json:"schedule_id,omitempty"`
	Schedule   *MaintenanceSchedule `gorm:"foreignKey:ScheduleID;constraint:OnDelete:SET NULL" json:"-"`
	// The scope, copied from the schedule when the run started. A cluster's runs
	// go when the cluster does.
	HiveID     *uuid.UUID `gorm:"type:uuid;index" json:"hive_id,omitempty"`
	Hive       *Hive      `gorm:"foreignKey:HiveID;constraint:OnDelete:CASCADE" json:"-"`
	Standalone bool       `gorm:"not null" json:"standalone"`

	Status string `gorm:"type:varchar(16);not null;index" json:"status"`
	Reboot string `gorm:"type:varchar(16);not null" json:"reboot"`
	// Error says why a run stopped, when it did not finish.
	Error string `gorm:"type:text;not null" json:"error,omitempty"`
	// StopRequested is an operator asking the run to stop after the node in
	// flight. Not an abort: a host halfway through a reboot is not something the
	// control plane can take back.
	StopRequested bool `gorm:"not null" json:"stop_requested"`

	WindowStart time.Time  `gorm:"type:timestamptz;not null" json:"window_start"`
	WindowEnd   time.Time  `gorm:"type:timestamptz;not null" json:"window_end"`
	StartedAt   time.Time  `gorm:"type:timestamptz;not null" json:"started_at"`
	FinishedAt  *time.Time `gorm:"type:timestamptz" json:"finished_at,omitempty"`

	Nodes []MaintenanceRunNode `gorm:"foreignKey:RunID;constraint:OnDelete:CASCADE" json:"nodes,omitempty"`
}

// A MaintenanceRunNode is one hypervisor's turn in a run.
type MaintenanceRunNode struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RunID uuid.UUID `gorm:"type:uuid;not null;index" json:"run_id"`

	// No foreign key: the record of a node's turn outlives the machine record,
	// which a capacity sync may delete. Name is kept for the same reason.
	HypervisorID uuid.UUID `gorm:"type:uuid;not null;index" json:"hypervisor_id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Position     int       `gorm:"not null" json:"position"`

	Status string `gorm:"type:varchar(16);not null" json:"status"`
	Reason string `gorm:"type:text;not null" json:"reason,omitempty"`

	TaskID *uuid.UUID `gorm:"type:uuid" json:"task_id,omitempty"`

	// CordonedAt is set when THIS RUN (or a failed run before it, whose cordon
	// this one inherited) cordoned the machine. A node it finds already cordoned
	// by somebody else is never uncordoned by it, and a node somebody cordons by
	// hand after this instant stays cordoned too.
	CordonedAt *time.Time `gorm:"type:timestamptz" json:"cordoned_at,omitempty"`
	// Restored is the cordon having been handed back.
	Restored bool `gorm:"not null" json:"restored"`

	StartedAt  *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`
	Deadline   *time.Time `gorm:"type:timestamptz" json:"deadline,omitempty"`
	FinishedAt *time.Time `gorm:"type:timestamptz" json:"finished_at,omitempty"`
}

// Duration is the window's length.
func (s MaintenanceSchedule) Duration() time.Duration {
	return time.Duration(s.DurationMinutes) * time.Minute
}

// WindowAt returns the opening of the window that contains now, if any.
//
// A recurring window is looked for on today's date and yesterday's, in the
// schedule's own zone: a window may be up to twelve hours long, so one opened
// late yesterday can still be open. A zone that fails to load opens nothing —
// the schedule was validated when written, so that is a zone database that
// changed underneath it, and guessing UTC would reboot hosts at the wrong hour.
func (s MaintenanceSchedule) WindowAt(now time.Time) (time.Time, bool) {
	dur := s.Duration()
	switch s.Kind {
	case MaintenanceOnce:
		if s.OnceAt == nil {
			return time.Time{}, false
		}
		at := s.OnceAt.UTC()
		if !now.Before(at) && now.Before(at.Add(dur)) {
			return at, true
		}
		return time.Time{}, false
	case MaintenanceRecurring:
		loc, err := time.LoadLocation(s.Timezone)
		if err != nil {
			return time.Time{}, false
		}
		hh, mm, err := parseClock(s.StartTime)
		if err != nil {
			return time.Time{}, false
		}
		local := now.In(loc)
		for back := 0; back <= 1; back++ {
			day := local.AddDate(0, 0, -back)
			open := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc)
			if s.Weekdays&(1<<int(open.Weekday())) == 0 {
				continue
			}
			if !now.Before(open) && now.Before(open.Add(dur)) {
				return open.UTC(), true
			}
		}
	}
	return time.Time{}, false
}

// NextOpenings returns up to n window openings strictly after now. Shown when a
// schedule is written, so a wrong zone or weekday is seen before it reboots
// anything.
func (s MaintenanceSchedule) NextOpenings(now time.Time, n int) []time.Time {
	var out []time.Time
	switch s.Kind {
	case MaintenanceOnce:
		if s.OnceAt != nil && s.OnceAt.After(now) {
			out = append(out, s.OnceAt.UTC())
		}
	case MaintenanceRecurring:
		loc, err := time.LoadLocation(s.Timezone)
		if err != nil {
			return nil
		}
		hh, mm, err := parseClock(s.StartTime)
		if err != nil {
			return nil
		}
		local := now.In(loc)
		for d := 0; d <= 7*n && len(out) < n; d++ {
			day := local.AddDate(0, 0, d)
			open := time.Date(day.Year(), day.Month(), day.Day(), hh, mm, 0, 0, loc)
			if s.Weekdays&(1<<int(open.Weekday())) == 0 || !open.After(now) {
				continue
			}
			out = append(out, open.UTC())
		}
	}
	return out
}

// Reasons a schedule is refused. Errors rather than refusal codes: these are
// malformed requests, not the fleet saying "not now".
var (
	ErrMaintenanceScope    = errors.New("name exactly one scope: a cluster, or the standalone pool")
	ErrMaintenanceName     = errors.New("a schedule needs a name of at most 100 characters")
	ErrMaintenanceKind     = errors.New("kind must be recurring or once")
	ErrMaintenanceDuration = fmt.Errorf("the window must be between %s and %s long",
		MaintenanceMinDuration, MaintenanceMaxDuration)
	ErrMaintenanceReboot   = errors.New("reboot must be if-needed, always or never")
	ErrMaintenanceTimezone = errors.New("unknown time zone; use an IANA name such as Europe/Helsinki")
	ErrMaintenanceClock    = errors.New("start_time must be HH:MM, 24-hour")
	ErrMaintenanceWeekdays = errors.New("a recurring window needs at least one weekday")
	ErrMaintenanceOnceAt   = errors.New("a one-off window needs once_at")
)

// ValidateMaintenanceSchedule returns the first thing wrong with a schedule.
func ValidateMaintenanceSchedule(s MaintenanceSchedule) error {
	if (s.HiveID == nil) != s.Standalone {
		return ErrMaintenanceScope
	}
	if name := strings.TrimSpace(s.Name); name == "" || len(name) > 100 {
		return ErrMaintenanceName
	}
	d := s.Duration()
	if d < MaintenanceMinDuration || d > MaintenanceMaxDuration {
		return ErrMaintenanceDuration
	}
	switch s.Reboot {
	case RebootIfNeeded, RebootAlways, RebootNever:
	default:
		return ErrMaintenanceReboot
	}
	switch s.Kind {
	case MaintenanceRecurring:
		if s.Weekdays&0x7f == 0 || s.Weekdays&^0x7f != 0 {
			return ErrMaintenanceWeekdays
		}
		if _, _, err := parseClock(s.StartTime); err != nil {
			return ErrMaintenanceClock
		}
		// An empty name loads as UTC, which is a silent default; refused so the
		// zone is always one somebody chose.
		if s.Timezone == "" {
			return ErrMaintenanceTimezone
		}
		if _, err := time.LoadLocation(s.Timezone); err != nil {
			return ErrMaintenanceTimezone
		}
	case MaintenanceOnce:
		if s.OnceAt == nil {
			return ErrMaintenanceOnceAt
		}
	default:
		return ErrMaintenanceKind
	}
	return nil
}

func parseClock(s string) (int, int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != 5 {
		return 0, 0, ErrMaintenanceClock
	}
	return t.Hour(), t.Minute(), nil
}

// MaintenanceRunIsTerminal is the single definition of a finished run.
func MaintenanceRunIsTerminal(status string) bool {
	return status != MaintenanceRunRunning
}
