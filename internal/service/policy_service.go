package service

import (
	"context"
	"errors"
	"fmt"
	"unicode"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var ErrPasswordPolicy = errors.New("password does not meet policy")

// ErrPolicyUnknownKind is a sentinel so handlers can answer 422 rather than
// 500 for a kind that does not exist. A refusal the caller caused is not a
// server fault, and reporting it as one puts an operator's typo on the 5xx
// dashboard.
var ErrPolicyUnknownKind = errors.New("policy: unknown policy kind")

type EffectiveResult struct {
	Kind    control.PolicyKind   `json:"kind"`
	Enabled bool                 `json:"enabled"`
	Config  control.PolicyConfig `json:"config"`
}

type PolicyViolation struct {
	Kind    control.PolicyKind `json:"kind"`
	Message string             `json:"message"`
}

type PolicyService struct {
	platform repo.PlatformPolicyRepo
	org      repo.OrgPolicyRepo
	users    repo.UserRepo
	passkeys repo.UserPasskeyRepo
}

func NewPolicyService(
	platform repo.PlatformPolicyRepo,
	org repo.OrgPolicyRepo,
	users repo.UserRepo,
	passkeys repo.UserPasskeyRepo,
) *PolicyService {
	return &PolicyService{
		platform: platform,
		org:      org,
		users:    users,
		passkeys: passkeys,
	}
}

// EffectivePolicy merges the platform and organization rows for one policy kind.
//
// A DISABLED row contributes nothing — neither its enabled bit nor its config.
// That second half was missing: the config was read from both rows
// unconditionally, so as long as ANY level had the kind switched on, a
// switched-off level's stale numbers went on tightening the result. A platform
// session_max_lifetime row with Enabled=false and MaxLifetimeMinutes=5 beat an
// enabled org row of 480, and every session in that organization expired after
// five minutes because of a policy an operator had explicitly turned off.
//
// "Disabled" is the identity element of this merge, and the zero values already
// behave that way for every operator here — max() for a minimum length,
// min-of-positives for a lifetime ceiling, OR for the complexity flags. So
// gating the READ is the entire fix; nothing downstream had to change.
//
// Note the merge can only ever TIGHTEN, which is why the defect was an
// availability and correctness problem rather than a privilege-escalation one.
// It is also why the restriction/permission split matters: restrictions merge
// with OR so an organization relying on its own strict setting does not lose it
// when the platform loosens.
func (s *PolicyService) EffectivePolicy(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) (*EffectiveResult, error) {
	pp, err := s.platform.GetByKind(ctx, kind)
	if err != nil {
		return nil, fmt.Errorf("get platform policy: %w", err)
	}

	var op *control.OrgPolicy
	if orgID != uuid.Nil {
		op, err = s.org.GetByOrgAndKind(ctx, orgID, kind)
		if err != nil {
			return nil, fmt.Errorf("get org policy: %w", err)
		}
	}

	if pp == nil && op == nil {
		return &EffectiveResult{Kind: kind, Enabled: false}, nil
	}

	result := &EffectiveResult{Kind: kind}

	switch kind {
	case control.PolicyKindRequireMFA, control.PolicyKindRequirePasskey:
		result.Enabled = (pp != nil && pp.Enabled) || (op != nil && op.Enabled)

	case control.PolicyKindPasswordMinLength:
		result.Enabled = (pp != nil && pp.Enabled) || (op != nil && op.Enabled)
		var pMin, oMin int
		if pp != nil && pp.Enabled {
			pMin = pp.Config.Data().MinLength
		}
		if op != nil && op.Enabled {
			oMin = op.Config.Data().MinLength
		}
		result.Config.MinLength = max(pMin, oMin)
		if result.Config.MinLength < 8 {
			result.Config.MinLength = 8
		}

	case control.PolicyKindPasswordComplexity:
		result.Enabled = (pp != nil && pp.Enabled) || (op != nil && op.Enabled)
		var pc, oc control.PolicyConfig
		if pp != nil && pp.Enabled {
			pc = pp.Config.Data()
		}
		if op != nil && op.Enabled {
			oc = op.Config.Data()
		}
		result.Config.RequireUppercase = pc.RequireUppercase || oc.RequireUppercase
		result.Config.RequireLowercase = pc.RequireLowercase || oc.RequireLowercase
		result.Config.RequireDigit = pc.RequireDigit || oc.RequireDigit
		result.Config.RequireSpecial = pc.RequireSpecial || oc.RequireSpecial

	case control.PolicyKindSessionMaxLifetime:
		result.Enabled = (pp != nil && pp.Enabled) || (op != nil && op.Enabled)
		var pMax, oMax int
		if pp != nil && pp.Enabled {
			pMax = pp.Config.Data().MaxLifetimeMinutes
		}
		if op != nil && op.Enabled {
			oMax = op.Config.Data().MaxLifetimeMinutes
		}
		switch {
		case pMax > 0 && oMax > 0:
			result.Config.MaxLifetimeMinutes = min(pMax, oMax)
		case pMax > 0:
			result.Config.MaxLifetimeMinutes = pMax
		default:
			result.Config.MaxLifetimeMinutes = oMax
		}
	}

	return result, nil
}

func (s *PolicyService) ValidatePassword(ctx context.Context, orgID uuid.UUID, password string) error {

	mlp, err := s.EffectivePolicy(ctx, orgID, control.PolicyKindPasswordMinLength)
	if err != nil {
		return fmt.Errorf("check password min length policy: %w", err)
	}
	if mlp != nil && mlp.Enabled && mlp.Config.MinLength > 0 {
		if len(password) < mlp.Config.MinLength {
			return fmt.Errorf("%w: password must be at least %d characters", ErrPasswordPolicy, mlp.Config.MinLength)
		}
	}

	cp, err := s.EffectivePolicy(ctx, orgID, control.PolicyKindPasswordComplexity)
	if err != nil {
		return fmt.Errorf("check password complexity policy: %w", err)
	}
	if cp != nil && cp.Enabled {
		if cp.Config.RequireUppercase && !containsUpper(password) {
			return fmt.Errorf("%w: password must contain an uppercase letter", ErrPasswordPolicy)
		}
		if cp.Config.RequireLowercase && !containsLower(password) {
			return fmt.Errorf("%w: password must contain a lowercase letter", ErrPasswordPolicy)
		}
		if cp.Config.RequireDigit && !containsDigit(password) {
			return fmt.Errorf("%w: password must contain a digit", ErrPasswordPolicy)
		}
		if cp.Config.RequireSpecial && !containsSpecial(password) {
			return fmt.Errorf("%w: password must contain a special character", ErrPasswordPolicy)
		}
	}

	return nil
}

func (s *PolicyService) CheckUserCompliance(ctx context.Context, userID, orgID uuid.UUID) ([]PolicyViolation, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	var violations []PolicyViolation

	mfaPolicy, err := s.EffectivePolicy(ctx, orgID, control.PolicyKindRequireMFA)
	if err != nil {
		return nil, err
	}
	if mfaPolicy != nil && mfaPolicy.Enabled && !user.MFAEnabled {
		violations = append(violations, PolicyViolation{
			Kind:    control.PolicyKindRequireMFA,
			Message: "MFA is required by policy but not enabled",
		})
	}

	pkPolicy, err := s.EffectivePolicy(ctx, orgID, control.PolicyKindRequirePasskey)
	if err != nil {
		return nil, err
	}
	if pkPolicy != nil && pkPolicy.Enabled {
		count, err := s.passkeys.CountByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("count passkeys: %w", err)
		}
		if count == 0 {
			violations = append(violations, PolicyViolation{
				Kind:    control.PolicyKindRequirePasskey,
				Message: "passkey is required by policy but none registered",
			})
		}
	}

	return violations, nil
}

