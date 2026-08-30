package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

var ErrBootstrapRefused = errors.New("bootstrap: refused by a precondition")

// BootstrapSnapshot is everything the gate and the steps need, read in one
// transaction so the decision cannot be made against a half-changed picture.
type BootstrapSnapshot struct {
	TargetHive tenant.Hive

	// AgentOnline is whether this host's agent is live. The agent creates the
	// cluster, so this is the precondition — not the control plane's own SSH
	// provisioning state, which no longer participates.
	AgentOnline bool

	// CurrentHiveID is what the host reports today. Nil is the only acceptable
	// value; anything else is a refusal.
	CurrentHiveID *uuid.UUID

	// HostAddress and HostNodeName identify the machine the cluster is created
	// on. The node name is what the capacity sync will match the row against
	// afterwards, exactly as it is on the arrival path.
	HostAddress  string
	HostNodeName string

	// HiveNodeCount is how many nodes the hive already has. Anything above zero
	// makes this an arrival rather than a bootstrap.
	HiveNodeCount int

	// ProvisioningState of the host, so a half-onboarded machine is refused
	// before an SSH session is opened against it.
	ProvisioningState string
}

type BootstrapDeps struct {
	FleetKey  func(ctx context.Context, orgID uuid.UUID) (pub string, priv []byte, err error)
	Snapshot  func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID) (BootstrapSnapshot, error)
	NewRunner func(addr, user string, priv []byte) (Runner, error)

	// MarkJoined records membership. The SAME function the arrival path uses, so
	// a node that became a cluster's first member and one that joined an existing
	// cluster end up in identical states — including unschedulable, which is the
	// point: nothing has observed this cluster's capacity yet.
	MarkJoined func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID, nodeName string) error

	EnqueueCapacitySyncAfter func(ctx context.Context, orgID, hiveID uuid.UUID, after time.Duration) error
	EnqueueCapacitySync      func(ctx context.Context, orgID, hiveID uuid.UUID) error

	// RecordOutcome stores why this bootstrap failed, or clears it on success.
	// The same dependency the arrival takes, for the same reason: this package
	// keeps needing no tenant database handle of its own.
	RecordOutcome func(ctx context.Context, orgID, hypervisorID uuid.UUID, operation string, err error) error
}

type BootstrapInput struct {
	OrganizationID uuid.UUID
	HypervisorID   uuid.UUID
	TargetHiveID   uuid.UUID
	Links          []tenant.CorosyncLink
}

