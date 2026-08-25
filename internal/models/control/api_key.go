package control

import (
	"time"

	"github.com/google/uuid"
)

// API key scopes. Scope names the SURFACE a credential may reach, and the three
// are mutually exclusive on purpose:
//
//	user   a person's own key. Carries no organization, so an org-scoped call
//	       must name one with X-Org-Id, and it is honoured only where that user
//	       holds a membership.
//	org    a tenant's key/secret pair. Already bound to one organization, and
//	       admitted everywhere EXCEPT the agent surface — so a human or an
//	       automation cannot present itself as an agent.
//	agent  the same shape as an org key, admitted ONLY on the agent surface — so
//	       a compromised hypervisor cannot turn its credential on the rest of
//	       the tenant's API.
//
// The exclusion runs both ways deliberately. Restricting only the agent would
// leave an org key able to enrol, heartbeat and report as a fleet agent, which
// is a way to forge the fleet's view of itself.
const (
	APIKeyScopeUser  = "user"
	APIKeyScopeOrg   = "org"
	APIKeyScopeAgent = "agent"
)

// APIKeyScopeIsValid is the single definition of a usable scope. An unknown
// scope must never read as "unrestricted": a credential whose surface cannot be
// determined is refused.
func APIKeyScopeIsValid(s string) bool {
	return s == APIKeyScopeUser || s == APIKeyScopeOrg || s == APIKeyScopeAgent
}

// Usable reports whether this key may authenticate at all, before any question
// of which surface it reaches.
func (k *APIKey) Usable(now time.Time) bool {
	if k == nil || k.Revoked || !APIKeyScopeIsValid(k.Scope) {
		return false
	}
	return k.ExpiresAt == nil || k.ExpiresAt.After(now)
}

// RequiresSecret reports whether this shape carries a secret half. User keys are
// a single opaque value; org and agent keys are a key/secret pair, so that the
// identifying half can be shown in a UI and logged while the secret half never
// leaves the moment it was minted.
func (k *APIKey) RequiresSecret() bool {
	return k != nil && (k.Scope == APIKeyScopeOrg || k.Scope == APIKeyScopeAgent)
}

type APIKey struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id" format:"uuid"`
	OrgID       *uuid.UUID `json:"org_id,omitempty" format:"uuid"`
	Scope       string     `gorm:"not null;default:''" json:"scope"`
	Purpose     string     `json:"purpose"`
	IsEphemeral bool       `json:"is_ephemeral"`
	Name        string     `gorm:"not null;default:''" json:"name"`
	KeyHash     string     `gorm:"uniqueIndex;not null" json:"-"`
	SecretHash  *string    `json:"-"`
	UserID      *uuid.UUID `json:"user_id,omitempty" format:"uuid"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" format:"date-time"`
	Revoked     bool       `gorm:"not null;default:false" json:"revoked"`
	Prefix      *string    `json:"prefix,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" format:"date-time"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamptz;column:created_at;not null;default:now()" format:"date-time"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" format:"date-time"`
}
