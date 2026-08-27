package fleet

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefusalView is one reason an operation is not permitted right now.
//
// A stable machine-readable code alongside the prose, because the UI keys on the
// code to decide what to offer next — "sync now" for drain_unknown, nothing at
// all for hive_too_small — and prose is not a contract.
type RefusalView struct {
	// Code is the stable half — like an HTTP status, what a client switches on
	// and what an operator learns to recognise.
	Code string `json:"code" example:"guests_present"`
	// Summary states the rule generically, with no numbers and no host in it, so
	// refusals of the same rule across many machines can be shown once. Grouping
	// on Detail instead means showing one host's figures as if they applied to
	// every host in the group.
	Summary string `json:"summary,omitempty"`
	// Detail states what is wrong on THIS machine, and names the offending
	// number.
	Detail string `json:"detail"`
	// Remedy states what to do, separately, so it can be rendered where somebody
	// is looking for it rather than repeated inline to people who already know.
	Remedy string `json:"remedy,omitempty"`
}

func refusalViews(rs []tenant.DepartureRefusal) []RefusalView {
	out := make([]RefusalView, 0, len(rs))
	for _, r := range rs {
		out = append(out, RefusalView{Code: r.Code, Summary: r.Summary, Detail: r.Detail, Remedy: r.Remedy})
	}
	return out
}

func arrivalRefusalViews(rs []tenant.ArrivalRefusal) []RefusalView {
	out := make([]RefusalView, 0, len(rs))
	for _, r := range rs {
		out = append(out, RefusalView{Code: r.Code, Summary: r.Summary, Detail: r.Detail, Remedy: r.Remedy})
	}
	return out
}

// CapacityView is one triple of numbers.
type CapacityView struct {
	CPU    int `json:"cpu"`
	RAMGB  int `json:"ram_gb"`
	DiskGB int `json:"disk_gb"`
}

// HypervisorView is one node as an operator needs to see it.
//
// It carries the DEPARTURE REFUSALS alongside the state, computed on the same
// read. That is the whole reason this view is worth having: an operator deciding
// whether to drain a host should see why it cannot leave before they act, not
// after a job fails. It also means the UI never has to reimplement the gate —
// which is the way two definitions of a rule start.
type HypervisorView struct {
	ID string `json:"id" format:"uuid"`

	// Absent when this machine belongs to no cluster, which is a normal state
	// rather than an error: a machine being onboarded is in none, and every
	// migration passes through it. omitempty rather than a zero uuid, so a client
	// cannot mistake 00000000-... for a hive it should go looking for.
	HiveID   string `json:"hive_id,omitempty" format:"uuid"`
	HiveName string `json:"hive_name,omitempty"`

	// Absent for the same reason: a node name is the cluster's word for this
	// machine and does not exist before it joins one.
	Name string `json:"name,omitempty" doc:"The provider's node name, not a hostname we chose. Absent until this machine joins a cluster."`

	// Address is how the control plane reaches the machine. Present whether or
	// not it is in a cluster, which is the point — it is a fact about the box.
	Address string `json:"address,omitempty"`

	// THREE STATUS AXES, THREE FIELDS. A machine can be provisioned AND cordoned
	// AND offline at the same instant, and one badge cannot say that: it would
	// have to pick, and it would pick wrong exactly when an operator is looking.
	// provisioning_state is the only one that is stored — it is a workflow.
	// Schedulable plus drain_state is intent plus observation; bee_status is
	// derived from the agent's heartbeat and never stored at all.
	ProvisioningState   string `json:"provisioning_state" enum:"pending,provisioning,provisioned,failed" doc:"Has the agent been installed? Stored, because it is a workflow rather than an observation. Independent of whether the machine is cordoned or reachable."`
	ProvisioningAttempt int    `json:"provisioning_attempt,omitempty" doc:"How many onboarding runs this machine has cost. Capped; a machine at the cap needs an explicit redrive."`
	ProvisioningError   string `json:"provisioning_error,omitempty" doc:"The last onboarding failure, kept so 'failed' is actionable without going to the logs."`

	// Why the last fleet operation on this machine failed.
	//
	// Surfaced because a DEPARTURE has no state column of its own: it either
	// succeeds and the machine leaves its cluster, or it fails and the job retries
	// on a backoff with nothing on screen to say so. An operator then sees a
	// cordoned, drained node that will not leave and no reason anywhere they would
	// think to look.
	LastOperation      string `json:"last_operation,omitempty" doc:"Which operation this outcome belongs to, e.g. depart."`
	LastOperationError string `json:"last_operation_error,omitempty" doc:"Empty when the last attempt succeeded — cleared rather than left, since a stale failure sends an operator to investigate something already fixed."`
	LastOperationAt    string `json:"last_operation_at,omitempty" format:"date-time"`

	Schedulable bool   `json:"schedulable" doc:"False means cordoned: the placement query excludes it, so no new guests land here."`
	DrainState  string `json:"drain_state" enum:"active,draining,drained,unknown" doc:"Derived, never stored. 'unknown' means the observation is too stale to say whether the node is empty — it is not a synonym for 'draining'."`

	ObservedGuests int `json:"observed_guests" doc:"Guests the provider reports, STOPPED ONES INCLUDED. A stopped guest blocks a cluster join exactly as a running one does."`
	HeldPlacements int `json:"held_placements" doc:"Placements still holding capacity here, from this system's own ledger. Disagreement with observed_guests is meaningful in both directions."`

	LastSyncedAt     string `json:"last_synced_at,omitempty" format:"date-time"`
	ObservationStale bool   `json:"observation_stale"`

	Allocatable CapacityView `json:"allocatable" doc:"Ceiling after overcommit and reservation."`
	Observed    CapacityView `json:"observed" doc:"Allocated to guests, as the provider reports it."`
	Reserved    CapacityView `json:"reserved" doc:"Held back for the host itself, set by an operator."`

	// The agent, when one is enrolled. Absent for a node the capacity sync found
	// in a cluster that nobody onboarded through this system.
	BeeID     string `json:"bee_id,omitempty" format:"uuid"`
	BeeStatus string `json:"bee_status,omitempty" enum:"online,unavailable,offline" doc:"Reachability: derived from the agent's heartbeat, never stored. Absent when no agent is enrolled, which is not the same as offline."`

	// DepartureRefusals is empty when this host may leave its hive right now.
	DepartureRefusals []RefusalView `json:"departure_refusals"`
}

