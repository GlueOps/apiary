package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

type akFakeRepo struct {
	byHash  map[string]*control.APIKey
	touched int
}

func newAKRepo() *akFakeRepo { return &akFakeRepo{byHash: map[string]*control.APIKey{}} }

func (f *akFakeRepo) Create(_ context.Context, k *control.APIKey) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	f.byHash[k.KeyHash] = k
	return nil
}
func (f *akFakeRepo) ByKeyHash(_ context.Context, h string) (*control.APIKey, error) {
	if k, ok := f.byHash[h]; ok {
		return k, nil
	}
	return nil, repo.ErrAPIKeyNotFound
}
func (f *akFakeRepo) TouchLastUsed(context.Context, uuid.UUID, time.Time) error {
	f.touched++
	return nil
}
func (f *akFakeRepo) ListForOrg(context.Context, uuid.UUID) ([]control.APIKey, error) {
	return nil, nil
}
func (f *akFakeRepo) ListForUser(context.Context, uuid.UUID) ([]control.APIKey, error) {
	return nil, nil
}
func (f *akFakeRepo) Revoke(context.Context, uuid.UUID) error { return nil }

func mint(t *testing.T, s *APIKeyService, scope string) *MintResult {
	t.Helper()
	in := MintInput{Scope: scope, Name: "test"}
	org := uuid.New()
	user := uuid.New()
	switch scope {
	case control.APIKeyScopeUser:
		in.UserID = &user
	default:
		in.OrgID = &org
	}
	res, err := s.Mint(context.Background(), in)
	if err != nil {
		t.Fatalf("mint %s: %v", scope, err)
	}
	return res
}

// The surface exclusion is the point of Scope, and it runs BOTH ways. Only
// restricting the agent would leave an org key able to enrol, heartbeat and
// report as a fleet agent — a way to forge the fleet's view of itself.
func TestAPIKeyScopesAreMutuallyExclusiveSurfaces(t *testing.T) {
	svc := NewAPIKeyService(newAKRepo())
	orgSurface := []string{control.APIKeyScopeUser, control.APIKeyScopeOrg}
	agentSurface := []string{control.APIKeyScopeAgent}

	agent := mint(t, svc, control.APIKeyScopeAgent)
	org := mint(t, svc, control.APIKeyScopeOrg)
	user := mint(t, svc, control.APIKeyScopeUser)

	t.Run("an agent key is refused on the org surface", func(t *testing.T) {
		_, err := svc.Authenticate(context.Background(), agent.Public, agent.Secret, orgSurface...)
		if !errors.Is(err, ErrAPIKeyWrongSurface) {
			t.Fatalf("err = %v, want ErrAPIKeyWrongSurface — a compromised hypervisor could "+
				"otherwise turn its credential on the rest of the tenant's API", err)
		}
	})

	t.Run("an org key is refused on the agent surface", func(t *testing.T) {
		_, err := svc.Authenticate(context.Background(), org.Public, org.Secret, agentSurface...)
		if !errors.Is(err, ErrAPIKeyWrongSurface) {
			t.Fatalf("err = %v, want ErrAPIKeyWrongSurface — a human or an automation could "+
				"otherwise present itself as a fleet agent", err)
		}
	})

	t.Run("a user key is refused on the agent surface", func(t *testing.T) {
		_, err := svc.Authenticate(context.Background(), user.Public, "", agentSurface...)
		if !errors.Is(err, ErrAPIKeyWrongSurface) {
			t.Fatalf("err = %v, want ErrAPIKeyWrongSurface", err)
		}
	})

	t.Run("each is admitted on its own surface", func(t *testing.T) {
		if _, err := svc.Authenticate(context.Background(), agent.Public, agent.Secret, agentSurface...); err != nil {
			t.Fatalf("agent key on the agent surface: %v", err)
		}
		if _, err := svc.Authenticate(context.Background(), org.Public, org.Secret, orgSurface...); err != nil {
			t.Fatalf("org key on the org surface: %v", err)
		}
		if _, err := svc.Authenticate(context.Background(), user.Public, "", orgSurface...); err != nil {
			t.Fatalf("user key on the org surface: %v", err)
		}
	})
}

