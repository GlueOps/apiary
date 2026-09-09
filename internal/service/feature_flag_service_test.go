package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
)

var flagTestSchema = testsupport.SchemaName("apiary_flagsvc_test")

func flagSvc(t *testing.T) (*FeatureFlagService, uuid.UUID) {
	t.Helper()
	db := testsupport.OpenSchema(t, flagTestSchema,
		&control.Organization{},
		&control.PlatformFeatureFlag{}, &control.OrgFeatureFlag{},
	)
	org := control.Organization{Name: "acme-" + uuid.NewString()[:8]}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	return NewFeatureFlagService(repo.NewFeatureFlagRepoGorm(db)), org.ID
}

// THE REGISTRY IS THE POINT: a key this build does not declare cannot be stored.
//
// The alternative fails silently, which is why it was rejected. Writing
// "legacy_lgoin" to a free-form store succeeds, returns 202, and leaves
// legacy_login exactly as it was — the operator believes they turned it off and
// nothing anywhere reports a problem.
func TestAnUnregisteredKeyCannotBeStored(t *testing.T) {
	svc, orgID := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	err := svc.SetPlatform(ctx, "legacy_lgoin", false, by, false)
	if !errors.Is(err, control.ErrUnknownFlag) {
		t.Fatalf("err = %v, want the typo refused", err)
	}
	if err := svc.SetOrg(ctx, orgID, "proxmox.provisionning", true, by); !errors.Is(err, control.ErrUnknownFlag) {
		t.Fatalf("err = %v, want the typo refused at the organization level too", err)
	}

	// And the real flag is untouched, which is what the typo would have hidden.
	eff, err := svc.Effective(ctx, orgID, "legacy_login")
	if err != nil {
		t.Fatal(err)
	}
	if !eff.Value || eff.Source != control.FlagSourceDefault {
		t.Fatalf("legacy_login = %v from %v; the mistyped write must leave it alone",
			eff.Value, eff.Source)
	}
}

// Naming a real flag at the wrong LEVEL is a different mistake from mistyping
// one, and telling them apart is the difference between "check the spelling"
// and "this one is installation-wide".
func TestAPlatformScopedFlagCannotBeSetOnOneOrganization(t *testing.T) {
	svc, orgID := flagSvc(t)

	err := svc.SetOrg(context.Background(), orgID, "legacy_login", false, uuid.New())
	if !errors.Is(err, control.ErrFlagNotTenantScoped) {
		t.Fatalf("err = %v, want a refusal that names the scope, not the spelling", err)
	}
	if errors.Is(err, control.ErrUnknownFlag) {
		t.Fatal("a real flag at the wrong level was reported as unknown, sending an operator " +
			"to check a spelling that was correct")
	}
}

// Targeting one tenant is a ROW, never a key. This is the question the dotted
// notation would have made awkward if the tenant lived in the key string.
func TestATenantIsTargetedByRowAndTheOthersAreUnaffected(t *testing.T) {
	svc, acme := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	// The installation-wide default: nobody provisions Proxmox.
	if err := svc.SetPlatform(ctx, "proxmox.provisioning", false, by, false); err != nil {
		t.Fatal(err)
	}
	// Except this one.
	if err := svc.SetOrg(ctx, acme, "proxmox.provisioning", true, by); err != nil {
		t.Fatal(err)
	}

	on, err := svc.Enabled(ctx, acme, "proxmox.provisioning")
	if err != nil || !on {
		t.Fatalf("acme = %v (%v), want its own override to win", on, err)
	}

	other := uuid.New() // an organization with no row of its own
	on, err = svc.Enabled(ctx, other, "proxmox.provisioning")
	if err != nil {
		t.Fatal(err)
	}
	if on {
		t.Fatal("targeting one organization switched it on for another; the override is " +
			"supposed to be a row, not a change to the default")
	}

	// And the platform operator can see who will not be reached by a change to
	// the default.
	over, err := svc.OverridesFor(ctx, "proxmox.provisioning")
	if err != nil {
		t.Fatal(err)
	}
	if len(over) != 1 || over[0].OrganizationID != acme {
		t.Fatalf("overrides = %+v, want exactly the one organization that set it", over)
	}
	// The NAME travels with the id. A platform operator reading this is asking
	// "who will a change to the default not reach", and a uuid does not answer it.
	if over[0].OrganizationName == "" {
		t.Fatal("the override names no organization, so the list is a column of uuids")
	}
}

