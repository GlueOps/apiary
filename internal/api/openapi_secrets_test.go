package api

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// secretShapedProperty is deliberately broad. It is a tripwire, not a
// classifier: the cost of a false positive is one line added to the allow-list
// below after somebody argues for it, and the cost of a false negative is a
// tenant's cloud credential in a JSON response.
var secretShapedProperty = regexp.MustCompile(`(?i)secret|password|passwd|private_?key|api_?key|token|plaintext|material`)

// allowedSecretShapedProperties is the complete, argued list of response
// properties that match the tripwire and are nonetheless correct.
//
// Every entry is an authentication token being handed to the client that just
// authenticated, or the TOTP shared secret at the one moment enrolment requires
// it. All of them belong to the CALLER's own session. None of them is another
// party's material, and none is a stored credential being read back out.
//
// Adding an entry here must be a deliberate edit with a reason, which is the
// whole mechanism: an accidental new field cannot be in this map.
var allowedSecretShapedProperties = map[string]bool{
	// NOT SECRET MATERIAL — a COUNT of how many of its old cluster's secret files
	// a departed machine still reports holding.
	//
	// It exists precisely so that exposure is visible: a host that left a cluster
	// and still carries its secrets authenticates against a fleet it is no longer
	// part of, and the number is how an operator sees that without querying the
	// database. It carries no contents and no digests — a digest would be a stable
	// identifier for the very bytes the disarm exists to destroy, which is why the
	// agent reports a count instead.
	//
	// It trips the tripwire on the word "secrets" in the name, which is the name
	// being accurate rather than the value being sensitive.
	"DisarmView.held_secrets": true,

	// The login/refresh/registration family: the caller's own session tokens,
	// returned to the caller who just proved who they are.
	"LoginResponseBody.access_token":  true,
	"LoginResponseBody.refresh_token": true,
	"LoginResponseBody.token_type":    true,
	// mfa_token is the short-lived handle for the second factor step. It is not
	// an access token and grants nothing on its own.
	"LoginResponseBody.mfa_token": true,

	// Fleet enrolment. A token minted for an operator to paste onto a host that
	// is being brought up, returned once at the moment it is created and stored
	// only as a hash.
	//
	// It is safe for the reason the whole category is safe: it is the RESULT of a
	// privileged action the caller just performed (platform-admin, impersonation
	// refused), not a stored credential being read back. There is deliberately no
	// endpoint that reissues or reveals it — a bee that loses its credential
	// enrols again, because an endpoint that handed one back would let anyone
	// holding a bee id take that host over.
	"MintTokenResponseBody.token": true,

	// The other half of enrolment: the agent-scoped key/secret pair minted for a
	// host that just redeemed a valid token.
	//
	// Safe for the same reason, and only for that reason. It is the RESULT of an
	// action the caller just completed by presenting a credential, returned once
	// at the moment of creation and stored only as a hash — not a stored secret
	// being read back out. No endpoint reissues or reveals it, deliberately: one
	// that did would let anyone holding a bee id take that hypervisor over. A
	// host that loses its secret enrols again.
	//
	// The public half is not secret material at all — it is the identifier the
	// secret is checked against — but it is listed because the sweep matches on
	// shape rather than on which half of a pair a name refers to.
	"EnrolResponseBody.secret": true,

	// The secret half of an org-scoped API key the caller just minted for their
	// own organization, on POST /api-keys.
	//
	// Safe for the same reason enrolment's is, and only for that reason: it is
	// the RESULT of a privileged action the caller just performed (admin role,
	// impersonation refused, and an org credential may not mint another one),
	// returned once at the moment of creation and stored only as a hash. It is
	// not a stored credential being read back — GET /api-keys returns nothing
	// but metadata and an 8-character prefix, and there is deliberately no
	// endpoint that reissues or reveals a key, because one that did would make
	// read access to the list equivalent to holding every credential in it. A
	// key that is lost is revoked and replaced.
	//
	// The public half is not in this list because it does not match the pattern,
	// not because it is exempt: it ships in the same one-time response, under
	// the same argument.
	"MintAPIKeyResponseBody.secret": true,

	// Impersonation start. This one is a token for a session whose SUBJECT is
	// somebody else, so it deserves its own argument rather than sitting under
	// the "caller's own session" heading above.
	//
	// It is safe for the same reason login's is: it is the RESULT of the
	// authorization that just happened, not a stored credential being read
	// back. Reaching it requires a platform-staff role on a non-impersonated
	// session AND the target's own consent (support access enabled, or an
	// active grant) — the service refuses otherwise. The token carries the
	// actor id, so everything done with it is attributed to the staff member as
	// well as the target, and there is no way to obtain one for a user who has
	// not consented.
	"ImpersonateStartResponseBody.access_token": true,
	"ImpersonateStartResponseBody.token_type":   true,

	"MFAVerifyResponseBody.access_token":  true,
	"MFAVerifyResponseBody.refresh_token": true,
	"MFAVerifyResponseBody.token_type":    true,

	"PasskeyFinishLoginResponseBody.access_token":  true,
	"PasskeyFinishLoginResponseBody.refresh_token": true,
	"PasskeyFinishLoginResponseBody.token_type":    true,

	"RefreshResponseBody.access_token":  true,
	"RefreshResponseBody.refresh_token": true,
	"RefreshResponseBody.token_type":    true,

	"RegisterResponseBody.access_token":  true,
	"RegisterResponseBody.refresh_token": true,
	"RegisterResponseBody.token_type":    true,

	// The TOTP shared secret, returned exactly once during enrolment because the
	// user has to type it into an authenticator. There is no way to enrol
	// without disclosing it to the enrolling user.
	"TOTPSetupResponseBody.secret": true,
}

