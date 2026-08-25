package estate

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

// TestMain drops the schemas this run created.
//
// The names are unique per invocation so two concurrent runs cannot truncate
// each other's fixtures. Without this they would simply accumulate instead —
// trading a collision problem for unbounded sprawl in a shared database.
func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(edgeControlSchema, edgeTenantSchema, edgeEmptySchema, edgeOtherTenantSchema)
	os.Exit(code)
}
