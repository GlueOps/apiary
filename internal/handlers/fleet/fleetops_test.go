package fleet

import (
	"errors"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/google/uuid"
)

func TestRefusalConflictIsA409CarryingEveryCode(t *testing.T) {
	// 409 rather than 422: the request is well-formed and would be valid at
	// another moment. And never 500 — a refusal is the gate working, and
	// reporting an operator's own drain policy as a server error puts it on
	// their 5xx dashboard.
	err := refusalConflict("cannot leave yet", []RefusalView{
		{Code: tenant.RefuseGuestsPresent, Detail: "2 guests are still here"},
		{Code: tenant.RefuseHiveTooSmall, Detail: "this hive has 2 nodes"},
	})

	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("want a huma status error, got %T", err)
	}
	if se.GetStatus() != 409 {
		t.Fatalf("want 409, got %d", se.GetStatus())
	}

	// Both codes must survive into the response. A client that keys on the code
	// to decide what to offer next — a sync for drain_unknown, nothing for
	// hive_too_small — cannot do that if only the first is reported.
	body, ok := se.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("want an ErrorModel, got %T", se)
	}
	if len(body.Errors) != 2 {
		t.Fatalf("want 2 detail entries, got %d", len(body.Errors))
	}
	locations := []string{body.Errors[0].Location, body.Errors[1].Location}
	for _, want := range []string{tenant.RefuseGuestsPresent, tenant.RefuseHiveTooSmall} {
		found := false
		for _, l := range locations {
			if l == want {
				found = true
			}
		}
		if !found {
			t.Errorf("code %q must appear in the response, got %v", want, locations)
		}
	}
}

func TestRefusalViewsPreserveCodesVerbatim(t *testing.T) {
	// The codes are a contract with the UI, which disables buttons and offers
	// remedies based on them. A translation layer that renamed one would break
	// the client silently, so this asserts they pass through untouched.
	in := []tenant.DepartureRefusal{
		{Code: tenant.RefuseNotCordoned, Detail: "a"},
		{Code: tenant.RefuseDrainUnknown, Detail: "b"},
		{Code: tenant.RefusePlacementsHeld, Detail: "c"},
	}
	got := refusalViews(in)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	for i := range in {
		if got[i].Code != in[i].Code || got[i].Detail != in[i].Detail {
			t.Errorf("entry %d changed: %+v -> %+v", i, in[i], got[i])
		}
	}
}

func TestArrivalRefusalViewsPreserveCodes(t *testing.T) {
	got := arrivalRefusalViews([]tenant.ArrivalRefusal{
		{Code: tenant.RefuseAlreadyInHive, Detail: "x"},
	})
	if len(got) != 1 || got[0].Code != tenant.RefuseAlreadyInHive {
		t.Fatalf("codes must pass through, got %+v", got)
	}
}

func TestHiveDetailResolvesTheMinimumRatherThanExposingTheRawZero(t *testing.T) {
	// Zero means "unset, use the default", and a client that rendered the raw
	// value would tell an operator their hive permits a departure at zero nodes.
	// Both are returned so the UI can show what was configured AND what applies.
	h := tenant.Hive{ID: uuid.New(), Name: "dc1", Provider: "proxmox"}
	v := hiveDetail(h, 4)
	if v.MinimumNodes != 0 {
		t.Errorf("the configured value passes through: want 0, got %d", v.MinimumNodes)
	}
	if v.EffectiveMinimumNodes != tenant.DefaultHiveMinimumNodes {
		t.Errorf("the effective value must resolve the default: want %d, got %d",
			tenant.DefaultHiveMinimumNodes, v.EffectiveMinimumNodes)
	}
	if v.NodeCount != 4 {
		t.Errorf("node count: want 4, got %d", v.NodeCount)
	}
}

func TestHiveDetailReportsWhetherASyncCanRunAtAll(t *testing.T) {
	// A hive with no API URL or credential is never synced, so every node in it
	// reads as never-observed and every departure refuses with drain_unknown.
	// That is a configuration problem, and the operator needs to see it as one
	// rather than as a fleet that mysteriously cannot be drained.
	cid := uuid.New()
	cases := []struct {
		name string
		hive tenant.Hive
		want bool
	}{
		{"nothing configured", tenant.Hive{Name: "a"}, false},
		{"url only", tenant.Hive{Name: "b", APIURL: "https://x/api2/json"}, false},
		{"credential only", tenant.Hive{Name: "c", CredentialID: &cid}, false},
		{"both", tenant.Hive{Name: "d", APIURL: "https://x/api2/json", CredentialID: &cid}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hiveDetail(c.hive, 1).Configured; got != c.want {
				t.Fatalf("configured: want %v, got %v", c.want, got)
			}
		})
	}
}

func TestHiveDetailNeverEmitsANilCredentialAsTheZeroUUID(t *testing.T) {
	// The same trap the bee view has: rendering an absent uuid as
	// 00000000-... sends a client looking for a credential that does not exist.
	v := hiveDetail(tenant.Hive{Name: "a"}, 0)
	if v.CredentialID != "" {
		t.Fatalf("an unbound credential must be empty, got %q", v.CredentialID)
	}
	if strings.Contains(v.CredentialID, "0000") {
		t.Fatal("the nil uuid must never be serialised as a credential id")
	}
}

func TestHiveDetailResolvesTheOvercommitRatio(t *testing.T) {
	// A stored zero would read as "this hive has no CPU at all", which makes
	// every host unschedulable and reports a full datacenter as empty.
	v := hiveDetail(tenant.Hive{Name: "a"}, 0)
	if v.CPUOvercommitRatio != tenant.DefaultCPUOvercommitRatio {
		t.Fatalf("want the default %v, got %v", tenant.DefaultCPUOvercommitRatio, v.CPUOvercommitRatio)
	}
}

func TestCapacityViewCarriesAllThreeDimensions(t *testing.T) {
	got := capacityView(tenant.Capacity{CPU: 8, RAMGB: 32, DiskGB: 500})
	if got.CPU != 8 || got.RAMGB != 32 || got.DiskGB != 500 {
		t.Fatalf("capacity mangled: %+v", got)
	}
}
