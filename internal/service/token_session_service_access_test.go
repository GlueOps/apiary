package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

type accessRuleFakeUsers struct {
	emailVerifiedAt map[uuid.UUID]time.Time
	user            *control.User
	err             error
	calls           int
}

func (f *accessRuleFakeUsers) GetByID(ctx context.Context, id uuid.UUID) (*control.User, error) {
	f.calls++
	return f.user, f.err
}

func (f *accessRuleFakeUsers) Create(ctx context.Context, u *control.User) error { return nil }
func (f *accessRuleFakeUsers) GetByEmail(ctx context.Context, email string) (*control.User, error) {
	return nil, nil
}
func (f *accessRuleFakeUsers) List(ctx context.Context) ([]control.User, error) { return nil, nil }
func (f *accessRuleFakeUsers) UpdateLastLogin(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return nil
}
func (f *accessRuleFakeUsers) UpdateProfile(ctx context.Context, userID uuid.UUID, in repo.UpdateUserProfileInput) error {
	return nil
}
func (f *accessRuleFakeUsers) SetMFAEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return nil
}
func (f *accessRuleFakeUsers) UpdateTokensRevokedAt(ctx context.Context, userID uuid.UUID, at time.Time) error {
	return nil
}
func (f *accessRuleFakeUsers) CountUsersWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *accessRuleFakeUsers) CountUsersWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *accessRuleFakeUsers) BulkRevokeTokensWithoutMFA(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *accessRuleFakeUsers) BulkRevokeTokensWithoutPasskey(ctx context.Context, orgID *uuid.UUID) (int64, error) {
	return 0, nil
}

type accessRuleFakeRules struct {
	err        error
	seenEmails []string
	orgEmails  []string
	denyOrg    error
}

func (f *accessRuleFakeRules) CheckAccess(ctx context.Context, email string) error {
	return f.CheckAuthentication(ctx, email)
}
func (f *accessRuleFakeRules) CheckAuthentication(ctx context.Context, email string) error {
	f.seenEmails = append(f.seenEmails, email)
	return f.err
}

// CheckOrgCreation records separately and can be armed to deny, so a test can
// prove that login never consults the create_org rule set.
func (f *accessRuleFakeRules) CheckOrgCreation(ctx context.Context, email string) error {
	f.orgEmails = append(f.orgEmails, email)
	return f.denyOrg
}
func (f *accessRuleFakeRules) CheckAccessFor(ctx context.Context, email string, purpose repo.AccessPurpose) error {
	if purpose == repo.AccessPurposeCreateOrg {
		return f.CheckOrgCreation(ctx, email)
	}
	return f.CheckAuthentication(ctx, email)
}
func (f *accessRuleFakeRules) List(ctx context.Context) ([]control.AuthAccessRule, error) {
	return nil, nil
}
func (f *accessRuleFakeRules) Create(ctx context.Context, rule *control.AuthAccessRule) error {
	return nil
}
func (f *accessRuleFakeRules) Delete(ctx context.Context, id uuid.UUID) error { return nil }

type accessRuleFakeSessions struct {
	created int
}

func (f *accessRuleFakeSessions) Create(ctx context.Context, s *control.TokenSession) error {
	f.created++
	return nil
}
func (f *accessRuleFakeSessions) GetByRefreshJTI(ctx context.Context, jti string) (*control.TokenSession, error) {
	return nil, nil
}
func (f *accessRuleFakeSessions) GetByAccessJTI(ctx context.Context, jti string) (*control.TokenSession, error) {
	return nil, nil
}
func (f *accessRuleFakeSessions) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	return nil
}
func (f *accessRuleFakeSessions) RevokeByFamilyID(ctx context.Context, familyID uuid.UUID, at time.Time) (int64, error) {
	return 0, nil
}
func (f *accessRuleFakeSessions) RevokeAllBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, at time.Time) (int64, error) {
	return 0, nil
}
func (f *accessRuleFakeSessions) ListActiveBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string) ([]control.TokenSession, error) {
	return nil, nil
}
func (f *accessRuleFakeSessions) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

type accessRuleFakeAudit struct{}

func (accessRuleFakeAudit) Create(ctx context.Context, e *control.AuthAuditEvent) error { return nil }
func (accessRuleFakeAudit) ListBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, limit, offset int) ([]control.AuthAuditEvent, error) {
	return nil, nil
}

type accessRuleHarness struct {
	svc         *TokenSessionService
	users       *accessRuleFakeUsers
	accessRules *accessRuleFakeRules
	sessions    *accessRuleFakeSessions
}

