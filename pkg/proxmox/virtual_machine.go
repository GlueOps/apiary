package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const (
	StatusVirtualMachineRunning = "running"
	StatusVirtualMachineStopped = "stopped"
	StatusVirtualMachinePaused  = "paused"

	UserDataISOFormat = "user-data-%d.iso"
	TagCloudInit      = "cloud-init"
	TagSeperator      = ";"

	volumeIdentifier = "cidata"
	blockSize        = 2048
)

var DefaultAgentWaitInterval = 100 * time.Millisecond

func (v *VirtualMachine) New(c *Client, nodeName string, vmid int) {
	v.client = c
	v.Node = nodeName
	v.VMID = StringOrUint64(clampIntToUint64(vmid))
}

func (v *VirtualMachine) Ping(ctx context.Context) error {
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/current", v.Node, v.VMID), &v); err != nil {
		return err
	}
	cfg := &VirtualMachineConfig{}
	if err := v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/config", v.Node, v.VMID), cfg); err != nil {
		return err
	}
	v.VirtualMachineConfig = cfg
	return nil
}

func (v *VirtualMachine) Config(ctx context.Context, options ...VirtualMachineOption) (*Task, error) {
	var upid UPID
	data := make(map[string]interface{})
	for _, opt := range options {
		data[opt.Name] = opt.Value
	}
	err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/config", v.Node, v.VMID), data, &upid)
	return NewTask(upid, v.client), err
}

func (v *VirtualMachine) ConfigSync(ctx context.Context, options ...VirtualMachineOption) error {
	data := make(map[string]interface{})
	for _, opt := range options {
		data[opt.Name] = opt.Value
	}
	return v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/config", v.Node, v.VMID), data, nil)
}

func (v *VirtualMachine) Feature(ctx context.Context, feature, snapname string) (VirtualMachineFeature, error) {
	var result VirtualMachineFeature
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/qemu/%d/feature", v.Node, v.VMID)}
	params := url.Values{}
	params.Set("feature", feature)
	if snapname != "" {
		params.Set("snapname", snapname)
	}
	u.RawQuery = params.Encode()
	err := v.client.Get(ctx, u.String(), &result)
	return result, err
}

func (v *VirtualMachine) DBusVMState(ctx context.Context, action string) error {
	data := map[string]interface{}{"action": action}
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/dbus-vmstate", v.Node, v.VMID), data, nil)
}

func (v *VirtualMachine) Monitor(ctx context.Context, command string) (s string, err error) {
	data := make(map[string]interface{})
	data["command"] = command
	err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/monitor", v.Node, v.VMID), data, &s)
	return s, err
}

func (v *VirtualMachine) TermProxy(ctx context.Context) (term *Term, err error) {
	return term, v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/termproxy", v.Node, v.VMID), nil, &term)
}

func (v *VirtualMachine) VNCProxy(ctx context.Context, config *VNCConfig) (vnc *VNC, err error) {
	return vnc, v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy?", v.Node, v.VMID), config, &vnc)
}

func (v *VirtualMachine) HasTag(value string) bool {
	if v.VirtualMachineConfig == nil {
		return false
	}

	if v.VirtualMachineConfig.Tags == "" {
		return false
	}

	if v.VirtualMachineConfig.TagsSlice == nil {
		v.SplitTags()
	}

	for _, tag := range v.VirtualMachineConfig.TagsSlice {
		if tag == value {
			return true
		}
	}

	return false
}

func (v *VirtualMachine) AddTag(ctx context.Context, value string) (*Task, error) {
	if v.HasTag(value) {
		return nil, ErrNoop
	}

	if v.VirtualMachineConfig.TagsSlice == nil {
		v.SplitTags()
	}

	v.VirtualMachineConfig.TagsSlice = append(v.VirtualMachineConfig.TagsSlice, value)
	v.VirtualMachineConfig.Tags = strings.Join(v.VirtualMachineConfig.TagsSlice, TagSeperator)

	return v.Config(ctx, VirtualMachineOption{
		Name:  "tags",
		Value: v.VirtualMachineConfig.Tags,
	})
}

