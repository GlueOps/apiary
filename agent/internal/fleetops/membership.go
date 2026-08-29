package fleetops

import (
	"context"
	"fmt"
	"strings"

	"github.com/glueops/apiary/pkg/proxmox"
)

// DesiredMember is one node the cluster should contain, as published.
type DesiredMember struct {
	Node         string
	HypervisorID string
	Address      string
	Fingerprint  string
}

// Desired is the composition the control plane published, plus this host's own
// name within it.
type Desired struct {
	Cluster string
	Self    string
	Members []DesiredMember

	// Departing names nodes the control plane KNOWS it removed, and it is the
	// only authority for an eviction. Absence from Members is NOT enough: a node
	// in corosync and missing from the composition is as likely to be one nobody
	// has recorded yet — manually joined, or waiting for the capacity sync to
	// adopt it — as one an operator took out, and the two need opposite
	// treatment. A constructive verb may act on inference; a destructive one may
	// not.
	Departing []string
}

// Verb is what this host should do to close the gap between desired and
// observed. It mirrors the control plane's tenant.MembershipVerb, redeclared
// here so the agent does not import the control plane's models.
type Verb string

const (
	VerbNone     Verb = "none"
	VerbCreate   Verb = "create"
	VerbJoin     Verb = "join"
	VerbEvict    Verb = "evict"
	VerbDissolve Verb = "dissolve"
	VerbWait     Verb = "wait"
)

// Observation is what this host can see about its own cluster.
type Observation struct {
	Cluster string
	Members []string
	Quorate bool
}

// Observe reads this host's own view of its cluster.
//
// The error is returned rather than folded into an empty Observation, because
// EMPTY IS AN ANSWER here: a standalone node's status reports no cluster name,
// so "no name" and "could not ask" must not collapse into the same value. They
// lead to opposite actions — one is converged, the other must not act at all.
func Observe(ctx context.Context, c Cluster) (Observation, error) {
	if c == nil {
		return Observation{}, ErrNoCluster
	}
	if err := c.Status(ctx); err != nil {
		if isNotClustered(err) {
			// A definite answer: this host is in no cluster.
			return Observation{}, nil
		}
		return Observation{}, err
	}
	return Observation{
		Cluster: c.ClusterName(),
		Members: c.NodeNames(),
		Quorate: c.Quorate(),
	}, nil
}

// Decide derives the verb for this host.
//
// The SAME rules as the control plane's tenant.Membership.Decide, and
// deliberately duplicated rather than shared: the agent module does not import
// the control plane, and a decision the host makes about its own cluster should
// not depend on a package that could change under it in a release it did not
// take. The control plane's copy is the tested specification; this one is what
// actually acts, and the lab test is what proves they agree.
func (d Desired) Decide(obs Observation) (Verb, string, string) {
	self := d.Self
	desired := d.desires(self)

	switch {
	case desired && obs.Cluster == "":
		return VerbJoin, "", "this host should be in " + d.Cluster + " and is standalone"

	case desired && !strings.EqualFold(obs.Cluster, d.Cluster):
		return VerbNone, "", "this host is in " + obs.Cluster + " but should be in " + d.Cluster +
			"; a move between clusters is a departure followed by an arrival, not a convergence"

	case !desired && obs.Cluster == "":
		return VerbNone, "", ""

	case !desired:
		if len(d.peers(self, obs)) == 0 {
			return VerbDissolve, "", "this host has left " + obs.Cluster +
				" and no other member remains to remove it"
		}
		return VerbWait, "", "this host is leaving " + obs.Cluster +
			"; a node that remains owns the removal"
	}

	stray, owned := d.strayOwnedBy(self, obs)
	if !owned {
		return VerbNone, "", ""
	}
	if !d.departing(stray) {
		return VerbNone, "", stray + " is in " + obs.Cluster +
			" but the control plane has no record of it; it is NOT being removed, because an " +
			"unrecorded node is one nobody has onboarded yet as often as it is one somebody took out"
	}
	return VerbEvict, stray, stray + " is in " + obs.Cluster + " and the control plane records it as departed"
}

