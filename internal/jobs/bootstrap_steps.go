package jobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
)

// AssertStandalone refuses a host that is already in a cluster.
//
// Checked against the HOST rather than against the control plane's record,
// because the two can disagree and only one of them is what `pvecm create` will
// act on. A machine whose row says fleetless while corosync says otherwise is
// exactly the case worth catching: creating a cluster on a node that is already
// a member either fails outright or, worse, succeeds and leaves two clusters
// each believing they own it.
func AssertStandalone(ctx context.Context, r Runner) error {
	name, err := ReadClusterName(ctx, r)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if name != "" {
		return fmt.Errorf("bootstrap: this host is already a member of cluster %q; the control "+
			"plane believed it belonged to none. Depart it, or reconcile the record, before "+
			"creating a cluster on it", name)
	}
	return nil
}

// CreateCluster runs `pvecm create` and verifies the result.
//
// The cluster NAME is the hive's name, and this is the one place the two are
// tied together. Everywhere else the hive's name is just a label — the arrival
// path deliberately compares against the target's own reported cluster rather
// than the hive name, because nothing enforces they match. Here the hive name is
// what the cluster is being called, so it must be a legal corosync cluster name
// rather than whatever an operator typed.
//
// Links are the same corosync rings arrival uses, and at least one is REQUIRED
// here for the same reason it is there: the ring address is not assumed to be
// the host's management address, because cluster traffic routinely belongs on a
// different network. A single-node cluster makes that look like a formality —
// right up to the moment a second node joins and inherits a ring pointing at
// whatever corosync guessed on the first.
func CreateCluster(
	ctx context.Context, r Runner, clusterName string, links []tenant.CorosyncLink,
) error {
	name := strings.TrimSpace(clusterName)
	if err := tenant.ValidateClusterName(name); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if err := tenant.ValidateCorosyncLinks(links); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	// Sorted for the same reason as the join: a stable command between attempts,
	// so a diff of two runs shows what changed rather than a reordering.
	var flags strings.Builder
	for _, l := range tenant.SortCorosyncLinks(links) {
		flag, value := l.Flag()
		flags.WriteString(" " + flag + " " + shellQuote(value))
	}

	out, err := r.Run(ctx, fmt.Sprintf("pvecm create %s%s 2>&1", shellQuote(name), flags.String()))
	if err != nil {
		return fmt.Errorf("bootstrap: pvecm create: %w (%s)", err, strings.TrimSpace(out))
	}

	// The command's own output is not the evidence, for the same reason it is not
	// on the join path: pvecm reports success before the local stack has
	// necessarily converged.
	return AssertClusterCreated(ctx, r, name)
}

// AssertClusterCreated verifies the node is now a quorate single-node cluster
// under the expected name.
//
// Three separate facts, and all three matter. The NAME catches a create that
// landed under something other than what was asked for. QUORACY catches a
// cluster that exists but cannot write to /etc/pve, which is indistinguishable
// from a healthy one until the first configuration change fails. The NODE COUNT
// catches the case this whole operation exists to avoid: a "create" that in fact
// joined something, leaving a cluster with members nobody here added.
func AssertClusterCreated(ctx context.Context, r Runner, wantName string) error {
	out, err := r.Run(ctx, `pvecm status 2>&1 || true`)
	if err != nil {
		return fmt.Errorf("bootstrap: read cluster status: %w (%s)", err, strings.TrimSpace(out))
	}

	var gotName, quorate string
	nodes := -1
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Name:"):
			gotName = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		case strings.HasPrefix(line, "Quorate:"):
			quorate = strings.TrimSpace(strings.TrimPrefix(line, "Quorate:"))
		case strings.HasPrefix(line, "Nodes:"):
			if _, err := fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "Nodes:")),
				"%d", &nodes); err != nil {
				nodes = -1
			}
		}
	}

	if gotName == "" {
		return fmt.Errorf("bootstrap: `pvecm create %s` reported success but the node is in no "+
			"cluster afterwards; its output was: %s", wantName, strings.TrimSpace(out))
	}
	if gotName != wantName {
		return fmt.Errorf("bootstrap: created cluster %q but the node reports %q; the control "+
			"plane would record membership of a cluster that does not exist under that name",
			wantName, gotName)
	}
	if !strings.EqualFold(quorate, "yes") {
		return fmt.Errorf("bootstrap: cluster %q was created but is not quorate (Quorate: %q); "+
			"/etc/pve is read-only in that state and every later configuration change fails",
			wantName, quorate)
	}
	if nodes != 1 {
		return fmt.Errorf("bootstrap: cluster %q reports %d nodes immediately after creation, "+
			"want exactly 1; something other than this host is already a member, so this was a "+
			"join wearing a create's clothes", wantName, nodes)
	}
	return nil
}