func (v *VirtualMachine) RemoveTag(ctx context.Context, value string) (*Task, error) {
	if !v.HasTag(value) {
		return nil, ErrNoop
	}

	if v.VirtualMachineConfig.TagsSlice == nil {
		v.SplitTags()
	}

	for i, tag := range v.VirtualMachineConfig.TagsSlice {
		if tag == value {
			v.VirtualMachineConfig.TagsSlice = append(
				v.VirtualMachineConfig.TagsSlice[:i],
				v.VirtualMachineConfig.TagsSlice[i+1:]...,
			)
		}
	}

	v.VirtualMachineConfig.Tags = strings.Join(v.VirtualMachineConfig.TagsSlice, TagSeperator)
	return v.Config(ctx, VirtualMachineOption{
		Name:  "tags",
		Value: v.VirtualMachineConfig.Tags,
	})
}

func (v *VirtualMachine) SplitTags() {
	v.VirtualMachineConfig.TagsSlice = strings.Split(v.VirtualMachineConfig.Tags, TagSeperator)
}

type CloudInitOption func(*cloudInitConfig)

type cloudInitConfig struct {
	storage string
}

func WithCloudInitStorage(name string) CloudInitOption {
	return func(c *cloudInitConfig) { c.storage = name }
}

func (v *VirtualMachine) CloudInit(ctx context.Context, device, userdata, metadata, vendordata, networkconfig string, opts ...CloudInitOption) error {
	var cfg cloudInitConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	isoName := fmt.Sprintf(UserDataISOFormat, v.VMID)

	isofilename, err := makeCloudInitISO(isoName, userdata, metadata, vendordata, networkconfig)
	if err != nil {
		return err
	}

	defer func() {
		if rerr := os.Remove(isofilename); rerr != nil {
			v.client.log.Warnf("failed to remove temp cloud-init iso %s: %v", isofilename, rerr)
		}
	}()

	node, err := v.client.Node(ctx, v.Node)
	if err != nil {
		return err
	}

	storage, err := resolveCloudInitStorage(ctx, node, &cfg)
	if err != nil {
		return err
	}

	task, err := storage.Upload("iso", isofilename)
	if err != nil {
		return err
	}

	if err := task.WaitFor(ctx, 5); err != nil {
		return err
	}

	_, err = v.AddTag(ctx, MakeTag(TagCloudInit))
	if err != nil && !IsErrNoop(err) {
		return err
	}

	task, err = v.Config(ctx, VirtualMachineOption{
		Name:  device,
		Value: fmt.Sprintf("%s:iso/%s,media=cdrom", storage.Name, isoName),
	}, VirtualMachineOption{
		Name:  "boot",
		Value: fmt.Sprintf("%s;%s", v.VirtualMachineConfig.Boot, device),
	})
	if err != nil {
		return err
	}

	return task.WaitFor(ctx, 2)
}

func resolveCloudInitStorage(ctx context.Context, node *Node, cfg *cloudInitConfig) (*Storage, error) {
	if cfg.storage == "" {
		return node.StorageISO(ctx)
	}
	s, err := node.Storage(ctx, cfg.storage)
	if err != nil {
		return nil, fmt.Errorf("cloud-init storage %q: %w", cfg.storage, err)
	}
	if !strings.Contains(s.Content, "iso") {
		return nil, fmt.Errorf("cloud-init storage %q does not accept iso content (got %q)", cfg.storage, s.Content)
	}
	return s, nil
}

func makeCloudInitISO(filename, userdata, metadata, vendordata, networkconfig string) (isopath string, err error) {
	isopath = filepath.Join(os.TempDir(), filename)

	isoFile, err := os.Create(isopath)
	if err != nil {
		return "", err
	}

	if err := isoFile.Close(); err != nil {
		return "", err
	}

	iso, err := file.OpenFromPath(isopath, false)
	if err != nil {
		return "", err
	}

	defer func() {
		if cerr := iso.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	fs, err := iso9660.Create(iso, 0, 0, blockSize, "")
	if err != nil {
		return "", err
	}

	if err = fs.Mkdir("/"); err != nil {
		return "", err
	}

	cifiles := map[string]string{
		"user-data": userdata,
		"meta-data": metadata,
	}
	if vendordata != "" {
		cifiles["vendor-data"] = vendordata
	}
	if networkconfig != "" {
		cifiles["network-config"] = networkconfig
	}

	for name, content := range cifiles {
		rw, err := fs.OpenFile("/"+name, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return "", err
		}

		if _, err = rw.Write([]byte(content)); err != nil {
			return "", err
		}

		if err = rw.Close(); err != nil {
			return "", err
		}
	}

	if err = fs.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		Joliet:           true,
		VolumeIdentifier: volumeIdentifier,
	}); err != nil {
		return "", err
	}

	return
}

