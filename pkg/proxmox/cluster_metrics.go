package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func (cl *Cluster) MetricsSubdirs(ctx context.Context) ([]string, error) {
	return cl.metricsDiridx(ctx, "/cluster/metrics")
}

type MetricsExportOptions struct {
	History bool

	LocalOnly bool

	StartTime int64

	NodeList string
}

func (cl *Cluster) MetricsExport(ctx context.Context, opts *MetricsExportOptions) (export *MetricsExportResponse, err error) {
	path := "/cluster/metrics/export"
	if opts != nil {
		q := url.Values{}
		if opts.History {
			q.Set("history", "1")
		}
		if opts.LocalOnly {
			q.Set("local-only", "1")
		}
		if opts.StartTime > 0 {
			q.Set("start-time", strconv.FormatInt(opts.StartTime, 10))
		}
		if opts.NodeList != "" {
			q.Set("node-list", opts.NodeList)
		}
		if enc := q.Encode(); enc != "" {
			path = path + "?" + enc
		}
	}
	export = &MetricsExportResponse{}
	err = cl.client.Get(ctx, path, export)
	return
}

func (cl *Cluster) MetricServers(ctx context.Context) (servers ClusterMetricServers, err error) {
	err = cl.client.Get(ctx, "/cluster/metrics/server", &servers)
	return
}

func (cl *Cluster) MetricServer(ctx context.Context, id string) (server *ClusterMetricServer, err error) {
	if id == "" {
		err = errors.New("metric server id can not be empty")
		return
	}
	server = &ClusterMetricServer{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/metrics/server/%s", id), server); err != nil {
		return
	}
	if server.ID == "" {
		server.ID = id
	}
	return
}

func (cl *Cluster) NewMetricServer(ctx context.Context, opts *ClusterMetricServerOptions) error {
	if opts == nil || opts.ID == "" {
		return errors.New("metric server id can not be empty")
	}

	return cl.client.Post(ctx, fmt.Sprintf("/cluster/metrics/server/%s", opts.ID), opts, nil)
}

func (cl *Cluster) UpdateMetricServer(ctx context.Context, id string, opts *ClusterMetricServerOptions) error {
	if id == "" {
		return errors.New("metric server id can not be empty")
	}
	if opts == nil {
		opts = &ClusterMetricServerOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/metrics/server/%s", id), opts, nil)
}

func (cl *Cluster) DeleteMetricServer(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("metric server id can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/metrics/server/%s", id), nil)
}
