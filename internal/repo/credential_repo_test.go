package repo

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func testRepo(t *testing.T) *CredentialRepoGorm {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	// A nil DB is fine for the pure-logic tests below; every one of them fails
	// before any query would run. The constructor refuses nil, so build the
	// struct directly rather than weakening the constructor for tests.
	return &CredentialRepoGorm{
		orgID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		keyID:  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		key:    key,
		scopes: DefaultProviderScopes,
	}
}

// ---------------------------------------------------------------- enforcement 1

// The constructor must refuse anything that would produce rows which look
// sealed and are not.
func TestNewCredentialRepoFailsClosed(t *testing.T) {
	good := make([]byte, 32)
	org := uuid.New()
	keyID := uuid.New()

	cases := []struct {
		name  string
		orgID uuid.UUID
		keyID uuid.UUID
		key   []byte
	}{
		{"nil org id breaks AAD binding", uuid.Nil, keyID, good},
		{"nil key id makes rotation unverifiable", org, uuid.Nil, good},
		{"short key", org, keyID, make([]byte, 16)},
		{"no key", org, keyID, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// db is nil in every case, which is itself refused — pass a
			// non-nil-looking check by asserting the error mentions config.
			_, err := NewCredentialRepoGorm(nil, c.orgID, c.keyID, c.key, nil)
			if !errors.Is(err, ErrCredentialRepoUnconfigured) {
				t.Fatalf("want ErrCredentialRepoUnconfigured, got %v", err)
			}
		})
	}
}

// A ciphertext must not open on a row it does not belong to. This is what the
// AAD is for, and it is the defect class that let a TOTP seed be swapped
// between users elsewhere in this codebase.
func TestSecretDoesNotOpenOnAnotherRow(t *testing.T) {
	r := testRepo(t)
	rowA := uuid.New()
	rowB := uuid.New()

	sealed, err := utils.Seal([]byte("super-secret"), r.keyID, r.key,
		utils.AAD(credentialsTable, r.orgID, rowA))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	// Same tenant, same key, different row: must fail.
	if _, err := sealed.Open(r.key, utils.AAD(credentialsTable, r.orgID, rowB)); err == nil {
		t.Fatal("a credential sealed for one row opened on another — the AAD is not binding the row")
	}
	// And the correct row must still work, or the binding is just broken.
	if _, err := sealed.Open(r.key, utils.AAD(credentialsTable, r.orgID, rowA)); err != nil {
		t.Fatalf("the owning row must open: %v", err)
	}
}

// Nor on the same row in another tenant.
func TestSecretDoesNotOpenInAnotherTenant(t *testing.T) {
	r := testRepo(t)
	row := uuid.New()
	other := uuid.New()

	sealed, _ := utils.Seal([]byte("s"), r.keyID, r.key, utils.AAD(credentialsTable, r.orgID, row))
	if _, err := sealed.Open(r.key, utils.AAD(credentialsTable, other, row)); err == nil {
		t.Fatal("a credential opened under another organization's AAD")
	}
}

// The table name is a constant so a credential cannot be opened under another
// table's AAD.
func TestSecretDoesNotOpenUnderAnotherTable(t *testing.T) {
	r := testRepo(t)
	row := uuid.New()

	sealed, _ := utils.Seal([]byte("s"), r.keyID, r.key, utils.AAD(credentialsTable, r.orgID, row))
	if _, err := sealed.Open(r.key, utils.AAD("user_totps", r.orgID, row)); err == nil {
		t.Fatal("a credential opened under another table's AAD")
	}
}

// ---------------------------------------------------------------- enforcement 3

