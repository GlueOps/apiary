package fleetops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// disarmStepTimeout bounds each command of the standalone conversion.
//
// Generous per step — stopping and starting pve-cluster and restarting pveproxy
// are seconds, not minutes — but finite, which is the point. A var so a test can
// shorten it.
var disarmStepTimeout = 2 * time.Minute

// PVEDir is the pmxcfs mount the writability probe writes into. A var so a test
// can point it at a temporary tree, as the other paths already allow.
var PVEDir = "/etc/pve"

// KindDisarm converts a node the cluster has already dropped into a standalone
// host, and destroys the hive secrets it is still carrying.
//
// A TASK, not a convergence. Everything else the agent does derives its action
// from observed against desired state, and this deliberately does not: the
// DISPOSITION is an operator's decision with a security consequence — accept the
// exposure, or rotate this node's own identity — and no observation implies it.
//
// It is nevertheless the AGENT's work. Writing files and restarting services on
// a host is local root work, and running it over SSH from the control plane made
// the control plane's own reachability a precondition of closing a credential
// exposure. The operator's choice travels as data on the task; the execution
// happens where the files are.
const KindDisarm = "fleet.datacenter.disarm"

// Dispositions, matching what the control plane records.
const (
	DisposeAcceptExposure = "accept-exposure"
	DisposeRotate         = "rotate"
)

// clusterSecretFiles are the HIVE's secrets, not this node's.
//
// A departed node keeps byte-identical copies of every one of them, so they
// still authenticate against the fleet it left. authkey.key outranks the rest
// and is the least obvious: PVE signs its authentication tickets with it, so a
// node holding one can mint a ticket for any user in the cluster it left.
var clusterSecretFiles = []string{
	"/etc/pve/priv/token.cfg",
	"/etc/pve/priv/authkey.key",
	"/etc/pve/priv/pve-root-ca.key",
	"/etc/pve/priv/pve-root-ca.srl",
	"/etc/pve/priv/authorized_keys",
	"/etc/pve/priv/known_hosts",
}

// DisarmSpec is what the control plane asks for.
type DisarmSpec struct {
	// Disposition is accept-exposure or rotate. Empty is treated as
	// accept-exposure, which destroys the hive's secrets but leaves this node's
	// own identity alone — the conservative half, and never the reverse.
	Disposition string `json:"disposition"`

	// Peers are the node names this host still carries configuration for. Their
	// directories under /etc/pve/nodes are removed, because a standalone host
	// holding another node's configuration will happily serve it.
	Peers []string `json:"peers"`
}

// Disarm makes this host standalone and proves the hive's secrets are gone.
func Disarm(ctx context.Context, d Deps, e *Env, rawSpec string) error {
	var spec DisarmSpec
	if err := json.Unmarshal([]byte(orEmptyJSON(rawSpec)), &spec); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}
	if d.Runner == nil {
		return fmt.Errorf("this host has no command runner, so it cannot be disarmed")
	}

	// FINGERPRINTED FIRST, so the destruction can be PROVEN rather than assumed.
	// Digests, never contents: this value reaches a log line, and the whole point
	// of the step it verifies is that these bytes stop existing.
	before := secretDigests()

	e.Phase(phaseEvicting)
	e.Logf("agent", "converting this host to standalone")

	steps := []string{
		"systemctl stop pve-cluster corosync 2>&1 || true",
		"killall pmxcfs 2>/dev/null || true",
		"sleep 2",
		// Local mode, so /etc/pve is writable without quorum — which this node no
		// longer has and is never getting back.
		"pmxcfs -l",
		"sleep 2",
		"rm -f " + CorosyncConf,
		"rm -rf /etc/corosync/*",
	}
	for _, p := range spec.Peers {
		if p = strings.TrimSpace(p); p == "" || strings.ContainsAny(p, "/ ") {
			// A name with a slash would escape the directory it is meant to name.
			return fmt.Errorf("refusing to remove a peer directory for the unsafe name %q", p)
		}
		steps = append(steps, "rm -rf /etc/pve/nodes/"+p)
	}
	for _, f := range clusterSecretFiles {
		steps = append(steps, "rm -f "+f)
	}
	steps = append(steps,
		"killall pmxcfs 2>/dev/null || true",
		"sleep 2",
		"systemctl start pve-cluster",
		"sleep 3",
	)

	if strings.EqualFold(spec.Disposition, DisposeRotate) {
		// ROTATE ACTS ON THIS NODE, not on the cluster it left.
		//
		// It replaces what this host holds of its OWN identity, so it shares
		// nothing with that cluster. Rotating the CLUSTER's secrets is a different
		// operation with a different blast radius and is not a step in removing
		// one node from it.
		//
		// Safe here for a specific reason: the control plane's access does not
		// live in /root/.ssh. The fleet key is in /etc/ssh/authorized_keys.d,
		// which pvecm does not touch, so this cannot cost the repair channel into
		// the host.
		e.Logf("agent", "rotating this host's own identity")
		steps = append(steps,
			"rm -f /root/.ssh/id_rsa /root/.ssh/id_rsa.pub "+
				"/root/.ssh/id_ed25519 /root/.ssh/id_ed25519.pub",
			`ssh-keygen -q -t ed25519 -N "" -C "root@$(hostname)" -f /root/.ssh/id_ed25519`,
		)
	}

	steps = append(steps,
		// Reissues this node's certificate against a CA it now owns alone.
		"pvecm updatecerts --force 2>&1 || true",
		"sleep 2",
		// pveproxy and pvedaemon hold the OLD CA in memory. Without this the API
		// refuses its own certificate — observed as "certificate signed by unknown
		// authority" while the files on disk verified against each other perfectly.
		"systemctl restart pveproxy pvedaemon 2>&1 || true",
		"sleep 3",
	)

	// EVERY STEP IS BOUNDED. `|| true` on some of these catches a bad exit code
	// and not a hang, and nothing else here has a deadline.
	//
	// `pvecm updatecerts --force` is the one that showed it: on a node with no
	// cluster it can block indefinitely, so the disarm never returned. The task
	// stayed claimed, was re-delivered on every heartbeat, and climbed past
	// attempt 700 — while holding the bee's only task slot, which is why that
	// host also reported "already has an outstanding task" and could not be
	// upgraded.
	//
	// A disarm that cannot finish must FAIL and say which step it was on. That is
	// recoverable — the operator can look at the host — where hanging forever is
	// not, and quietly blocks everything else the agent would do.
	for _, cmd := range steps {
		stepCtx, cancel := context.WithTimeout(ctx, disarmStepTimeout)
		err := d.Runner.Run(stepCtx, discard, "sh", "-c", cmd)
		cancel()
		if err != nil {
			if stepCtx.Err() != nil && ctx.Err() == nil {
				return fmt.Errorf("standalone conversion stalled at %q after %s; the host needs "+
					"looking at directly", cmd, disarmStepTimeout)
			}
			return fmt.Errorf("standalone conversion at %q: %w", cmd, err)
		}
	}

	e.Phase(phaseVerifying)
	return proveDisarmed(ctx, before, e)
}

