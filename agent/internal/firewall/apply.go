// Package firewall loads a rendered nft ruleset onto this host, and undoes it if
// doing so cut the host off.
//
// Applying a firewall to a machine you are administering over the network is the
// one operation that can destroy the means of fixing it. Every step here exists
// because of a specific way that happens:
//
//   - a ruleset that does not parse takes the WHOLE firewall down, because nft
//     rejects the file rather than the line;
//   - a ruleset that parses but denies the control plane leaves a host that is
//     healthy, running, and permanently unreachable;
//   - an agent that dies between applying and checking leaves the second case
//     with nobody to notice.
//
// So: check before applying, keep the previous ruleset, arm an undo that
// survives this process dying, apply, prove the control plane is still reachable,
// and only then disarm.
package firewall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Paths under the agent's own state directory, which systemd creates and only
// root can traverse.
const defaultStateDir = "/var/lib/apiary-agent"

// paths derives the three files this package writes.
//
// Derived from a directory rather than hard-coded so the sequence can be tested
// end to end against a temp directory. A test that had to be root, or that wrote
// to the real state directory, would be a test nobody runs — and this is the one
// package where an untested path can leave a machine unreachable.
type paths struct{ pending, rollback, pid string }

func pathsIn(dir string) paths {
	if dir == "" {
		dir = defaultStateDir
	}
	return paths{
		pending:  filepath.Join(dir, "firewall.pending.nft"),
		rollback: filepath.Join(dir, "firewall.rollback.nft"),
		pid:      filepath.Join(dir, "firewall.rollback.pid"),
	}
}

// RollbackAfter is how long the undo waits before firing.
//
// Long enough for the apply and one confirmation round trip to the control
// plane, short enough that a host which did cut itself off is back within a
// minute rather than at the end of a shift. It is a ceiling, not a delay: a
// successful confirm disarms it immediately.
const RollbackAfter = 90 * time.Second

// Runner executes commands. An interface so the whole sequence is exercised in
// tests without nft, without root, and without the ability to break the machine
// running them.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
	// Start launches a DETACHED process and returns its pid. The undo has to
	// outlive this agent: the case it exists for includes the agent dying between
	// applying a ruleset and discovering that the ruleset was the problem.
	Start(ctx context.Context, name string, args ...string) (int, error)
}

// Confirm reports whether the control plane is still reachable.
//
// This is the actual test of "did I just cut myself off", and it must be a real
// request over the network rather than a ping or a route check: the rule that
// locks a host out is usually a specific deny, and a check that does not traverse
// the same path as the agent's own traffic will pass while the agent cannot talk.
type Confirm func(ctx context.Context) error

// Plan is one convergence.
type Plan struct {
	Revision int
	Digest   string
	Content  string

	// Dir is where the staged ruleset, the rollback and its pid are written.
	// Empty means the agent's own state directory.
	Dir string
}

// Result says what happened, in the terms the control plane is told.
type Result struct {
	Applied    bool
	RolledBack bool
	// Reason is empty on success and otherwise says what stopped it, in words
	// intended for an operator reading a fleet listing rather than a log.
	Reason string
}

