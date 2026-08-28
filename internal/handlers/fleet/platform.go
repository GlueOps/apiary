package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/agentdist"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/buildinfo"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type HiveView struct {
	ID       string `json:"id" format:"uuid"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// BeeView is one AGENT, with the machine it runs on named alongside it.
//
// hive_id and address are facts about the MACHINE and are read from its
// Hypervisor row, not from this one. They used to be columns on the bee, which
// meant one machine had two answers to "which cluster is this in" with nothing
// keeping them in agreement — and they demonstrably diverged during a migration.
// There is one answer now; this view just reports it beside the agent so an
// operator does not have to cross-reference two lists.
type BeeView struct {
	ID string `json:"id" format:"uuid"`

	// HypervisorID is the machine this agent runs on. Absent only if the link is
	// broken, which should not happen — enrolment creates both rows in one
	// transaction — and is worth being able to see rather than hiding.
	HypervisorID string `json:"hypervisor_id,omitempty" format:"uuid"`

	// Absent when the machine belongs to no fleet, which is a real state rather
	// than an error: every migration passes through it, and a failed join leaves
	// a host sitting there. omitempty rather than a zero uuid, so a client
	// cannot mistake 00000000-... for a fleet it should go looking for.
	HiveID            string `json:"hive_id,omitempty" format:"uuid"`
	Address           string `json:"address,omitempty" doc:"How the control plane reaches the machine this agent runs on."`
	Status            string `json:"status" enum:"online,unavailable,offline" doc:"Derived, not stored. 'unavailable' means the host DECLARED it would be away — a reboot in progress is not an outage."`
	AgentVersion      string `json:"agent_version,omitempty"`
	LastSeenAt        string `json:"last_seen_at,omitempty" format:"date-time"`
	UnavailableUntil  string `json:"unavailable_until,omitempty" format:"date-time"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
}

type CreateHiveRequest struct {
	Body struct {
		Name        string `json:"name" required:"true" minLength:"1" maxLength:"100"`
		Provider    string `json:"provider,omitempty" enum:"proxmox" default:"proxmox"`
		Description string `json:"description,omitempty" maxLength:"1000"`
	}
}

type DeleteHiveRequest struct {
	ID string `path:"id" format:"uuid"`
}

type DeleteHiveResponse struct{}

type CreateHiveResponse struct{ Body HiveView }

type MintTokenRequest struct {
	Body struct {
		// IGNORED. Accepted only so a deployed client that still sends it keeps
		// working.
		//
		// It used to be REQUIRED and validated to exist with a 404, and then never
		// read: the row this handler writes carries the organization and nothing
		// else, because a host enrols into an ORGANIZATION and joins a fleet
		// afterwards as a task. The effect was that a cluster had to exist before
		// any agent could be provisioned — backwards, since onboarding is how a
		// machine reaches the point of being able to join one at all, and the
		// fleetless state a new host lands in is the same one every migrating host
		// passes through.
		//
		// Not deleted outright because huma rejects an unknown body field with 422
		// BEFORE the handler runs (pinned by
		// TestFleetOpsRejectsAnOrganizationSmuggledInTheBody). Removing the field
		// while ui/src/pages/fleet/enrolment-dialog.tsx still sends it would turn
		// every mint into a hard failure. Delete it in the same pass that stops the
		// UI sending it.
		HiveID     string `json:"hive_id,omitempty" format:"uuid" deprecated:"true" doc:"Ignored, and removed in a future release. A token admits a host into the ORGANIZATION; it never names a cluster."`
		TTLMinutes int    `json:"ttl_minutes,omitempty" minimum:"1" maximum:"1440" default:"60"`
		MaxUses    int    `json:"max_uses,omitempty" minimum:"0" maximum:"1000" doc:"0 means unlimited within the TTL. A rack is enrolled in a batch, so single-use tokens become a reason to script around the control."`
	}
}

