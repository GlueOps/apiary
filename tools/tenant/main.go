package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/glueops/apiary/internal/models/tenant"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&tenant.Credential{},
		&tenant.Estate{},
		&tenant.SshKey{},

		// The fleet. These moved out of control: the agent credential that admits
		// a host is org-scoped, and everything a fleet operation touches belongs
		// to the customer who owns the metal.
		&tenant.Hive{},
		&tenant.Bee{},
		&tenant.FleetTask{},
		&tenant.FleetTaskLog{},

		// Swarm before the firewall models: a FirewallBinding may be scoped to a
		// swarm and holds a composite foreign key into this table.
		&tenant.Swarm{},

		&tenant.FirewallRuleset{},
		// Sets before rules: a rule may hold a foreign key to one.
		&tenant.FirewallSet{},
		&tenant.FirewallSetElement{},
		&tenant.FirewallRule{},
		&tenant.FirewallBinding{},
		&tenant.FirewallSnapshot{},

		// The placement oracle, folded in from waggle. Hypervisor hangs off Hive
		// rather than a parallel Datacenter, and Placement carries hive_id so the
		// per-cluster VMID uniqueness can be expressed on the row.
		&tenant.Slot{},
		&tenant.Hypervisor{},
		&tenant.Pool{},
		&tenant.Placement{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.WriteString(os.Stdout, stmts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write schema: %v\n", err)
		os.Exit(1)
	}
}
