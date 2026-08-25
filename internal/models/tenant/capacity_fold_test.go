package tenant

import "testing"

func TestFoldCountsStoppedGuestsAndTemplates(t *testing.T) {
	// The whole reason ObservedGuests is a count and not a usage figure. A node
	// carrying only stopped guests consumes nothing and is rejected by a cluster
	// join all the same.
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 8, MemBytes: 16 << 30, DiskBytes: 200 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", CPUCores: 1, MemBytes: 1 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", CPUCores: 1, MemBytes: 1 << 30},
		{Type: "qemu", Node: "hv1", Status: "stopped", CPUCores: 2, MemBytes: 4 << 30, Template: true},
	}
	obs := FoldClusterResources(rows, nil)["hv1"]
	if obs.Guests != 3 {
		t.Fatalf("stopped guests and templates must all count: want 3, got %d", obs.Guests)
	}
	if obs.CPUTotal != 8 || obs.RAMGBTotal != 16 || obs.DiskGBTotal != 200 {
		t.Errorf("node totals wrong: %+v", obs)
	}
	if obs.CPUUsed != 4 || obs.RAMGBUsed != 6 {
		t.Errorf("allocated usage wrong: cpu=%d ram=%d", obs.CPUUsed, obs.RAMGBUsed)
	}
	if !obs.Online {
		t.Error("an online node must read as online")
	}
}

func TestFoldReportsAnEmptyNodeAsEmpty(t *testing.T) {
	rows := []ResourceRow{
		{Type: "node", Node: "hv2", Status: "online", CPUCores: 4, MemBytes: 8 << 30},
	}
	obs := FoldClusterResources(rows, nil)["hv2"]
	if obs.Guests != 0 {
		t.Fatalf("a node with no guest rows has no guests, got %d", obs.Guests)
	}
}

func TestFoldCountsGuestsOnAnUndescribedNode(t *testing.T) {
	// A guest whose node has no node row of its own must still be counted.
	// Dropping it would report that node as empty, which is exactly the reading
	// that lets a departure dismantle a host running workloads.
	rows := []ResourceRow{
		{Type: "qemu", Node: "hv-ghost", Status: "running", CPUCores: 2, MemBytes: 4 << 30},
	}
	obs := FoldClusterResources(rows, nil)["hv-ghost"]
	if obs.Guests != 1 {
		t.Fatalf("want 1 guest on an undescribed node, got %d", obs.Guests)
	}
	if obs.Online {
		t.Error("a node with no node row must not read as online")
	}
}

func TestFoldIgnoresStorageAndUnnamedRows(t *testing.T) {
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 4},
		{Type: "storage", Node: "hv1", Status: "available", DiskBytes: 900 << 30},
		{Type: "qemu", Node: "", Status: "running", CPUCores: 4},
	}
	obs := FoldClusterResources(rows, nil)["hv1"]
	if obs.Guests != 0 {
		t.Errorf("storage is not a guest, got %d", obs.Guests)
	}
	if obs.DiskGBTotal != 0 {
		t.Errorf("a storage row must not become the node's disk total, got %d", obs.DiskGBTotal)
	}
	if _, ok := FoldClusterResources(rows, nil)[""]; ok {
		t.Error("a row with no node must not produce an observation")
	}
}

func TestFoldTreatsAnUnknownStatusAsNotOnline(t *testing.T) {
	// Treating an unrecognised status as up would report a dead node as
	// available to the placement oracle.
	for _, st := range []string{"offline", "unknown", "", "ONLINE-ish"} {
		rows := []ResourceRow{{Type: "node", Node: "hv1", Status: st, CPUCores: 4}}
		if FoldClusterResources(rows, nil)["hv1"].Online {
			t.Errorf("status %q must not read as online", st)
		}
	}
	rows := []ResourceRow{{Type: "node", Node: "hv1", Status: "ONLINE", CPUCores: 4}}
	if !FoldClusterResources(rows, nil)["hv1"].Online {
		t.Error("status matching is case-insensitive")
	}
}

func TestFoldConvertsInGiBNotGB(t *testing.T) {
	// The columns are named GB and have always held GiB. Converting by 1000^3
	// would inflate every capacity figure by about 7%, which reads as free
	// headroom that is not there.
	rows := []ResourceRow{{Type: "node", Node: "hv1", Status: "online", MemBytes: 64 << 30}}
	if got := FoldClusterResources(rows, nil)["hv1"].RAMGBTotal; got != 64 {
		t.Fatalf("want 64 GiB, got %d", got)
	}
}

func TestFoldSumsGuestsAcrossTypes(t *testing.T) {
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16},
		{Type: "qemu", Node: "hv1", Status: "running", CPUCores: 4},
		{Type: "lxc", Node: "hv1", Status: "running", CPUCores: 2},
		{Type: "qemu", Node: "hv2", Status: "running", CPUCores: 8},
	}
	all := FoldClusterResources(rows, nil)
	if all["hv1"].Guests != 2 || all["hv1"].CPUUsed != 6 {
		t.Errorf("hv1 wrong: %+v", all["hv1"])
	}
	if all["hv2"].Guests != 1 {
		t.Errorf("hv2 wrong: %+v", all["hv2"])
	}
}

