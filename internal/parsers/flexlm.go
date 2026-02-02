package parsers

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"licet/internal/models"
	"licet/internal/util"
)

type FlexLMParser struct {
	lmutilPath string
}

func NewFlexLMParser(lmutilPath string) *FlexLMParser {
	if lmutilPath == "" {
		lmutilPath = util.FindBinary("lmutil")
	}
	return &FlexLMParser{lmutilPath: lmutilPath}
}

func (p *FlexLMParser) Query(hostname string) models.ServerQueryResult {
	result := NewServerQueryResult(hostname)

	// Execute lmstat command
	output, _ := ExecuteCommand("FlexLM", p.lmutilPath, "lmstat", "-i", "-a", "-c", hostname)

	// Parse output
	p.parseOutput(strings.NewReader(string(output)), &result)

	return result
}

func (p *FlexLMParser) parseOutput(reader io.Reader, result *models.ServerQueryResult) {
	scanner := bufio.NewScanner(reader)

	// Regular expressions
	serverUpRe := regexp.MustCompile(`([^\s]+):\s+license server UP.*v(\d+\.\d+\.\d+)`)
	cannotConnectRe := regexp.MustCompile(`Cannot connect to license server`)
	cannotReadRe := regexp.MustCompile(`Cannot read data`)
	errorStatusRe := regexp.MustCompile(`Error getting status`)
	vendorDownRe := regexp.MustCompile(`vendor daemon is down`)

	// Feature patterns (case-insensitive to match PHP behavior)
	featureRe := regexp.MustCompile(`(?i)users of\s+(.+?):\s+\(Total of (\d+) license[s]? issued;\s+Total of (\d+) license[s]? in use\)`)
	noFeatureRe := regexp.MustCompile(`(?i)no such feature exists`)
	uncountedRe := regexp.MustCompile(`(?i)users of\s+(.+?):\s+\(uncounted, node-locked\)`)

	// Track the last feature name to handle "no such feature exists" on next line
	var lastFeatureName string

	// Expiration patterns - matching PHP's three patterns
	// Old format: FEATURE VERSION COUNT DATE [VENDOR]
	expirationOldRe := regexp.MustCompile(`(?i)(\w+)\s+(\d+|\d+\.\d+)\s+(\d+)\s+(\d+-\w+-\d+)(?:\s+(\w+))?$`)
	// New format: FEATURE VERSION COUNT VENDOR DATE
	expirationNewRe := regexp.MustCompile(`(?i)(\w+)\s+(\d+|\d+\.\d+)\s+(\d+)\s+(\w+)\s+(\d+-\w+-\d+)$`)
	// Permanent: FEATURE VERSION COUNT VENDOR permanent
	expirationPermRe := regexp.MustCompile(`(?i)(\w+)\s+(\d+|\d+\.\d+)\s+(\d+)\s+(\w+)\s+(permanent)`)

	// User pattern - capture version for accurate per-version license counts
	userRe := regexp.MustCompile(`\s+(.+?)\s+(.+?)\s+(.+?)\s+\(v([^\)]+)\).*start\s+(\w+\s+\d+/\d+\s+\d+:\d+)`)

	currentFeature := ""
	featureMap := make(map[string]*models.Feature)
	// Track usage counts by feature name for aggregation
	usageMap := make(map[string]struct{ total, used int })

	for scanner.Scan() {
		line := scanner.Text()

		// Check for "no such feature exists" and remove the last feature if found
		if noFeatureRe.MatchString(line) && lastFeatureName != "" {
			// Remove from usageMap
			delete(usageMap, lastFeatureName)
			// Remove all features with this name from featureMap
			for key := range featureMap {
				if featureMap[key].Name == lastFeatureName {
					delete(featureMap, key)
				}
			}
			lastFeatureName = ""
			continue
		}

		// Check server status
		if matches := serverUpRe.FindStringSubmatch(line); matches != nil {
			result.Status.Service = "up"
			result.Status.Master = matches[1]
			if idx := strings.Index(matches[1], "."); idx != -1 {
				result.Status.Master = matches[1][:idx]
			}
			result.Status.Version = matches[2]
			continue
		}

		if cannotConnectRe.MatchString(line) {
			result.Status.Service = "down"
			result.Status.Message = fmt.Sprintf("Cannot connect to %s", result.Status.Hostname)
			return
		}

		if cannotReadRe.MatchString(line) {
			result.Status.Service = "down"
			result.Status.Message = fmt.Sprintf("Cannot read data from %s", result.Status.Hostname)
			return
		}

		if errorStatusRe.MatchString(line) {
			result.Status.Service = "down"
			result.Status.Message = fmt.Sprintf("Error getting status from %s", result.Status.Hostname)
			return
		}

		if vendorDownRe.MatchString(line) {
			result.Status.Service = "warning"
			result.Status.Message = fmt.Sprintf("Vendor daemon is down on %s", result.Status.Hostname)
			return
		}

		// Parse features (usage counts)
		if matches := featureRe.FindStringSubmatch(line); matches != nil {
			featureName := strings.TrimSpace(matches[1])
			total, _ := strconv.Atoi(matches[2])
			used, _ := strconv.Atoi(matches[3])

			// Store usage counts for later distribution to license pools
			usageMap[featureName] = struct{ total, used int }{total, used}
			currentFeature = featureName
			lastFeatureName = featureName // Track for "no such feature exists" check
			continue
		}

		if matches := uncountedRe.FindStringSubmatch(line); matches != nil {
			featureName := strings.TrimSpace(matches[1])
			featureMap[featureName] = &models.Feature{
				ServerHostname: result.Status.Hostname,
				Name:           featureName,
				TotalLicenses:  9999, // Uncounted
				UsedLicenses:   0,
				LastUpdated:    time.Now(),
			}
			currentFeature = featureName
			continue
		}

		// Parse expiration dates - try all three patterns (matching PHP behavior)
		var featureName, version, vendorDaemon, expirationStr string
		var numLicenses int
		var matched bool

		// Try permanent license pattern first
		if matches := expirationPermRe.FindStringSubmatch(line); matches != nil {
			featureName = matches[1]
			version = matches[2]
			numLicenses, _ = strconv.Atoi(matches[3])
			vendorDaemon = matches[4]
			expirationStr = "permanent"
			matched = true
		}

		// Try new format: FEATURE VERSION COUNT VENDOR DATE
		if !matched {
			if matches := expirationNewRe.FindStringSubmatch(line); matches != nil {
				featureName = matches[1]
				version = matches[2]
				numLicenses, _ = strconv.Atoi(matches[3])
				vendorDaemon = matches[4]
				expirationStr = matches[5]
				matched = true
			}
		}

		// Try old format: FEATURE VERSION COUNT DATE [VENDOR]
		if !matched {
			if matches := expirationOldRe.FindStringSubmatch(line); matches != nil {
				featureName = matches[1]
				version = matches[2]
				numLicenses, _ = strconv.Atoi(matches[3])
				expirationStr = matches[4]
				if len(matches) > 5 && matches[5] != "" {
					vendorDaemon = matches[5]
				}
				matched = true
			}
		}

		if matched {
			expDate := ParseExpirationDate(expirationStr)

			// Create a unique key for each license pool: name + version + expiration
			key := fmt.Sprintf("%s|%s|%s", featureName, version, expDate.Format("2006-01-02"))

			// Create feature with UsedLicenses = 0; will be updated after user parsing
			featureMap[key] = &models.Feature{
				ServerHostname: result.Status.Hostname,
				Name:           featureName,
				Version:        version,
				VendorDaemon:   vendorDaemon,
				TotalLicenses:  numLicenses,
				UsedLicenses:   0,
				ExpirationDate: expDate,
				LastUpdated:    time.Now(),
			}
			continue
		}

		// Parse users
		if matches := userRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			username := strings.TrimSpace(matches[1])
			host := strings.TrimSpace(matches[2])
			version := strings.TrimSpace(matches[4])
			checkedOutStr := strings.TrimSpace(matches[5])

			// Parse the checkout time (format: "Mon 1/2 15:04")
			checkedOut, err := time.Parse("Mon 1/2 15:04", checkedOutStr)
			if err != nil {
				log.Debugf("Failed to parse checkout time '%s': %v", checkedOutStr, err)
				continue
			}

			// Adjust to current year since FlexLM doesn't include year
			checkedOut = AdjustCheckoutTimeToCurrentYear(checkedOut)

			result.Users = append(result.Users, models.LicenseUser{
				ServerHostname: result.Status.Hostname,
				FeatureName:    currentFeature,
				Username:       username,
				Host:           host,
				CheckedOutAt:   checkedOut,
				Version:        version,
			})
		}
	}

	// Add any features from usageMap that don't have expiration data
	for featureName, usage := range usageMap {
		// Check if this feature already exists in featureMap
		hasFeature := false
		for _, feature := range featureMap {
			if feature.Name == featureName {
				hasFeature = true
				break
			}
		}

		// If not, create a feature from usage data
		if !hasFeature {
			featureMap[featureName] = &models.Feature{
				ServerHostname: result.Status.Hostname,
				Name:           featureName,
				TotalLicenses:  usage.total,
				UsedLicenses:   usage.used,
				LastUpdated:    time.Now(),
			}
		}
	}

	// Calculate used licenses per feature+version based on actual user counts
	// Count users per feature name + version
	userCountByFeatureVersion := make(map[string]int)
	userCountByFeatureName := make(map[string]int)
	hasUsersForFeature := make(map[string]bool)
	for _, user := range result.Users {
		key := fmt.Sprintf("%s|%s", user.FeatureName, user.Version)
		userCountByFeatureVersion[key]++
		userCountByFeatureName[user.FeatureName]++
		hasUsersForFeature[user.FeatureName] = true
	}

	// Update UsedLicenses for each feature based on actual user counts
	for key, feature := range featureMap {
		// Try to match by feature name + version first
		userKey := fmt.Sprintf("%s|%s", feature.Name, feature.Version)
		if count, ok := userCountByFeatureVersion[userKey]; ok {
			feature.UsedLicenses = count
		} else if feature.Version == "" {
			// For features without version (from usageMap only), use total count by feature name
			if count, ok := userCountByFeatureName[feature.Name]; ok {
				feature.UsedLicenses = count
			}
		} else if !hasUsersForFeature[feature.Name] {
			// No user checkout lines for this feature - fall back to usageMap proportional distribution
			if usage, hasUsage := usageMap[feature.Name]; hasUsage && usage.total > 0 {
				usedLicenses := (usage.used * feature.TotalLicenses) / usage.total
				if usedLicenses > feature.TotalLicenses {
					usedLicenses = feature.TotalLicenses
				}
				feature.UsedLicenses = usedLicenses
			}
		}
		featureMap[key] = feature
	}

	// Convert feature map to slice
	for _, feature := range featureMap {
		result.Features = append(result.Features, *feature)
	}

	if result.Status.Service == "" {
		result.Status.Service = "down"
		result.Status.Message = fmt.Sprintf("Unknown error from %s", result.Status.Hostname)
	}
}
