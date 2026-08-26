package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
)

var upgradeTestSchema = testsupport.SchemaName("apiary_repo_agent_upgrade_test")

func upgradeDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, upgradeTestSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{}, &tenant.FleetTask{})
}

// seedBee writes an agent with a version and a last-seen time.
func seedBee(t *testing.T, db *gorm.DB, version string, lastSeen time.Time) tenant.Bee {
	t.Helper()
	b := tenant.Bee{ID: uuid.New(), AgentVersion: version, LastSeenAt: &lastSeen}
	if err := db.Create(&b).Error; err != nil {
		t.Fatalf("seed bee: %v", err)
	}
	return b
}

func bySkip(got []UpgradeCandidate, b tenant.Bee) UpgradeCandidate {
	for _, c := range got {
		if c.BeeID == b.ID.String() {
			return c
		}
	}
	return UpgradeCandidate{}
}

// TestAgentsAlreadyOnTheTargetVersionAreSkipped.
//
// Reinstalling them restarts every agent in the fleet to reach the state they
// were already in, which is a fleet-wide blip for no change.
func TestAgentsAlreadyOnTheTargetVersionAreSkipped(t *testing.T) {
	db := upgradeDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	current := seedBee(t, db, "v2.0.0", now)
	old := seedBee(t, db, "v1.0.0", now)

	got, err := QueueAgentUpgrades(ctx, db, nil, "v2.0.0", uuid.New(), false, now, 2*time.Minute)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if c := bySkip(got, current); c.Queued || c.Skipped == "" {
		t.Fatalf("an agent already on the target was queued: %+v", c)
	}
	if c := bySkip(got, old); !c.Queued {
		t.Fatalf("an out-of-date agent was not queued: %+v", c)
	}
}

// TestAnOfflineAgentIsNotQueuedForLater.
//
// A task left waiting is collected whenever the host returns, and "whenever" is
// often during the next incident — an agent restarting itself in the middle of
// one is the last thing anybody wants.
func TestAnOfflineAgentIsNotQueuedForLater(t *testing.T) {
	db := upgradeDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	offline := seedBee(t, db, "v1.0.0", now.Add(-time.Hour))

	got, err := QueueAgentUpgrades(ctx, db, nil, "v2.0.0", uuid.New(), false, now, 2*time.Minute)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	c := bySkip(got, offline)
	if c.Queued {
		t.Fatal("an offline agent was handed a task it would collect during whatever comes next")
	}
	if c.Skipped == "" {
		t.Fatal("the skip was silent, so the operator has to work out which agents were missed")
	}

	var n int64
	_ = db.Model(&tenant.FleetTask{}).Count(&n).Error
	if n != 0 {
		t.Fatalf("%d tasks created for an offline fleet", n)
	}
}

// TestForceQueuesAnAgentAlreadyOnTheVersion, for repairing a damaged binary.
func TestForceQueuesAnAgentAlreadyOnTheVersion(t *testing.T) {
	db := upgradeDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	b := seedBee(t, db, "v2.0.0", now)
	got, err := QueueAgentUpgrades(ctx, db, nil, "v2.0.0", uuid.New(), true, now, 2*time.Minute)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if !bySkip(got, b).Queued {
		t.Fatalf("--force did not queue an agent already on the version: %+v", got)
	}
}

// TestAnAgentIsQueuedAtMostOnce.
//
// A bee may hold one task, which is what makes a fleet-wide upgrade rolling per
// host rather than something that needs its own concurrency control.
func TestAnAgentIsQueuedAtMostOnce(t *testing.T) {
	db := upgradeDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	b := seedBee(t, db, "v1.0.0", now)
	if _, err := QueueAgentUpgrades(ctx, db, nil, "v2.0.0", uuid.New(), false, now, 2*time.Minute); err != nil {
		t.Fatalf("first: %v", err)
	}
	got, err := QueueAgentUpgrades(ctx, db, nil, "v2.0.0", uuid.New(), false, now, 2*time.Minute)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if bySkip(got, b).Queued {
		t.Fatal("a second task was queued onto an agent that already held one")
	}

	var n int64
	_ = db.Model(&tenant.FleetTask{}).Count(&n).Error
	if n != 1 {
		t.Fatalf("%d tasks, want 1", n)
	}
}

// TestScopingToAFleetLeavesOtherFleetsAlone, since an upgrade is usually rolled
// one cluster at a time.
func TestScopingToAFleetLeavesOtherFleetsAlone(t *testing.T) {
	db := upgradeDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	hiveA := tenant.Hive{ID: uuid.New(), Name: "a-" + uuid.NewString()[:8], Provider: "proxmox"}
	hiveB := tenant.Hive{ID: uuid.New(), Name: "b-" + uuid.NewString()[:8], Provider: "proxmox"}
	if err := db.Create(&[]tenant.Hive{hiveA, hiveB}).Error; err != nil {
		t.Fatalf("seed hives: %v", err)
	}

	inA := seedBee(t, db, "v1.0.0", now)
	inB := seedBee(t, db, "v1.0.0", now)
	for _, pair := range []struct {
		bee  uuid.UUID
		hive uuid.UUID
	}{{inA.ID, hiveA.ID}, {inB.ID, hiveB.ID}} {
		hv := tenant.Hypervisor{ID: uuid.New(), Address: uuid.NewString()[:8],
			BeeID: &pair.bee, HiveID: &pair.hive}
		if err := db.Create(&hv).Error; err != nil {
			t.Fatalf("seed hypervisor: %v", err)
		}
	}

	got, err := QueueAgentUpgrades(ctx, db, &hiveA.ID, "v2.0.0", uuid.New(), false, now, 2*time.Minute)
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("%d candidates for one fleet, want 1", len(got))
	}
	if got[0].BeeID != inA.ID.String() {
		t.Fatalf("queued the wrong fleet's agent")
	}
}
