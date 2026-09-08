package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/glueops/apiary/internal/models/control"
)

// THE TENANT SURFACE IS READ ONLY, AND THE PROOF IS THAT NO WRITE ROUTE EXISTS.
//
// Not a write route that checks a role and refuses — none at all. A route that
// does not exist cannot be reached through a hole in a role check, and there is
// no flag a tenant is currently permitted to set. If that ever changes, it
// should change by someone deliberately adding a route and this test failing,
// rather than by a permission check quietly widening.
//
// Driven through the real router, because the property being asserted is which
// GROUP each endpoint is registered on — the one thing a unit test calling the
// handler closure directly cannot observe, and the most consequential mistake
// available in this tree.
func TestATenantCannotWriteAFeatureFlag(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Flag Customer")
	// An OWNER, the top of the membership ladder, to make the point that the
	// membership ladder buys nothing on a platform boundary.
	owner := hhtSeedUser(t, db, hhtEmail("flag-owner"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, owner.ID, org.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, owner.ID, org.ID)

	// Reading is allowed: a tenant may see what applies to it.
	rec := hhtReq(t, a, http.MethodGet, "/flags", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags answered %d for a member of the organization, want 200: %s",
			rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "legacy_login") {
		t.Fatalf("the listing omitted a declared flag; a flag missing from the list is "+
			"indistinguishable from one that is off: %s", rec.Body.String())
	}

	// Writing is not, at either level, by any route.
	for _, c := range []struct{ method, path string }{
		{http.MethodPut, "/flags/legacy_login"},
		{http.MethodPut, "/platform/flags/legacy_login"},
		{http.MethodDelete, "/platform/flags/legacy_login"},
		{http.MethodPut, "/platform/organizations/" + org.ID.String() + "/flags/proxmox.provisioning"},
		{http.MethodDelete, "/platform/organizations/" + org.ID.String() + "/flags/proxmox.provisioning"},
	} {
		rec := hhtReq(t, a, c.method, c.path, token, map[string]any{"value": true})
		if rec.Code < 400 {
			t.Fatalf("%s %s answered %d for an ordinary customer; a tenant that can set its own "+
				"flags can switch legacy authentication back on for itself: %s",
				c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

// The platform surface refuses a customer even though the route exists, and the
// refusal must not distinguish "not an admin" from "no such flag" — that would
// let any authenticated caller enumerate the registry.
func TestThePlatformFlagSurfaceRefusesACustomerWithoutLeakingTheRegistry(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	org := hhtSeedOrg(t, db, "Flag Probe Org")
	customer := hhtSeedUser(t, db, hhtEmail("flag-probe"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, customer.ID, org.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, customer.ID, org.ID)

	real := hhtReq(t, a, http.MethodPut, "/platform/flags/legacy_login", token,
		map[string]any{"value": false})
	fake := hhtReq(t, a, http.MethodPut, "/platform/flags/no_such_flag_at_all", token,
		map[string]any{"value": false})

	if real.Code != fake.Code {
		t.Fatalf("a declared flag answered %d and an undeclared one %d; the difference "+
			"enumerates the registry for any authenticated caller", real.Code, fake.Code)
	}
	if real.Code < 400 {
		t.Fatalf("an ordinary customer set an installation-wide flag: %s", real.Body.String())
	}
}

// A CUSTOMER MUST NOT BE ABLE TO ENUMERATE THE OTHER TENANTS.
//
// The directory answers "who else is on this installation", which is the first
// thing anyone with a foothold in one organization would want and is not any
// customer's to see. It is mounted on the admin group, so this is really
// asserting WHICH GROUP the route was registered on — the one property a unit
// test on the handler closure cannot observe, and the mistake that would not
// show up until somebody read the wrong list.
func TestACustomerCannotListTheOtherOrganizations(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	// A second organization, so there is genuinely something to leak.
	hhtSeedOrg(t, db, "Someone Else Entirely")

	mine := hhtSeedOrg(t, db, "My Own Org")
	owner := hhtSeedUser(t, db, hhtEmail("dir-owner"), control.RolePlatformCustomer)
	hhtSeedMembership(t, db, owner.ID, mine.ID, control.MemberRoleOwner)
	token := hhtSession(t, db, owner.ID, mine.ID)

	rec := hhtReq(t, a, http.MethodGet, "/platform/organizations", token, nil)
	if rec.Code < 400 {
		t.Fatalf("GET /platform/organizations answered %d for an ordinary customer who is an "+
			"OWNER of their own organization; the tenant directory is not theirs to read: %s",
			rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Someone Else Entirely") {
		t.Fatalf("the refusal echoed another tenant's name: %s", rec.Body.String())
	}
}

// THE PUBLIC READ IS REACHABLE WITH NO CREDENTIAL, AND CARRIES ONLY WHAT IT
// MUST.
//
// It exists for one ordering problem: the sign-in page decides whether to render
// the password form before anybody has signed in, so no authenticated endpoint
// can answer it. Everything about that makes it the endpoint most worth pinning
// — it is the only one in this package a stranger can reach.
func TestThePublicFlagReadIsAnonymousAndNarrow(t *testing.T) {
	db := hhtDB(t)
	a := hhtApp(t, db)

	// No token at all.
	rec := hhtReq(t, a, http.MethodGet, "/flags/public", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /flags/public answered %d with no credential, want 200 — the sign-in page "+
			"cannot authenticate to ask whether to show a sign-in form: %s",
			rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "legacy_login") {
		t.Fatalf("the sign-in page has no way to know whether to render its form: %s", body)
	}

	// Tenant-scoped flags must not appear AT ANY VALUE. An anonymous caller has
	// no organization, so a per-tenant answer is either somebody else's or
	// nobody's.
	for _, k := range []string{"proxmox.provisioning", "aws.provisioning"} {
		if strings.Contains(body, k) {
			t.Fatalf("%q is organization-scoped and reached an anonymous caller: %s", k, body)
		}
	}

	// And it carries the answer, not the configuration. Source, scope and the
	// built-in default describe how the installation is SET UP rather than what
	// it offers.
	for _, leak := range []string{`"source"`, `"scope"`, `"default"`, `"why"`} {
		if strings.Contains(body, leak) {
			t.Fatalf("the public read exposed %s, which describes how this installation is "+
				"configured rather than what it offers: %s", leak, body)
		}
	}
}
