package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

var sensitiveQueryKeys = map[string]bool{
	"token":         true,
	"code":          true,
	"state":         true,
	"access_token":  true,
	"refresh_token": true,
	"mfa_token":     true,
	"password":      true,
	"secret":        true,
	"api_key":       true,
}

func ClientIP(r *http.Request) string {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	return hostOnly(r.RemoteAddr)
}

func hostOnly(addr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return host
}

func splitCIDRs(list string) []string {
	var out []string
	for _, raw := range strings.Split(list, ",") {
		if raw = strings.TrimSpace(raw); raw != "" {
			out = append(out, raw)
		}
	}
	return out
}

func clientIPMiddleware(trustedCIDRs string) func(http.Handler) http.Handler {
	prefixes := splitCIDRs(trustedCIDRs)
	if len(prefixes) == 0 {
		return middleware.ClientIPFromRemoteAddr
	}

	for _, p := range prefixes {
		if _, err := netip.ParsePrefix(p); err != nil {
			panic(fmt.Sprintf(
				"config: TRUSTED_PROXY_CIDRS entry %q is not a valid CIDR prefix (e.g. 10.0.0.0/8): %v",
				p, err))
		}
	}
	return middleware.ClientIPFromXFF(prefixes...)
}

func requestLogger() func(http.Handler) http.Handler {
	return middleware.RequestLogger(&scrubbingLogFormatter{})
}

type scrubbingLogFormatter struct{}

func (f *scrubbingLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &scrubbingLogEntry{
		method: r.Method,
		path:   r.URL.Path,
		query:  scrubQuery(r.URL.RawQuery),
		ip:     ClientIP(r),
		reqID:  middleware.GetReqID(r.Context()),
		ctx:    r.Context(),
	}
}

func scrubQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "<unparseable>"
	}
	for k := range values {
		if sensitiveQueryKeys[strings.ToLower(k)] {
			values.Set(k, "<redacted>")
		}
	}
	return values.Encode()
}

type scrubbingLogEntry struct {
	method string
	path   string
	query  string
	ip     string
	reqID  string
	ctx    context.Context
}

func (e *scrubbingLogEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ any) {
	slog.InfoContext(e.ctx, "http request",
		"method", e.method,
		"path", e.path,
		"query", e.query,
		"status", status,
		"bytes", bytes,
		"duration", elapsed.String(),
		"remote_ip", e.ip,
		"request_id", e.reqID,
	)
}

func (e *scrubbingLogEntry) Panic(v any, stack []byte) {
	slog.ErrorContext(e.ctx, "http request panic",
		"method", e.method,
		"path", e.path,
		"panic", v,
		"stack", string(stack),
		"request_id", e.reqID,
	)
}
