package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/thoscut/licet/internal/models"
	"github.com/thoscut/licet/internal/services"
)

// AddServer handles POST /api/v1/servers - adds a new license server
func AddServer(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var server models.LicenseServer
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

		if err := licenseService.AddServer(server); err != nil {
			http.Error(w, "Failed to add server: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(server)
	}
}

// DeleteServer handles DELETE /api/v1/servers/{hostname} - removes a license server
func DeleteServer(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname := r.URL.Query().Get("hostname")
		if hostname == "" {
			http.Error(w, "Hostname is required", http.StatusBadRequest)
			return
		}

		if err := licenseService.DeleteServer(hostname); err != nil {
			http.Error(w, "Failed to delete server: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
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
