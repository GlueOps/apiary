package service

import (
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// A passkey is identity AND second factor in one object, and that is only true
// when the authenticator performed USER VERIFICATION — a PIN or a biometric.
// Possession alone is one factor.
//
// PasskeyService reads that from the assertion (MFA: cred.Flags.UserVerified)
// rather than assuming it, which is right. What makes the flag reliably true is
// this config: go-webauthn feeds Config.AuthenticatorSelection.UserVerification
// into BeginDiscoverableLogin, so a non-verifying assertion is rejected before
// any flag is read.
//
// Relaxing it to Preferred — the usual compatibility change, for security keys
// without a PIN — would resurrect the lockout the pre-authz split just fixed,
// and in its worst form:
//
//	login succeeds without UV  ->  MFAEnabled=true, MFA=false
//	the policy gate refuses every request
//	the pre-authz path refuses too, because the account HAS a factor
//	                                  (the passkey they just logged in with)
//	no remedy exists
//
// The user cannot enrol their way out, because RequireFirstFactorEnrolment
// correctly refuses to let a half-authenticated session add an authenticator to
// an account that already has one. So this is not a preference — it is what
// keeps a whole class of account recoverable.
func TestPasskeyConfigRequiresUserVerification(t *testing.T) {
	// The configuration the service actually builds, not a copy of it — a copy
	// would keep passing after someone relaxed the real one.
	cfg := passkeyWebAuthnConfig("APIary", "localhost", "http://localhost:5173")
	if _, err := webauthn.New(cfg); err != nil {
		t.Fatalf("the configuration this service builds is not valid: %v", err)
	}

	if cfg.AuthenticatorSelection.UserVerification != protocol.VerificationRequired {
		t.Fatal("user verification is not required")
	}

	// The dependency stated as an assertion: this value is what the login
	// ceremony inherits, so a passkey login cannot report MFA satisfied on
	// possession alone.
	if got := cfg.AuthenticatorSelection.UserVerification; got == protocol.VerificationPreferred ||
		got == protocol.VerificationDiscouraged {
		t.Fatalf("user verification is %q; a passkey login would then set MFA=false while "+
			"MFAEnabled stays true, and the account has no way to satisfy the policy: the "+
			"first-factor gate refuses enrolment to an account that already holds a passkey", got)
	}
}
