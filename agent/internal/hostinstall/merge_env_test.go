package hostinstall

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallPreservesTheCredentialOnboardingJustWrote.
//
// THE ORDERING THIS PROTECTS. Onboarding writes agent.env to hand the agent its
// Proxmox credential, and `enrol --install` runs immediately afterwards. Writing
// the whole file there would delete the credential seconds after it arrived, and
// every cluster task would then refuse on a host that had just been given
// everything it needed — with the control plane reporting a successful onboard.
func TestInstallPreservesTheCredentialOnboardingJustWrote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agent.env")
	existing := "# Managed by apiary\n" +
		"AGENT_BASE_URL=http://old:8282\n" +
		"AGENT_PROXMOX_TOKEN_ID=root@pam!apiary\n" +
		"AGENT_PROXMOX_SECRET=s3cr3t\n"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := mergeEnv(path, map[string]string{"AGENT_BASE_URL": "http://new:8282"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	body := string(got)

	if !strings.Contains(body, "AGENT_PROXMOX_TOKEN_ID=root@pam!apiary") ||
		!strings.Contains(body, "AGENT_PROXMOX_SECRET=s3cr3t") {
		t.Fatalf("the credential was dropped:\n%s", body)
	}
	if !strings.Contains(body, "AGENT_BASE_URL=http://new:8282") {
		t.Fatalf("the base URL was not updated:\n%s", body)
	}
	if strings.Contains(body, "http://old:8282") {
		t.Fatalf("the old base URL survived alongside the new one:\n%s", body)
	}
	// Comments survive, because this is a file people edit and silently
	// reformatting it on every convergence is how an operator stops trusting that
	// their edits last.
	if !strings.Contains(body, "# Managed by apiary") {
		t.Fatalf("the comment was dropped:\n%s", body)
	}
}

// TestMergingIntoAMissingFileJustCreatesIt, which is the first-install case.
func TestMergingIntoAMissingFileJustCreatesIt(t *testing.T) {
	t.Parallel()

	got, err := mergeEnv(filepath.Join(t.TempDir(), "absent.env"),
		map[string]string{"AGENT_BASE_URL": "http://cp:8282"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if strings.TrimSpace(string(got)) != "AGENT_BASE_URL=http://cp:8282" {
		t.Fatalf("got %q", got)
	}
}

// TestMergingIsStable, so two convergences over unchanged inputs produce the
// same file and a diff shows only what actually changed.
func TestMergingIsStable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "agent.env")
	if err := os.WriteFile(path, []byte("AGENT_ONE=1\nAGENT_TWO=2\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	set := map[string]string{"AGENT_THREE": "3", "AGENT_FOUR": "4"}

	first, err := mergeEnv(path, set)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := mergeEnv(path, set)
		if err != nil {
			t.Fatalf("merge: %v", err)
		}
		if !bytes.Equal(again, first) {
			t.Fatalf("output moved between runs:\n%s\nthen\n%s", first, again)
		}
	}
}
