package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrPasskeyNotFound     = errors.New("passkey not found")
	ErrPasskeyChallenge    = errors.New("invalid passkey challenge")
	ErrPasskeyRegistration = errors.New("passkey registration failed")
	ErrPasskeyLogin        = errors.New("passkey login failed")
)

type PasskeyService struct {
	webauthn    *webauthn.WebAuthn
	passkeys    repo.UserPasskeyRepo
	users       repo.UserRepo
	memberships repo.MembershipRepo
	tokens      *TokenService
	sessions    *TokenSessionService
	jwtSecret   []byte
	challenges  repo.AuthChallengeRepo
	audit       repo.AuthAuditRepo
}

func (s *PasskeyService) SetAudit(r repo.AuthAuditRepo) { s.audit = r }

func (s *PasskeyService) SetChallenges(r repo.AuthChallengeRepo) { s.challenges = r }

func NewPasskeyService(
	passkeys repo.UserPasskeyRepo,
	users repo.UserRepo,
	memberships repo.MembershipRepo,
	tokens *TokenService,
	sessions *TokenSessionService,
	jwtSecret string,
	rpID, rpOrigin, rpName string,
) (*PasskeyService, error) {
	if rpID == "" || rpOrigin == "" {
		return &PasskeyService{
			passkeys:    passkeys,
			users:       users,
			memberships: memberships,
			tokens:      tokens,
			sessions:    sessions,
			jwtSecret:   []byte(jwtSecret),
		}, nil
	}

	if rpName == "" {
		rpName = "APIary"
	}

	wconfig := passkeyWebAuthnConfig(rpName, rpID, rpOrigin)

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}

	return &PasskeyService{
		webauthn:    w,
		passkeys:    passkeys,
		users:       users,
		memberships: memberships,
		tokens:      tokens,
		sessions:    sessions,
		jwtSecret:   []byte(jwtSecret),
	}, nil
}

type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webauthnUser) WebAuthnName() string                       { return u.name }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func toWebAuthnUser(user *control.User, passkeys []control.UserPasskey) *webauthnUser {
	creds := make([]webauthn.Credential, 0, len(passkeys))
	for _, pk := range passkeys {
		credID, err := base64.RawURLEncoding.DecodeString(pk.CredentialID)
		if err != nil {
			slog.Warn("skipping passkey with invalid credential ID", "credential_id", pk.CredentialID, "error", err)
			continue
		}
		transports := make([]protocol.AuthenticatorTransport, 0)
		for _, t := range pk.Transports.Data() {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              credID,
			PublicKey:       pk.PublicKey,
			AttestationType: pk.AttestationType,
			Authenticator: webauthn.Authenticator{
				SignCount: pk.SignCount,
			},
			Transport: transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: pk.BackupEligible,
				BackupState:    pk.BackupState,
			},
		})
	}
	return &webauthnUser{
		id:          user.ID[:],
		name:        user.Email,
		displayName: user.FullName(),
		credentials: creds,
	}
}

type challengeClaims struct {
	SessionData string `json:"sd"`
	jwt.RegisteredClaims
}

const passkeyChallengeTTL = 5 * time.Minute

