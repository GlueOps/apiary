package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

func mgmtSets(values ...string) []tenant.FirewallSet {
	s := tenant.FirewallSet{Name: "mgmt", Family: tenant.FamilyIPv4, Purpose: tenant.SetPurposeManagement}
	for _, v := range values {
		s.Elements = append(s.Elements, tenant.FirewallSetElement{Value: v})
	}
	return []tenant.FirewallSet{s}
}

func deps(r *fakeRunner, sets []tenant.FirewallSet, beats bool) OnboardDeps {
	return OnboardDeps{
		FleetKey: func(context.Context, uuid.UUID) (string, []byte, error) {
			return "ssh-ed25519 AAAAC3Nz apiary-fleet", []byte("private"), nil
		},
		ManagementSets: func(context.Context, uuid.UUID) ([]tenant.FirewallSet, error) {
			return sets, nil
		},
		NewRunner: func(string, string, []byte) (Runner, error) { return r, nil },
		Bundle: func(string) (AgentBundle, error) {
			return AgentBundle{Binary: []byte("ELF")}, nil
		},
		BaseURL: "http://cp:8282",
		EnrolmentToken: func(context.Context, uuid.UUID) (string, error) {
			return "tok-123", nil
		},
		Heartbeated: func(context.Context, uuid.UUID, string, time.Time) (bool, error) {
			return beats, nil
		},
	}
}

// freshHost reports not-live on the pre-check and live afterwards, which is what
// a host being onboarded for the first time looks like. `beats: true` now means
// something different — an agent that is ALREADY running — so a test that wants
// to exercise enrolment must say so.
func freshHost(d OnboardDeps) OnboardDeps {
	calls := 0
	d.Heartbeated = func(context.Context, uuid.UUID, string, time.Time) (bool, error) {
		calls++
		return calls > 1, nil
	}
	return d
}

func input() OnboardInput {
	return OnboardInput{OrganizationID: uuid.New(), Address: "10.10.10.2", MachineID: "abc"}
}

func TestOnboardInstallsTheKeyBeforeTheAgent(t *testing.T) {
	r := &fakeRunner{}
	if err := Onboard(context.Background(), deps(r, mgmtSets("194.36.65.187"), true), input()); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	key := r.indexOf("authorized_keys.d/apiary")
	agent := r.indexOf("apiary-agent.service")
	if key == -1 || agent == -1 || key > agent {
		t.Fatalf("key=%d agent=%d — the managed key must land first; a later cluster join "+
			"replaces pmxcfs and destroys any key that only exists there, and doing this at "+
			"join time is too late", key, agent)
	}
}

// The refusal that keeps a lockout from being installed deliberately. An empty
// from= is not a tighter key — it is a key that works from nowhere, and the
// control plane's SSH is the repair channel for a broken agent.
func TestOnboardRefusesWhenNoManagementSetExists(t *testing.T) {
	r := &fakeRunner{}
	err := Onboard(context.Background(), deps(r, nil, true), input())

	if !errors.Is(err, ErrUnsafeFromClause) {
		t.Fatalf("got %v, want ErrUnsafeFromClause", err)
	}
	if len(r.cmds) != 0 {
		t.Fatalf("touched the host before refusing: %v", r.cmds)
	}
}

// Falling back to an unrestricted key when the intended restriction cannot be
// computed is exactly the quiet widening nobody reviews.
func TestOnboardDoesNotFallBackToAnUnrestrictedKey(t *testing.T) {
	r := &fakeRunner{}
	_ = Onboard(context.Background(), deps(r, nil, true), input())
	for _, c := range r.cmds {
		if c != "" && contains(c, "authorized_keys.d/apiary") {
			t.Fatalf("installed a key despite an unusable restriction: %q", c)
		}
	}
}

// Onboarding that reports success without a heartbeat leaves a machine in the
// fleet that nobody is watching — worse than a failure, because nothing will
// look at it again.
func TestOnboardFailsWithoutAHeartbeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := &fakeRunner{}
	err := Onboard(ctx, deps(r, mgmtSets("194.36.65.187"), false), input())
	if !errors.Is(err, ErrNoHeartbeat) {
		t.Fatalf("got %v, want ErrNoHeartbeat", err)
	}
	// The install still happened, so the host is not left half-configured and a
	// retry converges rather than starting over.
	if r.indexOf("apiary-agent.service") == -1 {
		t.Fatal("gave up before installing the agent")
	}
}

