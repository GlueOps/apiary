package apimiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
)

// recordingMemberships stands in for repo.MembershipRepo and records which
// (userID, orgID) pair AssertOrgRole actually asked about — that recording is
// what makes the IDOR case (see TestAssertOrgRole_OrganizationComesFrom...)
// able to prove the organization came from the principal.
//
// It mirrors the real Gorm repository's contract exactly: a missing row comes
// back as (nil, nil), NOT as gorm.ErrRecordNotFound — see
// internal/repo/membership_repo.go GetByUserAndOrg. A fake that returned an
// error for "not found" would drive a branch production never takes, and the
// "no membership" case would then be testing the wrong line.
//
// The seven methods AssertOrgRole does not use panic rather than returning zero
// values. If the assertion were ever rewritten to call ListByUserID and filter
// in Go, a fake returning an empty slice would let every refusal case keep
// passing for the wrong reason; a panic surfaces that rewrite immediately.
type recordingMemberships struct {
	t            *testing.T
	membership   *control.Membership
	err          error
	failIfCalled bool

	// byOrg, when non-nil, takes precedence over membership: it lets one fake
	// answer differently for the caller's org and for another tenant's org.
	byOrg map[uuid.UUID]*control.Membership

	callCount  int
	seenUserID uuid.UUID
	seenOrgID  uuid.UUID
}

func (r *recordingMemberships) GetByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*control.Membership, error) {
	r.callCount++
	r.seenUserID = userID
	r.seenOrgID = orgID
	if r.failIfCalled {
		r.t.Errorf("the membership store must not be consulted on this path, but it was asked about user=%s org=%s", userID, orgID)
	}
	if r.byOrg != nil {
		// Absent key => (nil, nil), the real repo's not-found shape.
		return r.byOrg[orgID], nil
	}
	return r.membership, r.err
}

func (r *recordingMemberships) ListByUserID(context.Context, uuid.UUID) ([]control.Membership, error) {
	panic("recordingMemberships.ListByUserID: not used by AssertOrgRole")
}

func (r *recordingMemberships) ListByOrgID(context.Context, uuid.UUID) ([]control.Membership, error) {
	panic("recordingMemberships.ListByOrgID: not used by AssertOrgRole")
}

func (r *recordingMemberships) GetByID(context.Context, uuid.UUID) (*control.Membership, error) {
	panic("recordingMemberships.GetByID: not used by AssertOrgRole")
}

func (r *recordingMemberships) Create(context.Context, *control.Membership) error {
	panic("recordingMemberships.Create: not used by AssertOrgRole")
}

func (r *recordingMemberships) Delete(context.Context, uuid.UUID) error {
	panic("recordingMemberships.Delete: not used by AssertOrgRole")
}

func (r *recordingMemberships) UpdateRole(context.Context, uuid.UUID, control.MemberRole) error {
	panic("recordingMemberships.UpdateRole: not used by AssertOrgRole")
}

func (r *recordingMemberships) UpdateSupportEnabled(context.Context, uuid.UUID, bool) error {
	panic("recordingMemberships.UpdateSupportEnabled: not used by AssertOrgRole")
}

// bearerPrincipal is the shape RequireUserAuth installs for an ordinary,
// non-impersonated bearer session.
func bearerPrincipal(userID, orgID uuid.UUID) *Principal {
	return &Principal{
		Method:         AuthMethodBearer,
		UserID:         &userID,
		OrganizationID: orgID,
		SessionJTI:     uuid.NewString(),
	}
}

// impersonatingPrincipal is what user_auth.go installs for a platform admin
// acting as the user: an ActorID AND the Impersonating flag, because
// Principal.IsImpersonating requires both.
func impersonatingPrincipal(userID, orgID uuid.UUID) *Principal {
	actorID := uuid.New()
	p := bearerPrincipal(userID, orgID)
	p.ActorID = &actorID
	p.Impersonating = true
	return p
}

