package utils

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// This file covers the sealing layer that every tenant credential and every
// TOTP seed in APIary passes through. It is deliberately property-driven rather
// than round-trip-driven: a round trip proves only that the code can undo
// itself, which stays true even if the AAD is dropped, the nonce is reused, or
// authentication is skipped. What actually protects a customer's cloud keys is
// the set of decryptions that must FAIL, so that is what is asserted here.

// ucrKeyA and ucrKeyB are two distinct, valid AES-256 keys. They are fixed
// rather than random so a failure is reproducible, and they differ in the first
// byte so "wrong key" is unambiguous even under a partial-comparison bug.
var (
	ucrKeyA = bytes.Repeat([]byte{0xA1}, 32)
	ucrKeyB = bytes.Repeat([]byte{0xB2}, 32)
)

// ucrMasterA is a valid 32-byte master key for HKDF derivation.
var ucrMasterA = bytes.Repeat([]byte{0x5A}, MasterKeyLen)

// ucrMustSeal seals under the given AAD and fails the test if sealing itself
// errors, so the assertions below are about opening, not about setup.
func ucrMustSeal(t *testing.T, plaintext []byte, key, aad []byte) Sealed {
	t.Helper()
	s, err := Seal(plaintext, LegacyKeyID, key, aad)
	if err != nil {
		t.Fatalf("Seal failed during setup: %v", err)
	}
	return s
}

// ucrFlipBit returns a copy of b with one bit of one byte inverted. Used to
// simulate the single-bit corruption that GCM authentication exists to catch —
// storage bit rot, a truncated column, or an attacker nudging a ciphertext.
func ucrFlipBit(b []byte, byteIdx, bitIdx int) []byte {
	out := append([]byte(nil), b...)
	out[byteIdx] ^= 1 << uint(bitIdx)
	return out
}

// Decrypting under a key that did not encrypt must be an ERROR, not a shorter
// or garbled plaintext. This is the difference between a credential read that
// fails loudly and one that hands a caller 32 bytes of noise which then gets
// presented to AWS as a secret access key — or worse, written back to another
// row as though it were the real material.
//
// The nil-plaintext assertion matters as much as the error: a caller that
// forgets to check err must get nothing, never a partial result.
func TestUCRDecryptWithWrongKeyFailsAndYieldsNoPlaintext(t *testing.T) {
	secret := []byte("AKIA-example-secret-access-key")

	ct, iv, tag, err := EncryptAESGCM(secret, ucrKeyA)
	if err != nil {
		t.Fatalf("EncryptAESGCM under key A failed: %v", err)
	}

	plain, err := DecryptAESGCM(ct, ucrKeyB, iv, tag)
	if err == nil {
		t.Fatalf("DecryptAESGCM accepted the WRONG key and returned %q; "+
			"a ciphertext must not open under a key that did not seal it, "+
			"or a stale/rotated key would silently yield material", plain)
	}
	if plain != nil {
		t.Fatalf("DecryptAESGCM returned plaintext %q alongside an error; "+
			"a caller that ignores err would treat this as a real secret", plain)
	}

	// And the correct key must still work, so the test above is not passing
	// merely because decryption is broken for everyone.
	back, err := DecryptAESGCM(ct, ucrKeyA, iv, tag)
	if err != nil {
		t.Fatalf("DecryptAESGCM rejected the CORRECT key: %v; sealing would be unusable", err)
	}
	if !bytes.Equal(back, secret) {
		t.Fatalf("round trip returned %q, want %q", back, secret)
	}
}

// The AAD binding is the reason a stolen ciphertext cannot be replayed onto a
// different row. credential_repo seals with AAD("credentials", orgID, rowID);
// if that binding stopped being enforced, an operator with UPDATE on the
// credentials table could copy tenant A's ciphertext columns onto tenant B's
// row and B would decrypt A's cloud credentials without any key compromise.
//
// Every case below keeps the KEY identical and varies only the AAD, so a pass
// cannot be explained by key separation.
func TestUCRAADBindsCiphertextToExactlyOneRow(t *testing.T) {
	orgA := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	orgB := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	rowA := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	rowB := uuid.MustParse("44444444-4444-4444-8444-444444444444")

	sealAAD := AAD("credentials", orgA, rowA)
	secret := []byte("tenant-A-row-A-secret")
	sealed := ucrMustSeal(t, secret, ucrKeyA, sealAAD)

	cases := []struct {
		name string
		aad  []byte
		why  string
	}{
		{
			name: "same org, different row",
			aad:  AAD("credentials", orgA, rowB),
			why:  "a ciphertext copied onto a sibling row in the same organization would open",
		},
		{
			name: "same row id, different org",
			aad:  AAD("credentials", orgB, rowA),
			why:  "a ciphertext lifted across a tenant boundary would open",
		},
		{
			name: "same ids, different table",
			aad:  AAD("user_totps", orgA, rowA),
			why:  "a credential ciphertext pasted into the TOTP table would open as an MFA seed",
		},
		{
			name: "no AAD at all",
			aad:  nil,
			why:  "a caller that forgot to pass AAD would silently read sealed material unbound",
		},
		{
			name: "empty AAD",
			aad:  []byte{},
			why:  "an empty AAD must not be treated as a wildcard",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := sealed.Open(ucrKeyA, tc.aad)
			if err == nil {
				t.Fatalf("Sealed.Open succeeded under the wrong AAD and returned %q: %s", plain, tc.why)
			}
			if plain != nil {
				t.Fatalf("Sealed.Open returned plaintext %q with an error: %s", plain, tc.why)
			}
		})
	}

	// The correct AAD must still open, otherwise the cases above prove nothing.
	back, err := sealed.Open(ucrKeyA, AAD("credentials", orgA, rowA))
	if err != nil {
		t.Fatalf("Sealed.Open rejected the CORRECT AAD: %v; credentials would be unreadable", err)
	}
	if !bytes.Equal(back, secret) {
		t.Fatalf("Sealed.Open returned %q, want %q", back, secret)
	}
}