// If the managed key path cannot be proved, nothing else should be touched —
// the agent install would otherwise proceed on a host the control plane may not
// be able to reach again.
func TestOnboardStopsWhenTheKeyPathIsNotProved(t *testing.T) {
	r := &fakeRunner{reconnectErr: errors.New("permission denied")}
	err := Onboard(context.Background(), deps(r, mgmtSets("194.36.65.187"), true), input())
	if err == nil {
		t.Fatal("onboard succeeded without proving the key path")
	}
	if r.indexOf("apiary-agent.service") != -1 {
		t.Fatal("installed the agent after failing to prove SSH access")
	}
}

// The derived clause reaches the host, so the key is restricted in practice and
// not merely in the model.
func TestOnboardWritesTheDerivedRestriction(t *testing.T) {
	r := &fakeRunner{}
	_ = Onboard(context.Background(), deps(r, mgmtSets("194.36.65.187", "192.168.10.60"), true), input())

	found := false
	for _, c := range r.cmds {
		if contains(c, "authorized_keys.d/apiary") && contains(c, "from=") {
			found = true
			if !contains(c, "192.168.10.60") || !contains(c, "194.36.65.187") {
				t.Fatalf("the restriction is missing a management address: %q", c)
			}
		}
	}
	if !found {
		t.Fatal("the key was written with no from= clause at all")
	}
}

