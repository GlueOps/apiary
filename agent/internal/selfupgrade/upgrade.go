// Package selfupgrade replaces this agent's own binary with a newer one.
//
// Downloading something and then executing it is the most dangerous thing this
// agent does, and each step here is a specific way it goes wrong:
//
//   - the download is truncated, and a truncated binary is not a smaller
//     program — it either fails to start or, occasionally, starts;
//   - the binary is for the wrong architecture or is corrupt, and the service
//     enters a restart loop that reads as a crash in the new version;
//   - the file cannot be replaced at all, because Linux refuses to write over a
//     running executable. That is ETXTBSY, and it is why this renames rather
//     than writing in place — the same failure that silently defeats `scp` onto
//     a running agent.
//
// So: verify the digest against what the control plane advertised, stage beside
// the target on the SAME filesystem, prove the new binary runs, keep the old one,
// and only then rename.
package selfupgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Runner executes commands, so the sequence is testable without replacing
// anything real.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// Plan is one upgrade.
type Plan struct {
	// Binary is the downloaded executable.
	Binary []byte
	// SHA256 is what the control plane advertised, as "sha256:<hex>".
	SHA256 string
	// Target is the path to replace. Empty means this running binary.
	Target string

	// BackupDir is where the replaced binary is kept. Empty means BackupDir,
	// which is the convention; a caller sets it to put backups on a specific
	// volume, and the tests set it to avoid writing outside a temp directory.
	BackupDir string
	// Unit is the systemd unit to restart afterwards. Empty skips the restart,
	// which is right for a foreground run.
	Unit string
}

// Result reports what happened, including where the previous binary went.
type Result struct {
	Target    string
	Backup    string
	Restarted bool
}

// Apply installs a downloaded binary in place of this one.
func Apply(ctx context.Context, r Runner, p Plan) (Result, error) {
	var res Result

	if len(p.Binary) == 0 {
		return res, fmt.Errorf("upgrade: refusing to install an empty binary")
	}

	// VERIFIED BEFORE ANYTHING IS WRITTEN. This is the whole security boundary of
	// the operation: without it an upgrade is "run whatever arrived over the
	// network", and a truncated response is indistinguishable from a short
	// program.
	sum := sha256.Sum256(p.Binary)
	got := "sha256:" + hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, p.SHA256) {
		return res, fmt.Errorf("upgrade: the download does not match the digest the control "+
			"plane advertised (%s, wanted %s); refusing to install it", got, p.SHA256)
	}

	target := p.Target
	if target == "" {
		exe, err := os.Executable()
		if err != nil {
			return res, fmt.Errorf("upgrade: locate this binary: %w", err)
		}
		// Resolved, or an upgrade through a symlink replaces the LINK with a
		// binary and leaves the real file untouched.
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		target = exe
	}
	res.Target = target

	// Staged in the target's OWN directory, because rename is atomic only within
	// a filesystem. Staging in /tmp and renaming across a mount boundary fails
	// with EXDEV, and the usual fallback — copy then delete — is exactly the
	// non-atomic replacement this exists to avoid.
	dir := filepath.Dir(target)
	staged := filepath.Join(dir, ".apiary-agent.upgrade")
	if err := os.WriteFile(staged, p.Binary, 0o755); err != nil {
		return res, fmt.Errorf("upgrade: stage the new binary beside %s: %w", target, err)
	}
	defer func() { _ = os.Remove(staged) }()

	// PROVE IT RUNS before committing to it. A binary for the wrong architecture,
	// or one built against a libc this host does not have, fails here with a
	// clear message instead of as a service that will not start — which reads as
	// a crash in the new version and sends someone to the release notes.
	if out, err := r.Run(ctx, staged, "version"); err != nil {
		return res, fmt.Errorf("upgrade: the downloaded binary does not run on this host, so "+
			"nothing was replaced: %w (%s)", err, strings.TrimSpace(out))
	}

	// The old binary is kept. An upgrade with no way back is not an upgrade, and
	// this is the copy an operator reaches for when the new one starts and then
	// misbehaves.
	//
	// COPIED ASIDE, not renamed, and kept OUT of the binary's directory.
	//
	// The old scheme renamed target to target+".previous", which put a second
	// executable on PATH beside the real one and left the target MISSING between
	// the two renames — a window in which systemd restarting finds nothing at
	// all. It also gave the accumulating-suffix bug something to accumulate:
	// ExecStart was found five ".previous" deep in a lab.
	//
	// Copy first, then one atomic rename over the target. The live path always
	// holds a complete binary — the old one or the new one, never neither — and
	// the backup is a plain file in a backup directory, which is what it is.
	backup, err := keepBackup(target, p.BackupDir)
	if err != nil {
		return res, err
	}
	res.Backup = backup

	// Nothing to roll back on failure now: the target is untouched until this
	// rename succeeds, and if it fails the old binary is still in place and
	// still running.
	if err := os.Rename(staged, target); err != nil {
		return res, fmt.Errorf("upgrade: install the new binary: %w", err)
	}

	// Best effort, and deliberately after the swap: backups the old scheme left
	// beside the binary are dead weight on PATH. Failing an upgrade that has
	// already succeeded because a stale file could not be removed would be
	// choosing the wrong thing to care about.
	sweepLegacyBackups(target)

	if p.Unit == "" {
		return res, nil
	}
	// Restarted LAST, and by systemd rather than by exec'ing ourselves: this
	// process IS the old binary, and the supervisor is what knows how to start
	// the new one with the right environment, state directory and limits.
	if out, err := r.Run(ctx, "systemctl", "restart", p.Unit); err != nil {
		return res, fmt.Errorf("upgrade: the new binary is installed but %s did not restart: "+
			"%w (%s)", p.Unit, err, strings.TrimSpace(out))
	}
	res.Restarted = true
	return res, nil
}