// AAD's byte format is a persistence contract, not an implementation detail:
// it is recomputed at read time from columns, so any change to the separator,
// the field order, or the UUID rendering makes every already-stored ciphertext
// permanently unopenable. Pinning the exact bytes turns that from a silent
// production outage into a failing test.
func TestUCRAADFormatIsStableAndUnambiguous(t *testing.T) {
	org := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	row := uuid.MustParse("33333333-3333-4333-8333-333333333333")

	got := string(AAD("credentials", org, row))
	want := "credentials:" + org.String() + ":" + row.String()
	if got != want {
		t.Fatalf("AAD produced %q, want %q; changing this format orphans every "+
			"ciphertext already written to the database", got, want)
	}

	if !bytes.Equal(AAD("credentials", org, row), AAD("credentials", org, row)) {
		t.Fatal("AAD is not deterministic; a value sealed on one call could not be opened on the next")
	}

	// The table name is the only variable-length field and it comes first, so a
	// colon inside it must not let one (table, org, row) triple imitate another.
	if bytes.Equal(AAD("a:b", org, row), AAD("a", org, row)) {
		t.Fatal("AAD collided across different table names; the binding is not injective")
	}
	if bytes.Equal(AAD("credentials", org, row), AAD("credentials", row, org)) {
		t.Fatal("AAD is symmetric in org and row; swapping the two would open the wrong row")
	}
}

// GCM's authentication tag is what makes stored ciphertext tamper-evident. If a
// single flipped bit anywhere — payload, nonce, or tag — could still decrypt,
// then anyone with write access to the database could grind a credential's
// value by mutation without ever holding the key.
//
// The nonce is included deliberately: it is stored in its own column and is the
// component most likely to be treated as "just metadata" by a future refactor.
func TestUCRSingleBitFlipIsRejected(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())
	secret := []byte("a-secret-long-enough-to-span-several-blocks-of-aes")

	ct, iv, tag, err := EncryptAESGCMWithAAD(secret, ucrKeyA, aad)
	if err != nil {
		t.Fatalf("EncryptAESGCMWithAAD failed: %v", err)
	}

	cases := []struct {
		name string
		why  string
		// mutate returns the (ct, iv, tag) triple with one bit flipped.
		mutate func() ([]byte, []byte, []byte)
	}{
		{
			name:   "first ciphertext byte",
			why:    "the leading bytes of a secret could be altered undetected",
			mutate: func() ([]byte, []byte, []byte) { return ucrFlipBit(ct, 0, 0), iv, tag },
		},
		{
			name:   "last ciphertext byte",
			why:    "a truncating or trailing corruption would pass as genuine",
			mutate: func() ([]byte, []byte, []byte) { return ucrFlipBit(ct, len(ct)-1, 7), iv, tag },
		},
		{
			name:   "nonce byte",
			why:    "the IV column would be an unauthenticated input an attacker could steer",
			mutate: func() ([]byte, []byte, []byte) { return ct, ucrFlipBit(iv, 0, 3), tag },
		},
		{
			name:   "last nonce byte",
			why:    "the counter end of the nonce would be malleable",
			mutate: func() ([]byte, []byte, []byte) { return ct, ucrFlipBit(iv, len(iv)-1, 1), tag },
		},
		{
			name:   "tag byte",
			why:    "authentication would not actually be checked",
			mutate: func() ([]byte, []byte, []byte) { return ct, iv, ucrFlipBit(tag, 5, 2) },
		},
		{
			name:   "last tag byte",
			why:    "a partial tag comparison would let the tail of the tag be forged",
			mutate: func() ([]byte, []byte, []byte) { return ct, iv, ucrFlipBit(tag, len(tag)-1, 0) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mct, miv, mtag := tc.mutate()
			plain, err := DecryptAESGCMWithAAD(mct, ucrKeyA, miv, mtag, aad)
			if err == nil {
				t.Fatalf("a single flipped bit in the %s still decrypted, returning %q: %s", tc.name, plain, tc.why)
			}
			if plain != nil {
				t.Fatalf("corrupted input in the %s produced plaintext %q alongside an error: %s", tc.name, plain, tc.why)
			}
		})
	}
}

