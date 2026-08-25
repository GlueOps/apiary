package api

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

// TestMain drops the schemas this package's DB-backed tests created.
//
// Schema names are unique per test binary invocation (testsupport.SchemaName),
// which is what stops two concurrent runs truncating each other's fixtures. The
// cost of that is sprawl: without this, every run leaves a schema behind in the
// shared test database forever. Add any new schema var to the call below rather
// than replacing it.
func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(hhtSchema)
	os.Exit(code)
}
