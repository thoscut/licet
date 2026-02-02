package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"licet/internal/config"
	"licet/internal/models"
)

// DBStatsService provides database statistics and maintenance operations
type DBStatsService struct {
	db     *sqlx.DB
	dbType string
	dbPath string // For SQLite
}

// NewDBStatsService creates a new database statistics service
func NewDBStatsService(db *sqlx.DB, cfg config.DatabaseConfig) *DBStatsService {
	dbPath := ""
	if cfg.Type == "sqlite" {
		dbPath = cfg.Database
	}
	return &DBStatsService{
		db:     db,
		dbType: cfg.Type,
		dbPath: dbPath,
	}
}

// GetDatabaseStats returns comprehensive database statistics
func (s *DBStatsService) GetDatabaseStats(ctx context.Context) (*models.DatabaseStats, error) {
	stats := &models.DatabaseStats{
		Type:        s.dbType,
		GeneratedAt: time.Now(),
		Tables:      make([]models.TableStats, 0),
	}

	// Get database file size for SQLite
	if s.dbType == "sqlite" && s.dbPath != "" {
		if info, err := os.Stat(s.dbPath); err == nil {
			stats.TotalSizeBytes = info.Size()
			stats.TotalSizeHuman = formatBytes(info.Size())
		}

		// Check for WAL and SHM files
		walPath := s.dbPath + "-wal"
		shmPath := s.dbPath + "-shm"
		if info, err := os.Stat(walPath); err == nil {
			stats.TotalSizeBytes += info.Size()
			stats.WALSizeBytes = info.Size()
		}
		if info, err := os.Stat(shmPath); err == nil {
			stats.TotalSizeBytes += info.Size()
		}
		stats.TotalSizeHuman = formatBytes(stats.TotalSizeBytes)
	}

	// Get table statistics
	tables, err := s.getTableStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table stats: %w", err)
	}
	stats.Tables = tables

	// Calculate totals
	for _, t := range tables {
		stats.TotalRows += t.RowCount
	}

	// Get database-specific info
	switch s.dbType {
	case "sqlite":
		s.getSQLiteStats(ctx, stats)
	case "postgres", "postgresql":
		s.getPostgresStats(ctx, stats)
	case "mysql":
		s.getMySQLStats(ctx, stats)
	}

	// Calculate space optimization potential
	stats.Recommendations = s.analyzeSpaceOptimization(ctx, stats)

	return stats, nil
}

// getTableStats returns statistics for each table
func (s *DBStatsService) getTableStats(ctx context.Context) ([]models.TableStats, error) {
	tables := []models.TableStats{}
	tableNames := []string{"servers", "features", "feature_usage", "license_events", "alerts", "alert_events"}

	for _, tableName := range tableNames {
		ts := models.TableStats{Name: tableName}

		// Get row count
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := s.db.GetContext(ctx, &count, query); err != nil {
			// Table might not exist, skip it
			continue
		}
		ts.RowCount = count

		// Get table size (database-specific)
		switch s.dbType {
		case "sqlite":
			// SQLite doesn't have direct table size, estimate based on page count
			var pageCount int64
			// This is an approximation
			s.db.GetContext(ctx, &pageCount, "SELECT COUNT(*) * 4096 FROM "+tableName+" LIMIT 1")
		case "postgres", "postgresql":
			var size int64
			query := fmt.Sprintf("SELECT pg_total_relation_size('%s')", tableName)
			s.db.GetContext(ctx, &size, query)
			ts.SizeBytes = size
			ts.SizeHuman = formatBytes(size)
		case "mysql":
			var size int64
			query := fmt.Sprintf("SELECT data_length + index_length FROM information_schema.tables WHERE table_name = '%s'", tableName)
			s.db.GetContext(ctx, &size, query)
			ts.SizeBytes = size
			ts.SizeHuman = formatBytes(size)
		}

		// Get oldest and newest record timestamps for time-based tables
		if tableName == "feature_usage" {
			var oldest, newest string
			s.db.GetContext(ctx, &oldest, "SELECT MIN(date) FROM feature_usage")
			s.db.GetContext(ctx, &newest, "SELECT MAX(date) FROM feature_usage")
			ts.OldestRecord = oldest
			ts.NewestRecord = newest
		}

		tables = append(tables, ts)
	}

	return tables, nil
}

