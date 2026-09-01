package repo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/tenant"
)

// A SUCCESS MAY RETIRE ITS OWN PREVIOUS FAILURE AND NOBODY ELSE'S.
//
// last_operation is a single slot, so a successful agent upgrade that cleared it
// unconditionally would erase a disarm failure nobody has resolved — replacing a
// true statement with silence, on a machine still holding the credentials of a
// cluster it has left. That is the exposure the whole departure sequence exists
// to close, and the fleet view is the only place it is visible.
func TestASuccessClearsOnlyItsOwnFailure(t *testing.T) {
	db := factsDB(t)
	ctx := context.Background()

	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	hv := tenant.Hypervisor{BeeID: &bee.ID, Address: "192.0.2.41"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	RecordTaskOutcomeForBee(ctx, db, bee.ID, KindDisarm,
		"/etc/pve/priv/pve-root-ca.srl is still byte-identical to the copy this node left with")

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.LastOperation != "disarm" {
		t.Fatalf("last_operation = %q, want the word an operator reads, not a wire kind",
			got.LastOperation)
	}
	if !strings.Contains(got.LastOperationError, "byte-identical") {
		t.Fatalf("last_operation_error = %q, want the agent's own account", got.LastOperationError)
	}

	// An unrelated success must leave it alone.
	RecordTaskOutcomeForBee(ctx, db, bee.ID, KindAgentUpgrade, "")

	var after tenant.Hypervisor
	if err := db.First(&after, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.LastOperationError == "" {
		t.Fatal("an agent upgrade succeeded and erased an unresolved disarm failure, leaving a " +
			"host that still holds a departed cluster's credentials looking clean")
	}

	// The same operation succeeding does retire it.
	RecordTaskOutcomeForBee(ctx, db, bee.ID, KindDisarm, "")

	var cleared tenant.Hypervisor
	if err := db.First(&cleared, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if cleared.LastOperationError != "" {
		t.Fatalf("the disarm succeeded and its own earlier failure is still shown: %q",
			cleared.LastOperationError)
	}
}

// An unlabelled kind records nothing rather than putting a wire constant in a
// sentence meant for a person.
func TestAnUnlabelledTaskKindRecordsNothing(t *testing.T) {
	if got := OperationNameForTaskKind("fleet.host.something-new"); got != "" {
		t.Fatalf("OperationNameForTaskKind = %q, want empty so it is added deliberately", got)
	}

	db := factsDB(t)
	bee := tenant.Bee{APIKeyID: uuid.New(), Status: tenant.BeeStatusOnline, EnrolledAt: time.Now().UTC()}
	if err := db.Create(&bee).Error; err != nil {
		t.Fatal(err)
	}
	hv := tenant.Hypervisor{BeeID: &bee.ID, Address: "192.0.2.42"}
	if err := db.Create(&hv).Error; err != nil {
		t.Fatal(err)
	}

	RecordTaskOutcomeForBee(context.Background(), db, bee.ID, "fleet.host.something-new", "it broke")

	var got tenant.Hypervisor
	if err := db.First(&got, "id = ?", hv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.LastOperation != "" || got.LastOperationError != "" {
		t.Fatalf("an unlabelled kind wrote %q/%q; the fleet view would render a wire constant "+
			"as prose", got.LastOperation, got.LastOperationError)
	}
}
