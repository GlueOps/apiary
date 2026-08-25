package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (v *VNet) Subnets(ctx context.Context) (subnets []*VNetSubnet, err error) {
	if v.Name == "" {
		return nil, errors.New("vnet name is required")
	}
	if err = v.client.Get(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/subnets", v.Name), &subnets); err != nil {
		return nil, err
	}
	for _, s := range subnets {
		s.client = v.client
		s.VNet = v.Name
	}
	return
}

func (v *VNet) Subnet(name string) *VNetSubnet {
	return &VNetSubnet{client: v.client, VNet: v.Name, ID: name}
}

func (v *VNet) NewSubnet(ctx context.Context, opts *SDNSubnetOptions) error {
	if v.Name == "" {
		return errors.New("vnet name is required")
	}
	if opts == nil || opts.Subnet == "" {
		return errors.New("subnet cidr is required")
	}
	if opts.Type == "" {
		opts.Type = "subnet"
	}
	if opts.VNet == "" {
		opts.VNet = v.Name
	}
	return v.client.Post(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/subnets", v.Name), opts, nil)
}

func (s *VNetSubnet) Read(ctx context.Context) error {
	if s.VNet == "" || s.ID == "" {
		return errors.New("vnet and subnet id are required")
	}
	return s.client.Get(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/subnets/%s", s.VNet, s.ID), s)
}

func (s *VNetSubnet) Update(ctx context.Context, opts *SDNSubnetOptions) error {
	if s.VNet == "" || s.ID == "" {
		return errors.New("vnet and subnet id are required")
	}
	if opts == nil {
		opts = &SDNSubnetOptions{}
	}
	return s.client.Put(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/subnets/%s", s.VNet, s.ID), opts, nil)
}

func (s *VNetSubnet) Delete(ctx context.Context) error {
	if s.VNet == "" || s.ID == "" {
		return errors.New("vnet and subnet id are required")
	}
	return s.client.Delete(ctx, fmt.Sprintf("/cluster/sdn/vnets/%s/subnets/%s", s.VNet, s.ID), nil)
}
