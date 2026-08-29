package fleetops

import (
	"context"
	"fmt"
	"strings"
)

// DesiredMember is one node the cluster should contain, as published.
type DesiredMember struct {
	Node         string
	HypervisorID string
	Address      string
	Fingerprint  string

	// HostKeys are this member's SSH host public keys. A joining host pins these
	// before pvecm add runs ssh-copy-id.
	HostKeys []string

	// RootKey is this member's root SSH public key, which the CLUSTER must trust
	// before that member can join. Points the other way from HostKeys: those
	// prove a peer's identity to a joiner, this proves a joiner's identity to a
	// peer.
	RootKey string
}

// Desired is the composition the control plane published, plus this host's own
// name within it.
type Desired struct {
	Cluster string

	// Self is this host's Proxmox node name as the control plane knows it, empty
	// for a machine that has never been in a cluster.
	Self string

	// SelfID is this machine's hypervisor id, which the control plane always
	// knows. It is what lets a NAMELESS machine recognise itself in the
	// composition — a host with no node name would otherwise conclude it is not
	// wanted and never join, and that is the state every arrival starts from.
	SelfID  string
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

	// VerbBlocked is a gap this host cannot close on its own — distinct from
	// VerbNone, which means converged. Conflating them made a host in the wrong
	// cluster report itself as converged.
	VerbBlocked Verb = "blocked"
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
	desired := d.desiresSelf()

	switch {
	case desired && obs.Cluster == "":
		return VerbJoin, "", "this host should be in " + d.Cluster + " and is standalone"

	case desired && !strings.EqualFold(obs.Cluster, d.Cluster):
		return VerbBlocked, "", "this host is in " + obs.Cluster + " but should be in " + d.Cluster +
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

// isSelf reports whether a composition entry is this host.
//
// Each identifier is only consulted when BOTH sides carry one. Comparing two
// empty strings as a match is how every unidentified member becomes "me".
func (d Desired) isSelf(m DesiredMember) bool {
	if d.SelfID != "" && m.HypervisorID != "" && m.HypervisorID == d.SelfID {
		return true
	}
	return d.Self != "" && m.Node != "" && strings.EqualFold(m.Node, d.Self)
}

// desires reports whether THIS host is in the composition, by either identifier.
func (d Desired) desiresSelf() bool {
	for _, m := range d.Members {
		if d.isSelf(m) {
			return true
		}
	}
	return false
}

// desires reports whether a node NAME is in the composition, which is how a
// member judges its peers: corosync reports names and nothing else.
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

// otherMembers is every member of the composition that is not this host,
// whether or not it is usable as a join target.
//
// The distinction from joinTargets is what keeps a create from splitting a
// cluster: "nobody else is supposed to be here" and "somebody else is supposed
// to be here but I cannot verify them" look identical in a filtered list and
// call for opposite actions.
func (d Desired) otherMembers() []string {
	out := make([]string, 0, len(d.Members))
	for _, m := range d.Members {
		if !d.isSelf(m) {
			name := m.Node
			if name == "" {
				name = m.Address
			}
			out = append(out, name)
		}
	}
	return out
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
		// A member with no host keys is excluded for the same reason as one with
		// no fingerprint: a join through it could only proceed by trusting
		// whatever answers on its address, and this is the operation where the
		// peer's identity matters most — the corosync auth key and the cluster
		// configuration are what an impostor would receive.
		// Both identifiers guarded against the empty case. An unguarded
		// `m.HypervisorID == d.SelfID` matches when BOTH are empty, which
		// silently treats every unidentified member as this host and empties the
		// target list — the same shape as the bug that would have formed a
		// duplicate cluster.
		if d.isSelf(m) || m.Fingerprint == "" || m.Address == "" || len(m.HostKeys) == 0 {
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

	// ADMIT BEFORE DECIDING. A node that is desired but not yet present cannot
	// install its own key — the cluster's authorized_keys is on the replicated
	// store, which it has no access to until it is a member. So a node already
	// in does it, the same way one already in owns an eviction.
	//
	// This is what makes a ROTATED node able to rejoin at all. Departing with
	// rotate replaces the node's root keypair, so the cluster still trusts the
	// key that was thrown away and `pvecm add --use_ssh` fails to authenticate
	// with "unable to copy ssh ID" and nothing else.
	admitAbsentMembers(ctx, d, e, want, obs)

	verb, target, why := want.Decide(obs)
	if why != "" {
		e.Logf("agent", "%s", why)
	}

	switch verb {
	case VerbNone, VerbWait, VerbBlocked:
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
		// AN EMPTY TARGET LIST IS TWO DIFFERENT FACTS, and treating them alike
		// splits a cluster in half.
		//
		// If the composition names no other member, this host is genuinely the
		// first and forming the cluster is correct. If it names others but none
		// is USABLE — no fingerprint, no address, no captured host keys — then
		// those peers may well have formed the cluster already, and creating here
		// produces a second cluster with the same name and no way to merge them.
		//
		// Found before it ran: every host in the lab predated host-key capture, so
		// the first node reassigned to a hive would have found no usable target
		// and created a duplicate of the cluster it was trying to join.
		if others := want.otherMembers(); len(others) > 0 {
			return VerbJoin, fmt.Errorf(
				"this host should join %s, whose other member(s) are %s, but none of them can be "+
					"joined through: a peer needs an address, a certificate fingerprint and "+
					"captured SSH host keys before a join can verify it. Refusing to form %s "+
					"instead, because those peers may already have done so and a second cluster "+
					"of the same name cannot be merged",
				want.Cluster, strings.Join(others, ", "), want.Cluster)
		}
		e.Logf("agent", "no other member is published for %s, so this host forms it", want.Cluster)
		return VerbCreate, createCluster(ctx, d, e, want.Cluster)
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

		// PIN THE PEER FIRST. pvecm add runs ssh-copy-id at the peer as its first
		// real step, and a host that has never seen that peer's key fails
		// verification — reported as "unable to copy ssh ID: exit code 1", which
		// names neither the host nor the key nor verification. These keys came
		// from the published composition, so they reached this host over its
		// authenticated control-plane session rather than from whatever answers
		// on the peer's address.
		if perr := PinPeer(ctx, []string{t.Address, t.Node}, t.HostKeys); perr != nil {
			last = perr
			e.Logf("agent", "cannot pin %s, so a join through it would have to trust the "+
				"network: %v", t.Node, perr)
			continue
		}
		force := false
		status, jerr := d.Cluster.JoinCluster(ctx, &proxmoxJoinOptions{
			Hostname:    t.Address,
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
	status, err := d.Cluster.CreateCluster(ctx, &proxmoxCreateOptions{ClusterName: name})
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

// admitAbsentMembers installs the root keys of desired members that are not yet
// in the cluster.
//
// Guarded three ways, because this writes to the cluster's replicated keyring:
//
//   - only a node that is itself a converged member does it, since a standalone
//     host has no cluster keyring to write and a departing one has no business
//     changing what the cluster trusts;
//   - only when QUORATE, because a pmxcfs write without quorum blocks rather
//     than failing, and blocking here would stall the whole convergence loop;
//   - only the deterministic owner, so five members do not each append the same
//     key and race on one file.
//
// Never fatal. A failure to admit means a join that cannot authenticate yet, and
// the joiner reports that clearly on its own next attempt — which is a better
// place to see it than a peer's log.
func admitAbsentMembers(ctx context.Context, d Deps, e *Env, want Desired, obs Observation) {
	if obs.Cluster == "" || !strings.EqualFold(obs.Cluster, want.Cluster) || !obs.Quorate {
		return
	}
	owner, ok := want.admissionOwner(obs)
	if !ok || !strings.EqualFold(owner, want.Self) {
		return
	}

	present := map[string]bool{}
	for _, n := range obs.Members {
		present[strings.ToLower(n)] = true
	}
	var keys []string
	for _, m := range want.Members {
		if m.RootKey == "" || present[strings.ToLower(m.Node)] || d.isSelfMember(want, m) {
			continue
		}
		keys = append(keys, m.RootKey)
	}
	if len(keys) == 0 {
		return
	}

	n, err := AdmitKeys(ctx, keys)
	if err != nil {
		e.Logf("agent", "could not admit %d pending member key(s): %v", len(keys), err)
		return
	}
	if n > 0 {
		e.Logf("agent", "admitted %d root key(s) so pending members can authenticate their join", n)
	}
}

// isSelfMember is Desired.isSelf, reachable from the helper above.
func (Deps) isSelfMember(d Desired, m DesiredMember) bool { return d.isSelf(m) }

// admissionOwner names the member responsible for admitting pending keys.
//
// The lowest-named member that is BOTH desired and present, which is the same
// rule the eviction owner uses and for the same reason: every member runs this
// against the same published document, so without a deterministic choice they
// all append the same key to one file at once.
func (d Desired) admissionOwner(obs Observation) (string, bool) {
	owner := ""
	for _, n := range obs.Members {
		if !d.desires(n) {
			continue
		}
		if owner == "" || strings.ToLower(n) < strings.ToLower(owner) {
			owner = n
		}
	}
	return owner, owner != ""
}
