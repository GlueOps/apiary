package jobs

import (
	"context"
	"strings"
	"testing"
)

// TestTheCredentialIsWrittenAsAFileNotAnArgument.
//
// The enrolment token is passed on the command line deliberately — single-use,
// expires in minutes, redeemed within a second. A Proxmox token is neither: it
// is long-lived and can create and destroy guests, and an argument is visible in
// the host's process list to every user on the machine.
func TestTheCredentialIsWrittenAsAFileNotAnArgument(t *testing.T) {
	t.Parallel()

	r := &fakeRunner{}
	err := WriteAgentCredential(context.Background(), r, "http://cp:8282", ProxmoxCredential{
		TokenID: "root@pam!apiary",
		Secret:  "the-secret-value",
	})
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, cmd := range r.cmds {
		if strings.Contains(cmd, "the-secret-value") {
			t.Fatalf("the secret was passed as a command, visible in the process list: %q", cmd)
		}
	}
	if len(r.uploads) == 0 || r.uploads[len(r.uploads)-1] != AgentEnvPath {
		t.Fatalf("the credential was not uploaded to %s; uploads were %v", AgentEnvPath, r.uploads)
	}
}

// TestAMachineInNoFleetIsNotAnError.
//
// Every migrating host passes through that state, and there is no cluster whose
// token would be the right one. Failing here would make a machine that is
// perfectly onboardable look like an onboarding failure.
func TestAMachineInNoFleetIsNotAnError(t *testing.T) {
	t.Parallel()

	r := &fakeRunner{}
	if err := WriteAgentCredential(context.Background(), r, "http://cp:8282",
		ProxmoxCredential{}); err != nil {
		t.Fatalf("an empty credential was an error: %v", err)
	}
	if len(r.uploads) != 0 {
		t.Fatalf("something was written for a machine with no credential: %v", r.uploads)
	}
}

// TestHalfACredentialIsNotWritten. A token id with no secret cannot
// authenticate, and writing it would make the agent report a configured token
// that fails on every call.
func TestHalfACredentialIsNotWritten(t *testing.T) {
	t.Parallel()

	for _, c := range []ProxmoxCredential{
		{TokenID: "root@pam!apiary"},
		{Secret: "only-the-secret"},
	} {
		r := &fakeRunner{}
		if err := WriteAgentCredential(context.Background(), r, "http://cp:8282", c); err != nil {
			t.Fatalf("write: %v", err)
		}
		if len(r.uploads) != 0 {
			t.Fatalf("half a credential was written: %+v", c)
		}
	}
}
