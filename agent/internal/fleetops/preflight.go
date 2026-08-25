// Package fleetops implements the tasks a bee performs on a Proxmox host.
//
// Every operation here can take a hypervisor out of service, and two of them can
// take the CLUSTER out of service if run at the wrong moment. So each starts
// with a pre-flight that refuses rather than proceeds, and the refusals are the
// most important code in this package.
//
// The rule throughout: a check that cannot reach a decision REFUSES. A cluster
// whose status cannot be read is not a cluster that is fine.
package fleetops

import (
	"context"
	"fmt"
	"strings"

	"github.com/glueops/apiary/pkg/proxmox"
)

// Cluster is the slice of the Proxmox client this package needs. An interface so
// the pre-flights — which are the part worth testing — can be tested without a
// hypervisor.
type Cluster interface {
	Status(ctx context.Context) error
	Quorate() bool
	MemberCount() int
	NodeNames() []string
	GuestsOn(ctx context.Context, node string) ([]Guest, error)
	DeleteConfigNode(ctx context.Context, node string) error
	JoinCluster(ctx context.Context, opts *proxmox.ClusterJoinOptions) (string, error)
}

// Guest is a VM or container, reduced to what a fleet decision needs.
type Guest struct {
	ID     string
	Name   string
	Type   string // qemu | lxc
	Status string // running | stopped
}

func (g Guest) running() bool { return strings.EqualFold(g.Status, "running") }

// quorumSurvivesLosing reports whether the cluster still has quorum with n
// fewer voting members.
//
// Corosync needs a strict majority of expected votes. Losing a node during a
// rotation is routine; losing the one that takes the cluster below majority
// makes every remaining node read-only, and recovering means manual
// intervention on hardware that is now refusing to cooperate.
//
// Computed rather than asked, because the API reports quorum for the CURRENT
// membership, and the question here is about a membership that does not exist
// yet.
func quorumSurvivesLosing(members, losing int) bool {
	remaining := members - losing
	if remaining < 1 {
		return false
	}
	// Majority of the ORIGINAL expected votes: corosync does not lower
	// expected_votes on its own when a node goes away, only when it is removed
	// from the config.
	return remaining*2 > members
}

// PreflightReport is what a refusal explains. Returned rather than logged so the
// executor can put every reason in the task's output at once — an operator
// retrying a drain wants all the blockers, not the first one.
type PreflightReport struct {
	Blockers []string
	Notes    []string
}

func (r *PreflightReport) blocked(format string, args ...any) {
	r.Blockers = append(r.Blockers, fmt.Sprintf(format, args...))
}

func (r *PreflightReport) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

func (r *PreflightReport) err(op string) error {
	if len(r.Blockers) == 0 {
		return nil
	}
	return fmt.Errorf("%s refused:\n  - %s", op, strings.Join(r.Blockers, "\n  - "))
}

// preflightReboot checks whether this node can be taken down.
func preflightReboot(ctx context.Context, cl Cluster, node string, allowGuestDowntime bool) (*PreflightReport, error) {
	r := &PreflightReport{}

	if err := cl.Status(ctx); err != nil {
		// Standalone hosts are legitimate: not every Proxmox install is a
		// cluster. But an error that is NOT "no cluster" means the state is
		// unknown, and unknown must not read as safe.
		if !isNotClustered(err) {
			return nil, fmt.Errorf("read cluster status: %w", err)
		}
		r.note("this host is not in a cluster; quorum is not a consideration")
	} else {
		if !cl.Quorate() {
			r.blocked("the cluster is not quorate right now; rebooting a node while quorum is " +
				"already lost is how a recoverable outage becomes a manual one")
		}
		if members := cl.MemberCount(); !quorumSurvivesLosing(members, 1) {
			r.blocked("removing this node from the voting set leaves %d of %d members, which is "+
				"not a majority; the cluster would go read-only", members-1, members)
		}
	}

	guests, err := cl.GuestsOn(ctx, node)
	if err != nil {
		return nil, fmt.Errorf("list guests on %s: %w", node, err)
	}
	var running []string
	for _, g := range guests {
		if g.running() {
			running = append(running, fmt.Sprintf("%s/%s (%s)", g.Type, g.ID, g.Name))
		}
	}
	switch {
	case len(running) == 0:
		r.note("no guests are running on this host")
	case allowGuestDowntime:
		r.note("%d guest(s) will lose service: %s", len(running), strings.Join(running, ", "))
	default:
		r.blocked("%d guest(s) are still running here: %s. Migrate them first, or set "+
			"allow_guest_downtime if this host is expected to take them down with it",
			len(running), strings.Join(running, ", "))
	}

	return r, nil
}

// isNotClustered distinguishes "this host is standalone" from "the cluster could
// not be read". Only the first is safe to continue past.
func isNotClustered(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "no cluster") ||
		strings.Contains(s, "not a cluster member") ||
		strings.Contains(s, "cluster not ready")
}
