package jobs

import (
	"fmt"
	"net/url"
	"strings"

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

func deriveTenantDBURL(provisionerURL, orgID string) (dbName string, tenantURL string, err error) {
	u, err := parseProvisionerURL(provisionerURL)
	if err != nil {
		return "", "", err
	}

	dbName, err = tenantDBName(orgID)
	if err != nil {
		return "", "", err
	}

	u.Path = "/" + dbName
	u.RawPath = ""

	return dbName, u.String(), nil
}

func tenantDBName(orgID string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return "", fmt.Errorf("invalid org id: %w", err)
	}
	return "tenant_" + strings.ReplaceAll(id.String(), "-", ""), nil
}

func tenantRoleName(orgID string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return "", fmt.Errorf("invalid org id: %w", err)
	}
	return "tenant_role_" + strings.ReplaceAll(id.String(), "-", ""), nil
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
