package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/glueops/apiary/internal/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	EnvDevelopment = "development"
	EnvTest        = "test"
)

const (
	FrontendModeNone  = "none"
	FrontendModeEmbed = "embed"
	FrontendModeProxy = "proxy"
)

func NormalizeEnv(env string) string {
	return strings.ToLower(strings.TrimSpace(env))
}

func IsDevEnv(env string) bool {
	switch NormalizeEnv(env) {
	case EnvDevelopment, EnvTest:
		return true
	default:
		return false
	}
}

func NormalizeFrontendMode(mode, env string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", FrontendModeNone:
		return FrontendModeNone, nil
	case FrontendModeEmbed:
		return FrontendModeEmbed, nil
	case FrontendModeProxy:
		if !IsDevEnv(env) {
			return "", fmt.Errorf(
				"config: FRONTEND_MODE=%s is development-only (ENV=%q); use %q to serve the "+
					"SPA compiled into the binary, or %q to serve the API alone",
				FrontendModeProxy, NormalizeEnv(env), FrontendModeEmbed, FrontendModeNone,
			)
		}
		return FrontendModeProxy, nil
	default:
		return "", fmt.Errorf(
			"config: FRONTEND_MODE=%q is not one of %q, %q, %q",
			mode, FrontendModeNone, FrontendModeEmbed, FrontendModeProxy,
		)
	}
}

func OriginOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if strings.Contains(host, ":") {

		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host
}

type Config struct {
	DatabaseURL          string `show:"redact"`
	BindPort             string
	BindHost             string
	Env                  string
	EncryptionMasterKey  string `show:"redact"`
	BasePath             string
	BaseURL              string
	FrontendMode         string
	ViteDevURL           string
	JWTSecret            string `show:"redact"`
	JWTIssuer            string
	JWTAccessTTLMin      int
	JWTRefreshTTLHour    int
	OAuthRedirectBase    string `key:"oauth_redirect_base"`
	GoogleClientID       string
	GoogleClientSecret   string `show:"redact"`
	GithubClientID       string
	GithubClientSecret   string `show:"redact"`
	KeycloakClientID     string
	KeycloakClientSecret string `show:"redact"`
	// Unlike Google's, a self-hosted issuer is not a constant: it carries the
	// deployment's host, port and realm, so discovery has nowhere to look
	// without it.
	KeycloakIssuerURL  string
	WebAuthnRPID       string `key:"webauthn_rp_id"`
	WebAuthnRPOrigin   string `key:"webauthn_rp_origin"`
	WebAuthnRPName     string `key:"webauthn_rp_name"`
	CORSAllowedOrigins string `key:"cors_allowed_origins"`

	TrustedProxyCIDRs string `key:"trusted_proxy_cidrs"`
	SMTPHost          string
	SMTPPort          int
	SMTPUser          string `show:"redact"`
	SMTPPassword      string `show:"redact"`
	SMTPFrom          string

	ProvisionerDatabaseURL string `show:"redact"`
	// DatabasePrefix names each tenant provisioned from now on: its database is
	// <prefix>_<org uuid> and its role <prefix>_role_<org uuid>. Unset, it is
	// "tenant". Existing tenants keep the name they were given, so it is safe to
	// change.
	DatabasePrefix string

	TenantPoolMaxSize            int
	TenantPoolMaxConns           int
	TenantPoolMaxIdleConns       int
	TenantPoolConnMaxIdleMin     int
	TenantPoolConnMaxLifetimeMin int
	TenantPoolEvictGraceSec      int

	OrgsMaxTotal      int
	OrgsMaxNewPerHour int

	AuthSocialLinkByEmail     bool
	AuthSeedPublicDomainRules bool

	AdmissionUnprovenOrgsPerDomain int
	AdmissionProvenDistinctDays    int
	AdmissionProvingIntervalHours  int
	AdmissionCohortIntervalHours   int
	AdmissionCohortMinSample       int
	AdmissionCohortProvenRatio     float64

	// FleetObservationMaxAge is how stale a capacity sync may be and still support
	// a decision that a hypervisor is empty.
	//
	// It gates a destructive step: a departure acts on "this node has no guests" by
	// dismantling the node's cluster membership. Past this age the drain state is
	// reported as unknown and the departure is refused rather than guessed at.
	FleetObservationMaxAge time.Duration

	// FleetCapacitySyncInterval is how often every hive's capacity is refreshed.
	//
	// It must be comfortably SHORTER than FleetObservationMaxAge. The drain state
	// reads an observation older than that max age as unknown and refuses the
	// departure, so a sync interval at or above it makes every departure refuse
	// most of the time — the gate working exactly as designed against a fleet
	// nobody is observing often enough.
	FleetCapacitySyncInterval time.Duration

	// FleetOnboardingTickInterval is how often pending machines are claimed and
	// handed to an onboarding worker.
	//
	// It is the LATENCY an operator sees between asking for a machine to be
	// onboarded and anything happening, so it is short. It costs one query per
	// organization per tick and nothing else when there is no work — the tick
	// claims and enqueues, and does no SSH of its own.
	FleetOnboardingTickInterval time.Duration
}