// A nonce reused under the same AES-GCM key is catastrophic: two ciphertexts
// under one nonce leak the XOR of their plaintexts and, worse, allow the
// authentication key to be recovered, which turns integrity protection off for
// every row sealed under that key. Every credential in a tenant shares one
// derived key, so this is not a theoretical concern here.
//
// Asserting on distinctness of the IVs directly (not merely of the ciphertexts)
// is what catches a refactor that hoists nonce generation out of the function.
func TestUCRNonceIsFreshOnEveryEncryption(t *testing.T) {
	const iterations = 200
	plaintext := []byte("identical plaintext every time")

	seenIV := make(map[string]int, iterations)
	seenCT := make(map[string]int, iterations)

	for i := range iterations {
		ct, iv, _, err := EncryptAESGCM(plaintext, ucrKeyA)
		if err != nil {
			t.Fatalf("EncryptAESGCM failed on iteration %d: %v", i, err)
		}
		if len(iv) != 12 {
			t.Fatalf("nonce length is %d, want the 12-byte GCM standard; a non-standard nonce "+
				"changes the derivation and orphans stored ciphertext", len(iv))
		}
		if prev, dup := seenIV[string(iv)]; dup {
			t.Fatalf("nonce reused between encryption %d and %d under the same key; "+
				"AES-GCM nonce reuse leaks plaintext XOR and the authentication key", prev, i)
		}
		seenIV[string(iv)] = i
		if prev, dup := seenCT[string(ct)]; dup {
			t.Fatalf("identical ciphertext produced on encryption %d and %d for the same "+
				"plaintext; sealing is deterministic and leaks equality of secrets", prev, i)
		}
		seenCT[string(ct)] = i
	}
}

// Two sealings of the same secret must not be byte-identical, because equal
// ciphertext columns would tell anyone with read access to the table which
// tenants share a credential value — without decrypting anything.
func TestUCRSealIsNotDeterministic(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())
	one := ucrMustSeal(t, []byte("shared-secret"), ucrKeyA, aad)
	two := ucrMustSeal(t, []byte("shared-secret"), ucrKeyA, aad)

	if one.Ciphertext == two.Ciphertext && one.IV == two.IV {
		t.Fatal("Seal produced identical ciphertext and IV for the same plaintext, key and AAD; " +
			"equal columns would reveal which rows hold the same secret to anyone with SELECT")
	}
	if one.IV == two.IV {
		t.Fatal("Seal reused the IV across two calls; see nonce-reuse consequences")
	}
}

// A key of the wrong length must be REFUSED, never zero-padded or truncated to
// fit. Silent padding would mean a misconfigured 16-byte key and a correct
// 32-byte key could produce material under a key nobody intended, and the
// failure would only surface as unopenable data much later.
func TestUCRWrongKeyLengthIsRejectedNotPadded(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())

	for _, n := range []int{0, 1, 15, 17, 23, 31, 33, 64} {
		t.Run(ucrItoa(n)+"-byte-key", func(t *testing.T) {
			bad := bytes.Repeat([]byte{0x01}, n)

			_, _, _, err := EncryptAESGCM([]byte("x"), bad)
			if err == nil {
				t.Fatalf("EncryptAESGCM accepted a %d-byte key; a wrong-length key must be "+
					"refused, not silently adjusted to a size AES accepts", n)
			}
			var kse aes.KeySizeError
			if !errors.As(err, &kse) {
				t.Fatalf("EncryptAESGCM(%d-byte key) returned %v, want a wrapped aes.KeySizeError "+
					"so callers can distinguish misconfiguration from corrupt data", n, err)
			}

			_, _, _, err = EncryptAESGCMWithAAD([]byte("x"), bad, aad)
			if !errors.As(err, &kse) {
				t.Fatalf("EncryptAESGCMWithAAD(%d-byte key) returned %v, want a wrapped aes.KeySizeError", n, err)
			}

			_, err = DecryptAESGCM([]byte("x"), bad, make([]byte, 12), make([]byte, 16))
			if !errors.As(err, &kse) {
				t.Fatalf("DecryptAESGCM(%d-byte key) returned %v, want a wrapped aes.KeySizeError", n, err)
			}

			_, err = DecryptAESGCMWithAAD([]byte("x"), bad, make([]byte, 12), make([]byte, 16), aad)
			if !errors.As(err, &kse) {
				t.Fatalf("DecryptAESGCMWithAAD(%d-byte key) returned %v, want a wrapped aes.KeySizeError", n, err)
			}
		})
	}

	// 16 and 24 byte keys are legal AES sizes and must keep working, so the
	// cases above are rejections of WRONG lengths, not of all non-32 lengths.
	for _, n := range []int{16, 24, 32} {
		if _, _, _, err := EncryptAESGCM([]byte("x"), bytes.Repeat([]byte{0x01}, n)); err != nil {
			t.Fatalf("EncryptAESGCM rejected a legal %d-byte AES key: %v", n, err)
		}
	}
}

// ucrItoa is a dependency-free int-to-string for subtest names.
func ucrItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Go's GCM implementation PANICS when handed a nonce of the wrong length. The
// nonce arrives from a database column, so a truncated or migrated-badly IV
// must come back as an error a handler can return as 500 — not as a panic that
// takes down the process. This test is the reason the length check in
// Decrypt*AESGCM* must not be "simplified away".
func TestUCRWrongNonceLengthIsAnErrorNotAPanic(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())
	ct, iv, tag, err := EncryptAESGCMWithAAD([]byte("secret"), ucrKeyA, aad)
	if err != nil {
		t.Fatalf("EncryptAESGCMWithAAD failed: %v", err)
	}

	cases := []struct {
		name string
		iv   []byte
	}{
		{"empty", nil},
		{"truncated", iv[:len(iv)-1]},
		{"one byte", iv[:1]},
		{"overlong", append(append([]byte{}, iv...), 0x00)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("a %s nonce panicked (%v) instead of returning an error; "+
						"a malformed iv column would crash the process", tc.name, r)
				}
			}()

			if plain, err := DecryptAESGCMWithAAD(ct, ucrKeyA, tc.iv, tag, aad); err == nil {
				t.Fatalf("a %s nonce decrypted successfully, returning %q", tc.name, plain)
			}
			if plain, err := DecryptAESGCM(ct, ucrKeyA, tc.iv, tag); err == nil {
				t.Fatalf("a %s nonce decrypted successfully in the no-AAD path, returning %q", tc.name, plain)
			}
		})
	}
}

