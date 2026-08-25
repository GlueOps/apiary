package database

import (
	"bytes"
	"context"
	"fmt"
	"github.com/glueops/apiary/internal/testsupport"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testSchema = testsupport.SchemaName("apiary_tenantmgr_test")

func testMasterKeyB64(t *testing.T) string {
	t.Helper()
	raw, err := utils.RandomBytes(utils.MasterKeyLen)
	if err != nil {
		t.Fatalf("random master key: %v", err)
	}
	return utils.EncodeB64(raw)
}

func testPool() TenantPoolConfig {
	return TenantPoolConfig{
		MaxSize:         2,
		MaxConns:        2,
		MaxIdleConns:    1,
		ConnMaxIdleTime: time.Minute,
		ConnMaxLifetime: time.Minute,
		EvictGrace:      0,
	}
}

func testControlDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the tenant pool suite; " +
			"this test does NOT pass, it did not run")
	}

	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + testSchema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	scoped := dsn + sep + "search_path=" + testSchema

	db, err := gorm.Open(postgres.Open(scoped), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open scoped test db: %v", err)
	}
	if err := db.AutoMigrate(&control.Organization{}); err != nil {
		t.Fatalf("migrate organizations: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db, scoped
}

func seedOrg(t *testing.T, db *gorm.DB, masterKeyB64, tenantDSN string) uuid.UUID {
	t.Helper()

	master, err := utils.MasterKeyFromBase64(masterKeyB64)
	if err != nil {
		t.Fatalf("master key: %v", err)
	}
	dekKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("derive dek key: %v", err)
	}
	dsnKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDSN)
	if err != nil {
		t.Fatalf("derive dsn key: %v", err)
	}

	org := control.Organization{Name: "pool-test-" + uuid.NewString()}
	if err := db.Create(&org).Error; err != nil {
		t.Fatalf("create org: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&control.Organization{}, "id = ?", org.ID) })

	aad := []byte(org.ID.String())

	dek, err := utils.RandomBytes(32)
	if err != nil {
		t.Fatalf("random dek: %v", err)
	}
	ct, iv, tag, err := utils.EncryptAESGCMWithAAD(dek, dekKey, aad)
	if err != nil {
		t.Fatalf("wrap dek: %v", err)
	}
	dsnCT, dsnIV, dsnTag, err := utils.EncryptAESGCMWithAAD([]byte(tenantDSN), dsnKey, aad)
	if err != nil {
		t.Fatalf("seal dsn: %v", err)
	}

	if err := db.Model(&org).Updates(map[string]any{
		"encrypted_tenant_key":  utils.EncodeB64(ct),
		"tenant_key_iv":         utils.EncodeB64(iv),
		"tenant_key_tag":        utils.EncodeB64(tag),
		"connection_string":     utils.EncodeB64(dsnCT),
		"connection_string_iv":  utils.EncodeB64(dsnIV),
		"connection_string_tag": utils.EncodeB64(dsnTag),
		"status":                "active",
	}).Error; err != nil {
		t.Fatalf("seal org: %v", err)
	}
	return org.ID
}

func TestNewTenantManagerRefusesAnUnboundedPool(t *testing.T) {
	key := testMasterKeyB64(t)
	db := &gorm.DB{}

	cases := map[string]TenantPoolConfig{
		"zero max size":      {MaxSize: 0, MaxConns: 2, MaxIdleConns: 1},
		"negative max size":  {MaxSize: -1, MaxConns: 2, MaxIdleConns: 1},
		"zero max conns":     {MaxSize: 4, MaxConns: 0, MaxIdleConns: 0},
		"idle exceeds open":  {MaxSize: 4, MaxConns: 2, MaxIdleConns: 3},
		"negative idle":      {MaxSize: 4, MaxConns: 2, MaxIdleConns: -1},
		"negative grace":     {MaxSize: 4, MaxConns: 2, MaxIdleConns: 1, EvictGrace: -time.Second},
		"negative lifetime":  {MaxSize: 4, MaxConns: 2, MaxIdleConns: 1, ConnMaxLifetime: -time.Second},
		"negative idle time": {MaxSize: 4, MaxConns: 2, MaxIdleConns: 1, ConnMaxIdleTime: -time.Second},
	}
	for name, cfg := range cases {
		if _, err := NewTenantManager(db, key, cfg); err == nil {
			t.Fatalf("%s: expected NewTenantManager to refuse the config, got nil", name)
		}
	}

	if _, err := NewTenantManager(db, "not-base64!!", testPool()); err == nil {
		t.Fatal("expected an error for a non-base64 master key")
	}
	if _, err := NewTenantManager(db, utils.EncodeB64([]byte("short")), testPool()); err == nil {
		t.Fatal("expected an error for a master key that is not 32 bytes")
	}
	if _, err := NewTenantManager(nil, key, testPool()); err == nil {
		t.Fatal("expected an error for a nil control db")
	}
}

func TestTenantManagerDerivesDistinctSubkeys(t *testing.T) {
	keyB64 := testMasterKeyB64(t)
	m, err := NewTenantManager(&gorm.DB{}, keyB64, testPool())
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}
	master, err := utils.MasterKeyFromBase64(keyB64)
	if err != nil {
		t.Fatalf("master key: %v", err)
	}
	if bytes.Equal(m.dekKey, master) || bytes.Equal(m.dsnKey, master) {
		t.Fatal("a subkey is the master key verbatim; there is no domain separation")
	}
	if bytes.Equal(m.dekKey, m.dsnKey) {
		t.Fatal("the DEK and DSN subkeys are identical")
	}
}