// SelfDigest is the sha256 of the binary currently running, as "sha256:<hex>".
//
// Computed once and cached: it cannot change while this process lives, since
// replacing the file leaves the running image untouched — that is the whole
// reason an upgrade renames rather than writes.
//
// It exists so "does this host need upgrading" is decided on what is ACTUALLY ON
// DISK rather than on a version label. The two disagree in an ordinary
// development build, and when they do a version comparison never converges: the
// agent upgrades, restarts, reports the same version, and is judged out of date
// again.
func SelfDigest() (string, error) {
	selfDigestOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			selfDigestErr = fmt.Errorf("digest self: locate this binary: %w", err)
			return
		}
		if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
			exe = resolved
		}
		f, err := os.Open(exe)
		if err != nil {
			selfDigestErr = fmt.Errorf("digest self: open %s: %w", exe, err)
			return
		}
		defer func() { _ = f.Close() }()
		h := sha256.New()
		// Streamed rather than read whole: the binary is around twenty megabytes
		// and this runs on a hypervisor.
		if _, err := io.Copy(h, f); err != nil {
			selfDigestErr = fmt.Errorf("digest self: read %s: %w", exe, err)
			return
		}
		selfDigestVal = "sha256:" + hex.EncodeToString(h.Sum(nil))
	})
	return selfDigestVal, selfDigestErr
}

var (
	selfDigestOnce sync.Once
	selfDigestVal  string
	selfDigestErr  error
)

// BackupDir is where replaced binaries are kept.
//
// /var/backups is the Debian convention for exactly this, and the point is that
// it is NOT the binary's own directory: a backup beside the binary is a second
// executable on PATH that an operator can run by accident and that tab
// completion offers, and it is what the accumulating-suffix bug fed on.
//
// Not under /var/lib/apiary-agent either, which the unit's StateDirectory=
// manages — state systemd owns is state systemd may clear, and a rollback copy
// should outlive that.
const BackupDir = "/var/backups/apiary-agent"

// backupsKept is how many previous binaries survive.
//
// More than one because the version you want to go back to is not always the
// one you just left — an upgrade that limps for a day before anyone notices is
// the case that needs the one before it. Bounded because these are 12MB each
// and nothing else prunes them.
const backupsKept = 3

// keepBackup copies the current binary into BackupDir and returns its path.
//
// A COPY rather than a rename, because the two paths are routinely on different
// filesystems and rename fails across a mount boundary with EXDEV — and because
// the target must never stop existing, which a rename away from it guarantees
// for as long as the next rename takes.
func keepBackup(target, dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		dir = BackupDir
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("upgrade: make the backup directory %s: %w", dir, err)
	}
	src, err := os.Open(target)
	if err != nil {
		return "", fmt.Errorf("upgrade: read the binary being replaced: %w", err)
	}
	defer func() { _ = src.Close() }()

	// Named by content, not by time: Date.now-style names collide on a fast
	// retry and say nothing about what they hold, while a digest tells an
	// operator at a glance whether two backups are the same build.
	sum, err := digestOf(target)
	if err != nil {
		return "", fmt.Errorf("upgrade: identify the binary being replaced: %w", err)
	}
	name := filepath.Base(target) + "-" + strings.TrimPrefix(sum, "sha256:")[:12]
	dst := filepath.Join(dir, name)

	// Staged and renamed inside BackupDir so an interrupted copy cannot leave a
	// truncated file that looks like a usable rollback.
	tmp := dst + ".partial"
	// 0600, not 0700. A backup does not need to be executable where it lies —
	// restoring one means installing it over the target, which sets the mode
	// there — and an executable copy of the agent sitting in a backup directory
	// is a thing that can be run by accident.
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", fmt.Errorf("upgrade: stage the backup: %w", err)
	}
	if _, err := io.Copy(f, src); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upgrade: write the backup: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upgrade: flush the backup: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upgrade: close the backup: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("upgrade: keep the backup: %w", err)
	}

	pruneBackups(dir, filepath.Base(target))
	return dst, nil
}

// digestOf hashes a file on disk.
func digestOf(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// pruneBackups keeps the most recent backupsKept for one binary name.
//
// Best effort throughout: a rollback copy that could not be pruned is clutter,
// and failing an upgrade over clutter would be the wrong trade.
func pruneBackups(dir, base string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type aged struct {
		path string
		mod  time.Time
	}
	var mine []aged
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), base+"-") ||
			strings.HasSuffix(e.Name(), ".partial") {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil {
			continue
		}
		mine = append(mine, aged{filepath.Join(dir, e.Name()), info.ModTime()})
	}
	if len(mine) <= backupsKept {
		return
	}
	sort.Slice(mine, func(i, j int) bool { return mine[i].mod.After(mine[j].mod) })
	for _, old := range mine[backupsKept:] {
		_ = os.Remove(old.path)
	}
}

// sweepLegacyBackups removes the <binary>.previous chain the old scheme left in
// the binary's own directory.
//
// Best effort. These are what turned up as
// apiary-agent.previous.previous.previous.previous.previous — each one an
// executable on PATH, none of them ever pruned.
func sweepLegacyBackups(target string) {
	for path := target + ".previous"; ; path += ".previous" {
		if _, err := os.Stat(path); err != nil {
			return
		}
		if err := os.Remove(path); err != nil {
			return
		}
	}
}
