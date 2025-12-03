package handlers

import (
	"encoding/json"
	"net/http"

	"licet/internal/config"
	"licet/internal/services"
)

// AddServer handles POST /api/v1/servers - adds a new license server to config file
func AddServer(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if settings page is enabled
		if !cfg.Server.SettingsEnabled {
			http.Error(w, "Settings page is disabled", http.StatusForbidden)
			return
		}

		var server config.LicenseServer
		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if server.Hostname == "" || server.Type == "" {
			http.Error(w, "Hostname and type are required", http.StatusBadRequest)
			return
		}

		// Validate server type (only supported types)
		validTypes := map[string]bool{
			"flexlm": true,
			"rlm":    true,
		}
		if !validTypes[server.Type] {
			http.Error(w, "Invalid server type", http.StatusBadRequest)
			return
		}

		// Write to config file
		configWriter := services.NewConfigWriter()
		if err := configWriter.AddServer(server); err != nil {
			http.Error(w, "Failed to add server: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Server added successfully. Please restart the application for changes to take effect.",
			"server":  server,
		})
	}
}

// DeleteServer handles DELETE /api/v1/servers - removes a license server from config file
func DeleteServer(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if settings page is enabled
		if !cfg.Server.SettingsEnabled {
			http.Error(w, "Settings page is disabled", http.StatusForbidden)
			return
		}

		hostname := r.URL.Query().Get("hostname")
		if hostname == "" {
			http.Error(w, "Hostname is required", http.StatusBadRequest)
			return
		}

		// Delete from config file
		configWriter := services.NewConfigWriter()
		if err := configWriter.DeleteServer(hostname); err != nil {
			http.Error(w, "Failed to delete server: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Server deleted successfully. Please restart the application for changes to take effect.",
		})
	}
}

// CheckUtilities handles GET /api/v1/utilities/check - checks available license utilities
func CheckUtilities() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checker := services.NewUtilityChecker()
		statuses := checker.CheckAll()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"utilities": statuses,
		})
	}
}

// TestServerConnection handles POST /api/v1/servers/test - tests connection to a license server
func TestServerConnection(cfg *config.Config, licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if settings page is enabled
		if !cfg.Server.SettingsEnabled {
			http.Error(w, "Settings page is disabled", http.StatusForbidden)
			return
		}

		var req struct {
			Hostname string `json:"hostname"`
			Type     string `json:"type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Hostname == "" || req.Type == "" {
			http.Error(w, "Hostname and type are required", http.StatusBadRequest)
			return
		}

		// Try to query the server
		result, err := licenseService.QueryServer(req.Hostname, req.Type)

		w.Header().Set("Content-Type", "application/json")
		if err != nil || result.Error != nil {
			errorMsg := "Connection failed"
			if err != nil {
				errorMsg = err.Error()
			} else if result.Error != nil {
				errorMsg = result.Error.Error()
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": errorMsg,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Connection successful",
			"status":  result.Status,
		})
	}
}

// UpdateEmailSettings handles POST /api/v1/settings/email - updates email configuration
func UpdateEmailSettings(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if settings page is enabled
		if !cfg.Server.SettingsEnabled {
			http.Error(w, "Settings page is disabled", http.StatusForbidden)
			return
		}

		var emailConfig struct {
			Enabled  bool     `json:"enabled"`
			From     string   `json:"from"`
			To       []string `json:"to"`
			SMTPHost string   `json:"smtp_host"`
			SMTPPort int      `json:"smtp_port"`
			Username string   `json:"username"`
			Password string   `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&emailConfig); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		configWriter := services.NewConfigWriter()
		if err := configWriter.UpdateEmailSettings(emailConfig.Enabled, emailConfig.From, emailConfig.To,
			emailConfig.SMTPHost, emailConfig.SMTPPort, emailConfig.Username, emailConfig.Password); err != nil {
			http.Error(w, "Failed to update email settings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Email settings updated successfully. Please restart the application for changes to take effect.",
		})
	}
}

// UpdateAlertSettings handles POST /api/v1/settings/alerts - updates alert configuration
func UpdateAlertSettings(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if settings page is enabled
		if !cfg.Server.SettingsEnabled {
			http.Error(w, "Settings page is disabled", http.StatusForbidden)
			return
		}

		var alertConfig struct {
			Enabled           bool `json:"enabled"`
			LeadTimeDays      int  `json:"lead_time_days"`
			ResendIntervalMin int  `json:"resend_interval_min"`
		}

		if err := json.NewDecoder(r.Body).Decode(&alertConfig); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		configWriter := services.NewConfigWriter()
		if err := configWriter.UpdateAlertSettings(alertConfig.Enabled, alertConfig.LeadTimeDays, alertConfig.ResendIntervalMin); err != nil {
			http.Error(w, "Failed to update alert settings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Alert settings updated successfully. Please restart the application for changes to take effect.",
		})
	}
}
