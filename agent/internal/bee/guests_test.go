package bee

import (
	"os"
	"path/filepath"
	"testing"
)

// countIn is CountLocalGuests against a temp root, so the test does not need a
// real /etc/pve.
func countIn(t *testing.T, root string) int {
	t.Helper()
	saved := guestConfigDirs
	guestConfigDirs = []string{
		filepath.Join(root, "qemu-server"),
		filepath.Join(root, "lxc"),
	}
	t.Cleanup(func() { guestConfigDirs = saved })
	n, err := CountLocalGuests()
	if err != nil {
		t.Fatalf("CountLocalGuests: %v", err)
	}
	return n
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# guest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// THE COUNT THE ARRIVAL GATE NEEDS. `pvecm add` discards the joining node's
// guest configuration and refuses a host that has any, stopped ones included, so
// this counts configuration present rather than anything about running state.
func TestLocalGuestsCountsVMsAndContainers(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "qemu-server", "100.conf"))
	write(t, filepath.Join(root, "qemu-server", "101.conf"))
	write(t, filepath.Join(root, "lxc", "200.conf"))

	if n := countIn(t, root); n != 3 {
		t.Fatalf("counted %d guests, want 3", n)
	}
}

// A host with no containers has no lxc directory. Treating that as a failure
// would make the commonest case unobservable, and an unobserved host is one the
// gate cannot protect.
func TestAMissingDirectoryIsZeroRatherThanAnError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "qemu-server", "100.conf"))

	if n := countIn(t, root); n != 1 {
		t.Fatalf("counted %d guests, want 1", n)
	}
}

func TestNoGuestsCountsZero(t *testing.T) {
	if n := countIn(t, t.TempDir()); n != 0 {
		t.Fatalf("counted %d guests on an empty host, want 0", n)
	}
}

// Only VMID-named files are guests. Counting anything else would refuse a join
// that is actually fine — and pmxcfs directories carry other files.
func TestNonGuestFilesAreNotCounted(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "qemu-server", "100.conf"))
	write(t, filepath.Join(root, "qemu-server", "notes.txt"))
	write(t, filepath.Join(root, "lxc", "template.conf"))
	if err := os.MkdirAll(filepath.Join(root, "lxc", "300.conf"), 0o755); err != nil {
		t.Fatal(err)
	}

	if n := countIn(t, root); n != 1 {
		t.Fatalf("counted %d guests, want 1 — only the VMID-named file is a guest", n)
	}
}
