package agent

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/glueops/apiary-agent/internal/bee"
	"github.com/glueops/apiary-agent/internal/hostinstall"
	"github.com/glueops/apiary-agent/internal/selfupgrade"
	"github.com/glueops/apiary-agent/internal/store"
)

// restartDelay is how long the detached restart waits.
//
// Long enough for this executor to return, for the bee to record the task as
// succeeded, and for the report loop to ship that — see upgradeTask for why
// that ordering is the whole difficulty of this task.
const restartDelay = 8 * time.Second

// upgradeTask replaces this agent's binary in response to a fleet task.
//
// THE TASK'S SUCCESS KILLS THE PROCESS REPORTING IT. Restarting inline would
// terminate the agent between finishing the work and shipping the outcome, and
// the control plane would show the task as running against a host that had
// already done it — until the agent came back, re-collected the same task, and
// did it again.
//
// So the restart is DETACHED AND DELAYED: the executor returns, the bee records
// the terminal status locally, the report loop ships it, and only then does the
// unit restart. The local record is what makes this safe rather than merely
// likely — a terminal status survives in the store, so an agent killed before
// the report lands reports it on restart, which is exactly the idempotency the
// task endpoint already documents.
func upgradeTask(cfg Config) bee.Executor {
	return func(ctx context.Context, _ store.Task, e *bee.Env) error {
		e.Phase("checking")

		// Read fresh rather than captured at startup: an agent that re-enrolled
		// since then holds a different credential, and the one this closure was
		// built with would 401 on the download.
		st, err := store.Open(ctx, cfg.StatePath)
		if err != nil {
			return err
		}
		defer func() { _ = st.Close() }()
		id, err := st.Identity(ctx)
		if err != nil {
			return err
		}
		if id == nil {
			return fmt.Errorf("this host has no credential, so it cannot fetch a binary")
		}

		client, err := bee.NewClient(cfg.BaseURL, id.KeyPublic, id.KeySecret)
		if err != nil {
			return err
		}
		dist, _, err := client.FleetAPI.FleetBeeAgentDist(ctx).Execute()
		if err != nil {
			return fmt.Errorf("ask the control plane what it is serving: %w", err)
		}

		var sha string
		for _, b := range dist.GetBuilds() {
			if b.GetOs() == runtime.GOOS && b.GetArch() == runtime.GOARCH {
				sha = b.GetSha256()
			}
		}
		if sha == "" {
			return fmt.Errorf("the control plane serves no agent for %s/%s",
				runtime.GOOS, runtime.GOARCH)
		}
		// COMPARED ON THE DIGEST, not the version.
		//
		// This is also the backstop for the control plane queueing a task it should
		// not have: even if the version comparison there said "out of date", an
		// agent already running these exact bytes has nothing to do, and doing it
		// anyway is a restart of the whole fleet to reach the state it was in.
		if self, derr := selfupgrade.SelfDigest(); derr == nil && self == sha {
			e.Logf("agent", "already running this build (%s); nothing to install", sha[:19])
			return nil
		} else if derr != nil {
			// Not fatal: the digest of the DOWNLOAD is still verified before
			// anything is installed, which is the check that matters. Losing this
			// one costs a needless reinstall, not safety.
			e.Logf("agent", "could not hash the running binary (%v); installing anyway", derr)
		}

		e.Phase("downloading")
		e.Logf("agent", "downloading %s for %s/%s", dist.GetVersion(), runtime.GOOS, runtime.GOARCH)
		body, err := selfupgrade.Download(ctx, &http.Client{Timeout: 10 * time.Minute},
			cfg.BaseURL, id.KeyPublic, id.KeySecret, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return err
		}

		e.Phase("installing")
		// Unit deliberately EMPTY: the restart is handled below, after this
		// executor has returned and its outcome has been recorded.
		res, err := selfupgrade.Apply(ctx, osRunner{}, selfupgrade.Plan{
			Binary: body, SHA256: sha,
		})
		if err != nil {
			return err
		}
		e.Logf("agent", "installed %s at %s; previous kept at %s",
			dist.GetVersion(), res.Target, res.Backup)

		// Declared BEFORE the restart is armed, and synchronously. The agent is
		// about to disappear for a few seconds, and a host that goes quiet without
		// saying so first is indistinguishable from one that has failed.
		if err := e.Declare(ctx, 2*time.Minute, "restarting after an agent upgrade"); err != nil {
			// Not fatal to the upgrade, which has already happened. Logged so the
			// brief silence has an explanation somewhere even though the control
			// plane did not record one.
			e.Logf("agent", "could not declare the restart window: %v", err)
		}

		if err := armRestart(restartDelay, hostinstall.UnitName); err != nil {
			return fmt.Errorf("the new binary is installed but the restart could not be "+
				"scheduled, so this host is still running the old one: %w", err)
		}
		e.Logf("agent", "restarting in %s", restartDelay)
		return nil
	}
}

// armRestart schedules a detached `systemctl restart`.
//
// Detached into its own session, because the thing it restarts is the process
// that started it: a child in this agent's process group would be killed by the
// same stop that the restart performs, and the unit would go down and stay down.
func armRestart(after time.Duration, unit string) error {
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("sleep %d; systemctl restart %s", int(after.Seconds()), unit))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
