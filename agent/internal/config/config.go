package config

import (
	"errors"
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

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
				return Config{}, err
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
