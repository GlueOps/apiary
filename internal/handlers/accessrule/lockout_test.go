package accessrule

import (
	"context"
	"errors"
	"testing"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

// stubUsers returns one user. The embedded nil interface is deliberate: if the
// guard ever starts calling something else on the repository, the test panics
// with the method name instead of silently passing.
type stubUsers struct {
	repo.UserRepo
	user *control.User
	err  error
}

func (s *stubUsers) GetByID(context.Context, uuid.UUID) (*control.User, error) {
	return s.user, s.err
}

func actorPrincipal() *apimiddleware.Principal {
	id := uuid.New()
	return &apimiddleware.Principal{UserID: &id}
}

// The lockout this guard exists for.
//
// CheckAuthentication runs on session ISSUE, and issue includes refresh. So an
// authenticate deny matching the actor ends their current session too, and
// reversing it requires signing in as a platform admin — which is precisely what
// they can no longer do. One request, unrecoverable, with the recovery path
// inside the blast radius.
func TestRefusesARuleThatWouldDenyTheActorsOwnSignIn(t *testing.T) {
	users := &stubUsers{user: &control.User{Email: "ops@dragonsecurity.io"}}

	cases := []struct {
		name        string
		kind, value string
		purpose     string
		policy      string
		wantRefusal bool
		why         string
	}{
		{
			name: "deny on the actor's exact address", kind: control.AccessRuleKindEmail,
			value: "ops@dragonsecurity.io", purpose: control.AccessRulePurposeAuthenticate,
			policy: control.AccessRulePolicyDeny, wantRefusal: true,
			why: "the actor could not sign in again to undo it",
		},
		{
			name: "deny on the actor's domain", kind: control.AccessRuleKindDomain,
			value: "dragonsecurity.io", purpose: control.AccessRulePurposeAuthenticate,
			policy: control.AccessRulePolicyDeny, wantRefusal: true,
			why: "a domain rule matches the actor through DomainCandidates",
		},
		{
			name: "case differs", kind: control.AccessRuleKindEmail,
			value: "OPS@DragonSecurity.IO", purpose: control.AccessRulePurposeAuthenticate,
			policy: control.AccessRulePolicyDeny, wantRefusal: true,
			why: "matching is case-insensitive, so a differently-cased value locks out just the same",
		},
		{
			// Survivable, and refusing it would block a legitimate act: an
			// operator closing their own public-mailbox domain to org creation.
			name: "create_org deny on the actor's own domain", kind: control.AccessRuleKindDomain,
			value: "dragonsecurity.io", purpose: control.AccessRulePurposeCreateOrg,
			policy: control.AccessRulePolicyDeny, wantRefusal: false,
			why: "it stops them founding an organization, not signing in",
		},
		{
			name: "join_org deny on the actor's own domain", kind: control.AccessRuleKindDomain,
			value: "dragonsecurity.io", purpose: control.AccessRulePurposeJoinOrg,
			policy: control.AccessRulePolicyDeny, wantRefusal: false,
			why: "joining is not signing in",
		},
		{
			// The whole point of the surface. Denying somebody ELSE must work.
			name: "authenticate deny on a different domain", kind: control.AccessRuleKindDomain,
			value: "competitor.example", purpose: control.AccessRulePurposeAuthenticate,
			policy: control.AccessRulePolicyDeny, wantRefusal: false,
			why: "this endpoint exists to let an operator shut somebody out",
		},
		{
			name: "ALLOW on the actor's own domain", kind: control.AccessRuleKindDomain,
			value: "dragonsecurity.io", purpose: control.AccessRulePurposeAuthenticate,
			policy: control.AccessRulePolicyAllow, wantRefusal: false,
			why: "permitting yourself locks nobody out",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := assertDoesNotLockOutActor(context.Background(), users, actorPrincipal(),
				c.kind, c.value, c.purpose, c.policy)
			got := errors.Is(err, ErrWouldLockOutActor)
			if got != c.wantRefusal {
				t.Fatalf("refused=%v want=%v — %s (err=%v)", got, c.wantRefusal, c.why, err)
			}
		})
	}
}

// A check that cannot reach a decision must deny. Here denying costs an operator
// one refused rule; admitting costs them the platform.
func TestRefusesWhenItCannotTellWhoTheActorIs(t *testing.T) {
	deny := func(u repo.UserRepo, p *apimiddleware.Principal) error {
		return assertDoesNotLockOutActor(context.Background(), u, p,
			control.AccessRuleKindDomain, "dragonsecurity.io",
			control.AccessRulePurposeAuthenticate, control.AccessRulePolicyDeny)
	}

	cases := []struct {
		name  string
		users repo.UserRepo
		p     *apimiddleware.Principal
	}{
		{"no user repository", nil, actorPrincipal()},
		{"no principal", &stubUsers{}, nil},
		{"principal without a user id", &stubUsers{}, &apimiddleware.Principal{}},
		{"the lookup failed", &stubUsers{err: context.DeadlineExceeded}, actorPrincipal()},
		{"the actor no longer exists", &stubUsers{user: nil}, actorPrincipal()},
		{"the actor's address cannot be keyed", &stubUsers{user: &control.User{Email: "not-an-address"}}, actorPrincipal()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := deny(c.users, c.p); !errors.Is(err, ErrWouldLockOutActor) {
				t.Fatalf("admitted on an undecidable check: %v", err)
			}
		})
	}
}