// Seal refuses uuid.Nil for the key id on purpose: rotation reads key_id to tell
// re-wrapped rows from un-rewrapped ones, and a zero id would make a rotation
// unverifiable — the operator could not prove every row had been re-sealed.
func TestUCRSealRequiresAKeyID(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())

	got, err := Seal([]byte("secret"), uuid.Nil, ucrKeyA, aad)
	if err == nil {
		t.Fatal("Seal accepted uuid.Nil as the key id; rows sealed with a zero key id " +
			"cannot be distinguished during key rotation, making rotation unverifiable")
	}
	if !got.IsZero() {
		t.Fatalf("Seal returned a non-zero Sealed %+v alongside its error; a caller that "+
			"ignores err would persist a half-formed row", got)
	}

	// A bad key must surface as an error from Seal, not as an empty Sealed that
	// a caller happily writes to the database as though the row were sealed.
	badKey, err := Seal([]byte("secret"), LegacyKeyID, bytes.Repeat([]byte{1}, 17), aad)
	if err == nil {
		t.Fatal("Seal accepted a 17-byte key; a misconfigured key must fail the write, " +
			"not produce a row that looks sealed")
	}
	var kse aes.KeySizeError
	if !errors.As(err, &kse) {
		t.Fatalf("Seal with a bad key returned %v, want a wrapped aes.KeySizeError so the "+
			"failure is attributable to configuration rather than to data", err)
	}
	if !badKey.IsZero() {
		t.Fatalf("Seal returned a populated Sealed %+v after a key error", badKey)
	}

	ok, err := Seal([]byte("secret"), LegacyKeyID, ucrKeyA, aad)
	if err != nil {
		t.Fatalf("Seal rejected the legacy key id: %v", err)
	}
	if ok.KeyID != LegacyKeyID {
		t.Fatalf("Seal recorded key id %s, want %s; the id is what a rotation checks against",
			ok.KeyID, LegacyKeyID)
	}
}

// LegacyKeyID is documented as stable across processes and restarts precisely so
// that rows sealed before a key registry exists remain identifiable afterwards.
// Pinning the literal is what stops a "harmless" change to the derivation input
// from stranding every already-written row under an id nothing recognises.
func TestUCRLegacyKeyIDIsPinned(t *testing.T) {
	const want = "10e3292c-22f4-58dc-a35d-b6a3930c7702"
	if got := LegacyKeyID.String(); got != want {
		t.Fatalf("LegacyKeyID is %s, want %s; changing it makes every row already tagged "+
			"with the old id unattributable during rotation", got, want)
	}
	if LegacyKeyID == uuid.Nil {
		t.Fatal("LegacyKeyID is the nil UUID, which Seal refuses outright")
	}
}

// Open must distinguish "there is nothing here" from "there is something here
// and it will not open". A caller that treats every Open error as "no value"
// would swallow tampering; a caller that treats ErrNotSealed as tampering would
// alarm on empty columns. Hence the sentinel, and hence errors.Is rather than a
// string match.
func TestUCROpenSentinelsDistinguishAbsentFromBroken(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())

	t.Run("fully empty is ErrNotSealed", func(t *testing.T) {
		var zero Sealed
		if !zero.IsZero() {
			t.Fatal("a zero-valued Sealed does not report IsZero")
		}
		plain, err := zero.Open(ucrKeyA, aad)
		if !errors.Is(err, ErrNotSealed) {
			t.Fatalf("Open on an unsealed value returned %v, want ErrNotSealed; callers "+
				"branch on this sentinel to tell an empty column from a corrupt one", err)
		}
		if plain != nil {
			t.Fatalf("Open on an unsealed value returned plaintext %q", plain)
		}
	})

	good := ucrMustSeal(t, []byte("secret"), ucrKeyA, aad)

	// A row with SOME columns populated is corruption, not absence, and must not
	// come back as ErrNotSealed — that would let a partially-wiped credential be
	// reported to an operator as "never configured".
	partials := []struct {
		name string
		s    Sealed
	}{
		{"ciphertext missing", Sealed{KeyID: LegacyKeyID, IV: good.IV, Tag: good.Tag}},
		{"iv missing", Sealed{KeyID: LegacyKeyID, Ciphertext: good.Ciphertext, Tag: good.Tag}},
		{"tag missing", Sealed{KeyID: LegacyKeyID, Ciphertext: good.Ciphertext, IV: good.IV}},
	}
	for _, tc := range partials {
		t.Run(tc.name, func(t *testing.T) {
			if tc.s.IsZero() {
				t.Fatalf("a partially populated Sealed (%s) reports IsZero; it would be "+
					"treated as an absent value rather than a damaged one", tc.name)
			}
			plain, err := tc.s.Open(ucrKeyA, aad)
			if err == nil {
				t.Fatalf("Open succeeded on a Sealed with %s, returning %q", tc.name, plain)
			}
			if errors.Is(err, ErrNotSealed) {
				t.Fatalf("Open on a Sealed with %s returned ErrNotSealed; damaged rows would "+
					"be reported to operators as never-configured", tc.name)
			}
			if plain != nil {
				t.Fatalf("Open on a Sealed with %s returned plaintext %q with an error", tc.name, plain)
			}
		})
	}

	t.Run("invalid base64", func(t *testing.T) {
		broken := good
		broken.Ciphertext = "not!valid!base64!"
		plain, err := broken.Open(ucrKeyA, aad)
		if err == nil {
			t.Fatalf("Open accepted non-base64 ciphertext and returned %q", plain)
		}
		if plain != nil {
			t.Fatalf("Open returned plaintext %q for non-base64 ciphertext", plain)
		}
	})
}

