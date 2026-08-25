package jobs

import (
	"context"
	"fmt"
	"strings"
)

// Runner executes a command on a hypervisor and reports what it said.
//
// An interface rather than an ssh.Client so the ordering below — which is the
// part that can lock the control plane out of a host — is testable without a
// hypervisor to lock itself out of.
type Runner interface {
	// Upload writes content to a path on the host, atomically.
	//
	// Separate from Run because a binary cannot be interpolated into a shell
	// command — and because writing in place would leave a half-copied
	// executable if the transfer failed, which systemd would then try to start.
	Upload(ctx context.Context, path string, mode uint32, content []byte) error

	// Run executes cmd and returns combined output. A non-zero exit is an error.
	Run(ctx context.Context, cmd string) (string, error)
	// Reconnect opens a NEW connection with the same credential. It is how a
	// change to sshd is proved rather than assumed: the existing session survives
	// a reload regardless, so only a fresh connection tests what changed.
	Reconnect(ctx context.Context) error
}

// OnboardPlan is what the worker installs on one host.
type OnboardPlan struct {
	// PublicKey is the control plane's fleet key, in authorized_keys form.
	PublicKey string
	// FromClause is the derived `from=` restriction, or "" for none.
	//
	// Empty means unrestricted, NOT "permit nothing" — the caller must have
	// already refused an unsafe derivation. See tenant.FromRestriction.Safe.
	FromClause string
}

// Managed paths. Chosen so that a cluster join cannot destroy them: PVE
// symlinks /root/.ssh/authorized_keys onto pmxcfs, and joining replaces that
// database wholesale.
const (
	managedDir     = "/etc/ssh/authorized_keys.d"
	managedKeyFile = managedDir + "/apiary"
	sshdDropIn     = "/etc/ssh/sshd_config.d/60-apiary.conf"
)

// InstallManagedKey puts the control plane's key somewhere a cluster join cannot
// erase, and proves the new path works before returning.
//
// ORDER IS THE WHOLE POINT, and it is not obvious:
//
//  1. Write the key file and the sshd drop-in.
//  2. Validate with `sshd -t`. A bad config means sshd will not come back, which
//     is the one failure here that is genuinely unrecoverable.
//  3. Reload — never restart. A reload keeps existing sessions, so the
//     connection running this is still alive if the change is wrong.
//  4. RECONNECT and prove it. The session that made the change survives a reload
//     no matter what, so it is not evidence. Only a new connection is.
//
// This must run BEFORE any cluster join. The join replaces pmxcfs, so a key that
// only exists in /root/.ssh/authorized_keys disappears mid-operation, and the
// control plane loses SSH to a host at the moment it is halfway through changing
// it. Doing this at join time is too late.
//
// Idempotent: every step is a write of known content or a check, so River
// retrying the job re-converges rather than accumulating.
func InstallManagedKey(ctx context.Context, r Runner, plan OnboardPlan) error {
	if strings.TrimSpace(plan.PublicKey) == "" {
		// Writing an empty managed file would remove the control plane's own
		// access on the next reload — with nothing left to fix it over.
		return fmt.Errorf("onboard: refusing to install an empty key file")
	}

	line := strings.TrimSpace(plan.PublicKey)
	if c := strings.TrimSpace(plan.FromClause); c != "" {
		line = c + " " + line
	}

	// 0755 on the directory and 0644 on the files: sshd's StrictModes refuses a
	// group- or world-writable path and says so only in the server log, so the
	// failure looks like a rejected key rather than a permissions problem.
	steps := []string{
		fmt.Sprintf("install -d -m 0755 %s", managedDir),
		// printf, not echo: a file that does not end in a newline concatenates
		// its last entry with whatever is appended next.
		fmt.Sprintf("printf '%%s\\n' %s > %s", shellQuote(line), managedKeyFile),
		fmt.Sprintf("chmod 0644 %s", managedKeyFile),
		fmt.Sprintf("printf '%%s\\n' %s > %s", shellQuote(sshdDropInContent(false)), sshdDropIn),
		fmt.Sprintf("chmod 0644 %s", sshdDropIn),
	}
	for _, cmd := range steps {
		if out, err := r.Run(ctx, cmd); err != nil {
			return fmt.Errorf("onboard: %s: %w (%s)", cmd, err, strings.TrimSpace(out))
		}
	}

	// Validate BEFORE reloading. sshd -t parses the whole configuration; a
	// reload on a broken config leaves the daemon refusing every connection.
	if out, err := r.Run(ctx, "sshd -t"); err != nil {
		return fmt.Errorf("onboard: sshd rejected the configuration, not reloading: %w (%s)",
			err, strings.TrimSpace(out))
	}

	// Reload, not restart: existing sessions survive, including this one.
	if out, err := r.Run(ctx, "systemctl reload ssh 2>/dev/null || systemctl reload sshd"); err != nil {
		return fmt.Errorf("onboard: reload sshd: %w (%s)", err, strings.TrimSpace(out))
	}

	// The proof. Everything above would also "succeed" against a host where the
	// managed path is never consulted, because this session authenticated before
	// any of it and a reload does not disturb it.
	if err := r.Reconnect(ctx); err != nil {
		return fmt.Errorf("onboard: the managed key path does not authenticate — the old path is "+
			"still in place, so access is not lost, but nothing has been proved: %w", err)
	}
	return nil
}