func (s *PolicyService) PreEnableCheck(ctx context.Context, orgID *uuid.UUID, kind control.PolicyKind) (int64, error) {
	switch kind {
	case control.PolicyKindRequireMFA:
		return s.users.CountUsersWithoutMFA(ctx, orgID)
	case control.PolicyKindRequirePasskey:
		return s.users.CountUsersWithoutPasskey(ctx, orgID)
	default:
		return 0, nil
	}
}

func (s *PolicyService) SetPlatformPolicy(ctx context.Context, createdBy uuid.UUID, kind control.PolicyKind, enabled bool, config control.PolicyConfig) (*control.PlatformPolicy, error) {
	if !kind.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrPolicyUnknownKind, kind)
	}

	if enabled && kind.IsCheckable() {
		count, err := s.PreEnableCheck(ctx, nil, kind)
		if err != nil {
			return nil, fmt.Errorf("pre-enable check: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("cannot enable %s: %d users are non-compliant", kind, count)
		}
	}

	policy := &control.PlatformPolicy{
		Kind:      kind,
		Enabled:   enabled,
		CreatedBy: createdBy,
	}
	policy.Config = datatypes.NewJSONType(config)

	if err := s.platform.Upsert(ctx, policy); err != nil {
		return nil, fmt.Errorf("upsert platform policy: %w", err)
	}

	if enabled && kind.IsCheckable() {
		switch kind {
		case control.PolicyKindRequireMFA:
			_, _ = s.users.BulkRevokeTokensWithoutMFA(ctx, nil)
		case control.PolicyKindRequirePasskey:
			_, _ = s.users.BulkRevokeTokensWithoutPasskey(ctx, nil)
		}
	}

	return policy, nil
}

