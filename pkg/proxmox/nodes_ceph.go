package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func (n *Node) CephIndex(ctx context.Context) (entries []*CephIndexEntry, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph", n.Name), &entries)
	return
}

func (n *Node) InitCeph(ctx context.Context, opts *CephInitOptions) (*Task, error) {
	body := map[string]any{}
	if opts != nil {
		if opts.Network != "" {
			body["network"] = opts.Network
		}
		if opts.ClusterNetwork != "" {
			body["cluster-network"] = opts.ClusterNetwork
		}
		if opts.Size != 0 {
			body["size"] = opts.Size
		}
		if opts.MinSize != 0 {
			body["min_size"] = opts.MinSize
		}
		if opts.PGBits != 0 {
			body["pg_bits"] = opts.PGBits
		}
		if opts.DisableCephx {
			body["disable_cephx"] = 1
		}
	}
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/init", n.Name), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) StartCeph(ctx context.Context, service string) (*Task, error) {
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/start", n.Name), cephServiceBody(service), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) StopCeph(ctx context.Context, service string) (*Task, error) {
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/stop", n.Name), cephServiceBody(service), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) RestartCeph(ctx context.Context, service string) (*Task, error) {
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/restart", n.Name), cephServiceBody(service), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func cephServiceBody(service string) map[string]any {
	body := map[string]any{}
	if service != "" {
		body["service"] = service
	}
	return body
}

func (n *Node) CephStatus(ctx context.Context) (status *ClusterCephStatus, err error) {
	status = &ClusterCephStatus{}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/status", n.Name), status)
	return
}

func (n *Node) CephLog(ctx context.Context, start, limit int) (entries []*CephLogEntry, err error) {
	q := url.Values{}
	if start > 0 {
		q.Set("start", strconv.Itoa(start))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path := fmt.Sprintf("/nodes/%s/ceph/log", n.Name)
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	err = n.client.Get(ctx, path, &entries)
	return
}

func (n *Node) CephCrush(ctx context.Context) (crush string, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/crush", n.Name), &crush)
	return
}

func (n *Node) CephRules(ctx context.Context) (rules []*CephRule, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/rules", n.Name), &rules)
	return
}

func (n *Node) CephCmdSafety(ctx context.Context, service, id, action string) (safety *CephCmdSafety, err error) {
	if service == "" || id == "" || action == "" {
		return nil, errors.New("ceph cmd-safety requires service, id, and action")
	}
	q := url.Values{}
	q.Set("service", service)
	q.Set("id", id)
	q.Set("action", action)
	safety = &CephCmdSafety{}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/cmd-safety?%s", n.Name, q.Encode()), safety)
	return
}
