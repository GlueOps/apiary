package database

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TenantContext struct {
	DB  *gorm.DB
	Key []byte
}

type TenantPoolConfig struct {
	MaxSize         int
	MaxConns        int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	EvictGrace      time.Duration
}

type tenantEntry struct {
	orgID string
	tCtx  *TenantContext
}

type loadCall struct {
	done chan struct{}
	tCtx *TenantContext
	err  error
}

type TenantManager struct {
	ControlDB *gorm.DB

	dekKey []byte
	dsnKey []byte
	cfg    TenantPoolConfig

	mu      sync.Mutex
	lru     *list.List
	index   map[string]*list.Element
	loading map[string]*loadCall
}

func OpenControlDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: quietLogger(),
	})
}

func AssertNoDatabaseAdminRights(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("control db is nil")
	}

	var (
		super      bool
		createDB   bool
		createRole bool
	)
	row := db.WithContext(ctx).Raw(
		`SELECT rolsuper, rolcreatedb, rolcreaterole FROM pg_roles WHERE rolname = current_user`,
	).Row()
	if err := row.Scan(&super, &createDB, &createRole); err != nil {
		return fmt.Errorf("check control role privileges: %w", err)
	}

	var held []string
	if super {
		held = append(held, "SUPERUSER")
	}
	if createDB {
		held = append(held, "CREATEDB")
	}
	if createRole {
		held = append(held, "CREATEROLE")
	}
	if len(held) > 0 {
		return fmt.Errorf(
			"control db role holds %s; DATABASE_URL must be a role with none of "+
				"SUPERUSER/CREATEDB/CREATEROLE — provisioning uses "+
				"PROVISIONER_DATABASE_URL in the worker (F6)",
			strings.Join(held, "+"),
		)
	}
	return nil
}

func NewTenantManager(controlDB *gorm.DB, masterKeyB64 string, pool TenantPoolConfig) (*TenantManager, error) {
	if controlDB == nil {
		return nil, fmt.Errorf("control db is nil")
	}

	master, err := utils.MasterKeyFromBase64(masterKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid master key: %w", err)
	}

	dekKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, fmt.Errorf("derive tenant dek key: %w", err)
	}
	dsnKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDSN)
	if err != nil {
		return nil, fmt.Errorf("derive tenant dsn key: %w", err)
	}

	if pool.MaxSize < 1 {
		return nil, fmt.Errorf("tenant pool max size must be >= 1, got %d", pool.MaxSize)
	}
	if pool.MaxConns < 1 {
		return nil, fmt.Errorf("tenant pool max conns must be >= 1, got %d", pool.MaxConns)
	}
	if pool.MaxIdleConns < 0 || pool.MaxIdleConns > pool.MaxConns {
		return nil, fmt.Errorf(
			"tenant pool max idle conns must be between 0 and max conns (%d), got %d",
			pool.MaxConns, pool.MaxIdleConns,
		)
	}
	if pool.ConnMaxIdleTime < 0 || pool.ConnMaxLifetime < 0 || pool.EvictGrace < 0 {
		return nil, fmt.Errorf("tenant pool durations must not be negative")
	}

	return &TenantManager{
		ControlDB: controlDB,
		dekKey:    dekKey,
		dsnKey:    dsnKey,
		cfg:       pool,
		lru:       list.New(),
		index:     make(map[string]*list.Element),
		loading:   make(map[string]*loadCall),
	}, nil
}

func (m *TenantManager) GetTenantContext(ctx context.Context, orgID string) (*TenantContext, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return nil, fmt.Errorf("invalid org id")
	}
	key := id.String()

	m.mu.Lock()
	if elem, ok := m.index[key]; ok {
		m.lru.MoveToFront(elem)
		tCtx := elem.Value.(*tenantEntry).tCtx
		m.mu.Unlock()
		return tCtx, nil
	}
	if call, ok := m.loading[key]; ok {
		m.mu.Unlock()
		select {
		case <-call.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return call.tCtx, call.err
	}

	call := &loadCall{done: make(chan struct{})}
	m.loading[key] = call
	m.mu.Unlock()

	call.tCtx, call.err = m.openTenant(ctx, key)

	m.mu.Lock()
	delete(m.loading, key)
	if call.err == nil {
		m.publishLocked(key, call.tCtx)
	}
	m.mu.Unlock()

	close(call.done)

	return call.tCtx, call.err
}

func (m *TenantManager) publishLocked(orgID string, tCtx *TenantContext) {
	elem := m.lru.PushFront(&tenantEntry{orgID: orgID, tCtx: tCtx})
	m.index[orgID] = elem

	for m.lru.Len() > m.cfg.MaxSize {
		back := m.lru.Back()
		if back == nil {
			return
		}
		evicted := m.lru.Remove(back).(*tenantEntry)
		delete(m.index, evicted.orgID)

		m.closeAfterGrace(evicted.tCtx.DB)
	}
}

func (m *TenantManager) closeAfterGrace(db *gorm.DB) {
	if db == nil {
		return
	}
	closeDB := func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if m.cfg.EvictGrace <= 0 {
		closeDB()
		return
	}
	time.AfterFunc(m.cfg.EvictGrace, closeDB)
}