func membershipWithRole(userID, orgID uuid.UUID, role control.MemberRole) *control.Membership {
	return &control.Membership{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Role:           role,
	}
}

// statusOf pulls the HTTP status out of a huma error. errorlint is configured
// with asserts:true so a bare type assertion on an error is a lint failure —
// and errors.As is the correct call regardless, since it keeps working if the
// mapped error is ever wrapped on its way out of a handler.
func statusOf(t *testing.T, err error) int {
	t.Helper()
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("expected a huma.StatusError so the refusal gets a real status, got %T: %v", err, err)
	}
	return se.GetStatus()
}

// --- the positive cases -----------------------------------------------------
//
// These four exist so that the refusal cases below mean something. A suite made
// only of refusals passes against a function whose entire body is
// `return nil, ErrNoMembership`; these are the only cases pinning AssertOrgRole
// to an actual decision.

func TestAssertOrgRole_AdminIsAllowedWhereAdminRequired(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	principal := bearerPrincipal(userID, orgID)
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleAdmin)}
	ctx := WithPrincipal(context.Background(), principal)

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin)

	if err != nil {
		t.Fatalf("an admin must be allowed where admin is required: got err=%v", err)
	}
	// The returned principal is the one from the context, not a copy: handlers
	// use it instead of re-reading the context, so a rebuilt or zeroed
	// principal would hand the handler a different organization than the one
	// the assertion was made against.
	if p != principal {
		t.Fatalf("AssertOrgRole must return the principal it checked: got %p, want %p", p, principal)
	}
	if p.OrganizationID != orgID {
		t.Fatalf("returned principal carries the wrong organization: got %s, want %s", p.OrganizationID, orgID)
	}
}

func TestAssertOrgRole_OwnerIsAllowedWhereAdminRequired(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleOwner)}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	// "At least admin" must be expressible as a rank comparison. If the check
	// ever degrades to string equality against minRole, an owner — the most
	// privileged role — is locked out of every admin-gated write.
	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin); err != nil {
		t.Fatalf("an owner outranks admin and must be allowed: got err=%v", err)
	}
}

func TestAssertOrgRole_MemberIsAllowedWhereMemberRequired(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleMember)}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	// This is the read path behind estates-list and estates-get. If plain
	// member stops ranking, every read is refused for the majority role.
	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleMember); err != nil {
		t.Fatalf("a member must be allowed where member is required: got err=%v", err)
	}
}

func TestAssertOrgRole_ImpersonatedAdminIsAllowedWithoutTheOption(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleAdmin)}
	ctx := WithPrincipal(context.Background(), impersonatingPrincipal(userID, orgID))

	// Same impersonating principal and same fake as the refusal case below,
	// only without RefuseImpersonation(). This is what proves that refusal is
	// caused by the OPTION and not by the principal shape or the fake — and it
	// pins that reads stay usable while an operator is troubleshooting.
	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin); err != nil {
		t.Fatalf("without RefuseImpersonation an impersonated admin must be allowed: got err=%v", err)
	}
}

// --- role refusals ----------------------------------------------------------

func TestAssertOrgRole_MemberIsRefusedAWrite(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleMember)}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin)

	if p != nil {
		t.Fatalf("a refused assertion must not hand back a usable principal: got %+v", p)
	}
	if !errors.Is(err, ErrInsufficientRole) {
		t.Fatalf("a member must be refused an admin-gated write: got err=%v, want ErrInsufficientRole", err)
	}
	// 403, never 500: a policy refusal surfacing as a server fault sends the
	// operator hunting an outage, and clients retry 5xx.
	if got := statusOf(t, OrgRoleError(err)); got != http.StatusForbidden {
		t.Fatalf("a wrong-role refusal must map to 403: got %d", got)
	}
}

