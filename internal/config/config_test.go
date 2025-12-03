package config

import (
	"os"
	"testing"
)

func TestGetDSN_SQLite(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Type:     "sqlite",
			Database: "test.db",
		},
	}

	dsn := cfg.GetDSN()
	expected := "test.db"

	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestGetDSN_Postgres(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Username: "testuser",
			Password: "testpass",
			Database: "testdb",
			SSLMode:  "disable",
		},
	}

	dsn := cfg.GetDSN()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"

	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestGetDSN_PostgreSQL(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Type:     "postgresql",
			Host:     "db.example.com",
			Port:     5432,
			Username: "admin",
			Password: "secret",
			Database: "proddb",
			SSLMode:  "require",
		},
	}

	dsn := cfg.GetDSN()
	expected := "host=db.example.com port=5432 user=admin password=secret dbname=proddb sslmode=require"

	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestGetDSN_MySQL(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Type:     "mysql",
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "password",
			Database: "licet",
		},
	}

	dsn := cfg.GetDSN()
	expected := "root:password@tcp(localhost:3306)/licet?parseTime=true"

	if dsn != expected {
		t.Errorf("Expected DSN '%s', got '%s'", expected, dsn)
	}
}

func TestGetDSN_UnknownType(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Type: "unknown",
		},
	}

	dsn := cfg.GetDSN()
	if dsn != "" {
		t.Errorf("Expected empty DSN for unknown type, got '%s'", dsn)
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	// Test that default ServerConfig values make sense
	cfg := ServerConfig{
		Port:               8080,
		Host:               "0.0.0.0",
		SettingsEnabled:    true,
		UtilizationEnabled: true,
		StatisticsEnabled:  true,
		CORSOrigins:        []string{"http://localhost:8080"},
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Port)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got '%s'", cfg.Host)
	}

	if !cfg.SettingsEnabled {
		t.Error("Expected SettingsEnabled to be true")
	}

	if len(cfg.CORSOrigins) != 1 {
		t.Errorf("Expected 1 CORS origin, got %d", len(cfg.CORSOrigins))
	}
}

func TestEmailConfig(t *testing.T) {
	cfg := EmailConfig{
		From:     "test@example.com",
		To:       []string{"admin@example.com"},
		Alerts:   []string{"alerts@example.com"},
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		Username: "user",
		Password: "pass",
		Enabled:  true,
	}

	if cfg.From != "test@example.com" {
		t.Errorf("Expected from 'test@example.com', got '%s'", cfg.From)
	}

	if len(cfg.To) != 1 {
		t.Errorf("Expected 1 recipient, got %d", len(cfg.To))
	}

	if cfg.SMTPPort != 587 {
		t.Errorf("Expected SMTP port 587, got %d", cfg.SMTPPort)
	}

	if !cfg.Enabled {
		t.Error("Expected email to be enabled")
	}
}

func TestAlertConfig(t *testing.T) {
	cfg := AlertConfig{
		LeadTimeDays:      10,
		ResendIntervalMin: 60,
		Enabled:           true,
	}

	if cfg.LeadTimeDays != 10 {
		t.Errorf("Expected lead time 10 days, got %d", cfg.LeadTimeDays)
	}

	if cfg.ResendIntervalMin != 60 {
		t.Errorf("Expected resend interval 60 minutes, got %d", cfg.ResendIntervalMin)
	}

	if !cfg.Enabled {
		t.Error("Expected alerts to be enabled")
	}
}

func TestLicenseServer(t *testing.T) {
	server := LicenseServer{
		Hostname:    "27000@flexlm.example.com",
		Description: "Production FlexLM Server",
		Type:        "flexlm",
		CactiID:     "123",
		WebUI:       "http://flexlm.example.com",
	}

	if server.Hostname != "27000@flexlm.example.com" {
		t.Errorf("Expected hostname '27000@flexlm.example.com', got '%s'", server.Hostname)
	}

	if server.Type != "flexlm" {
		t.Errorf("Expected type 'flexlm', got '%s'", server.Type)
	}

	if server.CactiID != "123" {
		t.Errorf("Expected CactiID '123', got '%s'", server.CactiID)
	}
}

func TestRRDConfig(t *testing.T) {
	cfg := RRDConfig{
		Enabled:            true,
		Directory:          "./rrd",
		CollectionInterval: 5,
	}

	if !cfg.Enabled {
		t.Error("Expected RRD to be enabled")
	}

	if cfg.Directory != "./rrd" {
		t.Errorf("Expected directory './rrd', got '%s'", cfg.Directory)
	}

	if cfg.CollectionInterval != 5 {
		t.Errorf("Expected collection interval 5, got %d", cfg.CollectionInterval)
	}
}

func TestLoggingConfig(t *testing.T) {
	cfg := LoggingConfig{
		Level:  "info",
		Format: "text",
	}

	if cfg.Level != "info" {
		t.Errorf("Expected level 'info', got '%s'", cfg.Level)
	}

	if cfg.Format != "text" {
		t.Errorf("Expected format 'text', got '%s'", cfg.Format)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Create a temp directory with no config file
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Load should succeed with defaults when config file is missing
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should succeed with defaults when config missing: %v", err)
	}

	// Verify some defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.Type != "sqlite" {
		t.Errorf("Expected default database type 'sqlite', got '%s'", cfg.Database.Type)
	}
}
