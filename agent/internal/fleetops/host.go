package fleetops

import (
	"context"
	"fmt"
	"os"
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

// JoinPasswordFromFile reads the cluster join password placed on the host.
//
// It is on disk rather than in the task spec because a task spec is stored as
// jsonb in the control database and kept after the task finishes; a cluster root
// password there would be a standing credential in a table that platform staff
// and anything with database access can read. On the host it is one file, on one
// machine, that provisioning already had to write.
//
// The mode check is not decoration: a world-readable cluster password is a
// cluster password every process on the hypervisor has.
func JoinPasswordFromFile(path string) func() (string, error) {
	return func() (string, error) {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			return "", fmt.Errorf("%s is mode %#o; a cluster join password must not be readable "+
				"by anything but root (chmod 600)", path, mode)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		pass := strings.TrimSpace(string(b))
		if pass == "" {
			return "", fmt.Errorf("%s is empty", path)
		}
		return pass, nil
	}
}
