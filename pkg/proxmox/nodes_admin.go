package proxmox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

func (n *Node) GetConfig(ctx context.Context) (cfg *NodeConfig, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/config", n.Name), &cfg)
	return
}

func (n *Node) GetConfigProperty(ctx context.Context, property string) (cfg *NodeConfig, err error) {
	if property == "" {
		return n.GetConfig(ctx)
	}
	q := url.Values{}
	q.Set("property", property)
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/config?%s", n.Name, q.Encode()), &cfg)
	return
}

func (n *Node) UpdateConfig(ctx context.Context, opts *NodeConfigOptions) error {
	if opts == nil {
		opts = &NodeConfigOptions{}
	}
	return n.client.Put(ctx, fmt.Sprintf("/nodes/%s/config", n.Name), opts, nil)
}

func (n *Node) Hosts(ctx context.Context) (hosts *NodeHosts, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/hosts", n.Name), &hosts)
	return
}

func (n *Node) UpdateHosts(ctx context.Context, data, digest string) error {
	body := map[string]string{"data": data}
	if digest != "" {
		body["digest"] = digest
	}
	return n.client.Post(ctx, fmt.Sprintf("/nodes/%s/hosts", n.Name), body, nil)
}

func (n *Node) Reboot(ctx context.Context) error {
	return n.client.Post(ctx, fmt.Sprintf("/nodes/%s/status", n.Name),
		map[string]string{"command": "reboot"}, nil)
}

func (n *Node) Shutdown(ctx context.Context) error {
	return n.client.Post(ctx, fmt.Sprintf("/nodes/%s/status", n.Name),
		map[string]string{"command": "shutdown"}, nil)
}

type NodeJournalOptions struct {
	Since       int64
	Until       int64
	StartCursor string
	EndCursor   string
	LastEntries int
}

func (n *Node) Journal(ctx context.Context, opts *NodeJournalOptions) (lines []string, err error) {
	path := fmt.Sprintf("/nodes/%s/journal", n.Name)
	if opts != nil {
		q := url.Values{}
		if opts.Since > 0 {
			q.Set("since", strconv.FormatInt(opts.Since, 10))
		}
		if opts.Until > 0 {
			q.Set("until", strconv.FormatInt(opts.Until, 10))
		}
		if opts.StartCursor != "" {
			q.Set("startcursor", opts.StartCursor)
		}
		if opts.EndCursor != "" {
			q.Set("endcursor", opts.EndCursor)
		}
		if opts.LastEntries > 0 {
			q.Set("lastentries", strconv.Itoa(opts.LastEntries))
		}
		if len(q) > 0 {
			path = path + "?" + q.Encode()
		}
	}
	err = n.client.Get(ctx, path, &lines)
	return
}

type NodeSyslogOptions struct {
	Start   int
	Limit   int
	Since   string
	Until   string
	Service string
}

