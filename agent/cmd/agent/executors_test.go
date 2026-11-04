package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary-agent/internal/bee"
	"github.com/glueops/apiary-agent/internal/fleetops"
	"github.com/glueops/apiary-agent/internal/store"
)

// recordingDetached records the step keys it is asked to run.
type recordingDetached struct{ keys []string }

func (r *recordingDetached) RunDetached(_ context.Context, _ func(string, string, ...any), key, _ string, _ ...string) error {
	r.keys = append(r.keys, key)
	return nil
}
func (r *recordingDetached) State(context.Context, string) (fleetops.StepState, error) {
	return fleetops.StepNotStarted, nil
}
func (r *recordingDetached) Forget(context.Context, string) error { return nil }

// cleanHost answers every command successfully and silently, which for
// `dpkg --audit` means a consistent package database.
type cleanHost struct{}

func (cleanHost) Run(context.Context, func(string, string, ...any), string, ...string) error {
	return nil
}

func testBeeEnv() *bee.Env {
	return &bee.Env{
		Logf:    func(string, string, ...any) {},
		Phase:   func(string) {},
		Declare: func(context.Context, time.Duration, string) error { return nil },
	}
}

// The dist-upgrade executor must hand fleetops the task's id: it is what keys
// each detached step to THIS task. Without it every step is refused.
func TestDistUpgradeExecutorKeysItsStepsByTheTaskID(t *testing.T) {
	det := &recordingDetached{}
	d := fleetops.Deps{Runner: cleanHost{}, Detached: det, Node: "hv-01",
		Reboot: func(context.Context) error { t.Fatal("rebooted under reboot=never"); return nil }}

	exec := executors(d, Config{})[fleetops.KindDistUpgrade]
	err := exec(context.Background(), store.Task{ID: "task-abc", Spec: `{"reboot":"never"}`}, testBeeEnv())
	if errors.Is(err, fleetops.ErrInsufficientSpace) {
		t.Skipf("the machine running the test is short of disk: %v", err)
	}
	if err != nil {
		t.Fatalf("dist-upgrade: %v", err)
	}
	if len(det.keys) == 0 {
		t.Fatal("no detached step ran")
	}
	for _, k := range det.keys {
		if !strings.HasPrefix(k, "task-abc-") {
			t.Fatalf("step key %q is not keyed by the task id", k)
		}
	}
}

// Production wires the systemd runner, with its records next to the agent's
// state. A nil Detached would silently run apt as the agent's own child, which
// an agent restart kills mid-install.
func TestFleetDepsWiresDetachedSteps(t *testing.T) {
	dir := t.TempDir()
	d := fleetDeps(Config{StatePath: filepath.Join(dir, "state.db")})
	if d.Detached == nil {
		t.Fatal("package steps are not detached")
	}
	if fi, err := os.Stat(filepath.Join(dir, "steps")); err != nil || !fi.IsDir() {
		t.Fatalf("the steps directory was not created next to the state: %v", err)
	}
}
