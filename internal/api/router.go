package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/buildinfo"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/handlers"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

type App struct {
	Router        *chi.Mux
	API           huma.API
	PublicBaseURL string
	BasePath      string
}

func Build(cfg config.Config, deps *app.Deps) (*App, error) {
	// Before any operation is registered: a validation failure must not hand
	// the caller back the body they sent, which for these endpoints includes
	// passwords and provider secrets. See stripErrorValues.
	stripErrorValues()

	router := chi.NewMux()
	router.Use(middleware.RequestID)

	router.Use(clientIPMiddleware(cfg.TrustedProxyCIDRs))
	router.Use(requestLogger())
	router.Use(middleware.Recoverer)

	corsOpts := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Real-Ip"},
		AllowCredentials: true,
		MaxAge:           300,
	}

	allowedOrigins, originFunc := corsOriginPolicy(cfg.CORSAllowedOrigins, cfg.Env, cfg.BaseURL)
	corsOpts.AllowedOrigins = allowedOrigins
	corsOpts.AllowOriginFunc = originFunc
	router.Use(cors.Handler(corsOpts))

	apiPrefix := cfg.BasePath + "/"
	authPrefix := cfg.BasePath + "/auth/"

	keyByClientIP := func(r *http.Request) (string, error) {
		return ClientIP(r), nil
	}
	// LimitBy, not Limit: it makes the key function a required argument. With
	// the deprecated form the key was an OPTION, so omitting it silently gave
	// one global bucket for every caller — a rate limiter that is really a
	// availability switch for the whole service.
	// A JSON body on 429, matching every other error this API returns.
	//
	// httprate's default handler writes text/plain, so a rate-limited client
	// got a response it could not parse: the generated SDKs decode by content
	// type and reported it as "undefined response type", which tells an operator
	// nothing about what actually happened. An API whose errors are JSON except
	// when rate-limited is an API whose clients cannot handle rate limiting.
	limited := httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"Too Many Requests","status":429,` +
			`"detail":"too many requests from this client; retry after the interval in Retry-After"}`))
	})

	globalLimiter := httprate.LimitBy(200, time.Minute, keyByClientIP, limited)
	authLimiter := httprate.LimitBy(20, time.Minute, keyByClientIP, limited)

	frontendLimiter := httprate.LimitBy(600, time.Minute, keyByClientIP, limited)
	router.Use(func(next http.Handler) http.Handler {
		globalLimited := globalLimiter(next)
		authLimited := authLimiter(next)
		frontendLimited := frontendLimiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, authPrefix):
				authLimited.ServeHTTP(w, r)
			case strings.HasPrefix(r.URL.Path, apiPrefix),

				strings.HasPrefix(r.URL.Path, "/admin/"),
				r.URL.Path == "/healthz",
				r.URL.Path == "/readyz",
				r.URL.Path == "/docs":
				globalLimited.ServeHTTP(w, r)
			default:

				frontendLimited.ServeHTTP(w, r)
			}
		})
	})

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data:; connect-src 'self'; font-src 'self' data:; "+
					"object-src 'none'; frame-ancestors 'none'; base-uri 'self'")

			if !config.IsDevEnv(cfg.Env) {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	})

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
			next.ServeHTTP(w, r)
		})
	})

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := deps.DBPool.Ping(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "readiness probe failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	httpAddr := fmt.Sprintf("%s:%s", cfg.BindHost, cfg.BindPort)
	publicBase := computePublicBaseURL(cfg.BaseURL, httpAddr)

	humaCfg := huma.DefaultConfig("APIary", buildinfo.Version)
	humaCfg.DocsPath = ""
	humaCfg.OpenAPIPath = cfg.BasePath + "/openapi"
	humaCfg.SchemasPath = cfg.BasePath + "/schemas"
	humaCfg.Servers = []*huma.Server{
		{
			URL:         fmt.Sprintf("http://%s:%s", cfg.BindHost, cfg.BindPort),
			Description: "API Server (local)",
		},
	}

	// Key grouping:::
	// humaCfg.Security = []map[string][]string{
	//	{"BearerAuth": {}}, => extra x-org-id where required
	//	{"ApiKeyAuth": {}},
	//	{"OrgKeyAuth": {}, "OrgKeySecret": {}},
	//	{"AgentKeyAuth": {}, "AgentKeySecret": {}}, => extra fleet-id where required ??
	//}
	humaCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"BearerAuth": {
			Type:         "http",
			Scheme:       "Bearer",
			BearerFormat: "JWT",
			Description:  "Standard JWT Bearer token",
		},
		"ApiKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-API-Key",
			Description: "API key authentication",
		},
		"OrgKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-ORG-Key",
			Description: "API key authentication",
		},
		"OrgKeySecret": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-ORG-Secret",
			Description: "API key authentication",
		},
		"AgentKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-AGENT-Key",
			Description: "API key authentication",
		},
		"AgentKeySecret": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-AGENT-Secret",
			Description: "API key authentication",
		},
	}

	apiServer := humachi.New(router, humaCfg)

	router.Get("/docs", func(w http.ResponseWriter, r *http.Request) {

		if !config.IsDevEnv(cfg.Env) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// script-src must name the host the bundle is actually fetched from.
		// This still said unpkg.com after the swap from RapiDoc to Scalar, which
		// loads from jsdelivr — so the CSP was blocking the docs bundle outright.
		//
		// connect-src stays 'self' deliberately, and is the reason this page
		// cannot reach an AI or MCP backend even if the Scalar configuration
		// below were wrong: the browser refuses the request whatever the script
		// asks for. Disabling the features in configuration is the control;
		// this is the backstop.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; "+
				"font-src 'self' data:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'")
		specURL := cfg.BasePath + "/openapi.json"
		serverURL := strings.TrimRight(publicBase, "/")
		_, _ = w.Write([]byte(renderScalarHTML(specURL, serverURL)))
	})

	frontendMode, err := config.NormalizeFrontendMode(cfg.FrontendMode, cfg.Env)
	if err != nil {
		return nil, err
	}
	switch frontendMode {
	case config.FrontendModeEmbed:
		dist, err := fs.Sub(webdist, "webdist")
		if err != nil {
			return nil, err
		}

		if _, err := fs.Stat(dist, "index.html"); err != nil {
			slog.Warn("FRONTEND_MODE=embed but no index.html is compiled into this binary; " +
				"run `yarn build` in ui/ before building the Go binary, or set FRONTEND_MODE=none")
		}
		if err := WireFrontend(router, frontendMode, cfg.Env, dist, ""); err != nil {
			return nil, err
		}
	case config.FrontendModeProxy:
		slog.Warn("FRONTEND_MODE=proxy: every unmatched path is reverse-proxied into this host's loopback interface — development only",
			"target", cfg.ViteDevURL, "env", config.NormalizeEnv(cfg.Env))
		if err := WireFrontend(router, frontendMode, cfg.Env, nil, cfg.ViteDevURL); err != nil {
			return nil, err
		}
	case config.FrontendModeNone:

	default:
		return nil, fmt.Errorf("api: unhandled frontend mode %q", frontendMode)
	}

	public := huma.NewGroup(apiServer, cfg.BasePath)
	// Registration is fallible now that the fleet lives in tenant databases.
	// Mounting a surface whose service could not be assembled would leave
	// endpoints that authenticate nobody, so refusing to start is the safe answer.
	if err := handlers.RegisterPublic(public, deps); err != nil {
		return nil, err
	}

	userProtected := huma.NewGroup(apiServer, cfg.BasePath)
	userProtected.UseMiddleware(apimiddleware.RequireUserAuth(userProtected, deps))
	if err := handlers.RegisterUserProtected(userProtected, deps); err != nil {
		return nil, err
	}

	// PRE-AUTHZ. Authenticated, but not yet authorized: this group exists so a
	// user whom policy is refusing can still reach the endpoints that let them
	// comply. Both middlewares are required — the first establishes who, the
	// second bounds the relaxation to accounts with no factor at all.
	preAuthz := huma.NewGroup(apiServer, cfg.BasePath)
	preAuthz.UseMiddleware(apimiddleware.RequireAuthenticated(preAuthz, deps))
	preAuthz.UseMiddleware(apimiddleware.RequireFirstFactorEnrolment(preAuthz, deps))
	handlers.RegisterPreAuthz(preAuthz, deps)

	// PLATFORM administration. Layered on top of RequireUserAuth because it reads
	// the principal that establishes — and it is a separate group so that the
	// role check is a property of the route rather than of each handler's first
	// three lines.
	//
	// This is the ROLE boundary. It is not the ownership boundary: a customer's
	// fleet is reached through membership on the group above, whoever you are.
	adminProtected := huma.NewGroup(apiServer, cfg.BasePath)
	adminProtected.UseMiddleware(apimiddleware.RequireUserAuth(adminProtected, deps))
	adminProtected.UseMiddleware(apimiddleware.RequireAdminAuth(adminProtected, deps))
	handlers.RegisterAdminProtected(adminProtected, deps)

	// A fourth identity, and therefore a fourth group. Fleet agents authenticate
	// with an agent-scoped key/secret pair, not a user JWT: no membership, no
	// MFA, no user behind it. Running them through RequireUserAuth would mean
	// inventing a principal that exists only to satisfy a middleware, which is
	// the sort of thing that later gets accepted somewhere it should not be.
	//
	// They ARE tenant-scoped. The key is verified in control, which yields the
	// organization, and that is what decides which tenant database the request
	// then touches — a bee has no way to name a different one.
	fleetSvc, err := service.NewFleetService(
		deps.Tenants,
		repo.NewEnrolmentRepoGorm(deps.ControlDB),
		deps.APIKeys,
		repo.NewAuthAuditRepoGorm(deps.ControlDB),
	)
	if err != nil {
		return nil, err
	}
	beeProtected := huma.NewGroup(apiServer, cfg.BasePath)
	beeProtected.UseMiddleware(apimiddleware.RequireBeeAuth(beeProtected, fleetSvc))
	if err := handlers.RegisterBeeProtected(beeProtected, deps); err != nil {
		return nil, err
	}

	return &App{
		Router:        router,
		API:           apiServer,
		PublicBaseURL: publicBase,
		BasePath:      cfg.BasePath,
	}, nil
}

func (a *App) PublicURLs() map[string]string {
	base := strings.TrimRight(a.PublicBaseURL, "/")
	apiBase := base + a.BasePath
	return map[string]string{
		"web":     base + "/",
		"docs":    base + "/docs",
		"openapi": apiBase + "/openapi.json",
	}
}

func corsOriginPolicy(raw, env, baseURL string) ([]string, func(*http.Request, string) bool) {
	denyAll := func(*http.Request, string) bool { return false }

	var entries []string
	wildcard := false
	for _, e := range strings.Split(raw, ",") {
		if e = strings.TrimSpace(e); e == "" {
			continue
		}
		if e == "*" {
			wildcard = true
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) > 0 {
		allowed := make([]string, 0, len(entries))
		for _, e := range entries {

			if o := config.OriginOf(e); o != "" {
				allowed = append(allowed, o)
				continue
			}
			slog.Error("CORS_ALLOWED_ORIGINS entry is not a scheme://host origin and was ignored", "entry", e)
		}
		if len(allowed) == 0 {
			slog.Error("CORS: every configured origin was unusable — refusing all cross-origin requests")
			return nil, denyAll
		}
		if wildcard {
			slog.Warn("CORS: \"*\" ignored because explicit origins are configured", "origins", allowed)
		}
		return allowed, nil
	}

	if config.IsDevEnv(env) {
		slog.Warn("CORS is reflecting every origin with credentials — development only", "env", config.NormalizeEnv(env))
		return nil, func(*http.Request, string) bool { return true }
	}

	if self := config.OriginOf(baseURL); self != "" {
		return []string{self}, nil
	}
	slog.Error("CORS: no allowed origins and no usable BASE_URL — refusing all cross-origin requests", "env", config.NormalizeEnv(env))
	return nil, denyAll
}

func computePublicBaseURL(publicURL, httpAddr string) string {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL != "" {
		if strings.HasPrefix(publicURL, "http://") || strings.HasPrefix(publicURL, "https://") {
			return publicURL
		}
		return "http://" + publicURL
	}

	addr := strings.TrimSpace(httpAddr)
	if addr == "" || addr == ":8080" {
		return "http://localhost:8080"
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "http://localhost:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	if strings.HasPrefix(addr, "127.0.0.1:") || strings.HasPrefix(addr, "localhost:") {
		return "http://" + addr
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	return "http://" + addr
}

const (
	scalarVersion   = "1.66.1"
	scalarIntegrity = "sha384-RkhHYpdjsrJH9sH8RmczPchxNiHEhmW300QwMB/8yg6feduTZu9FBN4W0DJnp50Z"
)

// renderScalarHTML builds the Scalar API reference page.
//
// Two Scalar features are switched off explicitly rather than left to their
// defaults:
//
//   - "agent" is Scalar's AI chat over the OpenAPI document. It defaults to
//     ENABLED on localhost, and this page is served only when IsDevEnv is true —
//     i.e. exactly where the default applies. Left alone it would offer an AI
//     panel on a page whose "try it" console holds a live bearer token for a
//     control plane that brokers tenant cloud credentials.
//   - "mcp" exposes the document to MCP-compatible tooling. It is off unless a
//     server is configured, but it is named here so that adding one later is a
//     deliberate edit to a line that already says disabled, rather than a new
//     key nobody weighed.
//
// The disabled flags are the control; connect-src 'self' on this route is the
// backstop if a future version changes these defaults.
func renderScalarHTML(specURL, serverURL string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>APIary API Docs</title>
  <style>html, body { height: 100%%; margin: 0; }</style>
</head>
<body>
  <script id="api-reference"
          data-url="%s"
          data-proxy-url=""
          data-configuration='{"servers":[{"url":"%s"}],"darkMode":true,"hideDownloadButton":false,"withDefaultFonts":false,"agent":{"disabled":true},"mcp":{"disabled":true}}'></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@%s/dist/browser/standalone.js"
          integrity="%s"
          crossorigin="anonymous"></script>
</body>
</html>`, specURL, serverURL, scalarVersion, scalarIntegrity)
}
