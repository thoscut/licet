package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thoscut/licet/internal/config"
)

func New(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	var driverName string
	var dsn string

	switch cfg.Type {
	case "postgres", "postgresql":
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)
	case "sqlite":
		driverName = "sqlite3"
		dsn = cfg.Database
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := sqlx.Connect(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hostname TEXT NOT NULL UNIQUE,
			description TEXT,
			type TEXT NOT NULL,
			cacti_id TEXT,
			webui TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_hostname TEXT NOT NULL,
			name TEXT NOT NULL,
			version TEXT,
			vendor_daemon TEXT,
			total_licenses INTEGER NOT NULL,
			used_licenses INTEGER NOT NULL,
			expiration_date TIMESTAMP,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(server_hostname, name, version)
		)`,
		`CREATE TABLE IF NOT EXISTS feature_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_hostname TEXT NOT NULL,
			feature_name TEXT NOT NULL,
			date DATE NOT NULL,
			time TIME NOT NULL,
			users_count INTEGER NOT NULL,
			UNIQUE(server_hostname, feature_name, date, time)
		)`,
		`CREATE TABLE IF NOT EXISTS license_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_date DATE NOT NULL,
			event_time TIME NOT NULL,
			event_type TEXT NOT NULL,
			feature_name TEXT NOT NULL,
			username TEXT NOT NULL,
			reason TEXT,
			UNIQUE(event_date, event_time, feature_name, username)
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_hostname TEXT NOT NULL,
			feature_name TEXT,
			alert_type TEXT NOT NULL,
			message TEXT NOT NULL,
			severity TEXT NOT NULL,
			sent BOOLEAN DEFAULT FALSE,
			sent_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS alert_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			datetime TIMESTAMP NOT NULL,
			type TEXT NOT NULL,
			hostname TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_features_server ON features(server_hostname)`,
		`CREATE INDEX IF NOT EXISTS idx_features_expiration ON features(expiration_date)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_server_feature ON feature_usage(server_hostname, feature_name)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_date ON feature_usage(date)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_sent ON alerts(sent)`,
		`CREATE INDEX IF NOT EXISTS idx_events_date ON license_events(event_date)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
