package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"licet/internal/services"
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

		features, err := licenseService.GetFeatures(r.Context(), server)
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

		usage, err := licenseService.GetFeatureUsageHistory(r.Context(), server, feature, days)
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
		alerts, err := alertService.GetUnsentAlerts(r.Context())
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

// GetCurrentUtilization returns current utilization for all features across all servers
func GetCurrentUtilization(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverFilter := r.URL.Query().Get("server")

		utilization, err := licenseService.GetCurrentUtilization(r.Context(), serverFilter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"utilization": utilization,
		})
	}
}

// GetUtilizationHistory returns time-series usage data for charting
func GetUtilizationHistory(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		feature := r.URL.Query().Get("feature")
		periodStr := r.URL.Query().Get("period")

		// Default to 7 days
		days := 7
		switch periodStr {
		case "7d":
			days = 7
		case "30d":
			days = 30
		case "90d":
			days = 90
		case "1y":
			days = 365
		}

		history, err := licenseService.GetUtilizationHistory(r.Context(), server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"history": history,
		})
	}
}

// GetUtilizationStats returns aggregated statistics
func GetUtilizationStats(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		daysStr := r.URL.Query().Get("days")

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		stats, err := licenseService.GetUtilizationStats(r.Context(), server, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stats": stats,
		})
	}
}

// GetUtilizationHeatmap returns hour-of-day usage patterns for heatmap visualization
func GetUtilizationHeatmap(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		daysStr := r.URL.Query().Get("days")

		days := 7 // Default to last 7 days for heatmap
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		heatmap, err := licenseService.GetHeatmapData(r.Context(), server, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"heatmap": heatmap,
		})
	}
}

// GetPredictiveAnalytics returns predictive analytics and anomaly detection
func GetPredictiveAnalytics(licenseService *services.LicenseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		feature := r.URL.Query().Get("feature")
		daysStr := r.URL.Query().Get("days")

		if server == "" || feature == "" {
			http.Error(w, "server and feature parameters required", http.StatusBadRequest)
			return
		}

		days := 30 // Default to 30 days for predictions
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		analytics, err := licenseService.GetPredictiveAnalytics(r.Context(), server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(analytics)
	}
}
