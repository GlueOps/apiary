// Package testsupport holds helpers shared by tests across packages. Nothing
// in the running binary imports it.
package testsupport

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// runID is unique to this test binary invocation.
//
// Fixed schema names were a real hazard, not a tidiness point: every DB-backed
// fixture in this tree creates its schema with CREATE SCHEMA IF NOT EXISTS and
// then TRUNCATEs the tables inside it. Two invocations pointed at the same
// TEST_DATABASE_URL therefore share the schema, and the second one's setup
// truncates the first one's fixtures mid-run. The failure is not a clean clash
// but arbitrary rows vanishing from under a running test — which reads as a
// flaky product bug rather than a test-harness problem. It happened on this
// machine during a parallel agent run.
//
// Random rather than the PID: PIDs are small and recycled, and a laptop and a
// CI runner sharing one database would collide on them eventually.
var runID = func() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic("testsupport: cannot generate a run id: " + err.Error())
	}
	return hex.EncodeToString(b)
}()

// SchemaName returns a schema name unique to this test binary invocation.
//
// Postgres identifiers truncate at 63 bytes, so a long base would silently
// collide again after truncation — hence the explicit cap rather than trusting
// callers to keep names short.
func SchemaName(base string) string {
	const maxIdent = 63
	suffix := "_" + runID
	if len(base)+len(suffix) > maxIdent {
		base = base[:maxIdent-len(suffix)]
	}
	return base + suffix
}

// DropSchemas removes schemas created by this run. Call it from TestMain after
// m.Run(), so a unique-per-run name does not turn a collision problem into
// unbounded schema sprawl in a shared database.
//
// Failures are reported rather than fatal: leaving a schema behind is untidy,
// but failing the suite over cleanup would turn a healthy run red.
func DropSchemas(names ...string) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		return
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "testsupport: cannot connect to drop test schemas: %v\n", err)
		return
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	for _, n := range names {
		if !isSafeIdent(n) {
			fmt.Fprintf(os.Stderr, "testsupport: refusing to drop unsafe schema name %q\n", n)
			continue
		}
		if err := db.Exec(`DROP SCHEMA IF EXISTS ` + n + ` CASCADE`).Error; err != nil {
			fmt.Fprintf(os.Stderr, "testsupport: could not drop schema %s: %v\n", n, err)
		}
	}
}

// isSafeIdent guards the string interpolation above. A schema name cannot be a
// bind parameter in DDL, so the only defence is refusing anything that is not a
// plain identifier — and this runs DROP ... CASCADE, which is not a statement to
// let an unexpected string reach.
func isSafeIdent(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r == '_':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return strings.HasPrefix(s, "apiary_")
}

// MustSkipWithoutDB skips loudly when no test database is configured.
//
// `go test` prints nothing for a skipped test without -v, so a suite that
// silently skips its only DB-backed coverage looks exactly like one that passed.
func MustSkipWithoutDB(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING; this test did NOT pass, it did not run")
	}
	assertLooksLikeATestDatabase(t, dsn)
	return dsn
}

// assertLooksLikeATestDatabase refuses a DSN whose database name does not say
// "test".
//
// This is a backstop, not the primary control — schema isolation is that — but
// it closes the one gap isolation cannot. The test and development databases now
// live on the SAME server, so the two DSNs differ by a suffix: apiary_test versus
// apiary. A single mistyped or copy-pasted character therefore points a suite
// that CREATEs and DROPs schemas at real data, and it has happened here: a run
// against the development database added a column outside goose and broke the
// next migration.
//
// Failing rather than skipping is deliberate. A skip is how the original problem
// hid — a suite that silently skipped its database coverage looked exactly like
// one that passed — so a misconfigured DSN has to be loud.
func assertLooksLikeATestDatabase(t *testing.T, dsn string) {
	t.Helper()

	name := databaseNameFromDSN(dsn)
	if name == "" {
		t.Fatalf("testsupport: cannot determine the database name from TEST_DATABASE_URL; "+
			"refusing to run schema-creating tests against an unidentified database (dsn shape: %q)",
			redactDSN(dsn))
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		t.Fatalf("testsupport: TEST_DATABASE_URL names database %q, which does not look like a test "+
			"database. This suite CREATEs and DROPs schemas; pointing it at a development or "+
			"production database destroys data. Use apiary_test (just test-db), or rename the "+
			"database so it contains \"test\".", name)
	}
}

