package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/jobs"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/glueops/apiary/pkg/proxmox"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var cycleSchema = testsupport.SchemaName("apiary_fleet_cycle")

// TestFleetCycleAgainstLiveClusters drives the WHOLE lifecycle on real metal:
// sync, cordon, the gate refusing twice, depart, arrive, and sync again.
//
// The pieces are each proven separately elsewhere. What this covers is the part
// no unit test can: that they COMPOSE. The gate reads what the sync wrote, the
// departure acts on what the gate allowed, and the arrival lands a host the
// departure left standalone — with a real Proxmox API and real clusters in
// between, where a units mistake or a stale observation shows up as a refusal
// nobody expected rather than as a tidy failing assertion.
//
// It lives in this package rather than beside the workers because internal/service
// imports internal/jobs; the reverse would be an import cycle.
//
// DESTRUCTIVE. It moves a node between two clusters and does not move it back.
func TestFleetCycleAgainstLiveClusters(t *testing.T) {
	env := os.Getenv
	for _, k := range []string{
		"APIARY_CYCLE_SOURCE_API", "APIARY_CYCLE_SOURCE_TOKEN_ID", "APIARY_CYCLE_SOURCE_SECRET",
		"APIARY_CYCLE_DEST_API", "APIARY_CYCLE_DEST_TOKEN_ID", "APIARY_CYCLE_DEST_SECRET",
		"APIARY_CYCLE_MOVING", "APIARY_CYCLE_SOURCE_PEER", "APIARY_CYCLE_DEST_PEER",
		"APIARY_CYCLE_LINK",
	} {
		if env(k) == "" {
			t.Skipf("%s is not set — SKIPPING; the fleet cycle was NOT exercised against real clusters", k)
		}
	}

	movingAddr, movingName := cyclePair(t, env("APIARY_CYCLE_MOVING"))
	sourcePeerAddr, _ := cyclePair(t, env("APIARY_CYCLE_SOURCE_PEER"))
	destPeerAddr, _ := cyclePair(t, env("APIARY_CYCLE_DEST_PEER"))

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	db := testsupport.OpenSchema(t, cycleSchema,
		&tenant.Credential{}, &tenant.Hive{}, &tenant.Hypervisor{}, &tenant.Bee{},
		&tenant.Slot{}, &tenant.Pool{}, &tenant.Placement{})

	orgID := uuid.New()

	// A bound credential neither hive's inventory read will use — this test talks
	// to the API with tokens straight out of the environment, because sealing one
	// needs a master key the lab does not have.
	//
	// It is here because the membership triggers only fire for a hive the control
	// plane could ACTUALLY sync: api_url set and a credential bound, the same
	// predicate the scheduled fan-out filters on and the "sync now" endpoint
	// refuses without. A hive missing one would exercise none of what the
	// depart and arrive sections below assert, and would do it silently.
	cred := tenant.Credential{Provider: "proxmox", Kind: tenant.CredentialKindStaticSecret}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatalf("seed a credential: %v", err)
	}

	source := tenant.Hive{Name: "lab-source", Provider: "proxmox",
		APIURL: env("APIARY_CYCLE_SOURCE_API"), CredentialID: &cred.ID}
	dest := tenant.Hive{Name: "lab-dest", Provider: "proxmox",
		APIURL: env("APIARY_CYCLE_DEST_API"), CredentialID: &cred.ID}
	for _, h := range []*tenant.Hive{&source, &dest} {
		if err := db.Create(h).Error; err != nil {
			t.Fatalf("create hive %s: %v", h.Name, err)
		}
	}

	syncDeps := jobs.CapacitySyncDeps{
		Inventory: func(ctx context.Context, _, hiveID uuid.UUID) ([]tenant.ResourceRow, error) {
			api, id, secret := env("APIARY_CYCLE_SOURCE_API"),
				env("APIARY_CYCLE_SOURCE_TOKEN_ID"), env("APIARY_CYCLE_SOURCE_SECRET")
			if hiveID == dest.ID {
				api, id, secret = env("APIARY_CYCLE_DEST_API"),
					env("APIARY_CYCLE_DEST_TOKEN_ID"), env("APIARY_CYCLE_DEST_SECRET")
			}
			return cycleInventory(ctx, api, id, secret)
		},
		TenantDB: func(context.Context, uuid.UUID) (*gorm.DB, error) { return db, nil },
	}
	hiveNames := map[uuid.UUID]string{source.ID: source.Name, dest.ID: dest.Name}
	// ONE hive per call, named by the caller.
	//
	// This used to sync BOTH hives at every call site, and that is precisely what
	// let the missing membership trigger hide for as long as it did: with
	// everything re-synced by hand after every step, no assertion could tell a hive
	// refreshed because the departure asked for it from one refreshed because the
	// test did. Per-hive is what makes "the source and nothing else" expressible.
	syncHive := func(hiveID uuid.UUID, stage string) {
		t.Helper()
		n, err := jobs.SyncHiveCapacity(ctx, syncDeps, orgID, hiveID)
		if err != nil {
			t.Fatalf("[%s] sync %s: %v", stage, hiveNames[hiveID], err)
		}
		t.Logf("[%s] ok  synced %s: %d node(s)", stage, hiveNames[hiveID], n)
	}

	// ---- 1. SYNC ------------------------------------------------------------
	// Genuine setup, and the only unconditional sync left in this test: nothing
	// has ever observed either cluster, and every gate below reads what this
	// writes.
	syncHive(source.ID, "sync")
	syncHive(dest.ID, "sync")

	moving := cycleHypervisor(t, db, source.ID, movingName)
	if moving.ObservedGuests != 0 {
		t.Fatalf("the node chosen to move has %d guest(s); this cycle needs an empty one",
			moving.ObservedGuests)
	}
	if moving.LastSyncedAt == nil {
		t.Fatal("the sync must stamp last_synced_at, or every gate below reads unknown")
	}

	var all []tenant.Hypervisor
	if err := db.Find(&all).Error; err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, h := range all {
		total += h.ObservedGuests
		t.Logf("    %-6s cpu=%d ram=%dGiB guests=%d schedulable=%v",
			cycleName(h), h.ObservedCPUTotal, h.ObservedRAMGBTotal, h.ObservedGuests, h.Schedulable)
	}
	// A zero everywhere would prove nothing about the counting, which is the one
	// number the whole gate rests on.
	if total == 0 {
		t.Fatal("no guests observed anywhere; the guest count is not being read from the API")
	}

	// Bees exist so the workers can resolve SSH addresses. The agent itself plays
	// no part in a departure or an arrival — both are driven over SSH from the
	// control plane.
	cycleSeedBee(t, db, &moving, movingAddr)
	peer := cycleOtherHypervisor(t, db, source.ID, moving.ID)
	cycleSeedBee(t, db, &peer, sourcePeerAddr)
	destPeer := cycleAnyHypervisor(t, db, dest.ID)
	cycleSeedBee(t, db, &destPeer, destPeerAddr)

	// ---- 2. THE GATE REFUSES AN UNCORDONED NODE -----------------------------
	if rs := cycleGate(t, db, source, moving); !cycleHas(rs, tenant.RefuseNotCordoned) {
		t.Fatalf("[gate] an uncordoned node must be refused, got %v", cycleCodes(rs))
	}
	t.Log("[gate] ok  refused: not cordoned")

	// ---- 3. CORDON ----------------------------------------------------------
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", moving.ID).
		Update("schedulable", false).Error; err != nil {
		t.Fatalf("cordon: %v", err)
	}
	moving = cycleHypervisor(t, db, source.ID, movingName)
	t.Log("[cordon] ok  node cordoned")

	// ---- 4. THE GATE REFUSES A HIVE THAT IS TOO SMALL -----------------------
	// Two nodes against the default minimum of three. This is the refusal that
	// protects the SURVIVOR: removing one of two loses quorum part-way through
	// delnode and leaves the remaining node read-only with guests unmanageable.
	if rs := cycleGate(t, db, source, moving); !cycleHas(rs, tenant.RefuseHiveTooSmall) {
		t.Fatalf("[gate] a 2-node hive must be refused at the default minimum, got %v", cycleCodes(rs))
	}
	t.Log("[gate] ok  refused: hive too small at the default minimum of 3")

	lowerMinimum := func(n int) {
		t.Helper()
		if err := db.Model(&tenant.Hive{}).Where("id = ?", source.ID).
			Update("minimum_nodes", n).Error; err != nil {
			t.Fatalf("lower the minimum: %v", err)
		}
		if err := db.First(&source, "id = ?", source.ID).Error; err != nil {
			t.Fatal(err)
		}
	}

	// The minimum is a floor on what REMAINS: a minimum of two still refuses,
	// because departing one of two leaves one.
	lowerMinimum(2)
	if rs := cycleGate(t, db, source, moving); !cycleHas(rs, tenant.RefuseHiveTooSmall) {
		t.Fatalf("[gate] departing one of two must be refused under a minimum of 2, got %v",
			cycleCodes(rs))
	}
	t.Log("[gate] ok  refused: a minimum of 2 is a floor the departure would break")

	lowerMinimum(1)
	if rs := cycleGate(t, db, source, moving); len(rs) != 0 {
		t.Fatalf("[gate] a cordoned, empty node leaving one behind under a minimum of 1 must "+
			"be allowed, got %v", cycleCodes(rs))
	}
	t.Log("[gate] ok  allowed once the hive's minimum was lowered to 1")

	// ---- 5. DEPART ----------------------------------------------------------
	//
	// triggered records what the operation asked to be re-observed. In production
	// these enqueue a River job; here there is no client, so the test runs the
	// syncs the operation asked for — and only those.
	var triggered []uuid.UUID
	// The DELAYED recorder, which actually waits.
	//
	// Arrival asks for the hive to be re-observed after a delay because Proxmox
	// has not populated /cluster/resources stats for a node at the instant it
	// joins. A harness that recorded the request and synced immediately would
	// reproduce exactly the bug the delay exists to fix — and did: this test
	// reported `cpu=0 ram=0GiB` for a host with four cores until it honoured the
	// wait. Sleeping here is the difference between demonstrating the fix and
	// demonstrating the defect.
	recordDelayedTrigger := func(stage string) func(context.Context, uuid.UUID, uuid.UUID, time.Duration) error {
		return func(_ context.Context, gotOrg, hiveID uuid.UUID, after time.Duration) error {
			if gotOrg != orgID {
				t.Errorf("[%s] the sync must be enqueued for this organization (%s), got %s",
					stage, orgID, gotOrg)
			}
			if after <= 0 {
				t.Errorf("[%s] arrival asked for an immediate sync (%s); Proxmox reports zero "+
					"capacity for a node at the moment it joins", stage, after)
			}
			t.Logf("[%s] ok  re-sync requested in %s", stage, after)
			time.Sleep(after)
			triggered = append(triggered, hiveID)
			return nil
		}
	}

	recordTrigger := func(stage string) func(context.Context, uuid.UUID, uuid.UUID) error {
		return func(_ context.Context, gotOrg, hiveID uuid.UUID) error {
			if gotOrg != orgID {
				t.Errorf("[%s] the sync must be enqueued for this organization (%s), got %s",
					stage, orgID, gotOrg)
			}
			triggered = append(triggered, hiveID)
			return nil
		}
	}

	if err := jobs.Depart(ctx, jobs.DepartDeps{
		FleetKey:  cycleFleetKey(t),
		NewRunner: func(a, u string, k []byte) (jobs.Runner, error) { return jobs.NewSSHRunner(a, u, k) },
		Snapshot: func(ctx context.Context, _, hypID uuid.UUID) (jobs.DepartureSnapshot, error) {
			return jobs.DepartureSnapshotFromDB(ctx, db, hypID)
		},
		MarkFleetless: func(ctx context.Context, _, hypervisorID uuid.UUID) error {
			return db.WithContext(ctx).Model(&tenant.Hypervisor{}).
				Where("id = ?", hypervisorID).
				Updates(map[string]any{"hive_id": nil, "swarm_id": nil, "name": nil}).Error
		},
		EnqueueCapacitySync: recordTrigger("depart"),
	}, jobs.DepartInput{OrganizationID: orgID, HypervisorID: moving.ID}); err != nil {
		t.Fatalf("[depart] %v", err)
	}
	t.Log("[depart] ok  the node left its cluster and was left standalone")

	cycleAssertStandalone(t, ctx, movingAddr)

	var moved tenant.Hypervisor
	if err := db.First(&moved, "id = ?", moving.ID).Error; err != nil {
		t.Fatalf("[depart] the machine record must SURVIVE a departure: %v", err)
	}
	if moved.HiveID != nil {
		t.Fatal("[depart] the machine must be recorded fleetless — a real state, not a transitional lie")
	}
	if moved.Address != movingAddr {
		t.Fatalf("[depart] the address must survive the departure, got %q", moved.Address)
	}
	t.Log("[depart] ok  recorded as belonging to no hive, machine record intact")

	// THE DEPARTURE ASKS FOR THE SOURCE HIVE TO BE RE-OBSERVED, and for nothing
	// else. Placement decides against this database, so without the trigger the
	// source keeps describing a cluster this node has already left for up to a
	// whole sync interval — three minutes by default. The departed host is
	// fleetless now and belongs to nothing there is anything left to observe, so
	// one hive is the whole answer here; the destination is refreshed by the
	// arrival below, because a migration syncs two hives by being two operations.
	if len(triggered) != 1 || triggered[0] != source.ID {
		t.Fatalf("[depart] the departure must trigger exactly one capacity sync, for the source "+
			"hive %s; got %v", source.ID, triggered)
	}

	// Driven BY what the departure recorded, not run beside it. A sync written
	// here unconditionally would leave the assertions below passing with the
	// trigger deleted from Depart, which is the state this section exists to end.
	//
	// The sync CLEARS the departed node's cluster fields rather than deleting the
	// row: same visible result for membership — the source hive is down to one
	// node — without destroying the machine's identity and onboarding history.
	for _, hiveID := range triggered {
		syncHive(hiveID, "depart trigger")
	}
	if n := cycleCount(t, db, source.ID); n != 1 {
		t.Fatalf("[depart] the source hive must be down to 1 node after a sync, got %d", n)
	}
	if err := db.First(&moved, "id = ?", moving.ID).Error; err != nil {
		t.Fatalf("[depart] the sync DELETED the machine record: %v", err)
	}
	t.Log("[depart] ok  the sync cleared the departed node's membership and kept the machine")

	// ---- 6. ARRIVE ----------------------------------------------------------
	triggered = nil
	if err := jobs.Arrive(ctx, jobs.ArriveDeps{
		FleetKey:  cycleFleetKey(t),
		NewRunner: func(a, u string, k []byte) (jobs.Runner, error) { return jobs.NewSSHRunner(a, u, k) },
		Snapshot: func(ctx context.Context, _, hypervisorID, hiveID uuid.UUID) (jobs.ArrivalSnapshot, error) {
			return jobs.ArrivalSnapshotFromDB(ctx, db, hypervisorID, hiveID)
		},
		// repo.MarkJoined itself, not a copy of its body.
		//
		// This used to inline the same three-column update, and that copy drifted
		// the moment production started leaving an arriving node unschedulable:
		// the assertion below would have kept passing on the strength of the
		// cordon applied before departure, while the real function's new
		// behaviour went unexercised against live clusters. The point of this test
		// is that the pieces compose, so it has to run the pieces.
		MarkJoined: func(ctx context.Context, _, hypervisorID, hiveID uuid.UUID, nodeName string) error {
			return repo.MarkJoined(ctx, db.WithContext(ctx), hypervisorID, hiveID, nodeName)
		},
		EnqueueCapacitySyncAfter: recordDelayedTrigger("arrive"),
	}, jobs.ArriveInput{
		OrganizationID: orgID,
		HypervisorID:   moved.ID,
		TargetHiveID:   dest.ID,
		Links:          []tenant.CorosyncLink{{Number: 0, Address: env("APIARY_CYCLE_LINK")}},
	}); err != nil {
		t.Fatalf("[arrive] %v", err)
	}
	t.Log("[arrive] ok  the node joined the destination cluster, quorate")

	if err := db.First(&moved, "id = ?", moved.ID).Error; err != nil {
		t.Fatal(err)
	}
	if moved.HiveID == nil || *moved.HiveID != dest.ID {
		t.Fatal("[arrive] membership must be recorded against the destination hive")
	}
	// The node name is written with the membership. Without it the next sync of
	// the destination cannot match this row and creates a SECOND record for the
	// machine that just joined — which is exactly what the polling loop below
	// would then observe as "2 nodes", passing for the wrong reason.
	if moved.Name == nil || *moved.Name != movingName {
		t.Fatalf("[arrive] the cluster's node name must be recorded with the membership, got %v",
			moved.Name)
	}

	// THE ARRIVAL ASKS FOR THE DESTINATION HIVE, and for nothing else. One hive is
	// the whole answer by construction rather than by luck: tenant.CheckArrival
	// refuses with already_in_hive whenever the joining machine still claims one,
	// so a host that got this far was fleetless and has no source hive to refresh.
	if len(triggered) != 1 || triggered[0] != dest.ID {
		t.Fatalf("[arrive] the arrival must trigger exactly one capacity sync, for the "+
			"destination hive %s; got %v", dest.ID, triggered)
	}

	// ---- 7. THE SYNC SEES THE NEW SHAPE -------------------------------------
	//
	// Polled rather than read once. A node that has just joined is briefly
	// reported by the cluster's own inventory as not-online — pvestatd has not
	// spoken for it yet — and the sync deliberately does not create a row for a
	// node it has no capacity figures for, because a zero-capacity row reads as a
	// FULL host to the placement oracle.
	//
	// So the node appears on a later sync, which is exactly what the schedule
	// does in production. Waiting here reproduces that rather than asserting on a
	// race the real system does not care about.
	//
	// The FIRST pass is the arrival's own trigger, run from what it recorded; the
	// later ones stand in for the scheduled tick. Only the first is a claim about
	// this change — that a membership change asks for an observation without
	// waiting for the timer — and it is the pass that would stop happening if the
	// enqueue were deleted from Arrive.
	deadline := time.Now().Add(2 * time.Minute)
	for pass := 0; ; pass++ {
		if pass == 0 {
			for _, hiveID := range triggered {
				syncHive(hiveID, "arrive trigger")
			}
		} else {
			syncHive(dest.ID, "after arrive")
		}
		if cycleCount(t, db, dest.ID) == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("[arrive] the destination hive must show 2 nodes within 2 minutes, got %d",
				cycleCount(t, db, dest.ID))
		}
		t.Log("[after arrive] the joined node is not reported online yet; waiting for the next sync")
		time.Sleep(10 * time.Second)
	}
	arrived := cycleHypervisor(t, db, dest.ID, movingName)
	if arrived.ID != moved.ID {
		t.Fatalf("[arrive] the destination sync created a NEW row (%s) for a machine already "+
			"in the table (%s); its onboarding history and identity stayed behind",
			arrived.ID, moved.ID)
	}
	if arrived.Schedulable {
		// THE CORDON NOW SURVIVES A MIGRATION, and that is a behaviour change worth
		// stating rather than discovering. It used to be lost because the row was
		// deleted on departure and recreated on arrival, so a node came back
		// accepting work with nobody having said so. The row is the machine now, so
		// the operator's intent travels with it: a host somebody cordoned arrives
		// in its new cluster still cordoned, and is uncordoned deliberately.
		t.Error("[arrive] a cordoned machine must arrive in its new hive still cordoned; the " +
			"cordon is operator intent about the machine, not about the row")
	}
	t.Logf("[arrive] ok  rediscovered in %s: cpu=%d ram=%dGiB guests=%d",
		dest.Name, arrived.ObservedCPUTotal, arrived.ObservedRAMGBTotal, arrived.ObservedGuests)

	t.Log("CYCLE COMPLETE: sync -> refused(not cordoned) -> cordon -> refused(hive too small) -> allowed -> depart -> arrive -> sync")
}