func TestGetTenantContextRejectsANonUUIDOrgID(t *testing.T) {
	m, err := NewTenantManager(&gorm.DB{}, testMasterKeyB64(t), testPool())
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}
	for _, bad := range []string{"", "   ", "not-a-uuid", "'; DROP TABLE organizations; --"} {
		if _, err := m.GetTenantContext(t.Context(), bad); err == nil {
			t.Fatalf("expected an error for org id %q", bad)
		}
	}
	if m.Len() != 0 {
		t.Fatalf("rejected ids created %d cache entries", m.Len())
	}
}

func TestTenantPoolIsCappedAndEvictedPoolsAreClosed(t *testing.T) {
	controlDB, dsn := testControlDB(t)
	keyB64 := testMasterKeyB64(t)

	m, err := NewTenantManager(controlDB, keyB64, testPool())
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	ids := []uuid.UUID{
		seedOrg(t, controlDB, keyB64, dsn),
		seedOrg(t, controlDB, keyB64, dsn),
		seedOrg(t, controlDB, keyB64, dsn),
	}

	first, err := m.GetTenantContext(t.Context(), ids[0].String())
	if err != nil {
		t.Fatalf("get first tenant: %v", err)
	}
	firstSQL, err := first.DB.DB()
	if err != nil {
		t.Fatalf("first sql db: %v", err)
	}

	if got := firstSQL.Stats().MaxOpenConnections; got != testPool().MaxConns {
		t.Fatalf("tenant pool MaxOpenConnections = %d, want %d (an unbounded pool can exhaust max_connections)", got, testPool().MaxConns)
	}

	for _, id := range ids[1:] {
		if _, err := m.GetTenantContext(t.Context(), id.String()); err != nil {
			t.Fatalf("get tenant %s: %v", id, err)
		}
	}

	if m.Len() != testPool().MaxSize {
		t.Fatalf("cached pools = %d, want the configured cap of %d", m.Len(), testPool().MaxSize)
	}

	if err := firstSQL.PingContext(t.Context()); err == nil {
		t.Fatal("the evicted tenant pool is still open; eviction without closing is a leak with a size cap bolted on")
	}
}

func TestGetTenantContextIsSingleFlighted(t *testing.T) {
	controlDB, dsn := testControlDB(t)
	keyB64 := testMasterKeyB64(t)

	pool := testPool()
	pool.MaxSize = 8
	m, err := NewTenantManager(controlDB, keyB64, pool)
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	id := seedOrg(t, controlDB, keyB64, dsn).String()

	const goroutines = 32
	results := make([]*TenantContext, goroutines)
	errs := make([]error, goroutines)

	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	for i := range goroutines {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			results[i], errs[i] = m.GetTenantContext(context.Background(), id)
		}()
	}
	start.Done()
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i, got := range results {
		if got != results[0] {
			t.Fatalf("goroutine %d got a different *TenantContext; %d concurrent cold requests opened more than one pool and leaked the losers", i, goroutines)
		}
	}
	if m.Len() != 1 {
		t.Fatalf("cached pools = %d, want 1", m.Len())
	}
}

func TestEvictAndCloseReleaseTenantPools(t *testing.T) {
	controlDB, dsn := testControlDB(t)
	keyB64 := testMasterKeyB64(t)

	m, err := NewTenantManager(controlDB, keyB64, testPool())
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}

	id := seedOrg(t, controlDB, keyB64, dsn).String()
	tCtx, err := m.GetTenantContext(t.Context(), id)
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	sqlDB, err := tCtx.DB.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}

	m.Evict(id)
	if m.Len() != 0 {
		t.Fatalf("cached pools after Evict = %d, want 0", m.Len())
	}
	if err := sqlDB.PingContext(t.Context()); err == nil {
		t.Fatal("Evict left the pool open; a pool pointing at a dropped database would survive")
	}

	other := seedOrg(t, controlDB, keyB64, dsn).String()
	otherCtx, err := m.GetTenantContext(t.Context(), other)
	if err != nil {
		t.Fatalf("get second tenant: %v", err)
	}
	otherSQL, err := otherCtx.DB.DB()
	if err != nil {
		t.Fatalf("second sql db: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := otherSQL.PingContext(t.Context()); err == nil {
		t.Fatal("Close left a tenant pool open")
	}
	if m.Len() != 0 {
		t.Fatalf("cached pools after Close = %d, want 0", m.Len())
	}
}