func newAccessRuleHarness(t *testing.T, users *accessRuleFakeUsers, rules *accessRuleFakeRules) accessRuleHarness {
	t.Helper()

	tokens, err := NewTokenService("test-secret-value", "apiary-test", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	sessions := &accessRuleFakeSessions{}
	svc := &TokenSessionService{
		tokens:   tokens,
		sessions: sessions,
		audit:    accessRuleFakeAudit{},
	}
	if users != nil {
		svc.users = users
	}
	if rules != nil {
		svc.accessRules = rules
	}
	return accessRuleHarness{svc: svc, users: users, accessRules: rules, sessions: sessions}
}

func accessRuleIssueInput() IssueTrackedInput {
	return IssueTrackedInput{
		SubjectID:   uuid.NewString(),
		SubjectKind: "user",
		OrgID:       uuid.NewString(),
		EventType:   "login",
	}
}

func TestIssueTracked_DeniesWhenUserLookupErrors(t *testing.T) {
	boom := errors.New("statement timeout")
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{}, err: boom},
		&accessRuleFakeRules{},
	)

	got, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput())
	if err == nil {
		t.Fatalf("IssueTracked returned tokens (%v) despite a failed user lookup; the deny list was never consulted", got)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("want the lookup error wrapped, got %v", err)
	}
	if h.sessions.created != 0 {
		t.Fatalf("a token session was persisted for a subject whose access could not be decided")
	}
	if len(h.accessRules.seenEmails) != 0 {
		t.Fatalf("CheckAuthentication was called with a zero-valued user: %q — an empty email has no \"@\" and every domain rule allows it", h.accessRules.seenEmails)
	}
}

func TestIssueTracked_DeniesWhenUserRowIsGone(t *testing.T) {
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: nil, err: nil},
		&accessRuleFakeRules{},
	)

	if _, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized for a vanished user row, got %v", err)
	}
	if h.sessions.created != 0 {
		t.Fatalf("a token session was persisted for a user row that does not exist")
	}
}

func TestIssueTracked_DeniesWhenUserRepoMissing(t *testing.T) {
	h := newAccessRuleHarness(t, nil, &accessRuleFakeRules{})

	if _, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized when the user repo is not wired, got %v", err)
	}
}

func TestIssueTracked_DeniesUnparsableSubject(t *testing.T) {
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{Email: "u@example.test"}},
		&accessRuleFakeRules{},
	)

	in := accessRuleIssueInput()
	in.SubjectID = "not-a-uuid"
	if _, err := h.svc.IssueTracked(context.Background(), in); err == nil {
		t.Fatalf("want an error for an unparsable subject id, got tokens")
	}
	if h.sessions.created != 0 {
		t.Fatalf("a token session was persisted for an unparsable subject id")
	}
}

func TestIssueTracked_PropagatesDenyVerdict(t *testing.T) {
	denied := errors.New("access denied")
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{Email: "blocked@example.test"}},
		&accessRuleFakeRules{err: denied},
	)

	if _, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput()); !errors.Is(err, denied) {
		t.Fatalf("want the deny verdict, got %v", err)
	}
	if h.sessions.created != 0 {
		t.Fatalf("a token session was persisted for a denied subject")
	}
}

func TestIssueTracked_AllowsPermittedUser(t *testing.T) {
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{Email: "ok@example.test"}},
		&accessRuleFakeRules{},
	)

	issued, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput())
	if err != nil {
		t.Fatalf("IssueTracked: %v", err)
	}
	if issued == nil || issued.AccessToken == "" {
		t.Fatalf("no access token issued for a permitted user")
	}
	if h.sessions.created != 1 {
		t.Fatalf("want 1 persisted session, got %d", h.sessions.created)
	}
	if len(h.accessRules.seenEmails) != 1 || h.accessRules.seenEmails[0] != "ok@example.test" {
		t.Fatalf("CheckAuthentication saw %q", h.accessRules.seenEmails)
	}
	if len(h.accessRules.orgEmails) != 0 {
		t.Fatalf("login consulted the create_org rule set (%q); org-creation rules must not gate authentication", h.accessRules.orgEmails)
	}
}

func TestIssueTracked_CreateOrgDenyDoesNotBlockLogin(t *testing.T) {
	// Finding 2: a user invited into an existing tenant on a domain that may not
	// mint new tenants must still be able to authenticate. Before the purpose
	// split there was one rule set and this login was refused.
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{Email: "invited@blocked.example.test"}},
		&accessRuleFakeRules{denyOrg: repo.ErrAccessDenied},
	)

	issued, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput())
	if err != nil {
		t.Fatalf("a create_org deny blocked authentication: %v", err)
	}
	if issued == nil || issued.AccessToken == "" {
		t.Fatalf("no access token issued for a user denied only org creation")
	}
	if len(h.accessRules.orgEmails) != 0 {
		t.Fatalf("IssueTracked called CheckOrgCreation (%q) — it must only ever ask the authenticate purpose", h.accessRules.orgEmails)
	}
}

func TestIssueTracked_SkipsWhenAccessRulesUnwired(t *testing.T) {
	h := newAccessRuleHarness(t,
		&accessRuleFakeUsers{user: &control.User{Email: "ok@example.test"}},
		nil,
	)

	if _, err := h.svc.IssueTracked(context.Background(), accessRuleIssueInput()); err != nil {
		t.Fatalf("IssueTracked with no access-rule repo: %v", err)
	}
	if h.users.calls != 0 {
		t.Fatalf("the user was loaded even though there is no deny list to check")
	}
}

// MarkEmailVerified satisfies repo.UserRepo. These tests are about the access
// rules consulted at token issuance and never read verification state, so the
// stamp is recorded rather than applied — a fake that silently did nothing
// would still compile if a future test started depending on it.
func (f *accessRuleFakeUsers) MarkEmailVerified(_ context.Context, userID uuid.UUID, at time.Time) error {
	if f.emailVerifiedAt == nil {
		f.emailVerifiedAt = map[uuid.UUID]time.Time{}
	}
	if _, already := f.emailVerifiedAt[userID]; !already {
		f.emailVerifiedAt[userID] = at
	}
	return nil
}
