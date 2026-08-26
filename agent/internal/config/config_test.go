package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// TestEveryConfigFieldCanBeSetFromTheEnvironment.
//
// THE BUG THIS PINS. viper's AutomaticEnv resolves on Get; Unmarshal only asks
// about keys that exist in SOME source. A key with a default exists, a key with
// nothing does not — so node_name, proxmox_token_id and proxmox_secret, the three
// with no default, were unreadable from the environment entirely.
//
// That is why no agent ever had a Proxmox client. Setting AGENT_PROXMOX_TOKEN_ID
// did nothing, the constructor correctly reported no token, and the refusal told
// operators to set exactly the variables the code could not read.
//
// Driven off the struct by reflection rather than a hand-written list, because a
// hand-written list is the thing that went out of date: the failure is a field
// added without a binding, and only the struct knows when one appears.
func TestEveryConfigFieldCanBeSetFromTheEnvironment(t *testing.T) {
	typ := reflect.TypeOf(Config{})

	for i := range typ.NumField() {
		f := typ.Field(i)
		key := f.Tag.Get("mapstructure")
		if key == "" {
			continue
		}
		// Only string fields are exercised: a duration needs a parseable value and
		// the binding is what is under test, not the parsing.
		if f.Type.Kind() != reflect.String {
			continue
		}

		t.Run(key, func(t *testing.T) {
			env := "AGENT_" + strings.ToUpper(key)
			want := "set-from-" + key
			t.Setenv(env, want)

			cfg, err := Load("", "", "")
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			got := reflect.ValueOf(cfg).Field(i).String()
			if got != want {
				t.Fatalf("%s did not reach Config.%s (got %q); the key is not bound, so the "+
					"environment cannot set it and an operator following the error message "+
					"changes nothing", env, f.Name, got)
			}
		})
	}
}
