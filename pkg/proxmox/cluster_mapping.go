package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) Mappings(ctx context.Context) (entries ClusterMappings, err error) {
	err = cl.client.Get(ctx, "/cluster/mapping", &entries)
	return
}

func (cl *Cluster) DirMappings(ctx context.Context, checkNode string) (mappings ClusterDirMappings, err error) {
	path := "/cluster/mapping/dir"
	if checkNode != "" {
		q := url.Values{}
		q.Set("check-node", checkNode)
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &mappings)
	return
}

func (cl *Cluster) DirMapping(ctx context.Context, id string) (m *ClusterDirMapping, err error) {
	if id == "" {
		err = errors.New("dir mapping id can not be empty")
		return
	}
	m = &ClusterDirMapping{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/mapping/dir/%s", id), m); err != nil {
		return
	}
	if m.ID == "" {
		m.ID = id
	}
	return
}

func (cl *Cluster) NewDirMapping(ctx context.Context, opts *ClusterDirMappingOptions) error {
	if opts == nil || opts.ID == "" {
		return errors.New("dir mapping id can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/mapping/dir", opts, nil)
}

func (cl *Cluster) UpdateDirMapping(ctx context.Context, id string, opts *ClusterDirMappingOptions) error {
	if id == "" {
		return errors.New("dir mapping id can not be empty")
	}
	if opts == nil {
		opts = &ClusterDirMappingOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/mapping/dir/%s", id), opts, nil)
}

func (cl *Cluster) DeleteDirMapping(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("dir mapping id can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/mapping/dir/%s", id), nil)
}

func (cl *Cluster) PCIMappings(ctx context.Context, checkNode string) (mappings ClusterPCIMappings, err error) {
	path := "/cluster/mapping/pci"
	if checkNode != "" {
		q := url.Values{}
		q.Set("check-node", checkNode)
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &mappings)
	return
}

func (cl *Cluster) PCIMapping(ctx context.Context, id string) (m *ClusterPCIMapping, err error) {
	if id == "" {
		err = errors.New("pci mapping id can not be empty")
		return
	}
	m = &ClusterPCIMapping{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/mapping/pci/%s", id), m); err != nil {
		return
	}
	if m.ID == "" {
		m.ID = id
	}
	return
}

func (cl *Cluster) NewPCIMapping(ctx context.Context, opts *ClusterPCIMappingOptions) error {
	if opts == nil || opts.ID == "" {
		return errors.New("pci mapping id can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/mapping/pci", opts, nil)
}

func (cl *Cluster) UpdatePCIMapping(ctx context.Context, id string, opts *ClusterPCIMappingOptions) error {
	if id == "" {
		return errors.New("pci mapping id can not be empty")
	}
	if opts == nil {
		opts = &ClusterPCIMappingOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/mapping/pci/%s", id), opts, nil)
}

func (cl *Cluster) DeletePCIMapping(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("pci mapping id can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/mapping/pci/%s", id), nil)
}

func (cl *Cluster) USBMappings(ctx context.Context, checkNode string) (mappings ClusterUSBMappings, err error) {
	path := "/cluster/mapping/usb"
	if checkNode != "" {
		q := url.Values{}
		q.Set("check-node", checkNode)
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &mappings)
	return
}

func (cl *Cluster) USBMapping(ctx context.Context, id string) (m *ClusterUSBMapping, err error) {
	if id == "" {
		err = errors.New("usb mapping id can not be empty")
		return
	}
	m = &ClusterUSBMapping{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/mapping/usb/%s", id), m); err != nil {
		return
	}
	if m.ID == "" {
		m.ID = id
	}
	return
}

func (cl *Cluster) NewUSBMapping(ctx context.Context, opts *ClusterUSBMappingOptions) error {
	if opts == nil || opts.ID == "" {
		return errors.New("usb mapping id can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/mapping/usb", opts, nil)
}

func (cl *Cluster) UpdateUSBMapping(ctx context.Context, id string, opts *ClusterUSBMappingOptions) error {
	if id == "" {
		return errors.New("usb mapping id can not be empty")
	}
	if opts == nil {
		opts = &ClusterUSBMappingOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/mapping/usb/%s", id), opts, nil)
}

func (cl *Cluster) DeleteUSBMapping(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("usb mapping id can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/mapping/usb/%s", id), nil)
}
