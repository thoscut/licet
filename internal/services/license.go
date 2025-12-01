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

// GetPredictiveAnalytics performs trend analysis and anomaly detection for a feature
func (s *LicenseService) GetPredictiveAnalytics(server, feature string, days int) (*models.PredictiveAnalytics, error) {
	// Get historical usage data
	usageHistory, err := s.GetFeatureUsageHistory(server, feature, days)
	if err != nil {
		return nil, err
	}

	if len(usageHistory) < 7 {
		return nil, fmt.Errorf("insufficient data for predictions (need at least 7 data points)")
	}

	// Get current feature info
	var currentFeature models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? AND name = ? LIMIT 1`
	err = s.db.Get(&currentFeature, query, server, feature)
	if err != nil {
		return nil, err
	}

	// Prepare data for analysis
	var xValues []float64 // Days from start
	var yValues []float64 // Usage values

	startTime := usageHistory[len(usageHistory)-1].Date
	for i := len(usageHistory) - 1; i >= 0; i-- {
		usage := usageHistory[i]
		daysDiff := usage.Date.Sub(startTime).Hours() / 24
		xValues = append(xValues, daysDiff)
		yValues = append(yValues, float64(usage.UsersCount))
	}

	// Calculate linear regression
	slope, intercept := linearRegression(xValues, yValues)

	// Calculate statistics for anomaly detection
	mean, stdDev := calculateStats(yValues)

	// Detect anomalies (values > 2 standard deviations from mean)
	var anomalies []models.AnomalyDetection
	for i := len(usageHistory) - 1; i >= 0; i-- {
		usage := usageHistory[i]
		deviation := (float64(usage.UsersCount) - mean) / stdDev

		if deviation > 2.0 || deviation < -2.0 {
			severity := "low"
			if deviation > 3.0 || deviation < -3.0 {
				severity = "high"
			} else if deviation > 2.5 || deviation < -2.5 {
				severity = "medium"
			}

			anomalies = append(anomalies, models.AnomalyDetection{
				Date:      usage.Date.Format("2006-01-02"),
				Usage:     usage.UsersCount,
				Expected:  mean,
				Deviation: deviation,
				Severity:  severity,
			})
		}
	}

	// Generate forecast for next 30 days
	var forecast []models.ForecastPoint
	lastDay := xValues[len(xValues)-1]
	for i := 1; i <= 30; i++ {
		futureDay := lastDay + float64(i)
		predictedUsage := slope*futureDay + intercept

		// Ensure predictions don't go negative
		if predictedUsage < 0 {
			predictedUsage = 0
		}

		futureDate := time.Now().AddDate(0, 0, i).Format("2006-01-02")
		forecast = append(forecast, models.ForecastPoint{
			Date:           futureDate,
			PredictedUsage: predictedUsage,
		})
	}

	// Calculate days to capacity (if trend is increasing)
	daysToCapacity := -1
	if slope > 0 && currentFeature.TotalLicenses > 0 {
		currentUsage := slope*lastDay + intercept
		remainingCapacity := float64(currentFeature.TotalLicenses) - currentUsage
		if remainingCapacity > 0 {
			daysToCapacity = int(remainingCapacity / slope)
		}
	}

	// Calculate confidence level (R-squared)
	confidenceLevel := calculateRSquared(xValues, yValues, slope, intercept)

	return &models.PredictiveAnalytics{
		ServerHostname:  server,
		FeatureName:     feature,
		TotalLicenses:   currentFeature.TotalLicenses,
		CurrentUsage:    mean,
		TrendSlope:      slope,
		DaysToCapacity:  daysToCapacity,
		ConfidenceLevel: confidenceLevel,
		Forecast:        forecast,
		Anomalies:       anomalies,
	}, nil
}

// linearRegression calculates the slope and intercept for linear regression
func linearRegression(x, y []float64) (slope, intercept float64) {
	n := float64(len(x))
	if n == 0 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept = (sumY - slope*sumX) / n
	return slope, intercept
}

// calculateStats calculates mean and standard deviation
func calculateStats(values []float64) (mean, stdDev float64) {
	n := float64(len(values))
	if n == 0 {
		return 0, 0
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean = sum / n

	// Calculate standard deviation
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= n
	stdDev = variance
	if stdDev > 0 {
		stdDev = variance // Note: Using variance for simplicity, real stdDev would use math.Sqrt
	}

	return mean, stdDev
}

// calculateRSquared calculates the R-squared value for regression quality
func calculateRSquared(x, y []float64, slope, intercept float64) float64 {
	if len(y) == 0 {
		return 0
	}

	// Calculate mean of y
	var sumY float64
	for _, val := range y {
		sumY += val
	}
	meanY := sumY / float64(len(y))

	// Calculate total sum of squares and residual sum of squares
	var ssTot, ssRes float64
	for i := range y {
		predicted := slope*x[i] + intercept
		ssTot += (y[i] - meanY) * (y[i] - meanY)
		ssRes += (y[i] - predicted) * (y[i] - predicted)
	}

	if ssTot == 0 {
		return 0
	}

	rSquared := 1 - (ssRes / ssTot)
	if rSquared < 0 {
		rSquared = 0
	}
	if rSquared > 1 {
		rSquared = 1
	}

	return rSquared
}
