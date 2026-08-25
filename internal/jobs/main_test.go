package jobs

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(admissionTestSchema, capacitySchema)
	os.Exit(code)
}