func (c *Config) Redacted() map[string]any {
	out := make(map[string]any)
	v := reflect.ValueOf(*c)
	t := v.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		key := f.Tag.Get("key")
		if key == "" {
			key = camelToSnake(f.Name)
		}
		val := v.Field(i).Interface()

		if f.Tag.Get("show") == "redact" {
			switch s := val.(type) {
			case string:
				if s != "" {
					val = "<redacted>"
				}
			}
		}
		out[key] = val
	}
	return out
}

func camelToSnake(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]

				if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var (
	once    sync.Once
	cached  *Config
	loadErr error
)

func Load() (*Config, error) {
	once.Do(func() {
		_ = godotenv.Load()

		v := viper.New()

		v.SetDefault("database.url", "postgres://user:pass@localhost:5432/db?sslmode=prefer")
		v.SetDefault("bind.port", "8080")
		v.SetDefault("bind.host", "127.0.0.1")

		v.SetDefault("env", "")
		v.SetDefault("encryption.master.key", "")
		v.SetDefault("base.path", "/api/v1")
		v.SetDefault("base.url", "http://localhost:8080")
		v.SetDefault("frontend.mode", FrontendModeNone)
		v.SetDefault("vite.dev.url", "http://localhost:5173")
		v.SetDefault("jwt.issuer", "apiary")
		v.SetDefault("jwt.access_ttl_min", 15)
		v.SetDefault("jwt.refresh_ttl_hour", 168)

		v.SetDefault("cors.allowed.origins", "")
		v.SetDefault("trusted.proxy.cidrs", "")
		v.SetDefault("smtp.host", "localhost")
		v.SetDefault("smtp.port", 1025)
		v.SetDefault("smtp.user", "")
		v.SetDefault("smtp.password", "")
		v.SetDefault("smtp.from", "noreply@example.com")

		v.SetDefault("provisioner.database.url", "")
		v.SetDefault("database.prefix", utils.DefaultTenantDatabasePrefix)
		v.SetDefault("tenant.pool.max_size", 64)
		v.SetDefault("tenant.pool.max_conns", 4)
		v.SetDefault("tenant.pool.max_idle_conns", 2)
		v.SetDefault("tenant.pool.conn_max_idle_min", 5)
		v.SetDefault("tenant.pool.conn_max_lifetime_min", 60)
		v.SetDefault("tenant.pool.evict_grace_sec", 60)
		v.SetDefault("orgs.max_total", 1000)

		v.SetDefault("orgs.max_new_per_hour", 200)
		v.SetDefault("auth.social.link_by_email", false)
		v.SetDefault("auth.seed_public_domain_rules", true)

		v.SetDefault("admission.unproven_orgs_per_domain", 3)
		v.SetDefault("admission.proven_distinct_days", 2)
		v.SetDefault("admission.proving_interval_hours", 24)
		v.SetDefault("admission.cohort_interval_hours", 24)
		v.SetDefault("admission.cohort_min_sample", 5)
		v.SetDefault("fleet.observation_max_age", "10m")
		v.SetDefault("fleet.capacity_sync_interval", "3m")
		v.SetDefault("fleet.onboarding_tick_interval", "30s")
		v.SetDefault("admission.cohort_proven_ratio", 0.25)

		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		keys := []string{
			"database.url",
			"bind.port",
			"bind.host",
			"env",
			"encryption.master.key",
			"base.path",
			"base.url",
			"frontend.mode",
			"vite.dev.url",
			"jwt.secret",
			"jwt.issuer",
			"jwt.access_ttl_min",
			"jwt.refresh_ttl_hour",
			"oauth.redirect.base",
			"google.client.id",
			"google.client.secret",
			"github.client.id",
			"github.client.secret",
			"keycloak.client.id",
			"keycloak.client.secret",
			"keycloak.issuer.url",
			"webauthn.rp.id",
			"webauthn.rp.origin",
			"webauthn.rp.name",
			"cors.allowed.origins",
			"trusted.proxy.cidrs",
			"smtp.host",
			"smtp.port",
			"smtp.user",
			"smtp.password",
			"smtp.from",
			"provisioner.database.url",
			"database.prefix",
			"tenant.pool.max_size",
			"tenant.pool.max_conns",
			"tenant.pool.max_idle_conns",
			"tenant.pool.conn_max_idle_min",
			"tenant.pool.conn_max_lifetime_min",
			"tenant.pool.evict_grace_sec",
			"orgs.max_total",
			"orgs.max_new_per_hour",
			"auth.social.link_by_email",
			"auth.seed_public_domain_rules",
			"admission.unproven_orgs_per_domain",
			"admission.proven_distinct_days",
			"admission.proving_interval_hours",
			"admission.cohort_interval_hours",
			"admission.cohort_min_sample",
			"fleet.observation_max_age",
			"fleet.capacity_sync_interval",
			"fleet.onboarding_tick_interval",
			"admission.cohort_proven_ratio",
		}
		for _, key := range keys {
			_ = v.BindEnv(key)
		}

		cfg := Config{
			DatabaseURL:          v.GetString("database.url"),
			BindPort:             v.GetString("bind.port"),
			BindHost:             v.GetString("bind.host"),
			Env:                  v.GetString("env"),
			EncryptionMasterKey:  v.GetString("encryption.master.key"),
			BasePath:             v.GetString("base.path"),
			BaseURL:              v.GetString("base.url"),
			FrontendMode:         v.GetString("frontend.mode"),
			ViteDevURL:           v.GetString("vite.dev.url"),
			JWTSecret:            v.GetString("jwt.secret"),
			JWTIssuer:            v.GetString("jwt.issuer"),
			JWTAccessTTLMin:      v.GetInt("jwt.access_ttl_min"),
			JWTRefreshTTLHour:    v.GetInt("jwt.refresh_ttl_hour"),
			OAuthRedirectBase:    v.GetString("oauth.redirect.base"),
			GoogleClientID:       v.GetString("google.client.id"),
			GoogleClientSecret:   v.GetString("google.client.secret"),
			GithubClientID:       v.GetString("github.client.id"),
			GithubClientSecret:   v.GetString("github.client.secret"),
			KeycloakClientID:     v.GetString("keycloak.client.id"),
			KeycloakClientSecret: v.GetString("keycloak.client.secret"),
			KeycloakIssuerURL:    v.GetString("keycloak.issuer.url"),
			WebAuthnRPID:         v.GetString("webauthn.rp.id"),
			WebAuthnRPOrigin:     v.GetString("webauthn.rp.origin"),
			WebAuthnRPName:       v.GetString("webauthn.rp.name"),
			CORSAllowedOrigins:   v.GetString("cors.allowed.origins"),
			TrustedProxyCIDRs:    v.GetString("trusted.proxy.cidrs"),
			SMTPHost:             v.GetString("smtp.host"),
			SMTPPort:             v.GetInt("smtp.port"),
			SMTPUser:             v.GetString("smtp.user"),
			SMTPPassword:         v.GetString("smtp.password"),
			SMTPFrom:             v.GetString("smtp.from"),

			ProvisionerDatabaseURL: v.GetString("provisioner.database.url"),
			DatabasePrefix:         v.GetString("database.prefix"),

			TenantPoolMaxSize:            v.GetInt("tenant.pool.max_size"),
			TenantPoolMaxConns:           v.GetInt("tenant.pool.max_conns"),
			TenantPoolMaxIdleConns:       v.GetInt("tenant.pool.max_idle_conns"),
			TenantPoolConnMaxIdleMin:     v.GetInt("tenant.pool.conn_max_idle_min"),
			TenantPoolConnMaxLifetimeMin: v.GetInt("tenant.pool.conn_max_lifetime_min"),
			TenantPoolEvictGraceSec:      v.GetInt("tenant.pool.evict_grace_sec"),

			OrgsMaxTotal:      v.GetInt("orgs.max_total"),
			OrgsMaxNewPerHour: v.GetInt("orgs.max_new_per_hour"),

			AuthSocialLinkByEmail:     v.GetBool("auth.social.link_by_email"),
			AuthSeedPublicDomainRules: v.GetBool("auth.seed_public_domain_rules"),

			AdmissionUnprovenOrgsPerDomain: v.GetInt("admission.unproven_orgs_per_domain"),
			AdmissionProvenDistinctDays:    v.GetInt("admission.proven_distinct_days"),
			AdmissionProvingIntervalHours:  v.GetInt("admission.proving_interval_hours"),
			AdmissionCohortIntervalHours:   v.GetInt("admission.cohort_interval_hours"),
			AdmissionCohortMinSample:       v.GetInt("admission.cohort_min_sample"),
			FleetObservationMaxAge:         v.GetDuration("fleet.observation_max_age"),
			FleetCapacitySyncInterval:      v.GetDuration("fleet.capacity_sync_interval"),
			FleetOnboardingTickInterval:    v.GetDuration("fleet.onboarding_tick_interval"),
			AdmissionCohortProvenRatio:     v.GetFloat64("admission.cohort_proven_ratio"),
		}

		cached = &cfg
	})
	return cached, loadErr
}

