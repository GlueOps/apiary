package apimiddleware

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The fail-open this file closes: RequireUserAuth set Impersonating only when
// uuid.Parse(claims.ActorID) SUCCEEDED, and swallowed the error. A token
// asserting imp=true whose act_id did not parse therefore produced a principal
// with Impersonating=false — an ordinary session, as far as everything
// downstream could tell. RefuseImpersonation() then never fired, and the
// impersonated operator could register a tenant cloud credential: the highest-
// value write in this system, and the exact thing the option exists to stop.
//
// These tests run the REAL middleware over a REAL signed token and a REAL live
// session row, into a handler guarded by the REAL AssertOrgRole. Nothing about
// the impersonation decision is simulated.

const (
	impTestSecret = "integrator-impersonation-suite-secret"
	impTestIssuer = "apiary-test"
)

func impDeps(db *gorm.DB) *app.Deps {
	return &app.Deps{
		Config: &config.Config{
			JWTSecret: impTestSecret,
			JWTIssuer: impTestIssuer,
		},
		ControlDB: db,
	}
}

// mintAccessToken signs the claims the way TokenService does, so the token
// under test is indistinguishable from a real one apart from act_id.
func mintAccessToken(t *testing.T, userID, orgID uuid.UUID, jti string, impersonate bool, actorID string) string {
	t.Helper()

	now := time.Now().UTC()
	claims := service.AuthClaims{
		OrgID:       orgID.String(),
		Type:        "access",
		ActorID:     actorID,
		Impersonate: impersonate,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID.String(),
			Issuer:    impTestIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(impTestSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// impersonationProbe wires RequireUserAuth in front of a write guarded exactly
// as the estate handlers guard theirs. reached reports whether the write ran.
func impersonationProbe(t *testing.T, db *gorm.DB, userID, orgID uuid.UUID) (humatest.TestAPI, *bool) {
	t.Helper()

	reached := false

	// Admin in the token's own organization, so role can never be the reason a
	// request is refused. The impersonation guard is the only thing left.
	fake := &recordingMemberships{
		t:          t,
		membership: membershipWithRole(userID, orgID, control.MemberRoleAdmin),
	}

	_, api := humatest.New(t, huma.DefaultConfig("imp-probe", "1.0.0"))
	api.UseMiddleware(RequireUserAuth(api, impDeps(db)))
	huma.Register(api, huma.Operation{
		OperationID: "imp-probe",
		Method:      http.MethodPost,
		Path:        "/probe",
	}, func(ctx context.Context, _ *struct{}) (*orgProbeOutput, error) {
		if _, err := AssertOrgRole(ctx, fake, control.MemberRoleAdmin, RefuseImpersonation()); err != nil {
			return nil, OrgRoleError(err)
		}
		reached = true
		out := &orgProbeOutput{}
		out.Body.OK = true
		return out, nil
	})
	return api, &reached
}

func TestRequireUserAuth_ImpersonationClaimWithAnUnresolvableActorIsRefused(t *testing.T) {
	db := testDB(t)
	userID, jti := seedRevokedSession(t, db, false)
	orgID := uuid.New()

	// Every way act_id can fail to yield an actor. Each one previously produced
	// Impersonating=false — a token that SAYS it is an impersonated session,
	// laundered into an ordinary one.
	for name, actorID := range map[string]string{
		"not a uuid":      "not-a-uuid",
		"empty":           "",
		"whitespace only": "   ",
		"truncated uuid":  "3f2504e0-4f89-11d3-9a0c",
	} {
		t.Run(name, func(t *testing.T) {
			api, reached := impersonationProbe(t, db, userID, orgID)

			resp := api.Post("/probe",
				"Authorization: Bearer "+mintAccessToken(t, userID, orgID, jti, true, actorID),
				map[string]string{})

			if *reached {
				t.Fatalf("a token claiming impersonation with an unresolvable actor performed the write; "+
					"RefuseImpersonation() never fired (act_id=%q, status %d)", actorID, resp.Code)
			}
			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("want 401 for an impersonation claim whose actor cannot be resolved, got %d: %s",
					resp.Code, resp.Body.String())
			}
		})
	}
}

func TestRequireUserAuth_WellFormedImpersonationIsStillRefusedAtTheWrite(t *testing.T) {
	// The positive control for the case above: a PROPERLY formed impersonation
	// token must get past authentication (401 is about the token, not the
	// action) and be stopped by RefuseImpersonation() with a 403. Without this,
	// the 401s above could just as well mean the whole probe is broken.
	db := testDB(t)
	userID, jti := seedRevokedSession(t, db, false)
	orgID := uuid.New()

	api, reached := impersonationProbe(t, db, userID, orgID)

	resp := api.Post("/probe",
		"Authorization: Bearer "+mintAccessToken(t, userID, orgID, jti, true, uuid.NewString()),
		map[string]string{})

	if *reached {
		t.Fatal("an impersonated session performed a write guarded by RefuseImpersonation()")
	}
	if resp.Code != http.StatusForbidden {
		t.Fatalf("want 403 for a well-formed impersonated write, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRequireUserAuth_OrdinarySessionIsUnaffected(t *testing.T) {
	// The other half of the control: closing the fail-open must not start
	// refusing ordinary, non-impersonated sessions. A stray act_id on a token
	// that does NOT claim impersonation is not an impersonation claim and must
	// not be treated as one.
	db := testDB(t)
	userID, jti := seedRevokedSession(t, db, false)
	orgID := uuid.New()

	for name, actorID := range map[string]string{
		"no actor":                            "",
		"stray unparseable actor without imp": "not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			api, reached := impersonationProbe(t, db, userID, orgID)

			resp := api.Post("/probe",
				"Authorization: Bearer "+mintAccessToken(t, userID, orgID, jti, false, actorID),
				map[string]string{})

			if resp.Code != http.StatusOK {
				t.Fatalf("an ordinary admin session must still be allowed to write, got %d: %s",
					resp.Code, resp.Body.String())
			}
			if !*reached {
				t.Fatal("the write did not run for an ordinary admin session")
			}
		})
	}
}
