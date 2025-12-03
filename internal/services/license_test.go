package services

import (
	"testing"

	"licet/internal/config"
	"licet/internal/models"
)

func TestGetAllServers(t *testing.T) {
	cfg := &config.Config{
		Servers: []config.LicenseServer{
			{
				Hostname:    "27000@test1.com",
				Description: "Test Server 1",
				Type:        "flexlm",
			},
			{
				Hostname:    "5053@test2.com",
				Description: "Test Server 2",
				Type:        "rlm",
			},
		},
	}

	// Create QueryService with nil storage (we're not testing storage operations)
	query := NewQueryService(cfg, nil)

	servers, err := query.GetAllServers()
	if err != nil {
		t.Fatalf("GetAllServers failed: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	if servers[0].Hostname != "27000@test1.com" {
		t.Errorf("Expected hostname '27000@test1.com', got '%s'", servers[0].Hostname)
	}

	if servers[0].Type != "flexlm" {
		t.Errorf("Expected type 'flexlm', got '%s'", servers[0].Type)
	}

	if servers[1].Type != "rlm" {
		t.Errorf("Expected type 'rlm', got '%s'", servers[1].Type)
	}
}

func TestGetAllServers_Empty(t *testing.T) {
	cfg := &config.Config{
		Servers: []config.LicenseServer{},
	}

	// Create QueryService with nil storage
	query := NewQueryService(cfg, nil)

	servers, err := query.GetAllServers()
	if err != nil {
		t.Fatalf("GetAllServers failed: %v", err)
	}

	if len(servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(servers))
	}
}

func TestFeatureAvailableLicenses(t *testing.T) {
	tests := []struct {
		name     string
		feature  models.Feature
		expected int
	}{
		{
			name: "Normal case",
			feature: models.Feature{
				TotalLicenses: 10,
				UsedLicenses:  5,
			},
			expected: 5,
		},
		{
			name: "All used",
			feature: models.Feature{
				TotalLicenses: 10,
				UsedLicenses:  10,
			},
			expected: 0,
		},
		{
			name: "Over-subscribed (should return 0)",
			feature: models.Feature{
				TotalLicenses: 10,
				UsedLicenses:  15,
			},
			expected: 0,
		},
		{
			name: "None used",
			feature: models.Feature{
				TotalLicenses: 10,
				UsedLicenses:  0,
			},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.feature.AvailableLicenses()
			if result != tt.expected {
				t.Errorf("Expected %d available licenses, got %d", tt.expected, result)
			}
		})
	}
}

func TestLinearRegression(t *testing.T) {
	// Test with known linear data: y = 2x + 3
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{5, 7, 9, 11, 13}

	slope, intercept := linearRegression(x, y)

	// Allow small floating point errors
	if slope < 1.99 || slope > 2.01 {
		t.Errorf("Expected slope ~2.0, got %f", slope)
	}

	if intercept < 2.99 || intercept > 3.01 {
		t.Errorf("Expected intercept ~3.0, got %f", intercept)
	}
}

func TestLinearRegression_EmptyData(t *testing.T) {
	x := []float64{}
	y := []float64{}

	slope, intercept := linearRegression(x, y)

	if slope != 0 || intercept != 0 {
		t.Errorf("Expected (0,0) for empty data, got (%f, %f)", slope, intercept)
	}
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name        string
		values      []float64
		expectedAvg float64
		expectedStd float64
		tolerance   float64
	}{
		{
			name:        "Simple case",
			values:      []float64{1, 2, 3, 4, 5},
			expectedAvg: 3.0,
			expectedStd: 1.414, // sqrt(2)
			tolerance:   0.01,
		},
		{
			name:        "All same values",
			values:      []float64{5, 5, 5, 5},
			expectedAvg: 5.0,
			expectedStd: 0.0,
			tolerance:   0.001,
		},
		{
			name:        "Two values",
			values:      []float64{0, 10},
			expectedAvg: 5.0,
			expectedStd: 5.0,
			tolerance:   0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mean, stdDev := calculateStats(tt.values)

			if mean < tt.expectedAvg-tt.tolerance || mean > tt.expectedAvg+tt.tolerance {
				t.Errorf("Expected mean ~%f, got %f", tt.expectedAvg, mean)
			}

			if stdDev < tt.expectedStd-tt.tolerance || stdDev > tt.expectedStd+tt.tolerance {
				t.Errorf("Expected stdDev ~%f, got %f", tt.expectedStd, stdDev)
			}
		})
	}
}

