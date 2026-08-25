// Package apikey is the management surface for API credentials: listing them,
// minting them, and revoking them.
//
// It is deliberately NOT the surface that issues agent credentials. Scope names
// which surface a key may reach (see control.APIKeyScope*), and an agent key is
// standing access to the fleet loop of one tenant — it is minted only by
// enrolment, against a token an operator pasted onto the host being brought up.
// Letting a member mint one here would turn "I can call the API" into "I can
// present myself as any hypervisor in this organization".
package apikey

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

// maxTTLHours caps a requested lifetime at roughly a year. A key with no expiry
// is still allowed (ttl_hours omitted): automation that has to keep working
// across a year is real. What is refused is an expiry so distant it is
// indistinguishable from none while looking like a control.
//
// The same number appears as maximum:"8760" on the field, because a struct tag
// cannot reference a constant. The clamp below is therefore the backstop for
// the day the two are edited apart — the wrong half to drop silently is the
// tag, and dropping it would mint century-long credentials with no complaint.
const maxTTLHours = 8760

// APIKeyView is the wire shape for a key that already exists.
//
// There is no field here that could hold key or secret material, and that is
// the whole design: only hashes are stored, so there is nothing to return even
// if a field existed. Prefix is the first 8 characters of the identifying half,
// kept in clear at mint time precisely so a key can be RECOGNISED in this list
// without being usable from it — it is to an API key what a fingerprint is to
// an estate credential.
type APIKeyView struct {
	ID         string `json:"id" format:"uuid"`
	Name       string `json:"name"`
	Scope      string `json:"scope" enum:"user,org" doc:"'user' is the caller's own key; 'org' is the organization's, visible to every member and revocable by admins. Agent keys are not listed here — they belong to the fleet surface and are managed through enrolment."`
	Prefix     string `json:"prefix,omitempty" doc:"First characters of the identifying half, so a key can be told apart in a list. Not a credential and not enough to derive one."`
	Revoked    bool   `json:"revoked"`
	CreatedAt  string `json:"created_at" format:"date-time"`
	ExpiresAt  string `json:"expires_at,omitempty" format:"date-time" doc:"Absent means the key does not expire."`
	LastUsedAt string `json:"last_used_at,omitempty" format:"date-time" doc:"Absent means the key has never authenticated a request. Best-effort: a failure to record a use never fails the request it belongs to."`
}

