package proxmox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

func (n *Node) CephPools(ctx context.Context) (pools []*CephPool, err error) {
	if err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/pool", n.Name), &pools); err != nil {
		return nil, err
	}
	for _, p := range pools {
		p.client = n.client
		p.Node = n.Name
	}
	return pools, nil
}

func (n *Node) CreateCephPool(ctx context.Context, opts *CephPoolOptions) (*Task, error) {
	if opts == nil || opts.Name == "" {
		return nil, errors.New("ceph pool name is required")
	}
	body, err := cephPoolBody(opts)
	if err != nil {
		return nil, err
	}
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/pool", n.Name), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) CephPool(name string) *CephPool {
	return &CephPool{client: n.client, Node: n.Name, PoolName: name}
}

func cephPoolBody(opts *CephPoolOptions) (map[string]any, error) {
	raw, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if opts.ErasureCoding != nil {
		body["erasure-coding"] = opts.ErasureCoding.String()
	}
	return body, nil
}

func (p *CephPool) SubResources(ctx context.Context) (subdirs []*CephPoolSubdir, err error) {
	if p.PoolName == "" {
		return nil, errors.New("ceph pool name is required")
	}
	return subdirs, p.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/pool/%s", p.Node, p.PoolName), &subdirs)
}

func (p *CephPool) Update(ctx context.Context, opts *CephPoolOptions) (*Task, error) {
	if p.PoolName == "" {
		return nil, errors.New("ceph pool name is required")
	}
	if opts == nil {
		return nil, errors.New("ceph pool options are required")
	}

	body, err := cephPoolBody(opts)
	if err != nil {
		return nil, err
	}
	delete(body, "name")
	var upid UPID
	if err := p.client.Put(ctx, fmt.Sprintf("/nodes/%s/ceph/pool/%s", p.Node, p.PoolName), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, p.client), nil
}

func (p *CephPool) Delete(ctx context.Context, force, removeStorages, removeECProfile bool) (*Task, error) {
	if p.PoolName == "" {
		return nil, errors.New("ceph pool name is required")
	}
	q := url.Values{}
	if force {
		q.Set("force", "1")
	}
	if removeStorages {
		q.Set("remove_storages", "1")
	}

	if !removeECProfile {
		q.Set("remove_ecprofile", "0")
	}
	path := fmt.Sprintf("/nodes/%s/ceph/pool/%s", p.Node, p.PoolName)
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var upid UPID
	if err := p.client.Delete(ctx, path, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, p.client), nil
}

func (p *CephPool) Status(ctx context.Context, verbose bool) (status *CephPoolStatus, err error) {
	if p.PoolName == "" {
		return nil, errors.New("ceph pool name is required")
	}
	path := fmt.Sprintf("/nodes/%s/ceph/pool/%s/status", p.Node, p.PoolName)
	if verbose {
		path += "?verbose=1"
	}
	return status, p.client.Get(ctx, path, &status)
}
