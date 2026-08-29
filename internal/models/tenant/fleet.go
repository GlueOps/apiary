package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// A Hive is one provider installation that fleet operations act on — for
// Proxmox, a cluster/datacenter. Bees belong to a hive, and a rotation is
// something you do to a hive.
//
// Hives live in the TENANT database. Fleet membership is a tenant concern: the
// agent credential that admits a host is org-scoped, and everything a fleet
// operation touches — bees, tasks, firewall rulesets — belongs to the customer
// who owns the metal. Only what is needed to work out WHICH tenant a caller is
// stays in control.
type Hive struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Unique within the organization, which here means unique within the
	// database — there is no org column to qualify it with, because the database
	// is the organization.
	Name string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`

	// Provider is the installation's kind. Only proxmox has fleet operations
	// today; the column exists so that adding another does not need a migration.
	Provider    string `gorm:"type:varchar(32);not null;default:'proxmox'" json:"provider"`
	Description string `gorm:"type:text" json:"description,omitempty"`

	// How the control plane reaches this installation's API, for capacity sync
	// and provisioning.
	//
	// These live on Hive rather than on a separate Datacenter row because a
	// Proxmox datacenter IS a hive — one cluster that fleet operations act on.
	// Modelling both meant hv5 could belong to hive A and datacenter B at once,
	// with nothing detecting the disagreement, and a placement oracle happily
	// scheduling onto a host that had left the fleet.
	APIURL string `gorm:"type:varchar(512);not null;default:''" json:"api_url,omitempty"`

	// CredentialID names the tenant Credential holding the API token. A uuid and
	// a real foreign key rather than loose text: an unresolvable credential is a
	// failure at provisioning time, in a job, where it is least visible.
	CredentialID *uuid.UUID  `gorm:"type:uuid" json:"credential_id,omitempty"`
	Credential   *Credential `gorm:"foreignKey:CredentialID;constraint:OnDelete:SET NULL" json:"-"`

	// InsecureSkipVerify accepts a self-signed API certificate.
	//
	// Proxmox ships one by default and clusters are routinely never given a
	// public cert, so this is the same escape hatch the bpg/proxmox Terraform
	// provider offers. It defaults to FALSE and must be set deliberately per
	// hive: the cost is real, because the sealed API token is handed to whatever
	// answers on that address, and a token that can create and destroy VMs is
	// exactly what a machine-in-the-middle wants.
	//
	// It is scoped to one hive rather than global so that turning it on for a lab
	// cluster does not quietly weaken a production one.
	InsecureSkipVerify bool `gorm:"not null;default:false" json:"insecure_skip_verify"`

	// CPUOvercommitRatio is the hive-wide default. A hypervisor may override it.
	// 1.0 means no overcommit; EffectiveCPUOvercommitRatio treats a non-positive
	// stored value as unset rather than as "zero capacity", so a row written
	// before this column existed does not make every host unschedulable.
	CPUOvercommitRatio float64 `gorm:"type:numeric(5,2);not null;default:1.0" json:"cpu_overcommit_ratio"`

	// MinimumNodes gates the fleet-cycle workflow: the smallest node count a
	// departure may be started from. Zero means use DefaultHiveMinimumNodes.
	//
	// Per hive rather than global because a test fleet is legitimately smaller
	// than a production one, and the alternative to a knob is an operator
	// switching the gate off entirely for the one hive that needed a different
	// number. See EffectiveMinimumNodes for why the default is three.
	MinimumNodes int `gorm:"not null;default:0" json:"minimum_nodes"`

	// The firewall ruleset this hive's bees should converge on: a pointer at one
	// FirewallSnapshot row, which holds the rendered body.
	//
	// FirewallRevision only ever INCREASES. A backout is a new revision carrying
	// an old body (FirewallSnapshot.RevertOf records which), never a move
	// backwards. This is not bookkeeping preference: a bee decides whether to
	// converge by comparing what it applied against this number, and the two
	// obvious comparisons disagree when it goes down. `applied != desired`
	// re-applies; `applied < desired` does nothing and leaves the host pinned to
	// the ruleset being backed out — the one case where the backout mattered.
	// Only ever moving up makes both behave identically, so no bee can hold a
	// wrong opinion about it, including an older one already deployed.
	//
	// FirewallDigest is the snapshot's digest, kept here so a lagging fleet can
	// be told from a converged one without reading every snapshot body.
	//
	// Publishing a revision writes the snapshot and moves this pointer in ONE
	// transaction. That is possible only because both rows are in this database:
	// while the pointer lived in control and the snapshot in tenant, no
	// transaction covered the pair, and a failure between the two writes could
	// leave this naming a snapshot that did not exist — every bee in the hive
	// asking for a body nobody could serve.
	//
	// Zero means nothing has been published, which is NOT the same as an empty
	// ruleset. See Bee.AppliedFirewallRevision.
	FirewallRevision    int        `gorm:"not null;default:0" json:"firewall_revision"`
	FirewallDigest      string     `gorm:"type:varchar(71);not null;default:''" json:"firewall_digest,omitempty"`
	FirewallPublishedAt *time.Time `gorm:"type:timestamptz" json:"firewall_published_at,omitempty"`

	// The cluster composition this hive's bees should converge on, published the
	// same way the firewall is and for the same reasons — see FirewallRevision
	// above, which this deliberately mirrors rather than inventing a second
	// convergence protocol beside it.
	//
	// MembershipRevision only ever INCREASES, including when a node is added back
	// after being removed. An agent decides whether to act by comparing what it
	// applied against this number, and a revision that moved backwards would let
	// two agents hold different opinions about the same value.
	//
	// The digest is over the RENDERED composition, so a revision bumped by an
	// edit that changed nothing about the members leaves it identical and no host
	// touches corosync. That matters more here than for the firewall: reloading
	// an identical ruleset is cheap, and re-running a cluster operation is not.
	//
	// Zero means nothing has been published, which is NOT "the cluster should be
	// empty". An agent reads absent as HOLD, exactly as it does for the firewall,
	// because the alternative is that a hive whose publication failed dissolves
	// itself node by node.
	MembershipRevision    int        `gorm:"not null;default:0" json:"membership_revision"`
	MembershipDigest      string     `gorm:"type:varchar(71);not null;default:''" json:"membership_digest,omitempty"`
	MembershipPublishedAt *time.Time `gorm:"type:timestamptz" json:"membership_published_at,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