func TestAssertOrgRole_UnrecognisedRoleIsRefused(t *testing.T) {
	cases := []struct {
		name string
		role control.MemberRole
	}{
		{"unknown role name", control.MemberRole("auditor")},
		// Case matters: the column stores exactly what was written, and
		// "OWNER" must not be treated as owner by a case-insensitive lookup.
		{"case variant of a real role", control.MemberRole("OWNER")},
		// The zero value a row would carry if it were written by a path that
		// bypassed the column default.
		{"empty role", control.MemberRole("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userID, orgID := uuid.New(), uuid.New()
			fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, tc.role)}
			ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

			p, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin)

			if p != nil {
				t.Fatalf("an unrecognised role must not hand back a principal: got %+v", p)
			}
			if !errors.Is(err, ErrInsufficientRole) {
				t.Fatalf("role %q must be refused: got err=%v, want ErrInsufficientRole", tc.role, err)
			}
			// The load-bearing half: a corrupt row in the memberships table
			// must produce a clean 403, not a 500 that lights up the operator's
			// error budget for what is really a data problem.
			if got := statusOf(t, OrgRoleError(err)); got != http.StatusForbidden {
				t.Fatalf("role %q must map to 403, not %d", tc.role, got)
			}
		})
	}
}

func TestAssertOrgRole_UnrecognisedRoleIsRefusedEvenWhenTheRequiredRoleIsUnmapped(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRole("auditor"))}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	// This is the ONLY configuration in which the "is this role known at all"
	// guard is observable. With any real minRole, an unknown role's rank of 0
	// is below the required rank anyway, so the guard's removal changes
	// nothing. Here both sides would be 0, `0 < 0` is false, and without the
	// guard an unknown role would be granted an unknown permission.
	p, err := AssertOrgRole(ctx, fake, control.MemberRole("reviewer"))

	if p != nil {
		t.Fatalf("an unknown role must never be granted an unmapped permission: got %+v", p)
	}
	if !errors.Is(err, ErrInsufficientRole) {
		t.Fatalf("an unknown role against an unmapped minRole must be refused: got err=%v", err)
	}
}

// --- impersonation ----------------------------------------------------------

func TestAssertOrgRole_ImpersonatedAdminIsRefusedUnderRefuseImpersonation(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	// Admin membership on purpose: the ROLE would pass, so a refusal here can
	// only be about impersonation and not about rank.
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleAdmin)}
	ctx := WithPrincipal(context.Background(), impersonatingPrincipal(userID, orgID))

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin, RefuseImpersonation())

	if p != nil {
		t.Fatalf("an impersonated write must not hand back a principal: got %+v", p)
	}
	if !errors.Is(err, ErrImpersonationRefused) {
		t.Fatalf("an impersonated operator must not register a credential: got err=%v, want ErrImpersonationRefused", err)
	}
}

func TestAssertOrgRole_ImpersonationIsRefusedBeforeTheMembershipLookup(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, failIfCalled: true}
	ctx := WithPrincipal(context.Background(), impersonatingPrincipal(userID, orgID))

	_, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin, RefuseImpersonation())

	if !errors.Is(err, ErrImpersonationRefused) {
		t.Fatalf("impersonation must be refused: got err=%v", err)
	}
	// Ordering, not just outcome: the refusal must not depend on the
	// membership store being reachable, and must not hand an impersonated
	// caller a lookup whose timing leaks whether a membership row exists.
	if fake.callCount != 0 {
		t.Fatalf("impersonation must be refused before any membership lookup: got %d lookups", fake.callCount)
	}
}

// --- principal and repository preconditions ---------------------------------

func TestAssertOrgRole_NoMembershipInTheTokensOrgIsRefused(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	// (nil, nil) is exactly what the real repo returns for a missing row.
	fake := &recordingMemberships{t: t, membership: nil, err: nil}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleMember)

	if p != nil {
		t.Fatalf("a non-member must not hand back a principal: got %+v", p)
	}
	if !errors.Is(err, ErrNoMembership) {
		t.Fatalf("a caller with no row in the token's org must be refused: got err=%v, want ErrNoMembership", err)
	}
	if got := statusOf(t, OrgRoleError(err)); got != http.StatusForbidden {
		t.Fatalf("a non-member must get 403, not %d", got)
	}
}

