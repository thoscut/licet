package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/services"
)

// AddServer handles POST /api/v1/servers - adds a new license server to config file
func AddServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Validate server type
		validTypes := map[string]bool{
			"flexlm": true,
			"rlm":    true,
			"spm":    true,
			"sesi":   true,
			"rvl":    true,
			"tweak":  true,
			"pixar":  true,
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
func DeleteServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
