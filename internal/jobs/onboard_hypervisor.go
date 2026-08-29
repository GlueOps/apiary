package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

// OnboardDeps is what onboarding needs from the rest of the system, declared
// here on the consumer side so the sequence can be tested without a tenant
// database or a hypervisor.
type OnboardDeps struct {
	// FleetKey returns the control plane's key for an organization: the public
	// half to install, and the private half to connect with.
	FleetKey func(ctx context.Context, orgID uuid.UUID) (pub string, priv []byte, err error)

	// ManagementSets returns the sets the `from=` clause is derived from.
	ManagementSets func(ctx context.Context, orgID uuid.UUID) ([]tenant.FirewallSet, error)

	// NewRunner opens a connection to a host.
	NewRunner func(addr, user string, priv []byte) (Runner, error)

	// EnrolmentToken mints a token this host can redeem to obtain its own
	// agent credential.
	//
	// Minted per onboarding rather than reused, and short-lived: it is the one
	// secret that crosses to the host, and its whole job is to be exchanged for
	// something better within seconds of arriving.
	EnrolmentToken func(ctx context.Context, orgID uuid.UUID) (string, error)

	// Bundle is the binary to install. Supplied by the caller rather than read
	// here, so the worker does not need a filesystem and the same sequence can be
	// driven by a test.
	//
	// A FUNCTION OF THE HOST'S ARCHITECTURE, because the control plane does not
	// know it until it has connected. Onboarding an arm64 node from an amd64
	// control plane by sending whatever binary happened to be first is an install
	// that succeeds and a service that will not start.
	Bundle func(goarch string) (AgentBundle, error)

	// ProxmoxCredential returns the API token the agent on this machine should
	// use, or an empty credential when the machine belongs to no fleet.
	//
	// The HIVE's credential, which is forced rather than chosen — Proxmox keeps
	// tokens on pmxcfs, so they are cluster-wide and a join destroys whatever the
	// joining node had. See WriteAgentCredential.
	ProxmoxCredential func(ctx context.Context, orgID, hypervisorID uuid.UUID) (ProxmoxCredential, error)

	// RecordHostKeys stores this machine's own SSH host public keys, so that a
	// host joining a cluster THROUGH it can pin them without trusting the
	// network. See the call site for why they are captured here.
	RecordHostKeys func(ctx context.Context, orgID, hypervisorID uuid.UUID, keys []string) error

	// BaseURL is the address the agent should enrol against and dial afterwards.
	//
	// Passed explicitly rather than left to the agent's default of
	// http://localhost:8080, which on a hypervisor is the hypervisor — the one
	// address guaranteed to be wrong.
	BaseURL string

	// Heartbeated reports whether the agent on this host has checked in since t.
	//
	// Takes the RAW machine id, and the implementation digests it before looking
	// anything up — see tenant.MachineIDDigest. The stored form is a digest, this
	// job carries the raw value because that is what the agent is handed, and when
	// the two sides were free to choose the comparison could never match: every
	// onboard would have ended in ErrNoHeartbeat however healthy the host.
	Heartbeated func(ctx context.Context, orgID uuid.UUID, machineID string, since time.Time) (bool, error)

	// RecordOutcome concludes the machine record the ticker claimed, writing
	// `provisioned` or `failed` and releasing the lease. It reports whether the
	// lease was still held; false means a later tick already took the row.
	//
	// A dependency rather than a direct write so this package keeps needing no
	// tenant database handle of its own, which is what lets the whole sequence be
	// driven by a test with no Postgres.
	RecordOutcome func(ctx context.Context, orgID, hypervisorID uuid.UUID, err error) (bool, error)
}

// OnboardInput identifies one host to bring into a fleet.
type OnboardInput struct {
	OrganizationID uuid.UUID

	// HypervisorID is the machine record being onboarded, and is what the
	// Proxmox credential is resolved through — a machine's credential is its
	// hive's, and only the row knows which hive that is.
	//
	// Optional, like it is on the job args: without one the host is still
	// onboarded and simply receives no cluster credential.
	HypervisorID uuid.UUID

	Address string
	User    string
	// MachineID is how the resulting MACHINE record is recognised, and how
	// onboarding knows which agent's heartbeat proves success. Raw here; the
	// column holds its digest.
	MachineID string
}

var (
	// ErrUnsafeFromClause refuses to write a restriction that permits nothing.
	ErrUnsafeFromClause = errors.New("onboard: the derived from= list is empty")
	// ErrNoHeartbeat means everything installed and the agent never checked in.
	ErrNoHeartbeat = errors.New("onboard: the agent did not check in")
)

