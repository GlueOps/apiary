// Package fleet carries the Bees surface: the endpoints a fleet agent on a
// Proxmox host talks to, and the platform-staff endpoints that drive them.
//
// Three route groups live here and must never be merged:
//
//   - RegisterPublic       enrolment only, authenticated by an enrolment token
//   - RegisterBeeProtected the agent loop, authenticated by a bee credential
//   - RegisterPlatform     staff endpoints, authenticated by a user session
//
// They use three different identities. Merging any two means one of them is
// reachable with the wrong credential.
package fleet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/agentdist"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/buildinfo"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

// newService assembles the fleet service from its two sides: the control
// database answers which tenant a caller is, and every fleet row lives in that
// tenant's database — which is why no FleetRepo appears here. One cannot be
// built until a request has been authenticated.
func newService(deps *app.Deps) (*service.FleetService, error) {
	return service.NewFleetService(
		deps.Tenants,
		repo.NewEnrolmentRepoGorm(deps.ControlDB),
		deps.APIKeys,
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
	)
}

type TaskView struct {
	ID      string `json:"id" format:"uuid"`
	Kind    string `json:"kind" example:"fleet.host.drain"`
	Spec    any    `json:"spec" doc:"Desired state. The bee decides how to reach it, so re-delivering the same task after a crash is safe."`
	Status  string `json:"status"`
	Attempt int    `json:"attempt"`
}

func toTaskView(t *tenant.FleetTask) *TaskView {
	if t == nil {
		return nil
	}
	var spec any
	if len(t.Spec) > 0 {
		_ = jsonUnmarshal(t.Spec, &spec)
	}
	return &TaskView{ID: t.ID.String(), Kind: t.Kind, Spec: spec, Status: t.Status, Attempt: t.Attempt}
}

type EnrolRequest struct {
	Body struct {
		EnrolmentToken string `json:"enrolment_token" required:"true" minLength:"16" maxLength:"128"`
		MachineID      string `json:"machine_id" required:"true" minLength:"1" maxLength:"200" doc:"The host's own stable identifier, typically /etc/machine-id. Hashed before storage; re-enrolling with the same value repoints this host's existing machine record rather than creating a second one."`
		Hostname       string `json:"hostname" required:"true" minLength:"1" maxLength:"253" doc:"How the control plane should reach this machine. Recorded against the MACHINE, not the agent: reinstalling an agent does not move a box."`
		AgentVersion   string `json:"agent_version,omitempty" maxLength:"64"`
	}
}

type EnrolResponse struct {
	Body struct {
		BeeID string `json:"bee_id" format:"uuid"`
		OrgID string `json:"organization_id" format:"uuid" doc:"The organization this host was admitted into. It joins a fleet afterwards, as a task — the same path a host migrating between fleets takes."`

		KeyPublic string `json:"key" doc:"The public half of this host's agent credential. Presented as the HTTP Basic username."`
		KeySecret string `json:"secret" doc:"Returned ONCE. Presented as the HTTP Basic password. There is no endpoint that reissues it, because such an endpoint would let anyone holding a bee id take the bee over. A host that loses it enrols again."`
	}
}

