// Package accessrule exposes auth_access_rules, the second platform-level
// configuration system.
//
// It is not part of the policy merge and shares nothing with it: platform and
// org policies decide what an authenticated user must satisfy, these rows decide
// whether an identity gets that far at all. Evaluation is entirely table-driven
// — EvaluateAccessRules reads the table and nothing else — so until this surface
// existed the mechanism was invisible to every operator who had not read the
// rows by hand, and 32 seeded denies were enforcing a policy nothing displayed.
//
// This is the PLATFORM boundary. One row binds every tenant: a create_org deny
// on a domain stops that domain minting organizations anywhere on the
// installation. That is why it is registered from RegisterAdminProtected and
// declares SecurityUserOnly — an organization credential has no user behind it
// and can never satisfy the platform gate, so advertising it would document a
// path that always 401s.
package accessrule

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// AccessRuleView is one row, with the two facts about seeding that the row
// itself does not state.
type AccessRuleView struct {
	ID      string `json:"id" format:"uuid"`
	Kind    string `json:"kind" enum:"email,domain" doc:"An email rule matches one address; a domain rule matches every address at that domain. Email beats domain."`
	Value   string `json:"value" example:"gmail.com"`
	Purpose string `json:"purpose" enum:"authenticate,create_org,join_org" doc:"Which question this rule answers: authenticate gates signing in, create_org gates founding an organization, join_org gates accepting an invitation. A rule of one purpose can never decide another."`
	Policy  string `json:"policy" enum:"allow,deny"`
	Reason  string `json:"reason,omitempty"`

	// Seeded marks a row written by the boot-time public-domain pass rather than
	// by an operator. Derived from the reason marker in repo, not string-matched
	// here or in a UI: the wording is repo's business and may change.
	Seeded bool `json:"seeded" doc:"Written by the boot-time public-domain seed, not by an operator."`

	// SeedOverridden is the divergence this whole surface exists to show: a
	// domain the platform ships as a create_org deny that somebody has since
	// turned into an allow. It survives every restart, so nothing else will ever
	// mention it again.
	SeedOverridden bool `json:"seed_overridden" doc:"A seeded public-domain deny that an operator has turned into an allow. Permanent — the seed pass never updates an existing row."`

	// SeedRestoresOnRestart answers the question an operator actually has before
	// pressing delete, and it is a different question from Seeded: restoration
	// keys on (kind, value, purpose), so a row whose reason was edited is still
	// restored, and a row is not restored at all when seeding is switched off.
	SeedRestoresOnRestart bool `json:"seed_restores_on_restart" doc:"Deleting this rule does not last: the next boot re-inserts it as a create_org deny."`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListRequest struct{}

type ListResponse struct {
	Body struct {
		Rules []AccessRuleView `json:"rules"`
		// SeedingEnabled is configuration, not a property of any row, and it
		// changes what every seeded row means: with it off, a delete finally
		// sticks. Reported once rather than implied by the rows.
		SeedingEnabled bool `json:"seeding_enabled" doc:"Whether auth.seed_public_domain_rules re-applies the public-domain corpus at every boot."`
	}
}

type CreateRequest struct {
	Body struct {
		Kind    string `json:"kind" required:"true" enum:"email,domain"`
		Value   string `json:"value" required:"true" minLength:"1" maxLength:"255" doc:"An address for kind=email, a bare domain for kind=domain. Normalised on the way in — a value that does not normalise is refused rather than stored as a rule that can never match."`
		Purpose string `json:"purpose" required:"true" enum:"authenticate,create_org,join_org" doc:"Required, with no default. A rule stored under the wrong purpose is not a weaker rule, it is a rule that never fires on the question you meant to ask."`
		Policy  string `json:"policy" required:"true" enum:"allow,deny"`
		Reason  string `json:"reason,omitempty" maxLength:"1000" doc:"Free text for whoever reads this row in a year."`
	}
}

type CreateResponse struct {
	Body AccessRuleView
}

type UpdateRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Policy string `json:"policy" required:"true" enum:"allow,deny"`
		Reason string `json:"reason,omitempty" maxLength:"1000" doc:"Replaces the existing reason; send empty to clear it."`
	}
}

type UpdateResponse struct {
	Body AccessRuleView
}

type DeleteRequest struct {
	ID string `path:"id" format:"uuid"`
}

type DeleteResponse struct {
	Body struct {
		Deleted bool `json:"deleted"`
		// SeedRestoresOnRestart is why this endpoint answers with a body at all.
		SeedRestoresOnRestart bool   `json:"seed_restores_on_restart"`
		Warning               string `json:"warning,omitempty"`
	}
}