// FleetlessHostView is a MACHINE belonging to no hive.
//
// It used to be a bee, because a host between fleets had no Hypervisor row at
// all — that row was whatever a capacity sync had observed in a cluster, so it
// appeared on joining and vanished on leaving. The row is the durable machine
// record now, so this list is simply the machine list filtered to hive_id IS
// NULL: onboarded but not yet joined, joined nowhere yet, or departed.
//
// Still its own list rather than a filter on the hypervisor list, because the
// question an operator asks here is different — "what can I add to a cluster" —
// and the answer carries arrival refusals rather than departure ones.
type FleetlessHostView struct {
	HypervisorID string `json:"hypervisor_id" format:"uuid"`
	Address      string `json:"address"`

	// The onboarding axis, which is the whole reason an unclustered machine is
	// worth listing at all: a machine that is still 'pending' or 'failed' has no
	// working agent, and pointing an arrival at it will fail at the SSH.
	ProvisioningState string `json:"provisioning_state" enum:"pending,provisioning,provisioned,failed"`
	ProvisioningError string `json:"provisioning_error,omitempty"`

	// Status is the AGENT's reachability and is absent when no agent is enrolled
	// — which is a different thing from an agent that is offline, and the
	// distinction matters here more than anywhere: a machine record with no bee is
	// one nothing has ever logged into.
	BeeID  string `json:"bee_id,omitempty" format:"uuid"`
	Status string `json:"status,omitempty" enum:"online,unavailable,offline"`

	LastSeenAt string `json:"last_seen_at,omitempty" format:"date-time"`

	// ArrivalRefusals answers "could this join a hive right now" without naming
	// one, so the list can be rendered before a target is chosen. Refusals that
	// depend on the target — an empty destination — are reported when the arrival
	// is requested.
	ArrivalRefusals []RefusalView `json:"arrival_refusals"`
}

type ListHypervisorsRequest struct {
	HiveID string `query:"hive_id" format:"uuid" required:"false" doc:"Restrict to one hive."`
}

type ListHypervisorsResponse struct {
	Body struct {
		Hypervisors []HypervisorView `json:"hypervisors"`
	}
}

type GetHypervisorRequest struct {
	ID string `path:"id" format:"uuid"`
}

type GetHypervisorResponse struct{ Body HypervisorView }

type ListFleetlessResponse struct {
	Body struct {
		Hosts []FleetlessHostView `json:"hosts"`
	}
}

type CordonRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Schedulable bool `json:"schedulable" doc:"False cordons the node: it stops being elected for new placements. Existing guests are NOT evicted — draining is a separate act."`
	}
}

type CordonResponse struct{ Body HypervisorView }

type ReprovisionRequest struct {
	ID string `path:"id" format:"uuid"`
}

type ReprovisionResponse struct{ Body HypervisorView }

type DepartRequest struct {
	ID string `path:"id" format:"uuid"`

	// The body is REQUIRED, and carries one field with no default.
	//
	// A departing node has read the cluster's ticket-signing key and its root CA,
	// because /etc/pve/priv is shared by every member. Departure destroys its
	// copy; it cannot make it un-seen. Whether that matters is a judgement about
	// the host, so the API asks rather than assumes — and huma rejects a request
	// that omits it before this handler runs.
	Body struct {
		CredentialDisposition string `json:"credential_disposition" required:"true" enum:"rotate,accept-exposure" doc:"What to do about the cluster secrets this host has seen. \"rotate\" if the host is not trusted: it can otherwise mint auth tickets for the cluster it left. \"accept-exposure\" if it is trusted and being redeployed."`
	}
}

type DepartResponse struct {
	Body struct {
		Queued bool   `json:"queued"`
		Detail string `json:"detail"`
	}
}

// CorosyncLinkInput is one ring on the joining node.
//
// The ring NUMBER is explicit rather than positional: an operator may run link0
// and link2, and renumbering to close the gap would put traffic on a ring the
// rest of the cluster is not using.
type CorosyncLinkInput struct {
	Number   int    `json:"number" minimum:"0" maximum:"7" doc:"Ring number. Proxmox supports link0-link7."`
	Address  string `json:"address" required:"true" minLength:"1" doc:"This node's address on that ring."`
	Priority *int   `json:"priority,omitempty" minimum:"0" doc:"knet link preference; higher wins. Absent leaves corosync's default, which is not the same as zero."`
}

type ArriveRequest struct {
	HypervisorID string `path:"hypervisor_id" format:"uuid"`
	Body         struct {
		TargetHiveID string `json:"target_hive_id" required:"true" format:"uuid"`

		// Required, and deliberately not defaulted to the host's management
		// address. A cluster's corosync network routinely differs from the one
		// the control plane reaches the host over, and guessing puts cluster
		// traffic on whichever interface was convenient — which works until that
		// network is busy and then presents as unexplained fencing.
		//
		// Plural because one ring makes the cluster network a single point of
		// failure for quorum, which is the failure that fences healthy nodes.
		// Proxmox supports link0-link7.
		Links []CorosyncLinkInput `json:"links" required:"true" minItems:"1" maxItems:"8" doc:"The joining node's corosync rings. Not assumed to be its management address."`
	}
}

type ArriveResponse struct {
	Body struct {
		Queued bool   `json:"queued"`
		Detail string `json:"detail"`
	}
}

type SyncHiveRequest struct {
	ID string `path:"id" format:"uuid"`
}

type SyncHiveResponse struct {
	Body struct {
		Queued bool   `json:"queued"`
		Detail string `json:"detail"`
	}
}

type ListHivesResponse struct {
	Body struct {
		Hives []HiveDetailView `json:"hives"`
	}
}

// HiveDetailView carries what an operator configures, and what the fleet-cycle
// gate depends on.
type HiveDetailView struct {
	ID          string `json:"id" format:"uuid"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description,omitempty"`

	APIURL             string `json:"api_url,omitempty"`
	CredentialID       string `json:"credential_id,omitempty" format:"uuid"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`

	CPUOvercommitRatio float64 `json:"cpu_overcommit_ratio"`

	MinimumNodes          int `json:"minimum_nodes" doc:"Smallest node count a departure may be started FROM. Zero means the default of 3. Below three, removing a node loses quorum part-way through and strands the remaining nodes read-only."`
	EffectiveMinimumNodes int `json:"effective_minimum_nodes" doc:"What minimum_nodes resolves to, so a client does not reimplement the zero-means-default rule."`

	NodeCount int `json:"node_count"`

	// Configured reports whether a capacity sync can run for this hive at all.
	// Without it, every node reads as never-synced and every departure refuses.
	Configured bool `json:"configured"`
}

type UpdateHiveRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Description *string `json:"description,omitempty"`
		APIURL      *string `json:"api_url,omitempty" doc:"e.g. https://10.0.0.1:8006/api2/json"`
		// NO format:"uuid" HERE, deliberately. The empty string is a legal value
		// meaning "unbind whatever is currently bound", which the handler below
		// implements — and a uuid format constraint rejects it during schema
		// validation, before any handler runs. That made the unbind branch
		// unreachable and turned the ordinary case of saving a cluster that has no
		// credential yet into a 422 with nothing useful in it, because the UI sends
		// the field on every save. A genuinely malformed id still gets a 422, from
		// the uuid.Parse below, with a message that says which field was wrong.
		CredentialID       *string  `json:"credential_id,omitempty" doc:"A credential id to bind, or \"\" to unbind. Omit the field to leave the binding unchanged."`
		InsecureSkipVerify *bool    `json:"insecure_skip_verify,omitempty" doc:"Accepts a self-signed API certificate. The sealed token is handed to whatever answers on that address, so set it deliberately and per hive."`
		CPUOvercommitRatio *float64 `json:"cpu_overcommit_ratio,omitempty" minimum:"1"`
		MinimumNodes       *int     `json:"minimum_nodes,omitempty" minimum:"1" doc:"Lower this only for a test fleet. Below three, a departure strands the nodes that remain."`
	}
}

type UpdateHiveResponse struct{ Body HiveDetailView }