func (v *VirtualMachine) TermWebSocket(term *Term) (chan []byte, chan []byte, chan error, func() error, error) {
	p := fmt.Sprintf("/nodes/%s/qemu/%d/vncwebsocket?port=%d&vncticket=%s",
		v.Node, v.VMID, term.Port, url.QueryEscape(term.Ticket))

	return v.client.TermWebSocket(p, term)
}

func (v *VirtualMachine) VNCWebSocket(vnc *VNC) (chan []byte, chan []byte, chan error, func() error, error) {
	p := fmt.Sprintf("/nodes/%s/qemu/%d/vncwebsocket?port=%d&vncticket=%s",
		v.Node, v.VMID, vnc.Port, url.QueryEscape(vnc.Ticket))

	return v.client.VNCWebSocket(p, vnc)
}

func (v *VirtualMachine) SpiceProxy(ctx context.Context) (spice *SpiceProxy, err error) {
	return spice, v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/spiceproxy", v.Node, v.VMID), nil, &spice)
}

func (v *VirtualMachine) IsRunning() bool {
	return v.Status == StatusVirtualMachineRunning && (v.QMPStatus == "" || v.QMPStatus == StatusVirtualMachineRunning)
}

func (v *VirtualMachine) Start(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/start", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) IsStopped() bool {
	return v.Status == StatusVirtualMachineStopped && (v.Lock != "suspended")
}

