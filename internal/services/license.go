package services

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/models"
	"github.com/thoscut/licet/internal/parsers"
)

type LicenseService struct {
	db            *sqlx.DB
	cfg           *config.Config
	parserFactory *parsers.ParserFactory
}

func NewLicenseService(db *sqlx.DB, cfg *config.Config) *LicenseService {
	// Use cross-platform binary path detection
	binPaths := GetDefaultBinaryPaths()

	return &LicenseService{
		db:            db,
		cfg:           cfg,
		parserFactory: parsers.NewParserFactory(binPaths),
	}
}

func (s *LicenseService) GetAllServers() ([]models.LicenseServer, error) {
	var servers []models.LicenseServer

	// Return configured servers from config
	for _, srv := range s.cfg.Servers {
		servers = append(servers, models.LicenseServer{
			Hostname:    srv.Hostname,
			Description: srv.Description,
			Type:        srv.Type,
			CactiID:     srv.CactiID,
			WebUI:       srv.WebUI,
		})
	}

	return servers, nil
}

func (s *LicenseService) QueryServer(hostname, serverType string) (models.ServerQueryResult, error) {
	parser, err := s.parserFactory.GetParser(serverType)
	if err != nil {
		return models.ServerQueryResult{}, err
	}

	log.Infof("Querying %s server: %s", serverType, hostname)
	result := parser.Query(hostname)

	// Store results in database
	if result.Error == nil && len(result.Features) > 0 {
		if err := s.storeFeatures(result.Features); err != nil {
			log.Errorf("Failed to store features: %v", err)
		}

		if err := s.recordUsage(result.Features); err != nil {
			log.Errorf("Failed to record usage: %v", err)
		}
	}

	return result, result.Error
}

func (s *LicenseService) storeFeatures(features []models.Feature) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT OR REPLACE INTO features
		(server_hostname, name, version, vendor_daemon, total_licenses, used_licenses, expiration_date, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, feature := range features {
		_, err := tx.Exec(query,
			feature.ServerHostname,
			feature.Name,
			feature.Version,
			feature.VendorDaemon,
			feature.TotalLicenses,
			feature.UsedLicenses,
			feature.ExpirationDate,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert feature %s: %w", feature.Name, err)
		}
	}

	return tx.Commit()
}

func (s *LicenseService) recordUsage(features []models.Feature) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	date := now.Format("2006-01-02")
	timeStr := now.Format("15:04:00")

	query := `
		INSERT OR IGNORE INTO feature_usage
		(server_hostname, feature_name, date, time, users_count)
		VALUES (?, ?, ?, ?, ?)
	`

	for _, feature := range features {
		_, err := tx.Exec(query,
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

func (s *LicenseService) GetFeatures(hostname string) ([]models.Feature, error) {
	var features []models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? ORDER BY name`
	err := s.db.Select(&features, query, hostname)
	return features, err
}

func (s *LicenseService) GetExpiringFeatures(days int) ([]models.Feature, error) {
	var features []models.Feature
	cutoff := time.Now().AddDate(0, 0, days)

	query := `
		SELECT * FROM features
		WHERE expiration_date <= ? AND expiration_date > ?
		ORDER BY expiration_date ASC
	`
	err := s.db.Select(&features, query, cutoff, time.Now())
	return features, err
}

func (s *LicenseService) GetFeatureUsageHistory(hostname, featureName string, days int) ([]models.FeatureUsage, error) {
	var usage []models.FeatureUsage
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT * FROM feature_usage
		WHERE server_hostname = ? AND feature_name = ? AND date >= ?
		ORDER BY date DESC, time DESC
	`
	err := s.db.Select(&usage, query, hostname, featureName, cutoff)
	return usage, err
}