// RegisterFleetOps mounts the hypervisor lifecycle: cordon, depart, arrive, sync.
//
// Reads are open to any member; every mutation requires admin AND refuses an
// impersonated session — the same rule the rest of this package uses, for the
// same reason. Cordoning a host stops work landing on it and departing one takes
// it out of a cluster; an operator acting as a customer must not do either under
// that customer's name.
func RegisterFleetOps(hapi huma.API, deps *app.Deps) error {
	tenantDB := func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error) {
		if orgID == uuid.Nil {
			return nil, huma.Error403Forbidden("no organization resolved")
		}
		tc, err := deps.Tenants.GetTenantContext(ctx, orgID.String())
		if err != nil {
			return nil, huma.NewError(http.StatusServiceUnavailable,
				"this organization's storage is not ready yet")
		}
		return tc.DB, nil
	}

	memberships := repo.NewMembershipRepoGorm(deps.ControlDB)
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

	thresholds := tenant.DrainThresholds{}
	if deps.Config != nil {
		thresholds.ObservationMaxAge = deps.Config.FleetObservationMaxAge
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hives-list",
		Method:      http.MethodGet,
		Path:        "/fleet/hives",
		Summary:     "List hives",
		Description: "Every hive in the caller's organization, with the node count and whether a " +
			"capacity sync can run for it. A hive with no API URL or credential is never synced, " +
			"so every node in it reads as never-observed and every departure refuses.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *struct{}) (*ListHivesResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		var hives []tenant.Hive
		if err := db.WithContext(ctx).Order("name").Find(&hives).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		counts, err := nodeCounts(ctx, db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &ListHivesResponse{}
		out.Body.Hives = make([]HiveDetailView, 0, len(hives))
		for _, h := range hives {
			out.Body.Hives = append(out.Body.Hives, hiveDetail(h, counts[h.ID]))
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hives-update",
		Method:      http.MethodPatch,
		Path:        "/fleet/hives/{id}",
		Summary:     "Update a hive's configuration",
		Description: "Only the fields present are changed. Setting an API URL and a credential is " +
			"what makes a hive syncable.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *UpdateHiveRequest) (*UpdateHiveResponse, error) {
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

		fields := map[string]any{}
		if in.Body.Description != nil {
			fields["description"] = *in.Body.Description
		}
		if in.Body.APIURL != nil {
			fields["api_url"] = *in.Body.APIURL
		}
		if in.Body.InsecureSkipVerify != nil {
			fields["insecure_skip_verify"] = *in.Body.InsecureSkipVerify
		}
		if in.Body.CPUOvercommitRatio != nil {
			fields["cpu_overcommit_ratio"] = *in.Body.CPUOvercommitRatio
		}
		if in.Body.MinimumNodes != nil {
			fields["minimum_nodes"] = *in.Body.MinimumNodes
		}
		if in.Body.CredentialID != nil {
			if *in.Body.CredentialID == "" {
				fields["credential_id"] = nil
			} else {
				cid, err := uuid.Parse(*in.Body.CredentialID)
				if err != nil {
					return nil, huma.Error422UnprocessableEntity("invalid credential id")
				}
				// Verified to exist in THIS tenant's database before it is bound. The
				// database is the organization, so a credential found here belongs to
				// the caller by construction.
				var n int64
				if err := db.WithContext(ctx).Model(&tenant.Credential{}).
					Where("id = ?", cid).Count(&n).Error; err != nil {
					return nil, mapErr(ctx, err)
				}
				if n == 0 {
					return nil, huma.Error422UnprocessableEntity("no such credential in this organization")
				}
				fields["credential_id"] = cid
			}
		}
		if len(fields) == 0 {
			return nil, huma.Error422UnprocessableEntity("no fields to update")
		}

		res := db.WithContext(ctx).Model(&tenant.Hive{}).Where("id = ?", id).Updates(fields)
		if res.Error != nil {
			return nil, mapErr(ctx, res.Error)
		}
		if res.RowsAffected == 0 {
			return nil, huma.Error404NotFound("hive not found")
		}

		var h tenant.Hive
		if err := db.WithContext(ctx).First(&h, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		counts, err := nodeCounts(ctx, db)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &UpdateHiveResponse{Body: hiveDetail(h, counts[h.ID])}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-hypervisors-declare",
		Method:        http.MethodPost,
		Path:          "/fleet/hypervisors",
		Summary:       "Bring hosts into the fleet by address",
		DefaultStatus: http.StatusCreated,
		Description: "Records intent for one or more machines and queues them for onboarding. " +
			"The ticker claims each on its next pass, connects over SSH with the fleet key, " +
			"installs the agent and enrols it.\n\n" +
			"THE HOST MUST ALREADY TRUST THE FLEET KEY. Install its public half in the host's " +
			"authorized_keys first — Settings → Fleet SSH keys has the command — or every " +
			"attempt fails as a refused connection.\n\n" +
			"Addresses can be pasted as a list: one per line, or separated by commas or spaces, " +
			"with blank lines and #-comments ignored. Each is reported on separately, because a " +
			"rack declared all-or-nothing would roll back thirty-one good hosts over one repeat.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DeclareHostsRequest) (*DeclareHostsResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, mapErr(ctx, err)
		}

		addresses := in.Body.Addresses
		if in.Body.Hosts != "" {
			parsed, perr := tenant.ParseHostList(in.Body.Hosts)
			if perr != nil {
				return nil, huma.Error422UnprocessableEntity(perr.Error())
			}
			addresses = append(addresses, parsed...)
		}
		if len(addresses) == 0 {
			return nil, huma.Error422UnprocessableEntity(
				"give at least one address, either in `addresses` or as pasted text in `hosts`")
		}

		hosts := make([]tenant.DeclaredHost, 0, len(addresses))
		for _, a := range addresses {
			if verr := tenant.ValidateHostAddress(a); verr != nil {
				return nil, huma.Error422UnprocessableEntity(verr.Error())
			}
			hosts = append(hosts, tenant.DeclaredHost{Address: strings.TrimSpace(a), User: in.Body.User})
		}

		results, err := repo.DeclareHosts(ctx, db, hosts)
		if err != nil {
			// 422, not 500: the request is well formed and the caller is not
			// wrong. Something has to be configured first, and the message says
			// what — accepting the hosts and failing per-machine in a worker is
			// what this replaces.
			if errors.Is(err, repo.ErrNoManagementSet) {
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
			return nil, mapErr(ctx, err)
		}

		out := &DeclareHostsResponse{}
		out.Body.Hosts = make([]DeclaredHostView, 0, len(results))
		for _, r := range results {
			if r.Created {
				out.Body.Created++
			}
			out.Body.Hosts = append(out.Body.Hosts, DeclaredHostView{
				Address: r.Address, ID: r.ID, Created: r.Created, Queued: r.Queued, Detail: r.Detail,
			})
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hypervisors-list",
		Method:      http.MethodGet,
		Path:        "/fleet/hypervisors",
		Summary:     "List machines with their three status axes",
		Description: "Every MACHINE this organization owns, in a cluster or not. Each carries " +
			"three independent statuses — provisioning_state (has the agent been installed), " +
			"schedulable plus drain_state (may new work land here), and bee_status " +
			"(is it answering right now) — because a machine can be provisioned AND cordoned AND " +
			"offline at the same instant, and one badge cannot say that.\n\n" +
			"When a node cannot leave its hive, the reasons why are computed by the same gate the " +
			"departure job runs, so a client never has to reimplement it. A machine in no hive " +
			"reports not_a_member and appears in /fleet/fleetless as well.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ListHypervisorsRequest) (*ListHypervisorsResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		q := db.WithContext(ctx).Model(&tenant.Hypervisor{})
		if in.HiveID != "" {
			hid, err := uuid.Parse(in.HiveID)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity("invalid hive id")
			}
			q = q.Where("hive_id = ?", hid)
		}
		var hvs []tenant.Hypervisor
		if err := q.Order("name").Find(&hvs).Error; err != nil {
			return nil, mapErr(ctx, err)
		}

		views, err := buildHypervisorViews(ctx, db, hvs, thresholds)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ListHypervisorsResponse{}
		out.Body.Hypervisors = views
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hypervisors-get",
		Method:      http.MethodGet,
		Path:        "/fleet/hypervisors/{id}",
		Summary:     "Get one hypervisor",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *GetHypervisorRequest) (*GetHypervisorResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hypervisor id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}
		var hv tenant.Hypervisor
		if err := db.WithContext(ctx).First(&hv, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		views, err := buildHypervisorViews(ctx, db, []tenant.Hypervisor{hv}, thresholds)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &GetHypervisorResponse{Body: views[0]}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hypervisors-cordon",
		Method:      http.MethodPost,
		Path:        "/fleet/hypervisors/{id}/cordon",
		Summary:     "Cordon or uncordon a hypervisor",
		Description: "Cordoning withdraws the node from NEW placements. It does not evict anything: " +
			"draining is a separate act, and conflating them would mean the only way to stop new " +
			"work was to move existing work. A cordon is the first step of every departure.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *CordonRequest) (*CordonResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hypervisor id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		// An explicit UPDATE of the one column. Not a struct save: GORM omits a
		// zero value on a field carrying a `default:` tag, so writing Schedulable
		// through a model would silently fail to cordon — the fail-open direction
		// on the control every departure depends on.
		res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Where("id = ?", id).
			Update("schedulable", in.Body.Schedulable)
		if res.Error != nil {
			return nil, mapErr(ctx, res.Error)
		}
		if res.RowsAffected == 0 {
			return nil, huma.Error404NotFound("hypervisor not found")
		}

		var hv tenant.Hypervisor
		if err := db.WithContext(ctx).First(&hv, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		views, err := buildHypervisorViews(ctx, db, []tenant.Hypervisor{hv}, thresholds)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &CordonResponse{Body: views[0]}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hypervisors-reprovision",
		Method:      http.MethodPost,
		Path:        "/fleet/hypervisors/{id}/reprovision",
		Summary:     "Redrive a machine's onboarding",
		Description: "Sets provisioning_state back to `pending` and clears the attempt count. This " +
			"RECORDS INTENT; it does not queue a job. Onboarding is self-scheduling — a ticker claims " +
			"pending rows under a lease — so there is nothing to enqueue and nothing that has to be " +
			"running for the request to be meaningful.\n\n" +
			"Allowed from `failed` and from `provisioned` — the latter is 'revive', and it is safe " +
			"because onboarding skips enrolment for a host that is already heartbeating, so a re-run " +
			"does not invalidate a working agent's credential.\n\n" +
			"Refused with 409 while a lease is live: redriving a row a worker holds puts two workers " +
			"on one host, both installing keys and reloading sshd. An EXPIRED lease is redrivable — " +
			"that is the recovery path for a worker killed mid-onboard.\n\n" +
			"provisioning_error is deliberately NOT cleared. An operator watching a machine retry " +
			"wants to see what it is retrying past, and it is overwritten by the next real failure.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ReprovisionRequest) (*ReprovisionResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hypervisor id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		var hv tenant.Hypervisor
		if err := db.WithContext(ctx).First(&hv, "id = ?", id).Error; err != nil {
			// Explicit, because mapErr does not translate ErrRecordNotFound and
			// would answer 500 for a machine that simply is not this tenant's.
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, huma.Error404NotFound("no such machine in this organization")
			}
			return nil, mapErr(ctx, err)
		}
		// Read first purely so the refusal can SAY which rule stopped it. The
		// conditional UPDATE below is what actually enforces them; this read is
		// explanatory and may be stale by the time it is used.
		if rs := reprovisionRefusals(hv, time.Now().UTC()); len(rs) > 0 {
			return nil, refusalConflict("this machine's onboarding cannot be redriven yet", rs)
		}

		moved, err := redriveOnboarding(ctx, db, id)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		if !moved {
			// The row moved between the read above and this write — most likely the
			// ticker claimed it. Reported rather than retried: a redrive that races
			// a claim and wins is exactly the two-workers-one-host case.
			return nil, refusalConflict("this machine's onboarding cannot be redriven yet",
				[]RefusalView{{
					Code: refusalStateChanged,
					Detail: "this machine's onboarding state changed while the request was being " +
						"handled, most likely because a worker claimed it. Re-read it and try again.",
				}})
		}

		if err := db.WithContext(ctx).First(&hv, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		views, err := buildHypervisorViews(ctx, db, []tenant.Hypervisor{hv}, thresholds)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		return &ReprovisionResponse{Body: views[0]}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hypervisors-depart",
		Method:      http.MethodPost,
		Path:        "/fleet/hypervisors/{id}/depart",
		Summary:     "Remove a hypervisor from its hive",
		Description: "Queues the departure. The host is left standalone and belonging to no fleet; " +
			"it does NOT join anything. A migration is a departure and then an arrival, and the " +
			"fleetless state between them is real rather than a window to hurry through.\n\n" +
			"Refused with 409 and a list of reason codes when the host is not drained, the hive is " +
			"too small, or the capacity observation is too stale to decide. The same gate runs again " +
			"inside the job, against state read in one transaction.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *DepartRequest) (*DepartResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hypervisor id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		var hv tenant.Hypervisor
		if err := db.WithContext(ctx).First(&hv, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		refusals, err := departureRefusals(ctx, db, hv, thresholds)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		if len(refusals) > 0 {
			// Checked here as well as in the job. Not redundant: this is what turns
			// an operator's click into an immediate, explained refusal instead of a
			// job that fails somewhere they have to go and look for it. The job's
			// own check remains authoritative, because state moves between the two.
			return nil, refusalConflict("this hypervisor cannot leave its hive yet", refusalViews(refusals))
		}

		if deps.Jobs == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "the job queue is not available")
		}
		if err := deps.Jobs.EnqueueDepartHypervisor(ctx, &workflow.DepartHypervisorArgs{
			OrganizationID:        p.OrganizationID.String(),
			HypervisorID:          id.String(),
			CredentialDisposition: in.Body.CredentialDisposition,
		}); err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &DepartResponse{}
		out.Body.Queued = true
		out.Body.Detail = "departure queued; the host will be left standalone and fleetless"
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hosts-fleetless",
		Method:      http.MethodGet,
		Path:        "/fleet/fleetless",
		Summary:     "List machines belonging to no hive",
		Description: "Machines with no hive: onboarded but not yet joined to a cluster, or " +
			"departed from one. Every migration passes through this state and a failed arrival " +
			"leaves a machine sitting in it.\n\n" +
			"Each entry carries its onboarding state, because a machine that has not finished " +
			"onboarding has no agent and an arrival pointed at it will fail at the SSH.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, _ *struct{}) (*ListFleetlessResponse, error) {
		p, err := member(ctx)
		if err != nil {
			return nil, err
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		var machines []tenant.Hypervisor
		if err := db.WithContext(ctx).
			Where("hive_id IS NULL").Order("address").Find(&machines).Error; err != nil {
			return nil, mapErr(ctx, err)
		}

		beeIDs := make([]uuid.UUID, 0, len(machines))
		for _, m := range machines {
			if m.BeeID != nil {
				beeIDs = append(beeIDs, *m.BeeID)
			}
		}
		bees := make(map[uuid.UUID]tenant.Bee, len(beeIDs))
		if len(beeIDs) > 0 {
			var bs []tenant.Bee
			if err := db.WithContext(ctx).Where("id IN ?", beeIDs).Find(&bs).Error; err != nil {
				return nil, mapErr(ctx, err)
			}
			for _, b := range bs {
				bees[b.ID] = b
			}
		}

		now := time.Now().UTC()
		out := &ListFleetlessResponse{}
		out.Body.Hosts = make([]FleetlessHostView, 0, len(machines))
		for _, m := range machines {
			v := FleetlessHostView{
				HypervisorID:      m.ID.String(),
				Address:           m.Address,
				ProvisioningState: m.ProvisioningState,
				ProvisioningError: m.ProvisioningError,
			}
			if m.BeeID != nil {
				v.BeeID = m.BeeID.String()
				if b, ok := bees[*m.BeeID]; ok {
					v.Status = b.LivenessAt(now, 2*time.Minute)
					if b.LastSeenAt != nil {
						v.LastSeenAt = b.LastSeenAt.UTC().Format(time.RFC3339)
					}
				}
			}
			// Target-independent refusals only. Anything about the destination —
			// an empty hive — is reported when an arrival is actually requested.
			//
			// ObservedGuests is zero rather than m.ObservedGuests on purpose: the
			// sync zeroes the observations when it clears a machine's membership,
			// so any non-zero value here would be describing a cluster the machine
			// has left. The arrival path re-asks the host itself before joining.
			v.ArrivalRefusals = arrivalRefusalViews(tenant.CheckArrival(tenant.ArrivalRequest{
				TargetHive:      tenant.Hive{Name: "(unspecified)"},
				CurrentHiveID:   m.HiveID,
				ObservedGuests:  0,
				TargetNodeCount: 1,
			}))
			out.Body.Hosts = append(out.Body.Hosts, v)
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hosts-arrive",
		Method:      http.MethodPost,
		Path:        "/fleet/fleetless/{hypervisor_id}/arrive",
		Summary:     "Add a standalone machine to a hive",
		Description: "Queues the arrival. Refused with 409 and reason codes when the host already " +
			"belongs to a hive, still carries guests, or the target has no nodes to join to.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *ArriveRequest) (*ArriveResponse, error) {
		p, err := admin(ctx)
		if err != nil {
			return nil, err
		}
		hypID, err := uuid.Parse(in.HypervisorID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid hypervisor id")
		}
		hiveID, err := uuid.Parse(in.Body.TargetHiveID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid target hive id")
		}
		db, err := tenantDB(ctx, p.OrganizationID)
		if err != nil {
			return nil, err
		}

		var hv tenant.Hypervisor
		if err := db.WithContext(ctx).First(&hv, "id = ?", hypID).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		var hive tenant.Hive
		if err := db.WithContext(ctx).First(&hive, "id = ?", hiveID).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		var nodes int64
		if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
			Where("hive_id = ?", hiveID).Count(&nodes).Error; err != nil {
			return nil, mapErr(ctx, err)
		}

		links := make([]tenant.CorosyncLink, 0, len(in.Body.Links))
		for _, l := range in.Body.Links {
			links = append(links, tenant.CorosyncLink{
				Number: l.Number, Address: l.Address, Priority: l.Priority,
			})
		}
		// 422 rather than a refusal code: a malformed ring set is a bad request,
		// not a fleet state that will resolve on its own. Checked here so the
		// operator is told immediately, and again in the job because state moves.
		if err := tenant.ValidateCorosyncLinks(links); err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		// ObservedGuests off the machine record itself, which is the row the gate
		// is about. It used to be looked up through hypervisors.bee_id — a link no
		// production path ever wrote — so it always missed and the guests_present
		// refusal never fired outside the lab. The job re-asks the host directly
		// before joining; this value only decides whether to bother connecting.
		refusals := tenant.CheckArrival(tenant.ArrivalRequest{
			TargetHive:      hive,
			CurrentHiveID:   hv.HiveID,
			ObservedGuests:  hv.ObservedGuests,
			TargetNodeCount: int(nodes),
		})
		if len(refusals) > 0 {
			return nil, refusalConflict("this host cannot join that hive yet", arrivalRefusalViews(refusals))
		}

		if deps.Jobs == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "the job queue is not available")
		}
		if err := deps.Jobs.EnqueueArriveHypervisor(ctx, &workflow.ArriveHypervisorArgs{
			OrganizationID: p.OrganizationID.String(),
			HypervisorID:   hypID.String(),
			TargetHiveID:   hiveID.String(),
			Links:          links,
		}); err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &ArriveResponse{}
		out.Body.Queued = true
		out.Body.Detail = fmt.Sprintf("arrival into %q queued", hive.Name)
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-hives-sync",
		Method:      http.MethodPost,
		Path:        "/fleet/hives/{id}/sync",
		Summary:     "Refresh a hive's capacity observations now",
		Description: "The same job the schedule runs, enqueued immediately. Most often wanted after " +
			"a departure refused with drain_unknown, which means the last observation is too stale " +
			"to say whether a node is empty.\n\n" +
			"The job is unique per hive while queued or running, so asking twice gets the sync " +
			"already on its way rather than a second one.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityUserOrKey,
	}, func(ctx context.Context, in *SyncHiveRequest) (*SyncHiveResponse, error) {
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
		var hive tenant.Hive
		if err := db.WithContext(ctx).First(&hive, "id = ?", id).Error; err != nil {
			return nil, mapErr(ctx, err)
		}
		if hive.APIURL == "" || hive.CredentialID == nil {
			// 422 rather than queuing a job that cannot succeed. The operator's next
			// action is to configure the hive, and a job failing in the background
			// does not tell them that.
			return nil, huma.Error422UnprocessableEntity(
				"this hive has no API URL or no credential bound, so its capacity cannot be read")
		}

		if deps.Jobs == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "the job queue is not available")
		}
		if err := deps.Jobs.EnqueueSyncHiveCapacity(ctx, &workflow.SyncHiveCapacityArgs{
			OrganizationID: p.OrganizationID.String(),
			HiveID:         id.String(),
		}); err != nil {
			return nil, mapErr(ctx, err)
		}

		out := &SyncHiveResponse{}
		out.Body.Queued = true
		out.Body.Detail = fmt.Sprintf("capacity sync for %q queued", hive.Name)
		return out, nil
	})

	return nil
}