func validateMasterKey(key string) error {
	if _, err := utils.MasterKeyFromBase64(key); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("config: JWT_SECRET must be set")
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		return fmt.Errorf("config: JWT_ISSUER must be set")
	}

	if err := validateMasterKey(c.EncryptionMasterKey); err != nil {
		return err
	}
	if c.JWTAccessTTLMin <= 0 {
		return fmt.Errorf("config: JWT_ACCESS_TTL_MIN must be > 0")
	}
	if c.JWTRefreshTTLHour <= 0 {
		return fmt.Errorf("config: JWT_REFRESH_TTL_HOUR must be > 0")
	}

	if _, err := NormalizeFrontendMode(c.FrontendMode, c.Env); err != nil {
		return err
	}

	if err := validateCORSOrigins(c.CORSAllowedOrigins, c.BaseURL, c.Env); err != nil {
		return err
	}

	if c.TenantPoolMaxSize < 1 {
		return fmt.Errorf("config: TENANT_POOL_MAX_SIZE must be >= 1, got %d", c.TenantPoolMaxSize)
	}

	if c.TenantPoolMaxConns < 1 {
		return fmt.Errorf(
			"config: TENANT_POOL_MAX_CONNS must be >= 1, got %d (0 would mean unlimited backends per tenant)",
			c.TenantPoolMaxConns,
		)
	}
	if c.TenantPoolMaxIdleConns < 0 {
		return fmt.Errorf("config: TENANT_POOL_MAX_IDLE_CONNS must be >= 0, got %d", c.TenantPoolMaxIdleConns)
	}
	if c.TenantPoolMaxIdleConns > c.TenantPoolMaxConns {
		return fmt.Errorf(
			"config: TENANT_POOL_MAX_IDLE_CONNS (%d) must be <= TENANT_POOL_MAX_CONNS (%d)",
			c.TenantPoolMaxIdleConns, c.TenantPoolMaxConns,
		)
	}
	if c.TenantPoolConnMaxIdleMin < 0 {
		return fmt.Errorf("config: TENANT_POOL_CONN_MAX_IDLE_MIN must be >= 0, got %d", c.TenantPoolConnMaxIdleMin)
	}
	if c.TenantPoolConnMaxLifetimeMin < 0 {
		return fmt.Errorf(
			"config: TENANT_POOL_CONN_MAX_LIFETIME_MIN must be >= 0, got %d",
			c.TenantPoolConnMaxLifetimeMin,
		)
	}

	if c.TenantPoolEvictGraceSec < 0 {
		return fmt.Errorf("config: TENANT_POOL_EVICT_GRACE_SEC must be >= 0, got %d", c.TenantPoolEvictGraceSec)
	}

	if c.OrgsMaxTotal < 1 {
		return fmt.Errorf("config: ORGS_MAX_TOTAL must be >= 1, got %d", c.OrgsMaxTotal)
	}
	if c.OrgsMaxNewPerHour < 1 {
		return fmt.Errorf("config: ORGS_MAX_NEW_PER_HOUR must be >= 1, got %d", c.OrgsMaxNewPerHour)
	}

	if c.AdmissionUnprovenOrgsPerDomain < 1 {
		return fmt.Errorf(
			"config: ADMISSION_UNPROVEN_ORGS_PER_DOMAIN must be >= 1, got %d (0 would refuse every "+
				"organization, including the first one a legitimate domain creates)",
			c.AdmissionUnprovenOrgsPerDomain,
		)
	}
	if c.AdmissionProvenDistinctDays < 2 {
		return fmt.Errorf(
			"config: ADMISSION_PROVEN_DISTINCT_DAYS must be >= 2, got %d (an organization's first token "+
				"session is issued during registration, so 1 proves every organization the next day and "+
				"makes the unproven-organization ceiling inert)",
			c.AdmissionProvenDistinctDays,
		)
	}
	if c.AdmissionProvingIntervalHours < 1 {
		return fmt.Errorf(
			"config: ADMISSION_PROVING_INTERVAL_HOURS must be >= 1, got %d",
			c.AdmissionProvingIntervalHours,
		)
	}
	if c.AdmissionProvingIntervalHours >= c.JWTRefreshTTLHour {
		slog.Warn(
			"ADMISSION_PROVING_INTERVAL_HOURS is not below JWT_REFRESH_TTL_HOUR; the proving pass reads "+
				"surviving token_sessions rows and TokenSessionCleanupWorker deletes them once both "+
				"tokens have expired, so a whole day of proving evidence can be lost between runs",
			"admission_proving_interval_hours", c.AdmissionProvingIntervalHours,
			"jwt_refresh_ttl_hour", c.JWTRefreshTTLHour,
		)
	}
	if c.AdmissionCohortIntervalHours < 1 {
		return fmt.Errorf(
			"config: ADMISSION_COHORT_INTERVAL_HOURS must be >= 1, got %d",
			c.AdmissionCohortIntervalHours,
		)
	}
	if c.AdmissionCohortMinSample < 1 {
		return fmt.Errorf(
			"config: ADMISSION_COHORT_MIN_SAMPLE must be >= 1, got %d",
			c.AdmissionCohortMinSample,
		)
	}
	if c.AdmissionCohortProvenRatio <= 0 || c.AdmissionCohortProvenRatio >= 1 {
		return fmt.Errorf(
			"config: ADMISSION_COHORT_PROVEN_RATIO must be > 0 and < 1, got %v (0 flags no domain and "+
				"1 flags every domain, and both read as a broken audit rather than as a setting)",
			c.AdmissionCohortProvenRatio,
		)
	}
	return nil
}