func TestAssertOrgRole_NilMembershipRepoDenies(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	// A perfectly valid principal against a repository that was never wired.
	// A check that cannot reach a decision must deny, not panic and not pass.
	p, err := AssertOrgRole(ctx, nil, control.MemberRoleMember)

	if p != nil {
		t.Fatalf("an unwired repository must not hand back a principal: got %+v", p)
	}
	if !errors.Is(err, ErrNoMembership) {
		t.Fatalf("an unwired repository must deny: got err=%v, want ErrNoMembership", err)
	}
	// Pins the %w wrapping on the "not wired" error: an unwrapped error here
	// would fall through OrgRoleError to a 500 whose body advertises internal
	// wiring state to the caller.
	if got := statusOf(t, OrgRoleError(err)); got != http.StatusForbidden {
		t.Fatalf("an unwired repository must still refuse with 403, not %d", got)
	}
}

func TestAssertOrgRole_RepositoryErrorDeniesAndIsNotReportedAsAPolicyRefusal(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, err: errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleMember)

	if p != nil {
		t.Fatalf("a store outage must deny: got %+v", p)
	}
	// A store outage laundered into "insufficient permissions" sends an
	// operator hunting a role problem that does not exist, and hides an
	// outage behind a 403 nobody pages on.
	if errors.Is(err, ErrNoMembership) || errors.Is(err, ErrInsufficientRole) {
		t.Fatalf("a store outage must not be reported as a policy refusal: got err=%v", err)
	}
	if got := statusOf(t, OrgRoleError(err)); got != http.StatusInternalServerError {
		t.Fatalf("a store outage must map to 500, got %d", got)
	}
}

func TestAssertOrgRole_NoPrincipalInContextIsRefused(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}

	// An operation mistakenly mounted outside RequireUserAuth. The failure mode
	// must be denial, not a membership lookup against a zero uuid — which, if
	// a zero-uuid row ever existed, would be a free grant.
	p, err := AssertOrgRole(context.Background(), fake, control.MemberRoleMember)

	if p != nil {
		t.Fatalf("an unauthenticated context must not hand back a principal: got %+v", p)
	}
	if !errors.Is(err, ErrNoMembership) {
		t.Fatalf("an unauthenticated context must be refused: got err=%v, want ErrNoMembership", err)
	}
	if got := statusOf(t, OrgRoleError(err)); got != http.StatusForbidden {
		t.Fatalf("an unauthenticated context must map to 403, not %d", got)
	}
	if fake.callCount != 0 {
		t.Fatalf("no principal means no lookup at all: got %d lookups", fake.callCount)
	}
}

// An organization key/secret pair acts WITHOUT a membership row.
//
// This replaces an assertion that every userless principal was refused, which
// was correct while org keys were unwired and nothing could present one. They
// are wired now, and the guarantee it was really protecting is narrower than
// "refuse": a machine principal must never be resolved AS IF it had a
// membership. That still holds — the store is never consulted — and its
// authority comes from the credential, which is bound to one organization at
// mint time and cannot name another.
func TestAssertOrgRole_OrgKeyActsWithoutAMembershipRow(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	machine := &Principal{
		Method:         AuthMethodOrgSecret,
		OrganizationID: uuid.New(),
	}
	ctx := WithPrincipal(context.Background(), machine)

	p, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin)
	if err != nil {
		t.Fatalf("an organization credential was refused admin: %v", err)
	}
	if p == nil || p.OrganizationID != machine.OrganizationID {
		t.Fatalf("the organization did not survive the assertion: got %+v", p)
	}
	if fake.callCount != 0 {
		t.Fatalf("a userless principal must not reach the membership store: got %d lookups", fake.callCount)
	}
}

