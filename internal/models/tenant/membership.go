package tenant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"sort"
	"strings"
	"time"
)

// Membership is the desired composition of one cluster, published to every
// member so that each can reconcile its own view of it.
//
// PER-CLUSTER, not per-host, and that is the whole design. If an agent only
// knew its own membership, nobody would own a departure: the node being removed
// cannot run `pvecm delnode` against itself, so the act belongs to a node that
// REMAINS — and a remaining node can only know to do it if it has been told what
// the cluster as a whole should contain. Publishing the composition to everyone
// dissolves the ownership problem instead of solving it: a member sees that
// lab-a should be {pve1}, corosync reports {pve1, pve2}, and pve2 is
// desired-absent, so it acts.
//
// It also fixes the post-departure quorum step, which is otherwise awkward.
// `pvecm expected` affects only the node it runs on and cannot be lowered while
// the departing node is still voting — see the pvecm-expected-cannot-be-lowered
// note. Under per-cluster convergence the node that needs it lowered IS the node
// doing the reconciling, and it acts once the vote is actually gone.
type Membership struct {
	// Cluster is the corosync cluster name, which is the hive's name. Carried so
	// an agent can tell "I am in the wrong cluster" from "I am in no cluster" —
	// two states that need opposite actions and that a member list alone cannot
	// distinguish.
	Cluster string `json:"cluster"`

	// Members is every hypervisor this cluster should contain, ordered
	// canonically.
	Members []MembershipMember `json:"members"`

	// Departing names nodes the control plane KNOWS it removed, and it is the
	// only authority for an eviction.
	//
	// ABSENCE FROM Members IS NOT ENOUGH, and this field exists because the
	// obvious design without it is destructive. A node present in corosync and
	// missing from the composition is not necessarily one an operator removed —
	// it is equally a node nobody has recorded YET: manually joined, or joined
	// and waiting for the capacity sync to adopt it, which is a window of a whole
	// sync interval. Evicting on omission cannot tell those apart, so it reads
	// the control plane's own incompleteness as an instruction and removes a
	// healthy node from a working cluster.
	//
	// The asymmetry is deliberate. A constructive verb may act on inference,
	// because the worst case of a wrong join is a node in a cluster it can be
	// removed from. A destructive one may act only on a fact the control plane
	// positively asserts, because the worst case of a wrong eviction is a node
	// removed from a cluster that has already forgotten its guests.
	Departing []string `json:"departing,omitempty"`
}

// MembershipMember is one hypervisor the cluster should contain.
type MembershipMember struct {
	// Node is the Proxmox node name. EMPTY for a machine that has never been in
	// a cluster, because there is no such name until it joins one — see
	// Hypervisor.Name. An agent matches itself by node name when it has one and
	// by hypervisor id otherwise, which is the only identifier that spans the
	// before and after of a first join.
	Node string `json:"node,omitempty"`

	// HypervisorID identifies the machine across a name it does not have yet.
	HypervisorID string `json:"hypervisor_id"`

	// Address is how corosync and pvecm reach this node.
	Address string `json:"address"`

	// Fingerprint of this node's API certificate, needed by a peer joining
	// through it. Without one a join cannot tell the intended cluster from
	// anything else answering on that address, so a member missing a fingerprint
	// is not a usable join target — see JoinTargets.
	Fingerprint string `json:"fingerprint,omitempty"`

	// HostKeys are this node's SSH host public keys, captured during its
	// onboarding over the control plane's authenticated session to it.
	//
	// Carried so a JOINING host can pin this one before `pvecm add` runs
	// ssh-copy-id. Without that the join fails with "unable to copy ssh ID: exit
	// code 1" and nothing else — no host, no key, no mention of verification.
	//
	// The alternative is ssh-keyscan or StrictHostKeyChecking=no, which both hand
	// the join — corosync auth key and cluster configuration included — to
	// whatever answers on the address. A join is the operation where the peer's
	// identity matters most, so it is the wrong place to stop checking. Here the
	// key reaches the joiner over its own authenticated control-plane session,
	// which is a different channel from the one it is about to open.
	HostKeys []string `json:"host_keys,omitempty"`

	// RootKey is this node's root SSH public key, which the CLUSTER must trust
	// before this node can join it — `pvecm add --use_ssh` authenticates as root
	// from the joiner to a peer, and has no password to fall back on.
	//
	// Published to the existing members so they can admit a node that is desired
	// but not yet present. The joiner cannot install its own key: the file lives
	// on the cluster's replicated store, which it has no access to until it is a
	// member. That is the same shape as an eviction — the work belongs to a node
	// that is already in.
	RootKey string `json:"root_key,omitempty"`

	// Links are this node's corosync rings, if the operator configured any.
	Links []CorosyncLink `json:"links,omitempty"`
}

