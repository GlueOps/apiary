package apimiddleware

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func ClientIPFromContext(ctx context.Context) string {
	return chimiddleware.GetClientIP(ctx)
}

func bearerToken(hctx huma.Context) string {
	v := strings.TrimSpace(hctx.Header("Authorization"))
	if v == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(v, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(v, prefix))
}

// AgentKeyHeader and AgentSecretHeader carry the two halves of an agent
// credential.
//
// Two explicit headers, matching the shape org keys already use (X-ORG-Key /
// X-ORG-Secret) rather than HTTP Basic. Basic makes a browser that wanders onto
// the endpoint pop a login dialog, reads as a second and different auth system
// beside everything else here, and hides the credential inside a base64 blob
// that logs and request dumps show as opaque.
//
// Its own pair rather than the org one: presenting a credential on the wrong
// header means the wrong middleware never reads it at all. Scope exclusion is
// what enforces the boundary — this makes crossing it hard to do by accident.
const (
	AgentKeyHeader    = "X-AGENT-Key"
	AgentSecretHeader = "X-AGENT-Secret"
)

// agentKeyFrom pulls the key/secret pair an agent presents.
//
// Both halves are required. Returning ok=false rather than an empty pair means
// the caller cannot accidentally treat "no credential" as "empty secret" and
// carry on to a comparison.
//
// The secret is NOT trimmed. Whitespace can be part of a generated secret, and
// silently altering it turns a valid credential into a failed comparison nobody
// can explain from the outside.
func agentKeyFrom(hctx huma.Context) (public, secret string, ok bool) {
	public = strings.TrimSpace(hctx.Header(AgentKeyHeader))
	secret = hctx.Header(AgentSecretHeader)
	if public == "" || strings.TrimSpace(secret) == "" {
		return "", "", false
	}
	return public, secret, true
}
