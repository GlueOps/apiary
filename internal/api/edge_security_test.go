package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
)

func buildWith(t *testing.T, mutate func(cfg *config.Config)) (*App, error) {
	t.Helper()

	tokenSvc, err := service.NewTokenService("test-secret", "apiary-test", 15*time.Minute, 168*time.Hour)
	if err != nil {
		t.Fatalf("token service: %v", err)
	}
	userRepo := repo.NewUserRepoGorm(nil)

	cfg := config.Config{
		JWTSecret:    "test-secret",
		JWTIssuer:    "apiary-test",
		BasePath:     "/api/v1",
		BaseURL:      "http://localhost:8080",
		Env:          "test",
		BindHost:     "127.0.0.1",
		BindPort:     "8080",
		FrontendMode: config.FrontendModeNone,
	}
	mutate(&cfg)

	deps := &app.Deps{
		Config: &cfg,
		TokenSessions: service.NewTokenSessionService(
			tokenSvc,
			repo.NewTokenSessionRepoGorm(nil),
			repo.NewAuthAuditRepoGorm(nil),
			&userRepo,
		),
	}
	return Build(cfg, deps)
}

func corsFor(t *testing.T, a *App, origin string) (allowOrigin string, allowCreds bool) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", origin)
	rec := httptest.NewRecorder()
	a.Router.ServeHTTP(rec, req)

	return rec.Header().Get("Access-Control-Allow-Origin"),
		strings.EqualFold(rec.Header().Get("Access-Control-Allow-Credentials"), "true")
}

func TestCORSDoesNotReflectArbitraryOriginsOutsideDevelopment(t *testing.T) {
	const evil = "https://evil.example"

	cases := []struct {
		name    string
		env     string
		origins string
		baseURL string
	}{
		{"staging with wildcard", "staging", "*", "http://localhost:8080"},
		{"staging with nothing configured", "staging", "", "http://localhost:8080"},
		{"prod not spelled production", "prod", "*", "http://localhost:8080"},
		{"unset env", "", "*", "http://localhost:8080"},
		{"env abbreviated to dev", "dev", "*", "http://localhost:8080"},
		{"production and no usable base url", "production", "*", "not-a-url"},
		{"production with an unusable origin list", "production", "https://app.example.com/app/", "not-a-url"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := buildWith(t, func(cfg *config.Config) {
				cfg.Env = tc.env
				cfg.CORSAllowedOrigins = tc.origins
				cfg.BaseURL = tc.baseURL
			})
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			got, creds := corsFor(t, a, evil)
			if got == evil {
				t.Fatalf("ENV=%q CORS_ALLOWED_ORIGINS=%q reflected the caller's origin (credentials=%v): "+
					"any web page could read authenticated responses", tc.env, tc.origins, creds)
			}
			if got == "*" {
				t.Fatalf("ENV=%q CORS_ALLOWED_ORIGINS=%q answered Access-Control-Allow-Origin: * "+
					"on a credentialed API", tc.env, tc.origins)
			}
			if got != "" {
				t.Fatalf("ENV=%q unexpectedly allowed origin %q", tc.env, got)
			}
		})
	}
}

