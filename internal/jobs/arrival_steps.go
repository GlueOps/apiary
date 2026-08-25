package jobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/glueops/apiary/internal/models/tenant"
)

const knownHostsFile = "/root/.ssh/known_hosts"

// JoinBlockers is what `pvecm add` would refuse on, read BEFORE attempting it.
//
// Worth checking ourselves rather than letting pvecm report them, for two
// reasons. Its refusal arrives as a block of prose after the command has already
// begun, and one of the four conditions is actively misleading: "this host
// already contains virtual guests" fires on a node with none of its own, because
// it counts the guest configs the node inherited as a replica of its former
// cluster. An operator reading that message goes looking for VMs that are not
// there.
type JoinBlockers struct {
	HasCorosyncConf bool
	HasAuthKey      bool
	CorosyncRunning bool
	// GuestConfigs counts guest configuration files under /etc/pve/nodes, across
	// ALL node directories — which is what pvecm counts.
	GuestConfigs int
	// ForeignNodeDirs are node directories that are not this node's own. These
	// are where inherited guest configs live, and clearing them is what resolves
	// a guest count on a host that has no guests.
	ForeignNodeDirs []string
}

// Blocked reports whether pvecm add would refuse.
func (b JoinBlockers) Blocked() bool {
	return b.HasCorosyncConf || b.HasAuthKey || b.CorosyncRunning || b.GuestConfigs > 0
}

// Describe explains the refusal in terms of what to do about it.
func (b JoinBlockers) Describe() string {
	var parts []string
	if b.HasCorosyncConf {
		parts = append(parts, "a cluster config is still present ("+corosyncConf+")")
	}
	if b.HasAuthKey {
		parts = append(parts, "a corosync auth key is still present (/etc/corosync/authkey)")
	}
	if b.CorosyncRunning {
		parts = append(parts, "corosync is still running")
	}
	if b.GuestConfigs > 0 {
		d := fmt.Sprintf("%d guest configuration(s) are visible under /etc/pve/nodes", b.GuestConfigs)
		if len(b.ForeignNodeDirs) > 0 {
			d += fmt.Sprintf(" — including in %v, which belong to a former cluster peer rather "+
				"than to this host; a departure clears those", b.ForeignNodeDirs)
		}
		parts = append(parts, d)
	}
	return strings.Join(parts, "; ")
}

// ReadJoinBlockers inspects a node for everything that would make a join refuse.
func ReadJoinBlockers(ctx context.Context, r Runner, ownNodeName string) (JoinBlockers, error) {
	// One command, one marker line per fact, so a partially-answering host cannot
	// be read as a clean bill of health — the same contract as the node counter,
	// for the same reason.
	cmd := fmt.Sprintf(`
conf=no; [ -f %s ] && conf=yes
akey=no; [ -f /etc/corosync/authkey ] && akey=yes
coro=$(systemctl is-active corosync 2>/dev/null || true)
guests=$(find /etc/pve/nodes -mindepth 3 -maxdepth 3 \( -name '*.conf' \) 2>/dev/null | wc -l)
dirs=$(/bin/ls /etc/pve/nodes 2>/dev/null | tr '\n' ' ')
echo "APIARY_CONF=$conf"
echo "APIARY_AKEY=$akey"
echo "APIARY_CORO=$coro"
echo "APIARY_GUESTS=$guests"
echo "APIARY_DIRS=$dirs"
`, corosyncConf)

	out, err := r.Run(ctx, cmd)
	if err != nil {
		return JoinBlockers{}, fmt.Errorf("arrive: read join preconditions: %w (%s)",
			err, strings.TrimSpace(out))
	}

	fields := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		for _, k := range []string{"CONF", "AKEY", "CORO", "GUESTS", "DIRS"} {
			if v, ok := strings.CutPrefix(line, "APIARY_"+k+"="); ok {
				fields[k] = v
			}
		}
	}
	// Every marker must be present. A missing one means the command did not run
	// to completion, and treating an absent answer as "no blocker" would let a
	// join proceed on a node nobody actually inspected.
	for _, k := range []string{"CONF", "AKEY", "CORO", "GUESTS", "DIRS"} {
		if _, ok := fields[k]; !ok {
			return JoinBlockers{}, fmt.Errorf(
				"arrive: the join precondition check did not answer for %s, so the node's state "+
					"is unknown and a join must not be attempted (output: %q)", k, strings.TrimSpace(out))
		}
	}

	b := JoinBlockers{
		HasCorosyncConf: fields["CONF"] == "yes",
		HasAuthKey:      fields["AKEY"] == "yes",
		CorosyncRunning: fields["CORO"] == "active",
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(fields["GUESTS"]))
	if convErr != nil {
		return JoinBlockers{}, fmt.Errorf("arrive: could not read a guest count from %q", fields["GUESTS"])
	}
	b.GuestConfigs = n
	for _, d := range strings.Fields(fields["DIRS"]) {
		if d != ownNodeName {
			b.ForeignNodeDirs = append(b.ForeignNodeDirs, d)
		}
	}
	return b, nil
}

