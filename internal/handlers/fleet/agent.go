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
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
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

		UnavailableForSeconds int    `json:"unavailable_for_seconds,omitempty" minimum:"0" maximum:"86400" doc:"Declared BEFORE acting: how long this host expects to be unreachable, e.g. across a reboot. Until it elapses, silence from this bee is expected rather than a fault."`
		UnavailableReason     string `json:"unavailable_reason,omitempty" maxLength:"200"`
		Back                  bool   `json:"back,omitempty" doc:"Withdraws a standing declaration: the host has returned."`

		WaitSeconds int `json:"wait_seconds,omitempty" minimum:"0" maximum:"25" doc:"Hold the request open for up to this long waiting for a task. The response is the same shape either way."`
	}
}

type HeartbeatResponse struct {
	Body struct {
		Task *TaskView `json:"task,omitempty" doc:"This bee's one outstanding task, or absent. A bee that already holds a task is handed the SAME one back."`
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

		res, err := svc.Heartbeat(ctx, service.HeartbeatInput{
			Session:           sess,
			AgentVersion:      in.Body.AgentVersion,
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
		return out, nil
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