type MintTokenResponse struct {
	Body struct {
		Token     string `json:"token" doc:"Returned once. Only its hash is stored — a leak of the table must not yield a token that can enrol a machine into the fleet."`
		ExpiresAt string `json:"expires_at" format:"date-time"`
		// BaseURL is the control plane address the agent should call, so the
		// enrolment command can be pasted rather than assembled. The agent
		// defaults to http://localhost:8080, which on a hypervisor is the
		// hypervisor — the one address that is always wrong. Empty when the
		// control plane does not know its own external address, and the caller
		// then has to supply --base-url by hand.
		BaseURL string `json:"base_url" doc:"The address the agent should enrol against. Empty if the control plane has no absolute BASE_URL configured, in which case --base-url must be supplied by hand."`
	}
}

type ListBeesRequest struct {
	HiveID string `query:"hive_id" format:"uuid" required:"false"`
}

type ListBeesResponse struct {
	Body struct {
		Bees []BeeView `json:"bees"`
	}
}

type DeleteBeeRequest struct {
	ID string `path:"id" format:"uuid"`
}

type DeleteBeeResponse struct {
	Body struct {
		Detail string `json:"detail"`

		// The machine this agent was running on, which SURVIVES. Returned because
		// it is where the operator goes next: reprovisioning is an operation on
		// the machine record, not on the agent that has just been removed.
		HypervisorID string `json:"hypervisor_id,omitempty" format:"uuid"`
	}
}

type CreateTaskRequest struct {
	Body struct {
		BeeID string          `json:"bee_id" required:"true" format:"uuid"`
		Kind  string          `json:"kind" required:"true" enum:"fleet.host.ping,fleet.host.dist-upgrade,fleet.host.agent-upgrade,fleet.datacenter.join,fleet.datacenter.evict" doc:"What the agent should do. An agent handed a kind it does not implement fails the task naming its version, rather than silently doing nothing."`
		Spec  json.RawMessage `json:"spec,omitempty" doc:"Desired state, not a script. NEVER put a credential here — specs are stored as jsonb and retained after the task completes; the agent reads secrets such as the cluster join password from its own host."`
	}
}

type CreateTaskResponse struct {
	Body struct {
		ID     string `json:"id" format:"uuid"`
		Status string `json:"status"`
	}
}

type TaskLogsRequest struct {
	ID    string `path:"id" format:"uuid"`
	Since int64  `query:"since" required:"false" doc:"Return lines with seq greater than this. The UI polls with the highest seq it has, so a tail is one small query rather than a re-read."`
}

// StoredTaskLogLine is what the UI reads. Named for the same reason
// TaskLogLine is — an anonymous struct becomes an unnamed schema, and the log
// viewer would be written against a type called Item2.
type StoredTaskLogLine struct {
	Seq      int64  `json:"seq"`
	Stream   string `json:"stream" enum:"stdout,stderr,agent"`
	Message  string `json:"message"`
	LoggedAt string `json:"logged_at" format:"date-time"`
}

type TaskLogsResponse struct {
	Body struct {
		Lines []StoredTaskLogLine `json:"lines"`
	}
}