func TestFoldCountsOurGuestOnceAndStrangersAsForeign(t *testing.T) {
	// The correction this whole path exists for. Each guest must be counted
	// EXACTLY once when a placement decision is made: ours through the ledger,
	// everybody else's through Foreign*Used. Getting it wrong in one direction
	// double-subtracts our own capacity; getting it wrong in the other counts a
	// hand-created VM zero times and lets the oracle keep filling a full node.
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 32, MemBytes: 128 << 30, DiskBytes: 1000 << 30},
		// Ours: in the ledger.
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 101, CPUCores: 4, MemBytes: 8 << 30, DiskBytes: 50 << 30},
		// A stranger's, created by hand on the same node.
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 900, CPUCores: 8, MemBytes: 16 << 30, DiskBytes: 100 << 30},
	}
	obs := FoldClusterResources(rows, map[int]struct{}{101: {}})

	got := obs["hv1"]
	if got.Guests != 2 {
		t.Fatalf("Guests must stay a TOTAL: want 2, got %d — the departure and arrival gates "+
			"read it, and pvecm refuses a node for a guest we placed just as readily as for "+
			"one we did not", got.Guests)
	}
	if got.CPUUsed != 12 || got.RAMGBUsed != 24 || got.DiskGBUsed != 150 {
		t.Errorf("Used must stay a TOTAL (Drift and the fleet view read it), got "+
			"cpu=%d ram=%d disk=%d", got.CPUUsed, got.RAMGBUsed, got.DiskGBUsed)
	}
	if got.ForeignCPUUsed != 8 || got.ForeignRAMGBUsed != 16 || got.ForeignDiskGBUsed != 100 {
		t.Fatalf("foreign usage must be the stranger's guest ALONE: want 8/16/100, got "+
			"%d/%d/%d — including ours here would double-subtract it, since it is already "+
			"counted through intended usage",
			got.ForeignCPUUsed, got.ForeignRAMGBUsed, got.ForeignDiskGBUsed)
	}
}

func TestFoldTreatsEveryGuestAsForeignWithoutALedger(t *testing.T) {
	// No managed set means nothing has been placed through this system, so every
	// guest on the node belongs to somebody else. This is the reading that fixes
	// the reported bug: a hypervisor full of hand-made VMs must not look empty.
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16, MemBytes: 64 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", VMID: 300, CPUCores: 4, MemBytes: 8 << 30},
		{Type: "lxc", Node: "hv1", Status: "stopped", VMID: 301, CPUCores: 2, MemBytes: 4 << 30},
	}
	got := FoldClusterResources(rows, nil)["hv1"]
	if got.ForeignCPUUsed != 6 || got.ForeignRAMGBUsed != 12 {
		t.Fatalf("every guest is foreign when the ledger is empty: want cpu=6 ram=12, got %d/%d",
			got.ForeignCPUUsed, got.ForeignRAMGBUsed)
	}
	// A stopped guest still holds its allocation on the node, exactly as it still
	// blocks a cluster join.
	if got.Guests != 2 {
		t.Errorf("want 2 guests, got %d", got.Guests)
	}
}

func TestFoldTreatsAGuestWithNoVMIDAsForeign(t *testing.T) {
	// A row whose vmid never arrived cannot be recognised as ours. Counting it as
	// foreign under-reports capacity; counting it as ours would subtract it
	// nothing at all, which is the failure being fixed. Fail toward under-use.
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16, MemBytes: 64 << 30},
		{Type: "qemu", Node: "hv1", Status: "running", CPUCores: 4, MemBytes: 8 << 30},
	}
	// The managed set deliberately contains 0 — proving the fold does not let an
	// absent id match a ledger entry that never existed either.
	got := FoldClusterResources(rows, map[int]struct{}{101: {}})["hv1"]
	if got.ForeignCPUUsed != 4 || got.ForeignRAMGBUsed != 8 {
		t.Fatalf("a guest with no vmid must count as foreign: want cpu=4 ram=8, got %d/%d",
			got.ForeignCPUUsed, got.ForeignRAMGBUsed)
	}
}

func TestFoldMatchesManagedVMIDsAcrossNodes(t *testing.T) {
	// The managed set is FLAT rather than per-node because a Proxmox vmid is
	// unique across the cluster. If Proxmox migrated one of our guests to another
	// host, it must still read as ours there — otherwise it is subtracted as
	// foreign on its new home while still counting as intended, i.e. twice.
	rows := []ResourceRow{
		{Type: "node", Node: "hv1", Status: "online", CPUCores: 16},
		{Type: "node", Node: "hv2", Status: "online", CPUCores: 16},
		{Type: "qemu", Node: "hv2", Status: "running", VMID: 101, CPUCores: 4},
	}
	all := FoldClusterResources(rows, map[int]struct{}{101: {}})
	if all["hv2"].ForeignCPUUsed != 0 {
		t.Fatalf("a managed guest is ours wherever the cluster reports it, got foreign cpu=%d "+
			"on hv2", all["hv2"].ForeignCPUUsed)
	}
	if all["hv2"].CPUUsed != 4 || all["hv2"].Guests != 1 {
		t.Errorf("it is still on hv2 and still counted there: %+v", all["hv2"])
	}
}