// Refusal codes for the onboarding redrive and for removing an agent.
//
// Stable strings, because a client keys on the code to decide what to offer
// next: "wait" for a held lease, "nothing" for a machine already queued, "go and
// set an address" for the last. Prose is not a contract, which is the same
// reason tenant.DepartureRefusal carries one.
const (
	refusalProvisioningLeaseHeld = "provisioning_lease_held"
	refusalAlreadyPending        = "provisioning_already_pending"
	refusalNoAddress             = "no_address"
	refusalStateChanged          = "provisioning_state_changed"
)

// redrivablePredicate is the rule deciding whether an onboarding may be pushed
// back to `pending`, written as SQL so it can be evaluated INSIDE the update
// that acts on it.
//
// That is the whole reason it is a predicate rather than a Go check over a row
// already loaded: the ticker claims rows concurrently, so a verdict reached at
// one instant and written at another can put a row back to pending while a
// worker holds it — which is two workers on one host, both installing keys and
// reloading sshd. Same argument as tenant.SchedulablePredicate, whose comment
// makes it for the placement query.
//
// THE LEASE IS THE GUARD, NOT THE STATE. `provisioning` with an expired lease is
// redrivable, and that is the recovery path for a worker killed mid-onboard. A
// NULL lease is treated as expired: the claim stamps state and lease in one
// statement so the combination cannot arise from correct code, and reading it as
// "held" would strand the row permanently with SQL as the only way out — the
// exact predicament redrive exists to end.
//
// Postgres now(), not a Go timestamp: the lease is written by a worker and read
// by an API replica, and the database is the one clock both agree on.
//
// ONE DEFINITION, in internal/models/tenant, shared with the ticker's claim
// query. The two rules are not the same set — see RedrivablePredicate — but they
// read the same lease and the same clock, and a second copy of either is how
// they stop agreeing.
const redrivablePredicate = tenant.RedrivablePredicate

