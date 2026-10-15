package fleetops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// hangingRunner blocks until its context is cancelled, which is what a command
// with no deadline does to a caller that has none either.
type hangingRunner struct{ ran []string }

func (h *hangingRunner) Run(ctx context.Context, _ func(string, string, ...any),
	_ string, args ...string) error {
	h.ran = append(h.ran, strings.Join(args, " "))
	<-ctx.Done()
	return ctx.Err()
}

// A DISARM THAT CANNOT FINISH MUST FAIL AND SAY WHERE.
//
// `pvecm updatecerts --force` can block indefinitely on a node with no cluster.
// It is guarded with `|| true`, which catches a bad exit code and not a hang,
// and nothing else bounded the command — so the disarm never returned. The task
// stayed claimed, was re-delivered on every heartbeat, and climbed past attempt
// 700 while holding the bee's only task slot, which also stopped that host being
// upgraded.
func TestDisarmFailsRatherThanHangingOnAStalledStep(t *testing.T) {
	old := disarmStepTimeout
	disarmStepTimeout = 50 * time.Millisecond
	t.Cleanup(func() { disarmStepTimeout = old })

	r := &hangingRunner{}
	var phases []string
	done := make(chan error, 1)
	go func() {
		done <- Disarm(context.Background(), Deps{Runner: r},
			testEnv(&phases, nil, nil), `{"disposition":"accept-exposure"}`)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a stalled disarm reported success")
		}
		if !strings.Contains(err.Error(), "stalled") {
			t.Fatalf("err = %v, want it to say the step stalled", err)
		}
		// And it names the command, so an operator knows where to look.
		if !strings.Contains(err.Error(), "systemctl") && !strings.Contains(err.Error(), "pmxcfs") {
			t.Fatalf("err = %v, want it to name the step", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Disarm did not return; it would hold the bee's only task slot forever")
	}
}

// failingRunner fails the command containing failOn at once, printing why, and
// succeeds at everything else.
type failingRunner struct {
	failOn string
	ran    []string
}

func (f *failingRunner) Run(_ context.Context, log func(string, string, ...any),
	_ string, args ...string) error {
	cmd := strings.Join(args, " ")
	f.ran = append(f.ran, cmd)
	if strings.Contains(cmd, f.failOn) {
		log("stderr", "%s", "[main] crit: unable to acquire pmxcfs lock: Resource temporarily unavailable")
		return fmt.Errorf("exit status 255")
	}
	return nil
}

// A step that FAILS is not a step that STALLED, and the output says why.
//
// Live: pmxcfs refused to start within a second on a held lock, and the task
// reported `stalled at "pmxcfs -l" after 2m0s` — cancel() ran before the
// deadline was checked, so every failure read as a timeout, and the step's
// output had been discarded, so nothing said what the lock was.
func TestADisarmStepThatFailsIsNotReportedAsAStall(t *testing.T) {
	r := &failingRunner{failOn: "pmxcfs -l"}
	var phases []string
	err := Disarm(context.Background(), Deps{Runner: r},
		testEnv(&phases, nil, nil), `{"disposition":"accept-exposure"}`)
	if err == nil {
		t.Fatal("a failed step reported success")
	}
	if strings.Contains(err.Error(), "stalled") {
		t.Fatalf("err = %v, want a failure, not a stall", err)
	}
	if !strings.Contains(err.Error(), "unable to acquire pmxcfs lock") {
		t.Fatalf("err = %v, want the step's own output in it", err)
	}
}

// pmxcfs is stopped with pkill and WAITED for, never with killall.
//
// killall is psmisc, absent on a stock PVE 9 host, and `killall … || true`
// turned "command not found" into success: the local-mode pmxcfs survived, held
// its lock, and pve-cluster could not start. Five hosts were left with /etc/pve
// unmounted that way.
func TestDisarmStopsPmxcfsWithoutKillall(t *testing.T) {
	dir := t.TempDir()
	oldCoro, oldPVEConf, oldSecrets, oldPVE := CorosyncConf, PVECorosyncConf, clusterSecretFiles, PVEDir
	CorosyncConf = filepath.Join(dir, "corosync.conf")
	PVECorosyncConf = filepath.Join(dir, "pve-corosync.conf")
	clusterSecretFiles = []string{filepath.Join(dir, "authkey.key")}
	PVEDir = dir
	t.Cleanup(func() {
		CorosyncConf, PVECorosyncConf, clusterSecretFiles, PVEDir = oldCoro, oldPVEConf, oldSecrets, oldPVE
	})

	r := &recordingRunner{}
	var phases []string
	_ = Disarm(context.Background(), Deps{Runner: r}, testEnv(&phases, nil, nil),
		`{"disposition":"accept-exposure"}`)

	stops := 0
	for _, c := range r.cmds {
		line := strings.Join(c, " ")
		if strings.Contains(line, "killall") {
			t.Fatalf("disarm runs killall, which a PVE 9 host does not have: %q", line)
		}
		if strings.Contains(line, "pkill -x pmxcfs") {
			if !strings.Contains(line, "pgrep -x pmxcfs") || !strings.Contains(line, "exit 1") {
				t.Fatalf("pmxcfs is killed but not waited for, or a survivor is not a failure: %q", line)
			}
			stops++
		}
	}
	if stops != 2 {
		t.Fatalf("pmxcfs stopped %d time(s), want 2 (before local mode and before pve-cluster starts)", stops)
	}
}

// The ordinary path still runs every step in order and proves the result.
func TestDisarmRunsTheConversionAndProvesIt(t *testing.T) {
	dir := t.TempDir()
	oldCoro, oldSecrets, oldPVE := CorosyncConf, clusterSecretFiles, PVEDir
	oldCap := capabilityBearingSecrets
	CorosyncConf = filepath.Join(dir, "corosync.conf")
	PVEDir = dir
	secret := filepath.Join(dir, "token.cfg")
	if err := os.WriteFile(secret, []byte("hive secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Both: one is what gets destroyed, the other is what gets proved. A stand-in
	// secret that is only in the first is removed and never checked.
	clusterSecretFiles = []string{secret}
	capabilityBearingSecrets = []string{secret}
	t.Cleanup(func() {
		CorosyncConf, clusterSecretFiles, PVEDir = oldCoro, oldSecrets, oldPVE
		capabilityBearingSecrets = oldCap
	})

	// A runner that actually removes the secret, as the real commands would.
	r := &recordingRunner{}
	var phases []string
	err := Disarm(context.Background(), Deps{Runner: r},
		testEnv(&phases, nil, nil), `{"disposition":"rotate","peers":["pve3"]}`)
	// The proof fails here because nothing removed the file — which is exactly
	// what it is for: commands exiting zero is not evidence.
	if err == nil {
		t.Fatal("the disarm reported success while the hive's secret was still on disk")
	}
	if !strings.Contains(err.Error(), "byte-identical") {
		t.Fatalf("err = %v, want the proof to say the secret survived", err)
	}
	// Rotation was asked for, so the keypair step must have been attempted.
	var sawRotate bool
	for _, c := range r.cmds {
		if strings.Contains(strings.Join(c, " "), "ssh-keygen") {
			sawRotate = true
		}
	}
	if !sawRotate {
		t.Fatalf("rotate was chosen but no keypair was regenerated: %v", r.cmds)
	}
}

// THE SERVICE RESTART IS QUEUED, NOT WAITED ON.
//
// `systemctl restart pveproxy pvedaemon` hung for minutes on pve5 with the
// disarm waiting on it: the task never terminated, was re-delivered on every
// heartbeat past attempt 1200, and held the bee's only task slot — which also
// stopped that host being upgraded.
//
// Nothing needs the restart to have finished. The hive's secrets are destroyed
// before it, and the proof reads files rather than the API.
func TestDisarmDoesNotWaitForTheServiceRestart(t *testing.T) {
	dir := t.TempDir()
	oldCoro, oldSecrets, oldPVE := CorosyncConf, clusterSecretFiles, PVEDir
	CorosyncConf = filepath.Join(dir, "corosync.conf")
	clusterSecretFiles = nil
	PVEDir = dir
	t.Cleanup(func() { CorosyncConf, clusterSecretFiles, PVEDir = oldCoro, oldSecrets, oldPVE })

	r := &recordingRunner{}
	var phases []string
	if err := Disarm(context.Background(), Deps{Runner: r},
		testEnv(&phases, nil, nil), `{"disposition":"accept-exposure"}`); err != nil {
		t.Fatalf("disarm: %v", err)
	}

	var restart string
	for _, c := range r.cmds {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "pveproxy") {
			restart = joined
		}
	}
	if restart == "" {
		t.Fatalf("the service restart never ran: %v", r.cmds)
	}
	if !strings.Contains(restart, "--no-block") {
		t.Fatalf("restart = %q, want --no-block — waiting on it donates the disarm's fate to a "+
			"service that may not come back on a node whose cluster was just dismantled", restart)
	}
}

// BOTH COPIES OF THE COROSYNC CONFIG MUST GO.
//
// /etc/pve/corosync.conf is the cluster's, on pmxcfs; /etc/corosync/corosync.conf
// is the local file corosync reads, and pve-cluster REMAKES it from the pmxcfs
// copy on start. Removing only the local one therefore achieves nothing, because
// a later step in this very sequence starts pve-cluster and puts it straight
// back.
//
// That is what happened on pve5: the disarm removed the local copy, started
// pve-cluster, and the proof refused with "still exists, so this node is not
// standalone" — correctly, on a node left carrying the configuration of a
// cluster it had left.
func TestDisarmRemovesTheClusterCopyOfCorosyncConfToo(t *testing.T) {
	dir := t.TempDir()
	oldCoro, oldPVEConf, oldSecrets, oldPVE := CorosyncConf, PVECorosyncConf, clusterSecretFiles, PVEDir
	CorosyncConf = filepath.Join(dir, "corosync.conf")
	PVECorosyncConf = filepath.Join(dir, "pve-corosync.conf")
	clusterSecretFiles = nil
	PVEDir = dir
	t.Cleanup(func() {
		CorosyncConf, PVECorosyncConf, clusterSecretFiles, PVEDir = oldCoro, oldPVEConf, oldSecrets, oldPVE
	})

	r := &recordingRunner{}
	var phases []string
	if err := Disarm(context.Background(), Deps{Runner: r},
		testEnv(&phases, nil, nil), `{"disposition":"accept-exposure"}`); err != nil {
		t.Fatalf("disarm: %v", err)
	}

	var sawPVECopy bool
	for _, c := range r.cmds {
		if strings.Contains(strings.Join(c, " "), PVECorosyncConf) {
			sawPVECopy = true
		}
	}
	if !sawPVECopy {
		t.Fatalf("the cluster's own copy at %s was never removed, so pve-cluster would remake "+
			"the local one on start: %v", PVECorosyncConf, r.cmds)
	}
}

// A SECOND DISARM ON AN ALREADY-DISARMED NODE MUST NOT FAIL ON A COUNTER.
//
// Observed on 192.168.240.12: two departures forty-nine seconds apart each
// queued a disarm. The first succeeded. The second failed with
// "/etc/pve/priv/pve-root-ca.srl is still byte-identical to the copy this node
// left the hive with, so the cluster's secret went with it" — on a host whose
// cluster secrets were provably already gone, and the failure is what the fleet
// view then reported.
//
// The .srl is OpenSSL's serial counter for the CA, not a secret. `pvecm
// updatecerts --force` regenerates the CA with the serial starting from the same
// value, so on a re-run it is byte-identical by construction. A file that grants
// nothing was able to veto the whole disarm.
func TestTheCASerialCounterCannotVetoADisarm(t *testing.T) {
	for _, f := range capabilityBearingSecrets {
		if strings.HasSuffix(f, ".srl") {
			t.Fatalf("%s is a serial counter, not a capability — it regenerates identically "+
				"and will fail every repeated disarm", f)
		}
	}

	// It is still destroyed; it just does not get a vote on the proof.
	var destroyed bool
	for _, f := range clusterSecretFiles {
		if strings.HasSuffix(f, "pve-root-ca.srl") {
			destroyed = true
		}
	}
	if !destroyed {
		t.Fatal("the CA serial is no longer destroyed; excluding it from the proof must not " +
			"exclude it from the removal")
	}

	// Every file that DOES get a vote must be one whose possession authenticates.
	for _, f := range capabilityBearingSecrets {
		switch f {
		case "/etc/pve/priv/token.cfg",
			"/etc/pve/priv/authkey.key",
			"/etc/pve/priv/pve-root-ca.key":
		default:
			t.Fatalf("%s proves rotation but does not mint or sign anything; a file PVE "+
				"recreates on its own will fail disarms that actually worked", f)
		}
	}
}
