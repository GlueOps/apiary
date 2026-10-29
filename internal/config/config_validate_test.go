package config

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Shared fixtures
//
// Everything in this file is prefixed "cfgt" because several agents are adding
// tests to this package concurrently and Go has one flat namespace per package.
// ---------------------------------------------------------------------------

// cfgtValidKey is a syntactically correct ENCRYPTION_MASTER_KEY: base64 of
// exactly utils.MasterKeyLen (32) bytes. Built rather than pasted so that a
// change to the required length breaks compilation here instead of silently
// making every "valid" config in this file invalid for the wrong reason.
var cfgtValidKey = base64.StdEncoding.EncodeToString(make([]byte, 32))

// cfgtValidConfig returns a Config that Validate() must accept.
//
// Every table row below starts from this and breaks exactly ONE rule, so a row
// that fails tells you which rule stopped enforcing itself rather than which
// unrelated field happened to be zero. Env is deliberately "production": the
// dev-only escape hatches (FRONTEND_MODE=proxy, CORS "*", shared provisioner
// DSN) must be exercised against the environment where they are refused, not
// against the one where they are waved through.
func cfgtValidConfig() *Config {
	return &Config{
		DatabaseURL:         "postgres://app:pw@db:5432/apiary?sslmode=require",
		BindPort:            "8080",
		BindHost:            "127.0.0.1",
		Env:                 "production",
		EncryptionMasterKey: cfgtValidKey,
		BasePath:            "/api/v1",
		BaseURL:             "https://apiary.example.com",
		FrontendMode:        FrontendModeEmbed,
		JWTSecret:           "a-jwt-secret",
		JWTIssuer:           "apiary",
		JWTAccessTTLMin:     15,
		JWTRefreshTTLHour:   168,
		CORSAllowedOrigins:  "https://apiary.example.com",

		TenantPoolMaxSize:            64,
		TenantPoolMaxConns:           4,
		TenantPoolMaxIdleConns:       2,
		TenantPoolConnMaxIdleMin:     5,
		TenantPoolConnMaxLifetimeMin: 60,
		TenantPoolEvictGraceSec:      60,

		OrgsMaxTotal:      1000,
		OrgsMaxNewPerHour: 200,

		AdmissionUnprovenOrgsPerDomain: 3,
		AdmissionProvenDistinctDays:    2,
		AdmissionProvingIntervalHours:  24,
		AdmissionCohortIntervalHours:   24,
		AdmissionCohortMinSample:       5,
		AdmissionCohortProvenRatio:     0.25,
	}
}

// cfgtRequireErrContains asserts that err is non-nil AND names the setting it is
// about. Config.Validate has no sentinel errors to match with errors.Is — every
// failure is a fmt.Errorf — so the only machine-checkable identity a failure has
// is the environment variable name it prints. Matching on that name is what
// makes each table row test ONE rule: if the rule under test were deleted, the
// config would either pass (err == nil) or trip a different rule with a
// different name, and either way this fails.
func cfgtRequireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Validate accepted a config that breaks the %s rule; "+
			"an invalid setting reaching the running server is exactly what this "+
			"validation exists to prevent", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Validate rejected the config but blamed the wrong setting:\n"+
			"  got:  %v\n"+
			"  want: an error naming %q\n"+
			"An operator reads this message to know which variable to fix; naming the "+
			"wrong one sends them to edit a setting that was never wrong.", err, want)
	}
}

// ---------------------------------------------------------------------------
// Config.Validate — one row per rule the function enforces
// ---------------------------------------------------------------------------