func (s *PolicyService) SetOrgPolicy(ctx context.Context, createdBy uuid.UUID, orgID uuid.UUID, kind control.PolicyKind, enabled bool, config control.PolicyConfig) (*control.OrgPolicy, error) {
	if !kind.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrPolicyUnknownKind, kind)
	}

	if enabled && kind.IsCheckable() {
		count, err := s.PreEnableCheck(ctx, &orgID, kind)
		if err != nil {
			return nil, fmt.Errorf("pre-enable check: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("cannot enable %s: %d users are non-compliant", kind, count)
		}
	}

	policy := &control.OrgPolicy{
		OrganizationID: orgID,
		Kind:           kind,
		Enabled:        enabled,
		CreatedBy:      createdBy,
	}
	policy.Config = datatypes.NewJSONType(config)

	if err := s.org.Upsert(ctx, policy); err != nil {
		return nil, fmt.Errorf("upsert org policy: %w", err)
	}

	if enabled && kind.IsCheckable() {
		switch kind {
		case control.PolicyKindRequireMFA:
			_, _ = s.users.BulkRevokeTokensWithoutMFA(ctx, &orgID)
		case control.PolicyKindRequirePasskey:
			_, _ = s.users.BulkRevokeTokensWithoutPasskey(ctx, &orgID)
		}
	}

	return policy, nil
}

func (s *PolicyService) ListPlatformPolicies(ctx context.Context) ([]control.PlatformPolicy, error) {
	return s.platform.List(ctx)
}

func (s *PolicyService) DeletePlatformPolicy(ctx context.Context, kind control.PolicyKind) error {
	return s.platform.Delete(ctx, kind)
}

func (s *PolicyService) ListOrgPolicies(ctx context.Context, orgID uuid.UUID) ([]control.OrgPolicy, error) {
	return s.org.ListByOrg(ctx, orgID)
}

func (s *PolicyService) DeleteOrgPolicy(ctx context.Context, orgID uuid.UUID, kind control.PolicyKind) error {
	return s.org.Delete(ctx, orgID, kind)
}

func (s *PolicyService) ListEffectivePolicies(ctx context.Context, orgID uuid.UUID) ([]EffectiveResult, error) {
	kinds := []control.PolicyKind{
		control.PolicyKindRequireMFA,
		control.PolicyKindRequirePasskey,
		control.PolicyKindPasswordMinLength,
		control.PolicyKindPasswordComplexity,
		control.PolicyKindSessionMaxLifetime,
	}

	var results []EffectiveResult
	for _, kind := range kinds {
		ep, err := s.EffectivePolicy(ctx, orgID, kind)
		if err != nil {
			return nil, err
		}
		if ep != nil {
			results = append(results, *ep)
		}
	}
	return results, nil
}

func containsUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func containsLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func containsSpecial(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
