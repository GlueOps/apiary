package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) SDNIPAMs(ctx context.Context, typ string) (ipams []*SDNIPAM, err error) {
	path := "/cluster/sdn/ipams"
	if typ != "" {
		q := url.Values{}
		q.Set("type", typ)
		path = path + "?" + q.Encode()
	}
	if err = cl.client.Get(ctx, path, &ipams); err != nil {
		return nil, err
	}
	for _, i := range ipams {
		i.client = cl.client
	}
	return
}

func (cl *Cluster) SDNIPAM(name string) *SDNIPAM {
	return &SDNIPAM{client: cl.client, IPAM: name}
}

func (cl *Cluster) NewSDNIPAM(ctx context.Context, opts *SDNIPAMOptions) error {
	if opts == nil || opts.IPAM == "" {
		return errors.New("sdn ipam name is required")
	}
	if opts.Type == "" {
		return errors.New("sdn ipam type is required")
	}
	return cl.client.Post(ctx, "/cluster/sdn/ipams", opts, nil)
}

func (i *SDNIPAM) Read(ctx context.Context) error {
	if i.IPAM == "" {
		return errors.New("sdn ipam name is required")
	}
	return i.client.Get(ctx, fmt.Sprintf("/cluster/sdn/ipams/%s", i.IPAM), i)
}

func (i *SDNIPAM) Update(ctx context.Context, opts *SDNIPAMOptions) error {
	if i.IPAM == "" {
		return errors.New("sdn ipam name is required")
	}
	if opts == nil {
		opts = &SDNIPAMOptions{}
	}
	return i.client.Put(ctx, fmt.Sprintf("/cluster/sdn/ipams/%s", i.IPAM), opts, nil)
}

func (i *SDNIPAM) Delete(ctx context.Context) error {
	if i.IPAM == "" {
		return errors.New("sdn ipam name is required")
	}
	return i.client.Delete(ctx, fmt.Sprintf("/cluster/sdn/ipams/%s", i.IPAM), nil)
}

func (i *SDNIPAM) Status(ctx context.Context) (entries []map[string]any, err error) {
	if i.IPAM == "" {
		return nil, errors.New("sdn ipam name is required")
	}
	err = i.client.Get(ctx, fmt.Sprintf("/cluster/sdn/ipams/%s/status", i.IPAM), &entries)
	return
}