// Apply loads a ruleset, and undoes it if the host stops being able to talk.
func Apply(ctx context.Context, r Runner, c Confirm, p Plan) (Result, error) {
	if strings.TrimSpace(p.Content) == "" {
		// An empty body is never an instruction to flush. The control plane sends
		// nothing at all when it means "hold", and an empty string reaching here is
		// a bug somewhere upstream — acting on it would strip the host's rules.
		return Result{}, fmt.Errorf("firewall: refusing to apply an empty ruleset; " +
			"an absent policy means hold what is running, never flush it")
	}

	// The digest is verified against the BODY before anything touches nft. The
	// control plane advertises a digest on the heartbeat and the body arrives on a
	// separate request, so a truncated or swapped response is detectable — and a
	// truncated nft file is not a smaller firewall, it is a file that either fails
	// to parse or, worse, parses with its final deny missing.
	if got := Digest(p.Content); got != p.Digest {
		return Result{}, fmt.Errorf("firewall: the ruleset does not match the digest the "+
			"control plane advertised (%s, wanted %s); refusing to apply a body that may be "+
			"truncated, since a truncated ruleset can parse cleanly with its last rule missing",
			got, p.Digest)
	}

	dir := p.Dir
	if dir == "" {
		dir = defaultStateDir
	}
	at := pathsIn(dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Result{}, fmt.Errorf("firewall: create %s: %w", dir, err)
	}
	if err := os.WriteFile(at.pending, []byte(p.Content), 0o600); err != nil {
		return Result{}, fmt.Errorf("firewall: stage the ruleset: %w", err)
	}

	// CHECK BEFORE APPLYING. nft rejects a bad file whole, so a ruleset that does
	// not parse does not partially load — but --check tells us that without the
	// host spending any time in an unknown state, and turns a fleet-wide outage
	// into a reported failure on one host.
	if out, err := r.Run(ctx, "nft", "--check", "-f", at.pending); err != nil {
		return Result{Reason: "the ruleset was rejected by nft"},
			fmt.Errorf("firewall: nft refused the ruleset, so nothing was applied: %w (%s)",
				err, strings.TrimSpace(out))
	}

	// Capture what is running, so there is something to go back to. Its absence is
	// not fatal: a host with no ruleset at all is the ordinary starting state, and
	// the rollback then correctly restores "nothing".
	current, err := r.Run(ctx, "nft", "list", "ruleset")
	if err != nil {
		current = ""
	}
	rollbackBody := "flush ruleset\n" + current
	if err := os.WriteFile(at.rollback, []byte(rollbackBody), 0o600); err != nil {
		return Result{}, fmt.Errorf("firewall: stage the rollback: %w", err)
	}

	// ARM THE UNDO BEFORE APPLYING, and detached, so it survives this process
	// being killed — which is one of the ways the host ends up unreachable with
	// nobody watching.
	pid, err := r.Start(ctx, "sh", "-c", fmt.Sprintf(
		"sleep %d; nft -f %s", int(RollbackAfter.Seconds()), at.rollback))
	if err != nil {
		return Result{}, fmt.Errorf("firewall: could not arm the rollback, so the ruleset was "+
			"NOT applied: %w", err)
	}
	_ = os.WriteFile(at.pid, []byte(strconv.Itoa(pid)), 0o600)

	if out, err := r.Run(ctx, "nft", "-f", at.pending); err != nil {
		// It failed to load, so nothing changed and the undo has nothing to do.
		disarm(ctx, r, at, pid)
		return Result{Reason: "the ruleset failed to load"},
			fmt.Errorf("firewall: applying the ruleset failed: %w (%s)", err, strings.TrimSpace(out))
	}

	// THE CONFIRM. Everything above proves the file was valid; only this proves
	// the host can still be reached, which is the property that actually matters.
	if err := c(ctx); err != nil {
		// Do not wait for the timer. The operator's fleet is one host down until
		// this resolves, and the timer's only job is covering the case where we are
		// not here to do it.
		if out, rbErr := r.Run(ctx, "nft", "-f", at.rollback); rbErr != nil {
			// Left armed on purpose: the immediate rollback failed, and the timer is
			// now the only thing that will restore this host.
			return Result{Reason: "unreachable after applying, and the rollback failed"},
				fmt.Errorf("firewall: the control plane became unreachable after applying "+
					"(%w) and the immediate rollback also failed (%v: %s); the armed rollback "+
					"fires in under %s", err, rbErr, strings.TrimSpace(out), RollbackAfter)
		}
		disarm(ctx, r, at, pid)
		return Result{RolledBack: true, Reason: "the control plane was unreachable after applying"},
			fmt.Errorf("firewall: revision %d made this host unreachable and was rolled back: %w",
				p.Revision, err)
	}

	disarm(ctx, r, at, pid)
	return Result{Applied: true}, nil
}

// disarm cancels the pending undo.
//
// Best effort throughout. A rollback that fires after a successful apply
// reinstates the previous ruleset, which is wrong but recoverable on the next
// convergence; failing the apply because the timer could not be killed would be
// worse, because it would report an applied policy as not applied.
func disarm(ctx context.Context, r Runner, at paths, pid int) {
	if pid > 0 {
		_, _ = r.Run(ctx, "kill", "-TERM", strconv.Itoa(-pid))
		_, _ = r.Run(ctx, "kill", "-TERM", strconv.Itoa(pid))
	}
	_ = os.Remove(at.pid)
}

// Digest is the same form the control plane stores, so the two are comparable
// without either side reformatting.
func Digest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// OSRunner runs commands for real.
type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// Start launches a detached process in its OWN process group.
//
// The group is what makes the undo survivable in both directions: it is not
// killed when this agent's group is signalled, and disarm can take the whole
// group down with one negative pid rather than leaving an orphaned `sleep` that
// fires minutes later against a host that has since converged.
func (OSRunner) Start(_ context.Context, name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = detachedAttr()
	// No context: cancelling it would kill the rollback, and the rollback exists
	// for the case where this process is gone.
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	// Reaped in the background so a disarmed rollback does not linger as a zombie
	// for the life of the agent.
	go func() { _ = cmd.Wait() }()
	return pid, nil
}

// StateDir is exported so the agent can report where it staged things.
func StateDir() string { return filepath.Clean(defaultStateDir) }