// A Sealed whose columns are swapped between rows, or whose IV comes from one
// sealing and tag from another, must not open. This is the concrete database
// tampering the columns invite: they are three independent text columns.
func TestUCRMixingComponentsFromDifferentSealingsFails(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())
	one := ucrMustSeal(t, []byte("secret-one"), ucrKeyA, aad)
	two := ucrMustSeal(t, []byte("secret-two"), ucrKeyA, aad)

	mixes := []struct {
		name string
		s    Sealed
	}{
		{"iv from another sealing", Sealed{KeyID: LegacyKeyID, Ciphertext: one.Ciphertext, IV: two.IV, Tag: one.Tag}},
		{"tag from another sealing", Sealed{KeyID: LegacyKeyID, Ciphertext: one.Ciphertext, IV: one.IV, Tag: two.Tag}},
		{"ciphertext from another sealing", Sealed{KeyID: LegacyKeyID, Ciphertext: two.Ciphertext, IV: one.IV, Tag: one.Tag}},
	}
	for _, tc := range mixes {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := tc.s.Open(ucrKeyA, aad)
			if err == nil {
				t.Fatalf("Open succeeded with the %s, returning %q; the three columns must be "+
					"authenticated as a unit or they can be recombined in the database", tc.name, plain)
			}
			if plain != nil {
				t.Fatalf("Open returned plaintext %q with the %s", plain, tc.name)
			}
		})
	}
}

