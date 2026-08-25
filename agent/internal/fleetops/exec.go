package fleetops

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner executes a command on the host. An interface so an executor's decision
// logic can be tested without running apt on the machine doing the testing.
type Runner interface {
	Run(ctx context.Context, log func(stream, format string, args ...any), name string, args ...string) error
}

type execRunner struct{}

// NewExecRunner runs commands for real.
func NewExecRunner() Runner { return execRunner{} }

// Run streams stdout and stderr line by line as the command produces them.
//
// Streaming rather than buffering is the point: a dist-upgrade takes minutes,
// and an operator watching a rotation needs to see it progressing. A command
// whose output only appears when it finishes is indistinguishable from one that
// has hung.
func (execRunner) Run(ctx context.Context, log func(stream, format string, args ...any), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(),
		// apt must never stop to ask a question: there is nobody at this
		// terminal, and a prompt would hang until the lease expired.
		"DEBIAN_FRONTEND=noninteractive",
		"APT_LISTCHANGES_FRONTEND=none",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}

	done := make(chan struct{}, 2)
	go func() { scan(stdout, "stdout", log); done <- struct{}{} }()
	go func() { scan(stderr, "stderr", log); done <- struct{}{} }()
	<-done
	<-done

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func scan(r interface{ Read([]byte) (int, error) }, stream string, log func(string, string, ...any)) {
	s := bufio.NewScanner(r)
	// apt can emit very long lines; the default 64KiB limit would truncate them
	// mid-sentence and the log would end abruptly for no visible reason.
	s.Buffer(make([]byte, 1<<16), 1<<20)
	for s.Scan() {
		log(stream, "%s", s.Text())
	}
}

// rebootRequired reports whether the distribution has asked for a reboot.
//
// Reading the marker rather than assuming: most upgrades do not need one, and
// rebooting a hypervisor that did not ask for it costs every guest on it a
// migration or an outage for nothing.
func rebootRequired() bool {
	for _, p := range []string{"/var/run/reboot-required", "/run/reboot-required"} {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