// Onboard installs the control plane's key and the agent on one hypervisor.
//
// The sequence is fixed and each step is a precondition for the next:
//
//  1. Derive the `from=` clause and REFUSE if it permits nothing. Writing an
//     empty restriction is not a tighter key, it is a key that works from
//     nowhere — and the control plane's SSH access is the repair channel for a
//     broken agent.
//  2. Connect with the fleet key.
//  3. Install the managed key path and PROVE it, before anything else touches
//     the host. A cluster join later replaces pmxcfs and destroys any key that
//     only exists there; doing this at join time would be too late.
//  4. Install the agent, write its configuration, enable the unit.
//  5. Wait for a heartbeat. Onboarding that reports success without one leaves a
//     machine in the fleet that nobody is watching, which is worse than a
//     failure because nothing will ever look at it again.
//
// Idempotent throughout: River retries, and every step is a write of known
// content or a check.
func Onboard(ctx context.Context, d OnboardDeps, in OnboardInput) error {
	pub, priv, err := d.FleetKey(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("onboard: fleet key: %w", err)
	}

	sets, err := d.ManagementSets(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("onboard: management sets: %w", err)
	}
	restriction := tenant.DeriveFromRestriction(sets)
	if !restriction.Safe() {
		// Deliberately fatal rather than falling back to no restriction. Silently
		// installing an unrestricted key because the intended one could not be
		// computed is the kind of quiet widening nobody reviews.
		return fmt.Errorf("%w: define a management set before onboarding, or the key would be "+
			"installed with a restriction that permits nothing", ErrUnsafeFromClause)
	}

	user := in.User
	if user == "" {
		user = "root"
	}
	runner, err := d.NewRunner(in.Address, user, priv)
	if err != nil {
		return fmt.Errorf("onboard: runner: %w", err)
	}
	if c, ok := runner.(interface{ Connect(context.Context) error }); ok {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}
	if c, ok := runner.(interface{ Close() error }); ok {
		defer func() { _ = c.Close() }()
	}

	// FIRST. Everything after this can be retried; losing SSH cannot.
	if err := InstallManagedKey(ctx, runner, OnboardPlan{
		PublicKey:  pub,
		FromClause: restriction.Render(),
	}); err != nil {
		return err
	}

	// Asked of the host rather than assumed. See OnboardDeps.Bundle.
	goarch, err := ReadArch(ctx, runner)
	if err != nil {
		return err
	}
	if d.Bundle == nil {
		return fmt.Errorf("onboard: no agent binary source configured")
	}
	bundle, err := d.Bundle(goarch)
	if err != nil {
		return err
	}
	if err := installAgent(ctx, runner, bundle); err != nil {
		return err
	}

	// BEFORE enrolment, because `enrol --install` merges into this file rather
	// than replacing it — so the credential has to already be there for install
	// to preserve it. Written the other way round, the agent would be given its
	// token and lose it seconds later.
	if d.ProxmoxCredential != nil {
		cred, cerr := d.ProxmoxCredential(ctx, in.OrganizationID, in.HypervisorID)
		if cerr != nil {
			// Not fatal. A host with no usable cluster credential is still worth
			// onboarding: it heartbeats, converges its firewall and can be joined to
			// a fleet — it just refuses cluster tasks, and says why. Failing here
			// would make a credential problem look like an onboarding problem.
			slog.WarnContext(ctx, "could not resolve a Proxmox credential for this machine; "+
				"cluster tasks will refuse until one is written",
				"hypervisor_id", in.HypervisorID, "error", cerr)
		} else if err := WriteAgentCredential(ctx, runner, d.BaseURL, cred); err != nil {
			return err
		}
	}

	// Capture this host's own SSH host keys while we hold an authenticated
	// session to it, which is the only moment they can be obtained without
	// trusting the network.
	//
	// They are for OTHER hosts, not this one: a node joining a cluster needs its
	// PEER's key in known_hosts before pvecm add runs ssh-copy-id, and under
	// agent-polled convergence there is no longer a moment where the control
	// plane holds SSH to both machines at once. Stored now, published in the
	// cluster's composition, and seeded by the joiner from a channel it already
	// trusts.
	//
	// Not fatal. A host whose keys could not be read still onboards, heartbeats
	// and converges its firewall; only a future join through it is affected, and
	// that failure is legible where this one would not be.
	if keys, herr := ReadHostKeys(ctx, runner); herr != nil {
		slog.WarnContext(ctx, "could not read this host's SSH host keys; a later join THROUGH "+
			"this node will fail host key verification until they are captured",
			"hypervisor_id", in.HypervisorID, "error", herr)
	} else if d.RecordHostKeys != nil {
		if rerr := d.RecordHostKeys(ctx, in.OrganizationID, in.HypervisorID, keys); rerr != nil {
			slog.WarnContext(ctx, "could not record this host's SSH host keys",
				"hypervisor_id", in.HypervisorID, "error", rerr)
		}
	}

	// The machine id may not have been known before we connected, and the host is
	// the authority on it either way. A node discovered by the capacity sync is
	// identified by (hive, name) with machine_id NULL, so the ticker that claimed
	// this row had nothing to pass; an operator naming a host by address has no
	// reason to know the value at all.
	//
	// Read here rather than at the top so it sits after InstallManagedKey, which
	// is the one step that must come first. A failure to read is fatal rather
	// than tolerated: every later check — is this host already live, has it
	// heartbeated since we enrolled it — is asked ABOUT this id, and asking them
	// about the empty string returns a confident, wrong answer.
	machineID := in.MachineID
	if machineID == "" {
		machineID, err = ReadMachineID(ctx, runner)
		if err != nil {
			return err
		}
	}

	// Skip enrolment entirely if this host is ALREADY a working agent.
	//
	// Enrolling mints a new credential and repoints the bee at it, which
	// invalidates the one a running agent holds in memory — it does not re-read
	// its store, so it 401s on every heartbeat from then on and never recovers.
	// A retry of this job would therefore take a healthy host and break it
	// permanently, which is the worst shape an idempotency bug can have: the
	// operation that is supposed to be safe to repeat is the one that does damage.
	alreadyLive, err := d.Heartbeated(ctx, in.OrganizationID, machineID, time.Now().Add(-2*time.Minute))
	if err != nil {
		return fmt.Errorf("onboard: check for an existing agent: %w", err)
	}

	// Enrol BEFORE starting the service.
	//
	// The unit runs `apiary-agent start`, which refuses without an identity — so
	// enabling first would give systemd a unit that fails, retries, and fails
	// again, producing a host that looks broken rather than one that is not yet
	// finished. Enrolment is what turns the token into the agent's own scoped
	// credential, and it is idempotent on machine id: a retry re-enrols onto the
	// same bee with a fresh credential rather than accumulating ghosts.
	if !alreadyLive {
		if err := enrolAgent(ctx, runner, d, in); err != nil {
			return err
		}
		// RESTART, not enable --now. The identity was just rewritten and the agent
		// reads it once at startup, so a running process would carry on presenting
		// the credential that has already been replaced. `enable --now` on an
		// active unit is a no-op, which is precisely the wrong behaviour here.
		if out, err := runner.Run(ctx, "systemctl enable apiary-agent.service && systemctl restart apiary-agent.service"); err != nil {
			return fmt.Errorf("onboard: start the agent: %w (%s)", err, strings.TrimSpace(out))
		}
	} else {
		// Already enrolled and healthy: converge the unit without disturbing the
		// identity it is using.
		if out, err := runner.Run(ctx, "systemctl enable --now apiary-agent.service"); err != nil {
			return fmt.Errorf("onboard: enable the agent: %w (%s)", err, strings.TrimSpace(out))
		}
	}

	// The only evidence that onboarding worked. Everything above proves the host
	// accepted commands; only a heartbeat proves the agent is running, reachable
	// and authenticated.
	since := time.Now().UTC()
	ok, err := waitForHeartbeat(ctx, d, in.OrganizationID, machineID, since)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: the binary and unit are installed, so the host is not left "+
			"half-configured — but it is not in the fleet until it checks in", ErrNoHeartbeat)
	}
	return nil
}