// DeriveKey exists so that one master key cannot be used to open material sealed
// for a different purpose. If the purpose were ignored, a compromised TOTP-seed
// key would also open every tenant DEK and every tenant DSN — the domain
// separation would be decorative.
func TestUCRDeriveKeySeparatesPurposes(t *testing.T) {
	purposes := []KeyPurpose{KeyPurposeTOTPSeed, KeyPurposeTenantDEK, KeyPurposeTenantDSN}

	derived := make(map[string]KeyPurpose, len(purposes))
	for _, p := range purposes {
		k, err := DeriveKey(ucrMasterA, p)
		if err != nil {
			t.Fatalf("DeriveKey(%s) failed: %v", p, err)
		}
		if len(k) != MasterKeyLen {
			t.Fatalf("DeriveKey(%s) returned %d bytes, want %d; a short key would silently "+
				"be rejected by AES or, worse, change the cipher strength", p, len(k), MasterKeyLen)
		}
		if bytes.Equal(k, ucrMasterA) {
			t.Fatalf("DeriveKey(%s) returned the master key itself; the master would then be "+
				"present in every subsystem that only needs one derived key", p)
		}
		if other, clash := derived[string(k)]; clash {
			t.Fatalf("DeriveKey produced the SAME key for purposes %q and %q; a key compromised "+
				"in one subsystem would open every other subsystem's data", other, p)
		}
		derived[string(k)] = p
	}

	// Determinism: the same master and purpose must derive the same key on every
	// process start, or a restart would orphan everything sealed before it.
	first, err := DeriveKey(ucrMasterA, KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	second, err := DeriveKey(ucrMasterA, KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("DeriveKey failed on second call: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("DeriveKey is not deterministic for the same master and purpose; a process " +
			"restart would be unable to open anything it sealed before")
	}

	// A different master must give a different key even for the same purpose.
	otherMaster := bytes.Repeat([]byte{0x5B}, MasterKeyLen)
	fromOther, err := DeriveKey(otherMaster, KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("DeriveKey with the other master failed: %v", err)
	}
	if bytes.Equal(first, fromOther) {
		t.Fatal("DeriveKey ignored the master key; every deployment would share one key")
	}
}

// The purpose strings are persistence contract: they are the HKDF info, so
// editing one re-derives a different key and every value already sealed under
// the old string becomes permanently unopenable. Pinning them makes that
// deliberate rather than accidental.
//
// The /v2 suffix records a DELIBERATE rename from the "eyrie/" prefix on
// 2026-08-23, which re-derived every key. Note what happened the first time it
// was attempted: a global find/replace rewrote the constants AND this test's
// expectations together, so this test still passed while asserting the wrong
// frozen values. Only the known-answer test below — which pins derived BYTES,
// not strings — caught it. That is why both exist.
func TestUCRKeyPurposeConstantsArePinned(t *testing.T) {
	cases := []struct {
		got  KeyPurpose
		want string
	}{
		{KeyPurposeTOTPSeed, "apiary/mfa/totp-seed/v2"},
		{KeyPurposeTenantDEK, "apiary/tenant/dek/v2"},
		{KeyPurposeTenantDSN, "apiary/tenant/dsn/v2"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Fatalf("key purpose is %q, want %q; changing an HKDF info string re-derives the "+
				"key and orphans every value already sealed under the old one", tc.got, tc.want)
		}
	}
	if MasterKeyLen != 32 {
		t.Fatalf("MasterKeyLen is %d, want 32; AES-256 and the documented "+
			"`openssl rand -base64 32` guidance both depend on it", MasterKeyLen)
	}
}

// Pinning the purpose STRINGS is not enough: the whole HKDF construction is the
// persistence contract. Swapping the hash, moving the purpose from HKDF's info
// parameter to its salt, or adding a fixed info prefix all keep the purposes
// distinct and the derivation deterministic — so every other test in this file
// stays green — while re-deriving a different key for every purpose and leaving
// every value already sealed under the old construction permanently unopenable.
//
// These vectors were produced by the current implementation and regenerated once,
// deliberately, for the v2 purposes on 2026-08-23. If one changes again, that is
// a key-rotation event: existing rows must be decrypted under the old purpose and
// re-sealed under the new one BEFORE the constant moves, never the other way
// round. Updating these hex strings to make a red test green is the one thing
// this test exists to prevent.
func TestUCRDeriveKeyIsPinnedToItsExactConstruction(t *testing.T) {
	cases := []struct {
		purpose KeyPurpose
		wantHex string
	}{
		{KeyPurposeTOTPSeed, "286b0330aca6cd8e755dcab260601cf84211a013882cc825713517ec7dd32e5f"},
		{KeyPurposeTenantDEK, "884d1b22546d0ebbdba748c0de1f31010a515e178cfe4727b02fcbba82ea3912"},
		{KeyPurposeTenantDSN, "9aee04c51e3dc16e0e0a41711d5bdb5cbf98009e6cbbec21e10ee96890eb2d12"},
	}
	for _, tc := range cases {
		t.Run(string(tc.purpose), func(t *testing.T) {
			got, err := DeriveKey(ucrMasterA, tc.purpose)
			if err != nil {
				t.Fatalf("DeriveKey(%s) failed: %v", tc.purpose, err)
			}
			if hex.EncodeToString(got) != tc.wantHex {
				t.Fatalf("DeriveKey(%s) derived %s, want %s; the HKDF construction changed, "+
					"which re-keys the whole deployment and orphans every stored ciphertext "+
					"sealed under the previous derivation", tc.purpose, hex.EncodeToString(got), tc.wantHex)
			}
		})
	}
}

// A master key of the wrong length must be refused rather than stretched to fit.
// HKDF will happily accept any input length, so without this guard a truncated
// or mis-decoded ENCRYPTION_MASTER_KEY (say, 8 bytes of entropy) would derive a
// perfectly valid-looking AES-256 key and the deployment would look healthy
// while being trivially brute-forceable.
func TestUCRDeriveKeyRejectsBadInput(t *testing.T) {
	cases := []struct {
		name    string
		master  []byte
		purpose KeyPurpose
		wantIn  string
	}{
		{"nil master", nil, KeyPurposeTenantDEK, "master key must be exactly"},
		{"short master", bytes.Repeat([]byte{1}, 31), KeyPurposeTenantDEK, "master key must be exactly"},
		{"long master", bytes.Repeat([]byte{1}, 33), KeyPurposeTenantDEK, "master key must be exactly"},
		{"weak 8-byte master", bytes.Repeat([]byte{1}, 8), KeyPurposeTenantDEK, "master key must be exactly"},
		{"empty purpose", ucrMasterA, KeyPurpose(""), "purpose is required"},
		{"whitespace purpose", ucrMasterA, KeyPurpose("   \t\n"), "purpose is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := DeriveKey(tc.master, tc.purpose)
			if err == nil {
				t.Fatalf("DeriveKey accepted %s and returned a %d-byte key; a bad master or an "+
					"unlabelled purpose must fail at startup, not produce a plausible key", tc.name, len(key))
			}
			if key != nil {
				t.Fatalf("DeriveKey returned a key alongside its error for %s; a caller that "+
					"ignores err would encrypt under it", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("DeriveKey(%s) said %q, want a message containing %q so an operator "+
					"can tell which input is wrong", tc.name, err, tc.wantIn)
			}
		})
	}
}

// MasterKeyFromBase64 and DeriveKeyFromBase64 are the startup path: they turn an
// environment variable into the key everything else is derived from. Every
// rejection here is a deployment that refuses to boot rather than one that boots
// with weak or absent encryption.
func TestUCRMasterKeyFromBase64Validates(t *testing.T) {
	valid := EncodeB64(ucrMasterA)

	// ucrMasterA's base64 happens to contain neither '+' nor '/', so it cannot
	// tell the standard alphabet from the URL-safe one. `openssl rand -base64 32`
	// — the command the error message itself recommends — emits a key containing
	// one of those two characters roughly three times in four, so a switch to
	// URLEncoding would refuse most real master keys at boot while every other
	// assertion here still passed.
	t.Run("pins the standard base64 alphabet", func(t *testing.T) {
		raw := make([]byte, MasterKeyLen)
		for i := range raw {
			raw[i] = byte(i * 11)
		}
		std := base64.StdEncoding.EncodeToString(raw)
		if !strings.ContainsRune(std, '+') || !strings.ContainsRune(std, '/') {
			t.Fatalf("test vector %q does not exercise '+' and '/'; it cannot distinguish "+
				"the standard alphabet from the URL-safe one", std)
		}

		got, err := MasterKeyFromBase64(std)
		if err != nil {
			t.Fatalf("MasterKeyFromBase64 rejected a standard-alphabet key containing '+' and '/': "+
				"%v; most keys from `openssl rand -base64 32` look like this and the deployment "+
				"would refuse to boot", err)
		}
		if !bytes.Equal(got, raw) {
			t.Fatalf("MasterKeyFromBase64 decoded to %x, want %x", got, raw)
		}

		// The URL-safe spelling of the same bytes must NOT be accepted, or the two
		// alphabets would silently both work and a key could be written two ways.
		if key, err := MasterKeyFromBase64(base64.URLEncoding.EncodeToString(raw)); err == nil {
			t.Fatalf("MasterKeyFromBase64 accepted the URL-safe encoding and returned %d bytes; "+
				"the stored/configured alphabet must be unambiguous", len(key))
		}
	})

	t.Run("accepts a valid key with surrounding whitespace", func(t *testing.T) {
		got, err := MasterKeyFromBase64("  " + valid + "\n")
		if err != nil {
			t.Fatalf("MasterKeyFromBase64 rejected a valid key with whitespace: %v; env files "+
				"routinely carry trailing newlines", err)
		}
		if !bytes.Equal(got, ucrMasterA) {
			t.Fatalf("MasterKeyFromBase64 decoded to %x, want %x", got, ucrMasterA)
		}
	})

	bad := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   \n"},
		{"not base64", "this is not base64!!"},
		{"decodes to 16 bytes", EncodeB64(bytes.Repeat([]byte{1}, 16))},
		{"decodes to 31 bytes", EncodeB64(bytes.Repeat([]byte{1}, 31))},
		{"decodes to 64 bytes", EncodeB64(bytes.Repeat([]byte{1}, 64))},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			key, err := MasterKeyFromBase64(tc.input)
			if err == nil {
				t.Fatalf("MasterKeyFromBase64 accepted %s and returned %d bytes; the service "+
					"would boot with a key nobody intended", tc.name, len(key))
			}
			if key != nil {
				t.Fatalf("MasterKeyFromBase64 returned a key alongside its error for %s", tc.name)
			}
			if _, err := DeriveKeyFromBase64(tc.input, KeyPurposeTenantDEK); err == nil {
				t.Fatalf("DeriveKeyFromBase64 accepted %s; the validation must not be bypassed "+
					"by the convenience wrapper", tc.name)
			}
		})
	}

	t.Run("wrapper matches the two-step derivation", func(t *testing.T) {
		direct, err := DeriveKey(ucrMasterA, KeyPurposeTOTPSeed)
		if err != nil {
			t.Fatalf("DeriveKey failed: %v", err)
		}
		viaB64, err := DeriveKeyFromBase64(valid, KeyPurposeTOTPSeed)
		if err != nil {
			t.Fatalf("DeriveKeyFromBase64 failed: %v", err)
		}
		if !bytes.Equal(direct, viaB64) {
			t.Fatal("DeriveKeyFromBase64 derived a different key than DeriveKey on the same " +
				"master; seeds written through one path would not open through the other")
		}
	})
}

