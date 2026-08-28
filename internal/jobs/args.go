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

	// HypervisorID names the machine record whose provisioning state this job
	// concludes, and is what the ticker claimed before enqueuing.
	//
	// Optional, because it is the ROW being driven rather than the HOST being
	// onboarded: without one the job still onboards the host and simply records
	// no outcome. Identifying the row by address instead would be wrong in the
	// case that matters — a machine whose address was corrected mid-flight would
	// have its outcome written to whichever row currently holds the old value.
	HypervisorID string `json:"hypervisor_id,omitempty"`
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

	// CredentialDisposition carries the operator's answer about the cluster
	// secrets this node has seen. Part of the job rather than read from config,
	// because it is a decision made about THIS departure — see
	// tenant.CheckDeparture.
	CredentialDisposition string `json:"credential_disposition"`
}

func (DepartHypervisorArgs) Kind() string { return "fleet.depart_hypervisor" }

// InsertOpts makes a departure unique per machine while one is outstanding.
//
// Without it every click queues another job. An operator watching a cordoned,
// drained node that will not leave clicks Depart again — reasonably — and gets a
// second concurrent `pvecm delnode` against the same node. On a small cluster
// delnode is not something to run twice at once: it strands the survivor
// read-only and leaves a lock that blocks every retry, which is a worse
// situation than the one the operator was trying to fix.
//
// All four of River's REQUIRED states plus retryable. A ByState missing any of
// available, pending, running or scheduled is rejected at insert — silently, for
// a periodic job — and retryable is included because a departure that failed and
// is waiting to retry is still outstanding: queueing a second one alongside it
// is exactly the collision this prevents.
func (DepartHypervisorArgs) InsertOpts() river.InsertOpts {
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

// InsertOpts makes an arrival unique per machine while one is outstanding.
//
// The same reasoning as the departure's, and a sharper edge. An operator
// watching a host that has not joined clicks Join again, and a second concurrent
// `pvecm add` against the same node is worse than a second delnode: the joining
// node rewrites its own corosync configuration and its keyring during a join,
// and two runs interleaving there produce a node that is a member of nothing and
// cannot be reached by the key that used to work.
//
// Retryable is included, because an arrival that failed and is waiting to retry
// is still outstanding — a second one queued alongside it is the collision.
func (ArriveHypervisorArgs) InsertOpts() river.InsertOpts {
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

// BootstrapHiveArgs creates a Proxmox cluster on a standalone host and records
// it as the hive's first node.
//
// The other half of ArriveHypervisorArgs. Arrival refuses with
// target_hive_empty when the hive has no nodes yet, which is every hive on the
// day it is created — so without this a new hive could never acquire its first
// node and every arrival into it refused forever.
type BootstrapHiveArgs struct {
	OrganizationID string `json:"organization_id"`
	HypervisorID   string `json:"hypervisor_id"`
	TargetHiveID   string `json:"target_hive_id"`

	// Links are the first node's corosync rings, carried for the same reason as
	// the arrival's: network topology is an operator's decision and nothing in
	// the database knows it. They matter MORE here — `pvecm create` writes the
	// ring configuration every later node inherits, so a single-ring cluster
	// created today is a single point of failure for quorum forever.
	Links []tenant.CorosyncLink `json:"links"`
}

func (BootstrapHiveArgs) Kind() string { return "fleet.bootstrap_hive" }

// Unique per machine while one is outstanding, for the arrival's reasons and
// one of its own: `pvecm create` run twice against the same host leaves a node
// holding a cluster configuration that no longer matches the keyring it
// generated the first time.
func (BootstrapHiveArgs) InsertOpts() river.InsertOpts {
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

// ClaimPendingOnboardsArgs is the periodic tick that drives onboarding forward.
//
// Nothing else moves a machine off `pending`. The state column, the lease, the
// attempt counter and the redrive endpoint all landed before this did, which
// meant a row pushed to pending sat there permanently while the API told the
// operator it "will be claimed on the next tick".
type ClaimPendingOnboardsArgs struct{}

func (ClaimPendingOnboardsArgs) Kind() string { return "fleet.claim_pending_onboards" }

func (ClaimPendingOnboardsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		// One tick at a time, fleet-wide. Two overlapping ticks would both scan for
		// claimable rows; the claim is atomic so they could not take the same row,
		// but the second would find nothing and exist only to hold a connection.
		//
		// All four of River's REQUIRED states, not just the two this rule cares
		// about. UniqueOpts.validate rejects a ByState missing any of available,
		// pending, running or scheduled — and the rejection surfaces at insert
		// time, which for a periodic job means the tick simply never appears.
		// Listing only available and running produced exactly that: a registered
		// periodic job, a registered worker, correct config, and no job ever
		// inserted.
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRunning,
				rivertype.JobStateScheduled,
			},
		},
	}
}
