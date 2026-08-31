package tenant

import (
	"gorm.io/datatypes"
	"math"
	"time"

	"github.com/google/uuid"
)

// Onboarding lifecycle states for Hypervisor.ProvisioningState.
//
// DELIBERATELY NOT NAMED Provisioning*, even though the column is
// `provisioning_state`. fleet_provisioning.go already exports
// ProvisioningAvailable/Declared/ServiceDown/Blocked/HostDown/Unknown for an
// entirely different axis — whether the control plane can reach a hive's Proxmox
// API right now. Two symbol sets one word apart get conflated within a month,
// and conflating "we cannot reach the API" with "the agent is not installed" is
// exactly the one-status-for-three-questions failure this model exists to undo.
// If the older set is ever renamed, APIReachability* is the name it wants.
const (
	// OnboardPending is recorded intent: somebody said "this machine is ours".
	// The ticker claims from here; nothing else creates work.
	OnboardPending = "pending"
	// OnboardProvisioning means a worker holds a lease on this row. See
	// ProvisioningLeaseExpiresAt for why the lease, not the state, is the guard.
	OnboardProvisioning = "provisioning"
	// OnboardProvisioned is terminal but redrivable — redriving it is "revive",
	// and it is safe because Onboard skips enrolment for a host already
	// heartbeating, so a re-run does not invalidate a working credential.
	OnboardProvisioned = "provisioned"
	OnboardFailed      = "failed"
)

