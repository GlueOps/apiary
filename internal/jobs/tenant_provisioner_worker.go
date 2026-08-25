package jobs

import (
	"context"

	"github.com/riverqueue/river"
)

type TenantProvisioner interface {
	ProvisionOrganization(ctx context.Context, organizationID string) error
}

type ProvisionTenantWorker struct {
	river.WorkerDefaults[ProvisionTenantArgs]
	provisioner TenantProvisioner
}

func NewProvisionTenantWorker(p TenantProvisioner) *ProvisionTenantWorker {
	return &ProvisionTenantWorker{provisioner: p}
}

func (w *ProvisionTenantWorker) Work(ctx context.Context, job *river.Job[ProvisionTenantArgs]) error {
	return w.provisioner.ProvisionOrganization(ctx, job.Args.OrganizationID)
}
