package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	// RefreshExpiresIn is the refresh token's lifetime in seconds.
	//
	// Sent because a client that knows only ExpiresIn cannot tell a session that
	// needs refreshing from one that is over. The UI was discarding a stored
	// session the moment the ACCESS token expired — fifteen minutes — while the
	// refresh token it was holding stayed valid for another seven days, so
	// closing a tab over lunch meant signing in again for no reason.
	RefreshExpiresIn int
}

type IssuedTokens struct {
	TokenPair
	AccessJTI        string
	RefreshJTI       string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type AuthClaims struct {
	OrgID       string   `json:"org_id,omitempty"`
	Type        string   `json:"typ"`
	AMR         []string `json:"amr,omitempty"`
	MFA         bool     `json:"mfa,omitempty"`
	MFAEnabled  bool     `json:"mfa_enabled,omitempty"`
	ActorID     string   `json:"act_id,omitempty"`
	Impersonate bool     `json:"imp,omitempty"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(secret, issuer string, accessTTL, refreshTTL time.Duration) (*TokenService, error) {
	if secret == "" {
		return nil, fmt.Errorf("jwt secret is required")
	}
	if issuer == "" {
		issuer = "nexus"
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}

	return &TokenService{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

type IssueOpts struct {
	AMR         []string
	MFA         bool
	MFAEnabled  bool
	ActorID     string
	Impersonate bool
}

func (s *TokenService) Issue(userID, orgID string, now time.Time, opts ...IssueOpts) (*IssuedTokens, error) {
	var opt IssueOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()
	accessExp := now.Add(s.accessTTL)
	refreshExp := now.Add(s.refreshTTL)

	accessClaims := AuthClaims{
		OrgID:       orgID,
		Type:        "access",
		AMR:         opt.AMR,
		MFA:         opt.MFA,
		MFAEnabled:  opt.MFAEnabled,
		ActorID:     opt.ActorID,
		Impersonate: opt.Impersonate,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessJTI,
			Subject:   userID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}

	refreshClaims := AuthClaims{
		OrgID:       orgID,
		Type:        "refresh",
		AMR:         opt.AMR,
		MFA:         opt.MFA,
		MFAEnabled:  opt.MFAEnabled,
		ActorID:     opt.ActorID,
		Impersonate: opt.Impersonate,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshJTI,
			Subject:   userID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := access.SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refresh.SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	return &IssuedTokens{
		TokenPair: TokenPair{
			AccessToken:      accessToken,
			RefreshToken:     refreshToken,
			TokenType:        "Bearer",
			ExpiresIn:        int(s.accessTTL.Seconds()),
			RefreshExpiresIn: int(s.refreshTTL.Seconds()),
		},
		AccessJTI:        accessJTI,
		RefreshJTI:       refreshJTI,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *TokenService) IssueMFAPending(userID, orgID string, now time.Time) (string, error) {
	tok, _, err := s.IssueMFAPendingWithJTI(userID, orgID, now)
	return tok, err
}

func (s *TokenService) IssueMFAPendingWithJTI(userID, orgID string, now time.Time) (string, uuid.UUID, error) {
	jti := uuid.New()
	claims := AuthClaims{
		OrgID: orgID,
		Type:  "mfa_pending",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			Subject:   userID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", uuid.Nil, err
	}
	return signed, jti, nil
}

func (s *TokenService) Parse(tokenStr, expectedType string) (*AuthClaims, error) {
	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Type != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
