package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func (cl *Cluster) Log(ctx context.Context, max int) (entries []*ClusterLogEntry, err error) {
	path := "/cluster/log"
	if max > 0 {
		q := url.Values{}
		q.Set("max", strconv.Itoa(max))
		path = path + "?" + q.Encode()
	}
	err = cl.client.Get(ctx, path, &entries)
	return
}

func (cl *Cluster) NotificationEndpointsSubdirs(ctx context.Context) ([]string, error) {
	return cl.notificationsDiridx(ctx, "/cluster/notifications/endpoints")
}

func (g *FirewallSecurityGroup) GetRule(ctx context.Context, pos int) (rule *FirewallRule, err error) {
	if g.Group == "" {
		return nil, errors.New("firewall security group: name is required")
	}
	rule = &FirewallRule{}
	err = g.client.Get(ctx, fmt.Sprintf("/cluster/firewall/groups/%s/%d", g.Group, pos), rule)
	return
}
