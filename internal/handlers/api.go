package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"licet/internal/middleware"
	"licet/internal/services"
)

// paginationConfig is the default pagination configuration for API endpoints
var paginationConfig = middleware.DefaultPaginationConfig()

func ListServers(query *services.QueryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servers, err := query.GetAllServers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if pagination is requested
		if r.URL.Query().Get("limit") != "" || r.URL.Query().Get("page") != "" {
			pagination := middleware.ParsePagination(r, paginationConfig)
			paginatedServers, total := middleware.ApplyPagination(servers, pagination)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(middleware.NewPaginatedResponse(paginatedServers, total, pagination))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"servers": servers,
			"total":   len(servers),
		})
	}
}

func GetServerStatus(query *services.QueryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")
		serverType := r.URL.Query().Get("type")

		if serverType == "" {
			serverType = "flexlm" // default
		}

		result, err := query.QueryServer(server, serverType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result.Status)
	}
}

func GetServerFeatures(storage *services.StorageService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")

		features, err := storage.GetFeatures(r.Context(), server)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if pagination is requested
		if r.URL.Query().Get("limit") != "" || r.URL.Query().Get("page") != "" {
			pagination := middleware.ParsePagination(r, paginationConfig)
			paginatedFeatures, total := middleware.ApplyPagination(features, pagination)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(middleware.NewPaginatedResponse(paginatedFeatures, total, pagination))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"features": features,
			"total":    len(features),
		})
	}
}

func GetServerUsers(query *services.QueryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := chi.URLParam(r, "server")
		serverType := r.URL.Query().Get("type")

		if serverType == "" {
			serverType = "flexlm"
		}

		result, err := query.QueryServer(server, serverType)
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

func GetFeatureUsage(storage *services.StorageService) http.HandlerFunc {
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

		usage, err := storage.GetFeatureUsageHistory(r.Context(), server, feature, days)
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

		// Check if pagination is requested
		if r.URL.Query().Get("limit") != "" || r.URL.Query().Get("page") != "" {
			pagination := middleware.ParsePagination(r, paginationConfig)
			paginatedAlerts, total := middleware.ApplyPagination(alerts, pagination)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(middleware.NewPaginatedResponse(paginatedAlerts, total, pagination))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"alerts": alerts,
			"total":  len(alerts),
		})
	}
}

func Health(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"version": version,
		})
	}
}

// GetCurrentUtilization returns current utilization for all features across all servers
func GetCurrentUtilization(analytics *services.AnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverFilter := r.URL.Query().Get("server")

		utilization, err := analytics.GetCurrentUtilization(r.Context(), serverFilter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if pagination is requested
		if r.URL.Query().Get("limit") != "" || r.URL.Query().Get("page") != "" {
			pagination := middleware.ParsePagination(r, paginationConfig)
			paginatedUtilization, total := middleware.ApplyPagination(utilization, pagination)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(middleware.NewPaginatedResponse(paginatedUtilization, total, pagination))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"utilization": utilization,
			"total":       len(utilization),
		})
	}
}

// GetUtilizationHistory returns time-series usage data for charting
func GetUtilizationHistory(analytics *services.AnalyticsService) http.HandlerFunc {
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

		history, err := analytics.GetUtilizationHistory(r.Context(), server, feature, days)
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
func GetUtilizationStats(analytics *services.AnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		daysStr := r.URL.Query().Get("days")

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		stats, err := analytics.GetUtilizationStats(r.Context(), server, days)
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
func GetUtilizationHeatmap(analytics *services.AnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		daysStr := r.URL.Query().Get("days")

		days := 7 // Default to last 7 days for heatmap
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		heatmap, err := analytics.GetHeatmapData(r.Context(), server, days)
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
func GetPredictiveAnalytics(analytics *services.AnalyticsService) http.HandlerFunc {
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

		predictions, err := analytics.GetPredictiveAnalytics(r.Context(), server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(predictions)
	}
}

// GetEnhancedStatistics returns comprehensive statistics for a feature
func GetEnhancedStatistics(enhancedAnalytics *services.EnhancedAnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		feature := r.URL.Query().Get("feature")
		daysStr := r.URL.Query().Get("days")

		if server == "" || feature == "" {
			http.Error(w, "server and feature parameters required", http.StatusBadRequest)
			return
		}

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		stats, err := enhancedAnalytics.GetEnhancedStatistics(r.Context(), server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// GetTrendAnalysis returns detailed trend analysis for a feature
func GetTrendAnalysis(enhancedAnalytics *services.EnhancedAnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server := r.URL.Query().Get("server")
		feature := r.URL.Query().Get("feature")
		daysStr := r.URL.Query().Get("days")

		if server == "" || feature == "" {
			http.Error(w, "server and feature parameters required", http.StatusBadRequest)
			return
		}

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		analysis, err := enhancedAnalytics.GetTrendAnalysis(r.Context(), server, feature, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(analysis)
	}
}

// GetCapacityPlanningReport returns a comprehensive capacity planning report
func GetCapacityPlanningReport(enhancedAnalytics *services.EnhancedAnalyticsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		daysStr := r.URL.Query().Get("days")

		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		report, err := enhancedAnalytics.GetCapacityPlanningReport(r.Context(), days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

// GetDatabaseStats returns comprehensive database statistics
func GetDatabaseStats(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := dbStats.GetDatabaseStats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// GetRetentionStats returns data retention statistics
func GetRetentionStats(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := dbStats.GetRetentionStats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// VacuumDatabase runs VACUUM to optimize the database
func VacuumDatabase(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := dbStats.VacuumDatabase(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// CleanupOldData removes data older than the specified number of days
func CleanupOldData(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		table := r.URL.Query().Get("table")
		daysStr := r.URL.Query().Get("days")

		if table == "" {
			http.Error(w, "table parameter required", http.StatusBadRequest)
			return
		}

		days := 90 // Default to 90 days
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		result, err := dbStats.CleanupOldData(r.Context(), table, days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// AnalyzeDatabase runs ANALYZE to update database statistics
func AnalyzeDatabase(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := dbStats.AnalyzeDatabase(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Database analyzed successfully",
		})
	}
}

// CheckpointWAL runs WAL checkpoint (SQLite only)
func CheckpointWAL(dbStats *services.DBStatsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := dbStats.CheckpointWAL(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "WAL checkpoint completed successfully",
		})
	}
}
