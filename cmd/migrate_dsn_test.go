package cmd

import "testing"

// The grant is DDL, so the role name is interpolated rather than bound — which
// makes parsing it out of the DSN the place a mistake would land in a GRANT
// statement. These are the shapes a tenant DSN actually takes.
func TestRoleAndDatabaseFrom(t *testing.T) {
	role, db, err := roleAndDatabaseFrom("postgres://tenant_role_abc:pw@db.internal:5432/tenant_abc?sslmode=disable")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if role != "tenant_role_abc" || db != "tenant_abc" {
		t.Fatalf("role=%q db=%q, want tenant_role_abc / tenant_abc", role, db)
	}

	for name, dsn := range map[string]string{
		"no role":     "postgres://db.internal:5432/tenant_abc",
		"no database": "postgres://tenant_role_abc:pw@db.internal:5432/",
		"not a url":   "://///",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := roleAndDatabaseFrom(dsn); err == nil {
				t.Fatal("accepted a DSN it could not identify; the result would be interpolated " +
					"into a GRANT statement")
			}
		})
	}
}

// swapDatabase has to keep the admin credentials and change only the database —
// the grant is per-database and cannot be issued from the control database.
func TestSwapDatabaseKeepsCredentialsAndChangesOnlyTheDatabase(t *testing.T) {
	got, err := swapDatabase("postgres://apiary:secret@db.internal:5432/apiary?sslmode=disable", "tenant_abc")
	if err != nil {
		t.Fatalf("swap: %v", err)
	}
	const want = "postgres://apiary:secret@db.internal:5432/tenant_abc?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