// refName returns the component schema a node points at, if it is a $ref.
func refName(node map[string]any) string {
	ref, _ := node["$ref"].(string)
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

// collectRefs gathers every component schema named anywhere under node.
func collectRefs(node any, out map[string]bool) {
	switch n := node.(type) {
	case map[string]any:
		if name := refName(n); name != "" {
			out[name] = true
		}
		for k, v := range n {
			if k == "$ref" {
				continue
			}
			collectRefs(v, out)
		}
	case []any:
		for _, v := range n {
			collectRefs(v, out)
		}
	}
}

// collectProperties walks a schema body and records every property name it
// declares, including names nested inside inline object and array schemas.
// It does NOT follow $ref — reachability is resolved separately so that each
// component schema is reported under its own name rather than under whichever
// response happened to reach it first.
func collectProperties(node any, prefix string, out map[string]bool) {
	n, ok := node.(map[string]any)
	if !ok {
		return
	}
	if props, ok := n["properties"].(map[string]any); ok {
		for name, sub := range props {
			out[prefix+name] = true
			collectProperties(sub, prefix+name+".", out)
		}
	}
	for _, k := range []string{"items", "additionalProperties"} {
		collectProperties(n[k], prefix, out)
	}
	for _, k := range []string{"allOf", "anyOf", "oneOf"} {
		if list, ok := n[k].([]any); ok {
			for _, sub := range list {
				collectProperties(sub, prefix, out)
			}
		}
	}
}

// TestNoResponseSchemaCarriesSecretMaterial sweeps every schema reachable from
// any response in the SERVED spec.
//
// Stated plainly: this is an ADDITION guard, not enforcement of an existing
// line. Deleting nothing in the handlers turns it red. It exists for the change
// it is meant to catch — a Secret, Identity-secret or plaintext field added to
// EstateView and copied through toView, which no compiler, linter or handler
// test would object to, and which would publish a tenant's cloud credential to
// every member of that organization and into the generated TypeScript client.
//
// It sweeps the whole document rather than just the estate schemas because the
// same mistake anywhere in a control plane that brokers cloud keys has the same
// consequence, and because sweeping is no more expensive than not.
func TestNoResponseSchemaCarriesSecretMaterial(t *testing.T) {
	a := testApp(t)
	doc := servedSpec(t, a)

	components, _ := doc["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatal("the served spec declares no paths: this sweep would pass vacuously")
	}

	// Roots: every schema mentioned by any response of any operation. Request
	// bodies are deliberately excluded — a secret going IN is the point of
	// estates-create; a secret coming OUT never is.
	roots := map[string]bool{}
	inline := map[string]any{}
	responses := 0
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		for method, rawOp := range item {
			op, ok := rawOp.(map[string]any)
			if !ok {
				continue
			}
			resps, _ := op["responses"].(map[string]any)
			for code, rawResp := range resps {
				resp, ok := rawResp.(map[string]any)
				if !ok {
					continue
				}
				content, _ := resp["content"].(map[string]any)
				for _, rawMedia := range content {
					media, ok := rawMedia.(map[string]any)
					if !ok {
						continue
					}
					schema, ok := media["schema"].(map[string]any)
					if !ok {
						continue
					}
					responses++
					collectRefs(schema, roots)
					if refName(schema) == "" {
						// An anonymous response body still ships to clients, so
						// it is checked under a name that locates it.
						inline[strings.ToUpper(method)+" "+path+" "+code] = schema
					}
				}
			}
		}
	}
	if responses == 0 {
		t.Fatal("no response declares a schema: this sweep would pass vacuously")
	}

	// Transitive closure. The cycle guard is `reached`: a schema that references
	// itself (or a pair that reference each other) must not spin here.
	reached := map[string]bool{}
	frontier := roots
	for len(frontier) > 0 {
		next := map[string]bool{}
		for name := range frontier {
			if reached[name] {
				continue
			}
			reached[name] = true
			body, ok := schemas[name].(map[string]any)
			if !ok {
				continue
			}
			found := map[string]bool{}
			collectRefs(body, found)
			for n := range found {
				if !reached[n] {
					next[n] = true
				}
			}
		}
		frontier = next
	}
	if len(reached) == 0 {
		t.Fatal("no component schema is reachable from any response: this sweep would pass vacuously")
	}

	check := func(owner string, body any) {
		props := map[string]bool{}
		collectProperties(body, "", props)

		names := make([]string, 0, len(props))
		for n := range props {
			names = append(names, n)
		}
		sort.Strings(names)

		for _, name := range names {
			if !secretShapedProperty.MatchString(name) {
				continue
			}
			if allowedSecretShapedProperties[owner+"."+name] {
				continue
			}
			t.Errorf("response schema %s declares property %q, which looks like secret material.\n"+
				"No endpoint in this control plane may return a stored credential: the fingerprint is what an\n"+
				"operator compares. If this really is safe, add %q to allowedSecretShapedProperties with the\n"+
				"reason it is safe — do not widen the pattern.", owner, name, owner+"."+name)
		}
	}

	for _, name := range sortedKeys(reached) {
		check(name, schemas[name])
	}
	for _, name := range sortedKeys(boolKeys(inline)) {
		check(name, inline[name])
	}

	t.Logf("swept %d response schemas reachable from %d responses", len(reached), responses)
}

func boolKeys(m map[string]any) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}
