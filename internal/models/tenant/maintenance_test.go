package tenant

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func recurring(days int, at, tz string, minutes int) MaintenanceSchedule {
	id := uuid.New()
	return MaintenanceSchedule{
		HiveID: &id, Name: "w", Kind: MaintenanceRecurring, Weekdays: days, StartTime: at,
		Timezone: tz, DurationMinutes: minutes, Reboot: RebootIfNeeded, Enabled: true,
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

const sunday = 1 << time.Sunday

func TestWindowIsWallClockInTheSchedulesZone(t *testing.T) {
	s := recurring(sunday, "02:00", "Europe/Helsinki", 120)
	// Helsinki is UTC+3 in summer and UTC+2 in winter: the same 02:00 local is
	// two different UTC instants, and both must open the window.
	for _, c := range []struct{ now, open string }{
		{"2026-07-05T00:30:00Z", "2026-07-04T23:00:00Z"}, // summer, 03:30 local
		{"2026-12-06T00:30:00Z", "2026-12-06T00:00:00Z"}, // winter, 02:30 local
	} {
		open, ok := s.WindowAt(mustTime(t, c.now))
		if !ok || !open.Equal(mustTime(t, c.open)) {
			t.Errorf("at %s: want window opened %s, got %v %v", c.now, c.open, open, ok)
		}
	}
	if _, ok := s.WindowAt(mustTime(t, "2026-07-05T01:30:00Z")); ok {
		t.Error("04:30 local is after a two-hour window")
	}
	if _, ok := s.WindowAt(mustTime(t, "2026-07-06T00:30:00Z")); ok {
		t.Error("Monday is not a Sunday window")
	}
}

// A long window opened late yesterday is still open after midnight.
func TestWindowSpansMidnight(t *testing.T) {
	s := recurring(sunday, "22:00", "UTC", 6*60)
	open, ok := s.WindowAt(mustTime(t, "2026-07-06T01:00:00Z")) // Monday 01:00
	if !ok || !open.Equal(mustTime(t, "2026-07-05T22:00:00Z")) {
		t.Fatalf("got %v %v", open, ok)
	}
}

func TestNextOpenings(t *testing.T) {
	s := recurring(1<<time.Tuesday|1<<time.Thursday, "03:15", "UTC", 60)
	got := s.NextOpenings(mustTime(t, "2026-10-30T12:00:00Z"), 3) // a Friday
	want := []string{"2026-11-03T03:15:00Z", "2026-11-05T03:15:00Z", "2026-11-10T03:15:00Z"}
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if !got[i].Equal(mustTime(t, want[i])) {
			t.Errorf("%d: want %s got %s", i, want[i], got[i])
		}
	}
}

func TestOnceWindow(t *testing.T) {
	at := mustTime(t, "2026-11-01T10:00:00Z")
	s := MaintenanceSchedule{Kind: MaintenanceOnce, OnceAt: &at, DurationMinutes: 30}
	if _, ok := s.WindowAt(at.Add(-time.Second)); ok {
		t.Error("not open before it opens")
	}
	if _, ok := s.WindowAt(at.Add(29 * time.Minute)); !ok {
		t.Error("open inside")
	}
	if _, ok := s.WindowAt(at.Add(30 * time.Minute)); ok {
		t.Error("closed at the end")
	}
}

func TestValidateMaintenanceSchedule(t *testing.T) {
	ok := recurring(sunday, "02:00", "Europe/Helsinki", 120)
	if err := ValidateMaintenanceSchedule(ok); err != nil {
		t.Fatalf("valid schedule refused: %v", err)
	}
	cases := map[string]struct {
		edit func(*MaintenanceSchedule)
		want error
	}{
		// The one that matters most: no scope must not mean "every unclustered
		// machine".
		"no scope":    {func(s *MaintenanceSchedule) { s.HiveID = nil }, ErrMaintenanceScope},
		"both scopes": {func(s *MaintenanceSchedule) { s.Standalone = true }, ErrMaintenanceScope},
		"too short":   {func(s *MaintenanceSchedule) { s.DurationMinutes = 5 }, ErrMaintenanceDuration},
		"too long":    {func(s *MaintenanceSchedule) { s.DurationMinutes = 13 * 60 }, ErrMaintenanceDuration},
		"no days":     {func(s *MaintenanceSchedule) { s.Weekdays = 0 }, ErrMaintenanceWeekdays},
		"bad days":    {func(s *MaintenanceSchedule) { s.Weekdays = 1 << 9 }, ErrMaintenanceWeekdays},
		"bad clock":   {func(s *MaintenanceSchedule) { s.StartTime = "2:00" }, ErrMaintenanceClock},
		"25 oclock":   {func(s *MaintenanceSchedule) { s.StartTime = "25:00" }, ErrMaintenanceClock},
		"no zone":     {func(s *MaintenanceSchedule) { s.Timezone = "" }, ErrMaintenanceTimezone},
		"bad zone":    {func(s *MaintenanceSchedule) { s.Timezone = "Mars/Olympus" }, ErrMaintenanceTimezone},
		"bad reboot":  {func(s *MaintenanceSchedule) { s.Reboot = "sometimes" }, ErrMaintenanceReboot},
		"bad kind":    {func(s *MaintenanceSchedule) { s.Kind = "cron" }, ErrMaintenanceKind},
		"blank name":  {func(s *MaintenanceSchedule) { s.Name = "  " }, ErrMaintenanceName},
		"once no time": {func(s *MaintenanceSchedule) {
			s.Kind = MaintenanceOnce
		}, ErrMaintenanceOnceAt},
	}
	for name, c := range cases {
		s := ok
		c.edit(&s)
		if err := ValidateMaintenanceSchedule(s); !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", name, c.want, err)
		}
	}
}
