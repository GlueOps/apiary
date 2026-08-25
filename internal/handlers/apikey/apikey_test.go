package apikey

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/glueops/apiary/internal/apimiddleware"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/testsupport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A schema dedicated to this suite. Other packages migrate and TRUNCATE their
// own fixtures in the same test database, and api_keys is a table more than one
// of them touches — sharing would have this suite failing for reasons that say
// nothing about scope.
var keysSchema = testsupport.SchemaName("apiary_apikey_handler")

type harness struct {
	api  humatest.TestAPI
	ctl  *gorm.DB
	repo repo.APIKeyRepo

	orgID   uuid.UUID
	userID  uuid.UUID
	otherID uuid.UUID // a second member of the SAME organization

	otherOrgID uuid.UUID // a second organization the caller is not in
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctl := testsupport.OpenSchema(t, keysSchema,
		&control.User{}, &control.Organization{}, &control.Membership{}, &control.APIKey{})

	// Rows, not tables: api_keys carries a unique index on key_hash, so a row
	// left by an earlier test would make the next one fail on an insert conflict
	// rather than on the thing it asserts.
	if err := ctl.Exec(`TRUNCATE api_keys, memberships, organizations, users RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate control tables: %v", err)
	}

	newOrg := func(name string) uuid.UUID {
		o := control.Organization{Name: name + "-" + uuid.NewString(), Status: "active"}
		if err := ctl.Create(&o).Error; err != nil {
			t.Fatalf("create org: %v", err)
		}
		return o.ID
	}
	newUser := func() uuid.UUID {
		u := control.User{
			Email:     "keys-" + uuid.NewString() + "@example.test",
			Password:  "x",
			FirstName: "Key",
			LastName:  "Tester",
			Role:      control.RolePlatformCustomer,
			IsActive:  true,
		}
		if err := ctl.Create(&u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		return u.ID
	}

	h := &harness{
		ctl:        ctl,
		repo:       repo.NewAPIKeyRepoGorm(ctl),
		orgID:      newOrg("keys"),
		otherOrgID: newOrg("keys-other"),
		userID:     newUser(),
		otherID:    newUser(),
	}

	deps := &app.Deps{
		ControlDB: ctl,
		APIKeys:   service.NewAPIKeyService(h.repo),
	}
	_, hapi := humatest.New(t)
	Register(hapi, deps)
	h.api = hapi
	return h
}

// grant seeds a membership. AssertOrgRole reads the organization from the
// PRINCIPAL, so this row is what every authorisation decision resolves against;
// no request parameter can select a different one.
func (h *harness) grant(t *testing.T, userID, orgID uuid.UUID, role control.MemberRole) {
	t.Helper()
	m := control.Membership{UserID: userID, OrganizationID: orgID, Role: role}
	if err := h.ctl.Create(&m).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
}

func (h *harness) setRole(t *testing.T, userID, orgID uuid.UUID, role control.MemberRole) {
	t.Helper()
	err := h.ctl.Model(&control.Membership{}).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Update("role", role).Error
	if err != nil {
		t.Fatalf("set role: %v", err)
	}
}

// user is a signed-in session principal for the given user in h.orgID.
func (h *harness) user(id uuid.UUID) *apimiddleware.Principal {
	uid := id
	return &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodBearer,
		UserID:         &uid,
		OrganizationID: h.orgID,
		SessionJTI:     uuid.NewString(),
	}
}

// impersonated is a principal that would otherwise be fully authorised. Both
// fields are required: Principal.IsImpersonating() needs Impersonating AND a
// non-nil ActorID, so setting only the flag would silently not be impersonation
// and the test would pass while asserting nothing.
func (h *harness) impersonated(id uuid.UUID) *apimiddleware.Principal {
	p := h.user(id)
	actor := uuid.New()
	p.ActorID = &actor
	p.Impersonating = true
	return p
}

// orgKey is what an organization key/secret pair presents: an organization, and
// deliberately no user behind it.
func (h *harness) orgKey() *apimiddleware.Principal {
	return &apimiddleware.Principal{
		Method:         apimiddleware.AuthMethodOrgSecret,
		OrganizationID: h.orgID,
	}
}

func (h *harness) do(t *testing.T, p *apimiddleware.Principal, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	ctx := apimiddleware.WithPrincipal(t.Context(), p)
	if body == nil {
		return h.api.DoCtx(ctx, method, path)
	}
	return h.api.DoCtx(ctx, method, path, body)
}

// seed writes a key straight into the table, so a test can assert about keys the
// endpoint itself refuses to create — an agent key, or another user's key.
func (h *harness) seed(t *testing.T, k control.APIKey) uuid.UUID {
	t.Helper()
	k.KeyHash = uuid.NewString()
	if k.RequiresSecret() {
		s := uuid.NewString()
		k.SecretHash = &s
	}
	if err := h.ctl.Create(&k).Error; err != nil {
		t.Fatalf("seed key: %v", err)
	}
	return k.ID
}

func detailOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return rec.Body.String()
	}
	return body.Detail
}

func mintBody(name, scope string) map[string]any {
	return map[string]any{"name": name, "scope": scope}
}

// TestAgentScopeIsRefusedOnThisSurface is the central rule.
//
// Agent keys are standing access to one tenant's fleet loop, and they are
// issued ONLY by enrolment, against a token an operator pasted onto the host
// being brought up. If this endpoint could mint one, "I can call the API"
// would become "I can present myself as any hypervisor in this organization" —
// and Authenticate would admit it, because scope is exactly what it checks.
//
// Run at every role up to owner, because the refusal is about the SURFACE
// rather than about privilege: there is no role that makes it acceptable, so a
// version of this check that only held for members would be worse than none.
func TestAgentScopeIsRefusedOnThisSurface(t *testing.T) {
	for _, role := range []control.MemberRole{
		control.MemberRoleMember, control.MemberRoleAdmin, control.MemberRoleOwner,
	} {
		t.Run(string(role), func(t *testing.T) {
			h := newHarness(t)
			h.grant(t, h.userID, h.orgID, role)

			rec := h.do(t, h.user(h.userID), http.MethodPost, "/api-keys",
				mintBody("a-bee", control.APIKeyScopeAgent))

			if rec.Code != http.StatusForbidden {
				t.Fatalf("minting an agent key as %s answered %d, want 403: %s",
					role, rec.Code, rec.Body.String())
			}
			// The message has to say WHY, not just no. An operator whose
			// deployment needs an agent credential has to be sent to enrolment,
			// not left thinking they have a permissions problem.
			if d := detailOf(t, rec); !strings.Contains(strings.ToLower(d), "enrol") {
				t.Errorf("refusal does not point at enrolment, so an operator cannot act on it: %q", d)
			}

			var n int64
			h.ctl.Model(&control.APIKey{}).Where("scope = ?", control.APIKeyScopeAgent).Count(&n)
			if n != 0 {
				t.Fatalf("an agent key was written despite the 403: the refusal is cosmetic")
			}
		})
	}
}

func TestUnknownScopeIsRefusedAsUnprocessable(t *testing.T) {
	h := newHarness(t)
	h.grant(t, h.userID, h.orgID, control.MemberRoleOwner)

	for _, scope := range []string{"", "admin", "platform", "USER "} {
		t.Run("scope="+scope, func(t *testing.T) {
			rec := h.do(t, h.user(h.userID), http.MethodPost, "/api-keys", mintBody("x", scope))
			// "USER " is deliberately in this table only to prove the trim-and-
			// lower normalisation exists: it must SUCCEED, not 422.
			want := http.StatusUnprocessableEntity
			if strings.EqualFold(strings.TrimSpace(scope), control.APIKeyScopeUser) {
				want = http.StatusOK
			}
			if rec.Code != want {
				t.Fatalf("scope %q answered %d, want %d: %s", scope, rec.Code, want, rec.Body.String())
			}
			if rec.Code >= 500 {
				t.Fatalf("a rejected scope is a decision, not a server fault")
			}
		})
	}
}

// TestOrgScopeNeedsAdminAndRefusesImpersonation covers the second rule: an org
// key is standing access to the whole tenant, so it needs a real admin.
func TestOrgScopeNeedsAdminAndRefusesImpersonation(t *testing.T) {
	t.Run("plain member is refused", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleMember)

		rec := h.do(t, h.user(h.userID), http.MethodPost, "/api-keys",
			mintBody("ci", control.APIKeyScopeOrg))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("answered %d, want 403: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("impersonated admin is refused", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleAdmin)

		rec := h.do(t, h.impersonated(h.userID), http.MethodPost, "/api-keys",
			mintBody("ci", control.APIKeyScopeOrg))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("answered %d, want 403: an operator acting as a user must not be able to "+
				"leave themselves standing access to the tenant: %s", rec.Code, rec.Body.String())
		}

		var n int64
		h.ctl.Model(&control.APIKey{}).Count(&n)
		if n != 0 {
			t.Fatalf("a key was minted by an impersonated session")
		}
	})

	t.Run("admin succeeds and gets both halves once", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleAdmin)

		rec := h.do(t, h.user(h.userID), http.MethodPost, "/api-keys",
			mintBody("ci", control.APIKeyScopeOrg))
		if rec.Code != http.StatusOK {
			t.Fatalf("answered %d, want 200: %s", rec.Code, rec.Body.String())
		}

		var got struct {
			Credential APIKeyView `json:"credential"`
			Key        string     `json:"key"`
			Secret     string     `json:"secret"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Key == "" || got.Secret == "" {
			t.Fatalf("an org key is a key/secret pair; the mint response gave key=%q secret=%q",
				got.Key, got.Secret)
		}
		if got.Credential.Scope != control.APIKeyScopeOrg {
			t.Errorf("minted scope %q, want org", got.Credential.Scope)
		}

		// The plaintext exists here and nowhere else. If the list can reproduce
		// it, the store is holding recoverable credentials.
		list := h.do(t, h.user(h.userID), http.MethodGet, "/api-keys", nil)
		if strings.Contains(list.Body.String(), got.Secret) || strings.Contains(list.Body.String(), got.Key) {
			t.Fatalf("the list reproduced minted key material; only hashes may be stored:\n%s",
				list.Body.String())
		}
	})
}

