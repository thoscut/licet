package parsers

import (
	"strings"
	"testing"
	"time"

	"licet/internal/models"
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

	// Check feature_alpha (2 users in the test data for v2023.1)
	if f, ok := featureMap["feature_alpha"]; !ok {
		t.Error("feature_alpha not found")
	} else {
		if f.TotalLicenses != 100 {
			t.Errorf("feature_alpha: expected 100 total licenses, got %d", f.TotalLicenses)
		}
		// UsedLicenses is now calculated from actual user checkouts, not "Users of" line
		if f.UsedLicenses != 2 {
			t.Errorf("feature_alpha: expected 2 used licenses (actual checkouts), got %d", f.UsedLicenses)
		}
	}

	// Check feature_beta (1 user in the test data for v2023.1)
	if f, ok := featureMap["feature_beta"]; !ok {
		t.Error("feature_beta not found")
	} else {
		if f.TotalLicenses != 50 {
			t.Errorf("feature_beta: expected 50 total licenses, got %d", f.TotalLicenses)
		}
		// UsedLicenses is now calculated from actual user checkouts, not "Users of" line
		if f.UsedLicenses != 1 {
			t.Errorf("feature_beta: expected 1 used license (actual checkouts), got %d", f.UsedLicenses)
		}
	}

	// Check feature_gamma (1 user in the test data for v2023.1)
	if f, ok := featureMap["feature_gamma"]; !ok {
		t.Error("feature_gamma not found")
	} else {
		if f.TotalLicenses != 25 {
			t.Errorf("feature_gamma: expected 25 total licenses, got %d", f.TotalLicenses)
		}
		// UsedLicenses is now calculated from actual user checkouts, not "Users of" line
		if f.UsedLicenses != 1 {
			t.Errorf("feature_gamma: expected 1 used license (actual checkouts), got %d", f.UsedLicenses)
		}
	}

	// Verify expiration dates are set
	for name, feature := range featureMap {
		if feature.ExpirationDate.IsZero() {
			t.Errorf("%s: expiration date not set", name)
		}
	}
}

func TestFlexLMParser_UserVersionCapture(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature1:  (Total of 15 licenses issued;  Total of 5 licenses in use)

  "feature1" v2023.1, vendor: myvendor
  floating license

    user1 machine1 /dev/tty (v2023.1) (server.example.com/27000 1234), start Mon 11/30 9:00
    user2 machine2 /dev/tty (v2023.1) (server.example.com/27000 1235), start Mon 11/30 10:15
    user3 machine3 /dev/tty (v2024.0) (server.example.com/27000 1236), start Tue 12/1 8:00

License files:
feature1 2023.1 10 myvendor 31-dec-2025
feature1 2024.0 5 myvendor 30-jun-2026
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

	// Verify users have correct CLIENT version captured (for display purposes)
	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(result.Users))
	}

	// Users should have their client software versions (not the license version)
	expectedVersions := map[string]string{
		"user1": "2023.1",
		"user2": "2023.1",
		"user3": "2024.0",
	}
	for _, user := range result.Users {
		expected := expectedVersions[user.Username]
		if user.Version != expected {
			t.Errorf("Expected user %s version '%s' (client version), got '%s'", user.Username, expected, user.Version)
		}
	}

	// License files section creates features for each version pool
	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features (2023.1 and 2024.0 pools), got %d", len(result.Features))
	}

	// Verify UsedLicenses are correctly distributed by client version
	featureMap := make(map[string]*models.Feature)
	for i := range result.Features {
		featureMap[result.Features[i].Version] = &result.Features[i]
	}

	// feature1 v2023.1: 2 users (user1, user2 have client version 2023.1)
	if f, ok := featureMap["2023.1"]; !ok {
		t.Error("feature1 v2023.1 not found")
	} else if f.UsedLicenses != 2 {
		t.Errorf("feature1 v2023.1: expected 2 used licenses, got %d", f.UsedLicenses)
	}

	// feature1 v2024.0: 1 user (user3 has client version 2024.0)
	if f, ok := featureMap["2024.0"]; !ok {
		t.Error("feature1 v2024.0 not found")
	} else if f.UsedLicenses != 1 {
		t.Errorf("feature1 v2024.0: expected 1 used license, got %d", f.UsedLicenses)
	}
}

