package auth

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/glueops/apiary/internal/app"
)

func RegisterPublic(hapi huma.API, deps *app.Deps) {
	RegisterSignup(hapi, deps)
	RegisterLogin(hapi, deps)
	RegisterSocialAuth(hapi, deps)
	RegisterRefresh(hapi, deps)
	RegisterPasskeyPublic(hapi, deps)
	RegisterTOTPPublic(hapi, deps)
	RegisterPasswordReset(hapi, deps)
	RegisterEmailVerification(hapi, deps)
}

func RegisterUserProtected(hapi huma.API, deps *app.Deps) {
	RegisterLogout(hapi, deps)
	// Listing and removing factors needs a session that has already satisfied
	// policy: deleting your only factor is not remediation.
	RegisterPasskeyUserProtected(hapi, deps)
	RegisterTOTPUserProtected(hapi, deps)
}

// RegisterPreAuthz mounts ENROLMENT, which has to be reachable by someone the
// policy is currently refusing.
//
// Otherwise an organization turning on require_mfa locks out every member who
// had not enrolled: login issues them a session, that session is refused
// everywhere by the gate, and the endpoint that would let them comply is behind
// it. They hold a valid token that can do exactly one thing — nothing.
//
// Authentication still applies in full here. What is relaxed is authorization,
// and only for accounts holding no factor at all — see
// RequireFirstFactorEnrolment, which is what keeps this from becoming a way for
// a stolen password to add its own authenticator.
func RegisterPreAuthz(hapi huma.API, deps *app.Deps) {
	RegisterPasskeyProtected(hapi, deps)
	RegisterTOTPProtected(hapi, deps)
}
