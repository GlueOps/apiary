package utils

import (
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const MasterKeyLen = 32

type KeyPurpose string

const (
	// These are the `info` parameter of the HKDF that derives every key in the
	// system, so the STRING IS PART OF THE KEY. They are not labels.
	//
	// The /v2 suffix records that they were renamed from an "eyrie/" prefix on
	// 2026-08-23, which re-derived every key. Anything sealed under the v1
	// derivation is unreadable to this build:
	//
	//   TenantDEK  sealed credentials in tenant databases
	//   TenantDSN  tenant connection strings — the databases become unreachable
	//   TOTPSeed   enrolled authenticators
	//
	// Nothing errors when this changes; the data simply stops opening. The
	// version bump exists so the break is legible rather than mysterious.
	//
	// CHANGING ONE AGAIN IS A KEY-ROTATION EVENT, not an edit: every affected row
	// must be decrypted under the old purpose and re-sealed under the new one,
	// with dual-read during rollout, BEFORE the constant moves. A global
	// find/replace reaches these — that is exactly how the 2026-08-23 change
	// happened, and it was caught only by
	// TestUCRDeriveKeyIsPinnedToItsExactConstruction, which pins derived bytes
	// rather than strings.
	KeyPurposeTOTPSeed  KeyPurpose = "apiary/mfa/totp-seed/v2"
	KeyPurposeTenantDEK KeyPurpose = "apiary/tenant/dek/v2"
	KeyPurposeTenantDSN KeyPurpose = "apiary/tenant/dsn/v2"

	// KeyPurposeQueuePayload seals the body of a queued email.
	//
	// The prefix differs from the three above, and that is deliberate rather than
	// an oversight. These strings are HKDF info parameters: changing one changes
	// every key derived under it, so the three above are frozen — renaming them
	// would make every sealed credential, every tenant DSN and every TOTP seed
	// permanently unreadable. Nothing has ever been sealed under this new purpose,
	// so it is free to carry the correct name, and new purposes should.
	KeyPurposeQueuePayload KeyPurpose = "apiary/queue/payload/v1"
)

func DeriveKey(master []byte, purpose KeyPurpose) ([]byte, error) {
	if len(master) != MasterKeyLen {
		return nil, fmt.Errorf("derive key: master key must be exactly %d bytes, got %d", MasterKeyLen, len(master))
	}
	if strings.TrimSpace(string(purpose)) == "" {
		return nil, errors.New("derive key: purpose is required")
	}
	key, err := hkdf.Key(sha256.New, master, nil, string(purpose), MasterKeyLen)
	if err != nil {
		return nil, fmt.Errorf("derive key: hkdf: %w", err)
	}
	return key, nil
}

func MasterKeyFromBase64(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, fmt.Errorf(
			"ENCRYPTION_MASTER_KEY must be set (base64 of %d bytes; generate with `openssl rand -base64 %d`)",
			MasterKeyLen, MasterKeyLen,
		)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_MASTER_KEY must be valid base64: %w", err)
	}
	if len(raw) != MasterKeyLen {
		return nil, fmt.Errorf(
			"ENCRYPTION_MASTER_KEY must decode to exactly %d bytes, got %d",
			MasterKeyLen, len(raw),
		)
	}
	return raw, nil
}

func DeriveKeyFromBase64(encoded string, purpose KeyPurpose) ([]byte, error) {
	master, err := MasterKeyFromBase64(encoded)
	if err != nil {
		return nil, err
	}
	return DeriveKey(master, purpose)
}

// LegacyKeyID identifies the single ENCRYPTION_MASTER_KEY, for deployments that
// have not configured a key registry.
//
// .env describes the intended end state: ENCRYPTION_KEYS holds several
// key-encryption keys as id:base64 pairs, "and leaving both blank keeps using
// the single master key above, which is read as the key with id legacy". Until
// that registry exists, sealing still needs a stable identifier — utils.Seal
// refuses uuid.Nil precisely so that a rotation can tell re-wrapped rows from
// un-rewrapped ones, and a zero id would make rotation unverifiable.
//
// Derived rather than generated so it is identical in every process and across
// restarts; a random id per boot would tag rows with an identifier that means
// nothing afterwards.
var LegacyKeyID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("apiary/encryption/legacy"))
