package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"io/fs"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/migrations"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const pgDriverName = "pgx"

var (
	migrateDBUrl string
	migrateScope string
	migrateOrg   string
	migrateSteps int
	migrateRiver bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigrations(cmd.Context(), gooseAction{kind: "up"})
	},
}

type gooseAction struct {
	kind    string
	version int64
}

func newMigrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply migrations (all by default, or --steps via up-by-one)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "up"})
		},
	}
}

func newMigrateUpToCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "up-to VERSION",
		Short: "Migrate up to a specific VERSION",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := parseVersion(args[0])
			if err != nil {
				return err
			}
			return runMigrations(cmd.Context(), gooseAction{kind: "up-to", version: v})
		},
	}
	return c
}

func newMigrateDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback migrations (one by default, or --steps N)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "down"})
		},
	}
}

func newMigrateDownToCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down-to VERSION",
		Short: "Rollback migrations to a specific VERSION",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := parseVersion(args[0])
			if err != nil {
				return err
			}
			return runMigrations(cmd.Context(), gooseAction{kind: "down-to", version: v})
		},
	}
}

func newMigrateRedoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "redo",
		Short: "Re-run the latest migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "redo"})
		},
	}
}

func newMigrateResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Rollback ALL migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "reset"})
		},
	}
}

func newMigrateStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "status"})
		},
	}
}

func newMigrateVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print current migration version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), gooseAction{kind: "version"})
		},
	}
}