func TestCfgtValidateRejectsEachBrokenRule(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "jwt secret unset",
			mutate:  func(c *Config) { c.JWTSecret = "" },
			wantErr: "JWT_SECRET",
		},
		{
			// Whitespace is not a secret. Accepting " " would boot a server that
			// signs tokens with a value an attacker can guess on the first try.
			name:    "jwt secret is only whitespace",
			mutate:  func(c *Config) { c.JWTSecret = "   \t\n " },
			wantErr: "JWT_SECRET",
		},
		{
			name:    "jwt issuer unset",
			mutate:  func(c *Config) { c.JWTIssuer = "" },
			wantErr: "JWT_ISSUER",
		},
		{
			name:    "jwt issuer is only whitespace",
			mutate:  func(c *Config) { c.JWTIssuer = "  " },
			wantErr: "JWT_ISSUER",
		},
		{
			// The master key is the root of every derived key (TOTP seeds, tenant
			// DEKs, tenant DSNs). An unset one must stop startup, not default to
			// something.
			name:    "encryption master key unset",
			mutate:  func(c *Config) { c.EncryptionMasterKey = "" },
			wantErr: "ENCRYPTION_MASTER_KEY",
		},
		{
			name:    "encryption master key is not base64",
			mutate:  func(c *Config) { c.EncryptionMasterKey = "not!valid!base64!" },
			wantErr: "ENCRYPTION_MASTER_KEY",
		},
		{
			// Valid base64 of the wrong length is the dangerous case: it decodes
			// fine, so only an explicit length check catches a 16-byte key being
			// used where 32 bytes of entropy were promised.
			name: "encryption master key decodes to the wrong length",
			mutate: func(c *Config) {
				c.EncryptionMasterKey = base64.StdEncoding.EncodeToString(make([]byte, 16))
			},
			wantErr: "ENCRYPTION_MASTER_KEY",
		},
		{
			name:    "access token ttl is zero",
			mutate:  func(c *Config) { c.JWTAccessTTLMin = 0 },
			wantErr: "JWT_ACCESS_TTL_MIN",
		},
		{
			name:    "access token ttl is negative",
			mutate:  func(c *Config) { c.JWTAccessTTLMin = -1 },
			wantErr: "JWT_ACCESS_TTL_MIN",
		},
		{
			name:    "refresh token ttl is zero",
			mutate:  func(c *Config) { c.JWTRefreshTTLHour = 0 },
			wantErr: "JWT_REFRESH_TTL_HOUR",
		},
		{
			name:    "refresh token ttl is negative",
			mutate:  func(c *Config) { c.JWTRefreshTTLHour = -24 },
			wantErr: "JWT_REFRESH_TTL_HOUR",
		},
		{
			name:    "frontend mode is not a known mode",
			mutate:  func(c *Config) { c.FrontendMode = "spa" },
			wantErr: "FRONTEND_MODE",
		},
		{
			// proxy mode forwards unmatched requests to a Vite dev server. Outside
			// development that is an unauthenticated proxy hanging off the API.
			name:    "frontend proxy mode outside development",
			mutate:  func(c *Config) { c.FrontendMode = FrontendModeProxy },
			wantErr: "FRONTEND_MODE",
		},
		{
			name:    "cors origin is not an absolute origin",
			mutate:  func(c *Config) { c.CORSAllowedOrigins = "apiary.example.com" },
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name:    "cors wildcard outside development",
			mutate:  func(c *Config) { c.CORSAllowedOrigins = "*" },
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "cors unset and base url is not an origin",
			mutate: func(c *Config) {
				c.CORSAllowedOrigins = ""
				c.BaseURL = "apiary.example.com"
			},
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name:    "tenant pool max size below one",
			mutate:  func(c *Config) { c.TenantPoolMaxSize = 0 },
			wantErr: "TENANT_POOL_MAX_SIZE",
		},
		{
			// 0 in database/sql means "unlimited", so an unguarded 0 here would let
			// a single tenant open connections until the Postgres backend limit is
			// exhausted for every other tenant.
			name:    "tenant pool max conns below one",
			mutate:  func(c *Config) { c.TenantPoolMaxConns = 0 },
			wantErr: "TENANT_POOL_MAX_CONNS",
		},
		{
			name:    "tenant pool max idle conns negative",
			mutate:  func(c *Config) { c.TenantPoolMaxIdleConns = -1 },
			wantErr: "TENANT_POOL_MAX_IDLE_CONNS",
		},
		{
			// The cross-field rule: idle > open is incoherent and silently clamped
			// by database/sql, so the operator's stated intent would be ignored.
			name: "tenant pool idle conns above max conns",
			mutate: func(c *Config) {
				c.TenantPoolMaxConns = 4
				c.TenantPoolMaxIdleConns = 5
			},
			wantErr: "TENANT_POOL_MAX_IDLE_CONNS",
		},
		{
			name:    "tenant pool conn max idle minutes negative",
			mutate:  func(c *Config) { c.TenantPoolConnMaxIdleMin = -1 },
			wantErr: "TENANT_POOL_CONN_MAX_IDLE_MIN",
		},
		{
			name:    "tenant pool conn max lifetime minutes negative",
			mutate:  func(c *Config) { c.TenantPoolConnMaxLifetimeMin = -1 },
			wantErr: "TENANT_POOL_CONN_MAX_LIFETIME_MIN",
		},
		{
			name:    "tenant pool evict grace negative",
			mutate:  func(c *Config) { c.TenantPoolEvictGraceSec = -1 },
			wantErr: "TENANT_POOL_EVICT_GRACE_SEC",
		},
		{
			name:    "orgs max total below one",
			mutate:  func(c *Config) { c.OrgsMaxTotal = 0 },
			wantErr: "ORGS_MAX_TOTAL",
		},
		{
			name:    "orgs max new per hour below one",
			mutate:  func(c *Config) { c.OrgsMaxNewPerHour = 0 },
			wantErr: "ORGS_MAX_NEW_PER_HOUR",
		},
		{
			// 0 here would refuse every organization, including the first one a
			// legitimate domain creates — a total outage dressed up as a limit.
			name:    "admission unproven orgs per domain below one",
			mutate:  func(c *Config) { c.AdmissionUnprovenOrgsPerDomain = 0 },
			wantErr: "ADMISSION_UNPROVEN_ORGS_PER_DOMAIN",
		},
		{
			// 1 is the trap value, not 0: registration itself issues the first token
			// session, so "1 distinct day" proves every organization the next day and
			// makes the unproven-org ceiling inert. The floor is 2 for that reason,
			// which is why this row uses 1 rather than 0.
			name:    "admission proven distinct days below two",
			mutate:  func(c *Config) { c.AdmissionProvenDistinctDays = 1 },
			wantErr: "ADMISSION_PROVEN_DISTINCT_DAYS",
		},
		{
			name:    "admission proving interval below one hour",
			mutate:  func(c *Config) { c.AdmissionProvingIntervalHours = 0 },
			wantErr: "ADMISSION_PROVING_INTERVAL_HOURS",
		},
		{
			name:    "admission cohort interval below one hour",
			mutate:  func(c *Config) { c.AdmissionCohortIntervalHours = 0 },
			wantErr: "ADMISSION_COHORT_INTERVAL_HOURS",
		},
		{
			name:    "admission cohort min sample below one",
			mutate:  func(c *Config) { c.AdmissionCohortMinSample = 0 },
			wantErr: "ADMISSION_COHORT_MIN_SAMPLE",
		},
		{
			// Both ends are closed: 0 flags no domain, 1 flags every domain, and
			// both read as a broken audit rather than as a setting.
			name:    "admission cohort ratio at zero",
			mutate:  func(c *Config) { c.AdmissionCohortProvenRatio = 0 },
			wantErr: "ADMISSION_COHORT_PROVEN_RATIO",
		},
		{
			name:    "admission cohort ratio negative",
			mutate:  func(c *Config) { c.AdmissionCohortProvenRatio = -0.5 },
			wantErr: "ADMISSION_COHORT_PROVEN_RATIO",
		},
		{
			name:    "admission cohort ratio at one",
			mutate:  func(c *Config) { c.AdmissionCohortProvenRatio = 1 },
			wantErr: "ADMISSION_COHORT_PROVEN_RATIO",
		},
		{
			name:    "admission cohort ratio above one",
			mutate:  func(c *Config) { c.AdmissionCohortProvenRatio = 1.5 },
			wantErr: "ADMISSION_COHORT_PROVEN_RATIO",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := cfgtValidConfig()
			tc.mutate(cfg)
			cfgtRequireErrContains(t, cfg.Validate(), tc.wantErr)
		})
	}
}

