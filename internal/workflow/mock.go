package workflow

import "context"

type MockEnqueuer struct{}

func (m *MockEnqueuer) EnqueueProvisionTenant(ctx context.Context, args *ProvisionTenantArgs) error {
	return nil
}

func (m *MockEnqueuer) EnqueueDestroyTenant(ctx context.Context, args *DestroyTenantArgs) error {
	return nil
}

// EnqueueSendEmail records nothing and succeeds: the mock exists so that a
// command which never runs a worker can still build a Deps.
func (m *MockEnqueuer) EnqueueSendEmail(context.Context, *SendEmailArgs) error { return nil }

func (m *MockEnqueuer) EnqueueDepartHypervisor(context.Context, *DepartHypervisorArgs) error {
	return nil
}
func (m *MockEnqueuer) EnqueueBootstrapHive(context.Context, *BootstrapHiveArgs) error {
	return nil
}

func (m *MockEnqueuer) EnqueueArriveHypervisor(context.Context, *ArriveHypervisorArgs) error {
	return nil
}
func (m *MockEnqueuer) EnqueueSyncHiveCapacity(context.Context, *SyncHiveCapacityArgs) error {
	return nil
}
