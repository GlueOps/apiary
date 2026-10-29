package fleet

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

// TestMain drops the schema this package's HTTP tests create.
//
// The schema name is unique per test binary invocation, so without this a
// developer running the suite repeatedly accumulates schemas in the shared test
// database rather than colliding with themselves.
func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(fleetopsSchema, platformSchema,
		lifecycleTenantSchema, lifecycleControlSchema, departureSchema)
	os.Exit(code)
}
