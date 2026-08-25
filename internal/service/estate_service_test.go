package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/glueops/apiary/internal/testsupport"
	"os"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This suite owns its own schemas. internal/repo's credential and estate tests
// AutoMigrate and then DROP TABLE estates/credentials in the DEFAULT search_path
// of the same TEST_DATABASE_URL, and `go test ./...` runs packages
// concurrently — sharing the default schema would have this suite's tables
// dropped mid-run and vice versa, surfacing as intermittent "relation does not
// exist" failures that look like service bugs.
var (
	estateSvcSchema  = testsupport.SchemaName("apiary_estatesvc_test")
	estateSvcSchemaB = testsupport.SchemaName("apiary_estatesvc_test_b")
)

// ------------------------------------------------------------------ the seam

// fakeTenants stands in for *database.TenantManager through the TenantResolver
// seam. It is what makes this layer testable at all: the real manager needs a
// provisioned control-plane organization carrying an encrypted DSN and a
// wrapped key, so testing against it would re-test the tenant manager rather
// than the service. It also lets a test CHOOSE the tenant key (so "the service
// derives per purpose rather than sealing under the raw key" is assertable) and
// return an error (so the ErrTenantUnavailable branch is reachable without
// depending on how a missing org happens to fail).
type fakeTenants struct {
	byOrg map[uuid.UUID]*database.TenantContext
	err   error
	seen  []string
}

func (f *fakeTenants) GetTenantContext(_ context.Context, orgID string) (*database.TenantContext, error) {
	f.seen = append(f.seen, orgID)
	if f.err != nil {
		return nil, f.err
	}
	id, err := uuid.Parse(strings.TrimSpace(orgID))
	if err != nil {
		return nil, fmt.Errorf("fake tenants: bad org id %q: %w", orgID, err)
	}
	tCtx, ok := f.byOrg[id]
	if !ok {
		return nil, fmt.Errorf("fake tenants: no tenant for %s", orgID)
	}
	return tCtx, nil
}

// --------------------------------------------------------------- the harness