// sshdDropInContent is the whole file, written every time rather than appended
// to, so a retry cannot produce two AuthorizedKeysFile lines. sshd takes the
// FIRST value for a keyword, so a duplicate would not error — it would silently
// win or lose depending on order.
func sshdDropInContent(withJoin bool) string {
	paths := managedDir + "/operators " + managedKeyFile
	if withJoin {
		paths += " " + joinKeyFile
	}
	return strings.Join([]string{
		"# Managed by apiary. Written whole on every convergence.",
		"#",
		"# /root/.ssh/authorized_keys is a symlink to /etc/pve/priv/authorized_keys,",
		"# which lives on pmxcfs. A cluster join REPLACES that database, destroying",
		"# every key in it. These paths are plain local files that survive a join, a",
		"# leave, a quorum loss and pvecm updatecerts.",
		"#",
		"# operators is never touched by automation; apiary is rendered whole, so an",
		"# operator key placed in it would be deleted without warning.",
		"AuthorizedKeysFile .ssh/authorized_keys " + paths,
	}, "\n")
}

// shellQuote wraps a value in single quotes for a POSIX shell.
//
// The content is a public key and a from= clause, both of which contain
// characters a shell would otherwise interpret — and the from= list is derived
// from operator-supplied addresses, so it is not something to interpolate raw
// into a command running as root.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ReadMachineID asks the host for its own /etc/machine-id.
//
// The caller cannot always supply one. A node discovered by the capacity sync is
// identified by (hive, name) and its machine_id column is NULL — the digest is
// only learned at enrolment, which is downstream of this job. So the ticker has
// nothing to pass, and the host is the authority on its own identity anyway.
//
// Read rather than generated, and never defaulted. The value decides which
// machine record a heartbeat belongs to: inventing one would make onboarding
// wait for a heartbeat that can never arrive under that id, and reusing another
// host's would silently merge two machines into one row.
//
// /etc/machine-id first, /var/lib/dbus/machine-id second. The second is where
// the value lives on hosts where /etc is read-only or the file was seeded by
// dbus, and systemd keeps them identical when both exist.
func ReadMachineID(ctx context.Context, r Runner) (string, error) {
	out, err := r.Run(ctx, "cat /etc/machine-id 2>/dev/null || cat /var/lib/dbus/machine-id 2>/dev/null")
	if err != nil {
		return "", fmt.Errorf("onboard: read machine id: %w", err)
	}
	id := strings.TrimSpace(out)
	if id == "" {
		return "", fmt.Errorf("onboard: this host reports no machine id: neither /etc/machine-id " +
			"nor /var/lib/dbus/machine-id has a value, so there is nothing to identify it by and " +
			"a heartbeat from it could not be matched to any machine record")
	}
	// Rejected rather than trimmed to the first line. A file holding more than one
	// value is a host in a state nobody intended, and picking one of them quietly
	// is how the wrong machine gets identified.
	if strings.ContainsAny(id, " \t\r\n") {
		return "", fmt.Errorf("onboard: this host's machine id is not a single value (%q); "+
			"refusing to guess which part identifies it", id)
	}
	return id, nil
}
