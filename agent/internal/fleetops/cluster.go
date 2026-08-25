package fleetops

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glueops/apiary/pkg/proxmox"
)

// pveCluster adapts the Proxmox client to the narrow Cluster interface the
// pre-flights use.
//
// The adapter exists so the decisions — which are the dangerous part — are
// testable against a fake, and so this package depends on four questions rather
// than on the whole client surface.
type pveCluster struct {
	client  *proxmox.Client
	cluster *proxmox.Cluster
}

// NewCluster builds the adapter against a local Proxmox API.
func NewCluster(client *proxmox.Client) (Cluster, error) {
	if client == nil {
		return nil, fmt.Errorf("fleetops: a Proxmox client is required")
	}
	return &pveCluster{client: client}, nil
}

// Status refreshes the cached cluster state. Quorate and MemberCount read from
// that snapshot, so a caller gets one consistent view of a cluster rather than
// three answers taken at three moments.
func (p *pveCluster) Status(ctx context.Context) error {
	cl, err := p.client.Cluster(ctx)
	if err != nil {
		return err
	}
	p.cluster = cl
	return cl.Status(ctx)
}

func (p *pveCluster) Quorate() bool {
	return p.cluster != nil && p.cluster.Quorate == 1
}

func (p *pveCluster) MemberCount() int {
	if p.cluster == nil {
		return 0
	}
	return len(p.cluster.Nodes)
}

func (p *pveCluster) NodeNames() []string {
	if p.cluster == nil {
		return nil
	}
	out := make([]string, 0, len(p.cluster.Nodes))
	for _, n := range p.cluster.Nodes {
		out = append(out, n.Name)
	}
	return out
}

// GuestsOn lists VMs and containers registered to a node.
//
// It asks the CLUSTER rather than the node, so it still answers for a node that
// is down — which is exactly the case an eviction cares about.
func (p *pveCluster) GuestsOn(ctx context.Context, node string) ([]Guest, error) {
	if p.cluster == nil {
		cl, err := p.client.Cluster(ctx)
		if err != nil {
			return nil, err
		}
		p.cluster = cl
	}
	rs, err := p.cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, err
	}
	var out []Guest
	for _, r := range rs {
		if r.Node != node {
			continue
		}
		out = append(out, Guest{
			ID:     strconv.FormatUint(r.VMID, 10),
			Name:   r.Name,
			Type:   r.Type,
			Status: r.Status,
		})
	}
	return out, nil
}

func (p *pveCluster) DeleteConfigNode(ctx context.Context, node string) error {
	if p.cluster == nil {
		cl, err := p.client.Cluster(ctx)
		if err != nil {
			return err
		}
		p.cluster = cl
	}
	return p.cluster.DeleteConfigNode(ctx, node)
}

func (p *pveCluster) JoinCluster(ctx context.Context, opts *proxmox.ClusterJoinOptions) (string, error) {
	// Deliberately NOT reusing a cached cluster: a host that is joining is not
	// yet a member, so there is no cluster of its own to have cached.
	cl, err := p.client.Cluster(ctx)
	if err != nil {
		return "", err
	}
	return cl.JoinCluster(ctx, opts)
}

// NewLocalClient builds a Proxmox API client for the hypervisor this agent runs
// on.
//
// An API token, never a password: tokens are scoped, individually revocable, and
// do not carry the root account's full authority. A fleet agent needs
// Sys.Audit to read cluster state, Sys.Modify to remove a node, and VM.Audit to
// see what is registered where — a narrower set than root@pam, and the
// difference matters on a host an attacker who reached one hypervisor would
// otherwise use to reach the whole datacenter.
func NewLocalClient(baseURL, tokenID, secret string) (*proxmox.Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("fleetops: no Proxmox API URL configured for this host")
	}
	if tokenID == "" || secret == "" {
		return nil, fmt.Errorf("fleetops: a Proxmox API token id and secret are required; a fleet " +
			"agent should hold a scoped token, not the root password")
	}
	// The two halves are concatenated into "PVEAPIToken=<id>=<secret>", so a
	// secret that is itself a whole token string produces "<id>=<id>=<secret>".
	// Proxmox answers 401 to that and says nothing about why, which is a long
	// afternoon: the credential looks right everywhere it is displayed.
	//
	// A Proxmox token secret is a UUID, so neither character can appear in a
	// correct one — matching on them cannot reject anything valid.
	if strings.ContainsAny(secret, "=!") {
		return nil, fmt.Errorf("fleetops: the Proxmox token SECRET looks like a whole token string. " +
			"A token is written <user>@<realm>!<name>=<uuid>; the part before the '=' is the token id " +
			"and only the uuid after it is the secret. Passing the whole string builds a doubled " +
			"header that Proxmox rejects as unauthorized")
	}
	return proxmox.NewClient(baseURL, proxmox.WithAPIToken(tokenID, secret)), nil
}
