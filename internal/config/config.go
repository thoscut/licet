package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Logging   LoggingConfig
	Servers   []LicenseServer
	Email     EmailConfig
	Alerts    AlertConfig
	RRD       RRDConfig
	Cache     CacheConfig
	RateLimit RateLimitConfig
	Export    ExportConfig
}

type ServerConfig struct {
	Port               int      `mapstructure:"port"`
	Host               string   `mapstructure:"host"`
	SettingsEnabled    bool     `mapstructure:"settings_enabled"`
	UtilizationEnabled bool     `mapstructure:"utilization_enabled"`
	StatisticsEnabled  bool     `mapstructure:"statistics_enabled"`
	CORSOrigins        []string `mapstructure:"cors_origins"`
	TLSEnabled         bool     `mapstructure:"tls_enabled"`
	TLSCertFile        string   `mapstructure:"tls_cert_file"`
	TLSKeyFile         string   `mapstructure:"tls_key_file"`
}

type DatabaseConfig struct {
	Type            string `mapstructure:"type"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // Maximum open connections (default: 25)
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // Maximum idle connections (default: 5)
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // Connection max lifetime in minutes (default: 0 = unlimited)
}

type LoggingConfig struct {
	Level  string
	Format string
}

type LicenseServer struct {
	Hostname    string
	Description string
	Type        string
	CactiID     string
	WebUI       string
}

type EmailConfig struct {
	From     string
	To       []string
	Alerts   []string
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	Enabled  bool
}

type AlertConfig struct {
	LeadTimeDays      int  `mapstructure:"lead_time_days"`
	ResendIntervalMin int  `mapstructure:"resend_interval_min"`
	Enabled           bool `mapstructure:"enabled"`
}

type RRDConfig struct {
	Enabled            bool
	Directory          string
	CollectionInterval int
}

type CacheConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	TTLSeconds int  `mapstructure:"ttl_seconds"`
	MaxEntries int  `mapstructure:"max_entries"`
}

type RateLimitConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	RequestsPerMinute int      `mapstructure:"requests_per_minute"`
	BurstSize         int      `mapstructure:"burst_size"`
	WhitelistedIPs    []string `mapstructure:"whitelisted_ips"`
	WhitelistedPaths  []string `mapstructure:"whitelisted_paths"`
}

type ExportConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	AllowedFormats []string `mapstructure:"allowed_formats"`
	MaxRecords    int      `mapstructure:"max_records"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/licet")

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.settings_enabled", true)
	viper.SetDefault("server.utilization_enabled", true)
	viper.SetDefault("server.statistics_enabled", true)
	viper.SetDefault("server.cors_origins", []string{"http://localhost:8080"})
	viper.SetDefault("server.tls_enabled", false)
	viper.SetDefault("server.tls_cert_file", "")
	viper.SetDefault("server.tls_key_file", "")
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.database", "licet.db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("alerts.lead_time_days", 10)
	viper.SetDefault("alerts.resend_interval_min", 60)
	viper.SetDefault("alerts.enabled", false)
	viper.SetDefault("email.enabled", false)
	viper.SetDefault("rrd.enabled", false)
	viper.SetDefault("rrd.collectionInterval", 5)

	// Cache defaults
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.ttl_seconds", 30)
	viper.SetDefault("cache.max_entries", 1000)

	// Rate limit defaults
	viper.SetDefault("ratelimit.enabled", true)
	viper.SetDefault("ratelimit.requests_per_minute", 100)
	viper.SetDefault("ratelimit.burst_size", 20)
	viper.SetDefault("ratelimit.whitelisted_ips", []string{"127.0.0.1", "::1"})
	viper.SetDefault("ratelimit.whitelisted_paths", []string{"/api/v1/health", "/static/"})

	// Export defaults
	viper.SetDefault("export.enabled", true)
	viper.SetDefault("export.allowed_formats", []string{"json", "csv"})
	viper.SetDefault("export.max_records", 10000)

	// Environment variables
	viper.SetEnvPrefix("LICET")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) GetDSN() string {
	switch c.Database.Type {
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Database.Host, c.Database.Port, c.Database.Username,
			c.Database.Password, c.Database.Database, c.Database.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Database.Username, c.Database.Password,
			c.Database.Host, c.Database.Port, c.Database.Database)
	case "sqlite":
		return c.Database.Database
	default:
		return ""
	}
}
