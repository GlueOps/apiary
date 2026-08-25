package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (n *Node) CephOSDs(ctx context.Context) (tree *CephOSDTree, err error) {
	return tree, n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/osd", n.Name), &tree)
}

func (n *Node) CreateCephOSD(ctx context.Context, opts *CephOSDCreateOptions) (*Task, error) {
	if opts == nil || opts.Dev == "" {
		return nil, errors.New("ceph osd dev is required")
	}
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/osd", n.Name), opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) CephOSD(osdID int) *CephOSD {
	return &CephOSD{client: n.client, Node: n.Name, ID: osdID}
}

func (o *CephOSD) SubResources(ctx context.Context) (items []map[string]interface{}, err error) {
	return items, o.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/osd/%d", o.Node, o.ID), &items)
}

func (o *CephOSD) Delete(ctx context.Context, cleanup bool) (*Task, error) {
	q := url.Values{}
	if cleanup {
		q.Set("cleanup", "1")
	}
	path := fmt.Sprintf("/nodes/%s/ceph/osd/%d", o.Node, o.ID)
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	var upid UPID
	if err := o.client.Delete(ctx, path, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, o.client), nil
}

func (o *CephOSD) In(ctx context.Context) error {
	return o.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/osd/%d/in", o.Node, o.ID), nil, nil)
}

func (o *CephOSD) Out(ctx context.Context) error {
	return o.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/osd/%d/out", o.Node, o.ID), nil, nil)
}

func (o *CephOSD) Scrub(ctx context.Context, deep bool) error {
	body := map[string]interface{}{}
	if deep {
		body["deep"] = 1
	}
	return o.client.Post(ctx, fmt.Sprintf("/nodes/%s/ceph/osd/%d/scrub", o.Node, o.ID), body, nil)
}

func (o *CephOSD) LVInfo(ctx context.Context, devType string) (info *CephOSDLVInfo, err error) {
	path := fmt.Sprintf("/nodes/%s/ceph/osd/%d/lv-info", o.Node, o.ID)
	if devType != "" {
		q := url.Values{}
		q.Set("type", devType)
		path = path + "?" + q.Encode()
	}
	return info, o.client.Get(ctx, path, &info)
}

func (o *CephOSD) Metadata(ctx context.Context) (details *CephOSDDetails, err error) {
	return details, o.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/osd/%d/metadata", o.Node, o.ID), &details)
}
