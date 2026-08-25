package apimiddleware

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type stubInventory struct {
	has   bool
	err   error
	calls int
}

func (s *stubInventory) HasVerifiedFactor(context.Context, uuid.UUID) (bool, error) {
	s.calls++
	return s.has, s.err
}

// ffProbe mounts a handler behind the gate with a principal already installed,
// so these tests exercise the gate alone rather than re-testing authentication.
func ffProbe(t *testing.T, inv FactorInventory, p *Principal) (humatest.TestAPI, *bool) {
	t.Helper()
	reached := false
	_, api := humatest.New(t, huma.DefaultConfig("ff-probe", "1.0.0"))
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(hctx, WithPrincipal(hctx.Context(), p)))
	})
	api.UseMiddleware(firstFactorGate(api, inv))
	huma.Register(api, huma.Operation{
		OperationID: "ff-probe", Method: http.MethodPost, Path: "/ff-probe",
	}, func(context.Context, *struct{}) (*struct{}, error) {
		reached = true
		return &struct{}{}, nil
	})
	return api, &reached
}

func uid() *uuid.UUID { u := uuid.New(); return &u }

// The case the whole split exists for: locked out by policy, nothing enrolled,
// must be able to enrol.
func TestFirstFactorGateAdmitsAPreAuthzSessionWithNoFactor(t *testing.T) {
	inv := &stubInventory{has: false}
	api, reached := ffProbe(t, inv, &Principal{UserID: uid(), PolicySatisfied: false})

	// 2xx rather than exactly 200: an empty response body is 204.
	if resp := api.Post("/ff-probe"); resp.Code >= 300 {
		t.Fatalf("a user with no factor cannot enrol one, got %d: %s", resp.Code, resp.Body.String())
	}
	if !*reached {
		t.Fatal("handler never ran")
	}
}

// The bypass this gate exists to prevent.
//
// Someone holding only a stolen PASSWORD gets a pre-authz session. If they can
// enrol on an account that already has a factor, they add their own
// authenticator and then satisfy the policy with it — the victim's factor never
// participates. That is strictly worse than the lockout the split fixes.
func TestFirstFactorGateRefusesEnrolmentWhenAFactorAlreadyExists(t *testing.T) {
	inv := &stubInventory{has: true}
	api, reached := ffProbe(t, inv, &Principal{UserID: uid(), PolicySatisfied: false})

	resp := api.Post("/ff-probe")
	if resp.Code != http.StatusForbidden {
		t.Fatalf("a half-authenticated caller enrolled an authenticator on a protected account "+
			"(got %d); a stolen password would then satisfy MFA on its own", resp.Code)
	}
	if *reached {
		t.Fatal("the handler ran despite the refusal")
	}
	// Same wording as the policy gate. A distinct message would confirm to
	// someone holding a stolen password that the account is protected.
	if body := resp.Body.String(); !contains(body, "mfa_required") {
		t.Fatalf("the refusal does not match the policy gate's shape: %s", body)
	}
}

// A satisfied session is the ordinary case — someone managing their own factors
// — and must not pay for an inventory lookup it does not need.
func TestFirstFactorGateLetsASatisfiedSessionThroughWithoutALookup(t *testing.T) {
	inv := &stubInventory{has: true}
	api, reached := ffProbe(t, inv, &Principal{UserID: uid(), PolicySatisfied: true})

	if resp := api.Post("/ff-probe"); resp.Code >= 300 {
		t.Fatalf("a fully authorized user cannot add a second factor, got %d", resp.Code)
	}
	if !*reached {
		t.Fatal("handler never ran")
	}
	if inv.calls != 0 {
		t.Fatalf("a satisfied session consulted the factor inventory %d times", inv.calls)
	}
}

// A check that cannot reach a decision must deny. Denying costs a user with no
// factor their remediation path; admitting hands an attacker an enrolment.
func TestFirstFactorGateDeniesWhenItCannotDecide(t *testing.T) {
	cases := []struct {
		name string
		inv  FactorInventory
		p    *Principal
	}{
		{"inventory errors", &stubInventory{err: context.DeadlineExceeded}, &Principal{UserID: uid()}},
		{"inventory unwired", nil, &Principal{UserID: uid()}},
		{"no principal", &stubInventory{}, nil},
		{"principal without a user", &stubInventory{}, &Principal{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			api, reached := ffProbe(t, c.inv, c.p)
			if resp := api.Post("/ff-probe"); resp.Code < 300 {
				t.Fatalf("admitted on an undecidable check (got %d)", resp.Code)
			}
			if *reached {
				t.Fatal("handler ran")
			}
		})
	}
}

func contains(h, n string) bool {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return true
		}
	}
	return false
}
