package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/glueops/apiary/internal/models/control"
	"gorm.io/gorm"
)

type TenantDestroyerService struct {
	controlDB        *gorm.DB
	controlDBURL     string
	provisionerDBURL string
}

func NewTenantDestroyerService(controlDB *gorm.DB, controlDBURL, provisionerDBURL string) *TenantDestroyerService {
	return &TenantDestroyerService{
		controlDB:        controlDB,
		controlDBURL:     controlDBURL,
		provisionerDBURL: provisionerDBURL,
	}
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

	dbName, err := tenantDBName(organizationID)
	if err != nil {
		return fmt.Errorf("derive tenant db name: %w", err)
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
