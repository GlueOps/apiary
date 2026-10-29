package tenant

import (
	"fmt"
	"strings"
	"time"
)

// DefaultHiveMinimumNodes is the fewest nodes a departure may leave a hive with.
//
// Three, because of corosync arithmetic that was observed doing damage rather
// than reasoned about: removing one node from a TWO-node cluster drops the
// survivor to one of two expected votes, so it loses quorum — and the next step
// of `pvecm delnode`, rewriting corosync.conf to lower the expected count, needs
// a writable /etc/pve, which needs the quorum it just lost. The removal fails
// half-done and leaves the remaining node read-only with its guests
// unmanageable. Removing one of three leaves two of three, still quorate, and
// none of it happens.
//
// So departing FROM fewer than three breaks the cluster it is leaving, and
// leaving fewer than three is the redundancy margin on top: a two-node
// survivor loses quorum the moment either node fails, and then cannot run the
// next departure at all. The default is the floor an operator reads it as.
const DefaultHiveMinimumNodes = 3

// EffectiveMinimumNodes is the fewest nodes a departure may LEAVE the cluster
// with, counting every departure already under way as gone.
//
// A floor on what remains, rather than a size a departure may start from,
// because it is the only reading that composes: departures checked one at a
// time against what is left cannot add up to more than any one would allow.
//
// A knob rather than the constant because a test fleet is legitimately smaller
// than three, and the alternative to a knob is an operator disabling the gate
// entirely for the one hive that needed a different number.
func (h Hive) EffectiveMinimumNodes() int {
	if h.MinimumNodes <= 0 {
		return DefaultHiveMinimumNodes
	}
	return h.MinimumNodes
}

// Departure refusal codes. Stable strings: they cross into an API response and
// a UI, and a caller keying on prose is a caller that breaks on a reword.
const (
	RefuseNotCordoned    = "not_cordoned"
	RefuseDrainUnknown   = "drain_unknown"
	RefuseGuestsPresent  = "guests_present"
	RefusePlacementsHeld = "placements_held"
	RefuseHiveTooSmall   = "hive_too_small"
	RefuseNotAMember     = "not_a_member"
	// RefuseAlreadyDeparting: a departure was requested for this host and has
	// not finished. Requesting it again would overwrite the first request.
	RefuseAlreadyDeparting = "already_departing"

	// RefuseCredentialsUndecided is raised when the caller has not said what to
	// do about the secrets the departing node has already seen. It is the one
	// refusal here that is not about the cluster's state — it is about a decision
	// only an operator can make. See CredentialDisposition.
	RefuseCredentialsUndecided = "credentials_undecided"
)

// What to do about the cluster secrets a departing node has seen.
//
// /etc/pve/priv is pmxcfs, so it is CLUSTER-WIDE: every node holds the same
// token.cfg, the same authkey.key — the key PVE signs auth tickets with — and
// the same pve-root-ca.key. A node that leaves has had all three.
//
// Measured on a live pair, 2026-08-26: after a departure in which the departing
// node was deliberately told nothing — corosync stopped and masked, no expected
// vote count ever set, delnode completed from the survivor — the departed node's
// copies were still byte-identical to the live cluster's:
//
//	token.cfg        40edfa5c…  ==  40edfa5c…
//	authkey.key      21303244…  ==  21303244…
//	pve-root-ca.key  b311e385…  ==  b311e385…
//
// Quorum does not protect them. It gates WRITES to pmxcfs; starting pve-cluster
// alone mounts the tree from the local config.db and serves every byte of it.
// So there is no arrangement of vote counts that keeps a departed node from
// reading what it already has, and the only question left is an operator's.
const (
	// CredentialsRotate: the departed node is not trusted, so it leaves with a
	// fresh identity — its own root keypair replaced, sharing nothing with the
	// cluster it left.
	//
	// Acts on the DEPARTING node, not on the remaining cluster. Rotating the
	// cluster's own secrets invalidates every session in the fleet, reissues
	// every node certificate, and replaces the token this control plane uses to
	// reach it: a fleet-wide disruption that one node leaving must not trigger.
	// It remains true that the departed node has already READ the cluster's
	// secrets and no departure can unsee that — closing it is a decision an
	// operator makes about the cluster, at a time of their choosing.
	CredentialsRotate = "rotate"

	// CredentialsAcceptExposure: the node is trusted and is being redeployed, so
	// its copy is destroyed and it keeps the identity it had. A deliberate
	// choice, recorded as one.
	CredentialsAcceptExposure = "accept-exposure"
)

