package fleetops

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		// Names the SETTINGS, because that is what is missing. Saying only that a
		// token is required sent an operator to check the permissions of a token
		// that was fully privileged and simply never given to this agent.
		return nil, fmt.Errorf("fleetops: no Proxmox API token is configured for this host " +
			"(set proxmox_token_id and proxmox_secret in /etc/apiary-agent/config.yaml, or " +
			"AGENT_PROXMOX_TOKEN_ID and AGENT_PROXMOX_SECRET in /etc/apiary-agent/agent.env); " +
			"a fleet agent should hold a scoped token, not the root password")
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
	return proxmox.NewClient(baseURL,
		proxmox.WithAPIToken(tokenID, secret),
		proxmox.WithHTTPClient(localAPIClient()),
	), nil
}

// pveRootCAPath is the certificate authority a Proxmox node issues its own API
// certificate from. A var so a test can point it at a fixture.
var pveRootCAPath = "/etc/pve/pve-root-ca.pem"

// localAPIClient trusts THIS node's own Proxmox CA.
//
// The agent talks to https://127.0.0.1:8006, and that certificate is issued by a
// CA which exists only on the node — it is not in any system trust store. So
// verification failed with
//
//	tls: failed to verify certificate: x509: certificate signed by unknown authority
//
// and every task that reads cluster state failed with it, including the
// dist-upgrade's post-reboot check. The certificate itself is fine: its SANs
// cover 127.0.0.1, localhost and the node's own name and address.
//
// READ FROM DISK on every client, not cached and not installed into the system
// store, because the CA changes underneath a running agent more often than it
// looks: `pvecm updatecerts` reissues it, a departure mints a fresh one so the
// node leaves sharing nothing, and joining a cluster replaces it with that
// cluster's. Anything that copied it once would be stale after each of those.
//
// Falls back to the system roots when the file cannot be read, rather than
// disabling verification. A missing CA is not a reason to start trusting
// anything that answers on the port.
func localAPIClient() *http.Client {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

	pem, err := os.ReadFile(pveRootCAPath)
	if err != nil {
		slog.Warn("could not read this node's Proxmox CA, so the local API is verified against "+
			"the system trust store, which does not contain it",
			"path", pveRootCAPath, "error", err)
		return &http.Client{Timeout: 30 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsCfg}}
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		slog.Warn("this node's Proxmox CA could not be parsed", "path", pveRootCAPath)
		return &http.Client{Timeout: 30 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsCfg}}
	}
	tlsCfg.RootCAs = pool
	return &http.Client{Timeout: 30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg}}
}
