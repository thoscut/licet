package services

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
	"licet/internal/config"
)

func TestConfigWriter_AddServer(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config
	initialConfig := map[string]interface{}{
		"servers": []interface{}{},
	}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	// Create config writer
	cw := &ConfigWriter{configPath: configPath}

	// Add a server
	server := config.LicenseServer{
		Hostname:    "27000@test.example.com",
		Description: "Test Server",
		Type:        "flexlm",
	}

	err := cw.AddServer(server)
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	// Verify the server was added
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	servers := result["servers"].([]interface{})
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	srv := servers[0].(map[string]interface{})
	if srv["hostname"] != "27000@test.example.com" {
		t.Errorf("Expected hostname '27000@test.example.com', got '%s'", srv["hostname"])
	}
}

func TestConfigWriter_AddServer_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"hostname": "27000@existing.com",
				"type":     "flexlm",
			},
		},
	}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	server := config.LicenseServer{
		Hostname: "27000@existing.com",
		Type:     "flexlm",
	}

	err := cw.AddServer(server)
	if err == nil {
		t.Error("Expected error for duplicate server, got nil")
	}
}

func TestConfigWriter_DeleteServer(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"hostname":    "27000@server1.com",
				"description": "Server 1",
				"type":        "flexlm",
			},
			map[string]interface{}{
				"hostname":    "5053@server2.com",
				"description": "Server 2",
				"type":        "rlm",
			},
		},
	}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	err := cw.DeleteServer("27000@server1.com")
	if err != nil {
		t.Fatalf("DeleteServer failed: %v", err)
	}

	// Verify only one server remains
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	servers := result["servers"].([]interface{})
	if len(servers) != 1 {
		t.Errorf("Expected 1 server after delete, got %d", len(servers))
	}

	srv := servers[0].(map[string]interface{})
	if srv["hostname"] != "5053@server2.com" {
		t.Errorf("Expected remaining server '5053@server2.com', got '%s'", srv["hostname"])
	}
}

func TestConfigWriter_DeleteServer_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{
		"servers": []interface{}{},
	}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	err := cw.DeleteServer("nonexistent@server.com")
	if err == nil {
		t.Error("Expected error for non-existent server, got nil")
	}
}

func TestConfigWriter_UpdateEmailSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	err := cw.UpdateEmailSettings(true, "from@test.com", []string{"to@test.com"}, "smtp.test.com", 587, "user", "pass")
	if err != nil {
		t.Fatalf("UpdateEmailSettings failed: %v", err)
	}

	// Verify email settings
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	email := result["email"].(map[string]interface{})
	if email["enabled"] != true {
		t.Error("Expected enabled to be true")
	}
	if email["from"] != "from@test.com" {
		t.Errorf("Expected from 'from@test.com', got '%s'", email["from"])
	}
	if email["smtp_port"] != 587 {
		t.Errorf("Expected smtp_port 587, got %v", email["smtp_port"])
	}
}

func TestConfigWriter_UpdateAlertSettings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	err := cw.UpdateAlertSettings(true, 14, 120)
	if err != nil {
		t.Fatalf("UpdateAlertSettings failed: %v", err)
	}

	// Verify alert settings
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	alerts := result["alerts"].(map[string]interface{})
	if alerts["enabled"] != true {
		t.Error("Expected enabled to be true")
	}
	if alerts["lead_time_days"] != 14 {
		t.Errorf("Expected lead_time_days 14, got %v", alerts["lead_time_days"])
	}
	if alerts["resend_interval_min"] != 120 {
		t.Errorf("Expected resend_interval_min 120, got %v", alerts["resend_interval_min"])
	}
}

func TestConfigWriter_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := map[string]interface{}{
		"servers": []interface{}{},
	}
	data, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0600)

	cw := &ConfigWriter{configPath: configPath}

	// Add multiple servers to verify atomic writes don't leave temp files
	for i := 0; i < 5; i++ {
		server := config.LicenseServer{
			Hostname: "27000@server" + string(rune('a'+i)) + ".com",
			Type:     "flexlm",
		}
		if err := cw.AddServer(server); err != nil {
			t.Fatalf("AddServer %d failed: %v", i, err)
		}
	}

	// Check no temp files remain
	files, _ := os.ReadDir(tmpDir)
	for _, f := range files {
		if f.Name() != "config.yaml" {
			t.Errorf("Found unexpected file: %s", f.Name())
		}
	}

	// Verify all 5 servers were added
	data, _ = os.ReadFile(configPath)
	var result map[string]interface{}
	yaml.Unmarshal(data, &result)

	servers := result["servers"].([]interface{})
	if len(servers) != 5 {
		t.Errorf("Expected 5 servers, got %d", len(servers))
	}
}
