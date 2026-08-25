package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (n *Node) HardwareIndex(ctx context.Context) ([]string, error) {
	var items []struct {
		Type string `json:"type"`
	}
	if err := n.client.Get(ctx, fmt.Sprintf("/nodes/%s/hardware", n.Name), &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Type)
	}
	return out, nil
}

type HardwarePCIOptions struct {
	ClassBlacklist []string
	Terse          bool
}

func (n *Node) ListPCIDevices(ctx context.Context, opts *HardwarePCIOptions) (devices []*PCIDevice, err error) {
	path := fmt.Sprintf("/nodes/%s/hardware/pci", n.Name)
	if opts != nil {
		q := url.Values{}
		if len(opts.ClassBlacklist) > 0 {
			q.Set("pci-class-blacklist", strings.Join(opts.ClassBlacklist, ";"))
		}
		if opts.Terse {
			q.Set("verbose", "0")
		}
		if len(q) > 0 {
			path = path + "?" + q.Encode()
		}
	}
	if err = n.client.Get(ctx, path, &devices); err != nil {
		return
	}
	for _, d := range devices {
		d.client = n.client
		d.Node = n.Name
	}
	return
}

func (n *Node) ListUSBDevices(ctx context.Context) (devices []*USBDevice, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/hardware/usb", n.Name), &devices)
	return
}

func (n *Node) PCIDevice(id string) *PCIDevice {
	return &PCIDevice{
		client: n.client,
		Node:   n.Name,
		ID:     id,
	}
}

func (d *PCIDevice) Index(ctx context.Context) ([]string, error) {
	if d.ID == "" {
		return nil, errors.New("pci id is required")
	}
	var items []struct {
		Method string `json:"method"`
	}
	if err := d.client.Get(ctx, fmt.Sprintf("/nodes/%s/hardware/pci/%s", d.Node, d.ID), &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Method)
	}
	return out, nil
}

func (d *PCIDevice) Mdev(ctx context.Context) (types []*PCIMdevType, err error) {
	if d.ID == "" {
		return nil, errors.New("pci id is required")
	}
	err = d.client.Get(ctx, fmt.Sprintf("/nodes/%s/hardware/pci/%s/mdev", d.Node, d.ID), &types)
	return
}