// --- helpers ---------------------------------------------------------------

func cycleInventory(ctx context.Context, api, tokenID, secret string) ([]tenant.ResourceRow, error) {
	client := proxmox.NewClient(strings.TrimRight(api, "/"),
		proxmox.WithAPIToken(tokenID, secret),
		// The lab ships a self-signed certificate, as every un-fronted Proxmox
		// install does. Mirrors Hive.InsecureSkipVerify.
		proxmox.WithInsecureSkipVerify())
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, err
	}
	rs, err := cluster.Resources(ctx)
	if err != nil {
		return nil, err
	}
	return MapProxmoxResources(rs), nil
}

func cycleFleetKey(t *testing.T) func(context.Context, uuid.UUID) (string, []byte, error) {
	t.Helper()
	key, err := os.ReadFile(os.Getenv("HOME") + "/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("read the operator key: %v", err)
	}
	return func(context.Context, uuid.UUID) (string, []byte, error) { return "", key, nil }
}

func cyclePair(t *testing.T, s string) (addr, name string) {
	t.Helper()
	i := strings.Index(s, ",")
	if i < 0 {
		t.Fatalf("want \"addr,nodename\", got %q", s)
	}
	return s[:i], s[i+1:]
}

// cycleSeedBee gives a discovered machine the two things enrolment would have:
// an address to reach it on and an agent row linked to it.
//
// The address goes on the MACHINE, not on the bee — it is how anything reaches
// the box, and the departure and arrival snapshots read it from there.
func cycleSeedBee(t *testing.T, db *gorm.DB, hv *tenant.Hypervisor, addr string) {
	t.Helper()
	now := time.Now().UTC()
	bee := tenant.Bee{
		APIKeyID: uuid.New(),
		Status:   tenant.BeeStatusOnline, LastSeenAt: &now,
	}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatalf("seed bee for %s: %v", cycleName(*hv), err)
	}
	machineID := tenant.MachineIDDigest(uuid.NewString())
	if err := db.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Updates(map[string]any{
			"bee_id": bee.ID, "address": addr, "machine_id": machineID,
		}).Error; err != nil {
		t.Fatalf("link bee to %s: %v", cycleName(*hv), err)
	}
	hv.BeeID = &bee.ID
	hv.Address = addr
	hv.MachineID = &machineID
}

