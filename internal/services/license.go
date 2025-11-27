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
	binPaths := map[string]string{
		"lmutil":      "/usr/local/bin/lmutil",
		"rlmstat":     "/usr/local/bin/rlmutil",
		"spmstat":     "/usr/local/bin/spmstat",
		"sesictrl":    "/usr/local/bin/sesictrl",
		"rvlstatus":   "/usr/local/bin/rvlstatus",
		"tlm_server":  "/usr/local/bin/tlm_server",
		"pixar_query": "/usr/local/bin/pixar_query.sh",
	}

	return &LicenseService{
		db:            db,
		cfg:           cfg,
		parserFactory: parsers.NewParserFactory(binPaths),
	}
}

func (s *LicenseService) GetAllServers() ([]models.LicenseServer, error) {
	var servers []models.LicenseServer

	// Get servers from database
	query := `SELECT * FROM servers ORDER BY hostname`
	dbServers := []models.LicenseServer{}
	if err := s.db.Select(&dbServers, query); err == nil {
		servers = append(servers, dbServers...)
	}

	// Also include configured servers from config (for backward compatibility)
	for _, srv := range s.cfg.Servers {
		// Check if this server already exists in database
		exists := false
		for _, dbSrv := range servers {
			if dbSrv.Hostname == srv.Hostname {
				exists = true
				break
			}
		}
		if !exists {
			servers = append(servers, models.LicenseServer{
				Hostname:    srv.Hostname,
				Description: srv.Description,
				Type:        srv.Type,
				CactiID:     srv.CactiID,
				WebUI:       srv.WebUI,
			})
		}
	}

	return servers, nil
}

// AddServer adds a new license server to the database
func (s *LicenseService) AddServer(server models.LicenseServer) error {
	query := `
		INSERT INTO servers (hostname, description, type, cacti_id, webui, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := s.db.Exec(query,
		server.Hostname,
		server.Description,
		server.Type,
		server.CactiID,
		server.WebUI,
	)
	if err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}
	return nil
}

// DeleteServer removes a license server from the database
func (s *LicenseService) DeleteServer(hostname string) error {
	query := `DELETE FROM servers WHERE hostname = ?`
	_, err := s.db.Exec(query, hostname)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}
	return nil
}

// UpdateServer updates an existing license server in the database
func (s *LicenseService) UpdateServer(server models.LicenseServer) error {
	query := `
		UPDATE servers
		SET description = ?, type = ?, cacti_id = ?, webui = ?, updated_at = CURRENT_TIMESTAMP
		WHERE hostname = ?
	`
	_, err := s.db.Exec(query,
		server.Description,
		server.Type,
		server.CactiID,
		server.WebUI,
		server.Hostname,
	)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}
	return nil
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
