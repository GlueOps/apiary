package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var showConfigOutput string

var showConfigCmd = &cobra.Command{
	Use:   "show-config",
	Short: "Show the current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		out := cmd.OutOrStdout()
		redacted := cfg.Redacted()

		switch strings.ToLower(showConfigOutput) {
		case "yaml", "yml":
			enc := yaml.NewEncoder(out)
			defer enc.Close()

			if err := enc.Encode(redacted); err != nil {
				return fmt.Errorf("failed to encode config as yaml: %w", err)
			}
			return nil

		case "json":
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)

			if err := enc.Encode(redacted); err != nil {
				return fmt.Errorf("failed to encode config as json: %w", err)
			}
			return nil

		default:
			return fmt.Errorf("unsupported output format %q (expected: yaml or json)", showConfigOutput)
		}
	},
}

func init() {
	showConfigCmd.Flags().StringVarP(&showConfigOutput, "output", "o", "yaml", "Output format: yaml or json")
	rootCmd.AddCommand(showConfigCmd)
}
