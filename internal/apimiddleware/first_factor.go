package apimiddleware

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

// FactorInventory answers one question: does this user already hold a verified
// authentication factor?
//
// Declared here, on the consumer side, so the rule below can be tested without
// standing up either repository.
type FactorInventory interface {
	HasVerifiedFactor(ctx context.Context, userID uuid.UUID) (bool, error)
}

type gormFactorInventory struct {
	totps    *repo.UserTOTPRepoGorm
	passkeys *repo.UserPasskeyRepoGorm
}

func (g gormFactorInventory) HasVerifiedFactor(ctx context.Context, userID uuid.UUID) (bool, error) {
	totp, err := g.totps.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if totp != nil && totp.Verified {
		return true, nil
	}
	keys, err := g.passkeys.ListByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(keys) > 0, nil
}

// RequireFirstFactorEnrolment guards the endpoints reachable from a pre-authz
// session, and it is what stops the split from becoming an MFA bypass.
//
// The danger is specific. Once enrolment is reachable without satisfying
// policy, someone holding only a stolen PASSWORD can register their own
// authenticator on an account that already has one — and then satisfy the
// policy with it. The victim's second factor never comes into it. That is
// strictly worse than the lockout the split fixes.
//
// So the relaxation is bounded to the case that motivated it: a session that
// has NOT satisfied policy may enrol only when the account holds no verified
// factor at all. Someone who already has one is not locked out — they have
// something to present, and presenting it yields a satisfied session that
// reaches these endpoints by the ordinary route.
//
// Cross-TYPE is the part worth stating, because it is the hole a per-endpoint
// check leaves. SetupTOTP already refuses a second TOTP, but nothing stopped a
// passkey being added to an account whose factor was a TOTP. The inventory is
// therefore about factors, not about the factor this endpoint happens to
// create.
//
// Layer it AFTER RequireAuthenticated: it reads the principal that establishes.
func RequireFirstFactorEnrolment(api huma.API, deps *app.Deps) func(huma.Context, func(huma.Context)) {
	var inv FactorInventory
	if deps != nil && deps.ControlDB != nil {
		inv = gormFactorInventory{
			totps:    repo.NewUserTOTPRepoGorm(deps.ControlDB),
			passkeys: repo.NewUserPasskeyRepoGorm(deps.ControlDB),
		}
	}
	return firstFactorGate(api, inv)
}

func firstFactorGate(api huma.API, inv FactorInventory) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		p, ok := PrincipalFromContext(hctx.Context())
		if !ok || p == nil || p.UserID == nil {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "not authenticated")
			return
		}

		// A satisfied session is the ordinary case: someone managing their own
		// factors, having already proved whatever policy asked for.
		if p.PolicySatisfied {
			next(hctx)
			return
		}

		if inv == nil {
			// A check that cannot reach a decision must deny, and denying here
			// costs a user with no factor their remediation path rather than
			// handing an attacker an enrolment they should not have.
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "enrolment is not available")
			return
		}

		has, err := inv.HasVerifiedFactor(hctx.Context(), *p.UserID)
		if err != nil {
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "enrolment is not available")
			return
		}
		if has {
			// Deliberately the SAME shape the policy gate uses. This caller must
			// present the factor they already hold; telling them anything more
			// specific would confirm to someone holding a stolen password that the
			// account is protected, and which way.
			_ = huma.WriteErr(api, hctx, http.StatusForbidden, "mfa_required")
			return
		}
		next(hctx)
	}
}

// AssertPolicySatisfied is for handlers that must not act on a pre-authz
// session even though their route is reachable from one.
//
// Nothing should need it if routes are mounted correctly; it exists so that a
// handler which cannot be split by route can still refuse.
func AssertPolicySatisfied(ctx context.Context) error {
	p, ok := PrincipalFromContext(ctx)
	if !ok || p == nil {
		return huma.Error401Unauthorized("not authenticated")
	}
	if !p.PolicySatisfied {
		return huma.Error403Forbidden("mfa_required")
	}
	return nil
}
