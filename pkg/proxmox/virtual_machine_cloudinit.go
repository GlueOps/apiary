package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

type VirtualMachineCloudInitPending struct {
	Delete  IntOrBool `json:"delete,omitempty"`
	Key     string    `json:"key,omitempty"`
	Pending string    `json:"pending,omitempty"`
	Value   string    `json:"value,omitempty"`
}

func (v *VirtualMachine) CloudInitPending(ctx context.Context) (pending []*VirtualMachineCloudInitPending, err error) {
	return pending, v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit", v.Node, v.VMID), &pending)
}

func (v *VirtualMachine) CloudInitRegenerate(ctx context.Context) error {
	return v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) CloudInitDump(ctx context.Context, kind string) (string, error) {
	if kind == "" {
		return "", errors.New("cloudinit dump type is required (user|meta|network)")
	}
	q := url.Values{}
	q.Set("type", kind)
	var out string
	err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit/dump?%s", v.Node, v.VMID, q.Encode()), &out)
	return out, err
}