type HeartbeatRequest struct {
	Body struct {
		AgentVersion string `json:"agent_version,omitempty" maxLength:"64"`
		// AgentDigest is the sha256 of the binary this agent is running. The
		// upgrade decision is made on this rather than on the version: a version is
		// a label somebody stamped, and only a release stamps the control plane and
		// the binaries it serves from one tag. Where they disagree a
		// version comparison never converges — the agent upgrades, restarts,
		// reports the same version and is judged out of date again.
		AgentDigest string `json:"agent_digest,omitempty" maxLength:"71"`

		UnavailableForSeconds int    `json:"unavailable_for_seconds,omitempty" minimum:"0" maximum:"86400" doc:"Declared BEFORE acting: how long this host expects to be unreachable, e.g. across a reboot. Until it elapses, silence from this bee is expected rather than a fault."`
		UnavailableReason     string `json:"unavailable_reason,omitempty" maxLength:"200"`
		Back                  bool   `json:"back,omitempty" doc:"Withdraws a standing declaration: the host has returned."`

		WaitSeconds int `json:"wait_seconds,omitempty" minimum:"0" maximum:"25" doc:"Hold the request open for up to this long waiting for a task. The response is the same shape either way."`

		// What this host is CURRENTLY enforcing, so convergence is reported rather
		// than assumed. A publish only says what should be running; marking a fleet
		// converged when an operator clicks Publish would report the policy as
		// enforced on every host that happened to be switched off.
		AppliedFirewallRevision int    `json:"applied_firewall_revision,omitempty" minimum:"0" doc:"The firewall revision this host has actually loaded. Omit or zero if it has applied none."`
		AppliedFirewallDigest   string `json:"applied_firewall_digest,omitempty" maxLength:"71" doc:"Digest of the body actually loaded, so a host that applied the right revision from a stale body is still detected as diverged."`

		// ObservedGuests closes a gap the capacity sync cannot reach.
		//
		// That sync runs per CLUSTER, so a machine in no cluster is never
		// observed at all — and fleetless is precisely the state a machine is in
		// when it is about to join one. observed_guests therefore read zero for
		// every candidate, and the arrival gate's guests_present refusal, which
		// exists because `pvecm add` discards the joining node's guest
		// configuration, could not fire for the only case it covers. Two lab
		// hosts sat at zero holding two containers each.
		//
		// Pointer so that "did not say" and "said none" stay distinct. An older
		// agent omits the field, and reading that as zero would be inventing an
		// observation nobody made — the same mistake, one level down.
		ObservedGuests *int `json:"observed_guests,omitempty" minimum:"0" maximum:"100000" doc:"VMs and containers registered to this host, stopped ones included. Counted from /etc/pve rather than the Proxmox API, so it survives a broken pveproxy."`
	}
}

type HeartbeatResponse struct {
	Body struct {
		Task *TaskView `json:"task,omitempty" doc:"This bee's one outstanding task, or absent. A bee that already holds a task is handed the SAME one back."`

		// Firewall is ABSENT when there is nothing to enforce — the machine is in
		// no fleet, or its fleet has published nothing. Absent means HOLD, never
		// flush: a host between fleets still needs its management access, and it is
		// least reachable at exactly that moment.
		//
		// Only the revision and digest travel here. The body can be tens of
		// kilobytes and would ride on every heartbeat from every host in the fleet
		// to say "unchanged"; the agent fetches it from /agent/firewall only when
		// these differ from what it applied.
		Firewall *FirewallDesiredView `json:"firewall,omitempty"`
	}
}

// FirewallDesiredView is the small half of the convergence protocol.
type FirewallDesiredView struct {
	Revision int    `json:"revision" doc:"Only ever increases. A backout is a NEW revision carrying an older body, so an agent comparing revisions can use != or < and behave identically."`
	Digest   string `json:"digest" doc:"Of the rendered body. Compared as well as the revision, so a host that loaded the right revision from a corrupted or hand-edited file is still seen as diverged."`
}

type AgentFirewallRequest struct{}

type AgentFirewallResponse struct {
	Body struct {
		// Absent when there is nothing to enforce. Same meaning as on the
		// heartbeat: hold what you have.
		Revision int    `json:"revision,omitempty"`
		Digest   string `json:"digest,omitempty"`
		Content  string `json:"content,omitempty" doc:"The complete nft ruleset, to be loaded with nft -f. It begins with a flush because it is desired state rather than a patch: anything already loaded that it does not declare must go."`
	}
}

type ReportRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Status string `json:"status" required:"true" enum:"running,succeeded,failed,cancelled"`
		Phase  string `json:"phase,omitempty" maxLength:"64"`
		Error  string `json:"error,omitempty" maxLength:"2000"`
	}
}

type ReportResponse struct {
	Body struct {
		Status string `json:"status"`
	}
}

// TaskLogLine is a NAMED type on purpose. An inline anonymous struct here gave
// the generators nothing to call it, and the Go SDK ended up exposing
// `[]Item1` — a type name that tells a consumer nothing about what it holds.
// The schema name is part of the API's surface, not an implementation detail.
type TaskLogLine struct {
	Seq      int64  `json:"seq" required:"true" minimum:"0" doc:"Monotonic per task, assigned by the bee. Resending a batch is a no-op, which is what makes the stream correct when a bee cannot confirm a write."`
	Stream   string `json:"stream,omitempty" enum:"stdout,stderr,agent"`
	Message  string `json:"message" required:"true" maxLength:"16384"`
	LoggedAt string `json:"logged_at,omitempty" format:"date-time"`
}

