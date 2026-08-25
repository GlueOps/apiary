package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"gorm.io/datatypes"
)

var (
	ErrTOTPAlreadySetup = errors.New("TOTP already configured")
	ErrTOTPNotSetup     = errors.New("TOTP not configured")
	ErrTOTPNotVerified  = errors.New("TOTP not verified")
	ErrInvalidTOTPCode  = errors.New("invalid TOTP code")
)

type TOTPSetupResult struct {
	Secret  string
	URI     string
	QRImage string
}

type MFAService struct {
	totps         repo.UserTOTPRepo
	passkeys      repo.UserPasskeyRepo
	users         repo.UserRepo
	encryptionKey []byte
	keyErr        error
	issuerName    string
	audit         repo.AuthAuditRepo
}

func (s *MFAService) SetAudit(r repo.AuthAuditRepo) { s.audit = r }

func NewMFAService(
	totps repo.UserTOTPRepo,
	passkeys repo.UserPasskeyRepo,
	users repo.UserRepo,
	encryptionKey string,
	issuerName string,
) *MFAService {

	key, err := utils.DeriveKeyFromBase64(encryptionKey, utils.KeyPurposeTOTPSeed)
	return &MFAService{
		totps:         totps,
		passkeys:      passkeys,
		users:         users,
		encryptionKey: key,
		keyErr:        err,
		issuerName:    issuerName,
	}
}

func (s *MFAService) SetupTOTP(ctx context.Context, userID uuid.UUID, email string) (*TOTPSetupResult, error) {
	existing, err := s.totps.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Verified {
		return nil, ErrTOTPAlreadySetup
	}

	if existing != nil {
		if err := s.totps.DeleteByUserID(ctx, userID); err != nil {
			return nil, err
		}
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuerName,
		AccountName: email,
	})
	if err != nil {
		return nil, err
	}

	encrypted, err := s.encrypt(key.Secret(), totpAAD(userID))
	if err != nil {
		return nil, err
	}

	record := &control.UserTOTP{
		UserID: userID,
		Secret: encrypted,
	}
	if err := s.totps.Create(ctx, record); err != nil {
		return nil, err
	}

	return &TOTPSetupResult{
		Secret: key.Secret(),
		URI:    key.URL(),
	}, nil
}

func (s *MFAService) VerifyAndActivateTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	record, err := s.totps.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if record == nil {
		return ErrTOTPNotSetup
	}
	if record.Verified {
		return ErrTOTPAlreadySetup
	}

	secret, err := s.decrypt(record.Secret, totpAAD(userID))
	if err != nil {
		return err
	}

	step, ok := matchTOTPStep(secret, code, time.Now().UTC())
	if !ok {
		return ErrInvalidTOTPCode
	}
	advanced, err := s.totps.SetLastUsedStep(ctx, record.ID, step)
	if err != nil {
		return err
	}
	if !advanced {
		return ErrInvalidTOTPCode
	}

	if err := s.totps.MarkVerified(ctx, record.ID); err != nil {
		return err
	}

	return s.users.SetMFAEnabled(ctx, userID, true)
}

func (s *MFAService) ValidateTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	record, err := s.totps.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if record == nil || !record.Verified {
		return ErrTOTPNotSetup
	}

	secret, err := s.decrypt(record.Secret, totpAAD(userID))
	if err != nil {
		return err
	}

	step, ok := matchTOTPStep(secret, code, time.Now().UTC())
	if !ok {
		return ErrInvalidTOTPCode
	}
	advanced, err := s.totps.SetLastUsedStep(ctx, record.ID, step)
	if err != nil {
		return err
	}
	if !advanced {
		return ErrInvalidTOTPCode
	}

	return nil
}

func (s *MFAService) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	if err := s.totps.DeleteByUserID(ctx, userID); err != nil {
		return err
	}

	count, err := s.passkeys.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if s.audit != nil {
		if err := s.audit.Create(ctx, &control.AuthAuditEvent{
			SubjectID:      userID,
			SubjectKind:    "user",
			OrganizationID: uuid.Nil,
			EventType:      "mfa_disabled",
			Metadata:       datatypes.NewJSONType(map[string]any{}),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to write auth audit event", "event_type", "mfa_disabled", "error", err)
		}
	}

	return s.users.SetMFAEnabled(ctx, userID, false)
}

func (s *MFAService) UserHasMFA(ctx context.Context, userID uuid.UUID) (bool, error) {
	t, err := s.totps.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if t != nil && t.Verified {
		return true, nil
	}

	count, err := s.passkeys.CountByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *MFAService) encrypt(plaintext string, aad []byte) (string, error) {

	if s.keyErr != nil {
		return "", s.keyErr
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), aad)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *MFAService) decrypt(encoded string, aad []byte) (string, error) {
	if s.keyErr != nil {
		return "", s.keyErr
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

const totpStepSeconds = 30

const totpAADTable = "user_totps"

func totpAAD(userID uuid.UUID) []byte {
	return utils.AAD(totpAADTable, uuid.Nil, userID)
}

const totpSkewSteps = 1

var totpValidateOpts = totp.ValidateOpts{
	Period:    totpStepSeconds,
	Skew:      0,
	Digits:    otp.DigitsSix,
	Algorithm: otp.AlgorithmSHA1,
}

func matchTOTPStep(secret, code string, now time.Time) (int64, bool) {
	base := now.UTC().Unix() / totpStepSeconds
	for delta := int64(totpSkewSteps); delta >= -totpSkewSteps; delta-- {
		step := base + delta
		at := time.Unix(step*totpStepSeconds, 0).UTC()
		ok, err := totp.ValidateCustom(code, secret, at, totpValidateOpts)
		if err != nil || !ok {
			continue
		}
		return step, true
	}
	return 0, false
}
