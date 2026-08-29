package bee

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/glueops/apiary-agent/internal/fleetops"
)

// appliedMembership is the composition this host has converged on, as
// established by this process.
//
// Held in memory rather than re-derived, for the same reason the firewall's is:
// it records what this agent applied and CONFIRMED, and it starts empty after a
// restart so the first heartbeat reports nothing, the control plane re-sends,
// and the host re-checks. Asserting a revision this process never verified is
// how a host is reported as converged on the strength of a claim nobody checked.
//
// The stakes are higher here than for the firewall. Re-applying an identical
// ruleset is cheap; re-evaluating cluster membership after a restart is also
// cheap, because the common answer is "converged, do nothing" — but only because
// the decision is made against what corosync actually reports, never against
// remembered state.
type appliedMembership struct {
	revision atomic.Int64
	digest   atomic.Value
}

func (a *appliedMembership) get() (int, string) {
	d, _ := a.digest.Load().(string)
	return int(a.revision.Load()), d
}

func (a *appliedMembership) set(rev int, digest string) {
	a.revision.Store(int64(rev))
	a.digest.Store(digest)
}

// desiredMembership is what the control plane last said this cluster should be.
type desiredMembership struct {
	revision int
	digest   string
}

// observeCluster reads what this host can see about its own cluster, for the
// heartbeat.
//
// Returns nil when there is nothing sound to report. A FAILED read is silence,
// never an empty observation: "I could not ask" and "I am standalone" lead to
// opposite decisions, and the control plane's gates act on the second.
func (b *Bee) observeCluster(ctx context.Context) *fleetops.Observation {
	if b.opts.Fleet.Cluster == nil {
		return nil
	}
	obs, err := fleetops.Observe(ctx, b.opts.Fleet.Cluster)
	if err != nil {
		slog.DebugContext(ctx, "could not read this host's cluster state; reporting nothing",
			"error", err)
		return nil
	}
	return &obs
}

// convergeMembership brings this host's cluster membership to the published
// composition.
//
// Doing nothing is the common case and the important one: a fleet whose
// composition has not changed must not touch corosync every poll.
func (b *Bee) convergeMembership(ctx context.Context, desiredRev int, desiredDigest string) {
	if desiredRev <= 0 || desiredDigest == "" {
		// HOLD. The machine is in no hive, its hive has published nothing, or the
		// control plane could not tell which. None of those means "leave your
		// cluster", and acting on any of them would dissolve an unpublished hive
		// node by node.
		return
	}

	haveRev, haveDigest := b.membership.get()
	// Both compared, as for the firewall. A revision that matches while the
	// digest does not means the composition changed without the number moving,
	// and it is the composition that is being enforced.
	if haveRev == desiredRev && haveDigest == desiredDigest {
		return
	}

	if b.opts.Fleet.Cluster == nil {
		// ONCE PER REVISION, not once per poll. A host with no Proxmox token can
		// never converge, so the unguarded version of this repeated every
		// heartbeat — observed on a lab node at three lines in ninety seconds,
		// which over a day is the only thing left in the journal. The condition is
		// worth one loud line each time the desired state changes, and worth
		// nothing at all in between.
		if b.warnedNoClient.Swap(int64(desiredRev)) != int64(desiredRev) {
			slog.WarnContext(ctx, "a cluster composition was published but this host has no "+
				"Proxmox client, so it cannot converge. Set proxmox_token_id and proxmox_secret "+
				"in the agent configuration", "desired_revision", desiredRev)
		}
		return
	}

	res, _, err := b.opts.Client.FleetAPI.FleetBeeMembership(ctx).Execute()
	if err != nil {
		slog.WarnContext(ctx, "could not fetch the desired cluster composition; holding",
			"desired_revision", desiredRev, "error", err)
		return
	}
	if res.GetRevision() == 0 {
		// The heartbeat named a revision and the body came back empty. Holding is
		// the only safe reading: the alternative is acting on a response that
		// contradicts the one before it, and the action is a cluster operation.
		slog.WarnContext(ctx, "the control plane named a membership revision but returned no "+
			"composition; holding", "desired_revision", desiredRev)
		return
	}

	want := fleetops.Desired{
		Cluster:   res.GetCluster(),
		Self:      res.GetSelf(),
		Departing: res.GetDeparting(),
	}
	for _, m := range res.GetMembers() {
		want.Members = append(want.Members, fleetops.DesiredMember{
			Node:         m.GetNode(),
			HypervisorID: m.GetHypervisorId(),
			Address:      m.GetAddress(),
			Fingerprint:  m.GetFingerprint(),
		})
	}

	e := b.membershipEnv(ctx)
	verb, cerr := fleetops.Converge(ctx, b.opts.Fleet, e, want)
	if cerr != nil {
		// NOT recorded as applied, so the next heartbeat reports the old revision
		// and the control plane keeps showing this host as diverged — which is
		// true, and is what an operator needs to see.
		slog.ErrorContext(ctx, "could not converge this host's cluster membership",
			"revision", res.GetRevision(), "verb", verb, "error", cerr)
		return
	}

	// A verb that ACTED does not get to declare itself converged. Re-observed and
	// re-decided, which is the same discipline that turned "pvecm exited 0" into
	// a verified membership read — an accepted request is not a joined cluster.
	obs, oerr := fleetops.Observe(ctx, b.opts.Fleet.Cluster)
	if oerr != nil {
		slog.WarnContext(ctx, "converged but could not confirm this host's cluster state",
			"revision", res.GetRevision(), "error", oerr)
		return
	}
	if again, _, why := want.Decide(obs); again != fleetops.VerbNone {
		slog.InfoContext(ctx, "this host is not converged yet", "verb", again, "why", why)
		return
	}

	b.membership.set(int(res.GetRevision()), res.GetDigest())
	slog.InfoContext(ctx, "cluster membership converged",
		"revision", res.GetRevision(), "cluster", want.Cluster, "verb", verb)
}

// membershipEnv adapts the agent's logging into what fleetops expects.
//
// Convergence has no task, so its output has nowhere to stream: the log lines go
// to the journal rather than to fleet_task_logs. That is the one thing a
// convergence loses against a dispatched task, and it is why a failure here is
// logged at ERROR with the verb attached — that line is the whole diagnosis.
func (b *Bee) membershipEnv(ctx context.Context) *fleetops.Env {
	return &fleetops.Env{
		Logf: func(_ string, format string, args ...any) {
			slog.InfoContext(ctx, "membership: "+fmt.Sprintf(format, args...))
		},
		Phase: func(string) {},
		// Convergence never reboots, so nothing here declares downtime. A verb
		// that needed one would have to be a task, because a declaration has to
		// reach the control plane BEFORE the silence and a convergence has no
		// outcome to report afterwards.
		Declare: func(context.Context, time.Duration, string) error { return nil },
	}
}
