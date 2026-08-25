package estate

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/repo"
)

// statusOf pulls the status back out of a huma error.
func statusOf(t *testing.T, err error) (int, string) {
	t.Helper()
	if err == nil {
		return 0, ""
	}
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("mapErr returned %T, which huma renders as 500 whatever it says: %v", err, err)
	}
	return se.GetStatus(), err.Error()
}

// TestMapErrConflictArmsAreReachedByErrorsIs covers the two 409 mappings.
//
// These are called DIRECTLY, and that is a deliberate, stated exception rather
// than the usual shortcut. They cannot be reached end to end today:
// EstateService.Register never sets DerivedScope, so nothing can disagree with
// it, and it passes the same provider string to both the credential and the
// estate, so the two cannot mismatch. A handler test for them would have to
// fabricate an error the production path cannot produce, which would be a test
// of the fabrication.
//
// Reported upward as a finding instead: two mapped statuses with no reachable
// producer. Covering them here keeps them correct for when a provider module
// lands and makes them live.
func TestMapErrConflictArmsAreReachedByErrorsIs(t *testing.T) {
	cases := map[string]error{
		"scope disagrees":   repo.ErrScopeDisagrees,
		"provider mismatch": repo.ErrProviderMismatch,
	}

	for name, base := range cases {
		t.Run(name, func(t *testing.T) {
			if got, _ := statusOf(t, mapErr(t.Context(), base)); got != http.StatusConflict {
				t.Errorf("bare %v mapped to %d, want 409: this is a mis-filing — a credential that is "+
					"not for the target it was filed under — not a malformed request", base, got)
			}

			// Wrapped, because that is how it actually arrives: the repository
			// wraps these with the offending values and the service passes them
			// up through a transaction closure. Matching on equality instead of
			// errors.Is would pass the bare case above and drop the real one to
			// 500.
			wrapped := fmt.Errorf("create estate: %w", base)
			if got, _ := statusOf(t, mapErr(t.Context(), wrapped)); got != http.StatusConflict {
				t.Errorf("wrapped %v mapped to %d, want 409 — mapErr is comparing errors by equality, "+
					"so every real (wrapped) occurrence becomes a 500", base, got)
			}
		})
	}
}

// TestMapErrDefaultBranchDoesNotEchoTheError pins the fallback.
//
// The default branch is the one place an arbitrary error from anywhere below —
// the driver, the sealing code, a provider client — reaches a response. If it
// returned err.Error() the caller would receive whatever that error happens to
// contain, and errors in this stack are constructed near credential material.
func TestMapErrDefaultBranchDoesNotEchoTheError(t *testing.T) {
	leak := errors.New("seal credential: aead open failed for AKIAIOSFODNN7EXAMPLE / " + theSecret)

	status, msg := statusOf(t, mapErr(t.Context(), leak))
	if status != http.StatusInternalServerError {
		t.Fatalf("an unrecognised error mapped to %d, want 500", status)
	}
	if msg != "estate operation failed" {
		t.Errorf("the 500 body is %q, want exactly \"estate operation failed\": returning the wrapped "+
			"error text is how driver internals and credential material reach a caller", msg)
	}
	if kind := mentionsSecret(msg); kind != "" {
		t.Errorf("the 500 body carries secret material (%s encoding): %s", kind, msg)
	}
}

// TestMapErrPassesNilThrough guards the one case that must NOT become an error:
// a nil error mapped to a 500 would turn every successful request into a
// failure, which is a different kind of outage but still an outage.
func TestMapErrPassesNilThrough(t *testing.T) {
	if err := mapErr(t.Context(), nil); err != nil {
		t.Fatalf("mapErr(nil) returned %v", err)
	}
}
