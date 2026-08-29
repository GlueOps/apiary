package fleetops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KnownHostsPath is the file pvecm's ssh-copy-id consults. A var so tests can
// point it at a temporary tree.
var KnownHostsPath = "/root/.ssh/known_hosts"

// pmxcfsPrefix is the mount whose contents are cluster-replicated. A var so the
// guard can be tested without an /etc/pve on the machine running the suite.
var pmxcfsPrefix = "/etc/pve/"

// PinPeer writes a peer's host keys into this host's known_hosts.
//
// WITHOUT THIS EVERY JOIN FAILS. `pvecm add <peer> --use_ssh` runs ssh-copy-id
// at the peer as its first real step; if this host has never seen that peer's
// host key, verification fails and pvecm reports the entire operation as:
//
//	unable to copy ssh ID: exit code 1
//
// That is the whole message. It names neither the host, nor the key, nor
// verification. A freshly provisioned host has no known_hosts at all, so an
// automated join hits it on every node.
//
// The keys come from the published composition, which means they reached this
// host over its authenticated control-plane session rather than from whatever
// answers on the peer's address. That distinction is the point:
// StrictHostKeyChecking=no and ssh-keyscan both produce a working join and both
// hand the corosync auth key and cluster configuration to an impostor. A join is
// the operation where the peer's identity matters most.
//
// Written under EVERY name the join may use — the address, and the node name —
// because known_hosts is keyed on the string used to connect, and pvecm resolves
// node names internally as well as using the address it was given.
//
// Idempotent: existing entries for these names are replaced, so a retry does not
// accumulate duplicates and a peer whose key legitimately changed is re-pinned
// rather than left with two conflicting entries.
func PinPeer(_ context.Context, names []string, hostKeys []string) error {
	var wanted []string
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			wanted = append(wanted, n)
		}
	}
	if len(wanted) == 0 || len(hostKeys) == 0 {
		return fmt.Errorf("refusing to pin a peer with no names or no host keys; the join would " +
			"have to trust whatever answers on its address")
	}

	// PVE OWNS THIS NAMESPACE, so check what the path actually is before writing
	// through it. Observed on a live fleet:
	//
	//	clustered:  /etc/ssh/ssh_known_hosts -> /etc/pve/priv/known_hosts
	//	            /root/.ssh/known_hosts   does not exist
	//	standalone: /root/.ssh/known_hosts   a real local file
	//	            /etc/ssh/ssh_known_hosts absent
	//
	// The system-wide file is NOT outside PVE's boundary — pve symlinks it into
	// pmxcfs, exactly as it does /root/.ssh/authorized_keys. Writing through such
	// a link is a cluster-wide, quorum-gated write to a store that a join
	// replaces wholesale, and this is called on a host that is about to join.
	//
	// A joiner is always standalone, which is the state where the user file is
	// real and local — so the ordinary path is safe. This refuses the case where
	// it is not, rather than relying on that remaining true.
	if err := refuseIfPVEOwned(KnownHostsPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(KnownHostsPath), 0o700); err != nil {
		return fmt.Errorf("create the ssh directory: %w", err)
	}
	existing, err := os.ReadFile(KnownHostsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", KnownHostsPath, err)
	}

	// Drop any line already naming one of these hosts, so a re-pin replaces
	// rather than appends. A stale entry left beside a fresh one makes ssh refuse
	// on a conflict, which is the same failure this exists to prevent.
	var kept []string
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if namesLine(line, wanted) {
			continue
		}
		kept = append(kept, line)
	}
	for _, n := range wanted {
		for _, k := range hostKeys {
			if k = strings.TrimSpace(k); k != "" {
				kept = append(kept, n+" "+k)
			}
		}
	}

	body := strings.Join(kept, "\n") + "\n"
	// 0600: ssh refuses to use a world-readable known_hosts in some
	// configurations, and there is no reason for anything but root to read it.
	if err := os.WriteFile(KnownHostsPath, []byte(body), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", KnownHostsPath, err)
	}
	return nil
}

// namesLine reports whether a known_hosts line is for one of these hosts.
func namesLine(line string, names []string) bool {
	f := strings.Fields(line)
	if len(f) == 0 {
		return false
	}
	for _, h := range strings.Split(f[0], ",") {
		for _, n := range names {
			if strings.EqualFold(strings.TrimSpace(h), n) {
				return true
			}
		}
	}
	return false
}

// refuseIfPVEOwned rejects a known_hosts path that leads into /etc/pve.
//
// Checked with Lstat and EvalSymlinks rather than by reading the name: the
// hazard is a SYMLINK, and every ordinary file call in this package follows one
// silently. os.WriteFile in particular opens O_WRONLY|O_CREATE|O_TRUNC, which
// follows the link and truncates the target — so the failure mode of not
// checking is not a refused write, it is rewriting the cluster's replicated
// known_hosts for every node at once.
func refuseIfPVEOwned(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing there yet. A plain file will be created, which is what a
			// standalone host has.
			return nil
		}
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		// A link we cannot resolve is not one to write through.
		return fmt.Errorf("%s is a symlink that cannot be resolved (%v); refusing to write "+
			"through it, because pve symlinks these files onto pmxcfs where a write is "+
			"cluster-wide", path, err)
	}
	if strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), pmxcfsPrefix) {
		return fmt.Errorf("%s resolves to %s, which is pmxcfs: a write there is cluster-wide, "+
			"needs quorum, and is destroyed wholesale by the next cluster join. Refusing to "+
			"pin a peer through it", path, target)
	}
	return nil
}
