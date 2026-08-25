package proxmox

import (
	"context"
	"fmt"
)

func (n *Node) Subdirs(ctx context.Context) ([]string, error) {
	return n.nodeDiridx(ctx, fmt.Sprintf("/nodes/%s", n.Name))
}

func (n *Node) FirewallSubdirs(ctx context.Context) ([]string, error) {
	return n.firewallDiridx(ctx, fmt.Sprintf("/nodes/%s/firewall", n.Name))
}

func (n *Node) DisksSubdirs(ctx context.Context) ([]string, error) {
	return n.disksDiridx(ctx, fmt.Sprintf("/nodes/%s/disks", n.Name))
}

func (r *NodeReplicationJob) Subdirs(ctx context.Context) ([]string, error) {
	if r.ID == "" {
		return nil, fmt.Errorf("replication id is required")
	}
	return r.replicationDiridx(ctx, fmt.Sprintf("/nodes/%s/replication/%s", r.Node, r.ID))
}

func (s *NodeService) Subdirs(ctx context.Context) ([]string, error) {
	if s.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}
	return s.serviceDiridx(ctx, fmt.Sprintf("/nodes/%s/services/%s", s.Node, s.Name))
}

func (t *Task) Subdirs(ctx context.Context) ([]string, error) {
	if t.UPID == "" {
		return nil, fmt.Errorf("task upid is required")
	}
	return t.taskDiridx(ctx, fmt.Sprintf("/nodes/%s/tasks/%s", t.Node, t.UPID))
}

type StorageStatus struct {
	Type string `json:"type,omitempty"`

	Content string `json:"content,omitempty"`

	Active int `json:"active,omitempty"`

	Enabled int `json:"enabled,omitempty"`

	Shared int `json:"shared,omitempty"`

	Total uint64 `json:"total,omitempty"`

	Used uint64 `json:"used,omitempty"`

	Avail uint64 `json:"avail,omitempty"`

	UsedFraction float64 `json:"used_fraction,omitempty"`
}

func (s *Storage) Status(ctx context.Context) (status *StorageStatus, err error) {
	err = s.client.Get(ctx, fmt.Sprintf("/nodes/%s/storage/%s", s.Node, s.Name), &status)
	return
}

func (n *Node) nodeDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, n.client, path)
}

func (n *Node) firewallDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, n.client, path)
}

func (n *Node) disksDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, n.client, path)
}

func (r *NodeReplicationJob) replicationDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, r.client, path)
}

func (s *NodeService) serviceDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, s.client, path)
}

func (t *Task) taskDiridx(ctx context.Context, path string) ([]string, error) {
	return decodeSubdirList(ctx, t.client, path)
}

func decodeSubdirList(ctx context.Context, c *Client, path string) ([]string, error) {
	var items []struct {
		Subdir string `json:"subdir"`
	}
	if err := c.Get(ctx, path, &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Subdir)
	}
	return out, nil
}
