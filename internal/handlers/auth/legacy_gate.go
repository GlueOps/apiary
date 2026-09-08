package auth

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
)

// refuseIfDisabled is the ENFORCEMENT half of a public flag.
//
// The sign-in page reads /flags/public and hides the password form when
// legacy_login is off. That is an affordance, not a control: the endpoint is
// still there, still documented in the OpenAPI spec, and still accepts a POST
// from anything that is not a browser. A flag that only hides a form has
// disabled nothing.
//
// So the handler asks too, and this is the half that actually decides.
//
// FAILS OPEN, DELIBERATELY, and this is the one place in this codebase where
// that is right. Everywhere else a check that cannot reach a decision denies —
// but the decision here is whether anybody may sign in AT ALL, and denying on a
// database blip locks every operator out of the system, including the ones who
// would fix it. The service already falls back to the compiled default, which
// for these two flags is on; reaching the error path means the store itself
// failed, and staying signed-in-able through that is the safer failure.
//
// It is logged at ERROR because "authentication was permitted without being able
// to check whether it should be" is exactly the sentence an incident review
// needs to find.
func refuseIfDisabled(ctx context.Context, deps *app.Deps, key, what string) error {
	if deps == nil || deps.Flags == nil {
		return nil
	}
	on, err := deps.Flags.PlatformEnabled(ctx, key)
	if err != nil {
		slog.ErrorContext(ctx, "could not read a feature flag gating authentication, so the "+
			"request was allowed through rather than locking everyone out",
			"flag", key, "error", err)
		return nil
	}
	if on {
		return nil
	}
	// 403 with a plain reason. There is nothing to protect here: the same fact is
	// served to anonymous callers by /flags/public, because the sign-in page needs
	// it. Being vague would only make an operator think their password was wrong.
	return huma.Error403Forbidden(what)
}

// The two gates, named so a reader of the handler sees which flag applies
// without going and looking it up.
func refuseIfLegacyLoginDisabled(ctx context.Context, deps *app.Deps) error {
	return refuseIfDisabled(ctx, deps, "legacy_login",
		"password sign-in is not enabled on this installation")
}

func refuseIfLegacyRegistrationDisabled(ctx context.Context, deps *app.Deps) error {
	return refuseIfDisabled(ctx, deps, "legacy_registration",
		"self-service registration is not enabled on this installation")
}

// Compile-time proof that the keys above are declared. A mistyped key would
// otherwise make the gate permanently open: Enabled returns an error for an
// unknown flag, refuseIfDisabled treats an error as fail-open, and nothing would
// ever fail.
var _ = func() struct{} {
	for _, k := range []string{"legacy_login", "legacy_registration"} {
		if _, ok := control.LookupFlag(k); !ok {
			panic("auth: gate names an undeclared feature flag: " + k)
		}
	}
	return struct{}{}
}()
