package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
)

// errUnwiredFlags is what a WRITE answers when storage is not wired.
//
// Reads fall back to the compiled default, because that is what the build does
// anyway. A write cannot do the same: reporting success for a change that was
// never stored is the failure this whole subsystem is meant to avoid.
var errUnwiredFlags = fmt.Errorf("feature flag storage is not wired")

// FeatureFlagService resolves flags and is the only place a write is validated
// against the registry.
type FeatureFlagService struct {
	flags repo.FeatureFlagRepo

	// alternatives answers whether any way to sign in survives, for the guard on
	// flags marked GuardsSignIn. Optional: nil means the guard cannot reach a
	// decision, and SetPlatform denies rather than proceeding — see the comment
	// there for why this one denies where the auth gate fails open.
	alternatives SignInAlternatives
}

func NewFeatureFlagService(flags repo.FeatureFlagRepo) *FeatureFlagService {
	return &FeatureFlagService{flags: flags}
}

// WithSignInAlternatives wires the lockout guard.
//
// Separate from the constructor so every existing caller and test keeps working
// unchanged, and so the guard's absence is visible at the wiring site rather
// than hidden in a nil argument.
func (s *FeatureFlagService) WithSignInAlternatives(a SignInAlternatives) *FeatureFlagService {
	s.alternatives = a
	return s
}

// Enabled is the one call the rest of the codebase should need.
//
// It takes the organization because almost every caller has one; pass uuid.Nil
// on the platform surface. An unregistered key returns the error rather than
// false: a caller asking about a flag that does not exist has a bug, and
// answering "off" would hide it behind behaviour that looks deliberate.
func (s *FeatureFlagService) Enabled(ctx context.Context, orgID uuid.UUID, key string) (bool, error) {
	eff, err := s.Effective(ctx, orgID, key)
	if err != nil {
		return false, err
	}
	return eff.Value, nil
}

