package proxmox

import (
	"context"
	"errors"
	"net/url"
)

func (cl *Cluster) SDNIndex(ctx context.Context) (entries []map[string]any, err error) {
	err = cl.client.Get(ctx, "/cluster/sdn", &entries)
	return
}

func (cl *Cluster) SDNLock(ctx context.Context, allowPending bool) (SDNLockToken, error) {
	body := map[string]any{}
	if allowPending {
		body["allow-pending"] = 1
	}
	var token string
	if err := cl.client.Post(ctx, "/cluster/sdn/lock", body, &token); err != nil {
		return "", err
	}
	return SDNLockToken(token), nil
}

func (cl *Cluster) SDNReleaseLock(ctx context.Context, token SDNLockToken, force bool) error {
	path := "/cluster/sdn/lock"
	q := url.Values{}
	if token != "" {
		q.Set("lock-token", string(token))
	}
	if force {
		q.Set("force", "1")
	}
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}
	return cl.client.Delete(ctx, path, nil)
}

func (cl *Cluster) SDNRollback(ctx context.Context, token SDNLockToken, releaseLock bool) error {
	body := map[string]any{}
	if token != "" {
		body["lock-token"] = string(token)
	}
	if !releaseLock {

		body["release-lock"] = 0
	}
	return cl.client.Post(ctx, "/cluster/sdn/rollback", body, nil)
}

func (cl *Cluster) SDNDryRun(ctx context.Context, node string) (*SDNDryRun, error) {
	if node == "" {
		return nil, errors.New("node is required")
	}
	q := url.Values{}
	q.Set("node", node)
	out := &SDNDryRun{}
	if err := cl.client.Get(ctx, "/cluster/sdn/dry-run?"+q.Encode(), out); err != nil {
		return nil, err
	}
	return out, nil
}
