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

	// Log query results at debug level
	if result.Error != nil {
		log.Debugf("Query error for %s: %v", hostname, result.Error)
	} else {
		log.Debugf("Query successful for %s: service=%s, features=%d, users=%d",
			hostname, result.Status.Service, len(result.Features), len(result.Users))
	}

	// Store results in database
	if result.Error == nil && len(result.Features) > 0 {
		log.Debugf("Storing %d features from %s to database", len(result.Features), hostname)
		if err := s.storeFeatures(result.Features); err != nil {
			log.Errorf("Failed to store features: %v", err)
		} else {
			log.Debugf("Successfully stored features from %s", hostname)
		}

		log.Debugf("Recording usage data for %d features from %s", len(result.Features), hostname)
		if err := s.recordUsage(result.Features); err != nil {
			log.Errorf("Failed to record usage: %v", err)
		} else {
			log.Debugf("Successfully recorded usage from %s", hostname)
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

// GetFeaturesWithExpiration returns features that have expiration dates, deduplicated
func (s *LicenseService) GetFeaturesWithExpiration(hostname string) ([]models.Feature, error) {
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

// GetCurrentUtilization returns current utilization for all features across all servers
func (s *LicenseService) GetCurrentUtilization(serverFilter string) ([]models.UtilizationData, error) {
	var utilization []models.UtilizationData

	query := `
		SELECT
			server_hostname,
			name as feature_name,
			total_licenses,
			used_licenses,
			(total_licenses - used_licenses) as available_licenses,
			CASE
				WHEN total_licenses > 0 THEN (used_licenses * 100.0 / total_licenses)
				ELSE 0
			END as utilization_pct,
			vendor_daemon
		FROM features
		WHERE total_licenses > 0
	`

	args := []interface{}{}
	if serverFilter != "" {
		query += " AND server_hostname = ?"
		args = append(args, serverFilter)
	}

	query += " ORDER BY utilization_pct DESC, feature_name ASC"

	err := s.db.Select(&utilization, query, args...)
	return utilization, err
}

// GetUtilizationHistory returns time-series usage data for charting
func (s *LicenseService) GetUtilizationHistory(server, feature string, days int) ([]models.UtilizationHistoryPoint, error) {
	var history []models.UtilizationHistoryPoint
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			datetime(date || ' ' || time) as timestamp,
			users_count
		FROM feature_usage
		WHERE 1=1
	`

	args := []interface{}{}
	if server != "" {
		query += " AND server_hostname = ?"
		args = append(args, server)
	}
	if feature != "" {
		query += " AND feature_name = ?"
		args = append(args, feature)
	}

	query += " AND date >= ? ORDER BY date ASC, time ASC"
	args = append(args, cutoff.Format("2006-01-02"))

	err := s.db.Select(&history, query, args...)
	return history, err
}

// GetUtilizationStats returns aggregated statistics
func (s *LicenseService) GetUtilizationStats(server string, days int) ([]models.UtilizationStats, error) {
	var stats []models.UtilizationStats
	cutoff := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			fu.server_hostname,
			fu.feature_name,
			AVG(fu.users_count) as avg_usage,
			MAX(fu.users_count) as peak_usage,
			MIN(fu.users_count) as min_usage,
			(SELECT total_licenses FROM features
			 WHERE server_hostname = fu.server_hostname
			   AND name = fu.feature_name
			 LIMIT 1) as total_licenses
		FROM feature_usage fu
		WHERE fu.date >= ?
	`

	args := []interface{}{cutoff.Format("2006-01-02")}
	if server != "" {
		query += " AND fu.server_hostname = ?"
		args = append(args, server)
	}

	query += " GROUP BY fu.server_hostname, fu.feature_name ORDER BY avg_usage DESC"

	err := s.db.Select(&stats, query, args...)
	return stats, err
}

// GetHeatmapData returns hour-of-day usage patterns for heatmap visualization
func (s *LicenseService) GetHeatmapData(server string, days int) ([]models.HeatmapData, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	// First, get all unique features
	featuresQuery := `
		SELECT DISTINCT server_hostname, feature_name
		FROM feature_usage
		WHERE date >= ?
	`
	args := []interface{}{cutoff.Format("2006-01-02")}
	if server != "" {
		featuresQuery += " AND server_hostname = ?"
		args = append(args, server)
	}

	type featureKey struct {
		ServerHostname string `db:"server_hostname"`
		FeatureName    string `db:"feature_name"`
	}
	var features []featureKey
	if err := s.db.Select(&features, featuresQuery, args...); err != nil {
		return nil, err
	}

	// For each feature, get hourly usage patterns
	var heatmapData []models.HeatmapData

	for _, feature := range features {
		hourlyQuery := `
			SELECT
				CAST(strftime('%H', time) AS INTEGER) as hour,
				AVG(users_count) as avg_usage,
				MAX(users_count) as peak_usage
			FROM feature_usage
			WHERE server_hostname = ?
			  AND feature_name = ?
			  AND date >= ?
			GROUP BY hour
			ORDER BY hour
		`

		var hourlyData []models.HeatmapHourly
		err := s.db.Select(&hourlyData, hourlyQuery,
			feature.ServerHostname,
			feature.FeatureName,
			cutoff.Format("2006-01-02"))

		if err != nil {
			log.Errorf("Failed to get heatmap data for %s:%s: %v",
				feature.ServerHostname, feature.FeatureName, err)
			continue
		}

		// Ensure we have data for all 24 hours (fill in zeros for missing hours)
		hourlyMap := make(map[int]models.HeatmapHourly)
		for _, h := range hourlyData {
			hourlyMap[h.Hour] = h
		}

		completeHourly := make([]models.HeatmapHourly, 24)
		for hour := 0; hour < 24; hour++ {
			if data, exists := hourlyMap[hour]; exists {
				completeHourly[hour] = data
			} else {
				completeHourly[hour] = models.HeatmapHourly{
					Hour:      hour,
					AvgUsage:  0,
					PeakUsage: 0,
				}
			}
		}

		heatmapData = append(heatmapData, models.HeatmapData{
			ServerHostname: feature.ServerHostname,
			FeatureName:    feature.FeatureName,
			HourlyData:     completeHourly,
		})
	}

	return heatmapData, nil
}
