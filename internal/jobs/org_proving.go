package jobs

import (
	"context"
	"fmt"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
)

// ProvenDistinctDaysDefault is the default number of DISTINCT UTC calendar days
// on which an organization must have token sessions before it counts as proven.
//
// Two is the floor, not a tuning knob with a lower setting: at one day every
// organization is proven by the very session issued at registration, which
// silently turns the unproven-organization ceiling off. Every entry point here
// refuses a threshold below two rather than quietly accepting it.
const ProvenDistinctDaysDefault = 2

// ErrProvingThresholdTooLow is returned instead of running a sweep that would
// mark every organization proven. A sweep that cannot make a meaningful
// decision must fail, never fall through to "proven".
var ErrProvingThresholdTooLow = fmt.Errorf(
	"jobs: proving threshold must be >= %d distinct UTC days", ProvenDistinctDaysDefault,
)

// ProvingSweepResult reports what one sweep did.
type ProvingSweepResult struct {
	OrganizationsAdvanced int64
	OrganizationsProven   int64
}

// OrganizationIsProven is THE predicate. The admission gate, the cohort audit
// and this sweep must all agree on what proven means; two definitions that
// drift is the failure mode this exists to prevent, so the answer is read from
// the single column the sweep stamps and computed nowhere else.
func OrganizationIsProven(org *control.Organization) bool {
	return org != nil && org.ProvenAt != nil
}

// CountUnprovenOrganizations is the ceiling's counting query, expressed once.
//
// The key is the registrable domain (eTLD+1), never the substring after "@":
// keying on the literal host hands an attacker unlimited free buckets via
// a@x.evilcorp.com / a@y.evilcorp.com. Callers must pass an already-normalised
// eTLD+1; an empty one is an error, because a gate that cannot identify the
// bucket must deny rather than count zero and allow.
//
// It counts only organizations whose OWNER HAS CONFIRMED their address, and
// that qualifier is the difference between a control and a weapon.
//
// The ceiling refuses the next organization for a domain once K unproven ones
// exist. If a merely ASSERTED address were enough to occupy a slot, anyone could
// register three times as a@victim-corp.com, b@ and c@ — addresses they cannot
// read — and every real employee of that company would be refused permanently.
// The ceiling has no time component and nothing clears it, so that refusal is
// forever. A quota keyed on something the attacker simply asserts is a
// denial-of-service tool pointed at your own prospects.
//
// A confirmed address is not assertable: occupying a slot now costs a mailbox at
// that domain, which is exactly the cost the ceiling assumes the attacker had to
// bear. Unconfirmed registrations still create their rows — they are cheap, and
// no tenant database is provisioned for them — they simply do not consume anyone
// else's budget.
func CountUnprovenOrganizations(ctx context.Context, db *gorm.DB, registrableDomain string) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("jobs: CountUnprovenOrganizations requires a database handle")
	}
	if registrableDomain == "" {
		return 0, fmt.Errorf("jobs: CountUnprovenOrganizations requires a registrable domain")
	}

	var count int64
	if err := db.WithContext(ctx).
		Model(&control.Organization{}).
		Where(`registrable_domain = ? AND proven_at IS NULL AND EXISTS (
			SELECT 1 FROM memberships m
			JOIN users u ON u.id = m.user_id
			WHERE m.organization_id = organizations.id
			  AND m.role = 'owner'
			  AND u.email_verified_at IS NOT NULL
		)`, registrableDomain).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count unproven organizations: %w", err)
	}
	return count, nil
}

// provingSweepSQL advances the per-organization proving counters.
//
// Why counters rather than a COUNT(DISTINCT) over token_sessions at read time:
// auth.token_session_cleanup DELETEs expired sessions, so the session history an
// organization would be judged on evaporates. An organization that genuinely
// returned on two days would silently become unproven again once its first
// session expired, and the ceiling would then refuse legitimate domains.
//
// The day is taken in UTC explicitly (AT TIME ZONE 'UTC'), not with a bare
// created_at::date, which would resolve in the connection's TimeZone and let the
// same two sessions count as one day or two depending on who ran the worker.
//
// proving_last_day is the high-water mark: only sessions on a day strictly after
// it are counted, so re-running the sweep can never re-count a day and inflate an
// organization into proven status without it actually returning.
const provingSweepSQL = `
WITH candidate_days AS (
	SELECT
		o.id AS organization_id,
		count(DISTINCT (s.created_at AT TIME ZONE 'UTC')::date) AS new_days,
		max((s.created_at AT TIME ZONE 'UTC')::date) AS last_day
	FROM organizations o
	JOIN token_sessions s ON s.organization_id = o.id
	WHERE o.proven_at IS NULL
		AND (
			o.proving_last_day IS NULL
			OR (s.created_at AT TIME ZONE 'UTC')::date > o.proving_last_day
		)
	GROUP BY o.id
)
UPDATE organizations o
SET
	proving_days_seen = o.proving_days_seen + c.new_days,
	proving_last_day = c.last_day,
	proven_at = CASE
		WHEN o.proving_days_seen + c.new_days >= ? THEN now()
		ELSE o.proven_at
	END,
	updated_at = now()
FROM candidate_days c
WHERE o.id = c.organization_id
RETURNING (o.proven_at IS NOT NULL) AS proven
`

type provingSweepRow struct {
	Proven bool `gorm:"column:proven"`
}

// SweepOrganizationProving advances proving state for every unproven
// organization that has token sessions on a day it has not been credited for
// yet, and stamps proven_at on those that reach the threshold.
//
// It is exported so the admission area can reuse the same definition (and so a
// backfill can call it) rather than restating the rule in a second place.
func SweepOrganizationProving(ctx context.Context, db *gorm.DB, distinctDays int) (ProvingSweepResult, error) {
	var result ProvingSweepResult

	// The threshold is checked before anything else: a sweep asked to prove on
	// one day must refuse whatever else is wrong with the call, because that
	// setting would mark every organization proven and disable the ceiling.
	if distinctDays < ProvenDistinctDaysDefault {
		return result, ErrProvingThresholdTooLow
	}
	if db == nil {
		return result, fmt.Errorf("jobs: SweepOrganizationProving requires a database handle")
	}

	var rows []provingSweepRow
	if err := db.WithContext(ctx).Raw(provingSweepSQL, distinctDays).Scan(&rows).Error; err != nil {
		return result, fmt.Errorf("advance organization proving: %w", err)
	}

	result.OrganizationsAdvanced = int64(len(rows))
	for _, row := range rows {
		// Only rows whose proven_at was NULL are candidates, so a non-null
		// value in the RETURNING (post-update) row is a new proof.
		if row.Proven {
			result.OrganizationsProven++
		}
	}
	return result, nil
}
