package proxmox

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
)

func (v *VirtualMachine) AgentCommandIndex(ctx context.Context) ([]*AgentCommandIndexEntry, error) {
	var out []*AgentCommandIndexEntry
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent", v.Node, v.VMID), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (v *VirtualMachine) AgentCommand(ctx context.Context, command string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"command": command,
	}
	var resp struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent", v.Node, v.VMID), body, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentPing(ctx context.Context) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/ping", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) AgentGetTime(ctx context.Context) (AgentTime, error) {
	var resp struct {
		Result AgentTime `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-time", v.Node, v.VMID), &resp); err != nil {
		return 0, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetTimezone(ctx context.Context) (string, error) {
	var resp struct {
		Result struct {
			Zone   string `json:"zone"`
			Offset int    `json:"offset"`
		} `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-timezone", v.Node, v.VMID), &resp); err != nil {
		return "", err
	}
	return resp.Result.Zone, nil
}

func (v *VirtualMachine) AgentGetUsers(ctx context.Context) ([]*AgentUser, error) {
	var resp struct {
		Result []*AgentUser `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-users", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetVCPUs(ctx context.Context) ([]*AgentVCPU, error) {
	var resp struct {
		Result []*AgentVCPU `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-vcpus", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetFsInfo(ctx context.Context) ([]*AgentFsInfo, error) {
	var resp struct {
		Result []*AgentFsInfo `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-fsinfo", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetMemoryBlocks(ctx context.Context) ([]*AgentMemoryBlock, error) {
	var resp struct {
		Result []*AgentMemoryBlock `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-memory-blocks", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetMemoryBlockInfo(ctx context.Context) (*AgentMemoryBlockInfo, error) {
	var resp struct {
		Result *AgentMemoryBlockInfo `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-memory-block-info", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentGetInfo(ctx context.Context) (*AgentInfo, error) {
	var resp struct {
		Result *AgentInfo `json:"result"`
	}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/info", v.Node, v.VMID), &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentFsfreezeFreeze(ctx context.Context) (int, error) {
	var resp struct {
		Result int `json:"result"`
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/fsfreeze-freeze", v.Node, v.VMID), nil, &resp); err != nil {
		return 0, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentFsfreezeThaw(ctx context.Context) (int, error) {
	var resp struct {
		Result int `json:"result"`
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/fsfreeze-thaw", v.Node, v.VMID), nil, &resp); err != nil {
		return 0, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentFsfreezeStatus(ctx context.Context) (AgentFsfreezeStatus, error) {
	var resp struct {
		Result AgentFsfreezeStatus `json:"result"`
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/fsfreeze-status", v.Node, v.VMID), nil, &resp); err != nil {
		return "", err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentFstrim(ctx context.Context) (map[string]interface{}, error) {
	var resp struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/fstrim", v.Node, v.VMID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (v *VirtualMachine) AgentShutdown(ctx context.Context) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/shutdown", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) AgentSuspendDisk(ctx context.Context) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/suspend-disk", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) AgentSuspendHybrid(ctx context.Context) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/suspend-hybrid", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) AgentSuspendRAM(ctx context.Context) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/suspend-ram", v.Node, v.VMID), nil, nil)
}

func (v *VirtualMachine) AgentFileRead(ctx context.Context, file string) (*AgentFileRead, error) {

	q := url.Values{}
	q.Set("file", file)
	var out AgentFileRead
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/file-read?%s", v.Node, v.VMID, q.Encode()), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (v *VirtualMachine) AgentFileWrite(ctx context.Context, file string, content []byte) error {
	body := map[string]interface{}{
		"file":    file,
		"content": base64.StdEncoding.EncodeToString(content),
		"encode":  0,
	}
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/file-write", v.Node, v.VMID), body, nil)
}