func TestCORSReflectsOnlyInAllowListedDevEnvironments(t *testing.T) {
	for _, env := range []string{"development", "  Development ", "test"} {
		t.Run(env, func(t *testing.T) {
			a, err := buildWith(t, func(cfg *config.Config) {
				cfg.Env = env
				cfg.CORSAllowedOrigins = ""
			})
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			got, creds := corsFor(t, a, "http://localhost:5173")
			if got != "http://localhost:5173" || !creds {
				t.Fatalf("ENV=%q should reflect for local development, got origin=%q credentials=%v", env, got, creds)
			}
		})
	}
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	a, err := buildWith(t, func(cfg *config.Config) {
		cfg.Env = "production"

		cfg.CORSAllowedOrigins = "https://App.Example.com, *, https://other.example.com/path, api.example.com"
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if got, creds := corsFor(t, a, "https://app.example.com"); got != "https://app.example.com" || !creds {
		t.Fatalf("configured origin not allowed: origin=%q credentials=%v", got, creds)
	}
	if got, _ := corsFor(t, a, "https://other.example.com"); got != "https://other.example.com" {
		t.Fatalf("entry with a path was not reduced to its origin: %q", got)
	}
	if got, _ := corsFor(t, a, "https://evil.example"); got != "" {
		t.Fatalf("unconfigured origin allowed: %q", got)
	}

	if got, _ := corsFor(t, a, "https://api.example.com"); got != "" {
		t.Fatalf("bare hostname entry was treated as an origin: %q", got)
	}
	if got, _ := corsFor(t, a, "http://api.example.com"); got != "" {
		t.Fatalf("bare hostname entry allowed a plaintext origin: %q", got)
	}
}

func TestCORSFallsBackToTheOwnOriginOfBaseURL(t *testing.T) {
	a, err := buildWith(t, func(cfg *config.Config) {
		cfg.Env = "production"
		cfg.CORSAllowedOrigins = ""
		cfg.BaseURL = "https://api.example.com/api/v1"
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got, _ := corsFor(t, a, "https://api.example.com"); got != "https://api.example.com" {
		t.Fatalf("own origin not allowed: %q", got)
	}
	if got, _ := corsFor(t, a, "https://evil.example"); got != "" {
		t.Fatalf("foreign origin allowed: %q", got)
	}
}

func TestCORSOriginPolicyNeverReturnsTheAllowAllShape(t *testing.T) {
	raws := []string{"", "*", " , ", "*,*", "not-an-origin", "https://a.example/path", "https://a.example"}
	envs := []string{"", "development", "test", "staging", "production", "PROD", "dev"}
	baseURLs := []string{"", "not-a-url", "http://localhost:8080", "ftp://x.example"}

	for _, raw := range raws {
		for _, env := range envs {
			for _, base := range baseURLs {
				origins, fn := corsOriginPolicy(raw, env, base)
				if len(origins) == 0 && fn == nil {
					t.Fatalf("corsOriginPolicy(%q, %q, %q) returned the allow-all shape "+
						"(empty AllowedOrigins, nil AllowOriginFunc)", raw, env, base)
				}
				if len(origins) > 0 && fn != nil {
					t.Fatalf("corsOriginPolicy(%q, %q, %q) returned both a list and a func; "+
						"AllowOriginFunc silently wins and the list is dead config", raw, env, base)
				}
				for _, o := range origins {
					if o == "*" || o == "" {
						t.Fatalf("corsOriginPolicy(%q, %q, %q) put %q in the allow list, which "+
							"collapses go-chi to allow-all", raw, env, base, o)
					}
				}
			}
		}
	}
}

func TestFrontendProxyIsRefusedOutsideDevelopment(t *testing.T) {
	for _, env := range []string{"", "staging", "prod", "production", "dev"} {
		t.Run("env="+env, func(t *testing.T) {
			_, err := buildWith(t, func(cfg *config.Config) {
				cfg.Env = env
				cfg.FrontendMode = "proxy"
			})
			if err == nil {
				t.Fatalf("Build mounted the vite reverse proxy with ENV=%q", env)
			}
			if !strings.Contains(err.Error(), "development-only") {
				t.Fatalf("unhelpful refusal for ENV=%q: %v", env, err)
			}
		})
	}

	router := chiLikeRouter{}
	if err := WireFrontend(&router, "PROXY", "staging", nil, "http://localhost:5173", "/api/v1"); err == nil {
		t.Fatal("WireFrontend mounted the proxy for ENV=staging")
	}
	if router.notFound != nil {
		t.Fatal("WireFrontend installed a NotFound handler despite refusing the mode")
	}
	if err := WireFrontend(&router, "proxy", "development", nil, "http://localhost:5173", "/api/v1"); err != nil {
		t.Fatalf("WireFrontend refused the proxy in development: %v", err)
	}
}

func TestFrontendUnknownModeIsAnError(t *testing.T) {
	if _, err := buildWith(t, func(cfg *config.Config) { cfg.FrontendMode = "prox" }); err == nil {
		t.Fatal("Build accepted FRONTEND_MODE=prox")
	}
}

type chiLikeRouter struct {
	notFound http.HandlerFunc
}

func (r *chiLikeRouter) Handle(string, http.Handler) {}
func (r *chiLikeRouter) NotFound(h http.HandlerFunc) { r.notFound = h }

func TestUnmatchedPathsAreRateLimited(t *testing.T) {
	a, err := buildWith(t, func(cfg *config.Config) {
		cfg.Env = "test"
		cfg.CORSAllowedOrigins = "http://localhost:8080"
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	const budget = 600
	limited := false
	for i := 0; i < budget+50 && !limited; i++ {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/not-a-route/%d", i), nil)
		req.RemoteAddr = "203.0.113.7:1234"
		rec := httptest.NewRecorder()
		a.Router.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			limited = true
			if i < 100 {
				t.Fatalf("unmatched paths were limited after only %d requests; that budget will break the vite dev proxy", i)
			}
		}
	}
	if !limited {
		t.Fatalf("%d requests to unmatched paths were all served: the rate limiter's default branch has no limiter",
			budget+50)
	}
}

// TestUnknownAPIRoutesDoNotReachTheFrontend pins the boundary between the API
// and the frontend fallback.
//
// The frontend is mounted on every unmatched path, so a request to an API route
// that does not exist reaches it. Neither mode then answers in a way an API
// client can read: embed serves index.html with 200 for a browser Accept header,
// so a missing endpoint reports SUCCESS and returns HTML; proxy forwards to Vite,
// whose config proxies the API prefix straight back, and the request rebounds
// until something gives up — the reported symptom was a 502 for a route that
// simply is not there.
//
// Both are worse than the truth, and the truth is a 404.
func TestUnknownAPIRoutesDoNotReachTheFrontend(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>spa</title>")},
	}

	modes := []struct {
		name string
		wire func(r *chiLikeRouter) error
	}{
		{"embed", func(r *chiLikeRouter) error {
			return WireFrontend(r, "embed", "production", dist, "", "/api/v1")
		}},
		{"proxy", func(r *chiLikeRouter) error {
			return WireFrontend(r, "proxy", "development", nil, "http://127.0.0.1:1", "/api/v1")
		}},
	}

	// Every method, because a fallback that only guards GET leaves a POST to a
	// retired endpoint being reverse-proxied. Every Accept header, because that
	// is precisely what decides the wrong answer in embed mode.
	cases := []struct{ method, path, accept string }{
		{http.MethodGet, "/api/v1/nope", "text/html"},
		{http.MethodGet, "/api/v1/nope", "application/json"},
		{http.MethodGet, "/api/v1/nope", "*/*"},
		{http.MethodGet, "/api/v1/fleet/nope", "text/html"},
		{http.MethodPost, "/api/v1/nope", "application/json"},
		{http.MethodDelete, "/api/v1/fleet/bees/x/nope", "*/*"},
		// The prefix itself, with nothing after it.
		{http.MethodGet, "/api/v1", "text/html"},
	}

	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			router := chiLikeRouter{}
			if err := m.wire(&router); err != nil {
				t.Fatalf("wire: %v", err)
			}
			if router.notFound == nil {
				t.Fatal("no NotFound handler was installed")
			}

			for _, c := range cases {
				req := httptest.NewRequest(c.method, c.path, nil)
				req.Header.Set("Accept", c.accept)
				rec := httptest.NewRecorder()
				router.notFound(rec, req)

				if rec.Code != http.StatusNotFound {
					t.Errorf("%s %s (Accept: %s) answered %d, want 404; body: %s",
						c.method, c.path, c.accept, rec.Code, rec.Body.String())
					continue
				}
				if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
					t.Errorf("%s %s (Accept: %s) answered 404 as %q — a client that parses "+
						"errors as JSON cannot read it", c.method, c.path, c.accept, ct)
				}
				if strings.Contains(rec.Body.String(), "<!doctype") {
					t.Errorf("%s %s (Accept: %s) was served the SPA", c.method, c.path, c.accept)
				}
			}

			// The guard must not swallow the frontend. A real page still renders.
			req := httptest.NewRequest(http.MethodGet, "/settings/security", nil)
			req.Header.Set("Accept", "text/html")
			rec := httptest.NewRecorder()
			router.notFound(rec, req)
			if m.name == "embed" && rec.Code != http.StatusOK {
				t.Errorf("an ordinary SPA route answered %d, want 200", rec.Code)
			}
			if m.name == "embed" && !strings.Contains(rec.Body.String(), "<!doctype") {
				t.Error("an ordinary SPA route was not served index.html")
			}
		})
	}
}
