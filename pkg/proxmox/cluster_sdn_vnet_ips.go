package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (v *VNet) CreateIP(ctx context.Context, opts *SDNVNetIPOptions) error {
	if v.Name == "" {
		return errors.New("vnet name is required")
	}
	if opts == nil || opts.IP == "" {
		return errors.New("vnet ip is required")
	}
	if opts.Zone == "" {
		return errors.New("vnet zone is required")
	}
	return v.client.Post(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/ips", v.Name), opts, nil)
}

func (v *VNet) UpdateIP(ctx context.Context, opts *SDNVNetIPOptions) error {
	if v.Name == "" {
		return errors.New("vnet name is required")
	}
	if opts == nil || opts.IP == "" {
		return errors.New("vnet ip is required")
	}
	if opts.Zone == "" {
		return errors.New("vnet zone is required")
	}
	return v.client.Put(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/ips", v.Name), opts, nil)
}

func (v *VNet) DeleteIP(ctx context.Context, opts *SDNVNetIPOptions) error {
	if v.Name == "" {
		return errors.New("vnet name is required")
	}
	if opts == nil || opts.IP == "" {
		return errors.New("vnet ip is required")
	}
	if opts.Zone == "" {
		return errors.New("vnet zone is required")
	}

	path := fmt.Sprintf("/cluster/sdn/vnets/%s/ips?zone=%s&ip=%s", v.Name, opts.Zone, opts.IP)
	if opts.MAC != "" {
		path += "&mac=" + opts.MAC
	}
	return v.client.Delete(ctx, path, nil)
}