// TestCfgtValidateBaselineIsAccepted is the control for the table above. Without
// it, a Validate() that rejected everything unconditionally would make every
// row pass while making the server unbootable.
func TestCfgtValidateBaselineIsAccepted(t *testing.T) {
	if err := cfgtValidConfig().Validate(); err != nil {
		t.Fatalf("Validate rejected a fully populated, correct production config: %v\n"+
			"Every row of the rejection table starts from this config, so if the baseline "+
			"is invalid those rows prove nothing — and in production this is a server that "+
			"refuses to start with correct settings.", err)
	}
}

// TestCfgtValidateAcceptsBoundaryValues pins the OPEN end of every inequality.
// The rejection table proves each rule fires; these rows prove none of them
// fires one step too early. A ">= 1" quietly becoming "> 1" is invisible to a
// test suite that only ever supplies obviously-bad values.
func TestCfgtValidateAcceptsBoundaryValues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"access ttl of one minute", func(c *Config) { c.JWTAccessTTLMin = 1 }},
		{"refresh ttl of one hour", func(c *Config) {
			c.JWTRefreshTTLHour = 1
			// Keep the proving interval below the refresh TTL so this row exercises
			// the TTL floor only; the interval/TTL relation has its own test.
			c.AdmissionProvingIntervalHours = 1
		}},
		{"tenant pool size of one", func(c *Config) { c.TenantPoolMaxSize = 1 }},
		{"one connection per tenant with no idle connections", func(c *Config) {
			c.TenantPoolMaxConns = 1
			c.TenantPoolMaxIdleConns = 0
		}},
		{"idle conns exactly equal to max conns", func(c *Config) {
			c.TenantPoolMaxConns = 4
			c.TenantPoolMaxIdleConns = 4
		}},
		{"zero idle timeout means no idle timeout", func(c *Config) { c.TenantPoolConnMaxIdleMin = 0 }},
		{"zero lifetime means unlimited lifetime", func(c *Config) { c.TenantPoolConnMaxLifetimeMin = 0 }},
		{"zero evict grace", func(c *Config) { c.TenantPoolEvictGraceSec = 0 }},
		{"one organization total", func(c *Config) { c.OrgsMaxTotal = 1 }},
		{"one new organization per hour", func(c *Config) { c.OrgsMaxNewPerHour = 1 }},
		{"one unproven org per domain", func(c *Config) { c.AdmissionUnprovenOrgsPerDomain = 1 }},
		{"exactly two distinct proving days", func(c *Config) { c.AdmissionProvenDistinctDays = 2 }},
		{"one hour proving interval", func(c *Config) { c.AdmissionProvingIntervalHours = 1 }},
		{"one hour cohort interval", func(c *Config) { c.AdmissionCohortIntervalHours = 1 }},
		{"cohort sample of one", func(c *Config) { c.AdmissionCohortMinSample = 1 }},
		{"cohort ratio just above zero", func(c *Config) { c.AdmissionCohortProvenRatio = 0.0001 }},
		{"cohort ratio just below one", func(c *Config) { c.AdmissionCohortProvenRatio = 0.9999 }},
		{"empty frontend mode means none", func(c *Config) { c.FrontendMode = "" }},
		{"cors unset falls back to an absolute BASE_URL", func(c *Config) { c.CORSAllowedOrigins = "" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := cfgtValidConfig()
			tc.mutate(cfg)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate rejected a value that sits exactly on the allowed "+
					"boundary: %v\nAn off-by-one in a bound refuses a legitimate "+
					"deployment at startup, which looks like an outage, not a config error.", err)
			}
		})
	}
}

// TestCfgtValidateWarnsButAcceptsProvingIntervalAboveRefreshTTL pins a
// deliberate non-rule.
//
// A proving interval at or above JWT_REFRESH_TTL_HOUR risks losing proving
// evidence between runs (TokenSessionCleanupWorker deletes token_sessions once
// both tokens have expired), so Validate logs a warning — but it must NOT
// refuse to start. Turning that warning into an error would take down every
// deployment that tuned these two knobs independently. This test exists so that
// "tighten the check" is a conscious decision rather than a drive-by edit.
func TestCfgtValidateWarnsButAcceptsProvingIntervalAboveRefreshTTL(t *testing.T) {
	for _, interval := range []int{168, 200} {
		cfg := cfgtValidConfig()
		cfg.JWTRefreshTTLHour = 168
		cfg.AdmissionProvingIntervalHours = interval

		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate refused ADMISSION_PROVING_INTERVAL_HOURS=%d against "+
				"JWT_REFRESH_TTL_HOUR=168: %v\nThis combination is a warning, not an "+
				"error; promoting it to a hard failure stops servers that boot today.",
				interval, err)
		}
	}
}

// TestCfgtValidateDevEnvUnlocksDevOnlyModes checks the escape hatches open in
// development and only in development. Both halves matter: a hatch that never
// opens makes local development impossible, and one that opens in production is
// the vulnerability.
func TestCfgtValidateDevEnvUnlocksDevOnlyModes(t *testing.T) {
	for _, env := range []string{EnvDevelopment, EnvTest, "  Development  ", "TEST"} {
		t.Run("env="+env, func(t *testing.T) {
			cfg := cfgtValidConfig()
			cfg.Env = env
			cfg.FrontendMode = FrontendModeProxy
			cfg.CORSAllowedOrigins = "*"
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate refused the development-only settings while ENV=%q: %v\n"+
					"ENV is normalized (trimmed and lowercased) before the dev check; if that "+
					"stops happening, a developer's ENV=\"Development\" is treated as production.",
					env, err)
			}
		})
	}

	for _, env := range []string{"production", "", "prod", "staging", "developmentx"} {
		t.Run("non-dev env="+env, func(t *testing.T) {
			cfg := cfgtValidConfig()
			cfg.Env = env
			cfg.CORSAllowedOrigins = "*"
			cfgtRequireErrContains(t, cfg.Validate(), "CORS_ALLOWED_ORIGINS")

			cfg = cfgtValidConfig()
			cfg.Env = env
			cfg.FrontendMode = FrontendModeProxy
			cfgtRequireErrContains(t, cfg.Validate(), "FRONTEND_MODE")
		})
	}
}

// ---------------------------------------------------------------------------
// validateCORSOrigins
// ---------------------------------------------------------------------------