// proveDisarmed refuses to report success on the strength of the commands having
// exited zero.
//
// A destruction that did not happen is exactly the failure this exists to
// prevent: the node is detached, looks departed, and still authenticates against
// the fleet it left.
func proveDisarmed(_ context.Context, before map[string]string, e *Env) error {
	if _, err := os.Stat(CorosyncConf); err == nil {
		return fmt.Errorf("%s still exists, so this node is not standalone", CorosyncConf)
	}
	// /etc/pve must be writable, or everything above ran against a read-only
	// filesystem and silently did nothing.
	probe := filepath.Join(PVEDir, ".apiary-disarm-probe")
	if err := os.WriteFile(probe, []byte("x"), 0o600); err != nil {
		return fmt.Errorf("the standalone conversion left %s unwritable, so every command above "+
			"ran against a read-only filesystem and silently did nothing: %w", PVEDir, err)
	}
	_ = os.Remove(probe)

	after := secretDigests()
	for _, f := range clusterSecretFiles {
		was, had := before[f]
		if !had || was == "absent" {
			continue // never held it; nothing to prove about this one
		}
		if now, present := after[f]; present && now != "absent" && now == was {
			return fmt.Errorf("%s is still byte-identical to the copy this node left the hive "+
				"with, so the cluster's secret went with it — the node is detached but not "+
				"disarmed", f)
		}
	}
	e.Logf("agent", "the hive's secrets are gone from this host")
	return nil
}

// secretDigests fingerprints what this node currently holds.
func secretDigests() map[string]string {
	out := make(map[string]string, len(clusterSecretFiles))
	for _, f := range clusterSecretFiles {
		raw, err := os.ReadFile(f)
		if err != nil {
			out[f] = "absent"
			continue
		}
		sum := sha256.Sum256(raw)
		out[f] = hex.EncodeToString(sum[:])
	}
	return out
}

// HeldClusterSecrets counts how many of the hive's secret files this host still
// carries.
//
// A COUNT, never the contents or the digests. The point of reporting it is that
// the control plane can VERIFY a disarm rather than trust the task's exit
// status, and a count is enough to do that: a departed machine holding any of
// them still authenticates against the cluster it left.
//
// Reported on every heartbeat, not only after a disarm. A node that has these
// and should not is a standing exposure whenever it arises — including when a
// disarm never ran because the host had no agent, or the task was lost.
func HeldClusterSecrets() int {
	n := 0
	for _, f := range clusterSecretFiles {
		if _, err := os.Stat(f); err == nil {
			n++
		}
	}
	return n
}