func rfc3339(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// RegisterUserProtected mounts a customer's own fleet.
//
// Every hive belongs to an organization, so authorization here is by MEMBERSHIP,
// not by platform role: adding a hypervisor to your fleet is your business, the
// same way adding an estate is. Platform staff have no standing grant over it —
// a support engineer who must act on a customer's fleet does so by impersonating
// a member who consented, which is the one capability that role has.
//
// The organization is taken from the session on every endpoint and used as a
// query PREDICATE, never trusted from the request. That is the IDOR seam: a hive
// id, a bee id and a task id all arrive from the caller, and without the
// predicate any admin of any tenant could drain another tenant's hypervisors.
func RegisterUserProtected(hapi huma.API, deps *app.Deps) error {
	svc, err := newService(deps)
	if err != nil {
		return err
	}

	// The fleet lives in the caller's own tenant database, so a handle cannot be
	// taken once at mount time — which database it is depends on who is asking.
	// Resolving per request is also the tenancy control: the organization comes
	// from the authenticated principal and can never be named by the request, so
	// there is no path on which one tenant's admin reaches another's hypervisors.
	tenantDB := func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error) {
		if orgID == uuid.Nil {
			return nil, huma.Error403Forbidden("no organization resolved")
		}
		tc, err := deps.Tenants.GetTenantContext(ctx, orgID.String())
		if err != nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "this organization's storage is not ready yet")
		}
		return tc.DB, nil
	}
	// The enrolment token is the one fleet row still in control: it is redeemed
	// by a caller who has no organization yet.
	controlDB := deps.ControlDB

	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)

	// Reading the fleet is open to any member; changing it requires admin, and
	// refuses an impersonated session — an operator acting as a customer must not
	// be able to reboot that customer's hypervisors under their name.
	member := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleMember)
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}
	admin := func(ctx context.Context) (*apimiddleware.Principal, error) {
		p, err := apimiddleware.AssertOrgRole(ctx, memberships, control.MemberRoleAdmin,
			apimiddleware.RefuseImpersonation())
		if err != nil {
			return nil, apimiddleware.OrgRoleError(err)
		}
		return p, nil
	}

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-hives-create",
		Method:        http.MethodPost,
		Path:          "/fleet/hives",
		Summary:       "Create a hive",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Fleet"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateHiveRequest) (*CreateHiveResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		provider := in.Body.Provider
		if provider == "" {
			provider = "proxmox"
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		// Validated as a COROSYNC cluster name, because that is what it becomes
		// on the day this cluster acquires its first node: `pvecm create` is
		// given the hive's name. Accepting "Lab A" here only defers the refusal
		// to a worker, after the operator has bound a credential, chosen rings
		// and clicked the button — with the reason in a job log.
		name := strings.TrimSpace(in.Body.Name)
		if err := tenant.ValidateClusterName(name); err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		// No organization column: the database is the organization.
		hive := tenant.Hive{
			Name: name, Provider: provider, Description: in.Body.Description,
		}
		if err := db.WithContext(ctx).Create(&hive).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		auditOK(ctx, db, p, tenant.AuditHiveCreate, tenant.AuditTargetHive, &hive.ID,
			map[string]any{"name": hive.Name, "provider": hive.Provider})
		return &CreateHiveResponse{Body: HiveView{ID: hive.ID.String(), Name: hive.Name, Provider: hive.Provider}}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hives-delete",
		Method:      http.MethodDelete,
		Path:        "/fleet/hives/{id}",
		Summary:     "Delete a hive",
		Description: "Refuses while the cluster still holds hypervisors. A hypervisor row is the " +
			"durable record of a physical machine — its machine id, its onboarding history, the " +
			"agent bound to it — so deleting the cluster out from under one would either destroy " +
			"that or leave it pointing at a cluster that no longer exists.\n\n" +
			"Depart the nodes first; each one becomes fleetless and keeps its identity, and the " +
			"cluster can then be removed.",
		DefaultStatus: http.StatusNoContent,
		Tags:          []string{"Fleet"},
		Security:      apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeleteHiveRequest) (*DeleteHiveResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hive id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		// Counted and deleted in ONE transaction. Checked outside it, a node could
		// be adopted into this cluster by a capacity sync between the count and the
		// delete, and the delete would then take the cluster out from under a live
		// machine record.
		// Captured outside the transaction because the audit row is written after
		// it, by which point the cluster is gone and its id resolves to nothing.
		var hiveName string
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var hive tenant.Hive
			if err := tx.First(&hive, "id = ?", id).Error; err != nil {
				return err
			}
			hiveName = hive.Name
			var members int64
			if err := tx.Model(&tenant.Hypervisor{}).
				Where("hive_id = ?", id).Count(&members).Error; err != nil {
				return err
			}
			if members > 0 {
				return huma.Error409Conflict(fmt.Sprintf(
					"this cluster still holds %d hypervisor(s); depart them first, which leaves "+
						"each machine record intact and fleetless", members))
			}
			// Swarms are scoped to the hive and mean nothing without it, so they go
			// with it. Hypervisors reference them by a composite key, and that is
			// exactly why the count above has to come first: with a member still
			// present this delete would be refused by the foreign key rather than by
			// a message anyone can act on.
			if err := tx.Where("hive_id = ?", id).Delete(&tenant.Swarm{}).Error; err != nil {
				return err
			}
			return tx.Delete(&tenant.Hive{}, "id = ?", id).Error
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, huma.Error404NotFound("no such cluster")
			}
			var se huma.StatusError
			if errors.As(err, &se) {
				// A refused delete is worth a row: someone tried to remove a
				// cluster that still holds machines, and an incident review reads
				// the absence of that as nobody having tried.
				auditRefused(ctx, db, p, tenant.AuditHiveDelete, tenant.AuditTargetHive, &id,
					err.Error(), map[string]any{"name": hiveName})
				return nil, err
			}
			return nil, mapErr(ctx, err)
		}
		// Recorded AFTER the delete, so the row describes something that happened
		// rather than something attempted. The cluster's name is in the metadata
		// because the id it refers to no longer resolves to anything.
		auditOK(ctx, db, p, tenant.AuditHiveDelete, tenant.AuditTargetHive, &id,
			map[string]any{"name": hiveName})
		return &DeleteHiveResponse{}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-enrolment-tokens-mint",
		Method:        http.MethodPost,
		Path:          "/fleet/enrolment-tokens",
		Summary:       "Mint an enrolment token",
		DefaultStatus: http.StatusCreated,
		Description: "Admits a host into the ORGANIZATION, not into a cluster. Joining one is a task, " +
			"so a brand-new host starts in the same fleetless state a migrating host passes through — " +
			"which means an agent can be provisioned before any cluster exists.\n\n" +
			"Impersonated sessions are refused: an operator acting as somebody else must not be able " +
			"to mint something that admits a machine to the fleet.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *MintTokenRequest) (*MintTokenResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		// No tenant database is opened and no hive is looked up. Both used to
		// happen and neither was ever used — see MintTokenRequest.HiveID. The
		// organization comes from the principal, and it is the only thing the row
		// below records.
		ttl := in.Body.TTLMinutes
		if ttl <= 0 {
			ttl = 60
		}

		raw, hash, err := svc.NewEnrolmentToken()
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		expires := time.Now().UTC().Add(time.Duration(ttl) * time.Minute)
		// Admits into the ORGANIZATION, not a hive. Joining a fleet is a task, so
		// a new host starts where a migrating host passes through.
		row := control.BeeEnrolmentToken{
			OrganizationID: p.OrganizationID, TokenHash: hash, MaxUses: in.Body.MaxUses,
			ExpiresAt: expires, CreatedByID: p.UserID,
		}
		if err := controlDB.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &MintTokenResponse{}
		out.Body.Token = raw
		out.Body.ExpiresAt = expires.Format(time.RFC3339)
		out.Body.BaseURL = agentBaseURL(deps.Config)
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-tasks-create",
		Method:        http.MethodPost,
		Path:          "/fleet/tasks",
		Summary:       "Queue work for a bee",
		DefaultStatus: http.StatusCreated,
		Description: "Refused with 409 while that bee already has an outstanding task. At most one at a time is what makes 'what is this hypervisor doing' answerable from a single row.\n\n" +
			"fleet.datacenter.evict is issued to a node that will REMAIN, naming the one to remove — a node cannot remove itself from a cluster it is leaving.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CreateTaskRequest) (*CreateTaskResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		beeID, err := uuid.Parse(in.Body.BeeID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid bee id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		if err := assertBeeExists(ctx, db, beeID); err != nil {
			return nil, err
		}
		spec := in.Body.Spec
		if len(spec) == 0 {
			spec = json.RawMessage(`{}`)
		}

		// GATED HERE, so the API, Terraform and the UI all get the same answer.
		//
		// The agent used to decide this alone by asking the Proxmox cluster API —
		// which is why a dist-upgrade needed a token for a job that is entirely
		// local. Every fact the decision needs is in this database, and refusing at
		// creation puts the reason in front of the operator who asked instead of in
		// a task that fails minutes later.
		//
		// Only dist-upgrade: it is the one kind that can interrupt what a host is
		// running. Ping, join and evict have their own gates or none.
		if in.Body.Kind == "fleet.host.dist-upgrade" {
			refusals, gerr := repo.CheckUpgradeForBee(ctx, db, beeID, spec,
				deps.Config.FleetObservationMaxAge, time.Now().UTC())
			if gerr != nil {
				return nil, mapErr(ctx, gerr)
			}
			if len(refusals) > 0 {
				return nil, refusalConflict("this host cannot be upgraded yet",
					refusalViews(refusals))
			}
		}

		task := tenant.FleetTask{
			BeeID: beeID, Kind: in.Body.Kind, Spec: datatypes.JSON(spec),
			Status: tenant.FleetTaskPending, CreatedByID: p.UserID,
		}
		if err := repo.NewFleetRepoGorm(db).CreateTask(ctx, &task); err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &CreateTaskResponse{}
		out.Body.ID = task.ID.String()
		out.Body.Status = task.Status
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-agent-upgrade",
		Method:        http.MethodPost,
		Path:          "/fleet/agent-upgrades",
		Summary:       "Upgrade every agent in a fleet",
		DefaultStatus: http.StatusCreated,
		Description: "Queues one upgrade task per agent. Each host downloads the binary this " +
			"control plane serves, verifies it against the advertised digest, proves it runs, " +
			"keeps the previous one, and restarts.\n\n" +
			"ROLLING BY CONSTRUCTION: a bee may hold at most one task, so no host does two " +
			"things at once. Hosts do proceed in parallel with each other, which is safe here " +
			"because an agent is not in the data path of anything it manages — an agent that " +
			"fails to come back costs control of that host, not its guests.\n\n" +
			"Agents already on the target version, offline, or holding another task are SKIPPED " +
			"and each says why. Queueing an offline agent would hand it a task it collects " +
			"whenever it returns, which is often during the next incident.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *AgentUpgradeRequest) (*AgentUpgradeResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		var hiveID *uuid.UUID
		if in.Body.HiveID != "" {
			id, perr := uuid.Parse(in.Body.HiveID)
			if perr != nil {
				return nil, huma.Error422UnprocessableEntity("invalid fleet id")
			}
			hiveID = &id
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		// The control plane's OWN version is the target. These binaries shipped
		// with it, so there is nothing else it could honestly upgrade an agent to
		// — and letting a caller name a version would mean advertising one this
		// server cannot serve.
		target := buildinfo.Version
		dist, derr := agentdist.Catalogue(target)
		if derr != nil || len(dist.Builds) == 0 {
			return nil, huma.Error409Conflict("this control plane carries no agent binaries, so " +
				"there is nothing for an agent to upgrade to")
		}

		// Every digest this control plane can serve. An agent running any of them
		// is current, whatever version anybody stamped on it.
		digests := make(map[string]bool, len(dist.Builds))
		for _, b := range dist.Builds {
			digests[b.SHA256] = true
		}

		results, err := repo.QueueAgentUpgrades(ctx, db, hiveID, target, digests, p.UserID,
			in.Body.Force, time.Now().UTC(), 2*time.Minute)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &AgentUpgradeResponse{}
		out.Body.Version = target
		out.Body.Agents = make([]UpgradeCandidateView, 0, len(results))
		for _, r := range results {
			if r.Queued {
				out.Body.Queued++
			}
			out.Body.Agents = append(out.Body.Agents, UpgradeCandidateView{
				BeeID: r.BeeID, AgentVersion: r.Version, Queued: r.Queued, Skipped: r.Skipped,
			})
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bees-list",
		Method:      http.MethodGet,
		Path:        "/fleet/bees",
		Summary:     "List fleet agents",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ListBeesRequest) (*ListBeesResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		// No organization predicate: the database IS the organization.
		//
		// A LEFT join, and that is the whole point. Fleet membership and the
		// address both live on the machine record now, so filtering or ordering by
		// either has to reach it — but an INNER join would drop every agent whose
		// machine is in no cluster, which is precisely the set an operator opens
		// this list to find during a migration.
		q := db.WithContext(ctx).Model(&tenant.Bee{}).
			Select("bees.*").
			Joins("LEFT JOIN hypervisors ON hypervisors.bee_id = bees.id").
			Order("hypervisors.address ASC")
		if in.HiveID != "" {
			id, err := uuid.Parse(in.HiveID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid hive id")
			}
			q = q.Where("hypervisors.hive_id = ?", id)
		}
		var rows []tenant.Bee
		if err := q.Find(&rows).Error; err != nil {
			return nil, mapErr(ctx, err)
		}

		machines, err := machinesByBee(ctx, db, rows)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &ListBeesResponse{}
		out.Body.Bees = make([]BeeView, 0, len(rows))
		for i := range rows {
			b := &rows[i]
			v := BeeView{
				ID: b.ID.String(),
				// Derived here rather than read from the column, so a late sweep
				// cannot leave an operator staring at a stale status mid-rotation.
				Status:            service.BeeLiveness(b),
				AgentVersion:      b.AgentVersion,
				LastSeenAt:        rfc3339(b.LastSeenAt),
				UnavailableUntil:  rfc3339(b.UnavailableUntil),
				UnavailableReason: b.UnavailableReason,
			}
			if m, ok := machines[b.ID]; ok {
				v.HypervisorID = m.ID.String()
				v.HiveID = hiveIDOf(m)
				v.Address = m.Address
			}
			out.Body.Bees = append(out.Body.Bees, v)
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bees-delete",
		Method:      http.MethodDelete,
		Path:        "/fleet/bees/{id}",
		Summary:     "Remove a fleet agent",
		Description: "Revokes this agent's credential and then deletes its record. The MACHINE " +
			"survives as agentless — its cluster membership, address and onboarding history are on " +
			"the hypervisor record, not on the agent — so this is 'remove the agent', never 'forget " +
			"the box'.\n\n" +
			"The credential is revoked FIRST and the revocation is verified before anything is " +
			"deleted. It lives in the control database while the agent record lives in the tenant " +
			"database, so no transaction covers both: deleting first and failing to revoke would " +
			"leave a live agent key with nothing pointing at it, and nothing lists agent-scoped keys.\n\n" +
			"Refused with 409 while an onboarding worker holds a lease on this machine — it is " +
			"installing and enrolling right now, and would either race the delete or resurrect the " +
			"record behind it.\n\n" +
			"The agent process on the host is NOT stopped. It keeps running and starts failing " +
			"authentication, which is the intended outcome: the credential is what this revokes.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeleteBeeRequest) (*DeleteBeeResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		beeID, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid agent id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		hypID, err := deleteAgent(ctx, db, controlDB, p.OrganizationID, beeID)
		if err != nil {
			return nil, err
		}
		auditOK(ctx, db, p, tenant.AuditAgentRemove, tenant.AuditTargetBee, &beeID,
			map[string]any{"hypervisor_id": hypID})

		out := &DeleteBeeResponse{}
		out.Body.HypervisorID = hypID
		out.Body.Detail = "agent removed and its credential revoked; the machine record survives"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-tasks-logs",
		Method:      http.MethodGet,
		Path:        "/fleet/tasks/{id}/logs",
		Summary:     "Read a task's streamed output",
		Description: "Pass `since` with the highest seq already held to tail cheaply.",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *TaskLogsRequest) (*TaskLogsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid task id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		// The task id comes from the path; existence in this database is what
		// stops one tenant reading another's rotation output.
		var owned int64
		if err := db.WithContext(ctx).Model(&tenant.FleetTask{}).
			Where("id = ?", id).
			Count(&owned).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		if owned == 0 {
			return nil, huma.Error404NotFound("no such task in this organization")
		}
		var rows []tenant.FleetTaskLog
		if err := db.WithContext(ctx).
			Where("task_id = ? AND seq > ?", id, in.Since).
			Order("seq ASC").Limit(2000).Find(&rows).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &TaskLogsResponse{}
		for _, r := range rows {
			out.Body.Lines = append(out.Body.Lines, StoredTaskLogLine{
				Seq: r.Seq, Stream: r.Stream, Message: r.Message,
				LoggedAt: r.LoggedAt.UTC().Format(time.RFC3339),
			})
		}
		return out, nil
	})
	return nil
}

