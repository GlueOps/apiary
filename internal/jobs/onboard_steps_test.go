package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner records commands and can be told to fail a chosen one, so the
// ordering — which is the part that can lock the control plane out of a host —
// is testable without a host to lock itself out of.
type fakeRunner struct {
	cmds         []string
	failOn       string
	reconnects   int
	reconnectErr error
	uploads      []string
	uploadSizes  []int
}

func (f *fakeRunner) Run(_ context.Context, cmd string) (string, error) {
	f.cmds = append(f.cmds, cmd)
	if f.failOn != "" && strings.Contains(cmd, f.failOn) {
		return "boom", errors.New("exit 1")
	}
	return "", nil
}

func (f *fakeRunner) Upload(_ context.Context, path string, _ uint32, content []byte) error {
	f.uploads = append(f.uploads, path)
	f.uploadSizes = append(f.uploadSizes, len(content))
	return nil
}

func (f *fakeRunner) Reconnect(context.Context) error {
	f.reconnects++
	return f.reconnectErr
}

func (f *fakeRunner) indexOf(sub string) int {
	for i, c := range f.cmds {
		if strings.Contains(c, sub) {
			return i
		}
	}
	return -1
}

func plan() OnboardPlan {
	return OnboardPlan{
		PublicKey:  "ssh-ed25519 AAAAC3Nz apiary-fleet",
		FromClause: `from="192.168.10.60/32,194.36.65.187/32"`,
	}
}

// The whole safety argument, as an ordering assertion.
func TestInstallValidatesBeforeReloadingAndProvesAfter(t *testing.T) {
	r := &fakeRunner{}
	if err := InstallManagedKey(context.Background(), r, plan()); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	write := r.indexOf("authorized_keys.d/apiary")
	validate := r.indexOf("sshd -t")
	reload := r.indexOf("reload")

	if write >= validate || validate >= reload {
		t.Fatalf("wrong order — write=%d validate=%d reload=%d; validating after a reload is "+
			"validating a daemon that has already refused to come back", write, validate, reload)
	}
	if r.reconnects != 1 {
		t.Fatalf("reconnected %d times; without a fresh connection nothing about the new path "+
			"has been proved — this session authenticated before the change and survives a reload",
			r.reconnects)
	}
}

// A reload on a configuration sshd rejects leaves the daemon refusing every
// connection, and that is the one failure here with no way back.
func TestInstallDoesNotReloadWhenValidationFails(t *testing.T) {
	r := &fakeRunner{failOn: "sshd -t"}
	err := InstallManagedKey(context.Background(), r, plan())
	if err == nil {
		t.Fatal("a rejected configuration was accepted")
	}
	if r.indexOf("reload") != -1 {
		t.Fatal("reloaded sshd after it rejected the configuration")
	}
	if r.reconnects != 0 {
		t.Fatal("reconnected despite never reloading")
	}
}

// Reload keeps existing sessions; restart does not. Restarting would drop the
// connection performing the change, mid-change.
func TestInstallReloadsRatherThanRestarts(t *testing.T) {
	r := &fakeRunner{}
	_ = InstallManagedKey(context.Background(), r, plan())
	for _, c := range r.cmds {
		if strings.Contains(c, "restart") {
			t.Fatalf("restarts sshd: %q", c)
		}
	}
}

// The failure this exists to catch: sshd reloaded, but the managed path is not
// actually consulted. The old path still works, so access is not lost — but
// nothing has been proved and onboarding must not continue.
func TestInstallFailsWhenTheNewPathDoesNotAuthenticate(t *testing.T) {
	r := &fakeRunner{reconnectErr: errors.New("permission denied")}
	err := InstallManagedKey(context.Background(), r, plan())
	if err == nil {
		t.Fatal("install succeeded without the new path authenticating")
	}
	if !strings.Contains(err.Error(), "not lost") {
		t.Fatalf("the error does not say access survived, which is what decides whether to "+
			"panic: %v", err)
	}
}

