package hostinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	cmds   []string
	failOn string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.cmds = append(f.cmds, cmd)
	if f.failOn != "" && strings.Contains(cmd, f.failOn) {
		return "boom", errors.New("exit 1")
	}
	return "", nil
}

// capture collects what would have been written, so the sequence can be asserted
// without root and without a systemd.
func capture(files map[string][]byte) writer {
	return func(path string, _ os.FileMode, content []byte) error {
		files[path] = content
		return nil
	}
}

const aUnit = "[Unit]\nDescription=x\n\n[Service]\nExecStart=/usr/local/bin/apiary-agent start\n\n[Install]\nWantedBy=multi-user.target\n"

// TestTheInstallEnablesAndStarts is the behaviour whose absence left a manually
// enrolled host unsupervised: enable without start is a host that is silent
// until someone power-cycles it, start without enable is one that works until it
// reboots — which is worse, because it looks fine for weeks.
func TestTheInstallEnablesAndStarts(t *testing.T) {
	t.Parallel()

	r := &fakeRunner{}
	files := map[string][]byte{}
	if err := installWith(context.Background(), r, capture(files),
		Plan{Unit: []byte(aUnit), BaseURL: "http://cp:8282", ExecPath: "/usr/sbin/apiary-agent"}); err != nil {
		t.Fatalf("install: %v", err)
	}

	joined := strings.Join(r.cmds, " | ")
	if !strings.Contains(joined, "daemon-reload") {
		t.Fatalf("no daemon-reload in %q: systemd would run the previous unit", joined)
	}
	if !strings.Contains(joined, "enable --now") {
		t.Fatalf("commands were %q, want `enable --now`: enabling alone never starts it, and "+
			"starting alone does not survive a reboot", joined)
	}
	if _, ok := files[UnitPath]; !ok {
		t.Fatalf("no unit written; wrote %v", keys(files))
	}
}

// TestTheUnitRunsTheBinaryThatIsActuallyHere.
//
// The shipped unit names /usr/local/bin because that is where the automated path
// puts the binary. A hand install puts it wherever the operator chose, and a
// unit naming a path that does not exist fails with status=203/EXEC — which
// reads as a permissions problem, not a wrong path.
func TestTheUnitRunsTheBinaryThatIsActuallyHere(t *testing.T) {
	t.Parallel()

	r := &fakeRunner{}
	files := map[string][]byte{}
	if err := installWith(context.Background(), r, capture(files),
		Plan{Unit: []byte(aUnit), ExecPath: "/usr/sbin/apiary-agent"}); err != nil {
		t.Fatalf("install: %v", err)
	}

	unit := string(files[UnitPath])
	if !strings.Contains(unit, "ExecStart=/usr/sbin/apiary-agent start") {
		t.Fatalf("unit has no ExecStart for the real path:\n%s", unit)
	}
	if strings.Contains(unit, "/usr/local/bin/apiary-agent") {
		t.Fatalf("the shipped path survived the rewrite:\n%s", unit)
	}
	// The argument must survive. Without `start` systemd runs the root command,
	// which prints help and exits 0 — reported as started and immediately dead,
	// with nothing in the journal saying why.
	if !strings.HasSuffix(strings.TrimSpace(execLine(unit)), " start") {
		t.Fatalf("the `start` argument was dropped: %q", execLine(unit))
	}
}

// TestTheEnvironmentFileCarriesTheControlPlane. The unit passes no --config, so
// without this the agent falls back to localhost:8080 and talks to the
// hypervisor itself.
func TestTheEnvironmentFileCarriesTheControlPlane(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{}
	if err := installWith(context.Background(), &fakeRunner{}, capture(files),
		Plan{Unit: []byte(aUnit), BaseURL: "http://10.0.0.1:8282", ExecPath: "/x"}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if got := string(files[EnvPath]); !strings.Contains(got, "AGENT_BASE_URL=http://10.0.0.1:8282") {
		t.Fatalf("env file is %q, want it to name the control plane", got)
	}
}

// TestAUnitWithNothingToRunIsRefused. Writing it would leave systemd with a
// service it cannot start and an operator with no idea why.
func TestAUnitWithNothingToRunIsRefused(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{}
	err := installWith(context.Background(), &fakeRunner{}, capture(files),
		Plan{Unit: []byte("[Unit]\nDescription=x\n"), ExecPath: "/x"})
	if err == nil {
		t.Fatal("a unit with no ExecStart was accepted")
	}
	if len(files) != 0 {
		t.Fatalf("wrote %v despite refusing", keys(files))
	}
}

// TestAFailedStartIsReportedRatherThanSwallowed, with systemd's own output —
// which is where the reason is.
func TestAFailedStartIsReportedRatherThanSwallowed(t *testing.T) {
	t.Parallel()

	err := installWith(context.Background(), &fakeRunner{failOn: "enable"}, capture(map[string][]byte{}),
		Plan{Unit: []byte(aUnit), ExecPath: "/x"})
	if err == nil {
		t.Fatal("a failing enable was reported as success")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error %q drops systemd's own output, which is where the reason is", err)
	}
}

// TestTheUnitIsWrittenAtomically. A crash mid-write leaves a truncated unit
// systemd refuses to parse, and that host does not come back after its next
// reboot.
func TestTheUnitIsWrittenAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "apiary-agent.service")
	if err := writeFile(target, 0o644, []byte(aUnit)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("the temporary file was left behind, so a later install sees a stale one")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != aUnit {
		t.Fatalf("content = %q (err %v)", got, err)
	}
}

func execLine(unit string) string {
	for _, l := range strings.Split(unit, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), execMarker) {
			return l
		}
	}
	return ""
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