// machinesByBee loads the machine record behind each agent, keyed on bee id.
//
// A second query rather than more columns on the join above: the join exists to
// filter and order, and a machine has enough of its own fields that scanning it
// into a flattened row is how the two rows start being confused for one again.
func machinesByBee(ctx context.Context, db *gorm.DB, bees []tenant.Bee) (map[uuid.UUID]*tenant.Hypervisor, error) {
	out := make(map[uuid.UUID]*tenant.Hypervisor, len(bees))
	if len(bees) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(bees))
	for i := range bees {
		ids = append(ids, bees[i].ID)
	}
	var machines []tenant.Hypervisor
	if err := db.WithContext(ctx).Where("bee_id IN ?", ids).Find(&machines).Error; err != nil {
		return nil, err
	}
	for i := range machines {
		m := &machines[i]
		if m.BeeID != nil {
			out[*m.BeeID] = m
		}
	}
	return out, nil
}

// hiveIDOf renders a machine's fleet, or "" when it belongs to none.
//
// HiveID is a POINTER and nil for a machine between fleets. Calling String() on
// it panics — uuid.UUID.String has a value receiver, so the nil pointer is
// dereferenced — and it would panic on exactly the hosts an operator opens the
// listing to find during a migration. It was on the bee when that was hit; it
// moved to the machine record with the column.
func hiveIDOf(h *tenant.Hypervisor) string {
	if h == nil || h.HiveID == nil {
		return ""
	}
	return h.HiveID.String()
}

