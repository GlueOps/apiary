package fleetops

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RebootHost asks systemd to reboot, then waits to be stopped by it.
//
// `systemctl reboot` rather than the reboot(2) syscall so that services stop in
// order — on a hypervisor that means guests are shut down or suspended by their
// own units rather than losing power underneath them.
//
// WAITS, because `systemctl reboot` returns as soon as the job is queued. It
// used to return nil right then, and the caller finished the task as done
// before the host went down — reporting success for a reboot nobody had seen,
// with the check after it never run. Now the shutdown stopping the agent is the
// normal way out: ctx is cancelled and that is returned, which bee reads as
// "resume after restart".
//
// nil means the host is still up rebootWait later. The caller then checks the
// boot id, which will say the reboot did not happen.
func RebootHost(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "reboot")
	if out, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("systemctl reboot: %w: %s", err, strings.TrimSpace(string(out)))
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(rebootWait):
		return nil
	}
}

// rebootWait is how long a requested reboot may take to stop this agent. A
// shutdown that stops guests in order can be slow; ten minutes without it is a
// reboot that is not coming.
var rebootWait = 10 * time.Minute
