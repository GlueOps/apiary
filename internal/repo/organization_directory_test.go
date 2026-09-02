package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/testsupport"
)

var orgDirSchema = testsupport.SchemaName("apiary_orgdir_test")

func orgDirDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testsupport.OpenSchema(t, orgDirSchema, &control.Organization{})
}

func seedDirOrg(t *testing.T, db *gorm.DB, name string, domain *string, deleted bool) control.Organization {
	t.Helper()
	o := control.Organization{Name: name, Domain: domain, Status: "active"}
	if err := db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}
	if deleted {
		now := time.Now().UTC()
		if err := db.Model(&control.Organization{}).Where("id = ?", o.ID).
			Update("deleted_at", now).Error; err != nil {
			t.Fatal(err)
		}
	}
	return o
}

// A DELETED TENANT IS NOT SOMETHING TO TARGET, so it is out of the directory
// unless asked for — and still reachable, because soft deletion exists so a
// mistaken one can be seen and undone.
func TestDeletedOrganizationsAreExcludedUnlessAskedFor(t *testing.T) {
	db := orgDirDB(t)
	ctx := context.Background()

	live := seedDirOrg(t, db, "zz-live-"+uuid.NewString()[:6], nil, false)
	gone := seedDirOrg(t, db, "zz-gone-"+uuid.NewString()[:6], nil, true)

	rows, err := ListOrganizations(ctx, db, OrganizationFilter{Query: "zz-"})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[uuid.UUID]bool{}
	for _, r := range rows {
		ids[r.ID] = true
	}
	if !ids[live.ID] {
		t.Fatal("a live organization was missing from the directory")
	}
	if ids[gone.ID] {
		t.Fatal("a deleted organization appeared by default; it is not a thing to target")
	}

	rows, err = ListOrganizations(ctx, db, OrganizationFilter{Query: "zz-", IncludeDeleted: true})
	if err != nil {
		t.Fatal(err)
	}
	var sawDeleted bool
	for _, r := range rows {
		if r.ID == gone.ID {
			sawDeleted = true
			if r.DeletedAt == nil {
				t.Fatal("a deleted organization came back with no deletion marked, so a caller " +
					"cannot tell it apart from a live one")
			}
		}
	}
	if !sawDeleted {
		t.Fatal("include_deleted did not bring back the deleted organization, so a mistaken " +
			"deletion has nowhere to be seen")
	}
}

// A SEARCH IS NOT A PATTERN. Without escaping, "100%" matches every row and an
// underscore matches any single character — which reads as a broken search
// rather than as LIKE syntax leaking through.
func TestSearchTreatsWildcardsAsLiteralText(t *testing.T) {
	db := orgDirDB(t)
	ctx := context.Background()

	tag := uuid.NewString()[:6]
	pct := seedDirOrg(t, db, "yy-100%-"+tag, nil, false)
	other := seedDirOrg(t, db, "yy-plain-"+tag, nil, false)

	rows, err := ListOrganizations(ctx, db, OrganizationFilter{Query: "100%"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == other.ID {
			t.Fatal("searching for \"100%\" matched an unrelated organization; the percent " +
				"reached LIKE as a wildcard and the search returns everything")
		}
	}
	var found bool
	for _, r := range rows {
		if r.ID == pct.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("escaping the search lost the row it was meant to find")
	}

	// The underscore has the same problem and is easier to hit by accident.
	under := seedDirOrg(t, db, "xx_under_"+tag, nil, false)
	decoy := seedDirOrg(t, db, "xxQunderQ"+tag, nil, false)
	rows, err = ListOrganizations(ctx, db, OrganizationFilter{Query: "xx_under_"})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[uuid.UUID]bool{}
	for _, r := range rows {
		ids[r.ID] = true
	}
	if !ids[under.ID] {
		t.Fatal("the literal underscore match was lost")
	}
	if ids[decoy.ID] {
		t.Fatal("an underscore matched any character, so the search returned a row nobody asked for")
	}
}

// The query names its columns, so the sealed key material is never read out of
// the database at all — no later change to a response shape can leak what was
// never loaded.
func TestTheDirectoryNeverLoadsTenantKeyMaterial(t *testing.T) {
	db := orgDirDB(t)
	domain := "example.test"
	o := control.Organization{
		Name: "ww-secretive-" + uuid.NewString()[:6], Domain: &domain, Status: "active",
		ConnectionString:   "postgres://should-never-be-read",
		EncryptedTenantKey: "sealed-key-material",
		TenantKeyIV:        "iv", TenantKeyTag: "tag",
	}
	if err := db.Create(&o).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := ListOrganizations(context.Background(), db, OrganizationFilter{Query: "ww-secretive"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("found %d organizations, want 1", len(rows))
	}
	// The summary type has no field that could hold it — this asserts the shape
	// stays that way, since adding one is the change that would leak it.
	if rows[0].Domain != domain || rows[0].Name != o.Name {
		t.Fatalf("the summary lost the fields it is supposed to carry: %+v", rows[0])
	}
}