func (s *PasskeyService) signChallenge(ctx context.Context, sessionJSON []byte, userID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	jti := uuid.New()

	if s.challenges != nil {
		if err := s.challenges.Create(ctx, &control.AuthChallenge{
			JTI:         jti,
			UserID:      userID,
			Kind:        "passkey_ceremony",
			MaxAttempts: 1,
			ExpiresAt:   now.Add(passkeyChallengeTTL),
		}); err != nil {
			return "", err
		}
	}

	claims := challengeClaims{
		SessionData: base64.RawURLEncoding.EncodeToString(sessionJSON),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(passkeyChallengeTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *PasskeyService) parseChallenge(ctx context.Context, tokenStr string) ([]byte, error) {
	claims := &challengeClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrPasskeyChallenge
	}

	if s.challenges != nil {
		jti, perr := uuid.Parse(claims.ID)
		if perr != nil {
			return nil, ErrPasskeyChallenge
		}
		consumed, cerr := s.challenges.Consume(ctx, jti, time.Now().UTC())
		if cerr != nil {
			return nil, cerr
		}
		if !consumed {

			return nil, ErrPasskeyChallenge
		}
	}

	return base64.RawURLEncoding.DecodeString(claims.SessionData)
}

type BeginRegistrationResult struct {
	Options   *protocol.CredentialCreation
	Challenge string
}

func (s *PasskeyService) BeginRegistration(ctx context.Context, userID uuid.UUID) (*BeginRegistrationResult, error) {
	if s.webauthn == nil {
		return nil, errors.New("webauthn not configured")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	existingKeys, err := s.passkeys.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	wUser := toWebAuthnUser(user, existingKeys)
	options, session, err := s.webauthn.BeginRegistration(wUser)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	challenge, err := s.signChallenge(ctx, sessionJSON, userID)
	if err != nil {
		return nil, err
	}

	return &BeginRegistrationResult{
		Options:   options,
		Challenge: challenge,
	}, nil
}

type FinishRegistrationInput struct {
	Challenge string
	Response  *protocol.ParsedCredentialCreationData
	Name      string
}

func (s *PasskeyService) FinishRegistration(ctx context.Context, userID uuid.UUID, in FinishRegistrationInput) (*control.UserPasskey, error) {
	if s.webauthn == nil {
		return nil, errors.New("webauthn not configured")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	existingKeys, err := s.passkeys.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessionJSON, err := s.parseChallenge(ctx, in.Challenge)
	if err != nil {
		return nil, ErrPasskeyChallenge
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionJSON, &session); err != nil {
		return nil, ErrPasskeyChallenge
	}

	wUser := toWebAuthnUser(user, existingKeys)
	cred, err := s.webauthn.CreateCredential(wUser, session, in.Response)
	if err != nil {
		return nil, fmt.Errorf("finish registration: %w", err)
	}

	credID := base64.RawURLEncoding.EncodeToString(cred.ID)
	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}

	name := in.Name
	if name == "" {
		name = "Passkey"
	}

	passkey := &control.UserPasskey{
		UserID:          userID,
		CredentialID:    credID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		AAGUID:          uuid.UUID(cred.Authenticator.AAGUID).String(),
		SignCount:       cred.Authenticator.SignCount,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
		Name:            name,
		Transports:      datatypes.NewJSONType(transports),
	}
	if err := s.passkeys.Create(ctx, passkey); err != nil {
		return nil, err
	}

	if err := s.users.SetMFAEnabled(ctx, userID, true); err != nil {
		return nil, err
	}

	return passkey, nil
}

type BeginLoginResult struct {
	Options   *protocol.CredentialAssertion
	Challenge string
}

// passkeyWebAuthnConfig is extracted so a test can assert against the
// configuration this service actually builds, rather than a copy of it.
//
// UserVerification is load-bearing beyond registration: go-webauthn feeds it
// into the LOGIN ceremony too, so requiring it is what makes
// cred.Flags.UserVerified reliably true when an assertion succeeds — and that
// flag is what lets a passkey count as both factors. See
// TestPasskeyConfigRequiresUserVerification for what relaxing it costs.
func passkeyWebAuthnConfig(rpName, rpID, rpOrigin string) *webauthn.Config {
	return &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},

		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
		},
	}
}

func (s *PasskeyService) BeginLogin(ctx context.Context) (*BeginLoginResult, error) {
	if s.webauthn == nil {
		return nil, errors.New("webauthn not configured")
	}

	options, session, err := s.webauthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("begin login: %w", err)
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	challenge, err := s.signChallenge(ctx, sessionJSON, uuid.Nil)
	if err != nil {
		return nil, err
	}

	return &BeginLoginResult{
		Options:   options,
		Challenge: challenge,
	}, nil
}

type FinishLoginInput struct {
	Challenge string
	Response  *protocol.ParsedCredentialAssertionData
	IP        string
	UA        string
}