// Bee statuses. "unavailable" is distinct from "offline" on purpose — see
// Bee.UnavailableUntil.
const (
	BeeStatusOnline      = "online"
	BeeStatusUnavailable = "unavailable"
	BeeStatusOffline     = "offline"
)

// A Bee is the agent on one machine.
//
// It holds facts about the AGENT ONLY. Everything about the machine underneath
// it — where it is, how to reach it, which cluster and swarm it belongs to, what
// the provider observes on it, how far its onboarding got — lives on Hypervisor,
// which is the durable machine record. The link is Hypervisor.BeeID.
//
// HiveID, Hostname and MachineID USED TO BE HERE and were moved deliberately.
// Hive membership was the worst of the three: Hypervisor carried it too, so one
// machine had two answers to "which cluster is this in" with nothing keeping
// them in agreement, and they demonstrably diverged during a migration —
// departure set Bee.HiveID to NULL while the capacity sync deleted the
// Hypervisor row, two mechanisms on two timers. Hostname became
// Hypervisor.Address and MachineID became Hypervisor.MachineID for the simpler
// reason that both are properties of the box, not of the software on it; an
// agent reinstalled on the same machine should not change either.
type Bee struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// APIKeyID is the agent-scoped control.APIKey this host authenticates with.
	//
	// The key itself — hash, secret, scope, revocation — stays in the control
	// database, because it is what resolves WHICH tenant is calling, and that
	// question has to be answerable before any tenant database can be opened.
	// This column is the other end of that lookup: authenticate in control, get
	// the organization and the key id, then find the bee here.
	//
	// It replaces a bare credential hash stored on this row. That could not
	// survive the move: finding a bee by its credential would have meant
	// searching every tenant database for a hash, which is not a lookup anyone
	// should build.
	//
	// No foreign key, deliberately — the referent is in another database. Unlike
	// the hive reference this one is unavoidable, and it is safe in the direction
	// that matters: a revoked or deleted key fails authentication in control and
	// never reaches this row at all.
	APIKeyID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"-"`

	Status       string     `gorm:"type:varchar(16);not null;default:'offline'" json:"status"`
	AgentVersion string     `gorm:"type:varchar(64)" json:"agent_version,omitempty"`
	LastSeenAt   *time.Time `gorm:"type:timestamptz;index" json:"last_seen_at,omitempty"`

	// UnavailableUntil is the heart of the design: an expected absence is
	// ASSERTED IN ADVANCE, never inferred from silence.
	//
	// A rotation task is "apt upgrade and reboot" — the bee is legitimately gone
	// for minutes. No heartbeat timeout can tell that apart from a dead host,
	// because both look identical: nothing arrives. So the bee declares, before
	// it acts, how long it expects to be away and why. Until that deadline the
	// control plane treats silence as expected. Past it, silence is a much
	// STRONGER signal than a missed heartbeat ever was, because the host gave its
	// own word it would be back.
	UnavailableUntil  *time.Time `gorm:"type:timestamptz" json:"unavailable_until,omitempty"`
	UnavailableReason string     `gorm:"type:varchar(200)" json:"unavailable_reason,omitempty"`

	// The firewall ruleset this host reports having loaded.
	//
	// Firewall state converges by PULL, not by dispatch: the heartbeat carries the
	// desired revision, the bee compares it against these, fetches and applies
	// when they differ, and reports back. Nothing fans out, and a host that was
	// offline during a change converges when it returns rather than needing its
	// missed change tracked for it.
	//
	// Reported, therefore not trusted. The digest is stored beside the revision
	// because a host that loaded a truncated or locally-edited ruleset reports the
	// same revision number as a healthy one — the number alone says only what the
	// bee INTENDED to load. Comparing the digest against the ruleset row is what
	// turns "converged" from a claim into something checkable.
	//
	// Zero means nothing applied yet, which is the correct state for a host
	// enrolled before any ruleset existed.
	//
	// NO DESIRED REVISION MEANS HOLD, NEVER FLUSH. An absent instruction is not
	// an instruction to apply an empty ruleset, and a bee that treats it as one
	// either strips the host bare or cuts it off entirely, depending on the
	// chain's default policy — both from silence, and both irreversible from the
	// control plane once the second one lands.
	//
	// This is not a hypothetical branch. A host between fleets belongs to no
	// hive, so "which revision should it be at" has no hive-scoped answer for the
	// whole of a migration; org-wide bindings supply one, but a bee must be
	// correct in the moment before it has resolved anything at all. Same rule as
	// everywhere else here: a check that cannot reach a decision does not act on
	// the absence.
	// AgentDigest is the sha256 of the binary this agent is RUNNING, as
	// "sha256:<hex>", reported by the agent about itself.
	//
	// The decision "does this host need upgrading" is made on this and not on
	// AgentVersion, because a version is a label somebody stamped and a digest is
	// what is actually on disk. The two disagree in an ordinary development
	// build: the catalogue advertises the CONTROL PLANE's version while the
	// binaries it serves carry whatever they were stamped with, and only a
	// release stamps both from one tag.
	//
	// When they disagree the version comparison never converges. The agent
	// upgrades, restarts, reports the same version it had, is judged out of date
	// again, and upgrades again — a loop that reinstalls the fleet's agents
	// forever and looks like the upgrade being broken. Comparing digests ends at
	// the first pass whatever anybody stamped.
	AgentDigest string `gorm:"type:varchar(71);not null;default:''" json:"agent_digest,omitempty"`

	AppliedFirewallRevision int        `gorm:"not null;default:0" json:"applied_firewall_revision"`
	AppliedFirewallDigest   string     `gorm:"type:varchar(71);not null;default:''" json:"applied_firewall_digest,omitempty"`
	AppliedFirewallAt       *time.Time `gorm:"type:timestamptz" json:"applied_firewall_at,omitempty"`

	// What this agent last reported about the cluster it can actually SEE.
	//
	// The host is the better observer and this is where that becomes true. The
	// capacity sync reads the Proxmox API per CLUSTER, so a machine in no cluster
	// is never observed at all — and fleetless is exactly the state a machine is
	// in when it is about to join one. It is also unreachable in the cases that
	// matter: a node that has left a cluster holds no credential the API would
	// accept, and .13 served a broken pveproxy for hours while its agent
	// heartbeated normally throughout.
	//
	// ObservedCluster EMPTY means standalone, and it is an observation rather
	// than a missing one: a standalone node's /cluster/status returns a node
	// entry with nodeid 0 and no cluster entry, so a successful read yielding no
	// name IS the standalone signal. ObservedMembershipAt distinguishes that from
	// "this agent has never said" — without it, an agent that has reported
	// nothing is indistinguishable from one reporting a standalone host, and the
	// two lead to opposite decisions.
	ObservedCluster string                       `gorm:"type:varchar(100);not null;default:''" json:"observed_cluster,omitempty"`
	ObservedMembers datatypes.JSONType[[]string] `gorm:"type:jsonb" json:"observed_members,omitzero"`
	ObservedQuorate bool                         `gorm:"not null;default:false" json:"observed_quorate"`

	// ObservedFingerprint is this host's own API certificate fingerprint, as the
	// host reads it.
	//
	// Needed because a join through the Proxmox API must present the target's
	// fingerprint, and without one `pvecm add` PROMPTS — which in a
	// non-interactive context hangs until something kills it and reports a
	// timeout that says nothing about the cause.
	//
	// Self-reported on purpose, and it is not a trust decision. The alternative
	// is the joining node reading the fingerprint off the peer it is about to
	// trust, which verifies nothing at all. Here the control plane learns it from
	// the peer's OWN authenticated agent session and publishes it to the joiner,
	// so the joiner checks a value that reached it over a different channel than
	// the one it is about to open.
	ObservedFingerprint  string     `gorm:"type:varchar(200);not null;default:''" json:"observed_fingerprint,omitempty"`
	ObservedMembershipAt *time.Time `gorm:"type:timestamptz" json:"observed_membership_at,omitempty"`

	// AppliedMembershipRevision is the composition this agent has converged on.
	// Reported rather than assumed, for the same reason the firewall is: marking
	// a hive converged when an operator clicked Publish would report every
	// switched-off host as compliant.
	AppliedMembershipRevision int        `gorm:"not null;default:0" json:"applied_membership_revision"`
	AppliedMembershipDigest   string     `gorm:"type:varchar(71);not null;default:''" json:"applied_membership_digest,omitempty"`
	AppliedMembershipAt       *time.Time `gorm:"type:timestamptz" json:"applied_membership_at,omitempty"`

	EnrolledAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"enrolled_at"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

// LivenessAt reports what the bee's status should be at a given moment, given
// when it was last seen and what it declared.
//
// Derived rather than stored so that "is this host up" has one definition. A
// stored status drifts the moment a sweep is late, and an operator staring at a
// stale column during a rotation is exactly when it matters most.
func (b *Bee) LivenessAt(now time.Time, graceAfterLastSeen time.Duration) string {
	if b == nil {
		return BeeStatusOffline
	}
	if b.UnavailableUntil != nil && b.UnavailableUntil.After(now) {
		return BeeStatusUnavailable
	}
	if b.LastSeenAt != nil && now.Sub(*b.LastSeenAt) <= graceAfterLastSeen {
		return BeeStatusOnline
	}
	return BeeStatusOffline
}

// Fleet task lifecycle. A task is terminal in exactly one of the last three.
const (
	FleetTaskPending   = "pending"
	FleetTaskClaimed   = "claimed"
	FleetTaskRunning   = "running"
	FleetTaskSucceeded = "succeeded"
	FleetTaskFailed    = "failed"
	FleetTaskCancelled = "cancelled"
)

// FleetTaskIsTerminal is the single definition of "this task is done", shared by
// the claim query and the reporting path so they cannot disagree about whether a
// bee is free to take another.
func FleetTaskIsTerminal(status string) bool {
	return status == FleetTaskSucceeded || status == FleetTaskFailed || status == FleetTaskCancelled
}

// A FleetTask is one unit of work for one bee.
//
// At most one task per bee is non-terminal at a time. That is not a queueing
// nicety — it is what makes "what is this hypervisor doing right now" answerable
// with a single row, which during a datacenter rotation is the only question
// anyone is asking.
type FleetTask struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BeeID uuid.UUID `gorm:"type:uuid;not null;index:idx_fleet_tasks_bee_status,priority:1" json:"bee_id"`
	Bee   Bee       `gorm:"foreignKey:BeeID;constraint:OnDelete:CASCADE" json:"-"`

	// HiveID is the fleet this task ACTS AGAINST, which is not the same question
	// as which fleet the bee is in — during a migration they disagree at every
	// step. "Leave fleet1" targets fleet1 and ends with the bee in none;
	// "confirm hv5 is no longer a member" targets fleet1 while the bee is
	// already out of it; "join fleet2" targets fleet2 before the bee is in it.
	//
	// Nullable, for work that is not about a fleet at all — converging a firewall,
	// a dist-upgrade — including work a host must still accept while it belongs
	// to no fleet.
	//
	// The bee echoes this back on every report, so the control plane reconciles
	// what was DONE against what was asked rather than inferring membership from
	// which task last succeeded.
	HiveID *uuid.UUID `gorm:"type:uuid;index" json:"hive_id,omitempty"`

	Kind string `gorm:"type:varchar(64);not null" json:"kind"`

	// Spec is DESIRED STATE, not a script. The bee decides how to reach it, which
	// is what lets the same task be re-delivered after a crash without the
	// control plane tracking which half of a shell command already ran.
	Spec datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"spec"`

	Status string `gorm:"type:varchar(16);not null;default:'pending';index:idx_fleet_tasks_bee_status,priority:2" json:"status"`
	Phase  string `gorm:"type:varchar(64)" json:"phase,omitempty"`

	// LeaseExpiresAt guards against two API replicas handing the same task to two
	// bees, and against a task staying claimed forever by a bee that died between
	// claiming and reporting.
	LeaseExpiresAt *time.Time `gorm:"type:timestamptz" json:"lease_expires_at,omitempty"`
	Attempt        int        `gorm:"not null;default:0" json:"attempt"`

	Result datatypes.JSON `gorm:"type:jsonb" json:"result,omitempty"`
	Error  string         `gorm:"type:text" json:"error,omitempty"`

	// NULL when an ORG API KEY queued this rather than a person.
	//
	// An org-scoped key is a first-class principal on these endpoints — they are
	// registered SecurityUserOrKey deliberately, so a CI job can drive a fleet —
	// and it has no user behind it. Dereferencing p.UserID for one panicked the
	// handler and returned a 500 with an empty body, which is how this was found.
	//
	// Nullable rather than substituted: attributing the action to whoever last
	// logged in would put a name against something they did not do, and an audit
	// column that is sometimes a lie is worse than one that is sometimes empty.
	// Same reasoning as control.BeeEnrolmentToken.CreatedByID.
	CreatedByID *uuid.UUID `gorm:"type:uuid" json:"created_by_id,omitempty"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	ClaimedAt   *time.Time `gorm:"type:timestamptz" json:"claimed_at,omitempty"`
	StartedAt   *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`
	FinishedAt  *time.Time `gorm:"type:timestamptz" json:"finished_at,omitempty"`
}

