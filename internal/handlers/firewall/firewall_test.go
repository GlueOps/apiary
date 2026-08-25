package firewall

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

// A swarm-scoped binding carries hive_id TOO, so a view that only exposes
// hive_id renders it identically to a whole-cluster binding. The blast-radius
// screen exists to be believed, and it would then report a ruleset reaching
// thirty hypervisors when it reaches ten — or the reverse once a fleet binding
// and a swarm binding sit side by side in the same list.
func TestBindingViewDistinguishesASwarmBindingFromAFleetOne(t *testing.T) {
	hive := uuid.New()
	swarm := uuid.New()
	ruleset := tenant.FirewallRuleset{ID: uuid.New(), Name: "management"}

	t.Run("org-wide", func(t *testing.T) {
		v := toBindingView(tenant.FirewallBinding{ID: uuid.New(), RulesetID: ruleset.ID, Ruleset: ruleset})
		if v.HiveID != "" || v.SwarmID != "" {
			t.Fatalf("org-wide binding rendered as hive=%q swarm=%q, want both empty", v.HiveID, v.SwarmID)
		}
	})

	t.Run("one fleet", func(t *testing.T) {
		v := toBindingView(tenant.FirewallBinding{ID: uuid.New(), HiveID: &hive, RulesetID: ruleset.ID, Ruleset: ruleset})
		if v.HiveID != hive.String() {
			t.Fatalf("hive_id = %q, want %q", v.HiveID, hive)
		}
		if v.SwarmID != "" {
			t.Fatalf("swarm_id = %q on a fleet-scoped binding, want empty", v.SwarmID)
		}
	})

	t.Run("one swarm", func(t *testing.T) {
		v := toBindingView(tenant.FirewallBinding{
			ID: uuid.New(), HiveID: &hive, SwarmID: &swarm, RulesetID: ruleset.ID, Ruleset: ruleset,
		})
		if v.SwarmID != swarm.String() {
			t.Fatalf("swarm_id = %q, want %q; without it on the wire a one-swarm ruleset is "+
				"indistinguishable from a whole-cluster one", v.SwarmID, swarm)
		}
		if v.HiveID != hive.String() {
			t.Fatalf("hive_id = %q, want %q; the scope is the PAIR", v.HiveID, hive)
		}
	})
}

// Every refusal here is a deliberate decision about something an operator typed.
// Reported as a 500 it tells the caller the system is broken and puts their own
// typo on a 5xx dashboard — which is exactly what happens to a new error that
// nobody added to this switch.
func TestSwarmRefusalsDoNotSurfaceAsServerErrors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{repo.ErrFirewallSwarmNotFound, http.StatusNotFound},
		{repo.ErrFirewallSwarmNeedsHive, http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		got := mapErr(context.Background(), c.err)
		var se huma.StatusError
		if !errors.As(got, &se) {
			t.Fatalf("%v mapped to %v, which carries no status", c.err, got)
		}
		if se.GetStatus() != c.status {
			t.Fatalf("%v mapped to %d, want %d", c.err, se.GetStatus(), c.status)
		}
	}
}