func validateCORSOrigins(raw, baseURL, env string) error {
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if entry == "*" {
			if IsDevEnv(env) {
				continue
			}
			return fmt.Errorf(
				"config: CORS_ALLOWED_ORIGINS=%q is development-only (ENV=%q); "+
					"list the exact scheme://host[:port] origins that may make "+
					"credentialed cross-origin requests",
				entry, NormalizeEnv(env),
			)
		}
		if OriginOf(entry) == "" {
			return fmt.Errorf(
				"config: CORS_ALLOWED_ORIGINS entry %q is not an absolute http(s) origin "+
					"(want scheme://host[:port]); it would match no browser Origin header",
				entry,
			)
		}
	}

	if strings.TrimSpace(raw) == "" && !IsDevEnv(env) && OriginOf(baseURL) == "" {
		return fmt.Errorf(
			"config: CORS_ALLOWED_ORIGINS is unset and BASE_URL=%q is not an absolute "+
				"http(s) URL, so no origin could make a cross-origin request",
			baseURL,
		)
	}
	return nil
}

func (c *Config) ValidateWorker() error {
	if err := c.Validate(); err != nil {
		return err
	}

	if err := checkFleetSyncCadence(c); err != nil {
		return err
	}
	if err := checkDatabasePrefix(c.DatabasePrefix); err != nil {
		return err
	}
	provisioner := strings.TrimSpace(c.ProvisionerDatabaseURL)
	if provisioner == "" && IsDevEnv(c.Env) {
		slog.Warn("PROVISIONER_DATABASE_URL is unset; falling back to DATABASE_URL because " +
			"ENV is a development environment. Set it to a dedicated CREATEDB+CREATEROLE role " +
			"before deploying: outside development this is a startup error.")
		c.ProvisionerDatabaseURL = c.DatabaseURL
		provisioner = strings.TrimSpace(c.DatabaseURL)
	}
	if provisioner == "" {
		return fmt.Errorf(
			"config: PROVISIONER_DATABASE_URL must be set for the worker: a DSN for a role holding " +
				"exactly CREATEDB + CREATEROLE, used only by tenant provisioning and destruction",
		)
	}
	if provisioner == strings.TrimSpace(c.DatabaseURL) && !IsDevEnv(c.Env) {
		return fmt.Errorf(
			"config: PROVISIONER_DATABASE_URL must not be the same DSN as DATABASE_URL (ENV=%q); "+
				"the request-path role must not hold CREATEDB or CREATEROLE",
			NormalizeEnv(c.Env),
		)
	}
	return nil
}

