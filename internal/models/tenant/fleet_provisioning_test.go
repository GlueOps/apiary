package tenant

import (
	"testing"
	"time"
)

func TestProvisioningStateSeparatesServiceFromPath(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	th := ProvisioningThresholds{Staleness: 2 * time.Minute, HostDownAfter: 30 * time.Minute}
	at := func(d time.Duration) *time.Time { u := now.Add(d); return &u }

	cases := []struct {
		name string
		bee  *Bee
		obs  ProvisioningObservation
		want string
	}{
		{
			name: "control plane reached it",
			bee:  &Bee{},
			obs:  ProvisioningObservation{ControlPlaneReachedAt: at(-10 * time.Second)},
			want: ProvisioningAvailable,
		},
		{
			// The correction this whole file exists for: the API is unreachable and
			// the host is entirely healthy. A declared reboot takes 8006 down with
			// it, and that must not read as a fault on either axis.
			name: "declared reboot outranks every observation",
			bee:  &Bee{UnavailableUntil: at(5 * time.Minute)},
			obs: ProvisioningObservation{
				AgentSaysListening:  false,
				AgentObservedAt:     at(-10 * time.Second),
				ControlPlaneTriedAt: at(-5 * time.Second),
			},
			want: ProvisioningDeclared,
		},
		{
			// Inside says the service is not up. Conclusive, and it does not need a
			// failed attempt from outside to be true.
			name: "agent reports nothing listening",
			bee:  &Bee{},
			obs: ProvisioningObservation{
				AgentSaysListening: false,
				AgentObservedAt:    at(-10 * time.Second),
			},
			want: ProvisioningServiceDown,
		},
		{
			// Listening inside, unreachable outside, and someone actually tried.
			// This is the firewall case — including a ruleset that has not
			// converged yet.
			name: "listening inside, unreachable outside",
			bee:  &Bee{},
			obs: ProvisioningObservation{
				AgentSaysListening:  true,
				AgentObservedAt:     at(-10 * time.Second),
				ControlPlaneTriedAt: at(-5 * time.Second),
			},
			want: ProvisioningBlocked,
		},
		{
			// Same host, nobody has tried lately. An idle fleet must not report as
			// blocked — that is how a status column becomes noise and gets ignored.
			name: "listening inside, nobody has tried",
			bee:  &Bee{},
			obs: ProvisioningObservation{
				AgentSaysListening: true,
				AgentObservedAt:    at(-10 * time.Second),
			},
			want: ProvisioningUnknown,
		},
		{
			// No recent view from inside: blocked and service-down are
			// indistinguishable, so claim neither.
			name: "stale agent observation cannot localise the fault",
			bee:  &Bee{},
			obs: ProvisioningObservation{
				AgentSaysListening:  true,
				AgentObservedAt:     at(-30 * time.Minute),
				ControlPlaneTriedAt: at(-5 * time.Second),
			},
			want: ProvisioningUnknown,
		},
		{
			// A success old enough to be meaningless must not keep a dead host
			// green.
			name: "stale success is not availability",
			bee:  &Bee{},
			obs: ProvisioningObservation{
				ControlPlaneReachedAt: at(-30 * time.Minute),
				AgentSaysListening:    false,
				AgentObservedAt:       at(-10 * time.Second),
			},
			want: ProvisioningServiceDown,
		},
		{
			name: "an expired declaration stops excusing anything",
			bee:  &Bee{UnavailableUntil: at(-1 * time.Minute)},
			obs: ProvisioningObservation{
				AgentSaysListening: false,
				AgentObservedAt:    at(-10 * time.Second),
			},
			want: ProvisioningServiceDown,
		},
		{
			name: "no evidence at all",
			bee:  &Bee{},
			obs:  ProvisioningObservation{},
			want: ProvisioningUnknown,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.bee.ProvisioningStateAt(now, c.obs, th); got != c.want {
				t.Fatalf("ProvisioningStateAt = %q, want %q", got, c.want)
			}
		})
	}
}