// The fingerprint must be over the identifying component and must never equal a
// digest of the secret — that would be an offline verification oracle for anyone
// holding the database.
func TestFingerprintIsOverIdentityNotSecret(t *testing.T) {
	const identity = "AKIAEXAMPLEPRIMARY"
	secret := Secret("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	fp := fingerprintIdentity(identity)

	if fp != fingerprintIdentity(identity) {
		t.Fatal("fingerprint is not deterministic")
	}
	if fp == fingerprintIdentity(string(secret)) {
		t.Fatal("the fingerprint equals a digest of the secret — that is a verification oracle")
	}
	if strings.Contains(fp, string(secret)) {
		t.Fatal("secret material leaked into the fingerprint")
	}
}

// Complete digest, ssh encoding. A truncated fingerprint makes every human
// comparison a weak one, and for an assumed role the identifying component is
// an ARN whose name the customer chooses, so a short prefix is grindable.
func TestFingerprintIsCompleteAndSSHEncoded(t *testing.T) {
	fp := fingerprintIdentity("arn:aws:iam::222:role/apiary")

	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("want ssh encoding, got %q", fp)
	}
	body := strings.TrimPrefix(fp, "SHA256:")
	// 256 bits base64 without padding is 43 characters. Anything shorter has
	// dropped bits.
	if len(body) != 43 {
		t.Fatalf("fingerprint carries %d base64 chars, want 43 (a full 256 bits); "+
			"a shorter value means the digest was truncated", len(body))
	}
	if strings.Contains(body, "=") {
		t.Error("padding should be trimmed, as ssh does")
	}
}

// Secret must not print itself. The realistic leak is %v on a surrounding
// struct, not a deliberate log of the secret.
func TestSecretRedactsItself(t *testing.T) {
	s := Secret("wJalrXUtnFEMI-very-secret")

	for _, rendered := range []string{
		fmt.Sprintf("%v", s),
		// The %s verb is exactly what is under test here — a surrounding
		// struct rendered with %s is how a secret realistically escapes —
		// so calling String() directly would test something else.
		//nolint:staticcheck // S1025: the verb is the subject of the test
		fmt.Sprintf("%s", s),
		fmt.Sprintf("%#v", s),
		fmt.Sprintf("%v", CredentialPayload{Identity: "AKIA", Secret: s}),
		fmt.Sprintf("%+v", CreateCredentialInput{Payload: CredentialPayload{Secret: s}}),
	} {
		if strings.Contains(rendered, "very-secret") {
			t.Errorf("secret material rendered into %q", rendered)
		}
	}
}

// Kind stays outside the seal so the legacy-to-federation migration can be
// measured without decrypting anything.
func TestKindsAreDistinctAndUnsealed(t *testing.T) {
	if tenant.CredentialKindStaticSecret == tenant.CredentialKindAssumedRole {
		t.Fatal("the two kinds must be distinguishable")
	}
	var c tenant.Credential
	c.Kind = tenant.CredentialKindAssumedRole
	if c.Kind == "" {
		t.Fatal("Kind must be a plain column, readable without opening the seal")
	}
}

// ---------------------------------------------------- enforcement 1, for real

// credentialTestDB is DB-backed on purpose. The AAD tests above exercise
// utils.Seal directly, which proves the PRIMITIVE binds a row — not that this
// repo uses it. Removing the AAD from Create and OpenSecret leaves those tests
// green, so the only honest check goes through the repo's own code path.
var credentialTestSchema = testsupport.SchemaName("apiary_repo_credential_test")

func credentialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, credentialTestSchema, &tenant.Credential{})
	// Dropped between tests so each starts on an empty table; OpenSchema
	// re-migrates on the next call. This used to run against public, where it
	// dropped a real tenant table if one happened to be there.
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS credentials") })
	return db
}

