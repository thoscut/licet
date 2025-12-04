package util

import (
	"testing"
)

func TestValidateHostname_PortAtHost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"Valid FlexLM format", "27000@license.example.com", true},
		{"Valid with IP", "27000@192.168.1.100", true},
		{"Invalid port (0)", "0@server.com", false},
		{"Invalid port (too high)", "70000@server.com", false},
		{"Invalid port (not a number)", "abc@server.com", false},
		{"Empty host", "27000@", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateHostname(tt.hostname)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidateHostname_HostColonPort(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"Valid host:port format", "license.example.com:5053", true},
		{"Valid with IP", "192.168.1.100:5053", true},
		{"Invalid port (0)", "server.com:0", false}, // Port 0 is not valid for license servers
		{"Invalid port (too high)", "server.com:70000", false},
		{"Invalid port (not a number)", "server.com:abc", false},
		{"Empty host", ":5053", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateHostname(tt.hostname)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidateHostname_JustHost(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"Valid hostname", "license.example.com", true},
		{"Valid short hostname", "localhost", true},
		{"Valid IP", "192.168.1.100", true},
		{"Invalid IP (octet > 255)", "192.168.1.300", false},
		{"Invalid hostname (starts with hyphen)", "-invalid.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateHostname(tt.hostname)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidateServerType(t *testing.T) {
	tests := []struct {
		name       string
		serverType string
		valid      bool
	}{
		{"FlexLM lowercase", "flexlm", true},
		{"FlexLM uppercase", "FLEXLM", true},
		{"FlexLM mixed", "FlexLM", true},
		{"RLM lowercase", "rlm", true},
		{"RLM uppercase", "RLM", true},
		{"Invalid type", "invalid", false},
		{"Empty type", "", false},
		{"SPM (not implemented)", "spm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerType(tt.serverType)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"Valid email", "user@example.com", true},
		{"Valid with plus", "user+tag@example.com", true},
		{"Valid with dots", "first.last@example.com", true},
		{"Empty (allowed)", "", true},
		{"Missing @", "userexample.com", false},
		{"Missing domain", "user@", false},
		{"Missing TLD", "user@example", false},
		{"Just @", "@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"Valid port 80", 80, true},
		{"Valid port 443", 443, true},
		{"Valid port 0", 0, true},
		{"Valid port 65535", 65535, true},
		{"Invalid negative", -1, false},
		{"Invalid too high", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		valid bool
	}{
		{"Positive", 10, true},
		{"Zero", 0, true},
		{"Negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveInt(tt.value, "test_field")
			if tt.valid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestSupportedServerTypes(t *testing.T) {
	types := SupportedServerTypes()

	// Check that at least flexlm and rlm are supported
	if !types["flexlm"] {
		t.Error("flexlm should be supported")
	}
	if !types["rlm"] {
		t.Error("rlm should be supported")
	}

	// Check that unsupported types return false
	if types["invalid"] {
		t.Error("invalid should not be supported")
	}
}