// TestCfgtValidateCORSOrigins covers the list parser directly, including the
// shapes Config.Validate cannot reach cleanly (multi-entry lists, blank
// entries, one bad entry among good ones).
//
// This is the gate in front of internal/api's corsOriginPolicy, which reflects
// these origins back with Access-Control-Allow-Credentials. An entry that slips
// through here becomes a credentialed cross-origin allowance.
func TestCfgtValidateCORSOrigins(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		baseURL string
		env     string
		wantErr bool
		// wantErrHas, when set, must appear in the message — used where the
		// error must blame a specific offending entry rather than the list.
		wantErrHas string
	}{
		{
			name: "single origin", raw: "https://app.test", baseURL: "https://app.test",
			env: "production",
		},
		{
			name: "several origins with surrounding whitespace",
			raw:  " https://a.test ,\thttps://b.test:8443 ", baseURL: "https://a.test",
			env: "production",
		},
		{
			// Blank entries come from trailing commas in .env files and are the
			// most common typo; they must be skipped, not treated as an origin.
			name: "empty entries are skipped",
			raw:  "https://a.test,,   ,https://b.test", baseURL: "https://a.test",
			env: "production",
		},
		{
			name: "one bad entry among good ones is rejected",
			raw:  "https://a.test,ftp://b.test,https://c.test", baseURL: "https://a.test",
			env: "production", wantErr: true, wantErrHas: "ftp://b.test",
		},
		{
			// A bare hostname matches no browser Origin header, so a deployment
			// configured this way silently allows nobody. Failing loudly at boot
			// is the difference between a config error and a mystery outage.
			name: "bare hostname is not an origin",
			raw:  "app.test", baseURL: "https://app.test",
			env: "production", wantErr: true, wantErrHas: "app.test",
		},
		{
			name: "wildcard is refused in production",
			raw:  "*", baseURL: "https://app.test", env: "production", wantErr: true,
		},
		{
			name: "wildcard is refused when ENV is unset",
			raw:  "*", baseURL: "https://app.test", env: "", wantErr: true,
		},
		{
			name: "wildcard is allowed in development",
			raw:  "*", baseURL: "https://app.test", env: EnvDevelopment,
		},
		{
			name: "wildcard alongside explicit origins is still refused in production",
			raw:  "https://a.test,*", baseURL: "https://a.test", env: "production", wantErr: true,
		},
		{
			name: "unset list falls back to an absolute BASE_URL",
			raw:  "", baseURL: "https://app.test", env: "production",
		},
		{
			name: "unset list with an unusable BASE_URL is refused in production",
			raw:  "", baseURL: "app.test", env: "production", wantErr: true,
			wantErrHas: "BASE_URL",
		},
		{
			name: "whitespace-only list counts as unset",
			raw:  "   ", baseURL: "", env: "production", wantErr: true,
			wantErrHas: "BASE_URL",
		},
		{
			// Development reflects every origin, so BASE_URL is never consulted;
			// requiring it there would break `go test` and bare `go run`.
			name: "unset list with an unusable BASE_URL is fine in development",
			raw:  "", baseURL: "", env: EnvTest,
		},
		{
			// The BASE_URL fallback is only reached when the list is empty. An
			// explicit list must stand on its own, whatever BASE_URL says.
			name: "explicit list does not require a usable BASE_URL",
			raw:  "https://a.test", baseURL: "", env: "production",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCORSOrigins(tc.raw, tc.baseURL, tc.env)
			switch {
			case tc.wantErr && err == nil:
				t.Fatalf("validateCORSOrigins(%q, %q, %q) accepted a list it must refuse; "+
					"these origins are reflected back with credentials allowed",
					tc.raw, tc.baseURL, tc.env)
			case !tc.wantErr && err != nil:
				t.Fatalf("validateCORSOrigins(%q, %q, %q) refused a legitimate list: %v; "+
					"this runs at startup, so a false rejection is a server that will not boot",
					tc.raw, tc.baseURL, tc.env, err)
			case tc.wantErr && tc.wantErrHas != "" && !strings.Contains(err.Error(), tc.wantErrHas):
				t.Fatalf("validateCORSOrigins refused the list but did not name %q:\n  got: %v\n"+
					"The operator has to find the offending entry in a comma-separated list; "+
					"an error that omits it makes them guess.", tc.wantErrHas, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// OriginOf
// ---------------------------------------------------------------------------

// TestCfgtOriginOf is the highest-stakes table in this file.
//
// internal/api/router.go builds the CORS allow-list by running every configured
// entry through OriginOf and comparing the result against the browser's Origin
// header. So OriginOf defines origin equality for this service. Two failure
// modes matter and they point in opposite directions:
//
//   - Too permissive: returning a non-empty origin for something that is not an
//     origin, or dropping a component (port, scheme) that distinguishes two
//     origins, lets a site the operator never listed make credentialed requests.
//   - Too strict: normalizing away something a browser also normalizes away
//     (a trailing slash, a default port, letter case in the host) makes a
//     correctly configured origin match nothing.
func TestCfgtOriginOf(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain https origin", "https://app.test", "https://app.test"},
		{"explicit non-default port is preserved", "http://localhost:5173", "http://localhost:5173"},

		// A browser sends "https://app.test", never "https://app.test/". If the
		// slash survived normalization, an operator writing the URL the way their
		// browser shows it would configure an origin that matches nothing.
		{"trailing slash is dropped", "https://app.test/", "https://app.test"},
		{"path is dropped", "https://app.test/some/page", "https://app.test"},
		{"query and fragment are dropped", "https://app.test/x?a=1#f", "https://app.test"},

		// Host comparison is case-insensitive in DNS and browsers lowercase the
		// Origin header, so the allow-list must be lowercased too.
		{"host is lowercased", "https://APP.Test", "https://app.test"},
		{"scheme is lowercased", "HTTPS://app.test", "https://app.test"},
		{"surrounding whitespace is trimmed", "  https://app.test\t", "https://app.test"},

		// Default ports are elided by browsers; keeping them would break matching.
		{"default http port is elided", "http://app.test:80", "http://app.test"},
		{"default https port is elided", "https://app.test:443", "https://app.test"},

		// ...but the SAME numbers on the OTHER scheme are not defaults, and
		// dropping them there would merge two distinct origins.
		{"443 on http is not a default", "http://app.test:443", "http://app.test:443"},
		{"80 on https is not a default", "https://app.test:80", "https://app.test:80"},

		// Userinfo must not survive: "https://evil.test@good.test" has host
		// good.test, and echoing the userinfo back would produce an allow-list
		// entry no browser ever sends.
		{"userinfo is dropped", "https://user:pw@app.test", "https://app.test"},
		{"userinfo is dropped and port kept", "https://user@app.test:8443", "https://app.test:8443"},

		// IPv6 literals must come back bracketed or the result is not a
		// syntactically valid origin at all.
		{"ipv6 literal is re-bracketed", "http://[::1]:8080", "http://[::1]:8080"},
		{"ipv6 literal without a port", "https://[2001:db8::1]", "https://[2001:db8::1]"},

		// Everything below is NOT an origin. Each must come back "" so that
		// validateCORSOrigins rejects it and corsOriginPolicy drops it.
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
		{"no scheme", "app.test", ""},
		{"no scheme with port", "app.test:8443", ""},
		{"protocol-relative", "//app.test", ""},
		{"ftp scheme", "ftp://app.test", ""},
		{"file scheme", "file:///etc/passwd", ""},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"data scheme", "data:text/html,<h1>x</h1>", ""},
		{"scheme with no host", "https://", ""},
		{"scheme with only a path", "https:///just/a/path", ""},
		{"unparseable url", "http://exa mple.test", ""},
		{"control character in url", "https://app.test\x7f", ""},
		{"the wildcard is not an origin", "*", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := OriginOf(tc.in)
			if got != tc.want {
				if tc.want == "" {
					t.Fatalf("OriginOf(%q) = %q, want \"\"\n"+
						"A non-empty result here becomes an entry in the CORS allow-list, "+
						"which is reflected back with Access-Control-Allow-Credentials.",
						tc.in, got)
				}
				t.Fatalf("OriginOf(%q) = %q, want %q\n"+
					"This value is compared byte-for-byte against the browser's Origin "+
					"header; any difference means the origin matches nothing (locked out) "+
					"or matches more than intended (bypass).", tc.in, got, tc.want)
			}
		})
	}
}

// TestCfgtOriginOfDistinguishesOrigins states the property the table above only
// implies: origins that differ in scheme, host or port must NOT normalize to the
// same string. This is the assertion that a "simplification" of OriginOf — say,
// returning only the host — has to break.
func TestCfgtOriginOfDistinguishesOrigins(t *testing.T) {
	distinct := []string{
		"https://app.test",
		"http://app.test",
		"https://app.test:8443",
		"http://app.test:8080",
		"https://api.app.test",
		"https://app.test.evil.test",
	}

	seen := make(map[string]string, len(distinct))
	for _, raw := range distinct {
		got := OriginOf(raw)
		if got == "" {
			t.Fatalf("OriginOf(%q) = \"\", but this is a well-formed origin", raw)
		}
		if prev, dup := seen[got]; dup {
			t.Fatalf("OriginOf collapsed two different origins onto %q: %q and %q\n"+
				"Allowing one of these would then allow the other, which is an "+
				"origin bypass — scheme, host and port each identify an origin.",
				got, prev, raw)
		}
		seen[got] = raw
	}
}

// ---------------------------------------------------------------------------
// NormalizeFrontendMode
// ---------------------------------------------------------------------------

func TestCfgtNormalizeFrontendMode(t *testing.T) {
	cases := []struct {
		name    string
		mode    string
		env     string
		want    string
		wantErr bool
	}{
		// Unset must mean "API only", never "guess". A default of embed or proxy
		// would attach a frontend to a deployment that never asked for one.
		{name: "unset defaults to none", mode: "", env: "production", want: FrontendModeNone},
		{name: "whitespace defaults to none", mode: "  \t ", env: "production", want: FrontendModeNone},
		{name: "explicit none", mode: "none", env: "production", want: FrontendModeNone},
		{name: "mixed case none", mode: " NoNe ", env: "production", want: FrontendModeNone},
		{name: "embed", mode: "embed", env: "production", want: FrontendModeEmbed},
		{name: "mixed case embed", mode: "EMBED", env: "production", want: FrontendModeEmbed},

		{name: "proxy in development", mode: "proxy", env: EnvDevelopment, want: FrontendModeProxy},
		{name: "proxy in test", mode: "proxy", env: EnvTest, want: FrontendModeProxy},
		{name: "mixed case proxy in development", mode: " Proxy ", env: " Development ", want: FrontendModeProxy},

		// The whole point of the mode: proxy forwards to a Vite dev server, so it
		// must never survive into anything that is not a development environment.
		{name: "proxy in production", mode: "proxy", env: "production", wantErr: true},
		{name: "proxy with unset env", mode: "proxy", env: "", wantErr: true},
		{name: "proxy in staging", mode: "proxy", env: "staging", wantErr: true},

		{name: "unknown mode", mode: "spa", env: EnvDevelopment, wantErr: true},
		{name: "near-miss mode", mode: "embedded", env: EnvDevelopment, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeFrontendMode(tc.mode, tc.env)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeFrontendMode(%q, env=%q) = %q, want an error; "+
						"an unrecognized or dev-only mode must stop startup rather than "+
						"be silently downgraded", tc.mode, tc.env, got)
				}
				if got != "" {
					t.Fatalf("NormalizeFrontendMode(%q, env=%q) returned mode %q alongside "+
						"an error; a caller that ignores the error would act on it",
						tc.mode, tc.env, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeFrontendMode(%q, env=%q) failed: %v", tc.mode, tc.env, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeFrontendMode(%q, env=%q) = %q, want %q; the returned "+
					"value selects how (and whether) the SPA is served",
					tc.mode, tc.env, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// camelToSnake
// ---------------------------------------------------------------------------

// TestCfgtCamelToSnake pins the acronym handling, which is the only interesting
// part. camelToSnake supplies the log field names for Config.Redacted, so a
// change here silently renames fields in structured logs and breaks whatever
// queries or dashboards read them.
func TestCfgtCamelToSnake(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"Env", "env"},
		{"BindPort", "bind_port"},
		{"BasePath", "base_path"},
		{"alreadysnake", "alreadysnake"},

		// A trailing acronym stays glued together: URL is one word, not U_R_L.
		{"DatabaseURL", "database_url"},
		{"ProvisionerDatabaseURL", "provisioner_database_url"},

		// An acronym followed by a normal word splits before the last letter of
		// the acronym, because that letter starts the next word.
		{"SMTPHost", "smtp_host"},
		{"JWTIssuer", "jwt_issuer"},
		{"JWTAccessTTLMin", "jwt_access_ttl_min"},

		// Digits are not lowercase letters, so no separator is inserted before a
		// capital that follows one... but one IS inserted before a capital that
		// starts a word.
		{"Base64Key", "base64_key"},

		{"A", "a"},
		{"ID", "id"},
		{"OrgsMaxNewPerHour", "orgs_max_new_per_hour"},
		{"AuthSeedPublicDomainRules", "auth_seed_public_domain_rules"},

		// This one is why several struct fields carry an explicit `key` tag. A
		// lowercase plural "s" glued to an acronym is unrecoverable: the rule
		// "split before the last capital of a run when a lowercase letter
		// follows" fires on the R of CIDRs, producing "cid_rs". The answer is
		// wrong, and pinning it documents that the tag is load-bearing rather
		// than decorative — see the companion test below.
		{"TrustedProxyCIDRs", "trusted_proxy_cid_rs"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := camelToSnake(tc.in); got != tc.want {
				t.Fatalf("camelToSnake(%q) = %q, want %q; this is the log field name for "+
					"the corresponding Config field", tc.in, got, tc.want)
			}
		})
	}
}

// TestCfgtCamelToSnakeTaggedFieldsAvoidTheAcronymBug asserts the reason the
// `key` tags exist: every field that carries one is a field camelToSnake would
// have named badly. If someone "fixes" camelToSnake so that a tag becomes
// redundant, this test tells them to delete the tag rather than leaving two
// sources of truth that disagree.
func TestCfgtCamelToSnakeTaggedFieldsAvoidTheAcronymBug(t *testing.T) {
	tagged := map[string]string{
		"OAuthRedirectBase": "oauth_redirect_base",
		"WebAuthnRPID":      "webauthn_rp_id",
		"WebAuthnRPOrigin":  "webauthn_rp_origin",
		"WebAuthnRPName":    "webauthn_rp_name",
		"TrustedProxyCIDRs": "trusted_proxy_cidrs",
	}

	typ := reflect.TypeOf(Config{})
	// The behavioural half: Redacted() must actually CONSULT the tag. Asserting
	// only that the tag is declared on the struct is metadata, not behaviour —
	// deleting the `key` lookup from Redacted() leaves the tags in place and the
	// startup log renaming itself to camelToSnake's wrong answers.
	out := (&Config{}).Redacted()

	for name, wantKey := range tagged {
		f, ok := typ.FieldByName(name)
		if !ok {
			t.Fatalf("Config no longer has a field named %q; update this test alongside "+
				"the struct so the log field names stay pinned", name)
		}
		if got := f.Tag.Get("key"); got != wantKey {
			t.Fatalf("Config.%s has key tag %q, want %q; this is the field name that "+
				"appears in structured logs", name, got, wantKey)
		}
		if camelToSnake(name) == wantKey {
			t.Fatalf("camelToSnake(%q) now produces %q on its own, so the `key` tag on "+
				"Config.%s is redundant — remove the tag rather than keeping two "+
				"definitions of the same log field name", name, wantKey, name)
		}
		if _, ok := out[wantKey]; !ok {
			t.Errorf("Config.Redacted() has no key %q, so the `key` tag on Config.%s is "+
				"declared but not honoured; the startup log field names are part of the "+
				"operational contract", wantKey, name)
		}
		if _, ok := out[camelToSnake(name)]; ok {
			t.Errorf("Config.Redacted() emitted Config.%s under %q instead of its `key` "+
				"tag %q; the tag exists precisely because camelToSnake gets this name "+
				"wrong", name, camelToSnake(name), wantKey)
		}
	}
}

// ---------------------------------------------------------------------------
// Config.Redacted
// ---------------------------------------------------------------------------

// cfgtSecretMarker is a per-field sentinel. Delimiters on both sides keep one
// field's marker from being a substring of another's.
func cfgtSecretMarker(field string) string {
	return "CFGT-SECRET-" + field + "-END"
}

// TestCfgtRedactedNeverLeaksAnyTaggedField is the reason Redacted exists.
//
// It fills EVERY string field of Config with a marker unique to that field,
// renders the whole returned map, and then asserts:
//
//   - no marker for a `show:"redact"` field appears ANYWHERE in the rendering,
//     not merely that the field's own entry says "<redacted>". Checking only the
//     marker would pass if the secret were appended to another value, nested in
//     a slice, or left behind under a second key.
//   - every non-redacted field's marker DOES appear, so "redact everything" is
//     not a way to pass this test. Redacted is used to log the effective config;
//     a version that hides the non-secrets is useless.
//
// Driving the field list off reflection rather than a hand-written list means a
// new secret added to Config without a `show:"redact"` tag is caught by the
// per-field expectations below rather than needing this test to be updated.
func TestCfgtRedactedNeverLeaksAnyTaggedField(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	val := reflect.New(typ).Elem()

	var stringFields []reflect.StructField
	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type.Kind() == reflect.String {
			val.Field(i).SetString(cfgtSecretMarker(f.Name))
			stringFields = append(stringFields, f)
		}
	}

	cfg := val.Interface().(Config)
	out := cfg.Redacted()

	// %#v renders keys and values, including anything nested, so a secret that
	// survives inside some other value is still found.
	rendered := fmt.Sprintf("%#v", out)

	for _, f := range stringFields {
		marker := cfgtSecretMarker(f.Name)
		redacted := f.Tag.Get("show") == "redact"

		if redacted && strings.Contains(rendered, marker) {
			t.Errorf("Config.Redacted() leaked the value of %s, which is tagged "+
				"show:\"redact\".\nrendered output: %s\n"+
				"Redacted() is what gets logged at startup; a leak here writes a live "+
				"credential into the log store, where it survives rotation of the "+
				"process but not of the secret.", f.Name, rendered)
		}
		if !redacted && !strings.Contains(rendered, marker) {
			t.Errorf("Config.Redacted() dropped or masked %s, which is NOT tagged "+
				"show:\"redact\".\nRedacted() exists to show the effective configuration; "+
				"hiding the non-secret fields makes the startup log useless for "+
				"diagnosing a misconfiguration.", f.Name)
		}
	}
}

