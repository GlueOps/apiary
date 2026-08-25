package tenant

import "strings"

// Guest resource types as the provider reports them.
const (
	ResourceTypeNode = "node"
	ResourceTypeQemu = "qemu"
	ResourceTypeLXC  = "lxc"
)

// ResourceRow is one row of a provider's cluster inventory, reduced to the
// fields a capacity observation needs.
//
// A neutral shape rather than the Proxmox client's own type, so the fold below
// stays a pure function this package can test without a hypervisor and without
// depending on pkg/proxmox. The mapping lives in the sync worker, which is the
// only place that should know a provider's wire format.
type ResourceRow struct {
	// Type is "node", "qemu", "lxc", "storage", …
	Type string
	// Node is which host the row belongs to. For a node row it is the node's own
	// name.
	Node string
	// Status is "running", "stopped", "online", "offline", …
	Status string

	// VMID is the provider's guest identifier, zero on a node or storage row.
	//
	// Carried purely so the fold can tell OUR guests from everybody else's: it is
	// matched against the placement ledger's VMIDs to decide which rows land in
	// the Foreign* totals. Cluster-wide unique in Proxmox, which is why the
	// managed set the fold takes is flat rather than per-node — a guest Proxmox
	// migrated to another host is still recognised as ours wherever it turns up.
	VMID int

	// Allocated capacity. For a node row these are the machine's totals; for a
	// guest row they are what the guest is configured to have.
	CPUCores  int
	MemBytes  uint64
	DiskBytes uint64

	// Template marks a guest template. Templates are counted as guests: they have
	// a configuration file under /etc/pve/nodes, and that is what a cluster join
	// counts when it refuses a node for "already contains virtual guests".
	Template bool
}

// NodeObservation is what a sync learned about one hypervisor.
type NodeObservation struct {
	Name string

	CPUTotal    int
	RAMGBTotal  int
	DiskGBTotal int

	// Used is ALLOCATED, not live utilisation — the sum of what the guests on
	// this node are configured to have.
	//
	// Deliberate, because of what these numbers are for. Hypervisor.Drift
	// subtracts INTENDED usage (the sum of this system's placements, which are
	// allocations) from these. Comparing an allocation against live utilisation
	// would make drift a measurement of how busy the guests are, which is not a
	// question anyone is asking of it, and would report a large negative drift on
	// a healthy idle fleet.
	CPUUsed    int
	RAMGBUsed  int
	DiskGBUsed int

	// Foreign*Used is the part of the above that this system did NOT place: the
	// guests whose VMID is absent from the placement ledger.
	//
	// This is the half a placement decision must subtract, and it exists as a
	// SEPARATE figure rather than by narrowing the totals above. Each guest has
	// to be counted exactly once, and the two halves are counted by different
	// means — ours through the ledger (which is read under the same lock as the
	// insert, so a concurrent placement cannot be missed), everyone else's
	// through the provider's inventory. Subtracting the TOTAL as well would
	// double-subtract our own guests; subtracting neither is the bug this field
	// was added to fix, where twenty hand-made VMs read as an empty node.
	//
	// The totals stay totals because two other readers need them and neither
	// wants the foreign-only figure: Hypervisor.Drift compares observed against
	// intended and is only meaningful over everything, and the fleet view reports
	// what the node is actually carrying.
	ForeignCPUUsed    int
	ForeignRAMGBUsed  int
	ForeignDiskGBUsed int

	// Guests counts every guest on the node — RUNNING, STOPPED AND TEMPLATES.
	//
	// The count, not the usage, and it is a separate field for a reason a usage
	// figure cannot express: a stopped guest consumes no CPU and no memory, so
	// every capacity-derived reading of "is this node empty" says yes while
	// `pvecm add` refuses the node outright. Observed on a real cluster, where a
	// node whose only guests were two stopped containers was rejected by the join
	// precheck.
	//
	// This is what the departure and arrival gates read. A sync that writes the
	// capacity numbers and leaves this at zero reports every node as empty.
	Guests int

	// Online is whether the provider considers the node up.
	Online bool
}

// bytesPerGiB is the divisor. Proxmox reports bytes and operators think in GiB;
// the model's columns are named GB but have always held GiB, so converting by
// 1000^3 here would silently inflate every capacity figure by 7%.
const bytesPerGiB = 1 << 30

// FoldClusterResources reduces a provider's flat inventory into one observation
// per node.
//
// Pure and deterministic: no clock, no database, no network. The sync worker
// fetches and maps; this decides what the numbers mean. That separation is what
// makes the guest-counting rule — the one that gates departures — testable
// without a cluster.
//
// Rows naming a node that has no node row of its own are still folded, so a
// guest on a node the inventory did not describe is counted rather than
// silently dropped. Losing such a guest would report the node as empty, which is
// the failure this whole path exists to prevent.
//
// `managed` is the set of VMIDs this system's placement ledger already accounts
// for. A guest in it is OURS and is left out of the Foreign* totals, because it
// is counted through the ledger instead; a guest outside it is somebody else's
// and is the capacity a placement decision has to subtract. Nil is a legal and
// meaningful argument — no ledger, so every guest is foreign — and it is what
// the pure-arithmetic tests pass.
func FoldClusterResources(rows []ResourceRow, managed map[int]struct{}) map[string]NodeObservation {
	out := make(map[string]NodeObservation)

	get := func(name string) NodeObservation {
		if o, ok := out[name]; ok {
			return o
		}
		return NodeObservation{Name: name}
	}

	for _, r := range rows {
		node := strings.TrimSpace(r.Node)
		if node == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(r.Type)) {
		case ResourceTypeNode:
			o := get(node)
			o.CPUTotal = r.CPUCores
			o.RAMGBTotal = int(r.MemBytes / bytesPerGiB)
			o.DiskGBTotal = int(r.DiskBytes / bytesPerGiB)
			// "online" is the provider's word for a node it can see. Anything else
			// — offline, unknown — is not online, and treating an unrecognised
			// status as up would report a dead node as available.
			o.Online = strings.EqualFold(strings.TrimSpace(r.Status), "online")
			out[node] = o

		case ResourceTypeQemu, ResourceTypeLXC:
			o := get(node)
			// Counted regardless of Status and regardless of Template. This is the
			// line the gates depend on; narrowing it to running guests is the bug
			// it exists to prevent.
			o.Guests++
			ram := int(r.MemBytes / bytesPerGiB)
			disk := int(r.DiskBytes / bytesPerGiB)
			o.CPUUsed += r.CPUCores
			o.RAMGBUsed += ram
			o.DiskGBUsed += disk

			// A guest we cannot recognise as ours is foreign, and that includes a
			// row whose VMID never arrived (zero). The direction matters: an
			// unmatched guest of ours is subtracted twice and the node merely looks
			// fuller than it is, while a foreign guest mistaken for ours is
			// subtracted not at all and the oracle fills a node that is already
			// full. Fail toward under-use.
			if _, ours := managed[r.VMID]; !ours {
				o.ForeignCPUUsed += r.CPUCores
				o.ForeignRAMGBUsed += ram
				o.ForeignDiskGBUsed += disk
			}
			out[node] = o
		}
	}
	return out
}
