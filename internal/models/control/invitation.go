package control

import (
	"time"

	"github.com/google/uuid"
)

// Invitation is an offer of membership in an organization, redeemable by
// whoever can read the mailbox it was sent to.
//
// The token is the whole authorization: accepting is an unauthenticated
// operation by design, because the invitee frequently has no account yet and
// cannot be asked to sign in first. That makes possession of the emailed token
// the only thing standing between a stranger and a membership, which is why
// only its hash is stored, why it expires, and why Email is bound into the row
// rather than taken from whoever presents it.
//
// AcceptedAt and RevokedAt are separate nullable stamps rather than a status
// enum: both are facts with a time, an invitation can only ever move into one
// of them, and a partial index on "still open" reads naturally from the two.
type Invitation struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index:idx_invitations_org_email,priority:1" json:"organization_id"`
	Organization   Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"-"`

	// InvitedByID is kept even after the inviter leaves the organization: the
	// question "who let this account in" has to remain answerable, and a
	// membership row that outlives its provenance cannot answer it.
	InvitedByID uuid.UUID `gorm:"type:uuid;not null;index" json:"invited_by_id"`

	// Email is lowercased on the way in. It is compared against the account that
	// redeems the token, so an invitation for one address can never be spent by
	// another even if the link is forwarded.
	Email string     `gorm:"type:varchar(255);not null;index:idx_invitations_org_email,priority:2" json:"email"`
	Role  MemberRole `gorm:"type:varchar(20);not null;default:'member'" json:"role"`

	TokenHash  string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time  `gorm:"type:timestamptz;not null;index" json:"expires_at"`
	AcceptedAt *time.Time `gorm:"type:timestamptz" json:"accepted_at,omitempty"`
	RevokedAt  *time.Time `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
}

// Open reports whether this invitation can still be redeemed.
func (i *Invitation) Open(now time.Time) bool {
	return i != nil && i.AcceptedAt == nil && i.RevokedAt == nil && i.ExpiresAt.After(now)
}