// assertBeeExists checks a bee is in THIS database, which is the whole of the
// ownership question now.
//
// The bee id arrives in a request body, so this is the IDOR seam for every fleet
// task: without it, an admin of any tenant could queue a dist-upgrade against
// another tenant's hypervisor. A bee that belongs to someone else answers as
// not-found, identical to one that does not exist, so the endpoint cannot be used
// to discover another organization's fleet.
//
// It used to join through hives to compare an organization column. That check has
// moved: the handle passed in was resolved from the authenticated principal, so a
// bee in another tenant is not merely rejected, it is not visible. The join would
// also have been actively wrong — an INNER join on a nullable hive_id drops every
// host that has left one fleet and not yet joined the next, which is precisely
// when someone is looking for it.
func assertBeeExists(ctx context.Context, db *gorm.DB, beeID uuid.UUID) error {
	var n int64
	err := db.WithContext(ctx).
		Model(&tenant.Bee{}).
		Where("id = ?", beeID).
		Count(&n).Error
	if err != nil {
		return huma.Error500InternalServerError("could not check fleet ownership")
	}
	if n == 0 {
		return huma.Error404NotFound("no such agent in this organization")
	}
	return nil
}

// deleteAgent revokes one agent's credential and then removes its record.
//
// THE ORDER IS THE WHOLE DESIGN. The credential is a control.APIKey in the
// CONTROL database; the agent row is a tenant.Bee in the TENANT database. No
// transaction spans the two, so one of the halves must be allowed to fail with
// the other already done, and the two orderings are not equally bad:
//
//   - delete first, revoke fails -> a usable agent key exists with nothing
//     pointing at it. It cannot be found again either: /api-keys deliberately
//     filters agent-scoped keys out of its listing, and the only record of which
//     key belonged to which host was the row just deleted.
//   - revoke first, delete fails -> a revoked credential on a still-visible
//     agent. Visible, harmless, and the request can simply be repeated.
//
// So: revoke, VERIFY the revocation, then delete. The verification is not
// ceremony — it is what makes the ordering mean anything, since an UPDATE that
// matched no row reports success.
//
// Returns the id of the machine this agent ran on, which survives the delete
// (hypervisors.bee_id is ON DELETE SET NULL). Under the old shape the machine
// WAS the cluster's node row and deleting the agent would have destroyed the
// only record of a host the cluster still had; that is no longer true, which is
// why membership is not a refusal here.
func deleteAgent(
	ctx context.Context, tdb, cdb *gorm.DB, orgID, beeID uuid.UUID,
) (string, error) {
	var bee tenant.Bee
	if err := tdb.WithContext(ctx).First(&bee, "id = ?", beeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Identical to "belongs to another tenant", because the handle was
			// resolved from the principal and another tenant's agent is not in this
			// database to be found. So this endpoint cannot be used to discover
			// somebody else's fleet.
			return "", huma.Error404NotFound("no such agent in this organization")
		}
		return "", mapErr(ctx, err)
	}

	// The machine record, read before the delete: bee_id is SET NULL on the way
	// out, so afterwards there is nothing left to join on.
	var machine tenant.Hypervisor
	haveMachine := true
	if err := tdb.WithContext(ctx).First(&machine, "bee_id = ?", beeID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", mapErr(ctx, err)
		}
		// An agent with no machine record should not happen — enrolment writes
		// both — but it must not block removing the agent, which is exactly the
		// cleanup somebody would be reaching for.
		haveMachine = false
	}

	if haveMachine && provisioningLeaseHeld(machine, time.Now().UTC()) {
		return "", refusalConflict("this agent cannot be removed yet", []RefusalView{{
			Code: refusalProvisioningLeaseHeld,
			Detail: "an onboarding worker holds a lease on this machine and is installing and " +
				"enrolling it right now; removing the agent under it either races the install or " +
				"leaves a record the worker recreates. Wait for the lease to expire.",
		}})
	}

	if err := revokeAgentKey(ctx, cdb, orgID, bee.APIKeyID); err != nil {
		return "", err
	}

	if err := tdb.WithContext(ctx).Delete(&tenant.Bee{}, "id = ?", beeID).Error; err != nil {
		return "", mapErr(ctx, err)
	}
	if haveMachine {
		return machine.ID.String(), nil
	}
	return "", nil
}

