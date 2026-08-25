package tenant

import "time"

// Provisioning states.
//
// These are deliberately NOT BeeStatus values. Agent liveness and provisioning
// reachability are separate services with separate lifecycles, and folding them
// into one column is wrong in both directions: the Proxmox API can be down on a
// host whose agent is reporting normally, and the agent can be dead on a host
// whose API serves fine. A single "health" answer has to pick one of those to be
// wrong about.
//
// The operator-facing distinction is the point. "This host is down" sends
// someone to a console. "Provisioning is unavailable on this host" sends them to
// pveproxy or to the firewall, and the two below say which.
const (
	// ProvisioningAvailable — the control plane reached the API.
	ProvisioningAvailable = "available"

	// ProvisioningDeclared — the host said in advance it would be gone. A reboot
	// takes the API down along with everything else, and that is not a fault.
	// This outranks every other state: during a declared window no observation is
	// evidence of anything, because the host is legitimately absent.
	ProvisioningDeclared = "declared_unavailable"

	// ProvisioningServiceDown — the agent, running ON the host, reports nothing
	// listening on the API port. The path is fine; the service is not. Fix
	// pveproxy.
	ProvisioningServiceDown = "service_down"

	// ProvisioningBlocked — the agent reports the API listening locally, and the
	// control plane still cannot reach it. The service is fine; the path is not.
	// Look at the firewall — most often a ruleset that has not converged, or one
	// that never whitelisted the control plane in the first place.
	//
	// This state is the entire reason the agent reports a local observation. From
	// outside, "blocked" and "service down" are the same silence, and an operator
	// handed that silence has two places to look and no way to choose.
	ProvisioningBlocked = "blocked"

	// ProvisioningHostDown — BOTH vantage points have been dark for a long time.
	// Not a provisioning problem: the machine itself is the likeliest
	// explanation, and the remedy is a console or an IPMI, not pveproxy and not
	// the firewall.
	//
	// This is the one conclusion drawn from correlated silence, and it is sound
	// precisely because the two signals are independent. The agent's channel is
	// outbound and polled; the API's is inbound and served. A firewall change
	// kills the second and not the first. A stopped pveproxy kills the second and
	// not the first. Nothing short of the host itself takes both — so when both
	// stay dark, the shared cause is the host.
	//
	// It needs its OWN, much longer threshold than staleness. A brief double
	// silence is an undeclared reboot or a blip, and calling that a dead host
	// would fire on every routine restart somebody forgot to declare.
	ProvisioningHostDown = "host_down"

	// ProvisioningUnknown — not enough recent evidence to say. Distinct from
	// "unavailable" on purpose: a check that cannot reach a decision must not
	// report one, and an unknown shown as a fault is how an operator learns to
	// ignore the column.
	ProvisioningUnknown = "unknown"
)

// ProvisioningObservation is what the two vantage points have most recently
// seen. Both halves carry their own timestamp because they arrive by different
// routes at different rates — the agent's on a heartbeat it initiates, the
// control plane's when it last tried to use the API.
type ProvisioningObservation struct {
	// AgentSaysListening is the view from INSIDE the host.
	AgentSaysListening bool
	AgentObservedAt    *time.Time

	// ControlPlaneReachedAt is the last time the control plane actually reached
	// the API, and ControlPlaneTriedAt the last time it attempted to. Both are
	// needed: without the attempt time, "not reached recently" cannot be told
	// apart from "not tried recently", and a quiet system would report every host
	// as broken.
	ControlPlaneReachedAt *time.Time
	ControlPlaneTriedAt   *time.Time
}

// ProvisioningThresholds carries the two windows apart rather than as two
// positional durations, which are the same type and would swap silently — and a
// swap here inverts the design, reporting a dead host for every momentary gap
// and never reporting one that has been dark for an hour.
type ProvisioningThresholds struct {
	// Staleness bounds how long a single observation stays meaningful.
	Staleness time.Duration

	// HostDownAfter is how long BOTH signals must stay dark before the host
	// itself becomes the likeliest explanation. Much longer than Staleness: this
	// one sends a person to a machine.
	HostDownAfter time.Duration
}

// ProvisioningStateAt reports whether provisioning can be attempted on this
// host, and when it cannot, which side of the boundary to look at.
//
// staleness bounds how long an observation stays meaningful. A view of the host
// from ten minutes ago is not evidence about the host now, and treating it as
// current is how a fault gets attributed to whichever vantage point happened to
// report last.
func (b *Bee) ProvisioningStateAt(now time.Time, obs ProvisioningObservation, th ProvisioningThresholds) string {
	// Declared absence first, before any observation is consulted. During a
	// reboot the agent has stopped reporting and the API has stopped answering,
	// so every other branch below would fire on evidence that means nothing. The
	// host told us this was coming; that statement outranks what we can see.
	//
	// Only while the window is OPEN. Once it lapses the declaration stops
	// excusing anything and starts incriminating: the host gave its word it would
	// be back, so continued silence past its own deadline is a stronger signal
	// than a missed heartbeat ever was.
	if b != nil && b.UnavailableUntil != nil && b.UnavailableUntil.After(now) {
		return ProvisioningDeclared
	}

	fresh := func(t *time.Time) bool {
		return t != nil && now.Sub(*t) <= th.Staleness
	}

	if fresh(obs.ControlPlaneReachedAt) {
		return ProvisioningAvailable
	}

	// Correlated silence, checked before the single-signal branches because it
	// explains both of them at once. Two independent channels dark for this long
	// have one likely shared cause, and reporting it as a provisioning fault
	// sends someone to debug a service on a machine that is not running.
	//
	// A failed ATTEMPT is required, not merely an absence of success: without one
	// nobody has looked at the API, and a silent agent alone says nothing about
	// whether the host is up. This branch rests on two observations, so it needs
	// two.
	if fresh(obs.ControlPlaneTriedAt) && b.silentFor(now) > th.HostDownAfter {
		return ProvisioningHostDown
	}

	// Nothing recent from inside the host, and not dark long enough to blame the
	// host: there is no basis for choosing between blocked and service-down, so
	// choose neither.
	if !fresh(obs.AgentObservedAt) {
		return ProvisioningUnknown
	}

	if !obs.AgentSaysListening {
		// The host itself says the API is not up. That is conclusive from the
		// better vantage point, and it holds whether or not the control plane has
		// tried recently.
		return ProvisioningServiceDown
	}

	// Listening locally, and unreached from here — but only a failed ATTEMPT
	// makes that a fault. Without one, this is simply a host nobody has needed
	// lately, and reporting it as blocked would light up an idle fleet.
	if fresh(obs.ControlPlaneTriedAt) {
		return ProvisioningBlocked
	}
	return ProvisioningUnknown
}

// silentFor is how long the bee has gone without reporting.
//
// It falls back to EnrolledAt when the bee has never checked in, rather than
// treating "never seen" as silence of zero length. Enrolment is performed BY the
// host, so it is proof the machine was up at that moment — which makes it a
// sound clock start, and makes a host that enrolled and never came back
// reachable by the host-down branch instead of sitting in unknown forever.
func (b *Bee) silentFor(now time.Time) time.Duration {
	if b == nil {
		return 0
	}
	since := b.LastSeenAt
	if since == nil {
		if b.EnrolledAt.IsZero() {
			return 0
		}
		since = &b.EnrolledAt
	}
	return now.Sub(*since)
}
