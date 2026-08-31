package fleetops

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Paths this reads. Vars so tests can point them at a fixture tree.
var (
	// MembersPath is pmxcfs's own summary of the cluster, and it is the whole
	// observation in one file: this node's name, the cluster's name, whether it
	// is quorate, and every member with its address and online state.
	MembersPath = "/etc/pve/.members"

	// NodesDir holds every node's guest configuration, replicated by pmxcfs — so
	// this host can enumerate ANOTHER node's guests without asking it, which is
	// exactly what an eviction has to do for a node that may be down.
	NodesDir = "/etc/pve/nodes"

	// CorosyncConf existing is the second standalone signal, cross-checked
	// against .members rather than trusted alone.
	CorosyncConf = "/etc/corosync/corosync.conf"

	// RunQemuDir holds a pid file per running VM. Running state is not in the
	// configuration, and it is only knowable for THIS host.
	RunQemuDir = "/run/qemu-server"
	RunLXCDir  = "/run/lxc"
)

// localCluster answers every cluster question from the local filesystem and
// pvecm, with no Proxmox API token.
//
// THE AGENT RUNS AS ROOT ON THE NODE, so authenticating to that node's own API
// over TLS to ask what it already knows locally was never buying anything — and
// it cost a great deal. Every one of these was a real failure on this fleet:
//
//   - a departed node serves a certificate from a CA it no longer holds, so its
//     own agent could not read its own cluster state ("certificate verify
//     failed") at exactly the moment that state mattered most;
//   - pveproxy takes minutes to come back after a reboot, so a post-upgrade check
//     had to wait for a service it did not need;
//   - `pvecm updatecerts` reissues certificates, so a join or a departure needed
//     pveproxy and pvedaemon restarted underneath the agent;
//   - API tokens live in /etc/pve/priv/token.cfg on pmxcfs, so a join destroys
//     them wholesale and a departure carries the cluster's copy away — which
//     forced the credential to be the HIVE's, and then forced it to be
//     distributed to every host, and then quietly failed to reach any host that
//     FORMED a cluster rather than joining one;
//   - and a host that never received one could do nothing at all.
//
// None of those exist here. /etc/pve is a filesystem this process can read, and
// pvecm is a command it can run. The codebase already knew this — CountLocalGuests
// reads /etc/pve "so it survives a broken pveproxy" — the reasoning simply never
// reached the cluster calls.
type localCluster struct {
	runner Runner

	// members is the last read, held so Quorate, MemberCount, NodeNames and
	// ClusterName all describe the same moment. Reading each separately can
	// report a node that is in a cluster, has no members, and is quorate — three
	// facts that were each true and never true together.
	members *membersFile
}

// NewLocalCluster reads this host's cluster state from its own filesystem.
func NewLocalCluster(r Runner) Cluster { return &localCluster{runner: r} }

// membersFile is pmxcfs's .members, decoded.
type membersFile struct {
	NodeName string `json:"nodename"`
	Cluster  struct {
		Name    string `json:"name"`
		Nodes   int    `json:"nodes"`
		Quorate int    `json:"quorate"`
	} `json:"cluster"`
	NodeList map[string]struct {
		ID     int    `json:"id"`
		Online int    `json:"online"`
		IP     string `json:"ip"`
	} `json:"nodelist"`
}

// Status reads this host's own view of its cluster.
//
// A MISSING cluster name is the standalone signal and returns errNotClustered,
// which callers already distinguish from a failed read — the same rule the API
// client used, where a successful status with no cluster entry meant standalone.
// The file being absent is a different thing entirely: pmxcfs is not mounted, so
// this is either not a Proxmox node or pve-cluster is not running, and that is a
// genuine failure rather than an observation.
func (l *localCluster) Status(_ context.Context) error {
	raw, err := os.ReadFile(MembersPath)
	if err != nil {
		l.members = nil
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist, so pmxcfs is not mounted here; "+
				"this host is either not a Proxmox node or pve-cluster is not running", MembersPath)
		}
		return fmt.Errorf("read %s: %w", MembersPath, err)
	}
	var m membersFile
	if err := json.Unmarshal(raw, &m); err != nil {
		l.members = nil
		return fmt.Errorf("parse %s: %w", MembersPath, err)
	}
	l.members = &m
	if strings.TrimSpace(m.Cluster.Name) == "" {
		return errNotClustered
	}
	return nil
}

