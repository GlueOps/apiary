package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// Fakes
//
// PolicyService's interesting behaviour is arithmetic on two rows: it reads a
// platform row and an org row and merges them. Nothing about that merge is
// exercised any better by going through Postgres, and a real database would
// hide the cases that matter most — a row that is absent versus a row that is
// present and switched off are two different inputs to the merge, and only one
// of them is easy to arrange in SQL.
//
// Each fake embeds the repo interface so that any method these tests do not
// deliberately implement panics on a nil interface rather than silently
// returning a zero value. A panic here means the service reached for data the
// test did not expect it to need, which is itself a finding.
// ---------------------------------------------------------------------------

type poltPlatformRepo struct {
	repo.PlatformPolicyRepo

	row       *control.PlatformPolicy
	getErr    error
	upsertErr error

	getCalls int
	upserted []*control.PlatformPolicy
}

func (f *poltPlatformRepo) GetByKind(_ context.Context, _ control.PolicyKind) (*control.PlatformPolicy, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.row, nil
}

func (f *poltPlatformRepo) Upsert(_ context.Context, p *control.PlatformPolicy) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = append(f.upserted, p)
	return nil
}

type poltOrgRepo struct {
	repo.OrgPolicyRepo

	row       *control.OrgPolicy
	getErr    error
	upsertErr error

	getCalls int
	upserted []*control.OrgPolicy
}

func (f *poltOrgRepo) GetByOrgAndKind(_ context.Context, _ uuid.UUID, _ control.PolicyKind) (*control.OrgPolicy, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.row, nil
}

func (f *poltOrgRepo) Upsert(_ context.Context, p *control.OrgPolicy) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = append(f.upserted, p)
	return nil
}

type poltUserRepo struct {
	repo.UserRepo

	user    *control.User
	userErr error

	withoutMFA        int64
	withoutMFAErr     error
	withoutPasskey    int64
	withoutPasskeyErr error

	// Recorded scopes. A nil element means "platform-wide"; a non-nil element
	// carries the organization the call was scoped to. The distinction is the
	// whole safety property of the org variants, so it is recorded rather than
	// just counted.
	countMFAScopes     []*uuid.UUID
	countPasskeyScopes []*uuid.UUID
	revokeMFAScopes    []*uuid.UUID
	revokePkScopes     []*uuid.UUID
}

func (f *poltUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*control.User, error) {
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.user, nil
}

func (f *poltUserRepo) CountUsersWithoutMFA(_ context.Context, orgID *uuid.UUID) (int64, error) {
	f.countMFAScopes = append(f.countMFAScopes, orgID)
	return f.withoutMFA, f.withoutMFAErr
}

func (f *poltUserRepo) CountUsersWithoutPasskey(_ context.Context, orgID *uuid.UUID) (int64, error) {
	f.countPasskeyScopes = append(f.countPasskeyScopes, orgID)
	return f.withoutPasskey, f.withoutPasskeyErr
}

func (f *poltUserRepo) BulkRevokeTokensWithoutMFA(_ context.Context, orgID *uuid.UUID) (int64, error) {
	f.revokeMFAScopes = append(f.revokeMFAScopes, orgID)
	return 0, nil
}

func (f *poltUserRepo) BulkRevokeTokensWithoutPasskey(_ context.Context, orgID *uuid.UUID) (int64, error) {
	f.revokePkScopes = append(f.revokePkScopes, orgID)
	return 0, nil
}

type poltPasskeyRepo struct {
	repo.UserPasskeyRepo

	count    int64
	countErr error

	countCalls int
	// The user each count was asked about. A compliance answer computed from
	// somebody else's passkeys is worse than no answer at all, so the subject
	// of the query is recorded rather than just counted.
	countedUsers []uuid.UUID
}

func (f *poltPasskeyRepo) CountByUserID(_ context.Context, userID uuid.UUID) (int64, error) {
	f.countCalls++
	f.countedUsers = append(f.countedUsers, userID)
	return f.count, f.countErr
}

// poltDeps bundles the four fakes so a test can reach the one it wants to
// interrogate after driving the real service.
type poltDeps struct {
	platform *poltPlatformRepo
	org      *poltOrgRepo
	users    *poltUserRepo
	passkeys *poltPasskeyRepo
}

func poltNewService(d *poltDeps) *PolicyService {
	return NewPolicyService(d.platform, d.org, d.users, d.passkeys)
}

// poltSide names the three states a single level of the hierarchy can be in.
// The author's design calls these inherit | allow | deny; in storage they are
// "no row", "row with Enabled=false" and "row with Enabled=true".
type poltSide int

const (
	poltAbsent poltSide = iota // inherit: this level expresses no opinion
	poltOff                    // this level explicitly does not impose the restriction
	poltOn                     // this level imposes the restriction
)

func (s poltSide) String() string {
	switch s {
	case poltAbsent:
		return "inherit"
	case poltOff:
		return "off"
	default:
		return "on"
	}
}

func poltPlatformRow(s poltSide, kind control.PolicyKind, cfg control.PolicyConfig) *control.PlatformPolicy {
	if s == poltAbsent {
		return nil
	}
	p := &control.PlatformPolicy{Kind: kind, Enabled: s == poltOn}
	p.Config = datatypes.NewJSONType(cfg)
	return p
}

func poltOrgRow(s poltSide, orgID uuid.UUID, kind control.PolicyKind, cfg control.PolicyConfig) *control.OrgPolicy {
	if s == poltAbsent {
		return nil
	}
	p := &control.OrgPolicy{OrganizationID: orgID, Kind: kind, Enabled: s == poltOn}
	p.Config = datatypes.NewJSONType(cfg)
	return p
}

// poltEffective drives the real EffectivePolicy with the two rows described.
func poltEffective(
	t *testing.T,
	kind control.PolicyKind,
	platform poltSide, platformCfg control.PolicyConfig,
	org poltSide, orgCfg control.PolicyConfig,
) *EffectiveResult {
	t.Helper()
	orgID := uuid.New()
	svc := poltNewService(&poltDeps{
		platform: &poltPlatformRepo{row: poltPlatformRow(platform, kind, platformCfg)},
		org:      &poltOrgRepo{row: poltOrgRow(org, orgID, kind, orgCfg)},
		users:    &poltUserRepo{},
		passkeys: &poltPasskeyRepo{},
	})
	got, err := svc.EffectivePolicy(t.Context(), orgID, kind)
	if err != nil {
		t.Fatalf("EffectivePolicy(%s): unexpected error: %v", kind, err)
	}
	if got == nil {
		t.Fatalf("EffectivePolicy(%s) returned nil with a nil error — every caller "+
			"dereferences the result after checking only err", kind)
	}
	return got
}

var poltAllKinds = []control.PolicyKind{
	control.PolicyKindRequireMFA,
	control.PolicyKindRequirePasskey,
	control.PolicyKindPasswordMinLength,
	control.PolicyKindPasswordComplexity,
	control.PolicyKindSessionMaxLifetime,
}

// ---------------------------------------------------------------------------
// EffectivePolicy: the enabled bit
// ---------------------------------------------------------------------------

// This table IS the specification of the merge for the on/off bit, and it is
// exhaustive over the nine (platform, org) state pairs, applied to every policy
// kind. The rule it encodes is that these are RESTRICTIONS, so they merge with
// OR: either level may tighten, neither may loosen the other.
//
// The row that earns the whole test is (on, off). An organization admin who
// switches their own require_mfa row off must not thereby escape a platform
// mandate — if that row ever flipped to "false", every tenant would hold a
// one-click opt-out of platform security policy. The mirror row (off, on) is
// just as load-bearing in the other direction: an organization that has chosen
// a strict setting keeps it when the platform loosens, which is the property
// the author states as "an organization does not lose its own strict setting".
func TestEffectivePolicyEnabledIsTheOrOfBothLevels(t *testing.T) {
	cases := []struct {
		platform poltSide
		org      poltSide
		want     bool
		reason   string
	}{
		{poltAbsent, poltAbsent, false, "inherit is the identity element: with no opinion anywhere, nothing is imposed"},
		{poltAbsent, poltOff, false, "org declines and platform inherits, so no restriction exists"},
		{poltAbsent, poltOn, true, "org may impose a restriction the platform never asked for"},
		{poltOff, poltAbsent, false, "platform declines and org inherits that decision"},
		{poltOff, poltOff, false, "both levels decline"},
		{poltOff, poltOn, true, "org keeps its own strict setting when the platform loosens"},
		{poltOn, poltAbsent, true, "platform mandate reaches an org that expresses no opinion"},
		{poltOn, poltOff, true, "an org switching its row off must NOT be an opt-out of a platform mandate"},
		{poltOn, poltOn, true, "both levels agree"},
	}

	for _, kind := range poltAllKinds {
		for _, tc := range cases {
			name := fmt.Sprintf("%s/platform=%s,org=%s", kind, tc.platform, tc.org)
			t.Run(name, func(t *testing.T) {
				got := poltEffective(t, kind, tc.platform, control.PolicyConfig{}, tc.org, control.PolicyConfig{})
				if got.Enabled != tc.want {
					t.Fatalf("effective %s with platform=%s org=%s: Enabled = %v, want %v — %s",
						kind, tc.platform, tc.org, got.Enabled, tc.want, tc.reason)
				}
				if got.Kind != kind {
					t.Fatalf("effective policy reported Kind = %q, want %q — ListEffectivePolicies "+
						"returns a flat slice and callers match on this field, so a wrong Kind "+
						"silently applies one policy's answer to another policy's question",
						got.Kind, kind)
				}
			})
		}
	}
}

