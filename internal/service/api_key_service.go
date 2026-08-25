package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/google/uuid"
)

var (
	// ErrAPIKeyRejected covers unknown, revoked, expired and wrong-secret. One
	// error for all four: this is reached by anything that can present a
	// credential, so distinguishing them would turn the endpoint into an oracle
	// over keys the caller does not hold.
	ErrAPIKeyRejected = errors.New("api key: rejected")

	// ErrAPIKeyWrongSurface is separate because it is NOT an authentication
	// failure — the credential is genuine, it is simply not admitted here. An
	// agent key on an org endpoint is a misconfigured deployment, and telling
	// its operator "rejected" would send them looking for a credential problem
	// that does not exist.
	ErrAPIKeyWrongSurface = errors.New("api key: not valid on this surface")
)

// apiKeyPrefixLen is how much of the identifying half is stored in clear, so a
// key can be recognised in a list without being usable from it.
const apiKeyPrefixLen = 8

type APIKeyService struct {
	keys repo.APIKeyRepo
}

func NewAPIKeyService(keys repo.APIKeyRepo) *APIKeyService {
	return &APIKeyService{keys: keys}
}

func hashAPISecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func randomAPIKeyHalf() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("api key: generate: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type MintInput struct {
	Scope   string
	Name    string
	Purpose string
	OrgID   *uuid.UUID
	UserID  *uuid.UUID
	TTL     time.Duration
}

// MintResult carries the plaintext halves. They exist here and nowhere else:
// only hashes are stored, so a key that is lost is replaced, never recovered.
type MintResult struct {
	Key    *control.APIKey
	Public string
	Secret string
}

func (s *APIKeyService) Mint(ctx context.Context, in MintInput) (*MintResult, error) {
	if !control.APIKeyScopeIsValid(in.Scope) {
		return nil, fmt.Errorf("api key: %q is not a scope", in.Scope)
	}
	// The shape has to match the scope, or a credential ends up admitted
	// somewhere its owner never intended.
	switch in.Scope {
	case control.APIKeyScopeUser:
		if in.UserID == nil {
			return nil, errors.New("api key: a user key must name its user")
		}
		if in.OrgID != nil {
			return nil, errors.New("api key: a user key must not be bound to an organization; " +
				"the organization comes from X-Org-Id and is checked against membership at each call")
		}
	case control.APIKeyScopeOrg, control.APIKeyScopeAgent:
		if in.OrgID == nil {
			return nil, errors.New("api key: an org or agent key must name its organization")
		}
	}

	public, err := randomAPIKeyHalf()
	if err != nil {
		return nil, err
	}
	key := &control.APIKey{
		OrgID: in.OrgID, UserID: in.UserID,
		Scope: in.Scope, Name: in.Name, Purpose: in.Purpose,
		KeyHash: hashAPISecret(public),
	}
	if p := public[:apiKeyPrefixLen]; true {
		key.Prefix = &p
	}
	if in.TTL > 0 {
		exp := time.Now().UTC().Add(in.TTL)
		key.ExpiresAt = &exp
	}

	out := &MintResult{Key: key, Public: public}
	if key.RequiresSecret() {
		secret, err := randomAPIKeyHalf()
		if err != nil {
			return nil, err
		}
		h := hashAPISecret(secret)
		key.SecretHash = &h
		out.Secret = secret
	}

	if err := s.keys.Create(ctx, key); err != nil {
		return nil, err
	}
	return out, nil
}

// Authenticated is what a verified credential establishes.
type Authenticated struct {
	Key *control.APIKey
}

// Authenticate verifies a presented credential and confirms it is admitted on
// the surface being called.
//
// wantScope is the surface, not a preference: the agent endpoints pass
// APIKeyScopeAgent and everything else passes the org/user pair, and the
// exclusion runs both ways. An org key must not reach the agent surface (a
// human or an automation forging an agent) and an agent key must not reach the
// org surface (a compromised hypervisor turning its credential on the rest of
// the tenant's API).
func (s *APIKeyService) Authenticate(ctx context.Context, public, secret string, wantScopes ...string) (*Authenticated, error) {
	key, err := s.keys.ByKeyHash(ctx, hashAPISecret(public))
	if err != nil || key == nil {
		return nil, ErrAPIKeyRejected
	}
	if !key.Usable(time.Now().UTC()) {
		return nil, ErrAPIKeyRejected
	}

	if key.RequiresSecret() {
		if key.SecretHash == nil || secret == "" {
			return nil, ErrAPIKeyRejected
		}
		// Constant time: a byte-by-byte comparison leaks how much of a guessed
		// secret was right, and this is reachable by anything that can send a
		// request.
		if subtle.ConstantTimeCompare([]byte(hashAPISecret(secret)), []byte(*key.SecretHash)) != 1 {
			return nil, ErrAPIKeyRejected
		}
	} else if secret != "" {
		// A user key carries no secret half. Accepting one anyway would mean a
		// caller could not tell which credential shape they actually hold.
		return nil, ErrAPIKeyRejected
	}

	admitted := false
	for _, w := range wantScopes {
		if key.Scope == w {
			admitted = true
			break
		}
	}
	if !admitted {
		return nil, fmt.Errorf("%w: this is a %s key", ErrAPIKeyWrongSurface, key.Scope)
	}

	// Best effort: an audit timestamp is worth less than the request it would
	// otherwise fail.
	if err := s.keys.TouchLastUsed(ctx, key.ID, time.Now().UTC()); err != nil {
		slog.WarnContext(ctx, "could not record api key use", "key_id", key.ID, "error", err)
	}
	return &Authenticated{Key: key}, nil
}
