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

// TestTheCredentialFileIsWrittenWholeSoAStaleTokenCannotSurvive.
//
// This file is written at onboarding and again after every join, and a join is
// exactly when the previous contents become wrong: the node's token database is
// replaced with the target cluster's, so the credential it held belongs to a
// cluster it has left.
//
// Written whole rather than merged for that reason. Merging would keep the old
// token id beside the new one if the keys ever differed, and a host holding two
// credentials where one is dead is worse than one holding neither — it fails
// intermittently rather than plainly.
func TestTheCredentialFileIsWrittenWholeSoAStaleTokenCannotSurvive(t *testing.T) {
	t.Parallel()

	r := &fakeRunner{}
	if err := WriteAgentCredential(context.Background(), r, "http://cp:8282", ProxmoxCredential{
		TokenID: "root@pam!fleet-b",
		Secret:  "second",
	}); err != nil {
		t.Fatalf("write: %v", err)
	}

	body := string(r.uploadBodies[len(r.uploadBodies)-1])
	if strings.Count(body, "AGENT_PROXMOX_TOKEN_ID=") != 1 {
		t.Fatalf("more than one token id in the file:\n%s", body)
	}
	if !strings.Contains(body, "AGENT_PROXMOX_TOKEN_ID=root@pam!fleet-b") {
		t.Fatalf("the new token id is missing:\n%s", body)
	}
	// The base URL travels with it: this file is written whole, so leaving it out
	// would strip the control plane address the agent needs to heartbeat at all.
	if !strings.Contains(body, "AGENT_BASE_URL=http://cp:8282") {
		t.Fatalf("the base URL was dropped, which would leave the agent unable to report:\n%s",
			body)
	}
}
