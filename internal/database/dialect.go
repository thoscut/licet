package database

// Dialect provides database-specific SQL expressions
type Dialect interface {
	// UpsertFeature returns the SQL for inserting or updating a feature
	UpsertFeature() string
	// InsertIgnoreUsage returns the SQL for inserting usage data (ignoring duplicates)
	InsertIgnoreUsage() string
	// TimestampConcat returns the SQL expression for concatenating date and time into a timestamp
	TimestampConcat() string
	// HourExtract returns the SQL expression for extracting hour from a time column
	HourExtract() string
	// Placeholder returns the appropriate placeholder for the given parameter index (1-based)
	Placeholder(index int) string
	// SupportsPositionalParams returns true if the dialect uses positional params ($1, $2)
	SupportsPositionalParams() bool
	// DeactivateFeaturesForServer returns the SQL to mark all features as inactive for a server
	DeactivateFeaturesForServer() string
}

// NewDialect creates a dialect for the given database type
func NewDialect(dbType string) Dialect {
	switch dbType {
	case "postgres", "postgresql":
		return &PostgresDialect{}
	case "mysql":
		return &MySQLDialect{}
	default:
		return &SQLiteDialect{}
	}
}

// PostgresDialect implements Dialect for PostgreSQL
type PostgresDialect struct{}

func (d *PostgresDialect) UpsertFeature() string {
	return `
		INSERT INTO features
		(server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		ON CONFLICT (server_hostname, name, version) DO UPDATE SET
			vendor_daemon = EXCLUDED.vendor_daemon,
			total_licenses = EXCLUDED.total_licenses,
			used_licenses = EXCLUDED.used_licenses,
			expiration_date = EXCLUDED.expiration_date,
			last_updated = EXCLUDED.last_updated,
			is_active = TRUE
	`
}

func (d *PostgresDialect) InsertIgnoreUsage() string {
	return `
		INSERT INTO feature_usage
		(server_hostname, feature_name, date, time, users_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (server_hostname, feature_name, date, time) DO NOTHING
	`
}

func (d *PostgresDialect) TimestampConcat() string {
	return "(date || ' ' || time)::timestamp"
}

func (d *PostgresDialect) HourExtract() string {
	return "EXTRACT(HOUR FROM time::time)::integer"
}

func (d *PostgresDialect) Placeholder(index int) string {
	return "$" + string(rune('0'+index))
}

func (d *PostgresDialect) SupportsPositionalParams() bool {
	return true
}

func (d *PostgresDialect) DeactivateFeaturesForServer() string {
	return `UPDATE features SET is_active = FALSE WHERE server_hostname = $1`
}

// MySQLDialect implements Dialect for MySQL
type MySQLDialect struct{}

func (d *MySQLDialect) UpsertFeature() string {
	return `
		INSERT INTO features
		(server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE)
		ON DUPLICATE KEY UPDATE
			vendor_daemon = VALUES(vendor_daemon),
			total_licenses = VALUES(total_licenses),
			used_licenses = VALUES(used_licenses),
			expiration_date = VALUES(expiration_date),
			last_updated = VALUES(last_updated),
			is_active = TRUE
	`
}

func (d *MySQLDialect) InsertIgnoreUsage() string {
	return `
		INSERT IGNORE INTO feature_usage
		(server_hostname, feature_name, date, time, users_count)
		VALUES (?, ?, ?, ?, ?)
	`
}

func (d *MySQLDialect) TimestampConcat() string {
	return "CONCAT(date, ' ', time)"
}

func (d *MySQLDialect) HourExtract() string {
	return "HOUR(time)"
}

func (d *MySQLDialect) Placeholder(index int) string {
	return "?"
}

func (d *MySQLDialect) SupportsPositionalParams() bool {
	return false
}

func (d *MySQLDialect) DeactivateFeaturesForServer() string {
	return `UPDATE features SET is_active = FALSE WHERE server_hostname = ?`
}

// SQLiteDialect implements Dialect for SQLite
type SQLiteDialect struct{}

func (d *SQLiteDialect) UpsertFeature() string {
	return `
		INSERT OR REPLACE INTO features
		(server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
	`
}

func (d *SQLiteDialect) InsertIgnoreUsage() string {
	return `
		INSERT OR IGNORE INTO feature_usage
		(server_hostname, feature_name, date, time, users_count)
		VALUES (?, ?, ?, ?, ?)
	`
}

func (d *SQLiteDialect) TimestampConcat() string {
	return "datetime(date || ' ' || time)"
}

func (d *SQLiteDialect) HourExtract() string {
	return "CAST(strftime('%H', time) AS INTEGER)"
}

func (d *SQLiteDialect) Placeholder(index int) string {
	return "?"
}

func (d *SQLiteDialect) SupportsPositionalParams() bool {
	return false
}

func (d *SQLiteDialect) DeactivateFeaturesForServer() string {
	return `UPDATE features SET is_active = 0 WHERE server_hostname = ?`
}