// redriveOnboarding pushes one machine's onboarding back to `pending`, and
// reports whether it moved.
//
// ONE STATEMENT. The rule and the write are the same statement so that a row
// claimed between deciding and writing is not overwritten — see
// redrivablePredicate. `false` therefore means "the rule said no at the instant
// of the write", which is the only instant that matters.
func redriveOnboarding(ctx context.Context, db *gorm.DB, id uuid.UUID) (bool, error) {
	res := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", id).Where(redrivablePredicate).
		Updates(map[string]any{
			"provisioning_state": tenant.OnboardPending,
			// Reset, or the redrive is a no-op for the one case it exists for: the
			// ticker stops at an attempt cap and leaves the row failed, so a redrive
			// that kept the count would be re-capped on the first tick.
			"provisioning_attempt": 0,
			// The lease is the exclusivity token. A pending row nobody holds must not
			// carry one, or the next claim's expiry arithmetic is deciding against a
			// lease belonging to a worker that is long gone.
			"provisioning_lease_expires_at": nil,
		})
	// provisioning_error and provisioning_started_at are deliberately left alone.
	// They describe the attempt being retried past, which is what an operator
	// watching a machine retry is looking for, and the next real failure
	// overwrites them.
	return res.RowsAffected == 1, res.Error
}

// provisioningLeaseHeld reports whether a worker is onboarding this machine
// right now.
//
// The Go half of redrivablePredicate's lease rule, and shared with the agent
// delete in platform.go so there is one answer to "is a worker inside this
// machine". Advisory only — every write that depends on it re-checks in SQL.
func provisioningLeaseHeld(hv tenant.Hypervisor, now time.Time) bool {
	if hv.ProvisioningState != tenant.OnboardProvisioning {
		return false
	}
	// A lease that was never stamped holds nothing. See redrivablePredicate.
	if hv.ProvisioningLeaseExpiresAt == nil {
		return false
	}
	return hv.ProvisioningLeaseExpiresAt.After(now)
}

