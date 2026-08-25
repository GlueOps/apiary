package proxmox

import (
	"context"
	"fmt"
)

func (n *Node) DNS(ctx context.Context) (*NodeDNS, error) {
	var dns NodeDNS
	if err := n.client.Get(ctx, fmt.Sprintf("/nodes/%s/dns", n.Name), &dns); err != nil {
		return nil, err
	}
	return &dns, nil
}

func (n *Node) UpdateDNS(ctx context.Context, dns *NodeDNS) error {
	return n.client.Put(ctx, fmt.Sprintf("/nodes/%s/dns", n.Name), dns, nil)
}
