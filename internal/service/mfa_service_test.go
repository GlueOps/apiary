package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

type mfaFakeTOTPRepo struct {
	rows map[uuid.UUID]*control.UserTOTP
}

func newMFAFakeTOTPRepo() *mfaFakeTOTPRepo {
	return &mfaFakeTOTPRepo{rows: map[uuid.UUID]*control.UserTOTP{}}
}

func (f *mfaFakeTOTPRepo) Create(_ context.Context, t *control.UserTOTP) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	f.rows[t.UserID] = &cp
	return nil
}

func (f *mfaFakeTOTPRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*control.UserTOTP, error) {
	row, ok := f.rows[userID]
	if !ok {
		return nil, nil
	}
	cp := *row
	return &cp, nil
}

func (f *mfaFakeTOTPRepo) MarkVerified(_ context.Context, id uuid.UUID) error {
	for _, row := range f.rows {
		if row.ID == id {
			row.Verified = true
			return nil
		}
	}
	return errors.New("no such totp row")
}

func (f *mfaFakeTOTPRepo) DeleteByUserID(_ context.Context, userID uuid.UUID) error {
	delete(f.rows, userID)
	return nil
}

func (f *mfaFakeTOTPRepo) SetLastUsedStep(_ context.Context, id uuid.UUID, step int64) (bool, error) {
	if step <= 0 {
		return false, errors.New("user_totp: refusing non-positive totp step")
	}
	for _, row := range f.rows {
		if row.ID == id {
			if row.LastUsedStep >= step {
				return false, nil
			}
			row.LastUsedStep = step
			return true, nil
		}
	}
	return false, nil
}

func testMasterKey(t *testing.T) (b64 string, raw []byte) {
	t.Helper()
	raw = make([]byte, utils.MasterKeyLen)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(raw), raw
}

