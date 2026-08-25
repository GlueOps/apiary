package apimiddleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

// Headers carrying a non-bearer credential. They match the security schemes the
// served spec declares, so what the documentation says and what the middleware
// reads cannot drift apart.
const (
	UserKeyHeader   = "X-API-Key"
	OrgKeyHeader    = "X-ORG-Key"
	OrgSecretHeader = "X-ORG-Secret"

	// OrgIDHeader selects which organization a USER credential is acting in.
	//
	// Only a user credential needs it: a person may belong to several
	// organizations, so their key identifies who they are and not which tenant
	// they mean. An org key is already bound to one and ignores this entirely —
	// letting a header redirect it would be handing a tenant-bound credential a
	// way to name a different tenant.
	OrgIDHeader = "X-Org-Id"
)

var errAmbiguousCredential = errors.New("more than one credential presented")

// presentedCredential is what the caller sent, before any of it is trusted.
type presentedCredential struct {
	bearer    string
	userKey   string
	orgKey    string
	orgSecret string
}

// credentialFrom reads the request's credential and refuses more than one.
//
// Refusing rather than picking a winner. With a precedence order, a client that
// keeps a stale Authorization header while adopting an org key silently
// authenticates as the wrong principal — and it works, so nobody notices until
// the audit log is read months later. There is no legitimate reason to send two.
func credentialFrom(hctx huma.Context) (presentedCredential, error) {
	c := presentedCredential{
		bearer:    bearerToken(hctx),
		userKey:   strings.TrimSpace(hctx.Header(UserKeyHeader)),
		orgKey:    strings.TrimSpace(hctx.Header(OrgKeyHeader)),
		orgSecret: hctx.Header(OrgSecretHeader),
	}
	kinds := 0
	if c.bearer != "" {
		kinds++
	}
	if c.userKey != "" {
		kinds++
	}
	if c.orgKey != "" || strings.TrimSpace(c.orgSecret) != "" {
		kinds++
	}
	if kinds > 1 {
		return presentedCredential{}, errAmbiguousCredential
	}
	return c, nil
}

// resolveAPIKeyPrincipal authenticates a key-based credential.
//
// It returns (nil, nil) when no key credential was presented, so the caller can
// fall through to bearer handling. Any other nil principal is a refusal.
func resolveAPIKeyPrincipal(
	ctx context.Context,
	keys *service.APIKeyService,
	memberships repo.MembershipRepo,
	c presentedCredential,
) (*Principal, error) {
	switch {
	case c.orgKey != "" || strings.TrimSpace(c.orgSecret) != "":
		return resolveOrgKey(ctx, keys, c)
	case c.userKey != "":
		return resolveUserKey(ctx, keys, memberships, c)
	default:
		return nil, nil
	}
}

func resolveOrgKey(ctx context.Context, keys *service.APIKeyService, c presentedCredential) (*Principal, error) {
	// Both halves, or neither. A key with an empty secret must never fall through
	// to a comparison against an empty stored value.
	if c.orgKey == "" || strings.TrimSpace(c.orgSecret) == "" {
		return nil, errors.New("an organization credential needs both a key and a secret")
	}
	if keys == nil {
		// A check that cannot reach a decision must deny.
		return nil, errors.New("api keys are not available")
	}

	// The scope is the SURFACE, and the exclusion runs both ways. Passing only
	// the org scope here is what stops an agent key — which has the same shape —
	// being used against the rest of the tenant's API by a hypervisor that was
	// compromised.
	auth, err := keys.Authenticate(ctx, c.orgKey, c.orgSecret, control.APIKeyScopeOrg)
	if err != nil || auth == nil || auth.Key == nil {
		return nil, service.ErrAPIKeyRejected
	}
	if auth.Key.OrgID == nil {
		// Mint refuses this shape; reaching it means the row was written by
		// something else, and a credential that cannot name its tenant must not act.
		return nil, service.ErrAPIKeyRejected
	}
	return &Principal{
		Method:         AuthMethodOrgSecret,
		OrganizationID: *auth.Key.OrgID,
		APIKeyID:       &auth.Key.ID,
	}, nil
}

func resolveUserKey(
	ctx context.Context,
	keys *service.APIKeyService,
	memberships repo.MembershipRepo,
	c presentedCredential,
) (*Principal, error) {
	if keys == nil {
		return nil, errors.New("api keys are not available")
	}
	// A user key carries no secret — RequiresSecret() is false for this scope —
	// so the key itself is the whole credential.
	auth, err := keys.Authenticate(ctx, c.userKey, "", control.APIKeyScopeUser)
	if err != nil || auth == nil || auth.Key == nil || auth.Key.UserID == nil {
		return nil, service.ErrAPIKeyRejected
	}

	p := &Principal{
		Method:   AuthMethodUserKey,
		UserID:   auth.Key.UserID,
		APIKeyID: &auth.Key.ID,
	}

	// X-Org-Id is OPTIONAL here, and that is deliberate: /me and the rest of a
	// person's own account do not act on a tenant, so requiring it would make
	// them unreachable with a user key. Endpoints that DO act on one call
	// AssertOrgRole, which refuses a principal with no organization and says so
	// specifically.
	raw := strings.TrimSpace(headerFromContext(ctx))
	if raw == "" {
		return p, nil
	}
	orgID, err := uuid.Parse(raw)
	if err != nil {
		return nil, errors.New("X-Org-Id is not a uuid")
	}

	// Membership is checked HERE, before the organization reaches the principal.
	// Everything downstream treats Principal.OrganizationID as established fact,
	// so an unverified header value written into it would be an IDOR into every
	// tenant this person is not a member of.
	if memberships == nil {
		return nil, errors.New("membership repository is not wired")
	}
	m, err := memberships.GetByUserAndOrg(ctx, *auth.Key.UserID, orgID)
	if err != nil || m == nil {
		// One answer for "not a member" and "lookup failed": neither may be used
		// to probe which organizations exist.
		return nil, ErrNoMembership
	}
	p.OrganizationID = orgID
	return p, nil
}

// orgIDCtxKey carries the raw X-Org-Id header from the middleware into
// resolveUserKey, which has a context rather than a huma.Context.
type orgIDCtxKey struct{}

func withOrgIDHeader(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, orgIDCtxKey{}, v)
}

func headerFromContext(ctx context.Context) string {
	v, _ := ctx.Value(orgIDCtxKey{}).(string)
	return v
}

// writeKeyAuthError answers a key-credential failure.
//
// One body for every rejection reason. Distinguishing "no such key" from "wrong
// secret" from "wrong surface" would let anything holding a routable connection
// enumerate which keys exist.
func writeKeyAuthError(api huma.API, hctx huma.Context, err error) {
	switch {
	case errors.Is(err, errAmbiguousCredential):
		// Safe to be specific: it describes the caller's own request and reveals
		// nothing about what exists on the server.
		_ = huma.WriteErr(api, hctx, http.StatusBadRequest,
			"send exactly one credential: a bearer token, a user API key, or an organization key and secret")
	default:
		_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid credential")
	}
}
