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

func TestFlexLMParser_MultipleUsersOfFeatures(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature_a:  (Total of 10 licenses issued;  Total of 5 licenses in use)

Users of feature_b:  (Total of 20 licenses issued;  Total of 3 licenses in use)

Users of feature_c:  (Total of 15 licenses issued;  Total of 8 licenses in use)
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

	if len(result.Features) != 3 {
		t.Fatalf("Expected 3 features, got %d", len(result.Features))
	}

	// Check that all three features are present
	featureNames := make(map[string]bool)
	for _, f := range result.Features {
		featureNames[f.Name] = true
	}

	expectedFeatures := []string{"feature_a", "feature_b", "feature_c"}
	for _, name := range expectedFeatures {
		if !featureNames[name] {
			t.Errorf("Expected feature '%s' not found", name)
		}
	}
}

func TestFlexLMParser_RealisticMultiFeatureOutput(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Realistic output with multiple features mixed with expiration data
	output := `lmutil - Copyright (c) 1989-2018 Flexera Software LLC. All Rights Reserved.
lmstat - Copyright (c) 1989-2018 Flexera Software LLC. All Rights Reserved.
Flexible License Manager status on Tue 12/1/2025 10:00

License server status: 27000@license-server.example.com
    license-server.example.com: license server UP (MASTER) v11.16.2

Vendor daemon status (on license-server.example.com):

     myvendor: UP v11.16.2

Feature usage info:

Users of feature_alpha:  (Total of 100 licenses issued;  Total of 45 licenses in use)

  "feature_alpha" v2023.1, vendor: myvendor
  floating license

    user1 machine1 /dev/tty (v2023.1) (license-server.example.com/27000 1234), start Mon 11/30 9:00
    user2 machine2 /dev/tty (v2023.1) (license-server.example.com/27000 1235), start Mon 11/30 10:15

Users of feature_beta:  (Total of 50 licenses issued;  Total of 12 licenses in use)

  "feature_beta" v2023.1, vendor: myvendor
  floating license

    user3 machine3 /dev/tty (v2023.1) (license-server.example.com/27000 1236), start Tue 12/1 8:00

Users of feature_gamma:  (Total of 25 licenses issued;  Total of 5 licenses in use)

  "feature_gamma" v2023.1, vendor: myvendor
  floating license

    user4 machine4 /dev/tty (v2023.1) (license-server.example.com/27000 1237), start Tue 12/1 9:30

License files on license-server.example.com:
feature_alpha 2023.1 100 myvendor 31-dec-2025
feature_beta 2023.1 50 myvendor 30-jun-2026
feature_gamma 2023.1 25 myvendor permanent
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27000@license-server.example.com",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	if result.Status.Service != "up" {
		t.Errorf("Expected service status 'up', got '%s'", result.Status.Service)
	}

	if len(result.Features) != 3 {
		t.Fatalf("Expected 3 features, got %d. Features found: %v", len(result.Features), func() []string {
			names := []string{}
			for _, f := range result.Features {
				names = append(names, f.Name)
			}
			return names
		}())
	}

	// Verify all three features are present
	featureMap := make(map[string]*models.Feature)
	for i := range result.Features {
		featureMap[result.Features[i].Name] = &result.Features[i]
	}

	// Check feature_alpha
	if f, ok := featureMap["feature_alpha"]; !ok {
		t.Error("feature_alpha not found")
	} else {
		if f.TotalLicenses != 100 {
			t.Errorf("feature_alpha: expected 100 total licenses, got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 45 {
			t.Errorf("feature_alpha: expected 45 used licenses, got %d", f.UsedLicenses)
		}
	}

	// Check feature_beta
	if f, ok := featureMap["feature_beta"]; !ok {
		t.Error("feature_beta not found")
	} else {
		if f.TotalLicenses != 50 {
			t.Errorf("feature_beta: expected 50 total licenses, got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 12 {
			t.Errorf("feature_beta: expected 12 used licenses, got %d", f.UsedLicenses)
		}
	}

	// Check feature_gamma
	if f, ok := featureMap["feature_gamma"]; !ok {
		t.Error("feature_gamma not found")
	} else {
		if f.TotalLicenses != 25 {
			t.Errorf("feature_gamma: expected 25 total licenses, got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 5 {
			t.Errorf("feature_gamma: expected 5 used licenses, got %d", f.UsedLicenses)
		}
	}

	// Verify expiration dates are set
	for name, feature := range featureMap {
		if feature.ExpirationDate.IsZero() {
			t.Errorf("%s: expiration date not set", name)
		}
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