// TestCfgtRedactedHidesRealisticSecrets is the concrete companion to the
// reflection test above: it uses values shaped like the real ones and searches
// for the secret substrings the task cares about, including the password
// embedded inside a DSN.
func TestCfgtRedactedHidesRealisticSecrets(t *testing.T) {
	const (
		dbPassword = "pgpw-do-not-log-9f2c"
		masterKey  = "bWFzdGVyLWtleS1kby1ub3QtbG9nLTAwMDAwMDAwMA=="
		jwtSecret  = "jwt-signing-secret-do-not-log"
		smtpPass   = "smtp-password-do-not-log"
		oauthPass  = "oauth-client-secret-do-not-log"
		// SMTP_USER is tagged show:"redact" too — it is commonly an API-key
		// identifier (SES/SendGrid), not a human name. Without a concrete value
		// here, dropping its tag would be invisible: the reflective test derives
		// its expectation from the very tag under test, so an untagged field is
		// simply expected to appear.
		smtpUser = "smtp-user-do-not-log"
	)

	cfg := cfgtValidConfig()
	cfg.DatabaseURL = "postgres://apiary:" + dbPassword + "@db.internal:5432/apiary?sslmode=require"
	cfg.ProvisionerDatabaseURL = "postgres://provisioner:" + dbPassword + "@db.internal:5432/apiary"
	cfg.EncryptionMasterKey = masterKey
	cfg.JWTSecret = jwtSecret
	cfg.SMTPPassword = smtpPass
	cfg.SMTPUser = smtpUser
	cfg.GoogleClientSecret = oauthPass
	cfg.GithubClientSecret = oauthPass

	rendered := fmt.Sprintf("%#v", cfg.Redacted())

	for _, secret := range []string{dbPassword, masterKey, jwtSecret, smtpPass, smtpUser, oauthPass} {
		if strings.Contains(rendered, secret) {
			t.Errorf("Config.Redacted() output contains the secret %q.\n"+
				"full output: %s\n"+
				"Anything Redacted() returns is written to the startup log; a credential "+
				"there has to be rotated, not deleted.", secret, rendered)
		}
	}

	// The non-secret half must still be legible, otherwise the startup log tells
	// an operator nothing about how the process was configured.
	for key, want := range map[string]any{
		"base_url":  "https://apiary.example.com",
		"bind_port": "8080",
		"env":       "production",
	} {
		if got := cfgtOut(t, cfg.Redacted(), key); got != want {
			t.Errorf("Config.Redacted()[%q] = %v, want %v; non-secret settings must "+
				"survive redaction or the log cannot be used to diagnose a "+
				"misconfiguration", key, got, want)
		}
	}
}

