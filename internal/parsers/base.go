package parsers

import (
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"licet/internal/models"
)

// PermanentExpirationDate is used for permanent licenses to prevent duplicate records
var PermanentExpirationDate = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

// NewServerQueryResult creates a new ServerQueryResult with default values
func NewServerQueryResult(hostname string) models.ServerQueryResult {
	return models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname:    hostname,
			Service:     "down",
			LastChecked: time.Now(),
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}
}

// ExecuteCommand executes a license server command and returns the output
// It logs the command and output at debug level
func ExecuteCommand(serverType, binaryPath string, args ...string) ([]byte, error) {
	cmd := exec.Command(binaryPath, args...)

	// Log command execution at debug level
	log.Debugf("Executing %s command: %s %s", serverType, binaryPath, strings.Join(args, " "))

	// Capture both stdout and stderr for debug logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("%s command finished with error: %v", serverType, err)
	}

	// Log raw output at debug level
	if log.IsLevelEnabled(log.DebugLevel) && len(output) > 0 {
		log.Debugf("%s command output:\n%s", serverType, string(output))
	}

	return output, err
}

// ParseExpirationDate parses an expiration date string and returns a time.Time
// Handles "permanent" and various date formats
// Returns PermanentExpirationDate (2099-01-01) for permanent licenses or unparseable dates
func ParseExpirationDate(expirationStr string) time.Time {
	if expirationStr == "" {
		return PermanentExpirationDate
	}

	expirationStr = strings.TrimSpace(expirationStr)

	// Handle permanent licenses
	if strings.ToLower(expirationStr) == "permanent" {
		return PermanentExpirationDate
	}

	// Handle special FlexLM date formats (matching PHP behavior)
	expirationStr = strings.Replace(expirationStr, "-jan-0000", "-jan-2036", 1)
	expirationStr = strings.Replace(expirationStr, "-jan-0", "-jan-2036", 1)

	// Try common date formats
	formats := []string{
		"2-Jan-2006",     // 1-Jan-2025
		"02-Jan-2006",    // 01-Jan-2025
		"2006-01-02",     // 2025-01-01
		"01/02/2006",     // 01/02/2025
		"1/2/2006",       // 1/2/2025
		"Jan 2, 2006",    // Jan 1, 2025
		"January 2 2006", // January 1 2025
	}

	for _, format := range formats {
		if expDate, err := time.Parse(format, expirationStr); err == nil {
			return expDate
		}
	}

	log.Debugf("Failed to parse expiration date '%s', using permanent date", expirationStr)
	return PermanentExpirationDate
}

// AdjustCheckoutTimeToCurrentYear takes a parsed time without year and adjusts it to the current year
// This is needed because most license servers don't include year in checkout times
func AdjustCheckoutTimeToCurrentYear(t time.Time) time.Time {
	now := time.Now()
	return time.Date(now.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		t.Nanosecond(), time.Local)
}

// FeatureMapToSlice converts a feature map to a slice
func FeatureMapToSlice(featureMap map[string]*models.Feature) []models.Feature {
	features := make([]models.Feature, 0, len(featureMap))
	for _, feature := range featureMap {
		features = append(features, *feature)
	}
	return features
}
