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
