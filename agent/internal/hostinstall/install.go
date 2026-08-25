// Package hostinstall turns an unpacked binary into a running system service.
//
// It exists because the two onboarding paths did not produce the same host. The
// automated path uploads the binary, the unit and the environment file, reloads
// systemd and enables the service; the manual path — copy the binary, run
// `apiary-agent enrol` — obtained a credential and stopped. The host had an
// identity and no supervision: no unit, nothing started, no watchdog, and
// nothing to bring the agent back after a reboot. It heartbeated once, from the
// operator's shell, and then looked exactly like a host nobody had onboarded.
package hostinstall

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Paths systemd and the unit already agree on. Named here rather than inlined so
// the install and the unit cannot drift apart silently.
const (
	UnitPath  = "/etc/systemd/system/apiary-agent.service"
	ConfigDir = "/etc/apiary-agent"
	EnvPath   = ConfigDir + "/agent.env"
	// StateDir is named for the record even though nothing here creates it: it is
	// what the unit's StateDirectory= resolves to, and what the agent's default
	// AGENT_STATE_PATH sits inside.
	StateDir   = "/var/lib/apiary-agent"
	UnitName   = "apiary-agent.service"
	execMarker = "ExecStart="
)

// Plan is what to install.
type Plan struct {
	// Unit is the service file, before ExecStart is pointed at this binary.
	Unit []byte

	// BaseURL goes into the environment file as AGENT_BASE_URL.
	//
	// The unit passes no --config, so the agent reads AGENT_-prefixed environment
	// and nothing else. Without this it falls back to its compiled default of
	// localhost:8080 and talks to the hypervisor itself, which fails as a
	// connection refused that mentions neither enrolment nor configuration.
	BaseURL string

	// ExecPath is the binary the unit should run. Empty means "the binary running
	// this command", which is almost always right and is what makes the install
	// work wherever the operator put it.
	ExecPath string
}

// Runner executes the commands the install needs. An interface so the sequence
// is testable without systemd, and without root.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// Install writes the unit and environment file and starts the service.
//
// Idempotent throughout. Re-running it is the documented way to repoint an agent
// at a different control plane or to repair a host whose unit was lost, and an
// install that only worked on a pristine machine would be useless for both.
func Install(ctx context.Context, p Plan) error {
	return installWith(ctx, execRunner{}, writeFile, p)
}

type writer func(path string, mode os.FileMode, content []byte) error

func writeFile(path string, mode os.FileMode, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	// Written beside the target and renamed, not written in place. A crash
	// halfway through leaves the previous unit intact rather than a truncated one
	// systemd will refuse to parse — and a truncated unit is a host that does not
	// come back after its next reboot.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, mode); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("install %s: %w", path, err)
	}
	return nil
}

func installWith(ctx context.Context, r Runner, w writer, p Plan) error {
	if len(p.Unit) == 0 {
		return fmt.Errorf("install: no unit to write")
	}

	execPath := p.ExecPath
	if execPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("install: locate this binary: %w", err)
		}
		// Resolved, because a unit pointing at a symlink breaks the moment the
		// link is replaced — and /proc/self/exe style paths are meaningless to
		// systemd.
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		execPath = exe
	}

	unit, err := pointExecStartAt(p.Unit, execPath)
	if err != nil {
		return err
	}

	// The state directory is deliberately NOT created here. systemd creates it
	// from the unit's StateDirectory=, and store.Open creates it too — a third
	// creation would only differ from those two when it was wrong.
	if err := w(UnitPath, 0o644, unit); err != nil {
		return err
	}
	if p.BaseURL != "" {
		// 0600: it names the control plane and may later hold tuning an operator
		// would rather not have every process on the host read.
		env := fmt.Appendf(nil, "AGENT_BASE_URL=%s\n", p.BaseURL)
		if err := w(EnvPath, 0o600, env); err != nil {
			return err
		}
	}

	if out, err := r.Run(ctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("install: systemctl daemon-reload: %w (%s)", err, strings.TrimSpace(out))
	}
	// enable --now, so the service both starts and survives a reboot. Enabling
	// without starting leaves a host that is onboarded and silent until someone
	// power-cycles it; starting without enabling leaves one that works until it
	// reboots, which is worse because it looks fine for weeks.
	if out, err := r.Run(ctx, "systemctl", "enable", "--now", UnitName); err != nil {
		return fmt.Errorf("install: systemctl enable --now %s: %w (%s)",
			UnitName, err, strings.TrimSpace(out))
	}
	return nil
}

// pointExecStartAt rewrites the unit's ExecStart to run this binary.
//
// The shipped unit says /usr/local/bin/apiary-agent because that is where the
// automated path puts it. A hand install puts the binary wherever the operator
// chose — /usr/sbin, /opt, a home directory during a trial — and a unit naming a
// path that does not exist fails at start with status=203/EXEC, which reads as a
// permissions problem rather than a wrong path.
//
// Rewritten rather than requiring the binary be moved: the install should work on
// the host as it is, not demand the host be rearranged first.
func pointExecStartAt(unit []byte, execPath string) ([]byte, error) {
	lines := bytes.Split(unit, []byte("\n"))
	found := false
	for i, line := range lines {
		if !bytes.HasPrefix(bytes.TrimSpace(line), []byte(execMarker)) {
			continue
		}
		// Everything after the binary is preserved — the shipped unit passes
		// `start`, and dropping the argument would leave systemd running the root
		// command, which prints help and exits 0. systemd would then report the
		// service as started and immediately dead, with nothing in the journal
		// that says why.
		rest := ""
		if fields := strings.Fields(string(line)); len(fields) > 1 {
			rest = " " + strings.Join(fields[1:], " ")
		}
		lines[i] = fmt.Appendf(nil, "%s%s%s", execMarker, execPath, rest)
		found = true
	}
	if !found {
		return nil, fmt.Errorf("install: the unit has no %s line, so there is nothing to run",
			execMarker)
	}
	return bytes.Join(lines, []byte("\n")), nil
}