// revokeAgentKey revokes one agent credential and proves it is no longer usable.
//
// SCOPED BY ORGANIZATION, which the shared APIKeyRepoGorm.Revoke is not: it
// updates by id alone. The id here comes out of a tenant database row, so in a
// healthy system it always belongs to this organization — but "always" is doing
// the work of an access control, and the predicate costs one clause. A row that
// somehow names another tenant's key now revokes nothing instead of revoking
// theirs.
//
// The read-back is what makes the caller's revoke-before-delete ordering a
// guarantee rather than a hope. An UPDATE matching no row succeeds, so
// RowsAffected == 0 covers two very different cases: the key is already gone
// (fine — there is nothing left to revoke) and the key exists under another
// organization (not fine — it is live and this request must not proceed to
// delete the only record of it). Counting the rows that are still usable
// separates them without needing to know which happened.
func revokeAgentKey(ctx context.Context, cdb *gorm.DB, orgID, keyID uuid.UUID) error {
	if keyID == uuid.Nil {
		// A zero id would match any row whose column was never set. Refusing is
		// right rather than skipping: an agent row with no credential id is a
		// broken row, and deleting it would hide the breakage.
		return huma.Error500InternalServerError("this agent has no credential recorded")
	}
	if err := cdb.WithContext(ctx).Model(&control.APIKey{}).
		Where("id = ? AND org_id = ?", keyID, orgID).
		Update("revoked", true).Error; err != nil {
		return mapErr(ctx, err)
	}

	var live int64
	if err := cdb.WithContext(ctx).Model(&control.APIKey{}).
		Where("id = ? AND revoked = ?", keyID, false).
		Count(&live).Error; err != nil {
		return mapErr(ctx, err)
	}
	if live > 0 {
		slog.ErrorContext(ctx, "refusing to delete an agent whose credential is still usable",
			"api_key_id", keyID, "organization_id", orgID)
		return huma.Error500InternalServerError(
			"this agent's credential could not be revoked, so it was not removed")
	}
	return nil
}

// AgentUpgradeRequest asks for a fleet's agents to be upgraded.
type AgentUpgradeRequest struct {
	Body struct {
		HiveID string `json:"hive_id,omitempty" format:"uuid" doc:"Limit to one fleet. Omit to cover every agent in the organization, including hosts in no fleet — which are the ones most likely to be forgotten."`
		Force  bool   `json:"force,omitempty" doc:"Queue agents that already report the target version. For repairing a damaged binary, not for routine use: it restarts every agent in scope."`
	}
}

type UpgradeCandidateView struct {
	BeeID        string `json:"bee_id"`
	AgentVersion string `json:"agent_version,omitempty" doc:"What this agent last reported running."`
	Queued       bool   `json:"queued"`
	Skipped      string `json:"skipped,omitempty" doc:"Why this agent was left alone. Per agent rather than a count, because \"14 of 20 queued\" leaves the operator to work out which six."`
}

type AgentUpgradeResponse struct {
	Body struct {
		Version string                 `json:"version" doc:"The version agents are being moved to, which is this control plane's own."`
		Queued  int                    `json:"queued"`
		Agents  []UpgradeCandidateView `json:"agents"`
	}
}
