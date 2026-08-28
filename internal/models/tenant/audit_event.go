package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Principal kinds. Everything acting inside the tenant boundary is one of these,
// and an event that cannot name one is not written.
//
// PrincipalKindOrgKey exists because an organization API key has no user behind
// it — it authenticates a tenant, not a person. Recording its actions against a
// nil user would put a fake identity in the trail, and dropping them would lose
// exactly the automation an operator most needs to account for.
const (
	PrincipalKindUser   = "user"
	PrincipalKindOrgKey = "org_key"
	PrincipalKindAgent  = "agent"
	// PrincipalKindSystem is for work no principal asked for: a scheduled sync,
	// a periodic reconcile. Distinct from "unknown", which is not a value this
	// accepts — an action nobody can be named for is the one worth noticing.
	PrincipalKindSystem = "system"
)

// AuditEvent records one action inside a tenant's boundary.
//
// Deliberately the same shape as control.AuthAuditEvent, one boundary down.
// That table answers "what happened to this account"; this one answers "what was
// done to this fleet, and by whom" — and until it existed the second question
// had no answer at all. A hypervisor carried last_operation and
// last_operation_error, which say WHAT was attempted and whether it failed, but
// never who attempted it and only ever for the most recent attempt. An operator
// finding a node uncordoned could not tell a colleague's deliberate action from
// the control plane resetting it, which is the difference between "fine" and "a
// node took placements before anything confirmed it was healthy".
//
// No organization column: the database is the organization, the same reason
// Hive has none.
type AuditEvent struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// PrincipalID and PrincipalKind are WHO. Not nullable, and not a user id:
	// an org key names no person, an agent names no person either, and both act.
	// The kind is what makes the id readable — a uuid alone cannot be looked up
	// without knowing which table it belongs to.
	PrincipalID   uuid.UUID `gorm:"type:uuid;not null;index" json:"principal_id"`
	PrincipalKind string    `gorm:"type:varchar(16);not null" json:"principal_kind"`

	// ActorID is the human behind an impersonated session, when there is one.
	//
	// Separate from PrincipalID because during impersonation both are true and
	// only recording one of them loses the fact that matters: the action was
	// taken as the customer, by a member of platform staff.
	ActorID *uuid.UUID `gorm:"type:uuid" json:"actor_id,omitempty"`

	// Action is what was done, as a stable dotted string. Stable because these
	// are read back months later and a renamed action silently empties a filter.
	Action string `gorm:"type:varchar(64);not null;index" json:"action"`

	// TargetKind and TargetID are WHAT it was done to. Nullable id, because some
	// actions name no single row — publishing a ruleset acts on a whole fleet.
	TargetKind string     `gorm:"type:varchar(32);not null;index:idx_audit_target" json:"target_kind"`
	TargetID   *uuid.UUID `gorm:"type:uuid;index:idx_audit_target" json:"target_id,omitempty"`

	// Outcome distinguishes an action that was requested from one that worked.
	// A refusal is worth recording — an operator who tried to depart a node and
	// was told no is a different fact from one who never tried.
	Outcome string `gorm:"type:varchar(16);not null;default:'ok'" json:"outcome"`
	// Detail carries a refusal's reason. Empty on success.
	Detail string `gorm:"type:text" json:"detail,omitempty"`

	SessionID *uuid.UUID                         `gorm:"type:uuid" json:"session_id,omitempty"`
	IPAddress string                             `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent string                             `gorm:"type:text" json:"user_agent,omitempty"`
	Metadata  datatypes.JSONType[map[string]any] `gorm:"type:jsonb" json:"metadata,omitzero"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();index:idx_audit_created,sort:desc" json:"created_at"`
}

func (AuditEvent) TableName() string { return "audit_events" }

// Outcomes.
const (
	AuditOutcomeOK      = "ok"
	AuditOutcomeRefused = "refused"
	AuditOutcomeFailed  = "failed"
)

// Target kinds.
const (
	AuditTargetHypervisor = "hypervisor"
	AuditTargetHive       = "hive"
	AuditTargetBee        = "agent"
	AuditTargetFirewall   = "firewall"
	AuditTargetEstate     = "estate"
	AuditTargetFleet      = "fleet"
)

// Actions. Dotted, and named for what an operator did rather than for the
// handler that did it — these are read by someone asking what happened, not by
// someone reading the code.
const (
	AuditHypervisorCordon      = "hypervisor.cordon"
	AuditHypervisorUncordon    = "hypervisor.uncordon"
	AuditHypervisorDepart      = "hypervisor.depart"
	AuditHypervisorArrive      = "hypervisor.arrive"
	AuditHypervisorBootstrap   = "hypervisor.bootstrap"
	AuditHypervisorReprovision = "hypervisor.reprovision"
	AuditHypervisorDeclare     = "hypervisor.declare"

	AuditHiveCreate = "cluster.create"
	AuditHiveUpdate = "cluster.update"
	AuditHiveDelete = "cluster.delete"
	AuditHiveSync   = "cluster.sync"

	AuditAgentRemove  = "agent.remove"
	AuditAgentUpgrade = "agent.upgrade"
	AuditAgentEnrol   = "agent.enrolment_token"
	AuditTaskCreate   = "agent.task"

	AuditFirewallPublish = "firewall.publish"
	AuditFirewallRevert  = "firewall.revert"

	AuditEstateRotate = "estate.rotate_credential"
)