func (n *Node) Syslog(ctx context.Context, opts *NodeSyslogOptions) (entries []*LogEntry, err error) {
	path := fmt.Sprintf("/nodes/%s/syslog", n.Name)
	if opts != nil {
		q := url.Values{}
		if opts.Start > 0 {
			q.Set("start", strconv.Itoa(opts.Start))
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Since != "" {
			q.Set("since", opts.Since)
		}
		if opts.Until != "" {
			q.Set("until", opts.Until)
		}
		if opts.Service != "" {
			q.Set("service", opts.Service)
		}
		if len(q) > 0 {
			path = path + "?" + q.Encode()
		}
	}
	err = n.client.Get(ctx, path, &entries)
	return
}

func (n *Node) Netstat(ctx context.Context) (entries []map[string]any, err error) {
	err = n.client.Get(ctx, fmt.Sprintf("/nodes/%s/netstat", n.Name), &entries)
	return
}

type NodeExecuteCommand struct {
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Args   map[string]interface{} `json:"args,omitempty"`
}

func (n *Node) Execute(ctx context.Context, commands []*NodeExecuteCommand) (results []map[string]any, err error) {
	if len(commands) == 0 {
		return nil, errors.New("at least one command is required")
	}
	body := map[string]interface{}{"commands": commands}
	err = n.client.Post(ctx, fmt.Sprintf("/nodes/%s/execute", n.Name), body, &results)
	return
}

type NodeConsoleCmd string

const (
	NodeConsoleLogin       NodeConsoleCmd = "login"
	NodeConsoleUpgrade     NodeConsoleCmd = "upgrade"
	NodeConsoleCephInstall NodeConsoleCmd = "ceph_install"
)

type NodeVNCShellOptions struct {
	Cmd       NodeConsoleCmd
	CmdOpts   string
	Width     int
	Height    int
	WebSocket bool
}

func (n *Node) VNCShell(ctx context.Context, opts *NodeVNCShellOptions) (vnc *VNC, err error) {
	body := map[string]interface{}{}
	if opts != nil {
		if opts.Cmd != "" {
			body["cmd"] = string(opts.Cmd)
		}
		if opts.CmdOpts != "" {
			body["cmd-opts"] = opts.CmdOpts
		}
		if opts.Width > 0 {
			body["width"] = opts.Width
		}
		if opts.Height > 0 {
			body["height"] = opts.Height
		}
		if opts.WebSocket {
			body["websocket"] = 1
		}
	}
	err = n.client.Post(ctx, fmt.Sprintf("/nodes/%s/vncshell", n.Name), body, &vnc)
	return
}

type NodeSpiceShellOptions struct {
	Cmd     NodeConsoleCmd
	CmdOpts string
	Proxy   string
}

func (n *Node) SpiceShell(ctx context.Context, opts *NodeSpiceShellOptions) (proxy *SpiceProxy, err error) {
	body := map[string]interface{}{}
	if opts != nil {
		if opts.Cmd != "" {
			body["cmd"] = string(opts.Cmd)
		}
		if opts.CmdOpts != "" {
			body["cmd-opts"] = opts.CmdOpts
		}
		if opts.Proxy != "" {
			body["proxy"] = opts.Proxy
		}
	}
	err = n.client.Post(ctx, fmt.Sprintf("/nodes/%s/spiceshell", n.Name), body, &proxy)
	return
}

type NodeTasksOptions struct {
	Errors       bool
	Limit        int
	Since        int64
	Until        int64
	Source       string
	Start        int
	StatusFilter string
	TypeFilter   string
	UserFilter   string
	VMID         int
}

func (n *Node) Tasks(ctx context.Context, opts *NodeTasksOptions) (tasks []*Task, err error) {
	path := fmt.Sprintf("/nodes/%s/tasks", n.Name)
	if opts != nil {
		q := url.Values{}
		if opts.Errors {
			q.Set("errors", "1")
		}
		if opts.Limit > 0 {
			q.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Since > 0 {
			q.Set("since", strconv.FormatInt(opts.Since, 10))
		}
		if opts.Until > 0 {
			q.Set("until", strconv.FormatInt(opts.Until, 10))
		}
		if opts.Source != "" {
			q.Set("source", opts.Source)
		}
		if opts.Start > 0 {
			q.Set("start", strconv.Itoa(opts.Start))
		}
		if opts.StatusFilter != "" {
			q.Set("statusfilter", opts.StatusFilter)
		}
		if opts.TypeFilter != "" {
			q.Set("typefilter", opts.TypeFilter)
		}
		if opts.UserFilter != "" {
			q.Set("userfilter", opts.UserFilter)
		}
		if opts.VMID > 0 {
			q.Set("vmid", strconv.Itoa(opts.VMID))
		}
		if len(q) > 0 {
			path = path + "?" + q.Encode()
		}
	}
	if err = n.client.Get(ctx, path, &tasks); err != nil {
		return
	}
	for _, t := range tasks {
		t.client = n.client
	}
	return
}

func (n *Node) RevertNetworkChanges(ctx context.Context) error {
	return n.client.Delete(ctx, fmt.Sprintf("/nodes/%s/network", n.Name), nil)
}
