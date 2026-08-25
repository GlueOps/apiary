package jobs

import (
	"context"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"

	"github.com/glueops/apiary/internal/migrations"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

const pgDriverName = "pgx"

type TenantProvisionerService struct {
	controlDB        *gorm.DB
	controlDBURL     string
	provisionerDBURL string

	dekKey []byte
	dsnKey []byte
}

func NewTenantProvisionerService(controlDB *gorm.DB, controlDBURL, provisionerDBURL string, masterKey []byte) (*TenantProvisionerService, error) {
	if controlDB == nil {
		return nil, fmt.Errorf("control db is nil")
	}
	if strings.TrimSpace(controlDBURL) == "" {
		return nil, fmt.Errorf("control db url is empty")
	}

	if strings.TrimSpace(provisionerDBURL) == "" {
		return nil, fmt.Errorf("provisioner db url is empty: set PROVISIONER_DATABASE_URL")
	}

	dekKey, err := utils.DeriveKey(masterKey, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, fmt.Errorf("derive tenant dek key: %w", err)
	}
	dsnKey, err := utils.DeriveKey(masterKey, utils.KeyPurposeTenantDSN)
	if err != nil {
		return nil, fmt.Errorf("derive tenant dsn key: %w", err)
	}

	return &TenantProvisionerService{
		controlDB:        controlDB,
		controlDBURL:     controlDBURL,
		provisionerDBURL: provisionerDBURL,
		dekKey:           dekKey,
		dsnKey:           dsnKey,
	}, nil
}

func (p *TenantProvisionerService) ProvisionOrganization(ctx context.Context, organizationID string) error {
	if p.controlDB == nil {
		return fmt.Errorf("control db is nil")
	}
	if strings.TrimSpace(p.controlDBURL) == "" {
		return fmt.Errorf("control db url is empty")
	}
	if strings.TrimSpace(p.provisionerDBURL) == "" {
		return fmt.Errorf("provisioner db url is empty")
	}

	if len(p.dekKey) != utils.MasterKeyLen || len(p.dsnKey) != utils.MasterKeyLen {
		return fmt.Errorf("tenant provisioner keys are not derived")
	}

	var org control.Organization
	if organizationID == "" {
		return nil
	}
	if err := p.controlDB.WithContext(ctx).First(&org, "id = ?", organizationID).Error; err != nil {
		return fmt.Errorf("load org: %w", err)
	}

	if org.ConnectionString != "" &&
		org.ConnectionStringIV != "" &&
		org.ConnectionStringTag != "" &&
		org.EncryptedTenantKey != "" &&
		org.TenantKeyIV != "" &&
		org.TenantKeyTag != "" {
		return nil
	}

	adminURL, err := provisionerAdminDBURL(p.provisionerDBURL)
	if err != nil {
		return err
	}

	dbName, tenantURL, err := deriveTenantDBURL(p.provisionerDBURL, org.ID.String())
	if err != nil {
		return err
	}

	adminSQL, err := sql.Open(pgDriverName, adminURL)
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer adminSQL.Close()

	if err := createDatabaseIfNotExists(ctx, adminSQL, dbName); err != nil {
		return err
	}

	tenantSQL, err := sql.Open(pgDriverName, tenantURL)
	if err != nil {
		return fmt.Errorf("open tenant db: %w", err)
	}
	defer tenantSQL.Close()

	if err := runTenantMigrations(tenantSQL); err != nil {
		return err
	}

	dek, err := utils.RandomBytes(32)
	if err != nil {
		return fmt.Errorf("generate tenant dek: %w", err)
	}

	ct, iv, tag, err := utils.EncryptAESGCMWithAAD(dek, p.dekKey, []byte(org.ID.String()))
	if err != nil {
		return fmt.Errorf("wrap tenant dek: %w", err)
	}

	roleName, err := tenantRoleName(org.ID.String())
	if err != nil {
		return err
	}
	rolePasswordBytes, err := utils.RandomBytes(32)
	if err != nil {
		return fmt.Errorf("generate tenant role password: %w", err)
	}
	rolePassword := utils.EncodeB64(rolePasswordBytes)

	if err := createTenantRole(ctx, adminSQL, tenantSQL, roleName, rolePassword, dbName); err != nil {
		return err
	}

	scopedURL, err := tenantRoleDBURL(p.controlDBURL, dbName, roleName, rolePassword)
	if err != nil {
		return err
	}

	dsnCT, dsnIV, dsnTag, err := utils.EncryptAESGCMWithAAD(
		[]byte(scopedURL), p.dsnKey, []byte(org.ID.String()),
	)
	if err != nil {
		return fmt.Errorf("encrypt tenant dsn: %w", err)
	}

	if err := p.controlDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fresh control.Organization
		if err := tx.First(&fresh, "id = ?", organizationID).Error; err != nil {
			return err
		}

		if fresh.ConnectionString != "" &&
			fresh.ConnectionStringIV != "" &&
			fresh.ConnectionStringTag != "" &&
			fresh.EncryptedTenantKey != "" &&
			fresh.TenantKeyIV != "" &&
			fresh.TenantKeyTag != "" {
			return nil
		}

		return tx.Model(&fresh).Updates(map[string]any{
			"connection_string":     utils.EncodeB64(dsnCT),
			"connection_string_iv":  utils.EncodeB64(dsnIV),
			"connection_string_tag": utils.EncodeB64(dsnTag),
			"encrypted_tenant_key":  utils.EncodeB64(ct),
			"tenant_key_iv":         utils.EncodeB64(iv),
			"tenant_key_tag":        utils.EncodeB64(tag),
			"status":                "active",
		}).Error
	}); err != nil {
		return fmt.Errorf("update org after provisioning: %w", err)
	}

	return nil
}

func runTenantMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	goose.SetBaseFS(migrations.TenantFS)
	if err := goose.Up(db, "tenant"); err != nil {
		return fmt.Errorf("tenant goose up: %w", err)
	}
	return nil
}

func createDatabaseIfNotExists(ctx context.Context, adminDB *sql.DB, dbName string) error {
	var exists bool
	if err := adminDB.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`,
		dbName,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check database existence: %w", err)
	}

	if exists {
		return nil
	}

	_, err := adminDB.ExecContext(ctx, `CREATE DATABASE `+quoteIdentifier(dbName))
	if err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func createTenantRole(ctx context.Context, adminDB, tenantDB *sql.DB, role, password, dbName string) error {

	verifier, err := scramSHA256Verifier(password)
	if err != nil {
		return fmt.Errorf("build scram verifier: %w", err)
	}

	var exists bool
	if err := adminDB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`, role,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check role existence: %w", err)
	}

	if exists {
		if _, err := adminDB.ExecContext(ctx,
			`ALTER ROLE `+quoteIdentifier(role)+` WITH LOGIN PASSWORD `+quoteLiteral(verifier),
		); err != nil {
			return fmt.Errorf("rotate tenant role password: %w", err)
		}
	} else {
		if _, err := adminDB.ExecContext(ctx,
			`CREATE ROLE `+quoteIdentifier(role)+
				` WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD `+
				quoteLiteral(verifier),
		); err != nil {
			return fmt.Errorf("create tenant role: %w", err)
		}
	}

	if _, err := adminDB.ExecContext(ctx,
		`GRANT CONNECT ON DATABASE `+quoteIdentifier(dbName)+` TO `+quoteIdentifier(role),
	); err != nil {
		return fmt.Errorf("grant connect: %w", err)
	}

	for _, stmt := range []string{
		`GRANT USAGE ON SCHEMA public TO ` + quoteIdentifier(role),
		// CREATE, so this role can run its own migrations.
		//
		// Tenant migrations run per tenant, as the tenant, and that is the design
		// rather than an accident: with a hundred tenants, a migration path that
		// depends on something enumerating databases from outside is a path that
		// drifts, and drift here means tenants quietly running different schemas
		// under code that assumes one. It also could not work — the mapping from
		// database to tenant is deliberately not readable by a human looking at
		// the cluster, because the connection string is sealed.
		//
		// This does NOT make the role a database administrator. rolsuper,
		// rolcreatedb and rolcreaterole are untouched and still asserted against
		// by database.AssertNoDatabaseAdminRights; the grant is schema-scoped and
		// confined to this tenant's own database.
		`GRANT CREATE ON SCHEMA public TO ` + quoteIdentifier(role),
		`GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ` + quoteIdentifier(role),
		`GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ` + quoteIdentifier(role),
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ` + quoteIdentifier(role),
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ` + quoteIdentifier(role),
	} {
		if _, err := tenantDB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("grant tenant privileges: %w", err)
		}
	}

	return nil
}

const scramIterations = 4096

func scramSHA256Verifier(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is empty")
	}

	salt, err := utils.RandomBytes(16)
	if err != nil {
		return "", fmt.Errorf("scram salt: %w", err)
	}

	salted, err := pbkdf2.Key(sha256.New, password, salt, scramIterations, sha256.Size)
	if err != nil {
		return "", fmt.Errorf("scram pbkdf2: %w", err)
	}

	clientKey := hmacSHA256(salted, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacSHA256(salted, []byte("Server Key"))

	return fmt.Sprintf(
		"SCRAM-SHA-256$%d:%s$%s:%s",
		scramIterations,
		utils.EncodeB64(salt),
		utils.EncodeB64(storedKey[:]),
		utils.EncodeB64(serverKey),
	), nil
}

func hmacSHA256(key, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
}

func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
