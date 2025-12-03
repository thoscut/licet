package parsers

import (
	"strings"
	"testing"
	"time"

	"licet/internal/models"
)

func TestRLMParser_ParseOutput_WithCheckouts(t *testing.T) {
	output := `rlm status on server.example.com (port 5053, pid 12345)

rlm software version v15.2 (build 1)

Startup time: 12/01/2025 08:00
Up for: 5 hours

foundry 5053 Yes 12345

arnold v20160712 license pool status
count: 10, inuse: 2, exp: 31-Dec-2025

arnold v20160712: user1@workstation1 1/0 at 12/01 09:15 (handle: 971)
arnold v20160712: user2@workstation2 1/0 at 12/01 10:30 (handle: 972)

maya v2023 license pool status
UNCOUNTED, inuse: 1, exp: permanent

maya v2023: joe@library 1/0 at 12/01 08:45 (handle: 41)
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Check that server status is up
	if result.Status.Service != "up" {
		t.Errorf("Expected status 'up', got '%s'", result.Status.Service)
	}

	// Check version
	if result.Status.Version != "15.2" {
		t.Errorf("Expected version '15.2', got '%s'", result.Status.Version)
	}

	// Check master server
	if result.Status.Master != "server.example.com" {
		t.Errorf("Expected master 'server.example.com', got '%s'", result.Status.Master)
	}

	// Check features
	if len(result.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(result.Features))
	}

	// Check arnold feature
	arnoldFound := false
	for _, f := range result.Features {
		if f.Name == "arnold" {
			arnoldFound = true
			if f.TotalLicenses != 10 {
				t.Errorf("Expected arnold total licenses 10, got %d", f.TotalLicenses)
			}
			if f.UsedLicenses != 2 {
				t.Errorf("Expected arnold used licenses 2, got %d", f.UsedLicenses)
			}
		}
	}
	if !arnoldFound {
		t.Error("arnold feature not found")
	}

	// Check users - this is the main test for the fix
	if len(result.Users) != 3 {
		t.Errorf("Expected 3 users, got %d", len(result.Users))
		for i, u := range result.Users {
			t.Logf("User %d: %s@%s using %s", i, u.Username, u.Host, u.FeatureName)
		}
	}

	// Check first arnold user
	user1Found := false
	for _, u := range result.Users {
		if u.Username == "user1" && u.Host == "workstation1" && u.FeatureName == "arnold" {
			user1Found = true
			// Check checkout time (should be 12/01 09:15)
			if u.CheckedOutAt.Month() != 12 || u.CheckedOutAt.Day() != 1 {
				t.Errorf("Expected checkout date 12/01, got %02d/%02d", u.CheckedOutAt.Month(), u.CheckedOutAt.Day())
			}
			if u.CheckedOutAt.Hour() != 9 || u.CheckedOutAt.Minute() != 15 {
				t.Errorf("Expected checkout time 09:15, got %02d:%02d", u.CheckedOutAt.Hour(), u.CheckedOutAt.Minute())
			}
		}
	}
	if !user1Found {
		t.Error("user1@workstation1 checkout not found")
	}

	// Check second arnold user
	user2Found := false
	for _, u := range result.Users {
		if u.Username == "user2" && u.Host == "workstation2" && u.FeatureName == "arnold" {
			user2Found = true
		}
	}
	if !user2Found {
		t.Error("user2@workstation2 checkout not found")
	}

	// Check maya user
	joeFound := false
	for _, u := range result.Users {
		if u.Username == "joe" && u.Host == "library" && u.FeatureName == "maya" {
			joeFound = true
		}
	}
	if !joeFound {
		t.Error("joe@library checkout not found")
	}
}

func TestRLMParser_ParseOutput_NoCheckouts(t *testing.T) {
	output := `rlm status on server.example.com (port 5053, pid 12345)

rlm software version v15.2 (build 1)

foundry 5053 Yes 12345

arnold v20160712 license pool status
count: 10, inuse: 0, exp: 31-Dec-2025
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Check that server status is up
	if result.Status.Service != "up" {
		t.Errorf("Expected status 'up', got '%s'", result.Status.Service)
	}

	// Check features
	if len(result.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(result.Features))
	}

	// Check no users
	if len(result.Users) != 0 {
		t.Errorf("Expected 0 users, got %d", len(result.Users))
	}
}

