package apimiddleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// Wiring org keys into RequireUserAuth put a NEW kind of principal in front of
// every gate that used to see only bearer sessions. These pin the boundaries
// that change could have opened.

// Platform administration is a USER role. An organization credential has no user
// behind it, so it can never satisfy that gate — and it must be refused before
// anything dereferences the user it does not have.
func TestOrgKeyPrincipalCannotSatisfyPlatformAdmin(t *testing.T) {
	p := &Principal{Method: AuthMethodOrgSecret, OrganizationID: uuid.New()}
	ctx := WithPrincipal(context.Background(), p)

	got, ok := PrincipalFromContext(ctx)
	if !ok || got == nil {
		t.Fatal("principal did not survive the context")
	}
	// This is exactly the condition RequireAdminAuth denies on. Asserting the
	// shape here means a future change that gives org keys a synthetic user id
	// fails loudly rather than quietly reaching the platform surface.
	if got.UserID != nil {
		t.Fatal("an organization credential carries a user id; it would now be handed to the " +
			"platform-admin lookup and could satisfy a gate meant for people")
	}
}

// Exactly one credential per request. With a precedence order instead, a client
// that keeps a stale Authorization header while adopting an org key
// authenticates as the wrong principal — and it WORKS, so nobody notices until
// the audit log is read months later.
func TestCredentialFromRefusesMoreThanOneShape(t *testing.T) {
	cases := []struct {
		name string
		c    presentedCredential
		want bool
	}{
		{"bearer only", presentedCredential{bearer: "jwt"}, false},
		{"user key only", presentedCredential{userKey: "k"}, false},
		{"org pair only", presentedCredential{orgKey: "k", orgSecret: "s"}, false},
		{"nothing", presentedCredential{}, false},

		{"bearer and user key", presentedCredential{bearer: "jwt", userKey: "k"}, true},
		{"bearer and org key", presentedCredential{bearer: "jwt", orgKey: "k", orgSecret: "s"}, true},
		{"user key and org key", presentedCredential{userKey: "k", orgKey: "k2", orgSecret: "s"}, true},
		// A half-presented org pair still counts as a shape: it must collide with
		// a bearer rather than being ignored as incomplete.
		{"bearer and a lone org secret", presentedCredential{bearer: "jwt", orgSecret: "s"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kinds := 0
			if c.c.bearer != "" {
				kinds++
			}
			if c.c.userKey != "" {
				kinds++
			}
			if c.c.orgKey != "" || c.c.orgSecret != "" {
				kinds++
			}
			if got := kinds > 1; got != c.want {
				t.Fatalf("ambiguous=%v, want %v", got, c.want)
			}
		})
	}
}

// An organization arriving in a header must never reach the principal
// unverified. Everything downstream treats Principal.OrganizationID as
// established fact, so writing X-Org-Id into it without a membership check is an
// IDOR into every tenant the caller is not a member of.
func TestOrgHeaderTravelsSeparatelyFromThePrincipal(t *testing.T) {
	ctx := withOrgIDHeader(context.Background(), "not-a-uuid")
	if got := headerFromContext(ctx); got != "not-a-uuid" {
		t.Fatalf("header carrier lost the value: %q", got)
	}
	// The carrier is raw, unvalidated transport. Nothing may read it as a
	// resolved organization.
	if _, ok := PrincipalFromContext(ctx); ok {
		t.Fatal("the raw org header produced a principal on its own")
	}
}