// databaseNameFromDSN pulls the database out of a URL-style DSN. It returns ""
// rather than guessing when the shape is unfamiliar, so the caller refuses
// instead of proceeding on a bad parse.
func databaseNameFromDSN(dsn string) string {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || u.Scheme == "" {
		return ""
	}
	return strings.Trim(u.Path, "/")
}

// redactDSN renders a DSN safe to print. The refusal path above reports the DSN
// it rejected, and a DSN carries a password.
func redactDSN(dsn string) string {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "<unparseable>"
	}
	if u.User != nil {
		u.User = url.User("<redacted>")
	}
	return u.Redacted()
}

// OpenSchema returns a database handle whose search_path is a schema private to
// this test binary invocation, and migrates models into it.
//
// It exists so that no test package has to assemble this itself. Every part of
// it is a step that was, at some point, omitted somewhere in this tree: opening
// the DSN unscoped writes into public, which against a developer's own
// DATABASE_URL means the suite migrates and truncates real tables; and a schema
// name shared between runs means two invocations tread on each other's
// fixtures. A helper is the only version of this that stays correct as test
// files are added.
//
// Pass a schema from SchemaName, held in a package-level var, so that TestMain
// can hand the same name to DropSchemas afterwards.
func OpenSchema(t *testing.T, schema string, models ...any) *gorm.DB {
	t.Helper()
	return OpenSchemaWithParams(t, schema, nil, models...)
}

// OpenSchemaWithParams is OpenSchema with extra DSN query parameters — a session
// TimeZone, say, for a suite asserting that date arithmetic is done in UTC
// rather than in whatever zone the server happens to run in.
func OpenSchemaWithParams(t *testing.T, schema string, params map[string]string, models ...any) *gorm.DB {
	t.Helper()

	dsn := MustSkipWithoutDB(t)
	if !isSafeIdent(schema) {
		// Not a defensive nicety: the name is interpolated into DDL below,
		// because a schema name cannot be a bind parameter.
		t.Fatalf("testsupport: %q is not a usable schema name — it must be lower-case, "+
			"start with %q, and come from SchemaName", schema, "apiary_")
	}

	// A separate short-lived connection creates the schema, because the scoped
	// handle below cannot connect to a search_path that does not exist yet.
	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("testsupport: open test db: %v", err)
	}
	createErr := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema).Error
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}
	if createErr != nil {
		t.Fatalf("testsupport: create schema %s: %v", schema, createErr)
	}

	scoped := appendDSNParams(dsn, mergeParams(map[string]string{"search_path": schema}, params))
	db, err := gorm.Open(postgres.Open(scoped), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("testsupport: open scoped test db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("testsupport: migrate into %s: %v", schema, err)
		}
	}
	return db
}

func mergeParams(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	maps.Copy(out, base)
	maps.Copy(out, extra)
	return out
}

// appendDSNParams adds query parameters to a URL-style or keyword-style DSN.
//
// Sorted, so a given set of parameters always produces the same string and a
// failure is reproducible from the logged DSN.
func appendDSNParams(dsn string, params map[string]string) string {
	if len(params) == 0 {
		return dsn
	}
	keys := slices.Sorted(maps.Keys(params))

	var b strings.Builder
	b.WriteString(dsn)
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	for _, k := range keys {
		b.WriteString(sep)
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(params[k])
		sep = "&"
	}
	return b.String()
}
