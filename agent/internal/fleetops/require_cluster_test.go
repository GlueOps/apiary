package fleetops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

	// Cluster nil with the construction error kept — exactly the state fleetDeps
	// leaves behind when the agent has no Proxmox token configured, which is the
	// state a real host was found in.
	d := Deps{ClusterErr: errors.New(
		"fleetops: no Proxmox API token is configured for this host " +
			"(set proxmox_token_id and proxmox_secret in /etc/apiary-agent/config.yaml)")}
	e := &Env{
		Logf:  func(string, string, ...any) {},
		Phase: func(string) {},
	}

	// dist-upgrade is deliberately NOT here any more. It is a local job — apt-get
	// and a reboot — and the control plane gates it before the task is created,
	// so it no longer needs a Proxmox client at all. Join and evict do: they are
	// cluster operations, and `pvecm delnode` has no local equivalent that knows
	// the cluster's view.
	for name, run := range map[string]func() error{
		"join":  func() error { return Join(context.Background(), d, e, "{}") },
		"evict": func() error { return Evict(context.Background(), d, e, "{}") },
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
			// looking for a network problem, and the actual cause is not guessable
			// from the symptom.
			if !strings.Contains(err.Error(), "token") {
				t.Errorf("%s: %q does not say what to do about it", name, err)
			}
		})
	}
}

// TestTheRefusalCarriesTheRealCauseRatherThanGuessing.
//
// The first version of this refusal named a cause it could not know: it blamed
// the token's permissions. On a host whose token was fully privileged and simply
// never configured, that sent the operator to check ACLs — a confidently wrong
// message is worse than one that admits it does not know.
func TestTheRefusalCarriesTheRealCauseRatherThanGuessing(t *testing.T) {
	t.Parallel()

	cause := errors.New("no Proxmox API token is configured for this host")
	d := Deps{ClusterErr: cause}

	err := requireCluster(d)
	if err == nil {
		t.Fatal("a nil cluster was accepted")
	}
	if !errors.Is(err, ErrNoCluster) {
		t.Fatalf("got %v, want it to wrap ErrNoCluster so callers can still match it", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("got %q, which drops the reason the client could not be built", err)
	}
}

// TestARefusalWithNoRecordedCauseSaysOnlyWhatItKnows.
func TestARefusalWithNoRecordedCauseSaysOnlyWhatItKnows(t *testing.T) {
	t.Parallel()

	err := requireCluster(Deps{})
	if !errors.Is(err, ErrNoCluster) {
		t.Fatalf("got %v, want ErrNoCluster", err)
	}
	// No invented cause appended.
	if strings.Contains(err.Error(), "role") || strings.Contains(err.Error(), "PVEAuditor") {
		t.Fatalf("%q guesses at a cause it does not have", err)
	}
}

// TestDistUpgradeNoLongerNeedsAProxmoxClient.
//
// It is a local job: apt-get update, apt-get dist-upgrade, and a reboot. Nothing
// about it is orchestrated through the Proxmox API. The API was only ever used
// for the reboot preflight, and the control plane now answers those questions
// from facts it already holds — so a host with no token can still be patched.
//
// Asserted by reaching the RUNNER: with no cluster the preflight is skipped, and
// the first thing dist-upgrade touches is apt.
func TestDistUpgradeNoLongerNeedsAProxmoxClient(t *testing.T) {
	t.Parallel()

	var ran []string
	d := Deps{
		Node: "pve1",
		Runner: runnerFunc(func(_ context.Context, _ func(string, string, ...any), name string, args ...string) error {
			ran = append(ran, name+" "+strings.Join(args, " "))
			return nil
		}),
	}
	e := &Env{
		Logf:    func(string, string, ...any) {},
		Phase:   func(string) {},
		Declare: func(context.Context, time.Duration, string) error { return nil },
	}

	// reboot: never, so the run ends after the packages and touches nothing else.
	if err := DistUpgrade(context.Background(), d, e, "", `{"reboot":"never"}`); err != nil {
		t.Fatalf("dist-upgrade refused without a Proxmox client: %v", err)
	}

	joined := strings.Join(ran, " | ")
	if !strings.Contains(joined, "apt-get update") || !strings.Contains(joined, "dist-upgrade") {
		t.Fatalf("the upgrade never ran: %v", ran)
	}
}

// runnerFunc adapts a function to the Runner interface.
type runnerFunc func(ctx context.Context, logf func(string, string, ...any), name string, args ...string) error

func (f runnerFunc) Run(ctx context.Context, logf func(string, string, ...any), name string, args ...string) error {
	return f(ctx, logf, name, args...)
}