// RenderMembership produces the canonical document and its digest.
//
// CANONICAL ORDERING IS LOAD-BEARING, not tidiness. The digest is what every
// agent compares against to decide whether to act, so a document whose byte
// order depends on the order rows came back from Postgres would change digest
// on a query plan change and send an entire fleet reconciling a composition
// nobody edited. Sorted by node name, then by hypervisor id for the members that
// have no name yet.
func RenderMembership(cluster string, members []MembershipMember) (Membership, string) {
	out := Membership{Cluster: cluster, Members: append([]MembershipMember(nil), members...)}
	sort.SliceStable(out.Members, func(i, j int) bool {
		a, b := out.Members[i], out.Members[j]
		if !strings.EqualFold(a.Node, b.Node) {
			return strings.ToLower(a.Node) < strings.ToLower(b.Node)
		}
		return a.HypervisorID < b.HypervisorID
	})
	for i := range out.Members {
		sort.SliceStable(out.Members[i].Links, func(x, y int) bool {
			return out.Members[i].Links[x].Number < out.Members[i].Links[y].Number
		})
	}
	// Marshalled through the same struct the agent decodes, so the digest covers
	// exactly the bytes that travel. Hashing a hand-built string here would let
	// the two drift, and the drift would show up as a fleet that reconciles
	// forever because the digest never matches what was served.
	body, err := json.Marshal(out)
	if err != nil {
		// Unreachable: every field is a string, an int or a slice of those. A
		// digest of nothing would compare equal across different compositions,
		// so this returns empty and callers treat an empty digest as "cannot
		// publish" rather than as a value.
		return out, ""
	}
	sum := sha256.Sum256(body)
	return out, "sha256:" + hex.EncodeToString(sum[:])
}

// MembershipVerb is what an agent derives from desired against observed.
//
// DERIVED FROM THE WORLD, never chosen by the control plane. To become a member
// of a cluster that has no other members is `pvecm create`; to become a member
// of one that has members is `pvecm add`; to stop being a member is `delnode`.
// Nothing needs to decide in advance which of those an operator asked for, which
// is worth noticing because a whole chicken-and-egg was built on choosing wrong:
// arrival refused with target_hive_empty until a cluster had a node, and a
// separate bootstrap operation existed solely to create the first one. Under
// convergence that distinction does not exist.
type MembershipVerb string

const (
	// VerbNone is the common case and the important one. A fleet whose
	// composition has not changed must not touch corosync every poll.
	VerbNone MembershipVerb = "none"

	// VerbCreate forms the cluster, because this node is desired in it and no
	// other desired member is in it yet.
	VerbCreate MembershipVerb = "create"

	// VerbJoin adds this node to a cluster that already has members.
	VerbJoin MembershipVerb = "join"

	// VerbEvict removes ANOTHER node that is present and no longer desired.
	VerbEvict MembershipVerb = "evict"

	// VerbDissolve makes this node standalone because it is desired-absent and
	// no other node remains to evict it. The one-node case, which under dispatch
	// was a distinct "last node" code path.
	VerbDissolve MembershipVerb = "dissolve"

	// VerbBlocked is a gap this host cannot close on its own.
	//
	// DISTINCT FROM VerbNone, which means converged. Both are "do nothing now",
	// and conflating them made a host in the WRONG cluster report itself as
	// converged: the control plane showed .12 settled into lab-a while corosync
	// had it in lab-b. A state nobody can reach without an operator must not read
	// as a state already reached.
	VerbBlocked MembershipVerb = "blocked"

	// VerbWait is desired-absent with peers still present: some other node owns
	// the eviction. Distinct from VerbNone so an agent can say why it is idle,
	// and so a departure that nobody picks up is visible as a wait rather than
	// as nothing happening.
	VerbWait MembershipVerb = "wait"
)

// Observed is what an agent reports about the cluster it can actually see.
type Observed struct {
	// Cluster is empty when this host is standalone. A successful read that
	// yields no cluster name IS the standalone signal — see the standalone
	// detection note — so empty here is an observation, not a missing one.
	Cluster string   `json:"cluster,omitempty"`
	Members []string `json:"members,omitempty"`
	Quorate bool     `json:"quorate"`
}

