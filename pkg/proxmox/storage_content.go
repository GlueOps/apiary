package proxmox

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
)

type StorageContentAllocOptions struct {
	Filename string `json:"filename"`
	Size     string `json:"size"`
	VMID     uint64 `json:"vmid"`
	Format   string `json:"format,omitempty"`
}

func (s *Storage) AllocContent(ctx context.Context, opts *StorageContentAllocOptions) (volid string, err error) {
	if opts == nil || opts.Filename == "" || opts.Size == "" || opts.VMID == 0 {
		return "", errors.New("filename, size, and vmid are required")
	}
	err = s.client.Post(ctx, fmt.Sprintf("/nodes/%s/storage/%s/content", s.Node, s.Name), opts, &volid)
	return
}

type StorageContentUpdateOptions struct {
	Notes     string `json:"notes,omitempty"`
	Protected *bool  `json:"protected,omitempty"`
}

func (s *Storage) UpdateContent(ctx context.Context, volume string, opts *StorageContentUpdateOptions) error {
	if volume == "" {
		return errors.New("volume is required")
	}
	if opts == nil {
		opts = &StorageContentUpdateOptions{}
	}
	return s.client.Put(ctx, fmt.Sprintf("/nodes/%s/storage/%s/content/%s", s.Node, s.Name, volume), opts, nil)
}

func (s *Storage) CopyContent(ctx context.Context, sourceVolume, targetVolume, targetNode string) (*Task, error) {
	if sourceVolume == "" || targetVolume == "" {
		return nil, errors.New("source and target volumes are required")
	}
	body := map[string]string{"target": targetVolume}
	if targetNode != "" {
		body["target_node"] = targetNode
	}
	var upid UPID
	if err := s.client.Post(ctx, fmt.Sprintf("/nodes/%s/storage/%s/content/%s", s.Node, s.Name, sourceVolume), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

func (s *Storage) OCIRegistryPull(ctx context.Context, reference, filename string) (*Task, error) {
	if reference == "" {
		return nil, errors.New("oci reference is required")
	}
	body := map[string]string{"reference": reference}
	if filename != "" {
		body["filename"] = filename
	}
	var upid UPID
	if err := s.client.Post(ctx, fmt.Sprintf("/nodes/%s/storage/%s/oci-registry-pull", s.Node, s.Name), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

type StorageFileRestoreEntry struct {
	Filepath string    `json:"filepath,omitempty"`
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text,omitempty"`
	Size     uint64    `json:"size,omitempty"`
	Mtime    int64     `json:"mtime,omitempty"`
	Leaf     IntOrBool `json:"leaf,omitempty"`
}

func (s *Storage) FileRestoreList(ctx context.Context, volume, filepath string) (entries []*StorageFileRestoreEntry, err error) {
	if volume == "" || filepath == "" {
		return nil, errors.New("volume and filepath are required")
	}

	q := url.Values{}
	q.Set("volume", volume)
	q.Set("filepath", base64.StdEncoding.EncodeToString([]byte(filepath)))
	err = s.client.Get(ctx, fmt.Sprintf("/nodes/%s/storage/%s/file-restore/list?%s", s.Node, s.Name, q.Encode()), &entries)
	return
}

func (s *Storage) RRDData(ctx context.Context, timeframe Timeframe, cf ConsolidationFunction) (data []*RRDData, err error) {
	q := url.Values{}
	q.Set("timeframe", string(timeframe))
	if cf != "" {
		q.Set("cf", string(cf))
	}
	err = s.client.Get(ctx, fmt.Sprintf("/nodes/%s/storage/%s/rrddata?%s", s.Node, s.Name, q.Encode()), &data)
	return
}
