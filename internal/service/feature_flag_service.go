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
}

func NewFeatureFlagService(flags repo.FeatureFlagRepo) *FeatureFlagService {
	return &FeatureFlagService{flags: flags}
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

// SetPlatform stores the installation-wide value.
func (s *FeatureFlagService) SetPlatform(
	ctx context.Context, key string, value bool, by uuid.UUID,
) error {
	if s == nil || s.flags == nil {
		return errUnwiredFlags
	}
	if _, ok := control.LookupFlag(key); !ok {
		return fmt.Errorf("%w: %q", control.ErrUnknownFlag, key)
	}
	return s.flags.SetPlatform(ctx, key, value, by)
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