// Effective resolves one flag and says which level decided it.
func (s *FeatureFlagService) Effective(
	ctx context.Context, orgID uuid.UUID, key string,
) (*control.EffectiveFlag, error) {
	f, ok := control.LookupFlag(key)
	if !ok {
		return nil, fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	// NIL RECEIVER READS AS "NOTHING IS OVERRIDDEN", not as a failure and not as
	// false. api.Build is exercised with partially-populated Deps, so this is
	// reachable; and for a flag, the safe answer when storage cannot be consulted
	// is the compiled default — which is what the build was written to do. Every
	// WRITE below refuses instead, because a write that silently did nothing
	// would report success for a change that never happened.
	if s == nil || s.flags == nil {
		eff := control.ResolveFlag(f, nil, nil)
		return &eff, nil
	}

	pf, err := s.flags.GetPlatform(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read the platform flag: %w", err)
	}

	var of *control.OrgFeatureFlag
	if f.Scope == control.ScopeTenant && orgID != uuid.Nil {
		of, err = s.flags.GetOrg(ctx, orgID, key)
		if err != nil {
			return nil, fmt.Errorf("read the organization flag: %w", err)
		}
	}

	eff := control.ResolveFlag(f, pf, of)
	return &eff, nil
}

// ListEffective resolves every declared flag for one organization.
//
// Two queries rather than one per flag: the registry is small and fully known,
// so the rows are fetched in bulk and matched in memory. A per-flag round trip
// would make a settings screen's cost grow with the registry.
func (s *FeatureFlagService) ListEffective(
	ctx context.Context, orgID uuid.UUID,
) ([]control.EffectiveFlag, error) {
	all := control.AllFlags()
	out := make([]control.EffectiveFlag, 0, len(all))

	if s == nil || s.flags == nil {
		for _, f := range all {
			out = append(out, control.ResolveFlag(f, nil, nil))
		}
		return out, nil
	}

	pRows, err := s.flags.ListPlatform(ctx)
	if err != nil {
		return nil, fmt.Errorf("list platform flags: %w", err)
	}
	platform := make(map[string]*control.PlatformFeatureFlag, len(pRows))
	for i := range pRows {
		platform[pRows[i].Key] = &pRows[i]
	}

	org := map[string]*control.OrgFeatureFlag{}
	if orgID != uuid.Nil {
		oRows, oerr := s.flags.ListOrg(ctx, orgID)
		if oerr != nil {
			return nil, fmt.Errorf("list organization flags: %w", oerr)
		}
		for i := range oRows {
			org[oRows[i].Key] = &oRows[i]
		}
	}

	for _, f := range all {
		out = append(out, control.ResolveFlag(f, platform[f.Key], org[f.Key]))
	}
	return out, nil
}

// ListPublicEffective resolves the flags an unauthenticated caller may read.
//
// NO ORGANIZATION IS CONSULTED, and not because the caller happens not to have
// one — because a public flag is platform-scoped by construction, so there is no
// organization row that could apply. Passing uuid.Nil would reach the same
// answer today; naming it here means a future tenant-scoped public flag fails to
// compile a decision rather than silently resolving to whatever the platform
// happens to say.
func (s *FeatureFlagService) ListPublicEffective(ctx context.Context) ([]control.EffectiveFlag, error) {
	pub := control.PublicFlags()
	out := make([]control.EffectiveFlag, 0, len(pub))

	if s == nil || s.flags == nil {
		for _, f := range pub {
			out = append(out, control.ResolveFlag(f, nil, nil))
		}
		return out, nil
	}

	rows, err := s.flags.ListPlatform(ctx)
	if err != nil {
		return nil, fmt.Errorf("list platform flags: %w", err)
	}
	platform := make(map[string]*control.PlatformFeatureFlag, len(rows))
	for i := range rows {
		platform[rows[i].Key] = &rows[i]
	}
	for _, f := range pub {
		out = append(out, control.ResolveFlag(f, platform[f.Key], nil))
	}
	return out, nil
}

// PlatformEnabled answers one platform-scoped flag with no organization in play.
//
// For the server's own gates — the sign-in handler asking whether it is allowed
// to run at all — where there is no tenant yet and inventing uuid.Nil at each
// call site invites somebody passing a real one and getting a per-tenant answer
// to an installation-wide question.
func (s *FeatureFlagService) PlatformEnabled(ctx context.Context, key string) (bool, error) {
	return s.Enabled(ctx, uuid.Nil, key)
}

// SetPlatform stores the installation-wide value.
//
// acknowledgeLockout is the operator saying, explicitly, that they intend to
// disable a way of signing in with nothing else configured. It is ignored for
// every flag that does not gate sign-in.
func (s *FeatureFlagService) SetPlatform(
	ctx context.Context, key string, value bool, by uuid.UUID, acknowledgeLockout bool,
) error {
	if s == nil || s.flags == nil {
		return errUnwiredFlags
	}
	f, ok := control.LookupFlag(key)
	if !ok {
		return fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	if err := s.checkLockout(ctx, f, value, by, acknowledgeLockout); err != nil {
		return err
	}
	return s.flags.SetPlatform(ctx, key, value, by)
}

// checkLockout refuses a write that would leave no way into the installation.
//
// ONLY ON THE DISABLING EDGE. Enabling a sign-in method cannot lock anybody out,
// and clearing a row restores the compiled default — which for legacy_login is
// on — so neither is checked. A guard that fired on those would be an obstacle
// with no failure behind it, and obstacles with no failure behind them are how
// operators learn to reach for the override reflexively.
//
// THIS ONE DENIES WHEN IT CANNOT DECIDE, which is the opposite of the gate on
// the login handler, and both are right. The login gate decides whether an
// individual request may proceed, and denying there on a store outage locks
// everyone out. This decides whether to make a change that is HARD TO UNDO by
// construction — undoing it requires signing in, which is the thing being
// removed. Refusing a configuration change costs an operator a retry; allowing
// one wrongly costs them the system.
func (s *FeatureFlagService) checkLockout(
	ctx context.Context, f control.Flag, value bool, actingUserID uuid.UUID, acknowledged bool,
) error {
	if !f.GuardsSignIn || value {
		return nil
	}
	if acknowledged {
		return nil
	}
	if s.alternatives == nil {
		return fmt.Errorf("%w: no check for other sign-in methods is wired, so this cannot be "+
			"confirmed safe", control.ErrWouldLockEveryoneOut)
	}
	ok, how, err := s.alternatives.Available(ctx, actingUserID)
	if err != nil {
		return fmt.Errorf("%w: whether another sign-in method exists could not be determined: %w",
			control.ErrWouldLockEveryoneOut, err)
	}
	if ok {
		_ = how // an alternative exists; nothing to refuse
		return nil
	}
	return fmt.Errorf("%w: no social provider is configured and you have no passkey, so "+
		"disabling %q would leave nobody able to sign in — including you, to undo it",
		control.ErrWouldLockEveryoneOut, f.Key)
}

// SetOrg stores one organization's override.
//
// REFUSED FOR A PLATFORM-SCOPED FLAG, with an error that says so rather than
// "unknown". Naming a real flag at the wrong level and mistyping one are
// different mistakes with different fixes, and collapsing them sends an
// operator to check a spelling that was right.
func (s *FeatureFlagService) SetOrg(
	ctx context.Context, orgID uuid.UUID, key string, value bool, by uuid.UUID,
) error {
	if s == nil || s.flags == nil {
		return errUnwiredFlags
	}
	f, ok := control.LookupFlag(key)
	if !ok {
		return fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	if f.Scope != control.ScopeTenant {
		return fmt.Errorf("%w: %q", control.ErrFlagNotTenantScoped, key)
	}
	return s.flags.SetOrg(ctx, orgID, key, value, by)
}

// ClearPlatform and ClearOrg drop a row so the level below decides again.
func (s *FeatureFlagService) ClearPlatform(ctx context.Context, key string) error {
	if s == nil || s.flags == nil {
		return errUnwiredFlags
	}
	if _, ok := control.LookupFlag(key); !ok {
		return fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	return s.flags.ClearPlatform(ctx, key)
}

func (s *FeatureFlagService) ClearOrg(ctx context.Context, orgID uuid.UUID, key string) error {
	if s == nil || s.flags == nil {
		return errUnwiredFlags
	}
	f, ok := control.LookupFlag(key)
	if !ok {
		return fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	if f.Scope != control.ScopeTenant {
		return fmt.Errorf("%w: %q", control.ErrFlagNotTenantScoped, key)
	}
	return s.flags.ClearOrg(ctx, orgID, key)
}

// OverridesFor lists the organizations that have set this flag themselves,
// which is the question a platform operator asks before changing the default.
func (s *FeatureFlagService) OverridesFor(
	ctx context.Context, key string,
) ([]repo.FlagOverride, error) {
	if s == nil || s.flags == nil {
		return nil, errUnwiredFlags
	}
	if _, ok := control.LookupFlag(key); !ok {
		return nil, fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	return s.flags.ListOrgOverridesFor(ctx, key)
}