func (d Desired) departing(node string) bool {
	for _, n := range d.Departing {
		if strings.EqualFold(n, node) {
			return true
		}
	}
	return false
}

func (d Desired) desires(node string) bool {
	if node == "" {
		return false
	}
	for _, m := range d.Members {
		if strings.EqualFold(m.Node, node) {
			return true
		}
	}
	return false
}

func (d Desired) peers(self string, obs Observation) []string {
	out := make([]string, 0, len(obs.Members))
	for _, n := range obs.Members {
		if !strings.EqualFold(n, self) {
			out = append(out, n)
		}
	}
	return out
}

// strayOwnedBy names the node this host should evict, if this host is the owner.
//
// ONE OWNER, the lowest-named desired member actually present. Every remaining
// node runs this against the same published document, so without a deterministic
// choice a three-node cluster losing a member issues three concurrent delnodes —
// two of which fail against a node that is already gone and report as failures,
// which is how a successful departure ends up looking broken.
func (d Desired) strayOwnedBy(self string, obs Observation) (string, bool) {
	owner := ""
	for _, n := range obs.Members {
		if !d.desires(n) {
			continue
		}
		if owner == "" || strings.ToLower(n) < strings.ToLower(owner) {
			owner = n
		}
	}
	if !strings.EqualFold(owner, self) {
		return "", false
	}
	stray := ""
	for _, n := range obs.Members {
		if d.desires(n) {
			continue
		}
		if stray == "" || strings.ToLower(n) < strings.ToLower(stray) {
			stray = n
		}
	}
	return stray, stray != ""
}

// joinTargets is every member this host could join through, excluding itself and
// any peer with no fingerprint or address.
//
// A peer without a fingerprint is EXCLUDED rather than tried without one: pvecm
// accepts a missing fingerprint by prompting, which non-interactively hangs
// until something kills it and reports a timeout that says nothing about why.
func (d Desired) joinTargets() []DesiredMember {
	out := make([]DesiredMember, 0, len(d.Members))
	for _, m := range d.Members {
		if strings.EqualFold(m.Node, d.Self) || m.Fingerprint == "" || m.Address == "" {
			continue
		}
		out = append(out, m)
	}
	return out
}

// ErrDissolveNeedsDeparture reports that a host is desired-absent and alone.
//
// Refused rather than converged, on purpose. Making a node standalone destroys
// its cluster secrets, and what should happen to the identity it keeps —
// accept the exposure, or rotate it — is an operator's decision that no desired
// state carries. The deliberate departure operation asks that question and acts
// on the answer; blind convergence would have to guess, and the safe guess and
// the correct guess are not the same one.
var ErrDissolveNeedsDeparture = fmt.Errorf(
	"fleetops: this host is the last member of a cluster it should no longer be in; " +
		"converting it to standalone destroys cluster secrets and the disposition of its own " +
		"identity is an operator decision, so run the departure operation rather than converging")

// Converge closes the gap between what was published and what this host sees.
//
// Returns whether the host is now converged. A verb that ACTED does not report
// converged on its own word — the caller re-observes, which is the same
// discipline that turned "pvecm exited 0" into a verified membership read.
func Converge(ctx context.Context, d Deps, e *Env, want Desired) (Verb, error) {
	if err := requireCluster(d); err != nil {
		return VerbNone, err
	}

	obs, err := Observe(ctx, d.Cluster)
	if err != nil {
		return VerbNone, fmt.Errorf("read this host's own cluster state: %w", err)
	}
	// The control plane knows this host's node name only once it has been in a
	// cluster. Before that the agent is the only one who knows it.
	if want.Self == "" {
		want.Self = d.Node
	}

	verb, target, why := want.Decide(obs)
	if why != "" {
		e.Logf("agent", "%s", why)
	}

	switch verb {
	case VerbNone, VerbWait:
		return verb, nil

	case VerbDissolve:
		return verb, ErrDissolveNeedsDeparture

	case VerbEvict:
		e.Phase(phaseEvicting)
		return verb, evictOne(ctx, d, e, target)

	case VerbJoin:
		return joinOrCreate(ctx, d, e, want)
	}
	return VerbNone, fmt.Errorf("fleetops: no rule for verb %q", verb)
}

