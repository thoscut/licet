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
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create pipe: %w", err)
		return result
	}

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start rlmstat: %w", err)
		result.Status.Message = "Failed to execute rlmstat"
		return result
	}

	p.parseOutput(stdout, &result)

	if err := cmd.Wait(); err != nil {
		log.Debugf("rlmstat command finished with error: %v", err)
	}

	return result
}

func (p *RLMParser) parseOutput(reader io.Reader, result *models.ServerQueryResult) {
	scanner := bufio.NewScanner(reader)

	statusRe := regexp.MustCompile(`rlm status on\s+([^\s]+)`)
	versionRe := regexp.MustCompile(`rlm software version v(\d+\.\d+)`)
	isvStatusRe := regexp.MustCompile(`^(\w+)\s+\d+\s+(\w+)\s+\d+`)
	featureHeaderRe := regexp.MustCompile(`^(\w+)\s+(v\d+\.\d+|\w\d+|\w\d+\.\d+)$`)
	featureLicenseRe := regexp.MustCompile(`^count:\s+(\d+)[,\s]+.*inuse:\s+(\d+)[,\s]+.*exp:\s+(\d+-\w+-\d{4}|\w+)`)
	userRe := regexp.MustCompile(`^(\w+)\s+v\d+\.\d+\s+(\w+@\w+)\s+\d+\.\d+\s+\w+\s+(\d+\.\d+\s+\d+\.\d+)`)

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
			currentFeature = matches[1]
			currentVersion = matches[2]
			continue
		}

		if matches := featureLicenseRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			total, _ := strconv.Atoi(matches[1])
			used, _ := strconv.Atoi(matches[2])
			expirationStr := matches[3]

			var expDate time.Time
			if expirationStr == "permanent" {
				expDate = time.Now().AddDate(100, 0, 0)
			} else {
				var err error
				expDate, err = time.Parse("2-Jan-2006", expirationStr)
				if err != nil {
					log.Debugf("Failed to parse expiration: %v", err)
					expDate = time.Now().AddDate(100, 0, 0)
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

		if matches := userRe.FindStringSubmatch(line); matches != nil && currentFeature != "" {
			userHost := matches[2]
			parts := strings.Split(userHost, "@")
			username := parts[0]
			host := ""
			if len(parts) > 1 {
				host = parts[1]
			}

			checkedOutStr := matches[3]
			checkedOut, _ := time.Parse("1.2 15.04", checkedOutStr)

			result.Users = append(result.Users, models.LicenseUser{
				ServerHostname: result.Status.Hostname,
				FeatureName:    currentFeature,
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
