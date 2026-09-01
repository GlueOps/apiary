package control

import "testing"

// THE NARROWER LEVEL WINS, IN EITHER DIRECTION.
//
// This is the whole reason flags are not policies. EffectivePolicy merges with
// OR and can only tighten, so an organization can never relax what the platform
// requires — correct for a security restriction, and useless for "off
// everywhere, except the one tenant still mid-migration".
func TestAnOrganizationOverridesThePlatformInBothDirections(t *testing.T) {
	f := Flag{Key: "proxmox.provisioning", Scope: ScopeTenant, Default: false}

	// Platform off, tenant on. An OR-merge would also say on — so this case
	// alone proves nothing, and the next one is the real test.
	got := ResolveFlag(f, &PlatformFeatureFlag{Value: false}, &OrgFeatureFlag{Value: true})
	if !got.Value || got.Source != FlagSourceOrganization {
		t.Fatalf("value=%v source=%v, want the organization's own on", got.Value, got.Source)
	}

	// Platform ON, tenant OFF. This is the direction a policy cannot express,
	// and the reason this is a separate subsystem.
	got = ResolveFlag(f, &PlatformFeatureFlag{Value: true}, &OrgFeatureFlag{Value: false})
	if got.Value {
		t.Fatal("an organization could not switch a flag OFF against the platform, which is " +
			"the direction an OR-merged policy already fails at")
	}
	if got.Source != FlagSourceOrganization {
		t.Fatalf("source = %v, want the organization named as the decider", got.Source)
	}
}

// AN ABSENT ROW IS "NOT SET", NEVER "FALSE".
//
// If absence meant off, deleting a row would silently disable a feature, and
// restoring the previous behaviour would require knowing what the default used
// to be — which is exactly the knowledge the deletion destroyed.
func TestAnAbsentRowFallsThroughRatherThanReadingAsOff(t *testing.T) {
	f := Flag{Key: "legacy_login", Scope: ScopePlatform, Default: true}

	got := ResolveFlag(f, nil, nil)
	if !got.Value {
		t.Fatal("a flag nobody has set read as off; deleting a row would silently disable it")
	}
	if got.Source != FlagSourceDefault {
		t.Fatalf("source = %v, want the built-in default named", got.Source)
	}

	// And an explicit false is distinguishable from absence by its source.
	got = ResolveFlag(f, &PlatformFeatureFlag{Value: false}, nil)
	if got.Value || got.Source != FlagSourcePlatform {
		t.Fatalf("value=%v source=%v, want an explicit platform off", got.Value, got.Source)
	}
}

// A stored organization row for a platform-scoped flag should be impossible —
// the write refuses it — but resolution is not the place to trust that.
func TestAPlatformScopedFlagIgnoresAnOrganizationRow(t *testing.T) {
	f := Flag{Key: "legacy_login", Scope: ScopePlatform, Default: false}

	got := ResolveFlag(f, &PlatformFeatureFlag{Value: false}, &OrgFeatureFlag{Value: true})
	if got.Value {
		t.Fatal("an organization row switched on a flag that has no per-tenant meaning; a row " +
			"that got past the write would silently take effect")
	}
}

// The registry is the closed set a write is checked against, and every entry
// has to carry enough to be listed and explained.
func TestEveryDeclaredFlagIsUsable(t *testing.T) {
	all := AllFlags()
	if len(all) == 0 {
		t.Fatal("no flags are declared")
	}
	seen := map[string]bool{}
	for _, f := range all {
		if f.Key == "" {
			t.Fatal("a flag has no key")
		}
		if seen[f.Key] {
			t.Fatalf("%q is declared twice; one of them silently wins", f.Key)
		}
		seen[f.Key] = true
		if f.Scope != ScopePlatform && f.Scope != ScopeTenant {
			t.Fatalf("%q has scope %q, which no write path admits", f.Key, f.Scope)
		}
		if f.Why == "" {
			t.Fatalf("%q has no description; a flag nobody can explain is one nobody dares "+
				"delete", f.Key)
		}
		if got, ok := LookupFlag(f.Key); !ok || got.Key != f.Key {
			t.Fatalf("%q is listed but not resolvable by key", f.Key)
		}
	}
	if _, ok := LookupFlag("legacy_lgoin"); ok {
		t.Fatal("a mistyped key resolved; the registry exists so that write is refused")
	}
}
