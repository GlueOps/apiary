package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Estate is a real estate being operated on — a hive, in APIary's vocabulary.
// In AWS that is a VPC and region; in Proxmox a datacenter. It is where a
// credential is bound to the thing it is used against.
//
// The binding is 1:1, which is why CredentialID carries a unique index rather
// than a plain one. Without it the schema would permit two estates to share a
// credential, and "which estate is this credential for?" would stop having one
// answer — taking the blast radius of a leaked credential from one estate to an
// unbounded number of them, silently.
type Estate struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name string    `gorm:"type:varchar(100);not null;default:''"`

	// Provider and Scope identify WHICH remote target this estate is. They live
	// here rather than on Credential because this is the record that means
	// "where" — the credential means "how to authenticate", and those are
	// different facts about different things.
	//
	// It was the other way around, and the two constraints contradicted each
	// other: with a 1:1 binding, a prod VPC and a staging VPC in one AWS account
	// and region are two estates needing two credentials, whose scopes are
	// identical — so UNIQUE (provider, scope) on credentials rejected the second
	// outright. Here the same constraint reads as "one estate per target", which
	// is the rule actually wanted.
	//
	// Scope is jsonb because the number of dimensions needed to name a target
	// VARIES BY PROVIDER, and that variation grows with every provider added:
	//
	//   aws      {"account":"111","region":"eu-west-1","vpc":"vpc-abc"}
	//   proxmox  {"cluster":"dc1"}
	//   hetzner  {"project":"prod"}
	//
	// A fixed column set would be wrong for whichever provider comes next, and
	// would mean a migration each time. UNIQUE (provider, scope) works directly
	// on jsonb: equality is by value and ignores key order, so a reordered
	// object is correctly rejected as a duplicate rather than admitted twice.
	//
	// Two things this does NOT give you, both handled above the database:
	//
	//   - Shape validation. {"acount":"111"} is a typo and also a perfectly
	//     valid, distinct scope. Each provider module declares its scope keys;
	//     Postgres cannot know them.
	//   - Canonicality. Some keys are derivable from the credential itself — an
	//     AWS account falls out of sts:GetCallerIdentity — so an operator need
	//     not type them. Derivable is not optional though: omit the account and
	//     two estates in different accounts, same region, collapse to one scope.
	//     Registration fills in what it can derive and refuses when an asserted
	//     value disagrees with the derived one, which catches a production
	//     credential attached to a staging estate.
	Provider string         `gorm:"type:varchar(50);not null;uniqueIndex:uniq_estates_provider_scope,priority:1"`
	Scope    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';uniqueIndex:uniq_estates_provider_scope,priority:2;index:idx_estates_scope,type:gin"`

	// ScopeVersion is the schema version of Scope.
	//
	// Versions must NOT coexist. Adding a key changes the jsonb value, so the
	// same logical target under v1 and v2 are two distinct rows that
	// UNIQUE (provider, scope) will happily accept — reintroducing exactly the
	// duplicate the constraint exists to prevent. A version bump is an eager
	// rewrite of every row, not a gradual drift.
	ScopeVersion int `gorm:"not null;default:1"`

	CredentialID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uniq_estates_credential"`
	Credential   Credential `gorm:"foreignKey:CredentialID;constraint:OnDelete:RESTRICT"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()" format:"date-time"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" format:"date-time"`
}
