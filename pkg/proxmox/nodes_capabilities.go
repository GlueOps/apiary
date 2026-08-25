package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

func (n *Node) CapabilitiesIndex(ctx context.Context) ([]string, error) {
	return n.capabilitiesDiridx(ctx, fmt.Sprintf("/nodes/%s/capabilities", n.Name))
}

func (n *Node) QEMUCapabilitiesIndex(ctx context.Context) ([]string, error) {
	return n.capabilitiesDiridx(ctx, fmt.Sprintf("/nodes/%s/capabilities/qemu", n.Name))
}

func (n *Node) QEMUCPUModels(ctx context.Context, arch string) (models []*QEMUCPUModel, err error) {
	path := fmt.Sprintf("/nodes/%s/capabilities/qemu/cpu", n.Name)
	if arch != "" {
		q := url.Values{}
		q.Set("arch", arch)
		path = path + "?" + q.Encode()
	}
	err = n.client.Get(ctx, path, &models)
	return
}

func (n *Node) QEMUCPUFlags(ctx context.Context, arch, accel string) (flags []*QEMUCPUFlag, err error) {
	q := url.Values{}
	if arch != "" {
		q.Set("arch", arch)
	}
	if accel != "" {
		q.Set("accel", accel)
	}
	path := fmt.Sprintf("/nodes/%s/capabilities/qemu/cpu-flags", n.Name)
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	err = n.client.Get(ctx, path, &flags)
	return
}

func (n *Node) QEMUMachineTypes(ctx context.Context, arch string) (types []*QEMUMachineType, err error) {
	path := fmt.Sprintf("/nodes/%s/capabilities/qemu/machines", n.Name)
	if arch != "" {
		q := url.Values{}
		q.Set("arch", arch)
		path = path + "?" + q.Encode()
	}
	err = n.client.Get(ctx, path, &types)
	return
}

func (n *Node) QEMUMigrationCapabilities(ctx context.Context) (caps *QEMUMigrationCapabilities, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/capabilities/qemu/migration", n.Name), &caps)
	return
}

func (n *Node) capabilitiesDiridx(ctx context.Context, path string) ([]string, error) {

	var items []map[string]string
	if err := n.client.Get(ctx, path, &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if v, ok := it["name"]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}
