package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var enrolmentTestSchema = testsupport.SchemaName("apiary_repo_enrolment_test")

// enrolmentDB stands in for the CONTROL database. Deliberately separate from
// fleetDB: the enrolment token is the one fleet row that stayed in control, and
// a fixture holding both sides would let a query cross the boundary and pass
// here while failing against real, separate databases.
func enrolmentDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, enrolmentTestSchema,
		// The organization is what a token resolves to, so the foreign key needs
		// the table. That dependency is the admission boundary showing up in the
		// fixture, which is what it should look like.
		&control.Organization{},
		&control.BeeEnrolmentToken{},
	)
}

func seedOrg(t *testing.T, db *gorm.DB) control.Organization {
	t.Helper()
	org := control.Organization{Name: "org-" + uuid.NewString()[:8]}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	return org
}

func seedToken(t *testing.T, db *gorm.DB, orgID uuid.UUID, maxUses int, ttl time.Duration) string {
	t.Helper()
	raw := uuid.NewString()
	// A person minted this one. The nullable case — a worker minting a token
	// during automated onboarding, where there is no user to name — is covered
	// separately below.
	creator := uuid.New()
	tok := control.BeeEnrolmentToken{
		OrganizationID: orgID, TokenHash: raw, MaxUses: maxUses,
		ExpiresAt: time.Now().UTC().Add(ttl), CreatedByID: &creator,
	}
	if err := db.Create(&tok).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}
	return raw
}

// The token is what tells an unauthenticated caller apart from every other
// tenant, so what it resolves to is the whole of the admission decision.
func TestRedeemTokenReturnsTheAdmittingOrganization(t *testing.T) {
	db := enrolmentDB(t)
	r := NewEnrolmentRepoGorm(db)
	org := seedOrg(t, db)
	token := seedToken(t, db, org.ID, 0, time.Hour)

	got, err := r.RedeemToken(context.Background(), token, time.Now().UTC())
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if got != org.ID {
		t.Fatalf("token admitted into %s, want %s — the organization it resolves to is what "+
			"decides which tenant database the host is then written into", got, org.ID)
	}
}

// The use limit is a limit. Two hosts racing a token with one use left must not
// both get in.
func TestEnrolmentTokenUseLimitHoldsUnderConcurrency(t *testing.T) {
	db := enrolmentDB(t)
	r := NewEnrolmentRepoGorm(db)
	org := seedOrg(t, db)

	// Several rounds, because one round reproduces the unlocked read only
	// sometimes: the window between reading `uses` and incrementing it is
	// microseconds wide. Deterministic when the lock is present — the
	// transactions serialise and exactly one host can win — so the rounds cost
	// nothing when the code is right and are what makes a regression visible.
	const rounds, racers = 8, 8
	for round := range rounds {
		token := seedToken(t, db, org.ID, 1, time.Hour)

		results := make(chan error, racers)
		start := make(chan struct{})
		for range racers {
			go func() {
				<-start
				_, err := r.RedeemToken(context.Background(), token, time.Now().UTC())
				results <- err
			}()
		}
		close(start)

		admitted := 0
		for range racers {
			if err := <-results; err == nil {
				admitted++
			} else if !errors.Is(err, ErrEnrolmentTokenInvalid) {
				t.Fatalf("round %d: unexpected error: %v", round, err)
			}
		}
		if admitted != 1 {
			t.Fatalf("round %d: %d of %d hosts redeemed a single-use token — the use "+
				"count is read without a lock, so the limit is advisory and a leaked token "+
				"admits every machine that races it", round, admitted, racers)
		}
	}
}

// Expired, revoked, used-up and unknown all answer the same way. This endpoint
// is reachable by anything that can route to the control plane, so a caller
// guessing tokens must not learn which of those it hit.
func TestRedeemTokenRefusesEveryClosedShapeIdentically(t *testing.T) {
	db := enrolmentDB(t)
	r := NewEnrolmentRepoGorm(db)
	org := seedOrg(t, db)
	now := time.Now().UTC()

	expired := seedToken(t, db, org.ID, 0, -time.Minute)

	usedUp := seedToken(t, db, org.ID, 1, time.Hour)
	if _, err := r.RedeemToken(context.Background(), usedUp, now); err != nil {
		t.Fatalf("priming the used-up token: %v", err)
	}

	revoked := seedToken(t, db, org.ID, 0, time.Hour)
	if err := db.Model(&control.BeeEnrolmentToken{}).
		Where("token_hash = ?", revoked).
		Update("revoked_at", now).Error; err != nil {
		t.Fatalf("revoke: %v", err)
	}

	for _, c := range []struct{ name, token string }{
		{"unknown", uuid.NewString()},
		{"expired", expired},
		{"used up", usedUp},
		{"revoked", revoked},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := r.RedeemToken(context.Background(), c.token, now)
			if !errors.Is(err, ErrEnrolmentTokenInvalid) {
				t.Fatalf("got %v, want ErrEnrolmentTokenInvalid", err)
			}
			if got != uuid.Nil {
				t.Fatalf("a refused redemption returned organization %s; admitting on a "+
					"rejected token is the one thing this must never do", got)
			}
		})
	}
}

// A token minted by the system carries no creator, and must still redeem.
//
// created_by_id was NOT NULL until 2026-08-27, which is what stopped
// OnboardDeps.EnrolmentToken from ever being wired: a worker is not a person,
// and the alternatives were inventing a service principal or attributing the
// mint to whoever last logged in. An audit column that is sometimes a lie is
// worse than one that is sometimes empty.
func TestRedeemsATokenMintedWithNoCreator(t *testing.T) {
	db := enrolmentDB(t)
	orgID := seedOrg(t, db).ID

	raw := uuid.NewString()
	tok := control.BeeEnrolmentToken{
		OrganizationID: orgID, TokenHash: raw, MaxUses: 1,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
		// No CreatedByID: this is the worker's path.
	}
	if err := db.Create(&tok).Error; err != nil {
		t.Fatalf("a token with no creator must be storable: %v", err)
	}

	got, err := NewEnrolmentRepoGorm(db).RedeemToken(context.Background(), raw, time.Now().UTC())
	if err != nil {
		t.Fatalf("a system-minted token must redeem: %v", err)
	}
	if got != orgID {
		t.Fatalf("redeemed into %s, want %s", got, orgID)
	}
}
