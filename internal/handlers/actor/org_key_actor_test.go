// Package actor_test holds one cross-package assertion: every endpoint that
// records WHO acted must refuse an organization API key rather than dereference
// a user id it does not have.
//
// It lives in its own package because the endpoints it covers are spread across
// three handler packages, and the defect is a property of the set rather than of
// any one of them. A copy per package would drift, and the sixth endpoint to be
// added is the one that would be missed.
package actor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"

	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/handlers/invitation"
	"github.com/glueops/apiary/internal/handlers/org"
	"github.com/glueops/apiary/internal/handlers/policy"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/glueops/apiary/internal/workflow"
)

var schema = testsupport.SchemaName("apiary_handlers_actor_test")

// orgKeyAPI mounts the three packages carrying actor-recording endpoints behind
// a principal shaped exactly like a live organization key: authenticated, bound
// to a tenant, and carrying NO user.
//
// AssertOrgRole admits such a principal as an admin without consulting the
// membership store — deliberate, and tested in apimiddleware — so every one of
// these handlers runs with p.UserID nil.
func orgKeyAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	db := testsupport.OpenSchema(t, schema,
		&control.User{}, &control.Organization{}, &control.Membership{},
		&control.Invitation{}, &control.OrgPolicy{}, &control.PlatformPolicy{})

	p := &apimiddleware.Principal{
		Method:          apimiddleware.AuthMethodOrgSecret,
		OrganizationID:  uuid.New(),
		PolicySatisfied: true,
	}
	if err := db.Create(&control.Organization{
		ID: p.OrganizationID, Name: "org-" + p.OrganizationID.String()[:8], Status: "active",
	}).Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}

	_, api := humatest.New(t, huma.DefaultConfig("actor-probe", "1.0.0"))
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(hctx, apimiddleware.WithPrincipal(hctx.Context(), p)))
	})

	deps := &app.Deps{
		ControlDB: db,
		Jobs:      &workflow.MockEnqueuer{},
		Config:    &config.Config{},
	}
	org.RegisterUserProtected(api, deps)
	invitation.RegisterOrgProtected(api, deps)
	policy.RegisterOrgProtected(api, deps)
	return api
}

// Each of these writes an actor into a durable record: an audit row, an inviter,
// the author of a policy change. Before this was closed every one dereferenced
// *p.UserID unguarded — and a nil dereference is a panic, not a 500, so a single
// call with the wrong credential ended the process for everybody.
//
// The refusal must also name the CREDENTIAL rather than a permission.
// "Insufficient permissions" would send the holder of a fully-privileged org key
// looking for a role to grant it, and no role would help.
func TestActorRecordingEndpointsRefuseAnOrganizationKey(t *testing.T) {
	probes := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"promote a member", http.MethodPatch, "/organization/members/" + uuid.NewString(),
			map[string]any{"role": "admin"}},
		{"remove a member", http.MethodDelete, "/organization/members/" + uuid.NewString(), nil},
		{"invite someone", http.MethodPost, "/invitations",
			map[string]any{"email": "someone@example.test", "role": "member"}},
		{"revoke an invitation", http.MethodDelete, "/invitations/" + uuid.NewString(), nil},
		{"set an organization policy", http.MethodPut, "/organization/policies/require_mfa",
			map[string]any{"state": "enabled"}},
	}

	api := orgKeyAPI(t)
	for _, pr := range probes {
		t.Run(pr.name, func(t *testing.T) {
			res := call(t, api, pr.method, pr.path, pr.body)
			if res.Code != http.StatusForbidden {
				t.Fatalf("an organization key got %d, want 403; body: %s", res.Code, res.Body)
			}
			if body := res.Body.String(); !strings.Contains(body, "rather than a person") {
				t.Fatalf("the refusal does not say why an org key cannot do this: %s", body)
			}
		})
	}
}

func call(t *testing.T, api humatest.TestAPI, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	switch method {
	case http.MethodPost:
		return api.Post(path, body)
	case http.MethodPatch:
		return api.Patch(path, body)
	case http.MethodPut:
		return api.Put(path, body)
	case http.MethodDelete:
		return api.Delete(path)
	default:
		t.Fatalf("unsupported method %q", method)
		return nil
	}
}