// ReadNodeName asks the host what it calls itself.
//
// Authoritative, and the reason it exists is that guessing does not work. The
// node name was previously derived from the machine's ADDRESS — its first label
// — which yields "" for an IP literal, and Proxmox metal is routinely recorded by
// IP. An empty name broke arrival twice over: ReadJoinBlockers filters
// /etc/pve/nodes against it, so the node's OWN directory read as a former peer's
// and the join was refused before it started; and MarkJoined refuses to record
// membership without one, so a join that did succeed could not be written down.
//
// A Proxmox node name is the short hostname, and pmxcfs creates
// /etc/pve/nodes/<that name> for it. `hostname -s` is the direct answer; the
// pmxcfs directory is not consulted because a machine mid-departure legitimately
// carries a former peer's directory too.
func ReadNodeName(ctx context.Context, r Runner) (string, error) {
	out, err := r.Run(ctx, `echo "APIARY_NODE=$(hostname -s 2>/dev/null || hostname)"`)
	if err != nil {
		return "", fmt.Errorf("arrive: read the node's own name: %w (%s)", err, strings.TrimSpace(out))
	}
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "APIARY_NODE="); ok {
			name := strings.TrimSpace(v)
			if name == "" {
				break
			}
			// Guard the value that reaches an rm -rf path and a database column.
			if strings.ContainsAny(name, "/ \t") {
				return "", fmt.Errorf("arrive: the host reports an unusable node name %q", name)
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("arrive: the host did not report a node name (output: %q)",
		strings.TrimSpace(out))
}

// ReadHostKeys returns a host's SSH host public keys, read over the control
// plane's OWN authenticated session to that host.
//
// This is the whole point: the keys are obtained from a connection already
// authenticated against a pinned host key, so seeding them on a third machine
// carries that assurance forward. `ssh-keyscan` would produce an equally
// working file by trusting whatever answered on the address — which is the same
// defect as StrictHostKeyChecking=no, moved one step earlier.
func ReadHostKeys(ctx context.Context, r Runner) ([]string, error) {
	out, err := r.Run(ctx,
		"cat /etc/ssh/ssh_host_ed25519_key.pub /etc/ssh/ssh_host_rsa_key.pub 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("arrive: read host keys: %w (%s)", err, strings.TrimSpace(out))
	}
	var keys []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		keys = append(keys, f[0]+" "+f[1])
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("arrive: the target presented no host keys to pin, so the join " +
			"could only proceed by trusting whatever answers on its address")
	}
	return keys, nil
}

// SeedKnownHosts installs a target's host keys on the joining node.
//
// Without this, `pvecm add --use_ssh` fails at its first real step — ssh-copy-id
// cannot verify the target — and reports the entire operation as:
//
//	unable to copy ssh ID: exit code 1
//
// That is the whole message. It names neither the host, nor the key, nor
// verification. An automated join hits it on EVERY node, because a freshly
// provisioned host has no known_hosts at all.
//
// Seeded under every name the join may use — the address, the short node name and
// the FQDN — because known_hosts is keyed on the string used to connect, and
// pvecm resolves node names internally as well as using the address it was given.
//
// Idempotent: existing entries for these names are removed first, so a retry does
// not accumulate duplicates and a target whose host key legitimately changed is
// re-pinned rather than left with two conflicting entries.
func SeedKnownHosts(ctx context.Context, r Runner, names []string, hostKeys []string) error {
	if len(names) == 0 || len(hostKeys) == 0 {
		return fmt.Errorf("arrive: refusing to seed known_hosts with no names or no keys")
	}
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			return fmt.Errorf("arrive: refusing to seed known_hosts for an empty name")
		}
	}

	var b strings.Builder
	b.WriteString("set -e; install -d -m 0700 /root/.ssh; touch " + knownHostsFile + "; ")
	// ssh-keygen -R rewrites the file removing every entry for that host. Run
	// before appending so a retry re-pins rather than duplicating.
	for _, n := range names {
		fmt.Fprintf(&b, "ssh-keygen -R %s -f %s >/dev/null 2>&1 || true; ",
			shellQuote(n), knownHostsFile)
	}
	for _, n := range names {
		for _, k := range hostKeys {
			fmt.Fprintf(&b, "printf '%%s\\n' %s >> %s; ",
				shellQuote(n+" "+k), knownHostsFile)
		}
	}
	b.WriteString("chmod 0600 " + knownHostsFile)

	if out, err := r.Run(ctx, b.String()); err != nil {
		return fmt.Errorf("arrive: seed known_hosts: %w (%s)", err, strings.TrimSpace(out))
	}

	// Verify by asking ssh-keygen to find each name, rather than trusting the
	// append. A known_hosts that did not take produces the bare "unable to copy
	// ssh ID" failure later, with nothing pointing back here.
	for _, n := range names {
		out, err := r.Run(ctx, fmt.Sprintf("ssh-keygen -F %s -f %s >/dev/null 2>&1 && echo FOUND || echo MISSING",
			shellQuote(n), knownHostsFile))
		if err != nil {
			return fmt.Errorf("arrive: verify known_hosts for %s: %w (%s)", n, err, strings.TrimSpace(out))
		}
		if strings.TrimSpace(lastLine(out)) != "FOUND" {
			return fmt.Errorf("arrive: %s is not in the joining node's known_hosts after seeding, "+
				"so the join would fail with an error naming none of this", n)
		}
	}
	return nil
}