// Owner is the ceiling, and it is what guards membership itself — promoting a
// role, and the last-owner protection. An automation credential that could make
// itself an owner would be one that cannot be revoked by the people it locked
// out.
func TestAssertOrgRole_OrgKeyIsCappedBelowOwner(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	machine := &Principal{Method: AuthMethodOrgSecret, OrganizationID: uuid.New()}
	ctx := WithPrincipal(context.Background(), machine)

	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleOwner); !errors.Is(err, ErrInsufficientRole) {
		t.Fatalf("an organization credential reached owner: got %v, want ErrInsufficientRole", err)
	}
}

// A credential with no organization cannot act on one, whatever it claims.
func TestAssertOrgRole_OrgKeyWithoutAnOrganizationIsRefused(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	ctx := WithPrincipal(context.Background(), &Principal{Method: AuthMethodOrgSecret})

	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleMember); !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("got %v, want ErrNoOrganization", err)
	}
}

// A principal claiming the organization method while carrying a user is a shape
// Mint refuses to produce. Reaching it means something else wrote the principal,
// and a contradictory credential must not be resolved either way.
func TestAssertOrgRole_OrgKeyCarryingAUserIsRefused(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	uid := uuid.New()
	ctx := WithPrincipal(context.Background(), &Principal{
		Method: AuthMethodOrgSecret, OrganizationID: uuid.New(), UserID: &uid,
	})

	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleMember); !errors.Is(err, ErrNoMembership) {
		t.Fatalf("got %v, want ErrNoMembership", err)
	}
	if fake.callCount != 0 {
		t.Fatalf("a contradictory principal reached the membership store: %d lookups", fake.callCount)
	}
}

// A bearer or user-key principal with no user is still refused, and still
// without a lookup. Loosening the org-key case must not have loosened this one.
func TestAssertOrgRole_UserlessBearerPrincipalIsStillRefused(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	ctx := WithPrincipal(context.Background(), &Principal{
		Method: AuthMethodBearer, OrganizationID: uuid.New(),
	})

	if _, err := AssertOrgRole(ctx, fake, control.MemberRoleMember); !errors.Is(err, ErrNoMembership) {
		t.Fatalf("got %v, want ErrNoMembership", err)
	}
	if fake.callCount != 0 {
		t.Fatalf("a userless bearer principal reached the membership store: %d lookups", fake.callCount)
	}
}

// A user key with no X-Org-Id has not chosen a tenant. That is a missing header,
// not a denial — answering 403 would send someone to request access they hold.
func TestAssertOrgRole_UserKeyWithoutAnOrganisationSaysSoDistinctly(t *testing.T) {
	fake := &recordingMemberships{t: t, failIfCalled: true}
	uid := uuid.New()
	ctx := WithPrincipal(context.Background(), &Principal{Method: AuthMethodUserKey, UserID: &uid})

	_, err := AssertOrgRole(ctx, fake, control.MemberRoleMember)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("got %v, want ErrNoOrganization", err)
	}
	if errors.Is(err, ErrNoMembership) {
		t.Fatal("a missing organization must not read as a missing membership")
	}
	if fake.callCount != 0 {
		t.Fatalf("looked up membership for no organization: %d lookups", fake.callCount)
	}
}

// --- the IDOR case, driven through a real huma operation --------------------

type orgProbeInput struct {
	// Every request-controlled position an attacker could aim at.
	OrgID string `path:"org_id"`
	Org   string `query:"org"`
	Body  struct {
		OrganizationID string `json:"organization_id"`
	}
}

type orgProbeOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func TestAssertOrgRole_OrganizationComesFromThePrincipalNotTheRequest(t *testing.T) {
	callerID, callerOrg, victimOrg := uuid.New(), uuid.New(), uuid.New()

	// Admin in the VICTIM's organization only. If the assertion ever resolves
	// the organization from the request, this fake says "yes, admin".
	fake := &recordingMemberships{
		t: t,
		byOrg: map[uuid.UUID]*control.Membership{
			victimOrg: membershipWithRole(callerID, victimOrg, control.MemberRoleAdmin),
		},
	}

	_, api := humatest.New(t, huma.DefaultConfig("org-probe", "1.0.0"))
	// Installed exactly the way RequireUserAuth installs it in production.
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		p := bearerPrincipal(callerID, callerOrg)
		next(huma.WithContext(hctx, WithPrincipal(hctx.Context(), p)))
	})
	huma.Register(api, huma.Operation{
		OperationID: "org-probe",
		Method:      http.MethodPost,
		Path:        "/probe/{org_id}",
	}, func(ctx context.Context, _ *orgProbeInput) (*orgProbeOutput, error) {
		if _, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin); err != nil {
			return nil, OrgRoleError(err)
		}
		out := &orgProbeOutput{}
		out.Body.OK = true
		return out, nil
	})

	path := "/probe/" + victimOrg.String() + "?org=" + victimOrg.String()
	body := map[string]string{"organization_id": victimOrg.String()}

	resp := api.Post(path, body)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("a request naming another tenant's organization must be refused: got %d, body=%s", resp.Code, resp.Body.String())
	}
	// The sharp assertion: not just that the answer was 403, but that the
	// organization the code looked up was the token's. Reading it from the
	// path, query or body is the IDOR that exposes another tenant's cloud
	// credentials.
	if fake.seenOrgID != callerOrg {
		t.Fatalf("membership was resolved against the wrong organization: got %s, want the principal's %s (request named %s)",
			fake.seenOrgID, callerOrg, victimOrg)
	}
	if fake.seenUserID != callerID {
		t.Fatalf("membership was resolved for the wrong user: got %s, want %s", fake.seenUserID, callerID)
	}

	// Positive control on the identical wiring and the identical
	// victim-org-laden request: grant the caller admin in their OWN
	// organization and the same call must now succeed. Without this, the 403
	// above could just as well mean the route was broken.
	fake.byOrg[callerOrg] = membershipWithRole(callerID, callerOrg, control.MemberRoleAdmin)

	resp = api.Post(path, body)

	if resp.Code != http.StatusOK {
		t.Fatalf("with admin in the principal's own organization the same request must succeed: got %d, body=%s", resp.Code, resp.Body.String())
	}
}

// --- OrgRoleError -----------------------------------------------------------

func TestOrgRoleError_NotAMemberAndWrongRoleAreIndistinguishable(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	// Both errors are produced by calling AssertOrgRole for real, so the actual
	// fmt.Errorf wrapping is exercised rather than a hand-built sentinel.
	_, notAMemberErr := AssertOrgRole(ctx, &recordingMemberships{t: t}, control.MemberRoleMember)
	_, wrongRoleErr := AssertOrgRole(ctx,
		&recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRoleMember)},
		control.MemberRoleAdmin)

	if !errors.Is(notAMemberErr, ErrNoMembership) || !errors.Is(wrongRoleErr, ErrInsufficientRole) {
		t.Fatalf("test setup produced the wrong errors: notAMember=%v wrongRole=%v", notAMemberErr, wrongRoleErr)
	}

	notAMemberBody, err := json.Marshal(OrgRoleError(notAMemberErr))
	if err != nil {
		t.Fatalf("marshal not-a-member error: %v", err)
	}
	wrongRoleBody, err := json.Marshal(OrgRoleError(wrongRoleErr))
	if err != nil {
		t.Fatalf("marshal wrong-role error: %v", err)
	}

	// Byte-identical, because any difference at all is an oracle: a prober who
	// can tell "you are not a member" from "you are the wrong role" can
	// enumerate another tenant's membership list one organization at a time.
	if !bytes.Equal(notAMemberBody, wrongRoleBody) {
		t.Fatalf("not-a-member and wrong-role must be indistinguishable:\n  not a member: %s\n  wrong role:   %s",
			notAMemberBody, wrongRoleBody)
	}
	if got := statusOf(t, OrgRoleError(notAMemberErr)); got != http.StatusForbidden {
		t.Fatalf("not-a-member must be 403, got %d", got)
	}
	if got := statusOf(t, OrgRoleError(wrongRoleErr)); got != http.StatusForbidden {
		t.Fatalf("wrong-role must be 403, got %d", got)
	}

	// The guard against the tempting "more helpful error message" refactor to
	// huma.Error403Forbidden(err.Error()) — the pattern already used for scope
	// errors in the estate handler. That single change would echo
	// `have "member", need at least "admin"`, telling a prober their exact role
	// in a tenant, or `caller holds no membership in this organization`,
	// confirming they have none. Byte-identity alone would not catch it,
	// because both arms would leak the same way.
	leaks := []string{"member", "admin", "owner", "auditor", "membership", "need at least"}
	for _, body := range [][]byte{notAMemberBody, wrongRoleBody} {
		lower := strings.ToLower(string(body))
		for _, term := range leaks {
			if strings.Contains(lower, term) {
				t.Fatalf("the 403 body leaks membership detail (%q): %s", term, body)
			}
		}
	}
}

