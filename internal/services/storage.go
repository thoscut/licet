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

// StoreFeatures stores features to the database using optimized batch operations.
// It first marks all existing features for the server as inactive, then upserts
// the new features as active. This ensures that replaced/removed licenses are
// no longer shown as active.
func (s *StorageService) StoreFeatures(ctx context.Context, features []models.Feature) error {
	if len(features) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First, deactivate all existing features for this server.
	// This ensures that replaced/removed licenses are marked as inactive.
	hostname := features[0].ServerHostname
	_, err = tx.ExecContext(ctx, s.dialect.DeactivateFeaturesForServer(), hostname)
	if err != nil {
		return fmt.Errorf("failed to deactivate features for %s: %w", hostname, err)
	}

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

// GetFeatures retrieves all active features for a server
func (s *StorageService) GetFeatures(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? AND is_active = 1 ORDER BY name`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetAllFeatures retrieves all features for a server, including inactive ones
func (s *StorageService) GetAllFeatures(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? ORDER BY name`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetFeaturesWithExpiration returns active features that have expiration dates
func (s *StorageService) GetFeaturesWithExpiration(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `
		SELECT id, server_hostname, name, version, vendor_daemon,
		       total_licenses, used_licenses, expiration_date, last_updated, is_active
		FROM features
		WHERE server_hostname = ?
		  AND expiration_date IS NOT NULL
		  AND is_active = 1
		ORDER BY expiration_date ASC, name ASC
	`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetAllFeaturesWithExpiration returns all features with expiration dates, including inactive ones
func (s *StorageService) GetAllFeaturesWithExpiration(ctx context.Context, hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `
		SELECT f.id, f.server_hostname, f.name, f.version, f.vendor_daemon,
		       f.total_licenses, f.used_licenses, f.expiration_date, f.last_updated, f.is_active
		FROM features f
		INNER JOIN (
			SELECT server_hostname, name, version, expiration_date, MAX(id) as max_id
			FROM features
			WHERE server_hostname = ?
			  AND expiration_date IS NOT NULL
			GROUP BY server_hostname, name, version, expiration_date
		) latest ON f.id = latest.max_id
		ORDER BY f.is_active DESC, f.expiration_date ASC, f.name ASC
	`
	err := s.db.SelectContext(ctx, &features, query, hostname)
	return features, err
}

// GetExpiringFeatures returns active features expiring within the specified number of days
func (s *StorageService) GetExpiringFeatures(ctx context.Context, days int) ([]models.Feature, error) {
	var features []models.Feature
	cutoff := time.Now().AddDate(0, 0, days)

	query := `
		SELECT * FROM features
		WHERE expiration_date <= ? AND expiration_date > ? AND is_active = 1
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