// The org lookup is skipped entirely when the caller has no organization, and
// that is not an optimisation: RegistrationService validates the password of a
// user who has no org yet by calling ValidatePassword with uuid.Nil. If
// EffectivePolicy queried org policies for the nil UUID it would be asking the
// database for rows belonging to "organization 00000000-...", and any row that
// ever landed there — a bad migration, a test fixture, a bug elsewhere — would
// start dictating policy to every self-service signup on the platform.
func TestEffectivePolicyDoesNotConsultOrgPoliciesForTheNilOrg(t *testing.T) {
	orgRepo := &poltOrgRepo{
		row: poltOrgRow(poltOn, uuid.Nil, control.PolicyKindRequireMFA, control.PolicyConfig{}),
	}
	svc := poltNewService(&poltDeps{
		platform: &poltPlatformRepo{row: nil},
		org:      orgRepo,
		users:    &poltUserRepo{},
		passkeys: &poltPasskeyRepo{},
	})

	got, err := svc.EffectivePolicy(t.Context(), uuid.Nil, control.PolicyKindRequireMFA)
	if err != nil {
		t.Fatalf("EffectivePolicy for the nil org: %v", err)
	}
	if orgRepo.getCalls != 0 {
		t.Fatalf("org policy repo was queried %d times for the nil organization, want 0 — "+
			"a registration that belongs to no org must not inherit policy from a row "+
			"stored against the zero UUID", orgRepo.getCalls)
	}
	if got.Enabled {
		t.Fatal("a policy row stored against the nil organization was applied to a caller " +
			"with no organization; only platform policy may reach an org-less caller")
	}
}

// Both repository lookups are on the critical path of every login and every
// password change. Neither may be swallowed: a database that is down must not
// read as "no policy configured", because for require_mfa that is the
// difference between refusing a session and minting one.
func TestEffectivePolicySurfacesRepositoryFailuresInsteadOfDefaultingOpen(t *testing.T) {
	poltBoom := errors.New("connection refused")

	t.Run("platform lookup", func(t *testing.T) {
		svc := poltNewService(&poltDeps{
			platform: &poltPlatformRepo{getErr: poltBoom},
			org:      &poltOrgRepo{},
			users:    &poltUserRepo{},
			passkeys: &poltPasskeyRepo{},
		})
		got, err := svc.EffectivePolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA)
		if !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the platform repo failure — a policy read that "+
				"fails must not be reported as 'no policy'", err)
		}
		if got != nil {
			t.Fatalf("EffectivePolicy returned %+v alongside an error; callers that check err "+
				"first are fine, but a non-nil result here invites use of a half-built policy", got)
		}
	})

	t.Run("org lookup", func(t *testing.T) {
		svc := poltNewService(&poltDeps{
			platform: &poltPlatformRepo{},
			org:      &poltOrgRepo{getErr: poltBoom},
			users:    &poltUserRepo{},
			passkeys: &poltPasskeyRepo{},
		})
		got, err := svc.EffectivePolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA)
		if !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the org repo failure", err)
		}
		if got != nil {
			t.Fatalf("EffectivePolicy returned %+v alongside an error", got)
		}
	})
}

// An unknown kind still resolves rather than blowing up, but it resolves to
// "nothing is imposed". The switch in EffectivePolicy has no default arm, so a
// kind added to the model without a merge rule silently produces a zero config;
// pinning that here means the next person to add a kind sees this test's name
// rather than a mysteriously permissive policy in production.
func TestEffectivePolicyOfAnUnhandledKindImposesNothing(t *testing.T) {
	const kind = control.PolicyKind("ip_allowlist_not_yet_implemented")
	got := poltEffective(t, kind,
		poltOn, control.PolicyConfig{MinLength: 64, RequireUppercase: true, MaxLifetimeMinutes: 5},
		poltOn, control.PolicyConfig{MinLength: 64, RequireUppercase: true, MaxLifetimeMinutes: 5})

	if got.Enabled {
		t.Fatal("an unhandled policy kind reported Enabled=true; the merge switch has no arm " +
			"for it, so nothing would ever read its config and 'enabled' is a lie")
	}
	if got.Config != (control.PolicyConfig{}) {
		t.Fatalf("Config = %+v, want the zero value — no merge rule ran for this kind, so any "+
			"non-zero field would be an unmerged copy of one side", got.Config)
	}
}

// ---------------------------------------------------------------------------
// EffectivePolicy: password_min_length
// ---------------------------------------------------------------------------

// Minimum length is a restriction, so the merge takes the LARGER of the two —
// the stricter one wins, from whichever side it came. The two rows that carry
// the property are the asymmetric ones: platform 20 / org 12 must resolve to
// 20 (a tenant cannot shorten a platform minimum) and platform 12 / org 20 must
// resolve to 20 (a tenant that wants to be stricter than the platform may be).
//
// The floor is separate and unconditional: whatever the two rows say, an
// enabled minimum is never below 8. It exists so that a fat-fingered
// `{"min_length": 1}` cannot turn the password policy into a formality.
func TestEffectiveMinLengthTakesTheStricterSideAndNeverGoesBelowEight(t *testing.T) {
	const kind = control.PolicyKindPasswordMinLength

	cases := []struct {
		name     string
		platform poltSide
		pMin     int
		org      poltSide
		oMin     int
		want     int
		reason   string
	}{
		{"org may tighten past the platform", poltOn, 12, poltOn, 20, 20,
			"an org that wants a longer minimum than the platform gets it"},
		{"org may not loosen the platform", poltOn, 20, poltOn, 12, 20,
			"a tenant must not be able to shorten a platform-mandated minimum"},
		{"platform only", poltOn, 16, poltAbsent, 0, 16,
			"inherit is the identity element: an absent org row leaves the platform value alone"},
		{"org only", poltAbsent, 0, poltOn, 16, 16,
			"an org may set a minimum the platform never set"},
		{"equal on both sides", poltOn, 14, poltOn, 14, 14,
			"agreement changes nothing"},
		{"both below the floor", poltOn, 4, poltOn, 6, 8,
			"the 8 character floor overrides a configured value that is shorter"},
		{"floor applies when nothing is configured", poltOn, 0, poltAbsent, 0, 8,
			"enabling the policy with an empty config still means 8, not 0"},
		{"the larger value survives the floor", poltOn, 30, poltOn, 4, 30,
			"the floor raises, it does not clamp down"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := poltEffective(t, kind,
				tc.platform, control.PolicyConfig{MinLength: tc.pMin},
				tc.org, control.PolicyConfig{MinLength: tc.oMin})

			if !got.Enabled {
				t.Fatalf("Enabled = false with platform=%s org=%s; the config assertion below "+
					"would be meaningless", tc.platform, tc.org)
			}
			if got.Config.MinLength != tc.want {
				t.Fatalf("effective min length with platform(%s)=%d org(%s)=%d: got %d, want %d — %s",
					tc.platform, tc.pMin, tc.org, tc.oMin, got.Config.MinLength, tc.want, tc.reason)
			}
		})
	}
}

