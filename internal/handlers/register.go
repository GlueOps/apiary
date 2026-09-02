package handlers

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/handlers/accessrule"
	"github.com/glueops/apiary/internal/handlers/admin"
	"github.com/glueops/apiary/internal/handlers/apikey"
	"github.com/glueops/apiary/internal/handlers/auth"
	"github.com/glueops/apiary/internal/handlers/estate"
	"github.com/glueops/apiary/internal/handlers/featureflag"
	"github.com/glueops/apiary/internal/handlers/firewall"
	"github.com/glueops/apiary/internal/handlers/fleet"
	"github.com/glueops/apiary/internal/handlers/invitation"
	"github.com/glueops/apiary/internal/handlers/me"
	"github.com/glueops/apiary/internal/handlers/org"
	"github.com/glueops/apiary/internal/handlers/policy"
	"github.com/glueops/apiary/internal/handlers/sshkey"
)

func RegisterPublic(api huma.API, deps *app.Deps) error {
	auth.RegisterPublic(api, deps)
	// Public by design — see the invitation package comment.
	invitation.RegisterPublic(api, deps)
	// Enrolment only. A host being brought up has no credential yet — that is
	// what the enrolment token is for.
	//
	// Fallible now that the fleet lives in tenant databases: the service needs a
	// tenant resolver and a key minter, and mounting the surface without them
	// would leave endpoints that authenticate nobody. Refusing to start is the
	// only safe answer.
	return fleet.RegisterPublic(api, deps)
}

func RegisterUserProtected(api huma.API, deps *app.Deps) error {
	estate.RegisterUserProtected(api, deps)
	auth.RegisterUserProtected(api, deps)
	me.RegisterUserProtected(api, deps)
	invitation.RegisterOrgProtected(api, deps)
	admin.RegisterUserProtected(api, deps)
	org.RegisterUserProtected(api, deps)
	// Credential self-service. Membership-gated: the boundary is whose keys
	// these are, and the scope of a key is what decides which surface it can
	// then reach.
	apikey.RegisterUserProtected(api, deps)
	// A customer's own fleet: hives, agents and the work queued against them.
	// Membership-gated, because the boundary is whose hypervisors these are.
	if err := fleet.RegisterUserProtected(api, deps); err != nil {
		return err
	}
	if err := fleet.RegisterFleetOps(api, deps); err != nil {
		return err
	}
	policy.RegisterOrgProtected(api, deps)
	// Feature flags, READ ONLY. A tenant sees what applies to it and which level
	// decided; setting one is platform work. There is no write route on this
	// surface at all — not one that checks a role and refuses — because a route
	// that does not exist cannot be reached through a hole in a role check.
	featureflag.RegisterProtected(api, deps)
	// Rule authoring. Membership-gated for the same reason the fleet is: whose
	// hypervisors these rules protect is a different question from job title.
	firewall.RegisterUserProtected(api, deps)
	// The credential the control plane reaches hypervisors with. Membership-gated:
	// whose hosts it opens is a tenant question.
	sshkey.RegisterUserProtected(api, deps)
	return nil
}

// RegisterAdminProtected mounts the PLATFORM boundary: things that bind every
// tenant, and that no customer and no support engineer may touch.
//
// It must be handed a group carrying RequireAdminAuth. The group is the gate —
// nothing here re-checks the role, so an endpoint added to this function is
// protected by existing.
//
// Note what is NOT here: a customer's fleet. Hives, bees and fleet tasks belong
// to an organization and are reached through membership, because whose
// hypervisor it is is a different question from what job title someone holds.
func RegisterAdminProtected(api huma.API, deps *app.Deps) {
	policy.RegisterAdminProtected(api, deps)
	// The OTHER platform-level configuration system. Access rules gate who may
	// authenticate, found an organization or accept an invitation, for every
	// tenant at once — a boundary no customer admin may reach.
	accessrule.RegisterAdminProtected(api, deps)
	// The THIRD platform-level configuration system, and deliberately not part of
	// the first. A policy merges with OR and can only tighten as it descends; a
	// flag's whole purpose is that the narrower level overrides, in either
	// direction. Same two-table shape, opposite resolution rule.
	featureflag.RegisterAdminProtected(api, deps)
	// The tenant directory. Platform-only: the names of every organization on
	// the installation are not any one customer's to see. It sits with the flags
	// because that is what needed it — setting a flag for one tenant otherwise
	// means pasting a uuid nobody has a way to look up.
	featureflag.RegisterOrganizationDirectory(api, deps)
}

// RegisterPreAuthz mounts the endpoints that must be reachable AFTER
// authentication but BEFORE policy is satisfied.
//
// The split undoes a conflation: a successful authentication traditionally
// produces authority, but policy sits between the two. Someone who has proved
// who they are and has not yet enrolled a factor is exactly the caller that
// needs enrolment, and exactly the caller the policy gate refuses.
//
// It must be handed a group carrying RequireAuthenticated followed by
// RequireFirstFactorEnrolment. The second is not optional: without it this
// group is reachable by anyone holding a password, on any account.
func RegisterPreAuthz(api huma.API, deps *app.Deps) {
	auth.RegisterPreAuthz(api, deps)
}

// RegisterBeeProtected mounts the fleet agent loop. It is separate from
// RegisterUserProtected because it runs behind a DIFFERENT middleware: a bee
// holds an agent-scoped key/secret pair, not a user JWT, and the two identities
// must never be interchangeable.
func RegisterBeeProtected(api huma.API, deps *app.Deps) error {
	return fleet.RegisterBeeProtected(api, deps)
}