type LogsRequest struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Lines []TaskLogLine `json:"lines" required:"true" maxItems:"500"`
	}
}

type LogsResponse struct {
	Body struct {
		Accepted int `json:"accepted" doc:"How many lines were NEW. A resent batch reports zero, which is how a bee can tell progress from retry."`
	}
}

func mapErr(ctx context.Context, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repo.ErrEnrolmentTokenInvalid), errors.Is(err, service.ErrEnrolmentRefused):
		// One answer for expired, revoked, used-up and unknown. This endpoint is
		// reachable by anything that can route to the control plane, so it must
		// not report which of those a guess hit.
		return huma.Error401Unauthorized("enrolment refused")
	case errors.Is(err, repo.ErrBeeNotFound):
		return huma.Error401Unauthorized("unknown bee")
	case errors.Is(err, repo.ErrTaskNotFound):
		return huma.Error404NotFound("no such task for this bee")
	case errors.Is(err, repo.ErrBeeBusy):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, service.ErrLogBatchTooLarge):
		return huma.Error413RequestEntityTooLarge(err.Error())
	default:
		slog.ErrorContext(ctx, "fleet operation failed", "error", err)
		return huma.Error500InternalServerError("fleet operation failed")
	}
}

// RegisterPublic mounts enrolment. It is unauthenticated in the user sense: a
// host being brought up has no credential yet, which is the entire point of an
// enrolment token.
func RegisterPublic(hapi huma.API, deps *app.Deps) error {
	svc, err := newService(deps)
	if err != nil {
		return err
	}

	huma.Register(hapi, huma.Operation{
		OperationID:   "fleet-bee-enrol",
		Method:        http.MethodPost,
		Path:          "/agent/enrol",
		Summary:       "Enrol a fleet agent into an organization",
		DefaultStatus: http.StatusCreated,
		Description: "Redeems an enrolment token and returns this host's agent-scoped key and secret. " +
			"The host is admitted into an ORGANIZATION, not a fleet — joining one is a task, so a new host and a host migrating between fleets follow the same path. " +
			"Idempotent on machine_id: a rebuilt hypervisor re-enrols onto the same MACHINE record with a fresh credential rather than leaving a permanently-offline ghost. " +
			"A machine a capacity sync has already discovered in a cluster is adopted rather than duplicated, matched by the node name its hostname implies.",
		Tags: []string{"Fleet"},
	}, func(ctx context.Context, in *EnrolRequest) (*EnrolResponse, error) {
		res, err := svc.Enrol(ctx, service.EnrolInput{
			EnrolmentToken: in.Body.EnrolmentToken,
			MachineID:      in.Body.MachineID,
			Hostname:       in.Body.Hostname,
			AgentVersion:   in.Body.AgentVersion,
			PeerAddress:    apimiddleware.ClientIPFromContext(ctx),
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &EnrolResponse{}
		out.Body.BeeID = res.Bee.ID.String()
		out.Body.OrgID = res.OrganizationID.String()
		out.Body.KeyPublic = res.KeyPublic
		out.Body.KeySecret = res.KeySecret
		return out, nil
	})
	return nil
}

// RegisterBeeProtected mounts the agent loop. It must be handed a group carrying
// RequireBeeAuth — see the package comment.
func RegisterBeeProtected(hapi huma.API, deps *app.Deps) error {
	svc, err := newService(deps)
	if err != nil {
		return err
	}

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-heartbeat",
		Method:      http.MethodPost,
		Path:        "/agent/heartbeat",
		Summary:     "Report liveness and collect work",
		Description: "One request carries both, deliberately: two endpoints would give a bee two ways to be 'seen', and a bee parked on a long poll would have to heartbeat separately to avoid looking dead. " +
			"Declare unavailable_for_seconds BEFORE doing anything that interrupts the host — a reboot is indistinguishable from a failure unless the host says in advance that it is coming back.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityAgent,
	}, func(ctx context.Context, in *HeartbeatRequest) (*HeartbeatResponse, error) {
		sess, ok := apimiddleware.BeeFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}

		// Recorded BEFORE the heartbeat's own wait. Heartbeat can hold the request
		// open for up to 25 seconds waiting for a task, and a convergence report
		// held behind that would arrive a poll late every time on a fleet that is
		// otherwise idle — which is most of the time.
		if err := svc.RecordAppliedFirewall(ctx, sess,
			in.Body.AppliedFirewallRevision, in.Body.AppliedFirewallDigest); err != nil {
			// Logged, never returned. A bee whose convergence report failed to
			// record is still alive, and refusing its heartbeat over bookkeeping
			// would make it look offline — a page for a host that is fine.
			slog.WarnContext(ctx, "could not record a bee's applied firewall revision",
				"bee_id", sess.Bee.ID, "error", err)
		}

		// Same treatment, and for the same reason: a host whose observation
		// failed to record is still alive, and refusing its heartbeat over
		// bookkeeping would make it look offline.
		if err := svc.RecordObservedGuests(ctx, sess, in.Body.ObservedGuests); err != nil {
			slog.WarnContext(ctx, "could not record a bee's observed guest count",
				"bee_id", sess.Bee.ID, "error", err)
		}

		res, err := svc.Heartbeat(ctx, service.HeartbeatInput{
			Session:           sess,
			AgentVersion:      in.Body.AgentVersion,
			AgentDigest:       in.Body.AgentDigest,
			UnavailableFor:    time.Duration(in.Body.UnavailableForSeconds) * time.Second,
			UnavailableReason: in.Body.UnavailableReason,
			Back:              in.Body.Back,
			Wait:              time.Duration(in.Body.WaitSeconds) * time.Second,
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &HeartbeatResponse{}
		out.Body.Task = toTaskView(res.Task)

		// Absent means HOLD. A failure to resolve it leaves it absent for the same
		// reason: a check that cannot reach a decision must not act, and the action
		// here would be flushing a host's firewall.
		if snap, err := svc.DesiredFirewallFor(ctx, sess); err != nil {
			slog.WarnContext(ctx, "could not resolve a bee's desired firewall; telling it to hold",
				"bee_id", sess.Bee.ID, "error", err)
		} else if snap != nil {
			out.Body.Firewall = &FirewallDesiredView{Revision: snap.Revision, Digest: snap.Digest}
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-firewall",
		Method:      http.MethodGet,
		Path:        "/agent/firewall",
		Summary:     "Fetch the ruleset this host should enforce",
		Description: "The complete nft ruleset, not a patch — it begins with a flush, so a host " +
			"that missed several revisions converges on this one rather than replaying a history " +
			"it never saw.\n\n" +
			"An EMPTY response means hold what you have. It is returned for a machine in no " +
			"fleet and for a fleet that has published nothing, and neither is an empty ruleset: " +
			"flushing there would strip a host of its management access at the moment it is " +
			"least reachable. Separate from the heartbeat because the body can be tens of " +
			"kilobytes and would otherwise ride on every poll from every host to say 'unchanged'.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityAgent,
	}, func(ctx context.Context, _ *AgentFirewallRequest) (*AgentFirewallResponse, error) {
		sess, ok := apimiddleware.BeeFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}
		snap, err := svc.DesiredFirewallFor(ctx, sess)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &AgentFirewallResponse{}
		if snap == nil {
			return out, nil
		}
		out.Body.Revision = snap.Revision
		out.Body.Digest = snap.Digest
		out.Body.Content = snap.Content
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-agent-dist",
		Method:      http.MethodGet,
		Path:        "/agent/dist",
		Summary:     "What agent binaries this control plane can serve",
		Description: "A hypervisor in a managed estate has no route to the internet, so the " +
			"control plane is the only host it can fetch a new agent from. Without this an " +
			"upgrade means copying a binary to every machine by hand, which is how a fleet ends " +
			"up running four different versions.\n\n" +
			"An EMPTY build list is not a failure. A server compiled without agent binaries is " +
			"an ordinary development build, and an agent reads it as nothing to do.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityAgent,
	}, func(ctx context.Context, _ *AgentDistRequest) (*AgentDistResponse, error) {
		if _, ok := apimiddleware.BeeFromContext(ctx); !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}
		dist, err := agentdist.Catalogue(buildinfo.Version)
		if err != nil {
			return nil, huma.Error500InternalServerError("could not read the agent distribution", err)
		}
		out := &AgentDistResponse{}
		out.Body.Version = dist.Version
		out.Body.Builds = make([]AgentBuildView, 0, len(dist.Builds))
		for _, b := range dist.Builds {
			out.Body.Builds = append(out.Body.Builds, AgentBuildView{
				OS: b.OS, Arch: b.Arch, Size: b.Size, SHA256: b.SHA256,
			})
		}
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-agent-binary",
		Method:      http.MethodGet,
		Path:        "/agent/dist/{os}/{arch}",
		Summary:     "Download the agent binary for one platform",
		Description: "Authenticated, even though the same bytes sit in the SPA's embedded tree. " +
			"Serving them as a static asset would hand the fleet's agent binary to anyone who " +
			"could reach the login page.\n\n" +
			"Verify the response against the sha256 from /agent/dist before running it.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityAgent,
	}, func(ctx context.Context, in *AgentBinaryRequest) (*AgentBinaryResponse, error) {
		if _, ok := apimiddleware.BeeFromContext(ctx); !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}
		body, err := agentdist.Binary(in.OS, in.Arch)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf(
				"this control plane carries no agent binary for %s/%s", in.OS, in.Arch))
		}
		return &AgentBinaryResponse{
			ContentType:        "application/octet-stream",
			ContentDisposition: `attachment; filename="apiary-agent"`,
			Body:               body,
		}, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-report-task",
		Method:      http.MethodPost,
		Path:        "/agent/tasks/{id}/status",
		Summary:     "Report progress or the outcome of a task",
		Description: "Terminal reports are idempotent. A bee that finishes and dies before its report is acknowledged will report again on restart, and rejecting the duplicate would strand the task against a host that considers it done.",
		Tags:        []string{"Fleet"},
		Security:    apimiddleware.SecurityAgent,
	}, func(ctx context.Context, in *ReportRequest) (*ReportResponse, error) {
		sess, ok := apimiddleware.BeeFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid task id")
		}

		task, err := svc.Report(ctx, service.ReportInput{
			Session: sess, TaskID: id,
			Status: in.Body.Status, Phase: in.Body.Phase, Error: in.Body.Error,
		})
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &ReportResponse{}
		out.Body.Status = task.Status
		return out, nil
	})

	huma.Register(hapi, huma.Operation{
		OperationID: "fleet-bee-append-logs",
		Method:      http.MethodPost,
		Path:        "/agent/tasks/{id}/logs",
		Summary:     "Stream task output back to the control plane",
		Description: "So a fleet operation can be watched without SSH-ing to a hypervisor mid-rotation, which during an evacuation is often the one thing you must not do. " +
			"Lines carry a per-task sequence assigned by the bee; resending a batch is a no-op.",
		Tags:     []string{"Fleet"},
		Security: apimiddleware.SecurityAgent,
	}, func(ctx context.Context, in *LogsRequest) (*LogsResponse, error) {
		sess, ok := apimiddleware.BeeFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not an authenticated bee")
		}
		id, err := uuid.Parse(in.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid task id")
		}

		lines := make([]service.LogLine, 0, len(in.Body.Lines))
		for _, l := range in.Body.Lines {
			at, _ := time.Parse(time.RFC3339, l.LoggedAt)
			lines = append(lines, service.LogLine{Seq: l.Seq, Stream: l.Stream, Message: l.Message, LoggedAt: at})
		}

		n, err := svc.AppendLogs(ctx, sess, id, lines)
		if err != nil {
			return nil, mapErr(ctx, err)
		}
		out := &LogsResponse{}
		out.Body.Accepted = n
		return out, nil
	})
	return nil
}

type AgentDistRequest struct{}

type AgentDistResponse struct {
	Body struct {
		Version string           `json:"version" doc:"The control plane's version. These binaries shipped with it, so an agent compares its own against this."`
		Builds  []AgentBuildView `json:"builds" doc:"Empty when this server carries no agent binaries, which is normal for a development build. An agent treats that as nothing to do rather than as a failure."`
	}
}

type AgentBuildView struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256" doc:"Verified by the agent before it installs anything. Without it an upgrade is 'execute whatever arrived'."`
}

type AgentBinaryRequest struct {
	OS   string `path:"os" maxLength:"16"`
	Arch string `path:"arch" maxLength:"16"`
}

// AgentBinaryResponse carries the executable itself.
//
// Raw bytes rather than base64 in JSON: the binary is around twenty megabytes,
// and base64 would inflate it by a third and force both ends to hold the whole
// thing in memory to encode and decode it.
type AgentBinaryResponse struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}
