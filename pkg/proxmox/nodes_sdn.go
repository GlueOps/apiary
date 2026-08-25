package proxmox

import (
	"context"
	"fmt"
)

func (n *Node) SDNIndex(ctx context.Context) ([]string, error) {
	return n.sdnDiridx(ctx, fmt.Sprintf("/nodes/%s/sdn", n.Name))
}

func (n *Node) SDNFabricIndex(ctx context.Context, fabric string) ([]string, error) {
	if fabric == "" {
		return nil, fmt.Errorf("fabric is required")
	}
	return n.sdnDiridx(ctx, fmt.Sprintf("/nodes/%s/sdn/fabrics/%s", n.Name, fabric))
}

func (n *Node) SDNFabricInterfaces(ctx context.Context, fabric string) (ifaces []*SDNFabricInterface, err error) {
	if fabric == "" {
		return nil, fmt.Errorf("fabric is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/fabrics/%s/interfaces", n.Name, fabric), &ifaces)
	return
}

func (n *Node) SDNFabricNeighbors(ctx context.Context, fabric string) (neighbors []*SDNFabricNeighbor, err error) {
	if fabric == "" {
		return nil, fmt.Errorf("fabric is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/fabrics/%s/neighbors", n.Name, fabric), &neighbors)
	return
}

func (n *Node) SDNFabricRoutes(ctx context.Context, fabric string) (routes []*SDNFabricRoute, err error) {
	if fabric == "" {
		return nil, fmt.Errorf("fabric is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/fabrics/%s/routes", n.Name, fabric), &routes)
	return
}

func (n *Node) SDNVNetIndex(ctx context.Context, vnet string) ([]string, error) {
	if vnet == "" {
		return nil, fmt.Errorf("vnet is required")
	}
	return n.sdnDiridx(ctx, fmt.Sprintf("/nodes/%s/sdn/vnets/%s", n.Name, vnet))
}

func (n *Node) SDNVNetMACVRF(ctx context.Context, vnet string) (entries []*SDNMACVRFEntry, err error) {
	if vnet == "" {
		return nil, fmt.Errorf("vnet is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/vnets/%s/mac-vrf", n.Name, vnet), &entries)
	return
}

func (n *Node) SDNZones(ctx context.Context) (zones []*SDNZoneStatus, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/zones", n.Name), &zones)
	return
}

func (n *Node) SDNZoneIndex(ctx context.Context, zone string) ([]string, error) {
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	return n.sdnDiridx(ctx, fmt.Sprintf("/nodes/%s/sdn/zones/%s", n.Name, zone))
}

func (n *Node) SDNZoneBridges(ctx context.Context, zone string) (bridges []*SDNZoneBridge, err error) {
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/zones/%s/bridges", n.Name, zone), &bridges)
	return
}

func (n *Node) SDNZoneContent(ctx context.Context, zone string) (content []*SDNZoneContent, err error) {
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/zones/%s/content", n.Name, zone), &content)
	return
}

func (n *Node) SDNZoneIPVRF(ctx context.Context, zone string) (entries []*SDNIPVRFEntry, err error) {
	if zone == "" {
		return nil, fmt.Errorf("zone is required")
	}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/sdn/zones/%s/ip-vrf", n.Name, zone), &entries)
	return
}

func (n *Node) sdnDiridx(ctx context.Context, path string) ([]string, error) {
	var items []struct {
		Subdir string `json:"subdir"`
	}
	if err := n.client.Get(ctx, path, &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Subdir)
	}
	return out, nil
}