func TestTenantKeySealedUnderTheMasterKeyOrTheWrongOrgIsRejected(t *testing.T) {
	controlDB, dsn := testControlDB(t)
	keyB64 := testMasterKeyB64(t)
	master, err := utils.MasterKeyFromBase64(keyB64)
	if err != nil {
		t.Fatalf("master key: %v", err)
	}

	m, err := NewTenantManager(controlDB, keyB64, testPool())
	if err != nil {
		t.Fatalf("NewTenantManager: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	legacy := seedOrg(t, controlDB, keyB64, dsn)
	dek, err := utils.RandomBytes(32)
	if err != nil {
		t.Fatalf("random dek: %v", err)
	}
	ct, iv, tag, err := utils.EncryptAESGCM(dek, master)
	if err != nil {
		t.Fatalf("legacy wrap: %v", err)
	}
	if err := controlDB.Model(&control.Organization{}).Where("id = ?", legacy).Updates(map[string]any{
		"encrypted_tenant_key": utils.EncodeB64(ct),
		"tenant_key_iv":        utils.EncodeB64(iv),
		"tenant_key_tag":       utils.EncodeB64(tag),
	}).Error; err != nil {
		t.Fatalf("write legacy key: %v", err)
	}
	if _, err := m.GetTenantContext(t.Context(), legacy.String()); err == nil {
		t.Fatal("a tenant key wrapped under the raw master key still decrypts; the purpose separation is not real")
	}

	victim := seedOrg(t, controlDB, keyB64, dsn)
	dekKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("derive dek key: %v", err)
	}
	ct2, iv2, tag2, err := utils.EncryptAESGCMWithAAD(dek, dekKey, []byte(uuid.NewString()))
	if err != nil {
		t.Fatalf("cross-org wrap: %v", err)
	}
	if err := controlDB.Model(&control.Organization{}).Where("id = ?", victim).Updates(map[string]any{
		"encrypted_tenant_key": utils.EncodeB64(ct2),
		"tenant_key_iv":        utils.EncodeB64(iv2),
		"tenant_key_tag":       utils.EncodeB64(tag2),
	}).Error; err != nil {
		t.Fatalf("write cross-org key: %v", err)
	}
	if _, err := m.GetTenantContext(t.Context(), victim.String()); err == nil {
		t.Fatal("a wrapped DEK from another organization decrypted; the org id is not authenticated")
	}
}

func TestAssertNoDatabaseAdminRights(t *testing.T) {
	controlDB, dsn := testControlDB(t)

	if err := AssertNoDatabaseAdminRights(t.Context(), nil); err == nil {
		t.Fatal("expected an error for a nil db — an unanswerable check must deny")
	}

	err := AssertNoDatabaseAdminRights(t.Context(), controlDB)
	if err == nil {
		t.Skip("TEST_DATABASE_URL's role is not a database admin; skipping the positive-detection half")
	}
	if !strings.Contains(err.Error(), "SUPERUSER") && !strings.Contains(err.Error(), "CREATE") {
		t.Fatalf("error does not name the offending attribute: %v", err)
	}

	role := "apiary_noadmin_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	pw := "test-" + uuid.NewString()
	if err := controlDB.Exec(fmt.Sprintf(
		`CREATE ROLE %q WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE PASSWORD '%s'`, role, pw,
	)).Error; err != nil {
		t.Skipf("cannot create a de-privileged role on this cluster: %v", err)
	}
	t.Cleanup(func() { controlDB.Exec(fmt.Sprintf(`DROP ROLE IF EXISTS %q`, role)) })

	scoped := strings.Replace(dsn, "://test:test@", "://"+role+":"+pw+"@", 1)
	if scoped == dsn {
		t.Skip("cannot rewrite TEST_DATABASE_URL credentials; skipping the negative-detection half")
	}
	plainDB, err := gorm.Open(postgres.Open(scoped), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect as the de-privileged role: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := plainDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := AssertNoDatabaseAdminRights(t.Context(), plainDB); err != nil {
		t.Fatalf("a role with no admin attributes was rejected: %v", err)
	}
}