func rfc3339(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toView(k control.APIKey) APIKeyView {
	v := APIKeyView{
		ID:         k.ID.String(),
		Name:       k.Name,
		Scope:      k.Scope,
		Revoked:    k.Revoked,
		CreatedAt:  k.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:  rfc3339(k.ExpiresAt),
		LastUsedAt: rfc3339(k.LastUsedAt),
	}
	if k.Prefix != nil {
		v.Prefix = *k.Prefix
	}
	return v
}

type ListAPIKeysRequest struct{}

type ListAPIKeysResponse struct {
	Body struct {
		Keys []APIKeyView `json:"keys"`
	}
}

type MintAPIKeyRequest struct {
	Body struct {
		Name string `json:"name" required:"true" minLength:"1" maxLength:"100" doc:"What this key is for. It is the only thing distinguishing two keys in a list, so a name like 'test' costs someone an outage later."`

		// No enum on purpose. Declaring enum:"user,org" would have huma refuse
		// "agent" with a generic schema-validation message, and the operator
		// reading it would go looking for a typo. Agent keys ARE a real scope,
		// they are simply not mintable here, and saying so plainly is the
		// difference between fixing a deployment and filing a bug. Declaring
		// enum:"user,org,agent" is worse still — it documents a scope this
		// endpoint always refuses.
		Scope string `json:"scope" required:"true" doc:"'user' for a key that acts as you, in whichever organization the request names; 'org' for a key that acts as the organization itself. Agent keys are issued only by fleet enrolment and are refused here."`

		TTLHours int `json:"ttl_hours,omitempty" minimum:"1" maximum:"8760" doc:"Lifetime in hours. Omit for a key that does not expire."`
	}
}

// MintAPIKeyResponse is the ONLY response in this package that carries
// credential material, and it carries it exactly once. Only hashes are stored,
// so a key that is not captured here is replaced, never recovered — there is
// deliberately no endpoint that reissues or reveals one.
type MintAPIKeyResponse struct {
	Body struct {
		Credential APIKeyView `json:"credential"`

		Public string `json:"key" doc:"The identifying half. For a user-scoped key this IS the whole credential; for an org-scoped key it is the HTTP Basic username. Returned once."`
		Secret string `json:"secret,omitempty" doc:"The secret half of an org-scoped key, presented as the HTTP Basic password. Returned ONCE and stored only as a hash — no endpoint reissues it. Absent for user-scoped keys, which are a single opaque value."`
	}
}

type RevokeAPIKeyRequest struct {
	ID string `path:"id" format:"uuid"`
}

type RevokeAPIKeyResponse struct {
	Body struct {
		Revoked bool `json:"revoked"`
	}
}

// The deliberate refusals of this package, as error values rather than inline
// huma.Error calls: mapErr then owns every status in one place, so a refusal
// cannot acquire a second, different status at a second call site.
var (
	errAgentScopeRefused = errors.New("agent-scoped keys are issued only by fleet enrolment, " +
		"against a token pasted onto the host being brought up. Minting one here would let anyone " +
		"who can call this API present itself as a hypervisor in this organization")

	errUnknownScope = errors.New("scope must be \"user\" or \"org\"")

	errNoUserBehindCredential = errors.New("this request is authenticated by an organization credential, " +
		"which has no user behind it. A user-scoped key belongs to a person and can only be minted by that person")

	errOrgKeyCannotMintOrgKey = errors.New("an organization credential may not mint another organization credential. " +
		"A leaked key that can issue its own replacement survives being revoked, which makes revocation no longer a remedy. " +
		"Mint org keys as a signed-in admin")

	// errKeyNotYours is what a caller gets for a key that does not exist, a key
	// belonging to another user, and a key belonging to another tenant — all
	// three, identically. Distinguishing them would turn this endpoint into an
	// oracle over key ids: a caller could learn which ids are real and which
	// organization they sit in without holding anything.
	errKeyNotYours = errors.New("no such key")
)

// mapErr turns a refusal into the right status.
//
// Every case below is a decision this package made on purpose, so none of them
// may surface as 500. A policy decision reported as a server error tells the
// caller the system is broken, sends them to retry something that will never
// succeed, and puts an operator's own rules on their 5xx dashboard.
func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, errAgentScopeRefused), errors.Is(err, errOrgKeyCannotMintOrgKey):
		// 403 rather than 422: the request is well formed and understood. It is
		// the caller who is not permitted to make it.
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, errNoUserBehindCredential):
		return huma.Error403Forbidden(err.Error())

	case errors.Is(err, errUnknownScope):
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, errKeyNotYours), errors.Is(err, repo.ErrAPIKeyNotFound):
		return huma.Error404NotFound(errKeyNotYours.Error())

	default:
		slog.ErrorContext(ctx, "api key operation failed", "error", err)
		return huma.Error500InternalServerError("api key operation failed")
	}
}

// keySets is the caller's complete authority over keys, resolved once.
//
// Both halves are fetched by IDENTITY, never by the id in the request: mine
// comes from the principal's user, org from the principal's organization. A
// handler can therefore only ever act on a key that is already in one of these
// two sets, which makes "may I touch this key" a lookup rather than a check
// somebody has to remember to write. There is no code path here that resolves a
// key by its id first and asks about ownership afterwards, because that is the
// shape that grows a hole the day someone adds an early return.
type keySets struct {
	mine []control.APIKey
	org  []control.APIKey
}