func TestCalculateStats_EmptyData(t *testing.T) {
	values := []float64{}

	mean, stdDev := calculateStats(values)

	if mean != 0 || stdDev != 0 {
		t.Errorf("Expected (0,0) for empty data, got (%f, %f)", mean, stdDev)
	}
}

func TestCalculateRSquared(t *testing.T) {
	// Perfect linear fit: y = 2x + 1
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{3, 5, 7, 9, 11}
	slope := 2.0
	intercept := 1.0

	rSquared := calculateRSquared(x, y, slope, intercept)

	// Should be very close to 1.0 for perfect fit
	if rSquared < 0.99 {
		t.Errorf("Expected R-squared ~1.0 for perfect fit, got %f", rSquared)
	}
}

func TestCalculateRSquared_PoorFit(t *testing.T) {
	// Data doesn't fit line well
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{1, 10, 2, 15, 3}
	slope := 0.0
	intercept := 6.2

	rSquared := calculateRSquared(x, y, slope, intercept)

	// Should be low for poor fit
	if rSquared > 0.5 {
		t.Errorf("Expected low R-squared for poor fit, got %f", rSquared)
	}
}

func TestCalculateRSquared_EmptyData(t *testing.T) {
	x := []float64{}
	y := []float64{}

	rSquared := calculateRSquared(x, y, 0, 0)

	if rSquared != 0 {
		t.Errorf("Expected 0 for empty data, got %f", rSquared)
	}
}

func TestCalculateRSquared_Bounds(t *testing.T) {
	// Test that R-squared is always between 0 and 1
	// Even with bad predictions, it should be clamped
	x := []float64{1, 2, 3}
	y := []float64{1, 1, 1}
	slope := 0.0
	intercept := 1.0

	rSquared := calculateRSquared(x, y, slope, intercept)

	if rSquared < 0 || rSquared > 1 {
		t.Errorf("R-squared should be between 0 and 1, got %f", rSquared)
	}
}

// Test that standard deviation is actually using sqrt(variance), not just variance
func TestCalculateStats_VerifyStdDevFormula(t *testing.T) {
	// Use values where we can easily calculate expected result
	values := []float64{1, 2, 3, 4, 5}

	// Expected: mean = 3, variance = 2, stddev = sqrt(2) ≈ 1.414
	mean, stdDev := calculateStats(values)

	if mean != 3.0 {
		t.Errorf("Expected mean 3.0, got %f", mean)
	}

	// Verify stdDev is sqrt(variance), not variance itself
	// variance = ((1-3)^2 + (2-3)^2 + (3-3)^2 + (4-3)^2 + (5-3)^2) / 5
	// variance = (4 + 1 + 0 + 1 + 4) / 5 = 2.0
	// stddev = sqrt(2.0) ≈ 1.414

	expectedStdDev := 1.414
	tolerance := 0.01

	if stdDev < expectedStdDev-tolerance || stdDev > expectedStdDev+tolerance {
		t.Errorf("Expected stdDev ~%f (sqrt of variance), got %f (might be raw variance)",
			expectedStdDev, stdDev)
	}

	// Additional check: stdDev should NOT equal variance (2.0) for this data
	if stdDev > 1.9 && stdDev < 2.1 {
		t.Errorf("stdDev appears to be variance (2.0) instead of sqrt(variance) (1.414), got %f", stdDev)
	}
}