// AuthorizeJoinKey authorizes the joining node's own root key on the target,
// restricted to the joining node's address.
//
// The joining node's OWN key, not one apiary mints — because pvecm decides this,
// not us. `pvecm add --use_ssh` runs ssh-copy-id, which copies
// /root/.ssh/id_rsa.pub from the joiner. There is no hook to substitute a
// different credential, so the ephemeral thing here is the AUTHORIZATION, not a
// keypair: the joiner's standing key is permitted on the target, from one
// address, for the duration of the join, and revoked immediately after.
//
// Written to authorized_keys.d rather than through /root/.ssh/authorized_keys,
// which is a symlink onto pmxcfs — a write there needs quorum and would replicate
// the grant to every node in the target hive.
func AuthorizeJoinKey(ctx context.Context, r Runner, joinerKey, fromAddr string) error {
	joinerKey = strings.TrimSpace(joinerKey)
	if joinerKey == "" {
		return fmt.Errorf("arrive: refusing to authorize an empty join key")
	}
	if strings.TrimSpace(fromAddr) == "" {
		// An unrestricted grant would authorize the joiner's key from anywhere,
		// and this file survives the join by design — so an omitted restriction is
		// not a looser temporary grant, it is a permanent one.
		return fmt.Errorf("arrive: refusing to authorize a join key with no from= restriction")
	}

	line := fmt.Sprintf("from=%q %s", fromAddr, joinerKey)
	steps := []string{
		fmt.Sprintf("install -d -m 0755 %s", managedDir),
		fmt.Sprintf("printf '%%s\\n' %s > %s", shellQuote(line), joinKeyFile),
		fmt.Sprintf("chmod 0644 %s", joinKeyFile),
		// The sshd drop-in is written WHOLE, with the join path included. It is
		// not appended to, and it is emphatically not guarded by a grep for the
		// path.
		//
		// A grep was tried and produced a silent failure on a live host: the
		// config already listed `.../authorized_keys.d/join-probe`, and `join` is
		// a SUBSTRING of `join-probe`, so the check matched, the path was never
		// added, and sshd never read the key file that had just been written.
		// The join then failed to authenticate while every step reported success.
		//
		// Rendering the file whole has no such failure mode, and it is the same
		// rule InstallManagedKey follows for the same reason: sshd takes the FIRST
		// value for a keyword, so a duplicated AuthorizedKeysFile line does not
		// error, it silently wins or loses depending on order.
		fmt.Sprintf("printf '%%s\\n' %s > %s", shellQuote(sshdDropInContent(true)), sshdDropIn),
		fmt.Sprintf("chmod 0644 %s", sshdDropIn),
		"sshd -t",
		"systemctl reload ssh 2>/dev/null || systemctl reload sshd",
	}
	for _, cmd := range steps {
		if out, err := r.Run(ctx, cmd); err != nil {
			return fmt.Errorf("arrive: authorize the join key at %q: %w (%s)",
				cmd, err, strings.TrimSpace(out))
		}
	}
	return nil
}

