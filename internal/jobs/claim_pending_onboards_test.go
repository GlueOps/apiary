package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
)

var claimSchema = testsupport.SchemaName("apiary_onboard_claim")

func claimDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, claimSchema, &tenant.Hive{}, &tenant.Hypervisor{}, &tenant.Bee{})
}

// seedMachine writes one hypervisor in a given provisioning state.
//
// Updates after Create, not values on it: ProvisioningState carries a
// `default:'provisioned'` tag, and GORM omits zero values for such fields —
// the same trap that made Schedulable=false store TRUE.
func seedMachine(t *testing.T, db *gorm.DB, state, address string) tenant.Hypervisor {
	t.Helper()
	hv := tenant.Hypervisor{ID: uuid.New(), Address: address}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("seed machine: %v", err)
	}
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Updates(map[string]any{"provisioning_state": state}).Error; err != nil {
		t.Fatalf("seed state: %v", err)
	}
	if err := db.First(&hv, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	return hv
}

func collector() (func(context.Context, river.JobArgs) error, *[]OnboardHypervisorArgs, *sync.Mutex) {
	var mu sync.Mutex
	var got []OnboardHypervisorArgs
	return func(_ context.Context, args river.JobArgs) error {
		mu.Lock()
		defer mu.Unlock()
		if a, ok := args.(OnboardHypervisorArgs); ok {
			got = append(got, a)
		}
		return nil
	}, &got, &mu
}

// TestTheTickClaimsPendingMachines is the behaviour whose absence left every
// redriven machine sitting in `pending` forever while the API said it would be
// claimed on the next tick.
func TestTheTickClaimsPendingMachines(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	org := uuid.New()

	hv := seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")

	insert, got, mu := collector()
	n, err := ClaimPendingOnboards(ctx, db, org, insert)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if n != 1 {
		t.Fatalf("claimed %d, want 1", n)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*got) != 1 {
		t.Fatalf("enqueued %d jobs, want 1", len(*got))
	}
	if (*got)[0].HypervisorID != hv.ID.String() {
		t.Fatalf("enqueued for %q, want %q", (*got)[0].HypervisorID, hv.ID)
	}
	// Empty on purpose: the stored column is a digest and this job wants the raw
	// value, so the host is asked instead.
	if (*got)[0].MachineID != "" {
		t.Fatalf("machine id = %q, want empty so the job reads it off the host", (*got)[0].MachineID)
	}

	var after tenant.Hypervisor
	if err := db.First(&after, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.ProvisioningState != tenant.OnboardProvisioning {
		t.Fatalf("state = %q, want %q", after.ProvisioningState, tenant.OnboardProvisioning)
	}
	if after.ProvisioningLeaseExpiresAt == nil {
		t.Fatal("no lease stamped: without one the row is claimable again immediately, " +
			"which is two workers on one host")
	}
	if after.ProvisioningAttempt != 1 {
		t.Fatalf("attempt = %d, want 1: the counter increments on the CLAIM, or a worker "+
			"that dies before recording anything never reaches the cap",
			after.ProvisioningAttempt)
	}
}

// TestASecondTickDoesNotTakeALeasedMachine is the exclusivity the lease exists
// for: two workers on one host would both install keys and reload sshd.
func TestASecondTickDoesNotTakeALeasedMachine(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	org := uuid.New()
	seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")

	insert, got, mu := collector()
	if _, err := ClaimPendingOnboards(ctx, db, org, insert); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	n, err := ClaimPendingOnboards(ctx, db, org, insert)
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if n != 0 {
		t.Fatalf("second tick claimed %d, want 0 while the lease is live", n)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*got) != 1 {
		t.Fatalf("%d jobs enqueued across two ticks, want 1", len(*got))
	}
}

// TestAnExpiredLeaseIsClaimedAgain is the recovery path for a worker killed
// mid-onboard — the reason the lease is the guard and the state is not.
func TestAnExpiredLeaseIsClaimedAgain(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	org := uuid.New()
	hv := seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")

	insert, _, _ := collector()
	if _, err := ClaimPendingOnboards(ctx, db, org, insert); err != nil {
		t.Fatalf("first tick: %v", err)
	}
	// Backdate the lease rather than sleeping: the rule is "expired", not "old".
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("provisioning_lease_expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire the lease: %v", err)
	}

	n, err := ClaimPendingOnboards(ctx, db, org, insert)
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if n != 1 {
		t.Fatalf("claimed %d after the lease lapsed, want 1: a worker that dies must not "+
			"strand its machine", n)
	}
}

