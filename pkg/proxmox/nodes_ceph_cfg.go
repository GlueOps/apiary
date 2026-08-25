package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (n *Node) CephCfg(ctx context.Context) (entries []map[string]interface{}, err error) {
	return entries, n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/cfg", n.Name), &entries)
}

func (n *Node) CephCfgDB(ctx context.Context) (entries []*CephCfgDBEntry, err error) {
	return entries, n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/cfg/db", n.Name), &entries)
}

func (n *Node) CephCfgRaw(ctx context.Context) (raw string, err error) {
	return raw, n.client.Get(ctx, fmt.Sprintf("/nodes/%s/ceph/cfg/raw", n.Name), &raw)
}

func (n *Node) CephCfgValue(ctx context.Context, configKeys string) (values CephCfgValue, err error) {
	if configKeys == "" {
		return nil, errors.New("config-keys is required")
	}
	q := url.Values{}
	q.Set("config-keys", configKeys)
	path := fmt.Sprintf("/nodes/%s/ceph/cfg/value?%s", n.Name, q.Encode())
	return values, n.client.Get(ctx, path, &values)
}
