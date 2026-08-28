package bee

import (
	"os"
	"path/filepath"
	"strings"
)

// guestConfigDirs are the per-node directories pmxcfs keeps guest configuration
// in. /etc/pve/local is a symlink to this node's own directory, which is the
// point: it is THIS machine's guests, not every guest the cluster knows about.
var guestConfigDirs = []string{
	"/etc/pve/local/qemu-server",
	"/etc/pve/local/lxc",
}

// CountLocalGuests reports how many VMs and containers are registered to this
// host.
//
// READ FROM pmxcfs, not from the Proxmox API. The API is the obvious source and
// the wrong one: it depends on pveproxy, and a host whose pveproxy is broken is
// exactly the host whose state the control plane most needs and can least get.
// One was found on 2026-08-27 serving a failed TLS handshake for hours — missing
// node certificates after a rename — while its agent was heartbeating normally
// the whole time. /etc/pve is mounted whether pveproxy runs or not.
//
// Counted the same way `pvecm add` cares about it: guest CONFIGURATION present
// on this node, stopped guests included, because a join discards the lot.
//
// A missing directory is zero, not an error. A host with no containers has no
// lxc directory, and treating that as a failure to observe would make the
// commonest case unobservable.
func CountLocalGuests() (int, error) {
	total := 0
	for _, dir := range guestConfigDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			// Guest configs are named by VMID. Anything else in here is not a
			// guest, and counting it would refuse a join that is actually fine.
			if base := strings.TrimSuffix(filepath.Base(e.Name()), ".conf"); !isVMID(base) {
				continue
			}
			total++
		}
	}
	return total, nil
}

func isVMID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
