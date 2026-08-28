package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// AuditFilter narrows a read of the trail.
type AuditFilter struct {
	Action     string
	TargetKind string
	TargetID   *uuid.UUID
	Principal  *uuid.UUID
	Limit      int
	Offset     int
}

// RecordAudit writes one event.
//
// Refuses an event with no principal rather than storing a nil uuid. Everything
// acting inside the tenant boundary has one — a user, an organization key, an
// agent, or the scheduler itself — and a row that names none is indistinguishable
// from a row whose principal was lost, which is the one thing a trail must never
// be ambiguous about.
func RecordAudit(ctx context.Context, db *gorm.DB, e *tenant.AuditEvent) error {
	if e == nil {
		return fmt.Errorf("audit: nothing to record")
	}
	if e.PrincipalID == uuid.Nil || strings.TrimSpace(e.PrincipalKind) == "" {
		return fmt.Errorf("audit: refusing to record %q with no principal; an action nobody "+
			"can be named for is the one worth noticing, not the one worth storing anonymously",
			e.Action)
	}
	if strings.TrimSpace(e.Action) == "" {
		return fmt.Errorf("audit: refusing to record an event with no action")
	}
	if e.Outcome == "" {
		e.Outcome = tenant.AuditOutcomeOK
	}
	return db.WithContext(ctx).Create(e).Error
}

// ListAudit reads the trail newest first.
func ListAudit(ctx context.Context, db *gorm.DB, f AuditFilter) ([]tenant.AuditEvent, error) {
	q := db.WithContext(ctx).Model(&tenant.AuditEvent{})
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.TargetKind != "" {
		q = q.Where("target_kind = ?", f.TargetKind)
	}
	if f.TargetID != nil {
		q = q.Where("target_id = ?", *f.TargetID)
	}
	if f.Principal != nil {
		q = q.Where("principal_id = ?", *f.Principal)
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []tenant.AuditEvent
	// id breaks the tie on created_at. Several events written in one request
	// share a timestamp to the microsecond, and an unstable order there makes
	// paging skip and repeat rows.
	err := q.Order("created_at DESC").Order("id DESC").
		Limit(limit).Offset(f.Offset).Find(&out).Error
	return out, err
}

// AuditMetadata builds the jsonb column from a plain map.
func AuditMetadata(m map[string]any) datatypes.JSONType[map[string]any] {
	if m == nil {
		m = map[string]any{}
	}
	return datatypes.NewJSONType(m)
}
