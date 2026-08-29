package fleetops

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// RebootHost asks systemd to reboot.
//
// `systemctl reboot` rather than the reboot(2) syscall so that services stop in
// order — on a hypervisor that means guests are shut down or suspended by their
// own units rather than losing power underneath them.
//
// It does not wait: the process is about to be killed by the very thing it
// asked for, and treating that as an error would report a successful reboot as
// a failure.
func RebootHost(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "reboot")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl reboot: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