// NodeName is what pmxcfs calls this host.
//
// Read from pmxcfs rather than taken from configuration. PVE derives the node
// name from the hostname at install and maps it to the management address in
// /etc/hosts, so they agree — but this is the value the rest of the cluster
// keys on, so reading it removes the need to trust that the agent was
// configured correctly.
func (l *localCluster) NodeName() string {
	if l.members == nil {
		return ""
	}
	return l.members.NodeName
}

func (l *localCluster) ClusterName() string {
	if l.members == nil {
		return ""
	}
	return l.members.Cluster.Name
}

func (l *localCluster) Quorate() bool {
	return l.members != nil && l.members.Cluster.Quorate == 1
}

func (l *localCluster) MemberCount() int {
	if l.members == nil {
		return 0
	}
	return len(l.members.NodeList)
}

// NodeNames is every member, sorted.
//
// Sorted because callers derive an eviction OWNER from the lowest name, and a
// map's iteration order is deliberately random in Go — an unsorted answer would
// make different members reach different conclusions from identical state.
func (l *localCluster) NodeNames() []string {
	if l.members == nil {
		return nil
	}
	out := make([]string, 0, len(l.members.NodeList))
	for n := range l.members.NodeList {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// OnlineNodeNames is the members corosync currently counts as present.
//
// Read from the same .members snapshot as everything else, so it describes the
// same moment. A node configured but down appears in NodeNames and not here,
// which is the distinction quorum turns on.
func (l *localCluster) OnlineNodeNames() []string {
	if l.members == nil {
		return nil
	}
	out := make([]string, 0, len(l.members.NodeList))
	for n, e := range l.members.NodeList {
		if e.Online == 1 {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// GuestsOn lists the VMs and containers registered to a node.
//
// From /etc/pve, which pmxcfs replicates, so this answers for ANY node in the
// cluster including one that is powered off — which is precisely the case an
// eviction cares about, and the reason the API version asked the cluster rather
// than the node.
//
// STATUS IS ONLY KNOWN FOR THIS HOST. Whether a guest is running is runtime
// state, not configuration, so it is not replicated. Callers that need it ask
// about themselves — the reboot preflight does — and the one caller that asks
// about another node cares only that guests are REGISTERED, because what an
// eviction risks is orphaning their configuration, not interrupting them.
func (l *localCluster) GuestsOn(_ context.Context, node string) ([]Guest, error) {
	if node == "" {
		return nil, fmt.Errorf("no node name was given")
	}
	base := filepath.Join(NodesDir, node)
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			// A node with no directory has no guests. Not an error: a node that
			// has just joined legitimately has none yet.
			return nil, nil
		}
		return nil, fmt.Errorf("read the guest configuration for %s: %w", node, err)
	}

	self := strings.EqualFold(node, l.NodeName())
	var out []Guest
	for _, kind := range []struct{ dir, typ string }{
		{"qemu-server", "qemu"},
		{"lxc", "lxc"},
	} {
		entries, err := os.ReadDir(filepath.Join(base, kind.dir))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s guests for %s: %w", kind.typ, node, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".conf")
			g := Guest{ID: id, Name: id, Type: kind.typ}
			if self {
				g.Status = l.runState(kind.typ, id)
			}
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// runState reports whether a guest on THIS host is running.
//
// From the runtime directories rather than by shelling out to `qm list`: this is
// consulted before a reboot, and a check that has to fork two commands and parse
// their tables has more ways to fail than a stat.
func (l *localCluster) runState(typ, id string) string {
	var probe string
	switch typ {
	case "qemu":
		probe = filepath.Join(RunQemuDir, id+".pid")
	case "lxc":
		probe = filepath.Join(RunLXCDir, id)
	default:
		return ""
	}
	if _, err := os.Stat(probe); err == nil {
		return "running"
	}
	return "stopped"
}

// IsStandalone reports whether this host is in no cluster, cross-checking
// pmxcfs against corosync.
//
// Both, because they fail differently. .members can carry a stale cluster name
// while corosync.conf is already gone mid-departure, and corosync.conf can exist
// on a node whose pve-cluster is not running. Standalone is only asserted when
// neither says otherwise.
func (l *localCluster) IsStandalone(ctx context.Context) bool {
	if err := l.Status(ctx); err != nil && !isNotClustered(err) {
		return false
	}
	if l.ClusterName() != "" {
		return false
	}
	_, err := os.Stat(CorosyncConf)
	return os.IsNotExist(err)
}

// discard swallows command output. The cluster writes below are short and their
// result is verified by re-reading .members afterwards, so streaming their
// output into a task log adds noise rather than diagnosis.
func discard(string, string, ...any) {}

// CreateCluster forms a cluster with this host as its first member.
//
// `pvecm create` is a local command run as root. It needs no credential, which
// is the whole point: the node that FORMS a cluster is by definition the one
// that has no cluster credential yet, and requiring one made this the single
// path that could never work. Two lab nodes formed their clusters and were left
// permanently unable to run any fleet task because of it.
func (l *localCluster) CreateCluster(ctx context.Context, opts *proxmoxCreateOptions) (string, error) {
	if opts == nil || strings.TrimSpace(opts.ClusterName) == "" {
		return "", fmt.Errorf("a cluster name is required")
	}
	args := []string{"create", opts.ClusterName}
	args = append(args, linkArgs(opts.Link0, opts.Link1, opts.Link2, opts.Link3)...)
	if err := l.runner.Run(ctx, discard, "pvecm", args...); err != nil {
		return "", fmt.Errorf("pvecm create %s: %w", opts.ClusterName, err)
	}
	return "cluster created", nil
}

// JoinCluster adds this host to an existing cluster.
//
// --use_ssh, which is what makes this tokenless. pvecm authenticates to the peer
// over SSH using PVE's OWN root key, which pvecm itself manages under
// /root/.ssh. That is exactly why the fleet's key lives in
// /etc/ssh/authorized_keys.d/apiary instead: PVE owns its own lifecycle there
// and is destructive on it, so keeping the control plane's access outside that
// boundary means a join cannot cost us the repair channel into the host.
//
// The fingerprint is still required. Without one pvecm PROMPTS, which
// non-interactively hangs until something kills it and reports a timeout that
// says nothing about the cause.
func (l *localCluster) JoinCluster(ctx context.Context, opts *proxmoxJoinOptions) (string, error) {
	if opts == nil || strings.TrimSpace(opts.Hostname) == "" {
		return "", fmt.Errorf("a peer to join through is required")
	}
	if strings.TrimSpace(opts.Fingerprint) == "" {
		return "", fmt.Errorf("the peer's certificate fingerprint is required; without it the " +
			"join cannot tell the intended cluster from anything else answering on that address")
	}
	args := []string{"add", opts.Hostname, "--use_ssh", "--fingerprint", opts.Fingerprint}
	if opts.Force != nil && *opts.Force {
		args = append(args, "--force")
	}
	args = append(args, linkArgs(opts.Link0, opts.Link1, opts.Link2, opts.Link3)...)
	if err := l.runner.Run(ctx, discard, "pvecm", args...); err != nil {
		return "", fmt.Errorf("pvecm add %s: %w", opts.Hostname, err)
	}
	return "join accepted", nil
}

// DeleteConfigNode removes another node from the cluster.
func (l *localCluster) DeleteConfigNode(ctx context.Context, node string) error {
	if strings.TrimSpace(node) == "" {
		return fmt.Errorf("no node name was given")
	}
	if err := l.runner.Run(ctx, discard, "pvecm", "delnode", node); err != nil {
		return fmt.Errorf("pvecm delnode %s: %w", node, err)
	}
	return nil
}

// Expected lowers this node's expected vote count.
//
// Only meaningful AFTER a departing node has stopped voting: corosync refuses to
// set expected_votes below the number of votes currently present, so this cannot
// be used as a preflight. It affects only the node it runs on, which under
// per-cluster convergence is the right node anyway — the one reconciling is the
// one that needs it lowered.
func (l *localCluster) Expected(ctx context.Context, votes int) error {
	if votes < 1 {
		return fmt.Errorf("expected votes must be at least 1, got %d", votes)
	}
	return l.runner.Run(ctx, discard, "pvecm", "expected", fmt.Sprint(votes))
}

// linkArgs renders the corosync ring flags pvecm accepts.
func linkArgs(links ...string) []string {
	var out []string
	for i, a := range links {
		if strings.TrimSpace(a) == "" {
			continue
		}
		out = append(out, fmt.Sprintf("--link%d", i), a)
	}
	return out
}
