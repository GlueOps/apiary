package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

func (c *Container) Clone(ctx context.Context, params *ContainerCloneOptions) (newid int, task *Task, err error) {
	var upid UPID

	if params == nil {
		params = &ContainerCloneOptions{}
	}
	if params.NewID <= 0 {
		cluster, err := c.client.Cluster(ctx)
		if err != nil {
			return newid, nil, err
		}
		newid, err = cluster.NextID(ctx)
		if err != nil {
			return newid, nil, err
		}
		params.NewID = newid
	} else {
		newid = params.NewID
	}
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/clone", c.Node, c.VMID), params, &upid); err != nil {
		return 0, nil, err
	}
	return newid, NewTask(upid, c.client), nil
}

func (c *Container) Delete(ctx context.Context, params *ContainerDeleteOptions) (task *Task, err error) {
	var upid UPID
	if err := c.client.DeleteWithParams(ctx, fmt.Sprintf("/nodes/%s/lxc/%d", c.Node, c.VMID), params, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, c.client), nil
}

func (c *Container) Ping(ctx context.Context) error {
	return c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/current", c.Node, c.VMID), &c)
}

func (c *Container) Config(ctx context.Context, options ...ContainerOption) (*Task, error) {
	var upid UPID
	data := make(map[string]interface{})
	for _, option := range options {
		data[option.Name] = option.Value
	}
	err := c.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/config", c.Node, c.VMID), data, &upid)
	return NewTask(upid, c.client), err
}

func (c *Container) Start(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/start", c.Node, c.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Stop(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/stop", c.Node, c.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Suspend(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/suspend", c.Node, c.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Reboot(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/reboot", c.Node, c.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Resume(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/resume", c.Node, c.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Shutdown(ctx context.Context, force bool, timeout int) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/shutdown", c.Node, c.VMID), map[string]interface{}{"forceStop": force, "timeout": timeout}, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) TermProxy(ctx context.Context) (term *Term, err error) {
	return term, c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/termproxy", c.Node, c.VMID), nil, &term)
}

func (c *Container) TermWebSocket(term *Term) (chan []byte, chan []byte, chan error, func() error, error) {
	p := fmt.Sprintf("/nodes/%s/lxc/%d/vncwebsocket?port=%d&vncticket=%s",
		c.Node, c.VMID, term.Port, url.QueryEscape(term.Ticket))

	return c.client.TermWebSocket(p, term)
}

func (c *Container) VNCWebSocket(vnc *VNC) (chan []byte, chan []byte, chan error, func() error, error) {
	p := fmt.Sprintf("/nodes/%s/lxc/%d/vncwebsocket?port=%d&vncticket=%s",
		c.Node, c.VMID, vnc.Port, url.QueryEscape(vnc.Ticket))

	return c.client.VNCWebSocket(p, vnc)
}

func (c *Container) Feature(ctx context.Context) (hasFeature bool, err error) {
	var feature struct {
		HasFeature bool `json:"hasFeature"`
	}
	err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/feature", c.Node, c.VMID), &feature)
	return feature.HasFeature, err
}

func (c *Container) Interfaces(ctx context.Context) (interfaces ContainerInterfaces, err error) {
	err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/interfaces", c.Node, c.VMID), &interfaces)
	return interfaces, err
}

func (c *Container) Migrate(ctx context.Context, params *ContainerMigrateOptions) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/migrate", c.Node, c.VMID), params, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Resize(ctx context.Context, disk, size string) (task *Task, err error) {
	var upid UPID
	if err := c.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/resize", c.Node, c.VMID), map[string]interface{}{"disk": disk, "size": size}, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) MoveVolume(ctx context.Context, params *VirtualMachineMoveDiskOptions) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/move_volume", c.Node, c.VMID), params, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) RRDData(ctx context.Context, timeframe Timeframe, consolidationFunction ...ConsolidationFunction) (rrddata []*RRDData, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/lxc/%d/rrddata", c.Node, c.VMID)}

	params := url.Values{}
	if len(consolidationFunction) > 0 {
		if len(consolidationFunction) != 1 {
			return nil, fmt.Errorf("only one consolidation function allowed")
		}

		params.Add("cf", string(consolidationFunction[0]))
	}

	params.Add("timeframe", string(timeframe))
	u.RawQuery = params.Encode()

	err = c.client.Get(ctx, u.String(), &rrddata)
	return
}

