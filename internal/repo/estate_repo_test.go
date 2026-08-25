package repo

import (
	"errors"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func testEstateRepo(t *testing.T) *EstateRepoGorm {
	t.Helper()
	// A nil handle is fine here: every scope test below fails before a query
	// would run. The constructor refuses nil, so build the struct directly
	// rather than weakening the constructor for tests.
	return &EstateRepoGorm{scopes: DefaultProviderScopes}
}

// Scope canonicalisation: the estate says WHERE, so this is where a target is
// validated. It moved here from the credential repo along with Scope itself.

func TestScopeRejectsUnknownKeys(t *testing.T) {
	r := testEstateRepo(t)
	// "acount" is a typo. jsonb would accept it happily as a distinct scope, and
	// the credential would then be one that no request ever selects.
	_, err := r.canonicalScope("aws",
		map[string]string{"region": "eu-west-1", "acount": "111"},
		map[string]string{"account": "111"})
	if !errors.Is(err, ErrScopeKeyUnknown) {
		t.Fatalf("want ErrScopeKeyUnknown for a typo'd key, got %v", err)
	}
}

func TestScopeRefusesWhenAssertionDisagreesWithDerivation(t *testing.T) {
	r := testEstateRepo(t)
	// The operator says this is the staging account; the credential itself says
	// otherwise. This is a production key pasted into a staging slot.
	_, err := r.canonicalScope("aws",
		map[string]string{"region": "eu-west-1", "account": "999999999999"},
		map[string]string{"account": "111111111111"})
	if !errors.Is(err, ErrScopeDisagrees) {
		t.Fatalf("want ErrScopeDisagrees when the asserted account is not the real one, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "999999999999") {
		t.Errorf("the error should name both values so an operator can see the mistake: %v", err)
	}
}

func TestScopeAcceptsAgreementAndMergesDerivedKeys(t *testing.T) {
	r := testEstateRepo(t)
	got, err := r.canonicalScope("aws",
		map[string]string{"region": "eu-west-1", "service": "ec2"},
		map[string]string{"account": "111111111111"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["account"] != "111111111111" {
		t.Errorf("derived key was not merged in: %v", got)
	}
}

// A derivable key is not an optional key: omit it and a primary/DR pair in two
// accounts collapses to one scope and collides on insert.
func TestScopeRefusesWhenADerivableKeyIsAbsent(t *testing.T) {
	r := testEstateRepo(t)
	_, err := r.canonicalScope("aws",
		map[string]string{"region": "eu-west-1", "service": "ec2"},
		nil)
	if !errors.Is(err, ErrScopeNotCanonical) {
		t.Fatalf("want ErrScopeNotCanonical when the account was neither supplied nor derived, got %v", err)
	}
}

func TestScopeRequiresRequiredKeys(t *testing.T) {
	r := testEstateRepo(t)
	if _, err := r.canonicalScope("proxmox", map[string]string{}, nil); !errors.Is(err, ErrScopeKeyMissing) {
		t.Fatalf("want ErrScopeKeyMissing, got %v", err)
	}
}

func TestScopeRefusesUnknownProvider(t *testing.T) {
	r := testEstateRepo(t)
	if _, err := r.canonicalScope("nimbus", map[string]string{"x": "y"}, nil); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("an unregistered provider must be refused rather than stored unvalidated, got %v", err)
	}
}

// Proxmox must not be forced into AWS vocabulary, and two clusters must coexist.
func TestScopeIsPerProviderNotOneShape(t *testing.T) {
	r := testEstateRepo(t)
	for _, cluster := range []string{"dc1", "dc2"} {
		got, err := r.canonicalScope("proxmox", map[string]string{"cluster": cluster}, nil)
		if err != nil {
			t.Fatalf("proxmox %s: %v", cluster, err)
		}
		if _, hasAccount := got["account"]; hasAccount {
			t.Error("proxmox scope must not carry AWS keys")
		}
	}
}

// ---------------------------------------------------------- the binding itself

var estateTestSchema = testsupport.SchemaName("apiary_repo_estate_test")

func estateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testsupport.OpenSchema(t, estateTestSchema, &tenant.Credential{}, &tenant.Estate{})
	// estates before credentials: the foreign key runs that way.
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS estates")
		db.Exec("DROP TABLE IF EXISTS credentials")
	})
	return db
}