// joinOrCreate resolves the one thing a standalone host cannot see: whether the
// cluster it should be in exists yet.
//
// It CANNOT be decided locally, because a standalone node has no cluster to
// read. So the host tries to join each published peer in turn, and forms the
// cluster only when every one of them refuses — which is what "no other member
// is in it yet" actually looks like from here.
//
// Trying first and creating second, never the reverse. Creating first would
// split a cluster in two the moment two hosts converged at once: both would
// find no peer, both would create, and the fleet would hold two clusters with
// the same name and no way to merge them. A failed join is recoverable; a split
// brain is hand-repaired under /etc/pve.
func joinOrCreate(ctx context.Context, d Deps, e *Env, want Desired) (Verb, error) {
	targets := want.joinTargets()
	if len(targets) == 0 {
		e.Logf("agent", "no peer is published for %s yet, so this host forms it", want.Cluster)
		return VerbCreate, createCluster(ctx, d, e, want.Cluster)
	}

	pass, err := d.JoinPass()
	if err != nil {
		return VerbJoin, fmt.Errorf("no cluster join password available on this host: %w", err)
	}

	e.Phase(phasePreflight)
	guests, gerr := d.Cluster.GuestsOn(ctx, d.Node)
	if gerr == nil && len(guests) > 0 {
		// The control plane gates this too, but its observation of a FLEETLESS
		// host is exactly the one that was structurally missing — so this is the
		// check that actually stands between a join and a discarded guest.
		return VerbJoin, fmt.Errorf("this host has %d guest(s); joining a cluster discards local "+
			"guest configuration, so they must be moved or removed first", len(guests))
	}

	e.Phase(phaseJoining)
	var last error
	for _, t := range targets {
		e.Logf("agent", "trying to join %s through %s (%s)", want.Cluster, t.Node, t.Address)
		force := false
		status, jerr := d.Cluster.JoinCluster(ctx, &proxmox.ClusterJoinOptions{
			Hostname:    t.Address,
			Password:    pass,
			Fingerprint: t.Fingerprint,
			Force:       &force,
		})
		if jerr != nil {
			last = jerr
			e.Logf("agent", "%s did not accept the join: %v", t.Node, jerr)
			continue
		}
		e.Logf("agent", "join accepted: %s", strings.TrimSpace(status))
		e.Phase(phaseVerifying)
		return VerbJoin, verifyJoined(ctx, d, e)
	}

	// Every published peer refused. That is what an unformed cluster looks like
	// from a host that cannot see it, so this host forms it — but only after
	// asking all of them, which is what keeps two simultaneous converges from
	// creating two clusters of the same name.
	e.Logf("agent", "no published peer accepted a join, so this host forms %s", want.Cluster)
	if last != nil {
		e.Logf("agent", "the last peer's refusal was: %v", last)
	}
	return VerbCreate, createCluster(ctx, d, e, want.Cluster)
}

func createCluster(ctx context.Context, d Deps, e *Env, name string) error {
	if name == "" {
		return fmt.Errorf("no cluster name was published, so there is nothing to create")
	}
	e.Phase(phaseJoining)
	e.Logf("agent", "forming cluster %s with this host as its first member", name)
	status, err := d.Cluster.CreateCluster(ctx, &proxmox.ClusterCreateOptions{ClusterName: name})
	if err != nil {
		return fmt.Errorf("create cluster %s: %w", name, err)
	}
	e.Logf("agent", "create accepted: %s", strings.TrimSpace(status))
	e.Phase(phaseVerifying)
	// Verified, not trusted. The API returns once the request is ACCEPTED and the
	// work runs afterwards, which is the same trap that made a join report green
	// having joined nothing.
	return verifyJoined(ctx, d, e)
}

// evictOne removes a node that is present and no longer desired.
//
// It reuses the eviction preflight rather than calling delnode directly, so a
// convergence cannot do something the explicit task would have refused —
// notably orphaning guests, which has no recovery path.
func evictOne(ctx context.Context, d Deps, e *Env, node string) error {
	spec := fmt.Sprintf(`{"node":%q}`, node)
	return Evict(ctx, d, e, spec)
}
