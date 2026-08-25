package tenant

import (
	"time"

	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
)

// Credential kinds. Kind is deliberately NOT sealed: if federation is the
// strong model and static keys are legacy, the question asked constantly is
// "how much legacy is left?", and that has to be answerable as
//
//	SELECT kind, count(*) FROM credentials
//
// without decrypting anything. Sealing the discriminator alongside the payload
// would make a deployment unable to measure its own migration, alert on
// lingering static secrets, or enforce "this tenant may not add new ones".
const (
	// CredentialKindStaticSecret is a long-lived provider key or token. The
	// sealed payload is the secret itself.
	CredentialKindStaticSecret = "static_secret"

	// CredentialKindAssumedRole is OIDC federation — AWS
	// AssumeRoleWithWebIdentity and equivalents. The sealed payload is
	// CONFIGURATION (role ARN, external ID), not a credential: nothing in it
	// grants access on its own, because the trust lives in the customer's IAM
	// trust policy pointing at our issuer. A database compromise therefore
	// yields no usable credential for these rows, which is the whole argument
	// for preferring them.
	CredentialKindAssumedRole = "assumed_role"
)

// Credential is authentication material — how to act against a target, not
// which target. WHERE lives on Estate, which binds to exactly one Credential.
//
// That split leaves this record purely secret-bearing, which is also the shape
// that survives the move to federation: there the payload is configuration
// rather than a secret, and only the binding remains.
//
// It lives in the TENANT database, so the organization is the database. There is
// deliberately no organization_id column: the connection already establishes it,
// and a column would be a second source of truth that can disagree with the
// first. An earlier index name (uniq_org_provider_scopekind_scope) implied one,
// which is why the name changed rather than the column being added.
//
// Sealing: this is the highest-value data in the system. Seal it with
// utils.Seal and open it with the embedded Sealed.Open, always passing
// utils.AAD("credentials", orgID, ID) as the additional authenticated data.
// The AAD is what binds a ciphertext to the row it belongs to — without it, an
// attacker with database write access can move a credential blob onto another
// row and have it decrypt cleanly.
type Credential struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name string    `gorm:"type:varchar(100);not null;default:''"`

	// Provider is duplicated from the Estate this credential is bound to, and
	// that duplication is deliberate: a credential is created BEFORE the estate
	// that points at it (Estate.CredentialID is NOT NULL), so at creation there
	// is no estate to read it from — and a payload cannot be validated without
	// knowing whose format it is in. The estate repository refuses a binding
	// where the two disagree.
	//
	// There is no uniqueness rule over the target here any more. Under a 1:1
	// binding, two estates in one AWS account and region legitimately need two
	// credentials whose targets are otherwise identical, so UNIQUE (provider,
	// scope) rejected the second outright. That constraint now lives on Estate,
	// where it reads as "one estate per target".
	Provider string `gorm:"type:varchar(50);not null"`

	// Kind discriminates static secrets from federated role assumption. See the
	// constants above for why it stays outside the seal.
	Kind string `gorm:"type:varchar(20);not null;default:'static_secret'"`

	// Fingerprint is a complete SHA-256 digest with ssh fingerprint semantics: it
	// identifies WHICH credential this is, so a rotation can be confirmed and an
	// operator can compare against what their provider console shows, out of
	// band, without either side revealing anything.
	//
	// It is computed over the IDENTIFYING component, never the secret — the AWS
	// access key ID, the Proxmox token id, the role ARN. That is the actual SSH
	// analogy: ssh fingerprints the PUBLIC half, and comparing one is safe
	// precisely because it discloses nothing.
	//
	// Hashing the secret instead would hand anyone holding this database an
	// offline verification oracle: they could confirm a guessed credential with
	// no API call, no provider-side audit entry, no rate limit and no detection.
	// If secret material ever must be fingerprinted, it needs an HMAC keyed from
	// the master key — at which cost the operator can no longer compute it
	// independently, and the property that made this useful is gone.
	//
	// Stored and displayed COMPLETE, in ssh's own encoding:
	//
	//	SHA256:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU
	//
	// which is 43 characters carrying all 256 bits — shorter than the 64 hex
	// characters it replaces, with nothing dropped. There is deliberately no
	// truncated "audit fragment": a comparison is only as strong as what is
	// actually compared, so surfacing a 12-character prefix would make every
	// human check a 48-bit one and reduce the digest to decoration.
	//
	// That is not merely untidy. For CredentialKindAssumedRole the identifying
	// component is a role ARN, and the customer chooses the role NAME — unlike
	// an AWS-assigned access key id, it is grindable. An attacker who can create
	// roles can search for one whose fingerprint shares a short prefix with a
	// target's, then rely on an operator eyeballing the prefix and approving the
	// wrong credential. Comparing the whole digest costs nothing and removes the
	// attack.
	//
	// Not to be confused with integrity: tamper-evidence over the payload is
	// already provided by the AES-GCM auth tag in Sealed, which fails Open if
	// the ciphertext or AAD was altered. This answers identity, not integrity.
	//
	// Deliberately un-indexed and non-unique: a tenant holds a handful of
	// credentials, so a scan is free, and one key legitimately serves several
	// scopes.
	// Identity is the NON-SECRET identifying component, stored in the clear: an
	// AWS access key id, a Proxmox token id like `root@pam!apiary`, a role ARN.
	//
	// It is stored rather than only fingerprinted because a credential has to be
	// USABLE. Every provider API this system calls needs both halves — Proxmox's
	// token auth is `PVEAPIToken=<identity>=<secret>` — and a fingerprint is a
	// one-way hash, so a credential holding only the digest can be compared
	// against a provider console and never actually used. That was the state
	// before this column existed: the capacity sync could unseal the secret and
	// still not authenticate.
	//
	// In the clear is correct rather than merely convenient. This is the value the
	// fingerprint is computed FROM, and the whole argument for fingerprinting it
	// is that disclosing it costs nothing — an access key id or a token id names a
	// credential, it does not authorise anything without the secret. Sealing it
	// would also defeat the out-of-band comparison the fingerprint exists for.
	//
	// The secret stays sealed and is never returned by any endpoint; this is not.
	Identity string `gorm:"type:varchar(255);not null;default:''"`

	Fingerprint string `gorm:"type:varchar(80);not null;default:''"`

	utils.Sealed `gorm:"embedded"`

	CreatedAt time.Time `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()" format:"date-time"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" format:"date-time"`
}