// With no rows at all the service short-circuits before the per-kind merge, so
// the 8 character floor is NOT applied and MinLength comes back 0. That is
// harmless today because every consumer gates on Enabled first, but it is an
// asymmetry worth pinning: a future caller that reads Config.MinLength to draw
// a signup form's hint would show "0" on a platform with no policy configured
// and "8" on one whose policy row exists but is switched off.
func TestEffectiveMinLengthIsZeroNotEightWhenNoPolicyRowExists(t *testing.T) {
	got := poltEffective(t, control.PolicyKindPasswordMinLength,
		poltAbsent, control.PolicyConfig{}, poltAbsent, control.PolicyConfig{})

	if got.Enabled {
		t.Fatal("no policy rows exist, yet the effective policy is enabled")
	}
	if got.Config.MinLength != 0 {
		t.Fatalf("MinLength = %d with no policy rows, want 0 — if this now returns the floor, "+
			"the short-circuit at the top of EffectivePolicy has changed and every caller that "+
			"reads Config without checking Enabled needs re-reading", got.Config.MinLength)
	}
}

// ---------------------------------------------------------------------------
// EffectivePolicy: password_complexity
// ---------------------------------------------------------------------------

// Each complexity flag is an independent restriction and merges with OR, so the
// effective policy is the UNION of the two levels' demands. The case that
// matters is the one where the two sides ask for different things: the platform
// demands an uppercase letter, the org demands a digit, and the result must
// demand both. If this were an override rather than a union, an org that only
// wanted to add a digit requirement would silently delete the platform's
// uppercase requirement just by saving its own row.
func TestEffectiveComplexityIsTheUnionOfBothLevelsDemands(t *testing.T) {
	const kind = control.PolicyKindPasswordComplexity

	cases := []struct {
		name   string
		pCfg   control.PolicyConfig
		oCfg   control.PolicyConfig
		want   control.PolicyConfig
		reason string
		pSide  poltSide
		oSide  poltSide
	}{
		{
			name:  "disjoint demands are unioned",
			pSide: poltOn, pCfg: control.PolicyConfig{RequireUppercase: true},
			oSide: poltOn, oCfg: control.PolicyConfig{RequireDigit: true},
			want:   control.PolicyConfig{RequireUppercase: true, RequireDigit: true},
			reason: "adding an org requirement must not drop a platform requirement",
		},
		{
			name:  "org cannot switch a platform flag off",
			pSide: poltOn, pCfg: control.PolicyConfig{RequireUppercase: true, RequireSpecial: true},
			oSide: poltOn, oCfg: control.PolicyConfig{},
			want:   control.PolicyConfig{RequireUppercase: true, RequireSpecial: true},
			reason: "false is 'no opinion', not 'not required' — the platform's flags survive",
		},
		{
			name:  "absent org row is the identity element",
			pSide: poltOn, pCfg: control.PolicyConfig{RequireLowercase: true, RequireDigit: true},
			oSide:  poltAbsent,
			want:   control.PolicyConfig{RequireLowercase: true, RequireDigit: true},
			reason: "inherit leaves the platform demands exactly as they were",
		},
		{
			name:  "absent platform row is the identity element",
			pSide: poltAbsent,
			oSide: poltOn, oCfg: control.PolicyConfig{RequireSpecial: true},
			want:   control.PolicyConfig{RequireSpecial: true},
			reason: "an org may demand complexity the platform never demanded",
		},
		{
			name:  "all four flags on both sides",
			pSide: poltOn, pCfg: control.PolicyConfig{RequireUppercase: true, RequireLowercase: true, RequireDigit: true, RequireSpecial: true},
			oSide: poltOn, oCfg: control.PolicyConfig{RequireUppercase: true, RequireLowercase: true, RequireDigit: true, RequireSpecial: true},
			want:   control.PolicyConfig{RequireUppercase: true, RequireLowercase: true, RequireDigit: true, RequireSpecial: true},
			reason: "agreement changes nothing",
		},
		{
			name:  "neither side demands anything",
			pSide: poltOn, pCfg: control.PolicyConfig{},
			oSide: poltOn, oCfg: control.PolicyConfig{},
			want:   control.PolicyConfig{},
			reason: "an enabled complexity policy with an empty config demands nothing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := poltEffective(t, kind, tc.pSide, tc.pCfg, tc.oSide, tc.oCfg)

			for _, f := range []struct {
				name string
				got  bool
				want bool
			}{
				{"RequireUppercase", got.Config.RequireUppercase, tc.want.RequireUppercase},
				{"RequireLowercase", got.Config.RequireLowercase, tc.want.RequireLowercase},
				{"RequireDigit", got.Config.RequireDigit, tc.want.RequireDigit},
				{"RequireSpecial", got.Config.RequireSpecial, tc.want.RequireSpecial},
			} {
				if f.got != f.want {
					t.Fatalf("effective %s = %v, want %v (platform=%+v, org=%+v) — %s",
						f.name, f.got, f.want, tc.pCfg, tc.oCfg, tc.reason)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EffectivePolicy: session_max_lifetime
// ---------------------------------------------------------------------------

// A maximum session lifetime is a restriction expressed as a ceiling, so the
// stricter side is the SMALLER number and the merge is min — the mirror image
// of min_length's max. Zero is not "expire immediately", it is "unset", so a
// zero on one side must not swallow a real ceiling on the other. Getting that
// backwards in either direction is a security bug: min-with-zero would log
// everyone out constantly, and max would let an org grant itself an eight hour
// session under a platform ceiling of one.
func TestEffectiveSessionLifetimeTakesTheSmallerConfiguredCeiling(t *testing.T) {
	const kind = control.PolicyKindSessionMaxLifetime

	cases := []struct {
		name     string
		platform poltSide
		pMax     int
		org      poltSide
		oMax     int
		want     int
		reason   string
	}{
		{"org tightens below the platform ceiling", poltOn, 480, poltOn, 60, 60,
			"an org may expire its own sessions sooner than the platform requires"},
		{"org cannot raise the platform ceiling", poltOn, 60, poltOn, 480, 60,
			"a tenant must not grant itself a longer session than the platform allows"},
		{"platform unset leaves the org ceiling", poltOn, 0, poltOn, 480, 480,
			"zero means unset, so it must not be treated as the strictest value"},
		{"org unset leaves the platform ceiling", poltOn, 480, poltOn, 0, 480,
			"zero on the org side must not erase the platform ceiling"},
		{"absent org row is the identity element", poltOn, 120, poltAbsent, 0, 120,
			"inherit leaves the platform ceiling alone"},
		{"absent platform row is the identity element", poltAbsent, 0, poltOn, 120, 120,
			"an org may set a ceiling the platform never set"},
		{"neither side configured", poltOn, 0, poltOn, 0, 0,
			"an enabled policy with no number configured imposes no ceiling"},
		{"equal on both sides", poltOn, 90, poltOn, 90, 90,
			"agreement changes nothing"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := poltEffective(t, kind,
				tc.platform, control.PolicyConfig{MaxLifetimeMinutes: tc.pMax},
				tc.org, control.PolicyConfig{MaxLifetimeMinutes: tc.oMax})

			if got.Config.MaxLifetimeMinutes != tc.want {
				t.Fatalf("effective session ceiling with platform(%s)=%d org(%s)=%d: got %d minutes, want %d — %s",
					tc.platform, tc.pMax, tc.org, tc.oMax, got.Config.MaxLifetimeMinutes, tc.want, tc.reason)
			}
		})
	}
}

// The "known divergence" test that used to sit here pinned the CURRENT-at-the-time
// behaviour: EffectivePolicy read Config from a row without checking whether that
// row was Enabled, so a switched-off policy went on tightening the result. That
// defect is fixed, and TestDisabledPolicyRowContributesNoConfig now pins the
// correct behaviour instead. The old test was removed rather than inverted
// because its whole subject was the divergence.

// ---------------------------------------------------------------------------
// ValidatePassword
// ---------------------------------------------------------------------------

// poltValidator builds a service whose effective password policy is exactly the
// platform rows described, with no org rows at all. Registration validates
// against the platform alone (orgID uuid.Nil), so this is the shape that guards
// the majority of real password checks.
func poltValidator(minLen poltSide, minCfg control.PolicyConfig, cx poltSide, cxCfg control.PolicyConfig) *PolicyService {
	rows := map[control.PolicyKind]*control.PlatformPolicy{
		control.PolicyKindPasswordMinLength:  poltPlatformRow(minLen, control.PolicyKindPasswordMinLength, minCfg),
		control.PolicyKindPasswordComplexity: poltPlatformRow(cx, control.PolicyKindPasswordComplexity, cxCfg),
	}
	return NewPolicyService(&poltKindedPlatformRepo{rows: rows}, &poltOrgRepo{}, &poltUserRepo{}, &poltPasskeyRepo{})
}

// poltKindedPlatformRepo answers per kind, which ValidatePassword needs because
// it consults two different policies in one call.
type poltKindedPlatformRepo struct {
	repo.PlatformPolicyRepo
	rows map[control.PolicyKind]*control.PlatformPolicy
	err  error
}

func (f *poltKindedPlatformRepo) GetByKind(_ context.Context, kind control.PolicyKind) (*control.PlatformPolicy, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows[kind], nil
}

// poltKindedOrgRepo is the org-side equivalent, and it records which
// organization each lookup named so that a caller passing the wrong org — or
// none at all — is visible rather than merely undetected.
type poltKindedOrgRepo struct {
	repo.OrgPolicyRepo
	rows      map[control.PolicyKind]*control.OrgPolicy
	askedOrgs []uuid.UUID
}

func (f *poltKindedOrgRepo) GetByOrgAndKind(_ context.Context, orgID uuid.UUID, kind control.PolicyKind) (*control.OrgPolicy, error) {
	f.askedOrgs = append(f.askedOrgs, orgID)
	return f.rows[kind], nil
}

// Every message in this table is shown to the end user verbatim: the password
// reset handler returns huma.Error422UnprocessableEntity(err.Error()). So the
// text is not decoration, it is the product — a message that says only
// "password does not meet policy" leaves someone guessing which of five rules
// they broke. Each case therefore asserts both the sentinel (so handlers keep
// mapping this to 422 rather than 500) and that the message names the rule.
func TestValidatePasswordNamesTheRuleThatFailed(t *testing.T) {
	allComplexity := control.PolicyConfig{
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}

	cases := []struct {
		name     string
		minSide  poltSide
		minCfg   control.PolicyConfig
		cxSide   poltSide
		cxCfg    control.PolicyConfig
		password string
		wantErr  bool
		wantFrag string
		reason   string
	}{
		{
			name: "too short", minSide: poltOn, minCfg: control.PolicyConfig{MinLength: 12},
			cxSide: poltAbsent, password: "short1!A", wantErr: true,
			wantFrag: "at least 12 characters",
			reason:   "the message must state the required length, not just that it failed",
		},
		{
			name: "exactly at the minimum is accepted", minSide: poltOn, minCfg: control.PolicyConfig{MinLength: 12},
			cxSide: poltAbsent, password: "abcdefghijkl", wantErr: false,
			reason: "the comparison is < not <=; a password of exactly MinLength must pass",
		},
		{
			name: "missing uppercase", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: allComplexity, password: "lower1!x", wantErr: true,
			wantFrag: "uppercase letter",
			reason:   "the user is told which class is missing",
		},
		{
			name: "missing lowercase", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: allComplexity, password: "UPPER1!X", wantErr: true,
			wantFrag: "lowercase letter",
			reason:   "an all-uppercase password must be told it needs a lowercase letter",
		},
		{
			name: "missing digit", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: allComplexity, password: "UPPERlower!", wantErr: true,
			wantFrag: "a digit",
			reason:   "the user is told a digit is missing rather than being left to guess",
		},
		{
			name: "missing special", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: allComplexity, password: "UPPERlower1", wantErr: true,
			wantFrag: "special character",
			reason:   "the special-character rule is enforced and named",
		},
		{
			name: "a space counts as a special character", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: control.PolicyConfig{RequireSpecial: true},
			password: "correct horse", wantErr: false,
			reason: "passphrases must not be rejected for lacking punctuation when they contain a space",
		},
		{
			name: "only the configured classes are demanded", minSide: poltAbsent,
			cxSide: poltOn, cxCfg: control.PolicyConfig{RequireDigit: true},
			password: "no-uppercase-needed-1", wantErr: false,
			reason: "a policy that asks only for a digit must not enforce the other three",
		},
		{
			name: "a satisfying password passes every rule", minSide: poltOn,
			minCfg: control.PolicyConfig{MinLength: 12}, cxSide: poltOn, cxCfg: allComplexity,
			password: "Sufficient1!x", wantErr: false,
			reason: "a password that meets every configured rule must not be rejected",
		},
		{
			name: "disabled policies enforce nothing", minSide: poltOff,
			minCfg: control.PolicyConfig{MinLength: 64}, cxSide: poltOff, cxCfg: allComplexity,
			password: "a", wantErr: false,
			reason: "a one character password is accepted when both password policies are switched off, " +
				"which proves the rules are gated on Enabled and not applied unconditionally",
		},
		{
			name: "no policy rows at all enforce nothing", minSide: poltAbsent, cxSide: poltAbsent,
			password: "a", wantErr: false,
			reason: "an unconfigured platform imposes no password rules",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := poltValidator(tc.minSide, tc.minCfg, tc.cxSide, tc.cxCfg)
			err := svc.ValidatePassword(t.Context(), uuid.Nil, tc.password)

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("ValidatePassword(%q) = %v, want nil — %s", tc.password, err, tc.reason)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidatePassword(%q) accepted a password that violates the policy — %s",
					tc.password, tc.reason)
			}
			if !errors.Is(err, ErrPasswordPolicy) {
				t.Fatalf("ValidatePassword(%q) = %v, which does not wrap ErrPasswordPolicy — the "+
					"password reset handler switches on that sentinel, so an unwrapped error "+
					"becomes a 500 and the user is told the server is broken instead of which "+
					"rule they broke", tc.password, err)
			}
			if !strings.Contains(err.Error(), tc.wantFrag) {
				t.Fatalf("ValidatePassword(%q) said %q, which does not mention %q — this string is "+
					"returned to the user verbatim in a 422, so it has to name the failed rule",
					tc.password, err.Error(), tc.wantFrag)
			}
		})
	}
}

// The eight character floor computed by EffectivePolicy is not decorative: it
// is what an operator gets when they enable password_min_length and leave the
// config empty. Without it, `{"enabled": true}` with no min_length would set
// the effective minimum to zero and the enabled policy would enforce nothing at
// all — the most dangerous possible reading of "I turned the password policy
// on".
func TestValidatePasswordAppliesTheEightCharacterFloorToAnEmptyConfig(t *testing.T) {
	svc := poltValidator(poltOn, control.PolicyConfig{}, poltAbsent, control.PolicyConfig{})

	if err := svc.ValidatePassword(t.Context(), uuid.Nil, "sevench"); err == nil {
		t.Fatal("a seven character password was accepted by an enabled min-length policy with " +
			"an empty config; enabling the policy must mean at least 8 characters, otherwise " +
			"turning it on enforces nothing")
	} else if !strings.Contains(err.Error(), "at least 8 characters") {
		t.Fatalf("message %q does not name the 8 character floor", err.Error())
	}

	if err := svc.ValidatePassword(t.Context(), uuid.Nil, "eightchr"); err != nil {
		t.Fatalf("an eight character password was rejected by the floor: %v", err)
	}
}

// When a password breaks several rules the user is told about the length first.
// That ordering is deliberate — length is the rule they can act on without
// reading a second message — and it is checked here so that a refactor which
// reorders the checks has to justify itself.
func TestValidatePasswordReportsLengthBeforeComplexity(t *testing.T) {
	svc := poltValidator(
		poltOn, control.PolicyConfig{MinLength: 16},
		poltOn, control.PolicyConfig{RequireUppercase: true, RequireDigit: true, RequireSpecial: true},
	)

	err := svc.ValidatePassword(t.Context(), uuid.Nil, "abc")
	if err == nil {
		t.Fatal("a three character all-lowercase password passed a 16 character policy with complexity")
	}
	if !strings.Contains(err.Error(), "at least 16 characters") {
		t.Fatalf("first reported failure was %q, want the length rule — a password that is both "+
			"too short and too simple should be told to get longer first", err.Error())
	}
}

// A failure reading policy must not read as "the password is fine". Both
// EffectivePolicy calls inside ValidatePassword are on the path of every
// registration and every password reset.
func TestValidatePasswordFailsClosedWhenPolicyCannotBeRead(t *testing.T) {
	poltBoom := errors.New("policy store unavailable")
	svc := NewPolicyService(
		&poltKindedPlatformRepo{err: poltBoom},
		&poltOrgRepo{}, &poltUserRepo{}, &poltPasskeyRepo{},
	)

	err := svc.ValidatePassword(t.Context(), uuid.Nil, "anything")
	if err == nil {
		t.Fatal("ValidatePassword accepted a password while the policy store was unreachable; " +
			"an unreadable policy must not be treated as an absent one")
	}
	if !errors.Is(err, poltBoom) {
		t.Fatalf("err = %v, want it to wrap the store failure", err)
	}
	if errors.Is(err, ErrPasswordPolicy) {
		t.Fatal("an infrastructure failure was reported as a password policy violation, which " +
			"the handler would turn into a 422 telling the user to pick a better password")
	}
}

// ---------------------------------------------------------------------------
// CheckUserCompliance
// ---------------------------------------------------------------------------

func poltComplianceService(mfaOn, passkeyOn bool, user *control.User, passkeys *poltPasskeyRepo) (*PolicyService, *poltUserRepo) {
	rows := map[control.PolicyKind]*control.PlatformPolicy{}
	if mfaOn {
		rows[control.PolicyKindRequireMFA] = poltPlatformRow(poltOn, control.PolicyKindRequireMFA, control.PolicyConfig{})
	}
	if passkeyOn {
		rows[control.PolicyKindRequirePasskey] = poltPlatformRow(poltOn, control.PolicyKindRequirePasskey, control.PolicyConfig{})
	}
	users := &poltUserRepo{user: user}
	return NewPolicyService(&poltKindedPlatformRepo{rows: rows}, &poltOrgRepo{}, users, passkeys), users
}

// CheckUserCompliance is what tells an operator which accounts are standing in
// the way of switching a mandate on. Each violation has to name the kind, or
// the operator is handed a list of users with no way to know what to ask them
// for.
func TestCheckUserComplianceReportsExactlyTheUnmetMandates(t *testing.T) {
	cases := []struct {
		name         string
		mfaPolicy    bool
		passkeyPolic bool
		userMFA      bool
		passkeyCount int64
		wantKinds    []control.PolicyKind
		reason       string
	}{
		{
			name: "no mandates, nothing to report", userMFA: false, passkeyCount: 0,
			wantKinds: nil,
			reason:    "a user without MFA is not non-compliant when nothing requires MFA",
		},
		{
			name: "MFA required and missing", mfaPolicy: true, userMFA: false,
			wantKinds: []control.PolicyKind{control.PolicyKindRequireMFA},
			reason:    "this is the row that blocks the mandate from being switched on",
		},
		{
			name: "MFA required and present", mfaPolicy: true, userMFA: true,
			wantKinds: nil,
			reason:    "a user who has already enrolled must not be reported as blocking",
		},
		{
			name: "passkey required and none registered", passkeyPolic: true, passkeyCount: 0,
			wantKinds: []control.PolicyKind{control.PolicyKindRequirePasskey},
		},
		{
			name: "passkey required and one registered", passkeyPolic: true, passkeyCount: 1,
			wantKinds: nil,
			reason:    "one passkey is enough; the check is count == 0, not count < 2",
		},
		{
			name: "both mandates unmet", mfaPolicy: true, passkeyPolic: true,
			userMFA: false, passkeyCount: 0,
			wantKinds: []control.PolicyKind{control.PolicyKindRequireMFA, control.PolicyKindRequirePasskey},
			reason:    "an operator needs both gaps in one pass, not one per round trip",
		},
		{
			name: "both mandates met", mfaPolicy: true, passkeyPolic: true,
			userMFA: true, passkeyCount: 2,
			wantKinds: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user := &control.User{ID: uuid.New(), MFAEnabled: tc.userMFA}
			pk := &poltPasskeyRepo{count: tc.passkeyCount}
			svc, _ := poltComplianceService(tc.mfaPolicy, tc.passkeyPolic, user, pk)

			got, err := svc.CheckUserCompliance(t.Context(), user.ID, uuid.Nil)
			if err != nil {
				t.Fatalf("CheckUserCompliance: %v", err)
			}

			for _, asked := range pk.countedUsers {
				if asked != user.ID {
					t.Fatalf("passkeys were counted for user %v while checking the compliance of "+
						"user %v — the verdict would be computed from a different account's "+
						"enrolments", asked, user.ID)
				}
			}

			var gotKinds []control.PolicyKind
			for _, v := range got {
				gotKinds = append(gotKinds, v.Kind)
				if strings.TrimSpace(v.Message) == "" {
					t.Fatalf("violation for %s carried an empty message; the operator UI shows "+
						"this string and would render a blank row", v.Kind)
				}
			}
			if len(gotKinds) != len(tc.wantKinds) {
				t.Fatalf("violations = %v, want %v — %s", gotKinds, tc.wantKinds, tc.reason)
			}
			for i := range gotKinds {
				if gotKinds[i] != tc.wantKinds[i] {
					t.Fatalf("violation %d was %s, want %s — %s", i, gotKinds[i], tc.wantKinds[i], tc.reason)
				}
			}
		})
	}
}