// TestTheTickSkipsMachinesItCannotReach: a row with no address burns an attempt
// per tick forever and can never progress until an operator records one.
func TestTheTickSkipsMachinesItCannotReach(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	seedMachine(t, db, tenant.OnboardPending, "")

	insert, _, _ := collector()
	n, err := ClaimPendingOnboards(ctx, db, uuid.New(), insert)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if n != 0 {
		t.Fatalf("claimed %d, want 0: there is nothing to connect to", n)
	}
}

// TestTheTickStopsAtTheAttemptCap: without it one misconfigured row becomes a
// permanent SSH attempt every tick.
func TestTheTickStopsAtTheAttemptCap(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	hv := seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("provisioning_attempt", tenant.MaxOnboardAttempts).Error; err != nil {
		t.Fatalf("set attempts: %v", err)
	}

	insert, _, _ := collector()
	n, err := ClaimPendingOnboards(ctx, db, uuid.New(), insert)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if n != 0 {
		t.Fatalf("claimed %d at the cap, want 0", n)
	}
}

// TestProvisionedMachinesAreLeftAlone. The tick must not re-onboard the fleet:
// enrolling a live host mints a credential the running agent never reads, which
// 401s it forever.
func TestProvisionedMachinesAreLeftAlone(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	seedMachine(t, db, tenant.OnboardProvisioned, "10.0.0.1")
	seedMachine(t, db, tenant.OnboardFailed, "10.0.0.2")

	insert, _, _ := collector()
	n, err := ClaimPendingOnboards(ctx, db, uuid.New(), insert)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if n != 0 {
		t.Fatalf("claimed %d, want 0: neither a working host nor a terminally failed one "+
			"is the ticker's to retry — redrive is", n)
	}
}

// TestRecordingAnOutcomeConcludesTheRow covers both terminal states and the
// lease release, which is what lets a redrive follow immediately.
func TestRecordingAnOutcomeConcludesTheRow(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		err   error
		state string
		msg   string
	}{
		{"success", nil, tenant.OnboardProvisioned, ""},
		{"failure", errors.New("no route to host"), tenant.OnboardFailed, "no route to host"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hv := seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")
			insert, _, _ := collector()
			if _, err := ClaimPendingOnboards(ctx, db, uuid.New(), insert); err != nil {
				t.Fatalf("claim: %v", err)
			}

			held, err := repo.RecordOnboardOutcome(ctx, db, hv.ID, tc.err)
			if err != nil {
				t.Fatalf("record: %v", err)
			}
			if !held {
				t.Fatal("the lease was reported lost immediately after claiming it")
			}

			var after tenant.Hypervisor
			if err := db.First(&after, "id = ?", hv.ID).Error; err != nil {
				t.Fatalf("reload: %v", err)
			}
			if after.ProvisioningState != tc.state {
				t.Fatalf("state = %q, want %q", after.ProvisioningState, tc.state)
			}
			if after.ProvisioningError != tc.msg {
				t.Fatalf("error = %q, want %q", after.ProvisioningError, tc.msg)
			}
			if after.ProvisioningLeaseExpiresAt != nil {
				t.Fatal("the lease was not released: a terminal row still holding one reads " +
					"as 'a worker is inside this machine' and refuses a redrive")
			}
		})
	}
}

// TestAnOverrunWorkerDoesNotStampAStaleVerdict. Its machine has already been
// taken by a later tick, and writing anyway would report `failed` on a host
// another worker is at that moment onboarding successfully.
func TestAnOverrunWorkerDoesNotStampAStaleVerdict(t *testing.T) {
	db := claimDB(t)
	ctx := context.Background()
	hv := seedMachine(t, db, tenant.OnboardPending, "10.0.0.1")

	insert, _, _ := collector()
	if _, err := ClaimPendingOnboards(ctx, db, uuid.New(), insert); err != nil {
		t.Fatalf("claim: %v", err)
	}
	// The first worker overran: its lease lapsed and a later tick re-claimed.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("provisioning_lease_expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire: %v", err)
	}

	held, err := repo.RecordOnboardOutcome(ctx, db, hv.ID, errors.New("timed out"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if held {
		t.Fatal("a run that lost its lease was allowed to conclude the row")
	}

	var after tenant.Hypervisor
	if err := db.First(&after, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if after.ProvisioningState == tenant.OnboardFailed {
		t.Fatal("a stale verdict was written over a machine that had been re-claimed")
	}
}
