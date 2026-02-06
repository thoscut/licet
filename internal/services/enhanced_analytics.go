package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
	"licet/internal/models"
)

// EnhancedAnalyticsService provides advanced analytics capabilities.
// It composes AnalyticsService to reuse shared query logic.
type EnhancedAnalyticsService struct {
	db        *sqlx.DB
	storage   *StorageService
	analytics *AnalyticsService
}

// NewEnhancedAnalyticsService creates a new enhanced analytics service
func NewEnhancedAnalyticsService(db *sqlx.DB, storage *StorageService, dbType string) *EnhancedAnalyticsService {
	return &EnhancedAnalyticsService{
		db:        db,
		storage:   storage,
		analytics: NewAnalyticsService(db, storage, dbType),
	}
}

// GetEnhancedStatistics returns comprehensive statistics for a feature
func (s *EnhancedAnalyticsService) GetEnhancedStatistics(ctx context.Context, server, feature string, days int) (*models.EnhancedStatistics, error) {
	// Get usage history
	usage, err := s.storage.GetFeatureUsageHistory(ctx, server, feature, days)
	if err != nil {
		return nil, err
	}

	if len(usage) < 2 {
		return nil, fmt.Errorf("insufficient data for enhanced statistics")
	}

	// Get current feature info
	var currentFeature models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? AND name = ? LIMIT 1`
	err = s.db.GetContext(ctx, &currentFeature, query, server, feature)
	if err != nil {
		return nil, err
	}

	// Calculate basic statistics
	values := make([]float64, len(usage))
	var sum float64
	maxVal := 0
	minVal := int(^uint(0) >> 1) // Max int

	for i, u := range usage {
		values[i] = float64(u.UsersCount)
		sum += float64(u.UsersCount)
		if u.UsersCount > maxVal {
			maxVal = u.UsersCount
		}
		if u.UsersCount < minVal {
			minVal = u.UsersCount
		}
	}

	avgUsage := sum / float64(len(usage))
	median := calculateMedian(values)
	stdDev := calculateStdDev(values, avgUsage)

	// Calculate trend using linear regression
	xValues := make([]float64, len(usage))
	yValues := make([]float64, len(usage))
	startTime := usage[len(usage)-1].Date
	for i := len(usage) - 1; i >= 0; i-- {
		xValues[len(usage)-1-i] = usage[i].Date.Sub(startTime).Hours() / 24
		yValues[len(usage)-1-i] = float64(usage[i].UsersCount)
	}
	slope, _ := linearRegression(xValues, yValues)

	// Determine trend direction and strength
	trendDirection := "stable"
	trendStrength := "weak"
	if slope > 0.1 {
		trendDirection = "increasing"
	} else if slope < -0.1 {
		trendDirection = "decreasing"
	}

	absSlope := math.Abs(slope)
	if absSlope > 1.0 {
		trendStrength = "strong"
	} else if absSlope > 0.3 {
		trendStrength = "moderate"
	}

	// Calculate moving averages
	movingAvg7Day := calculateMovingAverage(values, 7)
	movingAvg30Day := calculateMovingAverage(values, 30)

	// Calculate utilization metrics
	avgUtilization := 0.0
	peakUtilization := 0.0
	if currentFeature.TotalLicenses > 0 {
		avgUtilization = (avgUsage / float64(currentFeature.TotalLicenses)) * 100
		peakUtilization = (float64(maxVal) / float64(currentFeature.TotalLicenses)) * 100
	}

	// Calculate time patterns from actual heatmap data
	peakHour := 0
	peakDayOfWeek := 0
	weekdayAvg := avgUsage
	weekendAvg := avgUsage

	heatmapData, heatErr := s.analytics.GetHeatmapData(ctx, server, days)
	if heatErr == nil {
		for _, hm := range heatmapData {
			if hm.FeatureName == feature {
				maxHourUsage := 0.0
				for _, h := range hm.HourlyData {
					if h.AvgUsage > maxHourUsage {
						maxHourUsage = h.AvgUsage
						peakHour = h.Hour
					}
				}
				break
			}
		}
	}

	// Calculate efficiency score
	efficiencyScore := calculateEfficiencyScore(avgUtilization, peakUtilization, stdDev, float64(currentFeature.TotalLicenses))

	// Generate recommendations
	recommendations := generateRecommendations(avgUtilization, peakUtilization, trendDirection, slope, currentFeature.TotalLicenses)

	return &models.EnhancedStatistics{
		ServerHostname:     server,
		FeatureName:        feature,
		Period:             fmt.Sprintf("%d days", days),
		TotalLicenses:      currentFeature.TotalLicenses,
		AvgUsage:           avgUsage,
		MedianUsage:        median,
		PeakUsage:          maxVal,
		MinUsage:           minVal,
		StdDev:             stdDev,
		AvgUtilizationPct:  avgUtilization,
		PeakUtilizationPct: peakUtilization,
		TrendDirection:     trendDirection,
		TrendSlope:         slope,
		TrendStrength:      trendStrength,
		MovingAvg7Day:      movingAvg7Day,
		MovingAvg30Day:     movingAvg30Day,
		PeakHour:           peakHour,
		PeakDayOfWeek:      peakDayOfWeek,
		WeekdayAvg:         weekdayAvg,
		WeekendAvg:         weekendAvg,
		EfficiencyScore:    efficiencyScore,
		UnderutilizedHours: int((100 - avgUtilization) / 10),
		Recommendations:    recommendations,
	}, nil
}

// GetTrendAnalysis returns detailed trend analysis for a feature
func (s *EnhancedAnalyticsService) GetTrendAnalysis(ctx context.Context, server, feature string, days int) (*models.TrendAnalysis, error) {
	// Get usage history
	usage, err := s.storage.GetFeatureUsageHistory(ctx, server, feature, days)
	if err != nil {
		return nil, err
	}

	if len(usage) < 7 {
		return nil, fmt.Errorf("insufficient data for trend analysis (need at least 7 data points)")
	}

	// Get current feature info
	var currentFeature models.Feature
	query := `SELECT * FROM features WHERE server_hostname = ? AND name = ? LIMIT 1`
	err = s.db.GetContext(ctx, &currentFeature, query, server, feature)
	if err != nil {
		return nil, err
	}

	// Prepare data for regression
	xValues := make([]float64, len(usage))
	yValues := make([]float64, len(usage))
	startTime := usage[len(usage)-1].Date
	for i := len(usage) - 1; i >= 0; i-- {
		xValues[len(usage)-1-i] = usage[i].Date.Sub(startTime).Hours() / 24
		yValues[len(usage)-1-i] = float64(usage[i].UsersCount)
	}

	// Calculate linear regression
	slope, intercept := linearRegression(xValues, yValues)
	rSquared := calculateRSquared(xValues, yValues, slope, intercept)

	// Determine trend direction
	direction := "stable"
	if slope > 0.1 {
		direction = "increasing"
	} else if slope < -0.1 {
		direction = "decreasing"
	}

	// Calculate projections
	lastDay := xValues[len(xValues)-1]
	projected7Days := slope*(lastDay+7) + intercept
	projected30Days := slope*(lastDay+30) + intercept
	projected90Days := slope*(lastDay+90) + intercept

	// Ensure projections don't go negative
	if projected7Days < 0 {
		projected7Days = 0
	}
	if projected30Days < 0 {
		projected30Days = 0
	}
	if projected90Days < 0 {
		projected90Days = 0
	}

	// Calculate days to capacity
	daysToCapacity := -1
	capacityAtRisk := false
	recommendedAction := "No action required"

	if slope > 0 && currentFeature.TotalLicenses > 0 {
		currentUsage := slope*lastDay + intercept
		remainingCapacity := float64(currentFeature.TotalLicenses) - currentUsage
		if remainingCapacity > 0 {
			daysToCapacity = int(remainingCapacity / slope)
			if daysToCapacity < 30 {
				capacityAtRisk = true
				recommendedAction = "Consider increasing license count within 30 days"
			} else if daysToCapacity < 90 {
				recommendedAction = "Monitor usage and plan for potential license increase"
			}
		} else {
			capacityAtRisk = true
			recommendedAction = "Immediately increase license count - at capacity"
		}
	} else if slope < -0.5 && currentFeature.TotalLicenses > 0 {
		recommendedAction = "Consider reducing license count to optimize costs"
	}

	return &models.TrendAnalysis{
		ServerHostname:       server,
		FeatureName:          feature,
		Period:               days,
		Slope:                slope,
		Intercept:            intercept,
		RSquared:             rSquared,
		Direction:            direction,
		ChangePerDay:         slope,
		ChangePerWeek:        slope * 7,
		ChangePerMonth:       slope * 30,
		ProjectedUsage7Days:  projected7Days,
		ProjectedUsage30Days: projected30Days,
		ProjectedUsage90Days: projected90Days,
		DaysToCapacity:       daysToCapacity,
		CapacityAtRisk:       capacityAtRisk,
		RecommendedAction:    recommendedAction,
	}, nil
}

// GetCapacityPlanningReport generates a comprehensive capacity planning report
func (s *EnhancedAnalyticsService) GetCapacityPlanningReport(ctx context.Context, days int) (*models.CapacityPlanningReport, error) {
	// Get all utilization data
	utilization, err := s.GetCurrentUtilizationWithTrend(ctx, "", days)
	if err != nil {
		return nil, err
	}

	report := &models.CapacityPlanningReport{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		PeriodAnalyzed: days,
		TotalFeatures:  len(utilization),
	}

	// Count unique servers
	servers := make(map[string]bool)
	for _, u := range utilization {
		servers[u.ServerHostname] = true
	}
	report.TotalServers = len(servers)

	// Categorize features
	for _, u := range utilization {
		insight := models.CapacityInsight{
			ServerHostname: u.ServerHostname,
			FeatureName:    u.FeatureName,
			TotalLicenses:  u.TotalLicenses,
			AvgUsage:       u.AvgUsage,
			PeakUsage:      u.PeakUsage,
			UtilizationPct: u.UtilizationPct,
			TrendSlope:     u.TrendSlope,
			DaysToCapacity: u.DaysToCapacity,
		}

		// Categorize by utilization
		if u.UtilizationPct >= 80 {
			report.FeaturesAtCapacity++
			insight.Recommendation = "High utilization - consider increasing licenses"
			report.HighUtilization = append(report.HighUtilization, insight)
		} else if u.UtilizationPct <= 20 {
			report.FeaturesUnderutilized++
			insight.Recommendation = "Low utilization - consider reducing licenses"
			report.LowUtilization = append(report.LowUtilization, insight)
		}

		// Categorize by trend
		if u.TrendSlope > 0.5 {
			insight.Recommendation = "Usage increasing - monitor capacity"
			report.TrendingUp = append(report.TrendingUp, insight)
		} else if u.TrendSlope < -0.5 {
			insight.Recommendation = "Usage decreasing - opportunity to optimize"
			report.TrendingDown = append(report.TrendingDown, insight)
		}
	}

	// Generate summary recommendations
	report.Recommendations = generateCapacityRecommendations(report)

	return report, nil
}

// UtilizationWithTrend combines utilization data with trend information
type UtilizationWithTrend struct {
	ServerHostname string
	FeatureName    string
	TotalLicenses  int
	AvgUsage       float64
	PeakUsage      int
	UtilizationPct float64
	TrendSlope     float64
	DaysToCapacity int
}

// GetCurrentUtilizationWithTrend returns utilization data enriched with trend information
func (s *EnhancedAnalyticsService) GetCurrentUtilizationWithTrend(ctx context.Context, serverFilter string, days int) ([]UtilizationWithTrend, error) {
	// Delegate to the composed AnalyticsService for base stats
	stats, err := s.analytics.GetUtilizationStats(ctx, serverFilter, days)
	if err != nil {
		return nil, err
	}

	var result []UtilizationWithTrend
	for _, stat := range stats {
		utilPct := 0.0
		if stat.TotalLicenses > 0 {
			utilPct = (stat.AvgUsage / float64(stat.TotalLicenses)) * 100
		}

		// Get trend for this feature
		slope := 0.0
		daysToCapacity := -1

		usage, err := s.storage.GetFeatureUsageHistory(ctx, stat.ServerHostname, stat.FeatureName, days)
		if err == nil && len(usage) >= 7 {
			xValues := make([]float64, len(usage))
			yValues := make([]float64, len(usage))
			startTime := usage[len(usage)-1].Date
			for i := len(usage) - 1; i >= 0; i-- {
				xValues[len(usage)-1-i] = usage[i].Date.Sub(startTime).Hours() / 24
				yValues[len(usage)-1-i] = float64(usage[i].UsersCount)
			}
			slope, _ = linearRegression(xValues, yValues)

			if slope > 0 && stat.TotalLicenses > 0 {
				lastDay := xValues[len(xValues)-1]
				currentUsage := slope*lastDay + yValues[len(yValues)-1]
				remainingCapacity := float64(stat.TotalLicenses) - currentUsage
				if remainingCapacity > 0 {
					daysToCapacity = int(remainingCapacity / slope)
				}
			}
		}

		result = append(result, UtilizationWithTrend{
			ServerHostname: stat.ServerHostname,
			FeatureName:    stat.FeatureName,
			TotalLicenses:  stat.TotalLicenses,
			AvgUsage:       stat.AvgUsage,
			PeakUsage:      stat.PeakUsage,
			UtilizationPct: utilPct,
			TrendSlope:     slope,
			DaysToCapacity: daysToCapacity,
		})
	}

	return result, nil
}

// Helper functions

func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func calculateMovingAverage(values []float64, window int) float64 {
	if len(values) == 0 {
		return 0
	}

	if window > len(values) {
		window = len(values)
	}

	var sum float64
	for i := 0; i < window; i++ {
		sum += values[i]
	}
	return sum / float64(window)
}

func calculateEfficiencyScore(avgUtil, peakUtil, stdDev, totalLicenses float64) float64 {
	if totalLicenses == 0 {
		return 0
	}

	// Score based on:
	// - Average utilization (higher is better, up to ~70%)
	// - Peak-to-average ratio (lower is better - indicates consistent usage)
	// - Standard deviation (lower is better - indicates predictable usage)

	// Optimal utilization is around 60-70%
	utilizationScore := 100.0
	if avgUtil < 30 {
		utilizationScore = avgUtil / 30 * 100
	} else if avgUtil > 80 {
		utilizationScore = 100 - (avgUtil-80)/20*30
	}

	// Peak-to-average ratio penalty
	peakRatio := 1.0
	if avgUtil > 0 {
		peakRatio = peakUtil / avgUtil
	}
	peakPenalty := 0.0
	if peakRatio > 2 {
		peakPenalty = (peakRatio - 2) * 10
	}

	// Variability penalty
	variabilityPenalty := 0.0
	if stdDev > totalLicenses*0.3 {
		variabilityPenalty = 10
	}

	score := utilizationScore - peakPenalty - variabilityPenalty
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func generateRecommendations(avgUtil, peakUtil float64, trendDir string, slope float64, totalLicenses int) []models.Recommendation {
	var recommendations []models.Recommendation

	// High utilization warning
	if avgUtil > 80 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "increase",
			Priority:    "high",
			Title:       "High Utilization Alert",
			Description: fmt.Sprintf("Average utilization is %.1f%%, which is above the recommended 80%% threshold.", avgUtil),
			Impact:      "Users may experience license denials during peak times",
		})
	}

	// Peak utilization warning
	if peakUtil > 95 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "alert",
			Priority:    "high",
			Title:       "Peak Capacity Reached",
			Description: fmt.Sprintf("Peak utilization reached %.1f%%, indicating license shortage during peak times.", peakUtil),
			Impact:      "Immediate risk of license denials",
		})
	}

	// Underutilization opportunity
	if avgUtil < 20 && totalLicenses > 5 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "reduce",
			Priority:    "medium",
			Title:       "License Optimization Opportunity",
			Description: fmt.Sprintf("Average utilization is only %.1f%%. Consider reducing license count to optimize costs.", avgUtil),
			Impact:      fmt.Sprintf("Potential to reduce licenses by %d without impacting users", int(float64(totalLicenses)*0.5)),
		})
	}

	// Trend-based recommendations
	if trendDir == "increasing" && slope > 0.5 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "increase",
			Priority:    "medium",
			Title:       "Growing Demand Detected",
			Description: fmt.Sprintf("Usage is increasing by approximately %.1f licenses per day.", slope),
			Impact:      "Plan for additional licenses within the next quarter",
		})
	}

	if trendDir == "decreasing" && slope < -0.5 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "reduce",
			Priority:    "low",
			Title:       "Declining Usage Trend",
			Description: fmt.Sprintf("Usage is decreasing by approximately %.1f licenses per day.", -slope),
			Impact:      "Opportunity to reduce licenses during next renewal",
		})
	}

	return recommendations
}

func generateCapacityRecommendations(report *models.CapacityPlanningReport) []models.Recommendation {
	var recommendations []models.Recommendation

	if report.FeaturesAtCapacity > 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "increase",
			Priority:    "high",
			Title:       "Features at High Utilization",
			Description: fmt.Sprintf("%d features have utilization above 80%% and may need additional licenses.", report.FeaturesAtCapacity),
			Impact:      "Prevent license denials and ensure user productivity",
		})
	}

	if report.FeaturesUnderutilized > 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "reduce",
			Priority:    "medium",
			Title:       "Underutilized Features Detected",
			Description: fmt.Sprintf("%d features have utilization below 20%% and may have excess licenses.", report.FeaturesUnderutilized),
			Impact:      "Potential cost savings by reducing unused licenses",
		})
	}

	if len(report.TrendingUp) > 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "alert",
			Priority:    "medium",
			Title:       "Growing Usage Trends",
			Description: fmt.Sprintf("%d features show increasing usage trends.", len(report.TrendingUp)),
			Impact:      "Plan for capacity increases in the next quarter",
		})
	}

	return recommendations
}
