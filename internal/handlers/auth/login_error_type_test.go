package auth

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// The UI branches on this type URI to decide whether to offer "resend the
// confirmation link" instead of "check your password". That makes the string
// part of the API contract, not an implementation detail: changing it silently
// removes the only recovery path an unconfirmed user has, and nothing else in
// the tree would fail.
//
// Matching on the human-readable message was the alternative and is why this
// test exists — prose gets reworded by people who have no idea a client depends
// on it. RFC 7807's `type` is the field that is explicitly allowed to be relied
// upon; `detail` is explicitly not.
func TestEmailNotVerifiedCarriesItsStableTypeURI(t *testing.T) {
	err := emailNotVerifiedError()

	var em *huma.ErrorModel
	if !errors.As(err, &em) {
		t.Fatalf("emailNotVerifiedError returned %T, which carries no type field for the client to "+
			"branch on", err)
	}
	if em.Type != ErrTypeEmailNotVerified {
		t.Fatalf("type = %q, want %q — the UI routes unconfirmed users to the resend page on this "+
			"exact value", em.Type, ErrTypeEmailNotVerified)
	}
	if em.Status != 403 {
		t.Fatalf("status = %d, want 403", em.Status)
	}
	if em.Detail == "" {
		t.Fatal("detail is empty; the message is still what a human reads")
	}
}