// CLEARING IS NOT SETTING FALSE. A cleared flag falls back to what this build
// was written to do, which is the only way to restore a default without knowing
// what it used to be.
func TestClearingFallsBackRatherThanSwitchingOff(t *testing.T) {
	svc, orgID := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	// Acknowledged, because legacy_login gates sign-in and this service has no
	// alternatives checker wired — the guard denies rather than guessing. The
	// guard itself is tested in TestDisablingSignInIsRefusedWithNoWayBackIn.
	if err := svc.SetPlatform(ctx, "legacy_login", false, by, true); err != nil {
		t.Fatal(err)
	}
	eff, _ := svc.Effective(ctx, orgID, "legacy_login")
	if eff.Value {
		t.Fatal("the platform switch off did not take")
	}

	if err := svc.ClearPlatform(ctx, "legacy_login"); err != nil {
		t.Fatal(err)
	}
	eff, err := svc.Effective(ctx, orgID, "legacy_login")
	if err != nil {
		t.Fatal(err)
	}
	if !eff.Value {
		t.Fatal("clearing an override left the flag off; a deleted row must mean 'not set', " +
			"or removing one silently disables a feature")
	}
	if eff.Source != control.FlagSourceDefault {
		t.Fatalf("source = %v, want the built-in default", eff.Source)
	}

	// Clearing what was never set is success: the requested state is reached.
	if err := svc.ClearPlatform(ctx, "legacy_login"); err != nil {
		t.Fatalf("clearing an unset flag failed: %v", err)
	}
}

// A caller asking about a flag that does not exist has a bug, and answering
// "off" would hide it behind behaviour that looks deliberate.
func TestAskingAboutAnUnknownFlagIsAnErrorNotAFalse(t *testing.T) {
	svc, orgID := flagSvc(t)

	on, err := svc.Enabled(context.Background(), orgID, "no.such.flag")
	if err == nil {
		t.Fatalf("an unknown flag answered %v instead of failing; a typo at a call site would "+
			"read as a deliberately disabled feature", on)
	}
	if !strings.Contains(err.Error(), "no.such.flag") {
		t.Fatalf("err = %v, want the key named", err)
	}
}

