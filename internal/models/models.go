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
	ID              int64     `db:"id" json:"id"`
	ServerHostname  string    `db:"server_hostname" json:"server_hostname"`
	Name            string    `db:"name" json:"name"`
	Version         string    `db:"version" json:"version"`
	VendorDaemon    string    `db:"vendor_daemon" json:"vendor_daemon"`
	TotalLicenses   int       `db:"total_licenses" json:"total_licenses"`
	UsedLicenses    int       `db:"used_licenses" json:"used_licenses"`
	ExpirationDate  time.Time `db:"expiration_date" json:"expiration_date"`
	DaysToExpire    int       `json:"days_to_expire"`
	LastUpdated     time.Time `db:"last_updated" json:"last_updated"`
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
	ID             int64     `db:"id" json:"id"`
	ServerHostname string    `db:"server_hostname" json:"server_hostname"`
	FeatureName    string    `db:"feature_name" json:"feature_name"`
	AlertType      string    `db:"alert_type" json:"alert_type"` // expiration, down, denial
	Message        string    `db:"message" json:"message"`
	Severity       string    `db:"severity" json:"severity"` // info, warning, critical
	Sent           bool      `db:"sent" json:"sent"`
	SentAt         *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// LicenseEvent represents a license checkout or denial event
type LicenseEvent struct {
	ID             int64     `db:"id" json:"id"`
	Date           time.Time `db:"event_date" json:"event_date"`
	Time           time.Time `db:"event_time" json:"event_time"`
	EventType      string    `db:"event_type" json:"event_type"` // IN, OUT, DENIED
	FeatureName    string    `db:"feature_name" json:"feature_name"`
	Username       string    `db:"username" json:"username"`
	Reason         string    `db:"reason" json:"reason"`
}

// ServerQueryResult represents the result of querying a license server
type ServerQueryResult struct {
	Status      ServerStatus
	Features    []Feature
	Users       []LicenseUser
	Error       error
}
