package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) QEMUCPUFlags(ctx context.Context, arch, accel string) (flags []*QEMUCPUFlag, err error) {
	path := "/cluster/qemu/cpu-flags"
	q := url.Values{}
	if arch != "" {
		q.Set("arch", arch)
	}
	if accel != "" {
		q.Set("accel", accel)
	}
	if enc := q.Encode(); enc != "" {
		path = path + "?" + enc
	}
	err = cl.client.Get(ctx, path, &flags)
	return
}

func (cl *Cluster) CustomCPUModels(ctx context.Context) (models []*CustomCPUModel, err error) {
	if err = cl.client.Get(ctx, "/cluster/qemu/custom-cpu-models", &models); err != nil {
		return nil, err
	}
	for _, m := range models {
		m.client = cl.client
	}
	return
}

func (cl *Cluster) CustomCPUModel(cputype string) *CustomCPUModel {
	return &CustomCPUModel{client: cl.client, CPUType: cputype}
}

func (cl *Cluster) NewCustomCPUModel(ctx context.Context, opts *CustomCPUModelOptions) error {
	if opts == nil || opts.CPUType == "" {
		return errors.New("custom cpu model: cputype is required")
	}
	if opts.ReportedModel == "" {
		return errors.New("custom cpu model: reported-model is required")
	}
	return cl.client.Post(ctx, "/cluster/qemu/custom-cpu-models", opts, nil)
}

func (m *CustomCPUModel) Read(ctx context.Context) error {
	if m.CPUType == "" {
		return errors.New("custom cpu model: cputype is required")
	}
	return m.client.Get(ctx, fmt.Sprintf("/cluster/qemu/custom-cpu-models/%s", m.CPUType), m)
}

func (m *CustomCPUModel) Update(ctx context.Context, opts *CustomCPUModelOptions) error {
	if m.CPUType == "" {
		return errors.New("custom cpu model: cputype is required")
	}
	if opts == nil {
		opts = &CustomCPUModelOptions{}
	}
	return m.client.Put(ctx, fmt.Sprintf("/cluster/qemu/custom-cpu-models/%s", m.CPUType), opts, nil)
}

func (m *CustomCPUModel) Delete(ctx context.Context) error {
	if m.CPUType == "" {
		return errors.New("custom cpu model: cputype is required")
	}
	return m.client.Delete(ctx, fmt.Sprintf("/cluster/qemu/custom-cpu-models/%s", m.CPUType), nil)
}