func contains(h, n string) bool { return len(n) > 0 && len(h) >= len(n) && idx(h, n) >= 0 }
func idx(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

// A unit enabled with no binary starts, fails, and looks like a broken agent
// rather than an install that never happened.
func TestOnboardRefusesToEnableAUnitWithNoBinary(t *testing.T) {
	r := &fakeRunner{}
	d := deps(r, mgmtSets("194.36.65.187"), true)
	d.Bundle = func(string) (AgentBundle, error) { return AgentBundle{}, nil }

	if err := Onboard(context.Background(), d, input()); err == nil {
		t.Fatal("enabled a unit with nothing to run")
	}
	for _, c := range r.cmds {
		if contains(c, "enable --now") {
			t.Fatal("enabled the service despite having no binary")
		}
	}
}

// The binary, unit and config all reach the host — the gap that made the first
// version of this worker enable a unit nothing had installed.
//
// Installed to /usr/sbin: this runs as root, and /usr/local/bin is on every
// unprivileged user's PATH.
//
// The BINARY only. The unit and the environment file are no longer uploaded from
// here: the agent writes both itself during `enrol --install`, so a second copy
// shipped from the control plane would be a second definition of the unit that
// has to agree with the agent's own — and the one an operator installs by hand
// would drift from the one onboarding installs.
func TestOnboardUploadsTheBinary(t *testing.T) {
	r := &fakeRunner{}
	if err := Onboard(context.Background(), deps(r, mgmtSets("194.36.65.187"), true), input()); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	// Staged and moved, not written in place. Linux refuses to write over a
	// running executable, so an onboard re-run against a host whose agent is up
	// would fail on the copy — and quietly enough to read as a successful
	// install of the new binary.
	staged := false
	for _, u := range r.uploads {
		if contains(u, ".apiary-agent.incoming") {
			staged = true
		}
	}
	if !staged {
		t.Fatalf("the binary was not staged before being moved into place; uploads were %v",
			r.uploads)
	}
	if r.indexOf("mv -f") == -1 || r.indexOf(AgentPath) == -1 {
		t.Fatalf("the staged binary was never moved into place; commands were %v", r.cmds)
	}

	for _, u := range r.uploads {
		if u == "/etc/systemd/system/apiary-agent.service" || u == "/etc/apiary-agent/agent.env" {
			t.Fatalf("%s was uploaded from the control plane; the agent writes it itself, and "+
				"two definitions of the unit drift", u)
		}
	}
}

// The unit runs `apiary-agent start`, which refuses without an identity. Enabling
// before enrolling gives systemd a unit that fails and retries — a host that
// looks broken rather than one that is not yet finished.
func TestOnboardEnrolsBeforeStarting(t *testing.T) {
	r := &fakeRunner{}
	if err := Onboard(context.Background(), freshHost(deps(r, mgmtSets("194.36.65.187"), false)), input()); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	enrol := r.indexOf("apiary-agent enrol")
	start := r.indexOf("restart apiary-agent")
	if enrol == -1 {
		t.Fatal("never enrolled; the agent would start with no identity")
	}
	if start == -1 || enrol > start {
		t.Fatalf("enrol=%d start=%d — starting first means systemd retries a unit that cannot "+
			"succeed yet", enrol, start)
	}
}

// A failed enrolment is exactly when somebody pastes the error into a ticket.
func TestEnrolmentErrorsDoNotCarryTheToken(t *testing.T) {
	r := &fakeRunner{failOn: "apiary-agent enrol"}
	d := freshHost(deps(r, mgmtSets("194.36.65.187"), false))
	d.EnrolmentToken = func(context.Context, uuid.UUID) (string, error) { return "SUPERSECRET", nil }

	err := Onboard(context.Background(), d, input())
	if err == nil {
		t.Fatal("a failed enrolment was treated as success")
	}
	if contains(err.Error(), "SUPERSECRET") {
		t.Fatalf("the live token is in the error text: %v", err)
	}
}

// The worst shape an idempotency bug can have: the operation that is supposed to
// be safe to repeat is the one that does damage.
//
// Enrolling mints a new credential and repoints the bee at it, invalidating the
// one a running agent holds in memory. The agent does not re-read its store, so
// it 401s on every heartbeat from then on and never recovers — a retry of this
// job takes a healthy host and breaks it permanently.
func TestOnboardDoesNotReEnrolAHealthyAgent(t *testing.T) {
	r := &fakeRunner{}
	minted := 0
	d := deps(r, mgmtSets("194.36.65.187"), true) // already heartbeating
	d.EnrolmentToken = func(context.Context, uuid.UUID) (string, error) {
		minted++
		return "tok", nil
	}

	if err := Onboard(context.Background(), d, input()); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if minted != 0 {
		t.Fatalf("minted %d enrolment tokens for a host that was already live; the running "+
			"agent's credential is now invalid and it cannot recover", minted)
	}
	if r.indexOf("apiary-agent enrol") != -1 {
		t.Fatal("re-enrolled a healthy agent")
	}
}

// A fresh host must be restarted after enrolling, because the agent reads its
// identity once at startup. `enable --now` on an already-active unit is a no-op,
// which would leave the process using the credential that was just replaced.
func TestOnboardRestartsAfterEnrolling(t *testing.T) {
	r := &fakeRunner{}
	d := freshHost(deps(r, mgmtSets("194.36.65.187"), false))

	if err := Onboard(context.Background(), d, input()); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if r.indexOf("restart apiary-agent") == -1 {
		t.Fatal("never restarted after enrolling; the agent would keep presenting the " +
			"credential that was just replaced")
	}
}

// The machine id must be bound to the onboarded record BEFORE the agent enrols.
// Enrolment finds its record by machine id first, and its only other evidence is
// the address the request came from, which on a dual-stack host is the other
// family: seven hosts declared on IPv4 became fourteen records.
func TestOnboardClaimsTheMachineIDBeforeEnrolling(t *testing.T) {
	r := &fakeRunner{}
	d := freshHost(deps(r, mgmtSets("194.36.65.187"), false))
	in := input()
	in.HypervisorID = uuid.New()

	var claimed string
	enrolsBeforeClaim := -1
	d.ClaimMachineID = func(_ context.Context, _, hvID uuid.UUID, machineID string) error {
		if hvID != in.HypervisorID {
			t.Errorf("claimed onto %s, want the record being onboarded %s", hvID, in.HypervisorID)
		}
		claimed = machineID
		enrolsBeforeClaim = r.indexOf("apiary-agent enrol")
		return nil
	}

	if err := Onboard(context.Background(), d, in); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if claimed != in.MachineID {
		t.Fatalf("claimed %q, want the host's machine id %q", claimed, in.MachineID)
	}
	if enrolsBeforeClaim != -1 {
		t.Fatal("the agent enrolled before its machine id was bound to this record, so enrolment " +
			"could not find it and created a second one")
	}
}

// A refused claim means two records disagree about which host is which.
// Enrolling anyway is exactly how the duplicate gets made.
func TestOnboardStopsWhenTheClaimIsRefused(t *testing.T) {
	r := &fakeRunner{}
	d := freshHost(deps(r, mgmtSets("194.36.65.187"), false))
	in := input()
	in.HypervisorID = uuid.New()
	refused := errors.New("this machine's id already belongs to another hypervisor record")
	d.ClaimMachineID = func(context.Context, uuid.UUID, uuid.UUID, string) error { return refused }

	err := Onboard(context.Background(), d, in)
	if !errors.Is(err, refused) {
		t.Fatalf("got %v, want the refusal surfaced", err)
	}
	if r.indexOf("apiary-agent enrol") != -1 {
		t.Fatal("enrolled after the claim was refused")
	}
}