// getSQLiteStats gets SQLite-specific statistics
func (s *DBStatsService) getSQLiteStats(ctx context.Context, stats *models.DatabaseStats) {
	// Get page size
	var pageSize int64
	s.db.GetContext(ctx, &pageSize, "PRAGMA page_size")
	stats.PageSize = pageSize

	// Get page count
	var pageCount int64
	s.db.GetContext(ctx, &pageCount, "PRAGMA page_count")
	stats.PageCount = pageCount

	// Get freelist count (unused pages)
	var freelistCount int64
	s.db.GetContext(ctx, &freelistCount, "PRAGMA freelist_count")
	stats.FreelistCount = freelistCount

	// Calculate fragmentation percentage
	if pageCount > 0 {
		stats.FragmentationPct = float64(freelistCount) / float64(pageCount) * 100
	}

	// Get auto_vacuum setting
	var autoVacuum int
	s.db.GetContext(ctx, &autoVacuum, "PRAGMA auto_vacuum")
	switch autoVacuum {
	case 0:
		stats.AutoVacuum = "none"
	case 1:
		stats.AutoVacuum = "full"
	case 2:
		stats.AutoVacuum = "incremental"
	}

	// Get journal mode
	var journalMode string
	s.db.GetContext(ctx, &journalMode, "PRAGMA journal_mode")
	stats.JournalMode = journalMode

	// Get integrity check (quick)
	var integrityResult string
	s.db.GetContext(ctx, &integrityResult, "PRAGMA quick_check(1)")
	stats.IntegrityOK = integrityResult == "ok"
}

// getPostgresStats gets PostgreSQL-specific statistics
func (s *DBStatsService) getPostgresStats(ctx context.Context, stats *models.DatabaseStats) {
	// Get database size
	var size int64
	s.db.GetContext(ctx, &size, "SELECT pg_database_size(current_database())")
	stats.TotalSizeBytes = size
	stats.TotalSizeHuman = formatBytes(size)

	// Get dead tuples (need vacuum)
	var deadTuples int64
	s.db.GetContext(ctx, &deadTuples, `
		SELECT COALESCE(SUM(n_dead_tup), 0)
		FROM pg_stat_user_tables
	`)
	stats.DeadTuples = deadTuples
}

// getMySQLStats gets MySQL-specific statistics
func (s *DBStatsService) getMySQLStats(ctx context.Context, stats *models.DatabaseStats) {
	// Get database size
	var size int64
	s.db.GetContext(ctx, &size, `
		SELECT COALESCE(SUM(data_length + index_length), 0)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
	`)
	stats.TotalSizeBytes = size
	stats.TotalSizeHuman = formatBytes(size)
}

