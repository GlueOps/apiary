package apimiddleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

var (
	ErrNoMembership = errors.New("caller holds no membership in this organization")

	// ErrNoOrganization is distinct from ErrNoMembership on purpose. A user API
	// key identifies a person, not a tenant, and a person may belong to several —
	// so the organization arrives in X-Org-Id and may simply be absent. "You
	// forgot a header" and "you are not a member there" need different answers,
	// or the first reads as the second and sends someone to ask for access they
	// already have.
	ErrNoOrganization   = errors.New("no organization selected for this request")
	ErrInsufficientRole = errors.New("caller's role is below the required role")

	// ErrNoActingUser is what Principal.ActingUser returns for an org key.
	//
	// Not an authorization failure — the key may hold every role the operation
	// requires. It is a failure to identify a PERSON, which these operations need
	// because they record who acted, so the message names the credential to use
	// rather than the permission to acquire.
	ErrNoActingUser = errors.New(
		"this action records who performed it, and an organization API key identifies " +
			"the organization rather than a person; use a user session or a user API key")
	ErrImpersonationRefused = errors.New("impersonated sessions may not perform this action")
)

// orgRoleRank orders membership roles. AssertOrgRole compares ranks rather than
// strings so that "at least admin" is expressible; comparing role names
// directly would silently accept an unknown role as equal to none.
var orgRoleRank = map[control.MemberRole]int{
	control.MemberRoleMember: 1,
	control.MemberRoleAdmin:  2,
	control.MemberRoleOwner:  3,
}

// AssertOrgRoleOption adjusts the assertion.
type AssertOrgRoleOption func(*assertOrgRoleOpts)

type assertOrgRoleOpts struct {
	refuseImpersonation bool
	requireActingUser   bool
}

// RefuseImpersonation rejects a session created by a platform admin acting as
// the user. Apply it to anything that grants standing access rather than
// reading state — inviting a member, changing a role, or registering a
// credential. An impersonated operator troubleshooting an account should not be
// able to leave themselves a way back in.
func RefuseImpersonation() AssertOrgRoleOption {
	return func(o *assertOrgRoleOpts) { o.refuseImpersonation = true }
}

// RequireActingUser rejects a credential that names no person. Apply it to
// anything that records WHO acted — an audit row, an inviter, the author of a
// policy change.
//
// An organization key is tenant-bound by design and carries no UserID, and this
// gate admits one as an admin without a membership row. The handler behind it
// then has an actor column it cannot fill. Declared here rather than checked
// mid-handler so that the requirement reads off the route, beside the role and
// the impersonation rule, instead of being something a reader has to find.
func RequireActingUser() AssertOrgRoleOption {
	return func(o *assertOrgRoleOpts) { o.requireActingUser = true }
}

