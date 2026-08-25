package config

import (
	"strings"
	"testing"
	"time"
)

func TestWorkerRefusesASyncIntervalThatExpiresTheFleet(t *testing.T) {
	// The failure this guards is not a crash but a fleet that silently refuses
	// every departure: an observation older than the max age reads as unknown, so
	// a sync interval at or above it means every observation is expired by the
	// time the next one lands, and the gate — working exactly as designed —
	// blocks everything.
	for _, tc := range []struct {
		name     string
		interval time.Duration
		maxAge   time.Duration
		wantErr  bool
	}{
		{"comfortably inside", 3 * time.Minute, 10 * time.Minute, false},
		{"exactly half", 5 * time.Minute, 10 * time.Minute, false},
		{"just over half — one missed sync expires everything", 6 * time.Minute, 10 * time.Minute, true},
		{"equal", 10 * time.Minute, 10 * time.Minute, true},
		{"longer than the window", 30 * time.Minute, 10 * time.Minute, true},
		{"unset interval is not checked", 0, 10 * time.Minute, false},
		{"unset max age is not checked", 10 * time.Minute, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				FleetCapacitySyncInterval: tc.interval,
				FleetObservationMaxAge:    tc.maxAge,
			}
			err := checkFleetSyncCadence(c)
			if tc.wantErr && err == nil {
				t.Fatal("expected a refusal")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "drain_unknown") {
				t.Errorf("the message must name the symptom an operator would actually see, got %q",
					err.Error())
			}
		})
	}
}
