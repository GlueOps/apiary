package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/tenant"
)

// connectingRunner refuses every command until Connect has been called, which is
// exactly how the ssh Runner behaves: it is constructed unconnected and opens its
// session in an optional Connect the Runner interface does not declare.
//
// THE FAKE THAT WOULD HAVE CAUGHT IT. The bootstrap job called NewRunner and went
// straight to a command, so the first thing it did on a real host was fail with
// "not connected ()" — a message naming neither the host nor the operation,
// arriving as a retryable job that failed identically until it exhausted its
// attempts. The existing fakes all answered commands without being connected, so
// nothing noticed.
type connectingRunner struct {
	connected bool
	closed    bool
	connects  int
	cmds      []string
	cluster   string // what `pvecm status` reports; empty means standalone
	created   bool
}

func (r *connectingRunner) Connect(context.Context) error {
	r.connects++
	r.connected = true
	return nil
}

func (r *connectingRunner) Close() error {
	r.closed = true
	return nil
}

func (r *connectingRunner) Upload(context.Context, string, uint32, []byte) error { return nil }

func (r *connectingRunner) Reconnect(context.Context) error {
	r.connected = true
	return nil
}

func (r *connectingRunner) Run(_ context.Context, cmd string) (string, error) {
	if !r.connected {
		return "", errors.New("not connected ()")
	}
	r.cmds = append(r.cmds, cmd)
	switch {
	case strings.Contains(cmd, "pvecm create"):
		r.created = true
		r.cluster = "lab-a"
		return "Corosync Cluster Engine Authentication key generator.\n", nil
	case strings.Contains(cmd, "APIARY_CLUSTER="):
		return "APIARY_CLUSTER=" + r.cluster + "\n", nil
	case strings.Contains(cmd, "pvecm status"):
		if r.cluster == "" {
			return "Error: Corosync config '/etc/pve/corosync.conf' does not exist\n", nil
		}
		return "Name: " + r.cluster + "\nNodes: 1\nQuorate: Yes\n", nil
	case strings.Contains(cmd, "APIARY_NODE="):
		return "APIARY_NODE=pve\n", nil
	}
	return "", nil
}

func bootstrapDeps(r Runner, snap BootstrapSnapshot, joined *struct {
	hive uuid.UUID
	node string
}) BootstrapDeps {
	return BootstrapDeps{
		FleetKey: func(context.Context, uuid.UUID) (string, []byte, error) {
			return "ssh-ed25519 AAAA", []byte("private"), nil
		},
		Snapshot: func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (BootstrapSnapshot, error) {
			return snap, nil
		},
		NewRunner: func(string, string, []byte) (Runner, error) { return r, nil },
		MarkJoined: func(_ context.Context, _, _, hiveID uuid.UUID, nodeName string) error {
			joined.hive = hiveID
			joined.node = nodeName
			return nil
		},
		EnqueueCapacitySyncAfter: func(context.Context, uuid.UUID, uuid.UUID, time.Duration) error {
			return nil
		},
		EnqueueCapacitySync: func(context.Context, uuid.UUID, uuid.UUID) error { return nil },
	}
}

func standaloneSnapshot() BootstrapSnapshot {
	return BootstrapSnapshot{
		TargetHive:        tenant.Hive{ID: uuid.New(), Name: "lab-a"},
		CurrentHiveID:     nil,
		HostAddress:       "192.168.240.11",
		HostNodeName:      "pve",
		HiveNodeCount:     0,
		ProvisioningState: "provisioned",
		AgentOnline:       true,
	}
}

func TestBootstrapConnectsBeforeItRunsAnything(t *testing.T) {
	r := &connectingRunner{}
	snap := standaloneSnapshot()
	var joined struct {
		hive uuid.UUID
		node string
	}

	err := BootstrapHive(context.Background(), bootstrapDeps(r, snap, &joined), BootstrapInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   snap.TargetHive.ID,
		Links:          []tenant.CorosyncLink{{Number: 0, Address: "192.168.240.11"}},
	})
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if r.connects != 1 {
		t.Fatalf("Connect called %d times, want exactly 1 — the ssh runner answers nothing "+
			"until it is connected", r.connects)
	}
	if !r.created {
		t.Fatal("no cluster was created")
	}
	if !r.closed {
		t.Error("the session was not closed; a job that leaks one leaks a connection per retry")
	}
	// Membership is what makes the cluster real to the control plane. Without it
	// the host stays fleetless on screen and the button that just worked is still
	// on offer.
	if joined.hive != snap.TargetHive.ID {
		t.Fatalf("membership recorded against %v, want the target cluster %v", joined.hive, snap.TargetHive.ID)
	}
	if joined.node != "pve" {
		t.Fatalf("node name recorded as %q, want the one the host reports", joined.node)
	}
}

// A host that is already clustered must be refused against the HOST's answer,
// not the database's — and reaching that answer requires being connected.
func TestBootstrapRefusesAHostThatIsAlreadyClustered(t *testing.T) {
	r := &connectingRunner{cluster: "someone-elses"}
	snap := standaloneSnapshot()
	var joined struct {
		hive uuid.UUID
		node string
	}

	err := BootstrapHive(context.Background(), bootstrapDeps(r, snap, &joined), BootstrapInput{
		OrganizationID: uuid.New(),
		HypervisorID:   uuid.New(),
		TargetHiveID:   snap.TargetHive.ID,
		Links:          []tenant.CorosyncLink{{Number: 0, Address: "192.168.240.11"}},
	})
	if err == nil {
		t.Fatal("a host already in a cluster was used to create another one")
	}
	if !strings.Contains(err.Error(), "someone-elses") {
		t.Fatalf("the refusal does not name the cluster the host is actually in: %v", err)
	}
	if r.created {
		t.Fatal("pvecm create ran on a host that is already a member")
	}
}
