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

type RLMParser struct {
	rlmstatPath string
}

func NewRLMParser(rlmstatPath string) *RLMParser {
	if rlmstatPath == "" {
		rlmstatPath = util.FindBinary("rlmutil")
	}
	return &RLMParser{rlmstatPath: rlmstatPath}
}

func (p *RLMParser) Query(hostname string) models.ServerQueryResult {
	result := NewServerQueryResult(hostname)

	// Execute rlmstat command
	output, _ := ExecuteCommand("RLM", p.rlmstatPath, "rlmstat", "-a", "-c", hostname)

	// Parse output
	p.parseOutput(strings.NewReader(string(output)), &result)

	return result
}

func (p *RLMParser) parseOutput(reader io.Reader, result *models.ServerQueryResult) {
	scanner := bufio.NewScanner(reader)

	statusRe := regexp.MustCompile(`rlm status on\s+([^\s]+)`)
	versionRe := regexp.MustCompile(`rlm software version v([\d\.]+)`)
	isvStatusRe := regexp.MustCompile(`^(\w+)\s+\d+\s+(\w+)\s+\d+`)
	featureHeaderRe := regexp.MustCompile(`(?i)^([\w\+]+)\s+(\w[\d\.]+).*pool.*$`)
	featureLicenseRe := regexp.MustCompile(`^count:\s+(\d+)[,\s]+.*inuse:\s+(\d+)[,\s]+.*exp:\s+(\d+-\w+-\d{4}|\w+)`)
	uncountedLicenseRe := regexp.MustCompile(`^UNCOUNTED[,\s]+.*inuse:\s+(\d+)(?:[,\s]+.*exp:\s+(\d+-\w+-\d{4}|\w+))?`)
	userRe := regexp.MustCompile(`^([\w\+]+)\s+(v[\d\.]+):\s+([\w\.\-]+@[\w\-]+)\s+\d+\/\d+\s+at\s+(\d+\/\d+\s+\d+:\d+)`)

	// List of known utility/command names that should not be treated as features
	excludedNames := map[string]bool{
		"rlm":       true,
		"rlmutil":   true,
		"rlmstat":   true,
		"rlmdown":   true,
		"rlmreread": true,
	}

	currentFeature := ""
	currentVersion := ""
	currentVendor := ""
	featureMap := make(map[string]*models.Feature)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := statusRe.FindStringSubmatch(line); matches != nil {
			result.Status.Service = "up"
			result.Status.Master = matches[1]
			continue
		}

		if matches := versionRe.FindStringSubmatch(line); matches != nil {
			result.Status.Version = matches[1]
			continue
		}

		if matches := isvStatusRe.FindStringSubmatch(line); matches != nil {
			if matches[2] != "Yes" {
				result.Status.Service = "warning"
				result.Status.Message = fmt.Sprintf("ISV %s appears to be down on %s", matches[1], result.Status.Hostname)
			}
			currentVendor = matches[1]
			continue
		}

		if matches := featureHeaderRe.FindStringSubmatch(line); matches != nil {
			featureName := matches[1]
			// Skip known utility/command names
			if excludedNames[featureName] {
				log.Debugf("Skipping excluded name '%s' from feature list", featureName)
				currentFeature = ""
				currentVersion = ""
				continue
			}
			currentFeature = featureName
			currentVersion = matches[2]
			continue
		}

		if matches := featureLicenseRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			total, _ := strconv.Atoi(matches[1])
			used, _ := strconv.Atoi(matches[2])
			expDate := ParseExpirationDate(matches[3])

			featureMap[currentFeature] = &models.Feature{
				ServerHostname: result.Status.Hostname,
				Name:           currentFeature,
				Version:        currentVersion,
				VendorDaemon:   currentVendor,
				TotalLicenses:  total,
				UsedLicenses:   used,
				ExpirationDate: expDate,
				LastUpdated:    time.Now(),
			}
			continue
		}

		if matches := uncountedLicenseRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			used, _ := strconv.Atoi(matches[1])
			expDate := ParseExpirationDate(matches[2])

			featureMap[currentFeature] = &models.Feature{
				ServerHostname: result.Status.Hostname,
				Name:           currentFeature,
				Version:        currentVersion,
				VendorDaemon:   currentVendor,
				TotalLicenses:  999, // UNCOUNTED licenses
				UsedLicenses:   used,
				ExpirationDate: expDate,
				LastUpdated:    time.Now(),
			}
			continue
		}

		if matches := userRe.FindStringSubmatch(line); matches != nil {
			featureName := matches[1]
			version := matches[2]
			userHost := matches[3]
			parts := strings.Split(userHost, "@")
			username := parts[0]
			host := ""
			if len(parts) > 1 {
				host = parts[1]
			}

			checkedOutStr := matches[4]
			checkedOut, err := time.Parse("01/02 15:04", checkedOutStr)
			if err != nil {
				log.Debugf("Failed to parse RLM checkout time '%s': %v", checkedOutStr, err)
				continue
			}

			// Adjust to current year since RLM doesn't include year
			checkedOut = AdjustCheckoutTimeToCurrentYear(checkedOut)

			result.Users = append(result.Users, models.LicenseUser{
				ServerHostname: result.Status.Hostname,
				FeatureName:    featureName,
				Username:       username,
				Host:           host,
				CheckedOutAt:   checkedOut,
				Version:        version,
			})
		}
	}

	for _, feature := range featureMap {
		result.Features = append(result.Features, *feature)
	}

	if result.Status.Service == "" {
		result.Status.Service = "down"
		result.Status.Message = fmt.Sprintf("Unable to connect to %s", result.Status.Hostname)
	}
}
