package fleet

import (
	"net"
	"net/url"
	"strings"

	"github.com/glueops/apiary/internal/config"
)

// agentBaseURL returns the address an agent should enrol against, or "" when
// the control plane cannot honestly name one.
//
// The agent's own default is http://localhost:8080. On a hypervisor that
// resolves to the hypervisor, so it is the one address guaranteed to be wrong,
// and it fails as a connection refused a long way from anything that mentions
// enrolment. Handing the operator a command that carries the address removes
// the guess.
//
// Two BASE_URL values are refused rather than passed on, because both would
// reproduce that failure on the far end:
//
//   - a relative one, which config permits in development, cannot be dialled at
//     all; and
//   - a loopback one, which is correct for a browser on the same machine and
//     useless to every hypervisor that is not that machine.
//
// Refusing is not a fallback to something worse — the empty string is a
// statement that the operator has to supply --base-url, which the UI can say
// out loud. Returning localhost instead would look like an answer and behave
// like the bug.
func agentBaseURL(c *config.Config) string {
	if c == nil {
		return ""
	}
	raw := strings.TrimSpace(c.BaseURL)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return ""
	}
	if isLoopbackHost(u.Hostname()) {
		return ""
	}
	return strings.TrimSuffix(raw, "/")
}

// isLoopbackHost reports whether a hostname only ever means "this machine".
//
// Both spellings and both families, since BASE_URL is written by hand: the
// literal name, anything under .localhost, and any address in 127.0.0.0/8 or
// ::1 — 127.0.0.2 is as unreachable from a hypervisor as 127.0.0.1 is.
func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return false
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