// AgentBundle is what the control plane has to put on a host.
//
// THE BINARY AND NOTHING ELSE. It used to carry the systemd unit and the
// environment file too, and both were always empty because nothing populated
// them — an automated onboard therefore failed at "refusing to enable a unit
// with no binary to run", which is the honest report of a path that had never
// been wired.
//
// They are gone rather than filled in, because the agent already writes both:
// `apiary-agent install` embeds the unit and derives the environment file from
// its own configuration. Uploading a second copy from here would mean two
// definitions of the unit that have to agree, and the one an operator installs
// by hand would drift from the one onboarding installs — which is exactly the
// kind of difference nobody notices until a watchdog is missing from half the
// fleet.
type AgentBundle struct {
	// Binary for the host's architecture.
	Binary []byte
}

// installAgent puts the binary on the host.
//
// Just the binary. The unit and the environment file are written by the agent's
// own `install` command, which runs as part of enrolment below — see
// AgentBundle for why the control plane no longer carries copies of them.
//
// Idempotent, because River retries and a partially applied install must
// converge rather than accumulate. The upload is an atomic write, so a transfer
// that dies halfway cannot leave a truncated executable that systemd then tries
// to start — which fails in a way that looks like a broken agent rather than a
// broken copy.
// AgentPath is where the agent is installed.
//
// /usr/sbin, not /usr/local/bin. The agent runs as root and its whole job is
// upgrading and rebooting the host, so it belongs where root-only binaries
// belong — and /usr/local/bin is on every unprivileged user's PATH, which makes
// a root daemon look like something any of them could usefully run.
//
// An agent ALREADY installed somewhere else keeps working. `apiary-agent
// upgrade` resolves its own path and replaces itself in place, and the unit's
// ExecStart is rewritten to whatever the binary actually is — so this changes
// where NEW hosts land, not where existing ones are moved to.
const AgentPath = "/usr/sbin/apiary-agent"

