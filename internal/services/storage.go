package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"licet/internal/database"
	"licet/internal/models"
)

// StorageService handles feature storage and retrieval operations
type StorageService struct {
	db      *sqlx.DB
	dialect database.Dialect
}

// NewStorageService creates a new storage service
func NewStorageService(db *sqlx.DB, dbType string) *StorageService {
	return &StorageService{
		db:      db,
		dialect: database.NewDialect(dbType),
	}
}

// StoreFeatures stores features to the database using optimized batch operations
func (s *StorageService) StoreFeatures(ctx context.Context, features []models.Feature) error {
	if len(features) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PreparexContext(ctx, s.dialect.UpsertFeature())
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, feature := range features {
		_, err := stmt.ExecContext(ctx,
			feature.ServerHostname,
			feature.Name,
			feature.Version,
			feature.VendorDaemon,
			feature.TotalLicenses,
			feature.UsedLicenses,
			feature.ExpirationDate,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to insert feature %s: %w", feature.Name, err)
		}
	}

	return tx.Commit()
}

// RecordUsage records feature usage history using optimized batch operations
func (s *StorageService) RecordUsage(ctx context.Context, features []models.Feature) error {
	if len(features) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	date := now.Format("2006-01-02")
	timeStr := now.Format("15:04:00")

	stmt, err := tx.PreparexContext(ctx, s.dialect.InsertIgnoreUsage())
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, feature := range features {
		_, err := stmt.ExecContext(ctx,
			feature.ServerHostname,
			feature.Name,
			date,
			timeStr,
			feature.UsedLicenses,
		)
		if err != nil {
			return fmt.Errorf("failed to record usage for %s: %w", feature.Name, err)
		}
	}

	return tx.Commit()
}

// GetFeatures retrieves all features for a server
func (s *StorageService) GetFeatures(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? ORDER BY name`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetFeaturesWithExpiration returns features that have expiration dates, deduplicated
func (s *StorageService) GetFeaturesWithExpiration(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `
		SELECT f.id, f.server_hostname, f.name, f.version, f.vendor_daemon,
		       f.total_licenses, f.used_licenses, f.expiration_date, f.last_updated
		FROM features f
		INNER JOIN (
			SELECT server_hostname, name, version, expiration_date, MAX(id) as max_id
			FROM features
			WHERE server_hostname = ?
			  AND expiration_date IS NOT NULL
			GROUP BY server_hostname, name, version, expiration_date
		) latest ON f.id = latest.max_id
		ORDER BY f.expiration_date ASC, f.name ASC
	`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetExpiringFeatures returns features expiring within the specified number of days
func (s *StorageService) GetExpiringFeatures(ctx context.Context, days int) ([]models.Feature, error) {
	var features []models.Feature
	cutoff := time.Now().AddDate(0, 0, days)

	query := `
		SELECT * FROM features
		WHERE expiration_date <= ? AND expiration_date > ?
		ORDER BY expiration_date ASC
	`
	err := s.db.SelectContext(ctx, &features, query, cutoff, time.Now())
	return features, err
}

// GetFeatureUsageHistory returns historical usage data for a specific feature
func (s *StorageService) GetFeatureUsageHistory(ctx context.Context, hostname, featureName string, days int) ([]models.FeatureUsage, error) {
	var usage []models.FeatureUsage
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT * FROM feature_usage
		WHERE server_hostname = ? AND feature_name = ? AND date >= ?
		ORDER BY date DESC, time DESC
	`
	err := s.db.SelectContext(ctx, &usage, query, hostname, featureName, cutoff)
	return usage, err
}
