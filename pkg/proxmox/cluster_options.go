package proxmox

import (
	"context"
	"encoding/json"
	"reflect"
)

func (cl *Cluster) ClusterOptions(ctx context.Context) (opts *ClusterOptionsResponse, err error) {
	opts = &ClusterOptionsResponse{}
	err = cl.client.Get(ctx, "/cluster/options", opts)
	return
}

func (cl *Cluster) UpdateClusterOptions(ctx context.Context, opts *ClusterOptionsUpdate) error {
	if opts == nil {
		opts = &ClusterOptionsUpdate{}
	}
	return cl.client.Put(ctx, "/cluster/options", opts, nil)
}

func (r *ClusterOptionsResponse) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	type alias ClusterOptionsResponse
	scalar, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	tmp := alias{}
	if err := json.Unmarshal(scalar, &tmp); err != nil {
		return err
	}
	*r = ClusterOptionsResponse(tmp)

	typed := map[string]struct{}{}
	t := reflect.TypeOf(*r)
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := tag
		if c := indexComma(tag); c >= 0 {
			name = tag[:c]
		}
		typed[name] = struct{}{}
	}
	extra := map[string]any{}
	for k, v := range raw {
		if _, ok := typed[k]; ok {
			continue
		}
		var any any
		if err := json.Unmarshal(v, &any); err != nil {
			return err
		}
		extra[k] = any
	}
	if len(extra) > 0 {
		r.Extra = extra
	}
	return nil
}

func (u ClusterOptionsUpdate) MarshalJSON() ([]byte, error) {
	type alias ClusterOptionsUpdate
	scalar, err := json.Marshal(alias(u))
	if err != nil {
		return nil, err
	}
	if len(u.Extra) == 0 {
		return scalar, nil
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(scalar, &m); err != nil {
		return nil, err
	}
	for k, v := range u.Extra {
		if _, exists := m[k]; exists {
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = raw
	}
	return json.Marshal(m)
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}