// Agent liveness and provisioning reachability must be independently derivable.
// If either could be computed from the other, one of them is redundant and the
// conflation this design rejects has crept back in.
func TestProvisioningIsIndependentOfAgentLiveness(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	th := ProvisioningThresholds{Staleness: 2 * time.Minute, HostDownAfter: 30 * time.Minute}
	at := func(d time.Duration) *time.Time { u := now.Add(d); return &u }

	// Agent silent, API reachable: the agent process died on a host that is
	// otherwise serving.
	deadAgent := &Bee{LastSeenAt: at(-1 * time.Hour)}
	if got := deadAgent.LivenessAt(now, time.Minute); got != BeeStatusOffline {
		t.Fatalf("liveness = %q, want %q", got, BeeStatusOffline)
	}
	if got := deadAgent.ProvisioningStateAt(now, ProvisioningObservation{
		ControlPlaneReachedAt: at(-5 * time.Second),
	}, th); got != ProvisioningAvailable {
		t.Fatalf("provisioning = %q, want %q — a dead agent must not condemn a working API", got, ProvisioningAvailable)
	}

	// Agent healthy, API down: pveproxy has stopped on a host that is up.
	liveAgent := &Bee{LastSeenAt: at(-5 * time.Second)}
	if got := liveAgent.LivenessAt(now, time.Minute); got != BeeStatusOnline {
		t.Fatalf("liveness = %q, want %q", got, BeeStatusOnline)
	}
	if got := liveAgent.ProvisioningStateAt(now, ProvisioningObservation{
		AgentSaysListening: false,
		AgentObservedAt:    at(-5 * time.Second),
	}, th); got != ProvisioningServiceDown {
		t.Fatalf("provisioning = %q, want %q — a live agent must not vouch for a stopped API", got, ProvisioningServiceDown)
	}
}

// Correlated silence. The branch concludes something much stronger than the
// others — go look at the machine — so the cases that must NOT reach it matter
// as much as the one that must.
func TestProvisioningHostDownNeedsBothSignalsDark(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	th := ProvisioningThresholds{Staleness: 2 * time.Minute, HostDownAfter: 30 * time.Minute}
	at := func(d time.Duration) *time.Time { u := now.Add(d); return &u }

	cases := []struct {
		name string
		bee  *Bee
		obs  ProvisioningObservation
		want string
	}{
		{
			name: "agent silent for an hour and the api will not answer",
			bee:  &Bee{LastSeenAt: at(-1 * time.Hour)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningHostDown,
		},
		{
			// One signal is not two. A silent agent on its own is an agent
			// problem; nobody has asked the host whether it is alive.
			name: "agent silent for an hour but nobody tried the api",
			bee:  &Bee{LastSeenAt: at(-1 * time.Hour)},
			obs:  ProvisioningObservation{},
			want: ProvisioningUnknown,
		},
		{
			// The independence that makes the branch sound, shown directly: the
			// API answering proves the host is up, whatever the agent is doing.
			name: "agent silent for an hour but the api answers",
			bee:  &Bee{LastSeenAt: at(-1 * time.Hour)},
			obs:  ProvisioningObservation{ControlPlaneReachedAt: at(-5 * time.Second)},
			want: ProvisioningAvailable,
		},
		{
			// Short double silence is an undeclared reboot, not a dead machine.
			// Firing here would condemn a host on every restart someone forgot to
			// declare.
			name: "both dark, but only briefly",
			bee:  &Bee{LastSeenAt: at(-5 * time.Minute)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningUnknown,
		},
		{
			// A declared dist-upgrade runs long and takes both signals with it.
			// The host said so in advance; that outranks the inference.
			name: "declared window still open",
			bee:  &Bee{LastSeenAt: at(-45 * time.Minute), UnavailableUntil: at(15 * time.Minute)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningDeclared,
		},
		{
			// Past its own deadline the declaration incriminates: the host
			// promised to return and did not.
			name: "declared window lapsed and it never came back",
			bee:  &Bee{LastSeenAt: at(-90 * time.Minute), UnavailableUntil: at(-30 * time.Minute)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningHostDown,
		},
		{
			// Enrolment is performed BY the host, so it proves the machine was up
			// then. A host that enrolled and never reported is reachable by this
			// branch rather than sitting in unknown forever.
			name: "enrolled long ago and never once checked in",
			bee:  &Bee{EnrolledAt: now.Add(-3 * time.Hour)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningHostDown,
		},
		{
			name: "enrolled moments ago and not yet checked in",
			bee:  &Bee{EnrolledAt: now.Add(-20 * time.Second)},
			obs:  ProvisioningObservation{ControlPlaneTriedAt: at(-10 * time.Second)},
			want: ProvisioningUnknown,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.bee.ProvisioningStateAt(now, c.obs, th); got != c.want {
				t.Fatalf("ProvisioningStateAt = %q, want %q", got, c.want)
			}
		})
	}
}