// A Hypervisor is the durable record of one MACHINE.
//
// It is not the same row as a Bee and not a second identity for one. A Bee is
// the AGENT running on this machine — its credential, its heartbeat, its applied
// firewall. This row is the machine itself: how to reach it, which cluster it is
// in, which swarm, what the provider last observed on it, and how far its
// onboarding got. Bee therefore carries no hive_id: two columns answering "which
// cluster is this in" with nothing keeping them in agreement is the bug this
// shape removes, and they already diverged during a migration.
//
// THE SYNC NO LONGER OWNS THIS ROW. It used to: a node that stopped being
// reported had its row DELETED, which made membership self-healing at the cost
// of destroying the machine record and its onboarding history every time a host
// left a cluster. The sync now CLEARS the cluster-derived fields instead
// (hive_id, swarm_id, name, the observations, last_synced_at), which achieves
// the same visible result — the machine reads as ours-but-unclustered — without
// losing everything else known about it.
//
// The consequence to hold on to: HiveID and Name are NULLABLE, and NULL is a
// normal state, not a broken one. A machine being onboarded is in no cluster and
// has no Proxmox node name, because a node name does not exist until it joins
// one.
//
// TWO WAYS OF MATCHING ONE ROW, and they must converge. Enrolment identifies a
// machine by MachineID; the capacity sync identifies one by (HiveID, Name),
// because Proxmox's /cluster/resources reports no machine id. The rule is: match
// on MachineID when the incoming fact carries one, else on (HiveID, Name), and
// when a machine onboarded by address later appears in a cluster under a node
// name the two records MERGE rather than both existing.
type Hypervisor struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// HiveID is nullable: a machine we are onboarding, or one that has left a
	// cluster, belongs to no hive. Same reasoning that already made Bee.HiveID
	// nullable — a migration passes through "no cluster" and that middle state is
	// not a blip to paper over.
	//
	// ON DELETE SET NULL, not CASCADE. Under the old shape the row was the
	// cluster's node and died with it; now it is the machine, which outlives any
	// cluster it happens to be in. Cascading would delete the machine records and
	// their onboarding history because somebody removed a hive.
	//
	// Note the interaction with the swarm foreign key below, which is RESTRICT:
	// deleting a hive that still has hypervisors assigned to its swarms aborts on
	// that RESTRICT before this SET NULL is reached. That is the pre-existing
	// behaviour and it is also what keeps SET NULL from ever leaving a row with
	// swarm_id set and hive_id NULL, which the CHECK below forbids.
	//
	// The plain index is not redundant with ux_hypervisor_hive_name any more.
	// That index became partial, and its predicate names `name IS NOT NULL`,
	// which the planner cannot infer from `hive_id = ?` — so it stopped being
	// usable for the hive_id lookups the sync, the views and every membership
	// count do.
	HiveID *uuid.UUID `gorm:"type:uuid;uniqueIndex:ux_hypervisor_hive_name,priority:1,where:hive_id IS NOT NULL AND name IS NOT NULL;index;check:ck_hypervisors_swarm_needs_hive,swarm_id IS NULL OR hive_id IS NOT NULL" json:"hive_id,omitempty"`
	Hive   *Hive      `gorm:"foreignKey:HiveID;constraint:OnDelete:SET NULL" json:"-"`

	// Name is the provider's node name, not a hostname we chose, and it is
	// nullable because there is no such name until the machine joins a cluster.
	//
	// The unique index over (hive_id, name) is therefore PARTIAL. A plain unique
	// index over nullable columns constrains NOTHING for the rows that are null —
	// Postgres treats NULL as distinct from NULL, so every unclustered machine
	// would be unique against every other one and the index would silently stop
	// being the identity the sync matches on. Same trap already documented on
	// FirewallBinding and on Placement.VMID.
	Name *string `gorm:"size:255;uniqueIndex:ux_hypervisor_hive_name,priority:2" json:"name,omitempty"`

	// MachineID is the host's own stable identifier (/etc/machine-id), hashed. It
	// is what makes re-enrolment idempotent: a machine rebuilt and re-enrolled
	// under the same id repoints its existing record rather than accumulating a
	// second one, and an address change does not orphan it.
	//
	// NULLABLE, because the sync legitimately discovers nodes in a cluster that
	// nobody onboarded through this system and whose machine id we have never
	// read. Unique where present, and PARTIAL for the same NULL-distinctness
	// reason as the index above.
	//
	// Never serialised. It is a hashed inventory of somebody's hosts; there is no
	// caller that needs it on the wire, and Bee.MachineID was `json:"-"` before
	// it moved here for exactly that reason.
	MachineID *string `gorm:"type:varchar(64);uniqueIndex:ux_hypervisor_machine,where:machine_id IS NOT NULL" json:"-"`

	// Address is how we reach this machine — the thing SSH and the agent's
	// enrolment use. A fact about the machine, which is why it lives here and not
	// on the agent row.
	//
	// Distinct from Name on purpose: Name is what the cluster calls the node and
	// only exists inside a cluster, Address is what the network calls the box and
	// exists from the moment somebody racks it.
	Address string `gorm:"type:varchar(253);not null;default:''" json:"address"`

	// SSHHostKeys are this machine's own SSH host public keys, captured during
	// onboarding over the control plane's authenticated session to it.
	//
	// STORED because the node that needs them is not the node that has them. A
	// joining host must have its PEER's key in known_hosts before pvecm add runs
	// ssh-copy-id, or the whole join fails with "unable to copy ssh ID: exit code
	// 1" — a message naming neither the host, nor the key, nor verification.
	//
	// The control plane used to read them at join time, holding SSH sessions to
	// both machines at once. Under agent-polled convergence the joining host acts
	// on its own, so there is no such moment: the keys have to have been captured
	// earlier and travel to the joiner through the composition it already fetches.
	// That preserves the assurance rather than weakening it — the key reaches the
	// joiner over its authenticated control-plane session, not from whatever
	// answers on the peer's address, which is the defect ssh-keyscan and
	// StrictHostKeyChecking=no both have.
	//
	// Public keys only. There is nothing secret here; the value is in WHERE they
	// came from.
	SSHHostKeys datatypes.JSONType[[]string] `gorm:"type:jsonb" json:"ssh_host_keys,omitzero"`

	// DepartingAt marks a machine an operator has removed from its hive but
	// which the cluster has not yet dropped.
	//
	// IT KEEPS hive_id AND name, and that is the whole point. Clearing them the
	// moment an operator clicks Depart leaves nothing for the composition to
	// NAME, and a remaining member can only run `pvecm delnode <node>` if it is
	// told which node — so eviction could never be armed. Absence from the
	// member list is deliberately not enough: a node in corosync that the control
	// plane merely has no record of is as likely to be one nobody has onboarded
	// yet, and evicting on that inference orphans guests.
	//
	// It is also the truthful state. The machine IS still in the cluster until a
	// peer removes it; what has changed is the intention, not the fact. The
	// record catches up when a remaining member observes it gone, at which point
	// hive_id, name and swarm_id are cleared together as they always were.
	DepartingAt *time.Time `gorm:"type:timestamptz;index" json:"departing_at,omitempty"`

	// CredentialDisposition is what an operator chose for a departing machine's
	// secrets: accept-exposure or rotate.
	//
	// Recorded rather than acted on immediately, because the act belongs AFTER
	// the cluster has dropped the node — disarming a node that is still a member
	// destroys secrets it is still using. It is carried to the departed host on a
	// task once a remaining member confirms the removal.
	//
	// It is also the one part of a departure that no observation implies. Every
	// other step is derivable from desired against observed state; this is a
	// decision with a security consequence, and six months later this column is
	// the only record that somebody made it.
	CredentialDisposition string `gorm:"type:varchar(32);not null;default:''" json:"credential_disposition,omitempty"`

	// SwarmID is the named group this hypervisor belongs to, or nil for spare
	// capacity.
	//
	// NULLABLE because "in the cluster, in no group" is a real and useful state:
	// metal that is racked, joined and healthy but not yet promised to anything.
	// An operator adds it to a swarm when they want it carrying that swarm's
	// work, and that act is what makes the capacity accountable to someone.
	//
	// Constrained to the SAME HIVE as this hypervisor by a composite foreign key
	// on (hive_id, swarm_id), not by a plain reference to swarms(id). A plain FK
	// would happily let a hypervisor in dc1 join a swarm in dc2 — which reads as
	// a grouping but schedules as a lie, since the two machines cannot take each
	// other's work. The same denormalise-to-express-a-constraint trick Placement
	// uses for VMID uniqueness.
	//
	// THAT FOREIGN KEY STOPPED BEING ENFORCED for part of the table when hive_id
	// became nullable, and this is the single sharpest edge in this change.
	// Postgres composite foreign keys default to MATCH SIMPLE, which SKIPS THE
	// CHECK ENTIRELY when any referencing column is NULL. So an unclustered
	// machine — hive_id NULL — could hold any swarm_id at all, including one
	// belonging to a cluster it has left, and the database would not object.
	// The CHECK on HiveID above (`swarm_id IS NULL OR hive_id IS NOT NULL`) is
	// what closes that hole: it removes the one nullability combination the
	// foreign key cannot police. MATCH FULL would also have worked, but GORM
	// cannot emit it and it would additionally forbid the legitimate
	// hive-without-swarm row.
	//
	// RESTRICT, not SET NULL, and not by preference. Postgres nulls EVERY column
	// of a composite foreign key on SET NULL — verified: it issues
	// `SET hive_id = NULL, swarm_id = NULL`. That used to fail the NOT NULL on
	// hive_id and report the swarm deletion as a confusing error about the wrong
	// column; now that hive_id is nullable it would SUCCEED, silently evicting a
	// machine from its cluster because somebody deleted a swarm. Worse failure,
	// same answer.
	//
	// RESTRICT is also the honest semantics. Deleting `fleet-prod` would otherwise
	// turn fifteen hypervisors into spare capacity from one statement, with
	// nothing recording that somebody meant to. Unassign them first, which is the
	// same rule Pool applies to a Slot in use.
	SwarmID *uuid.UUID `gorm:"type:uuid;index" json:"swarm_id,omitempty"`
	Swarm   *Swarm     `gorm:"foreignKey:HiveID,SwarmID;references:HiveID,ID;constraint:OnDelete:RESTRICT" json:"-"`

	// BeeID is the enrolled agent on this node, when there is one.
	//
	// Nullable and unique. Nullable because the cluster reports nodes we have not
	// enrolled; unique because two nodes claiming one agent would mean fleet
	// operations acting on the wrong machine. Partial-unique, since Postgres
	// treats NULLs as distinct and every un-enrolled node would otherwise collide
	// on a plain unique index.
	BeeID *uuid.UUID `gorm:"type:uuid;uniqueIndex:ux_hypervisor_bee,where:bee_id IS NOT NULL" json:"bee_id,omitempty"`
	Bee   *Bee       `gorm:"foreignKey:BeeID;constraint:OnDelete:SET NULL" json:"-"`

	// --- OBSERVED: what the provider last reported. ---------------------------
	//
	// Prefixed, because they describe the world rather than anything this system
	// decided, and the sync is their only writer.
	//
	// ONE OF THEM IS NOW A PLACEMENT INPUT — see ObservedForeign*Used below — so
	// the blanket "never used as the basis for a placement decision" that stood
	// here is no longer true, and the reasoning behind it deserves restating
	// rather than deleting. The hazard it named was a RACE: two concurrent
	// placements reading the same free capacity off a stale row and both fitting.
	// That argument is untouched, because a concurrent placement moves INTENDED
	// usage, never the foreign term, and intended is still computed under the
	// same lock as the insert.
	//
	// The foreign term carries a different risk — plain staleness, a sync that
	// has not run since somebody hand-created a VM — and a lock cannot close it.
	// What closes it is syncing on membership change instead of only on the timer.
	//
	// Drift detection remains the other reason both halves are kept: comparing
	// observed usage against the placements we intended is how a VM created
	// outside this system, or one that failed to be cleaned up, becomes visible.
	ObservedCPUTotal    int `gorm:"not null;default:0" json:"observed_cpu_total"`
	ObservedRAMGBTotal  int `gorm:"not null;default:0" json:"observed_ram_gb_total"`
	ObservedDiskGBTotal int `gorm:"not null;default:0" json:"observed_disk_gb_total"`

	// ObservedGuests is how many guests the provider reports on this node,
	// STOPPED ONES INCLUDED.
	//
	// A count, not a usage figure, and that is the whole reason it exists rather
	// than being inferred from ObservedCPUUsed. A stopped guest consumes no CPU
	// and no memory, so every usage-based reading of "is this node empty" says
	// yes while `pvecm add` refuses the node with "this host already contains
	// virtual guests". Observed on a real cluster: a node whose only guests were
	// two stopped containers was rejected by the join precheck.
	//
	// Counts guests belonging to this node. A node holding the cluster's replica
	// of ANOTHER node's guest configs is a separate hazard the departure sequence
	// clears explicitly — see PruneDepartedNodeDir.
	ObservedGuests int `gorm:"not null;default:0" json:"observed_guests"`

	ObservedCPUUsed    int `gorm:"not null;default:0" json:"observed_cpu_used"`
	ObservedRAMGBUsed  int `gorm:"not null;default:0" json:"observed_ram_gb_used"`
	ObservedDiskGBUsed int `gorm:"not null;default:0" json:"observed_disk_gb_used"`

	// ObservedForeign*Used is the part of the above belonging to guests THIS
	// SYSTEM DID NOT PLACE, and it is the term that makes placement honest.
	//
	// Without it the oracle computed `allocatable - intended`, so a guest nobody
	// placed through this system was counted ZERO times and a hypervisor carrying
	// twenty hand-made VMs reported its full ceiling as free. waggle has the
	// mirror-image hazard documented on its own exclude set — count a guest in
	// both the discovery totals and the ledger and you double-subtract it — and
	// the resolution is the same in both directions: each guest counted exactly
	// once, ours through the ledger, everyone else's through these columns.
	//
	// Stored SEPARATELY from the totals rather than replacing them, because the
	// totals have two readers that need every guest: Drift, which is only
	// meaningful against the whole, and the fleet view, which reports what the
	// node is actually carrying. ObservedGuests likewise stays a total — the
	// departure and arrival gates count OUR guests too, since `pvecm add` refuses
	// a node for a VM we placed just as readily as for one we did not.
	//
	// Written by the sync in the same statement as LastSyncedAt, for the reason
	// ObservedGuests already documents: a freshness stamp over columns left at
	// their zero default is a fail-OPEN, and here it would read as a node with no
	// foreign load at all.
	ObservedForeignCPUUsed    int `gorm:"not null;default:0" json:"observed_foreign_cpu_used"`
	ObservedForeignRAMGBUsed  int `gorm:"not null;default:0" json:"observed_foreign_ram_gb_used"`
	ObservedForeignDiskGBUsed int `gorm:"not null;default:0" json:"observed_foreign_disk_gb_used"`

	// LastSyncedAt is how old the observations are. A placement made against a
	// stale row is a placement made against a guess, so a caller that cares must
	// check this rather than assume the numbers are current.
	LastSyncedAt *time.Time `gorm:"type:timestamptz" json:"last_synced_at,omitempty"`

	// --- RESERVED: held back from scheduling, set by an operator. -------------
	//
	// This is the hypervisor's own overhead — the memory the host needs to run,
	// the cores you want left for it — not capacity anything has taken. Distinct
	// from Observed*Used, which is what guests are actually consuming, and from
	// intended usage, which is the sum of our own placements. Three different
	// questions; the previous shape had them as three similarly-named columns
	// with nothing saying which was which.
	CPUReserved    int `gorm:"not null;default:0" json:"cpu_reserved"`
	RAMGBReserved  int `gorm:"not null;default:0" json:"ram_gb_reserved"`
	DiskGBReserved int `gorm:"not null;default:0" json:"disk_gb_reserved"`

	// CPUOvercommitRatio overrides the hive's. Non-positive means "use the
	// hive's", which is why Allocatable takes the hive rather than reading this
	// alone.
	CPUOvercommitRatio float64 `gorm:"type:numeric(5,2);not null;default:0" json:"cpu_overcommit_ratio"`

	// Schedulable is the operator's switch. It withdraws a node from new
	// placements WITHOUT evicting what is already there — draining is a separate
	// act, and conflating them would mean the only way to stop new work was to
	// move existing work.
	//
	// Enforced by the placement QUERY, not in Go — see SchedulablePredicate for
	// why that has to be evaluated inside the same locking read as the insert.
	//
	// TRAP: `db.Create` CANNOT write false here. GORM omits zero-value fields that
	// carry a `default:` tag and lets the column default apply, so creating a row
	// with Schedulable:false stores TRUE. Verified against Postgres, not inferred.
	// The direction is the dangerous one: it silently fails to cordon, and the
	// cordon is the first step of every drain and a precondition of every
	// departure — so a create path meaning "start this host cordoned" would
	// produce a host the placement query still hands guests to.
	//
	// CORDON WITH AN UPDATE. `Update("schedulable", false)` is explicit,
	// unaffected by any of this, and is the only form verified to work here.
	//
	// Create-with-Select was tried as a way to store false directly and is NOT
	// recommended: `Select("*")` also forces the primary key, which fights the
	// gen_random_uuid() default, and a field-list Select leaves GORM with no id to
	// read back. Both were observed producing rows that could not then be found by
	// the struct's own ID. If a host must begin life cordoned, create it and
	// update it — two statements in one transaction, and both do what they say.
	//
	// The default:true tag is kept
	// because it IS right for a newly discovered node — the sync creates rows for
	// hosts nobody has expressed intent about, and those are schedulable.
	Schedulable bool `gorm:"not null;default:true" json:"schedulable"`

	// --- ONBOARDING: the one status that is a workflow. ------------------------
	//
	// THREE STATUSES, NOT ONE. "Status" on a machine means at least three
	// unrelated things and they compose freely — a machine can be provisioned AND
	// cordoned AND offline at the same instant. Only this axis is stored, because
	// only it is a workflow rather than an observation:
	//
	//   provisioning   has the agent been installed?   STORED, here
	//   schedulability may new work land here?          Schedulable + derived drain state
	//   reachability   is it answering right now?       derived, never stored (Bee.LivenessAt)
	//
	// A stored copy of the other two drifts the moment a sweep is late, and a UI
	// badge that folds all three into one word misleads precisely when it matters.

	// ProvisioningState is self-scheduling, not enqueued by an API: creating the
	// row records intent and a ticker reconciles toward it.
	//
	//	pending ──claim(lease)──► provisioning ──► provisioned
	//	   ▲                           │
	//	   └──────── redrive ──────────┴────────► failed
	//
	// TRAP, the same one documented at length on Schedulable: a `default:` tag
	// means db.Create CANNOT write the zero value, and for a string that is "".
	// A create path meaning "start this machine pending" must therefore UPDATE
	// the column explicitly — a struct save will store 'provisioned' and the
	// ticker will never look at the row again.
	//
	// The default is 'provisioned' rather than 'pending' and that is a migration
	// decision, not a taste one: every row that exists when this ships is a node
	// a live cluster is reporting, and defaulting them to pending would enqueue
	// an SSH onboarding run against every hypervisor in every tenant on the first
	// tick after deploy.
	ProvisioningState string `gorm:"type:varchar(16);not null;default:'provisioned';check:ck_hypervisors_provisioning_state,provisioning_state IN ('pending','provisioning','provisioned','failed')" json:"provisioning_state"`

	// ProvisioningLeaseExpiresAt is what makes the loop self-healing rather than
	// dependent on somebody noticing.
	//
	// The claim is one transaction — SELECT ... FOR UPDATE SKIP LOCKED over rows
	// that are pending, or provisioning with an EXPIRED lease — so a worker killed
	// mid-onboard leaves a row that reclaims itself. SKIP LOCKED is correct here
	// and deliberately unlike FleetRepoGorm.ClaimNextTask, which avoids it because
	// one BEE must not be handed two tasks; any ticker may take any free machine.
	//
	// It is also the guard on redrive. Redriving a row whose lease is still live
	// puts two workers on one host, both installing keys and reloading sshd.
	ProvisioningLeaseExpiresAt *time.Time `gorm:"type:timestamptz" json:"provisioning_lease_expires_at,omitempty"`

	// ProvisioningAttempt is capped by the ticker. An unreachable host otherwise
	// generates SSH attempts forever; stopping at a cap and requiring an explicit
	// redrive keeps the failure visible instead of letting it become background
	// noise nobody reads.
	ProvisioningAttempt int `gorm:"not null;default:0" json:"provisioning_attempt"`

	// ProvisioningError is the last failure, kept so 'failed' is actionable
	// without going to the logs. Not cleared on a new attempt until that attempt
	// succeeds — an operator looking at a retrying row wants to know what it is
	// retrying past.
	ProvisioningError string `gorm:"type:text;not null;default:''" json:"provisioning_error,omitempty"`

	// LastOperationError is why the last fleet operation on this machine failed,
	// and LastOperationAt is when.
	//
	// Separate from ProvisioningError because they answer different questions.
	// Provisioning is a workflow with its own state column, and a failure there
	// leaves the machine in `failed` where an operator can see it. A DEPARTURE has
	// no state column: the job either succeeds and the machine leaves its cluster,
	// or it fails and River retries it on a backoff — invisibly, because nothing
	// on the machine records that anything was attempted.
	//
	// That is not hypothetical. A departure repeatedly failed with `lookup pve2:
	// no such host` while the operator, seeing a cordoned and drained node that
	// would not leave, had no way to learn the reason existed. A retryable job in
	// a queue nobody reads is the same as no job at all.
	LastOperationError string     `gorm:"type:text;not null;default:''" json:"last_operation_error,omitempty"`
	LastOperation      string     `gorm:"type:varchar(64);not null;default:''" json:"last_operation,omitempty"`
	LastOperationAt    *time.Time `gorm:"type:timestamptz" json:"last_operation_at,omitempty"`

	ProvisioningStartedAt *time.Time `gorm:"type:timestamptz" json:"provisioning_started_at,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at" format:"date-time"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at" format:"date-time"`

	// Intended usage, filled from the placement ledger for a read. NOT a column —
	// `gorm:"-"` — because the moment it is stored it becomes a cache of a number
	// that changes underneath it, and the way that fails is overcommit.
	//
	// A placement decision must compute this itself under the same lock as the
	// insert. These fields exist so a VIEW can show committed capacity without
	// the caller reassembling the join.
	Intended Capacity `gorm:"-" json:"intended"`
}

