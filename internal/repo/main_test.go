package repo

import (
	"os"
	"testing"

	"github.com/glueops/apiary/internal/testsupport"
)

// TestMain drops the per-run schemas this package creates.
//
// The names are unique per invocation, which is what stops two concurrent runs
// treading on each other — but unique names left behind would accumulate one set
// of schemas per run in a shared database. Dropping them here is the other half
// of that trade.
func TestMain(m *testing.M) {
	code := m.Run()
	testsupport.DropSchemas(
		accessRuleTestSchema,
		authFailureTestSchema,
		credentialTestSchema,
		declareTestSchema,
		estateTestSchema,
		firewallTestSchema,
		publishTestSchema,
		enrolmentTestSchema,
		fleetTestSchema,
		factsTestSchema,
		sshKeyTestSchema,
		socialTestSchema,
	)
	os.Exit(code)
}