// JoinCluster runs the join and verifies membership.
//
// --use_ssh deliberately. Without it pvecm prompts for the target's root password
// and calls the HTTPS join API, which means a password this system does not hold
// and must not hold — and which breaks outright when TOTP is enabled on
// root@pam. The SSH path skips that branch entirely: it needs no password and
// makes no API call, which is why TOTP on the target is irrelevant to it.
func JoinCluster(
	ctx context.Context, r Runner, targetAddr string, links []tenant.CorosyncLink, wantHive string,
) error {
	if strings.TrimSpace(targetAddr) == "" {
		return fmt.Errorf("arrive: a target address is required")
	}
	if err := tenant.ValidateCorosyncLinks(links); err != nil {
		return fmt.Errorf("arrive: %w", err)
	}

	// Sorted so the command is stable between attempts and a diff of two runs
	// shows what actually changed rather than a reordering.
	var flags strings.Builder
	for _, l := range tenant.SortCorosyncLinks(links) {
		flag, value := l.Flag()
		flags.WriteString(" " + flag + " " + shellQuote(value))
	}

	out, err := r.Run(ctx, fmt.Sprintf("pvecm add %s --use_ssh%s 2>&1",
		shellQuote(targetAddr), flags.String()))
	if err != nil {
		return fmt.Errorf("arrive: pvecm add: %w (%s)", err, strings.TrimSpace(out))
	}

	// The command's own output is not the evidence. It reports "successfully
	// added node" before the local cluster stack has necessarily converged, and a
	// join that half-succeeded says the same thing.
	return AssertJoined(ctx, r, wantHive)
}

// ReadClusterName returns the cluster a node is currently a member of, or "" if
// it belongs to none.
//
// Used to learn what the JOIN TARGET is actually in, so the joiner can be checked
// against that rather than against the hive's name. Nothing enforces that an
// apiary hive is named after the Proxmox cluster it represents, and asserting on
// the hive's name would fail a correct join purely because an operator called the
// hive something else — while still not catching the case that matters.
//
// The case that matters is joining the WRONG cluster: pvecm add is pointed at an
// address, and if something else answers there the node ends up in a fleet nobody
// asked for, with pvecm reporting success. Comparing the joiner against the
// target's own cluster catches exactly that, and needs no naming convention.
func ReadClusterName(ctx context.Context, r Runner) (string, error) {
	out, err := r.Run(ctx,
		`echo "APIARY_CLUSTER=$(pvecm status 2>/dev/null | awk -F': +' '/^Name:/{print $2}')"`)
	if err != nil {
		return "", fmt.Errorf("arrive: read the cluster name: %w (%s)", err, strings.TrimSpace(out))
	}
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "APIARY_CLUSTER="); ok {
			return strings.TrimSpace(v), nil
		}
	}
	return "", fmt.Errorf("arrive: could not read a cluster name (output: %q)", strings.TrimSpace(out))
}

// AssertJoined verifies the node is a quorate member of the named cluster.
func AssertJoined(ctx context.Context, r Runner, wantHive string) error {
	cmd := `
name=$(pvecm status 2>/dev/null | awk -F': +' '/^Name:/{print $2}')
quor=$(pvecm status 2>/dev/null | awk -F': +' '/^Quorate:/{print $2}')
echo "APIARY_CLUSTER=${name:-none}"
echo "APIARY_QUORATE=${quor:-no}"
`
	out, err := r.Run(ctx, cmd)
	if err != nil {
		return fmt.Errorf("arrive: verify membership: %w (%s)", err, strings.TrimSpace(out))
	}

	var cluster, quorate string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "APIARY_CLUSTER="); ok {
			cluster = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "APIARY_QUORATE="); ok {
			quorate = strings.TrimSpace(v)
		}
	}
	if cluster == "" || quorate == "" {
		return fmt.Errorf("arrive: could not read cluster membership from the node (output: %q)",
			strings.TrimSpace(out))
	}
	if !strings.EqualFold(cluster, wantHive) {
		return fmt.Errorf("arrive: the node reports cluster %q, expected %q", cluster, wantHive)
	}
	if !strings.EqualFold(quorate, "yes") {
		return fmt.Errorf("arrive: the node joined cluster %q but is NOT quorate, so it cannot "+
			"write to /etc/pve and is not usefully a member yet", wantHive)
	}
	return nil
}

// AssertManagedKeySurvived re-proves the control plane's own access after a join.
//
// The join replaces the node's pmxcfs wholesale, destroying every key in
// /etc/pve/priv/authorized_keys — observed on a real join, where the joining
// node's keyring went from three entries to one. The managed key lives in
// authorized_keys.d precisely so it survives that, and this is where that claim
// gets checked rather than assumed.
//
// Reconnect, not a file read. The file being present proves it is on disk; only a
// fresh connection proves sshd still honours it, and losing SSH to a host is the
// one failure here with no way back.
func AssertManagedKeySurvived(ctx context.Context, r Runner) error {
	if err := r.Reconnect(ctx); err != nil {
		return fmt.Errorf("arrive: the join completed but the control plane can no longer "+
			"authenticate to this host — the managed key did not survive it: %w", err)
	}
	return nil
}