func (s *PasskeyService) FinishLogin(ctx context.Context, in FinishLoginInput) (*IssuedTokens, string, error) {
	if s.webauthn == nil {
		return nil, "", errors.New("webauthn not configured")
	}

	sessionJSON, err := s.parseChallenge(ctx, in.Challenge)
	if err != nil {
		return nil, "", ErrPasskeyChallenge
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionJSON, &session); err != nil {
		return nil, "", ErrPasskeyChallenge
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		userID, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, err
		}
		user, err := s.users.GetByID(ctx, userID)
		if err != nil || user == nil {
			return nil, ErrUserNotFound
		}
		keys, err := s.passkeys.ListByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		return toWebAuthnUser(user, keys), nil
	}

	cred, err := s.webauthn.ValidateDiscoverableLogin(handler, session, in.Response)
	if err != nil {
		return nil, "", fmt.Errorf("validate login: %w", err)
	}

	credID := base64.RawURLEncoding.EncodeToString(cred.ID)
	passkey, err := s.passkeys.GetByCredentialID(ctx, credID)
	if err != nil || passkey == nil {
		return nil, "", ErrPasskeyLogin
	}

	if cred.Authenticator.CloneWarning {
		slog.WarnContext(ctx, "webauthn clone warning: authenticator sign count did not advance",
			"user_id", passkey.UserID.String(),
			"credential_id", credID,
			"sign_count", cred.Authenticator.SignCount,
		)
		return nil, "", ErrPasskeyLogin
	}

	now := time.Now().UTC()
	if err := s.passkeys.UpdateSignCount(ctx, passkey.ID, cred.Authenticator.SignCount, now); err != nil {
		return nil, "", err
	}

	user, err := s.users.GetByID(ctx, passkey.UserID)
	if err != nil || user == nil {
		return nil, "", ErrPasskeyLogin
	}
	if !user.IsActive {
		return nil, "", ErrUserInactive
	}

	memberships, err := s.getMemberships(ctx, user.ID)
	if err != nil || len(memberships) == 0 {
		return nil, "", ErrNoOrganization
	}

	orgID := memberships[0].OrganizationID.String()
	issued, err := s.sessions.IssueTracked(ctx, IssueTrackedInput{
		SubjectID:   user.ID.String(),
		SubjectKind: "user",
		OrgID:       orgID,
		Opts: IssueOpts{
			AMR: []string{"passkey"},

			MFA:        cred.Flags.UserVerified,
			MFAEnabled: true,
		},
		IPAddress: in.IP,
		UserAgent: in.UA,
		EventType: "login",
	})
	if err != nil {
		return nil, "", err
	}

	return issued, orgID, nil
}

func (s *PasskeyService) getMemberships(ctx context.Context, userID uuid.UUID) ([]control.Membership, error) {
	return s.memberships.ListByUserID(ctx, userID)
}

func (s *PasskeyService) List(ctx context.Context, userID uuid.UUID) ([]control.UserPasskey, error) {
	return s.passkeys.ListByUserID(ctx, userID)
}

func (s *PasskeyService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.passkeys.Delete(ctx, id, userID); err != nil {
		// The repository speaks gorm; this package's callers must not have to.
		// Without this translation ErrPasskeyNotFound was returned by nothing at
		// all, so the handler's 404 branch was unreachable and deleting a passkey
		// that does not exist -- or belongs to somebody else -- answered 500. An
		// id the caller supplied that does not resolve is a client error, not a
		// server fault, and reporting it as one puts every stale UI click on the
		// 5xx dashboard.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPasskeyNotFound
		}
		return err
	}

	if s.audit != nil {
		if err := s.audit.Create(ctx, &control.AuthAuditEvent{
			SubjectID:      userID,
			SubjectKind:    "user",
			OrganizationID: uuid.Nil,
			EventType:      "passkey_removed",
			Metadata:       datatypes.NewJSONType(map[string]any{"passkey_id": id.String()}),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to write auth audit event", "event_type", "passkey_removed", "error", err)
		}
	}

	return nil
}
