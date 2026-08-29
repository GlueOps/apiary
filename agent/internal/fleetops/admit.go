package fleetops

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ClusterAuthorizedKeys is the cluster's root authorized_keys.
//
// ON PMXCFS DELIBERATELY, unlike every other path this package writes. It is
// cluster state by design — /root/.ssh/authorized_keys is a symlink to it — so
// one quorate member writing it admits the key on every node at once, which is
// exactly what is wanted. Contrast KnownHostsPath, where landing on pmxcfs is a
// bug and is refused.
var ClusterAuthorizedKeys = "/etc/pve/priv/authorized_keys"

// RootKeyPaths are this host's own root SSH public keys, best first.
var RootKeyPaths = []string{
	"/root/.ssh/id_ed25519.pub",
	"/root/.ssh/id_rsa.pub",
}

// LocalRootKey is this host's root SSH public key.
//
// The key a PEER must trust before this host can join it: `pvecm add --use_ssh`
// runs ssh-copy-id from here to the peer and has to authenticate as root, with
// no password available non-interactively.
//
// Read fresh, because it CHANGES. A departure with rotate replaces this keypair
// — that is the point of rotating — and the cluster then still trusts the key
// that was thrown away. Reporting it continuously is what lets a rotated node
// ever rejoin anything.
func LocalRootKey() (string, error) {
	for _, p := range RootKeyPaths {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		f := strings.Fields(strings.TrimSpace(string(raw)))
		if len(f) >= 2 {
			// Type and material only. The trailing comment is "root@<hostname>",
			// which changes with a rename and would read as a key rotation to
			// anything comparing stored values.
			return f[0] + " " + f[1], nil
		}
	}
	return "", fmt.Errorf("this host has no readable root SSH public key, so no cluster could " +
		"be told to trust it")
}

// AdmitKeys ensures the cluster trusts every key given.
//
// THE JOINER CANNOT DO THIS ITSELF. The file is on the cluster's replicated
// store, which a node has no access to until it is a member — so admitting a
// node is work that belongs to one already in, exactly as an eviction does. The
// symmetry is not a coincidence: both are things only the cluster can do to a
// machine outside it.
//
// Append-only and idempotent. This file is PVE's, it carries every node's key
// and an operator's own, and rewriting it wholesale would lock people out of the
// cluster to admit one machine. Keys already present are left exactly as they
// are, including their comments.
//
// Returns how many keys were newly admitted.
func AdmitKeys(_ context.Context, keys []string) (int, error) {
	var wanted []string
	for _, k := range keys {
		if k = strings.TrimSpace(k); k != "" {
			wanted = append(wanted, k)
		}
	}
	if len(wanted) == 0 {
		return 0, nil
	}

	raw, err := os.ReadFile(ClusterAuthorizedKeys)
	if err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("read %s: %w", ClusterAuthorizedKeys, err)
	}

	// Compared on type+material, ignoring the comment: the same key appears as
	// "root@pve2" on one node and "root@pve" after a rename, and treating those
	// as different admits a duplicate on every convergence.
	have := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) >= 2 {
			have[f[0]+" "+f[1]] = true
		}
	}

	var add []string
	for _, k := range wanted {
		f := strings.Fields(k)
		if len(f) < 2 || have[f[0]+" "+f[1]] {
			continue
		}
		add = append(add, k)
		have[f[0]+" "+f[1]] = true
	}
	if len(add) == 0 {
		return 0, nil
	}
	sort.Strings(add)

	body := string(raw)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += strings.Join(add, "\n") + "\n"

	// 0600: it is the cluster's root keyring. pmxcfs enforces its own ownership,
	// but the mode is stated rather than inherited from whatever created it.
	if err := os.WriteFile(ClusterAuthorizedKeys, []byte(body), 0o600); err != nil {
		return 0, fmt.Errorf("write %s: %w", ClusterAuthorizedKeys, err)
	}
	return len(add), nil
}