func TestFlexLMParser_UserCheckoutWithoutVPrefix(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Many FlexLM servers output version without "v" prefix: (2023.1) instead of (v2023.1)
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature1:  (Total of 10 licenses issued;  Total of 3 licenses in use)

  "feature1" v2023.1, vendor: myvendor
  floating license

    user1 machine1 /dev/tty (2023.1) (server.example.com/27000 1234), start Mon 1/2 9:00
    user2 machine2 /dev/tty (2023.1) (server.example.com/27000 1235), start Mon 1/2 10:15
    user3 machine3 /dev/tty (2024.0) (server.example.com/27000 1236), start Tue 1/3 8:00

License files:
feature1 2023.1 10 myvendor 31-dec-2025
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

	// Verify users are parsed correctly even without "v" prefix
	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d (checkout lines without 'v' prefix not parsed)", len(result.Users))
	}

	// Users should have their client software versions (for display)
	expectedVersions := map[string]string{
		"user1": "2023.1",
		"user2": "2023.1",
		"user3": "2024.0",
	}
	for _, user := range result.Users {
		expected := expectedVersions[user.Username]
		if user.Version != expected {
			t.Errorf("Expected user %s version '%s' (client version), got '%s'", user.Username, expected, user.Version)
		}
	}
}

func TestFlexLMParser_UserCheckoutWithYearInDate(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Some FlexLM servers include year in checkout date: "Mon 1/2/24 9:00" or "Mon 1/2/2024 9:00"
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature1:  (Total of 10 licenses issued;  Total of 3 licenses in use)

  "feature1" v2023.1, vendor: myvendor
  floating license

    user1 machine1 /dev/tty (v2023.1) (server.example.com/27000 1234), start Mon 1/2/24 9:00
    user2 machine2 /dev/tty (v2023.1) (server.example.com/27000 1235), start Mon 1/2/2024 10:15
    user3 machine3 /dev/tty (v2024.0) (server.example.com/27000 1236), start Tue 1/3 8:00

License files:
feature1 2023.1 10 myvendor 31-dec-2025
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

	// Verify all 3 users are parsed correctly with different date formats
	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d (checkout lines with year in date not parsed)", len(result.Users))
	}

	// Verify user1 (2-digit year format)
	if result.Users[0].Username != "user1" {
		t.Errorf("Expected first user to be 'user1', got '%s'", result.Users[0].Username)
	}
	if result.Users[0].CheckedOutAt.Year() != 2024 {
		t.Errorf("Expected user1 checkout year 2024, got %d", result.Users[0].CheckedOutAt.Year())
	}

	// Verify user2 (4-digit year format)
	if result.Users[1].Username != "user2" {
		t.Errorf("Expected second user to be 'user2', got '%s'", result.Users[1].Username)
	}
	if result.Users[1].CheckedOutAt.Year() != 2024 {
		t.Errorf("Expected user2 checkout year 2024, got %d", result.Users[1].CheckedOutAt.Year())
	}

	// Verify user3 (no year format - should be adjusted to current year)
	if result.Users[2].Username != "user3" {
		t.Errorf("Expected third user to be 'user3', got '%s'", result.Users[2].Username)
	}
}

