package config

import "time"

// Config holds all application configuration.
type Config struct {
	ACS      ACSConfig      `mapstructure:"acs"`
	API      APIConfig      `mapstructure:"api"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// APIConfig holds REST API server settings.
type APIConfig struct {
	Addr string `mapstructure:"addr"`
}

// ACSConfig holds ACS server settings.
type ACSConfig struct {
	CWMPPort              int           `mapstructure:"cwmp_port"`
	ConnectionRequestPort int           `mapstructure:"connection_request_port"`
	APIPort               int           `mapstructure:"api_port"`
	BaseURL               string        `mapstructure:"base_url"`
	FirmwareBaseURL       string        `mapstructure:"firmware_base_url"`
	SessionTimeout        time.Duration `mapstructure:"session_timeout"`
	MaxConcurrentSessions int           `mapstructure:"max_concurrent_sessions"`
	TLS                   TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS settings for ACS.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig holds MongoDB, Postgres, and Redis settings.
type DatabaseConfig struct {
	MongoURI    string `mapstructure:"mongo_uri"`
	MongoDB     string `mapstructure:"mongo_db"`
	PostgresDSN string `mapstructure:"postgres_dsn"`
	RedisAddr   string `mapstructure:"redis_addr"`
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL string `mapstructure:"url"`
}

// AuthConfig holds JWT and auth settings.
type AuthConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessTTL     time.Duration `mapstructure:"access_ttl"`
	RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}
