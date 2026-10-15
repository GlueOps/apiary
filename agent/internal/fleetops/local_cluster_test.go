package fleetops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// membersJSON is the real shape, taken from a live lab node.
const membersJSON = `{
"nodename": "pve3",
"version": 25,
"cluster": { "name": "lab-b", "version": 9, "nodes": 3, "quorate": 1 },
"nodelist": {
  "pve3": { "id": 1, "online": 1, "ip": "192.168.240.13"},
  "pve5": { "id": 3, "online": 1, "ip": "192.168.240.15"},
  "pve4": { "id": 2, "online": 1, "ip": "192.168.240.14"}
  }
}`

func fixture(t *testing.T, members string) (Cluster, string) {
	t.Helper()
	dir := t.TempDir()
	mp := filepath.Join(dir, ".members")
	if members != "" {
		if err := os.WriteFile(mp, []byte(members), 0o644); err != nil {
			t.Fatalf("write members: %v", err)
		}
	}
	nodes := filepath.Join(dir, "nodes")
	om, on, oc := MembersPath, NodesDir, CorosyncConf
	MembersPath, NodesDir, CorosyncConf = mp, nodes, filepath.Join(dir, "corosync.conf")
	t.Cleanup(func() { MembersPath, NodesDir, CorosyncConf = om, on, oc })
	return NewLocalCluster(&recordingRunner{}), dir
}

// The whole observation comes from one file, with no API, no TLS and no token.
func TestLocalClusterReadsEverythingFromMembers(t *testing.T) {
	cl, _ := fixture(t, membersJSON)
	if err := cl.Status(context.Background()); err != nil {
		t.Fatalf("status: %v", err)
	}
	if got := cl.ClusterName(); got != "lab-b" {
		t.Fatalf("cluster = %q, want lab-b", got)
	}
	if got := cl.NodeName(); got != "pve3" {
		t.Fatalf("node = %q, want pve3", got)
	}
	if !cl.Quorate() {
		t.Fatal("quorate = false, want true")
	}
	if got := cl.MemberCount(); got != 3 {
		t.Fatalf("members = %d, want 3", got)
	}
	// SORTED. Callers derive an eviction owner from the lowest name, and Go
	// randomises map iteration — an unsorted answer would make different members
	// reach different conclusions from identical state.
	got := cl.NodeNames()
	want := []string{"pve3", "pve4", "pve5"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %v, want %v (sorted)", got, want)
		}
	}
}

// A cluster name that is absent means standalone, and it must be reported as a
// SENTINEL rather than as a failed read: the two lead to opposite actions.
func TestLocalClusterReportsStandaloneDistinctlyFromFailure(t *testing.T) {
	cl, _ := fixture(t, `{"nodename":"pve1","nodelist":{"pve1":{"id":1,"online":1}}}`)
	err := cl.Status(context.Background())
	if !isNotClustered(err) {
		t.Fatalf("err = %v, want the not-clustered sentinel", err)
	}
	if cl.ClusterName() != "" {
		t.Fatalf("cluster = %q, want empty", cl.ClusterName())
	}
	// The node name is still readable, which is what a standalone host needs to
	// locate itself in a composition it is about to join.
	if cl.NodeName() != "pve1" {
		t.Fatalf("node = %q, want pve1", cl.NodeName())
	}
}

// pmxcfs not being mounted is a genuine failure, NOT standalone. Reading it as
// standalone would let a host act on a cluster state it never established.
func TestLocalClusterTreatsMissingPmxcfsAsAFailureNotStandalone(t *testing.T) {
	cl, _ := fixture(t, "")
	err := cl.Status(context.Background())
	if err == nil {
		t.Fatal("a missing .members was accepted")
	}
	if isNotClustered(err) {
		t.Fatalf("err = %v, want a failure rather than the standalone sentinel", err)
	}
}

// Guests are enumerated from the replicated configuration, so this answers for
// a node that is powered off — which is exactly the case an eviction cares
// about, and why the API version asked the cluster rather than the node.
func TestGuestsAreReadableForAnotherNode(t *testing.T) {
	cl, dir := fixture(t, membersJSON)
	for _, p := range []string{"nodes/pve4/qemu-server/101.conf", "nodes/pve4/lxc/202.conf"} {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("name: x\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := cl.Status(context.Background()); err != nil {
		t.Fatalf("status: %v", err)
	}

	got, err := cl.GuestsOn(context.Background(), "pve4")
	if err != nil {
		t.Fatalf("guests: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("guests = %+v, want 2", got)
	}
	if got[0].ID != "101" || got[0].Type != "qemu" {
		t.Fatalf("first guest = %+v, want qemu 101", got[0])
	}
	if got[1].ID != "202" || got[1].Type != "lxc" {
		t.Fatalf("second guest = %+v, want lxc 202", got[1])
	}
	// Running state is runtime, not configuration, so it is not replicated and
	// must not be claimed for another node.
	if got[0].Status != "" {
		t.Fatalf("status = %q for another node, want unknown", got[0].Status)
	}
}

// A node with no directory has no guests. Not an error: a node that has just
// joined legitimately has none yet.
func TestGuestsOnAnUnknownNodeIsEmptyNotAnError(t *testing.T) {
	cl, _ := fixture(t, membersJSON)
	got, err := cl.GuestsOn(context.Background(), "pve9")
	if err != nil {
		t.Fatalf("guests: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("guests = %+v, want none", got)
	}
}

// pvecm add's exit code is not the outcome: it exits 0 when its join task fails.
// Live, "unable to copy ssh ID: Permission denied" was followed by exit 0 and the
// agent reported the join accepted. A join that took effect leaves a corosync
// configuration on this host; one that failed before doing anything does not.
func TestAJoinThatLeftNoCorosyncConfigIsAFailure(t *testing.T) {
	cl, dir := fixture(t, "")
	opts := &proxmoxJoinOptions{Hostname: "10.0.0.1", Fingerprint: "AA:BB"}

	if _, err := cl.JoinCluster(context.Background(), opts); err == nil {
		t.Fatal("a join that left no corosync configuration was reported as accepted")
	}

	if err := os.WriteFile(filepath.Join(dir, "corosync.conf"), []byte("totem {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cl.JoinCluster(context.Background(), opts); err != nil {
		t.Fatalf("a join that took effect was refused: %v", err)
	}
}

// Link 0 reaches pvecm as --link0, so the cluster is formed and joined on the
// address the control plane registered rather than whatever the node name
// resolves to.
func TestLink0IsPassedToPvecm(t *testing.T) {
	r := &recordingRunner{}
	dir := t.TempDir()
	oc := CorosyncConf
	CorosyncConf = filepath.Join(dir, "corosync.conf")
	t.Cleanup(func() { CorosyncConf = oc })
	if err := os.WriteFile(CorosyncConf, []byte("totem {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cl := NewLocalCluster(r)

	_, _ = cl.JoinCluster(context.Background(), &proxmoxJoinOptions{
		Hostname: "10.0.0.1", Fingerprint: "AA:BB", Link0: "10.0.0.7"})
	_, _ = cl.CreateCluster(context.Background(), &proxmoxCreateOptions{
		ClusterName: "lab-a", Link0: "10.0.0.7"})

	for i, c := range r.cmds {
		if !strings.Contains(strings.Join(c, " "), "--link0 10.0.0.7") {
			t.Fatalf("command %d = %v, want --link0 10.0.0.7", i, c)
		}
	}
	if len(r.cmds) != 2 {
		t.Fatalf("ran %d commands, want 2", len(r.cmds))
	}
}