func (v *VirtualMachine) Reset(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/reset", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Shutdown(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/shutdown", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Stop(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) IsPaused() bool {
	return v.Status == StatusVirtualMachineRunning && v.QMPStatus == StatusVirtualMachinePaused
}

func (v *VirtualMachine) Pause(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/suspend", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) IsHibernated() bool {
	return v.Status == StatusVirtualMachineStopped && v.Lock == "suspended"
}

func (v *VirtualMachine) Hibernate(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/suspend", v.Node, v.VMID), map[string]string{"todisk": "1"}, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Resume(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/resume", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Reboot(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/reboot", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Delete(ctx context.Context, options *VirtualMachineDeleteOptions) (task *Task, err error) {
	if ok, err := v.deleteCloudInitISO(ctx); err != nil || !ok {
		return nil, err
	}

	var upid UPID

	if err = v.client.DeleteWithParams(ctx, fmt.Sprintf("/nodes/%s/qemu/%d", v.Node, v.VMID), options, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) deleteCloudInitISO(ctx context.Context) (ok bool, err error) {
	if !v.HasTag(MakeTag(TagCloudInit)) {
		return true, nil
	}

	node, err := v.client.Node(ctx, v.Node)
	if err != nil {
		return false, err
	}

	storages, err := node.Storages(ctx)
	if err != nil {
		return false, err
	}

	isoFilename := fmt.Sprintf(UserDataISOFormat, v.VMID)
	for _, s := range storages {
		if s.Enabled == 0 || !strings.Contains(s.Content, "iso") {
			continue
		}
		iso, ierr := s.ISO(ctx, isoFilename)
		if ierr != nil {

			continue
		}
		task, terr := iso.Delete(ctx)
		if terr != nil {
			return false, terr
		}
		if werr := task.WaitFor(ctx, 5); werr != nil {
			return false, werr
		}
		return true, nil
	}

	return true, nil
}

func (v *VirtualMachine) MigratePreconditions(ctx context.Context, target string) (preconditions *VirtualMachineMigratePreconditions, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/qemu/%d/migrate", v.Node, v.VMID)}
	if target != "" {
		params := url.Values{}
		params.Add("target", target)
		u.RawQuery = params.Encode()
	}
	err = v.client.Get(ctx, u.String(), &preconditions)
	return
}

func (v *VirtualMachine) Migrate(
	ctx context.Context,
	params *VirtualMachineMigrateOptions,
) (task *Task, err error) {
	var upid UPID

	if params == nil {
		params = &VirtualMachineMigrateOptions{}
	}

	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/migrate", v.Node, v.VMID), params, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) RemoteMigrate(ctx context.Context, params *VirtualMachineRemoteMigrateOptions) (task *Task, err error) {
	var upid UPID
	if params == nil {
		params = &VirtualMachineRemoteMigrateOptions{}
	}
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/remote_migrate", v.Node, v.VMID), params, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Clone(ctx context.Context, params *VirtualMachineCloneOptions) (newid int, task *Task, err error) {
	var upid UPID

	if params == nil {
		params = &VirtualMachineCloneOptions{}
	}

	if params.NewID == 0 {
		cluster, err := v.client.Cluster(ctx)
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

	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/clone", v.Node, v.VMID), params, &upid); err != nil {
		return newid, nil, err
	}

	return newid, NewTask(upid, v.client), nil
}

func (v *VirtualMachine) ResizeDisk(ctx context.Context, disk, size string) (*Task, error) {
	var upid UPID

	if err := v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/resize", v.Node, v.VMID), map[string]string{
		"disk": disk,
		"size": size,
	}, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) UnlinkDisk(ctx context.Context, diskID string, force bool) (task *Task, err error) {
	var upid UPID

	params := map[string]string{"idlist": diskID}
	if force {
		params["force"] = "1"
	}
	err = v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/unlink", v.Node, v.VMID), params, &upid)
	if err != nil {
		return
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) MoveDisk(ctx context.Context, disk string, params *VirtualMachineMoveDiskOptions) (task *Task, err error) {
	var upid UPID

	if params == nil {
		params = &VirtualMachineMoveDiskOptions{}
	}

	if disk != "" {
		params.Disk = disk
	}

	err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/move_disk", v.Node, v.VMID), params, &upid)
	if err != nil {
		return
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) AgentGetHostName(ctx context.Context) (hostname string, err error) {
	node, err := v.client.Node(ctx, v.Node)
	if err != nil {
		return
	}

	var resp struct {
		Result *AgentHostName `json:"result"`
	}
	if err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-host-name", node.Name, v.VMID), &resp); err != nil {
		return
	}
	if resp.Result == nil {
		err = fmt.Errorf("result is empty")
		return
	}
	hostname = resp.Result.HostName
	return
}

func (v *VirtualMachine) AgentGetNetworkIFaces(ctx context.Context) (iFaces []*AgentNetworkIface, err error) {
	var resp struct {
		Result []*AgentNetworkIface `json:"result"`
	}
	if err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", v.Node, v.VMID), &resp); err != nil {
		return
	}
	for _, iface := range resp.Result {
		if iface.Name == "lo" {
			continue
		}
		iFaces = append(iFaces, iface)
	}
	return
}

func (v *VirtualMachine) WaitForAgent(ctx context.Context, seconds int) error {
	timeout := time.After(time.Duration(seconds) * time.Second)
	ticker := time.NewTicker(DefaultWaitInterval)
	defer ticker.Stop()

	for {
		_, err := v.AgentOsInfo(ctx)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "500 QEMU guest agent is not running") {
			return err
		}

		select {
		case <-timeout:
			return ErrTimeout
		case <-ticker.C:
		}
	}
}

func (v *VirtualMachine) AgentExec(ctx context.Context, command []string, inputData string) (pid int, err error) {
	tmpdata := map[string]interface{}{}
	err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec", v.Node, v.VMID),
		map[string]interface{}{
			"command":    command,
			"input-data": inputData,
		},
		&tmpdata)

	p := tmpdata["pid"]
	if p == nil {
		return 0, fmt.Errorf("no pid returned from agent exec command")
	}
	pid = int(p.(float64))
	return
}

func (v *VirtualMachine) AgentExecStatus(ctx context.Context, pid int) (status *AgentExecStatus, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/exec-status?pid=%d", v.Node, v.VMID, pid), &status)
	if err != nil {
		return nil, err
	}

	return
}

func (v *VirtualMachine) WaitForAgentExecExit(ctx context.Context, pid, seconds int) (*AgentExecStatus, error) {
	timeout := time.After(time.Duration(seconds) * time.Second)
	ticker := time.NewTicker(DefaultAgentWaitInterval)
	defer ticker.Stop()

	for {
		status, err := v.AgentExecStatus(ctx, pid)
		if err != nil {
			return nil, err
		}
		if status.Exited != 0 {
			return status, nil
		}

		select {
		case <-timeout:
			return nil, ErrTimeout
		case <-ticker.C:
		}
	}
}

func (v *VirtualMachine) AgentOsInfo(ctx context.Context) (info *AgentOsInfo, err error) {
	var resp struct {
		Result *AgentOsInfo `json:"result"`
	}
	if err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/get-osinfo", v.Node, v.VMID), &resp); err != nil {
		return
	}
	if resp.Result == nil {
		err = fmt.Errorf("result is empty")
		return
	}
	info = resp.Result
	return
}

