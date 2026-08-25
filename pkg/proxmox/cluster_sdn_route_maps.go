package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func (cl *Cluster) SDNRouteMaps(ctx context.Context, running bool) (maps []*SDNRouteMapID, err error) {
	path := "/cluster/sdn/route-maps"
	if running {
		path += "?running=1"
	}
	err = cl.client.Get(ctx, path, &maps)
	return
}

func (cl *Cluster) SDNRouteMapEntries(ctx context.Context, pending, running bool) (entries []*SDNRouteMapEntry, err error) {
	path := "/cluster/sdn/route-maps/entries"
	q := url.Values{}
	if pending {
		q.Set("pending", "1")
	}
	if running {
		q.Set("running", "1")
	}
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	if err = cl.client.Get(ctx, path, &entries); err != nil {
		return nil, err
	}
	for _, e := range entries {
		e.client = cl.client
	}
	return
}

func (cl *Cluster) SDNRouteMapEntriesFor(ctx context.Context, routeMapID string, pending, running bool) (entries []*SDNRouteMapEntry, err error) {
	if routeMapID == "" {
		return nil, errors.New("route-map id is required")
	}
	path := fmt.Sprintf("/cluster/sdn/route-maps/entries/%s", routeMapID)
	q := url.Values{}
	if pending {
		q.Set("pending", "1")
	}
	if running {
		q.Set("running", "1")
	}
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	if err = cl.client.Get(ctx, path, &entries); err != nil {
		return nil, err
	}
	for _, e := range entries {
		e.client = cl.client
		e.RouteMapID = routeMapID
	}
	return
}

func (cl *Cluster) SDNRouteMapEntry(routeMapID string, order uint16) *SDNRouteMapEntry {
	return &SDNRouteMapEntry{client: cl.client, RouteMapID: routeMapID, Order: order}
}

func (cl *Cluster) NewSDNRouteMapEntry(ctx context.Context, opts *SDNRouteMapEntryOptions) error {
	if opts == nil || opts.RouteMapID == "" {
		return errors.New("route-map id is required")
	}
	if opts.Action == "" {
		return errors.New("route-map entry action is required")
	}
	return cl.client.Post(ctx, "/cluster/sdn/route-maps/entries", opts, nil)
}

func (e *SDNRouteMapEntry) Read(ctx context.Context) error {
	if e.RouteMapID == "" {
		return errors.New("route-map id is required")
	}
	return e.client.Get(ctx, fmt.Sprintf("/cluster/sdn/route-maps/entries/%s/entry/%s", e.RouteMapID, strconv.FormatUint(uint64(e.Order), 10)), e)
}

func (e *SDNRouteMapEntry) Update(ctx context.Context, opts *SDNRouteMapEntryOptions) error {
	if e.RouteMapID == "" {
		return errors.New("route-map id is required")
	}
	if opts == nil {
		opts = &SDNRouteMapEntryOptions{}
	}
	return e.client.Put(ctx, fmt.Sprintf("/cluster/sdn/route-maps/entries/%s/entry/%s", e.RouteMapID, strconv.FormatUint(uint64(e.Order), 10)), opts, nil)
}

func (e *SDNRouteMapEntry) Delete(ctx context.Context) error {
	if e.RouteMapID == "" {
		return errors.New("route-map id is required")
	}
	return e.client.Delete(ctx, fmt.Sprintf("/cluster/sdn/route-maps/entries/%s/entry/%s", e.RouteMapID, strconv.FormatUint(uint64(e.Order), 10)), nil)
}
