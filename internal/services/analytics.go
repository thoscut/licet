package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"licet/internal/database"
	"licet/internal/models"
)

// AnalyticsService handles utilization analytics and predictive operations
type AnalyticsService struct {
	db      *sqlx.DB
	storage *StorageService
	dialect database.Dialect
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sqlx.DB, storage *StorageService, dbType string) *AnalyticsService {
	return &AnalyticsService{
		db:      db,
		storage: storage,
		dialect: database.NewDialect(dbType),
	}
}

// GetCurrentUtilization returns current utilization for all features across all servers
func (s *AnalyticsService) GetCurrentUtilization(ctx context.Context, serverFilter string) ([]models.UtilizationData, error) {
	var utilization []models.UtilizationData

	query := `
		SELECT
			f.server_hostname,
			f.name as feature_name,
			f.version,
			f.total_licenses,
			f.used_licenses,
			(f.total_licenses - f.used_licenses) as available_licenses,
			CASE
				WHEN f.total_licenses > 0 THEN (f.used_licenses * 100.0 / f.total_licenses)
				ELSE 0
			END as utilization_pct,
			f.vendor_daemon
		FROM features f
		INNER JOIN (
			SELECT server_hostname, name, MAX(last_updated) as latest
			FROM features
			WHERE total_licenses > 0
			GROUP BY server_hostname, name
		) latest ON f.server_hostname = latest.server_hostname
		           AND f.name = latest.name
		           AND f.last_updated = latest.latest
		WHERE f.total_licenses > 0
	`

	args := []interface{}{}
	if serverFilter != "" {
		query += " AND f.server_hostname = ?"
		args = append(args, serverFilter)
	}

	query += " ORDER BY utilization_pct DESC, feature_name ASC"

	err := s.db.SelectContext(ctx, &utilization, query, args...)
	return utilization, err
}

// GetUtilizationHistory returns time-series usage data for charting
func (s *AnalyticsService) GetUtilizationHistory(ctx context.Context, server, feature string, days int) ([]models.UtilizationHistoryPoint, error) {
	var history []models.UtilizationHistoryPoint
	cutoff := time.Now().AddDate(0, 0, -days)

	query := fmt.Sprintf(`
		SELECT
			%s as timestamp,
			users_count
		FROM feature_usage
		WHERE 1=1
	`, s.dialect.TimestampConcat())

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

	err := s.db.SelectContext(ctx, &history, query, args...)
	return history, err
}

// GetUtilizationStats returns aggregated statistics
func (s *AnalyticsService) GetUtilizationStats(ctx context.Context, server string, days int) ([]models.UtilizationStats, error) {
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

	err := s.db.SelectContext(ctx, &stats, query, args...)
	return stats, err
}

// GetHeatmapData returns hour-of-day usage patterns for heatmap visualization
func (s *AnalyticsService) GetHeatmapData(ctx context.Context, server string, days int) ([]models.HeatmapData, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	// Get all unique features
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
	if err := s.db.SelectContext(ctx, &features, featuresQuery, args...); err != nil {
		return nil, err
	}

	// For each feature, get hourly usage patterns
	var heatmapData []models.HeatmapData

	hourlyQuery := fmt.Sprintf(`
		SELECT
			%s as hour,
			AVG(users_count) as avg_usage,
			MAX(users_count) as peak_usage
		FROM feature_usage
		WHERE server_hostname = ?
		  AND feature_name = ?
		  AND date >= ?
		GROUP BY hour
		ORDER BY hour
	`, s.dialect.HourExtract())

	for _, feature := range features {
		var hourlyData []models.HeatmapHourly
		err := s.db.SelectContext(ctx, &hourlyData, hourlyQuery,
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
func (s *AnalyticsService) GetPredictiveAnalytics(ctx context.Context, server, feature string, days int) (*models.PredictiveAnalytics, error) {
	// Get historical usage data
	usageHistory, err := s.storage.GetFeatureUsageHistory(ctx, server, feature, days)
	if err != nil {
		return nil, err
	}

	if len(usageHistory) < 7 {
		return nil, fmt.Errorf("insufficient data for predictions (need at least 7 data points)")
	}

	// Get current feature info
	var currentFeature models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? AND name = ? LIMIT 1`
	err = s.db.GetContext(ctx, &currentFeature, query, server, feature)
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

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denom
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
	stdDev = math.Sqrt(variance)

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