func (v *VirtualMachine) AgentSetUserPassword(ctx context.Context, password string, username string) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/agent/set-user-password", v.Node, v.VMID), map[string]string{"password": password, "username": username}, nil)
}

func (v *VirtualMachine) SendKey(ctx context.Context, key string) error {
	data := map[string]interface{}{"key": key}
	return v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/sendkey", v.Node, v.VMID), data, nil)
}

func (v *VirtualMachine) GetFirewallIPSet(ctx context.Context) (ipsets []*FirewallIPSet, err error) {
	return ipsets, v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset", v.Node, v.VMID), &ipsets)
}

func (v *VirtualMachine) NewFirewallIPSet(ctx context.Context, ipset FirewallIPSetCreationOption) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset", v.Node, v.VMID), ipset, nil)
}

func (v *VirtualMachine) DeleteFirewallIPSet(ctx context.Context, name string, force bool) error {
	return v.client.Delete(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s", v.Node, v.VMID, name), map[string]interface{}{"force": force})
}

func (v *VirtualMachine) GetFirewallIPSetEntries(ctx context.Context, name string) (entries []*FirewallIPSetEntry, err error) {
	return entries, v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s", v.Node, v.VMID, name), &entries)
}

func (v *VirtualMachine) NewFirewallIPSetEntry(ctx context.Context, name string, entry FirewallIPSetEntryCreationOption) error {
	return v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s", v.Node, v.VMID, name), entry, nil)
}

func (v *VirtualMachine) DeleteFirewallIPSetEntry(ctx context.Context, name string, cidr string, digest string) error {
	return v.client.Delete(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s/%s", v.Node, v.VMID, name, cidr), map[string]interface{}{
		"digest": digest,
	})
}

func (v *VirtualMachine) GetFirewallIPSetEntry(ctx context.Context, name string, cidr string) (entry *FirewallIPSetEntry, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s/%s", v.Node, v.VMID, name, cidr), &entry)
	return
}

func (v *VirtualMachine) UpdateFirewallIPSetEntry(ctx context.Context, name string, cidr string, entry *FirewallIPSetEntryUpdateOption) error {
	return v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/ipset/%s/%s", v.Node, v.VMID, name, cidr), entry, nil)
}

func (v *VirtualMachine) FirewallOptionGet(ctx context.Context) (firewallOption *FirewallVirtualMachineOption, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", v.Node, v.VMID), firewallOption)
	return
}

func (v *VirtualMachine) FirewallOptionSet(ctx context.Context, firewallOption *FirewallVirtualMachineOption) error {
	return v.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/options", v.Node, v.VMID), firewallOption, nil)
}

func (v *VirtualMachine) FirewallRules(ctx context.Context) (rules []*FirewallRule, err error) {
	if err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", v.Node, v.VMID), &rules); err != nil {
		return nil, err
	}
	for _, r := range rules {
		r.client = v.client
		r.kind = fwRuleKindQemu
		r.node = v.Node
		r.vmid = uint64(v.VMID)
	}
	return rules, nil
}

func (v *VirtualMachine) FirewallRule(pos int) *FirewallRule {
	return &FirewallRule{
		client: v.client,
		kind:   fwRuleKindQemu,
		node:   v.Node,
		vmid:   uint64(v.VMID),
		Pos:    pos,
	}
}

func (v *VirtualMachine) NewFirewallRule(ctx context.Context, rule *FirewallRule) error {
	if err := v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", v.Node, v.VMID), rule, nil); err != nil {
		return err
	}
	rule.client = v.client
	rule.kind = fwRuleKindQemu
	rule.node = v.Node
	rule.vmid = uint64(v.VMID)
	return nil
}

func (v *VirtualMachine) NewSnapshot(ctx context.Context, name string) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", v.Node, v.VMID), map[string]string{"snapname": name}, &upid); err != nil {
		return nil, err
	}

	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) Snapshots(ctx context.Context) (snapshots []*VirtualMachineSnapshot, err error) {
	if err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", v.Node, v.VMID), &snapshots); err != nil {
		return nil, err
	}
	for _, s := range snapshots {
		s.client = v.client
		s.Node = v.Node
		s.VMID = clampToInt(uint64(v.VMID))
	}
	return snapshots, nil
}

func (v *VirtualMachine) Snapshot(name string) *VirtualMachineSnapshot {
	return &VirtualMachineSnapshot{
		client: v.client,
		Node:   v.Node,
		VMID:   clampToInt(uint64(v.VMID)),
		Name:   name,
	}
}

