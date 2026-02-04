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

	// Verify users have correct version captured
	// Note: Users get the LICENSE version (from inline feature info), not the client version
	if len(result.Users) != 3 {
		t.Fatalf("Expected 3 users, got %d", len(result.Users))
	}

	// All users should have the license version (2023.1) from the inline feature info
	for _, user := range result.Users {
		if user.Version != "2023.1" {
			t.Errorf("Expected user %s version '2023.1' (license version), got '%s'", user.Username, user.Version)
		}
	}

	// With inline version parsing, features from usageMap get the inline version
	// The License files section creates additional features
	if len(result.Features) < 1 {
		t.Fatalf("Expected at least 1 feature, got %d", len(result.Features))
	}

	// Find the feature from usageMap (it should have the inline version 2023.1)
	var usageFeature *models.Feature
	for i := range result.Features {
		if result.Features[i].Name == "feature1" && result.Features[i].Version == "2023.1" {
			usageFeature = &result.Features[i]
			break
		}
	}

	if usageFeature == nil {
		t.Error("feature1 v2023.1 not found")
	} else {
		// All 3 users should be counted under this feature
		if usageFeature.UsedLicenses != 3 {
			t.Errorf("feature1 v2023.1: expected 3 used licenses, got %d", usageFeature.UsedLicenses)
		}
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

	// All users should have the license version (2023.1) from the inline feature info
	// not their client versions (which differ)
	for _, user := range result.Users {
		if user.Version != "2023.1" {
			t.Errorf("Expected user %s version '2023.1' (license version), got '%s'", user.Username, user.Version)
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
	// User gets the LICENSE version (from inline feature info), not the client version
	if result.Users[0].Version != "2026.0630" {
		t.Errorf("Expected first user version '2026.0630' (license version), got '%s'", result.Users[0].Version)
	}

	// Verify second user
	if result.Users[1].Username != "sebastian.mann" {
		t.Errorf("Expected second user 'sebastian.mann', got '%s'", result.Users[1].Username)
	}
	if result.Users[1].FeatureName != "cfd_preppost_pro" {
		t.Errorf("Expected second user feature 'cfd_preppost_pro', got '%s'", result.Users[1].FeatureName)
	}
	// Second user also gets the LICENSE version of their feature
	if result.Users[1].Version != "2026.0630" {
		t.Errorf("Expected second user version '2026.0630' (license version), got '%s'", result.Users[1].Version)
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