// maxDatabasePrefixLen keeps the longest name built from the prefix — the role,
// <prefix>_role_<32 hex> — inside PostgreSQL's 63-byte identifier limit.
// PostgreSQL does not refuse a longer name, it truncates it, and truncation cuts
// into the org uuid: two tenants would then be asked for the same database.
const maxDatabasePrefixLen = 63 - len("_role_") - 32

var databasePrefixPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// checkDatabasePrefix refuses a tenant name prefix PostgreSQL would mangle.
//
// Lowercase only: the names are quoted when created, so an uppercase prefix
// would make a database every unquoted psql reference fails to find.
func checkDatabasePrefix(prefix string) error {
	p := utils.TenantDatabasePrefix(prefix)
	if !databasePrefixPattern.MatchString(p) {
		return fmt.Errorf("config: DATABASE_PREFIX %q must be lowercase letters, digits and underscores, starting with a letter", prefix)
	}
	if len(p) > maxDatabasePrefixLen {
		return fmt.Errorf("config: DATABASE_PREFIX %q is %d characters; at most %d fit in a PostgreSQL identifier alongside the org id", prefix, len(p), maxDatabasePrefixLen)
	}
	return nil
}

// checkFleetSyncCadence refuses a sync interval that cannot keep observations
// fresh.
//
// Extracted so it is testable without assembling a whole valid Config — the
// failure it prevents is subtle enough to deserve a test, and it is not a
// crash: it is a fleet where every hypervisor departure refuses with
// drain_unknown, which reads as a broken gate rather than a scheduling mistake.
//
// Half, not merely less: one missed sync must not be enough to expire every
// observation, because a single failed API call is an ordinary event.
func checkFleetSyncCadence(c *Config) error {
	if c.FleetCapacitySyncInterval <= 0 || c.FleetObservationMaxAge <= 0 {
		return nil
	}
	if c.FleetCapacitySyncInterval*2 > c.FleetObservationMaxAge {
		return fmt.Errorf(
			"config: fleet.capacity_sync_interval (%s) must be at most half of "+
				"fleet.observation_max_age (%s); otherwise a single missed sync expires every "+
				"observation and every hypervisor departure refuses with drain_unknown",
			c.FleetCapacitySyncInterval, c.FleetObservationMaxAge)
	}
	return nil
}