func (c *Container) Template(ctx context.Context) error {
	return c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/template", c.Node, c.VMID), nil, nil)
}

func (c *Container) VNCProxy(ctx context.Context, vncOptions VNCProxyOptions) (vnc *VNC, err error) {
	return vnc, c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/vncproxy", c.Node, c.VMID), vncOptions, &vnc)
}

func (c *Container) Snapshots(ctx context.Context) (snapshots []*ContainerSnapshot, err error) {
	if err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot", c.Node, c.VMID), &snapshots); err != nil {
		return nil, err
	}
	for _, s := range snapshots {
		s.client = c.client
		s.Node = c.Node
		s.VMID = clampToInt(uint64(c.VMID))
	}
	return snapshots, nil
}

func (c *Container) NewSnapshot(ctx context.Context, snapName string) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot", c.Node, c.VMID), map[string]interface{}{"snapname": snapName}, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) Snapshot(name string) *ContainerSnapshot {
	return &ContainerSnapshot{
		client: c.client,
		Node:   c.Node,
		VMID:   clampToInt(uint64(c.VMID)),
		Name:   name,
	}
}

func (s *ContainerSnapshot) Rollback(ctx context.Context, start bool) (task *Task, err error) {
	var upid UPID
	if err := s.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot/%s/rollback", s.Node, s.VMID, s.Name), map[string]interface{}{"start": start}, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

func (s *ContainerSnapshot) Delete(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := s.client.Delete(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot/%s", s.Node, s.VMID, s.Name), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

func (s *ContainerSnapshot) Config(ctx context.Context) (config map[string]interface{}, err error) {
	return config, s.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot/%s/config", s.Node, s.VMID, s.Name), &config)
}

func (s *ContainerSnapshot) UpdateConfig(ctx context.Context, options *ContainerSnapshotUpdateOptions) error {
	if options == nil {
		options = &ContainerSnapshotUpdateOptions{}
	}
	return s.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot/%s/config", s.Node, s.VMID, s.Name), options, nil)
}

func (s *ContainerSnapshot) SubResources(ctx context.Context) (entries []*ContainerSnapshotIndexEntry, err error) {
	err = s.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/snapshot/%s", s.Node, s.VMID, s.Name), &entries)
	return
}

func (c *Container) Firewall(ctx context.Context) (firewall *Firewall, err error) {
	return firewall, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall", c.Node, c.VMID), &firewall)
}

func (c *Container) GetFirewallAliases(ctx context.Context) (aliases []*FirewallAlias, err error) {
	return aliases, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases", c.Node, c.VMID), &aliases)
}

func (c *Container) NewFirewallAlias(ctx context.Context, alias *FirewallAlias) error {
	return c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases", c.Node, c.VMID), alias, nil)
}

func (c *Container) GetFirewallAlias(ctx context.Context, name string) (alias *FirewallAlias, err error) {
	return alias, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases/%s", c.Node, c.VMID, name), &alias)
}

func (c *Container) UpdateFirewallAlias(ctx context.Context, name string, alias *FirewallAlias) error {
	return c.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases/%s", c.Node, c.VMID, name), alias, nil)
}

func (c *Container) DeleteFirewallAlias(ctx context.Context, name string) error {
	return c.client.Delete(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/aliases/%s", c.Node, c.VMID, name), nil)
}

func (c *Container) GetFirewallIPSet(ctx context.Context) (ipsets []*FirewallIPSet, err error) {
	return ipsets, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset", c.Node, c.VMID), &ipsets)
}

func (c *Container) NewFirewallIPSet(ctx context.Context, ipset FirewallIPSetCreationOption) error {
	return c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset", c.Node, c.VMID), ipset, nil)
}

func (c *Container) DeleteFirewallIPSet(ctx context.Context, name string, force bool) error {
	return c.client.Delete(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s", c.Node, c.VMID, name), map[string]interface{}{"force": force})
}

func (c *Container) GetFirewallIPSetEntries(ctx context.Context, name string) (entries []*FirewallIPSetEntry, err error) {
	return entries, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s", c.Node, c.VMID, name), &entries)
}

func (c *Container) NewFirewallIPSetEntry(ctx context.Context, name string, entry FirewallIPSetEntryCreationOption) error {
	return c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s", c.Node, c.VMID, name), entry, nil)
}

func (c *Container) DeleteFirewallIPSetEntry(ctx context.Context, name string, cidr string, digest string) error {
	return c.client.Delete(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s/%s", c.Node, c.VMID, name, cidr), map[string]interface{}{
		"digest": digest,
	})
}

func (c *Container) GetFirewallIPSetEntry(ctx context.Context, name string, cidr string) (entry *FirewallIPSetEntry, err error) {
	err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s/%s", c.Node, c.VMID, name, cidr), &entry)
	return
}

func (c *Container) UpdateFirewallIPSetEntry(ctx context.Context, name string, cidr string, entry *FirewallIPSetEntryUpdateOption) error {
	return c.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/ipset/%s/%s", c.Node, c.VMID, name, cidr), entry, nil)
}

func (c *Container) FirewallRules(ctx context.Context) (rules []*FirewallRule, err error) {
	if err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules", c.Node, c.VMID), &rules); err != nil {
		return nil, err
	}
	for _, r := range rules {
		r.client = c.client
		r.kind = fwRuleKindLXC
		r.node = c.Node
		r.vmid = uint64(c.VMID)
	}
	return rules, nil
}

func (c *Container) FirewallRule(pos int) *FirewallRule {
	return &FirewallRule{
		client: c.client,
		kind:   fwRuleKindLXC,
		node:   c.Node,
		vmid:   uint64(c.VMID),
		Pos:    pos,
	}
}

func (c *Container) NewFirewallRule(ctx context.Context, rule *FirewallRule) error {
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/rules", c.Node, c.VMID), rule, nil); err != nil {
		return err
	}
	rule.client = c.client
	rule.kind = fwRuleKindLXC
	rule.node = c.Node
	rule.vmid = uint64(c.VMID)
	return nil
}

func (c *Container) GetFirewallOptions(ctx context.Context) (options *FirewallVirtualMachineOption, err error) {
	return options, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/options", c.Node, c.VMID), &options)
}

func (c *Container) UpdateFirewallOptions(ctx context.Context, options *FirewallVirtualMachineOption) error {
	return c.client.Put(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/options", c.Node, c.VMID), options, nil)
}

func (c *Container) HasTag(value string) bool {
	if c.ContainerConfig == nil {
		return false
	}

	if c.ContainerConfig.Tags == "" {
		return false
	}

	if c.ContainerConfig.TagsSlice == nil {
		c.SplitTags()
	}

	for _, tag := range c.ContainerConfig.TagsSlice {
		if tag == value {
			return true
		}
	}

	return false
}

func (c *Container) AddTag(ctx context.Context, value string) (*Task, error) {
	if c.HasTag(value) {
		return nil, ErrNoop
	}

	if c.ContainerConfig.TagsSlice == nil {
		c.SplitTags()
	}

	c.ContainerConfig.TagsSlice = append(c.ContainerConfig.TagsSlice, value)
	c.ContainerConfig.Tags = strings.Join(c.ContainerConfig.TagsSlice, TagSeperator)
	c.Tags = c.ContainerConfig.Tags

	return c.Config(ctx, ContainerOption{
		Name:  "tags",
		Value: c.ContainerConfig.Tags,
	})
}

func (c *Container) RemoveTag(ctx context.Context, value string) (*Task, error) {
	if !c.HasTag(value) {
		return nil, ErrNoop
	}

	if c.ContainerConfig.TagsSlice == nil {
		c.SplitTags()
	}

	for i, tag := range c.ContainerConfig.TagsSlice {
		if tag == value {
			c.ContainerConfig.TagsSlice = append(
				c.ContainerConfig.TagsSlice[:i],
				c.ContainerConfig.TagsSlice[i+1:]...,
			)
		}
	}

	c.ContainerConfig.Tags = strings.Join(c.ContainerConfig.TagsSlice, TagSeperator)
	c.Tags = c.ContainerConfig.Tags
	return c.Config(ctx, ContainerOption{
		Name:  "tags",
		Value: c.ContainerConfig.Tags,
	})
}

func (c *Container) SplitTags() {
	c.ContainerConfig.TagsSlice = strings.Split(c.ContainerConfig.Tags, TagSeperator)
}

func (c *Container) Pending(ctx context.Context) (pending []*ContainerPending, err error) {
	return pending, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/pending", c.Node, c.VMID), &pending)
}

func (c *Container) RRD(ctx context.Context, ds string, timeframe Timeframe, consolidationFunction ...ConsolidationFunction) (rrd *ContainerRRD, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/lxc/%d/rrd", c.Node, c.VMID)}
	params := url.Values{}
	if len(consolidationFunction) > 0 {
		if len(consolidationFunction) != 1 {
			return nil, fmt.Errorf("only one consolidation function allowed")
		}
		params.Add("cf", string(consolidationFunction[0]))
	}
	params.Add("ds", ds)
	params.Add("timeframe", string(timeframe))
	u.RawQuery = params.Encode()
	err = c.client.Get(ctx, u.String(), &rrd)
	return
}

func (c *Container) RemoteMigrate(ctx context.Context, params *ContainerRemoteMigrateOptions) (task *Task, err error) {
	var upid UPID
	if err := c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/remote_migrate", c.Node, c.VMID), params, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, c.client), nil
}

func (c *Container) SpiceProxy(ctx context.Context) (spice *SpiceProxy, err error) {
	return spice, c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/spiceproxy", c.Node, c.VMID), nil, &spice)
}

func (c *Container) FirewallLog(ctx context.Context) (entries []*FirewallLogEntry, err error) {
	return entries, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/log", c.Node, c.VMID), &entries)
}

func (c *Container) FirewallRefs(ctx context.Context) (refs []*FirewallRef, err error) {
	return refs, c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/firewall/refs", c.Node, c.VMID), &refs)
}

func (c *Container) DirIndex(ctx context.Context) (entries []*ContainerDirIndexEntry, err error) {
	err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d", c.Node, c.VMID), &entries)
	return
}

func (c *Container) StatusIndex(ctx context.Context) (entries []*ContainerStatusIndexEntry, err error) {
	err = c.client.Get(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status", c.Node, c.VMID), &entries)
	return
}

func (c *Container) MigratePreconditions(ctx context.Context, target string) (preconditions *ContainerMigratePreconditions, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/lxc/%d/migrate", c.Node, c.VMID)}
	if target != "" {
		params := url.Values{}
		params.Add("target", target)
		u.RawQuery = params.Encode()
	}
	err = c.client.Get(ctx, u.String(), &preconditions)
	return
}

func (c *Container) MigrationTunnel(ctx context.Context, options *ContainerMigrationTunnelOptions) (tunnel *ContainerMigrationTunnel, err error) {
	if options == nil {
		options = &ContainerMigrationTunnelOptions{}
	}
	err = c.client.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/mtunnel", c.Node, c.VMID), options, &tunnel)
	return
}

func (c *Container) MigrationTunnelWebSocketPath(tunnel *ContainerMigrationTunnel) string {
	q := url.Values{}
	if tunnel != nil {
		q.Set("socket", tunnel.Socket)
		q.Set("ticket", tunnel.Ticket)
	}
	return fmt.Sprintf("/nodes/%s/lxc/%d/mtunnelwebsocket?%s", c.Node, c.VMID, q.Encode())
}