func estateSvcTestDB(t *testing.T, schema string) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — SKIPPING the estate service suite; " +
			"this test does NOT pass, it did not run")
	}

	bootstrap, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := bootstrap.Exec(`CREATE SCHEMA IF NOT EXISTS ` + schema).Error; err != nil {
		t.Fatalf("create schema %s: %v", schema, err)
	}
	if sqlDB, err := bootstrap.DB(); err == nil {
		_ = sqlDB.Close()
	}

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"search_path="+schema), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open scoped test db: %v", err)
	}
	if err := db.AutoMigrate(&tenant.Credential{}, &tenant.Estate{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec(`TRUNCATE estates, credentials CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}

	assertRestrictFK(t, db, schema)

	t.Cleanup(func() {
		db.Exec(`TRUNCATE estates, credentials CASCADE`)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// assertRestrictFK pins the constraint that forces Delete's ordering.
//
// In production it comes from the migration; here it comes from AutoMigrate
// reading the `constraint:OnDelete:RESTRICT` tag on tenant.Estate. If
// AutoMigrate ever stops emitting it, TestDeleteRemovesTheCredentialAsWellAsThe
// Estate would still pass while proving strictly less than it claims — the
// ordering would no longer be forced by anything.
func assertRestrictFK(t *testing.T, db *gorm.DB, schema string) {
	t.Helper()

	var delType string
	err := db.Raw(`
		SELECT c.confdeltype
		FROM pg_constraint c
		JOIN pg_class r ON r.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = r.relnamespace
		WHERE c.contype = 'f' AND r.relname = 'estates' AND n.nspname = ?`, schema).
		Scan(&delType).Error
	if err != nil {
		t.Fatalf("look up estates foreign key: %v", err)
	}
	if delType != "r" {
		t.Fatalf("estates -> credentials foreign key has ON DELETE %q, want 'r' (RESTRICT); "+
			"without RESTRICT the delete ordering is no longer enforced by the schema", delType)
	}
}

// tenantKey is the key the fake resolver hands back. Distinctive bytes so a
// test can tell "sealed under this" from "sealed under something derived".
func estateSvcTenantKey() []byte {
	k := make([]byte, 32)
	for i := range k {
		k[i] = byte(0xA0 + i)
	}
	return k
}

func estateSvcMasterKey() []byte {
	k := make([]byte, utils.MasterKeyLen)
	for i := range k {
		k[i] = byte(i)
	}
	return k
}

type estateFixture struct {
	svc  *EstateService
	db   *gorm.DB
	org  uuid.UUID
	key  []byte
	fake *fakeTenants
}

func newEstateFixture(t *testing.T, scopes repo.ProviderScopeRegistry) estateFixture {
	t.Helper()

	db := estateSvcTestDB(t, estateSvcSchema)
	org := uuid.New()
	key := estateSvcTenantKey()

	fake := &fakeTenants{byOrg: map[uuid.UUID]*database.TenantContext{
		org: {DB: db, Key: key},
	}}
	svc, err := NewEstateService(fake, estateSvcMasterKey(), scopes)
	if err != nil {
		t.Fatalf("construct estate service: %v", err)
	}
	return estateFixture{svc: svc, db: db, org: org, key: key, fake: fake}
}

func awsRegistration(org uuid.UUID, name, region, account, identity, secret string) RegisterEstateInput {
	return RegisterEstateInput{
		OrgID: org,

		Name:     name,
		Provider: "aws",
		Scope:    map[string]string{"region": region, "account": account},

		CredentialName: name + "-key",
		CredentialKind: tenant.CredentialKindStaticSecret,

		Identity: identity,
		Secret:   repo.Secret(secret),
	}
}

func estateCountRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func readCredentialRow(t *testing.T, db *gorm.DB, id uuid.UUID) tenant.Credential {
	t.Helper()
	var c tenant.Credential
	if err := db.First(&c, "id = ?", id).Error; err != nil {
		t.Fatalf("read credentials row %s: %v", id, err)
	}
	return c
}

// opener builds a repo bound to (db, org, key) so a test can attempt to OPEN a
// row the service sealed. The service deliberately exposes no path that returns
// plaintext, so the opening half is necessarily test-side; the SEALING half —
// the thing under test — runs entirely through Register/RotateCredential.
func estateOpener(t *testing.T, db *gorm.DB, org uuid.UUID, key []byte) *repo.CredentialRepoGorm {
	t.Helper()
	r, err := repo.NewCredentialRepoGorm(db, org, utils.LegacyKeyID, key, nil)
	if err != nil {
		t.Fatalf("build opener: %v", err)
	}
	return r
}

// ------------------------------------------------------ failing closed, no DB

// A nil *EstateService is exactly what cmd/serve.go leaves in deps.Estates when
// there is no tenant manager. Every method must refuse rather than dereference
// the nil receiver, or the no-tenant-manager deployment panics on every estate
// request instead of answering 503.
func TestNilEstateServiceDeniesRatherThanPanicking(t *testing.T) {
	var s *EstateService
	org, id := uuid.New(), uuid.New()

	check := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, ErrTenantUnavailable) {
			t.Errorf("%s on a nil service returned %v, want ErrTenantUnavailable "+
				"(the handler maps only that to 503; anything else becomes a 500)", name, err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a nil service panicked instead of refusing: %v", r)
		}
	}()

	_, err := s.Register(t.Context(), RegisterEstateInput{OrgID: org})
	check("Register", err)
	_, err = s.List(t.Context(), org)
	check("List", err)
	_, err = s.Get(t.Context(), org, id)
	check("Get", err)
	check("Delete", s.Delete(t.Context(), org, id))
	_, err = s.RotateCredential(t.Context(), org, id, "AKIA", repo.Secret("s"))
	check("RotateCredential", err)
}

// A tenant that cannot be resolved must become ErrTenantUnavailable (503), not
// a nil-pointer dereference of the *TenantContext and not a bare 500. The cause
// must stay reachable, because bind wraps with "%w: %w" and an operator needs
// to know WHY the tenant would not resolve.
func TestUnresolvableTenantIsErrTenantUnavailableNotAPanic(t *testing.T) {
	boom := errors.New("tenant database is still provisioning")
	svc, err := NewEstateService(&fakeTenants{err: boom}, estateSvcMasterKey(), nil)
	if err != nil {
		t.Fatalf("construct: %v", err)
	}
	org, id := uuid.New(), uuid.New()

	check := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, ErrTenantUnavailable) {
			t.Errorf("%s returned %v, want ErrTenantUnavailable; the handler keys 503 off "+
				"exactly this sentinel, so losing the wrapping regresses it to 500", name, err)
		}
		// The cause has to survive too, or the operator sees "not available"
		// with no way to tell provisioning from a broken DSN.
		if !errors.Is(err, boom) {
			t.Errorf("%s dropped the underlying cause: %v", name, err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("an unresolvable tenant panicked instead of refusing: %v", r)
		}
	}()

	_, e := svc.Register(t.Context(), RegisterEstateInput{OrgID: org})
	check("Register", e)
	_, e = svc.List(t.Context(), org)
	check("List", e)
	_, e = svc.Get(t.Context(), org, id)
	check("Get", e)
	check("Delete", svc.Delete(t.Context(), org, id))
	_, e = svc.RotateCredential(t.Context(), org, id, "AKIA", repo.Secret("s"))
	check("RotateCredential", e)
}

func TestNewEstateServiceFailsClosed(t *testing.T) {
	// A TYPED nil is the dangerous one: once the parameter is an interface,
	// `tenants == nil` no longer catches (*database.TenantManager)(nil), and a
	// service constructed around it panics later inside GetTenantContext's
	// mutex instead of being refused here.
	if _, err := NewEstateService((*database.TenantManager)(nil), estateSvcMasterKey(), nil); err == nil {
		t.Error("a typed-nil *database.TenantManager was accepted; it panics on first use")
	}
	if _, err := NewEstateService(nil, estateSvcMasterKey(), nil); err == nil {
		t.Error("a nil resolver was accepted")
	}

	for _, n := range []int{0, 16, 33} {
		key := make([]byte, n)
		if n == 0 {
			key = nil
		}
		if _, err := NewEstateService(&fakeTenants{}, key, nil); err == nil {
			t.Errorf("a %d-byte master key was accepted, want refusal at %d", n, utils.MasterKeyLen)
		}
	}

	// A nil registry must DEFAULT, not disable validation: a service that
	// validates nothing would store scopes no request ever resolves to.
	svc, err := NewEstateService(&fakeTenants{}, estateSvcMasterKey(), nil)
	if err != nil {
		t.Fatalf("nil scopes should be accepted and defaulted: %v", err)
	}
	if _, ok := svc.scopes["aws"]; !ok {
		t.Error("nil scopes did not fall back to repo.DefaultProviderScopes")
	}
}

// ----------------------------------------------------------------- atomicity

// Register creates the credential BEFORE the estate, because Estate.CredentialID
// is NOT NULL. That makes a partial failure representable, and a credential with
// no estate is the worst possible leftover: unreachable through the API,
// invisible in every listing, and still a live secret at the provider. The
// transaction is the only thing preventing it.
func TestRegisterIsAtomicWhenTheEstateInsertFails(t *testing.T) {
	t.Run("db-level failure: duplicate provider+scope", func(t *testing.T) {
		f := newEstateFixture(t, nil)

		if _, err := f.svc.Register(t.Context(),
			awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "s1")); err != nil {
			t.Fatalf("first register: %v", err)
		}

		before := estateCountRows(t, f.db, "credentials")

		// Same (provider, scope): the credential INSERT succeeds inside the
		// transaction and the estate INSERT then trips
		// uniq_estates_provider_scope.
		_, err := f.svc.Register(t.Context(),
			awsRegistration(f.org, "prod-again", "eu-west-1", "111111111111", "AKIATWO", "s2"))
		if err == nil {
			t.Fatal("a duplicate (provider, scope) must be refused")
		}

		if after := estateCountRows(t, f.db, "credentials"); after != before {
			t.Fatalf("credentials went from %d to %d: the failed registration left an "+
				"ORPHANED sealed credential — unreachable through the API and still live "+
				"at the provider", before, after)
		}
	})

	t.Run("validation-level failure: unknown provider", func(t *testing.T) {
		f := newEstateFixture(t, nil)

		before := estateCountRows(t, f.db, "credentials")

		// The credential repo accepts any provider string; only the estate
		// repo's canonicalScope knows "nimbus" is not a provider. So the
		// credential row is written and then must be rolled back.
		_, err := f.svc.Register(t.Context(), RegisterEstateInput{
			OrgID:          f.org,
			Name:           "ghost",
			Provider:       "nimbus",
			Scope:          map[string]string{"region": "eu-west-1"},
			CredentialName: "ghost-key",
			Identity:       "NIMBUS-ID",
			Secret:         repo.Secret("s"),
		})
		if !errors.Is(err, repo.ErrUnknownProvider) {
			t.Fatalf("want ErrUnknownProvider (handler -> 422), got %v", err)
		}
		if after := estateCountRows(t, f.db, "credentials"); after != before {
			t.Fatalf("credentials went from %d to %d: a rejected registration left an "+
				"orphaned sealed credential behind", before, after)
		}
		if n := estateCountRows(t, f.db, "estates"); n != 0 {
			t.Fatalf("estates = %d, want 0", n)
		}
	})
}

// --------------------------------------------------------------------- delete

// Deleting only the estate would leave a sealed secret nothing references:
// invisible in every listing, unreachable through the API, still live at the
// provider. Both rows go.
func TestDeleteRemovesTheCredentialAsWellAsTheEstate(t *testing.T) {
	f := newEstateFixture(t, nil)

	view, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "s1"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := f.svc.Delete(t.Context(), f.org, view.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if n := estateCountRows(t, f.db, "estates"); n != 0 {
		t.Errorf("estates = %d, want 0", n)
	}
	// The one that matters. A surviving row here is an orphaned live secret.
	if n := estateCountRows(t, f.db, "credentials"); n != 0 {
		t.Errorf("credentials = %d, want 0: the estate was deleted but its sealed "+
			"credential survived, unreachable and still valid at the provider", n)
	}
}

// A missing estate must be ErrEstateNotFound (404), not a panic on the nil
// *Estate whose CredentialID the delete path reads next, and it must not take
// anything else with it.
func TestDeleteOfAnUnknownEstateIsNotFoundAndDeletesNothing(t *testing.T) {
	f := newEstateFixture(t, nil)

	if _, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "s1")); err != nil {
		t.Fatalf("register: %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("deleting an unknown estate panicked: %v", r)
			}
		}()
		if err := f.svc.Delete(t.Context(), f.org, uuid.New()); !errors.Is(err, repo.ErrEstateNotFound) {
			t.Fatalf("want ErrEstateNotFound (handler -> 404), got %v", err)
		}
	}()

	if n := estateCountRows(t, f.db, "estates"); n != 1 {
		t.Errorf("estates = %d, want the unrelated estate untouched", n)
	}
	if n := estateCountRows(t, f.db, "credentials"); n != 1 {
		t.Errorf("credentials = %d, want the unrelated credential untouched", n)
	}
}

// --------------------------------------------------------------------- rotate

// The changing fingerprint is the ONLY signal an operator has that a rotation
// actually took; the unchanged binding is what makes rotation safe to run
// against a live estate.
func TestRotateChangesTheFingerprintAndCiphertextButNotTheBinding(t *testing.T) {
	f := newEstateFixture(t, nil)

	before, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAOLD", "old-secret"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	beforeRow := readCredentialRow(t, f.db, before.CredentialID)

	after, err := f.svc.RotateCredential(t.Context(), f.org, before.ID, "AKIANEW", repo.Secret("new-secret"))
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if after.Fingerprint == before.Fingerprint {
		t.Error("fingerprint did not change: an operator has no way to confirm the rotation took")
	}
	// Whole digest, never truncated: a comparison is only as strong as what is
	// compared, and for an assumed role the identifying component is an ARN
	// whose name the customer chooses, so a short prefix is grindable.
	if !strings.HasPrefix(after.Fingerprint, "SHA256:") || len(after.Fingerprint) != len("SHA256:")+43 {
		t.Errorf("fingerprint %q is not a complete SHA256: digest", after.Fingerprint)
	}

	afterRow := readCredentialRow(t, f.db, before.CredentialID)
	if afterRow.Ciphertext == beforeRow.Ciphertext {
		t.Error("ciphertext unchanged: the old secret is still what is stored")
	}
	if afterRow.IV == beforeRow.IV {
		t.Error("IV reused across a rotation under the same key — nonce reuse in AES-GCM is catastrophic")
	}

	// The binding is 1:1 and must survive: rotating must not orphan the old
	// credential nor repoint the estate.
	if after.ID != before.ID || after.CredentialID != before.CredentialID {
		t.Fatalf("binding moved: estate %s->%s credential %s->%s",
			before.ID, after.ID, before.CredentialID, after.CredentialID)
	}
	if fmt.Sprint(after.Scope) != fmt.Sprint(before.Scope) {
		t.Errorf("scope changed across a rotation: %v -> %v", before.Scope, after.Scope)
	}
	if n := estateCountRows(t, f.db, "credentials"); n != 1 {
		t.Errorf("credentials = %d, want 1: rotation is in place, not create-and-swap", n)
	}

	reread, err := f.svc.Get(t.Context(), f.org, before.ID)
	if err != nil {
		t.Fatalf("get after rotate: %v", err)
	}
	if reread.CredentialID != before.CredentialID {
		t.Errorf("stored estate points at %s, want %s", reread.CredentialID, before.CredentialID)
	}
	if reread.Fingerprint != after.Fingerprint {
		t.Errorf("Get reports fingerprint %q but rotate returned %q", reread.Fingerprint, after.Fingerprint)
	}
}

// The rotated ciphertext must open under AAD("credentials", orgID, credentialID)
// for THIS row — that is what stops an attacker with database write access from
// moving a credential blob onto another row, or another tenant's row, and having
// it decrypt cleanly. The sealing runs through service.RotateCredential; only
// the opening is test-side, because the service exposes no path to plaintext.
func TestRotatedCiphertextStillOpensUnderThisRowsAAD(t *testing.T) {
	f := newEstateFixture(t, nil)

	view, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAOLD", "old-secret"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := f.svc.RotateCredential(t.Context(), f.org, view.ID, "AKIANEW", repo.Secret("new-secret")); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	dek, err := utils.DeriveKey(f.key, utils.KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	row := readCredentialRow(t, f.db, view.CredentialID)

	got, err := estateOpener(t, f.db, f.org, dek).OpenSecret(t.Context(), &row)
	if err != nil {
		t.Fatalf("the rotated row must open under its own AAD: %v", err)
	}
	if string(got) != "new-secret" {
		t.Fatalf("rotated row yielded %q, want the new secret", string(got))
	}

	// Flip one AAD input at a time. Each must fail.
	if _, err := estateOpener(t, f.db, uuid.New(), dek).OpenSecret(t.Context(), &row); err == nil {
		t.Error("the rotated ciphertext opened under a DIFFERENT organization: the AAD is " +
			"not binding the tenant, so a blob moved between tenant databases would decrypt")
	}

	moved := row
	moved.ID = uuid.New()
	if _, err := estateOpener(t, f.db, f.org, dek).OpenSecret(t.Context(), &moved); err == nil {
		t.Error("the rotated ciphertext opened on a DIFFERENT row id: the AAD is not binding " +
			"the row, so an attacker with write access could repoint a credential")
	}
}

func TestRotateOnAnUnknownEstateIsNotFound(t *testing.T) {
	f := newEstateFixture(t, nil)

	view, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAOLD", "old-secret"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	before := readCredentialRow(t, f.db, view.CredentialID)

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("rotating an unknown estate panicked: %v", r)
			}
		}()
		_, err := f.svc.RotateCredential(t.Context(), f.org, uuid.New(), "AKIANEW", repo.Secret("new"))
		if !errors.Is(err, repo.ErrEstateNotFound) {
			t.Fatalf("want ErrEstateNotFound (handler -> 404), got %v", err)
		}
	}()

	// A rotate that missed must not have touched anybody else's material.
	after := readCredentialRow(t, f.db, view.CredentialID)
	if after.Ciphertext != before.Ciphertext || after.Fingerprint != before.Fingerprint {
		t.Error("a failed rotation overwrote an unrelated credential")
	}
}

// ------------------------------------------------------------ key derivation

// The service must seal under a key DERIVED per purpose from the tenant key,
// never the tenant key itself. One key for every purpose means a compromise
// anywhere is a compromise everywhere, and it makes rotating for one reason
// force re-encryption of everything.
//
// This goes through the service's own sealing path rather than exercising HKDF
// directly, so it fails if the service stops deriving — which is the whole
// point; a direct HKDF test proves the primitive works, not that this code uses
// it.
func TestTenantKeyIsDerivedPerPurposeAndNotUsedRaw(t *testing.T) {
	f := newEstateFixture(t, nil)

	view, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "the-secret"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	row := readCredentialRow(t, f.db, view.CredentialID)

	if _, err := estateOpener(t, f.db, f.org, f.key).OpenSecret(t.Context(), &row); err == nil {
		t.Error("the stored credential opened under the RAW tenant key: the service is not " +
			"deriving a per-purpose data key, so one key now covers every purpose")
	}

	dek, err := utils.DeriveKey(f.key, utils.KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	got, err := estateOpener(t, f.db, f.org, dek).OpenSecret(t.Context(), &row)
	if err != nil {
		t.Fatalf("the stored credential must open under DeriveKey(tenantKey, KeyPurposeTenantDEK): %v", err)
	}
	if string(got) != "the-secret" {
		t.Fatalf("opened %q, want the registered secret", string(got))
	}
}

// ---------------------------------------------------------- tenant isolation

// The organization must actually SELECT the tenant database. Any bind that
// ignores it — a cached or fixed tenant context — hands one customer a handle
// on another's cloud credentials. This is the service-level half of "the
// organization comes from the principal, never from a request parameter":
// getting the org right is worthless if the org then does not steer anything.
func TestOneOrganizationNeverSeesAnothersEstates(t *testing.T) {
	dbA := estateSvcTestDB(t, estateSvcSchema)
	dbB := estateSvcTestDB(t, estateSvcSchemaB)

	orgA, orgB := uuid.New(), uuid.New()
	fake := &fakeTenants{byOrg: map[uuid.UUID]*database.TenantContext{
		orgA: {DB: dbA, Key: estateSvcTenantKey()},
		orgB: {DB: dbB, Key: estateSvcTenantKey()},
	}}
	svc, err := NewEstateService(fake, estateSvcMasterKey(), nil)
	if err != nil {
		t.Fatalf("construct: %v", err)
	}

	a, err := svc.Register(t.Context(),
		awsRegistration(orgA, "a-prod", "eu-west-1", "111111111111", "AKIAA", "secret-a"))
	if err != nil {
		t.Fatalf("register under A: %v", err)
	}

	listB, err := svc.List(t.Context(), orgB)
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if len(listB) != 0 {
		t.Fatalf("organization B sees %d of A's estates — this is the IDOR that exposes "+
			"another tenant's cloud credentials", len(listB))
	}
	if _, err := svc.Get(t.Context(), orgB, a.ID); !errors.Is(err, repo.ErrEstateNotFound) {
		t.Fatalf("B fetching A's estate id returned %v, want ErrEstateNotFound", err)
	}

	// Same scope under B must be accepted: the uniqueness is per tenant
	// database, so a collision here would prove the two share one.
	if _, err := svc.Register(t.Context(),
		awsRegistration(orgB, "b-prod", "eu-west-1", "111111111111", "AKIAB", "secret-b")); err != nil {
		t.Fatalf("register under B: %v", err)
	}

	listA, err := svc.List(t.Context(), orgA)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != a.ID {
		t.Fatalf("A's listing changed when B registered: %+v", listA)
	}
}

// ------------------------------------------------------------- what leaks out

// EstateView is what leaves this service. Nothing above this layer has a reason
// to see sealed material, and a field that does not exist cannot be serialised
// into a response by accident. This goes red when sealed material is ADDED to
// the view, which is the realistic regression.
func TestRegisteredViewCarriesNoSealedMaterial(t *testing.T) {
	f := newEstateFixture(t, nil)

	const secret = "wJalrXUtnFEMI-distinctive-secret"

	view, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", secret))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	row := readCredentialRow(t, f.db, view.CredentialID)

	got, err := f.svc.Get(t.Context(), f.org, view.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	list, err := f.svc.List(t.Context(), f.org)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list returned %d estates, want 1", len(list))
	}
	rotated, err := f.svc.RotateCredential(t.Context(), f.org, view.ID, "AKIANEW", repo.Secret(secret))
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	rotatedRow := readCredentialRow(t, f.db, view.CredentialID)

	banned := map[string]string{
		"the secret itself":      secret,
		"the ciphertext":         row.Ciphertext,
		"the iv":                 row.IV,
		"the tag":                row.Tag,
		"the rotated ciphertext": rotatedRow.Ciphertext,
		"the rotated iv":         rotatedRow.IV,
		"the rotated tag":        rotatedRow.Tag,
	}

	for name, v := range map[string]EstateView{
		"Register": *view,
		"Get":      *got,
		"List":     list[0],
		"Rotate":   *rotated,
	} {
		rendered := fmt.Sprintf("%+v", v)
		for what, needle := range banned {
			if needle == "" {
				continue
			}
			if strings.Contains(rendered, needle) {
				t.Errorf("the view returned by %s carries %s: %s", name, what, rendered)
			}
		}
		// The fingerprint is the one credential-identifying value that SHOULD
		// be there, complete.
		if !strings.HasPrefix(v.Fingerprint, "SHA256:") || len(v.Fingerprint) != len("SHA256:")+43 {
			t.Errorf("%s returned fingerprint %q, want a complete SHA256: digest", name, v.Fingerprint)
		}
	}
}

// -------------------------------------------------------------- scope wiring

// The registry is the only thing that stops an unvalidated scope being stored,
// so it has to REACH the repo. NewEstateRepoGorm silently falls back to
// repo.DefaultProviderScopes when handed nil, so a registry dropped on the way
// through is a no-op that nothing else would notice.
func TestScopeRegistryIsThreadedThroughToTheEstateRepo(t *testing.T) {
	only := repo.ProviderScopeRegistry{
		"proxmox": {Provider: "proxmox", Required: []string{"cluster"}},
	}
	f := newEstateFixture(t, only)

	_, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "s"))
	if !errors.Is(err, repo.ErrUnknownProvider) {
		t.Fatalf("aws is not in this registry, want ErrUnknownProvider (handler -> 422), got %v", err)
	}

	if _, err := f.svc.Register(t.Context(), RegisterEstateInput{
		OrgID:          f.org,
		Name:           "dc1",
		Provider:       "proxmox",
		Scope:          map[string]string{"cluster": "dc1"},
		CredentialName: "dc1-token",
		Identity:       "root@pam!apiary",
		Secret:         repo.Secret("s"),
	}); err != nil {
		t.Fatalf("proxmox is in this registry and must be accepted: %v", err)
	}
}

// --------------------------------------------------------------------- list

// List joins each estate to its credential through a map lookup that fails
// SILENTLY to a zero-value Credential — an estate whose join misses renders an
// empty fingerprint and name rather than an error, so the operator's only
// credential-identifying value just disappears.
func TestListJoinsEachEstateToItsCredentialsFingerprint(t *testing.T) {
	f := newEstateFixture(t, nil)

	first, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "prod", "eu-west-1", "111111111111", "AKIAONE", "s1"))
	if err != nil {
		t.Fatalf("register one: %v", err)
	}
	second, err := f.svc.Register(t.Context(),
		awsRegistration(f.org, "staging", "us-east-1", "222222222222", "AKIATWO", "s2"))
	if err != nil {
		t.Fatalf("register two: %v", err)
	}

	list, err := f.svc.List(t.Context(), f.org)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list returned %d estates, want 2", len(list))
	}

	byID := map[uuid.UUID]EstateView{}
	for _, v := range list {
		byID[v.ID] = v
	}

	for _, want := range []*EstateView{first, second} {
		got, ok := byID[want.ID]
		if !ok {
			t.Fatalf("estate %s missing from the listing", want.ID)
		}
		if got.Fingerprint == "" {
			t.Errorf("estate %s listed with an empty fingerprint: the credential join "+
				"missed and failed silently to a zero-value Credential", want.ID)
		}
		if got.Fingerprint != want.Fingerprint {
			t.Errorf("estate %s listed with fingerprint %q, want %q — the join returned "+
				"the WRONG credential", want.ID, got.Fingerprint, want.Fingerprint)
		}

		single, err := f.svc.Get(t.Context(), f.org, want.ID)
		if err != nil {
			t.Fatalf("get %s: %v", want.ID, err)
		}
		if got.CredentialName != single.CredentialName || got.CredentialKind != single.CredentialKind {
			t.Errorf("list and get disagree for %s: %+v vs %+v", want.ID, got, single)
		}
	}
}
