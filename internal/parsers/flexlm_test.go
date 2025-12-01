package parsers

import (
	"strings"
	"testing"
	"time"

	"github.com/thoscut/licet/internal/models"
)

func TestFlexLMParser_ParsePermanentLicenses(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature1:  (Total of 10 licenses issued;  Total of 5 licenses in use)

License files:
feature1 1.0 10 vendor1 permanent
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if result.Status.Service != "up" {
		t.Errorf("Expected service status 'up', got '%s'", result.Status.Service)
	}

	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature, got %d", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "feature1" {
		t.Errorf("Expected feature name 'feature1', got '%s'", feature.Name)
	}

	if feature.TotalLicenses != 10 {
		t.Errorf("Expected 10 total licenses, got %d", feature.TotalLicenses)
	}

	if feature.UsedLicenses != 5 {
		t.Errorf("Expected 5 used licenses, got %d", feature.UsedLicenses)
	}

	// Check that permanent license has far future expiration
	if feature.ExpirationDate.Before(time.Now().AddDate(50, 0, 0)) {
		t.Errorf("Expected far future expiration for permanent license, got %v", feature.ExpirationDate)
	}
}

func TestFlexLMParser_ParseOldExpirationFormat(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

License files:
feature2 2.0 5 31-dec-2025 vendor2
feature3 3.0 3 15-jan-2024
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features, got %d", len(result.Features))
	}

	// Find feature2
	var feature2 *models.Feature
	for i := range result.Features {
		if result.Features[i].Name == "feature2" {
			feature2 = &result.Features[i]
			break
		}
	}

	if feature2 == nil {
		t.Fatal("feature2 not found")
	}

	if feature2.VendorDaemon != "vendor2" {
		t.Errorf("Expected vendor daemon 'vendor2', got '%s'", feature2.VendorDaemon)
	}

	expectedDate, _ := time.Parse("2-Jan-2006", "31-Dec-2025")
	if !feature2.ExpirationDate.Equal(expectedDate) {
		t.Errorf("Expected expiration date %v, got %v", expectedDate, feature2.ExpirationDate)
	}
}

func TestFlexLMParser_ParseNewExpirationFormat(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

License files:
feature4 4.0 8 vendor3 20-jun-2026
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature, got %d", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "feature4" {
		t.Errorf("Expected feature name 'feature4', got '%s'", feature.Name)
	}

	if feature.VendorDaemon != "vendor3" {
		t.Errorf("Expected vendor daemon 'vendor3', got '%s'", feature.VendorDaemon)
	}

	expectedDate, _ := time.Parse("2-Jan-2006", "20-Jun-2026")
	if !feature.ExpirationDate.Equal(expectedDate) {
		t.Errorf("Expected expiration date %v, got %v", expectedDate, feature.ExpirationDate)
	}
}

func TestFlexLMParser_CaseInsensitiveFeatures(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test with uppercase "Users of" and "Total of"
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

USERS OF feature5:  (TOTAL OF 3 licenses issued;  TOTAL OF 2 licenses in use)
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature, got %d (case-insensitive matching failed)", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "feature5" {
		t.Errorf("Expected feature name 'feature5', got '%s'", feature.Name)
	}
}

func TestFlexLMParser_FilterNoSuchFeature(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of badfeature:  (Total of 5 licenses issued;  Total of 0 licenses in use)
  No such feature exists.

Users of goodfeature:  (Total of 3 licenses issued;  Total of 2 licenses in use)
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Should only have goodfeature, badfeature should be filtered out
	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature (badfeature filtered), got %d", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "goodfeature" {
		t.Errorf("Expected feature name 'goodfeature', got '%s'", feature.Name)
	}
}

func TestFlexLMParser_DateReplacement(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test both -jan-0000 and -jan-0 replacements
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

License files:
feature6 1.0 5 vendor1 1-jan-0000
feature7 2.0 3 vendor2 1-jan-0
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features, got %d", len(result.Features))
	}

	expectedDate, _ := time.Parse("2-Jan-2006", "1-Jan-2036")

	for _, feature := range result.Features {
		if !feature.ExpirationDate.Equal(expectedDate) {
			t.Errorf("Feature %s: Expected date to be replaced to %v, got %v",
				feature.Name, expectedDate, feature.ExpirationDate)
		}
	}
}

func TestFlexLMParser_UncountedLicenses(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of nodelocked:  (uncounted, node-locked)
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature, got %d", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "nodelocked" {
		t.Errorf("Expected feature name 'nodelocked', got '%s'", feature.Name)
	}

	if feature.TotalLicenses != 9999 {
		t.Errorf("Expected uncounted to have 9999 licenses, got %d", feature.TotalLicenses)
	}
}

func TestFlexLMParser_ErrorConditions(t *testing.T) {
	testCases := []struct {
		name           string
		output         string
		expectedStatus string
		expectedMsg    string
	}{
		{
			name: "Cannot connect",
			output: `lmstat - Copyright (c) 1989-2023 Flexera.
Cannot connect to license server system`,
			expectedStatus: "down",
			expectedMsg:    "Cannot connect to",
		},
		{
			name: "Cannot read data",
			output: `lmstat - Copyright (c) 1989-2023 Flexera.
Cannot read data from license server`,
			expectedStatus: "down",
			expectedMsg:    "Cannot read data from",
		},
		{
			name: "Error getting status",
			output: `lmstat - Copyright (c) 1989-2023 Flexera.
Error getting status`,
			expectedStatus: "down",
			expectedMsg:    "Error getting status from",
		},
		{
			name: "Vendor daemon down",
			output: `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1
vendor daemon is down`,
			expectedStatus: "warning",
			expectedMsg:    "Vendor daemon is down on",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}
			result := models.ServerQueryResult{
				Status: models.ServerStatus{
					Hostname: "27000@server.example.com",
					Service:  "down",
				},
				Features: []models.Feature{},
				Users:    []models.LicenseUser{},
			}

			parser.parseOutput(strings.NewReader(tc.output), &result)

			if result.Status.Service != tc.expectedStatus {
				t.Errorf("Expected status '%s', got '%s'", tc.expectedStatus, result.Status.Service)
			}

			if !strings.Contains(result.Status.Message, tc.expectedMsg) {
				t.Errorf("Expected message to contain '%s', got '%s'", tc.expectedMsg, result.Status.Message)
			}
		})
	}
}