// reprovisionRefusals explains why a redrive is not allowed, or returns nothing.
//
// Explanatory, not enforcing: redrivablePredicate is what enforces, in the same
// statement as the write. This exists so an operator gets a reason instead of a
// bare conflict, and it is pure so the rule can be tested without a database.
func reprovisionRefusals(hv tenant.Hypervisor, now time.Time) []RefusalView {
	var out []RefusalView

	if hv.Address == "" {
		// The ticker reaches a machine over SSH at its address. Without one there
		// is nothing to onboard, and a pending row with no address is a machine
		// that fails on every tick forever.
		out = append(out, RefusalView{
			Code: refusalNoAddress,
			Detail: "this machine has no address, so onboarding has nothing to connect to. " +
				"Record how the control plane reaches it first.",
		})
	}

	switch {
	case hv.ProvisioningState == tenant.OnboardPending:
		// Not silently idempotent. Resetting the attempt count of a row already
		// queued looks identical to a successful redrive while changing nothing an
		// operator asked for, so it says what is true instead.
		out = append(out, RefusalView{
			Code:   refusalAlreadyPending,
			Detail: "this machine is already queued for onboarding and will be claimed on the next tick.",
		})
	case provisioningLeaseHeld(hv, now):
		out = append(out, RefusalView{
			Code: refusalProvisioningLeaseHeld,
			Detail: "a worker is onboarding this machine right now. Redriving it would put a second " +
				"worker on the same host, both installing keys and reloading sshd. The lease expires " +
				"on its own if that worker dies.",
		})
	}
	return out
}

// refusalConflict renders a gate refusal as a 409 carrying the codes.
//
// 409 rather than 422: the request is well-formed and would be valid at another
// moment. And never 500 — a refusal is the gate working, and reporting an
// operator's own drain policy as a server error puts it on their 5xx dashboard.
func refusalConflict(message string, rs []RefusalView) error {
	details := make([]error, 0, len(rs))
	for _, r := range rs {
		details = append(details, &huma.ErrorDetail{
			Message:  r.Detail,
			Location: r.Code,
		})
	}
	return huma.Error409Conflict(message, details...)
}

// nodeCounts returns how many hypervisors each hive has.
//
// The NOT NULL predicate is load-bearing now that a machine record outlives its
// cluster membership. Without it every unclustered machine folds into one NULL
// group, which scans into a uuid.Nil key and then answers `counts[hv.HiveID]`
// for any caller that also passed a nil hive — reporting the number of MACHINES
// IN NO CLUSTER as the node count of a hive, straight into the departure gate's
// minimum-nodes arithmetic.
func nodeCounts(ctx context.Context, db *gorm.DB) (map[uuid.UUID]int, error) {
	type row struct {
		HiveID uuid.UUID
		N      int
	}
	var rows []row
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Select("hive_id, COUNT(*) AS n").Where("hive_id IS NOT NULL").
		Group("hive_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]int, len(rows))
	for _, r := range rows {
		out[r.HiveID] = r.N
	}
	return out, nil
}

func hiveDetail(h tenant.Hive, nodes int) HiveDetailView {
	v := HiveDetailView{
		ID:                    h.ID.String(),
		Name:                  h.Name,
		Provider:              h.Provider,
		Description:           h.Description,
		APIURL:                h.APIURL,
		InsecureSkipVerify:    h.InsecureSkipVerify,
		CPUOvercommitRatio:    h.EffectiveCPUOvercommitRatio(),
		MinimumNodes:          h.MinimumNodes,
		EffectiveMinimumNodes: h.EffectiveMinimumNodes(),
		NodeCount:             nodes,
		Configured:            h.APIURL != "" && h.CredentialID != nil,
	}
	if h.CredentialID != nil {
		v.CredentialID = h.CredentialID.String()
	}
	return v
}

// departureRefusals runs the gate for one hypervisor.
//
// Reads the placement count and the hive node count in ONE transaction with the
// hypervisor, for the same reason the job's snapshot does: a verdict assembled
// from separate reads describes no state that ever existed.
func departureRefusals(
	ctx context.Context, db *gorm.DB, hv tenant.Hypervisor, th tenant.DrainThresholds,
) ([]tenant.DepartureRefusal, error) {
	var (
		hive   tenant.Hive
		held   int64
		nodes  int64
		result []tenant.DepartureRefusal
	)
	// A machine in no cluster has nothing to depart from, and the gate says so
	// itself. Answered before the transaction because every read inside it is
	// scoped by hive_id: against NULL the hive lookup fails with record-not-found
	// and the whole LIST endpoint 500s, on exactly the machines an operator opened
	// it to find.
	if hv.HiveID == nil {
		return tenant.CheckDepartureState(tenant.DepartureRequest{Hypervisor: hv}), nil
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&hive, "id = ?", *hv.HiveID).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenant.Placement{}).
			Where("hypervisor_id = ? AND state IN ?", hv.ID,
				[]string{tenant.PlacementPending, tenant.PlacementActive}).
			Count(&held).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenant.Hypervisor{}).
			Where("hive_id = ?", *hv.HiveID).Count(&nodes).Error; err != nil {
			return err
		}
		drain := tenant.DrainStateAt(time.Now().UTC(), hv, int(held), th)
		result = tenant.CheckDepartureState(tenant.DepartureRequest{
			Hive: hive, Hypervisor: hv, Drain: drain, HiveNodeCount: int(nodes),
		})
		return nil
	})
	return result, err
}