func waitForStableStep(t *testing.T, need time.Duration) {
	t.Helper()
	for {
		elapsed := time.Duration(time.Now().UTC().UnixNano()%int64(totpStepSeconds*time.Second)) * time.Nanosecond
		if totpStepSeconds*time.Second-elapsed >= need {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func currentStep() int64 {
	return time.Now().UTC().Unix() / totpStepSeconds
}

func TestMFAKeyIsDerivedAndDomainSeparated(t *testing.T) {
	b64, raw := testMasterKey(t)

	svc := NewMFAService(newMFAFakeTOTPRepo(), nil, nil, b64, "apiary-test")
	if svc.keyErr != nil {
		t.Fatalf("unexpected key error: %v", svc.keyErr)
	}

	if bytes.Equal(svc.encryptionKey, raw) {
		t.Fatal("TOTP key is the master key verbatim: the HKDF domain separation is dead, " +
			"so one key compromise exposes TOTP seeds, tenant DEKs and tenant DSNs at once")
	}

	want, err := utils.DeriveKey(raw, utils.KeyPurposeTOTPSeed)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if !bytes.Equal(svc.encryptionKey, want) {
		t.Fatal("TOTP key is not HKDF(master, info=apiary/mfa/totp-seed/v2)")
	}

	dek, err := utils.DeriveKey(raw, utils.KeyPurposeTenantDEK)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if bytes.Equal(svc.encryptionKey, dek) {
		t.Fatal("the TOTP-seed key and the tenant-DEK key are identical")
	}
}

func TestMFADeniesEveryOperationWhenTheMasterKeyIsUnusable(t *testing.T) {
	for _, master := range []string{"", "   ", "not-base64!!", base64.StdEncoding.EncodeToString([]byte("too-short"))} {
		rows := newMFAFakeTOTPRepo()
		svc := NewMFAService(rows, nil, nil, master, "apiary-test")
		if svc.keyErr == nil {
			t.Fatalf("master key %q was accepted", master)
		}

		userID := uuid.New()
		if _, err := svc.SetupTOTP(context.Background(), userID, "u@example.test"); err == nil {
			t.Fatalf("SetupTOTP succeeded with master key %q", master)
		}
		if len(rows.rows) != 0 {
			t.Fatalf("SetupTOTP stored a row despite an unusable key (master %q)", master)
		}
	}
}

func TestTOTPSeedCannotBeMovedToAnotherUsersRow(t *testing.T) {
	b64, _ := testMasterKey(t)
	rows := newMFAFakeTOTPRepo()
	svc := NewMFAService(rows, nil, nil, b64, "apiary-test")
	ctx := context.Background()

	attacker := uuid.New()
	victim := uuid.New()

	setup, err := svc.SetupTOTP(ctx, attacker, "attacker@example.test")
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}
	rows.rows[attacker].Verified = true

	waitForStableStep(t, 3*time.Second)
	code, err := totp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if err := svc.ValidateTOTP(ctx, attacker, code); err != nil {
		t.Fatalf("attacker's own factor should validate: %v", err)
	}

	if err := rows.Create(ctx, &control.UserTOTP{
		UserID:   victim,
		Secret:   rows.rows[attacker].Secret,
		Verified: true,
	}); err != nil {
		t.Fatalf("seed copy: %v", err)
	}

	waitForStableStep(t, 3*time.Second)
	code, err = totp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if err := svc.ValidateTOTP(ctx, victim, code); err == nil {
		t.Fatal("a TOTP seed relocated onto another user's row still authenticates: " +
			"the ciphertext is not bound to its row")
	}
}

func TestValidateTOTPRecordsTheMintedStepNotTheWallClock(t *testing.T) {
	b64, _ := testMasterKey(t)
	rows := newMFAFakeTOTPRepo()
	svc := NewMFAService(rows, nil, nil, b64, "apiary-test")
	ctx := context.Background()

	userID := uuid.New()
	setup, err := svc.SetupTOTP(ctx, userID, "u@example.test")
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}
	rows.rows[userID].Verified = true

	waitForStableStep(t, 5*time.Second)
	now := time.Now().UTC()
	base := now.Unix() / totpStepSeconds

	minted := base - 1
	code, err := totp.GenerateCode(setup.Secret, time.Unix(minted*totpStepSeconds, 0).UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if err := svc.ValidateTOTP(ctx, userID, code); err != nil {
		t.Fatalf("a code from the previous step should still validate: %v", err)
	}
	if got := rows.rows[userID].LastUsedStep; got != minted {
		t.Fatalf("recorded step %d, want the minted step %d (recording the wall clock leaves the code spendable again)", got, minted)
	}
}

func TestValidateTOTPRejectsAReplayAcrossTheStepBoundary(t *testing.T) {
	b64, _ := testMasterKey(t)
	rows := newMFAFakeTOTPRepo()
	svc := NewMFAService(rows, nil, nil, b64, "apiary-test")
	ctx := context.Background()

	userID := uuid.New()
	setup, err := svc.SetupTOTP(ctx, userID, "u@example.test")
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}
	rows.rows[userID].Verified = true

	waitForStableStep(t, 5*time.Second)
	base := currentStep()
	minted := base - 1

	rows.rows[userID].LastUsedStep = minted

	code, err := totp.GenerateCode(setup.Secret, time.Unix(minted*totpStepSeconds, 0).UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if err := svc.ValidateTOTP(ctx, userID, code); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("replay of an already-spent code accepted one step later (err=%v); "+
			"the guard is comparing against the clock, not the code", err)
	}
}

func TestMatchTOTPStepResolvesTheWindowAndRejectsGarbage(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "apiary-test", AccountName: "u@example.test"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	secret := key.Secret()

	now := time.Unix(1_700_000_017, 0).UTC()
	base := now.Unix() / totpStepSeconds

	for _, delta := range []int64{-1, 0, 1} {
		want := base + delta
		code, err := totp.GenerateCode(secret, time.Unix(want*totpStepSeconds, 0).UTC())
		if err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
		got, ok := matchTOTPStep(secret, code, now)
		if !ok {
			t.Fatalf("code for step %+d not accepted", delta)
		}
		if got != want {
			t.Fatalf("code for step %+d resolved to %d, want %d", delta, got, want)
		}
	}

	code, err := totp.GenerateCode(secret, time.Unix((base+2)*totpStepSeconds, 0).UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if _, ok := matchTOTPStep(secret, code, now); ok {
		t.Fatal("a code two steps out was accepted; the skew window widened")
	}
	if _, ok := matchTOTPStep(secret, "000000", now); ok {
		if code != "000000" {
			t.Fatal("a fixed guess matched")
		}
	}
}

type mfaFakeUserRepo struct {
	repo.UserRepo
	mfaEnabled map[uuid.UUID]bool
}

func (f *mfaFakeUserRepo) SetMFAEnabled(_ context.Context, userID uuid.UUID, enabled bool) error {
	if f.mfaEnabled == nil {
		f.mfaEnabled = map[uuid.UUID]bool{}
	}
	f.mfaEnabled[userID] = enabled
	return nil
}

func TestActivationCodeIsBurnedAndCannotBeReusedToPassMFA(t *testing.T) {
	b64, _ := testMasterKey(t)
	rows := newMFAFakeTOTPRepo()
	users := &mfaFakeUserRepo{}
	svc := NewMFAService(rows, nil, users, b64, "apiary-test")
	ctx := context.Background()

	userID := uuid.New()
	setup, err := svc.SetupTOTP(ctx, userID, "u@example.test")
	if err != nil {
		t.Fatalf("SetupTOTP: %v", err)
	}

	waitForStableStep(t, 5*time.Second)
	code, err := totp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if err := svc.VerifyAndActivateTOTP(ctx, userID, code); err != nil {
		t.Fatalf("VerifyAndActivateTOTP: %v", err)
	}
	if !rows.rows[userID].Verified || !users.mfaEnabled[userID] {
		t.Fatal("activation did not take effect")
	}
	if rows.rows[userID].LastUsedStep == 0 {
		t.Fatal("the activation code was not burned; it is still spendable at /auth/mfa/verify")
	}

	if err := svc.ValidateTOTP(ctx, userID, code); !errors.Is(err, ErrInvalidTOTPCode) {
		t.Fatalf("the activation code still authenticates (err=%v)", err)
	}
}
