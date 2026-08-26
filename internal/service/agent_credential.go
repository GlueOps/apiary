package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
)

// AgentProxmoxCredential returns the API token the agent on one machine should
// hold.
//
// THE HIVE'S CREDENTIAL, and that is forced rather than chosen. Proxmox keeps
// API tokens in /etc/pve/priv/token.cfg, which lives on pmxcfs and is therefore
// replicated across the whole cluster: a token minted for one node exists on
// every node, a cluster join REPLACES that database and destroys whatever the
// joining node had, and a departure carries the cluster's copy away with it. A
// per-host credential cannot survive the lifecycle this system automates.
//
// A machine in NO hive returns an empty credential and no error. That is a real
// state — every migrating host passes through it — and there is no cluster whose
// token would be the right one. The agent then refuses cluster tasks and says
// why, which is correct: there is no cluster to act on.
func (s *ProxmoxInventoryService) AgentProxmoxCredential(
	ctx context.Context, orgID, hypervisorID uuid.UUID,
) (tokenID, secret, apiURL string, err error) {
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return "", "", "", fmt.Errorf("agent credential: open the tenant database: %w", err)
	}

	var hv tenant.Hypervisor
	if err := tCtx.DB.WithContext(ctx).First(&hv, "id = ?", hypervisorID).Error; err != nil {
		return "", "", "", fmt.Errorf("agent credential: load the machine: %w", err)
	}
	if hv.HiveID == nil {
		return "", "", "", nil
	}
	return s.AgentProxmoxCredentialForHive(ctx, orgID, *hv.HiveID)
}

// AgentProxmoxCredentialForHive returns the token an agent in one fleet should
// hold.
//
// Named by FLEET rather than by machine because arrival needs it before the
// machine belongs to that fleet. A join replaces the node's token database with
// the target cluster's — see the note on pmxcfs above — so the credential the
// agent must end up with is the one for the cluster it is JOINING, at a moment
// when its own row still says it is fleetless.
func (s *ProxmoxInventoryService) AgentProxmoxCredentialForHive(
	ctx context.Context, orgID, hiveID uuid.UUID,
) (tokenID, secret, apiURL string, err error) {
	tCtx, err := s.tenants.GetTenantContext(ctx, orgID.String())
	if err != nil {
		return "", "", "", fmt.Errorf("agent credential: open the tenant database: %w", err)
	}

	var hive tenant.Hive
	if err := tCtx.DB.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
		return "", "", "", fmt.Errorf("agent credential: load the fleet: %w", err)
	}
	if hive.CredentialID == nil {
		// A fleet with no credential recorded. Not an error for the same reason as
		// no hive: the onboarding proceeds, and the agent says what is missing.
		return "", "", "", nil
	}

	dek, err := utils.DeriveKey(tCtx.Key, utils.KeyPurposeTenantDEK)
	if err != nil {
		return "", "", "", fmt.Errorf("agent credential: derive the tenant data key: %w", err)
	}
	creds, err := repo.NewCredentialRepoGorm(tCtx.DB, orgID, utils.LegacyKeyID, dek, s.scopes)
	if err != nil {
		return "", "", "", err
	}
	cred, err := creds.Get(ctx, *hive.CredentialID)
	if err != nil {
		return "", "", "", fmt.Errorf("agent credential: load the credential for fleet %q: %w",
			hive.Name, err)
	}
	if strings.TrimSpace(cred.Identity) == "" {
		// The identity is the token id and is stored in plaintext on purpose — the
		// secret alone cannot authenticate. A credential without one is unusable
		// and saying so is more use than handing the agent half of it.
		return "", "", "", fmt.Errorf("agent credential: the credential for fleet %q has no "+
			"identity recorded, so the API token id is unknown; rotate it to record one",
			hive.Name)
	}
	opened, err := creds.OpenSecret(ctx, cred)
	if err != nil {
		// Deliberately carries no credential material and no more of the error than
		// OpenSecret says: an unseal failure is exactly when somebody pastes the
		// message into a ticket.
		return "", "", "", fmt.Errorf("agent credential: unseal the credential for fleet %q: %w",
			hive.Name, err)
	}

	// The hive's API URL is NOT passed on. It is how the CONTROL PLANE reaches
	// the cluster — often one node's address or a VIP — and the agent runs on the
	// hypervisor itself, where the right answer is its own loopback. Handing it
	// the control plane's view would make every agent in a fleet talk to one node.
	return cred.Identity, string(opened), "", nil
}
