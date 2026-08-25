package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (cl *Cluster) CephFlags(ctx context.Context) (flags []*CephFlag, err error) {
	err = cl.client.Get(ctx, "/cluster/ceph/flags", &flags)
	return
}

func (cl *Cluster) CephFlag(ctx context.Context, flag string) (value string, err error) {
	if flag == "" {
		err = errors.New("ceph flag: name is required")
		return
	}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/ceph/flags/%s", flag), &value)
	return
}

func (cl *Cluster) CephMetadata(ctx context.Context) (meta *CephMetadata, err error) {
	meta = &CephMetadata{}
	err = cl.client.Get(ctx, "/cluster/ceph/metadata", meta)
	return
}

func (cl *Cluster) SetCephFlags(ctx context.Context, opts *CephFlagsUpdateOptions) (*Task, error) {
	if opts == nil {
		opts = &CephFlagsUpdateOptions{}
	}
	var upid UPID
	if err := cl.client.Put(ctx, "/cluster/ceph/flags", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) SetCephFlag(ctx context.Context, flag string, value bool) error {
	if flag == "" {
		return errors.New("ceph flag: name is required")
	}
	body := map[string]bool{"value": value}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/ceph/flags/%s", flag), body, nil)
}
