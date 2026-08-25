package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (cl *Cluster) HAGroups(ctx context.Context) (groups []*HAGroup, err error) {
	err = cl.client.Get(ctx, "/cluster/ha/groups", &groups)
	return
}

func (cl *Cluster) HAGroup(ctx context.Context, name string) (group *HAGroup, err error) {
	group = &HAGroup{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/ha/groups/%s", name), group)
	return
}

func (cl *Cluster) NewHAGroup(ctx context.Context, opts *HAGroupCreateOption) error {
	return cl.client.Post(ctx, "/cluster/ha/groups", opts, nil)
}

func (cl *Cluster) HAGroupUpdate(ctx context.Context, name string, opts *HAGroupUpdateOption) error {
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/ha/groups/%s", name), opts, nil)
}

func (cl *Cluster) HAGroupDelete(ctx context.Context, name string) error {
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/ha/groups/%s", name), nil)
}

func (cl *Cluster) HAResources(ctx context.Context, typ string) (resources []*HAResource, err error) {
	path := "/cluster/ha/resources"
	if typ != "" {
		path += "?type=" + typ
	}
	err = cl.client.Get(ctx, path, &resources)
	return
}

func (cl *Cluster) HAResource(ctx context.Context, sid string) (resource *HAResource, err error) {
	resource = &HAResource{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/ha/resources/%s", sid), resource)
	return
}

func (cl *Cluster) NewHAResource(ctx context.Context, opts *HAResourceCreateOption) error {
	return cl.client.Post(ctx, "/cluster/ha/resources", opts, nil)
}

func (cl *Cluster) HAResourceUpdate(ctx context.Context, sid string, opts *HAResourceUpdateOption) error {
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/ha/resources/%s", sid), opts, nil)
}

func (cl *Cluster) HAResourceDelete(ctx context.Context, sid string, purge bool) error {
	path := fmt.Sprintf("/cluster/ha/resources/%s", sid)
	if purge {
		path += "?purge=1"
	}
	return cl.client.Delete(ctx, path, nil)
}

func (cl *Cluster) HAResourceMigrate(ctx context.Context, sid, node string) error {
	return cl.client.Post(ctx, fmt.Sprintf("/cluster/ha/resources/%s/migrate", sid), map[string]string{"node": node}, nil)
}

func (cl *Cluster) HAResourceRelocate(ctx context.Context, sid, node string) error {
	return cl.client.Post(ctx, fmt.Sprintf("/cluster/ha/resources/%s/relocate", sid), map[string]string{"node": node}, nil)
}

func (cl *Cluster) HARules(ctx context.Context, resource, typ string) (rules []*HARule, err error) {
	path := "/cluster/ha/rules"
	query := ""
	if resource != "" {
		query += "resource=" + resource
	}
	if typ != "" {
		if query != "" {
			query += "&"
		}
		query += "type=" + typ
	}
	if query != "" {
		path += "?" + query
	}
	err = cl.client.Get(ctx, path, &rules)
	return
}

func (cl *Cluster) HARule(ctx context.Context, name string) (rule *HARule, err error) {
	rule = &HARule{}
	err = cl.client.Get(ctx, fmt.Sprintf("/cluster/ha/rules/%s", name), rule)
	return
}

func (cl *Cluster) NewHARule(ctx context.Context, opts *HARuleCreateOption) error {
	return cl.client.Post(ctx, "/cluster/ha/rules", opts, nil)
}

func (cl *Cluster) HARuleUpdate(ctx context.Context, name string, opts *HARuleUpdateOption) error {
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/ha/rules/%s", name), opts, nil)
}

func (cl *Cluster) HARuleDelete(ctx context.Context, name string) error {
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/ha/rules/%s", name), nil)
}

func (cl *Cluster) HAStatus(ctx context.Context) (status []*HAStatusEntry, err error) {
	err = cl.client.Get(ctx, "/cluster/ha/status/current", &status)
	return
}

func (cl *Cluster) HAManagerStatus(ctx context.Context) (status *HAManagerStatus, err error) {
	status = &HAManagerStatus{}
	err = cl.client.Get(ctx, "/cluster/ha/status/manager_status", status)
	return
}

func (cl *Cluster) HAArm(ctx context.Context) error {
	return cl.client.Post(ctx, "/cluster/ha/status/arm-ha", nil, nil)
}

func (cl *Cluster) HADisarm(ctx context.Context, resourceMode string) error {
	if resourceMode == "" {
		return errors.New("ha disarm: resource-mode is required (\"freeze\" or \"ignore\")")
	}
	return cl.client.Post(ctx, "/cluster/ha/status/disarm-ha", map[string]string{"resource-mode": resourceMode}, nil)
}
