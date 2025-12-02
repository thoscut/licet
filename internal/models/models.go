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
