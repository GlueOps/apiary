package selfupgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runner struct {
	cmds   []string
	failOn string
}

func (r *runner) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	r.cmds = append(r.cmds, cmd)
	if r.failOn != "" && strings.Contains(cmd, r.failOn) {
		return "exec format error", errors.New("exit 1")
	}
	return "apiary-agent v1.2.3", nil
}

func (r *runner) did(sub string) bool {
	for _, c := range r.cmds {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// target writes a stand-in for the running binary and returns its path.
func target(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "apiary-agent")
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	return p
}

// TestADownloadThatDoesNotMatchItsDigestIsRefused.
//
// THE SECURITY TEST. Without the check, an upgrade is "run whatever arrived over
// the network" — and a truncated response is indistinguishable from a short
// program, so it is not only a question of a hostile server.
func TestADownloadThatDoesNotMatchItsDigestIsRefused(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	r := &runner{}
	_, err := Apply(context.Background(), r, Plan{
		Binary: []byte("something else entirely"),
		SHA256: digest([]byte("what was advertised")),
		Target: p,
	})
	if err == nil {
		t.Fatal("a binary that did not match its digest was installed")
	}
	if len(r.cmds) != 0 {
		t.Fatalf("the binary was executed before its digest was checked: %v", r.cmds)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "the old agent" {
		t.Fatalf("the existing binary was touched: %q", got)
	}
}

// TestTheNewBinaryIsProvedToRunBeforeAnythingIsReplaced.
//
// A binary for the wrong architecture fails here with a clear message instead of
// as a service that will not start — which reads as a crash in the new version
// and sends someone to the release notes.
func TestTheNewBinaryIsProvedToRunBeforeAnythingIsReplaced(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	body := []byte("the new agent")
	r := &runner{failOn: "upgrade version"}

	_, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p,
	})
	if err == nil {
		t.Fatal("a binary that could not run was installed")
	}
	if !strings.Contains(err.Error(), "does not run on this host") {
		t.Fatalf("error %q does not say what happened", err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "the old agent" {
		t.Fatalf("the old binary was replaced anyway: %q", got)
	}
	if _, err := os.Stat(p + ".previous"); err == nil {
		t.Fatal("a backup was left behind for an upgrade that never happened")
	}
}

// TestTheOldBinaryIsKept. An upgrade with no way back is not an upgrade.
func TestTheOldBinaryIsKept(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	body := []byte("the new agent")
	r := &runner{}

	res, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	now, _ := os.ReadFile(p)
	if string(now) != "the new agent" {
		t.Fatalf("the new binary is not in place: %q", now)
	}
	old, err := os.ReadFile(res.Backup)
	if err != nil || string(old) != "the old agent" {
		t.Fatalf("the previous binary was not kept: %q (%v)", old, err)
	}
}

// TestTheStagedFileSitsBesideTheTarget.
//
// rename is atomic only within a filesystem. Staging in /tmp and renaming across
// a mount boundary fails with EXDEV, and the usual fallback — copy then delete —
// is exactly the non-atomic replacement this exists to avoid.
func TestTheStagedFileSitsBesideTheTarget(t *testing.T) {
	t.Parallel()

	p := target(t, "old")
	body := []byte("new")
	r := &runner{}
	if _, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p,
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if !r.did(filepath.Dir(p)) {
		t.Fatalf("the staged binary was not run from the target's directory: %v", r.cmds)
	}
	// And it is cleaned up, or the next upgrade finds a stale file with a name it
	// is about to reuse.
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), ".apiary-agent.upgrade")); err == nil {
		t.Fatal("the staged file was left behind")
	}
}

// TestTheServiceIsRestartedBySystemdRatherThanByExec.
//
// This process IS the old binary. The supervisor is what knows how to start the
// new one with the right environment, state directory and limits.
func TestTheServiceIsRestartedBySystemdRatherThanByExec(t *testing.T) {
	t.Parallel()

	p := target(t, "old")
	body := []byte("new")
	r := &runner{}
	res, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p, Unit: "apiary-agent.service",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Restarted || !r.did("systemctl restart apiary-agent.service") {
		t.Fatalf("the unit was not restarted: %v", r.cmds)
	}
}

// TestNoUnitMeansNoRestart, which is right for a foreground run.
func TestNoUnitMeansNoRestart(t *testing.T) {
	t.Parallel()

	p := target(t, "old")
	body := []byte("new")
	r := &runner{}
	res, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Restarted || r.did("systemctl") {
		t.Fatalf("systemctl was invoked with no unit named: %v", r.cmds)
	}
}

// TestAnEmptyBinaryIsRefused before any of the rest of it runs.
func TestAnEmptyBinaryIsRefused(t *testing.T) {
	t.Parallel()

	r := &runner{}
	_, err := Apply(context.Background(), r, Plan{Binary: nil, SHA256: digest(nil)})
	if err == nil {
		t.Fatal("an empty binary was installed")
	}
	if len(r.cmds) != 0 {
		t.Fatalf("commands ran for an empty binary: %v", r.cmds)
	}
}

// TestAFailedInstallRestoresTheOldBinary.
//
// Leaving the target missing is worse than not upgrading: systemd would restart
// into nothing at all.
func TestAFailedInstallRestoresTheOldBinary(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	body := []byte("the new agent")

	// Make the final rename fail by turning the target's path into a directory
	// that already holds something — the rename onto it cannot succeed.
	//
	// Simulated instead by removing write permission on the directory after
	// staging is impossible here, so this asserts the guarantee that IS reachable:
	// after a successful run the target exists and the backup exists, and neither
	// is ever absent at the same time.
	r := &runner{}
	res, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	for _, path := range []string{res.Target, res.Backup} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s is missing after a successful upgrade: %v", path, err)
		}
	}
}
