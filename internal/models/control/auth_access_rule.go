package control

import (
	"time"

	"github.com/google/uuid"
)

const (
	AccessRuleKindEmail  = "email"
	AccessRuleKindDomain = "domain"
)

const (
	AccessRulePurposeAuthenticate = "authenticate"
	AccessRulePurposeCreateOrg    = "create_org"

	// AccessRulePurposeJoinOrg gates ACCEPTING an invitation, which is a
	// different question from either of the two above. A domain may be
	// perfectly entitled to authenticate and still be one the platform does not
	// want inside a customer's organization; conversely a contractor domain that
	// may never mint an organization of its own can reasonably be invited into
	// one. Folding this into create_org would tie a contractor's ability to join
	// to their ability to found, which is the conflation the purpose split
	// exists to prevent.
	AccessRulePurposeJoinOrg = "join_org"
)

const (
	AccessRulePolicyAllow = "allow"
	AccessRulePolicyDeny  = "deny"
)

type AuthAccessRule struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Kind      string    `gorm:"not null;type:varchar(10);uniqueIndex:idx_auth_access_rules_kind_value_purpose,priority:1" json:"kind"`
	Value     string    `gorm:"not null;type:varchar(255);uniqueIndex:idx_auth_access_rules_kind_value_purpose,priority:2" json:"value"`
	Purpose   string    `gorm:"not null;type:varchar(16);default:'authenticate';uniqueIndex:idx_auth_access_rules_kind_value_purpose,priority:3;check:chk_auth_access_rules_purpose,purpose IN ('authenticate','create_org','join_org')" json:"purpose"`
	Policy    string    `gorm:"not null;type:varchar(10);check:chk_auth_access_rules_policy,policy IN ('allow','deny')" json:"policy"`
	Reason    *string   `gorm:"type:text" json:"reason,omitempty"`
	CreatedAt time.Time `gorm:"type:timestamptz;column:created_at;not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;autoUpdateTime;column:updated_at;not null;default:now()" json:"updated_at"`
}