// Decide derives the verb for the node named self.
//
// Ownership is ASYMMETRIC and that asymmetry is what makes the loop terminate. A
// node acts on ITSELF to join, because only it can, and on OTHERS to evict,
// because the node being removed cannot. Getting that backwards produces either
// a departure nobody owns or five remaining nodes racing five delnodes for the
// same peer.
//
// target is the node to act on: the peer to join through for a join, the node to
// remove for an evict, and empty otherwise.
func (m Membership) Decide(self Self, obs Observed) (verb MembershipVerb, target string, why string) {
	desired := m.desires(self)

	switch {
	case desired && obs.Cluster == "":
		// Desired in, standalone. Whether that is a create or a join depends on
		// whether any OTHER desired member is already in the cluster, which this
		// node cannot see from here — it has no cluster to read. The caller
		// resolves it by asking a peer; see JoinTargets.
		return VerbJoin, "", "this host should be in " + m.Cluster + " and is standalone"

	case desired && !strings.EqualFold(obs.Cluster, m.Cluster):
		// BLOCKED, not converged. Leaving a cluster destroys guest configuration,
		// so an operator departs it deliberately; until they do, this host cannot
		// close the gap and must not be reported as having closed it.
		// In the WRONG cluster. Not converged automatically: leaving a cluster
		// destroys guest configuration and the machine may be carrying work
		// nobody has accounted for. An operator departs it deliberately, which
		// makes it desired-absent there and lets the loop above take over.
		return VerbBlocked, "", "this host is in " + obs.Cluster + " but should be in " + m.Cluster +
			"; a move between clusters is a departure followed by an arrival, not a convergence"

	case !desired && obs.Cluster == "":
		// Desired absent and already standalone. Converged.
		return VerbNone, "", ""

	case !desired:
		// Desired absent, still in a cluster. Somebody has to remove it, and it
		// cannot be this node. If no peer remains, nobody ever will.
		if len(m.peersPresent(self, obs)) == 0 {
			return VerbDissolve, "", "this host has left " + obs.Cluster +
				" and no other member remains to remove it"
		}
		return VerbWait, "", "this host is leaving " + obs.Cluster +
			"; a node that remains owns the removal"
	}

	// Desired in, and in the right cluster. Look outward: is anything present
	// that should not be?
	stray, owned := m.strayOwnedBy(self, obs)
	if !owned {
		return VerbNone, "", ""
	}
	if !m.departing(stray) {
		// Reported, never acted on. This is the control plane not knowing about a
		// node yet, which is a discrepancy worth surfacing and not an instruction
		// to remove anything.
		return VerbNone, "", stray + " is in " + obs.Cluster +
			" but the control plane has no record of it; it is NOT being removed, because an " +
			"unrecorded node is one nobody has onboarded yet as often as it is one somebody took out"
	}
	return VerbEvict, stray, stray + " is in " + obs.Cluster + " and the control plane records it as departed"
}

// departing reports whether the control plane positively asserts this node was
// removed.
func (m Membership) departing(node string) bool {
	for _, n := range m.Departing {
		if strings.EqualFold(n, node) {
			return true
		}
	}
	return false
}

// Self is how a host identifies itself within a composition.
//
// TWO IDENTIFIERS, because neither works alone. A machine that has never been in
// a cluster has no Proxmox node name — the name is assigned when it joins one —
// so a composition entry for it carries an empty Node, and matching on names
// would make the host conclude it is not wanted and never join. That is the
// exact state every arrival starts from.
//
// The hypervisor id spans that gap: it identifies the machine before, during and
// after it has a name. The name is still matched, because the OTHER members of a
// cluster are known to each other by node name and nothing else — corosync
// reports names, not database ids.
type Self struct {
	// ID is this machine's hypervisor id. Always known.
	ID string
	// Node is its Proxmox node name, empty until it has been in a cluster.
	Node string
}

// desires reports whether this host is in the desired composition.
func (m Membership) desires(self Self) bool {
	for _, mem := range m.Members {
		if isSelfMember(mem, self) {
			return true
		}
	}
	return false
}

// isSelfMember reports whether a composition entry is the host asking.
//
// Each identifier is consulted only when BOTH sides carry one: comparing two
// empty strings as a match makes every unidentified member "me", which empties
// the join-target list and looks exactly like a cluster with no peers.
func isSelfMember(mem MembershipMember, self Self) bool {
	if self.ID != "" && mem.HypervisorID != "" && mem.HypervisorID == self.ID {
		return true
	}
	return self.Node != "" && mem.Node != "" && strings.EqualFold(mem.Node, self.Node)
}

// desiresNode reports whether a node NAME is in the composition, which is how a
// member judges its peers: corosync reports names, and nothing it says can be
// matched to a hypervisor id.
func (m Membership) desiresNode(node string) bool {
	if node == "" {
		return false
	}
	for _, mem := range m.Members {
		if strings.EqualFold(mem.Node, node) {
			return true
		}
	}
	return false
}

