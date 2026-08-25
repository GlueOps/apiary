package workflow

import "testing"

// The admission periodic jobs are inserted by the worker's own scheduler, never
// from the request path, so Enqueuer must NOT grow methods for them: every test
// double in the tree (MockEnqueuer, the counting enqueuer in internal/service)
// would otherwise stop compiling and get "fixed" by weakening it.
var _ Enqueuer = (*MockEnqueuer)(nil)

func TestAdmissionArgsCarryNoRequestPathPayload(t *testing.T) {
	if kind := (OrgProvingArgs{}).Kind(); kind != "admission.org_proving" {
		t.Fatalf("OrgProvingArgs kind = %q, want admission.org_proving", kind)
	}
	if kind := (DomainCohortAuditArgs{}).Kind(); kind != "admission.domain_cohort_audit" {
		t.Fatalf("DomainCohortAuditArgs kind = %q, want admission.domain_cohort_audit", kind)
	}
}
