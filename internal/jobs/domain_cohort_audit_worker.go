package jobs

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CohortAuditOptions tunes the retrospective pass.
//
// MinSample exists so a single organization that has not come back twice yet
// cannot flag its whole domain: with a sample of one, the ratio is 0 or 1 and
// carries no information. ProvenRatio is the proven fraction below which a
// cohort is considered suspicious.
type CohortAuditOptions struct {
	MinSample   int
	ProvenRatio float64
}

// DomainCohortRow is one registrable-domain cohort as read from the database.
type DomainCohortRow struct {
	RegistrableDomain   string `gorm:"column:registrable_domain"`
	OrganizationsTotal  int64  `gorm:"column:organizations_total"`
	OrganizationsProven int64  `gorm:"column:organizations_proven"`
}

// domainCohortSQL groups organizations by the registrable domain stamped at
// admission. count(proven_at) counts non-NULL values only, so proven is defined
// by exactly the column SweepOrganizationProving stamps — the audit never
// re-derives "proven" from token sessions, which is how the two definitions
// would drift apart.
const domainCohortSQL = `
SELECT
	o.registrable_domain AS registrable_domain,
	count(*) AS organizations_total,
	count(o.proven_at) AS organizations_proven
FROM organizations o
WHERE o.registrable_domain IS NOT NULL
	AND o.registrable_domain <> ''
GROUP BY o.registrable_domain
`

type DomainCohortAuditWorker struct {
	river.WorkerDefaults[workflow.DomainCohortAuditArgs]
	db   *gorm.DB
	opts CohortAuditOptions
}

// NewDomainCohortAuditWorker builds the SHADOW-MODE cohort pass.
//
// Shadow mode is a hard requirement, not a default: this worker records and logs
// and blocks nothing, so the thresholds can be tuned against real traffic before
// anything is enforced. There is deliberately no "enforce" flag to set.
func NewDomainCohortAuditWorker(db *gorm.DB, opts CohortAuditOptions) *DomainCohortAuditWorker {
	if db == nil {
		panic("jobs: DomainCohortAuditWorker requires a control database handle")
	}
	if opts.MinSample < 1 {
		panic(fmt.Sprintf("jobs: DomainCohortAuditWorker requires a minimum sample >= 1, got %d", opts.MinSample))
	}
	if opts.ProvenRatio <= 0 || opts.ProvenRatio >= 1 {
		// A ratio of 0 flags nothing and a ratio of 1 flags every cohort with a
		// single unproven organization in it; both make the pass useless, and a
		// useless observer is worse than none because it looks like coverage.
		panic(fmt.Sprintf("jobs: DomainCohortAuditWorker requires a proven ratio in (0,1), got %v", opts.ProvenRatio))
	}
	return &DomainCohortAuditWorker{db: db, opts: opts}
}

func (w *DomainCohortAuditWorker) Work(
	ctx context.Context,
	_ *river.Job[workflow.DomainCohortAuditArgs],
) error {
	var rows []DomainCohortRow
	if err := w.db.WithContext(ctx).Raw(domainCohortSQL).Scan(&rows).Error; err != nil {
		return fmt.Errorf("read domain cohorts: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	findings := EvaluateDomainCohorts(rows, w.opts, time.Now().UTC())

	// The ONLY write this worker performs. It must never touch organizations or
	// auth_access_rules: the moment it does, shadow mode is over and an
	// untuned threshold starts refusing real customers.
	if err := w.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "registrable_domain"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"organizations_total",
				"organizations_proven",
				"proven_ratio",
				"flagged",
				"evaluated_at",
			}),
		}).
		CreateInBatches(&findings, 100).Error; err != nil {
		return fmt.Errorf("record domain cohort findings: %w", err)
	}

	flagged := 0
	for _, finding := range findings {
		if !finding.Flagged {
			continue
		}
		flagged++
		slog.WarnContext(ctx, "domain cohort below proven ratio (SHADOW MODE: nothing was blocked)",
			"registrable_domain", finding.RegistrableDomain,
			"organizations_total", finding.OrganizationsTotal,
			"organizations_proven", finding.OrganizationsProven,
			"proven_ratio", finding.ProvenRatio,
			"min_sample", w.opts.MinSample,
			"threshold_ratio", w.opts.ProvenRatio,
			"enforced", false,
		)
	}

	for _, rollup := range rollupPrivateSuffixes(rows, w.opts) {
		// A private suffix (blogspot.com, and every "free subdomain with mail"
		// provider) hands out eTLD+1s for nothing, so the per-domain ceiling
		// sees each one as a separate paid-for bucket. The rollup is the only
		// place that cluster is visible.
		slog.WarnContext(ctx, "private public-suffix cluster below proven ratio (SHADOW MODE: nothing was blocked)",
			"public_suffix", rollup.Suffix,
			"registrable_domains", rollup.Domains,
			"organizations_total", rollup.OrganizationsTotal,
			"organizations_proven", rollup.OrganizationsProven,
			"proven_ratio", rollup.ProvenRatio,
			"enforced", false,
		)
	}

	log.Printf(
		"[admission.domain_cohort_audit] evaluated %d registrable domains, flagged %d (shadow mode, nothing blocked)",
		len(findings), flagged,
	)
	return nil
}