// End-to-end at the layer the repositories actually use: seal under a derived
// key and the row's AAD, then prove that neither a sibling row's AAD nor a
// different purpose's key opens it. This is the composition that protects a
// credential; testing the pieces separately would not catch a Seal that dropped
// the AAD on the floor.
func TestUCRSealedRowResistsCrossRowAndCrossPurposeOpening(t *testing.T) {
	dek, err := DeriveKey(ucrMasterA, KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("DeriveKey(DEK) failed: %v", err)
	}
	totpKey, err := DeriveKey(ucrMasterA, KeyPurposeTOTPSeed)
	if err != nil {
		t.Fatalf("DeriveKey(TOTP) failed: %v", err)
	}

	org := uuid.New()
	rowA, rowB := uuid.New(), uuid.New()
	secret := []byte("super-secret-cloud-credential")

	sealed := ucrMustSeal(t, secret, dek, AAD("credentials", org, rowA))

	if plain, err := sealed.Open(totpKey, AAD("credentials", org, rowA)); err == nil {
		t.Fatalf("a credential sealed under the tenant DEK opened under the TOTP-seed key, "+
			"returning %q; purpose separation would be decorative", plain)
	}
	if plain, err := sealed.Open(dek, AAD("credentials", org, rowB)); err == nil {
		t.Fatalf("a credential sealed for row %s opened under row %s's AAD, returning %q; "+
			"ciphertext could be replayed between rows by anyone with UPDATE", rowA, rowB, plain)
	}

	back, err := sealed.Open(dek, AAD("credentials", org, rowA))
	if err != nil {
		t.Fatalf("the correct key and AAD failed to open the row: %v", err)
	}
	if !bytes.Equal(back, secret) {
		t.Fatalf("opened %q, want %q", back, secret)
	}
}