func TestFlexLMParser_AnsysFormat(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Real-world Ansys FlexLM format with 4 fields before version (including process ID)
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27003@flex-1.example.de
    flex-1.example.de: license server UP v11.18.1

Feature usage info:

Users of cfd_preppost:  (Total of 12 licenses issued;  Total of 1 license in use)

  "cfd_preppost" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    sebastian.mamm caews19.example.de caews19.example.de 3068 (v2025.0506) (flex-1/27003 236), start Thu 1/29 11:39

Users of cfd_preppost_pro:  (Total of 12 licenses issued;  Total of 1 license in use)

  "cfd_preppost_pro" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    sebastian.mann caews19.example.de caews19.example.de 3068 (v2025.0506) (flex-1/27003 4023), start Thu 1/29 11:39
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27003@flex-1.example.de",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Verify users are parsed correctly
	if len(result.Users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(result.Users))
	}

	// Verify first user
	if result.Users[0].Username != "sebastian.mamm" {
		t.Errorf("Expected first user 'sebastian.mamm', got '%s'", result.Users[0].Username)
	}
	if result.Users[0].FeatureName != "cfd_preppost" {
		t.Errorf("Expected first user feature 'cfd_preppost', got '%s'", result.Users[0].FeatureName)
	}
	// User gets the CLIENT version (for display purposes)
	if result.Users[0].Version != "2025.0506" {
		t.Errorf("Expected first user version '2025.0506' (client version), got '%s'", result.Users[0].Version)
	}

	// Verify second user
	if result.Users[1].Username != "sebastian.mann" {
		t.Errorf("Expected second user 'sebastian.mann', got '%s'", result.Users[1].Username)
	}
	if result.Users[1].FeatureName != "cfd_preppost_pro" {
		t.Errorf("Expected second user feature 'cfd_preppost_pro', got '%s'", result.Users[1].FeatureName)
	}
	// Second user also gets the CLIENT version
	if result.Users[1].Version != "2025.0506" {
		t.Errorf("Expected second user version '2025.0506' (client version), got '%s'", result.Users[1].Version)
	}

	// Verify features have correct UsedLicenses
	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features, got %d", len(result.Features))
	}

	featureMap := make(map[string]*models.Feature)
	for i := range result.Features {
		featureMap[result.Features[i].Name] = &result.Features[i]
	}

	if f, ok := featureMap["cfd_preppost"]; !ok {
		t.Error("cfd_preppost feature not found")
	} else {
		if f.UsedLicenses != 1 {
			t.Errorf("cfd_preppost: expected 1 used license, got %d", f.UsedLicenses)
		}
		if f.TotalLicenses != 12 {
			t.Errorf("cfd_preppost: expected 12 total licenses, got %d", f.TotalLicenses)
		}
	}

	if f, ok := featureMap["cfd_preppost_pro"]; !ok {
		t.Error("cfd_preppost_pro feature not found")
	} else {
		if f.UsedLicenses != 1 {
			t.Errorf("cfd_preppost_pro: expected 1 used license, got %d", f.UsedLicenses)
		}
	}
}

func TestFlexLMParser_MultipleLicenseFilesAggregation(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test case where multiple license file entries exist for the same feature/version/expiration
	// These should be aggregated (summed) rather than overwriting each other
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27003@flex-1.example.de
    flex-1.example.de: license server UP v11.18.1

Feature usage info:

Users of ans_act:  (Total of 25 licenses issued;  Total of 2 licenses in use)

  "ans_act" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    user1 machine1.example.de machine1.example.de 1234 (v2025.0506) (flex-1/27003 100), start Thu 1/29 11:39
    user2 machine2.example.de machine2.example.de 5678 (v2025.0506) (flex-1/27003 101), start Thu 1/29 12:00

License files:
ans_act 2026.0630 10 ansyslmd permanent
ans_act 2026.0630 15 ansyslmd permanent
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27003@flex-1.example.de",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Should have 1 feature (aggregated from two license file entries)
	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature (aggregated), got %d", len(result.Features))
	}

	feature := result.Features[0]
	if feature.Name != "ans_act" {
		t.Errorf("Expected feature name 'ans_act', got '%s'", feature.Name)
	}

	// Total licenses should be 10 + 15 = 25 (aggregated from both license file entries)
	if feature.TotalLicenses != 25 {
		t.Errorf("Expected 25 total licenses (10 + 15 aggregated), got %d", feature.TotalLicenses)
	}

	// Used licenses should be 2 (from the two users)
	if feature.UsedLicenses != 2 {
		t.Errorf("Expected 2 used licenses, got %d", feature.UsedLicenses)
	}

	// Verify users were parsed correctly
	if len(result.Users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(result.Users))
	}
}

func TestFlexLMParser_MultipleLicensePoolsDifferentVersions(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test that multiple license pools with different versions are kept SEPARATE
	// (not aggregated like same version/expiration)
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of myfeature:  (Total of 30 licenses issued;  Total of 5 licenses in use)

License files:
myfeature 2023.1 10 myvendor 31-dec-2025
myfeature 2024.0 20 myvendor 31-dec-2025
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

	// Should have 2 separate features (different versions)
	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features (different versions kept separate), got %d", len(result.Features))
	}

	// Verify each version pool has correct count
	versionCounts := make(map[string]int)
	for _, f := range result.Features {
		if f.Name == "myfeature" {
			versionCounts[f.Version] = f.TotalLicenses
		}
	}

	if versionCounts["2023.1"] != 10 {
		t.Errorf("Expected version 2023.1 to have 10 licenses, got %d", versionCounts["2023.1"])
	}
	if versionCounts["2024.0"] != 20 {
		t.Errorf("Expected version 2024.0 to have 20 licenses, got %d", versionCounts["2024.0"])
	}
}