// AssertOrgRole resolves the caller's membership in the organization their
// token names, and requires at least minRole.
//
// The organization comes from the PRINCIPAL, never from the request. Reading it
// from a path or body parameter is the obvious IDOR here, and in this system it
// would grant a stranger a membership-shaped view of another tenant's cloud
// credentials.
//
// It returns the principal so a handler can use it without fetching twice.
func AssertOrgRole(
	ctx context.Context,
	memberships repo.MembershipRepo,
	minRole control.MemberRole,
	opts ...AssertOrgRoleOption,
) (*Principal, error) {
	var o assertOrgRoleOpts
	for _, fn := range opts {
		fn(&o)
	}

	// p == nil is load-bearing, not defensive noise: PrincipalFromContext type-
	// asserts to *Principal, so a context carrying a TYPED nil *Principal returns
	// (nil, true) and the p.UserID read below dereferences nil. A panic is not a
	// denial -- it is a 500 on a path whose whole job is to refuse -- and a check
	// that cannot reach a decision must deny.
	p, ok := PrincipalFromContext(ctx)
	if !ok || p == nil {
		return nil, ErrNoMembership
	}
	if o.refuseImpersonation && p.IsImpersonating() {
		return nil, ErrImpersonationRefused
	}
	// Ahead of the role ranking on purpose. An org key ranks admin and would
	// clear the bar, and answering "insufficient permissions" would send its
	// holder looking for a role to grant a credential no role can fix.
	if o.requireActingUser && p.UserID == nil {
		return nil, ErrNoActingUser
	}

	// An org key/secret pair has no user behind it, so there is no membership row
	// to look up. Its authority comes from the credential itself, which is bound
	// to exactly one organization at mint time and cannot name another.
	//
	// It ranks ADMIN, never owner. An org key does the work — estates, fleets,
	// firewall rules — but owner is what guards membership itself: promoting a
	// role, and the last-owner protection. A leaked automation credential that
	// could make itself an owner would be a credential that cannot be revoked by
	// the people it locked out.
	if p.Method == AuthMethodOrgSecret {
		if p.UserID != nil {
			// A shape that contradicts itself. Mint refuses to produce it, so
			// reaching here means something else wrote the principal.
			return nil, fmt.Errorf("%w: an organization credential must not carry a user", ErrNoMembership)
		}
		if p.OrganizationID == uuid.Nil {
			return nil, ErrNoOrganization
		}
		need, ranked := orgRoleRank[minRole]
		if !ranked {
			return nil, fmt.Errorf("%w: unrecognised required role %q", ErrInsufficientRole, minRole)
		}
		if orgRoleRank[control.MemberRoleAdmin] < need {
			return nil, fmt.Errorf("%w: an organization credential ranks %q, need at least %q",
				ErrInsufficientRole, control.MemberRoleAdmin, minRole)
		}
		return p, nil
	}

	if p.UserID == nil {
		return nil, ErrNoMembership
	}
	// A user key names a person, not a tenant. Without X-Org-Id there is nothing
	// to check membership against, and looking one up for the nil organization
	// would answer "not a member" to what is really a missing header.
	if p.OrganizationID == uuid.Nil {
		return nil, ErrNoOrganization
	}
	if memberships == nil {
		// A missing repository cannot establish a role, and a check that cannot
		// reach a decision must deny.
		return nil, fmt.Errorf("%w: membership repository is not wired", ErrNoMembership)
	}

	m, err := memberships.GetByUserAndOrg(ctx, *p.UserID, p.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("resolve membership: %w", err)
	}
	if m == nil {
		return nil, ErrNoMembership
	}

	have, known := orgRoleRank[m.Role]
	if !known {
		// An unrecognised role is not a licence to proceed.
		return nil, fmt.Errorf("%w: unrecognised role %q", ErrInsufficientRole, m.Role)
	}
	need, ranked := orgRoleRank[minRole]
	if !ranked {
		// Fail CLOSED on a required role this map has never heard of. Reading the
		// map bare gave an unmapped minRole a rank of 0, and no real role ranks
		// below 0 -- so "require billing-admin" silently degraded into "require
		// nothing" and admitted every member of the organization. Adding a role
		// constant without adding it here must break loudly, not open the door.
		return nil, fmt.Errorf("%w: unrecognised required role %q", ErrInsufficientRole, minRole)
	}
	if have < need {
		return nil, fmt.Errorf("%w: have %q, need at least %q", ErrInsufficientRole, m.Role, minRole)
	}
	return p, nil
}

// OrgRoleError maps an assertion failure onto an HTTP response without
// distinguishing "you are not a member" from "you are the wrong role" — both are
// 403 with the same body, so the endpoint cannot be used to probe another
// tenant's membership list.
func OrgRoleError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrImpersonationRefused):
		return huma.Error403Forbidden("impersonated sessions may not perform this action")
	case errors.Is(err, ErrNoOrganization):
		// 400, not 403: nothing was denied, the request never named a tenant.
		// Answering 403 would send someone to request access they already hold.
		return huma.Error400BadRequest("this endpoint acts on one organization; send it in the X-Org-Id header")
	case errors.Is(err, ErrNoActingUser):
		// 403 with the sentinel's own text. "Insufficient permissions" would be
		// the wrong diagnosis and send the holder of a fully-privileged org key
		// looking for a role to grant it; the credential is what has to change.
		return huma.Error403Forbidden(ErrNoActingUser.Error())
	case errors.Is(err, ErrNoMembership), errors.Is(err, ErrInsufficientRole):
		return huma.Error403Forbidden("insufficient permissions for this organization")
	default:
		return huma.NewError(http.StatusInternalServerError, "permission check failed")
	}
}
