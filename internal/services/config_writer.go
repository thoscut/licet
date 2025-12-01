package services

import (
	"fmt"
	"os"

	"github.com/thoscut/licet/internal/config"
	"gopkg.in/yaml.v3"
)

// ConfigWriter handles writing server configurations to config.yaml
type ConfigWriter struct {
	configPath string
}

// NewConfigWriter creates a new config writer
func NewConfigWriter() *ConfigWriter {
	// Try to find config file in standard locations
	configPaths := []string{
		"./config.yaml",
		"./config/config.yaml",
		"/etc/licet/config.yaml",
	}

	configPath := "./config.yaml" // default
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	return &ConfigWriter{
		configPath: configPath,
	}
}

// AddServer adds a new server to the config file
func (cw *ConfigWriter) AddServer(server config.LicenseServer) error {
	// Read the current config file
	data, err := os.ReadFile(cw.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get or create servers array
	serversInterface, ok := configData["servers"]
	if !ok {
		configData["servers"] = []interface{}{}
		serversInterface = configData["servers"]
	}

	servers, ok := serversInterface.([]interface{})
	if !ok {
		return fmt.Errorf("servers is not an array in config file")
	}

	// Check if server already exists
	for _, s := range servers {
		if srv, ok := s.(map[string]interface{}); ok {
			if hostname, exists := srv["hostname"]; exists && hostname == server.Hostname {
				return fmt.Errorf("server %s already exists", server.Hostname)
			}
		}
	}

	// Add new server
	newServer := map[string]interface{}{
		"hostname":    server.Hostname,
		"description": server.Description,
		"type":        server.Type,
	}
	if server.CactiID != "" {
		newServer["cacti_id"] = server.CactiID
	}
	if server.WebUI != "" {
		newServer["webui"] = server.WebUI
	}

	servers = append(servers, newServer)
	configData["servers"] = servers

	// Write back to file
	output, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cw.configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DeleteServer removes a server from the config file
func (cw *ConfigWriter) DeleteServer(hostname string) error {
	// Read the current config file
	data, err := os.ReadFile(cw.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get servers array
	serversInterface, ok := configData["servers"]
	if !ok {
		return fmt.Errorf("no servers found in config file")
	}

	servers, ok := serversInterface.([]interface{})
	if !ok {
		return fmt.Errorf("servers is not an array in config file")
	}

	// Find and remove server
	found := false
	newServers := []interface{}{}
	for _, s := range servers {
		if srv, ok := s.(map[string]interface{}); ok {
			if h, exists := srv["hostname"]; exists && h == hostname {
				found = true
				continue // skip this server
			}
		}
		newServers = append(newServers, s)
	}

	if !found {
		return fmt.Errorf("server %s not found", hostname)
	}

	configData["servers"] = newServers

	// Write back to file
	output, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cw.configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateEmailSettings updates email configuration in the config file
func (cw *ConfigWriter) UpdateEmailSettings(enabled bool, from string, to []string, smtpHost string, smtpPort int, username, password string) error {
	data, err := os.ReadFile(cw.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update email section
	emailConfig := map[string]interface{}{
		"enabled":   enabled,
		"from":      from,
		"to":        to,
		"smtp_host": smtpHost,
		"smtp_port": smtpPort,
	}

	if username != "" {
		emailConfig["username"] = username
	}
	if password != "" {
		emailConfig["password"] = password
	}

	configData["email"] = emailConfig

	// Write back to file
	output, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cw.configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateAlertSettings updates alert configuration in the config file
func (cw *ConfigWriter) UpdateAlertSettings(enabled bool, leadTimeDays, resendIntervalMin int) error {
	data, err := os.ReadFile(cw.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update alerts section
	alertConfig := map[string]interface{}{
		"enabled":             enabled,
		"lead_time_days":      leadTimeDays,
		"resend_interval_min": resendIntervalMin,
	}

	configData["alerts"] = alertConfig

	// Write back to file
	output, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cw.configPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