func TestFlexLMParser_MultipleLicensePoolsDifferentExpirations(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test that multiple license pools with same version but different expirations are kept SEPARATE
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of myfeature:  (Total of 25 licenses issued;  Total of 3 licenses in use)

License files:
myfeature 2023.1 10 myvendor 31-dec-2025
myfeature 2023.1 15 myvendor 30-jun-2026
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

	// Should have 2 separate features (different expirations)
	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features (different expirations kept separate), got %d", len(result.Features))
	}

	// Verify each expiration pool has correct count
	expectedDate1, _ := time.Parse("2-Jan-2006", "31-Dec-2025")
	expectedDate2, _ := time.Parse("2-Jan-2006", "30-Jun-2026")

	expirationCounts := make(map[string]int)
	for _, f := range result.Features {
		if f.Name == "myfeature" {
			expirationCounts[f.ExpirationDate.Format("2006-01-02")] = f.TotalLicenses
		}
	}

	if expirationCounts[expectedDate1.Format("2006-01-02")] != 10 {
		t.Errorf("Expected Dec 2025 pool to have 10 licenses, got %d", expirationCounts[expectedDate1.Format("2006-01-02")])
	}
	if expirationCounts[expectedDate2.Format("2006-01-02")] != 15 {
		t.Errorf("Expected Jun 2026 pool to have 15 licenses, got %d", expirationCounts[expectedDate2.Format("2006-01-02")])
	}
}

func TestFlexLMParser_CheckoutMatchingByFeatureName(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test that checkouts are matched to features by name when client version differs from license version
	// This is critical for correct UsedLicenses counting
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of ans_act:  (Total of 25 licenses issued;  Total of 3 licenses in use)

  "ans_act" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    user1 host1.example.de host1.example.de 1234 (v2025.0506) (server/27000 100), start Mon 1/15 9:00
    user2 host2.example.de host2.example.de 2345 (v2024.0101) (server/27000 101), start Mon 1/15 10:00
    user3 host3.example.de host3.example.de 3456 (v2025.0506) (server/27000 102), start Mon 1/15 11:00

License files:
ans_act 2026.0630 25 ansyslmd permanent
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

	// Should have 1 feature
	if len(result.Features) != 1 {
		t.Fatalf("Expected 1 feature, got %d", len(result.Features))
	}

	feature := result.Features[0]

	// All 3 users should be counted even though their client versions (2025.0506, 2024.0101)
	// differ from the license version (2026.0630) - counting falls back to feature name
	if feature.UsedLicenses != 3 {
		t.Errorf("Expected 3 used licenses (all users matched by feature name), got %d", feature.UsedLicenses)
	}

	// Verify users have their CLIENT versions (for display)
	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(result.Users))
	}
	expectedVersions := map[string]string{
		"user1": "2025.0506",
		"user2": "2024.0101",
		"user3": "2025.0506",
	}
	for _, user := range result.Users {
		expected := expectedVersions[user.Username]
		if user.Version != expected {
			t.Errorf("Expected user %s to have client version '%s', got '%s'", user.Username, expected, user.Version)
		}
	}
}

func TestFlexLMParser_UsedLicensesProportionalFallback(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test that when there are no checkout lines, UsedLicenses is calculated proportionally
	// from the "Users of" line based on pool size
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of myfeature:  (Total of 100 licenses issued;  Total of 50 licenses in use)

License files:
myfeature 2023.1 60 myvendor 31-dec-2025
myfeature 2024.0 40 myvendor 31-dec-2026
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

	// Should have 2 features (different versions/expirations)
	if len(result.Features) != 2 {
		t.Fatalf("Expected 2 features, got %d", len(result.Features))
	}

	// Without checkout lines, used licenses should be proportionally distributed
	// Pool with 60 licenses: 50 * 60 / 100 = 30 used
	// Pool with 40 licenses: 50 * 40 / 100 = 20 used
	for _, f := range result.Features {
		if f.Version == "2023.1" {
			expectedUsed := (50 * 60) / 100 // = 30
			if f.UsedLicenses != expectedUsed {
				t.Errorf("Version 2023.1: expected %d used licenses (proportional), got %d", expectedUsed, f.UsedLicenses)
			}
		}
		if f.Version == "2024.0" {
			expectedUsed := (50 * 40) / 100 // = 20
			if f.UsedLicenses != expectedUsed {
				t.Errorf("Version 2024.0: expected %d used licenses (proportional), got %d", expectedUsed, f.UsedLicenses)
			}
		}
	}
}