// Every declared flag appears in a listing, including the ones nobody has
// touched. A flag missing from the list is indistinguishable from one that is off.
func TestListingIncludesFlagsNobodyHasSet(t *testing.T) {
	svc, orgID := flagSvc(t)

	got, err := svc.ListEffective(context.Background(), orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(control.AllFlags()) {
		t.Fatalf("listed %d of %d declared flags", len(got), len(control.AllFlags()))
	}
	for _, e := range got {
		if e.Key == "" || e.Scope == "" {
			t.Fatalf("a listed flag is unusable: %+v", e)
		}
	}
}

type fakeAlternatives struct {
	ok  bool
	how string
	err error
}

func (f fakeAlternatives) Available(context.Context, uuid.UUID) (bool, string, error) {
	return f.ok, f.how, f.err
}

// THE ONE FLAG CHANGE THAT CAN LOCK EVERYONE OUT OF THE INSTALLATION.
//
// Turning off password sign-in with nothing else configured removes the ability
// to sign in — including the ability to sign in and turn it back on. That makes
// it hard to undo by construction, which is why the write is refused rather than
// merely warned about.
func TestDisablingSignInIsRefusedWithNoWayBackIn(t *testing.T) {
	svc, _ := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	guarded := svc.WithSignInAlternatives(fakeAlternatives{ok: false})

	err := guarded.SetPlatform(ctx, "legacy_login", false, by, false)
	if !errors.Is(err, control.ErrWouldLockEveryoneOut) {
		t.Fatalf("err = %v, want the write refused", err)
	}
	if !strings.Contains(err.Error(), "including you") {
		t.Fatalf("err = %q, want it to say the operator could not undo it either", err)
	}

	// The value must not have been written. A refusal that stored the value
	// anyway would be the worst of both.
	eff, ferr := guarded.Effective(ctx, uuid.Nil, "legacy_login")
	if ferr != nil {
		t.Fatal(ferr)
	}
	if !eff.Value {
		t.Fatal("the refused write took effect anyway")
	}
}

// ONLY ON THE DISABLING EDGE, and only for a flag that gates sign-in. A guard
// that fired on enabling, on clearing, or on unrelated flags would be an
// obstacle with no failure behind it — and obstacles with no failure behind them
// teach operators to reach for the override without reading it.
func TestTheLockoutGuardDoesNotFireOnSafeChanges(t *testing.T) {
	svc, orgID := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()
	guarded := svc.WithSignInAlternatives(fakeAlternatives{ok: false})

	// Enabling a sign-in method cannot lock anybody out.
	if err := guarded.SetPlatform(ctx, "legacy_login", true, by, false); err != nil {
		t.Fatalf("enabling sign-in was refused: %v", err)
	}
	// Clearing restores the compiled default, which is on.
	if err := guarded.ClearPlatform(ctx, "legacy_login"); err != nil {
		t.Fatalf("clearing was refused: %v", err)
	}
	// A flag that gates nothing about signing in.
	if err := guarded.SetPlatform(ctx, "legacy_registration", false, by, false); err != nil {
		t.Fatalf("an unrelated flag was caught by the sign-in guard: %v", err)
	}
	if err := guarded.SetOrg(ctx, orgID, "proxmox.provisioning", false, by); err != nil {
		t.Fatalf("a tenant flag was caught by the sign-in guard: %v", err)
	}
}

// An alternative existing, or the operator saying they know, both let it through.
func TestDisablingSignInIsAllowedWithAWayBackIn(t *testing.T) {
	svc, _ := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	withProvider := svc.WithSignInAlternatives(fakeAlternatives{ok: true, how: "Google"})
	if err := withProvider.SetPlatform(ctx, "legacy_login", false, by, false); err != nil {
		t.Fatalf("refused with a provider configured: %v", err)
	}

	noneButAcknowledged := svc.WithSignInAlternatives(fakeAlternatives{ok: false})
	if err := noneButAcknowledged.SetPlatform(ctx, "legacy_login", false, by, true); err != nil {
		t.Fatalf("refused an explicitly acknowledged lockout: %v", err)
	}
}

// THIS GUARD DENIES WHEN IT CANNOT DECIDE, which is the opposite of the gate on
// the login handler — and both are right. The login gate decides whether one
// request proceeds, and denying there on an outage locks everyone out. This
// decides whether to make a change that is hard to undo by construction.
// Refusing a configuration change costs a retry; allowing one wrongly costs the
// system.
func TestTheLockoutGuardDeniesWhenItCannotDecide(t *testing.T) {
	svc, _ := flagSvc(t)
	ctx := context.Background()
	by := uuid.New()

	// The checker errored.
	broken := svc.WithSignInAlternatives(fakeAlternatives{err: errors.New("store is down")})
	if err := broken.SetPlatform(ctx, "legacy_login", false, by, false); !errors.Is(err, control.ErrWouldLockEveryoneOut) {
		t.Fatalf("err = %v, want a refusal when the check could not be made", err)
	}

	// No checker wired at all.
	unwired, _ := flagSvc(t)
	if err := unwired.SetPlatform(ctx, "legacy_login", false, by, false); !errors.Is(err, control.ErrWouldLockEveryoneOut) {
		t.Fatalf("err = %v, want a refusal when no check is wired", err)
	}
}