type VirtualMachineSnapshotUpdateOptions struct {
	Description string `json:"description,omitempty"`
}

func (s *VirtualMachineSnapshot) Rollback(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = s.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s/rollback", s.Node, s.VMID, s.Name), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

func (s *VirtualMachineSnapshot) Delete(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err := s.client.Delete(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s", s.Node, s.VMID, s.Name), &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, s.client), nil
}

func (s *VirtualMachineSnapshot) Config(ctx context.Context) (config map[string]interface{}, err error) {
	return config, s.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s/config", s.Node, s.VMID, s.Name), &config)
}

func (s *VirtualMachineSnapshot) UpdateConfig(ctx context.Context, options *VirtualMachineSnapshotUpdateOptions) error {
	if options == nil {
		options = &VirtualMachineSnapshotUpdateOptions{}
	}
	return s.client.Put(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s/config", s.Node, s.VMID, s.Name), options, nil)
}

func (s *VirtualMachineSnapshot) SubResources(ctx context.Context) (entries []*VirtualMachineSnapshotIndexEntry, err error) {
	err = s.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s", s.Node, s.VMID, s.Name), &entries)
	return
}

func (v *VirtualMachine) RRD(ctx context.Context, ds string, timeframe Timeframe, consolidationFunction ...ConsolidationFunction) (rrd *VirtualMachineRRD, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/qemu/%d/rrd", v.Node, v.VMID)}
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
	err = v.client.Get(ctx, u.String(), &rrd)
	return
}

func (v *VirtualMachine) RRDData(ctx context.Context, timeframe Timeframe, consolidationFunction ...ConsolidationFunction) (rrddata []*RRDData, err error) {
	u := url.URL{Path: fmt.Sprintf("/nodes/%s/qemu/%d/rrddata", v.Node, v.VMID)}

	params := url.Values{}
	if len(consolidationFunction) > 0 {
		if len(consolidationFunction) != 1 {
			return nil, fmt.Errorf("only one consolidation function allowed")
		}

		params.Add("cf", string(consolidationFunction[0]))
	}

	params.Add("timeframe", string(timeframe))
	u.RawQuery = params.Encode()

	err = v.client.Get(ctx, u.String(), &rrddata)
	return
}

func (v *VirtualMachine) ConvertToTemplate(ctx context.Context) (task *Task, err error) {
	var upid UPID
	if err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/template", v.Node, v.VMID), nil, &upid); err != nil {
		return nil, err
	}
	return NewTask(upid, v.client), nil
}

func (v *VirtualMachine) UnmountCloudInitISO(ctx context.Context, device string) error {
	if !v.HasTag(MakeTag(TagCloudInit)) {
		return nil
	}

	_, err := v.Config(ctx, VirtualMachineOption{
		Name:  device,
		Value: "none,media=cdrom",
	})
	if err != nil {
		return err
	}

	if _, err = v.deleteCloudInitISO(ctx); err != nil {
		return err
	}
	return nil
}

func (v *VirtualMachine) Pending(ctx context.Context) (pending *PendingConfiguration, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/pending", v.Node, v.VMID), &pending)
	return
}

func (v *VirtualMachine) DirIndex(ctx context.Context) (entries []*VirtualMachineDirIndexEntry, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d", v.Node, v.VMID), &entries)
	return
}

func (v *VirtualMachine) StatusIndex(ctx context.Context) (entries []*VirtualMachineStatusIndexEntry, err error) {
	err = v.client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status", v.Node, v.VMID), &entries)
	return
}

func (v *VirtualMachine) MigrationTunnel(ctx context.Context, options *VirtualMachineMigrationTunnelOptions) (tunnel *VirtualMachineMigrationTunnel, err error) {
	if options == nil {
		options = &VirtualMachineMigrationTunnelOptions{}
	}
	err = v.client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/mtunnel", v.Node, v.VMID), options, &tunnel)
	return
}

func (v *VirtualMachine) MigrationTunnelWebSocketPath(tunnel *VirtualMachineMigrationTunnel) string {
	q := url.Values{}
	if tunnel != nil {
		q.Set("socket", tunnel.Socket)
		q.Set("ticket", tunnel.Ticket)
	}
	return fmt.Sprintf("/nodes/%s/qemu/%d/mtunnelwebsocket?%s", v.Node, v.VMID, q.Encode())
}
