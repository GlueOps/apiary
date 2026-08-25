package agent

import (
	"os"

	"github.com/glueops/apiary-agent/internal/hostinstall"
	"github.com/spf13/cobra"
)

var installExecPath string

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and start this host's agent service",
	Long: "Writes the systemd unit and environment file, reloads systemd, and enables and starts\n" +
		"the service.\n\n" +
		"Enrolment obtains a credential; it does not make the agent a service. A host that has\n" +
		"only been enrolled has an identity and no supervision — nothing restarts the agent if it\n" +
		"dies, nothing catches it if it wedges, and nothing brings it back after a reboot. It\n" +
		"heartbeats once, from the shell it was run in, and afterwards looks identical to a host\n" +
		"nobody onboarded.\n\n" +
		"Safe to re-run. Re-running is how an agent is repointed at a different control plane, and\n" +
		"how a host whose unit was lost is repaired.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if err := hostinstall.Install(cmd.Context(), hostinstall.Plan{
			Unit:     hostinstall.Unit,
			BaseURL:  cfg.BaseURL,
			ExecPath: installExecPath,
		}); err != nil {
			return err
		}

		cmd.Printf("installed %s and started %s\n", hostinstall.UnitPath, hostinstall.UnitName)
		cmd.Printf("control plane: %s\n", cfg.BaseURL)
		if _, err := os.Stat(cfg.StatePath); err != nil {
			// Installed but not enrolled. The unit will start, fail to
			// authenticate and keep retrying — which is recoverable and looks
			// alarming, so it is worth saying now rather than leaving in the
			// journal for whoever looks next.
			cmd.Printf("\nthis host has no credential yet: run `apiary-agent enrol --token ... "+
				"--base-url %s` and the service will pick it up on its next attempt\n", cfg.BaseURL)
		}
		return nil
	},
}

func init() {
	installCmd.Flags().StringVar(&installExecPath, "exec-path", "",
		"Path the unit should run. Defaults to this binary, wherever it currently is, "+
			"so the unit does not name a path that does not exist.")
	rootCmd.AddCommand(installCmd)
}
