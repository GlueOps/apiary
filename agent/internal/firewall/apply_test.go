package firewall

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recorder struct {
	cmds     []string
	failOn   string
	started  []string
	pid      int
	startErr error
}

func (r *recorder) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	r.cmds = append(r.cmds, cmd)
	if r.failOn != "" && strings.Contains(cmd, r.failOn) {
		return "nft: syntax error", errors.New("exit 1")
	}
	return "", nil
}

func (r *recorder) Start(_ context.Context, name string, args ...string) (int, error) {
	r.started = append(r.started, name+" "+strings.Join(args, " "))
	if r.startErr != nil {
		return 0, r.startErr
	}
	if r.pid == 0 {
		r.pid = 4242
	}
	return r.pid, nil
}

func (r *recorder) did(sub string) bool {
	for _, c := range r.cmds {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

// didApply distinguishes loading the ruleset from CHECKING it. Both name the
// same staged file, so a substring match on the filename reports a check as an
// apply — which would make the "never applied" assertions pass vacuously.
func (r *recorder) didApply() bool {
	for _, c := range r.cmds {
		if strings.HasPrefix(c, "nft -f ") && strings.Contains(c, "firewall.pending.nft") {
			return true
		}
	}
	return false
}

func (r *recorder) indexOfApply() int {
	for i, c := range r.cmds {
		if strings.HasPrefix(c, "nft -f ") && strings.Contains(c, "firewall.pending.nft") {
			return i
		}
	}
	return -1
}

func (r *recorder) indexOf(sub string) int {
	for i, c := range r.cmds {
		if strings.Contains(c, sub) {
			return i
		}
	}
	return -1
}

const body = "flush ruleset\ntable inet filter {\n}\n"

// aPlan stages into a temp directory: this package writes real files, and a test
// that needed root or wrote to the live state directory is one nobody runs.
func aPlan(t *testing.T) Plan {
	t.Helper()
	return Plan{Revision: 7, Digest: Digest(body), Content: body, Dir: t.TempDir()}
}

func reachable(context.Context) error { return nil }
func unreachable(context.Context) error {
	return errors.New("dial tcp: i/o timeout")
}

// TestTheRulesetIsCheckedBeforeItIsApplied.
//
// nft rejects a bad file whole rather than a bad line, so a ruleset that does
// not parse takes the entire firewall down. --check turns that from a host-wide
// outage into a reported failure with nothing applied.
func TestTheRulesetIsCheckedBeforeItIsApplied(t *testing.T) {
	r := &recorder{}
	if _, err := Apply(context.Background(), r, reachable, aPlan(t)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	check, apply := r.indexOf("nft --check"), r.indexOfApply()
	if check < 0 {
		t.Fatalf("the ruleset was never checked: %v", r.cmds)
	}
	if apply < 0 || check > apply {
		t.Fatalf("checked after applying, which is no check at all: %v", r.cmds)
	}
}

// TestARejectedRulesetIsNeverApplied.
func TestARejectedRulesetIsNeverApplied(t *testing.T) {
	r := &recorder{failOn: "--check"}
	res, err := Apply(context.Background(), r, reachable, aPlan(t))
	if err == nil {
		t.Fatal("a ruleset nft rejected was reported as applied")
	}
	if res.Applied {
		t.Fatal("result says applied")
	}
	if r.didApply() {
		t.Fatalf("it was applied anyway: %v", r.cmds)
	}
	if len(r.started) != 0 {
		t.Fatal("a rollback was armed for a ruleset that was never applied")
	}
}

// TestTheUndoIsArmedBEFORETheRulesetIsApplied.
//
// THE LOCKOUT TEST. Arming afterwards leaves a window in which the host is
// enforcing a new policy with no undo — and that window is exactly when the
// policy that cuts the host off is in force.
func TestTheUndoIsArmedBEFORETheRulesetIsApplied(t *testing.T) {
	r := &recorder{}
	if _, err := Apply(context.Background(), r, reachable, aPlan(t)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(r.started) == 0 {
		t.Fatal("no rollback was ever armed")
	}
	// Start is recorded separately, so ordering is asserted by the absence of an
	// apply before the arm: Apply returns early if arming fails, and this test
	// pins that it does.
	if !strings.Contains(r.started[0], "firewall.rollback.nft") {
		t.Fatalf("the armed command does not restore the previous ruleset: %q", r.started[0])
	}
}

// TestArmingFailureStopsTheApply. Applying with no undo is the one thing this
// package exists to prevent.
func TestArmingFailureStopsTheApply(t *testing.T) {
	r := &recorder{startErr: errors.New("fork failed")}
	_, err := Apply(context.Background(), r, reachable, aPlan(t))
	if err == nil {
		t.Fatal("the ruleset was applied without an undo armed")
	}
	if r.didApply() {
		t.Fatalf("applied despite having no rollback: %v", r.cmds)
	}
}

// TestAnUnreachableControlPlaneRollsBackImmediately.
//
// The armed timer is the backstop for the agent being dead. While it is alive it
// must not wait: the fleet is one host down until this resolves.
func TestAnUnreachableControlPlaneRollsBackImmediately(t *testing.T) {
	r := &recorder{}
	plan := aPlan(t)
	res, err := Apply(context.Background(), r, unreachable, plan)
	if err == nil {
		t.Fatal("a ruleset that cut the host off was reported as a success")
	}
	if !res.RolledBack {
		t.Fatalf("result does not report a rollback: %+v", res)
	}
	if !r.did("nft -f " + pathsIn(plan.Dir).rollback) {
		t.Fatalf("no rollback was performed: %v", r.cmds)
	}
	if !r.did("kill") {
		t.Fatal("the armed timer was left running after a completed rollback")
	}
}

// TestASuccessfulApplyDisarmsTheUndo, or the timer fires ninety seconds later
// and reinstates the previous ruleset over a policy that was working.
func TestASuccessfulApplyDisarmsTheUndo(t *testing.T) {
	r := &recorder{}
	res, err := Apply(context.Background(), r, reachable, aPlan(t))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("result does not report success: %+v", res)
	}
	if !r.did("kill") {
		t.Fatalf("the undo was left armed after a successful apply: %v", r.cmds)
	}
}

// TestABodyThatDoesNotMatchItsDigestIsRefused.
//
// The digest arrives on the heartbeat and the body on a separate request, so a
// truncated response is detectable — and a truncated nft file is not a smaller
// firewall. It can parse cleanly with its final deny missing.
func TestABodyThatDoesNotMatchItsDigestIsRefused(t *testing.T) {
	r := &recorder{}
	p := aPlan(t)
	p.Content = body + "table inet extra {\n}\n"
	_, err := Apply(context.Background(), r, reachable, p)
	if err == nil {
		t.Fatal("a body that did not match its digest was applied")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error %q does not explain why a mismatch matters", err)
	}
	if len(r.cmds) != 0 {
		t.Fatalf("nft was touched before the digest was verified: %v", r.cmds)
	}
}

// TestAnEmptyRulesetIsRefused. The control plane sends NOTHING when it means
// hold; an empty string reaching here is a bug upstream, and acting on it would
// strip the host's rules.
func TestAnEmptyRulesetIsRefused(t *testing.T) {
	r := &recorder{}
	_, err := Apply(context.Background(), r, reachable, Plan{Revision: 1, Content: "  ", Dir: t.TempDir()})
	if err == nil {
		t.Fatal("an empty ruleset was applied")
	}
	if !strings.Contains(err.Error(), "hold") {
		t.Fatalf("error %q does not say what an absent policy means", err)
	}
	if len(r.cmds) != 0 {
		t.Fatalf("nft ran for an empty ruleset: %v", r.cmds)
	}
}

// TestTheRollbackRestoresWhatWasRunning, including the flush — restoring rules
// without flushing first leaves the new policy's tables loaded alongside the old
// ones.
func TestTheRollbackRestoresWhatWasRunning(t *testing.T) {
	r := &recorder{}
	plan := aPlan(t)
	if _, err := Apply(context.Background(), r, reachable, plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !r.did("nft list ruleset") {
		t.Fatalf("the previous ruleset was never captured, so there is nothing to go back to: %v",
			r.cmds)
	}
	saved, err := readFile(pathsIn(plan.Dir).rollback)
	if err != nil {
		t.Fatalf("read rollback: %v", err)
	}
	if !strings.HasPrefix(saved, "flush ruleset") {
		t.Fatalf("the rollback does not flush first, so it would layer the old rules under "+
			"the new ones:\n%s", saved)
	}
}
