package repo

import (
	"context"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthAuditRepo interface {
	Create(ctx context.Context, e *control.AuthAuditEvent) error
	ListBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, limit, offset int) ([]control.AuthAuditEvent, error)
}

type AuthAuditRepoGorm struct {
	db *gorm.DB
}

func NewAuthAuditRepoGorm(db *gorm.DB) *AuthAuditRepoGorm {
	return &AuthAuditRepoGorm{db: db}
}

var _ AuthAuditRepo = (*AuthAuditRepoGorm)(nil)

func (r *AuthAuditRepoGorm) Create(ctx context.Context, e *control.AuthAuditEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *AuthAuditRepoGorm) ListBySubject(ctx context.Context, subjectID uuid.UUID, subjectKind string, limit, offset int) ([]control.AuthAuditEvent, error) {
	var events []control.AuthAuditEvent
	err := r.db.WithContext(ctx).
		Where("subject_id = ? AND subject_kind = ?", subjectID, subjectKind).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error
	return events, err
}
