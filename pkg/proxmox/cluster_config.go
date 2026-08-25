package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (cl *Cluster) JoinAPIVersion(ctx context.Context) (version int, err error) {
	err = cl.client.Get(ctx, "/cluster/config/apiversion", &version)
	return
}

func (cl *Cluster) JoinInfo(ctx context.Context, node string) (info *ClusterJoinInfo, err error) {
	path := "/cluster/config/join"
	if node != "" {
		path = path + "?node=" + node
	}
	info = &ClusterJoinInfo{}
	err = cl.client.Get(ctx, path, info)
	return
}

func (cl *Cluster) JoinCluster(ctx context.Context, opts *ClusterJoinOptions) (status string, err error) {
	if opts == nil || opts.Hostname == "" {
		err = errors.New("cluster join: hostname is required")
		return
	}
	if opts.Password == "" {
		err = errors.New("cluster join: password is required")
		return
	}
	if opts.Fingerprint == "" {
		err = errors.New("cluster join: fingerprint is required")
		return
	}
	err = cl.client.Post(ctx, "/cluster/config/join", opts, &status)
	return
}

func (cl *Cluster) ConfigNodes(ctx context.Context) (nodes []*ClusterConfigNodeEntry, err error) {
	err = cl.client.Get(ctx, "/cluster/config/nodes", &nodes)
	return
}

func (cl *Cluster) QDevice(ctx context.Context) (status map[string]any, err error) {
	status = map[string]any{}
	err = cl.client.Get(ctx, "/cluster/config/qdevice", &status)
	return
}

func (cl *Cluster) Totem(ctx context.Context) (totem map[string]any, err error) {
	totem = map[string]any{}
	err = cl.client.Get(ctx, "/cluster/config/totem", &totem)
	return
}

func (cl *Cluster) CreateCluster(ctx context.Context, opts *ClusterCreateOptions) (status string, err error) {
	if opts == nil || opts.ClusterName == "" {
		err = errors.New("cluster create: clustername is required")
		return
	}
	err = cl.client.Post(ctx, "/cluster/config", opts, &status)
	return
}

func (cl *Cluster) AddConfigNode(ctx context.Context, node string, opts *ClusterAddNodeOptions) (result *ClusterAddNodeResult, err error) {
	if node == "" {
		err = errors.New("cluster add node: node is required")
		return
	}
	if opts == nil {
		opts = &ClusterAddNodeOptions{}
	}
	result = &ClusterAddNodeResult{}
	err = cl.client.Post(ctx, fmt.Sprintf("/cluster/config/nodes/%s", node), opts, result)
	return
}

func (cl *Cluster) DeleteConfigNode(ctx context.Context, node string) error {
	if node == "" {
		return errors.New("cluster delete node: node is required")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/config/nodes/%s", node), nil)
}
