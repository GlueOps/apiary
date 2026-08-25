package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// docsPage fetches /docs from the real router, in a development environment
// (the route 404s anywhere else).
func docsPage(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	a := testApp(t)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	a.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /docs = %d, want 200", rec.Code)
	}
	return rec
}

var scriptSrcRe = regexp.MustCompile(`src="(https://[^"]+)"`)

// TestDocsCSPAllowsTheHostItActuallyLoadsFrom pins the CSP to the bundle.
//
// The swap from RapiDoc to Scalar moved the bundle from unpkg to jsdelivr and
// left script-src naming unpkg, so the browser blocked the docs bundle and the
// page rendered blank. Nothing failed loudly: the handler still returned 200
// with correct-looking HTML. Derive the expectation from the markup rather than
// restating the host, so changing the CDN without changing the CSP fails here.
func TestDocsCSPAllowsTheHostItActuallyLoadsFrom(t *testing.T) {
	rec := docsPage(t)
	body := rec.Body.String()
	csp := rec.Header().Get("Content-Security-Policy")

	if csp == "" {
		t.Fatal("/docs served no Content-Security-Policy")
	}

	m := scriptSrcRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no absolute script src found in the docs page")
	}
	// Reduce https://cdn.jsdelivr.net/npm/... to its origin.
	origin := m[1]
	if i := strings.Index(origin[len("https://"):], "/"); i >= 0 {
		origin = origin[:len("https://")+i]
	}

	if !strings.Contains(csp, origin) {
		t.Fatalf("the docs page loads its bundle from %s, but script-src does not allow it.\nCSP: %s",
			origin, csp)
	}
}

// TestDocsDisablesAgentAndMCP is the actual ask: Scalar's AI agent defaults to
// ENABLED on localhost, and this page is only ever served on localhost, so
// omitting the flag turns it on for every developer.
func TestDocsDisablesAgentAndMCP(t *testing.T) {
	body := docsPage(t).Body.String()

	start := strings.Index(body, "data-configuration='")
	if start < 0 {
		t.Fatal("no data-configuration attribute on the docs page")
	}
	start += len("data-configuration='")
	end := strings.Index(body[start:], "'")
	if end < 0 {
		t.Fatal("unterminated data-configuration attribute")
	}

	var cfg struct {
		Agent *struct {
			Disabled bool `json:"disabled"`
		} `json:"agent"`
		MCP *struct {
			Disabled bool `json:"disabled"`
		} `json:"mcp"`
	}
	raw := body[start : start+end]
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("data-configuration is not valid JSON (Scalar would ignore the whole "+
			"configuration, re-enabling every default): %v\n%s", err, raw)
	}

	if cfg.Agent == nil || !cfg.Agent.Disabled {
		t.Error(`Scalar's AI agent is not disabled. It defaults to enabled on localhost, ` +
			`which is the only place this page is served. Set "agent":{"disabled":true}.`)
	}
	if cfg.MCP == nil || !cfg.MCP.Disabled {
		t.Error(`Scalar's MCP integration is not explicitly disabled. Set "mcp":{"disabled":true}.`)
	}
}

// TestDocsForbidsOffOriginConnections is the backstop: even if a future Scalar
// version changed the defaults above, connect-src 'self' stops the page
// reaching an AI or MCP backend from a console that holds a live bearer token.
func TestDocsForbidsOffOriginConnections(t *testing.T) {
	csp := docsPage(t).Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self'") {
		t.Fatalf("connect-src must stay 'self' on the docs page; got: %s", csp)
	}
}
