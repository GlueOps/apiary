package jobs

import (
	"context"

	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
)

type TenantDestroyer interface {
	DestroyOrganization(ctx context.Context, organizationID string) error
}

type DestroyTenantWorker struct {
	river.WorkerDefaults[workflow.DestroyTenantArgs]
	destroyer TenantDestroyer
}

func NewDestroyTenantWorker(d TenantDestroyer) *DestroyTenantWorker {
	return &DestroyTenantWorker{destroyer: d}
}

func (w *DestroyTenantWorker) Work(ctx context.Context, job *river.Job[workflow.DestroyTenantArgs]) error {
	return w.destroyer.DestroyOrganization(ctx, job.Args.OrganizationID)
}
