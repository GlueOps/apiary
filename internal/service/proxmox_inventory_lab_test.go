package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/pkg/proxmox"
)

// TestProxmoxInventoryAgainstALiveCluster checks the mapping against a real API.
//
// The unit tests assert what MapProxmoxResources does with rows I wrote. This
// asserts the rows are the shape Proxmox actually sends — which is where the
// units live, and where the CPU-versus-MaxCPU trap would show up as a fleet
// reporting zero cores.
//
// Read-only. It calls /cluster/resources and asserts nothing about state it
// changes, because it changes none.
//
//	APIARY_LAB_API=https://10.10.10.4:8006/api2/json \
//	APIARY_LAB_TOKEN_ID='root@pam!apiary-sync' APIARY_LAB_TOKEN_SECRET=... \
//	  go test ./internal/service/ -run TestProxmoxInventoryAgainstALiveCluster -v
func TestProxmoxInventoryAgainstALiveCluster(t *testing.T) {
	api := os.Getenv("APIARY_LAB_API")
	tokenID := os.Getenv("APIARY_LAB_TOKEN_ID")
	secret := os.Getenv("APIARY_LAB_TOKEN_SECRET")
	if api == "" || tokenID == "" || secret == "" {
		t.Skip("APIARY_LAB_API / APIARY_LAB_TOKEN_ID / APIARY_LAB_TOKEN_SECRET are not set — " +
			"SKIPPING; the Proxmox inventory mapping was NOT checked against a real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := proxmox.NewClient(api,
		proxmox.WithAPIToken(tokenID, secret),
		// The lab ships a self-signed certificate, as every un-fronted Proxmox
		// install does. Mirrors Hive.InsecureSkipVerify, which is per-hive and off
		// by default for the reason stated there.
		proxmox.WithInsecureSkipVerify(),
	)

	cluster, err := client.Cluster(ctx)
	if err != nil {
		t.Fatalf("reach the cluster: %v", err)
	}
	resources, err := cluster.Resources(ctx)
	if err != nil {
		t.Fatalf("read cluster resources: %v", err)
	}
	t.Logf("ok  /cluster/resources returned %d row(s)", len(resources))

	rows := MapProxmoxResources(resources)
	obs := tenant.FoldClusterResources(rows, nil)
	if len(obs) == 0 {
		t.Fatal("no nodes observed — the mapping produced nothing from a live cluster")
	}

	var sawNode bool
	for name, o := range obs {
		t.Logf("    %-8s cpu=%d ram=%dGiB disk=%dGiB guests=%d online=%v",
			name, o.CPUTotal, o.RAMGBTotal, o.DiskGBTotal, o.Guests, o.Online)
		if o.Online {
			sawNode = true
			// The trap worth catching against a real API: reading the `CPU`
			// utilisation ratio instead of MaxCPU gives zero on any idle node, and
			// a fleet reporting zero cores is unschedulable everywhere.
			if o.CPUTotal <= 0 {
				t.Errorf("node %q reports %d cores — a live node has at least one, so the "+
					"mapping is reading a utilisation ratio rather than a core count",
					name, o.CPUTotal)
			}
			if o.RAMGBTotal <= 0 {
				t.Errorf("node %q reports %d GiB of memory", name, o.RAMGBTotal)
			}
		}
	}
	if !sawNode {
		t.Fatal("no node read as online, though the API answered")
	}
}
