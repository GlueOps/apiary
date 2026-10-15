package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedDeclaredHost is a record as an operator declares it and the onboarding
// ticker claims it: an address, no machine id, no agent, provisioning started.
func seedDeclaredHost(t *testing.T, db *gorm.DB, address string) tenant.Hypervisor {
	t.Helper()
	now := time.Now().UTC()
	hv := tenant.Hypervisor{Address: address, ProvisioningStartedAt: &now}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatalf("create hypervisor: %v", err)
	}
	return hv
}

// TestADualStackHostOnboardedOnIPv4IsOneRecord.
//
// Seen in production: seven hosts declared on IPv4 and onboarded over SSH
// enrolled their agents from IPv6, because the control plane's hostname has an
// AAAA record. Enrolment matches by machine id, then by address, and neither
// matched, so every host became two records: the declared one with no agent,
// and a new one on the IPv6 address holding it.
func TestADualStackHostOnboardedOnIPv4IsOneRecord(t *testing.T) {
	db := fleetDB(t)
	hv := seedDeclaredHost(t, db, "95.216.2.244")
	raw := uuid.NewString()

	if err := ClaimMachineID(context.Background(), db, hv.ID, raw); err != nil {
		t.Fatalf("claim: %v", err)
	}
	res, err := NewFleetRepoGorm(db).EnrolBee(context.Background(), EnrolBeeInput{
		MachineID:   tenant.MachineIDDigest(raw),
		Address:     "pve1",
		PeerAddress: "2a01:4f9:2a:1be0::2",
		APIKeyID:    uuid.New(),
		Now:         time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("enrol: %v", err)
	}

	m := machineFor(t, db, res.Bee.ID)
	if m.ID != hv.ID {
		t.Fatalf("the agent landed on a new record %s instead of the onboarded one %s; the "+
			"declared host is left with no agent to act on it", m.ID, hv.ID)
	}
	if m.Address != "95.216.2.244" {
		t.Fatalf("address became %q: enrolment replaced the address onboarding reached the host at", m.Address)
	}
	var n int64
	if err := db.Model(&tenant.Hypervisor{}).
		Where("address IN ?", []string{"95.216.2.244", "2a01:4f9:2a:1be0::2"}).
		Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d records for one host, want 1", n)
	}
}

func TestClaimMachineIDIsIdempotent(t *testing.T) {
	db := fleetDB(t)
	hv := seedDeclaredHost(t, db, "95.216.40.47")
	raw := uuid.NewString()

	for i := range 2 {
		if err := ClaimMachineID(context.Background(), db, hv.ID, raw); err != nil {
			t.Fatalf("claim %d: %v; a retried onboard must be able to claim again", i+1, err)
		}
	}
}

// The id already on another record means the host is in the fleet twice.
// Moving the id would only move the duplicate, so an operator has to choose.
func TestClaimMachineIDRefusesAnIDAnotherRecordHolds(t *testing.T) {
	db := fleetDB(t)
	raw := uuid.NewString()
	digest := tenant.MachineIDDigest(raw)
	if err := db.Create(&tenant.Hypervisor{Address: "2a01:4f9:2b:d95::2", MachineID: &digest}).Error; err != nil {
		t.Fatal(err)
	}
	hv := seedDeclaredHost(t, db, "95.216.46.50")

	err := ClaimMachineID(context.Background(), db, hv.ID, raw)
	if !errors.Is(err, ErrMachineIDHeldElsewhere) {
		t.Fatalf("got %v, want ErrMachineIDHeldElsewhere", err)
	}
	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.MachineID != nil {
		t.Fatal("the refused claim still wrote the machine id")
	}
}

// A record that already belongs to one machine, whose address now answers as
// another, is not this host's record.
func TestClaimMachineIDRefusesARecordOfAnotherMachine(t *testing.T) {
	db := fleetDB(t)
	theirs := tenant.MachineIDDigest(uuid.NewString())
	now := time.Now().UTC()
	hv := tenant.Hypervisor{Address: "95.216.27.160", MachineID: &theirs, ProvisioningStartedAt: &now}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	err := ClaimMachineID(context.Background(), db, hv.ID, uuid.NewString())
	if !errors.Is(err, ErrMachineIDMismatch) {
		t.Fatalf("got %v, want ErrMachineIDMismatch", err)
	}
}
