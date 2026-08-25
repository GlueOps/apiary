package jobs

import (
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type ProvisionTenantArgs struct {
	OrganizationID string `json:"organization_id"`
}

func (ProvisionTenantArgs) Kind() string { return "tenant.provision" }

// OnboardHypervisorArgs brings one host into an organization's fleet.
//
// The machine id identifies the resulting machine record AND is how the job knows
// which agent's heartbeat proves it worked — so a retry recognises the host it
// already installed rather than treating it as new.
//
// It is the RAW /etc/machine-id here, not the digest. Everything that stores or
// looks one up puts it through tenant.MachineIDDigest first; carrying the raw
// value on the job is what lets the agent be handed the same value it will
// present at enrolment.
type OnboardHypervisorArgs struct {
	OrganizationID string `json:"organization_id"`
	Address        string `json:"address"`
	User           string `json:"user,omitempty"`
	MachineID      string `json:"machine_id"`
}

func (OnboardHypervisorArgs) Kind() string { return "fleet.onboard_hypervisor" }

// DepartHypervisorArgs removes one host from its hive, leaving it standalone and
// belonging to no fleet.
//
// Identified by hypervisor id rather than by address: the address is how the job
// REACHES the host, and it can change, but which node of which hive is being
// removed must not be open to interpretation by the time a worker is dismantling
// cluster membership. The addresses come from the snapshot the job reads.
//
// Departure does not join anything. A migration is a departure and then an
// arrival, each gated separately, with a legitimate fleetless state in between.
type DepartHypervisorArgs struct {
	OrganizationID string `json:"organization_id"`
	HypervisorID   string `json:"hypervisor_id"`
}

func (DepartHypervisorArgs) Kind() string { return "fleet.depart_hypervisor" }

// ArriveHypervisorArgs adds one standalone host to a hive.
//
// Identified by HYPERVISOR id, the same as departure. It used to be identified by
// bee id, and that asymmetry was forced by a fact that is no longer one: a
// Hypervisor row was whatever a capacity sync had observed in a cluster, so a
// host that had not joined one had only its bee to be named by. The Hypervisor is
// the durable machine record now and exists before any cluster does.
//
// The change also fixes the merge. The join writes the cluster's node name onto
// this machine when it succeeds, and the worker can only do that if it was told
// which machine record to write to.
type ArriveHypervisorArgs struct {
	OrganizationID string `json:"organization_id"`
	HypervisorID   string `json:"hypervisor_id"`
	TargetHiveID   string `json:"target_hive_id"`

	// Links are the joining node's corosync rings. Carried on the job rather than
	// looked up, because they are an operator's decision about network topology
	// and nothing in the database knows them.
	//
	// Plural: a single ring makes the cluster network a single point of failure
	// for quorum. Proxmox supports link0-link7.
	Links []tenant.CorosyncLink `json:"links"`
}

func (ArriveHypervisorArgs) Kind() string { return "fleet.arrive_hypervisor" }

// SyncHiveCapacityArgs refreshes one hive's capacity observations from its
// provider API.
//
// Unique per hive while queued or running: a sync that takes longer than the
// scheduling interval must not stack up behind itself, which on an unreachable
// cluster is exactly what would happen — each tick adding another job that will
// also time out.
type SyncHiveCapacityArgs struct {
	OrganizationID string `json:"organization_id"`
	HiveID         string `json:"hive_id"`
}

func (SyncHiveCapacityArgs) Kind() string { return "fleet.sync_hive_capacity" }

func (SyncHiveCapacityArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRunning,
				rivertype.JobStateRetryable,
				rivertype.JobStateScheduled,
			},
		},
	}
}

// SyncAllHivesArgs is the periodic tick that fans out one sync per hive.
type SyncAllHivesArgs struct{}

func (SyncAllHivesArgs) Kind() string { return "fleet.sync_all_hives" }

func (SyncAllHivesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRunning,
				rivertype.JobStateRetryable,
				rivertype.JobStateScheduled,
			},
		},
	}
}