// cycleName renders a machine's node name for a log line. Nullable now: a
// machine in no cluster has none.
func cycleName(h tenant.Hypervisor) string {
	if h.Name == nil {
		return "(unnamed)"
	}
	return *h.Name
}

func cycleHypervisor(t *testing.T, db *gorm.DB, hiveID uuid.UUID, name string) tenant.Hypervisor {
	t.Helper()
	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND name = ?", hiveID, name).Error; err != nil {
		t.Fatalf("hypervisor %q: %v", name, err)
	}
	return hv
}

func cycleOtherHypervisor(t *testing.T, db *gorm.DB, hiveID, notID uuid.UUID) tenant.Hypervisor {
	t.Helper()
	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ? AND id <> ?", hiveID, notID).Error; err != nil {
		t.Fatalf("a peer: %v", err)
	}
	return hv
}

func cycleAnyHypervisor(t *testing.T, db *gorm.DB, hiveID uuid.UUID) tenant.Hypervisor {
	t.Helper()
	var hv tenant.Hypervisor
	if err := db.First(&hv, "hive_id = ?", hiveID).Error; err != nil {
		t.Fatalf("a node: %v", err)
	}
	return hv
}

func cycleCount(t *testing.T, db *gorm.DB, hiveID uuid.UUID) int {
	t.Helper()
	var n int64
	if err := db.Model(&tenant.Hypervisor{}).Where("hive_id = ?", hiveID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return int(n)
}

// cycleRemaining is what the hive keeps once hv leaves, counted the way the
// departure job counts it.
func cycleRemaining(t *testing.T, db *gorm.DB, hive tenant.Hive, hv tenant.Hypervisor) int {
	t.Helper()
	m, err := repo.LoadHiveMembers(context.Background(), db, hive.ID, hive.Name)
	if err != nil {
		t.Fatal(err)
	}
	return m.RemainingAfter(hv.ID)
}

func cycleGate(t *testing.T, db *gorm.DB, hive tenant.Hive, hv tenant.Hypervisor) []tenant.DepartureRefusal {
	t.Helper()
	var held int64
	if err := db.Model(&tenant.Placement{}).
		Where("hypervisor_id = ? AND state IN ?", hv.ID,
			[]string{tenant.PlacementPending, tenant.PlacementActive}).
		Count(&held).Error; err != nil {
		t.Fatal(err)
	}
	drain := tenant.DrainStateAt(time.Now().UTC(), hv, int(held), tenant.DrainThresholds{})

	// The STATE gate, because that is what these lab tests measure: whether a real
	// cluster's observed shape produces the right verdict. The credential
	// disposition is not a fact about the cluster — it is an operator's answer,
	// supplied per request — so including it here would refuse every host and make
	// the ALLOW case, the one that distinguishes a working gate from one that
	// refuses everything, impossible to reach. cycleGateExecuting covers it.
	return tenant.CheckDepartureState(tenant.DepartureRequest{
		Hive: hive, Hypervisor: hv, Drain: drain, HiveNodeCount: cycleCount(t, db, hive.ID),
		RemainingNodes: cycleRemaining(t, db, hive, hv),
	})
}

// cycleGateExecuting is the gate as a departure actually runs it: the state
// rules plus the operator's decision about the cluster secrets this node holds.
func cycleGateExecuting(t *testing.T, db *gorm.DB, hive tenant.Hive, hv tenant.Hypervisor,
	disposition string) []tenant.DepartureRefusal {
	t.Helper()
	var held int64
	if err := db.Model(&tenant.Placement{}).
		Where("hypervisor_id = ? AND state IN ?", hv.ID,
			[]string{tenant.PlacementPending, tenant.PlacementActive}).
		Count(&held).Error; err != nil {
		t.Fatal(err)
	}
	drain := tenant.DrainStateAt(time.Now().UTC(), hv, int(held), tenant.DrainThresholds{})
	return tenant.CheckDeparture(tenant.DepartureRequest{
		Hive: hive, Hypervisor: hv, Drain: drain, HiveNodeCount: cycleCount(t, db, hive.ID),
		RemainingNodes:        cycleRemaining(t, db, hive, hv),
		CredentialDisposition: disposition,
	})
}

func cycleHas(rs []tenant.DepartureRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

func cycleCodes(rs []tenant.DepartureRefusal) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Code)
	}
	return out
}

