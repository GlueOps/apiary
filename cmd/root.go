package cmd

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/spf13/cobra"
)

// loadDotEnv makes .env visible to every subcommand and, critically, to every
// process a subcommand execs.
//
// It used to happen only inside config.Load(), which meant a command that never
// loaded config never saw .env. `generate migrations` is exactly that: it execs
// `atlas`, and atlas.hcl resolves its dev database with
// getenv("ATLAS_DEV_URL"). With the variable absent atlas does not fail — it
// falls back to spinning up docker://postgres/18/dev, which then times out
// after ~75 seconds against a Docker network it cannot reach. The error names a
// dial timeout to a random IP and says nothing about the missing variable, so
// the setting looks broken when it is merely unread.
//
// godotenv.Load does not overwrite variables already present, so a real
// environment still wins over the file — which is what a deployment needs.
func loadDotEnv() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: could not read .env: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "apiary",
	Short: "Compute Control Plane",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		loadDotEnv()
	},
	Long: "APIary is GlueOps' provider-agnostic compute API, providing a common interface for provisioning and managing compute across AWS, Hetzner, Proxmox, and other infrastructure providers",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