// analyzeSpaceOptimization analyzes the database and provides optimization recommendations
func (s *DBStatsService) analyzeSpaceOptimization(ctx context.Context, stats *models.DatabaseStats) []models.SpaceRecommendation {
	recommendations := []models.SpaceRecommendation{}

	// Check fragmentation
	if s.dbType == "sqlite" && stats.FragmentationPct > 10 {
		recommendations = append(recommendations, models.SpaceRecommendation{
			Type:        "vacuum",
			Priority:    "high",
			Title:       "Database Fragmentation",
			Description: fmt.Sprintf("Database has %.1f%% fragmentation. Running VACUUM will reclaim space.", stats.FragmentationPct),
			Impact:      fmt.Sprintf("Could reclaim approximately %s", formatBytes(int64(float64(stats.TotalSizeBytes)*stats.FragmentationPct/100))),
			Action:      "Run VACUUM to defragment the database",
		})
	}

	// Check feature_usage table size
	for _, table := range stats.Tables {
		if table.Name == "feature_usage" && table.RowCount > 1000000 {
			recommendations = append(recommendations, models.SpaceRecommendation{
				Type:        "retention",
				Priority:    "medium",
				Title:       "Large Usage History",
				Description: fmt.Sprintf("feature_usage table has %d rows. Consider implementing data retention.", table.RowCount),
				Impact:      "Reducing old data can significantly reduce database size",
				Action:      "Configure data retention policy to delete data older than 90-365 days",
			})
		}

		// Check for inactive features
		if table.Name == "features" {
			var inactiveCount int64
			s.db.GetContext(ctx, &inactiveCount, "SELECT COUNT(*) FROM features WHERE is_active = 0")
			if inactiveCount > 100 {
				recommendations = append(recommendations, models.SpaceRecommendation{
					Type:        "cleanup",
					Priority:    "low",
					Title:       "Inactive Features",
					Description: fmt.Sprintf("There are %d inactive feature records that could be archived.", inactiveCount),
					Impact:      "Minor space savings, cleaner data",
					Action:      "Archive or delete old inactive feature records",
				})
			}
		}
	}

	// Check WAL size for SQLite
	if s.dbType == "sqlite" && stats.WALSizeBytes > 50*1024*1024 { // 50MB
		recommendations = append(recommendations, models.SpaceRecommendation{
			Type:        "checkpoint",
			Priority:    "medium",
			Title:       "Large WAL File",
			Description: fmt.Sprintf("WAL file is %s. Running checkpoint will reduce it.", formatBytes(stats.WALSizeBytes)),
			Impact:      fmt.Sprintf("Reclaim up to %s", formatBytes(stats.WALSizeBytes)),
			Action:      "Run PRAGMA wal_checkpoint(TRUNCATE)",
		})
	}

	return recommendations
}