// out fetches a key from a Redacted() map and fails loudly if it is missing,
// so a renamed key surfaces as "key absent" rather than as a nil comparison.
func cfgtOut(t *testing.T, m map[string]any, key string) any {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("Config.Redacted() has no key %q; the startup log field names are "+
			"part of the operational contract", key)
	}
	return v
}

// TestCfgtRedactedDistinguishesUnsetFromSet pins a subtle and deliberate
// behaviour: an EMPTY secret stays empty rather than becoming "<redacted>".
//
// That distinction is the only way a startup log can answer "is GITHUB_CLIENT_SECRET
// actually configured?". If every secret rendered as "<redacted>" regardless,
// an operator debugging a broken OAuth login would see a fully configured
// process and go looking somewhere else.
func TestCfgtRedactedDistinguishesUnsetFromSet(t *testing.T) {
	var empty Config
	emptyOut := empty.Redacted()

	for _, key := range []string{"jwt_secret", "encryption_master_key", "smtp_password", "database_url"} {
		if got := cfgtOut(t, emptyOut, key); got != "" {
			t.Errorf("Config.Redacted()[%q] = %v for an UNSET secret, want \"\"; "+
				"marking an unset secret as %q makes a missing credential look "+
				"configured in the startup log", key, got, "<redacted>")
		}
	}

	set := Config{JWTSecret: "x", EncryptionMasterKey: "y", SMTPPassword: "z", DatabaseURL: "d"}
	setOut := set.Redacted()
	for _, key := range []string{"jwt_secret", "encryption_master_key", "smtp_password", "database_url"} {
		if got := cfgtOut(t, setOut, key); got != "<redacted>" {
			t.Errorf("Config.Redacted()[%q] = %v for a SET secret, want %q", key, got, "<redacted>")
		}
	}
}

