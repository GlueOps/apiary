package agent

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/glueops/apiary-agent/internal/bee"
	"github.com/glueops/apiary-agent/internal/hostinstall"
	"github.com/glueops/apiary-agent/internal/selfupgrade"
	"github.com/glueops/apiary-agent/internal/store"
	"github.com/spf13/cobra"
)

var (
	upgradeForce  bool
	upgradeDryRun bool
	upgradeUnit   string
)

// osRunner runs the two commands an upgrade needs: the staged binary's own
// `version`, and systemctl.
type osRunner struct{}

func (osRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Replace this agent with the one the control plane is serving",
	Long: "Downloads the agent binary this control plane ships with, verifies it against the\n" +
		"digest the control plane advertises, and installs it in place of this one.\n\n" +
		"The control plane is the source because a hypervisor in a managed estate has no route\n" +
		"to the internet. Without this, upgrading means copying a binary onto every machine by\n" +
		"hand, which is how a fleet ends up running four different versions.\n\n" +
		"The previous binary is kept alongside as .previous, and the new one is proved to run\n" +
		"before anything is replaced.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		st, err := store.Open(ctx, cfg.StatePath)
		if err != nil {
			return err
		}
		defer func() { _ = st.Close() }()

		id, err := st.Identity(ctx)
		if err != nil {
			return err
		}
		if id == nil {
			return fmt.Errorf("this host is not enrolled, so it has no credential to fetch a " +
				"binary with; run `apiary-agent enrol` first")
		}

		client, err := bee.NewClient(cfg.BaseURL, id.KeyPublic, id.KeySecret)
		if err != nil {
			return err
		}
		dist, _, err := client.FleetAPI.FleetBeeAgentDist(ctx).Execute()
		if err != nil {
			return fmt.Errorf("ask the control plane what it is serving: %w", err)
		}

		builds := dist.GetBuilds()
		if len(builds) == 0 {
			// Not an error. A control plane built without agent binaries is an
			// ordinary development build, and failing here would make `upgrade`
			// look broken on a server that is simply not distributing anything.
			cmd.Printf("this control plane is not serving any agent binaries; nothing to do\n")
			return nil
		}

		var want *struct {
			sha  string
			size int64
		}
		for _, b := range builds {
			if b.GetOs() == runtime.GOOS && b.GetArch() == runtime.GOARCH {
				want = &struct {
					sha  string
					size int64
				}{b.GetSha256(), b.GetSize()}
				break
			}
		}
		if want == nil {
			return fmt.Errorf("this control plane serves no agent for %s/%s",
				runtime.GOOS, runtime.GOARCH)
		}

		// Compared, and the comparison is deliberately not clever. Versions here
		// are release tags from one pipeline, so "different" is the only signal
		// worth acting on — ordering them would mean parsing a scheme that also
		// produces 1.2.4-SNAPSHOT-abc123, and getting that subtly wrong means an
		// agent that refuses the upgrade it needs.
		if dist.GetVersion() == Version && !upgradeForce {
			cmd.Printf("already running %s, which is what the control plane is serving\n", Version)
			return nil
		}

		cmd.Printf("this agent: %s\ncontrol plane: %s (%s/%s, %d bytes)\n",
			Version, dist.GetVersion(), runtime.GOOS, runtime.GOARCH, want.size)
		if upgradeDryRun {
			cmd.Printf("\n--dry-run: nothing downloaded or replaced\n")
			return nil
		}

		httpc := &http.Client{Timeout: 10 * time.Minute}
		body, err := selfupgrade.Download(ctx, httpc, cfg.BaseURL,
			id.KeyPublic, id.KeySecret, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return err
		}

		res, err := selfupgrade.Apply(ctx, osRunner{}, selfupgrade.Plan{
			Binary: body,
			SHA256: want.sha,
			Unit:   upgradeUnit,
		})
		if err != nil {
			return err
		}

		cmd.Printf("installed %s at %s\n", dist.GetVersion(), res.Target)
		cmd.Printf("previous binary kept at %s\n", res.Backup)
		if res.Restarted {
			cmd.Printf("restarted %s\n", upgradeUnit)
		} else {
			cmd.Printf("NOT restarted: this process is still the old binary, and the new one "+
				"takes over when %s next starts\n", hostinstall.UnitName)
		}
		return nil
	},
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false,
		"Reinstall even when the versions already match. For repairing a damaged binary.")
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false,
		"Report what would be installed and stop. Downloads nothing.")
	upgradeCmd.Flags().StringVar(&upgradeUnit, "unit", hostinstall.UnitName,
		"The systemd unit to restart once the binary is replaced. Empty skips the restart.")
	rootCmd.AddCommand(upgradeCmd)
}
