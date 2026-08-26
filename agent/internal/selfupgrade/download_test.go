package selfupgrade

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTheDownloadCarriesTheAgentCredential. The endpoint is authenticated
// precisely so the fleet's binary is not handed to anyone who can reach the
// login page.
func TestTheDownloadCarriesTheAgentCredential(t *testing.T) {
	t.Parallel()

	var gotKey, gotSecret, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-AGENT-Key")
		gotSecret = r.Header.Get("X-AGENT-Secret")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("binary"))
	}))
	defer srv.Close()

	body, err := Download(context.Background(), srv.Client(), srv.URL,
		"the-key", "the-secret", "linux", "amd64")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if string(body) != "binary" {
		t.Fatalf("body = %q", body)
	}
	if gotKey != "the-key" || gotSecret != "the-secret" {
		t.Fatalf("credential not sent: key=%q secret=%q", gotKey, gotSecret)
	}
	if gotPath != "/api/v1/agent/dist/linux/amd64" {
		t.Fatalf("path = %q", gotPath)
	}
}

// TestANonOKResponseIsReportedWithItsBody.
//
// The common failure is a control plane that carries no binary for this
// platform, and the answer to that question is in the body rather than the
// status.
func TestANonOKResponseIsReportedWithItsBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("this control plane carries no agent binary for linux/riscv64"))
	}))
	defer srv.Close()

	_, err := Download(context.Background(), srv.Client(), srv.URL, "k", "s", "linux", "riscv64")
	if err == nil {
		t.Fatal("a 404 was treated as a binary")
	}
	if !strings.Contains(err.Error(), "no agent binary") {
		t.Fatalf("error %q drops the server's own explanation", err)
	}
}

// TestAnOversizedResponseIsRefusedRatherThanBuffered.
//
// Without a ceiling, a wrong path answered by the SPA's HTML fallback — or a
// redirect somewhere unexpected — streams into memory until the host runs out.
// That failure is a hypervisor going down, not an upgrade that did not happen.
func TestAnOversizedResponseIsRefusedRatherThanBuffered(t *testing.T) {
	t.Parallel()

	chunk := strings.Repeat("x", 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < 4; i++ {
			_, _ = w.Write([]byte(chunk))
		}
	}))
	defer srv.Close()

	// Borrow the real cap by pointing at a smaller one for the duration.
	original := maxBinaryForTest
	maxBinaryForTest = 2 << 20
	defer func() { maxBinaryForTest = original }()

	_, err := Download(context.Background(), srv.Client(), srv.URL, "k", "s", "linux", "amd64")
	if err == nil {
		t.Fatal("an oversized response was accepted")
	}
	if !strings.Contains(err.Error(), "not an agent binary") {
		t.Fatalf("error %q does not say why it stopped", err)
	}
}
