package fleet

import (
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Two schemas, because the two halves of removing an agent live in two
// DATABASES and the whole point of the ordering is that no transaction covers
// both. A single handle behind both would make the failure this suite is about
// — a delete that lands while the revoke does not — unreachable.
var (
	lifecycleTenantSchema  = testsupport.SchemaName("apiary_fleet_life_tenant")
	lifecycleControlSchema = testsupport.SchemaName("apiary_fleet_life_ctl")
)

func lifecycleDBs(t *testing.T) (tdb, cdb *gorm.DB) {
	t.Helper()
	tdb = testsupport.OpenSchema(t, lifecycleTenantSchema,
		&tenant.Hive{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.FleetTask{}, &tenant.FleetTaskLog{})
	cdb = testsupport.OpenSchema(t, lifecycleControlSchema, &control.APIKey{})

	// Rows, not tables: bees.api_key_id is unique and hypervisors carry partial
	// unique indexes, so a row left by an earlier test fails the next one for a
	// reason that says nothing about the code under test.
	if err := tdb.Exec(
		`TRUNCATE fleet_task_logs, fleet_tasks, hypervisors, bees, hives RESTART IDENTITY CASCADE`,
	).Error; err != nil {
		t.Fatalf("truncate tenant tables: %v", err)
	}
	if err := cdb.Exec(`TRUNCATE api_keys RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate control tables: %v", err)
	}
	return tdb, cdb
}

// seedAgentKey writes one agent credential in the control database.
func seedAgentKey(t *testing.T, cdb *gorm.DB, orgID uuid.UUID) uuid.UUID {
	t.Helper()
	key := control.APIKey{
		OrgID:   &orgID,
		Scope:   control.APIKeyScopeAgent,
		Name:    "agent-" + uuid.NewString()[:8],
		KeyHash: uuid.NewString(),
	}
	if err := cdb.Create(&key).Error; err != nil {
		t.Fatalf("seed agent key: %v", err)
	}
	return key.ID
}

// seedAgent writes a bee and the machine record pointing at it.
func seedAgent(t *testing.T, tdb *gorm.DB, keyID uuid.UUID, hv tenant.Hypervisor) (uuid.UUID, uuid.UUID) {
	t.Helper()
	bee := tenant.Bee{APIKeyID: keyID}
	if err := tdb.Create(&bee).Error; err != nil {
		t.Fatalf("seed bee: %v", err)
	}
	hv.BeeID = &bee.ID
	if hv.Address == "" {
		hv.Address = "10.0.0." + uuid.NewString()[:2]
	}
	if err := tdb.Create(&hv).Error; err != nil {
		t.Fatalf("seed hypervisor: %v", err)
	}
	// provisioning_state and the lease carry `default:` tags, so a struct create
	// cannot write anything but the default into them. The explicit UPDATE is the
	// only form that works — the same trap the cordon handler documents.
	if err := tdb.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
		Updates(map[string]any{
			"provisioning_state":            hv.ProvisioningState,
			"provisioning_lease_expires_at": hv.ProvisioningLeaseExpiresAt,
			"provisioning_attempt":          hv.ProvisioningAttempt,
		}).Error; err != nil {
		t.Fatalf("set provisioning columns: %v", err)
	}
	return bee.ID, hv.ID
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error and got none")
	}
	se, ok := err.(huma.StatusError)
	if !ok {
		t.Fatalf("expected a huma status error, got %T: %v", err, err)
	}
	return se.GetStatus()
}

func keyIsRevoked(t *testing.T, cdb *gorm.DB, keyID uuid.UUID) bool {
	t.Helper()
	var k control.APIKey
	if err := cdb.First(&k, "id = ?", keyID).Error; err != nil {
		t.Fatalf("read back the key: %v", err)
	}
	return k.Revoked
}

func beeExists(t *testing.T, tdb *gorm.DB, beeID uuid.UUID) bool {
	t.Helper()
	var n int64
	if err := tdb.Model(&tenant.Bee{}).Where("id = ?", beeID).Count(&n).Error; err != nil {
		t.Fatalf("count bees: %v", err)
	}
	return n > 0
}

// Removing an agent must revoke its credential, drop its record and its task
// history, and LEAVE THE MACHINE — agentless, still in its cluster, still
// carrying its onboarding state.
func TestDeleteAgentRevokesTheCredentialAndKeepsTheMachine(t *testing.T) {
	tdb, cdb := lifecycleDBs(t)
	org := uuid.New()

	keyID := seedAgentKey(t, cdb, org)
	beeID, hvID := seedAgent(t, tdb, keyID, tenant.Hypervisor{
		Address:           "10.9.9.9",
		ProvisioningState: tenant.OnboardProvisioned,
	})

	task := tenant.FleetTask{BeeID: beeID, Kind: "fleet.host.ping", CreatedByID: uuid.New()}
	if err := tdb.Create(&task).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}

	gotHV, err := deleteAgent(t.Context(), tdb, cdb, org, beeID)
	if err != nil {
		t.Fatalf("deleteAgent: %v", err)
	}
	if gotHV != hvID.String() {
		t.Fatalf("returned hypervisor id %q, want %q — an operator is sent here to reprovision, "+
			"so naming the wrong machine sends them to the wrong box", gotHV, hvID)
	}
	if !keyIsRevoked(t, cdb, keyID) {
		t.Fatal("the agent's credential is still usable after its record was deleted; nothing " +
			"lists agent-scoped keys, so it can never be found again")
	}
	if beeExists(t, tdb, beeID) {
		t.Fatal("the agent record survived the delete")
	}

	var hv tenant.Hypervisor
	if err := tdb.First(&hv, "id = ?", hvID).Error; err != nil {
		t.Fatalf("the MACHINE must survive removing its agent: %v", err)
	}
	if hv.BeeID != nil {
		t.Fatalf("the machine still points at the deleted agent: %v", *hv.BeeID)
	}
	if hv.ProvisioningState != tenant.OnboardProvisioned {
		t.Fatalf("provisioning_state = %q, want it untouched at %q", hv.ProvisioningState,
			tenant.OnboardProvisioned)
	}

	var tasks int64
	if err := tdb.Model(&tenant.FleetTask{}).Where("bee_id = ?", beeID).Count(&tasks).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if tasks != 0 {
		t.Fatalf("%d task rows still reference the deleted agent", tasks)
	}
}

// THE ORDERING TEST. If the credential cannot be revoked, the agent record must
// still be there afterwards — it is the only thing that records which key
// belongs to which host, and /api-keys deliberately does not list agent keys.
//
// Driven by a key belonging to ANOTHER organization, which is also the check
// that revocation is org-scoped: APIKeyRepoGorm.Revoke updates by id alone, and
// using it here would revoke the other tenant's credential and report success.
func TestDeleteAgentKeepsTheRecordWhenTheCredentialCannotBeRevoked(t *testing.T) {
	tdb, cdb := lifecycleDBs(t)
	org, victim := uuid.New(), uuid.New()

	keyID := seedAgentKey(t, cdb, victim)
	beeID, _ := seedAgent(t, tdb, keyID, tenant.Hypervisor{
		Address:           "10.9.9.8",
		ProvisioningState: tenant.OnboardProvisioned,
	})

	_, err := deleteAgent(t.Context(), tdb, cdb, org, beeID)
	if err == nil {
		t.Fatal("deleting an agent whose credential could not be revoked must fail")
	}
	if got := statusOf(t, err); got != 500 {
		t.Fatalf("status %d, want 500", got)
	}
	if keyIsRevoked(t, cdb, keyID) {
		t.Fatal("another organization's credential was revoked")
	}
	if !beeExists(t, tdb, beeID) {
		t.Fatal("the agent record was deleted while its credential stayed live, which is the " +
			"one outcome the revoke-first ordering exists to prevent: a usable agent key with " +
			"nothing pointing at it and no listing that shows it")
	}
}

// A worker mid-onboard is inside this machine right now. Removing the agent
// under it either races the install or has the worker recreate the record.
func TestDeleteAgentRefusesWhileAnOnboardingLeaseIsLive(t *testing.T) {
	tdb, cdb := lifecycleDBs(t)
	org := uuid.New()

	live := time.Now().UTC().Add(5 * time.Minute)
	keyID := seedAgentKey(t, cdb, org)
	beeID, _ := seedAgent(t, tdb, keyID, tenant.Hypervisor{
		Address:                    "10.9.9.7",
		ProvisioningState:          tenant.OnboardProvisioning,
		ProvisioningLeaseExpiresAt: &live,
	})

	_, err := deleteAgent(t.Context(), tdb, cdb, org, beeID)
	if got := statusOf(t, err); got != 409 {
		t.Fatalf("status %d, want 409 — a refusal is the gate working, not a server error", got)
	}
	if keyIsRevoked(t, cdb, keyID) {
		t.Fatal("a refused delete revoked the credential anyway, which breaks a healthy agent " +
			"the request did not remove")
	}
	if !beeExists(t, tdb, beeID) {
		t.Fatal("a refused delete removed the agent anyway")
	}
}

// An EXPIRED lease is not held. This is the recovery path for a worker killed
// mid-onboard, and a gate that could not tell the two apart would strand the
// machine permanently.
func TestDeleteAgentProceedsOnceTheLeaseHasExpired(t *testing.T) {
	tdb, cdb := lifecycleDBs(t)
	org := uuid.New()

	stale := time.Now().UTC().Add(-time.Minute)
	keyID := seedAgentKey(t, cdb, org)
	beeID, _ := seedAgent(t, tdb, keyID, tenant.Hypervisor{
		Address:                    "10.9.9.6",
		ProvisioningState:          tenant.OnboardProvisioning,
		ProvisioningLeaseExpiresAt: &stale,
	})

	if _, err := deleteAgent(t.Context(), tdb, cdb, org, beeID); err != nil {
		t.Fatalf("an expired lease holds nothing and must not block removal: %v", err)
	}
	if !keyIsRevoked(t, cdb, keyID) {
		t.Fatal("the credential was not revoked")
	}
}

// An agent id from another tenant must be indistinguishable from one that does
// not exist, or the endpoint enumerates somebody else's fleet.
func TestDeleteAgentAnswersNotFoundForAnAgentThatIsNotHere(t *testing.T) {
	tdb, cdb := lifecycleDBs(t)
	if _, err := deleteAgent(t.Context(), tdb, cdb, uuid.New(), uuid.New()); statusOf(t, err) != 404 {
		t.Fatalf("status %d, want 404", statusOf(t, err))
	}
}

// redrivablePredicate is evaluated inside the UPDATE that acts on it, so this
// drives the real statement rather than a Go reimplementation of the rule.
//
// The attempt counter is the marker. Every row starts at 7 and the update sets
// it to 0, which is what makes an untouched row distinguishable from a row that
// was rewritten to the state it already had.
func TestRedriveOnlyMovesRowsNoWorkerHolds(t *testing.T) {
	tdb, _ := lifecycleDBs(t)

	past := time.Now().UTC().Add(-time.Minute)
	future := time.Now().UTC().Add(10 * time.Minute)

	cases := []struct {
		name  string
		state string
		lease *time.Time
		moved bool
	}{
		{"failed", tenant.OnboardFailed, nil, true},
		{"provisioned is redrivable — that is 'revive'", tenant.OnboardProvisioned, nil, true},
		{"provisioning with an expired lease is the crash-recovery path",
			tenant.OnboardProvisioning, &past, true},
		{"provisioning with a lease that was never stamped holds nothing",
			tenant.OnboardProvisioning, nil, true},
		{"provisioning under a live lease must NOT move",
			tenant.OnboardProvisioning, &future, false},
		{"already pending must NOT be silently rewritten", tenant.OnboardPending, nil, false},
	}

	ids := make([]uuid.UUID, len(cases))
	for i, c := range cases {
		hv := tenant.Hypervisor{Address: "10.8.0." + uuid.NewString()[:2]}
		if err := tdb.Create(&hv).Error; err != nil {
			t.Fatalf("seed %s: %v", c.name, err)
		}
		if err := tdb.Model(&tenant.Hypervisor{}).Where("id = ?", hv.ID).
			Updates(map[string]any{
				"provisioning_state":            c.state,
				"provisioning_lease_expires_at": c.lease,
				"provisioning_attempt":          7,
			}).Error; err != nil {
			t.Fatalf("set provisioning columns for %s: %v", c.name, err)
		}
		ids[i] = hv.ID
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			moved, err := redriveOnboarding(t.Context(), tdb, ids[i])
			if err != nil {
				t.Fatalf("redrive: %v", err)
			}
			if moved != c.moved {
				t.Fatalf("moved = %v, want %v", moved, c.moved)
			}

			var hv tenant.Hypervisor
			if err := tdb.First(&hv, "id = ?", ids[i]).Error; err != nil {
				t.Fatalf("read back: %v", err)
			}
			if c.moved {
				if hv.ProvisioningState != tenant.OnboardPending {
					t.Fatalf("state = %q, want %q", hv.ProvisioningState, tenant.OnboardPending)
				}
				if hv.ProvisioningAttempt != 0 {
					t.Fatalf("attempt = %d, want 0: a redrive that keeps the count is a no-op for "+
						"the one case it exists for, a machine stopped at the cap",
						hv.ProvisioningAttempt)
				}
				if hv.ProvisioningLeaseExpiresAt != nil {
					t.Fatal("a pending row nobody holds still carries a lease")
				}
				return
			}
			if hv.ProvisioningState != c.state {
				t.Fatalf("state = %q, want it untouched at %q", hv.ProvisioningState, c.state)
			}
			if hv.ProvisioningAttempt != 7 {
				t.Fatalf("attempt = %d, want it untouched at 7 — the row was rewritten despite "+
					"the predicate refusing it", hv.ProvisioningAttempt)
			}
		})
	}
}

// The prose half of the same rule. Pure, so the reasons an operator is shown can
// be pinned without a database.
func TestReprovisionRefusalsExplainEveryRefusal(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Minute)
	future := now.Add(10 * time.Minute)

	cases := []struct {
		name string
		hv   tenant.Hypervisor
		want []string
	}{
		{
			name: "failed and addressable",
			hv:   tenant.Hypervisor{Address: "10.0.0.1", ProvisioningState: tenant.OnboardFailed},
		},
		{
			name: "provisioned is revivable",
			hv:   tenant.Hypervisor{Address: "10.0.0.1", ProvisioningState: tenant.OnboardProvisioned},
		},
		{
			name: "expired lease is reclaimable",
			hv: tenant.Hypervisor{Address: "10.0.0.1", ProvisioningState: tenant.OnboardProvisioning,
				ProvisioningLeaseExpiresAt: &past},
		},
		{
			name: "live lease",
			hv: tenant.Hypervisor{Address: "10.0.0.1", ProvisioningState: tenant.OnboardProvisioning,
				ProvisioningLeaseExpiresAt: &future},
			want: []string{refusalProvisioningLeaseHeld},
		},
		{
			name: "already queued",
			hv:   tenant.Hypervisor{Address: "10.0.0.1", ProvisioningState: tenant.OnboardPending},
			want: []string{refusalAlreadyPending},
		},
		{
			name: "no address to connect to",
			hv:   tenant.Hypervisor{ProvisioningState: tenant.OnboardFailed},
			want: []string{refusalNoAddress},
		},
		{
			name: "both reasons are reported, not just the first",
			hv:   tenant.Hypervisor{ProvisioningState: tenant.OnboardPending},
			want: []string{refusalNoAddress, refusalAlreadyPending},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reprovisionRefusals(c.hv, now)
			codes := make([]string, 0, len(got))
			for _, r := range got {
				codes = append(codes, r.Code)
				if r.Detail == "" {
					t.Fatalf("refusal %q carries no explanation", r.Code)
				}
			}
			if len(codes) != len(c.want) {
				t.Fatalf("refusals %v, want %v", codes, c.want)
			}
			for i := range codes {
				if codes[i] != c.want[i] {
					t.Fatalf("refusals %v, want %v", codes, c.want)
				}
			}
		})
	}
}
