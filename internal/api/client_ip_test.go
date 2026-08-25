package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/config"
)

func TestClientIPForSecurityDecisionsIgnoresTheXRealIPHeader(t *testing.T) {
	const (
		peer  = "203.0.113.9"
		spoof = "198.51.100.7"
	)

	a, err := buildWith(t, func(cfg *config.Config) {

		cfg.TrustedProxyCIDRs = ""
	})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	var seen string
	huma.Register(huma.NewGroup(a.API, a.BasePath), huma.Operation{
		OperationID: "test-client-ip",
		Method:      http.MethodGet,
		Path:        "/client-ip",
		Summary:     "report the client address a security decision would use",
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		seen = apimiddleware.ClientIPFromContext(ctx)
		return &struct{}{}, nil
	})

	req := httptest.NewRequest(http.MethodGet, a.BasePath+"/client-ip", nil)
	req.RemoteAddr = peer + ":54321"
	req.Header.Set("X-Real-Ip", spoof)
	req.Header.Set("X-Forwarded-For", spoof)
	rec := httptest.NewRecorder()
	a.Router.ServeHTTP(rec, req)

	if rec.Code < 200 || rec.Code > 299 {
		t.Fatalf("GET %s/client-ip: status %d, want 2xx", a.BasePath, rec.Code)
	}
	switch seen {
	case spoof:
		t.Fatalf("the address used for security decisions was the caller's own header (%q); "+
			"a lockout keyed on it is reset by every request", seen)
	case "":
		t.Fatalf("no client address was available to the handler; a lockout keyed on it " +
			"collapses every attacker into one shared bucket")
	case peer:
	default:
		t.Fatalf("client address = %q, want the peer address %q", seen, peer)
	}
}