// seedWarning is the whole reason DELETE does not answer 204.
//
// Re-inserting on the next boot is correct and stays as it is: fail-safe is the
// right default for a table that decides who may mint tenants. What is not
// correct is doing it silently. An operator who deletes a seeded deny believes
// they have permitted the domain, and they have — until the next restart, which
// may be weeks away and will have no connection to this action.
const seedWarning = "This rule is part of the seeded public-domain corpus and WILL be re-created as a create_org deny the next time the service starts, silently reversing this deletion. " +
	"To permit this domain durably, set the rule's policy to allow instead: the seed pass never updates an existing row, so an allow survives every boot."

func toView(rule control.AuthAccessRule, seeding bool) AccessRuleView {
	seeded := repo.IsSeededRule(rule)
	covered := repo.SeedCorpusCovers(rule)
	v := AccessRuleView{
		ID:      rule.ID.String(),
		Kind:    rule.Kind,
		Value:   rule.Value,
		Purpose: rule.Purpose,
		Policy:  rule.Policy,
		Seeded:  seeded,
		// Corpus membership is ORed in so that editing the reason on a seeded row
		// cannot hide the override. The corpus is what the platform ships; the
		// reason is only the marker the seeder happens to write.
		SeedOverridden:        (seeded || covered) && rule.Policy == control.AccessRulePolicyAllow,
		SeedRestoresOnRestart: covered && seeding,
		CreatedAt:             rule.CreatedAt,
		UpdatedAt:             rule.UpdatedAt,
	}
	if rule.Reason != nil {
		v.Reason = *rule.Reason
	}
	return v
}

// seedingEnabled reports whether the boot-time pass is switched on.
//
// A nil Config reads as enabled: that is the shipped default, and api.Build is
// exercised with partially-populated Deps. Warning about a restore that will not
// happen costs an operator a second look; staying silent about one that will is
// the failure this surface exists to prevent.
func seedingEnabled(deps *app.Deps) bool {
	if deps == nil || deps.Config == nil {
		return true
	}
	return deps.Config.AuthSeedPublicDomainRules
}

