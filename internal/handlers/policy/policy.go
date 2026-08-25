// Package policy exposes platform and organization policy management.
//
// Two route groups, two audiences, one merge. The platform group is staff-only
// and sets the floor; the organization group is org-admin and can only ever
// tighten it. Both return the EFFECTIVE result alongside their own row, because
// the question an operator is actually asking is "what applies", and a screen
// that shows only the level you are editing cannot answer it.
package policy

import (
	"context"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/service"
)

// PolicyConfigView mirrors control.PolicyConfig on the wire. Named rather than
// inline so the generated SDKs get a type a UI can hold, instead of an anonymous
// schema.
type PolicyConfigView struct {
	MinLength          int  `json:"min_length,omitempty" minimum:"0" maximum:"256"`
	RequireUppercase   bool `json:"require_uppercase,omitempty"`
	RequireLowercase   bool `json:"require_lowercase,omitempty"`
	RequireDigit       bool `json:"require_digit,omitempty"`
	RequireSpecial     bool `json:"require_special,omitempty"`
	MaxLifetimeMinutes int  `json:"max_lifetime_minutes,omitempty" minimum:"0" maximum:"525600"`
}

func toConfig(v PolicyConfigView) control.PolicyConfig {
	return control.PolicyConfig{
		MinLength: v.MinLength, RequireUppercase: v.RequireUppercase,
		RequireLowercase: v.RequireLowercase, RequireDigit: v.RequireDigit,
		RequireSpecial: v.RequireSpecial, MaxLifetimeMinutes: v.MaxLifetimeMinutes,
	}
}

func fromConfig(c control.PolicyConfig) PolicyConfigView {
	return PolicyConfigView{
		MinLength: c.MinLength, RequireUppercase: c.RequireUppercase,
		RequireLowercase: c.RequireLowercase, RequireDigit: c.RequireDigit,
		RequireSpecial: c.RequireSpecial, MaxLifetimeMinutes: c.MaxLifetimeMinutes,
	}
}

// PolicyView is one policy kind, at one level, with the effective result.
//
// `state` is the three-state the whole design turns on. It is reported as a
// word rather than as a nullable boolean because "inherit" and "explicitly off"
// are different things that a bare `enabled: false` cannot distinguish — and
// confusing them is how an operator switches a policy off and finds it still
// applying.
type PolicyView struct {
	Kind  string `json:"kind" enum:"require_mfa,require_passkey,password_min_length,password_complexity,session_max_lifetime"`
	State string `json:"state" enum:"inherit,enabled,disabled" doc:"inherit means this level expresses no opinion. For an ORGANIZATION that hands the decision to the platform. For the PLATFORM there is no level above, so it means the policy is simply not enforced — effective_enabled can never differ from the platform's own setting, because the effective view is computed with no organization."`

	Config PolicyConfigView `json:"config"`

	// Effective is the merge of platform and organization. Restrictions merge
	// toward strictness, so a level that is disabled contributes nothing at all —
	// neither its switch nor its numbers.
	EffectiveEnabled bool             `json:"effective_enabled"`
	EffectiveConfig  PolicyConfigView `json:"effective_config"`
}

type ListResponse struct {
	Body struct {
		Policies []PolicyView `json:"policies"`
	}
}

type SetRequest struct {
	Kind string `path:"kind" enum:"require_mfa,require_passkey,password_min_length,password_complexity,session_max_lifetime"`
	Body struct {
		State  string           `json:"state" required:"true" enum:"enabled,disabled" doc:"To return this level to 'inherit', DELETE the policy instead."`
		Config PolicyConfigView `json:"config,omitempty"`
	}
}

type SetResponse struct {
	Body PolicyView
}

type DeleteRequest struct {
	Kind string `path:"kind" enum:"require_mfa,require_passkey,password_min_length,password_complexity,session_max_lifetime"`
}

type DeleteResponse struct {
	Body struct {
		State string `json:"state" doc:"Always 'inherit' — deleting a policy is how a level withdraws its opinion."`
	}
}

type ImpactRequest struct {
	Kind string `path:"kind" enum:"require_mfa,require_passkey"`
}

type ImpactResponse struct {
	Body struct {
		NonCompliantUsers int64  `json:"non_compliant_users"`
		Message           string `json:"message"`
	}
}

func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, service.ErrPolicyUnknownKind):
		return huma.Error422UnprocessableEntity(err.Error())
	default:
		slog.ErrorContext(ctx, "policy operation failed", "error", err)
		return huma.Error500InternalServerError("policy operation failed")
	}
}

// stateOf renders the three-state. Absent from storage is "inherit"; present is
// whichever way its switch is set.
func stateOf(present bool, enabled bool) string {
	switch {
	case !present:
		return "inherit"
	case enabled:
		return "enabled"
	default:
		return "disabled"
	}
}

var allKinds = []control.PolicyKind{
	control.PolicyKindRequireMFA,
	control.PolicyKindRequirePasskey,
	control.PolicyKindPasswordMinLength,
	control.PolicyKindPasswordComplexity,
	control.PolicyKindSessionMaxLifetime,
}
