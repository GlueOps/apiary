package cmd

import (
	"bytes"
	"testing"

	"github.com/glueops/apiary/internal/utils"
)

func resealTestKeys(t *testing.T) (oldKey, newKey []byte) {
	t.Helper()
	master := make([]byte, utils.MasterKeyLen)
	for i := range master {
		master[i] = 0x27
	}
	var err error
	if oldKey, err = utils.DeriveKey(master, legacyPurposeTenantDSN); err != nil {
		t.Fatalf("derive old: %v", err)
	}
	if newKey, err = utils.DeriveKey(master, utils.KeyPurposeTenantDSN); err != nil {
		t.Fatalf("derive new: %v", err)
	}
	if bytes.Equal(oldKey, newKey) {
		t.Fatal("the legacy and current purposes derive the same key, so this migration is a no-op " +
			"and one of the constants has been edited to match the other")
	}
	return oldKey, newKey
}

// openEither is what makes the tool idempotent AND safe to use as a checker.
// Getting it wrong in either direction is bad: too eager and a re-run
// double-wraps already-migrated data, too strict and a partially-completed run
// can never be finished.
func TestOpenEitherDistinguishesStaleFromMigrated(t *testing.T) {
	oldKey, newKey := resealTestKeys(t)
	aad := []byte("11111111-1111-1111-1111-111111111111")
	secret := []byte("postgres://tenant:hunter2@db/tenant_x")

	t.Run("sealed under the previous purpose", func(t *testing.T) {
		ct, iv, tag, err := utils.EncryptAESGCMWithAAD(secret, oldKey, aad)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		got, current, err := openEither(ct, iv, tag, oldKey, newKey, aad)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if current {
			t.Fatal("reported as already migrated; the tool would skip it and leave it unreadable")
		}
		if !bytes.Equal(got, secret) {
			t.Fatal("recovered the wrong plaintext")
		}
	})

	t.Run("already sealed under the current purpose", func(t *testing.T) {
		ct, iv, tag, err := utils.EncryptAESGCMWithAAD(secret, newKey, aad)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		got, current, err := openEither(ct, iv, tag, oldKey, newKey, aad)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if !current {
			t.Fatal("an already-migrated value was reported as stale; a re-run would re-wrap it, " +
				"and the tool would never converge")
		}
		if !bytes.Equal(got, secret) {
			t.Fatal("recovered the wrong plaintext")
		}
	})

	t.Run("sealed under neither", func(t *testing.T) {
		other := make([]byte, len(oldKey))
		copy(other, oldKey)
		other[0] ^= 0xFF
		ct, iv, tag, err := utils.EncryptAESGCMWithAAD(secret, other, aad)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if _, _, err := openEither(ct, iv, tag, oldKey, newKey, aad); err == nil {
			t.Fatal("a value that opens under neither derivation was accepted; the tool would " +
				"write back garbage and destroy the only copy")
		}
	})

	t.Run("wrong AAD is not silently accepted", func(t *testing.T) {
		ct, iv, tag, err := utils.EncryptAESGCMWithAAD(secret, oldKey, aad)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		// The AAD is the organization id: this is a value from a DIFFERENT
		// tenant. Accepting it would re-wrap one org's DSN onto another's row.
		if _, _, err := openEither(ct, iv, tag, oldKey, newKey, []byte("22222222-2222-2222-2222-222222222222")); err == nil {
			t.Fatal("opened with the wrong organization's AAD; the tool could move one tenant's " +
				"connection string onto another tenant's row")
		}
	})
}
