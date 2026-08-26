package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrEnrolmentRefused = errors.New("fleet: enrolment refused")
	ErrLogBatchTooLarge = errors.New("fleet: log batch is too large")
)

const (
	// How long a claimed task stays leased before another claim may take it.
	// Generous, because the work is measured in minutes: a lease shorter than a
	// kernel upgrade would let a second claim reissue a task that is still
	// running.
	fleetTaskLease = 15 * time.Minute

	// A bee is presumed online this long after its last heartbeat. Only relevant
	// when it has NOT declared unavailability — a declared absence is honoured
	// for as long as it was declared.
	beeOnlineGrace = 90 * time.Second

	// Long-poll ceiling. Below any sensible proxy idle timeout, so a parked
	// request is closed by us with an empty answer rather than by an
	// intermediary with an error the bee has to distinguish from a real fault.
	maxHeartbeatWait = 25 * time.Second
	heartbeatPoll    = 1 * time.Second

	// Log limits. A runaway apt-get can emit megabytes, and an unbounded sink is
	// a way for one misbehaving host to fill the control database.
	maxLogLinesPerBatch = 500
	maxLogLineBytes     = 8 << 10
)

// EnrolmentRedeemer consumes an enrolment token and reports which organization
// it admits a host into. Control-side: the caller is unauthenticated and has no
// tenant yet.
type EnrolmentRedeemer interface {
	RedeemToken(ctx context.Context, tokenHash string, now time.Time) (uuid.UUID, error)
}

// AgentKeyMinter issues the agent-scoped credential a bee authenticates with.
// Control-side, because that credential is what resolves the tenant.
type AgentKeyMinter interface {
	Mint(ctx context.Context, in MintInput) (*MintResult, error)
	// Revoke retires a credential a re-enrolment has superseded. Without it the
	// displaced key stays valid in the control database for the life of the
	// organization, with nothing pointing at it and nothing able to find it.
	Revoke(ctx context.Context, id uuid.UUID) error
	Authenticate(ctx context.Context, public, secret string, wantScopes ...string) (*Authenticated, error)
}

// BeeSession is an authenticated bee together with the tenancy it was resolved
// through.
//
// The organization is carried explicitly rather than read off the bee, because
// a tenant row has no organization column — the database IS the organization.
// Everything downstream needs it to reopen that database, so losing it here
// would mean re-authenticating to get it back.
type BeeSession struct {
	Bee            *tenant.Bee
	OrganizationID uuid.UUID
	APIKeyID       uuid.UUID
}

// FleetService owns the two-sided shape the fleet now has: control answers which
// tenant is calling, and every fleet row lives in that tenant's database.
//
// It therefore holds no FleetRepo. One cannot exist before a request arrives,
// because which database it binds to is not known until the caller is
// authenticated.
type FleetService struct {
	tenants   TenantResolver
	enrolment EnrolmentRedeemer
	keys      AgentKeyMinter
	audit     repo.AuthAuditRepo

	// newFleet builds a repository over an already-resolved tenant handle. A
	// field rather than a direct call so tests can drive this service without a
	// provisioned tenant — the same reason TenantResolver is declared on this
	// side.
	newFleet func(db *gorm.DB) repo.FleetRepo
}

// NewFleetService accepts missing dependencies and denies at request time
// rather than refusing to build.
//
// That is the convention here — see Deps.Admission — and it exists because
// api.Build is exercised with partially-populated Deps in the router tests,
// which assert routing and auth properties and have no interest in a tenant
// manager. Refusing here would make an unrelated test suite carry a database.
//
// Denying is not the same as ignoring. Every entry point below checks before it
// acts, so a genuinely misconfigured deployment answers 503 on the fleet surface
// and nothing else, instead of panicking on the first agent heartbeat.
func NewFleetService(tenants TenantResolver, enrolment EnrolmentRedeemer, keys AgentKeyMinter, audit repo.AuthAuditRepo) (*FleetService, error) {
	return &FleetService{
		tenants:   tenants,
		enrolment: enrolment,
		keys:      keys,
		audit:     audit,
		newFleet:  func(db *gorm.DB) repo.FleetRepo { return repo.NewFleetRepoGorm(db) },
	}, nil
}