func runMigrations(ctx context.Context, action gooseAction) error {
	scope := strings.ToLower(strings.TrimSpace(migrateScope))
	if scope == "" {
		scope = "all"
	}
	if scope != "control" && scope != "tenants" && scope != "all" {
		return fmt.Errorf("invalid --scope %q (must be control|tenants|all)", migrateScope)
	}

	dbURL, err := resolveControlDBURL()
	if err != nil {
		return err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	controlSQL, err := sql.Open(pgDriverName, dbURL)
	if err != nil {
		return err
	}
	defer controlSQL.Close()

	var orgs []control.Organization
	if scope != "control" && (action.kind == "down" || action.kind == "down-to" || action.kind == "reset") {
		orgs = fetchOrgsBestEffort(controlSQL, migrateOrg)
	}

	if scope == "control" || scope == "all" {
		log.Printf("Control DB: goose %s", action.kind)
		if err := runGooseAction(controlSQL, migrations.ControlFS, "control", action); err != nil {
			return fmt.Errorf("control migrations failed: %w", err)
		}

		if migrateRiver {
			if err := runRiver(ctx, controlSQL, action); err != nil {
				return fmt.Errorf("river migrations failed: %w", err)
			}
		}
	}

	if scope == "tenants" || scope == "all" {

		if orgs == nil {
			orgs = fetchOrgsBestEffort(controlSQL, migrateOrg)
		}
		if len(orgs) == 0 {
			log.Println("No tenants found (or control schema not available).")
			return nil
		}

		for _, org := range orgs {
			if org.ConnectionString == "" {
				log.Printf("Skipping org %s (%s): empty connection_string", org.Name, org.ID)
				continue
			}

			dsn, err := decryptOrgDSN(org)
			if err != nil {
				log.Printf("Skipping org %s (%s): %v", org.Name, org.ID, err)
				continue
			}

			log.Printf("Tenant DB [%s / %s]: goose %s", org.Name, org.ID, action.kind)

			// Heal the schema grant before migrating.
			//
			// Tenant roles provisioned before CREATE was granted can run their
			// existing tables but cannot add one, so every migration that
			// introduces a table fails for them with "permission denied for
			// schema public" — and it fails for ALL of them at once, which at any
			// scale is a fleet-wide stall on a one-word grant.
			//
			// Done from the admin connection because a role cannot grant this to
			// itself, and here rather than in a migration because a migration runs
			// as the very role that lacks the privilege.
			if err := ensureTenantSchemaGrant(ctx, dsn); err != nil {
				log.Printf("Could not ensure schema grant for %s: %v", org.ID, err)
			}

			tenantSQL, err := sql.Open(pgDriverName, dsn)
			if err != nil {
				log.Printf("Failed to connect tenant DB %s: %v", org.ID, err)
				continue
			}

			if err := runGooseAction(tenantSQL, migrations.TenantFS, "tenant", action); err != nil {
				log.Printf("Tenant migrations failed for %s: %v", org.ID, err)
			}
			_ = tenantSQL.Close()
		}
	}
	return nil
}

func runGooseAction(db *sql.DB, base fs.FS, dir string, action gooseAction) error {
	goose.SetBaseFS(base)

	switch action.kind {
	case "up":
		if migrateSteps > 0 {
			for i := 0; i < migrateSteps; i++ {
				if err := goose.UpByOne(db, dir); err != nil {
					return err
				}
			}
			return nil
		}
		return goose.Up(db, dir)

	case "up-to":
		return goose.UpTo(db, dir, action.version)

	case "down":
		steps := migrateSteps
		if steps <= 0 {
			steps = 1
		}
		for i := 0; i < steps; i++ {
			if err := goose.Down(db, dir); err != nil {
				return err
			}
		}
		return nil

	case "down-to":
		return goose.DownTo(db, dir, action.version)

	case "redo":
		return goose.Redo(db, dir)

	case "reset":
		return goose.Reset(db, dir)

	case "status":
		return goose.Status(db, dir)

	case "version":
		return goose.Version(db, dir)

	default:
		return fmt.Errorf("unknown goose action: %s", action.kind)
	}
}

func runRiver(ctx context.Context, controlSQL *sql.DB, action gooseAction) error {

	switch action.kind {
	case "status", "version":
		return nil
	}

	log.Printf("Control DB: river migrate (%s)", riverDirectionFor(action))
	m, err := rivermigrate.New(riverdatabasesql.New(controlSQL), nil)
	if err != nil {
		return err
	}

	dir := riverDirectionFor(action)

	var opts *rivermigrate.MigrateOpts
	if dir == rivermigrate.DirectionDown {
		opts = &rivermigrate.MigrateOpts{}
		if action.kind == "reset" {
			opts.TargetVersion = -1
		} else if migrateSteps > 0 {
			opts.MaxSteps = migrateSteps
		}
	}

	_, err = m.Migrate(ctx, dir, opts)
	return err
}

func riverDirectionFor(action gooseAction) rivermigrate.Direction {
	switch action.kind {
	case "down", "down-to", "reset":
		return rivermigrate.DirectionDown
	default:
		return rivermigrate.DirectionUp
	}
}

func resolveControlDBURL() (string, error) {
	if migrateDBUrl != "" {
		return migrateDBUrl, nil
	}
	cfg, err := config.Load()
	if err == nil && cfg.DatabaseURL != "" {
		return cfg.DatabaseURL, nil
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v, nil
	}
	return "", errors.New("DATABASE_URL is required (via --db-url, config, or env var)")
}

func fetchOrgsBestEffort(controlSQL *sql.DB, orgFilter string) []control.Organization {
	controlGorm, err := gorm.Open(postgres.New(postgres.Config{Conn: controlSQL}), &gorm.Config{})
	if err != nil {
		log.Printf("Could not init gorm for control DB: %v", err)
		return nil
	}

	var orgs []control.Organization

	q := controlGorm.Model(&control.Organization{}).
		Select("id", "name", "connection_string", "connection_string_iv", "connection_string_tag")
	if orgFilter != "" {
		q = q.Where("id::text = ? OR name = ?", orgFilter, orgFilter)
	}
	if err := q.Find(&orgs).Error; err != nil {
		log.Printf("Could not fetch orgs: %v", err)
		return nil
	}
	return orgs
}

func parseVersion(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid VERSION %q", s)
	}
	return v, nil
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.PersistentFlags().StringVar(&migrateDBUrl, "db-url", "", "Control DB URL (or DATABASE_URL / config fallback)")
	migrateCmd.PersistentFlags().StringVar(&migrateScope, "scope", "all", "Migration scope: control|tenants|all")
	migrateCmd.PersistentFlags().StringVar(&migrateOrg, "org", "", "Limit tenant migrations to a single org (id or name)")
	migrateCmd.PersistentFlags().IntVar(&migrateSteps, "steps", 0, "Number of steps (for up-by-one/down). 0 means default behavior.")
	migrateCmd.PersistentFlags().BoolVar(&migrateRiver, "river", true, "Also migrate River tables in the control DB")

	migrateCmd.AddCommand(newMigrateUpCmd())
	migrateCmd.AddCommand(newMigrateUpToCmd())
	migrateCmd.AddCommand(newMigrateDownCmd())
	migrateCmd.AddCommand(newMigrateDownToCmd())
	migrateCmd.AddCommand(newMigrateRedoCmd())
	migrateCmd.AddCommand(newMigrateResetCmd())
	migrateCmd.AddCommand(newMigrateStatusCmd())
	migrateCmd.AddCommand(newMigrateVersionCmd())
}

