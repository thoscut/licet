package models

import (
	"time"
)

// LicenseServer represents a configured license server
type LicenseServer struct {
	ID          int64     `db:"id" json:"id"`
	Hostname    string    `db:"hostname" json:"hostname"`
	Description string    `db:"description" json:"description"`
	Type        string    `db:"type" json:"type"`
	CactiID     string    `db:"cacti_id" json:"cacti_id,omitempty"`
	WebUI       string    `db:"webui" json:"webui,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ServerStatus represents the current status of a license server
type ServerStatus struct {
	Hostname    string    `json:"hostname"`
	Service     string    `json:"service"` // up, down, warning
	Master      string    `json:"master"`
	Version     string    `json:"version"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

// Feature represents a license feature
type Feature struct {
	ID             int64     `db:"id" json:"id"`
	ServerHostname string    `db:"server_hostname" json:"server_hostname"`
	Name           string    `db:"name" json:"name"`
	Version        string    `db:"version" json:"version"`
	VendorDaemon   string    `db:"vendor_daemon" json:"vendor_daemon"`
	TotalLicenses  int       `db:"total_licenses" json:"total_licenses"`
	UsedLicenses   int       `db:"used_licenses" json:"used_licenses"`
	ExpirationDate time.Time `db:"expiration_date" json:"expiration_date"`
	DaysToExpire   int       `json:"days_to_expire"`
	LastUpdated    time.Time `db:"last_updated" json:"last_updated"`
}

// AvailableLicenses returns the number of available (unused) licenses
func (f *Feature) AvailableLicenses() int {
	available := f.TotalLicenses - f.UsedLicenses
	if available < 0 {
		return 0
	}
	return available
}

// DaysToExpiration returns the number of days until the license expires
// Returns negative number if already expired
func (f *Feature) DaysToExpiration() int {
	duration := time.Until(f.ExpirationDate)
	days := int(duration.Hours() / 24)
	return days
}

// FeatureUsage represents historical usage data
type FeatureUsage struct {
	ID             int64     `db:"id" json:"id"`
	ServerHostname string    `db:"server_hostname" json:"server_hostname"`
	FeatureName    string    `db:"feature_name" json:"feature_name"`
	Date           time.Time `db:"date" json:"date"`
	Time           time.Time `db:"time" json:"time"`
	UsersCount     int       `db:"users_count" json:"users_count"`
}

// LicenseUser represents a user currently using a license
type LicenseUser struct {
	ServerHostname string    `json:"server_hostname"`
	FeatureName    string    `json:"feature_name"`
	Username       string    `json:"username"`
	Host           string    `json:"host"`
	CheckedOutAt   time.Time `json:"checked_out_at"`
	Version        string    `json:"version,omitempty"`
	Display        string    `json:"display,omitempty"`
}

// Alert represents a license alert
type Alert struct {
	ID             int64      `db:"id" json:"id"`
	ServerHostname string     `db:"server_hostname" json:"server_hostname"`
	FeatureName    string     `db:"feature_name" json:"feature_name"`
	AlertType      string     `db:"alert_type" json:"alert_type"` // expiration, down, denial
	Message        string     `db:"message" json:"message"`
	Severity       string     `db:"severity" json:"severity"` // info, warning, critical
	Sent           bool       `db:"sent" json:"sent"`
	SentAt         *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// LicenseEvent represents a license checkout or denial event
type LicenseEvent struct {
	ID          int64     `db:"id" json:"id"`
	Date        time.Time `db:"event_date" json:"event_date"`
	Time        time.Time `db:"event_time" json:"event_time"`
	EventType   string    `db:"event_type" json:"event_type"` // IN, OUT, DENIED
	FeatureName string    `db:"feature_name" json:"feature_name"`
	Username    string    `db:"username" json:"username"`
	Reason      string    `db:"reason" json:"reason"`
}

// ServerQueryResult represents the result of querying a license server
type ServerQueryResult struct {
	Status   ServerStatus
	Features []Feature
	Users    []LicenseUser
	Error    error
}

// UtilizationData represents current utilization for a feature
type UtilizationData struct {
	ServerHostname    string  `json:"server_hostname" db:"server_hostname"`
	FeatureName       string  `json:"feature_name" db:"feature_name"`
	Version           string  `json:"version" db:"version"`
	TotalLicenses     int     `json:"total_licenses" db:"total_licenses"`
	UsedLicenses      int     `json:"used_licenses" db:"used_licenses"`
	AvailableLicenses int     `json:"available_licenses" db:"available_licenses"`
	UtilizationPct    float64 `json:"utilization_pct" db:"utilization_pct"`
	VendorDaemon      string  `json:"vendor_daemon" db:"vendor_daemon"`
}

// UtilizationHistoryPoint represents a single data point in utilization history
type UtilizationHistoryPoint struct {
	Timestamp  string `json:"timestamp" db:"timestamp"`
	UsersCount int    `json:"users_count" db:"users_count"`
}

// UtilizationStats represents aggregated statistics for a feature
type UtilizationStats struct {
	ServerHostname string  `json:"server_hostname" db:"server_hostname"`
	FeatureName    string  `json:"feature_name" db:"feature_name"`
	AvgUsage       float64 `json:"avg_usage" db:"avg_usage"`
	PeakUsage      int     `json:"peak_usage" db:"peak_usage"`
	MinUsage       int     `json:"min_usage" db:"min_usage"`
	TotalLicenses  int     `json:"total_licenses" db:"total_licenses"`
}

// HeatmapData represents hour-of-day usage patterns for heatmap visualization
type HeatmapData struct {
	ServerHostname string          `json:"server_hostname"`
	FeatureName    string          `json:"feature_name"`
	HourlyData     []HeatmapHourly `json:"hourly_data"`
}

// HeatmapHourly represents usage data for a specific hour
type HeatmapHourly struct {
	Hour      int     `json:"hour" db:"hour"` // 0-23
	AvgUsage  float64 `json:"avg_usage" db:"avg_usage"`
	PeakUsage int     `json:"peak_usage" db:"peak_usage"`
}

// PredictiveAnalytics represents forecast data for a feature
type PredictiveAnalytics struct {
	ServerHostname  string             `json:"server_hostname"`
	FeatureName     string             `json:"feature_name"`
	TotalLicenses   int                `json:"total_licenses"`
	CurrentUsage    float64            `json:"current_usage"`
	TrendSlope      float64            `json:"trend_slope"`      // Licenses per day
	DaysToCapacity  int                `json:"days_to_capacity"` // -1 if decreasing or no risk
	ConfidenceLevel float64            `json:"confidence_level"` // 0.0 to 1.0
	Forecast        []ForecastPoint    `json:"forecast"`
	Anomalies       []AnomalyDetection `json:"anomalies"`
}

// ForecastPoint represents a single prediction point
type ForecastPoint struct {
	Date           string  `json:"date"`
	PredictedUsage float64 `json:"predicted_usage"`
}

// AnomalyDetection represents detected anomalies in usage patterns
type AnomalyDetection struct {
	Date      string  `json:"date"`
	Usage     int     `json:"usage"`
	Expected  float64 `json:"expected"`
	Deviation float64 `json:"deviation"` // Standard deviations from mean
	Severity  string  `json:"severity"`  // low, medium, high
}

// EnhancedStatistics represents comprehensive statistics for license usage
type EnhancedStatistics struct {
	ServerHostname string `json:"server_hostname"`
	FeatureName    string `json:"feature_name"`
	Period         string `json:"period"` // e.g., "30 days"

	// Basic statistics
	TotalLicenses int     `json:"total_licenses"`
	AvgUsage      float64 `json:"avg_usage"`
	MedianUsage   float64 `json:"median_usage"`
	PeakUsage     int     `json:"peak_usage"`
	MinUsage      int     `json:"min_usage"`
	StdDev        float64 `json:"std_dev"`

	// Utilization metrics
	AvgUtilizationPct  float64 `json:"avg_utilization_pct"`
	PeakUtilizationPct float64 `json:"peak_utilization_pct"`

	// Trend analysis
	TrendDirection string  `json:"trend_direction"` // "increasing", "decreasing", "stable"
	TrendSlope     float64 `json:"trend_slope"`     // Licenses per day
	TrendStrength  string  `json:"trend_strength"`  // "strong", "moderate", "weak"

	// Moving averages
	MovingAvg7Day  float64 `json:"moving_avg_7day"`
	MovingAvg30Day float64 `json:"moving_avg_30day"`

	// Time patterns
	PeakHour    int     `json:"peak_hour"`     // 0-23
	PeakDayOfWeek int   `json:"peak_day_of_week"` // 0=Sunday, 6=Saturday
	WeekdayAvg  float64 `json:"weekday_avg"`
	WeekendAvg  float64 `json:"weekend_avg"`

	// Efficiency metrics
	EfficiencyScore    float64 `json:"efficiency_score"`    // 0-100
	UnderutilizedHours int     `json:"underutilized_hours"` // Hours with <10% utilization

	// Recommendations
	Recommendations []Recommendation `json:"recommendations"`
}

// Recommendation represents a license optimization recommendation
type Recommendation struct {
	Type        string `json:"type"`        // "reduce", "increase", "redistribute", "alert"
	Priority    string `json:"priority"`    // "high", "medium", "low"
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`      // Expected impact of following the recommendation
}

// TrendAnalysis represents detailed trend analysis for a feature
type TrendAnalysis struct {
	ServerHostname string  `json:"server_hostname"`
	FeatureName    string  `json:"feature_name"`
	Period         int     `json:"period_days"`

	// Linear regression results
	Slope     float64 `json:"slope"`
	Intercept float64 `json:"intercept"`
	RSquared  float64 `json:"r_squared"` // Goodness of fit

	// Trend interpretation
	Direction      string  `json:"direction"`       // "increasing", "decreasing", "stable"
	ChangePerDay   float64 `json:"change_per_day"`
	ChangePerWeek  float64 `json:"change_per_week"`
	ChangePerMonth float64 `json:"change_per_month"`

	// Projections
	ProjectedUsage7Days  float64 `json:"projected_usage_7_days"`
	ProjectedUsage30Days float64 `json:"projected_usage_30_days"`
	ProjectedUsage90Days float64 `json:"projected_usage_90_days"`

	// Capacity warnings
	DaysToCapacity   int  `json:"days_to_capacity"`   // -1 if not applicable
	CapacityAtRisk   bool `json:"capacity_at_risk"`
	RecommendedAction string `json:"recommended_action"`
}

// SeasonalPattern represents detected seasonal patterns in usage
type SeasonalPattern struct {
	ServerHostname string            `json:"server_hostname"`
	FeatureName    string            `json:"feature_name"`
	HourlyPattern  []float64         `json:"hourly_pattern"`   // 24 values for each hour
	DailyPattern   []float64         `json:"daily_pattern"`    // 7 values for each day of week
	HasDailyCycle  bool              `json:"has_daily_cycle"`
	HasWeeklyCycle bool              `json:"has_weekly_cycle"`
	PeakPeriods    []PeakPeriod      `json:"peak_periods"`
}

// PeakPeriod represents a detected peak usage period
type PeakPeriod struct {
	StartHour int     `json:"start_hour"`
	EndHour   int     `json:"end_hour"`
	DaysOfWeek []int  `json:"days_of_week"` // 0=Sunday, 6=Saturday
	AvgUsage  float64 `json:"avg_usage"`
	Label     string  `json:"label"` // e.g., "Business Hours", "Morning Peak"
}

// CapacityInsight represents utilization insight for a single feature
type CapacityInsight struct {
	ServerHostname    string  `json:"server_hostname"`
	FeatureName       string  `json:"feature_name"`
	TotalLicenses     int     `json:"total_licenses"`
	AvgUsage          float64 `json:"avg_usage"`
	PeakUsage         int     `json:"peak_usage"`
	UtilizationPct    float64 `json:"utilization_pct"`
	TrendSlope        float64 `json:"trend_slope"`
	DaysToCapacity    int     `json:"days_to_capacity"`
	Recommendation    string  `json:"recommendation"`
}

// CapacityPlanningReport represents capacity planning insights
type CapacityPlanningReport struct {
	GeneratedAt    string `json:"generated_at"`
	PeriodAnalyzed int    `json:"period_analyzed_days"`

	// Overall summary
	TotalServers          int `json:"total_servers"`
	TotalFeatures         int `json:"total_features"`
	FeaturesAtCapacity    int `json:"features_at_capacity"`
	FeaturesUnderutilized int `json:"features_underutilized"`

	// Detailed analysis
	HighUtilization []CapacityInsight `json:"high_utilization"` // >80%
	LowUtilization  []CapacityInsight `json:"low_utilization"`  // <20%
	TrendingUp      []CapacityInsight `json:"trending_up"`
	TrendingDown    []CapacityInsight `json:"trending_down"`

	// Summary recommendations
	Recommendations []Recommendation `json:"recommendations"`
}
