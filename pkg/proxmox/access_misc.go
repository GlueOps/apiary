package proxmox

import (
	"context"
	"errors"
)

func (c *Client) AccessIndex(ctx context.Context) ([]string, error) {
	var items []struct {
		Subdir string `json:"subdir"`
	}
	if err := c.Get(ctx, "/access", &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Subdir)
	}
	return out, nil
}

func (c *Client) OpenIDIndex(ctx context.Context) ([]string, error) {
	var items []struct {
		Subdir string `json:"subdir"`
	}
	if err := c.Get(ctx, "/access/openid", &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Subdir)
	}
	return out, nil
}

func (c *Client) GetTicket(ctx context.Context) error {
	return c.Get(ctx, "/access/ticket", nil)
}

type VerifyVNCTicketOptions struct {
	AuthID    string `json:"authid"`
	Path      string `json:"path"`
	Privs     string `json:"privs"`
	VNCTicket string `json:"vncticket"`
	Port      int    `json:"port,omitempty"`
}

func (c *Client) VerifyVNCTicket(ctx context.Context, opts *VerifyVNCTicketOptions) error {
	if opts == nil || opts.AuthID == "" || opts.Path == "" || opts.VNCTicket == "" {
		return errors.New("authid, path, and vncticket are required")
	}
	return c.Post(ctx, "/access/vncticket", opts, nil)
}
