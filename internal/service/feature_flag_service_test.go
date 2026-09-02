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

	err := svc.SetPlatform(ctx, "legacy_lgoin", false, by)
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
	if err := svc.SetPlatform(ctx, "proxmox.provisioning", false, by); err != nil {
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

	if err := svc.SetPlatform(ctx, "legacy_login", false, by); err != nil {
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
