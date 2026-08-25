package apimiddleware

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(accessTestSchema)
	os.Exit(code)
}
