package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/glueops/apiary-agent/internal/bee"
	"github.com/glueops/apiary-agent/internal/store"
	apiary "github.com/glueops/apiary/sdk/go"
	"github.com/spf13/cobra"
)

var (
	enrolToken     string
	enrolMachineID string
)

var enrolCmd = &cobra.Command{
	Use:   "enrol",
	Short: "Join this host to a hive",
	Long: "Redeems an enrolment token and stores the credential this host will use from then on.\n\n" +
		"Safe to re-run: enrolment is keyed on the machine id, so a rebuilt host rejoins as the same bee\n" +
		"rather than appearing as a second, permanently-offline one.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if enrolToken == "" {
			return fmt.Errorf("an enrolment token is required (--token)")
		}

		machineID := enrolMachineID
		if machineID == "" {
			// No override: read the host's own identifier, and fail if there is
			// none. A bee that invented one would enrol as a NEW host on every
			// restart and quietly multiply the fleet.
			machineID, err = bee.MachineID()
			if err != nil {
				return err
			}
		}
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("read hostname: %w", err)
		}

		st, err := store.Open(ctx, cfg.StatePath)
		if err != nil {
			return err
		}
		defer func() { _ = st.Close() }()

		// No credential yet — that is what the enrolment token is for.
		client, err := bee.NewClient(cfg.BaseURL, "", "")
		if err != nil {
			return err
		}

		body := *apiary.NewEnrolRequestBody(enrolToken, hostname, machineID)
		body.SetAgentVersion(Version)

		res, _, err := client.FleetAPI.FleetBeeEnrol(ctx).EnrolRequestBody(body).Execute()
		if err != nil {
			return fmt.Errorf("enrol: %w", err)
		}

		if err := st.SaveIdentity(ctx, store.Identity{
			BeeID:      res.GetBeeId(),
			OrgID:      res.GetOrganizationId(),
			KeyPublic:  res.GetKey(),
			KeySecret:  res.GetSecret(),
			BaseURL:    cfg.BaseURL,
			EnrolledAt: time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("store identity: %w", err)
		}

		// The secret is deliberately not printed. It is returned once and stored
		// once; echoing it puts a live credential into shell history and whatever
		// captured the terminal.
		//
		// No fleet is reported because none was joined: enrolment admits the host
		// to an organization, and joining a fleet is a later task.
		cmd.Printf("enrolled as bee %s in organization %s; not yet in a fleet\n",
			res.GetBeeId(), res.GetOrganizationId())
		return nil
	},
}

func init() {
	enrolCmd.Flags().StringVar(&enrolToken, "token", "", "Enrolment token minted by platform staff")
	enrolCmd.Flags().StringVar(&enrolMachineID, "machine-id", "",
		"Override the host identifier. For development on a machine with no /etc/machine-id; "+
			"on a real hypervisor let it be read, so a rebuild rejoins as the same bee.")
	rootCmd.AddCommand(enrolCmd)
}
