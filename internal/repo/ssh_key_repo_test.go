package repo

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

var sshKeyTestSchema = testsupport.SchemaName("apiary_repo_ssh_key_test")

func sshKeyDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, sshKeyTestSchema, &tenant.SshKey{})
}

func sshKeyRepo(t *testing.T, db *gorm.DB, orgID uuid.UUID) *SSHKeyRepoGorm {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	r, err := NewSSHKeyRepoGorm(db, orgID, uuid.New(), key)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	return r
}

// The private half must be usable by River and by nothing else. If it is stored
// in clear, a read of the tenant database is root on that customer's fleet.
func TestMintSealsThePrivateKeyAndReturnsAUsablePublicOne(t *testing.T) {
	db := sshKeyDB(t)
	r := sshKeyRepo(t, db, uuid.New())

	row, err := r.Mint(context.Background(), "fleet onboarding")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	// The public half must parse as an authorized_keys line, or it cannot be
	// whitelisted by templating it onto a host.
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(row.PublicKey))
	if err != nil {
		t.Fatalf("public key is not an authorized_keys line: %v", err)
	}
	if row.Fingerprint != ssh.FingerprintSHA256(pub) {
		t.Fatalf("fingerprint %q does not match the public key; an operator comparing it against "+
			"`ssh-keygen -lf` on the hypervisor would be checking nothing", row.Fingerprint)
	}
	if !strings.HasPrefix(row.Fingerprint, "SHA256:") {
		t.Fatalf("fingerprint %q is not in OpenSSH's own form, so it will not compare directly "+
			"against what the host reports", row.Fingerprint)
	}

	// Nothing recognisable as a private key may be in the stored row.
	var stored tenant.SshKey
	if err := db.First(&stored, "id = ?", row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if strings.Contains(stored.EncryptedPrivateKey, "PRIVATE KEY") {
		t.Fatal("the private key is stored in clear; a read of this table is root on the fleet")
	}
	if stored.PrivateKeyID == uuid.Nil {
		t.Fatal("no key id recorded, so a rotation cannot tell a re-wrapped row from one it has " +
			"not reached")
	}

	// And it round-trips into something OpenSSH accepts.
	priv, err := r.PrivateKey(context.Background(), row.ID)
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(priv)
	if err != nil {
		t.Fatalf("unsealed private key does not parse: %v", err)
	}
	if !bytes.Equal(ssh.MarshalAuthorizedKey(signer.PublicKey()), ssh.MarshalAuthorizedKey(pub)) {
		t.Fatal("the unsealed private key does not match the stored public key")
	}
}

// The AAD binds a sealed key to its row AND its tenant. Without it a ciphertext
// copied from one row to another would open, which for this key means one
// tenant's onboarding key becoming another's.
func TestASealedKeyDoesNotOpenAgainstAnotherRowOrTenant(t *testing.T) {
	db := sshKeyDB(t)
	orgA := uuid.New()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	keyID := uuid.New()

	repoA, err := NewSSHKeyRepoGorm(db, orgA, keyID, key)
	if err != nil {
		t.Fatal(err)
	}
	rowA, err := repoA.Mint(context.Background(), "a")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	rowB, err := repoA.Mint(context.Background(), "b")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	// Move A's ciphertext onto B's row — the attack an AAD exists to stop.
	if err := db.Model(&tenant.SshKey{}).Where("id = ?", rowB.ID).Updates(map[string]any{
		"encrypted_private_key": rowA.EncryptedPrivateKey,
		"private_iv":            rowA.PrivateIV,
		"private_tag":           rowA.PrivateTag,
	}).Error; err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if _, err := repoA.PrivateKey(context.Background(), rowB.ID); err == nil {
		t.Fatal("a private key moved between rows still opened; the AAD is not binding the row id")
	}

	// And the same row read as a different tenant.
	repoB, err := NewSSHKeyRepoGorm(db, uuid.New(), keyID, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repoB.PrivateKey(context.Background(), rowA.ID); err == nil {
		t.Fatal("another tenant opened this key; the AAD is not binding the organization")
	}
}

func TestARevokedKeyCannotBeUnsealed(t *testing.T) {
	db := sshKeyDB(t)
	r := sshKeyRepo(t, db, uuid.New())

	row, err := r.Mint(context.Background(), "rotate me")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if err := r.Revoke(context.Background(), row.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := r.PrivateKey(context.Background(), row.ID); !errors.Is(err, ErrSSHKeyRevoked) {
		t.Fatalf("err = %v, want ErrSSHKeyRevoked — a revoked key must not still reach a host", err)
	}

	// The row survives, so "which key onboarded this hypervisor" stays answerable.
	//
	// Asserted by finding THIS key rather than by counting: the tests in this
	// package share one schema, so a total is a count of whatever else has run.
	// That distinction is the difference between a test that passes alone and
	// fails in a suite, which is the worst kind to leave behind.
	keys, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.ID == row.ID {
			found = true
			if k.RevokedAt == nil {
				t.Fatal("the key is listed but not marked revoked")
			}
		}
	}
	if !found {
		t.Fatal("a revoked key should be kept, not deleted — otherwise which key onboarded a " +
			"hypervisor stops being answerable after a rotation")
	}
}

// The construction guards are the reason no method can be handed the wrong
// tenant's key: there is no way to build a repo without naming one.
func TestTheRepoRefusesToBeBuiltWithoutATenantAndKey(t *testing.T) {
	db := sshKeyDB(t)
	key := []byte("0123456789abcdef0123456789abcdef")
	for name, fn := range map[string]func() (*SSHKeyRepoGorm, error){
		"no db":     func() (*SSHKeyRepoGorm, error) { return NewSSHKeyRepoGorm(nil, uuid.New(), uuid.New(), key) },
		"no org":    func() (*SSHKeyRepoGorm, error) { return NewSSHKeyRepoGorm(db, uuid.Nil, uuid.New(), key) },
		"no key id": func() (*SSHKeyRepoGorm, error) { return NewSSHKeyRepoGorm(db, uuid.New(), uuid.Nil, key) },
		"no key":    func() (*SSHKeyRepoGorm, error) { return NewSSHKeyRepoGorm(db, uuid.New(), uuid.New(), nil) },
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := fn(); err == nil {
				t.Fatal("built a repo that cannot seal or unseal safely")
			}
		})
	}
}