// peersPresent is every observed member other than self.
func (m Membership) peersPresent(self Self, obs Observed) []string {
	out := make([]string, 0, len(obs.Members))
	for _, n := range obs.Members {
		if !strings.EqualFold(n, self.Node) {
			out = append(out, n)
		}
	}
	return out
}

// strayOwnedBy returns the node this host should evict, if any.
//
// ONE OWNER PER EVICTION, chosen as the lowest-named desired member that is
// actually present. Every remaining node runs this same loop against the same
// published document, so without a deterministic owner a three-node cluster
// losing one member issues three concurrent delnodes for it — two of which fail
// against a node that is already gone and report as failures, which is how a
// successful departure ends up looking broken.
//
// Lowest name rather than, say, the oldest member: it is derivable from the
// document alone, so every node computes the same answer without coordinating,
// and it stays stable while the eviction is in flight.
func (m Membership) strayOwnedBy(self Self, obs Observed) (string, bool) {
	owner := ""
	for _, n := range obs.Members {
		if !m.desiresNode(n) {
			continue
		}
		if owner == "" || strings.ToLower(n) < strings.ToLower(owner) {
			owner = n
		}
	}
	if !strings.EqualFold(owner, self.Node) {
		return "", false
	}
	// Deterministic among several strays too, so two evictions do not interleave.
	stray := ""
	for _, n := range obs.Members {
		if m.desiresNode(n) {
			continue
		}
		if stray == "" || strings.ToLower(n) < strings.ToLower(stray) {
			stray = n
		}
	}
	return stray, stray != ""
}

// JoinTargets is every member this host could join through, best first.
//
// A join needs a peer that is ALREADY in the cluster, and the joining node
// cannot tell which of the desired members that is — it has no cluster to read.
// So it is handed all of them and tries in order; a peer that is itself
// standalone refuses the join, which is the correct outcome rather than an
// error to prevent.
//
// Members with no fingerprint are excluded rather than tried without one.
// `pvecm add` accepts a missing fingerprint by prompting, which in a
// non-interactive context hangs until it is killed — reported as a timeout that
// says nothing about the cause.
func (m Membership) JoinTargets(self Self) []MembershipMember {
	out := make([]MembershipMember, 0, len(m.Members))
	for _, mem := range m.Members {
		if isSelfMember(mem, self) || mem.Fingerprint == "" || mem.Address == "" {
			continue
		}
		out = append(out, mem)
	}
	return out
}

// MembershipDesired is a published composition together with the revision it
// was published at, as served to one agent.
//
// The revision and digest are the hive's, not recomputed here: they are what the
// heartbeat advertised, and an agent that converges must report back the same
// pair it was told to reach. Recomputing them at serve time is how an agent ends
// up reporting a digest the control plane never asked for and is judged diverged
// on every subsequent poll.
type MembershipDesired struct {
	Revision   int
	Digest     string
	Membership Membership

	// SelfID is this machine's hypervisor id, always known. It is what lets a
	// machine with no node name recognise itself in the composition.
	SelfID string

	// Self is this host's Proxmox node name as the control plane knows it, or
	// empty for a machine that has never been in a cluster.
	Self string
}

// MembershipSnapshot is one published composition, kept by revision.
//
// STORED rather than re-derived, and that is what makes issue-then-acknowledge
// coherent. An agent is handed revision N and converges on it; if the
// composition moves to N+1 while it works, the control plane must keep
// advertising N until N is acknowledged. Deriving the body at serve time cannot
// do that — the agent would fetch today's members under yesterday's number,
// apply them, report a digest nobody issued, and never be judged converged.
//
// It mirrors FirewallSnapshot deliberately. The same problem produced the same
// answer there, and having two shapes for "the body at a revision" is how the
// two drift.
type MembershipSnapshot struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	HiveID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_membership_snapshot_rev,priority:1" json:"hive_id"`

	// Revision only ever increases. Unique per hive, which is what stops two
	// concurrent publishes claiming one number with different bodies — the thing
	// every agent uses as its whole basis for deciding it has converged.
	Revision int `gorm:"not null;uniqueIndex:ux_membership_snapshot_rev,priority:2" json:"revision"`

	// Content is the rendered Membership, exactly as served.
	Content datatypes.JSONType[Membership] `gorm:"type:jsonb;not null" json:"content"`

	// Digest of Content, so a lagging agent is told from a converged one without
	// decoding every body.
	Digest string `gorm:"type:varchar(71);not null" json:"digest"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

func (MembershipSnapshot) TableName() string { return "membership_snapshots" }
