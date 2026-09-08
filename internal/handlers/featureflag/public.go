package featureflag

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/glueops/apiary/internal/app"
)

// PublicFlagView is deliberately thinner than FlagView.
//
// Key and value only. The authenticated listing also carries the source, the
// scope, the built-in default and the description, all of which describe how
// this installation is CONFIGURED rather than what it currently offers — and an
// anonymous caller has no business knowing that a value was set deliberately
// rather than inherited, or what it would fall back to.
//
// The question this answers is "which sign-in methods can I offer", and that is
// the whole of what it should return.
type PublicFlagView struct {
	Key   string `json:"key"`
	Value bool   `json:"value"`
}

type PublicFlagsResponse struct {
	Body struct {
		Flags []PublicFlagView `json:"flags"`
	}
}

// RegisterPublicFlags mounts the one flag read that needs no credential.
//
// It exists because of a genuine ordering problem: the sign-in page must decide
// whether to render the password form BEFORE anyone has signed in, so no
// authenticated endpoint can answer it. Every other flag read requires a
// principal.
//
// What reaches this response is doubly constrained — a flag must be marked
// Public AND be platform-scoped — and both are checked in PublicFlags rather
// than here, so a second caller cannot acquire a looser rule by accident.
func RegisterPublicFlags(hapi huma.API, deps *app.Deps) {
	huma.Register(hapi, huma.Operation{
		OperationID: "flags-public",
		Method:      http.MethodGet,
		Path:        "/flags/public",
		Summary:     "The flags a sign-in page needs before anyone has signed in",
		Description: "Unauthenticated, and only the flags explicitly marked public — which today " +
			"is which sign-in methods this installation offers.\n\n" +
			"NOT A VIEW OF THE FLAG SYSTEM. It carries a key and a value and nothing else: no " +
			"source, no scope, no built-in default. Those describe how the installation is " +
			"configured rather than what it offers, and an anonymous caller has no business " +
			"knowing whether a value was set deliberately or inherited.\n\n" +
			"Organization-scoped flags never appear here at any value. They resolve differently " +
			"per tenant and an anonymous caller has no tenant, so publishing one would either " +
			"leak another organization's configuration or answer with a value true for nobody.",
		Tags: []string{"Auth"},
		// No Security field: this is the one endpoint in the package that is
		// meant to be reachable by anybody, and saying so by omission would be
		// indistinguishable from forgetting.
		Security: []map[string][]string{},
	}, func(ctx context.Context, _ *struct{}) (*PublicFlagsResponse, error) {
		eff, err := deps.Flags.ListPublicEffective(ctx)
		if err != nil {
			// The sign-in page cannot render without an answer, so this must not
			// be a 500 that leaves a blank screen. A read failure falls back to
			// the compiled defaults inside the service, so reaching here means
			// the store itself errored, which is worth reporting honestly.
			return nil, huma.Error503ServiceUnavailable("could not read the public flags", err)
		}
		out := &PublicFlagsResponse{}
		out.Body.Flags = make([]PublicFlagView, 0, len(eff))
		for _, e := range eff {
			out.Body.Flags = append(out.Body.Flags, PublicFlagView{Key: e.Key, Value: e.Value})
		}
		return out, nil
	})
}