func TestFlexLMParser_InlineVersionParsingVariants(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Test various inline version formats that appear after "Users of" line
	output := `lmstat - Copyright (c) 1989-2023 Flexera.
License server status: 27000@server.example.com
    server.example.com: license server UP v11.18.1

Feature usage info:

Users of feature_a:  (Total of 10 licenses issued;  Total of 1 license in use)

  "feature_a" v2023.1, vendor: myvendor, expiry: 31-dec-2025
  floating license

    user1 host1 /dev/tty (v2022.0) (server/27000 100), start Mon 1/15 9:00

Users of feature_b:  (Total of 10 licenses issued;  Total of 1 license in use)

  "feature_b" v2024.0506, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:12345
  floating license

    user2 host2 /dev/tty (v2023.0101) (server/27000 101), start Mon 1/15 10:00

Users of feature_c:  (Total of 10 licenses issued;  Total of 1 license in use)

  "feature_c" v1.0, vendor: simplevendor
  floating license

    user3 host3 /dev/tty (v0.9) (server/27000 102), start Mon 1/15 11:00
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

	// Should have 3 features
	if len(result.Features) != 3 {
		t.Fatalf("Expected 3 features, got %d", len(result.Features))
	}

	// All users should have their respective CLIENT versions (for display)
	expectedVersions := map[string]string{
		"user1": "2022.0",
		"user2": "2023.0101",
		"user3": "0.9",
	}

	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(result.Users))
	}

	for _, user := range result.Users {
		expected := expectedVersions[user.Username]
		if user.Version != expected {
			t.Errorf("User %s: expected client version '%s', got '%s'", user.Username, expected, user.Version)
		}
	}

	// Verify features have correct versions from inline parsing
	featureVersions := make(map[string]string)
	for _, f := range result.Features {
		featureVersions[f.Name] = f.Version
	}

	if featureVersions["feature_a"] != "2023.1" {
		t.Errorf("feature_a: expected version '2023.1', got '%s'", featureVersions["feature_a"])
	}
	if featureVersions["feature_b"] != "2024.0506" {
		t.Errorf("feature_b: expected version '2024.0506', got '%s'", featureVersions["feature_b"])
	}
	if featureVersions["feature_c"] != "1.0" {
		t.Errorf("feature_c: expected version '1.0', got '%s'", featureVersions["feature_c"])
	}
}

func TestFlexLMParser_ComprehensiveRealWorldAnsys(t *testing.T) {
	parser := &FlexLMParser{lmutilPath: "/usr/local/bin/lmutil"}

	// Comprehensive test with real-world Ansys FlexLM output format
	// Tests: multiple features, inline versions, client version differs from license version,
	// multiple license file entries with same key (aggregation)
	output := `lmutil - Copyright (c) 1989-2023 Flexera Software LLC. All Rights Reserved.
lmstat - Copyright (c) 1989-2023 Flexera Software LLC. All Rights Reserved.
Flexible License Manager status on Thu 1/30/2025 10:00

[Detecting lmgrd processes...]
License server status: 27003@flex-1.example.de
    License file(s) on flex-1.example.de: /opt/ansys/license/license.dat

flex-1.example.de: license server UP (MASTER) v11.18.1

Vendor daemon status (on flex-1.example.de):

     ansyslmd: UP v11.18.1

Feature usage info:

Users of ans_act:  (Total of 25 licenses issued;  Total of 3 licenses in use)

  "ans_act" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    alice host1.example.de host1.example.de 1001 (v2025.0506) (flex-1/27003 100), start Thu 1/30 8:00
    bob host2.example.de host2.example.de 2002 (v2024.0101) (flex-1/27003 101), start Thu 1/30 9:00
    charlie host3.example.de host3.example.de 3003 (v2025.0506) (flex-1/27003 102), start Thu 1/30 10:00

Users of cfd_solve:  (Total of 10 licenses issued;  Total of 2 licenses in use)

  "cfd_solve" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

    dave host4.example.de host4.example.de 4004 (v2025.0506) (flex-1/27003 200), start Thu 1/30 7:30
    eve host5.example.de host5.example.de 5005 (v2025.0506) (flex-1/27003 201), start Thu 1/30 8:30

Users of mech_struct:  (Total of 15 licenses issued;  Total of 0 licenses in use)

  "mech_struct" v2026.0630, vendor: ansyslmd, expiry: permanent(no expiration date)
  vendor_string: customer:00411180
  floating license

License files on flex-1.example.de (27003):
ans_act 2026.0630 15 ansyslmd permanent
ans_act 2026.0630 10 ansyslmd permanent
cfd_solve 2026.0630 10 ansyslmd permanent
mech_struct 2026.0630 15 ansyslmd permanent
`

	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "27003@flex-1.example.de",
			Service:  "down",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Verify server status
	if result.Status.Service != "up" {
		t.Errorf("Expected service 'up', got '%s'", result.Status.Service)
	}
	if result.Status.Version != "11.18.1" {
		t.Errorf("Expected version '11.18.1', got '%s'", result.Status.Version)
	}

	// Should have 3 features
	if len(result.Features) != 3 {
		t.Fatalf("Expected 3 features, got %d", len(result.Features))
	}

	featureMap := make(map[string]*models.Feature)
	for i := range result.Features {
		featureMap[result.Features[i].Name] = &result.Features[i]
	}

	// Test ans_act: 15 + 10 = 25 aggregated, 3 users
	if f, ok := featureMap["ans_act"]; !ok {
		t.Error("ans_act not found")
	} else {
		if f.TotalLicenses != 25 {
			t.Errorf("ans_act: expected 25 total (15+10 aggregated), got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 3 {
			t.Errorf("ans_act: expected 3 used, got %d", f.UsedLicenses)
		}
		if f.Version != "2026.0630" {
			t.Errorf("ans_act: expected version '2026.0630', got '%s'", f.Version)
		}
	}

	// Test cfd_solve: 10 licenses, 2 users
	if f, ok := featureMap["cfd_solve"]; !ok {
		t.Error("cfd_solve not found")
	} else {
		if f.TotalLicenses != 10 {
			t.Errorf("cfd_solve: expected 10 total, got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 2 {
			t.Errorf("cfd_solve: expected 2 used, got %d", f.UsedLicenses)
		}
	}

	// Test mech_struct: 15 licenses, 0 users
	if f, ok := featureMap["mech_struct"]; !ok {
		t.Error("mech_struct not found")
	} else {
		if f.TotalLicenses != 15 {
			t.Errorf("mech_struct: expected 15 total, got %d", f.TotalLicenses)
		}
		if f.UsedLicenses != 0 {
			t.Errorf("mech_struct: expected 0 used, got %d", f.UsedLicenses)
		}
	}

	// Verify all users parsed correctly with their CLIENT versions (for display)
	if len(result.Users) != 5 {
		t.Fatalf("Expected 5 users, got %d", len(result.Users))
	}

	// Users should have their client software versions
	expectedUserVersions := map[string]string{
		"alice":   "2025.0506",
		"bob":     "2024.0101",
		"charlie": "2025.0506",
		"dave":    "2025.0506",
		"eve":     "2025.0506",
	}
	for _, user := range result.Users {
		expected := expectedUserVersions[user.Username]
		if user.Version != expected {
			t.Errorf("User %s: expected client version '%s', got '%s'", user.Username, expected, user.Version)
		}
	}

	// Verify specific users
	userFeatures := make(map[string]string)
	for _, u := range result.Users {
		userFeatures[u.Username] = u.FeatureName
	}

	expectedUserFeatures := map[string]string{
		"alice":   "ans_act",
		"bob":     "ans_act",
		"charlie": "ans_act",
		"dave":    "cfd_solve",
		"eve":     "cfd_solve",
	}

	for user, expectedFeature := range expectedUserFeatures {
		if userFeatures[user] != expectedFeature {
			t.Errorf("User %s: expected feature '%s', got '%s'", user, expectedFeature, userFeatures[user])
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
