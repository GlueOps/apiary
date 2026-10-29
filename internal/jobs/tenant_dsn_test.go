package jobs

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	testControlDSN     = "postgres://apiary:controlsecret@db.internal:5432/apiary?sslmode=disable"
	testProvisionerDSN = "postgres://apiary_provisioner:provsecret@db.internal:5432/postgres?sslmode=disable"
	testOrgID          = "6f1c0f2e-1a2b-4c3d-8e4f-5a6b7c8d9e01"
)

func mustMasterKey(t *testing.T) []byte {
	t.Helper()
	key, err := utils.RandomBytes(utils.MasterKeyLen)
	if err != nil {
		t.Fatalf("random master key: %v", err)
	}
	return key
}

func TestProvisionerAdminDBURLRefusesEmptyCredential(t *testing.T) {
	if _, err := provisionerAdminDBURL(""); err == nil {
		t.Fatal("expected an error for an empty provisioner db url, got nil — an empty provisioner credential must never silently become the control credential")
	}
	if _, err := provisionerAdminDBURL("   "); err == nil {
		t.Fatal("expected an error for a whitespace-only provisioner db url, got nil")
	}
	if _, _, err := deriveTenantDBURL("", "", testOrgID); err == nil {
		t.Fatal("expected an error for an empty provisioner db url in deriveTenantDBURL, got nil")
	}
}

func TestProvisioningURLsCarryTheProvisionerRoleOnly(t *testing.T) {
	adminURL, err := provisionerAdminDBURL(testProvisionerDSN)
	if err != nil {
		t.Fatalf("provisionerAdminDBURL: %v", err)
	}
	assertRole(t, adminURL, "apiary_provisioner")

	dbName, tenantURL, err := deriveTenantDBURL(testProvisionerDSN, "", testOrgID)
	if err != nil {
		t.Fatalf("deriveTenantDBURL: %v", err)
	}
	if want := "tenant_6f1c0f2e1a2b4c3d8e4f5a6b7c8d9e01"; dbName != want {
		t.Fatalf("tenant db name = %q, want %q", dbName, want)
	}

	assertRole(t, tenantURL, "apiary_provisioner")

	if strings.Contains(adminURL, "controlsecret") || strings.Contains(tenantURL, "controlsecret") {
		t.Fatal("a provisioning DSN carried the control credential")
	}
}