func (m *TenantManager) Evict(orgID string) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return
	}
	key := id.String()

	m.mu.Lock()
	defer m.mu.Unlock()
	elem, ok := m.index[key]
	if !ok {
		return
	}
	evicted := m.lru.Remove(elem).(*tenantEntry)
	delete(m.index, key)
	m.closeAfterGrace(evicted.tCtx.DB)
}

func (m *TenantManager) Close() error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	entries := make([]*tenantEntry, 0, m.lru.Len())
	for elem := m.lru.Front(); elem != nil; elem = elem.Next() {
		entries = append(entries, elem.Value.(*tenantEntry))
	}
	m.lru.Init()
	m.index = make(map[string]*list.Element)
	m.mu.Unlock()

	var firstErr error
	for _, e := range entries {
		if e.tCtx == nil || e.tCtx.DB == nil {
			continue
		}
		sqlDB, err := e.tCtx.DB.DB()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := sqlDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *TenantManager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lru.Len()
}

func (m *TenantManager) openTenant(ctx context.Context, orgID string) (*TenantContext, error) {
	// deleted_at IS NULL is the whole enforcement of a soft delete.
	//
	// Without it, deleting an organization changes nothing that matters: the
	// tenant database still opens, every agent still converges, and every
	// org-scoped credential still works — the row is simply marked. A soft delete
	// nothing checks is not a delete, it is a label.
	//
	// It also has to be HERE rather than in each caller. This is the single door
	// to a tenant database, so a check at this point cannot be forgotten by
	// whoever adds the next tenant-scoped service.
	var org control.Organization
	if err := m.ControlDB.WithContext(ctx).
		First(&org, "id = ? AND deleted_at IS NULL", orgID).Error; err != nil {
		return nil, fmt.Errorf("failed to find org %s: %w", orgID, err)
	}

	ct, err1 := utils.DecodeB64(org.EncryptedTenantKey)
	iv, err2 := utils.DecodeB64(org.TenantKeyIV)
	tag, err3 := utils.DecodeB64(org.TenantKeyTag)
	if err1 != nil || err2 != nil || err3 != nil {
		return nil, fmt.Errorf("failed to decode encrypted tenant key components for org %s", orgID)
	}

	tenantKey, err := utils.DecryptAESGCMWithAAD(ct, m.dekKey, iv, tag, []byte(org.ID.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt tenant key for org %s: %w", orgID, err)
	}

	dsn, err := m.decryptConnectionString(&org)
	if err != nil {
		return nil, err
	}

	tenantDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: quietLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant db: %w", err)
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to access tenant sql db for org %s: %w", orgID, err)
	}
	sqlDB.SetMaxOpenConns(m.cfg.MaxConns)
	sqlDB.SetMaxIdleConns(m.cfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(m.cfg.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(m.cfg.ConnMaxLifetime)

	return &TenantContext{
		DB:  tenantDB,
		Key: tenantKey,
	}, nil
}

func (m *TenantManager) decryptConnectionString(org *control.Organization) (string, error) {
	if org.ConnectionStringIV == "" || org.ConnectionStringTag == "" {
		return "", fmt.Errorf(
			"org %s has an unencrypted connection_string; re-provision it to store an encrypted, "+
				"tenant-scoped DSN (F6)", org.ID)
	}

	ct, err1 := utils.DecodeB64(org.ConnectionString)
	iv, err2 := utils.DecodeB64(org.ConnectionStringIV)
	tag, err3 := utils.DecodeB64(org.ConnectionStringTag)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("failed to decode connection string components for org %s", org.ID)
	}

	plain, err := utils.DecryptAESGCMWithAAD(ct, m.dsnKey, iv, tag, []byte(org.ID.String()))
	if err != nil {
		return "", fmt.Errorf("failed to decrypt connection string for org %s: %w", org.ID, err)
	}
	return string(plain), nil
}

// quietLogger is GORM's default logger with ONE flag changed.
//
// ErrRecordNotFound is how First reports an empty result, and one query asks
// that question constantly: the agent heartbeat looks for a bee's single
// outstanding task, and an idle fleet has none. Every bee, on every poll, logged
// a warning carrying the full SQL — a handful of hosts produce a steady stream
// of them, and a log whose normal case is a warning is a log where a real
// warning cannot be seen.
//
// NO ERROR LOGGING IS LOST. GORM's Trace logs when
//
//	err != nil && LogLevel >= Error && (!errors.Is(err, ErrRecordNotFound) || !IgnoreRecordNotFoundError)
//
// so anything that is not ErrRecordNotFound still satisfies the first clause and
// still logs, whatever this flag says. Slow queries are a separate branch and are
// untouched. The error also still reaches the caller either way — this changes
// what GORM announces, not what happens.
//
// Everything else matches logger.Default deliberately, so the diff is one flag
// rather than a quiet change of destination or formatting.
func quietLogger() logger.Interface {
	return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	})
}
