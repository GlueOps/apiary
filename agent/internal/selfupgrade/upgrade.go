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
	"os"
	"path/filepath"
	"strings"
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
	backup := target + ".previous"
	if err := os.Rename(target, backup); err != nil {
		return res, fmt.Errorf("upgrade: set the current binary aside: %w", err)
	}
	res.Backup = backup

	if err := os.Rename(staged, target); err != nil {
		// Put it back. Leaving the target missing is worse than not upgrading:
		// systemd would restart into nothing at all.
		if rbErr := os.Rename(backup, target); rbErr != nil {
			return res, fmt.Errorf("upgrade: could not install the new binary (%w) AND could "+
				"not restore the old one (%v); %s holds the previous agent", err, rbErr, backup)
		}
		return res, fmt.Errorf("upgrade: install the new binary: %w", err)
	}

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
