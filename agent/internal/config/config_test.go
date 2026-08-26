package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestACommandRunByHandSeesTheServiceConfiguration.
//
// THE BUG THIS FIXES. AutomaticEnv reads the PROCESS environment, and a shell has
// none of these — AGENT_BASE_URL lives in a file that only systemd loads. So
// every subcommand run by hand silently fell back to http://localhost:8080,
// which on a hypervisor is the hypervisor: `apiary-agent upgrade` on a correctly
// configured host reported connection refused with the real setting sitting in
// /etc/apiary-agent/agent.env the whole time.
func TestACommandRunByHandSeesTheServiceConfiguration(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, "agent.env")
	if err := os.WriteFile(env, []byte("AGENT_BASE_URL=http://10.10.10.1:8282\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("AGENT_BASE_URL", "")
	if err := os.Unsetenv("AGENT_BASE_URL"); err != nil {
		t.Fatalf("unset: %v", err)
	}
	loadEnvFile(env)

	if got := os.Getenv("AGENT_BASE_URL"); got != "http://10.10.10.1:8282" {
		t.Fatalf("AGENT_BASE_URL = %q, want the value from the env file", got)
	}
}

// TestAnExportedVariableBeatsTheFile.
//
// An exported variable is more specific than a file, and somebody who exports
// AGENT_BASE_URL to reach a different control plane means it.
func TestAnExportedVariableBeatsTheFile(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, "agent.env")
	if err := os.WriteFile(env, []byte("AGENT_BASE_URL=http://from-the-file:8282\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("AGENT_BASE_URL", "http://exported:9999")
	loadEnvFile(env)

	if got := os.Getenv("AGENT_BASE_URL"); got != "http://exported:9999" {
		t.Fatalf("AGENT_BASE_URL = %q, want the exported value to win", got)
	}
}

// TestTheEnvFileFormatSystemdAccepts, since an operator editing it by hand will
// use quoting and comments and expect them to work.
func TestTheEnvFileFormatSystemdAccepts(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, "agent.env")
	body := "# a comment\n\n" +
		"AGENT_ONE=plain\n" +
		"AGENT_TWO=\"double quoted\"\n" +
		"AGENT_THREE='single quoted'\n" +
		"  AGENT_FOUR = spaced  \n" +
		"not a pair\n"
	if err := os.WriteFile(env, []byte(body), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	for _, k := range []string{"AGENT_ONE", "AGENT_TWO", "AGENT_THREE", "AGENT_FOUR"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
	}

	loadEnvFile(env)

	for k, want := range map[string]string{
		"AGENT_ONE":   "plain",
		"AGENT_TWO":   "double quoted",
		"AGENT_THREE": "single quoted",
		"AGENT_FOUR":  "spaced",
	} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

// TestAMissingEnvFileIsNotAnError. A host configured entirely through the
// environment, or through --base-url, has none and is correct.
func TestAMissingEnvFileIsNotAnError(t *testing.T) {
	loadEnvFile(filepath.Join(t.TempDir(), "nope.env"))
}

// TestLoadStillHonoursTheOverride, which is what the onboarding path passes and
// what the UI's enrolment command carries.
func TestLoadStillHonoursTheOverride(t *testing.T) {
	cfg, err := Load("", "http://override:1234", "")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.BaseURL != "http://override:1234" {
		t.Fatalf("BaseURL = %q, want the explicit override to win", cfg.BaseURL)
	}
}