// Every case above resolves policy from the platform rows alone, with the org
// argument left as uuid.Nil. That leaves the orgID parameter itself unproven:
// CheckUserCompliance could ignore it entirely and every one of those cases
// would still pass. It must not, because a mandate can exist ONLY at the org
// level — an org that switches require_mfa on has no platform row backing it,
// and a compliance sweep that resolved policy platform-wide would hand that
// org's operator a clean bill of health for members who have never enrolled,
// which is exactly the report that precedes locking them all out.
func TestCheckUserComplianceSeesMandatesThatExistOnlyAtTheOrgLevel(t *testing.T) {
	t.Run("require_mfa set by the org alone", func(t *testing.T) {
		orgID := uuid.New()
		user := &control.User{ID: uuid.New(), MFAEnabled: false}
		orgRepo := &poltKindedOrgRepo{rows: map[control.PolicyKind]*control.OrgPolicy{
			control.PolicyKindRequireMFA: poltOrgRow(poltOn, orgID, control.PolicyKindRequireMFA, control.PolicyConfig{}),
		}}
		svc := NewPolicyService(
			&poltKindedPlatformRepo{}, orgRepo,
			&poltUserRepo{user: user}, &poltPasskeyRepo{},
		)

		got, err := svc.CheckUserCompliance(t.Context(), user.ID, orgID)
		if err != nil {
			t.Fatalf("CheckUserCompliance: %v", err)
		}
		if len(got) != 1 || got[0].Kind != control.PolicyKindRequireMFA {
			t.Fatalf("violations = %+v, want exactly one require_mfa violation — the mandate is "+
				"set on the organization the user belongs to, so a user without MFA is not "+
				"compliant with it", got)
		}
		if len(orgRepo.askedOrgs) == 0 {
			t.Fatal("org policies were never consulted for a user who belongs to an organization; " +
				"only platform mandates would ever be reported")
		}
		for _, asked := range orgRepo.askedOrgs {
			if asked != orgID {
				t.Fatalf("policy was resolved for organization %v while checking a member of %v",
					asked, orgID)
			}
		}
	})

	t.Run("require_passkey set by the org alone", func(t *testing.T) {
		orgID := uuid.New()
		user := &control.User{ID: uuid.New(), MFAEnabled: true}
		orgRepo := &poltKindedOrgRepo{rows: map[control.PolicyKind]*control.OrgPolicy{
			control.PolicyKindRequirePasskey: poltOrgRow(poltOn, orgID, control.PolicyKindRequirePasskey, control.PolicyConfig{}),
		}}
		pk := &poltPasskeyRepo{count: 0}
		svc := NewPolicyService(
			&poltKindedPlatformRepo{}, orgRepo,
			&poltUserRepo{user: user}, pk,
		)

		got, err := svc.CheckUserCompliance(t.Context(), user.ID, orgID)
		if err != nil {
			t.Fatalf("CheckUserCompliance: %v", err)
		}
		if len(got) != 1 || got[0].Kind != control.PolicyKindRequirePasskey {
			t.Fatalf("violations = %+v, want exactly one require_passkey violation from the "+
				"org-level mandate", got)
		}
		if pk.countCalls == 0 {
			t.Fatal("passkeys were never counted, so the org's passkey mandate was not seen at all")
		}
	})

	t.Run("an org mandate does not reach a user checked without an org", func(t *testing.T) {
		orgID := uuid.New()
		user := &control.User{ID: uuid.New(), MFAEnabled: false}
		orgRepo := &poltKindedOrgRepo{rows: map[control.PolicyKind]*control.OrgPolicy{
			control.PolicyKindRequireMFA: poltOrgRow(poltOn, orgID, control.PolicyKindRequireMFA, control.PolicyConfig{}),
		}}
		svc := NewPolicyService(
			&poltKindedPlatformRepo{}, orgRepo,
			&poltUserRepo{user: user}, &poltPasskeyRepo{},
		)

		got, err := svc.CheckUserCompliance(t.Context(), user.ID, uuid.Nil)
		if err != nil {
			t.Fatalf("CheckUserCompliance: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("violations = %+v, want none — one organization's mandate must not be "+
				"reported against a user checked outside any organization", got)
		}
	})
}