func newCredential(t *testing.T, db *gorm.DB, provider, identity string) *tenant.Credential {
	t.Helper()
	key := make([]byte, 32)
	r, err := NewCredentialRepoGorm(db, uuid.New(), uuid.New(), key, nil)
	if err != nil {
		t.Fatalf("credential repo: %v", err)
	}
	c, err := r.Create(t.Context(), provider, CreateCredentialInput{
		Name:    identity,
		Payload: CredentialPayload{Identity: identity, Secret: Secret("s")},
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	return c
}

// This is the case that was impossible before scope moved. Two VPCs in ONE aws
// account and region are two estates, each with its own credential — and under
// UNIQUE (provider, scope) on credentials the second was rejected outright.
func TestTwoEstatesInOneAccountAndRegion(t *testing.T) {
	db := estateTestDB(t)
	r, err := NewEstateRepoGorm(db, nil)
	if err != nil {
		t.Fatalf("estate repo: %v", err)
	}

	for _, vpc := range []string{"vpc-prod", "vpc-staging"} {
		cred := newCredential(t, db, "aws", "AKIA"+strings.ToUpper(vpc))
		_, err := r.Create(t.Context(), "aws", CreateEstateInput{
			Name:          vpc,
			AssertedScope: map[string]string{"region": "eu-west-1", "vpc": vpc},
			DerivedScope:  map[string]string{"account": "111111111111"},
			CredentialID:  cred.ID,
		})
		if err != nil {
			t.Fatalf("estate %s in a shared account and region must be storable: %v", vpc, err)
		}
	}
}

// One estate per target, though — that is what the constraint means now.
func TestTwoEstatesCannotShareOneTarget(t *testing.T) {
	db := estateTestDB(t)
	r, _ := NewEstateRepoGorm(db, nil)

	scope := map[string]string{"region": "eu-west-1", "vpc": "vpc-prod"}
	derived := map[string]string{"account": "111111111111"}

	c1 := newCredential(t, db, "aws", "AKIAONE")
	if _, err := r.Create(t.Context(), "aws", CreateEstateInput{
		Name: "prod", AssertedScope: scope, DerivedScope: derived, CredentialID: c1.ID,
	}); err != nil {
		t.Fatalf("first estate: %v", err)
	}

	c2 := newCredential(t, db, "aws", "AKIATWO")
	if _, err := r.Create(t.Context(), "aws", CreateEstateInput{
		Name: "prod-again", AssertedScope: scope, DerivedScope: derived, CredentialID: c2.ID,
	}); err == nil {
		t.Fatal("two estates claimed the same target — UNIQUE (provider, scope) is not in force")
	}
}

// The 1:1 binding: a credential belongs to exactly one estate, or a leak's blast
// radius stops being bounded by one estate.
func TestOneCredentialCannotServeTwoEstates(t *testing.T) {
	db := estateTestDB(t)
	r, _ := NewEstateRepoGorm(db, nil)
	cred := newCredential(t, db, "proxmox", "svc@pve!apiary")

	if _, err := r.Create(t.Context(), "proxmox", CreateEstateInput{
		Name: "dc1", AssertedScope: map[string]string{"cluster": "dc1"}, CredentialID: cred.ID,
	}); err != nil {
		t.Fatalf("first estate: %v", err)
	}
	if _, err := r.Create(t.Context(), "proxmox", CreateEstateInput{
		Name: "dc2", AssertedScope: map[string]string{"cluster": "dc2"}, CredentialID: cred.ID,
	}); err == nil {
		t.Fatal("one credential was bound to two estates — the 1:1 binding is not enforced")
	}
}

// Provider is stored on both records because a credential predates the estate
// that binds it, so disagreement is representable and must be refused.
func TestEstateRefusesAProviderMismatch(t *testing.T) {
	db := estateTestDB(t)
	r, _ := NewEstateRepoGorm(db, nil)
	cred := newCredential(t, db, "proxmox", "svc@pve!apiary")

	_, err := r.Create(t.Context(), "aws", CreateEstateInput{
		Name:          "wrong",
		AssertedScope: map[string]string{"region": "eu-west-1"},
		DerivedScope:  map[string]string{"account": "111111111111"},
		CredentialID:  cred.ID,
	})
	if !errors.Is(err, ErrProviderMismatch) {
		t.Fatalf("an aws estate holding a proxmox credential must be refused, got %v", err)
	}
}
