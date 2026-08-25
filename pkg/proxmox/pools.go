package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) NewPool(ctx context.Context, poolid, comment string) error {
	return c.Post(ctx, "/pools", map[string]string{
		"poolid":  poolid,
		"comment": comment,
	}, nil)
}

func (c *Client) Pools(ctx context.Context) (pools Pools, err error) {
	err = c.Get(ctx, "/pools", &pools)
	for _, pool := range pools {
		pool.client = c
	}
	return
}

func (c *Client) Pool(ctx context.Context, poolid string, filters ...string) (pool *Pool, err error) {
	u := url.URL{Path: "/pools/"}

	params := url.Values{}
	params.Add("poolid", poolid)

	if f := strings.ReplaceAll(strings.Join(filters, ""), " ", ""); f != "" {
		params.Add("type", f)
	}

	u.RawQuery = params.Encode()

	var pools Pools
	if err = c.Get(ctx, u.String(), &pools); err != nil {
		return nil, err
	}

	switch len(pools) {
	case 0:

		return nil, fmt.Errorf("pool not found")
	case 1:
		pool = pools[0]
	default:

		return nil, fmt.Errorf("multiple pools found for poolid: %s", poolid)
	}

	pool.client = c

	return
}

func (p *Pool) Update(ctx context.Context, opt *PoolUpdateOption) error {
	data, err := poolUpdatePayload(p.PoolID, opt)
	if err != nil {
		return err
	}
	return p.client.Put(ctx, "/pools", data, nil)
}

func (p *Pool) Delete(ctx context.Context) error {
	u := url.URL{Path: "/pools"}
	q := url.Values{}
	q.Set("poolid", p.PoolID)
	u.RawQuery = q.Encode()
	return p.client.Delete(ctx, u.String(), nil)
}

func poolUpdatePayload(poolid string, opt *PoolUpdateOption) (map[string]interface{}, error) {
	data := map[string]interface{}{"poolid": poolid}
	if opt == nil {
		return data, nil
	}
	raw, err := json.Marshal(opt)
	if err != nil {
		return nil, err
	}
	var fields map[string]interface{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for k, v := range fields {
		data[k] = v
	}
	return data, nil
}

func (p *Pool) GetDeprecated(ctx context.Context, filters ...string) (*Pool, error) {
	u := url.URL{Path: fmt.Sprintf("/pools/%s", p.PoolID)}
	if f := strings.ReplaceAll(strings.Join(filters, ""), " ", ""); f != "" {
		q := url.Values{}
		q.Set("type", f)
		u.RawQuery = q.Encode()
	}

	pool := &Pool{client: p.client, PoolID: p.PoolID}
	if err := p.client.Get(ctx, u.String(), pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (p *Pool) UpdateDeprecated(ctx context.Context, opt *PoolUpdateOption) error {
	return p.client.Put(ctx, fmt.Sprintf("/pools/%s", p.PoolID), opt, nil)
}

func (p *Pool) DeleteDeprecated(ctx context.Context) error {
	return p.client.Delete(ctx, fmt.Sprintf("/pools/%s", p.PoolID), nil)
}