// Counting passkeys is a per-user query. When no passkey mandate is in force
// there is nothing to count, and issuing the query anyway would put a needless
// round trip on a path that runs for every user in a compliance sweep.
func TestCheckUserComplianceDoesNotCountPasskeysWithoutAPasskeyMandate(t *testing.T) {
	user := &control.User{ID: uuid.New(), MFAEnabled: false}
	pk := &poltPasskeyRepo{count: 0}
	svc, _ := poltComplianceService(true /*mfa*/, false /*passkey*/, user, pk)

	if _, err := svc.CheckUserCompliance(t.Context(), user.ID, uuid.Nil); err != nil {
		t.Fatalf("CheckUserCompliance: %v", err)
	}
	if pk.countCalls != 0 {
		t.Fatalf("passkeys were counted %d times with no passkey mandate in force, want 0 — "+
			"the count is only meaningful when something requires a passkey", pk.countCalls)
	}
}

// An unknown user is an error, not "compliant". If this ever returned an empty
// violation slice for a missing row, a compliance sweep over a stale user list
// would report a clean bill of health for accounts that do not exist, and an
// operator would switch a mandate on believing everyone had enrolled.
func TestCheckUserComplianceRefusesAnUnknownUser(t *testing.T) {
	svc, _ := poltComplianceService(true, true, nil /* GetByID returns nil, nil */, &poltPasskeyRepo{})

	got, err := svc.CheckUserCompliance(t.Context(), uuid.New(), uuid.Nil)
	if err == nil {
		t.Fatalf("CheckUserCompliance returned %v and no error for a user that does not exist; "+
			"a missing user must not read as a compliant one", got)
	}
	if !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("err = %v, want it to say the user was not found", err)
	}
	if got != nil {
		t.Fatalf("violations = %v alongside an error, want nil", got)
	}
}

