package service

import (
	"math"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/pkg/proxmox"
)

func TestMapProxmoxResourcesUsesMaxCPUNotUtilisation(t *testing.T) {
	// The trap this test exists for: ClusterResource has both CPU and MaxCPU.
	// CPU is a 0..1 utilisation ratio, so an idle 32-core node reports about
	// 0.02 — which truncates to zero cores and makes the whole fleet look like it
	// has no capacity at all.
	rs := proxmox.ClusterResources{
		{Type: "node", Node: "hv1", Status: "online", CPU: 0.02, MaxCPU: 32,
			MaxMem: 128 << 30, MaxDisk: 900 << 30},
	}
	rows := MapProxmoxResources(rs)
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].CPUCores != 32 {
		t.Fatalf("want 32 cores from MaxCPU, got %d — CPU is a utilisation ratio, not a count",
			rows[0].CPUCores)
	}
	if rows[0].MemBytes != 128<<30 || rows[0].DiskBytes != 900<<30 {
		t.Errorf("memory and disk pass through as bytes: %+v", rows[0])
	}
}

func TestMapProxmoxResourcesCarriesTemplateAndStatus(t *testing.T) {
	// Both feed the guest count that gates departures, and both are easy to drop:
	// Template is an integer flag rather than a bool, and a stopped guest is
	// still a guest.
	rs := proxmox.ClusterResources{
		{Type: "qemu", Node: "hv1", Status: "stopped", MaxCPU: 4, MaxMem: 8 << 30, Template: 1},
		{Type: "lxc", Node: "hv1", Status: "running", MaxCPU: 1, MaxMem: 1 << 30},
	}
	rows := MapProxmoxResources(rs)
	if !rows[0].Template {
		t.Error("Template=1 must map to true")
	}
	if rows[0].Status != "stopped" {
		t.Errorf("status must pass through, got %q", rows[0].Status)
	}
	if rows[1].Template {
		t.Error("Template=0 must map to false")
	}
}

func TestMapProxmoxResourcesFeedsTheFoldEndToEnd(t *testing.T) {
	// The whole chain the departure gate depends on: Proxmox rows in, guest count
	// out — with a STOPPED guest counted, because that is the case that blocks a
	// cluster join while consuming no capacity at all.
	rs := proxmox.ClusterResources{
		{Type: "node", Node: "hv1", Status: "online", MaxCPU: 16, MaxMem: 64 << 30, MaxDisk: 500 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", MaxCPU: 1, MaxMem: 2 << 30},
		{Type: "storage", Node: "hv1", Status: "available", MaxDisk: 9000 << 30},
	}
	obs := tenant.FoldClusterResources(MapProxmoxResources(rs), nil)["hv1"]
	if obs.Guests != 1 {
		t.Fatalf("a stopped container is still a guest: want 1, got %d", obs.Guests)
	}
	if obs.CPUTotal != 16 || obs.RAMGBTotal != 64 || obs.DiskGBTotal != 500 {
		t.Errorf("node totals wrong: %+v", obs)
	}
	if !obs.Online {
		t.Error("an online node must read as online")
	}
}

func TestMapProxmoxResourcesSkipsNilRows(t *testing.T) {
	rows := MapProxmoxResources(proxmox.ClusterResources{nil, {Type: "node", Node: "hv1"}})
	if len(rows) != 1 {
		t.Fatalf("a nil row must be skipped, not panic or pass through: got %d rows", len(rows))
	}
}

func TestMapProxmoxResourcesCarriesVMID(t *testing.T) {
	// The vmid is how the fold tells our guests from everybody else's. Dropping
	// it makes every guest unmatched, which the fold reads as foreign — safe, but
	// it means our own placements are subtracted twice and the fleet reports
	// capacity it has.
	rs := proxmox.ClusterResources{
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 101, MaxCPU: 4},
		{Type: "lxc", Node: "hv1", Status: "running", VMID: 202, MaxCPU: 1},
		{Type: "node", Node: "hv1", Status: "online", MaxCPU: 32},
	}
	rows := MapProxmoxResources(rs)
	if rows[0].VMID != 101 || rows[1].VMID != 202 {
		t.Fatalf("vmids must pass through: got %d and %d", rows[0].VMID, rows[1].VMID)
	}
	if rows[2].VMID != 0 {
		t.Errorf("a node row has no vmid, got %d", rows[2].VMID)
	}
}

func TestMapProxmoxResourcesSaturatesAnAbsurdVMID(t *testing.T) {
	// Proxmox sends the vmid as a uint64 and the neutral row holds an int.
	// Narrowing by conversion alone WRAPS, and a wrapped id is a small plausible
	// number that could collide with a real vmid in the placement ledger — which
	// would mark a stranger's VM as ours and stop its capacity being subtracted
	// at all. Saturating merely fails to match, so the guest stays foreign.
	rs := proxmox.ClusterResources{
		{Type: "qemu", Node: "hv1", Status: "running", VMID: math.MaxUint64, MaxCPU: 4},
	}
	got := MapProxmoxResources(rs)[0].VMID
	if got < 0 {
		t.Fatalf("a narrowed vmid must never go negative, got %d", got)
	}
	if got != math.MaxInt {
		t.Fatalf("want the id saturated at MaxInt, got %d", got)
	}
	// The property that actually matters: it does not land on a real vmid.
	if _, collides := map[int]struct{}{101: {}, 202: {}}[got]; collides {
		t.Fatal("a narrowed vmid collided with a ledger entry, which would hide a foreign guest")
	}
}

func TestMapProxmoxResourcesFeedsForeignAccountingEndToEnd(t *testing.T) {
	// The whole chain for the bug being fixed: Proxmox rows in, foreign capacity
	// out. One guest is in our ledger, one was made by hand on the same node.
	rs := proxmox.ClusterResources{
		{Type: "node", Node: "hv1", Status: "online", MaxCPU: 32, MaxMem: 128 << 30, MaxDisk: 1000 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 101, MaxCPU: 4, MaxMem: 8 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 900, MaxCPU: 8, MaxMem: 16 << 30},
	}
	obs := tenant.FoldClusterResources(MapProxmoxResources(rs), map[int]struct{}{101: {}})["hv1"]
	if obs.Guests != 2 || obs.CPUUsed != 12 {
		t.Fatalf("the totals stay totals: %+v", obs)
	}
	if obs.ForeignCPUUsed != 8 || obs.ForeignRAMGBUsed != 16 {
		t.Fatalf("only the hand-made guest is foreign: want cpu=8 ram=16, got %d/%d",
			obs.ForeignCPUUsed, obs.ForeignRAMGBUsed)
	}
}
