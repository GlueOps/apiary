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
		Target: p, BackupDir: filepath.Dir(p) + "/backups",
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
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
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
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
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
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
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
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups", Unit: "apiary-agent.service",
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
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
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
func TestAFailedInstallLeavesTheOldBinaryRunning(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	body := []byte("the new agent")

	// Fails at "prove it runs", which is the realistic failure: a binary for the
	// wrong architecture, or built against a libc this host does not have.
	r := &runner{failOn: ".apiary-agent.upgrade"}
	if _, err := Apply(context.Background(), r, Plan{
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
	}); err == nil {
		t.Fatal("a binary that does not run on this host was installed anyway")
	}

	// THE GUARANTEE, and it is now structural rather than restored by a rollback:
	// the target is not touched until the one rename that replaces it, so a
	// failure before that point cannot leave it missing or half-written. The old
	// scheme renamed the target AWAY first, which opened a window where systemd
	// restarting would have found nothing at all.
	now, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("the binary is gone after a failed upgrade: %v", err)
	}
	if string(now) != "the old agent" {
		t.Fatalf("the binary was replaced by one that does not run: %q", now)
	}
}

// Backups do not live beside the binary. One that does is a second executable
// on PATH — offered by tab completion, runnable by accident — and it is what the
// accumulating-suffix bug fed on.
func TestBackupsAreKeptOutOfTheBinarysDirectory(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	body := []byte("the new agent")
	backups := filepath.Dir(p) + "/backups"

	res, err := Apply(context.Background(), &runner{}, Plan{
		Binary: body, SHA256: digest(body), Target: p, BackupDir: backups,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if filepath.Dir(res.Backup) != backups {
		t.Fatalf("backup landed in %s, want %s", filepath.Dir(res.Backup), backups)
	}
	if _, err := os.Stat(p + ".previous"); err == nil {
		t.Fatal("a .previous backup was left beside the binary, on PATH")
	}
	old, err := os.ReadFile(res.Backup)
	if err != nil || string(old) != "the old agent" {
		t.Fatalf("the backup does not hold the replaced binary: %q (%v)", old, err)
	}
}

// A host damaged by the old scheme is cleaned up on its next upgrade, rather
// than needing to be visited. Found five deep in a lab.
func TestLegacyBackupsBesideTheBinaryAreSweptUp(t *testing.T) {
	t.Parallel()

	p := target(t, "the old agent")
	for _, suffix := range []string{".previous", ".previous.previous", ".previous.previous.previous"} {
		if err := os.WriteFile(p+suffix, []byte("an older agent"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	body := []byte("the new agent")
	if _, err := Apply(context.Background(), &runner{}, Plan{
		Binary: body, SHA256: digest(body), Target: p, BackupDir: filepath.Dir(p) + "/backups",
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	for _, suffix := range []string{".previous", ".previous.previous", ".previous.previous.previous"} {
		if _, err := os.Stat(p + suffix); err == nil {
			t.Errorf("%s survived the upgrade", p+suffix)
		}
	}
}