// buildHypervisorViews assembles the operator-facing view for a set of nodes.
func buildHypervisorViews(
	ctx context.Context, db *gorm.DB, hvs []tenant.Hypervisor, th tenant.DrainThresholds,
) ([]HypervisorView, error) {
	if len(hvs) == 0 {
		return []HypervisorView{}, nil
	}

	hiveIDs := make([]uuid.UUID, 0, len(hvs))
	hvIDs := make([]uuid.UUID, 0, len(hvs))
	beeIDs := make([]uuid.UUID, 0, len(hvs))
	for _, h := range hvs {
		// Skipped, not appended as a zero: a nil hive id is a machine in no
		// cluster, and `id IN (00000000-...)` is a predicate that matches nothing
		// while looking like it matched something.
		if h.HiveID != nil {
			hiveIDs = append(hiveIDs, *h.HiveID)
		}
		hvIDs = append(hvIDs, h.ID)
		if h.BeeID != nil {
			beeIDs = append(beeIDs, *h.BeeID)
		}
	}

	byHive := make(map[uuid.UUID]tenant.Hive, len(hiveIDs))
	if len(hiveIDs) > 0 {
		var hives []tenant.Hive
		if err := db.WithContext(ctx).Where("id IN ?", hiveIDs).Find(&hives).Error; err != nil {
			return nil, err
		}
		for _, h := range hives {
			byHive[h.ID] = h
		}
	}

	counts, err := nodeCounts(ctx, db)
	if err != nil {
		return nil, err
	}

	// Held placements per node, in one query rather than one per node — this list
	// is rendered on a page an operator refreshes.
	type heldRow struct {
		HypervisorID uuid.UUID
		N            int
	}
	var heldRows []heldRow
	if err := db.WithContext(ctx).Model(&tenant.Placement{}).
		Select("hypervisor_id, COUNT(*) AS n").
		Where("hypervisor_id IN ? AND state IN ?", hvIDs,
			[]string{tenant.PlacementPending, tenant.PlacementActive}).
		Group("hypervisor_id").Scan(&heldRows).Error; err != nil {
		return nil, err
	}
	held := make(map[uuid.UUID]int, len(heldRows))
	for _, r := range heldRows {
		held[r.HypervisorID] = r.N
	}

	bees := make(map[uuid.UUID]tenant.Bee)
	if len(beeIDs) > 0 {
		var bs []tenant.Bee
		if err := db.WithContext(ctx).Where("id IN ?", beeIDs).Find(&bs).Error; err != nil {
			return nil, err
		}
		for _, b := range bs {
			bees[b.ID] = b
		}
	}

	now := time.Now().UTC()
	out := make([]HypervisorView, 0, len(hvs))
	for _, hv := range hvs {
		// A machine in no cluster gets the ZERO hive, and the arithmetic below
		// then uses DefaultCPUOvercommitRatio for its allocatable ceiling. That is
		// acceptable rather than accidental: the sync zeroes the observations when
		// it clears membership, so the ceiling is computed from zero totals and
		// comes out zero whatever ratio is applied.
		var hive tenant.Hive
		if hv.HiveID != nil {
			hive = byHive[*hv.HiveID]
		}
		drain := tenant.DrainStateAt(now, hv, held[hv.ID], th)

		v := HypervisorView{
			ID:                  hv.ID.String(),
			HiveID:              hiveIDOfMachine(hv),
			HiveName:            hive.Name,
			Name:                nodeNameOf(hv),
			Address:             hv.Address,
			ProvisioningState:   hv.ProvisioningState,
			ProvisioningAttempt: hv.ProvisioningAttempt,
			ProvisioningError:   hv.ProvisioningError,
			LastOperation:       hv.LastOperation,
			LastOperationError:  hv.LastOperationError,
			Schedulable:         hv.Schedulable,
			DrainState:          drain.State,
			ObservedGuests:      drain.ObservedGuests,
			HeldPlacements:      drain.HeldPlacements,
			ObservationStale:    drain.Stale,
			Allocatable:         capacityView(hv.Allocatable(hive)),
			Observed: CapacityView{
				CPU: hv.ObservedCPUUsed, RAMGB: hv.ObservedRAMGBUsed, DiskGB: hv.ObservedDiskGBUsed,
			},
			Reserved: CapacityView{
				CPU: hv.CPUReserved, RAMGB: hv.RAMGBReserved, DiskGB: hv.DiskGBReserved,
			},
			DepartureRefusals: refusalViews(tenant.CheckDepartureState(tenant.DepartureRequest{
				Hive: hive, Hypervisor: hv, Drain: drain, HiveNodeCount: nodeCountFor(counts, hv),
			})),
		}
		if hv.LastSyncedAt != nil {
			v.LastSyncedAt = hv.LastSyncedAt.UTC().Format(time.RFC3339)
		}
		if hv.LastOperationAt != nil {
			v.LastOperationAt = hv.LastOperationAt.UTC().Format(time.RFC3339)
		}
		if hv.BeeID != nil {
			v.BeeID = hv.BeeID.String()
			if b, ok := bees[*hv.BeeID]; ok {
				v.BeeStatus = b.LivenessAt(now, 2*time.Minute)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func capacityView(c tenant.Capacity) CapacityView {
	return CapacityView{CPU: c.CPU, RAMGB: c.RAMGB, DiskGB: c.DiskGB}
}

// hiveIDOfMachine renders a machine's cluster, or "" when it is in none.
//
// HiveID is a POINTER now that the machine record outlives its cluster
// membership. Calling String() on it panics — uuid.UUID.String has a value
// receiver, so the nil pointer is dereferenced — and it would panic on exactly
// the machines an operator opens this listing to find: the ones being onboarded
// and the ones mid-migration. The same trap as hiveIDOf on the bee listing, which
// is the one that was actually hit.
func hiveIDOfMachine(h tenant.Hypervisor) string {
	if h.HiveID == nil {
		return ""
	}
	return h.HiveID.String()
}

// nodeNameOf renders the cluster's name for a machine, or "" when it has none.
func nodeNameOf(h tenant.Hypervisor) string {
	if h.Name == nil {
		return ""
	}
	return *h.Name
}

// nodeCountFor answers "how many nodes does this machine's hive have", and ZERO
// for a machine in no hive.
//
// The zero is what CheckDeparture reads as not_a_member, so this is not merely
// avoiding a nil dereference — indexing the map with a zero uuid would hand the
// gate a count belonging to no hive at all.
func nodeCountFor(counts map[uuid.UUID]int, h tenant.Hypervisor) int {
	if h.HiveID == nil {
		return 0
	}
	return counts[*h.HiveID]
}

// DeclareHostsRequest names machines to bring into the fleet.
//
// TWO WAYS TO SAY IT, on purpose. `addresses` is what a client sends; `hosts` is
// what a person pastes — out of a spreadsheet, an inventory file or a terminal —
// and requiring that to be reformatted into a JSON array by hand is exactly
// where a typo enters a list of machines about to be connected to.
type DeclareHostsRequest struct {
	Body struct {
		Addresses []string `json:"addresses,omitempty" maxItems:"512" doc:"Hostnames or IPs. Not URLs and not host:port — the control plane connects over SSH."`
		Hosts     string   `json:"hosts,omitempty" maxLength:"32768" doc:"The same thing as pasted text: one per line, or separated by commas or spaces. Blank lines and #-comments are ignored."`
		User      string   `json:"user,omitempty" maxLength:"64" doc:"SSH user. Empty means root, which is what a Proxmox node has."`
	}
}

type DeclaredHostView struct {
	Address string `json:"address"`
	ID      string `json:"id,omitempty"`
	Created bool   `json:"created" doc:"False when this address was already known. Not a failure: re-pasting a list is ordinary, and the existing machine is left exactly as it was rather than pushed back to pending."`
	Queued  bool   `json:"queued" doc:"Whether this machine is now waiting for the onboarding ticker."`
	Detail  string `json:"detail,omitempty"`
}

type DeclareHostsResponse struct {
	Body struct {
		Created int                `json:"created"`
		Hosts   []DeclaredHostView `json:"hosts"`
	}
}