// BootstrapHive creates a Proxmox cluster on a standalone host and records it as
// the hive's first node.
//
// THE MISSING HALF OF THE FLEET CYCLE. Arrival joins a host to a cluster that
// already exists and refuses with target_hive_empty when it does not; departure
// removes one. Nothing created the first cluster, so a new hive could never
// acquire its first node and every arrival into it refused forever — the
// chicken-and-egg an operator meets the moment they add a hive.
//
// Deliberately NOT folded into Arrive. They look similar and are not: `pvecm
// add` replaces the joining node's cluster configuration and refuses a node
// carrying guests, while `pvecm create` replaces nothing and is safe on a host
// already running workloads. Sharing a gate would mean either refusing a
// bootstrap that is fine or permitting an arrival that is not.
func BootstrapHive(ctx context.Context, d BootstrapDeps, in BootstrapInput) error {
	if d.Snapshot == nil || d.NewRunner == nil || d.FleetKey == nil {
		return fmt.Errorf("bootstrap: dependencies are not wired")
	}

	snap, err := d.Snapshot(ctx, in.OrganizationID, in.HypervisorID, in.TargetHiveID)
	if err != nil {
		return err
	}

	// The gate first, and before any SSH session is opened. A refusal must cost
	// nothing on the host.
	refusals := tenant.CheckBootstrap(tenant.BootstrapRequest{
		TargetHive:        snap.TargetHive,
		CurrentHiveID:     snap.CurrentHiveID,
		HiveNodeCount:     snap.HiveNodeCount,
		ProvisioningState: snap.ProvisioningState,
		// The job re-checks because state moves between the handler and here. It
		// gates on the same thing: the agent creates the cluster, so a live agent
		// is the precondition and the control plane's own SSH state is not.
		AgentOnline: snap.AgentOnline,
	})
	if len(refusals) > 0 {
		return fmt.Errorf("%w: %w", ErrBootstrapRefused, tenant.BootstrapRefusalsError(refusals))
	}

	if snap.HostAddress == "" {
		return fmt.Errorf("bootstrap: the host has no address recorded, so there is nowhere to " +
			"open a session; a capacity sync or an enrolment records one")
	}

	_, priv, err := d.FleetKey(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("bootstrap: fleet key: %w", err)
	}
	// Through the SAME connect helper the departure and arrival use, rather than
	// NewRunner alone.
	//
	// The ssh Runner is constructed unconnected and opens its session in an
	// optional Connect the interface does not declare, so calling NewRunner and
	// going straight to a command produced "not connected ()" from the first
	// step — a message that names neither the host nor the operation, arriving
	// as a retryable job that failed identically twenty-five times.
	runner, closeRunner, err := connect(ctx, BootstrapRunnerDeps(d), snap.HostAddress, "root", priv)
	if err != nil {
		return fmt.Errorf("bootstrap: connect to %s: %w", snap.HostAddress, err)
	}
	defer closeRunner()

	// Asked of the HOST, not of the database. The row said fleetless; this is
	// what corosync says, and only one of them is what pvecm will act on.
	if err := AssertStandalone(ctx, runner); err != nil {
		return err
	}

	if err := CreateCluster(ctx, runner, snap.TargetHive.Name, in.Links); err != nil {
		return err
	}

	// The node name comes from the host, for the same reason it does on arrival:
	// a clustered row with no name, or the wrong one, is a row the next capacity
	// sync cannot match and duplicates on the spot.
	nodeName := snap.HostNodeName
	if nodeName == "" {
		read, err := ReadNodeName(ctx, runner)
		if err != nil {
			return fmt.Errorf("bootstrap: the cluster was created but its node name could not "+
				"be read, so membership cannot be recorded without duplicating the row: %w", err)
		}
		nodeName = read
	}

	if d.MarkJoined != nil {
		if err := d.MarkJoined(ctx, in.OrganizationID, in.HypervisorID, snap.TargetHive.ID,
			nodeName); err != nil {
			return fmt.Errorf("bootstrap: cluster %q was created on this host but recording its "+
				"membership failed, so the control plane still shows it as fleetless: %w",
				snap.TargetHive.Name, err)
		}
	}

	// Re-observed shortly rather than now, for the reason spelled out on the
	// arrival path: Proxmox has not populated /cluster/resources stats for a node
	// at the moment its cluster comes up, so an immediate sync records zeros and
	// stamps them as freshly observed.
	//
	// Logged rather than returned. The cluster exists and membership is recorded;
	// failing the job here would retry a `pvecm create` that has already
	// succeeded, and the second attempt refuses because the node is now clustered.
	if hiveIsSyncable(snap.TargetHive) {
		enqueue := func(ctx context.Context) error {
			if d.EnqueueCapacitySyncAfter != nil {
				return d.EnqueueCapacitySyncAfter(ctx, in.OrganizationID, snap.TargetHive.ID,
					ArrivalResyncDelay)
			}
			if d.EnqueueCapacitySync != nil {
				return d.EnqueueCapacitySync(ctx, in.OrganizationID, snap.TargetHive.ID)
			}
			return nil
		}
		if err := enqueue(ctx); err != nil {
			slog.WarnContext(ctx, "bootstrap: the cluster was created and membership recorded "+
				"but a capacity sync could not be triggered; the hive reads as having no "+
				"observed capacity until the scheduled sync runs",
				"organization_id", in.OrganizationID, "hive_id", snap.TargetHive.ID, "error", err)
		}
	}
	return nil
}

// BootstrapRunnerDeps adapts BootstrapDeps to the shape `connect` takes, so this
// worker shares the optional Connect/Close dance with the other two rather than
// reimplementing it — which is how it came to be skipped here in the first place.
func BootstrapRunnerDeps(d BootstrapDeps) DepartDeps {
	return DepartDeps{NewRunner: d.NewRunner}
}
