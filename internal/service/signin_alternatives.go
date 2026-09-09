package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/models/control"
)

// SignInAlternatives answers whether any way into the installation survives
// besides the password form.
//
// An INTERFACE so the flag service does not acquire a dependency on the auth
// configuration or the passkey store. The direction matters: flags are the
// general mechanism and sign-in is one thing they happen to gate, so the
// knowledge flows inward as a small question rather than the flag service
// learning what a provider is.
type SignInAlternatives interface {
	// Available reports whether an alternative exists, and names it. The name is
	// for the refusal message: "configure a provider" and "you have a passkey"
	// lead an operator to completely different next steps.
	Available(ctx context.Context, actingUserID uuid.UUID) (bool, string, error)
}

// PasskeyCounter is the slice of the passkey store this needs.
type PasskeyCounter interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]control.UserPasskey, error)
}

// ConfiguredSignInAlternatives answers from the running configuration and the
// acting user's own credentials.
type ConfiguredSignInAlternatives struct {
	// Providers are the (name, id, secret) triples the social service handles.
	Providers []SocialProviderConfig
	Passkeys  PasskeyCounter
}

// SocialProviderConfig is one provider as configured.
type SocialProviderConfig struct {
	Name         string
	ClientID     string
	ClientSecret string
}

// Configured requires BOTH halves. A client id with no secret is a provider
// somebody started wiring up and did not finish; treating it as a working way in
// is how the guard would wave through the exact lockout it exists to prevent.
func (p SocialProviderConfig) Configured() bool {
	return strings.TrimSpace(p.ClientID) != "" && strings.TrimSpace(p.ClientSecret) != ""
}

// Available checks the installation first, then the caller.
//
// The order is deliberate. A configured provider is a way in for EVERYONE, which
// is the thing actually being protected. The acting user's passkey is weaker —
// it only guarantees that one person can get back in — but it is enough to make
// the change reversible by the person making it, which is the difference between
// an inconvenience and an outage.
func (c ConfiguredSignInAlternatives) Available(
	ctx context.Context, actingUserID uuid.UUID,
) (bool, string, error) {
	for _, p := range c.Providers {
		if p.Configured() {
			return true, fmt.Sprintf("the %s provider is configured", p.Name), nil
		}
	}

	if c.Passkeys != nil && actingUserID != uuid.Nil {
		keys, err := c.Passkeys.ListByUserID(ctx, actingUserID)
		if err != nil {
			// REPORTED, NOT SWALLOWED. This runs on the refusal path of a write,
			// so an error here must not read as "no alternative" — that would
			// refuse a legitimate change on a database blip — nor as "an
			// alternative exists", which would wave through a lockout. The caller
			// decides, and it denies the write.
			return false, "", fmt.Errorf("check the acting user's passkeys: %w", err)
		}
		if len(keys) > 0 {
			return true, "you have a passkey registered, so you could sign in and undo this", nil
		}
	}

	return false, "", nil
}