// EvaluateDomainCohorts is the pure decision, split out so it can be tested
// without a database and so the thresholds are readable in one place.
func EvaluateDomainCohorts(rows []DomainCohortRow, opts CohortAuditOptions, evaluatedAt time.Time) []control.DomainCohortFinding {
	findings := make([]control.DomainCohortFinding, 0, len(rows))
	for _, row := range rows {
		findings = append(findings, evaluateDomainCohort(row, opts, evaluatedAt))
	}
	return findings
}

func evaluateDomainCohort(row DomainCohortRow, opts CohortAuditOptions, evaluatedAt time.Time) control.DomainCohortFinding {
	ratio := provenRatio(row.OrganizationsTotal, row.OrganizationsProven)

	// Below the minimum sample the ratio is noise, so it is recorded but never
	// flagged: a brand new domain with one organization that has not returned
	// yet is the single most common shape of legitimate traffic.
	flagged := row.OrganizationsTotal >= int64(opts.MinSample) && ratio < opts.ProvenRatio

	return control.DomainCohortFinding{
		RegistrableDomain:   row.RegistrableDomain,
		OrganizationsTotal:  row.OrganizationsTotal,
		OrganizationsProven: row.OrganizationsProven,
		ProvenRatio:         ratio,
		Flagged:             flagged,
		EvaluatedAt:         evaluatedAt,
	}
}

func provenRatio(total, proven int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(proven) / float64(total)
}

type suffixRollup struct {
	Suffix              string
	Domains             int
	OrganizationsTotal  int64
	OrganizationsProven int64
	ProvenRatio         float64
}

// publicSuffixOf groups the rollup on the SAME suffix lookup that
// repo.NormalizeEmailIdentity used to key each bucket in the first place. It
// resolves to repo.PublicSuffixOf rather than calling publicsuffix directly so
// there is exactly one public-suffix entry point in the codebase; a second
// direct caller could normalise differently (case, in particular) and split one
// operator's cohort across two rollups.
var publicSuffixOf = repo.PublicSuffixOf

// rollupPrivateSuffixes groups cohorts under a PRIVATE public suffix.
//
// ICANN suffixes are skipped on purpose: rolling every .com cohort together
// says nothing. A private suffix means one operator is handing out those
// eTLD+1s, usually for free, so many distinct buckets under it is one actor,
// not many.
func rollupPrivateSuffixes(rows []DomainCohortRow, opts CohortAuditOptions) []suffixRollup {
	bySuffix := make(map[string]*suffixRollup)

	for _, row := range rows {
		suffix, icann := publicSuffixOf(row.RegistrableDomain)
		if icann || suffix == "" || suffix == row.RegistrableDomain {
			continue
		}
		// An unknown TLD ("foo.internal", "x.test") also comes back as a
		// non-ICANN suffix with a single label. Grouping every domain under a
		// bare TLD says nothing, so only multi-label private suffixes — the
		// actual free-subdomain providers — roll up.
		if !strings.Contains(suffix, ".") {
			continue
		}
		entry, ok := bySuffix[suffix]
		if !ok {
			entry = &suffixRollup{Suffix: suffix}
			bySuffix[suffix] = entry
		}
		entry.Domains++
		entry.OrganizationsTotal += row.OrganizationsTotal
		entry.OrganizationsProven += row.OrganizationsProven
	}

	out := make([]suffixRollup, 0, len(bySuffix))
	for _, entry := range bySuffix {
		entry.ProvenRatio = provenRatio(entry.OrganizationsTotal, entry.OrganizationsProven)
		if entry.Domains < opts.MinSample || entry.ProvenRatio >= opts.ProvenRatio {
			continue
		}
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Suffix < out[j].Suffix })
	return out
}