// VacuumDatabase runs VACUUM to optimize the database
func (s *DBStatsService) VacuumDatabase(ctx context.Context) (*models.VacuumResult, error) {
	result := &models.VacuumResult{
		StartedAt: time.Now(),
	}

	// Get size before
	var sizeBefore int64
	if s.dbType == "sqlite" && s.dbPath != "" {
		if info, err := os.Stat(s.dbPath); err == nil {
			sizeBefore = info.Size()
		}
	}
	result.SizeBeforeBytes = sizeBefore

	// Run VACUUM
	switch s.dbType {
	case "sqlite":
		if _, err := s.db.ExecContext(ctx, "VACUUM"); err != nil {
			return nil, fmt.Errorf("vacuum failed: %w", err)
		}
	case "postgres", "postgresql":
		if _, err := s.db.ExecContext(ctx, "VACUUM ANALYZE"); err != nil {
			return nil, fmt.Errorf("vacuum failed: %w", err)
		}
	case "mysql":
		// MySQL doesn't have VACUUM, use OPTIMIZE TABLE for each table
		tables := []string{"servers", "features", "feature_usage", "license_events", "alerts", "alert_events"}
		for _, table := range tables {
			s.db.ExecContext(ctx, fmt.Sprintf("OPTIMIZE TABLE %s", table))
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Get size after
	var sizeAfter int64
	if s.dbType == "sqlite" && s.dbPath != "" {
		if info, err := os.Stat(s.dbPath); err == nil {
			sizeAfter = info.Size()
		}
	}
	result.SizeAfterBytes = sizeAfter
	result.SpaceSavedBytes = sizeBefore - sizeAfter
	result.SpaceSavedHuman = formatBytes(result.SpaceSavedBytes)
	result.Success = true

	return result, nil
}

// CleanupOldData removes data older than the specified number of days
func (s *DBStatsService) CleanupOldData(ctx context.Context, tableName string, days int) (*models.CleanupResult, error) {
	result := &models.CleanupResult{
		TableName: tableName,
		StartedAt: time.Now(),
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)

	var query string
	var dateColumn string

	switch tableName {
	case "feature_usage":
		dateColumn = "date"
		query = "DELETE FROM feature_usage WHERE date < ?"
	case "license_events":
		dateColumn = "event_date"
		query = "DELETE FROM license_events WHERE event_date < ?"
	case "alerts":
		dateColumn = "created_at"
		query = "DELETE FROM alerts WHERE created_at < ? AND sent = 1"
	case "alert_events":
		dateColumn = "datetime"
		query = "DELETE FROM alert_events WHERE datetime < ?"
	default:
		return nil, fmt.Errorf("cleanup not supported for table: %s", tableName)
	}

	// Get count before deletion
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s < ?", tableName, dateColumn)
	var rowsToDelete int64
	if err := s.db.GetContext(ctx, &rowsToDelete, countQuery, cutoffDate); err != nil {
		return nil, fmt.Errorf("failed to count rows: %w", err)
	}
	result.RowsDeleted = rowsToDelete

	// Execute deletion
	res, err := s.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return nil, fmt.Errorf("cleanup failed: %w", err)
	}

	affected, _ := res.RowsAffected()
	result.RowsDeleted = affected
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Success = true

	return result, nil
}

// GetRetentionStats returns statistics about data retention
func (s *DBStatsService) GetRetentionStats(ctx context.Context) (*models.RetentionStats, error) {
	stats := &models.RetentionStats{}

	// Feature usage stats
	s.db.GetContext(ctx, &stats.UsageRecordsTotal, "SELECT COUNT(*) FROM feature_usage")
	s.db.GetContext(ctx, &stats.UsageRecords30Days, "SELECT COUNT(*) FROM feature_usage WHERE date >= date('now', '-30 days')")
	s.db.GetContext(ctx, &stats.UsageRecords90Days, "SELECT COUNT(*) FROM feature_usage WHERE date >= date('now', '-90 days')")
	s.db.GetContext(ctx, &stats.UsageRecords365Days, "SELECT COUNT(*) FROM feature_usage WHERE date >= date('now', '-365 days')")

	// License events stats
	s.db.GetContext(ctx, &stats.EventsTotal, "SELECT COUNT(*) FROM license_events")
	s.db.GetContext(ctx, &stats.Events30Days, "SELECT COUNT(*) FROM license_events WHERE event_date >= date('now', '-30 days')")

	// Alerts stats
	s.db.GetContext(ctx, &stats.AlertsTotal, "SELECT COUNT(*) FROM alerts")
	s.db.GetContext(ctx, &stats.AlertsSent, "SELECT COUNT(*) FROM alerts WHERE sent = 1")

	// Calculate potential savings
	olderThan90 := stats.UsageRecordsTotal - stats.UsageRecords90Days
	if olderThan90 > 0 {
		stats.PotentialSavingsRows = olderThan90
		// Rough estimate: ~100 bytes per row
		stats.PotentialSavingsBytes = olderThan90 * 100
		stats.PotentialSavingsHuman = formatBytes(stats.PotentialSavingsBytes)
	}

	return stats, nil
}

// AnalyzeDatabase runs ANALYZE to update statistics (helps query planning)
func (s *DBStatsService) AnalyzeDatabase(ctx context.Context) error {
	switch s.dbType {
	case "sqlite":
		_, err := s.db.ExecContext(ctx, "ANALYZE")
		return err
	case "postgres", "postgresql":
		_, err := s.db.ExecContext(ctx, "ANALYZE")
		return err
	case "mysql":
		tables := []string{"servers", "features", "feature_usage", "license_events", "alerts", "alert_events"}
		for _, table := range tables {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ANALYZE TABLE %s", table)); err != nil {
				return err
			}
		}
		return nil
	}
	return nil
}

// CheckpointWAL checkpoints the WAL file (SQLite only)
func (s *DBStatsService) CheckpointWAL(ctx context.Context) error {
	if s.dbType != "sqlite" {
		return fmt.Errorf("WAL checkpoint only supported for SQLite")
	}
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// GetDatabasePath returns the database file path (SQLite only)
func (s *DBStatsService) GetDatabasePath() string {
	if s.dbType == "sqlite" {
		absPath, err := filepath.Abs(s.dbPath)
		if err != nil {
			return s.dbPath
		}
		return absPath
	}
	return ""
}

// formatBytes formats bytes into human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