func TestOrgRoleError_ImpersonationRefusalIsItsOwnMessageAndStill403(t *testing.T) {
	if OrgRoleError(nil) != nil {
		t.Fatalf("OrgRoleError(nil) must be nil, or every success path turns into an error response")
	}

	impersonation := OrgRoleError(ErrImpersonationRefused)
	if got := statusOf(t, impersonation); got != http.StatusForbidden {
		t.Fatalf("an impersonation refusal is a policy decision, so 403 and never 500: got %d", got)
	}

	shared := OrgRoleError(ErrNoMembership)
	impersonationBody, err := json.Marshal(impersonation)
	if err != nil {
		t.Fatalf("marshal impersonation error: %v", err)
	}
	sharedBody, err := json.Marshal(shared)
	if err != nil {
		t.Fatalf("marshal shared error: %v", err)
	}

	// Deliberately DIFFERENT from the shared membership refusal, and safe to
	// be different: it discloses a property of the caller's own session, not
	// anyone's membership in any organization, so it is not the probing oracle
	// the previous test forbids. An operator who sees it knows to drop
	// impersonation rather than to go asking for a role change.
	if bytes.Equal(impersonationBody, sharedBody) {
		t.Fatalf("an impersonation refusal must be distinguishable from a membership refusal: both are %s", impersonationBody)
	}

	// An error this mapper does not recognise is a genuine fault, and must not
	// be silently downgraded into a 403 that hides a bug.
	if got := statusOf(t, OrgRoleError(errors.New("boom"))); got != http.StatusInternalServerError {
		t.Fatalf("an unrecognised error must map to 500, got %d", got)
	}
}

// --- integrator additions: two fail-open holes closed after the parallel run --
//
// Both were reported by implementers as real but out of their area. Each test
// below was watched go RED against the code as it stood before the fix.