// Every dependency failure inside the compliance check has to reach the caller.
// Swallowing any of them turns "we could not tell" into "there is nothing
// wrong", which is exactly the answer that lets a mandate be enabled over the
// top of accounts that cannot satisfy it.
func TestCheckUserComplianceSurfacesEveryDependencyFailure(t *testing.T) {
	poltBoom := errors.New("store unavailable")
	userID := uuid.New()

	t.Run("user lookup", func(t *testing.T) {
		svc := NewPolicyService(
			&poltKindedPlatformRepo{}, &poltOrgRepo{},
			&poltUserRepo{userErr: poltBoom}, &poltPasskeyRepo{},
		)
		if _, err := svc.CheckUserCompliance(t.Context(), userID, uuid.Nil); !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the user lookup failure", err)
		}
	})

	t.Run("policy lookup", func(t *testing.T) {
		svc := NewPolicyService(
			&poltKindedPlatformRepo{err: poltBoom}, &poltOrgRepo{},
			&poltUserRepo{user: &control.User{ID: userID}}, &poltPasskeyRepo{},
		)
		if _, err := svc.CheckUserCompliance(t.Context(), userID, uuid.Nil); !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the policy lookup failure", err)
		}
	})

	t.Run("passkey count", func(t *testing.T) {
		svc, _ := poltComplianceService(false, true,
			&control.User{ID: userID}, &poltPasskeyRepo{countErr: poltBoom})
		got, err := svc.CheckUserCompliance(t.Context(), userID, uuid.Nil)
		if !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the passkey count failure", err)
		}
		if got != nil {
			t.Fatalf("violations = %v alongside an error, want nil — a partial list would be "+
				"read as the complete one", got)
		}
	})
}

// ---------------------------------------------------------------------------
// SetPlatformPolicy / SetOrgPolicy
// ---------------------------------------------------------------------------

// An unrecognised kind must be refused before anything is written. The kind is
// a free-form string on the wire, and a typo that reached storage would sit
// there forever: EffectivePolicy has no arm for an unknown kind, so the row
// would be permanently inert while the operator's UI showed a policy in place.
func TestSetPolicyRejectsAnUnknownKindWithoutWriting(t *testing.T) {
	// A near-miss for the real "require_mfa": exactly the shape of value an
	// operator or a hand-written client actually sends by mistake.
	const bogus = control.PolicyKind("requires_mfa")

	t.Run("platform", func(t *testing.T) {
		d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
		got, err := poltNewService(d).SetPlatformPolicy(t.Context(), uuid.New(), bogus, true, control.PolicyConfig{})
		if err == nil {
			t.Fatalf("SetPlatformPolicy accepted the unknown kind %q and returned %+v", bogus, got)
		}
		if !strings.Contains(err.Error(), string(bogus)) {
			t.Fatalf("err = %v, want it to echo the rejected kind so the typo is visible", err)
		}
		if len(d.platform.upserted) != 0 {
			t.Fatalf("%d rows were written for an unknown kind, want 0 — an inert policy row is "+
				"worse than no row because the console shows it as configured", len(d.platform.upserted))
		}
	})

	t.Run("org", func(t *testing.T) {
		d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
		got, err := poltNewService(d).SetOrgPolicy(t.Context(), uuid.New(), uuid.New(), bogus, true, control.PolicyConfig{})
		if err == nil {
			t.Fatalf("SetOrgPolicy accepted the unknown kind %q and returned %+v", bogus, got)
		}
		if len(d.org.upserted) != 0 {
			t.Fatalf("%d rows were written for an unknown kind, want 0", len(d.org.upserted))
		}
	})
}

// Turning a mandate on while users cannot satisfy it would lock those users
// out, so the write is refused and the operator is told how many accounts are
// in the way. The count has to appear in the message: "cannot enable" alone
// gives an operator no idea whether they are three enrolments away or three
// thousand.
func TestSetPolicyRefusesToEnableAMandateOverNonCompliantUsers(t *testing.T) {
	t.Run("platform", func(t *testing.T) {
		d := &poltDeps{
			platform: &poltPlatformRepo{},
			org:      &poltOrgRepo{},
			users:    &poltUserRepo{withoutMFA: 7},
			passkeys: &poltPasskeyRepo{},
		}
		_, err := poltNewService(d).SetPlatformPolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA, true, control.PolicyConfig{})
		if err == nil {
			t.Fatal("require_mfa was enabled platform-wide while 7 users had no MFA; every one " +
				"of them is now locked out of a platform they cannot re-enter")
		}
		if !strings.Contains(err.Error(), "7") {
			t.Fatalf("err = %q, want it to state how many users are blocking (7)", err.Error())
		}
		if len(d.platform.upserted) != 0 {
			t.Fatal("the policy was written despite the pre-enable check failing")
		}
		if len(d.users.revokeMFAScopes) != 0 {
			t.Fatal("tokens were revoked for a policy that was never enabled")
		}

		// The mirror of the org assertion below: the platform path must count
		// across the WHOLE platform. Scoped to any single organization it would
		// report zero for every tenant but that one, and the mandate would be
		// switched on over the top of everybody else's non-compliant accounts.
		if len(d.users.countMFAScopes) != 1 {
			t.Fatalf("pre-enable count ran %d times, want 1", len(d.users.countMFAScopes))
		}
		if scope := d.users.countMFAScopes[0]; scope != nil {
			t.Fatalf("platform pre-enable count was scoped to organization %v, want nil "+
				"(every user on the platform) — a scoped count answers a different question "+
				"than the one a platform-wide mandate asks", *scope)
		}
	})

	t.Run("org", func(t *testing.T) {
		orgID := uuid.New()
		d := &poltDeps{
			platform: &poltPlatformRepo{},
			org:      &poltOrgRepo{},
			users:    &poltUserRepo{withoutPasskey: 3},
			passkeys: &poltPasskeyRepo{},
		}
		_, err := poltNewService(d).SetOrgPolicy(t.Context(), uuid.New(), orgID, control.PolicyKindRequirePasskey, true, control.PolicyConfig{})
		if err == nil {
			t.Fatal("require_passkey was enabled for an org while 3 of its members had no passkey")
		}
		if len(d.org.upserted) != 0 {
			t.Fatal("the org policy was written despite the pre-enable check failing")
		}

		// The scope of the pre-enable count is the safety property here. If the
		// org path counted platform-wide it would block an org whose own members
		// are all compliant, purely because some unrelated tenant is not.
		if len(d.users.countPasskeyScopes) != 1 {
			t.Fatalf("pre-enable count ran %d times, want 1", len(d.users.countPasskeyScopes))
		}
		if scope := d.users.countPasskeyScopes[0]; scope == nil || *scope != orgID {
			t.Fatalf("pre-enable count was scoped to %v, want organization %v — an org-scoped "+
				"policy must be gated on its own members, not the whole platform", scope, orgID)
		}
	})
}