// cycleAssertStandalone checks the departed host against the MACHINE, not the
// database. The database records what this system believes; the point of an
// end-to-end run is to check that belief against the metal.
func cycleAssertStandalone(t *testing.T, ctx context.Context, addr string) {
	t.Helper()
	key, err := os.ReadFile(os.Getenv("HOME") + "/.ssh/id_ed25519")
	if err != nil {
		t.Fatal(err)
	}
	r, err := jobs.NewSSHRunner(addr, "root", key)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Connect(ctx); err != nil {
		t.Fatalf("connect to the departed host: %v", err)
	}
	defer func() { _ = r.Close() }()

	out, err := r.Run(ctx, "test -f /etc/pve/corosync.conf && echo CLUSTERED || echo STANDALONE")
	if err != nil {
		t.Fatalf("check standalone: %v", err)
	}
	if !strings.Contains(out, "STANDALONE") {
		t.Fatalf("the departed host still has a cluster config: %q", strings.TrimSpace(out))
	}

	// The design premise, checked on the metal after a LEAVE as well as a join.
	out, err = r.Run(ctx, "grep -c ssh- /etc/ssh/authorized_keys.d/apiary 2>/dev/null || echo 0")
	if err != nil {
		t.Fatalf("check the managed key: %v", err)
	}
	if strings.TrimSpace(out) == "0" {
		t.Error("the managed key did not survive the departure")
	}
}
