package tenant_test

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

// TestMain drops the schema this package's database-backed tests create. The
// name is unique per test binary invocation, so without this a developer running
// the suite repeatedly accumulates schemas in the shared test database.
func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(swarmSchema)
	os.Exit(code)
}
