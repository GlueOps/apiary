package apimiddleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

var (
	ErrNotPlatformStaff  = errors.New("caller is not platform staff")
	ErrNestedImpersonate = errors.New("an impersonated session may not act as platform staff")
)

// AssertPlatformRole requires the caller to hold one of the given PLATFORM
// roles — the role on the user record, not a membership role in some
// organization.
//
// The two are unrelated ladders and conflating them is the obvious escalation
// here: anyone can be an owner of an organization they created, and owner
// outranks admin on the membership ladder, so a check that accepted a
// membership role would let any customer reach staff-only endpoints.
//
// An impersonated session is refused unconditionally, whatever role it carries.
// Impersonation issues a session whose SUBJECT is the target user, so the role
// read here is the target's; a support engineer impersonating a customer who
// happens to be staff would otherwise be able to start a second impersonation
// from inside the first, and the audit trail would name the wrong actor.
func AssertPlatformRole(ctx context.Context, users repo.UserRepo, allowed ...control.UserRole) (*Principal, *control.User, error) {
	p, ok := PrincipalFromContext(ctx)
	if !ok || p == nil || p.UserID == nil {
		return nil, nil, ErrNotPlatformStaff
	}
	if p.IsImpersonating() {
		return nil, nil, ErrNestedImpersonate
	}
	if users == nil {
		// A check that cannot reach a decision must deny.
		return nil, nil, fmt.Errorf("%w: user repository is not wired", ErrNotPlatformStaff)
	}

	user, err := users.GetByID(ctx, *p.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve platform role: %w", err)
	}
	if user == nil || !user.IsActive {
		return nil, nil, ErrNotPlatformStaff
	}

	// An empty allowed set denies rather than admitting everyone. A caller that
	// forgets to name a role must not thereby open the endpoint.
	for _, r := range allowed {
		if user.Role == r {
			return p, user, nil
		}
	}
	return nil, nil, ErrNotPlatformStaff
}

// PlatformRoleError maps the assertion onto a response that does not reveal
// whether the caller merely lacks the role or is not staff at all.
func PlatformRoleError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNestedImpersonate):
		return huma.Error403Forbidden("an impersonated session may not perform this action")
	case errors.Is(err, ErrNotPlatformStaff):
		return huma.Error403Forbidden("insufficient permissions")
	default:
		return huma.Error500InternalServerError("permission check failed")
	}
}
