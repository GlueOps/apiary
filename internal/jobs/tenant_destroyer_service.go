package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"
	"gorm.io/gorm"
)

type TenantDestroyerService struct {
	controlDB        *gorm.DB
	controlDBURL     string
	provisionerDBURL string
	databasePrefix   string

	dsnKey []byte
}

func NewTenantDestroyerService(controlDB *gorm.DB, controlDBURL, provisionerDBURL, databasePrefix string, masterKey []byte) (*TenantDestroyerService, error) {
	dsnKey, err := utils.DeriveKey(masterKey, utils.KeyPurposeTenantDSN)
	if err != nil {
		return nil, fmt.Errorf("derive tenant dsn key: %w", err)
	}
	return &TenantDestroyerService{
		controlDB:        controlDB,
		controlDBURL:     controlDBURL,
		provisionerDBURL: provisionerDBURL,
		databasePrefix:   databasePrefix,
		dsnKey:           dsnKey,
	}, nil
}

// tenantDatabase is the database an organization's data actually lives in.
//
// It is read from the org's stored DSN rather than derived from the configured
// prefix. The prefix only names NEW tenants, so deriving it here would point a
// destroy at the wrong name for every tenant provisioned before the prefix was
// last changed — and DROP DATABASE IF EXISTS would succeed against nothing,
// leaving the real database behind with nobody recording that it exists.
//
// An org with no stored DSN never finished provisioning; its database, if one
// was created at all, has the name provisioning would have given it.
func (d *TenantDestroyerService) tenantDatabase(org *control.Organization) (string, error) {
	if org.ConnectionString == "" {
		return tenantDBName(d.databasePrefix, org.ID.String())
	}
	ct, err1 := utils.DecodeB64(org.ConnectionString)
	iv, err2 := utils.DecodeB64(org.ConnectionStringIV)
	tag, err3 := utils.DecodeB64(org.ConnectionStringTag)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", fmt.Errorf("decode connection string components")
	}
	plain, err := utils.DecryptAESGCMWithAAD(ct, d.dsnKey, iv, tag, []byte(org.ID.String()))
	if err != nil {
		return "", fmt.Errorf("decrypt connection string: %w", err)
	}
	return databaseFromDSN(string(plain))
}

func (d *TenantDestroyerService) DestroyOrganization(ctx context.Context, organizationID string) error {
	var org control.Organization
	if err := d.controlDB.WithContext(ctx).First(&org, "id = ?", organizationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("load org: %w", err)
	}

	if org.Status != "destroying" {
		return fmt.Errorf("org status is %q, expected destroying", org.Status)
	}

	dbName, err := d.tenantDatabase(&org)
	if err != nil {
		return fmt.Errorf("resolve tenant db name: %w", err)
	}

	adminURL, err := provisionerAdminDBURL(d.provisionerDBURL)
	if err != nil {
		return fmt.Errorf("derive admin url: %w", err)
	}

	adminDB, err := sql.Open(pgDriverName, adminURL)
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, `DROP DATABASE IF EXISTS `+quoteIdentifier(dbName)); err != nil {
		return fmt.Errorf("drop database %s: %w", dbName, err)
	}

	return d.controlDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		orgID := org.ID.String()
		deletes := []struct {
			table string
			where string
		}{
			{"lair_desired_resources", "lair_desired_generation_id IN (SELECT id FROM lair_desired_generations WHERE organization_id = ?)"},
			{"lair_desired_generations", "organization_id = ?"},
			{"task_executions", "task_id IN (SELECT id FROM tasks WHERE organization_id = ?)"},
			{"tasks", "organization_id = ?"},
			{"lairs", "organization_id = ?"},
			{"token_sessions", "organization_id = ?"},
			{"tenant_resource_subscriptions", "organization_id = ?"},
			{"impersonation_grants", "organization_id = ?"},
			{"auth_audit_events", "organization_id = ?"},
			{"memberships", "organization_id = ?"},
			{"organizations", "id = ?"},
		}

		for _, d := range deletes {
			if err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s", d.table, d.where), orgID).Error; err != nil {
				return fmt.Errorf("delete from %s: %w", d.table, err)
			}
		}
		return nil
	})
}
