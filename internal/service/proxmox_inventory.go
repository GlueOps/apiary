package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/glueops/apiary/pkg/proxmox"
	"github.com/google/uuid"
)

// ProxmoxInventoryService reads a hive's live inventory from its Proxmox API.
//
// The ONLY place in this system that knows Proxmox's wire format. Everything
// downstream — the capacity fold, the drain state, the departure and arrival
// gates — works on tenant.ResourceRow, so adding a second provider is a second
// mapper here rather than a change to any of the logic that decides things.
type ProxmoxInventoryService struct {
	tenants TenantResolver
	scopes  repo.ProviderScopeRegistry
}

func NewProxmoxInventoryService(t TenantResolver, scopes repo.ProviderScopeRegistry) *ProxmoxInventoryService {
	return &ProxmoxInventoryService{tenants: t, scopes: scopes}
}

// Inventory returns the hive's flat resource list, mapped into the neutral shape.
func (s *ProxmoxInventoryService) Inventory(ctx context.Context, orgID, hiveID uuid.UUID) ([]tenant.ResourceRow, error) {
	if s == nil {
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}

	var hive tenant.Hive
	if err := tCtx.DB.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
		return nil, fmt.Errorf("load hive %s: %w", hiveID, err)
	}
	if strings.TrimSpace(hive.APIURL) == "" {
		return nil, fmt.Errorf("hive %q has no API URL configured, so its capacity cannot be "+
			"read; set one before scheduling a sync", hive.Name)
	}
	if hive.CredentialID == nil {
		return nil, fmt.Errorf("hive %q has no credential bound, so its API cannot be called", hive.Name)
	}

	client, err := s.clientFor(ctx, tCtx, orgID, hive)
	if err != nil {
		return nil, err
	}

	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("reach the Proxmox cluster for hive %q: %w", hive.Name, err)
	}
	resources, err := cluster.Resources(ctx)
	if err != nil {
		return nil, fmt.Errorf("read cluster resources for hive %q: %w", hive.Name, err)
	}
	return MapProxmoxResources(resources), nil
}

// clientFor builds an authenticated Proxmox client for one hive.
func (s *ProxmoxInventoryService) clientFor(
	ctx context.Context, tCtx *database.TenantContext, orgID uuid.UUID, hive tenant.Hive,
) (*proxmox.Client, error) {
	dek, err := utils.DeriveKey(tCtx.Key, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, fmt.Errorf("derive tenant data key: %w", err)
	}
	creds, err := repo.NewCredentialRepoGorm(tCtx.DB, orgID, utils.LegacyKeyID, dek, s.scopes)
	if err != nil {
		return nil, err
	}
	cred, err := creds.Get(ctx, *hive.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("load the credential for hive %q: %w", hive.Name, err)
	}
	secret, err := creds.OpenSecret(ctx, cred)
	if err != nil {
		// Deliberately does not include the credential material or the error's
		// own detail beyond what OpenSecret says — an unseal failure is exactly
		// when someone pastes the message into a ticket.
		return nil, fmt.Errorf("unseal the credential for hive %q: %w", hive.Name, err)
	}
	if strings.TrimSpace(cred.Identity) == "" {
		return nil, fmt.Errorf("the credential for hive %q has no identity recorded, so the API "+
			"token id is unknown and the secret alone cannot authenticate; rotate the credential "+
			"to record it", hive.Name)
	}

	opts := []proxmox.Option{proxmox.WithAPIToken(cred.Identity, string(secret))}
	if hive.InsecureSkipVerify {
		// Per-hive and off by default. The cost is real: the sealed token is handed
		// to whatever answers on that address, and a token that can create and
		// destroy VMs is exactly what a machine-in-the-middle wants.
		opts = append(opts, proxmox.WithInsecureSkipVerify())
	}
	return proxmox.NewClient(strings.TrimRight(hive.APIURL, "/"), opts...), nil
}

