package services

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"licet/internal/config"
)

func TestNewDBStatsService(t *testing.T) {
	cfg := config.DatabaseConfig{
		Type:     "sqlite",
		Database: ":memory:",
	}

	db, err := sqlx.Connect("sqlite3", cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	service := NewDBStatsService(db, cfg)
	if service == nil {
		t.Fatal("Expected non-nil service")
	}

	if service.dbType != "sqlite" {
		t.Errorf("Expected dbType 'sqlite', got '%s'", service.dbType)
	}
}

func TestGetDatabaseStats(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "licet_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := config.DatabaseConfig{
		Type:     "sqlite",
		Database: tmpFile.Name(),
	}

	db, err := sqlx.Connect("sqlite3", cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (
			id INTEGER PRIMARY KEY,
			hostname TEXT UNIQUE
		);
		CREATE TABLE IF NOT EXISTS features (
			id INTEGER PRIMARY KEY,
			server_hostname TEXT,
			name TEXT,
			is_active INTEGER DEFAULT 1
		);
		CREATE TABLE IF NOT EXISTS feature_usage (
			id INTEGER PRIMARY KEY,
			server_hostname TEXT,
			feature_name TEXT,
			date TEXT,
			time TEXT,
			users_count INTEGER
		);
		CREATE TABLE IF NOT EXISTS license_events (
			id INTEGER PRIMARY KEY,
			event_date TEXT
		);
		CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY,
			created_at TEXT,
			sent INTEGER
		);
		CREATE TABLE IF NOT EXISTS alert_events (
			id INTEGER PRIMARY KEY,
			datetime TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO servers (hostname) VALUES ('test-server');
		INSERT INTO features (server_hostname, name, is_active) VALUES ('test-server', 'feature1', 1);
		INSERT INTO features (server_hostname, name, is_active) VALUES ('test-server', 'feature2', 0);
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	service := NewDBStatsService(db, cfg)
	ctx := context.Background()

	stats, err := service.GetDatabaseStats(ctx)
	if err != nil {
		t.Fatalf("GetDatabaseStats failed: %v", err)
	}

	if stats.Type != "sqlite" {
		t.Errorf("Expected type 'sqlite', got '%s'", stats.Type)
	}

	if stats.TotalRows < 3 {
		t.Errorf("Expected at least 3 total rows, got %d", stats.TotalRows)
	}

	// Check tables exist in stats
	foundServers := false
	foundFeatures := false
	for _, table := range stats.Tables {
		if table.Name == "servers" {
			foundServers = true
			if table.RowCount != 1 {
				t.Errorf("Expected 1 row in servers, got %d", table.RowCount)
			}
		}
		if table.Name == "features" {
			foundFeatures = true
			if table.RowCount != 2 {
				t.Errorf("Expected 2 rows in features, got %d", table.RowCount)
			}
		}
	}

	if !foundServers {
		t.Error("servers table not found in stats")
	}
	if !foundFeatures {
		t.Error("features table not found in stats")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestVacuumDatabase(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "licet_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := config.DatabaseConfig{
		Type:     "sqlite",
		Database: tmpFile.Name(),
	}

	db, err := sqlx.Connect("sqlite3", cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a table and insert/delete data to create fragmentation
	_, err = db.Exec(`
		CREATE TABLE test_data (id INTEGER PRIMARY KEY, data TEXT);
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	for i := 0; i < 100; i++ {
		db.Exec("INSERT INTO test_data (data) VALUES (?)", "test data string to take up space")
	}
	db.Exec("DELETE FROM test_data WHERE id > 50")

	service := NewDBStatsService(db, cfg)
	ctx := context.Background()

	result, err := service.VacuumDatabase(ctx)
	if err != nil {
		t.Fatalf("VacuumDatabase failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected vacuum to succeed")
	}

	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestAnalyzeDatabase(t *testing.T) {
	cfg := config.DatabaseConfig{
		Type:     "sqlite",
		Database: ":memory:",
	}

	db, err := sqlx.Connect("sqlite3", cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS servers (id INTEGER PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS features (id INTEGER PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS feature_usage (id INTEGER PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS license_events (id INTEGER PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS alerts (id INTEGER PRIMARY KEY);
		CREATE TABLE IF NOT EXISTS alert_events (id INTEGER PRIMARY KEY);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	service := NewDBStatsService(db, cfg)
	ctx := context.Background()

	err = service.AnalyzeDatabase(ctx)
	if err != nil {
		t.Fatalf("AnalyzeDatabase failed: %v", err)
	}
}

func TestGetRetentionStats(t *testing.T) {
	cfg := config.DatabaseConfig{
		Type:     "sqlite",
		Database: ":memory:",
	}

	db, err := sqlx.Connect("sqlite3", cfg.Database)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create test tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS feature_usage (
			id INTEGER PRIMARY KEY,
			date TEXT
		);
		CREATE TABLE IF NOT EXISTS license_events (
			id INTEGER PRIMARY KEY,
			event_date TEXT
		);
		CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY,
			sent INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO feature_usage (date) VALUES (date('now'));
		INSERT INTO feature_usage (date) VALUES (date('now', '-10 days'));
		INSERT INTO feature_usage (date) VALUES (date('now', '-100 days'));
		INSERT INTO license_events (event_date) VALUES (date('now'));
		INSERT INTO alerts (sent) VALUES (1);
		INSERT INTO alerts (sent) VALUES (0);
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	service := NewDBStatsService(db, cfg)
	ctx := context.Background()

	stats, err := service.GetRetentionStats(ctx)
	if err != nil {
		t.Fatalf("GetRetentionStats failed: %v", err)
	}

	if stats.UsageRecordsTotal != 3 {
		t.Errorf("Expected 3 total usage records, got %d", stats.UsageRecordsTotal)
	}

	if stats.UsageRecords30Days != 2 {
		t.Errorf("Expected 2 usage records in last 30 days, got %d", stats.UsageRecords30Days)
	}

	if stats.EventsTotal != 1 {
		t.Errorf("Expected 1 event, got %d", stats.EventsTotal)
	}

	if stats.AlertsTotal != 2 {
		t.Errorf("Expected 2 alerts, got %d", stats.AlertsTotal)
	}

	if stats.AlertsSent != 1 {
		t.Errorf("Expected 1 sent alert, got %d", stats.AlertsSent)
	}
}
