package jobs

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
)

// bootstrapRunner answers the two commands a bootstrap issues: reading the
// cluster name, and `pvecm create` followed by a status read.
//
// It models the ORDER those answers change in. A real host reports no cluster
// before the create and the new one after, so a runner that answered the same
// thing throughout would let a broken CreateCluster pass — which is exactly the
// failure the verification after the command exists to catch.
type bootstrapRunner struct {
	cluster   string // what `pvecm status` reports; "" means standalone
	quorate   bool
	nodes     int
	created   bool
	createErr error
	cmds      []string
}

func (r *bootstrapRunner) Upload(context.Context, string, uint32, []byte) error { return nil }
func (r *bootstrapRunner) Reconnect(context.Context) error                      { return nil }

func (r *bootstrapRunner) Run(_ context.Context, cmd string) (string, error) {
	r.cmds = append(r.cmds, cmd)
	switch {
	case strings.Contains(cmd, "pvecm create"):
		if r.createErr != nil {
			return "", r.createErr
		}
		r.created = true
		return "Corosync Cluster Engine Authentication key generator.\n", nil
	case strings.Contains(cmd, "APIARY_CLUSTER="):
		if r.created && r.cluster != "" {
			return "APIARY_CLUSTER=" + r.cluster + "\n", nil
		}
		if !r.created {
			return "APIARY_CLUSTER=\n", nil
		}
		return "APIARY_CLUSTER=" + r.cluster + "\n", nil
	case strings.Contains(cmd, "pvecm status"):
		if !r.created {
			return "Cannot initialize CMAP service\n", nil
		}
		q := "No"
		if r.quorate {
			q = "Yes"
		}
		return fmt.Sprintf("Cluster information\n-------------------\nName:             %s\n"+
			"Config Version:   1\nQuorum information\n------------------\nQuorate:          %s\n"+
			"Votequorum information\n----------------------\nNodes:            %d\n",
			r.cluster, q, r.nodes), nil
	}
	return "", nil
}

// At least one corosync ring is required, exactly as it is on the arrival path:
// the address is not assumed to be the host's management address.
func oneLink() []tenant.CorosyncLink {
	return []tenant.CorosyncLink{{Number: 0, Address: "10.10.10.9"}}
}

// A bootstrap with no ring named must be refused before anything runs, for the
// reason the arrival path gives: cluster traffic routinely belongs on a
// different network, and a first node that guesses hands the guess to every node
// that joins it later.
func TestCreateClusterRequiresACorosyncLink(t *testing.T) {
	r := &bootstrapRunner{cluster: "dev", quorate: true, nodes: 1}
	if err := CreateCluster(context.Background(), r, "dev", nil); err == nil {
		t.Fatal("a bootstrap naming no corosync ring must be refused")
	}
	if len(r.cmds) != 0 {
		t.Fatalf("nothing may run on the host when the links are unusable, got %v", r.cmds)
	}
}

func TestAssertStandaloneRefusesAClusteredHost(t *testing.T) {
	// The row said fleetless; corosync says otherwise, and only one of them is
	// what pvecm will act on. Creating a cluster here either fails or leaves two
	// clusters each believing they own the node.
	r := &bootstrapRunner{cluster: "lab-dc", created: true}
	err := AssertStandalone(context.Background(), r)
	if err == nil {
		t.Fatal("a host already in a cluster must be refused before pvecm create runs")
	}
	if !strings.Contains(err.Error(), "lab-dc") {
		t.Errorf("the error must name the cluster the host is actually in, got %q", err)
	}
}

func TestAssertStandaloneAcceptsAStandaloneHost(t *testing.T) {
	if err := AssertStandalone(context.Background(), &bootstrapRunner{}); err != nil {
		t.Fatalf("a standalone host must be accepted: %v", err)
	}
}

func TestCreateClusterVerifiesRatherThanTrustingTheCommand(t *testing.T) {
	// pvecm reports success before the local stack has necessarily converged, so
	// its exit code is not the evidence — the same reason the join path asserts
	// afterwards.
	r := &bootstrapRunner{cluster: "dev", quorate: true, nodes: 1}
	if err := CreateCluster(context.Background(), r, "dev", oneLink()); err != nil {
		t.Fatalf("a clean create must pass: %v", err)
	}
	var ran bool
	for _, c := range r.cmds {
		if strings.Contains(c, "pvecm create 'dev'") {
			ran = true
		}
	}
	if !ran {
		t.Fatalf("the create was not issued with the quoted cluster name, got %v", r.cmds)
	}
}

// The failure this whole operation exists to avoid: a "create" that in fact
// joined something. A cluster reporting more than one node immediately after
// creation has members nobody here added.
func TestCreateClusterRefusesAClusterThatAlreadyHasOtherNodes(t *testing.T) {
	r := &bootstrapRunner{cluster: "dev", quorate: true, nodes: 3}
	err := CreateCluster(context.Background(), r, "dev", oneLink())
	if err == nil {
		t.Fatal("a freshly created cluster reporting 3 nodes must fail; something other than " +
			"this host is already a member")
	}
	if !strings.Contains(err.Error(), "join wearing a create") {
		t.Errorf("the error must say what went wrong, got %q", err)
	}
}

// A cluster that exists but cannot write to /etc/pve is indistinguishable from a
// healthy one until the first configuration change fails.
func TestCreateClusterRefusesANonQuorateResult(t *testing.T) {
	r := &bootstrapRunner{cluster: "dev", quorate: false, nodes: 1}
	if err := CreateCluster(context.Background(), r, "dev", oneLink()); err == nil {
		t.Fatal("a non-quorate cluster must fail the bootstrap")
	}
}

func TestCreateClusterRefusesAClusterCreatedUnderTheWrongName(t *testing.T) {
	r := &bootstrapRunner{cluster: "something-else", quorate: true, nodes: 1}
	err := CreateCluster(context.Background(), r, "dev", oneLink())
	if err == nil {
		t.Fatal("a cluster created under a different name must fail; the control plane would " +
			"record membership of a cluster that does not exist under the name it recorded")
	}
}

// The name is validated BEFORE the session does anything, because the
// alternative is discovering Proxmox's rule on real metal.
func TestCreateClusterRefusesAnIllegalNameWithoutRunningAnything(t *testing.T) {
	r := &bootstrapRunner{}
	if err := CreateCluster(context.Background(), r, "not a valid name", oneLink()); err == nil {
		t.Fatal("an illegal cluster name must be refused")
	}
	if len(r.cmds) != 0 {
		t.Fatalf("nothing may run on the host for a name that cannot be accepted, got %v", r.cmds)
	}
}

// Links are optional. A single-node cluster on a simple network wants corosync's
// own choice; naming rings matters once a second node joins over another network.
func TestCreateClusterPassesCorosyncLinksThrough(t *testing.T) {
	r := &bootstrapRunner{cluster: "dev", quorate: true, nodes: 1}
	links := []tenant.CorosyncLink{{Number: 0, Address: "10.10.10.2"}}
	if err := CreateCluster(context.Background(), r, "dev", links); err != nil {
		t.Fatalf("create with links: %v", err)
	}
	var found bool
	for _, c := range r.cmds {
		if strings.Contains(c, "pvecm create") && strings.Contains(c, "10.10.10.2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the link was not passed to pvecm create, got %v", r.cmds)
	}
}
