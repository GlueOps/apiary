package fleetops

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestClusterTasksRefuseWithoutAClientRatherThanPanicking.
//
// THE CRASH LOOP THIS ENDS. A nil Cluster used to be dereferenced three calls
// deep, which killed the agent; systemd restarted it — Restart=always with no
// rate limit, deliberately — the agent collected the SAME task and panicked
// again. One task reached attempt 125 that way, against a host whose only real
// problem was a Proxmox token with no API role.
//
// The agent already logs "fleet tasks will refuse rather than run" at startup
// when it cannot build a client. This is that promise being kept.
func TestClusterTasksRefuseWithoutAClientRatherThanPanicking(t *testing.T) {
	t.Parallel()

	// Deliberately zero: Cluster nil is exactly the state fleetDeps leaves behind
	// when the token cannot reach the API.
	var d Deps
	e := &Env{
		Logf:  func(string, string, ...any) {},
		Phase: func(string) {},
	}

	for name, run := range map[string]func() error{
		"dist-upgrade": func() error { return DistUpgrade(context.Background(), d, e, "", "{}") },
		"join":         func() error { return Join(context.Background(), d, e, "{}") },
		"evict":        func() error { return Evict(context.Background(), d, e, "{}") },
	} {
		t.Run(name, func(t *testing.T) {
			// A panic here fails the test rather than taking the runner down, which
			// is the same guarantee runExecutor gives the agent.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s panicked on a nil cluster instead of refusing: %v", name, r)
				}
			}()

			err := run()
			if err == nil {
				t.Fatalf("%s ran without a Proxmox client", name)
			}
			if !errors.Is(err, ErrNoCluster) {
				t.Fatalf("%s failed with %v, want ErrNoCluster", name, err)
			}
			// The message has to name the fix. An operator reading "no client" goes
			// looking for a network problem; the actual cause is a token with no
			// role, and it is not guessable from the symptom.
			if !strings.Contains(err.Error(), "token") {
				t.Errorf("%s: %q does not say what to do about it", name, err)
			}
		})
	}
}