// fleetFor opens the calling organization's fleet.
//
// Every method goes through here rather than holding a handle, so there is no
// path on which a request touches a tenant database other than its own.
func (s *FleetService) fleetFor(ctx context.Context, orgID uuid.UUID) (repo.FleetRepo, error) {
	// A check that cannot reach a decision must deny. A nil organization would
	// otherwise resolve to whatever GetTenantContext makes of an empty string,
	// and a missing resolver would panic on the first agent heartbeat.
	if orgID == uuid.Nil || isNilResolver(s.tenants) {
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}
	return s.newFleet(tCtx.DB), nil
}

// hashFleetSecret is the one place a fleet secret becomes a stored value. Enrolment
// tokens and bee credentials are both compared, never recovered, so both are
// hashed rather than sealed.
func hashFleetSecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func newSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("fleet: generate secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type EnrolInput struct {
	EnrolmentToken string
	MachineID      string
	Hostname       string
	AgentVersion   string

	// PeerAddress is where the enrolment request came from. See
	// repo.preferredAddress: an agent knows its hostname and not how the control
	// plane reaches it, and the address of the request that just arrived is the
	// one candidate that is reachable by demonstration.
	PeerAddress string
}

type EnrolResult struct {
	Bee            *tenant.Bee
	OrganizationID uuid.UUID

	// The plaintext halves of the agent credential. They exist here and nowhere
	// else: only hashes are stored, so a key that is lost is replaced rather than
	// recovered.
	KeyPublic string
	KeySecret string
}

func (s *FleetService) Enrol(ctx context.Context, in EnrolInput) (*EnrolResult, error) {
	machine := strings.TrimSpace(in.MachineID)
	if machine == "" {
		return nil, fmt.Errorf("%w: a machine id is required", ErrEnrolmentRefused)
	}
	host := strings.TrimSpace(in.Hostname)
	if host == "" {
		return nil, fmt.Errorf("%w: a hostname is required", ErrEnrolmentRefused)
	}
	now := time.Now().UTC()

	// Redeem FIRST, and in its own transaction. The steps below touch a different
	// database, so nothing can cover the whole enrolment atomically, and the
	// order decides which way a crash falls: burning a use on a token that
	// admitted nobody is recoverable by minting another, while leaving a spent
	// token redeemable is not.
	if s.enrolment == nil || s.keys == nil {
		return nil, ErrTenantUnavailable
	}
	orgID, err := s.enrolment.RedeemToken(ctx, hashFleetSecret(in.EnrolmentToken), now)
	if err != nil {
		return nil, err
	}

	// The agent credential is an org-scoped API key on the agent surface. Scope
	// is the surface boundary and the exclusion runs both ways: this key cannot
	// reach the org endpoints, and an org key cannot reach the agent ones.
	minted, err := s.keys.Mint(ctx, MintInput{
		Scope: control.APIKeyScopeAgent,
		OrgID: &orgID,
		Name:  host,
	})
	if err != nil {
		return nil, err
	}

	fleet, err := s.fleetFor(ctx, orgID)
	if err != nil {
		return nil, err
	}

	enrolled, err := fleet.EnrolBee(ctx, repo.EnrolBeeInput{
		// The machine id is hashed. It identifies a physical host, and a database
		// is not the place to keep an inventory of them in clear.
		//
		// Through tenant.MachineIDDigest rather than the local hashFleetSecret, even
		// though the two compute the same thing today. The stored form has a second
		// reader — the onboarding job's proof-of-success lookup — and when the two
		// sides of that comparison were free to pick their own hashing, they picked
		// differently and the lookup could never match.
		MachineID:    tenant.MachineIDDigest(machine),
		Address:      host,
		PeerAddress:  in.PeerAddress,
		AgentVersion: strings.TrimSpace(in.AgentVersion),
		APIKeyID:     minted.Key.ID,
		Now:          now,
	})
	if err != nil {
		return nil, err
	}

	// Revoke the credential this enrolment displaced.
	//
	// AFTER the enrolment, deliberately. The new key is already installed and the
	// agent is already using it, so revoking the old one can only close access
	// that nothing needs. Doing it before would leave a window where a crash
	// between the two left the host with no working credential at all.
	//
	// A FAILURE HERE DOES NOT FAIL THE ENROLMENT. The host in front of us has
	// been admitted and holds a working key; refusing now would strand it while
	// leaving the stale credential live anyway — strictly worse on both counts.
	// It is audited instead, because a credential that outlives its replacement
	// is exactly the kind of thing that should be visible rather than silent.
	if enrolled.ReplacedKeyID != nil {
		if err := s.keys.Revoke(ctx, *enrolled.ReplacedKeyID); err != nil {
			s.auditKeyRevokeFailed(ctx, orgID, *enrolled.ReplacedKeyID, err)
		}
	}

	return &EnrolResult{
		Bee:            enrolled.Bee,
		OrganizationID: orgID,
		KeyPublic:      minted.Public,
		KeySecret:      minted.Secret,
	}, nil
}

