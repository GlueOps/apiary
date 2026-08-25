package fleet

import (
	"testing"

	"github.com/glueops/apiary/internal/config"
)

// TestAgentBaseURLRefusesWhatCannotBeDialledFromAHypervisor covers the case the
// enrolment command actually failed on: the address is fine from a browser and
// meaningless from the host being enrolled.
func TestAgentBaseURLRefusesWhatCannotBeDialledFromAHypervisor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		base string
		want string
	}{
		{"an absolute address is handed over", "https://apiary.example.com", "https://apiary.example.com"},
		{"a port is kept, since the agent has to dial it", "http://10.10.10.1:8282", "http://10.10.10.1:8282"},
		{"a path prefix is kept", "https://example.com/apiary", "https://example.com/apiary"},
		{"a trailing slash is dropped so the command reads cleanly", "https://example.com/", "https://example.com"},

		{"unset means the operator must say", "", ""},
		{"whitespace is not an address", "   ", ""},
		{"relative cannot be dialled", "apiary.example.com", ""},
		{"scheme-only is not an address", "https://", ""},

		{"localhost is the bug this exists to prevent", "http://localhost:8080", ""},
		{"including with a port that is otherwise right", "http://localhost:8282", ""},
		{"and spelled as an address", "http://127.0.0.1:8282", ""},
		{"anywhere in 127.0.0.0/8, not just .1", "http://127.0.0.2:8282", ""},
		{"and in v6", "http://[::1]:8282", ""},
		{"and under the .localhost suffix", "http://apiary.localhost:8282", ""},
		{"cased differently, since BASE_URL is hand-written", "http://LocalHost:8282", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := agentBaseURL(&config.Config{BaseURL: tc.base}); got != tc.want {
				t.Fatalf("agentBaseURL(%q) = %q, want %q", tc.base, got, tc.want)
			}
		})
	}
}

// TestAgentBaseURLToleratesNoConfig guards the router tests, which build the API
// with partially-populated Deps.
func TestAgentBaseURLToleratesNoConfig(t *testing.T) {
	t.Parallel()
	if got := agentBaseURL(nil); got != "" {
		t.Fatalf("agentBaseURL(nil) = %q, want %q", got, "")
	}
}