// mapErr gives every deliberate refusal its own status. None of these are server
// faults, and a policy decision reported as 500 puts an operator's own rules on
// their error dashboard while telling them nothing about what to fix.
func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		return huma.Error404NotFound("access rule not found")

	case errors.Is(err, repo.ErrInvalidRuleKind),
		errors.Is(err, repo.ErrInvalidRulePurpose),
		errors.Is(err, repo.ErrInvalidRulePolicy),
		errors.Is(err, repo.ErrInvalidRuleValue):
		// The message names what is wrong with the submitted value and discloses
		// nothing about any other row.
		return huma.Error422UnprocessableEntity(err.Error())

	case isUniqueViolation(err):
		// (kind, value, purpose) is unique, and that is deliberate: two rules on
		// the same question would make precedence depend on row order. Editing
		// the existing rule is the operation the caller wants.
		return huma.Error409Conflict("a rule already exists for this kind, value and purpose — change that rule instead of adding a second one")

	default:
		slog.ErrorContext(ctx, "access rule operation failed", "error", err)
		return huma.Error500InternalServerError("access rule operation failed")
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// RegisterAdminProtected mounts the access rule surface.
//
// It MUST be handed a group carrying RequireAdminAuth. No handler below
// re-checks the role: the group is the gate, so an endpoint added here is
// protected by existing rather than by its author remembering.
func RegisterAdminProtected(hapi huma.API, deps *app.Deps) {
	rules := repo.NewAuthAccessRuleAdminRepoGorm(deps.ControlDB)
	usersRepo := repo.NewUserRepoGorm(deps.ControlDB)
	users := &usersRepo

	// The principal is read on every mutation for the log line. These rows have
	// no updated_by column, so the log is the only record of who changed a
	// platform-wide access decision.
	actor := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, ok := apimiddleware.PrincipalFromContext(ctx)
		if !ok || p == nil || p.UserID == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		return p, nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-access-rules-list",
		Method:      http.MethodGet,
		Path:        "/platform/access-rules",
		Summary:     "List auth access rules",
		Description: "Every rule that gates authentication, organization creation and invitation acceptance, " +
			"across every tenant. Rows written by the boot-time public-domain seed are marked, as are the ones " +
			"an operator has overridden to allow — that divergence exists only in this table and nothing else reports it.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, _ *ListRequest) (*ListResponse, error) {
		list, err := rules.List(ctx)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		seeding := seedingEnabled(deps)

		out := &ListResponse{}
		out.Body.SeedingEnabled = seeding
		out.Body.Rules = make([]AccessRuleView, 0, len(list))
		for _, r := range list {
			out.Body.Rules = append(out.Body.Rules, toView(r, seeding))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-access-rules-create",
		Method:      http.MethodPost,
		Path:        "/platform/access-rules",
		Summary:     "Create an auth access rule",
		Description: "Purpose is required and has no default. Email rules beat domain rules, and among domain " +
			"rules the most specific wins; anything that is not exactly 'allow' denies.",
		Tags:          []string{"Platform"},
		Security:      apimiddleware.SecurityUserOnly,
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateRequest) (*CreateResponse, error) {
		p, err := actor(ctx)
		if err != nil {
			return nil, err
		}
		if !repo.AccessRuleKindIsValid(in.Body.Kind) {
			return nil, huma.Error422UnprocessableEntity(`kind must be "email" or "domain"`)
		}
		// Refused here rather than at the repo, and much earlier than at
		// evaluation: EvaluateAccessRules denies outright on a purpose it does
		// not recognise, so a rule stored under a typo would not merely fail to
		// fire, it would be a row an operator reads as active policy while the
		// question it names is decided by something else entirely.
		if !repo.AccessPurposeIsValid(repo.AccessPurpose(in.Body.Purpose)) {
			return nil, huma.Error422UnprocessableEntity(`purpose must be "authenticate", "create_org" or "join_org" — a rule under an unknown purpose matches nothing and silently enforces no policy`)
		}
		if in.Body.Policy != control.AccessRulePolicyAllow && in.Body.Policy != control.AccessRulePolicyDeny {
			return nil, huma.Error422UnprocessableEntity(`policy must be "allow" or "deny"`)
		}

		rule := &control.AuthAccessRule{
			Kind:    in.Body.Kind,
			Value:   in.Body.Value,
			Purpose: in.Body.Purpose,
			Policy:  in.Body.Policy,
			Reason:  reasonPtr(in.Body.Reason),
		}
		if err := assertDoesNotLockOutActor(ctx, users, p,
			in.Body.Kind, in.Body.Value, in.Body.Purpose, in.Body.Policy); err != nil {
			return nil, lockoutErr()
		}

		if err := rules.Create(ctx, rule); err != nil {
			return nil, mapErr(ctx, err)
		}
		slog.InfoContext(ctx, "auth access rule created",
			"actor_user_id", p.UserID, "rule_id", rule.ID,
			"kind", rule.Kind, "value", rule.Value, "purpose", rule.Purpose, "policy", rule.Policy)

		return &CreateResponse{Body: toView(*rule, seedingEnabled(deps))}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-access-rules-update",
		Method:      http.MethodPut,
		Path:        "/platform/access-rules/{id}",
		Summary:     "Change a rule's policy",
		Description: "Policy and reason only. Kind, value and purpose are the rule's identity — retargeting one " +
			"is a delete and a create, so that both facts stay visible. This is also the durable way to permit a " +
			"seeded public domain: an update survives every restart, whereas a delete is undone by the next one.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *UpdateRequest) (*UpdateResponse, error) {
		p, err := actor(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid access rule id")
		}
		if in.Body.Policy != control.AccessRulePolicyAllow && in.Body.Policy != control.AccessRulePolicyDeny {
			return nil, huma.Error422UnprocessableEntity(`policy must be "allow" or "deny"`)
		}

		// Read before writing: UpdatePolicy changes only policy and reason, so the
		// kind, value and purpose that decide whether this locks the actor out are
		// on the stored row rather than in the request.
		existing, err := rules.Get(ctx, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		if err := assertDoesNotLockOutActor(ctx, users, p,
			existing.Kind, existing.Value, existing.Purpose, in.Body.Policy); err != nil {
			return nil, lockoutErr()
		}

		rule, err := rules.UpdatePolicy(ctx, id, in.Body.Policy, reasonPtr(in.Body.Reason))
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		slog.InfoContext(ctx, "auth access rule policy changed",
			"actor_user_id", p.UserID, "rule_id", rule.ID,
			"kind", rule.Kind, "value", rule.Value, "purpose", rule.Purpose, "policy", rule.Policy)

		return &UpdateResponse{Body: toView(*rule, seedingEnabled(deps))}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "platform-access-rules-delete",
		Method:      http.MethodDelete,
		Path:        "/platform/access-rules/{id}",
		Summary:     "Delete an auth access rule",
		Description: "Answers with a body rather than 204 because deleting a SEEDED rule does not last: the " +
			"next boot re-creates it as a create_org deny. The response says so explicitly. Re-creating is " +
			"fail-safe and stays that way — being silent about it was the problem.",
		Tags:     []string{"Platform"},
		Security: apimiddleware.SecurityUserOnly,
	}, func(ctx context.Context, in *DeleteRequest) (*DeleteResponse, error) {
		p, err := actor(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid access rule id")
		}

		// Read before deleting: once the row is gone there is no way to tell the
		// caller what they just removed, and that answer is the point of this
		// endpoint. Not found here means not found, which is a 404 either way.
		rule, err := rules.Get(ctx, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		restores := repo.SeedCorpusCovers(*rule) && seedingEnabled(deps)

		if err := rules.Delete(ctx, id); err != nil {
			return nil, mapErr(ctx, err)
		}
		slog.InfoContext(ctx, "auth access rule deleted",
			"actor_user_id", p.UserID, "rule_id", rule.ID,
			"kind", rule.Kind, "value", rule.Value, "purpose", rule.Purpose,
			"seeded", repo.IsSeededRule(*rule), "seed_restores_on_restart", restores)

		out := &DeleteResponse{}
		out.Body.Deleted = true
		out.Body.SeedRestoresOnRestart = restores
		if restores {
			out.Body.Warning = seedWarning
		}
		return out, nil
	})
}

// reasonPtr maps the wire's empty string to a NULL reason. The column is
// nullable and the difference is worth keeping: an empty string reads as a
// reason somebody wrote and left blank.
func reasonPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// ErrWouldLockOutActor refuses a rule that would deny the person writing it.
var ErrWouldLockOutActor = errors.New("access rule: this rule would deny your own sign-in")

// assertDoesNotLockOutActor refuses an authenticate DENY that matches the
// caller's own address.
//
// This is the one refusal on this surface that is not about policy correctness.
// CheckAuthentication runs on session ISSUE, and session issue includes refresh
// (internal/service/token_session_service.go) — so a platform admin who denies
// their own domain cannot log in, cannot refresh the session they are holding,
// and cannot undo it, because undoing it requires being signed in as a platform
// admin. One request, total and unrecoverable, with the recovery path inside the
// blast radius.
//
// Only 'authenticate' and only 'deny'. A create_org or join_org deny is
// survivable — it stops the actor founding or joining an organization, not
// signing in — and refusing those too would block the legitimate act of an
// operator closing their own public-mailbox domain to org creation.
//
// This does NOT stop an admin locking out someone ELSE, or writing a rule that
// locks out every other admin. Guarding against that would mean deciding who is
// allowed to be shut out, which is precisely the decision this surface exists to
// let an operator make. It guards the case where the person who could undo it is
// the person being shut out.
func assertDoesNotLockOutActor(
	ctx context.Context,
	users repo.UserRepo,
	p *apimiddleware.Principal,
	kind, value, purpose, policy string,
) error {
	if purpose != control.AccessRulePurposeAuthenticate || policy != control.AccessRulePolicyDeny {
		return nil
	}
	if users == nil || p == nil || p.UserID == nil {
		// A check that cannot reach a decision must deny — and here denying costs
		// an operator one refused rule, while admitting costs them the platform.
		return ErrWouldLockOutActor
	}
	actor, err := users.GetByID(ctx, *p.UserID)
	if err != nil || actor == nil {
		return ErrWouldLockOutActor
	}

	id, err := repo.NormalizeEmailIdentity(actor.Email)
	if err != nil {
		// The actor's own address cannot be keyed, so whether the rule matches it
		// is unknowable. Refuse rather than guess.
		return ErrWouldLockOutActor
	}

	switch kind {
	case control.AccessRuleKindEmail:
		if strings.EqualFold(strings.TrimSpace(value), id.Email) {
			return ErrWouldLockOutActor
		}
	case control.AccessRuleKindDomain:
		// DomainCandidates is what the evaluator matches against — the literal
		// domain and the eTLD+1 — so comparing against it is comparing against the
		// same thing that would decide the login.
		for _, c := range id.DomainCandidates() {
			if strings.EqualFold(strings.TrimSpace(value), c) {
				return ErrWouldLockOutActor
			}
		}
	}
	return nil
}

// lockoutErr is the one refusal on this surface that explains itself fully.
//
// Naming the consequence rather than the rule: an operator who is told "invalid"
// will try again, and the second attempt is the one that succeeds against a
// weaker check somewhere else.
func lockoutErr() error {
	return huma.Error409Conflict(
		"This rule would deny your own sign-in. Authentication rules are checked when a session " +
			"is issued AND when one is refreshed, so applying it would end your current session and " +
			"leave nobody able to undo it — reversing it requires signing in as a platform admin. " +
			"Write it against a value that does not match your own address, or have another platform " +
			"admin apply it.")
}
