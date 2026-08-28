package fleet

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
)

// principalOf reads WHO is acting from the request's own principal.
//
// Derived rather than passed, because a parameter is a thing a handler can get
// wrong and this is the field the whole trail rests on. An organization key has
// no user id — it authenticates a tenant, not a person — so it is recorded as
// itself, by key id, rather than as a nil user. That distinction is the point:
// "the automation did this" and "we do not know who did this" must never look
// the same in a trail.
func principalOf(p *apimiddleware.Principal) (id uuid.UUID, kind string, ok bool) {
	if p == nil {
		return uuid.Nil, "", false
	}
	if p.UserID != nil {
		return *p.UserID, tenant.PrincipalKindUser, true
	}
	if p.APIKeyID != nil {
		return *p.APIKeyID, tenant.PrincipalKindOrgKey, true
	}
	return uuid.Nil, "", false
}

// auditEntry is one action about to be recorded.
type auditEntry struct {
	Action     string
	TargetKind string
	TargetID   *uuid.UUID
	Outcome    string
	Detail     string
	Metadata   map[string]any
}

// audit records an action against the tenant's trail.
//
// NEVER fails the request. An action that succeeded and could not be written
// down is worse reported as a failure — the caller would retry work that has
// already happened, which for a departure or a delete is not a safe thing to do
// twice. The write failing is itself logged, loudly, because a trail with silent
// holes is worse than one that is honestly incomplete.
func audit(ctx context.Context, db *gorm.DB, p *apimiddleware.Principal, e auditEntry) {
	id, kind, ok := principalOf(p)
	if !ok {
		slog.ErrorContext(ctx, "an action ran with no identifiable principal and was not audited",
			"action", e.Action, "target_kind", e.TargetKind)
		return
	}
	ev := &tenant.AuditEvent{
		PrincipalID:   id,
		PrincipalKind: kind,
		Action:        e.Action,
		TargetKind:    e.TargetKind,
		TargetID:      e.TargetID,
		Outcome:       e.Outcome,
		Detail:        e.Detail,
		Metadata:      repo.AuditMetadata(e.Metadata),
	}
	// Impersonation is recorded as both: the session acted as the customer, and
	// a named member of platform staff is behind it. Losing either half loses
	// the reason anyone would look.
	if p.IsImpersonating() {
		ev.ActorID = p.ActorID
	}
	if p.SessionJTI != "" {
		if sid, err := uuid.Parse(p.SessionJTI); err == nil {
			ev.SessionID = &sid
		}
	}
	if err := repo.RecordAudit(ctx, db, ev); err != nil {
		slog.ErrorContext(ctx, "an action was NOT written to the audit trail",
			"action", e.Action, "target_kind", e.TargetKind, "error", err)
	}
}

// auditOK records a successful action.
func auditOK(ctx context.Context, db *gorm.DB, p *apimiddleware.Principal,
	action, targetKind string, targetID *uuid.UUID, meta map[string]any) {
	audit(ctx, db, p, auditEntry{
		Action: action, TargetKind: targetKind, TargetID: targetID,
		Outcome: tenant.AuditOutcomeOK, Metadata: meta,
	})
}

// auditRefused records an action the control plane declined.
//
// Worth a row of its own. Someone who tried to depart a node and was told no is
// a different fact from someone who never tried, and the second is what an
// incident review reads as "nobody noticed".
func auditRefused(ctx context.Context, db *gorm.DB, p *apimiddleware.Principal,
	action, targetKind string, targetID *uuid.UUID, detail string, meta map[string]any) {
	audit(ctx, db, p, auditEntry{
		Action: action, TargetKind: targetKind, TargetID: targetID,
		Outcome: tenant.AuditOutcomeRefused, Detail: detail, Metadata: meta,
	})
}
