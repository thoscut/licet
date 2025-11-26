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

	"github.com/thoscut/licet/internal/models"
	log "github.com/sirupsen/logrus"
)

type FlexLMParser struct {
	lmutilPath string
}

func NewFlexLMParser(lmutilPath string) *FlexLMParser {
	if lmutilPath == "" {
		lmutilPath = "/usr/local/bin/lmutil"
	}
	return &FlexLMParser{lmutilPath: lmutilPath}
}

func (p *FlexLMParser) Query(hostname string) models.ServerQueryResult {
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname:    hostname,
			Service:     "down",
			LastChecked: time.Now(),
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	// Execute lmstat command
	cmd := exec.Command(p.lmutilPath, "lmstat", "-i", "-a", "-c", hostname)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create pipe: %w", err)
		return result
	}

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start lmstat: %w", err)
		result.Status.Message = "Failed to execute lmstat"
		return result
	}

	// Parse output
	p.parseOutput(stdout, &result)

	if err := cmd.Wait(); err != nil {
		log.Debugf("lmstat command finished with error: %v", err)
	}

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

	// Feature patterns
	featureRe := regexp.MustCompile(`users of\s+(.+?):\s+\(Total of (\d+) license[s]? issued;\s+Total of (\d+) license[s]? in use\)`)
	uncountedRe := regexp.MustCompile(`users of\s+(.+?):\s+\(uncounted, node-locked\)`)

	// Expiration pattern
	expirationRe := regexp.MustCompile(`(\w+)\s+(\d+|\d+\.\d+)\s+(\d+)\s+(\d+-\w+-\d+)\s+(\w+)`)

	// User pattern
	userRe := regexp.MustCompile(`\s+(.+?)\s+(.+?)\s+(.+?)\s+\(v\d+\.\d+\).*start\s+(\w+\s+\d+/\d+\s+\d+:\d+)`)

	currentFeature := ""
	featureMap := make(map[string]*models.Feature)

	for scanner.Scan() {
		line := scanner.Text()

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

		// Parse features
		if matches := featureRe.FindStringSubmatch(line); matches != nil {
			featureName := strings.TrimSpace(matches[1])
			total, _ := strconv.Atoi(matches[2])
			used, _ := strconv.Atoi(matches[3])

			if _, exists := featureMap[featureName]; !exists {
				featureMap[featureName] = &models.Feature{
					ServerHostname: result.Status.Hostname,
					Name:           featureName,
					TotalLicenses:  0,
					UsedLicenses:   0,
					LastUpdated:    time.Now(),
				}
			}
			featureMap[featureName].TotalLicenses += total
			featureMap[featureName].UsedLicenses += used
			currentFeature = featureName
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

		// Parse expiration dates
		if matches := expirationRe.FindStringSubmatch(line); matches != nil {
			featureName := matches[1]
			version := matches[2]
			numLicenses, _ := strconv.Atoi(matches[3])
			expirationStr := strings.Replace(matches[4], "-jan-0", "-jan-2036", 1)
			vendorDaemon := matches[5]

			expDate, err := time.Parse("2-Jan-2006", expirationStr)
			if err != nil {
				log.Debugf("Failed to parse expiration date '%s': %v", expirationStr, err)
				expDate = time.Now().AddDate(100, 0, 0) // Far future for permanent
			}

			if feature, exists := featureMap[featureName]; exists {
				feature.Version = version
				feature.VendorDaemon = vendorDaemon
				feature.ExpirationDate = expDate
			} else {
				featureMap[featureName] = &models.Feature{
					ServerHostname: result.Status.Hostname,
					Name:           featureName,
					Version:        version,
					VendorDaemon:   vendorDaemon,
					TotalLicenses:  numLicenses,
					ExpirationDate: expDate,
					LastUpdated:    time.Now(),
				}
			}
			continue
		}

		// Parse users
		if matches := userRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			username := strings.TrimSpace(matches[1])
			host := strings.TrimSpace(matches[2])
			checkedOutStr := strings.TrimSpace(matches[4])

			checkedOut, err := time.Parse("Mon 1/2 15:04", checkedOutStr)
			if err != nil {
				// Try current year
				checkedOut, _ = time.Parse("Mon 1/2 15:04", checkedOutStr)
				if checkedOut.Year() == 0 {
					checkedOut = checkedOut.AddDate(time.Now().Year(), 0, 0)
				}
			}

			result.Users = append(result.Users, models.LicenseUser{
				ServerHostname: result.Status.Hostname,
				FeatureName:    currentFeature,
				Username:       username,
				Host:           host,
				CheckedOutAt:   checkedOut,
			})
		}
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
