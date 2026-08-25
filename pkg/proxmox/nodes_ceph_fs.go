package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (n *Node) CephFSs(ctx context.Context) (fss []*CephFS, err error) {
	if err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/fs", n.Name), &fss); err != nil {
		return nil, err
	}
	for _, f := range fss {
		f.client = n.client
		f.Node = n.Name
	}
	return fss, nil
}

func (n *Node) CreateCephFS(ctx context.Context, name string, opts *CephFSOptions) (*Task, error) {
	if name == "" {
		return nil, errors.New("cephfs name is required")
	}
	if opts == nil {
		opts = &CephFSOptions{}
	}
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/fs/%s", n.Name, name), opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) CephFS(name string) *CephFS {
	return &CephFS{client: n.client, Node: n.Name, Name: name}
}

func (f *CephFS) Delete(ctx context.Context, removePools, removeStorages bool) (*Task, error) {
	if f.Name == "" {
		return nil, errors.New("cephfs name is required")
	}
	q := url.Values{}
	if removePools {
		q.Set("remove-pools", "1")
	}
	if removeStorages {
		q.Set("remove-storages", "1")
	}
	path := fmt.Sprintf("/nodes/%s/ceph/fs/%s", f.Node, f.Name)
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var upid UPID
	if err := f.client.Delete(ctx, path, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, f.client), nil
}