func loadKeySets(ctx context.Context, keys repo.APIKeyRepo, p *apimiddleware.Principal) (*keySets, error) {
	out := &keySets{}

	if p.UserID != nil {
		mine, err := keys.ListForUser(ctx, *p.UserID)
		if err != nil {
			return nil, err
		}
		// Filtered by scope, not trusted from the column. ListForUser matches on
		// user_id alone, and a row that carried both a user and an org would
		// otherwise be presented as this person's personal key while actually
		// holding tenant-wide authority.
		for _, k := range mine {
			if k.Scope == control.APIKeyScopeUser {
				out.mine = append(out.mine, k)
			}
		}
	}

	if p.OrganizationID != uuid.Nil {
		org, err := keys.ListForOrg(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		// Agent keys share the org_id column and are deliberately dropped here.
		// They are the fleet's credentials, one per enrolled host, and they are
		// managed where they are issued — an operator revoking a bee's key from
		// a list of "organization keys" would be unaware they had just cut a
		// hypervisor out of the control plane.
		for _, k := range org {
			if k.Scope == control.APIKeyScopeOrg {
				out.org = append(out.org, k)
			}
		}
	}
	return out, nil
}

// find locates a key the caller may act on, and reports which set it came from.
func (s *keySets) find(id uuid.UUID) (control.APIKey, string, bool) {
	for _, k := range s.mine {
		if k.ID == id {
			return k, control.APIKeyScopeUser, true
		}
	}
	for _, k := range s.org {
		if k.ID == id {
			return k, control.APIKeyScopeOrg, true
		}
	}
	return control.APIKey{}, "", false
}

func Register(hapi huma.API, deps *app.Deps) {
	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)
	keys := repo.NewAPIKeyRepoGorm(deps.ControlDB)

	huma.Register(hapi, huma.Operation{
		OperationID: "api-keys-list",
		Method:      http.MethodGet,
		Path:        "/api-keys",
		Summary:     "List API keys",
		Description: "Your own user-scoped keys, plus the organization's org-scoped keys. " +
			"Never returns key or secret material: only hashes are stored, and a key that is lost is replaced rather than recovered. " +
			"Agent keys are not listed — they belong to the fleet surface.",
		Tags:     []string{"API Keys"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *ListAPIKeysRequest) (*ListAPIKeysResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		sets, err := loadKeySets(ctx, keys, p)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		// Deduped by id. The two queries are independent, so a row carrying both
		// a user and an org would otherwise be listed twice.
		seen := map[uuid.UUID]bool{}
		merged := make([]control.APIKey, 0, len(sets.mine)+len(sets.org))
		for _, k := range append(append([]control.APIKey{}, sets.mine...), sets.org...) {
			if seen[k.ID] {
				continue
			}
			seen[k.ID] = true
			merged = append(merged, k)
		}
		sort.SliceStable(merged, func(i, j int) bool {
			return merged[i].CreatedAt.After(merged[j].CreatedAt)
		})

		out := &ListAPIKeysResponse{}
		out.Body.Keys = make([]APIKeyView, 0, len(merged))
		for _, k := range merged {
			out.Body.Keys = append(out.Body.Keys, toView(k))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "api-keys-create",
		Method:      http.MethodPost,
		Path:        "/api-keys",
		Summary:     "Mint an API key",
		Description: "Returns the plaintext credential ONCE. Only its hash is stored, so a key that is not captured " +
			"from this response is replaced rather than recovered — no endpoint reissues or reveals one.\n\n" +
			"A user-scoped key needs nothing beyond membership: it acts as you and can do nothing you cannot. " +
			"An org-scoped key is standing access to the whole tenant, so it requires admin and refuses impersonated sessions — " +
			"an operator acting as a user must not be able to leave themselves a way back in. " +
			"Agent-scoped keys are refused outright; they are issued only by fleet enrolment.",
		Tags:     []string{"API Keys"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *MintAPIKeyRequest) (*MintAPIKeyResponse, error) {
		scope := strings.ToLower(strings.TrimSpace(in.Body.Scope))

		// Scope is checked BEFORE any role assertion so that the surface refusal
		// is what an operator sees, whatever their role. An admin asking for an
		// agent key would otherwise pass the role gate and get the same message
		// as everyone else only by luck of ordering.
		switch scope {
		case control.APIKeyScopeAgent:
			return nil, mapErr(ctx, errAgentScopeRefused)
		case control.APIKeyScopeUser, control.APIKeyScopeOrg:
		default:
			return nil, mapErr(ctx, errUnknownScope)
		}

		minRole := control.MemberRoleMember
		var opts []apimiddleware.AssertOrgRoleOption
		if scope == control.APIKeyScopeOrg {
			minRole = control.MemberRoleAdmin
			opts = append(opts, apimiddleware.RefuseImpersonation())
		}
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, minRole, opts...)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}

		if deps.APIKeys == nil {
			// Reachable: api.Build is exercised with partially-populated Deps.
			// Minting through a nil service would panic, and a panic is a 500 on
			// a path whose job is to hand out a credential.
			return nil, huma.NewError(http.StatusServiceUnavailable, "key minting is not available")
		}

		mint := service.MintInput{
			Scope: scope,
			Name:  strings.TrimSpace(in.Body.Name),
		}
		switch scope {
		case control.APIKeyScopeUser:
			if p.UserID == nil {
				return nil, mapErr(ctx, errNoUserBehindCredential)
			}
			uid := *p.UserID
			mint.UserID = &uid
			// Deliberately NOT bound to an organization. A user key names a
			// person; the organization arrives in X-Org-Id and is checked
			// against membership on every call, so the key follows the person's
			// access rather than freezing a copy of it.
		case control.APIKeyScopeOrg:
			// AssertOrgRole ranks an org credential as admin, so without this it
			// would pass the gate above and mint its own successor. That makes
			// revocation stop being a remedy: whoever holds a leaked key issues a
			// fresh one before the old is cut off.
			if p.Method == apimiddleware.AuthMethodOrgSecret {
				return nil, mapErr(ctx, errOrgKeyCannotMintOrgKey)
			}
			org := p.OrganizationID
			mint.OrgID = &org
		}

		if h := in.Body.TTLHours; h > 0 {
			if h > maxTTLHours {
				h = maxTTLHours
			}
			mint.TTL = time.Duration(h) * time.Hour
		}

		res, err := deps.APIKeys.Mint(ctx, mint)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &MintAPIKeyResponse{}
		out.Body.Credential = toView(*res.Key)
		out.Body.Public = res.Public
		out.Body.Secret = res.Secret
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "api-keys-revoke",
		Method:      http.MethodDelete,
		Path:        "/api-keys/{id}",
		Summary:     "Revoke an API key",
		Description: "Revocation is permanent and takes effect on the next request the key makes. " +
			"You may revoke your own user-scoped keys. Revoking one of the organization's org-scoped keys " +
			"requires admin and refuses impersonated sessions, for the same reason minting one does: it changes " +
			"the tenant's standing access, and that has to be attributable to a real admin. " +
			"Nobody may revoke another person's user key.",
		Tags:     []string{"API Keys"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *RevokeAPIKeyRequest) (*RevokeAPIKeyResponse, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid key id")
		}

		sets, err := loadKeySets(ctx, keys, p)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		// The key has to be in the caller's own sets to be found at all. Another
		// user's key and another tenant's key are not "denied" here — they were
		// never fetched.
		key, from, ok := sets.find(id)
		if !ok {
			return nil, mapErr(ctx, errKeyNotYours)
		}

		if from == control.APIKeyScopeOrg {
			// Re-asserted rather than folded into the gate above, because the
			// required role depends on WHICH key this turned out to be, and that
			// is not known until it has been found.
			if _, err := apimiddleware.AssertOrgRole(ctx, memberships,
				control.MemberRoleAdmin, apimiddleware.RefuseImpersonation()); err != nil {
				return nil, apimiddleware.OrgRoleError(err)
			}
		}

		if err := keys.Revoke(ctx, key.ID); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &RevokeAPIKeyResponse{}
		out.Body.Revoked = true
		return out, nil
	})
}

// RegisterUserProtected mounts this package on the membership-gated group.
func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	Register(hapi, deps)
}
