package repo

import (
	"context"
	"errors"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnrolmentRepoGorm redeems enrolment tokens. It is the CONTROL-database half of
// enrolment.
//
// The token has to live in control because the caller redeeming it is
// unauthenticated and has no organization yet — that is the entire point of an
// enrolment token, and it is why this one table cannot follow the rest of the
// fleet into the tenant database. Searching every tenant database for a token
// hash is not a lookup anyone should build.
//
// The split is therefore the boundary the whole fleet-in-tenant design rests on:
// control answers "which tenant is this?", and nothing else about a fleet.
type EnrolmentRepoGorm struct{ db *gorm.DB }

func NewEnrolmentRepoGorm(db *gorm.DB) *EnrolmentRepoGorm { return &EnrolmentRepoGorm{db: db} }

// RedeemToken consumes one use of a token and returns the organization it admits
// the host into.
//
// One transaction with the row locked, because the use count is a limit: two
// hosts redeeming concurrently against a token with one use left must not both
// succeed. The check is made against the row read UNDER the lock, never against
// a value fetched earlier.
func (r *EnrolmentRepoGorm) RedeemToken(ctx context.Context, tokenHash string, now time.Time) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token control.BeeEnrolmentToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// One answer for unknown, expired, revoked and used-up. This is
				// reachable by anything that can route to the control plane, so it
				// must not report which of those a guess hit.
				return ErrEnrolmentTokenInvalid
			}
			return err
		}
		if !token.Open(now) {
			return ErrEnrolmentTokenInvalid
		}
		if err := tx.Model(&control.BeeEnrolmentToken{}).
			Where("id = ?", token.ID).
			Update("uses", gorm.Expr("uses + 1")).Error; err != nil {
			return err
		}
		orgID = token.OrganizationID
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return orgID, nil
}