// DepartureRefusal is one reason a host may not leave its hive.
type DepartureRefusal struct {
	// Code identifies the rule. It is the stable half — like an HTTP status, it
	// is what a client switches on and what an operator learns to recognise.
	Code string

	// Summary states the rule GENERICALLY, with no numbers and no host in it.
	//
	// It exists because the same rule refuses many hosts with different figures.
	// Grouping refusals by code to avoid repeating them means picking one of
	// those details to display, and "the provider reports 2 guest(s)" shown for a
	// group that also contains a host with five is not a shorter message — it is
	// a wrong one. The summary is the half that is true of every host the rule
	// refused, so it can be shown once.
	Summary string

	// Detail states what is wrong ON THIS HOST, and names the offending number.
	// "3 guests are still on this node" is actionable; "not drained" sends
	// somebody to the provider's UI to work out why.
	//
	// Per host, so it cannot be grouped — see Summary.
	Detail string

	// Remedy states WHAT TO DO, separately.
	//
	// Separate because it is a different kind of sentence with a different
	// audience and a different place on screen. Nobody needs "401 means send
	// credentials" inline with every 401; they need it once, where they are
	// looking for what to do next. Folding the two together makes the common case
	// — an operator who already knows this rule — read the same instruction every
	// time, and makes the message too long to render beside the thing it is about.
	//
	// It is also the half that is easy to lose. Trimming a message for width
	// takes the remedy first, because the statement of the problem feels like the
	// important part — and then the non-obvious step goes with it. Giving it its
	// own field means shortening the message cannot silently delete it.
	Remedy string
}

func (r DepartureRefusal) Error() string { return r.Code + ": " + r.Detail }

// DepartureRequest is everything the gate decides on. Every field is supplied
// by the caller, and the gate reads nothing else — no clock, no database — so
// the policy is inspectable and the same inputs always give the same verdict.
type DepartureRequest struct {
	Hive       Hive
	Hypervisor Hypervisor

	// Drain is the derived state from DrainStateAt, computed by the caller in the
	// transaction it counted placements in.
	Drain DrainStatus

	// HiveNodeCount is how many nodes the hive has RIGHT NOW, including the one
	// leaving. Counted from the synced hypervisor rows rather than from the
	// cluster's own opinion, because that is the count the rest of this system
	// acts on.
	HiveNodeCount int

	// CredentialDisposition is what to do about the secrets this node has seen,
	// and it has NO DEFAULT on purpose.
	//
	// Every other field here describes the cluster; this one is a judgement about
	// trust, and the two safe answers cost different things — rotating disrupts
	// every API client in the hive, accepting leaves a departed host able to mint
	// tickets for it. Picking either silently would be this system deciding, on
	// an operator's behalf, which of those they can afford. An empty value
	// refuses, in the same spirit as every other gate here: a check that cannot
	// reach a decision denies rather than guesses.
	CredentialDisposition string

	// RemainingNodes is how many members the cluster will still have once this
	// host and every other departure already requested have left. The departure
	// minimum is a floor on this number; see repo.HiveMembers.RemainingAfter.
	RemainingNodes int

	// NotCounted names the hive's other machines left out of RemainingNodes, each
	// with why, so a refusal explains a count lower than the node list shows.
	NotCounted []string
}

// CheckDeparture returns every reason the host may not leave, or nil.
//
// EVERY reason, not the first. An operator who has to fix five preconditions
// discovers them one job failure at a time otherwise, and each round trip is a
// worker run against a production cluster. The list is also the honest answer:
// "not cordoned AND four guests present" is one situation, not two sequential
// problems.
//
// A refusal is not an error condition of this system. It is the gate doing its
// job, and callers must not report it as a 500 — see the handler mapping.
// CheckDeparture is the gate at EXECUTION: every state precondition, plus the
// operator's decision about the secrets this node has seen.
//
// Split from CheckDepartureState because the two are answered in different
// places. The state gates describe the cluster and an operator fixes them by
// acting on it — cordon the host, move the guests, run a sync — so they belong
// in the preview, where they can be fixed before anything is attempted. The
// credential decision is not fixable and not pre-existing: it is a question
// asked at the moment of departure and answered in the same request.
//
// Evaluating it in the preview would refuse every host, always, and leave a
// Depart control that can never enable — reporting an unanswered question as a
// blocked cluster.
func CheckDeparture(req DepartureRequest) []DepartureRefusal {
	out := CheckDepartureState(req)

	// Only when the state gates are otherwise satisfied. A host with four guests
	// on it is not yet at the point where its credentials are the question, and
	// stacking an unanswerable item onto a list of fixable ones buries them.
	if len(out) > 0 {
		return out
	}
	switch req.CredentialDisposition {
	case CredentialsAcceptExposure:
	case CredentialsRotate:
	default:
		out = append(out, DepartureRefusal{
			Code:    RefuseCredentialsUndecided,
			Summary: "no decision has been recorded about the cluster secrets this host has seen",
			Detail: "this host holds the cluster's token.cfg, its ticket-signing authkey.key " +
				"and its root CA private key, because /etc/pve/priv is shared by every node; " +
				"departure destroys its copy but cannot make it un-seen",
			Remedy: "Choose one. Rotate, if this host is not trusted — it can otherwise mint " +
				"auth tickets for the cluster it left, which no SSH key removal prevents. " +
				"Accept the exposure, if it is trusted and being redeployed. There is no " +
				"safe default: rotating disrupts every API client in the fleet, and accepting " +
				"leaves a credential behind.",
		})
	}
	return out
}

