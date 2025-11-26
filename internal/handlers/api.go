package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/thoscut/licet/internal/services"
)

func ListServers(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servers, err := licenseService.GetAllServers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"servers": servers,
		})
	}
}

func GetServerStatus(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")
		serverType := r.URL.Query().Get("type")

		if serverType == "" {
			serverType = "flexlm" // default
		}

		result, err := licenseService.QueryServer(server, serverType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result.Status)
	}
}

func GetServerFeatures(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")

		features, err := licenseService.GetFeatures(server)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"features": features,
		})
	}
}

func GetServerUsers(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")
		serverType := r.URL.Query().Get("type")

		if serverType == "" {
			serverType = "flexlm"
		}

		result, err := licenseService.QueryServer(server, serverType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": result.Users,
		})
	}
}

func GetFeatureUsage(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		feature := chi.URLParam(r, "feature")
		server := r.URL.Query().Get("server")
		daysStr := r.URL.Query().Get("days")

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		usage, err := licenseService.GetFeatureUsageHistory(server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"usage": usage,
		})
	}
}

func GetAlerts(alertService *services.AlertService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts, err := alertService.GetUnsentAlerts()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alerts": alerts,
		})
	}
}

func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}
}
