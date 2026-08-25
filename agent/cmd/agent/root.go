package agent

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/glueops/apiary-agent/internal/config"
	"github.com/spf13/cobra"
)

// Version is stamped at build time via -ldflags. It is reported on every
// heartbeat, which is how an operator sees that half a datacenter is still
// running last month's agent before starting a rotation that depends on a task
// kind only the new one understands.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Config the agent runs with, resolved once.
type Config struct {
	BaseURL       string
	StatePath     string
	HeartbeatWait time.Duration

	// The local hypervisor. All optional: a host that is not a Proxmox node runs
	// the agent perfectly well and simply refuses fleet tasks when handed one.
	// An agent that would not start without a hypervisor could not be deployed
	// ahead of one.
	NodeName         string
	ProxmoxURL       string
	ProxmoxTokenID   string
	ProxmoxSecret    string
	JoinPasswordPath string
}

func loadConfig() (Config, error) {
	c, err := config.Load(cfgPath, baseURLOverride, statePath)
	if err != nil {
		return Config{}, err
	}
	node := c.NodeName
	if node == "" {
		// Proxmox identifies a node by its short hostname, so that is the
		// sensible default — but it is overridable, because a host whose
		// hostname was changed after installation still answers to the name the
		// cluster knows it by, and guessing wrong makes every guest lookup
		// silently return nothing.
		if h, err := os.Hostname(); err == nil {
			node, _, _ = strings.Cut(h, ".")
		}
	}
	return Config{
		BaseURL:          c.BaseURL,
		StatePath:        c.StatePath,
		HeartbeatWait:    25 * time.Second,
		NodeName:         node,
		ProxmoxURL:       c.ProxmoxURL,
		ProxmoxTokenID:   c.ProxmoxTokenID,
		ProxmoxSecret:    c.ProxmoxSecret,
		JoinPasswordPath: c.JoinPasswordPath,
	}, nil
}

var (
	cfgPath         string
	baseURLOverride string
	statePath       string
)

var rootCmd = &cobra.Command{
	Use:   "apiary-agent",
	Short: "Remote execution agent",
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the agent version",
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Printf("apiary-agent %s (commit %s, built %s)\n", Version, Commit, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&baseURLOverride, "base-url", "", "Override control plane base URL")
	rootCmd.PersistentFlags().StringVar(&statePath, "state-path", "", "Override state file path")
}
