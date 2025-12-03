package services

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"licet/internal/config"
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

// readConfig reads and parses the config file
func (cw *ConfigWriter) readConfig() (map[string]interface{}, error) {
	data, err := os.ReadFile(cw.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configData map[string]interface{}
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return configData, nil
}

// writeConfigAtomic writes config data atomically (write to temp, then rename)
func (cw *ConfigWriter) writeConfigAtomic(configData map[string]interface{}) error {
	output, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temp file in same directory (for atomic rename)
	dir := filepath.Dir(cw.configPath)
	tmpFile, err := os.CreateTemp(dir, "config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(output); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, cw.configPath); err != nil {
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	tmpPath = "" // Prevent cleanup since rename succeeded
	return nil
}

// UpdateSection updates a section of the config file with the given data
func (cw *ConfigWriter) UpdateSection(section string, data map[string]interface{}) error {
	configData, err := cw.readConfig()
	if err != nil {
		return err
	}

	configData[section] = data

	return cw.writeConfigAtomic(configData)
}

// AddServer adds a new server to the config file
func (cw *ConfigWriter) AddServer(server config.LicenseServer) error {
	configData, err := cw.readConfig()
	if err != nil {
		return err
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

	return cw.writeConfigAtomic(configData)
}

// DeleteServer removes a server from the config file
func (cw *ConfigWriter) DeleteServer(hostname string) error {
	configData, err := cw.readConfig()
	if err != nil {
		return err
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

	return cw.writeConfigAtomic(configData)
}

// UpdateEmailSettings updates email configuration in the config file
func (cw *ConfigWriter) UpdateEmailSettings(enabled bool, from string, to []string, smtpHost string, smtpPort int, username, password string) error {
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

	return cw.UpdateSection("email", emailConfig)
}

// UpdateAlertSettings updates alert configuration in the config file
func (cw *ConfigWriter) UpdateAlertSettings(enabled bool, leadTimeDays, resendIntervalMin int) error {
	alertConfig := map[string]interface{}{
		"enabled":             enabled,
		"lead_time_days":      leadTimeDays,
		"resend_interval_min": resendIntervalMin,
	}

	return cw.UpdateSection("alerts", alertConfig)
}