// CheckDepartureState returns every reason the CLUSTER is not ready for this
// host to leave. It is what the preview shows, and it reads nothing about
// credentials — see CheckDeparture.
func CheckDepartureState(req DepartureRequest) []DepartureRefusal {
	var out []DepartureRefusal

	// Membership first. Everything below is a statement about a node in a hive,
	// and for a fleetless host the honest answer is that there is nothing to
	// leave — not that it is insufficiently drained.
	// Nil pointer, not uuid.Nil: Hypervisor.HiveID became nullable when the row
	// became the durable machine record, and "in no cluster" is now a normal
	// state rather than an unset value that happened to read as the zero UUID.
	if req.Hypervisor.HiveID == nil || req.HiveNodeCount == 0 {
		return []DepartureRefusal{{
			Code:    RefuseNotAMember,
			Summary: "the host belongs to no cluster",
			Detail:  "this host does not belong to a cluster",
			Remedy:  "Nothing to do — it is already standalone.",
		}}
	}

	// Already on its way out. Answered alone, like not_a_member: every gate below
	// was passed when the departure was requested, and passing them again would
	// re-record it — resetting the age that shows a stalled departure, replacing
	// the operator's credential decision, and re-snapshotting the root key the
	// rotation check compares against.
	if d := req.Hypervisor.DepartingAt; d != nil {
		return []DepartureRefusal{{
			Code:    RefuseAlreadyDeparting,
			Summary: "a departure has already been requested for this host",
			Detail: fmt.Sprintf("its departure was requested %s ago and has not finished",
				time.Since(*d).Truncate(time.Minute)),
			Remedy: "Wait for a remaining member to remove it from the cluster. A departure " +
				"that does not finish usually means no remaining member is quorate to evict it.",
		}}
	}

	// The hive-size gate runs even when the node is not drained, because it is
	// the one refusal an operator cannot resolve by waiting — it needs another
	// host added, or the knob changed deliberately. Reporting it only after the
	// drain completes wastes the whole drain.
	//
	// A FLOOR: the cluster may not be LEFT with fewer than the minimum.
	//
	// ONE EXCEPTION: leaving nobody, under a minimum of one. That is not a
	// cluster shrunk below its floor but a cluster dismantled — there is no
	// survivor whose quorum could suffer — and it is only reachable on a hive
	// whose operator lowered the minimum to one on purpose.
	min := req.Hive.EffectiveMinimumNodes()
	dismantling := req.RemainingNodes == 0 && min <= 1
	if req.RemainingNodes < min && !dismantling {
		out = append(out, DepartureRefusal{
			Code:    RefuseHiveTooSmall,
			Summary: "the departure would leave the cluster below its minimum",
			Detail:  tooSmallDetail(req, min),
			Remedy: "Add a node first, wait for a node that is arriving to finish joining, or " +
				"lower this cluster's minimum deliberately if it is a test cluster. Shrinking a " +
				"cluster this far risks quorum and leaves the survivors read-only with their " +
				"guests unmanageable if another node fails.",
		})
	}

	if !req.Drain.Cordoned {
		out = append(out, DepartureRefusal{
			Code:    RefuseNotCordoned,
			Summary: "the host still accepts placements",
			Detail:  "this host still accepts placements",
			Remedy: "Cordon it, so the oracle stops landing new guests on a node that is " +
				"about to leave.",
		})
	}

	switch req.Drain.State {
	case DrainStateUnknown:
		// The project's standing rule: a check that cannot reach a decision
		// denies. Here the cost of the other choice is concrete — dismantling the
		// cluster membership of a node that turns out to have been running
		// workloads all along.
		//
		// WHY it cannot decide is a separate question from THAT it cannot, and
		// they have different fixes. An old observation and no observation at all
		// arrive here identically, because both are read off one timestamp — so
		// this used to report every silence as a stale sync and send the operator
		// to run one. On a freshly rebuilt host that advice is unfollowable: there
		// is no agent on it to sync, and the age it quotes is just how long the
		// machine has been standing there unobserved. Seen on a lab fleet rebuilt
		// from scratch, where every node refused with a stale sync it had never
		// had.
		//
		// The distinction is already in hand: the agent is what observes a host,
		// and a machine with no agent linked has never had an observer rather than
		// a lapsed one.
		//
		// Same code either way. It is a stable string that a client switches on,
		// and this is one rule with two causes, not two rules.
		if req.Hypervisor.BeeID == nil {
			out = append(out, DepartureRefusal{
				Code:    RefuseDrainUnknown,
				Summary: "no agent is observing the host, so nothing can say whether it is empty",
				Detail: "no agent has enrolled on this node, so nothing has ever reported " +
					"whether it is empty",
				Remedy: "Onboard it, so it has an agent to report with. Refusing rather than " +
					"assuming: a node nothing is watching is not the same as an empty one.",
			})
			break
		}
		out = append(out, DepartureRefusal{
			Code:    RefuseDrainUnknown,
			Summary: "the host's agent has not reported recently enough to tell whether it is empty",
			Detail: fmt.Sprintf("this node's last report is %s old, too stale to tell whether "+
				"it is empty", req.Drain.ObservationAge.Truncate(time.Second)),
			Remedy: "Check the agent is running and reaching the control plane, then retry. " +
				"Refusing rather than assuming: a node that looks empty in a stale " +
				"observation may not be.",
		})
	default:
		// Reported separately even though both mean "not empty". They are
		// different facts with different owners and different fixes: guests are
		// moved with the provider's tooling, held placements are released through
		// this system. Collapsing them into one message tells the operator to look
		// in the wrong place half the time.
		if req.Drain.ObservedGuests > 0 {
			out = append(out, DepartureRefusal{
				Code:    RefuseGuestsPresent,
				Summary: "the provider still reports guests on the host, stopped ones included",
				Detail: fmt.Sprintf("the provider reports %d guest(s) on this node, "+
					"stopped ones included", req.Drain.ObservedGuests),
				// The second sentence is the non-obvious half and the reason this
				// field exists. Delete the guests and retry immediately and the gate
				// refuses again, because it reads the last observation rather than
				// the provider — and the operator concludes the gate is broken.
				Remedy: "Move or delete them, then wait for a sync to observe the result. " +
					"A stopped guest blocks a cluster join exactly as a running one does.",
			})
		}
		if req.Drain.HeldPlacements > 0 {
			out = append(out, DepartureRefusal{
				Code:    RefusePlacementsHeld,
				Summary: "placements on the host still hold capacity",
				Detail: fmt.Sprintf("%d placement(s) on this node still hold capacity "+
					"(pending or active)", req.Drain.HeldPlacements),
				Remedy: "Release them here, not with the provider's tooling — otherwise the " +
					"ledger keeps this node's capacity committed after it has left.",
			})
		}
	}

	return out
}

// DepartureRefusalsError renders a refusal list as one error.
//
// Every reason in one string rather than the first: the caller turning this into
// a job failure or an API response must not silently drop the rest, because the
// operator's next action depends on the whole list.
func DepartureRefusalsError(rs []DepartureRefusal) error {
	if len(rs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		parts = append(parts, r.Code+" ("+r.Detail+")")
	}
	return fmt.Errorf("departure refused: %s", strings.Join(parts, "; "))
}

// tooSmallDetail says what the remaining count is and, when the hive has
// machines it left out, which and why. The node list shows every machine
// assigned to the hive, so "would be left with 2" beside four listed nodes
// is only actionable once it names the two that were not counted.
func tooSmallDetail(req DepartureRequest, min int) string {
	d := fmt.Sprintf("hive %q would be left with %d node(s) and its minimum is %d",
		req.Hive.Name, req.RemainingNodes, min)
	if len(req.NotCounted) > 0 {
		d += "; not counted: " + strings.Join(req.NotCounted, ", ")
	}
	return d
}
