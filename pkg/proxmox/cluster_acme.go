package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

func (cl *Cluster) ACMEDirectories(ctx context.Context) (dirs []*ACMEDirectory, err error) {
	err = cl.client.Get(ctx, "/cluster/acme/directories", &dirs)
	return
}

func (cl *Cluster) ACMEChallengeSchema(ctx context.Context) (schemas []*ACMEChallengeSchema, err error) {
	err = cl.client.Get(ctx, "/cluster/acme/challenge-schema", &schemas)
	return
}

func (cl *Cluster) ACMETermsOfService(ctx context.Context, directory string) (tosURL string, err error) {
	path := "/cluster/acme/tos"
	if directory != "" {
		q := url.Values{}
		q.Set("directory", directory)
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &tosURL)
	return
}

func (cl *Cluster) ACMEMeta(ctx context.Context, directory string) (meta *ACMEMeta, err error) {
	path := "/cluster/acme/meta"
	if directory != "" {
		q := url.Values{}
		q.Set("directory", directory)
		path = path + "?" + q.Encode()
	}
	meta = &ACMEMeta{}
	err = cl.client.Get(ctx, path, meta)
	return
}

func (cl *Cluster) ACMEAccounts(ctx context.Context) (accounts []*ACMEAccountIndex, err error) {
	err = cl.client.Get(ctx, "/cluster/acme/account", &accounts)
	return
}

func (cl *Cluster) ACMEAccount(ctx context.Context, name string) (account *ACMEAccount, err error) {
	if name == "" {
		name = "default"
	}
	account = &ACMEAccount{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/acme/account/%s", name), account)
	return
}

func (cl *Cluster) NewACMEAccount(ctx context.Context, opts *ACMEAccountOptions) (*Task, error) {
	if opts == nil || opts.Contact == "" {
		return nil, errors.New("acme account contact is required")
	}
	var upid UPID
	if err := cl.client.Post(ctx, "/cluster/acme/account", opts, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) UpdateACMEAccount(ctx context.Context, name, contact string) (*Task, error) {
	if name == "" {
		name = "default"
	}
	body := map[string]any{}
	if contact != "" {
		body["contact"] = contact
	}
	var upid UPID
	if err := cl.client.Put(ctx, fmt.Sprintf("/cluster/acme/account/%s", name), body, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) DeleteACMEAccount(ctx context.Context, name string) (*Task, error) {
	if name == "" {
		name = "default"
	}
	var upid UPID
	if err := cl.client.Delete(ctx, fmt.Sprintf("/cluster/acme/account/%s", name), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, cl.client), nil
}

func (cl *Cluster) ACMEPlugins(ctx context.Context, pluginType string) (plugins []*ACMEPlugin, err error) {
	path := "/cluster/acme/plugins"
	if pluginType != "" {
		q := url.Values{}
		q.Set("type", pluginType)
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &plugins)
	return
}

func (cl *Cluster) ACMEPlugin(ctx context.Context, id string) (plugin *ACMEPlugin, err error) {
	if id == "" {
		err = errors.New("acme plugin id can not be empty")
		return
	}
	plugin = &ACMEPlugin{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/acme/plugins/%s", id), plugin)
	return
}

func (cl *Cluster) NewACMEPlugin(ctx context.Context, opts *ACMEPluginOptions) error {
	if opts == nil || opts.ID == "" {
		return errors.New("acme plugin id can not be empty")
	}
	if opts.Type == "" {
		return errors.New("acme plugin type is required")
	}
	return cl.client.Post(ctx, "/cluster/acme/plugins", opts, nil)
}

func (cl *Cluster) UpdateACMEPlugin(ctx context.Context, id string, opts *ACMEPluginOptions) error {
	if id == "" {
		return errors.New("acme plugin id can not be empty")
	}
	if opts == nil {
		opts = &ACMEPluginOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/acme/plugins/%s", id), opts, nil)
}

func (cl *Cluster) DeleteACMEPlugin(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("acme plugin id can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/acme/plugins/%s", id), nil)
}
