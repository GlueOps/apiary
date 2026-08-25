package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

func (n *Node) APT(ctx context.Context) (entries []*APTIndexEntry, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/apt", n.Name), &entries)
	return
}

func (n *Node) APTUpdates(ctx context.Context) (updates []*APTUpdate, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/apt/update", n.Name), &updates)
	return
}

func (n *Node) APTUpdate(ctx context.Context, notify, quiet bool) (*Task, error) {
	body := map[string]interface{}{}
	if notify {
		body["notify"] = 1
	}
	if quiet {
		body["quiet"] = 1
	}
	var upid UPID
	if err := n.client.Post(ctx, fmt.Sprintf("/nodes/%s/apt/update", n.Name), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, n.client), nil
}

func (n *Node) APTChangelog(ctx context.Context, name, version string) (changelog string, err error) {
	q := url.Values{}
	q.Set("name", name)
	if version != "" {
		q.Set("version", version)
	}
	path := fmt.Sprintf("/nodes/%s/apt/changelog?%s", n.Name, q.Encode())
	err = n.client.Get(ctx, path, &changelog)
	return
}

func (n *Node) APTRepositories(ctx context.Context) (repos *APTRepositories, err error) {
	repos = &APTRepositories{}
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/apt/repositories", n.Name), repos)
	return
}

func (n *Node) APTChangeRepository(ctx context.Context, path string, index int, enabled bool, digest string) error {
	body := map[string]interface{}{
		"path":    path,
		"index":   index,
		"enabled": 0,
	}
	if enabled {
		body["enabled"] = 1
	}
	if digest != "" {
		body["digest"] = digest
	}
	return n.client.Post(ctx, fmt.Sprintf("/nodes/%s/apt/repositories", n.Name), body, nil)
}

func (n *Node) APTAddRepository(ctx context.Context, handle, digest string) error {
	body := map[string]interface{}{"handle": handle}
	if digest != "" {
		body["digest"] = digest
	}
	return n.client.Put(ctx, fmt.Sprintf("/nodes/%s/apt/repositories", n.Name), body, nil)
}

func (n *Node) APTVersions(ctx context.Context) (versions []*APTPackageVersion, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/apt/versions", n.Name), &versions)
	return
}