func installAgent(ctx context.Context, r Runner, b AgentBundle) error {
	if len(b.Binary) == 0 {
		return fmt.Errorf("onboard: this control plane carries no agent binary for that host, " +
			"so there is nothing to install; build one into internal/agentdist/dist")
	}
	if out, err := r.Run(ctx, "install -d -m 0755 "+path.Dir(AgentPath)); err != nil {
		return fmt.Errorf("onboard: create directories: %w (%s)", err, strings.TrimSpace(out))
	}
	// Uploaded to a staging path and moved into place, not written over. Linux
	// refuses to write over a running executable (ETXTBSY), so an onboard that
	// re-runs against a host whose agent is already up would otherwise fail on
	// the copy — and the failure is silent enough that it reads as the new
	// binary having been installed.
	staged := path.Join(path.Dir(AgentPath), ".apiary-agent.incoming")
	if err := r.Upload(ctx, staged, 0o755, b.Binary); err != nil {
		return fmt.Errorf("onboard: upload the agent: %w", err)
	}
	if out, err := r.Run(ctx, "mv -f "+staged+" "+AgentPath); err != nil {
		return fmt.Errorf("onboard: install the agent binary: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

// enrolAgent exchanges a short-lived token for the agent's own credential.
//
// The token is passed on the command line, which is visible in the host's
// process list for the moment the command runs. That is accepted deliberately:
// it is single-use, expires in minutes, and is redeemed within a second of being
// sent — whereas the alternative, writing it to a file, leaves it on disk to be
// forgotten. The credential it yields never appears in a command at all; the
// agent writes it straight into its own state.
func enrolAgent(ctx context.Context, r Runner, d OnboardDeps, in OnboardInput) error {
	if d.EnrolmentToken == nil {
		return fmt.Errorf("onboard: no enrolment token source configured")
	}
	token, err := d.EnrolmentToken(ctx, in.OrganizationID)
	if err != nil {
		return fmt.Errorf("onboard: mint enrolment token: %w", err)
	}

	// --base-url explicitly, not left to the environment file: enrol runs as a
	// one-shot command outside systemd, so nothing has loaded that file yet.
	// --install, so the same command that obtains the credential also writes the
	// unit and starts the service. Enrolment alone leaves a host credentialled
	// and unsupervised: no unit, nothing restarting the agent if it dies, and
	// nothing bringing it back after a reboot.
	//
	// --base-url explicitly rather than left to the environment file: enrol runs
	// as a one-shot command outside systemd, so nothing has loaded that file yet
	// — and on the first onboard the file does not exist, because `install` is
	// what creates it.
	cmd := fmt.Sprintf("AGENT_BASE_URL=%s %s enrol --token %s --base-url %s --install",
		shellQuote(d.BaseURL), AgentPath, shellQuote(token), shellQuote(d.BaseURL))
	out, err := r.Run(ctx, cmd)
	if err != nil {
		// The token is NOT included in the error. A failed enrolment is exactly
		// the moment somebody pastes the message into a ticket.
		return fmt.Errorf("onboard: enrol: %w (%s)", err, strings.TrimSpace(redactToken(out, token)))
	}
	return nil
}

// redactToken keeps a live credential out of an error string, a log line and
// whatever reads them afterwards.
func redactToken(out, token string) string {
	if token == "" {
		return out
	}
	return strings.ReplaceAll(out, token, "<token>")
}

// waitForHeartbeat polls until the agent checks in or the context ends.
//
// Polling the control plane's own record rather than asking the host: the
// question is not "is a process running" but "has this agent successfully
// reached us and authenticated", and only one of those is evidence that
// onboarding worked. A process can be up and unable to talk to anything.
// Takes the machine id rather than the whole input, because the id it must poll
// on is the one READ FROM THE HOST when the caller supplied none — passing the
// input would poll on the empty string and never see a heartbeat however healthy
// the agent.
func waitForHeartbeat(
	ctx context.Context, d OnboardDeps, orgID uuid.UUID, machineID string, since time.Time,
) (bool, error) {
	const interval = 5 * time.Second
	for {
		ok, err := d.Heartbeated(ctx, orgID, machineID, since)
		if err != nil {
			return false, fmt.Errorf("onboard: check heartbeat: %w", err)
		}
		if ok {
			return true, nil
		}
		select {
		case <-ctx.Done():
			// Not an error in itself: River will retry, and the agent may well have
			// come up in the meantime. The caller distinguishes this from a failure.
			return false, nil
		case <-time.After(interval):
		}
	}
}
