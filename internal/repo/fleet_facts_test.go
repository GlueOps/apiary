package repo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var factsTestSchema = testsupport.SchemaName("apiary_repo_facts_test")

func factsDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, factsTestSchema,
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
	)
}

// seedOnboardedHost seeds the pair the onboarding job later asks about: an agent
// with a heartbeat, and the machine underneath it storing the DIGEST of a raw
// machine id the caller keeps hold of.
func seedOnboardedHost(t *testing.T, db *gorm.DB, hiveID uuid.UUID, seen *time.Time) (raw string, hv tenant.Hypervisor) {
	t.Helper()
	b := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline,
		LastSeenAt: seen, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&b).Error; err != nil {
		t.Fatalf("create bee: %v", err)
	}
	raw = uuid.NewString()
	digest := tenant.MachineIDDigest(raw)
	name := "node-" + raw[:8]
	hv = tenant.Hypervisor{HiveID: &hiveID, Name: &name, Address: "10.0.0.1",
		MachineID: &digest, BeeID: &b.ID}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("create hypervisor: %v", err)
	}
	return raw, hv
}

// The whole reason these three moved out of cmd/worker.go.
//
// Enrolment stores MachineIDDigest(raw); the onboarding job holds the raw
// /etc/machine-id it just read off the host. Comparing them directly matches
// nothing, so Heartbeated answers false for a perfectly healthy host and the
// onboard ends in ErrNoHeartbeat — fleet-wide, for every host, with no error
// anywhere that names the cause. Nothing about the types catches it: both sides
// are strings.
func TestHeartbeatedDigestsTheMachineID(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	hive := fleetSeedHiveIn(t, db)

	before := time.Now().UTC().Add(-time.Minute)
	seen := time.Now().UTC()
	raw, hv := seedOnboardedHost(t, db, hive.ID, &seen)

	ok, err := Heartbeated(ctx, db, raw, before)
	if err != nil {
		t.Fatalf("Heartbeated: %v", err)
	}
	if !ok {
		t.Fatal("a host whose agent has just checked in was reported as never heard from — " +
			"the query is comparing the RAW machine id against the stored digest, which can " +
			"never match, so every onboard would fail with ErrNoHeartbeat")
	}

	// And the digest is not merely being ignored: the stored value itself must
	// not be accepted, or the test above would pass against a query that
	// compared nothing at all.
	if ok, err := Heartbeated(ctx, db, *hv.MachineID, before); err != nil {
		t.Fatalf("Heartbeated: %v", err)
	} else if ok {
		t.Fatal("the already-digested id matched too, so the query is not digesting its " +
			"argument — it would answer true for whatever the caller happens to pass")
	}
}

// The heartbeat has to be proof that THIS attempt worked. An agent left running
// by an earlier attempt is still checking in, so a query that ignores `since`
// reports success the instant the job starts and the install is never verified.
func TestHeartbeatedIgnoresACheckInFromBeforeTheInstall(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	hive := fleetSeedHiveIn(t, db)

	stale := time.Now().UTC().Add(-time.Hour)
	raw, _ := seedOnboardedHost(t, db, hive.ID, &stale)

	ok, err := Heartbeated(ctx, db, raw, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("Heartbeated: %v", err)
	}
	if ok {
		t.Fatal("a check-in from an hour before the install counted as proof the install " +
			"worked — a leftover agent from a previous attempt would make every retry " +
			"report success without installing anything")
	}
}

// A machine with no agent linked to it must not answer true. The join is what
// crosses from the machine to the agent, and a query that lost it would report
// every enrolled machine as heartbeating.
func TestHeartbeatedNeedsAnAgentUnderTheMachine(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	hive := fleetSeedHiveIn(t, db)

	raw := uuid.NewString()
	digest := tenant.MachineIDDigest(raw)
	name := "lonely-" + raw[:8]
	if err := db.Create(&tenant.Hypervisor{HiveID: &hive.ID, Name: &name,
		Address: "10.0.0.9", MachineID: &digest}).Error; err != nil {
		t.Fatalf("create hypervisor: %v", err)
	}

	ok, err := Heartbeated(ctx, db, raw, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Heartbeated: %v", err)
	}
	if ok {
		t.Fatal("a machine with no agent row reported a heartbeat — the join to bees is " +
			"missing, so onboarding would be verified against nothing")
	}
}

