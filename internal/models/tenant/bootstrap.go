package tenant

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Bootstrap refusal codes. Stable strings, same contract as the arrival and
// departure ones.
const (
	RefuseBootstrapInHive        = "already_in_hive"
	RefuseBootstrapHiveNotEmpty  = "hive_not_empty"
	RefuseBootstrapUnprovisioned = "not_provisioned"
)

// BootstrapRefusal is one reason a host may not become a hive's first node.
type BootstrapRefusal struct {
	Code   string
	Detail string
}

func (r BootstrapRefusal) Error() string { return r.Code + ": " + r.Detail }

// BootstrapRequest is everything the bootstrap gate decides on.
type BootstrapRequest struct {
	// TargetHive is the hive being created on this host.
	TargetHive Hive

	// CurrentHiveID is what the host currently reports. Nil is the only
	// acceptable value: `pvecm create` refuses a node that is already clustered.
	CurrentHiveID *uuid.UUID

	// HiveNodeCount is how many nodes the target hive already has. Anything above
	// zero means the cluster exists and this is an arrival, not a bootstrap.
	HiveNodeCount int

	// ProvisioningState of the host. Kept for the message, no longer gated on —
	// see the refusal below.
	ProvisioningState string

	// AgentOnline is whether this host's agent is currently live.
	//
	// This is the precondition that actually matters, because the agent is what
	// creates the cluster: it derives the verb from what corosync reports and
	// runs `pvecm create` locally. The control plane opens no connection.
	AgentOnline bool
}

// CheckBootstrap returns every reason the host may not create this hive, or nil.
//
// Every reason at once, for the same purpose as CheckArrival and CheckDeparture:
// an operator fixing preconditions one job failure at a time spends a worker run
// against real metal per round trip.
//
// NOTE WHAT IS NOT GATED HERE, because it is the difference between this and
// arrival and the reason the two cannot share a gate.
//
// GUESTS DO NOT BLOCK A BOOTSTRAP. `pvecm add` refuses a node carrying any guest
// at all, stopped ones included, because the joining node's configuration is
// replaced by the cluster's and its guests would be orphaned. `pvecm create`
// replaces nothing — the node becomes a single-node cluster and keeps
// everything it was already running. Refusing on guests here would make it
// impossible to adopt an existing Proxmox host into this system without first
// emptying it, which is the opposite of what adoption means.
//
// THERE IS NO MINIMUM SIZE. A hive's minimum_nodes governs departure, where
// shrinking below quorum strands the survivors. A cluster has to start at one
// node; requiring three would make every hive unreachable from the beginning.
func CheckBootstrap(req BootstrapRequest) []BootstrapRefusal {
	var out []BootstrapRefusal

	if req.CurrentHiveID != nil {
		out = append(out, BootstrapRefusal{
			Code: RefuseBootstrapInHive,
			Detail: fmt.Sprintf(
				"this host already belongs to hive %s; a node cannot create a cluster while it "+
					"is a member of one. Depart it first — the fleetless state between is a real "+
					"state, and creating a cluster on a node another cluster still counts as a "+
					"member would leave two clusters believing they own it",
				*req.CurrentHiveID),
		})
	}

	if req.HiveNodeCount > 0 {
		out = append(out, BootstrapRefusal{
			Code: RefuseBootstrapHiveNotEmpty,
			Detail: fmt.Sprintf(
				"hive %q already has %d node(s), so the cluster exists; join this host to it "+
					"instead of creating it again. Running `pvecm create` against an existing "+
					"cluster name does not join anything — it makes a second, separate cluster "+
					"that happens to share a name",
				req.TargetHive.Name, req.HiveNodeCount),
		})
	}

	// THE PRECONDITION IS A LIVE AGENT, not a provisioning state.
	//
	// This used to refuse any host that was not `provisioned`, because "every
	// step of a bootstrap runs over SSH as the control plane, and a
	// half-installed host is one where that access is not yet established".
	// Neither half of that is true any more: the agent creates the cluster
	// itself, deriving the verb from what corosync reports and running `pvecm
	// create` locally, and the control plane opens no connection at all.
	//
	// Gating on the old state refused a host whose agent was enrolled and
	// heartbeating because a LATER onboarding step — proving the control plane's
	// own managed SSH key — had failed on a reconnect race. That failure is worth
	// knowing about, since the managed key is the fallback for a host whose agent
	// dies, but it says nothing about whether this machine can form a cluster.
	// It is reported on the host rather than blocking the operation.
	if !req.AgentOnline {
		out = append(out, BootstrapRefusal{
			Code: RefuseBootstrapUnprovisioned,
			Detail: "this host has no live agent, and the agent is what creates the cluster — " +
				"it reads what corosync reports and runs `pvecm create` locally. Wait for it to " +
				"check in, or finish onboarding if it has not been installed yet" +
				provisioningSuffix(req.ProvisioningState),
		})
	}

	return out
}

// BootstrapRefusalsError renders a refusal list as one error.
func BootstrapRefusalsError(rs []BootstrapRefusal) error {
	if len(rs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		parts = append(parts, r.Error())
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}

// provisioningSuffix names the onboarding state when it is not clean, so a host
// that never finished onboarding still says so — as context on the real
// refusal rather than as the refusal itself.
func provisioningSuffix(state string) string {
	if state == "" || state == OnboardProvisioned {
		return "."
	}
	return fmt.Sprintf(" (onboarding is %q).", state)
}
