package control

import (
	"time"

	"github.com/google/uuid"
)

type DomainCohortFinding struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RegistrableDomain   string    `gorm:"type:varchar(253);not null;uniqueIndex:idx_domain_cohort_findings_registrable_domain" json:"registrable_domain"`
	OrganizationsTotal  int64     `gorm:"not null;default:0" json:"organizations_total"`
	OrganizationsProven int64     `gorm:"not null;default:0" json:"organizations_proven"`
	ProvenRatio         float64   `gorm:"type:double precision;not null;default:0" json:"proven_ratio"`
	Flagged             bool      `gorm:"not null;default:false;index" json:"flagged"`
	EvaluatedAt         time.Time `gorm:"type:timestamptz;not null;default:now()" json:"evaluated_at"`
}

func (DomainCohortFinding) TableName() string { return "domain_cohort_findings" }
