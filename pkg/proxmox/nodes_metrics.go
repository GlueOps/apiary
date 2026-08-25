package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (n *Node) RRD(ctx context.Context, ds string, timeframe Timeframe, cf ConsolidationFunction) (rrd *NodeRRDImage, err error) {
	if ds == "" {
		return nil, errors.New("ds is required")
	}
	q := url.Values{}
	q.Set("ds", ds)
	q.Set("timeframe", string(timeframe))
	if cf != "" {
		q.Set("cf", string(cf))
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/rrd?%s", n.Name, q.Encode()), &rrd)
	return
}

func (n *Node) RRDData(ctx context.Context, timeframe Timeframe, cf ConsolidationFunction) (data []*RRDData, err error) {
	q := url.Values{}
	q.Set("timeframe", string(timeframe))
	if cf != "" {
		q.Set("cf", string(cf))
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/rrddata?%s", n.Name, q.Encode()), &data)
	return
}

func (n *Node) QueryURLMetadata(ctx context.Context, fileURL string, verifyTLS *bool) (meta *NodeURLMetadata, err error) {
	if fileURL == "" {
		return nil, errors.New("url is required")
	}
	q := url.Values{}
	q.Set("url", fileURL)
	if verifyTLS != nil && !*verifyTLS {
		q.Set("verify-certificates", "0")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/query-url-metadata?%s", n.Name, q.Encode()), &meta)
	return
}

func (n *Node) QueryOCIRepoTags(ctx context.Context, reference string) (tags []string, err error) {
	if reference == "" {
		return nil, errors.New("reference is required")
	}
	q := url.Values{}
	q.Set("reference", reference)
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/query-oci-repo-tags?%s", n.Name, q.Encode()), &tags)
	return
}

func (n *Node) VzdumpDefaults(ctx context.Context, storage string) (defaults map[string]any, err error) {
	path := fmt.Sprintf("/nodes/%s/vzdump/defaults", n.Name)
	if storage != "" {
		q := url.Values{}
		q.Set("storage", storage)
		path = path + "?" + q.Encode()
	}
	err = n.client.Get(ctx, path, &defaults)
	return
}
