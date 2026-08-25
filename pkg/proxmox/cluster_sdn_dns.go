package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) SDNDNSList(ctx context.Context, typ string) (dns []*SDNDNS, err error) {
	path := "/cluster/sdn/dns"
	if typ != "" {
		q := url.Values{}
		q.Set("type", typ)
		path = path + "?" + q.Encode()
	}
	if err = cl.client.Get(ctx, path, &dns); err != nil {
		return nil, err
	}
	for _, d := range dns {
		d.client = cl.client
	}
	return
}

func (cl *Cluster) SDNDNS(name string) *SDNDNS {
	return &SDNDNS{client: cl.client, DNS: name}
}

func (cl *Cluster) NewSDNDNS(ctx context.Context, opts *SDNDNSOptions) error {
	if opts == nil || opts.DNS == "" {
		return errors.New("sdn dns name is required")
	}
	if opts.Type == "" {
		return errors.New("sdn dns type is required")
	}
	return cl.client.Post(ctx, "/cluster/sdn/dns", opts, nil)
}

func (d *SDNDNS) Read(ctx context.Context) error {
	if d.DNS == "" {
		return errors.New("sdn dns name is required")
	}
	return d.client.Get(ctx, fmt.Sprintf("/cluster/sdn/dns/%s", d.DNS), d)
}

func (d *SDNDNS) Update(ctx context.Context, opts *SDNDNSOptions) error {
	if d.DNS == "" {
		return errors.New("sdn dns name is required")
	}
	if opts == nil {
		opts = &SDNDNSOptions{}
	}
	return d.client.Put(ctx, fmt.Sprintf("/cluster/sdn/dns/%s", d.DNS), opts, nil)
}

func (d *SDNDNS) Delete(ctx context.Context) error {
	if d.DNS == "" {
		return errors.New("sdn dns name is required")
	}
	return d.client.Delete(ctx, fmt.Sprintf("/cluster/sdn/dns/%s", d.DNS), nil)
}
