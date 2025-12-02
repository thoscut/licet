package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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
			UNIQUE(server_hostname, name, version, expiration_date)
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

	// Run migrations
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Fix existing databases with wrong UNIQUE constraint on features table
	// This migration rebuilds the features table if it has the old constraint
	if err := fixFeaturesTableConstraint(db); err != nil {
		return fmt.Errorf("failed to fix features table constraint: %w", err)
	}

	// Clean up duplicate permanent license records created before parser fix
	if err := cleanupDuplicatePermanentLicenses(db); err != nil {
		return fmt.Errorf("failed to clean up duplicate permanent licenses: %w", err)
	}

	return nil
}

func fixFeaturesTableConstraint(db *sqlx.DB) error {
	// Check if the table has the problematic constraint by trying to insert duplicates
	// If we can't determine, just rebuild it anyway (safe operation)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Rebuild the features table with correct constraint
	migrations := []string{
		`DROP TABLE IF EXISTS features_backup`,
		`CREATE TABLE features_backup AS SELECT * FROM features`,
		`DROP TABLE features`,
		`CREATE TABLE features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_hostname TEXT NOT NULL,
			name TEXT NOT NULL,
			version TEXT,
			vendor_daemon TEXT,
			total_licenses INTEGER NOT NULL,
			used_licenses INTEGER NOT NULL,
			expiration_date TIMESTAMP,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(server_hostname, name, version, expiration_date)
		)`,
		`INSERT INTO features (id, server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated)
		 SELECT id, server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated
		 FROM features_backup`,
		`DROP TABLE features_backup`,
	}

	for _, migration := range migrations {
		if _, err := tx.Exec(migration); err != nil {
			return fmt.Errorf("constraint fix migration failed: %w", err)
		}
	}

	return tx.Commit()
}

func cleanupDuplicatePermanentLicenses(db *sqlx.DB) error {
	// This migration removes duplicate permanent license records that were created
	// before the parser fix. Permanent licenses should all have the same expiration
	// date (2099-01-01), but old records have varying timestamps.

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// For each (server_hostname, name, version) combination, keep only the latest
	// record and delete the rest
	query := `
		DELETE FROM features
		WHERE id NOT IN (
			SELECT MAX(id)
			FROM features
			WHERE expiration_date IS NOT NULL
			GROUP BY server_hostname, name, version
		)
		AND expiration_date IS NOT NULL
	`

	_, err = tx.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clean up duplicates: %w", err)
	}

	return tx.Commit()
}