func TestRLMParser_CheckoutTimeFormat(t *testing.T) {
	// Test with the exact format from RLM documentation
	output := `rlm status on server.example.com (port 5053)

rlm software version v15.2

foundry 5053 Yes 12345

test v1.0 license pool status
count: 5, inuse: 1, exp: permanent

test v1.0: joe@library 1/0 at 07/06 13:27 (handle: 41)
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Check users
	if len(result.Users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(result.Users))
	}

	user := result.Users[0]
	if user.Username != "joe" {
		t.Errorf("Expected username 'joe', got '%s'", user.Username)
	}
	if user.Host != "library" {
		t.Errorf("Expected host 'library', got '%s'", user.Host)
	}
	if user.FeatureName != "test" {
		t.Errorf("Expected feature 'test', got '%s'", user.FeatureName)
	}

	// Check time parsing - should be July 6, 13:27 of current year
	now := time.Now()
	expectedMonth := time.July
	expectedDay := 6
	expectedHour := 13
	expectedMinute := 27

	if user.CheckedOutAt.Year() != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), user.CheckedOutAt.Year())
	}
	if user.CheckedOutAt.Month() != expectedMonth {
		t.Errorf("Expected month %s, got %s", expectedMonth, user.CheckedOutAt.Month())
	}
	if user.CheckedOutAt.Day() != expectedDay {
		t.Errorf("Expected day %d, got %d", expectedDay, user.CheckedOutAt.Day())
	}
	if user.CheckedOutAt.Hour() != expectedHour {
		t.Errorf("Expected hour %d, got %d", expectedHour, user.CheckedOutAt.Hour())
	}
	if user.CheckedOutAt.Minute() != expectedMinute {
		t.Errorf("Expected minute %d, got %d", expectedMinute, user.CheckedOutAt.Minute())
	}
}

func TestRLMParser_IgnoreUtilityNames(t *testing.T) {
	// Test that utility names like "rlmutil" are not treated as features
	output := `rlm status on server.example.com (port 5053)

rlm software version v15.2

foundry 5053 Yes 12345

rlmutil v15.2 license pool status
count: 5, inuse: 2, exp: permanent

arnold v20160712 license pool status
count: 10, inuse: 3, exp: 31-Dec-2025
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Check that we only have 1 feature (arnold), not 2 (should exclude rlmutil)
	if len(result.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(result.Features))
		for _, f := range result.Features {
			t.Logf("Found feature: %s", f.Name)
		}
	}

	// Verify that the feature is arnold, not rlmutil
	if len(result.Features) > 0 {
		if result.Features[0].Name == "rlmutil" {
			t.Error("rlmutil should not be treated as a feature")
		}
		if result.Features[0].Name != "arnold" {
			t.Errorf("Expected feature 'arnold', got '%s'", result.Features[0].Name)
		}
	}
}

func TestRLMParser_FeatureHeaderWithAdditionalContent(t *testing.T) {
	// Test that feature headers with additional content after version are matched
	// This tests that lines with "pool" in them are properly parsed as feature headers
	output := `rlm status on server.example.com (port 5053)

rlm software version v15.2

foundry 5053 Yes 12345

arnold v20160712 license pool status (issued: 31-Dec-2025)
count: 10, inuse: 3, exp: 31-Dec-2025

maya v2024 floating license pool
count: 5, inuse: 2, exp: permanent

nuke v13.2 license pool status
count: 20, inuse: 8, exp: 15-Jun-2026
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Should find all 3 features despite additional content after version
	if len(result.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(result.Features))
		for _, f := range result.Features {
			t.Logf("Found feature: %s (v%s)", f.Name, f.Version)
		}
	}

	// Check all features are captured correctly
	foundArnold := false
	foundMaya := false
	foundNuke := false

	for _, f := range result.Features {
		switch f.Name {
		case "arnold":
			foundArnold = true
			if f.Version != "v20160712" {
				t.Errorf("Expected arnold version 'v20160712', got '%s'", f.Version)
			}
		case "maya":
			foundMaya = true
			if f.Version != "v2024" {
				t.Errorf("Expected maya version 'v2024', got '%s'", f.Version)
			}
		case "nuke":
			foundNuke = true
			if f.Version != "v13.2" {
				t.Errorf("Expected nuke version 'v13.2', got '%s'", f.Version)
			}
		}
	}

	if !foundArnold {
		t.Error("arnold feature not found")
	}
	if !foundMaya {
		t.Error("maya feature not found")
	}
	if !foundNuke {
		t.Error("nuke feature not found")
	}
}

func TestRLMParser_DontMatchCheckoutLinesAsFeatures(t *testing.T) {
	// Test that checkout lines (with colon after version) are NOT matched as features
	output := `rlm status on server.example.com (port 5053)

rlm software version v15.2

foundry 5053 Yes 12345

arnold v20160712 license pool status
count: 10, inuse: 2, exp: 31-Dec-2025

arnold v20160712: user1@workstation1 1/0 at 12/01 09:15 (handle: 971)
arnold v20160712: user2@workstation2 1/0 at 12/01 10:30 (handle: 972)
`

	parser := NewRLMParser("")
	result := models.ServerQueryResult{
		Status: models.ServerStatus{
			Hostname: "server.example.com",
		},
		Features: []models.Feature{},
		Users:    []models.LicenseUser{},
	}

	parser.parseOutput(strings.NewReader(output), &result)

	// Should only find 1 feature (arnold), not count checkout lines as features
	if len(result.Features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(result.Features))
		for _, f := range result.Features {
			t.Logf("Found feature: %s", f.Name)
		}
	}

	// Should find 2 users
	if len(result.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(result.Users))
	}
}