// A context carrying a TYPED nil *Principal is the case the original guard
// missed: PrincipalFromContext type-asserts to *Principal, so it hands back
// (nil, true) — ok is TRUE — and the old `!ok || p.UserID == nil` then
// dereferenced nil and panicked. A panic on the deny path is a 500 where a 403
// belongs, and "a check that cannot reach a decision must DENY" is violated by
// a check that crashes instead of deciding.
func TestAssertOrgRole_TypedNilPrincipalDeniesInsteadOfPanicking(t *testing.T) {
	memberships := &recordingMemberships{t: t, failIfCalled: true}

	// Exactly how a context ends up holding a typed nil: WithPrincipal is handed
	// a nil *Principal by a caller that did not check its own construction.
	var missing *Principal
	ctx := WithPrincipal(context.Background(), missing)

	// If AssertOrgRole panics, the test fails here rather than reporting a
	// misleading assertion failure further down.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AssertOrgRole panicked on a typed-nil principal instead of denying: %v", r)
		}
	}()

	p, err := AssertOrgRole(ctx, memberships, control.MemberRoleMember)
	if !errors.Is(err, ErrNoMembership) {
		t.Fatalf("a typed-nil principal must deny with ErrNoMembership, got %v", err)
	}
	if p != nil {
		t.Fatalf("a refused assertion must not hand back a principal, got %+v", p)
	}
	if statusOf(t, OrgRoleError(err)) != http.StatusForbidden {
		t.Fatalf("the refusal must map to 403, not 500")
	}
	if memberships.callCount != 0 {
		t.Fatalf("the membership store was consulted for a caller with no principal")
	}
}

// The rank map was read bare — `orgRoleRank[minRole]` — so a minRole absent from
// the map ranked 0, and no real role ranks below 0. "Require this role" then
// silently meant "require nothing": every member of the organization passed.
// The hole opens the moment somebody adds a role constant to control without
// also adding it to orgRoleRank, which is a one-line omission with no compiler
// or test consequence before this test existed.
func TestAssertOrgRole_UnmappedRequiredRoleDeniesRatherThanAdmittingEveryone(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()

	// A role constant that exists for the caller but that orgRoleRank has never
	// heard of on the REQUIRED side — the shape of a future "billing-admin".
	const unmapped = control.MemberRole("reviewer")
	if _, ranked := orgRoleRank[unmapped]; ranked {
		t.Fatalf("this test needs a role absent from orgRoleRank; %q is mapped", unmapped)
	}

	// The caller holds a perfectly ordinary, fully recognised membership. That
	// is the point: before the fix, requiring an unmapped role admitted the
	// LOWEST-ranked real role in the organization.
	for _, role := range []control.MemberRole{
		control.MemberRoleMember,
		control.MemberRoleAdmin,
		control.MemberRoleOwner,
	} {
		t.Run(string(role), func(t *testing.T) {
			memberships := &recordingMemberships{
				t:          t,
				membership: membershipWithRole(userID, orgID, role),
			}
			ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

			p, err := AssertOrgRole(ctx, memberships, unmapped)
			if err == nil {
				t.Fatalf("requiring the unmapped role %q admitted a %q — the requirement evaluated to nothing", unmapped, role)
			}
			if !errors.Is(err, ErrInsufficientRole) {
				t.Fatalf("want ErrInsufficientRole, got %v", err)
			}
			if p != nil {
				t.Fatalf("a refused assertion must not hand back a principal, got %+v", p)
			}
			if statusOf(t, OrgRoleError(err)) != http.StatusForbidden {
				t.Fatalf("the refusal must map to 403, not 500")
			}
		})
	}
}

// TestAssertOrgRole_UnrecognisedRoleIsRefused passes with the `known` guard
// deleted: the rank comparison alone already refuses an unknown role, because
// rank 0 sits below every real minRole. The guard's only observable effect is
// its MESSAGE, which names the offending role — so that is what pins it, and
// deleting the guard turns this red.
func TestUnrecognisedRoleIsNamedByItsOwnGuard(t *testing.T) {
	userID, orgID := uuid.New(), uuid.New()
	fake := &recordingMemberships{t: t, membership: membershipWithRole(userID, orgID, control.MemberRole("reviewer"))}
	ctx := WithPrincipal(context.Background(), bearerPrincipal(userID, orgID))

	_, err := AssertOrgRole(ctx, fake, control.MemberRoleMember)
	if err == nil {
		t.Fatal("an unrecognised role must be refused")
	}
	if !strings.Contains(err.Error(), "unrecognised role") || !strings.Contains(err.Error(), "reviewer") {
		t.Fatalf("the refusal must come from the unknown-role guard and name the role, got: %v", err)
	}
}