func decryptOrgDSN(org control.Organization) (string, error) {
	if org.ConnectionStringIV == "" || org.ConnectionStringTag == "" {
		return "", fmt.Errorf(
			"connection_string is not encrypted; re-provision the tenant to store an " +
				"encrypted, tenant-scoped DSN (F6)")
	}

	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	dsnKey, err := utils.DeriveKeyFromBase64(cfg.EncryptionMasterKey, utils.KeyPurposeTenantDSN)
	if err != nil {
		return "", err
	}

	ct, err1 := utils.DecodeB64(org.ConnectionString)
	iv, err2 := utils.DecodeB64(org.ConnectionStringIV)
	tag, err3 := utils.DecodeB64(org.ConnectionStringTag)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("could not decode connection string components")
	}

	plain, err := utils.DecryptAESGCMWithAAD(ct, dsnKey, iv, tag, []byte(org.ID.String()))
	if err != nil {
		return "", fmt.Errorf("could not decrypt connection string: %w", err)
	}
	return string(plain), nil
}

// ensureTenantSchemaGrant gives a tenant role CREATE on its own database's
// public schema, idempotently.
//
// It connects with the ADMIN credentials to the TENANT database: the grant is
// per-database, so it cannot be issued from the control database, and the role
// that needs it cannot issue it to itself.
//
// A failure is logged and not fatal. If the grant was already there the
// migration proceeds regardless, and if it genuinely could not be issued the
// migration fails immediately afterwards with a clearer message than this could
// produce.
func ensureTenantSchemaGrant(ctx context.Context, tenantDSN string) error {
	role, dbName, err := roleAndDatabaseFrom(tenantDSN)
	if err != nil {
		return err
	}
	adminURL, err := resolveControlDBURL()
	if err != nil {
		return err
	}
	adminOnTenant, err := swapDatabase(adminURL, dbName)
	if err != nil {
		return err
	}

	db, err := sql.Open(pgDriverName, adminOnTenant)
	if err != nil {
		return fmt.Errorf("open %s as admin: %w", dbName, err)
	}
	defer func() { _ = db.Close() }()

	// The role name comes from a DSN this process just decrypted, not from a
	// request — but it is interpolated into DDL, where an identifier cannot be a
	// bind parameter, so it is quoted rather than trusted.
	if _, err := db.ExecContext(ctx,
		`GRANT CREATE ON SCHEMA public TO `+pq.QuoteIdentifier(role)); err != nil {
		return fmt.Errorf("grant create on %s: %w", dbName, err)
	}
	return nil
}

// roleAndDatabaseFrom pulls the connecting role and database out of a tenant DSN.
func roleAndDatabaseFrom(dsn string) (role, database string, err error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "", "", fmt.Errorf("tenant dsn is not a url")
	}
	if u.User == nil || u.User.Username() == "" {
		return "", "", fmt.Errorf("tenant dsn names no role")
	}
	db := strings.Trim(u.Path, "/")
	if db == "" {
		return "", "", fmt.Errorf("tenant dsn names no database")
	}
	return u.User.Username(), db, nil
}

// swapDatabase points a DSN at a different database, keeping its credentials.
func swapDatabase(dsn, database string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "", fmt.Errorf("admin dsn is not a url")
	}
	u.Path = "/" + database
	u.RawPath = ""
	return u.String(), nil
}
