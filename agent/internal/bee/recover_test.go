package bee

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glueops/apiary-agent/internal/store"
)

// TestAPanickingExecutorFailsItsTaskInsteadOfTheAgent.
//
// THE CRASH LOOP. A panic in an executor kills the process; systemd restarts it
// — Restart=always with no rate limit, which is deliberate — the agent collects
// the SAME task and panics again. The host is then in a loop no amount of
// waiting resolves, and the only evidence is a stack trace in the journal of a
// machine nobody is watching. One task reached attempt 125 that way.
//
// The blast radius is the reason this is worth a recover rather than "do not
// write panicking executors": tasks are queued per bee, so a FLEET-WIDE
// operation that panics takes down every agent in the fleet identically. A
// failed task is recoverable and visible; a fleet of crash-looping agents is
// neither.
func TestAPanickingExecutorFailsItsTaskInsteadOfTheAgent(t *testing.T) {
	t.Parallel()

	boom := func(context.Context, store.Task, *Env) error {
		var cluster *struct{ Status func() }
		// A nil dereference, which is what actually happened: a task reached a nil
		// Proxmox client three calls deep.
		cluster.Status()
		return nil
	}

	err := runExecutor(context.Background(), boom,
		store.Task{ID: "t-1", Kind: "fleet.host.dist-upgrade"}, &Env{})
	if err == nil {
		t.Fatal("a panicking executor reported success")
	}
	if !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("error %q does not say the agent panicked", err)
	}
	// The kind is in the message because the task is rendered beside the machine
	// in the control plane, and "something panicked" sends nobody anywhere.
	if !strings.Contains(err.Error(), "fleet.host.dist-upgrade") {
		t.Fatalf("error %q does not name the task that did it", err)
	}
}

// TestANormalErrorIsUnchanged, so the recover does not reshape ordinary
// failures on its way past them.
func TestANormalErrorIsUnchanged(t *testing.T) {
	t.Parallel()

	sentinel := errSentinel("the host refused")
	fine := func(context.Context, store.Task, *Env) error { return sentinel }

	err := runExecutor(context.Background(), fine, store.Task{ID: "t-2"}, &Env{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want the executor's own error untouched", err)
	}
}

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
