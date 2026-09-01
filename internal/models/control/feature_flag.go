package control

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Feature flags are NOT policies, and the difference is the merge.
//
// A policy is a security restriction, so EffectivePolicy merges the platform and
// organization rows with OR and can only ever TIGHTEN — an organization relying
// on its own strict setting must not lose it when the platform loosens. A flag
// is the opposite: the whole point of "legacy_login is off everywhere, except
// this one tenant still mid-migration" is that the narrower level WINS, in
// either direction. An OR-merge cannot express that.
//
// So this is a sibling of the policy tables rather than an extension of them.
// Same two-table shape, opposite resolution rule.

// FlagScope says at which levels a flag may carry a stored row.
type FlagScope string

const (
	// ScopePlatform: one row for the whole installation. A per-organization row
	// is refused — there is no such thing as one tenant's legacy login page.
	ScopePlatform FlagScope = "platform"

	// ScopeTenant: a platform row acts as the default for every organization,
	// and an organization row overrides it for that one.
	ScopeTenant FlagScope = "tenant"
)

// Flag is one declared feature flag.
//
// THE REGISTRY DECLARES THE FLAG; THE DATABASE SUPPLIES THE TENANT. A key is
// never per-organization — "acme.proxmox.provisioning" is the pair (acme,
// proxmox.provisioning) rendered in dotted form, not a key anybody declared. So
// adding a tenant adds no flags, and adding a flag is one struct literal that
// applies to every tenant that exists or ever will.
//
// Declared in code rather than free-form because of how the other choice fails.
// Writing "legacy_lgoin" to a free-form store succeeds, returns 202, and leaves
// legacy_login exactly as it was — the operator believes they turned it off and
// nothing anywhere reports a problem. An unknown key here is refused at the
// write, in the same spirit as every other unknown input in this system.
type Flag struct {
	Key   string
	Scope FlagScope

	// Default is what applies when NEITHER level has a row.
	//
	// An absent row means "not set, fall through" and never "false". If absence
	// meant off, deleting a row would silently disable a feature, and the way to
	// restore a default would be to know what it used to be.
	Default bool

	// Why is shown beside the flag wherever it is listed. A flag nobody can
	// explain is a flag nobody dares delete.
	Why string
}

// flagRegistry is every flag this build knows about.
//
// Keys are dotted for grouping only; the string is opaque to storage and is not
// parsed. "proxmox.provisioning" is one key, not a path.
var flagRegistry = map[string]Flag{
	"legacy_login": {
		Key: "legacy_login", Scope: ScopePlatform, Default: true,
		Why: "the pre-OIDC username and password login form",
	},
	"legacy_registration": {
		Key: "legacy_registration", Scope: ScopePlatform, Default: true,
		Why: "self-service registration without an invitation",
	},
	"proxmox.provisioning": {
		Key: "proxmox.provisioning", Scope: ScopeTenant, Default: false,
		Why: "provision Proxmox guests for this organization",
	},
	"aws.provisioning": {
		Key: "aws.provisioning", Scope: ScopeTenant, Default: false,
		Why: "provision AWS instances for this organization",
	},
}

// LookupFlag returns the declared flag, or false for an unregistered key.
func LookupFlag(key string) (Flag, bool) {
	f, ok := flagRegistry[key]
	return f, ok
}

// AllFlags returns every declared flag, ordered by key so that a listing, a
// diff and a screen all agree on the order.
func AllFlags() []Flag {
	out := make([]Flag, 0, len(flagRegistry))
	for _, f := range flagRegistry {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// ErrUnknownFlag and ErrFlagNotTenantScoped are the two ways a write is refused.
var (
	ErrUnknownFlag = fmt.Errorf("no such feature flag")

	// ErrFlagNotTenantScoped is deliberately distinct from "unknown". An operator
	// who names a real flag at the wrong level has made a different mistake from
	// one who mistyped it, and telling them apart is the difference between
	// "check the spelling" and "this one is installation-wide".
	ErrFlagNotTenantScoped = fmt.Errorf("this flag is set for the whole installation, not per organization")
)

// PlatformFeatureFlag is the installation-wide value for one key.
//
// Separate table rather than a nullable organization_id on one, mirroring
// PlatformPolicy and OrgPolicy. A nullable column would need a unique index over
// (key, organization_id) where Postgres treats NULLs as distinct — so two
// platform rows for the same key would both be accepted — and the partial
// indexes that fix it cannot be expressed in GORM tags, which means Atlas would
// diff them away on the next generated migration.
type PlatformFeatureFlag struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key       string    `gorm:"not null;type:varchar(120);uniqueIndex" json:"key"`
	Value     bool      `gorm:"not null" json:"value"`
	UpdatedBy uuid.UUID `gorm:"type:uuid;not null" json:"updated_by"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

// OrgFeatureFlag is one organization's override of a tenant-scoped flag.
type OrgFeatureFlag struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_org_feature_flags_org_key" json:"organization_id"`
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"-"`
	Key            string       `gorm:"not null;type:varchar(120);uniqueIndex:idx_org_feature_flags_org_key" json:"key"`
	Value          bool         `gorm:"not null" json:"value"`
	UpdatedBy      uuid.UUID    `gorm:"type:uuid;not null" json:"updated_by"`
	CreatedAt      time.Time    `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"type:timestamptz;autoUpdateTime;not null;default:now()" json:"updated_at"`
}

// FlagSource says which level decided an effective value, so a listing can show
// "on, because the platform says so" apart from "on, for this tenant only".
type FlagSource string

const (
	FlagSourceDefault      FlagSource = "default"
	FlagSourcePlatform     FlagSource = "platform"
	FlagSourceOrganization FlagSource = "organization"
)

// EffectiveFlag is one resolved answer with the reason attached.
type EffectiveFlag struct {
	Key     string     `json:"key"`
	Value   bool       `json:"value"`
	Source  FlagSource `json:"source"`
	Scope   FlagScope  `json:"scope"`
	Default bool       `json:"default"`
	Why     string     `json:"why,omitempty"`
}

// ResolveFlag applies the resolution rule: organization row, then platform row,
// then the compiled default. FIRST HIT WINS, in either direction — that
// override is what a flag is for and what a policy deliberately cannot do.
//
// Both pointers are nil-safe so a caller with no organization (the platform
// surface, or an unauthenticated request) passes nil rather than inventing a
// zero row that would read as "explicitly false".
func ResolveFlag(f Flag, platform *PlatformFeatureFlag, org *OrgFeatureFlag) EffectiveFlag {
	out := EffectiveFlag{
		Key: f.Key, Scope: f.Scope, Default: f.Default, Why: f.Why,
		Value: f.Default, Source: FlagSourceDefault,
	}
	if platform != nil {
		out.Value, out.Source = platform.Value, FlagSourcePlatform
	}
	// Checked last so it wins, and only where the flag admits one. A stored
	// organization row for a platform-scoped flag should be impossible — the
	// write refuses it — but reading is not the place to trust that.
	if org != nil && f.Scope == ScopeTenant {
		out.Value, out.Source = org.Value, FlagSourceOrganization
	}
	return out
}