// Capacity is what the oracle decides against: the allocatable ceiling on one
// node, and the intended usage already committed against it.
//
// Intended usage is NOT a column. It is the sum of this node's placements
// through their pools' slots, read in the same transaction as the decision, so
// that two concurrent placements cannot both see the same free capacity. A
// stored counter cannot give that guarantee — it is a cache of a number that
// changes underneath it.
type Capacity struct {
	CPU    int
	RAMGB  int
	DiskGB int
}

// AllocatableCPU is the ceiling after overcommit and reservation.
//
// Overcommit multiplies the node's real cores; reservation is then subtracted,
// not multiplied, because the host's own overhead is real work and does not
// become cheaper because guests are allowed to share.
//
// Floors at zero: a reservation larger than the overcommitted total is a
// misconfiguration, and returning a negative ceiling would make it look like
// capacity to any caller comparing with >=.
func (h Hypervisor) AllocatableCPU(hive Hive) int {
	ratio := h.CPUOvercommitRatio
	if ratio <= 0 {
		ratio = hive.EffectiveCPUOvercommitRatio()
	}
	return max(int(math.Floor(float64(h.ObservedCPUTotal)*ratio))-h.CPUReserved, 0)
}

// Allocatable is the full ceiling. RAM and disk are NOT overcommitted.
//
// That is a deliberate asymmetry rather than an omission. Overcommitting CPU
// degrades guests; overcommitting RAM kills them, because a hypervisor that
// cannot satisfy a page has nothing to give back and the OOM killer chooses for
// it. Disk is left alone for the same reason at a slower pace: a full datastore
// stops every guest on it at once, and thin provisioning hides how close you are
// until it does.
func (h Hypervisor) Allocatable(hive Hive) Capacity {
	return Capacity{
		CPU:    h.AllocatableCPU(hive),
		RAMGB:  max(h.ObservedRAMGBTotal-h.RAMGBReserved, 0),
		DiskGB: max(h.ObservedDiskGBTotal-h.DiskGBReserved, 0),
	}
}

