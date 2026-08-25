package repo

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Each suite in this package gets its own schema, private to this test binary
// invocation. These tests previously ran against public, which against a
// developer's own DATABASE_URL means migrating and dropping tables in the real
// control database.
var authFailureTestSchema = testsupport.SchemaName("apiary_repo_auth_failure_test")

func authFailureTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, authFailureTestSchema, &control.AuthFailure{})
}

func uniqueIdentifier(prefix string) string {
	return prefix + "-" + uuid.NewString()
}

func TestRecordFailureWithZeroThresholdNeverLocks(t *testing.T) {
	db := authFailureTestDB(t)
	r := NewAuthFailureRepoGorm(db)
	ctx := context.Background()

	id := uniqueIdentifier("alert-only")
	t.Cleanup(func() { _ = r.Clear(ctx, id, "login_email") })

	now := time.Now().UTC()
	for i := 1; i <= 25; i++ {
		count, locked, err := r.RecordFailure(ctx, id, "login_email", 0, 0, now)
		if err != nil {
			t.Fatalf("RecordFailure %d: %v", i, err)
		}
		if locked {
			t.Fatalf("failure %d locked a zero-threshold counter", i)
		}
		if count != i {
			t.Fatalf("failure %d reported count %d", i, count)
		}
	}

	locked, err := r.IsLocked(ctx, id, "login_email", now)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if locked {
		t.Fatal("a zero-threshold counter reports itself locked")
	}
}

func TestRecordFailureLocksOnTheFirstFailureWhenThresholdIsOne(t *testing.T) {
	db := authFailureTestDB(t)
	r := NewAuthFailureRepoGorm(db)
	ctx := context.Background()

	id := uniqueIdentifier("threshold-one")
	t.Cleanup(func() { _ = r.Clear(ctx, id, "login_ip") })

	now := time.Now().UTC()
	count, locked, err := r.RecordFailure(ctx, id, "login_ip", 1, time.Minute, now)
	if err != nil {
		t.Fatalf("RecordFailure: %v", err)
	}
	if count != 1 {
		t.Fatalf("count %d, want 1", count)
	}
	if !locked {
		t.Fatal("threshold 1 did not lock on the first failure; the create branch ignores the threshold")
	}

	got, err := r.IsLocked(ctx, id, "login_ip", now)
	if err != nil {
		t.Fatalf("IsLocked: %v", err)
	}
	if !got {
		t.Fatal("the lock was reported but not persisted")
	}
}

func TestRecordFailureLocksAtTheThreshold(t *testing.T) {
	db := authFailureTestDB(t)
	r := NewAuthFailureRepoGorm(db)
	ctx := context.Background()

	id := uniqueIdentifier("threshold-ten")
	t.Cleanup(func() { _ = r.Clear(ctx, id, "login_ip") })

	now := time.Now().UTC()
	for i := 1; i <= 10; i++ {
		count, locked, err := r.RecordFailure(ctx, id, "login_ip", 10, 15*time.Minute, now)
		if err != nil {
			t.Fatalf("RecordFailure %d: %v", i, err)
		}
		if count != i {
			t.Fatalf("failure %d reported count %d", i, count)
		}
		if locked != (i >= 10) {
			t.Fatalf("failure %d: locked=%v", i, locked)
		}
	}
}

func TestRecordFailureCountsEveryConcurrentAttempt(t *testing.T) {
	db := authFailureTestDB(t)
	r := NewAuthFailureRepoGorm(db)
	ctx := context.Background()

	id := uniqueIdentifier("concurrent")
	t.Cleanup(func() { _ = r.Clear(ctx, id, "login_ip") })

	const attempts = 12
	now := time.Now().UTC()

	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := r.RecordFailure(ctx, id, "login_ip", 10, 15*time.Minute, now); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent RecordFailure: %v", err)
	}

	var row control.AuthFailure
	if err := db.Where("identifier = ? AND kind = ?", id, "login_ip").First(&row).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if row.Count != attempts {
		t.Fatalf("count %d after %d concurrent failures; attempts were lost to a read-modify-write race", row.Count, attempts)
	}
	if row.LockedUntil == nil || !row.LockedUntil.After(now) {
		t.Fatal("the counter passed its threshold without locking")
	}
}

func TestDeleteStalePrunesOnlyExpiredUnlockedCounters(t *testing.T) {
	db := authFailureTestDB(t)
	r := NewAuthFailureRepoGorm(db)
	ctx := context.Background()

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	stale := uniqueIdentifier("stale")
	fresh := uniqueIdentifier("fresh")
	lockedOld := uniqueIdentifier("locked-old")
	for _, id := range []string{stale, fresh, lockedOld} {
		t.Cleanup(func() { _ = r.Clear(ctx, id, "login_ip") })
	}

	lockHeldUntil := now.Add(10 * time.Minute)
	rows := []control.AuthFailure{
		{Identifier: stale, Kind: "login_ip", Count: 3, WindowStart: now.Add(-48 * time.Hour), LastFailedAt: now.Add(-48 * time.Hour)},
		{Identifier: fresh, Kind: "login_ip", Count: 3, WindowStart: now, LastFailedAt: now},

		{Identifier: lockedOld, Kind: "login_ip", Count: 10, WindowStart: now.Add(-48 * time.Hour), LastFailedAt: now.Add(-48 * time.Hour), LockedUntil: &lockHeldUntil},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	if _, err := r.DeleteStale(ctx, cutoff); err != nil {
		t.Fatalf("DeleteStale: %v", err)
	}

	exists := func(id string) bool {
		var n int64
		if err := db.Model(&control.AuthFailure{}).Where("identifier = ?", id).Count(&n).Error; err != nil {
			t.Fatalf("count: %v", err)
		}
		return n > 0
	}
	if exists(stale) {
		t.Fatal("a stale, unlocked counter survived the prune; auth_failures grows without bound")
	}
	if !exists(fresh) {
		t.Fatal("a counter inside its window was pruned")
	}
	if !exists(lockedOld) {
		t.Fatal("a live lock was pruned out from under the decision that set it")
	}
}
