package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) SDNControllers(ctx context.Context, typ string) (controllers []*SDNController, err error) {
	path := "/cluster/sdn/controllers"
	if typ != "" {
		q := url.Values{}
		q.Set("type", typ)
		path = path + "?" + q.Encode()
	}
	if err = cl.client.Get(ctx, path, &controllers); err != nil {
		return nil, err
	}
	for _, c := range controllers {
		c.client = cl.client
	}
	return
}

func (cl *Cluster) SDNController(name string) *SDNController {
	return &SDNController{client: cl.client, Controller: name}
}

func (cl *Cluster) NewSDNController(ctx context.Context, opts *SDNControllerOptions) error {
	if opts == nil || opts.Controller == "" {
		return errors.New("sdn controller name is required")
	}
	if opts.Type == "" {
		return errors.New("sdn controller type is required")
	}
	return cl.client.Post(ctx, "/cluster/sdn/controllers", opts, nil)
}

func (c *SDNController) Read(ctx context.Context) error {
	if c.Controller == "" {
		return errors.New("sdn controller name is required")
	}
	return c.client.Get(ctx, fmt.Sprintf("/cluster/sdn/controllers/%s", c.Controller), c)
}

func (c *SDNController) Update(ctx context.Context, opts *SDNControllerOptions) error {
	if c.Controller == "" {
		return errors.New("sdn controller name is required")
	}
	if opts == nil {
		opts = &SDNControllerOptions{}
	}
	return c.client.Put(ctx, fmt.Sprintf("/cluster/sdn/controllers/%s", c.Controller), opts, nil)
}

func (c *SDNController) Delete(ctx context.Context) error {
	if c.Controller == "" {
		return errors.New("sdn controller name is required")
	}
	return c.client.Delete(ctx, fmt.Sprintf("/cluster/sdn/controllers/%s", c.Controller), nil)
}
