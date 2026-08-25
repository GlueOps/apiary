package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/google/uuid"
)

func testApp(t *testing.T) *App {
	t.Helper()

	tokenSvc, err := service.NewTokenService("test-secret", "apiary-test", 15*time.Minute, 168*time.Hour)
	if err != nil {
		t.Fatalf("token service: %v", err)
	}

	userRepo := repo.NewUserRepoGorm(nil)
	deps := &app.Deps{
		Config: &config.Config{
			JWTSecret: "test-secret",
			JWTIssuer: "apiary-test",
			BasePath:  "/api/v1",
			BaseURL:   "http://localhost:8080",
			Env:       "test",
		},
		TokenSessions: service.NewTokenSessionService(
			tokenSvc,
			repo.NewTokenSessionRepoGorm(nil),
			repo.NewAuthAuditRepoGorm(nil),
			&userRepo,
		),
	}

	cfg := *deps.Config
	cfg.FrontendMode = "none"
	cfg.CORSAllowedOrigins = "http://localhost:8080"
	cfg.BindHost = "127.0.0.1"
	cfg.BindPort = "8080"

	built, err := Build(cfg, deps)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	return built
}

func concreteURL(path string) string {
	path = strings.ReplaceAll(path, "{id}", uuid.NewString())
	path = strings.ReplaceAll(path, "{provider}", "google")
	return path
}

func operations(t *testing.T, a *App) []struct {
	Method  string
	Path    string
	ID      string
	Secured bool
} {
	t.Helper()

	var out []struct {
		Method  string
		Path    string
		ID      string
		Secured bool
	}

	spec := a.API.OpenAPI()
	for path, item := range spec.Paths {
		for method, op := range map[string]*huma.Operation{
			http.MethodGet:    item.Get,
			http.MethodPost:   item.Post,
			http.MethodPut:    item.Put,
			http.MethodPatch:  item.Patch,
			http.MethodDelete: item.Delete,
		} {
			if op == nil {
				continue
			}
			secured := false
			for _, req := range op.Security {
				if _, ok := req["BearerAuth"]; ok {
					secured = true
				}
			}
			out = append(out, struct {
				Method  string
				Path    string
				ID      string
				Secured bool
			}{method, path, op.OperationID, secured})
		}
	}
	return out
}

func TestRoutesAreActuallyMounted(t *testing.T) {
	a := testApp(t)
	ops := operations(t, a)

	if len(ops) == 0 {
		t.Fatal("no operations registered: handlers.RegisterPublic/RegisterUserProtected are not wiring anything")
	}

	// Every operation that must exist. RegisterSignup was defined and never
	// called from RegisterPublic, so /auth/register 404'd in a shipped build
	// while every other signal looked healthy — the handler compiled, the
	// service was wired, and the admission gate it fronts was fully tested.
	// A route is only real once something registers it.
	want := []string{
		"auth-register", "auth-login", "auth-logout", "auth-refresh",
		"auth-forgot-password", "auth-reset-password",
		"auth-verify-email", "auth-resend-verification",
		"invitations-preview", "invitations-accept",
		"org-invitations-create", "org-invitations-list", "org-invitations-revoke",
		"org-members-list", "org-members-update-role", "org-members-remove",
		"admin-impersonate-start", "admin-impersonate-end",
		"platform-policies-list", "platform-policies-set", "platform-policies-clear",
		"org-policies-list", "org-policies-set", "org-policies-clear",
		"fleet-bee-enrol", "fleet-bee-heartbeat", "fleet-bees-list", "fleet-tasks-create",
	}
	have := map[string]bool{}
	for _, op := range ops {
		have[op.ID] = true
	}
	for _, id := range want {
		if !have[id] {
			t.Errorf("operation %q is not registered", id)
		}
	}
	t.Logf("%d operations registered", len(ops))
}

func TestSecuredOperationsRejectAnonymousCallers(t *testing.T) {
	a := testApp(t)

	checked := 0
	for _, op := range operations(t, a) {
		if !op.Secured {
			continue
		}
		checked++

		t.Run(op.ID, func(t *testing.T) {
			req := httptest.NewRequest(op.Method, concreteURL(op.Path), nil)
			rec := httptest.NewRecorder()
			a.Router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s declares BearerAuth but answered an anonymous caller with %d, not 401 — "+
					"it is almost certainly registered on the public group",
					op.Method, op.Path, rec.Code)
			}
		})
	}

	if checked == 0 {
		t.Fatal("no operation declares BearerAuth — the protected group is empty, which cannot be right")
	}
	t.Logf("%d secured operations rejected anonymous callers", checked)
}