// Enabling a mandate is not just a row: the sessions that were minted before it
// existed have to go, or a user who never enrolled keeps browsing on a token
// issued under the old rules until it expires on its own. The scope of that
// revocation is the security property — the org path must pass its own org, and
// the platform path must pass nil for "everyone".
func TestEnablingAMandateRevokesTheSessionsThatPredateIt(t *testing.T) {
	t.Run("platform require_mfa revokes platform-wide", func(t *testing.T) {
		d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
		if _, err := poltNewService(d).SetPlatformPolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA, true, control.PolicyConfig{}); err != nil {
			t.Fatalf("SetPlatformPolicy: %v", err)
		}
		if len(d.users.revokeMFAScopes) != 1 {
			t.Fatalf("BulkRevokeTokensWithoutMFA called %d times, want 1 — without it, every "+
				"user who has not enrolled keeps an already-minted session that the new mandate "+
				"never gets a chance to challenge", len(d.users.revokeMFAScopes))
		}
		if d.users.revokeMFAScopes[0] != nil {
			t.Fatalf("platform-wide revocation was scoped to org %v, want nil (everyone)", *d.users.revokeMFAScopes[0])
		}
		if len(d.users.revokePkScopes) != 0 {
			t.Fatal("enabling require_mfa also revoked sessions for missing passkeys")
		}
	})

	t.Run("org require_passkey revokes only that org", func(t *testing.T) {
		orgID := uuid.New()
		d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
		if _, err := poltNewService(d).SetOrgPolicy(t.Context(), uuid.New(), orgID, control.PolicyKindRequirePasskey, true, control.PolicyConfig{}); err != nil {
			t.Fatalf("SetOrgPolicy: %v", err)
		}
		if len(d.users.revokePkScopes) != 1 {
			t.Fatalf("BulkRevokeTokensWithoutPasskey called %d times, want 1 — an org that turns "+
				"the passkey mandate on keeps handing service to its own members' pre-existing "+
				"sessions until this runs", len(d.users.revokePkScopes))
		}
		if scope := d.users.revokePkScopes[0]; scope == nil || *scope != orgID {
			t.Fatalf("revocation was scoped to %v, want organization %v — a nil scope here would "+
				"sign out every user on the platform because one tenant changed its own policy",
				scope, orgID)
		}
	})
}

// Switching a mandate off, or setting a policy that has no compliance notion at
// all, must not sign anybody out and must not run a pre-enable count. A
// password length change that logged the whole platform out would be an
// outage caused by an ordinary configuration edit.
func TestSetPolicyRevokesNothingWhenNoMandateIsBeingTurnedOn(t *testing.T) {
	cases := []struct {
		name    string
		kind    control.PolicyKind
		enabled bool
	}{
		{"disabling require_mfa", control.PolicyKindRequireMFA, false},
		{"disabling require_passkey", control.PolicyKindRequirePasskey, false},
		{"enabling password_min_length", control.PolicyKindPasswordMinLength, true},
		{"enabling session_max_lifetime", control.PolicyKindSessionMaxLifetime, true},
		{"enabling password_complexity", control.PolicyKindPasswordComplexity, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orgID := uuid.New()
			d := &poltDeps{
				platform: &poltPlatformRepo{},
				org:      &poltOrgRepo{},
				// Non-zero counts: if a pre-enable check ran for these kinds it
				// would refuse the write, and the assertions below would notice.
				users:    &poltUserRepo{withoutMFA: 9, withoutPasskey: 9},
				passkeys: &poltPasskeyRepo{},
			}
			svc := poltNewService(d)

			if _, err := svc.SetPlatformPolicy(t.Context(), uuid.New(), tc.kind, tc.enabled, control.PolicyConfig{}); err != nil {
				t.Fatalf("SetPlatformPolicy(%s, enabled=%v): %v — no pre-enable check should apply here", tc.kind, tc.enabled, err)
			}
			if _, err := svc.SetOrgPolicy(t.Context(), uuid.New(), orgID, tc.kind, tc.enabled, control.PolicyConfig{}); err != nil {
				t.Fatalf("SetOrgPolicy(%s, enabled=%v): %v", tc.kind, tc.enabled, err)
			}

			if n := len(d.users.revokeMFAScopes) + len(d.users.revokePkScopes); n != 0 {
				t.Fatalf("%d bulk token revocations ran for %s(enabled=%v), want 0 — signing users "+
					"out is only justified when a mandate they may not satisfy is switched on",
					n, tc.kind, tc.enabled)
			}
			if n := len(d.users.countMFAScopes) + len(d.users.countPasskeyScopes); n != 0 {
				t.Fatalf("%d pre-enable compliance counts ran for %s(enabled=%v), want 0",
					n, tc.kind, tc.enabled)
			}
			if len(d.platform.upserted) != 1 || len(d.org.upserted) != 1 {
				t.Fatalf("writes: platform=%d org=%d, want 1 each",
					len(d.platform.upserted), len(d.org.upserted))
			}
		})
	}
}

// The row handed to the repository has to carry the config and, for the org
// variant, the organization. A dropped OrganizationID would attach the policy
// to the zero UUID, where EffectivePolicy never looks it up for a real org —
// the operator would see a saved policy that does nothing.
func TestSetPolicyPersistsTheConfigAndOrganization(t *testing.T) {
	cfg := control.PolicyConfig{MinLength: 18, RequireDigit: true, MaxLifetimeMinutes: 45}
	author := uuid.New()
	orgID := uuid.New()

	d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
	svc := poltNewService(d)

	pp, err := svc.SetPlatformPolicy(t.Context(), author, control.PolicyKindPasswordMinLength, true, cfg)
	if err != nil {
		t.Fatalf("SetPlatformPolicy: %v", err)
	}
	if got := pp.Config.Data(); got != cfg {
		t.Fatalf("platform policy config = %+v, want %+v — the caller's settings were not stored", got, cfg)
	}
	if pp.CreatedBy != author {
		t.Fatalf("CreatedBy = %v, want %v — the audit trail for a policy change names who made it", pp.CreatedBy, author)
	}
	if len(d.platform.upserted) != 1 || d.platform.upserted[0] != pp {
		t.Fatal("the returned platform policy is not the row that was written")
	}

	op, err := svc.SetOrgPolicy(t.Context(), author, orgID, control.PolicyKindPasswordMinLength, true, cfg)
	if err != nil {
		t.Fatalf("SetOrgPolicy: %v", err)
	}
	if op.OrganizationID != orgID {
		t.Fatalf("OrganizationID = %v, want %v — a policy saved against the wrong org is invisible "+
			"to EffectivePolicy and silently enforces nothing", op.OrganizationID, orgID)
	}
	if got := op.Config.Data(); got != cfg {
		t.Fatalf("org policy config = %+v, want %+v", got, cfg)
	}
}

