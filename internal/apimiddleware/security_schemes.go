package apimiddleware

// Security requirement sets, declared once so that what the documentation
// advertises and what the middleware accepts cannot drift apart.
//
// Each entry in the outer slice is an ALTERNATIVE (OpenAPI reads the list as
// OR); the map inside one entry is a conjunction (AND). So the org pair below
// means "both headers together", not "either header".
//
// Referencing these from operations rather than spelling out {"BearerAuth": {}}
// is the point: a scheme declared in components but named by no operation still
// appears in Scalar's auth panel, so the docs offer a credential type every
// endpoint rejects. Someone follows them, mints a key, and gets a bare 401.
var (
	// SecurityUserOrKey is what RequireUserAuth admits: a session token, a user
	// API key, or an organization key/secret pair.
	//
	// The agent pair is deliberately absent. Scope is the surface boundary and
	// the exclusion runs both ways — a compromised hypervisor must not be able to
	// turn its credential on the rest of the tenant's API.
	SecurityUserOrKey = []map[string][]string{
		{"BearerAuth": {}},
		{"ApiKeyAuth": {}},
		{"OrgKeyAuth": {}, "OrgKeySecret": {}},
	}

	// SecurityUserOnly is for the platform surface, which is gated on a USER
	// role. An organization credential has no user behind it and can never
	// satisfy it, so offering it here would document a path that always 401s.
	SecurityUserOnly = []map[string][]string{
		{"BearerAuth": {}},
		{"ApiKeyAuth": {}},
	}

	// SecuritySessionOnly is for managing the session and its second factor:
	// logout, TOTP enrolment, passkeys.
	//
	// A long-lived key is refused here deliberately. An API key that can disable
	// MFA defeats MFA — the factor exists to make a stolen bearer insufficient,
	// and a credential that can remove it is a credential that can remove the
	// protection over itself. Proving you hold a live session is the point of
	// these endpoints, and a key does not prove that.
	SecuritySessionOnly = []map[string][]string{
		{"BearerAuth": {}},
	}

	// SecurityAgent is the hypervisor agent surface, and nothing else reaches it.
	SecurityAgent = []map[string][]string{
		{"AgentKeyAuth": {}, "AgentKeySecret": {}},
	}
)
