package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/config"
	"github.com/spf13/cobra"
)

const psqlSessionTimeout = 72 * time.Hour

var pagerMode string

var dbPsqlCmd = &cobra.Command{
	Use:   "psql",
	Short: "Open a psql session to the app database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.DatabaseURL == "" {
			return errors.New("database.url is empty")
		}

		psql := "psql"
		if runtime.GOOS == "windows" {
			psql = "psql.exe"
		}

		ctx, cancel := context.WithTimeout(context.Background(), psqlSessionTimeout)
		defer cancel()

		sanitized, password, err := splitDSNSecret(cfg.DatabaseURL)
		if err != nil {
			return err
		}

		psqlArgs := []string{"-P", "pager=" + pagerMode, sanitized}
		psqlCmd := exec.CommandContext(ctx, psql, psqlArgs...)

		psqlCmd.Env = os.Environ()
		if password != "" {
			psqlCmd.Env = append(psqlCmd.Env, "PGPASSWORD="+password)
		}
		psqlCmd.Stdin = os.Stdin
		psqlCmd.Stdout = os.Stdout
		psqlCmd.Stderr = os.Stderr

		fmt.Println("Launching psql…")
		return psqlCmd.Run()
	},
}

func splitDSNSecret(raw string) (sanitized, password string, err error) {
	u, perr := url.Parse(strings.TrimSpace(raw))
	if perr != nil {

		_ = perr
		return "", "", errors.New("database.url is not a valid URL")
	}
	switch u.Scheme {
	case "postgres", "postgresql":
	default:
		return "", "", errors.New(
			"database.url must be a postgres:// or postgresql:// URL for `apiary psql`")
	}
	if u.User != nil {
		if pw, ok := u.User.Password(); ok {

			password = pw
			u.User = url.User(u.User.Username())
		}
	}

	return u.String(), password, nil
}

func init() {
	dbPsqlCmd.Flags().StringVar(&pagerMode, "pager", "off", "psql pager mode: off, on, always")

	rootCmd.AddCommand(dbPsqlCmd)
}
