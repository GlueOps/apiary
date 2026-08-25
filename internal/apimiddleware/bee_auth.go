package apimiddleware

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/service"
)

type beeCtxKey struct{}

// WithBee puts an authenticated bee session on the context.
func WithBee(ctx context.Context, s *service.BeeSession) context.Context {
	return context.WithValue(ctx, beeCtxKey{}, s)
}

// BeeFromContext returns the authenticated bee session, if there is one.
//
// It carries the organization alongside the bee. A tenant row has no
// organization column — the database is the organization — so a handler holding
// only the bee could not reopen the database it came from.
func BeeFromContext(ctx context.Context) (*service.BeeSession, bool) {
	s, ok := ctx.Value(beeCtxKey{}).(*service.BeeSession)
	return s, ok && s != nil && s.Bee != nil
}

// RequireBeeAuth authenticates a fleet agent by its agent-scoped API key.
//
// Deliberately NOT the user middleware. A bee presents a key/secret pair on the
// AGENT surface, not a JWT: it has no membership, no MFA, and no user behind it.
// Running it through RequireUserAuth would mean inventing a principal for it,
// and a principal that exists only to satisfy a middleware is exactly the sort
// of thing that later gets accepted somewhere it should not be.
//
// It IS tenant-scoped, though — the key is org-scoped, and every fleet row it
// reaches lives in that organization's database. That is the whole reason the
// key is verified in control first: the organization it yields is what decides
// which database is opened next, and a bee that authenticated has by
// construction no way to name a different one.
//
// The surfaces exclude each other in both directions. An org key cannot reach
// these endpoints (a human or an automation forging an agent), and an agent key
// cannot reach the org endpoints (a compromised hypervisor turning its
// credential on the rest of the tenant's API).
func RequireBeeAuth(api huma.API, fleet *service.FleetService) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		public, secret, ok := agentKeyFrom(hctx)
		if !ok {
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "missing agent credential")
			return
		}
		if fleet == nil {
			// A middleware that cannot reach a decision must deny.
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "fleet is not available")
			return
		}

		sess, err := fleet.Authenticate(hctx.Context(), public, secret)
		if err != nil || sess == nil {
			// One answer for an unknown key, a wrong secret, a key on the wrong
			// surface, and a database fault. Anything enumerating credentials must
			// not learn which of those it hit.
			_ = huma.WriteErr(api, hctx, http.StatusUnauthorized, "invalid agent credential")
			return
		}

		next(huma.WithContext(hctx, WithBee(hctx.Context(), sess)))
	}
}