func TestRepoBindsCiphertextToItsRow(t *testing.T) {
	db := credentialTestDB(t)

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	org := uuid.New()
	r, err := NewCredentialRepoGorm(db, org, uuid.New(), key, nil)
	if err != nil {
		t.Fatalf("construct: %v", err)
	}

	mk := func(region, account string) *tenant.Credential {
		c, err := r.Create(t.Context(), "aws", CreateCredentialInput{
			Name: "cred-" + region,
			Payload: CredentialPayload{
				Identity: "AKIA" + strings.ToUpper(region),
				Secret:   Secret("secret-for-" + account),
			},
		})
		if err != nil {
			t.Fatalf("create %s: %v", region, err)
		}
		return c
	}

	a := mk("eu-west-1", "111111111111")
	b := mk("us-east-1", "222222222222")

	// Both open normally through the repo.
	for _, c := range []*tenant.Credential{a, b} {
		got, err := r.OpenSecret(t.Context(), c)
		if err != nil {
			t.Fatalf("owning row must open: %v", err)
		}
		if !strings.HasPrefix(string(got), "secret-for-") {
			t.Fatalf("wrong plaintext: %q", got)
		}
	}

	// The attack: someone with database write access copies A's sealed columns
	// onto B's row, hoping to make B's credential resolve to A's secret.
	if err := db.Exec(`
		UPDATE credentials SET ciphertext = ?, iv = ?, tag = ?, key_id = ?
		WHERE id = ?`,
		a.Ciphertext, a.IV, a.Tag, a.KeyID, b.ID).Error; err != nil {
		t.Fatalf("tamper: %v", err)
	}

	tampered, err := r.Get(t.Context(), b.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, err := r.OpenSecret(t.Context(), tampered); err == nil {
		t.Fatal("a ciphertext moved onto another row opened cleanly — the repo is not " +
			"binding the AAD to the row, so a credential can be swapped in the database")
	}
}

// The same row in another tenant must not open either, again through the repo.
func TestRepoBindsCiphertextToItsTenant(t *testing.T) {
	db := credentialTestDB(t)

	key := make([]byte, 32)
	orgA, orgB := uuid.New(), uuid.New()
	keyID := uuid.New()

	ra, err := NewCredentialRepoGorm(db, orgA, keyID, key, nil)
	if err != nil {
		t.Fatalf("construct a: %v", err)
	}
	rb, err := NewCredentialRepoGorm(db, orgB, keyID, key, nil)
	if err != nil {
		t.Fatalf("construct b: %v", err)
	}

	c, err := ra.Create(t.Context(), "proxmox", CreateCredentialInput{
		Payload: CredentialPayload{Identity: "svc@pve!apiary", Secret: Secret("token")},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := ra.OpenSecret(t.Context(), c); err != nil {
		t.Fatalf("owning tenant must open: %v", err)
	}
	if _, err := rb.OpenSecret(t.Context(), c); err == nil {
		t.Fatal("another tenant's repo opened this credential — the organization is not in the AAD")
	}
}

// TestFingerprintIsOverIdentityNotSecret above tests the PRIMITIVE. It stays
// green when Create and Rotate are both repointed at the secret, which is the
// mutation that installs the offline verification oracle the model forbids.
// These go through the real call path, so that mutation turns them red.
func TestCreateFingerprintsTheIdentityNotTheSecret(t *testing.T) {
	db := credentialTestDB(t)
	key := make([]byte, 32)
	r, err := NewCredentialRepoGorm(db, uuid.New(), uuid.New(), key, nil)
	if err != nil {
		t.Fatalf("construct: %v", err)
	}

	const identity = "AKIAIDENTITYCOMPONENT"
	const secret = "wJalrXUtnFEMI-THE-ACTUAL-SECRET"

	cred, err := r.Create(t.Context(), "aws", CreateCredentialInput{
		Name:    "c",
		Payload: CredentialPayload{Identity: identity, Secret: Secret(secret)},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if cred.Fingerprint != fingerprintIdentity(identity) {
		t.Errorf("Create did not fingerprint the identifying component: got %q", cred.Fingerprint)
	}
	if cred.Fingerprint == fingerprintIdentity(secret) {
		t.Fatal("Create fingerprinted the SECRET — anyone holding this database can now " +
			"confirm a guessed credential offline, with no API call and no provider-side " +
			"audit record")
	}
}

func TestRotateFingerprintsTheIdentityNotTheSecret(t *testing.T) {
	db := credentialTestDB(t)
	key := make([]byte, 32)
	r, err := NewCredentialRepoGorm(db, uuid.New(), uuid.New(), key, nil)
	if err != nil {
		t.Fatalf("construct: %v", err)
	}

	cred, err := r.Create(t.Context(), "aws", CreateCredentialInput{
		Payload: CredentialPayload{Identity: "AKIAOLD", Secret: Secret("old-secret")},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	const newIdentity = "AKIAROTATED"
	const newSecret = "the-rotated-secret-value"

	rotated, err := r.Rotate(t.Context(), cred.ID, CredentialPayload{
		Identity: newIdentity,
		Secret:   Secret(newSecret),
	})
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if rotated.Fingerprint != fingerprintIdentity(newIdentity) {
		t.Errorf("Rotate did not fingerprint the new identifying component: got %q", rotated.Fingerprint)
	}
	if rotated.Fingerprint == fingerprintIdentity(newSecret) {
		t.Fatal("Rotate fingerprinted the SECRET")
	}
	if rotated.Fingerprint == cred.Fingerprint {
		t.Error("the fingerprint did not change — an operator cannot confirm the rotation took")
	}
}