// Bookable is what a placement may actually use: the allocatable ceiling minus
// what guests THIS SYSTEM DID NOT PLACE are already holding.
//
// One definition, called by both Fits and CandidateFor, for the same reason
// PlacementHoldsCapacity is one definition: two spellings of "how much room is
// there" drift, and the direction they drift in is overcommit.
//
// It does NOT subtract intended usage. That half stays a caller-supplied
// argument because it must be read under the same lock as the insert — see Fits
// — whereas the foreign term comes off the row, since nothing a concurrent
// placement does can change it.
//
// Floors at zero for the reason AllocatableCPU does: a node whose foreign load
// already exceeds its ceiling is oversubscribed, and a negative figure reads as
// capacity to every caller comparing with >=.
func (h Hypervisor) Bookable(hive Hive) Capacity {
	a := h.Allocatable(hive)
	return Capacity{
		CPU:    max(a.CPU-h.ObservedForeignCPUUsed, 0),
		RAMGB:  max(a.RAMGB-h.ObservedForeignRAMGBUsed, 0),
		DiskGB: max(a.DiskGB-h.ObservedForeignDiskGBUsed, 0),
	}
}

// Fits reports whether a slot still fits once `intended` is already committed.
//
// Takes intended usage as an argument rather than reading a column, so the
// caller is forced to supply a number it computed under the same lock as the
// insert. A signature that could be satisfied from a stale row would be an
// invitation to race.
//
// THE CORDON CHECK BELOW IS A BACKSTOP, NOT THE ENFORCEMENT POINT. That is
// SchedulablePredicate, evaluated inside the placement query's SELECT ... FOR
// UPDATE — see drain.go, which explains why eligibility has to be decided inside
// that read and notes that CandidateFor deliberately does not re-check it. This
// check is kept anyway because it can only ever refuse: a caller that forgot the
// predicate gets nothing placed rather than a guest on a cordoned host, and a
// row read before a cordon and used after it is refused on the value it carries.
// It is not sufficient on its own — a filter applied to rows already read cannot
// see a cordon set in between — so removing the predicate on the strength of
// this would reopen exactly the race the predicate exists to close.
func (h Hypervisor) Fits(hive Hive, intended Capacity, s Slot) bool {
	if !h.Schedulable {
		return false
	}
	b := h.Bookable(hive)
	return intended.CPU+s.VCPU <= b.CPU &&
		intended.RAMGB+s.RAMGB <= b.RAMGB &&
		intended.DiskGB+s.DiskGB <= b.DiskGB
}

// Drift is observed usage minus intended usage.
//
// Positive means the node is carrying more than this system placed — a guest
// created by hand, or one whose deletion failed. It is the reason both numbers
// are kept: neither alone can tell you that, and a placement oracle that cannot
// see foreign guests will keep filling a node that is already full.
func (h Hypervisor) Drift(intended Capacity) Capacity {
	return Capacity{
		CPU:    h.ObservedCPUUsed - intended.CPU,
		RAMGB:  h.ObservedRAMGBUsed - intended.RAMGB,
		DiskGB: h.ObservedDiskGBUsed - intended.DiskGB,
	}
}
