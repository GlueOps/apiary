package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/jobs"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var gateSchema = testsupport.SchemaName("apiary_fleet_gate")

// TestDepartureGateAgainstLiveClusters drives the departure gate on observations
// this system actually read from Proxmox.
//
// The cycle test next door proves the gate ALLOWS a departure and then performs
// one. It cannot prove the refusals, because it needs an empty node to get
// anywhere and stops on its own precondition the moment one is not: put a
// container on the node it moves and it fails with "this cycle needs an empty
// one" before CheckDeparture is called at all. So the refusal that matters most
// in production — a host still carrying guests must not leave its cluster — has
// only ever been exercised against rows a test wrote by hand.
//
// That is the gap this closes. Every number here comes from a real
// /cluster/resources read of a real cluster, through the same fold and the same
// gate the workers use.
//
// NON-DESTRUCTIVE, and deliberately so. It reads the provider and writes only to
// its own schema; it never calls Depart, never touches a guest, and never
// changes cluster membership. It is safe to run against a lab that is in the
// middle of something else, which is the only way a refusal test gets run often
// enough to be worth having.
func TestDepartureGateAgainstLiveClusters(t *testing.T) {
	env := os.Getenv
	for _, k := range []string{
		"APIARY_CYCLE_SOURCE_API", "APIARY_CYCLE_SOURCE_TOKEN_ID", "APIARY_CYCLE_SOURCE_SECRET",
	} {
		if env(k) == "" {
			t.Skipf("%s is not set — SKIPPING; the departure gate was NOT exercised against "+
				"real observations", k)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	db := testsupport.OpenSchema(t, gateSchema,
		&tenant.Credential{}, &tenant.Hive{}, &tenant.Hypervisor{}, &tenant.Bee{},
		&tenant.Slot{}, &tenant.Pool{}, &tenant.Placement{})

	orgID := uuid.New()
	cred := tenant.Credential{Provider: "proxmox", Kind: tenant.CredentialKindStaticSecret}
	if err := db.Create(&cred).Error; err != nil {
		t.Fatalf("create credential: %v", err)
	}
	hive := tenant.Hive{
		Name: "lab-gate", Provider: "proxmox",
		APIURL: env("APIARY_CYCLE_SOURCE_API"), CredentialID: &cred.ID,
	}
	if err := db.Create(&hive).Error; err != nil {
		t.Fatalf("create hive: %v", err)
	}

	syncDeps := jobs.CapacitySyncDeps{
		Inventory: func(ctx context.Context, _, _ uuid.UUID) ([]tenant.ResourceRow, error) {
			return cycleInventory(ctx, env("APIARY_CYCLE_SOURCE_API"),
				env("APIARY_CYCLE_SOURCE_TOKEN_ID"), env("APIARY_CYCLE_SOURCE_SECRET"))
		},
		TenantDB: func(context.Context, uuid.UUID) (*gorm.DB, error) { return db, nil },
	}
	n, err := jobs.SyncHiveCapacity(ctx, syncDeps, orgID, hive.ID)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	t.Logf("[sync] ok  observed %d node(s) from the live cluster", n)

	var nodes []tenant.Hypervisor
	if err := db.Where("hive_id = ?", hive.ID).Order("name").Find(&nodes).Error; err != nil {
		t.Fatalf("load nodes: %v", err)
	}
	var withGuests, empty *tenant.Hypervisor
	for i := range nodes {
		h := &nodes[i]
		t.Logf("        %-6s guests=%d cpu=%d ram=%dGiB", cycleName(*h),
			h.ObservedGuests, h.ObservedCPUTotal, h.ObservedRAMGBTotal)
		if h.ObservedGuests > 0 && withGuests == nil {
			withGuests = h
		}
		if h.ObservedGuests == 0 && empty == nil {
			empty = h
		}
	}

	// ---- THE ONE THAT MATTERS ------------------------------------------------
	t.Run("a node carrying guests may not leave", func(t *testing.T) {
		if withGuests == nil {
			t.Skip("no node in this cluster is carrying guests — SKIPPING; the guests_present " +
				"refusal was NOT exercised. Start or create one guest anywhere in the cluster " +
				"to cover it.")
		}
		// Cordoned first, so the refusal under test is the only one left and a
		// pass cannot come from not_cordoned standing in for it.
		hv := *withGuests
		hv.Schedulable = false

		rs := cycleGate(t, db, hive, hv)
		if !cycleHas(rs, tenant.RefuseGuestsPresent) {
			t.Fatalf("[gate] %s reports %d guest(s) and was not refused; got %v — this is the "+
				"refusal that stops a live host being torn out of its cluster",
				cycleName(hv), hv.ObservedGuests, cycleCodes(rs))
		}
		t.Logf("[gate] ok  refused: guests_present (%s, %d guest(s) observed)",
			cycleName(hv), hv.ObservedGuests)

		// STOPPED GUESTS COUNT TOO, and this is the only place that is checked
		// against a provider rather than a fixture. The fold counts every guest
		// regardless of status, because a stopped one blocks a cluster join just as
		// a running one does — so an operator who stops a container and assumes the
		// node is now empty is wrong, and the gate has to keep saying so.
		//
		// Checked for EVERY node rather than only the one chosen above. The chosen
		// node is simply the first with any guests, which is very likely to be one
		// whose guests are all running — so asserting only there would quietly miss
		// the stopped case exactly when the cluster contains one. This is the
		// invariant that matters anyway: what the provider reports and what the
		// fold recorded must agree per node, because a guest the gate cannot see is
		// a guest it cannot protect.
		rows, err := cycleInventory(ctx, env("APIARY_CYCLE_SOURCE_API"),
			env("APIARY_CYCLE_SOURCE_TOKEN_ID"), env("APIARY_CYCLE_SOURCE_SECRET"))
		if err != nil {
			t.Fatalf("re-read inventory: %v", err)
		}
		raw := map[string]int{}
		stopped := map[string]int{}
		for _, r := range rows {
			if r.Type != tenant.ResourceTypeQemu && r.Type != tenant.ResourceTypeLXC {
				continue
			}
			raw[r.Node]++
			if r.Status != "running" {
				stopped[r.Node]++
			}
		}
		totalStopped := 0
		for i := range nodes {
			h := nodes[i]
			name := cycleName(h)
			if raw[name] != h.ObservedGuests {
				t.Errorf("the provider reports %d guest(s) on %s but the fold recorded %d; "+
					"a guest the gate cannot see is a guest it cannot protect",
					raw[name], name, h.ObservedGuests)
			}
			totalStopped += stopped[name]
			if stopped[name] > 0 {
				t.Logf("[gate] ok  %s: %d of %d guest(s) are STOPPED and still counted",
					name, stopped[name], raw[name])
			}
		}
		if totalStopped == 0 {
			t.Log("[gate] note: no stopped guests in this cluster, so the " +
				"stopped-guests-still-count rule was not exercised by this run")
		}
	})

	// ---- the positive control ------------------------------------------------
	//
	// Without this the test could pass by refusing everything, which is the
	// failure mode a gate has: a check that never allows anything looks identical
	// to a check that works, right up until a migration is needed.
	t.Run("an empty cordoned node in a large enough hive is allowed", func(t *testing.T) {
		if empty == nil {
			t.Skip("every node in this cluster is carrying guests — SKIPPING; the ALLOW path " +
				"was not exercised, so this run cannot tell a working gate from one that " +
				"refuses everything")
		}
		hv := *empty
		hv.Schedulable = false
		// Minimum lowered to what this cluster actually has, so the size refusal is
		// not the thing being measured here.
		h := hive
		h.MinimumNodes = len(nodes)

		if rs := cycleGate(t, db, h, hv); len(rs) != 0 {
			t.Fatalf("[gate] %s is empty and cordoned in a hive at its minimum and was still "+
				"refused: %v", cycleName(hv), cycleCodes(rs))
		}
		t.Logf("[gate] ok  allowed: %s (empty, cordoned)", cycleName(hv))

		// The same host, through the gate a real departure uses. A node the
		// cluster is ready to lose still may not leave until somebody has said
		// what happens to the hive's token.cfg, ticket-signing key and root CA —
		// all of which it holds, because /etc/pve/priv is shared by every member.
		if rs := cycleGateExecuting(t, db, h, hv, ""); !cycleHas(rs, tenant.RefuseCredentialsUndecided) {
			t.Fatalf("[gate] executing with no credential decision must refuse; got %v",
				cycleCodes(rs))
		}
		t.Logf("[gate] ok  refused: credentials_undecided (%s, state is ready but nobody has "+
			"said what happens to the secrets it holds)", cycleName(hv))

		for _, d := range []string{tenant.CredentialsRotate, tenant.CredentialsAcceptExposure} {
			if rs := cycleGateExecuting(t, db, h, hv, d); len(rs) != 0 {
				t.Fatalf("[gate] %q is a decision and must be accepted; refused with %v",
					d, cycleCodes(rs))
			}
			t.Logf("[gate] ok  allowed with disposition %q", d)
		}
	})

	// ---- freshness -----------------------------------------------------------
	t.Run("a stale observation refuses rather than guesses", func(t *testing.T) {
		hv := nodes[0]
		hv.Schedulable = false
		old := time.Now().UTC().Add(-24 * time.Hour)
		hv.LastSyncedAt = &old

		rs := cycleGate(t, db, hive, hv)
		if !cycleHas(rs, tenant.RefuseDrainUnknown) {
			t.Fatalf("[gate] a day-old observation was treated as current, got %v; the guest "+
				"count the gate trusts would be a day out of date", cycleCodes(rs))
		}
		t.Logf("[gate] ok  refused: drain_unknown (observation aged 24h)")
	})

	// ---- membership ----------------------------------------------------------
	t.Run("a node in no cluster cannot leave one", func(t *testing.T) {
		hv := nodes[0]
		hv.Schedulable = false
		hv.HiveID = nil

		rs := cycleGate(t, db, hive, hv)
		if !cycleHas(rs, tenant.RefuseNotAMember) {
			t.Fatalf("[gate] a fleetless machine was offered a departure, got %v",
				cycleCodes(rs))
		}
		t.Logf("[gate] ok  refused: not_a_member")
	})
}