// Writing an empty managed file removes the control plane's own access on the
// next reload, with nothing left to fix it over.
func TestInstallRefusesAnEmptyKey(t *testing.T) {
	r := &fakeRunner{}
	p := plan()
	p.PublicKey = "   "
	if err := InstallManagedKey(context.Background(), r, p); err == nil {
		t.Fatal("an empty key file was installed")
	}
	if len(r.cmds) != 0 {
		t.Fatalf("ran %d commands before refusing: %v", len(r.cmds), r.cmds)
	}
}

// The from= clause is derived from operator-supplied addresses and interpolated
// into a command running as root.
func TestInstallQuotesWhatItWrites(t *testing.T) {
	r := &fakeRunner{}
	p := plan()
	p.FromClause = `from="1.2.3.4"; rm -rf /; echo '`
	_ = InstallManagedKey(context.Background(), r, p)

	for _, c := range r.cmds {
		if strings.Contains(c, "rm -rf /;") && !strings.Contains(c, `'\''`) {
			t.Fatalf("interpolated unquoted: %q", c)
		}
	}
	if got := shellQuote(`a'b`); got != `'a'\''b'` {
		t.Fatalf("shellQuote(%q) = %q", `a'b`, got)
	}
}

// Written whole, never appended. sshd takes the FIRST value for a keyword, so a
// duplicated AuthorizedKeysFile would not error — it would silently win or lose
// depending on order, which is worse than failing.
func TestDropInIsWrittenWholeNotAppended(t *testing.T) {
	r := &fakeRunner{}
	_ = InstallManagedKey(context.Background(), r, plan())
	for _, c := range r.cmds {
		if strings.Contains(c, sshdDropIn) && strings.Contains(c, ">>") {
			t.Fatalf("appends to the drop-in: %q", c)
		}
	}
	if n := strings.Count(sshdDropInContent(false), "AuthorizedKeysFile"); n != 1 {
		t.Fatalf("the drop-in declares AuthorizedKeysFile %d times", n)
	}
	// The operator file must be listed, or the next reload removes every human's
	// access to the host.
	if !strings.Contains(sshdDropInContent(false), "/operators") {
		t.Fatal("the drop-in does not list the operator key file")
	}
}

func TestSshdDropInStaysASingleAuthorizedKeysFileLineWithAJoinPath(t *testing.T) {
	// sshd takes the FIRST value for a keyword, so a second AuthorizedKeysFile
	// line does not error — it silently wins or loses depending on order. The
	// join path therefore has to be part of the one line, not appended as another.
	withJoin := sshdDropInContent(true)
	if n := strings.Count(withJoin, "AuthorizedKeysFile"); n != 1 {
		t.Fatalf("want exactly one AuthorizedKeysFile line, got %d:\n%s", n, withJoin)
	}
	for _, want := range []string{"/operators", "/apiary", "/join"} {
		if !strings.Contains(withJoin, want) {
			t.Errorf("the join-enabled drop-in must list %s:\n%s", want, withJoin)
		}
	}
	if strings.Contains(sshdDropInContent(false), "authorized_keys.d/join\n") {
		t.Error("the plain drop-in must not list the join path")
	}
}

func TestSshdDropInIsRenderedWholeRatherThanMatchedOnASubstring(t *testing.T) {
	// The bug this guards, observed on a live host: an earlier version guarded
	// adding the join path with `grep -q /etc/ssh/authorized_keys.d/join`. The
	// host's config already listed `.../authorized_keys.d/join-probe`, `join` is
	// a substring of `join-probe`, so the guard matched, the path was never
	// added, and sshd never read the key file that had just been written. Every
	// step reported success and the join could not authenticate.
	//
	// Asserted as a property of the rendered content: the join path appears as a
	// complete path, and a longer path that merely starts with it is not what
	// satisfies the check.
	withJoin := sshdDropInContent(true)
	at := strings.Index(withJoin, "AuthorizedKeysFile")
	if at < 0 {
		t.Fatal("the drop-in names no AuthorizedKeysFile at all, so sshd would fall back to " +
			"its default and never read the join key")
	}
	fields := strings.Fields(withJoin[at:])
	found := false
	for _, f := range fields {
		if f == joinKeyFile {
			found = true
		}
	}
	if !found {
		t.Fatalf("the join path must appear as a whole field, not as a prefix of another:\n%s", withJoin)
	}
}
