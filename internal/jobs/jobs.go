package jobs

import (
	"context"

	"github.com/glueops/apiary/internal/workflow"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

type RiverEnqueuer struct {
	client *river.Client[pgx.Tx]
}

func NewEnqueuer(pool *pgxpool.Pool) (*RiverEnqueuer, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, err
	}
	return &RiverEnqueuer{client: client}, nil
}

var _ workflow.Enqueuer = (*RiverEnqueuer)(nil)

func (e *RiverEnqueuer) EnqueueProvisionTenant(ctx context.Context, args *workflow.ProvisionTenantArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

func (e *RiverEnqueuer) EnqueueDestroyTenant(ctx context.Context, args *workflow.DestroyTenantArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

func (e *RiverEnqueuer) EnqueueSendEmail(ctx context.Context, args *workflow.SendEmailArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

func (e *RiverEnqueuer) EnqueueDepartHypervisor(ctx context.Context, args *workflow.DepartHypervisorArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

func (e *RiverEnqueuer) EnqueueArriveHypervisor(ctx context.Context, args *workflow.ArriveHypervisorArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}

// EnqueueSyncHiveCapacity is deliberately tolerant of River's unique-job
// rejection. The job is unique per hive while queued or running, so an operator
// pressing "sync now" twice — or pressing it while the scheduled tick already
// has one in flight — gets the sync that is already coming rather than an error
// about a duplicate they cannot see.
func (e *RiverEnqueuer) EnqueueSyncHiveCapacity(ctx context.Context, args *workflow.SyncHiveCapacityArgs) error {
	_, err := e.client.Insert(ctx, args, nil)
	return err
}
