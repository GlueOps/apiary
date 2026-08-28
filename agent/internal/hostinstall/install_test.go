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

// The unit must not sandbox away the agent's primary job.
//
// This agent exists to run `apt-get dist-upgrade` on a Proxmox host. Three
// directives break that, each failing at the NEXT step so the previous fix looks
// like progress: without CAP_SETUID apt cannot drop to _apt and `apt-get update`
// exits 100; ProtectHome makes /root read-only and the proxmox-kernel postinst
// cannot write /root/.ssh; RestrictNamespaces blocks the unshare(2) that
// proxmox-boot-sync uses, leaving initramfs-tools unconfigured.
//
// Measured against a live node, not reasoned about. This test is here so the
// next person tightening the unit has to delete an explanation rather than
// silently re-break a dist-upgrade that only fails on hosts they do not own.
func TestTheUnitDoesNotSandboxAwayDistUpgrade(t *testing.T) {
	// Directives only, never a substring of the whole file — the unit EXPLAINS
	// which options are excluded and why, so a naive Contains matches the very
	// comment that documents the exclusion.
	directives := map[string]bool{}
	for _, line := range strings.Split(string(Unit), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		directives[line] = true
	}
	has := func(d string) bool { return directives[d] }

	for _, forbidden := range []string{
		"ProtectHome=yes",
		"RestrictNamespaces=yes",
		"ProtectSystem=strict",
		"ProtectKernelModules=yes",
	} {
		if has(forbidden) {
			t.Errorf("%s breaks apt dist-upgrade on Proxmox; see the comment above it in the unit", forbidden)
		}
	}

	// apt drops to uid 42 for its download methods and cannot without this.
	if !has("AmbientCapabilities=CAP_SETUID CAP_SETGID") {
		t.Error("the unit must grant CAP_SETUID, or apt-get update fails before reading a package")
	}

	// The narrowing that costs nothing must stay — this is not licence to drop
	// hardening wholesale.
	for _, required := range []string{
		"NoNewPrivileges=yes",
		"RestrictSUIDSGID=yes",
		"LockPersonality=yes",
	} {
		if !has(required) {
			t.Errorf("%s costs this agent nothing and must stay", required)
		}
	}
}

// THE FIVE-DEEP FAULT. RefreshUnit resolved its own path with os.Executable(),
// which on Linux reads /proc/self/exe and follows the INODE — so once Apply had
// renamed the running binary to <path>.previous, that is what the unit was
// rewritten to point at. The next start ran the backup, the next upgrade renamed
// THAT, and the suffixes accumulated.
//
// Found in the lab as apiary-agent.previous.previous.previous.previous.previous:
// five upgrades, five reported successes, and an agent still on the build it
// started with.
func TestCanonicalExecPathStripsAccumulatedBackups(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/usr/sbin/apiary-agent", "/usr/sbin/apiary-agent"},
		{"/usr/sbin/apiary-agent.previous", "/usr/sbin/apiary-agent"},
		{"/usr/sbin/apiary-agent.previous.previous.previous.previous.previous",
			"/usr/sbin/apiary-agent"},
		{"/usr/local/bin/apiary-agent", "/usr/local/bin/apiary-agent"},
		// A path that merely CONTAINS the word is not a backup of anything.
		{"/opt/previous/apiary-agent", "/opt/previous/apiary-agent"},
	}
	for _, c := range cases {
		if got := CanonicalExecPath(c.in); got != c.want {
			t.Errorf("CanonicalExecPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The unit is what systemd actually runs, so it is what an upgrade must replace.
func TestUnitExecPathReadsWhatSystemdWouldRun(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{"plain", "ExecStart=/usr/sbin/apiary-agent start", "/usr/sbin/apiary-agent"},
		{"no args", "ExecStart=/usr/sbin/apiary-agent", "/usr/sbin/apiary-agent"},
		// systemd's prefix modifiers are not part of the path.
		{"ignore-errors prefix", "ExecStart=-/usr/sbin/apiary-agent start", "/usr/sbin/apiary-agent"},
		{"indented", "  ExecStart=/opt/apiary/agent start", "/opt/apiary/agent"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "apiary-agent.service")
			body := "[Unit]\nDescription=x\n\n[Service]\n" + c.line + "\n"
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := unitExecPathFrom(path)
			if err != nil {
				t.Fatalf("unitExecPathFrom: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
