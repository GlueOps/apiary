package jobs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
)

func parseProvisionerURL(provisionerURL string) (*url.URL, error) {
	trimmed := strings.TrimSpace(provisionerURL)
	if trimmed == "" {
		return nil, fmt.Errorf("provisioner db url is empty: set PROVISIONER_DATABASE_URL")
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("provisioner db url is not a valid url")
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("unsupported db scheme %q", u.Scheme)
	}

	if u.User == nil || u.User.Username() == "" {
		return nil, fmt.Errorf("provisioner db url has no role; it must name the CREATEDB+CREATEROLE role")
	}

	return u, nil
}

func provisionerAdminDBURL(provisionerURL string) (string, error) {
	u, err := parseProvisionerURL(provisionerURL)
	if err != nil {
		return "", err
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/postgres"
	}
	u.RawPath = ""

	return u.String(), nil
}

func deriveTenantDBURL(provisionerURL, prefix, orgID string) (dbName string, tenantURL string, err error) {
	u, err := parseProvisionerURL(provisionerURL)
	if err != nil {
		return "", "", err
	}

	dbName, err = tenantDBName(prefix, orgID)
	if err != nil {
		return "", "", err
	}

	u.Path = "/" + dbName
	u.RawPath = ""

	return dbName, u.String(), nil
}

// tenantDBName is the database a NEW tenant is created in.
//
// It is only ever used to choose a name at provisioning time. An existing
// tenant's database is whatever its stored DSN says, which is what makes the
// prefix safe to change: tenants provisioned under an earlier prefix keep the
// name they were given.
//
// One known gap: a provisioning job that created its database and failed before
// storing the DSN, then retried after the prefix changed, creates a second
// database under the new name and leaves the first behind. Nothing records the
// first, so it is found only by listing databases on the server.
func tenantDBName(prefix, orgID string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return "", fmt.Errorf("invalid org id: %w", err)
	}
	return utils.TenantDatabasePrefix(prefix) + "_" + strings.ReplaceAll(id.String(), "-", ""), nil
}

func tenantRoleName(prefix, orgID string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return "", fmt.Errorf("invalid org id: %w", err)
	}
	return utils.TenantDatabasePrefix(prefix) + "_role_" + strings.ReplaceAll(id.String(), "-", ""), nil
}

// databaseFromDSN is the database a DSN connects to.
func databaseFromDSN(dsn string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil {
		return "", fmt.Errorf("tenant dsn is not a valid url")
	}
	db := strings.Trim(u.Path, "/")
	if db == "" {
		return "", fmt.Errorf("tenant dsn names no database")
	}
	return db, nil
}

func tenantRoleDBURL(controlURL, dbName, role, password string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(controlURL))
	if err != nil {
		return "", fmt.Errorf("control db url is not a valid url")
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported db scheme %q", u.Scheme)
	}
	u.User = url.UserPassword(role, password)
	u.Path = "/" + dbName
	u.RawPath = ""
	return u.String(), nil
}
