package workflow

import (
	"context"
	"fmt"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
)

type Enqueuer interface {
	EnqueueProvisionTenant(ctx context.Context, args *ProvisionTenantArgs) error
	EnqueueDestroyTenant(ctx context.Context, args *DestroyTenantArgs) error
	EnqueueSendEmail(ctx context.Context, args *SendEmailArgs) error

	// Fleet lifecycle. A migration is a departure and then an arrival, enqueued
	// separately by an operator, because the fleetless state between them is real
	// and a failed arrival must leave a standalone host rather than a
	// half-migrated one.
	EnqueueDepartHypervisor(ctx context.Context, args *DepartHypervisorArgs) error
	EnqueueArriveHypervisor(ctx context.Context, args *ArriveHypervisorArgs) error
	EnqueueSyncHiveCapacity(ctx context.Context, args *SyncHiveCapacityArgs) error
}

// SendEmailArgs is a queued outbound email.
//
// Mail used to be sent from a bare goroutine that logged on failure and dropped
// it. A transient SMTP fault therefore lost the message outright: for a
// registration confirmation that means an account which exists, cannot log in
// (the confirmation gate), and whose organization is never provisioned — with a
// single log line as the only trace. Going through River buys retries with
// backoff, and a failure that is visible in the job table rather than in
// scrollback.
//
// The BODY IS SEALED, and that is the point of this shape. It contains a live
// token — a confirmation link, a password reset, an invitation — and River
// persists args as JSONB in the control DB and retains them after completion.
// Storing the rendered body in clear would put a working credential next to the
// table that deliberately keeps only its SHA-256, which is the whole reason that
// table hashes. Sealed, the queue holds ciphertext the worker can open and a
// database reader cannot.
//
// Subject and To stay in clear: the worker needs no key to route or log a job,
// and neither is a credential.
type SendEmailArgs struct {
	To      string `json:"to"`
	Subject string `json:"subject"`

	// The sealed body, as base64 produced by utils.Seal.
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Tag        string `json:"tag"`
	KeyID      string `json:"key_id"`
}

func (SendEmailArgs) Kind() string { return "email.send" }

type ProvisionTenantArgs struct {
	OrganizationID string `json:"organization_id"`
}

func (ProvisionTenantArgs) Kind() string { return "tenant.provision" }

type DestroyTenantArgs struct {
	OrganizationID string `json:"organization_id"`
}

func (DestroyTenantArgs) Kind() string { return "tenant.destroy" }

type TokenSessionCleanupArgs struct{}

func (TokenSessionCleanupArgs) Kind() string { return "auth.token_session_cleanup" }

type OrgProvingArgs struct{}

func (OrgProvingArgs) Kind() string { return "admission.org_proving" }

type DomainCohortAuditArgs struct{}

func (DomainCohortAuditArgs) Kind() string { return "admission.domain_cohort_audit" }

// queuePayloadKeyID names the key a sealed payload was written under. utils.Seal
// refuses uuid.Nil, because a key id is what lets a rotation tell re-wrapped rows
// from un-rewrapped ones. Queue payloads are short-lived and never rotated in
// place — a job outlives its key by minutes, not months — so a single stable id
// is honest here, and it still distinguishes payloads from any future scheme.
var QueuePayloadKeyID = uuid.MustParse("00000000-0000-0000-0000-0000000000e1")

// recipientID turns an address into a stable uuid for the AAD, so the binding
// survives an address of any length and needs no separate column.
func recipientID(to string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("mailto:"+to))
}

// emailBodyAAD binds a sealed body to its recipient. A body lifted from one job
// row and replayed on another addressed elsewhere will not open, so a queue
// writer cannot redirect somebody's confirmation link to an address they control.
//
// Sealing and opening are written here, together, precisely so the AAD cannot
// drift between them: a mismatch would not be a compile error, it would be every
// email failing to send at runtime.
func emailBodyAAD(to string) []byte {
	return utils.AAD("email_jobs", uuid.Nil, recipientID(to))
}

// SealEmailBody renders a SendEmailArgs with its body sealed under the queue
// payload key.
func SealEmailBody(key []byte, to, subject, htmlBody string) (*SendEmailArgs, error) {
	sealed, err := utils.Seal([]byte(htmlBody), QueuePayloadKeyID, key, emailBodyAAD(to))
	if err != nil {
		return nil, fmt.Errorf("email: seal body: %w", err)
	}
	return &SendEmailArgs{
		To:         to,
		Subject:    subject,
		Ciphertext: sealed.Ciphertext,
		IV:         sealed.IV,
		Tag:        sealed.Tag,
		KeyID:      sealed.KeyID.String(),
	}, nil
}

// OpenEmailBody is the worker's half.
func OpenEmailBody(key []byte, args *SendEmailArgs) (string, error) {
	keyID, err := uuid.Parse(args.KeyID)
	if err != nil {
		return "", fmt.Errorf("email: job carries an unparseable key id: %w", err)
	}
	body, err := utils.Sealed{
		KeyID:      keyID,
		Ciphertext: args.Ciphertext,
		IV:         args.IV,
		Tag:        args.Tag,
	}.Open(key, emailBodyAAD(args.To))
	if err != nil {
		return "", fmt.Errorf("email: open sealed body: %w", err)
	}
	return string(body), nil
}

// DepartHypervisorArgs removes one host from its hive, leaving it standalone and
// fleetless. Mirrors jobs.DepartHypervisorArgs and is matched to it by Kind —
// the same split every job kind here uses, so a handler never imports the worker
// package.
type DepartHypervisorArgs struct {
	OrganizationID string `json:"organization_id"`
	HypervisorID   string `json:"hypervisor_id"`

	// Mirrors jobs.DepartHypervisorArgs.
	CredentialDisposition string `json:"credential_disposition"`
}

func (DepartHypervisorArgs) Kind() string { return "fleet.depart_hypervisor" }

// ArriveHypervisorArgs adds one standalone host to a hive.
//
// Keyed on the HYPERVISOR, symmetrically with departure. The old asymmetry — keyed
// on the bee — was forced by a Hypervisor row meaning "something a capacity sync
// observed in a cluster", which a host outside one did not have. That row is the
// durable machine record now and exists before any cluster membership does.
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

// SyncHiveCapacityArgs refreshes one hive's capacity observations on demand.
//
// The scheduled fan-out does this on a timer; this is the same job enqueued by an
// operator who does not want to wait for the next tick — most often because a
// departure just refused with drain_unknown.
type SyncHiveCapacityArgs struct {
	OrganizationID string `json:"organization_id"`
	HiveID         string `json:"hive_id"`
}

func (SyncHiveCapacityArgs) Kind() string { return "fleet.sync_hive_capacity" }
