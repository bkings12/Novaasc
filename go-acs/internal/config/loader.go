package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Load reads configuration from file and environment.
func Load(path string) (*Config, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && path == "" {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var c Config
	if err := v.Unmarshal(&c, func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = mapstructure.StringToTimeDurationHookFunc()
	}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Apply defaults (viper parses "30s" as duration)
	if c.ACS.SessionTimeout == 0 {
		c.ACS.SessionTimeout = 30 * time.Second
	}
	if c.ACS.MaxConcurrentSessions == 0 {
		c.ACS.MaxConcurrentSessions = 10000
	}

	return &c, nil
}