// TestCfgtRedactedEmitsEveryFieldExactlyOnce guards the key-derivation half of
// Redacted. Two fields colliding on one key would silently drop one of them
// from the startup log — and if a redacted field lost to a non-redacted one,
// or vice versa, the surviving value could be the wrong one.
func TestCfgtRedactedEmitsEveryFieldExactlyOnce(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	got := (&Config{}).Redacted()

	if len(got) != typ.NumField() {
		t.Fatalf("Config.Redacted() returned %d keys for a struct with %d fields; "+
			"two fields derive the same log key and one of them is being silently "+
			"discarded", len(got), typ.NumField())
	}

	// Non-string fields must come through with their real values and types, not
	// stringified — structured log backends index them as numbers and booleans.
	typed := (&Config{JWTAccessTTLMin: 15, AdmissionCohortProvenRatio: 0.25, AuthSocialLinkByEmail: true}).Redacted()
	for key, want := range map[string]any{
		"jwt_access_ttl_min":                 15,
		"admission_cohort_proven_ratio":      0.25,
		"auth_social_link_by_email":          true,
		"admission_unproven_orgs_per_domain": 0,
	} {
		if v := cfgtOut(t, typed, key); v != want {
			t.Errorf("Config.Redacted()[%q] = %#v, want %#v (same type); redaction must "+
				"not change the type of a non-secret field", key, v, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Config.ValidateWorker
// ---------------------------------------------------------------------------

// TestCfgtValidateWorkerRequiresASeparateProvisionerDSN covers the extra rule
// the worker adds on top of Validate.
//
// PROVISIONER_DATABASE_URL is a role holding CREATEDB + CREATEROLE, used only
// by tenant provisioning and destruction. The whole point of it being a
// separate DSN is that the request-path role must NOT hold those privileges: if
// the two are the same, an SQL injection anywhere in the request path inherits
// the ability to create and drop databases and roles. Outside development that
// is a startup error, not a warning.
func TestCfgtValidateWorkerRequiresASeparateProvisionerDSN(t *testing.T) {
	cases := []struct {
		name         string
		mutate       func(*Config)
		wantErr      bool
		wantErrHas   string
		wantFellBack bool
	}{
		{
			name: "distinct provisioner dsn in production",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "postgres://provisioner:pw@db:5432/apiary"
			},
		},
		{
			name: "identical dsn in production is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = c.DatabaseURL
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			// Whitespace must not be a way around the rule: "  dsn " and "dsn" are
			// the same connection string, and a config file with a stray space
			// would otherwise pass a check it does not actually satisfy.
			name: "identical dsn differing only by whitespace is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "  " + c.DatabaseURL + "\t"
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			// The mirror image, and it is NOT redundant: the comparison trims both
			// sides, and only the row above pins the provisioner side. With the
			// whitespace on DATABASE_URL instead, dropping the TrimSpace around
			// c.DatabaseURL makes the two strings unequal and the shared-DSN check
			// silently stops firing.
			name: "identical dsn where DATABASE_URL carries the whitespace is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = c.DatabaseURL
				c.DatabaseURL = "  " + c.DatabaseURL + "\t"
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			name: "unset provisioner dsn in production is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = ""
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			name: "whitespace-only provisioner dsn in production is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "   "
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			// Development convenience: unset falls back to DATABASE_URL with a
			// warning, so `make dev` works without provisioning a second role.
			name: "unset provisioner dsn falls back to DATABASE_URL in development",
			mutate: func(c *Config) {
				c.Env = EnvDevelopment
				c.ProvisionerDatabaseURL = ""
			},
			wantFellBack: true,
		},
		{
			name: "identical dsn is tolerated in development",
			mutate: func(c *Config) {
				c.Env = EnvTest
				c.ProvisionerDatabaseURL = c.DatabaseURL
			},
		},
		{
			// The dev fallback needs something to fall back TO. With both unset
			// there is no DSN at all, and the worker must not start.
			name: "both dsns unset in development is still refused",
			mutate: func(c *Config) {
				c.Env = EnvDevelopment
				c.DatabaseURL = ""
				c.ProvisionerDatabaseURL = ""
			},
			wantErr: true, wantErrHas: "PROVISIONER_DATABASE_URL",
		},
		{
			// The startup gate, not just the helper: a prefix PostgreSQL would
			// mangle stops the worker before it names a tenant with it.
			name: "an invalid database prefix is refused",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "postgres://provisioner:pw@db:5432/apiary"
				c.DatabasePrefix = "Apiary"
			},
			wantErr: true, wantErrHas: "DATABASE_PREFIX",
		},
		{
			name: "a valid database prefix is accepted",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "postgres://provisioner:pw@db:5432/apiary"
				c.DatabasePrefix = "apiary_"
			},
		},
		{
			// ValidateWorker runs Validate first; a broken base config must fail
			// here too rather than being skipped because the worker has its own path.
			name: "base config errors still propagate",
			mutate: func(c *Config) {
				c.ProvisionerDatabaseURL = "postgres://provisioner:pw@db:5432/apiary"
				c.JWTSecret = ""
			},
			wantErr: true, wantErrHas: "JWT_SECRET",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := cfgtValidConfig()
			tc.mutate(cfg)
			originalDB := cfg.DatabaseURL

			err := cfg.ValidateWorker()

			if tc.wantErr {
				if err == nil {
					t.Fatalf("ValidateWorker accepted the config; the provisioner role holds " +
						"CREATEDB and CREATEROLE, so letting the worker start with a bad or " +
						"shared DSN hands those privileges to the request path")
				}
				if !strings.Contains(err.Error(), tc.wantErrHas) {
					t.Fatalf("ValidateWorker failed but blamed the wrong setting:\n  got:  %v\n"+
						"  want: an error naming %q", err, tc.wantErrHas)
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateWorker refused a valid worker config: %v", err)
			}
			if tc.wantFellBack && cfg.ProvisionerDatabaseURL != originalDB {
				t.Fatalf("ValidateWorker did not apply the development fallback: "+
					"ProvisionerDatabaseURL = %q, want it copied from DatabaseURL (%q).\n"+
					"The fallback is a mutation of the config the worker then uses; "+
					"without it the provisioner connects with an empty DSN.",
					cfg.ProvisionerDatabaseURL, originalDB)
			}
		})
	}
}

// TestCfgtValidateWorkerDoesNotFallBackOutsideDevelopment states the negative
// half of the fallback explicitly: in production, an unset PROVISIONER_DATABASE_URL
// must NOT be quietly filled in from DATABASE_URL. If the fallback ever loses
// its IsDevEnv guard, the same-DSN check that follows would then be comparing a
// value the code itself just assigned — and every production worker would run
// tenant provisioning as the request-path role.
func TestCfgtValidateWorkerDoesNotFallBackOutsideDevelopment(t *testing.T) {
	for _, env := range []string{"production", "staging", ""} {
		t.Run("env="+env, func(t *testing.T) {
			cfg := cfgtValidConfig()
			cfg.Env = env
			cfg.ProvisionerDatabaseURL = ""

			err := cfg.ValidateWorker()
			if err == nil {
				t.Fatalf("ValidateWorker accepted an unset PROVISIONER_DATABASE_URL with ENV=%q", env)
			}
			if cfg.ProvisionerDatabaseURL != "" {
				t.Fatalf("ValidateWorker copied DATABASE_URL into ProvisionerDatabaseURL with "+
					"ENV=%q (got %q); the dev-only fallback must not run outside development",
					env, cfg.ProvisionerDatabaseURL)
			}
		})
	}
}
