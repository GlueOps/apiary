package jobs

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/migrations"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var driftSchema = testsupport.SchemaName("apiary_migration_drift")

// TestGooseMigrationsMatchTheModels catches model/migration drift.
//
// This exists because of a specific escape. The whole suite migrates with
// AutoMigrate (internal/testsupport/schema.go), while PRODUCTION migrates with
// goose over internal/migrations. Nothing ran goose, so when three
// `observed_foreign_*` columns were added to Hypervisor and written by the
// capacity sync but never made it into a migration, every test stayed green and
// the tenant schema was simply wrong — the sync would have failed at runtime with
// "column does not exist" on the first real deployment.
//
// A green suite proving nothing about the schema an operator actually gets is the
// gap this closes. It applies the REAL migrations and then asks GORM, for every
// model it manages, whether each field has a column.
//
// It deliberately checks only that migrations are not MISSING anything the models
// declare. The reverse — a column no model mentions — is legitimate: dropped
// fields leave columns behind until a later migration removes them.
func TestGooseMigrationsMatchTheModels(t *testing.T) {
	dsn := testsupport.MustSkipWithoutDB(t)

	// A temporary DATABASE, not a schema. The migrations are schema-qualified to
	// "public" throughout (`ALTER TABLE "public"."hives" ...`), so they cannot be
	// applied into a search_path-scoped schema the way the rest of the suite
	// isolates itself. That is not a defect: production gives every tenant its own
	// database, so "public" is exactly where these run. Reproducing that is the
	// point — a drift test against a differently-shaped schema proves less.
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	adminDB, err := admin.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	// Registered FIRST so it runs LAST: t.Cleanup is LIFO, and the drop below
	// needs this connection still open. A `defer` here would close it before any
	// cleanup ran, and the drop would fail with "sql: database is closed" —
	// leaking a database per run, which is exactly what happened.
	t.Cleanup(func() { _ = adminDB.Close() })

	dbName := driftSchema // already unique per test binary invocation
	if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + dbName); err != nil {
		t.Fatalf("clear stale drift database: %v", err)
	}
	if _, err := adminDB.Exec("CREATE DATABASE " + dbName); err != nil {
		t.Skipf("cannot CREATE DATABASE (%v) — SKIPPING; migration drift was NOT checked", err)
	}
	t.Cleanup(func() {
		// Terminate stragglers first: a live connection blocks the drop and would
		// leave a database behind on every run.
		_, _ = adminDB.Exec(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", dbName)
		if _, err := adminDB.Exec("DROP DATABASE IF EXISTS " + dbName); err != nil {
			t.Logf("could not drop drift database %s: %v", dbName, err)
		}
	})

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	u.Path = "/" + dbName
	target, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open drift database: %v", err)
	}
	raw, err := target.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	// Registered LAST so it runs FIRST: an open connection to the target blocks
	// DROP DATABASE.
	t.Cleanup(func() { _ = raw.Close() })

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	goose.SetBaseFS(migrations.TenantFS)
	if err := goose.Up(raw, "tenant"); err != nil {
		t.Fatalf("goose up over the REAL migrations failed: %v", err)
	}

	// Every tenant model production is expected to carry. Keep in step with
	// tools/tenant/main.go — a model registered there and absent here is simply
	// unchecked.
	models := []any{
		&tenant.Hive{}, &tenant.Swarm{}, &tenant.Bee{}, &tenant.Hypervisor{},
		&tenant.Slot{}, &tenant.Pool{}, &tenant.Placement{},
		&tenant.Credential{}, &tenant.Estate{},
		&tenant.FirewallRuleset{}, &tenant.FirewallRule{}, &tenant.FirewallBinding{},
		&tenant.FirewallSet{}, &tenant.FirewallSetElement{}, &tenant.FirewallSnapshot{},
		&tenant.FleetTask{}, &tenant.FleetTaskLog{}, &tenant.SshKey{},
	}

	var missing []string
	m := target.Migrator()
	for _, model := range models {
		stmt := &gorm.Statement{DB: target}
		if err := stmt.Parse(model); err != nil {
			t.Fatalf("parse model %T: %v", model, err)
		}
		table := stmt.Schema.Table
		if !m.HasTable(model) {
			missing = append(missing, fmt.Sprintf("TABLE %s (model %T) is in no migration", table, model))
			continue
		}
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.DataType == "" {
				continue // association, or gorm:"-"
			}
			if !m.HasColumn(model, f.DBName) {
				missing = append(missing,
					fmt.Sprintf("%s.%s (model %T field %s)", table, f.DBName, model, f.Name))
			}
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("the models declare %d column(s) that NO migration creates, so production's schema "+
			"does not match the code (the suite is green only because everything else uses "+
			"AutoMigrate):\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}
