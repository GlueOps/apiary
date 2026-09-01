package repo

import (
	"context"
	"log/slog"

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
//
// Called for the JOB-driven operations, which is now only the ones that still
// run as jobs. Agent-driven work reports through RecordTaskOutcomeForBee below;
// the two write the same columns and follow the same rule about whose failure a
// success is entitled to clear.
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

// RecordTaskOutcomeForBee stores why an agent task failed on the machine it ran
// on, or retires that machine's record of the same task having failed before.
//
// This is the writer the fleet view was missing. The operations an operator
// actually watches — disarm, upgrade, and every convergence behind them — moved
// to agent tasks, and a task that failed recorded its reason on the TASK, where
// only a database query could reach it. Beside the machine there was nothing, so
// a host that could not be disarmed looked exactly like one nobody had asked to
// disarm.
//
// SUCCESS CLEARS ONLY ITS OWN KIND. last_operation is a single slot, so a
// successful agent upgrade that cleared unconditionally would erase a disarm
// failure nobody has resolved — replacing a true statement with silence, on a
// machine still holding a departed cluster's credentials. The same rule as
// MarkFleetless and MarkJoined: an operation is entitled to retire its own
// previous failure and nothing else's.
//
// Best effort by construction. It is a legibility record, and failing an agent's
// task report because the summary could not be written would lose the report
// itself — which is the thing that actually drives convergence.
func RecordTaskOutcomeForBee(
	ctx context.Context, db *gorm.DB, beeID uuid.UUID, kind, taskError string,
) {
	op := OperationNameForTaskKind(kind)
	if op == "" {
		return
	}
	updates := map[string]any{}
	if taskError != "" {
		detail := taskError
		if len(detail) > operationErrorLimit {
			detail = detail[:operationErrorLimit]
		}
		updates["last_operation"] = op
		updates["last_operation_error"] = detail
		updates["last_operation_at"] = gorm.Expr("now()")
	} else {
		updates["last_operation_error"] = gorm.Expr(
			"CASE WHEN last_operation = ? THEN '' ELSE last_operation_error END", op)
	}
	if err := db.WithContext(ctx).Model(&tenant.Hypervisor{}).
		Where("bee_id = ?", beeID).Updates(updates).Error; err != nil {
		slog.ErrorContext(ctx, "an agent task finished and its outcome could not be recorded "+
			"beside the machine, so the fleet view will not show why it failed",
			"bee_id", beeID, "kind", kind, "error", err)
	}
}

// OperationNameForTaskKind turns a task kind into the word the fleet view puts
// in "last <operation> failed".
//
// A mapping rather than a string trim, because the kinds are a wire contract
// shared with the agent and the labels are prose an operator reads. Trimming
// "fleet.host." off a kind produces "agent-upgrade", which renders as "last
// agent-upgrade failed" — a machine name in a sentence meant for a person.
//
// An unrecognised kind returns empty and records nothing. A new kind should be
// added here deliberately, and until it is, saying nothing beside the machine is
// better than labelling it with a wire constant.
func OperationNameForTaskKind(kind string) string {
	switch kind {
	case KindDisarm:
		return "disarm"
	case KindAgentUpgrade:
		return "agent upgrade"
	case KindDistUpgrade:
		return "system upgrade"
	default:
		return ""
	}
}
