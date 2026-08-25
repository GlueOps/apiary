package jobs

import (
	"context"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
)

type captureLog struct {
	mu    sync.Mutex
	stmts []string
}

func (l *captureLog) record(q string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stmts = append(l.stmts, q)
}

func (l *captureLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.stmts))
	copy(out, l.stmts)
	return out
}

type captureDriver struct {
	log        *captureLog
	roleExists bool
}

func (d *captureDriver) Open(string) (driver.Conn, error) {
	return &captureConn{log: d.log, roleExists: d.roleExists}, nil
}

type captureConn struct {
	log        *captureLog
	roleExists bool
}

func (c *captureConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("capture driver: Prepare is not supported; the code under test must use ExecContext/QueryContext")
}
func (c *captureConn) Close() error { return nil }
func (c *captureConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("capture driver: no transactions")
}

func (c *captureConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.log.record(query)
	return driver.RowsAffected(0), nil
}

func (c *captureConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.log.record(query)
	return &captureRows{val: c.roleExists}, nil
}

type captureRows struct {
	val  bool
	done bool
}

func (r *captureRows) Columns() []string { return []string{"exists"} }
func (r *captureRows) Close() error      { return nil }
func (r *captureRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.val
	return nil
}

var captureDriverSeq int

func openCapturing(t *testing.T, roleExists bool) (*sql.DB, *captureLog) {
	t.Helper()
	log := &captureLog{}
	captureDriverSeq++
	name := "apiary-capture-" + strconv.Itoa(captureDriverSeq)
	sql.Register(name, &captureDriver{log: log, roleExists: roleExists})
	db, err := sql.Open(name, "capture")
	if err != nil {
		t.Fatalf("open capturing db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, log
}

func TestCreateTenantRoleNeverPutsThePasswordInStatementText(t *testing.T) {
	for _, tc := range []struct {
		name       string
		roleExists bool
		wantStmt   string
	}{
		{"create", false, "CREATE ROLE"},
		{"rotate", true, "ALTER ROLE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adminDB, adminLog := openCapturing(t, tc.roleExists)
			tenantDB, _ := openCapturing(t, tc.roleExists)

			const password = "Sup3rSecretTenantPassword=="
			if err := createTenantRole(t.Context(), adminDB, tenantDB, "tenant_role_x", password, "tenant_x"); err != nil {
				t.Fatalf("createTenantRole: %v", err)
			}

			stmts := adminLog.all()
			var passwordStmt string
			for _, s := range stmts {
				if strings.Contains(s, password) {
					t.Fatalf("the plaintext password appears in statement text: %q", s)
				}
				if strings.Contains(s, "PASSWORD ") {
					passwordStmt = s
				}
			}
			if passwordStmt == "" {
				t.Fatalf("no PASSWORD statement was issued; statements were %q", stmts)
			}
			if !strings.Contains(passwordStmt, tc.wantStmt) {
				t.Fatalf("expected a %s statement, got %q", tc.wantStmt, passwordStmt)
			}
			if !strings.Contains(passwordStmt, "'SCRAM-SHA-256$4096:") {
				t.Fatalf("expected a pre-computed SCRAM-SHA-256 verifier, got %q", passwordStmt)
			}
		})
	}
}

func TestScramSHA256VerifierMatchesRFC5802(t *testing.T) {
	const password = "correct horse battery staple"

	got, err := scramSHA256Verifier(password)
	if err != nil {
		t.Fatalf("scramSHA256Verifier: %v", err)
	}

	rest, ok := strings.CutPrefix(got, "SCRAM-SHA-256$")
	if !ok {
		t.Fatalf("missing SCRAM-SHA-256 prefix: %q", got)
	}
	head, keys, ok := strings.Cut(rest, "$")
	if !ok {
		t.Fatalf("missing key section: %q", got)
	}
	iterStr, saltB64, ok := strings.Cut(head, ":")
	if !ok {
		t.Fatalf("missing salt: %q", got)
	}
	storedB64, serverB64, ok := strings.Cut(keys, ":")
	if !ok {
		t.Fatalf("missing server key: %q", got)
	}

	iter, err := strconv.Atoi(iterStr)
	if err != nil || iter != scramIterations {
		t.Fatalf("iterations = %q, want %d", iterStr, scramIterations)
	}

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		t.Fatalf("salt is not base64: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("salt length = %d, want 16", len(salt))
	}

	salted, err := pbkdf2.Key(sha256.New, password, salt, iter, sha256.Size)
	if err != nil {
		t.Fatalf("pbkdf2: %v", err)
	}
	ck := hmac.New(sha256.New, salted)
	ck.Write([]byte("Client Key"))
	stored := sha256.Sum256(ck.Sum(nil))
	sk := hmac.New(sha256.New, salted)
	sk.Write([]byte("Server Key"))

	if storedB64 != base64.StdEncoding.EncodeToString(stored[:]) {
		t.Fatal("StoredKey does not match SHA256(HMAC(SaltedPassword, \"Client Key\"))")
	}
	if serverB64 != base64.StdEncoding.EncodeToString(sk.Sum(nil)) {
		t.Fatal("ServerKey does not match HMAC(SaltedPassword, \"Server Key\")")
	}
	if strings.Contains(got, password) {
		t.Fatal("the verifier contains the plaintext password")
	}
}

func TestScramSHA256VerifierIsSaltedPerCall(t *testing.T) {
	a, err := scramSHA256Verifier("same-password")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	b, err := scramSHA256Verifier("same-password")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if a == b {
		t.Fatal("two verifiers for the same password are identical; the salt is not random")
	}
	if _, err := scramSHA256Verifier(""); err == nil {
		t.Fatal("expected an error for an empty password")
	}
}

func TestCreateTenantRoleProducesALoginRoleAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the SCRAM verifier round-trip; " +
			"this test does NOT pass, it did not run")
	}

	for _, name := range sql.Drivers() {
		if name == "postgres" {
			t.Fatal(`a driver named "postgres" is registered; this test's premise (and the F6 fix) needs re-checking`)
		}
	}

	adminDB, err := sql.Open(pgDriverName, dsn)
	if err != nil {
		t.Fatalf("open admin db with driver %q: %v", pgDriverName, err)
	}
	defer adminDB.Close()
	if err := adminDB.PingContext(t.Context()); err != nil {
		t.Fatalf("ping test db: %v", err)
	}

	role := "tenant_role_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	pwBytes, err := utils.RandomBytes(32)
	if err != nil {
		t.Fatalf("random password: %v", err)
	}
	password := utils.EncodeB64(pwBytes)

	var dbName string
	if err := adminDB.QueryRowContext(t.Context(), `SELECT current_database()`).Scan(&dbName); err != nil {
		t.Fatalf("current_database: %v", err)
	}

	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), `DROP ROLE IF EXISTS `+quoteIdentifier(role))
	})

	if err := createTenantRole(t.Context(), adminDB, adminDB, role, password, dbName); err != nil {
		t.Fatalf("createTenantRole: %v", err)
	}

	var stored string
	if err := adminDB.QueryRowContext(t.Context(),
		`SELECT rolpassword FROM pg_authid WHERE rolname = $1`, role,
	).Scan(&stored); err != nil {
		t.Fatalf("read rolpassword: %v", err)
	}
	if !strings.HasPrefix(stored, "SCRAM-SHA-256$") {
		t.Fatalf("stored secret is not a SCRAM verifier: %q", stored)
	}
	if strings.Contains(stored, password) {
		t.Fatal("the plaintext password is stored in pg_authid")
	}

	roleURL, err := tenantRoleDBURL(dsn, dbName, role, password)
	if err != nil {
		t.Fatalf("tenantRoleDBURL: %v", err)
	}
	roleDB, err := sql.Open(pgDriverName, roleURL)
	if err != nil {
		t.Fatalf("open as tenant role: %v", err)
	}
	defer roleDB.Close()
	if err := roleDB.PingContext(t.Context()); err != nil {
		t.Fatalf("the tenant role cannot authenticate with the plaintext password; the SCRAM verifier is wrong: %v", err)
	}
}
