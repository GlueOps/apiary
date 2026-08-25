package bee

import (
	"net/url"
	"strings"

	apiary "github.com/glueops/apiary/sdk/go"
)

// NewClient builds an SDK client that presents a bee credential on every
// request.
//
// The credential goes in a default header rather than through the SDK's
// ContextAccessToken, so it cannot be forgotten at a call site: every request
// this client makes is authenticated by construction, and adding an endpoint
// later needs no thought about auth.
// NewClient authenticates with the agent-scoped key/secret PAIR.
//
// Two headers, not a bearer token. The agent surface is deliberately reachable
// only by this shape: an org key cannot be presented here and an agent key
// cannot be presented on the org endpoints, so a compromised hypervisor cannot
// turn its credential on the rest of the tenant's API. Sending it as a Bearer
// token would not merely fail, it would be indistinguishable from a user
// session at the wrong door.
func NewClient(baseURL, keyPublic, keySecret string) (*apiary.APIClient, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}

	cfg := apiary.NewConfiguration()
	// The base URL ONLY. Every path in the generated client already carries the
	// /api/v1 prefix, because the served spec does — appending it here produced
	// /api/v1/api/v1/... which the SPA fallback answers with HTML, and the SDK
	// reports that as the memorably unhelpful "undefined response type".
	cfg.Servers = apiary.ServerConfigurations{{URL: u.String()}}
	if keyPublic != "" && keySecret != "" {
		cfg.AddDefaultHeader("X-AGENT-Key", keyPublic)
		cfg.AddDefaultHeader("X-AGENT-Secret", keySecret)
	}
	return apiary.NewAPIClient(cfg), nil
}