// TestUserScopeNeedsOnlyTheCallerThemselves is the other half of the mint rule:
// a user key acts as the caller and can do nothing the caller cannot, so no
// elevated role is involved.
func TestUserScopeNeedsOnlyTheCallerThemselves(t *testing.T) {
	h := newHarness(t)
	h.grant(t, h.userID, h.orgID, control.MemberRoleMember)

	rec := h.do(t, h.user(h.userID), http.MethodPost, "/api-keys",
		mintBody("laptop", control.APIKeyScopeUser))
	if rec.Code != http.StatusOK {
		t.Fatalf("answered %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var got struct {
		Credential APIKeyView `json:"credential"`
		Key        string     `json:"key"`
		Secret     string     `json:"secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Key == "" {
		t.Fatal("no credential was returned, so the caller has nothing to use")
	}
	if got.Secret != "" {
		t.Errorf("a user key is a single opaque value; returning a secret half means "+
			"Authenticate will reject the pair the caller was just handed: %q", got.Secret)
	}

	var stored control.APIKey
	if err := h.ctl.First(&stored, "id = ?", got.Credential.ID).Error; err != nil {
		t.Fatalf("load minted key: %v", err)
	}
	if stored.UserID == nil || *stored.UserID != h.userID {
		t.Errorf("user key is not bound to its user: %+v", stored.UserID)
	}
	// Not bound to an organization on purpose: the organization arrives in
	// X-Org-Id and is checked against membership on every call, so the key
	// follows the person's access rather than freezing a copy of it.
	if stored.OrgID != nil {
		t.Errorf("user key was bound to organization %s; membership is what should decide "+
			"where it works, and a frozen binding outlives the membership", stored.OrgID)
	}
}

// TestOrgCredentialCannotMintKeys covers the two refusals that only an
// organization key/secret pair can reach. AssertOrgRole ranks such a credential
// as admin, so without these it would sail through the gate.
func TestOrgCredentialCannotMintKeys(t *testing.T) {
	t.Run("cannot mint a user key: there is no user behind it", func(t *testing.T) {
		h := newHarness(t)
		rec := h.do(t, h.orgKey(), http.MethodPost, "/api-keys",
			mintBody("x", control.APIKeyScopeUser))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("answered %d, want 403: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("cannot mint another org key: revocation would stop being a remedy", func(t *testing.T) {
		h := newHarness(t)
		rec := h.do(t, h.orgKey(), http.MethodPost, "/api-keys",
			mintBody("x", control.APIKeyScopeOrg))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("answered %d, want 403: a leaked key that can issue its own replacement "+
				"survives being revoked: %s", rec.Code, rec.Body.String())
		}
		var n int64
		h.ctl.Model(&control.APIKey{}).Count(&n)
		if n != 0 {
			t.Fatalf("an org credential minted a successor")
		}
	})
}

// TestListIsScopedToTheCallerAndTheirOrganization asserts the read boundary:
// the caller's own user keys and the organization's org keys, and nothing else.
func TestListIsScopedToTheCallerAndTheirOrganization(t *testing.T) {
	h := newHarness(t)
	h.grant(t, h.userID, h.orgID, control.MemberRoleMember)
	h.grant(t, h.otherID, h.orgID, control.MemberRoleMember)

	mine := h.seed(t, control.APIKey{UserID: &h.userID, Scope: control.APIKeyScopeUser, Name: "mine"})
	ours := h.seed(t, control.APIKey{OrgID: &h.orgID, Scope: control.APIKeyScopeOrg, Name: "ours"})
	theirs := h.seed(t, control.APIKey{UserID: &h.otherID, Scope: control.APIKeyScopeUser, Name: "theirs"})
	bee := h.seed(t, control.APIKey{OrgID: &h.orgID, Scope: control.APIKeyScopeAgent, Name: "bee"})
	elsewhere := h.seed(t, control.APIKey{OrgID: &h.otherOrgID, Scope: control.APIKeyScopeOrg, Name: "elsewhere"})

	rec := h.do(t, h.user(h.userID), http.MethodGet, "/api-keys", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("answered %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Keys []APIKeyView `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	seen := map[string]bool{}
	for _, k := range got.Keys {
		seen[k.ID] = true
	}
	for _, want := range []uuid.UUID{mine, ours} {
		if !seen[want.String()] {
			t.Errorf("key %s is missing from the list", want)
		}
	}
	for name, forbidden := range map[string]uuid.UUID{
		"another member's personal key": theirs,
		// An agent key listed here invites an operator to revoke it from a list
		// labelled "organization keys", unaware they have just cut a hypervisor
		// out of the control plane. Agent keys are managed where they are issued.
		"an agent key":             bee,
		"another tenant's org key": elsewhere,
	} {
		if seen[forbidden.String()] {
			t.Errorf("the list disclosed %s (%s)", name, forbidden)
		}
	}
}

// TestRevocationIsScoped is the third rule. Nothing outside the caller's own two
// sets may be revoked, and an organization's key needs the same authority its
// creation did.
func TestRevocationIsScoped(t *testing.T) {
	t.Run("own user key", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleMember)
		id := h.seed(t, control.APIKey{UserID: &h.userID, Scope: control.APIKeyScopeUser, Name: "mine"})

		rec := h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("answered %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var k control.APIKey
		if err := h.ctl.First(&k, "id = ?", id).Error; err != nil {
			t.Fatalf("reload: %v", err)
		}
		if !k.Revoked {
			t.Fatal("the endpoint answered 200 but the key is still usable")
		}
	})

	t.Run("another member's user key is not even visible", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleOwner)
		h.grant(t, h.otherID, h.orgID, control.MemberRoleMember)
		id := h.seed(t, control.APIKey{UserID: &h.otherID, Scope: control.APIKeyScopeUser, Name: "theirs"})

		rec := h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		// 404 rather than 403, and identical to the id that does not exist:
		// telling an owner "that key is real, but not yours" makes the endpoint
		// an oracle over key ids.
		if rec.Code != http.StatusNotFound {
			t.Fatalf("an owner revoked another member's personal key: answered %d: %s",
				rec.Code, rec.Body.String())
		}
		var k control.APIKey
		if err := h.ctl.First(&k, "id = ?", id).Error; err != nil {
			t.Fatalf("reload: %v", err)
		}
		if k.Revoked {
			t.Fatal("the key was revoked despite the 404")
		}
	})

	t.Run("an agent key is not revocable here", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleOwner)
		id := h.seed(t, control.APIKey{OrgID: &h.orgID, Scope: control.APIKeyScopeAgent, Name: "bee"})

		rec := h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("answered %d, want 404: an agent credential cut from this surface is a "+
				"hypervisor dropped out of the control plane by someone who thought they were "+
				"tidying an API key list: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("another tenant's org key", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleOwner)
		id := h.seed(t, control.APIKey{OrgID: &h.otherOrgID, Scope: control.APIKeyScopeOrg, Name: "elsewhere"})

		rec := h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("answered %d, want 404: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("org key needs admin, and refuses impersonation", func(t *testing.T) {
		h := newHarness(t)
		h.grant(t, h.userID, h.orgID, control.MemberRoleMember)
		id := h.seed(t, control.APIKey{OrgID: &h.orgID, Scope: control.APIKeyScopeOrg, Name: "ours"})

		rec := h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("a plain member revoked the organization's key: answered %d: %s",
				rec.Code, rec.Body.String())
		}

		h.setRole(t, h.userID, h.orgID, control.MemberRoleAdmin)
		rec = h.do(t, h.impersonated(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("an impersonated admin revoked the organization's key: answered %d: %s",
				rec.Code, rec.Body.String())
		}

		rec = h.do(t, h.user(h.userID), http.MethodDelete, "/api-keys/"+id.String(), nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("a real admin could not revoke the organization's key: answered %d: %s",
				rec.Code, rec.Body.String())
		}
		var k control.APIKey
		if err := h.ctl.First(&k, "id = ?", id).Error; err != nil {
			t.Fatalf("reload: %v", err)
		}
		if !k.Revoked {
			t.Fatal("the endpoint answered 200 but the org key is still usable")
		}
	})
}

// TestNoMembershipReachesNothing is the outer gate: every operation here is
// membership-gated, so a caller with no membership in the organization they
// named must not even be able to list.
func TestNoMembershipReachesNothing(t *testing.T) {
	h := newHarness(t)
	id := h.seed(t, control.APIKey{OrgID: &h.orgID, Scope: control.APIKeyScopeOrg, Name: "ours"})

	cases := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api-keys", nil},
		{http.MethodPost, "/api-keys", mintBody("x", control.APIKeyScopeUser)},
		{http.MethodDelete, "/api-keys/" + id.String(), nil},
	}
	for _, tc := range cases {
		rec := h.do(t, h.user(h.userID), tc.method, tc.path, tc.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s answered %d for a non-member, want 403: %s",
				tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}