// Documents a real defect rather than asserting the behaviour is desirable.
//
// Seal accepts an empty plaintext and reports success, but the Sealed it returns
// can NEVER be opened: an empty ciphertext base64-encodes to "", and Open's
// completeness check rejects an empty Ciphertext field before it ever reaches
// GCM. The value is accepted, persisted, and then permanently unreadable — and
// the error an operator eventually sees ("sealed value is incomplete") points at
// database corruption rather than at the write that caused it.
//
// Nothing in the repository layer currently reaches this: credential_repo
// rejects an empty secret with ErrSecretMissing before sealing. That guard is
// the only thing standing between this and a silently unreadable row, which is
// exactly why it is worth pinning here — if a future caller seals a value that
// happens to be empty, this test says what will happen.
func TestUCRSealOfEmptyPlaintextProducesAnUnopenableValue(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())

	sealed, err := Seal([]byte{}, LegacyKeyID, ucrKeyA, aad)
	if err != nil {
		// If this ever starts erroring, the defect has been fixed at the write
		// side, which is the better fix. Update this test rather than deleting it.
		t.Skipf("Seal now rejects an empty plaintext (%v); the defect this test documents is fixed", err)
	}

	if sealed.Ciphertext != "" {
		t.Fatalf("expected an empty ciphertext column for an empty plaintext, got %q", sealed.Ciphertext)
	}
	if sealed.IsZero() {
		t.Fatal("a Sealed with an IV and tag but no ciphertext reports IsZero, so Open would " +
			"return ErrNotSealed and the row would look like it was never sealed at all")
	}

	plain, err := sealed.Open(ucrKeyA, aad)
	if err == nil {
		t.Fatalf("Seal of an empty plaintext is now openable, returning %q; the round trip is "+
			"repaired and this test should be replaced with a plain round-trip assertion", plain)
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected the completeness check to reject it, got %v", err)
	}
	if errors.Is(err, ErrNotSealed) {
		t.Fatal("an empty-plaintext seal reports ErrNotSealed, which a caller would read as " +
			"'no value configured' rather than 'this write can never be read back'")
	}
}

// The no-AAD helpers must still be genuine AEAD: EncryptAESGCM's output must not
// open under a non-empty AAD, and vice versa. Otherwise the two helper pairs
// would be interchangeable and a refactor could quietly drop the AAD binding
// while every test still passed.
func TestUCRAADAndNoAADHelpersAreNotInterchangeable(t *testing.T) {
	aad := AAD("credentials", uuid.New(), uuid.New())

	ct, iv, tag, err := EncryptAESGCM([]byte("secret"), ucrKeyA)
	if err != nil {
		t.Fatalf("EncryptAESGCM failed: %v", err)
	}
	if plain, err := DecryptAESGCMWithAAD(ct, ucrKeyA, iv, tag, aad); err == nil {
		t.Fatalf("a ciphertext sealed with NO aad opened under a real aad, returning %q; "+
			"the aad would not be authenticated at all", plain)
	}

	ct2, iv2, tag2, err := EncryptAESGCMWithAAD([]byte("secret"), ucrKeyA, aad)
	if err != nil {
		t.Fatalf("EncryptAESGCMWithAAD failed: %v", err)
	}
	if plain, err := DecryptAESGCM(ct2, ucrKeyA, iv2, tag2); err == nil {
		t.Fatalf("a ciphertext sealed WITH an aad opened through the no-aad path, returning %q; "+
			"a caller could strip the row binding by choosing the other helper", plain)
	}
}

// Base64 helpers are the storage encoding for all three sealed columns. A
// mismatch between the encoder and decoder (say, one switching to URL encoding)
// would make previously written rows undecodable, so pin the alphabet by
// round-tripping bytes that exercise the '+' and '/' characters.
func TestUCRBase64HelpersUseStandardAlphabet(t *testing.T) {
	raw := []byte{0xFB, 0xFF, 0xBF, 0x00, 0x10, 0x83}
	enc := EncodeB64(raw)
	if strings.ContainsAny(enc, "-_") {
		t.Fatalf("EncodeB64 produced %q using the URL-safe alphabet; existing columns are "+
			"standard base64 and would stop decoding", enc)
	}
	back, err := DecodeB64(enc)
	if err != nil {
		t.Fatalf("DecodeB64 rejected EncodeB64's own output %q: %v", enc, err)
	}
	if !bytes.Equal(back, raw) {
		t.Fatalf("base64 round trip returned %x, want %x", back, raw)
	}
	if _, err := DecodeB64("!!!not base64!!!"); err == nil {
		t.Fatal("DecodeB64 accepted invalid base64; a corrupt column would decode to something")
	}
}

// RandomBytes underpins every nonce. It must return the requested length and
// must not return a constant.
func TestUCRRandomBytesLengthAndVariability(t *testing.T) {
	for _, n := range []int{0, 1, 12, 16, 32} {
		b, err := RandomBytes(n)
		if err != nil {
			t.Fatalf("RandomBytes(%d) failed: %v", n, err)
		}
		if len(b) != n {
			t.Fatalf("RandomBytes(%d) returned %d bytes; a short nonce would collide", n, len(b))
		}
	}
	seen := make(map[string]bool, 64)
	for range 64 {
		b, err := RandomBytes(16)
		if err != nil {
			t.Fatalf("RandomBytes failed: %v", err)
		}
		if seen[string(b)] {
			t.Fatal("RandomBytes repeated a 16-byte value within 64 draws; it is not random, " +
				"and every nonce and token in the system derives from it")
		}
		seen[string(b)] = true
	}
}
