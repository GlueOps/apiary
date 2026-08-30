package tenant

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func bootstrapHas(rs []BootstrapRefusal, code string) bool {
	for _, r := range rs {
		if r.Code == code {
			return true
		}
	}
	return false
}

func bootstrapCodes(rs []BootstrapRefusal) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.Code)
	}
	return out
}

// The case the whole operation exists for: a fleetless, provisioned host and an
// empty hive is exactly when a cluster should be created.
func TestBootstrapAllowsAFleetlessHostIntoAnEmptyHive(t *testing.T) {
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardProvisioned,
		AgentOnline:       true,
	})
	if len(rs) != 0 {
		t.Fatalf("a provisioned fleetless host must be allowed to create an empty hive, got %v",
			bootstrapCodes(rs))
	}
}

// GUESTS DO NOT BLOCK A BOOTSTRAP, and this is the difference between this gate
// and the arrival one.
//
// `pvecm add` refuses a node carrying any guest, stopped ones included, because
// the joining node's configuration is replaced by the cluster's. `pvecm create`
// replaces nothing: the node becomes a single-node cluster and keeps what it was
// running. Refusing here would make it impossible to adopt an existing Proxmox
// host without emptying it first, which is the opposite of adoption.
func TestBootstrapDoesNotRefuseAHostCarryingGuests(t *testing.T) {
	// There is no guest field on the request at all, which is the strongest form
	// this assertion can take: the gate cannot refuse on something it is not told.
	// Stated as a test anyway, because "we deliberately do not check X" is
	// invisible in code and someone will add the field back.
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardProvisioned,
		AgentOnline:       true,
	})
	for _, r := range rs {
		if strings.Contains(r.Code, "guest") {
			t.Fatalf("the bootstrap gate refused on guests (%s); pvecm create is safe on a host "+
				"that is already running workloads, and refusing here makes adopting an "+
				"existing Proxmox host impossible without emptying it", r.Code)
		}
	}
}

func TestBootstrapRefusesAHostAlreadyInAHive(t *testing.T) {
	id := uuid.New()
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		CurrentHiveID:     &id,
		ProvisioningState: OnboardProvisioned,
		AgentOnline:       true,
	})
	if !bootstrapHas(rs, RefuseBootstrapInHive) {
		t.Fatalf("a clustered host must not create a second cluster on itself, got %v",
			bootstrapCodes(rs))
	}
}

// The refusal that makes this operation distinct from arrival. A hive with nodes
// already has its cluster; running `pvecm create` again does not join it, it
// makes a SECOND cluster that happens to share a name.
func TestBootstrapRefusesAHiveThatAlreadyHasNodes(t *testing.T) {
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		HiveNodeCount:     1,
		ProvisioningState: OnboardProvisioned,
		AgentOnline:       true,
	})
	if !bootstrapHas(rs, RefuseBootstrapHiveNotEmpty) {
		t.Fatalf("a hive that already has nodes must refuse a bootstrap, got %v",
			bootstrapCodes(rs))
	}
}

// A half-onboarded host with a LIVE AGENT is allowed, which is the inversion of
// what this test used to assert.
//
// It required `provisioned` because "every step of a bootstrap runs over SSH as
// the control plane". The agent creates the cluster now and the control plane
// opens no connection, so an unfinished onboarding says something about the
// SSH fallback and nothing about whether this machine can form a cluster.
func TestBootstrapAllowsAHalfOnboardedHostWithALiveAgent(t *testing.T) {
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardPending,
		AgentOnline:       true,
	})
	if bootstrapHas(rs, RefuseBootstrapUnprovisioned) {
		t.Fatalf("a host with a live agent was refused for the control plane's own SSH state: %v",
			bootstrapCodes(rs))
	}
}

// Every reason at once, for the same purpose as the departure and arrival gates:
// an operator fixing preconditions one job failure at a time spends a worker run
// against real metal per round trip.
func TestBootstrapReportsEveryRefusalAtOnce(t *testing.T) {
	id := uuid.New()
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		CurrentHiveID:     &id,
		HiveNodeCount:     2,
		ProvisioningState: OnboardPending,
		// No agent, which is what RefuseBootstrapUnprovisioned now means.
		AgentOnline: false,
	})
	for _, want := range []string{
		RefuseBootstrapInHive, RefuseBootstrapHiveNotEmpty, RefuseBootstrapUnprovisioned,
	} {
		if !bootstrapHas(rs, want) {
			t.Errorf("want %s among the refusals, got %v", want, bootstrapCodes(rs))
		}
	}
}

// Proxmox's own constraint, read off a running node rather than inferred:
// PVE::API2::ClusterConfig declares clustername with format 'pve-node' and
// maxLength 15, and pve_verify_node_name is the expression below.
func TestClusterNameMatchesWhatProxmoxAccepts(t *testing.T) {
	for _, ok := range []string{"dev", "lab-dc", "a", "a1", "prod-dc-01", "abcdefghijklmno"} {
		if err := ValidateClusterName(ok); err != nil {
			t.Errorf("ValidateClusterName(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []struct{ name, why string }{
		{"", "empty"},
		{"-dev", "starts with a hyphen"},
		{"dev-", "ends with a hyphen"},
		{"dev_1", "underscore is not permitted"},
		{"dev.lab", "a dot is not permitted"},
		{"has space", "a space is not permitted"},
		{"abcdefghijklmnop", "sixteen characters, one over the limit"},
	} {
		if err := ValidateClusterName(bad.name); err == nil {
			t.Errorf("ValidateClusterName(%q) = nil, want an error (%s); pvecm create would "+
				"refuse it on the host, after the session is already open", bad.name, bad.why)
		}
	}
}

// THE PRECONDITION IS A LIVE AGENT, not a provisioning state.
//
// This gate used to refuse any host that was not `provisioned`, because "every
// step of a bootstrap runs over SSH as the control plane". Neither half is true
// now: the agent creates the cluster itself and the control plane opens no
// connection at all. The old rule refused a host whose agent was enrolled and
// heartbeating because a later onboarding step — proving the control plane's own
// managed SSH key — had lost a reconnect race.
func TestBootstrapGatesOnTheAgentNotOnSSHProvisioning(t *testing.T) {
	// Onboarding never finished proving the managed key, but the agent is live.
	// The agent is what runs `pvecm create`, so this must be allowed.
	rs := CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardFailed,
		AgentOnline:       true,
	})
	if len(rs) != 0 {
		t.Fatalf("a host with a live agent was refused for the control plane's own SSH state: %v",
			bootstrapCodes(rs))
	}

	// No agent is a real refusal: nothing would create the cluster.
	rs = CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardProvisioned,
		AgentOnline:       false,
	})
	if !slices.Contains(bootstrapCodes(rs), RefuseBootstrapUnprovisioned) {
		t.Fatalf("a host with no live agent was allowed to create a cluster: %v",
			bootstrapCodes(rs))
	}

	// And an unfinished onboarding is still SAID, as context on the refusal
	// rather than as the refusal itself.
	rs = CheckBootstrap(BootstrapRequest{
		TargetHive:        Hive{Name: "dev"},
		ProvisioningState: OnboardPending,
		AgentOnline:       false,
	})
	var detail string
	for _, r := range rs {
		if r.Code == RefuseBootstrapUnprovisioned {
			detail = r.Detail
		}
	}
	if !strings.Contains(detail, OnboardPending) {
		t.Fatalf("the refusal does not mention the unfinished onboarding: %q", detail)
	}
}