func TestPublicOperationsDoNotRequireAuth(t *testing.T) {
	a := testApp(t)

	for _, op := range operations(t, a) {
		if op.Secured {
			continue
		}

		t.Run(op.ID, func(t *testing.T) {
			req := httptest.NewRequest(op.Method, concreteURL(op.Path), nil)
			rec := httptest.NewRecorder()
			a.Router.ServeHTTP(rec, req)

			if rec.Code == http.StatusUnauthorized && strings.Contains(rec.Body.String(), "missing bearer token") {
				t.Fatalf("%s %s declares no security but is behind RequireUserAuth: %s",
					op.Method, op.Path, rec.Body.String())
			}
		})
	}
}

// TestInvitationSurfaceKeepsItsPublicPrivateSplit pins, BY NAME, which side of
// the auth boundary each invitation and membership endpoint sits on.
//
// The generic sweeps above compare each operation's declared security against
// its actual behaviour, which is the right check but cannot notice an endpoint
// that has been deleted or renamed — an empty sweep passes. This one names the
// operations, so removing or renaming one fails here rather than silently
// reducing coverage.
//
// It asserts REACHABILITY, not the declared metadata. An earlier version of this
// test read op.Secured, which is the security block the handler declares; moving
// the public accept endpoints behind the bearer middleware left it green,
// because a handler that declares no security still declares no security no
// matter which group registers it. Verified by mutation.
//
// Both mistakes it guards against are silent. Accept behind the bearer
// middleware breaks the flow for the people it exists for — an invitee usually
// has no account yet, so requiring a session would force them to register, which
// mints an organization of their own, which is exactly what an invitee should
// not be doing. The creation endpoints in front of it would let anyone add
// themselves to any organization.
func TestInvitationSurfaceKeepsItsPublicPrivateSplit(t *testing.T) {
	a := testApp(t)

	// Possession of the emailed token IS the authorization for these two.
	mustBeReachableAnonymously := map[string]bool{
		"invitations-preview": true,
		"invitations-accept":  true,
		// A host being brought up has no credential yet; the enrolment token is
		// the authorization.
		"fleet-bee-enrol": true,
	}
	mustRejectAnonymously := map[string]bool{
		"org-invitations-create":  true,
		"org-invitations-list":    true,
		"org-invitations-revoke":  true,
		"org-members-list":        true,
		"org-members-update-role": true,
		"org-members-remove":      true,
		"admin-impersonate-start": true,
		"admin-impersonate-end":   true,
		"platform-policies-list":  true,
		"platform-policies-set":   true,
		"fleet-hives-create":      true,
		"fleet-tasks-logs":        true,
		"org-policies-list":       true,
		"org-policies-set":        true,
		"fleet-bees-list":         true,
		"fleet-tasks-create":      true,
	}

	seen := map[string]bool{}
	for _, op := range operations(t, a) {
		if !mustBeReachableAnonymously[op.ID] && !mustRejectAnonymously[op.ID] {
			continue
		}
		seen[op.ID] = true

		t.Run(op.ID, func(t *testing.T) {
			req := httptest.NewRequest(op.Method, concreteURL(op.Path), nil)
			rec := httptest.NewRecorder()
			a.Router.ServeHTTP(rec, req)

			blocked := rec.Code == http.StatusUnauthorized

			if mustBeReachableAnonymously[op.ID] && blocked {
				t.Fatalf("%s %s answered an anonymous caller with 401 — it is public by design; "+
					"an invitee has no account yet and cannot be asked to sign in first: %s",
					op.Method, op.Path, rec.Body.String())
			}
			if mustRejectAnonymously[op.ID] && !blocked {
				t.Fatalf("%s %s answered an anonymous caller with %d, not 401 — it reads or grants "+
					"organization membership and must require a session",
					op.Method, op.Path, rec.Code)
			}
		})
	}

	for id := range mustBeReachableAnonymously {
		if !seen[id] {
			t.Errorf("operation %q was not found at all — this test is checking nothing about it", id)
		}
	}
	for id := range mustRejectAnonymously {
		if !seen[id] {
			t.Errorf("operation %q was not found at all — this test is checking nothing about it", id)
		}
	}
}
