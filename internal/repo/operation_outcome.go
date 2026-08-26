package repo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/tenant"
)

// operationErrorLimit caps what is stored.
//
// The value is rendered beside the machine, and an SSH failure can carry a wall
// of remote output. Truncated rather than dropped: the first line is almost
// always the reason, and a truncated message still tells an operator more than
// an empty column.
const operationErrorLimit = 1000

// RecordOperationOutcome stores why the last fleet operation on a machine
// failed, or clears it when one succeeds.
//
// Written for operations that have NO state column of their own — a departure,
// an arrival — where the only other record is a retryable job in a queue nobody
// reads. Provisioning has provisioning_state and provisioning_error and does not
// use this.
//
// The success path clears rather than leaves. A stale failure beside a machine
// that has since succeeded sends an operator to investigate something that is
// already fixed, which is worse than saying nothing.
func RecordOperationOutcome(
	ctx context.Context, db *gorm.DB, hypervisorID uuid.UUID, operation string, err error,
) error {
	detail := ""
	if err != nil {
		detail = err.Error()
		if len(detail) > operationErrorLimit {
			detail = detail[:operationErrorLimit]
		}
	}
	return db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("id = ?", hypervisorID).
		Updates(map[string]any{
			"last_operation":       operation,
			"last_operation_error": detail,
			"last_operation_at":    gorm.Expr("now()"),
		}).Error
}