// Departure has to clear all three, together. Leaving the node name behind is
// the quiet one: the row still claims to be a node in a cluster it left, and the
// next capacity sync of the OLD hive matches it by name and adopts it back.
func TestMarkFleetlessClearsMembershipNameAndSwarm(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	hive := fleetSeedHiveIn(t, db)

	sw := tenant.Swarm{HiveID: hive.ID, Name: "prod-" + uuid.NewString()[:8]}
	if err := db.Create(&sw).Error; err != nil {
		t.Fatalf("create swarm: %v", err)
	}
	seen := time.Now().UTC()
	_, hv := seedOnboardedHost(t, db, hive.ID, &seen)
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("swarm_id", sw.ID).Error; err != nil {
		t.Fatalf("assign swarm: %v", err)
	}

	if err := MarkFleetless(ctx, db, hv.ID); err != nil {
		t.Fatalf("MarkFleetless: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.HiveID != nil {
		t.Fatalf("the departed host still belongs to hive %s", *got.HiveID)
	}
	if got.SwarmID != nil {
		t.Fatalf("the departed host still belongs to swarm %s — ck_hypervisors_swarm_needs_hive "+
			"exists to make this state impossible, so the update either violated it or the "+
			"column was never cleared", *got.SwarmID)
	}
	if got.Name != nil {
		t.Fatalf("the departed host kept the node name %q, which only means anything inside "+
			"the cluster it just left — the old hive's next capacity sync will match it by "+
			"name and adopt it back", *got.Name)
	}
}

// The guard is the whole function. Recording membership with no node name leaves
// a row the capacity sync cannot match against anything Proxmox reports, so the
// sync treats the node as new and the cluster ends up with two rows for one
// machine — and no write must happen on the way to that refusal.
// A node that has just joined must NOT be schedulable.
//
// Its capacity has not been observed in the new cluster — Proxmox reports zero
// cores for a node the instant it joins — its firewall has not converged, and
// the agent may not have checked in. An operator makes it schedulable
// deliberately, having seen what the sync eventually reported.
//
// The trap this guards is GORM's: Schedulable carries a `default:true` tag, and
// a struct-based write omits the zero value and lets the column default apply,
// so "set it to false" silently stores TRUE. This asserts the stored row, not
// the call.
func TestMarkJoinedLeavesTheNodeUnschedulable(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	to := fleetSeedHiveIn(t, db)

	seen := time.Now().UTC()
	_, hv := seedOnboardedHost(t, db, to.ID, &seen)
	// Explicitly schedulable first, so the test cannot pass because the row
	// happened to be false already.
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Update("schedulable", true).Error; err != nil {
		t.Fatalf("make schedulable: %v", err)
	}
	if err := MarkFleetless(ctx, db, hv.ID); err != nil {
		t.Fatalf("MarkFleetless: %v", err)
	}

	if err := MarkJoined(ctx, db, hv.ID, to.ID, "pve9"); err != nil {
		t.Fatalf("MarkJoined: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Schedulable {
		t.Fatal("a freshly joined node is schedulable — the placement oracle can land " +
			"guests on a host whose capacity has not been observed in this cluster, whose " +
			"firewall has not converged, and whose agent may not have checked in")
	}
	// And membership was still recorded: cordoning must not come at the cost of
	// the thing this function exists to do.
	if got.HiveID == nil || *got.HiveID != to.ID {
		t.Fatalf("hive membership was not recorded: %v", got.HiveID)
	}
	if got.Name == nil || *got.Name != "pve9" {
		t.Fatalf("the node name was not recorded: %v", got.Name)
	}
}

func TestMarkJoinedRefusesAnEmptyNodeName(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	from := fleetSeedHiveIn(t, db)
	to := fleetSeedHiveIn(t, db)

	seen := time.Now().UTC()
	_, hv := seedOnboardedHost(t, db, from.ID, &seen)

	if err := MarkJoined(ctx, db, hv.ID, to.ID, ""); err == nil {
		t.Fatal("membership was recorded with no node name — the next capacity sync cannot " +
			"match that row to any node the API reports, so it adopts the host a second time")
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.HiveID == nil || *got.HiveID != from.ID {
		t.Fatalf("the refusal still moved the host to another hive (%v) — a rejected join "+
			"must leave the row exactly as it found it", got.HiveID)
	}
}

func TestMarkJoinedRecordsHiveAndNodeName(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()
	to := fleetSeedHiveIn(t, db)

	seen := time.Now().UTC()
	_, hv := seedOnboardedHost(t, db, to.ID, &seen)
	if err := MarkFleetless(ctx, db, hv.ID); err != nil {
		t.Fatalf("MarkFleetless: %v", err)
	}

	if err := MarkJoined(ctx, db, hv.ID, to.ID, "pve7"); err != nil {
		t.Fatalf("MarkJoined: %v", err)
	}
	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.HiveID == nil || *got.HiveID != to.ID {
		t.Fatalf("hive membership was not recorded: %v", got.HiveID)
	}
	if got.Name == nil || *got.Name != "pve7" {
		t.Fatalf("the node name was not recorded: %v", got.Name)
	}
}

func fleetSeedHiveIn(t *testing.T, db *gorm.DB) tenant.Hive {
	t.Helper()
	h := tenant.Hive{Name: "hive-" + uuid.NewString()[:8], Provider: "proxmox"}
	if err := db.Create(&h).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}
	return h
}

// THE GAP THIS CLOSES. The capacity sync runs per CLUSTER, so a machine in no
// cluster is never observed by it — and fleetless is exactly the state a machine
// is in when it is about to join one. observed_guests therefore read zero for
// every join candidate, and the arrival gate's guests_present refusal, which
// exists because `pvecm add` discards the joining node's guest configuration,
// could not fire for the only case it covers.
//
// Measured on a five-node lab on 2026-08-27: two fleetless hosts recorded at
// observed_guests=0 while each held two containers.
func TestAFleetlessHostsOwnGuestCountIsRecorded(t *testing.T) {
	db := factsDB(t)

	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	hv := tenant.Hypervisor{BeeID: &bee.ID, Address: "192.0.2.20"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	if err := RecordObservedGuestsIfFleetless(context.Background(), db, bee.ID, 2, false); err != nil {
		t.Fatalf("record: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedGuests != 2 {
		t.Fatalf("observed_guests = %d, want 2 — the gate reads this column", got.ObservedGuests)
	}
	// Stamped, because the gates also refuse on a stale observation. Recording
	// the count and leaving the machine looking never-observed would trade one
	// refusal for another.
	if got.LastSyncedAt == nil {
		t.Fatal("last_synced_at was not stamped, so the observation still reads as never made")
	}
}

// A CLUSTERED machine is the capacity sync's to observe: it sees the whole
// cluster and can tell this host's guests from its peers'. Two writers on one
// column would take turns and flap on every tick where they disagree.
//
// The predicate is in the UPDATE rather than in a read-then-write on purpose, so
// a machine that joins between an agent's report and this write cannot have the
// stale observation land afterwards.
func TestAClusteredHostsGuestCountIsLeftToTheCapacitySync(t *testing.T) {
	db := factsDB(t)

	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	name := "hv1"
	hv := tenant.Hypervisor{BeeID: &bee.ID, HiveID: &hive.ID, Name: &name, ObservedGuests: 7}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	// The host reports that it IS in a cluster, which is what makes this the
	// capacity sync's to observe.
	if err := RecordObservedGuestsIfFleetless(context.Background(), db, bee.ID, 0, true); err != nil {
		t.Fatalf("record: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedGuests != 7 {
		t.Fatalf("observed_guests = %d, want the sync's 7 left alone", got.ObservedGuests)
	}
}

// ASSIGNED IS NOT JOINED, and this is the gap that state opened.
//
// Under convergence a machine is given a hive_id as an intention and joins on
// its own afterwards. In between it is in a hive by the record and in no cluster
// in fact: the per-cluster capacity sync cannot see it, and the old predicate
// skipped it because hive_id was set. Nothing wrote, last_synced_at froze, the
// drain gate read "unknown", and the host could never be departed — observed on
// a lab node stuck behind a 16-minute-old sync that was never going to be
// superseded.
func TestAnAssignedButUnjoinedHostIsStillObservedByItsAgent(t *testing.T) {
	db := factsDB(t)

	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	// hive_id set, but the host reports no cluster: assigned, not joined.
	hv := tenant.Hypervisor{BeeID: &bee.ID, HiveID: &hive.ID, Address: "192.0.2.21"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	if err := RecordObservedGuestsIfFleetless(context.Background(), db, bee.ID, 3, false); err != nil {
		t.Fatalf("record: %v", err)
	}

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ObservedGuests != 3 {
		t.Fatalf("observed_guests = %d, want 3 — nothing else observes this host", got.ObservedGuests)
	}
	if got.LastSyncedAt == nil {
		t.Fatal("last_synced_at was not stamped, so the drain gate reads unknown and departure " +
			"is refused forever")
	}
}

// A machine assigned to a hive before it has a node name must be ADOPTED when
// it appears under that name, not duplicated.
//
// This is the window AssignToHive opens: an arrival records membership as an
// intention, so the row sits in the hive unnamed until the host joins and
// reports what it became. A pool of `hive_id IS NULL` excluded exactly those
// rows, so the sync created a second record — carrying no machine id, no agent
// and none of the onboarding history — and that duplicate then held
// (hive_id, name), which meant the real row could never be named and stayed
// NULL permanently. Observed on a lab node.
func TestRecordingANodeNameSaysSoWhenADuplicateHoldsIt(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()

	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	// The real machine: assigned to the hive, agent attached, not yet named.
	real := tenant.Hypervisor{BeeID: &bee.ID, HiveID: &hive.ID, Address: "192.0.2.30"}
	if err := db.Create(&real).Error; err != nil {
		t.Fatal(err)
	}
	// The duplicate a sync would have created, already holding the name.
	dup := "pve9"
	if err := db.Create(&tenant.Hypervisor{
		HiveID: &hive.ID, Name: &dup, Address: "192.0.2.30",
	}).Error; err != nil {
		t.Fatal(err)
	}

	err := RecordObservedNodeName(ctx, db, bee.ID, "pve9")
	if err == nil {
		t.Fatal("a duplicate machine record was reported as success; it is permanent and " +
			"the real row can never be named while it stands")
	}
	if !strings.Contains(err.Error(), "TWO records") {
		t.Fatalf("err = %v, want it to name the actual problem rather than look like a "+
			"retryable write failure", err)
	}
}

// The ordinary case still works: an unnamed row takes the name its host reports.
func TestANodeNameIsRecordedOnce(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()

	hive := tenant.Hive{Name: "dc-" + uuid.NewString()[:8]}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatal(err)
	}
	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	hv := tenant.Hypervisor{BeeID: &bee.ID, HiveID: &hive.ID, Address: "192.0.2.31"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	if err := RecordObservedNodeName(ctx, db, bee.ID, "pve8"); err != nil {
		t.Fatalf("record: %v", err)
	}
	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name == nil || *got.Name != "pve8" {
		t.Fatalf("name = %v, want pve8", got.Name)
	}

	// A rename must NOT come from a heartbeat: once a machine has a name, the
	// capacity sync and the composition are both keyed on it, and moving it out
	// from under them would strand every reference.
	if err := RecordObservedNodeName(ctx, db, bee.ID, "somethingelse"); err != nil {
		t.Fatalf("second record: %v", err)
	}
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if *got.Name != "pve8" {
		t.Fatalf("name = %q, want the original — a heartbeat must not rename a machine", *got.Name)
	}
}
