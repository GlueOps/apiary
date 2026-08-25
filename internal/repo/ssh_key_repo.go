package repo

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

var (
	ErrSSHKeyNotFound = errors.New("ssh key: not found")
	ErrSSHKeyRevoked  = errors.New("ssh key: has been revoked")
)

// sshKeysTable is the AAD domain separator, matching the pattern credentials
// use. It keeps a sealed private key from opening against a row in another
// table even under the same tenant key.
const sshKeysTable = "ssh_keys"

// SSHKeyRepoGorm stores a tenant's onboarding keypairs.
//
// Bound to one organization and one key at construction, like CredentialRepoGorm
// — so there is no method that could be handed the wrong tenant's DEK, and no
// call site that has to remember to pass the right one.
type SSHKeyRepoGorm struct {
	db    *gorm.DB
	orgID uuid.UUID
	keyID uuid.UUID
	key   []byte
}

func NewSSHKeyRepoGorm(db *gorm.DB, orgID, keyID uuid.UUID, key []byte) (*SSHKeyRepoGorm, error) {
	switch {
	case db == nil:
		return nil, errors.New("ssh key repo: database handle is nil")
	case orgID == uuid.Nil:
		return nil, errors.New("ssh key repo: organization is required")
	case keyID == uuid.Nil:
		return nil, errors.New("ssh key repo: key id is required")
	case len(key) == 0:
		return nil, errors.New("ssh key repo: encryption key is required")
	}
	return &SSHKeyRepoGorm{db: db, orgID: orgID, keyID: keyID, key: key}, nil
}

// Mint generates a keypair, seals the private half, and stores it.
//
// Ed25519 rather than RSA: short enough to template into an authorized_keys
// entry without wrapping, fast, and with no key-size decision for anyone to get
// wrong. Every OpenSSH since 6.5 accepts it.
//
// The private key is generated HERE and never leaves. There is no import path
// and no export endpoint, so the only way this key reaches a host is River using
// it — which is what makes "who could have used this key" a question with an
// answer.
// MintOptions narrows a key to one operation. Zero value mints a fleet key.
type MintOptions struct {
	Purpose     string
	ExpiresAt   *time.Time
	ScopeHostID *uuid.UUID
	ScopeTaskID *uuid.UUID
}

// Mint creates a fleet key — the standing, unscoped credential.
func (r *SSHKeyRepoGorm) Mint(ctx context.Context, name string) (*tenant.SshKey, error) {
	return r.MintWith(ctx, name, MintOptions{Purpose: tenant.SSHKeyPurposeFleet})
}

// MintWith creates a key with an explicit purpose and scope.
//
// The purpose is written into the row rather than left to the column default,
// because Go now branches on it: a struct returned straight from Mint would
// carry the zero value while the stored row carried "fleet", and the two would
// disagree until something re-read it. Atlas also regenerates the schema from
// these models, so a default is not a place to keep a value the code depends on.
//
// It REFUSES to store a key whose shape contradicts its purpose — a join key
// without scope or expiry — rather than leaving that to the caller. Such a row
// is an ephemeral credential that has quietly become permanent, and the private
// half of a join key lives on a hypervisor.
func (r *SSHKeyRepoGorm) MintWith(ctx context.Context, name string, opt MintOptions) (*tenant.SshKey, error) {
	// FIRST, before a keypair is generated or anything is sealed. A contradictory
	// shape is a caller mistake, not a runtime failure, and refusing it here means
	// the check cannot be reached only after work that might fail for its own
	// reasons — nor skipped because an earlier step errored first.
	purpose := opt.Purpose
	if purpose == "" {
		purpose = tenant.SSHKeyPurposeFleet
	}
	if purpose == tenant.SSHKeyPurposeJoin &&
		(opt.ExpiresAt == nil || opt.ScopeHostID == nil || opt.ScopeTaskID == nil) {
		return nil, fmt.Errorf("ssh key: a join key must carry an expiry and a host and task " +
			"scope; its private half is written to a hypervisor, and an unscoped one is a " +
			"fleet credential left on somebody else's disk")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("ssh key: a name is required")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ssh key: generate: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("ssh key: encode public key: %w", err)
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("ssh key: encode private key: %w", err)
	}

	// The id is generated app-side so the AAD can name the row before the insert.
	// Sealing against an id the database has not assigned yet is impossible, and
	// sealing without the row id would let a ciphertext be moved between rows.
	id := uuid.New()
	sealed, err := utils.Seal(
		pemBytes(pemBlock),
		r.keyID, r.key,
		utils.AAD(sshKeysTable, r.orgID, id),
	)
	if err != nil {
		return nil, fmt.Errorf("ssh key: seal: %w", err)
	}

	row := &tenant.SshKey{
		ID:          id,
		Name:        name,
		Purpose:     purpose,
		ExpiresAt:   opt.ExpiresAt,
		ScopeHostID: opt.ScopeHostID,
		ScopeTaskID: opt.ScopeTaskID,
		// authorized_keys form, so it can be pasted onto a host unchanged.
		PublicKey:           strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))),
		EncryptedPrivateKey: sealed.Ciphertext,
		PrivateIV:           sealed.IV,
		PrivateTag:          sealed.Tag,
		PrivateKeyID:        sealed.KeyID,
		// OpenSSH's own SHA256 form, so it compares directly against
		// `ssh-keygen -lf` on the hypervisor.
		Fingerprint: ssh.FingerprintSHA256(sshPub),
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// PrivateKey unseals a key for use. It is the ONLY way to obtain the private
// half, and it exists for River, not for any handler.
//
// A revoked key is refused here rather than filtered in the query, so the caller
// learns why it cannot connect instead of seeing a key that has simply vanished.
func (r *SSHKeyRepoGorm) PrivateKey(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var row tenant.SshKey
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSSHKeyNotFound
		}
		return nil, err
	}
	if !row.Active() {
		return nil, ErrSSHKeyRevoked
	}

	plain, err := utils.Sealed{
		KeyID:      row.PrivateKeyID,
		Ciphertext: row.EncryptedPrivateKey,
		IV:         row.PrivateIV,
		Tag:        row.PrivateTag,
	}.Open(r.key, utils.AAD(sshKeysTable, r.orgID, row.ID))
	if err != nil {
		return nil, fmt.Errorf("ssh key %s: %w", id, err)
	}
	return plain, nil
}

func (r *SSHKeyRepoGorm) List(ctx context.Context) ([]tenant.SshKey, error) {
	var out []tenant.SshKey
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&out).Error
	return out, err
}

// Revoke marks a key unusable. The row stays so that "which key onboarded this
// hypervisor" remains answerable after a rotation.
func (r *SSHKeyRepoGorm) Revoke(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&tenant.SshKey{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Either it does not exist or it was already revoked. Both are the state
		// the caller wanted, so this is not an error — but it is worth not
		// claiming to have revoked something that was never here.
		var n int64
		if err := r.db.WithContext(ctx).Model(&tenant.SshKey{}).Where("id = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrSSHKeyNotFound
		}
	}
	return nil
}

// pemBytes renders the PEM block ssh.MarshalPrivateKey returns. Kept separate so
// the encoding import does not sit in the middle of the sealing logic.
func pemBytes(b *pem.Block) []byte { return pem.EncodeToMemory(b) }
