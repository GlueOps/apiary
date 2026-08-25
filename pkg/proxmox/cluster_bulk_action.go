package proxmox

import (
	"context"
)

func (cl *Cluster) BulkActionSubdirs(ctx context.Context) ([]string, error) {
	return cl.bulkActionDiridx(ctx, "/cluster/bulk-action")
}

func (cl *Cluster) BulkActionGuestSubdirs(ctx context.Context) ([]string, error) {
	return cl.bulkActionDiridx(ctx, "/cluster/bulk-action/guest")
}

func (cl *Cluster) BulkStartGuests(ctx context.Context, opts *BulkStartOptions) (*Task, error) {
	if opts == nil {
		opts = &BulkStartOptions{}
	}
	var upid UPID
	if err := cl.client.Post(ctx, "/cluster/bulk-action/guest/start", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) BulkShutdownGuests(ctx context.Context, opts *BulkShutdownOptions) (*Task, error) {
	if opts == nil {
		opts = &BulkShutdownOptions{}
	}
	var upid UPID
	if err := cl.client.Post(ctx, "/cluster/bulk-action/guest/shutdown", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) BulkSuspendGuests(ctx context.Context, opts *BulkSuspendOptions) (*Task, error) {
	if opts == nil {
		opts = &BulkSuspendOptions{}
	}
	var upid UPID
	if err := cl.client.Post(ctx, "/cluster/bulk-action/guest/suspend", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) BulkMigrateGuests(ctx context.Context, opts *BulkMigrateOptions) (*Task, error) {
	if opts == nil {
		opts = &BulkMigrateOptions{}
	}
	var upid UPID
	if err := cl.client.Post(ctx, "/cluster/bulk-action/guest/migrate", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}
