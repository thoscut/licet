package parsers

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/models"
	"github.com/thoscut/licet/internal/util"
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
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname:    hostname,
			Service:     "down",
			LastChecked: time.Now(),
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	cmd := exec.Command(p.rlmstatPath, "rlmstat", "-a", "-c", hostname)

	// Log command execution at debug level
	log.Debugf("Executing RLM command: %s %s", p.rlmstatPath, strings.Join(cmd.Args[1:], " "))

	// Capture both stdout and stderr for debug logging
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("rlmstat command finished with error: %v", err)
	}

	// Log raw output at debug level
	if log.IsLevelEnabled(log.DebugLevel) && len(output) > 0 {
		log.Debugf("RLM command output for %s:\n%s", hostname, string(output))
	}

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
	userRe := regexp.MustCompile(`^([\w\+]+)\s+v[\d\.]+:\s+([\w\.\-]+@[\w\-]+)\s+\d+\/\d+\s+at\s+(\d+\/\d+\s+\d+:\d+)`)

	// List of known utility/command names that should not be treated as features
	excludedNames := map[string]bool{
		"rlm":      true,
		"rlmutil":  true,
		"rlmstat":  true,
		"rlmdown":  true,
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
			expirationStr := matches[3]

			var expDate time.Time
			if expirationStr == "permanent" {
				expDate = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
			} else {
				var err error
				expDate, err = time.Parse("2-Jan-2006", expirationStr)
				if err != nil {
					log.Debugf("Failed to parse expiration: %v", err)
					expDate = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
				}
			}

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
			expirationStr := matches[2]
			if expirationStr == "" {
				expirationStr = "permanent"
			}

			var expDate time.Time
			if expirationStr == "permanent" {
				expDate = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
			} else {
				var err error
				expDate, err = time.Parse("2-Jan-2006", expirationStr)
				if err != nil {
					log.Debugf("Failed to parse expiration: %v", err)
					expDate = time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
				}
			}

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
			userHost := matches[2]
			parts := strings.Split(userHost, "@")
			username := parts[0]
			host := ""
			if len(parts) > 1 {
				host = parts[1]
			}

			checkedOutStr := matches[3]
			checkedOut, err := time.Parse("01/02 15:04", checkedOutStr)
			if err != nil {
				log.Debugf("Failed to parse RLM checkout time '%s': %v", checkedOutStr, err)
				continue
			}

			// Since RLM doesn't include year, we need to set it to current year
			now := time.Now()
			checkedOut = time.Date(now.Year(), checkedOut.Month(), checkedOut.Day(),
				checkedOut.Hour(), checkedOut.Minute(), checkedOut.Second(),
				checkedOut.Nanosecond(), time.Local)

			result.Users = append(result.Users, models.LicenseUser{
				ServerHostname: result.Status.Hostname,
				FeatureName:    featureName,
				Username:       username,
				Host:           host,
				CheckedOutAt:   checkedOut,
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