// Log stream names. "agent" is the bee's own narration — what it decided and
// why — kept separate from the command output it is narrating, so an operator
// reading a failed rotation can tell the two apart.
const (
	FleetLogStreamStdout = "stdout"
	FleetLogStreamStderr = "stderr"
	FleetLogStreamAgent  = "agent"
)

// FleetTaskLog is one line of output streamed back while a task runs.
//
// This is what makes a fleet operation debuggable without SSH-ing to a
// hypervisor mid-rotation — which during an evacuation is often the one thing
// you must not do.
//
// Seq is assigned BY THE BEE and is unique per task, which is what makes the
// stream correct under at-least-once delivery. The bee batches lines and retries
// a batch it could not confirm, so the same lines arrive twice; ordering by
// arrival would interleave a retry into the middle of later output, and
// deduplicating by content would collapse two genuinely identical lines. A
// monotonic per-task sequence does both jobs, and the unique index makes the
// retry a no-op rather than a duplicate.
type FleetTaskLog struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uniq_fleet_task_logs_task_seq,priority:1" json:"task_id"`
	Task   FleetTask `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"-"`

	Seq int64 `gorm:"not null;uniqueIndex:uniq_fleet_task_logs_task_seq,priority:2" json:"seq"`

	Stream  string `gorm:"type:varchar(8);not null;default:'stdout'" json:"stream"`
	Message string `gorm:"type:text;not null" json:"message"`

	// LoggedAt is the BEE's clock and may disagree with ReceivedAt — a host with
	// bad NTP is common enough in a datacenter that silently trusting either one
	// would mislead. Both are kept so the discrepancy is visible rather than
	// resolved by guesswork.
	LoggedAt   time.Time `gorm:"type:timestamptz;not null" json:"logged_at"`
	ReceivedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"received_at"`
}

// FirewallConverged reports whether this bee has loaded the given ruleset.
//
// Both halves are required. A revision match alone is what the bee claims; the
// digest is what it actually loaded, and they part company precisely in the
// cases worth catching — a partial write, a truncated fetch, a hand-edit on the
// host. A convergence view built on the number alone shows a green fleet in
// exactly those situations.
//
// A desired revision of zero means no ruleset has been published, in which case
// every bee is trivially converged: there is nothing to be behind.
func (b *Bee) FirewallConverged(desiredRevision int, desiredDigest string) bool {
	if desiredRevision == 0 {
		return true
	}
	if b == nil {
		return false
	}
	return b.AppliedFirewallRevision == desiredRevision && b.AppliedFirewallDigest == desiredDigest
}

// DefaultCPUOvercommitRatio is 1.0 — no overcommit. Chosen as the default
// because the alternative fails silently: a host scheduled past its real
// capacity does not refuse work, it degrades every guest on it at once.
const DefaultCPUOvercommitRatio = 1.0

// EffectiveCPUOvercommitRatio treats a non-positive value as unset.
//
// A zero would otherwise mean "this hive has no CPU at all", which is a
// perfectly valid reading of the number and a catastrophic one: every host
// becomes unschedulable and the oracle reports a full datacenter as empty.
func (h Hive) EffectiveCPUOvercommitRatio() float64 {
	if h.CPUOvercommitRatio <= 0 {
		return DefaultCPUOvercommitRatio
	}
	return h.CPUOvercommitRatio
}