// A failed write must not be followed by a mass sign-out. The revocation is
// only correct once the policy it enforces is actually in place; running it
// after a failed upsert would log everyone out to enforce a rule that does not
// exist.
func TestSetPolicyDoesNotRevokeSessionsWhenTheWriteFails(t *testing.T) {
	poltBoom := errors.New("unique violation")

	t.Run("platform", func(t *testing.T) {
		d := &poltDeps{
			platform: &poltPlatformRepo{upsertErr: poltBoom},
			org:      &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{},
		}
		got, err := poltNewService(d).SetPlatformPolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA, true, control.PolicyConfig{})
		if !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the upsert failure", err)
		}
		if got != nil {
			t.Fatalf("a policy was returned (%+v) even though it was never stored", got)
		}
		if len(d.users.revokeMFAScopes) != 0 {
			t.Fatal("sessions were revoked for a policy whose write failed; the platform is now " +
				"signed out in service of a mandate that was never saved")
		}
	})

	t.Run("org", func(t *testing.T) {
		d := &poltDeps{
			platform: &poltPlatformRepo{}, org: &poltOrgRepo{upsertErr: poltBoom},
			users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{},
		}
		got, err := poltNewService(d).SetOrgPolicy(t.Context(), uuid.New(), uuid.New(), control.PolicyKindRequirePasskey, true, control.PolicyConfig{})
		if !errors.Is(err, poltBoom) {
			t.Fatalf("err = %v, want it to wrap the upsert failure", err)
		}
		if got != nil {
			t.Fatalf("a policy was returned (%+v) even though it was never stored", got)
		}
		if len(d.users.revokePkScopes) != 0 {
			t.Fatal("sessions were revoked for an org policy whose write failed")
		}
	})
}

// A pre-enable count that errors must abort the change rather than being read
// as "zero users are non-compliant" — that reading is precisely the one that
// enables a mandate over the top of a fleet that cannot meet it.
func TestSetPolicyAbortsWhenThePreEnableCountFails(t *testing.T) {
	poltBoom := errors.New("count failed")

	d := &poltDeps{
		platform: &poltPlatformRepo{}, org: &poltOrgRepo{},
		users:    &poltUserRepo{withoutMFAErr: poltBoom},
		passkeys: &poltPasskeyRepo{},
	}
	_, err := poltNewService(d).SetPlatformPolicy(t.Context(), uuid.New(), control.PolicyKindRequireMFA, true, control.PolicyConfig{})
	if !errors.Is(err, poltBoom) {
		t.Fatalf("err = %v, want it to wrap the pre-enable count failure — an unanswerable "+
			"compliance question must not be answered with 'nobody is affected'", err)
	}
	if len(d.platform.upserted) != 0 {
		t.Fatal("the mandate was enabled without knowing whether anyone could satisfy it")
	}
}

// ListEffectivePolicies is the API surface an operator's console reads, and it
// has to return every kind rather than only the ones with rows — a kind that
// vanished from the response would render as absent rather than as "not
// enforced", which are different claims.
func TestListEffectivePoliciesCoversEveryKnownKind(t *testing.T) {
	d := &poltDeps{platform: &poltPlatformRepo{}, org: &poltOrgRepo{}, users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{}}
	got, err := poltNewService(d).ListEffectivePolicies(t.Context(), uuid.New())
	if err != nil {
		t.Fatalf("ListEffectivePolicies: %v", err)
	}
	if len(got) != len(poltAllKinds) {
		t.Fatalf("got %d effective policies, want %d — every known kind must be reported even "+
			"when nothing is configured for it", len(got), len(poltAllKinds))
	}
	seen := map[control.PolicyKind]bool{}
	for _, r := range got {
		seen[r.Kind] = true
		if r.Enabled {
			t.Fatalf("%s reported as enabled with no policy rows at all", r.Kind)
		}
	}
	for _, k := range poltAllKinds {
		if !seen[k] {
			t.Fatalf("%s is missing from the effective policy list", k)
		}
	}
}

// TestDisabledPolicyRowContributesNoConfig is the regression guard for a merge
// that respected the enabled BIT of each level but not the config it carried.
//
// A disabled row is "inherit": the identity element of this merge. Reading its
// numbers anyway meant that as long as ANY level had the kind switched on, a
// switched-off level went on tightening the result — so an operator could turn a
// platform policy OFF and have it keep applying, which is indistinguishable from
// the setting being ignored.
//
// The table is the specification. Every row states the two levels' states, the
// expected outcome, and why — so the merge's rules can be read off it without
// reading the implementation.
func TestDisabledPolicyRowContributesNoConfig(t *testing.T) {
	orgID := uuid.New()

	t.Run("session lifetime", func(t *testing.T) {
		cases := []struct {
			name     string
			platform *control.PlatformPolicy
			org      *control.OrgPolicy
			wantOn   bool
			wantMins int
			why      string
		}{
			{
				name:     "disabled platform must not cap an enabled org",
				platform: poltPlatformRow(poltOff, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 5}),
				org:      poltOrgRow(poltOn, orgID, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 480}),
				wantOn:   true, wantMins: 480,
				why: "the organization asked for 8 hours; the 5-minute ceiling belongs to a " +
					"platform policy that is switched OFF, and honouring it expires every " +
					"session in that organization every five minutes",
			},
			{
				name:     "disabled org must not cap an enabled platform",
				platform: poltPlatformRow(poltOn, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 480}),
				org:      poltOrgRow(poltOff, orgID, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 5}),
				wantOn:   true, wantMins: 480,
				why: "same defect from the other side: a stale org row the operator disabled",
			},
			{
				name:     "both enabled still takes the tighter",
				platform: poltPlatformRow(poltOn, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 60}),
				org:      poltOrgRow(poltOn, orgID, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 480}),
				wantOn:   true, wantMins: 60,
				why: "a restriction merges toward strictness when both levels mean it",
			},
			{
				name:     "both disabled expresses nothing",
				platform: poltPlatformRow(poltOff, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 5}),
				org:      poltOrgRow(poltOff, orgID, control.PolicyKindSessionMaxLifetime, control.PolicyConfig{MaxLifetimeMinutes: 10}),
				wantOn:   false, wantMins: 0,
				why: "inherit on both sides is the identity element",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				svc := poltNewService(&poltDeps{
					platform: &poltPlatformRepo{row: tc.platform},
					org:      &poltOrgRepo{row: tc.org},
					users:    &poltUserRepo{}, passkeys: &poltPasskeyRepo{},
				})
				got, err := svc.EffectivePolicy(context.Background(), orgID, control.PolicyKindSessionMaxLifetime)
				if err != nil {
					t.Fatalf("EffectivePolicy: %v", err)
				}
				if got.Enabled != tc.wantOn {
					t.Fatalf("Enabled = %v, want %v — %s", got.Enabled, tc.wantOn, tc.why)
				}
				if got.Config.MaxLifetimeMinutes != tc.wantMins {
					t.Fatalf("MaxLifetimeMinutes = %d, want %d — %s",
						got.Config.MaxLifetimeMinutes, tc.wantMins, tc.why)
				}
			})
		}
	})

	t.Run("password minimum length", func(t *testing.T) {
		svc := poltNewService(&poltDeps{
			platform: &poltPlatformRepo{row: poltPlatformRow(poltOff, control.PolicyKindPasswordMinLength, control.PolicyConfig{MinLength: 24})},
			org:      &poltOrgRepo{row: poltOrgRow(poltOn, orgID, control.PolicyKindPasswordMinLength, control.PolicyConfig{MinLength: 10})},
			users:    &poltUserRepo{}, passkeys: &poltPasskeyRepo{},
		})
		got, err := svc.EffectivePolicy(context.Background(), orgID, control.PolicyKindPasswordMinLength)
		if err != nil {
			t.Fatalf("EffectivePolicy: %v", err)
		}
		if got.Config.MinLength != 10 {
			t.Fatalf("MinLength = %d, want 10 — a platform policy the operator switched OFF is "+
				"still forcing 24-character passwords on this organization", got.Config.MinLength)
		}
	})

	t.Run("password complexity flags", func(t *testing.T) {
		svc := poltNewService(&poltDeps{
			platform: &poltPlatformRepo{row: poltPlatformRow(poltOff, control.PolicyKindPasswordComplexity, control.PolicyConfig{
				RequireUppercase: true, RequireSpecial: true,
			})},
			org:   &poltOrgRepo{row: poltOrgRow(poltOn, orgID, control.PolicyKindPasswordComplexity, control.PolicyConfig{RequireDigit: true})},
			users: &poltUserRepo{}, passkeys: &poltPasskeyRepo{},
		})
		got, err := svc.EffectivePolicy(context.Background(), orgID, control.PolicyKindPasswordComplexity)
		if err != nil {
			t.Fatalf("EffectivePolicy: %v", err)
		}
		if got.Config.RequireUppercase || got.Config.RequireSpecial {
			t.Fatalf("a disabled platform row's complexity flags survived the merge "+
				"(uppercase=%v special=%v); OR unions them in unless the read is gated",
				got.Config.RequireUppercase, got.Config.RequireSpecial)
		}
		if !got.Config.RequireDigit {
			t.Fatal("the enabled organization row's own flag was dropped")
		}
	})
}