// auditKeyRevokeFailed records a credential that outlived the enrolment which
// replaced it.
//
// Best-effort on purpose, at both levels: the enrolment has already succeeded,
// so neither the revocation nor this record may fail it. The error's text is
// recorded but no key material is — the id identifies the row without being
// usable, and a failed revocation is exactly the moment someone pastes a log
// line into a ticket.
func (s *FleetService) auditKeyRevokeFailed(ctx context.Context, orgID, keyID uuid.UUID, cause error) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Create(ctx, &control.AuthAuditEvent{
		SubjectID:      keyID,
		SubjectKind:    "agent",
		OrganizationID: orgID,
		EventType:      "agent_key_revoke_failed",
		Metadata: datatypes.NewJSONType(map[string]any{
			"api_key_id": keyID.String(),
			"error":      cause.Error(),
		}),
	})
}

// Authenticate resolves a bee from the agent credential it presents.
//
// Two steps, in this order, because the first is what makes the second possible:
// the key is verified in the control database, which yields the organization,
// and only then can that organization's database be opened to find the bee.
func (s *FleetService) Authenticate(ctx context.Context, public, secret string) (*BeeSession, error) {
	if s.keys == nil {
		return nil, ErrAPIKeyRejected
	}
	auth, err := s.keys.Authenticate(ctx, public, secret, control.APIKeyScopeAgent)
	if err != nil || auth == nil || auth.Key == nil {
		return nil, ErrAPIKeyRejected
	}

	// An agent key without an organization cannot resolve a tenant, and a check
	// that cannot reach a decision must deny. Mint refuses this shape, so
	// reaching it means the row was written by something else.
	if auth.Key.OrgID == nil {
		return nil, ErrAPIKeyRejected
	}
	fleet, err := s.fleetFor(ctx, *auth.Key.OrgID)
	if err != nil {
		return nil, err
	}
	bee, err := fleet.BeeByAPIKeyID(ctx, auth.Key.ID)
	if err != nil {
		return nil, err
	}
	return &BeeSession{
		Bee:            bee,
		OrganizationID: *auth.Key.OrgID,
		APIKeyID:       auth.Key.ID,
	}, nil
}

type HeartbeatInput struct {
	// Session rather than a bare bee: a tenant row carries no organization, so
	// without it there is nothing to reopen the right database with.
	Session      *BeeSession
	AgentVersion string

	// UnavailableFor is the bee declaring, before it acts, roughly how long it
	// will be gone. Zero leaves any standing declaration untouched.
	UnavailableFor    time.Duration
	UnavailableReason string
	// Back withdraws a standing declaration: the bee has returned.
	Back bool

	// Wait asks the handler to hold the request open until a task appears.
	Wait time.Duration
}

type HeartbeatResult struct {
	Task   *tenant.FleetTask
	Status string
}

// Heartbeat records liveness, applies any declared absence, and returns this
// bee's one outstanding task — waiting for one to appear if asked.
//
// Liveness and task delivery share a request on purpose. Two endpoints would
// mean two ways for a bee to be "seen", and a bee parked on a long poll would
// have to heartbeat separately to avoid looking dead — which is precisely the
// confusion this design exists to remove.
func (s *FleetService) Heartbeat(ctx context.Context, in HeartbeatInput) (*HeartbeatResult, error) {
	if in.Session == nil || in.Session.Bee == nil {
		return nil, repo.ErrBeeNotFound
	}
	fleet, err := s.fleetFor(ctx, in.Session.OrganizationID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	hb := repo.HeartbeatInput{
		BeeID:            in.Session.Bee.ID,
		Now:              now,
		AgentVersion:     in.AgentVersion,
		ClearUnavailable: in.Back,
	}
	if !in.Back && in.UnavailableFor > 0 {
		until := now.Add(in.UnavailableFor)
		hb.UnavailableUntil = &until
		hb.UnavailableReason = in.UnavailableReason
	}
	if err := fleet.RecordHeartbeat(ctx, hb); err != nil {
		return nil, err
	}

	wait := min(in.Wait, maxHeartbeatWait)

	deadline := time.Now().Add(wait)
	for {
		task, err := fleet.ClaimNextTask(ctx, in.Session.Bee.ID, fleetTaskLease)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return &HeartbeatResult{Task: task, Status: tenant.BeeStatusOnline}, nil
		}
		if wait <= 0 || time.Now().After(deadline) {
			return &HeartbeatResult{Status: tenant.BeeStatusOnline}, nil
		}
		// Sleep between polls rather than holding a transaction open. A parked
		// long-poll must not occupy a database connection for its whole life, or
		// a few hundred bees exhaust the pool doing nothing.
		select {
		case <-ctx.Done():
			// The bee hung up. Not an error worth reporting: it will be back.
			return &HeartbeatResult{Status: tenant.BeeStatusOnline}, nil
		case <-time.After(heartbeatPoll):
		}
	}
}

