package config

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	BaseURL        string        `mapstructure:"base_url"`
	StatePath      string        `mapstructure:"state_path"`
	PollMin        time.Duration `mapstructure:"poll_min"`
	PollMax        time.Duration `mapstructure:"poll_max"`
	HeartbeatEvery time.Duration `mapstructure:"heartbeat_every"`
	CommandTimeout time.Duration `mapstructure:"command_timeout"`

	// The local hypervisor. Read from AGENT_* environment or the config file.
	NodeName         string `mapstructure:"node_name"`
	ProxmoxURL       string `mapstructure:"proxmox_url"`
	ProxmoxTokenID   string `mapstructure:"proxmox_token_id"`
	ProxmoxSecret    string `mapstructure:"proxmox_secret"`
	JoinPasswordPath string `mapstructure:"join_password_path"`
}

func Load(configPath, baseURLOverride, statePathOverride string) (Config, error) {
	v := viper.New()

	v.SetEnvPrefix("AGENT")
	v.AutomaticEnv()

	v.SetDefault("base_url", "http://localhost:8080")
	v.SetDefault("state_path", "/var/lib/apiary-agent/state.db")
	v.SetDefault("poll_min", 2*time.Second)
	v.SetDefault("poll_max", 10*time.Second)
	v.SetDefault("heartbeat_every", 15*time.Second)
	v.SetDefault("command_timeout", 5*time.Minute)
	v.SetDefault("proxmox_url", "https://127.0.0.1:8006/api2/json")
	v.SetDefault("join_password_path", "/etc/apiary-agent/cluster-join.secret")

	// The environment file the agent's own `install` writes, and the one systemd
	// loads for the service. Read FIRST so a command run from a shell sees the
	// same settings the service does.
	//
	// Without it every subcommand run by hand silently fell back to
	// http://localhost:8080: AutomaticEnv reads the PROCESS environment, and a
	// shell has none of these — AGENT_BASE_URL lives in a file that only systemd
	// loads. `apiary-agent upgrade` on a correctly configured host therefore
	// dialled the host itself and reported connection refused, with the real
	// setting sitting in /etc/apiary-agent/agent.env the whole time.
	//
	// It does not override anything already exported: a variable set in the
	// caller's environment is more specific than a file, and someone who exports
	// AGENT_BASE_URL to point at a different control plane means it.
	loadEnvFile(defaultEnvFile)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
				return Config{}, err
			}
		}
	} else {
		// No --config: look where the agent's configuration actually lives. A file
		// sitting at the conventional path that nothing reads unless a flag names
		// it is a file that will be written and then wondered about.
		v.SetConfigName("config")
		v.AddConfigPath(defaultConfigDir)
		if err := v.ReadInConfig(); err != nil {
			// Absent is the normal case — `install` writes the env file, not this —
			// so only a malformed file is an error.
			if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
				if !os.IsNotExist(err) {
					return Config{}, err
				}
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	if baseURLOverride != "" {
		cfg.BaseURL = baseURLOverride
	}
	if statePathOverride != "" {
		cfg.StatePath = statePathOverride
	}

	return cfg, nil
}

// Conventional locations, named once so the reader and `install` cannot disagree
// about where configuration lives.
const (
	defaultConfigDir = "/etc/apiary-agent"
	defaultEnvFile   = defaultConfigDir + "/agent.env"
)

// loadEnvFile exports KEY=VALUE lines from a systemd EnvironmentFile.
//
// Deliberately minimal, and deliberately NOT a general parser: this reads a file
// this agent wrote, in the format systemd accepts, and anything it does not
// understand is skipped rather than guessed at. Quoting is handled because
// systemd permits it and an operator editing the file by hand will use it.
//
// Missing is not an error. A host configured entirely through the environment,
// or through --base-url, has no such file and is correct.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		// Never overrides. An exported variable is more specific than a file, and
		// someone who exports AGENT_BASE_URL to reach a different control plane
		// means it.
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
