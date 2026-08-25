package apimiddleware

import (
	"context"

	"github.com/google/uuid"
)

type AuthMethod string

const (
	AuthMethodBearer    AuthMethod = "bearer"
	AuthMethodUserKey   AuthMethod = "user_api_key"
	AuthMethodOrgSecret AuthMethod = "org_key_secret"
)

type Principal struct {
	Method         AuthMethod
	UserID         *uuid.UUID
	OrganizationID uuid.UUID
	APIKeyID       *uuid.UUID
	SessionJTI     string
	ActorID        *uuid.UUID
	Impersonating  bool

	// PolicySatisfied separates AUTHN from AUTHZ.
	//
	// Authentication answers who this is. Authorization answers whether they may
	// act, and policy — require_mfa, require_passkey — sits between the two. The
	// traditional shortcut is to let a successful authentication produce
	// authority, which works right up to the moment policy refuses someone who
	// has not yet enrolled: they hold a valid token, and the endpoint that would
	// let them comply is behind the gate refusing them.
	//
	// So a principal can exist and be genuine while not yet being allowed to do
	// anything but satisfy the policy. False means exactly that. Every principal
	// admitted by RequireUserAuth has this true, because passing that gate is
	// what it records.
	//
	// A key credential is satisfied by construction: policy of this kind applies
	// to interactive sessions, and a key has no second factor to present.
	PolicySatisfied bool
}

func (p *Principal) IsImpersonating() bool {
	return p.Impersonating && p.ActorID != nil
}

type principalCtxKey struct{}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	v, ok := ctx.Value(principalCtxKey{}).(*Principal)
	return v, ok
}