// MapProxmoxResources translates Proxmox's inventory into the neutral shape.
//
// Exported and free of any client, so the mapping is testable against literal
// API rows — which is where the units matter. Proxmox reports memory and disk in
// BYTES and CPU as a core count in MaxCPU (the `CPU` field is a utilisation
// ratio, not a count, and using it would report every idle node as having no
// cores).
func MapProxmoxResources(rs proxmox.ClusterResources) []tenant.ResourceRow {
	out := make([]tenant.ResourceRow, 0, len(rs))
	for _, r := range rs {
		if r == nil {
			continue
		}
		out = append(out, tenant.ResourceRow{
			Type:   r.Type,
			Node:   r.Node,
			Status: r.Status,
			// MaxCPU, not CPU. MaxCPU is the configured core count; CPU is a 0..1
			// utilisation ratio that truncates to zero on anything not busy.
			// Through the same saturating clamp as the VMID. A wrapped core count
			// is a small positive number, so a node would report a handful of cores
			// and the placement oracle would treat a large host as nearly full —
			// or, wrapped the other way, hand out capacity that does not exist.
			CPUCores:  clampVMID(r.MaxCPU),
			MemBytes:  r.MaxMem,
			DiskBytes: r.MaxDisk,
			// Proxmox reports template as 0/1 rather than a bool.
			Template: r.Template != 0,
			VMID:     clampVMID(r.VMID),
		})
	}
	return out
}

// clampVMID narrows Proxmox's uint64 vmid to an int, SATURATING rather than
// wrapping.
//
// The guard is here rather than in pkg/proxmox because pkg/proxmox's own
// clamping helper is unexported and this is the only place the wire type is
// read. Saturating matters because the result is matched against the placement
// ledger to decide which guests are ours: a wrapped id is a small, entirely
// plausible number that could COLLIDE with a real VMID, which would mark a
// stranger's VM as ours and stop its capacity being subtracted at all — the
// exact failure the foreign accounting exists to close. A saturated id simply
// matches nothing, so the guest counts as foreign, which is the safe direction.
func clampVMID(v uint64) int {
	if v > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(v)
}

// NodeAddresses returns each cluster member's name mapped to the address the
// CLUSTER knows it by.
//
// A second call, because /cluster/resources — which Inventory reads — describes
// capacity and does not carry node addresses. /cluster/status does, and it is
// the only place this system can learn how to reach a node it did not onboard.
//
// That gap had a visible cost. The capacity sync creates a Hypervisor row for
// any member nobody onboarded here, with the cluster's name for it and no
// address, because it had none to record. Onboarding then looks a host up by
// address, misses that row entirely, and creates a SECOND one for the same
// physical machine — observed on a live four-node cluster, where pve4 and pve5
// each ended up with a name-only row from the sync and an address-only row from
// a later declaration.
//
// Best-effort by contract: a hive whose API is unreachable returns an error the
// caller is expected to tolerate. An address is an improvement to a record, not
// a precondition for observing capacity, and failing a whole sync because one
// optional field could not be filled would trade a cosmetic gap for a real one.
func (s *ProxmoxInventoryService) NodeAddresses(
	ctx context.Context, orgID, hiveID uuid.UUID,
) (map[string]string, error) {
	if s == nil {
		return nil, ErrTenantUnavailable
	}
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTenantUnavailable, err)
	}

	var hive tenant.Hive
	if err := tCtx.DB.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
		return nil, fmt.Errorf("load hive %s: %w", hiveID, err)
	}
	if strings.TrimSpace(hive.APIURL) == "" || hive.CredentialID == nil {
		return nil, fmt.Errorf("hive %q has no API URL or credential, so its node addresses "+
			"cannot be read", hive.Name)
	}

	client, err := s.clientFor(ctx, tCtx, orgID, hive)
	if err != nil {
		return nil, err
	}
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("reach the Proxmox cluster for hive %q: %w", hive.Name, err)
	}
	if err := cluster.Status(ctx); err != nil {
		return nil, fmt.Errorf("read cluster status for hive %q: %w", hive.Name, err)
	}

	out := make(map[string]string, len(cluster.Nodes))
	for _, n := range cluster.Nodes {
		if n == nil {
			continue
		}
		name := strings.TrimSpace(n.Name)
		ip := strings.TrimSpace(n.IP)
		if name == "" || ip == "" {
			continue
		}
		out[name] = ip
	}
	return out, nil
}
