package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A validation failure must not echo the submitted body back.
//
// huma's ErrorDetail.Value carries "the value at the given location", and for a
// failed REQUIRED-property check that location is the WHOLE body. Every one of
// the endpoints below therefore returned the caller's own password or provider
// secret in a 422 — the kind of response client error trackers and proxies
// retain. stripErrorValues in errors.go removes it globally; this pins that.
//
// Global rather than per-field on purpose: the next endpoint to accept a secret
// must be covered without anyone remembering to think about it.
func TestValidationErrorsNeverEchoSubmittedSecrets(t *testing.T) {
	a := testApp(t)
	for _, c := range []struct{ path, body, secret string }{
		{"/api/v1/auth/login", `{"email":"a@b.test"}`, ""},
		{"/api/v1/auth/login", `{"password":"hunter2-REAL-PASSWORD"}`, "hunter2-REAL-PASSWORD"},
		{"/api/v1/auth/register", `{"password":"REG-PASSWORD-LEAK"}`, "REG-PASSWORD-LEAK"},
		{"/api/v1/auth/reset-password", `{"new_password":"RESET-PASSWORD-LEAK"}`, "RESET-PASSWORD-LEAK"},
	} {
		req := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		a.Router.ServeHTTP(rec, req)
		if c.secret != "" && strings.Contains(rec.Body.String(), c.secret) {
			t.Errorf("%s returned the submitted secret in a %d body: %s", c.path, rec.Code, rec.Body.String())
		}
	}
}
