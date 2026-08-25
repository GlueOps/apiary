package cmd

import (
	"fmt"

	"github.com/glueops/apiary/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\nCommit: %s\nDate: %s\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