func TestTenantRoleDBURLKeepsControlNetworkPathAndSwapsOnlyTheRole(t *testing.T) {
	got, err := tenantRoleDBURL(testControlDSN, "tenant_x", "tenant_role_x", "p@ss word/#")
	if err != nil {
		t.Fatalf("tenantRoleDBURL: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if u.Host != "db.internal:5432" {
		t.Fatalf("host = %q, want the control DSN's host", u.Host)
	}
	if u.Query().Get("sslmode") != "disable" {
		t.Fatal("control DSN query string was dropped")
	}
	if u.User.Username() != "tenant_role_x" {
		t.Fatalf("user = %q, want tenant_role_x", u.User.Username())
	}
	pw, _ := u.User.Password()
	if pw != "p@ss word/#" {
		t.Fatalf("password did not survive url encoding: %q", pw)
	}
	if strings.Contains(got, "controlsecret") {
		t.Fatal("the control role's password leaked into a tenant DSN")
	}
}

func TestDSNParseErrorsDoNotLeakThePassword(t *testing.T) {
	bad := "postgres://user:supersecretpassword@ho st:5432/db"
	for name, err := range map[string]error{
		"admin": func() error { _, e := provisionerAdminDBURL(bad); return e }(),
		"tenant": func() error {
			_, _, e := deriveTenantDBURL(bad, "", testOrgID)
			return e
		}(),
		"role": func() error { _, e := tenantRoleDBURL(bad, "d", "r", "p"); return e }(),
	} {
		if err == nil {
			t.Fatalf("%s: expected a parse error", name)
		}
		if strings.Contains(err.Error(), "supersecretpassword") {
			t.Fatalf("%s: error message leaks the DSN password: %q", name, err.Error())
		}
	}
}

func TestProvisionerAdminDBURLRejectsUnusableCredentials(t *testing.T) {
	cases := map[string]string{
		"no role":      "postgres://db.internal:5432/postgres",
		"empty role":   "postgres://:secret@db.internal:5432/postgres",
		"wrong scheme": "mysql://u:p@db.internal:3306/postgres",
	}
	for name, dsn := range cases {
		if _, err := provisionerAdminDBURL(dsn); err == nil {
			t.Fatalf("%s: expected an error, got nil", name)
		}
	}
}

func TestNewTenantProvisionerServiceRefusesAMissingProvisionerCredential(t *testing.T) {
	master := mustMasterKey(t)
	if _, err := NewTenantProvisionerService(&gorm.DB{}, testControlDSN, "", "", master); err == nil {
		t.Fatal("expected an error when PROVISIONER_DATABASE_URL is empty, got nil")
	}
	if _, err := NewTenantProvisionerService(nil, testControlDSN, testProvisionerDSN, "", master); err == nil {
		t.Fatal("expected an error for a nil control db, got nil")
	}
	if _, err := NewTenantProvisionerService(&gorm.DB{}, "", testProvisionerDSN, "", master); err == nil {
		t.Fatal("expected an error for an empty control db url, got nil")
	}
	if _, err := NewTenantProvisionerService(&gorm.DB{}, testControlDSN, testProvisionerDSN, "", master[:16]); err == nil {
		t.Fatal("expected an error for a short master key, got nil")
	}
}

func TestProvisionerDerivesDistinctSubkeys(t *testing.T) {
	master := mustMasterKey(t)
	p, err := NewTenantProvisionerService(&gorm.DB{}, testControlDSN, testProvisionerDSN, "", master)
	if err != nil {
		t.Fatalf("NewTenantProvisionerService: %v", err)
	}
	if bytes.Equal(p.dekKey, master) {
		t.Fatal("tenant DEK key is the master key verbatim")
	}
	if bytes.Equal(p.dsnKey, master) {
		t.Fatal("tenant DSN key is the master key verbatim")
	}
	if bytes.Equal(p.dekKey, p.dsnKey) {
		t.Fatal("tenant DEK key and tenant DSN key are identical; there is no domain separation")
	}
	if len(p.dekKey) != utils.MasterKeyLen || len(p.dsnKey) != utils.MasterKeyLen {
		t.Fatalf("derived key lengths = %d/%d, want %d", len(p.dekKey), len(p.dsnKey), utils.MasterKeyLen)
	}
}

func TestProvisionOrganizationRefusesUnderivedKeys(t *testing.T) {
	p := &TenantProvisionerService{
		controlDB:        &gorm.DB{},
		controlDBURL:     testControlDSN,
		provisionerDBURL: testProvisionerDSN,
	}
	err := p.ProvisionOrganization(t.Context(), testOrgID)
	if err == nil {
		t.Fatal("expected an error when the derived keys are absent, got nil")
	}
	if !strings.Contains(err.Error(), "not derived") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertRole(t *testing.T, rawURL, wantUser string) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	if u.User == nil || u.User.Username() != wantUser {
		t.Fatalf("dsn role = %v, want %q", u.User, wantUser)
	}
}

func TestTenantNamesTakeTheConfiguredPrefix(t *testing.T) {
	const hex = "6f1c0f2e1a2b4c3d8e4f5a6b7c8d9e01"
	cases := []struct{ prefix, db, role string }{
		{"", "tenant_" + hex, "tenant_role_" + hex},
		{"apiary", "apiary_" + hex, "apiary_role_" + hex},
		// The separator is added here, so a prefix written with one still gets one.
		{"apiary_", "apiary_" + hex, "apiary_role_" + hex},
	}
	for _, c := range cases {
		db, err := tenantDBName(c.prefix, testOrgID)
		if err != nil {
			t.Fatalf("tenantDBName(%q): %v", c.prefix, err)
		}
		role, err := tenantRoleName(c.prefix, testOrgID)
		if err != nil {
			t.Fatalf("tenantRoleName(%q): %v", c.prefix, err)
		}
		if db != c.db || role != c.role {
			t.Errorf("prefix %q: got db %q role %q, want %q and %q", c.prefix, db, role, c.db, c.role)
		}
	}
}

// A destroy must drop the database the tenant was given, not the one the
// current prefix would give it — otherwise changing DATABASE_PREFIX silently
// orphans every existing tenant's data the first time one is deleted.
func TestDestroyerDropsTheStoredDatabaseNotOneDerivedFromThePrefix(t *testing.T) {
	master := mustMasterKey(t)
	d, err := NewTenantDestroyerService(&gorm.DB{}, testControlDSN, testProvisionerDSN, "apiary", master)
	if err != nil {
		t.Fatalf("NewTenantDestroyerService: %v", err)
	}

	org := control.Organization{}
	org.ID = uuid.MustParse(testOrgID)

	// Never finished provisioning: whatever was created has the derived name.
	got, err := d.tenantDatabase(&org)
	if err != nil {
		t.Fatalf("tenantDatabase without a stored dsn: %v", err)
	}
	if want := "apiary_6f1c0f2e1a2b4c3d8e4f5a6b7c8d9e01"; got != want {
		t.Fatalf("unprovisioned org: db = %q, want %q", got, want)
	}

	// Provisioned before the prefix changed: the stored DSN wins.
	legacy := "tenant_6f1c0f2e1a2b4c3d8e4f5a6b7c8d9e01"
	dsnKey, err := utils.DeriveKey(master, utils.KeyPurposeTenantDSN)
	if err != nil {
		t.Fatalf("derive dsn key: %v", err)
	}
	ct, iv, tag, err := utils.EncryptAESGCMWithAAD(
		[]byte("postgres://tenant_role_x:pw@db.internal:5432/"+legacy+"?sslmode=disable"),
		dsnKey, []byte(testOrgID),
	)
	if err != nil {
		t.Fatalf("encrypt dsn: %v", err)
	}
	org.ConnectionString = utils.EncodeB64(ct)
	org.ConnectionStringIV = utils.EncodeB64(iv)
	org.ConnectionStringTag = utils.EncodeB64(tag)

	got, err = d.tenantDatabase(&org)
	if err != nil {
		t.Fatalf("tenantDatabase with a stored dsn: %v", err)
	}
	if got != legacy {
		t.Fatalf("provisioned org: db = %q, want the stored %q", got, legacy)
	}
}
