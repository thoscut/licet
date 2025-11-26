package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	Servers  []LicenseServer
	Email    EmailConfig
	Alerts   AlertConfig
	RRD      RRDConfig
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	Type     string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
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
	LeadTimeDays      int
	ResendIntervalMin int
	Enabled           bool
}

type RRDConfig struct {
	Enabled           bool
	Directory         string
	CollectionInterval int
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
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.database", "licet.db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("alerts.leadTimedays", 10)
	viper.SetDefault("alerts.resendIntervalMin", 60)
	viper.SetDefault("alerts.enabled", false)
	viper.SetDefault("email.enabled", false)
	viper.SetDefault("rrd.enabled", false)
	viper.SetDefault("rrd.collectionInterval", 5)

	// Environment variables
	viper.SetEnvPrefix("PLW")
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