// The secret half must actually be checked. Presenting the public half alone
// would make an org key a bearer token that is also printed in listings.
func TestAPIKeyPairRequiresItsSecret(t *testing.T) {
	svc := NewAPIKeyService(newAKRepo())
	surfaces := []string{control.APIKeyScopeOrg}
	org := mint(t, svc, control.APIKeyScopeOrg)

	for name, secret := range map[string]string{
		"no secret":    "",
		"wrong secret": "0000",
		"public half":  org.Public,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Authenticate(context.Background(), org.Public, secret, surfaces...); !errors.Is(err, ErrAPIKeyRejected) {
				t.Fatalf("err = %v, want ErrAPIKeyRejected", err)
			}
		})
	}
}

// A user key carries no secret half, so accepting one anyway would leave a
// caller unable to tell which credential shape they hold.
func TestAUserKeyRefusesASecret(t *testing.T) {
	svc := NewAPIKeyService(newAKRepo())
	user := mint(t, svc, control.APIKeyScopeUser)
	if _, err := svc.Authenticate(context.Background(), user.Public, "anything", control.APIKeyScopeUser); !errors.Is(err, ErrAPIKeyRejected) {
		t.Fatalf("err = %v, want ErrAPIKeyRejected", err)
	}
}

func TestRevokedAndExpiredKeysAreRefused(t *testing.T) {
	r := newAKRepo()
	svc := NewAPIKeyService(r)
	surfaces := []string{control.APIKeyScopeOrg}

	revoked := mint(t, svc, control.APIKeyScopeOrg)
	revoked.Key.Revoked = true
	if _, err := svc.Authenticate(context.Background(), revoked.Public, revoked.Secret, surfaces...); !errors.Is(err, ErrAPIKeyRejected) {
		t.Fatalf("a revoked key authenticated: %v", err)
	}

	expired := mint(t, svc, control.APIKeyScopeOrg)
	past := time.Now().UTC().Add(-time.Minute)
	expired.Key.ExpiresAt = &past
	if _, err := svc.Authenticate(context.Background(), expired.Public, expired.Secret, surfaces...); !errors.Is(err, ErrAPIKeyRejected) {
		t.Fatalf("an expired key authenticated: %v", err)
	}
}

// A user key bound to an organization would silently bypass the membership
// check that X-Org-Id exists to force.
func TestMintRefusesShapesThatContradictTheirScope(t *testing.T) {
	svc := NewAPIKeyService(newAKRepo())
	org := uuid.New()
	user := uuid.New()

	if _, err := svc.Mint(context.Background(), MintInput{
		Scope: control.APIKeyScopeUser, UserID: &user, OrgID: &org,
	}); err == nil {
		t.Fatal("minted a user key bound to an organization; that key would act in that org " +
			"without the membership check X-Org-Id forces")
	}
	if _, err := svc.Mint(context.Background(), MintInput{Scope: control.APIKeyScopeOrg}); err == nil {
		t.Fatal("minted an org key with no organization")
	}
	if _, err := svc.Mint(context.Background(), MintInput{Scope: "root"}); err == nil {
		t.Fatal("minted a key with an unknown scope; a credential whose surface cannot be " +
			"determined must be refused, not treated as unrestricted")
	}
}

// Only hashes are stored. A dump of api_keys must not yield working credentials.
func TestOnlyHashesAreStored(t *testing.T) {
	r := newAKRepo()
	svc := NewAPIKeyService(r)
	org := mint(t, svc, control.APIKeyScopeOrg)

	for _, k := range r.byHash {
		if k.KeyHash == org.Public {
			t.Fatal("the public half is stored in clear")
		}
		if k.SecretHash != nil && *k.SecretHash == org.Secret {
			t.Fatal("the secret half is stored in clear")
		}
		if k.Prefix == nil || len(*k.Prefix) >= len(org.Public) {
			t.Fatal("the prefix should identify the key without being usable as it")
		}
	}
}
