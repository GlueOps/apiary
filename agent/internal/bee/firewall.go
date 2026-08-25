package bee

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/glueops/apiary-agent/internal/firewall"
)

// applied is what this host is currently enforcing, as last established.
//
// Held in memory rather than re-read from nft on every poll. The value reported
// to the control plane is what this agent applied and confirmed; deriving it
// from the live ruleset would mean re-hashing the host's entire firewall every
// few seconds to answer a question that only changes when this code changes it.
//
// It starts at zero after a restart, which is correct: the agent has not applied
// anything in this process, so the first heartbeat reports nothing, the control
// plane sends the desired revision, and the host reconverges. Re-applying an
// identical ruleset is cheap and the alternative — asserting a revision this
// process never verified — is how a host is reported as compliant on the
// strength of a claim nobody checked.
type appliedFirewall struct {
	revision atomic.Int64
	digest   atomic.Value
}

func (a *appliedFirewall) get() (int, string) {
	d, _ := a.digest.Load().(string)
	return int(a.revision.Load()), d
}

func (a *appliedFirewall) set(rev int, digest string) {
	a.revision.Store(int64(rev))
	a.digest.Store(digest)
}

// desiredFirewall is what the control plane last said this host should enforce.
type desiredFirewall struct {
	revision int
	digest   string
}

// converge brings this host's firewall to the revision the control plane names.
//
// Called with what the heartbeat reported as desired. Doing nothing is the
// common case and the important one: a fleet whose policy has not changed must
// not reload its firewall every poll.
func (b *Bee) converge(ctx context.Context, desiredRev int, desiredDigest string) {
	if desiredRev <= 0 || desiredDigest == "" {
		// HOLD. The machine is in no fleet, its fleet has published nothing, or the
		// control plane could not tell which. None of those is an empty ruleset,
		// and flushing on any of them would strip a host of its management access
		// at the moment it is least reachable.
		return
	}

	haveRev, haveDigest := b.firewall.get()
	// Both compared, not just the revision. A host that loaded the right revision
	// from a corrupted or hand-edited file matches on the number and not on the
	// content, and it is the content that is being enforced.
	if haveRev == desiredRev && haveDigest == desiredDigest {
		return
	}

	res, _, err := b.opts.Client.FleetAPI.FleetBeeFirewall(ctx).Execute()
	if err != nil {
		slog.WarnContext(ctx, "could not fetch the desired firewall; holding what is running",
			"desired_revision", desiredRev, "error", err)
		return
	}
	content := res.GetContent()
	if content == "" {
		// The heartbeat named a revision and the body came back empty. Holding is
		// the only safe reading: the alternative is flushing the host's rules on
		// the strength of a response that contradicts the one before it.
		slog.WarnContext(ctx, "the control plane named a firewall revision but returned no "+
			"ruleset; holding what is running", "desired_revision", desiredRev)
		return
	}

	slog.InfoContext(ctx, "applying a firewall revision",
		"from_revision", haveRev, "to_revision", res.GetRevision(), "bytes", len(content))

	out, err := firewall.Apply(ctx, b.opts.FirewallRunner, b.confirmReachable, firewall.Plan{
		Revision: int(res.GetRevision()),
		Digest:   res.GetDigest(),
		Content:  content,
	})
	if err != nil {
		// Not recorded as applied, so the next heartbeat reports the OLD revision
		// and the control plane keeps showing this host as diverged — which is
		// true, and is what an operator needs to see.
		slog.ErrorContext(ctx, "could not apply the firewall revision",
			"revision", res.GetRevision(), "rolled_back", out.RolledBack, "error", err)
		return
	}

	b.firewall.set(int(res.GetRevision()), res.GetDigest())
	slog.InfoContext(ctx, "firewall revision applied",
		"revision", res.GetRevision(), "digest", res.GetDigest())
}

// confirmReachable is the test of whether the ruleset just applied cut this host
// off from the control plane.
//
// A heartbeat, because it is the agent's own traffic on the agent's own path. A
// ping or a TCP dial would traverse a different path and pass while the thing
// that actually matters — this agent talking to this control plane — is blocked.
//
// wait is zero: the confirm must answer in seconds, and the ordinary heartbeat
// holds the request open for up to twenty-five waiting for work.
func (b *Bee) confirmReachable(ctx context.Context) error {
	if _, err := b.heartbeat(ctx, heartbeatOpts{}); err != nil {
		return fmt.Errorf("the control plane did not answer after the ruleset was applied: %w", err)
	}
	return nil
}