type ReportInput struct {
	Session *BeeSession
	TaskID  uuid.UUID
	Status  string
	Phase   string
	Result  []byte
	Error   string
}

func (s *FleetService) Report(ctx context.Context, in ReportInput) (*tenant.FleetTask, error) {
	if in.Session == nil || in.Session.Bee == nil {
		return nil, repo.ErrBeeNotFound
	}
	switch in.Status {
	case tenant.FleetTaskRunning, tenant.FleetTaskSucceeded,
		tenant.FleetTaskFailed, tenant.FleetTaskCancelled:
	default:
		return nil, fmt.Errorf("fleet: %q is not a status a bee may report", in.Status)
	}

	fleet, err := s.fleetFor(ctx, in.Session.OrganizationID)
	if err != nil {
		return nil, err
	}
	return fleet.ReportTask(ctx, repo.ReportTaskInput{
		BeeID:  in.Session.Bee.ID,
		TaskID: in.TaskID,
		Status: in.Status,
		Phase:  in.Phase,
		Result: in.Result,
		Error:  in.Error,
		Now:    time.Now().UTC(),
	})
}

type LogLine struct {
	Seq      int64
	Stream   string
	Message  string
	LoggedAt time.Time
}

// AppendLogs stores a batch of streamed output.
//
// Lines are truncated rather than rejected: losing the tail of one enormous line
// is a far better outcome during a rotation than discarding a batch that also
// carried the error message explaining what went wrong.
func (s *FleetService) AppendLogs(ctx context.Context, sess *BeeSession, taskID uuid.UUID, lines []LogLine) (int, error) {
	if sess == nil || sess.Bee == nil {
		return 0, repo.ErrBeeNotFound
	}
	if len(lines) > maxLogLinesPerBatch {
		return 0, fmt.Errorf("%w: %d lines, limit is %d", ErrLogBatchTooLarge, len(lines), maxLogLinesPerBatch)
	}

	now := time.Now().UTC()
	rows := make([]tenant.FleetTaskLog, 0, len(lines))
	for _, l := range lines {
		msg := l.Message
		if len(msg) > maxLogLineBytes {
			msg = msg[:maxLogLineBytes] + "\n… truncated by the control plane at 8 KiB"
		}
		stream := l.Stream
		switch stream {
		case tenant.FleetLogStreamStdout, tenant.FleetLogStreamStderr, tenant.FleetLogStreamAgent:
		default:
			stream = tenant.FleetLogStreamStdout
		}
		at := l.LoggedAt
		if at.IsZero() {
			at = now
		}
		rows = append(rows, tenant.FleetTaskLog{
			Seq: l.Seq, Stream: stream, Message: msg, LoggedAt: at, ReceivedAt: now,
		})
	}
	fleet, err := s.fleetFor(ctx, sess.OrganizationID)
	if err != nil {
		return 0, err
	}
	return fleet.AppendTaskLogs(ctx, sess.Bee.ID, taskID, rows)
}

// BeeLiveness is the read side of the declared-absence model, exposed so that
// handlers and any future sweep agree on what "offline" means.
func BeeLiveness(b *tenant.Bee) string {
	return b.LivenessAt(time.Now().UTC(), beeOnlineGrace)
}

// NewEnrolmentToken returns a token to hand to an operator and the hash to
// store. The raw value exists only in this return and in whatever the operator
// pastes onto a host.
func (s *FleetService) NewEnrolmentToken() (raw, hash string, err error) {
	raw, err = newSecret()
	if err != nil {
		return "", "", err
	}
	return raw, hashFleetSecret(raw), nil
}
