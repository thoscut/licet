package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"licet/internal/models"
	"licet/internal/services"
)

// ExportHandler handles data export operations
type ExportHandler struct {
	licenseService *services.LicenseService
}

// NewExportHandler creates a new export handler
func NewExportHandler(licenseService *services.LicenseService) *ExportHandler {
	return &ExportHandler{
		licenseService: licenseService,
	}
}

// ExportServers exports server list in requested format
func (h *ExportHandler) ExportServers(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	servers, err := h.licenseService.GetAllServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.writeServersCSV(w, servers)
	default:
		h.writeJSON(w, map[string]interface{}{
			"servers":     servers,
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"count":       len(servers),
		})
	}
}

// ExportFeatures exports features for a server in requested format
func (h *ExportHandler) ExportFeatures(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	server := r.URL.Query().Get("server")
	if server == "" {
		http.Error(w, "server parameter required", http.StatusBadRequest)
		return
	}

	features, err := h.licenseService.GetFeatures(r.Context(), server)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.writeFeaturesCSV(w, features, server)
	default:
		h.writeJSON(w, map[string]interface{}{
			"server":      server,
			"features":    features,
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"count":       len(features),
		})
	}
}

// ExportUtilization exports utilization data in requested format
func (h *ExportHandler) ExportUtilization(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	serverFilter := r.URL.Query().Get("server")

	utilization, err := h.licenseService.GetCurrentUtilization(r.Context(), serverFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.writeUtilizationCSV(w, utilization)
	default:
		h.writeJSON(w, map[string]interface{}{
			"utilization": utilization,
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"count":       len(utilization),
		})
	}
}

// ExportUtilizationHistory exports historical utilization data
func (h *ExportHandler) ExportUtilizationHistory(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	server := r.URL.Query().Get("server")
	feature := r.URL.Query().Get("feature")
	daysStr := r.URL.Query().Get("days")

	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	history, err := h.licenseService.GetUtilizationHistory(r.Context(), server, feature, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.writeHistoryCSV(w, history, server, feature)
	default:
		h.writeJSON(w, map[string]interface{}{
			"server":      server,
			"feature":     feature,
			"days":        days,
			"history":     history,
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"count":       len(history),
		})
	}
}

// ExportStats exports utilization statistics
func (h *ExportHandler) ExportStats(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	server := r.URL.Query().Get("server")
	daysStr := r.URL.Query().Get("days")

	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	stats, err := h.licenseService.GetUtilizationStats(r.Context(), server, days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.writeStatsCSV(w, stats)
	default:
		h.writeJSON(w, map[string]interface{}{
			"server":      server,
			"days":        days,
			"stats":       stats,
			"exported_at": time.Now().UTC().Format(time.RFC3339),
			"count":       len(stats),
		})
	}
}

// ExportReport generates a comprehensive utilization report
func (h *ExportHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	server := r.URL.Query().Get("server")
	daysStr := r.URL.Query().Get("days")

	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	// Gather all data for the report
	servers, _ := h.licenseService.GetAllServers()
	utilization, _ := h.licenseService.GetCurrentUtilization(r.Context(), server)
	stats, _ := h.licenseService.GetUtilizationStats(r.Context(), server, days)
	heatmap, _ := h.licenseService.GetHeatmapData(r.Context(), server, days)

	report := map[string]interface{}{
		"report_type":   "license_utilization",
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"period_days":   days,
		"server_filter": server,
		"summary": map[string]interface{}{
			"total_servers":  len(servers),
			"total_features": len(utilization),
		},
		"servers":     servers,
		"utilization": utilization,
		"statistics":  stats,
		"heatmap":     heatmap,
	}

	switch format {
	case "csv":
		h.writeReportCSV(w, utilization, stats)
	default:
		h.writeJSON(w, report)
	}
}

// Helper methods for writing responses

func (h *ExportHandler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=export_%s.json", time.Now().Format("20060102_150405")))
	json.NewEncoder(w).Encode(data)
}

func (h *ExportHandler) writeServersCSV(w http.ResponseWriter, servers []models.LicenseServer) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=servers_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"ID", "Hostname", "Description", "Type", "WebUI", "Created At", "Updated At"})

	// Write data
	for _, server := range servers {
		writer.Write([]string{
			strconv.FormatInt(server.ID, 10),
			server.Hostname,
			server.Description,
			server.Type,
			server.WebUI,
			server.CreatedAt.Format(time.RFC3339),
			server.UpdatedAt.Format(time.RFC3339),
		})
	}
}

func (h *ExportHandler) writeFeaturesCSV(w http.ResponseWriter, features []models.Feature, server string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=features_%s_%s.csv", sanitizeFilename(server), time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"Server", "Feature", "Version", "Vendor Daemon",
		"Total Licenses", "Used Licenses", "Available",
		"Expiration Date", "Last Updated",
	})

	// Write data
	for _, feature := range features {
		writer.Write([]string{
			feature.ServerHostname,
			feature.Name,
			feature.Version,
			feature.VendorDaemon,
			strconv.Itoa(feature.TotalLicenses),
			strconv.Itoa(feature.UsedLicenses),
			strconv.Itoa(feature.AvailableLicenses()),
			feature.ExpirationDate.Format("2006-01-02"),
			feature.LastUpdated.Format(time.RFC3339),
		})
	}
}

func (h *ExportHandler) writeUtilizationCSV(w http.ResponseWriter, utilization []models.UtilizationData) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=utilization_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"Server", "Feature", "Version", "Vendor Daemon",
		"Total Licenses", "Used Licenses", "Available",
		"Utilization %",
	})

	// Write data
	for _, util := range utilization {
		writer.Write([]string{
			util.ServerHostname,
			util.FeatureName,
			util.Version,
			util.VendorDaemon,
			strconv.Itoa(util.TotalLicenses),
			strconv.Itoa(util.UsedLicenses),
			strconv.Itoa(util.AvailableLicenses),
			fmt.Sprintf("%.2f", util.UtilizationPct),
		})
	}
}

func (h *ExportHandler) writeHistoryCSV(w http.ResponseWriter, history []models.UtilizationHistoryPoint, server, feature string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=history_%s_%s_%s.csv", sanitizeFilename(server), sanitizeFilename(feature), time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Timestamp", "Users Count"})

	// Write data
	for _, point := range history {
		writer.Write([]string{
			point.Timestamp,
			strconv.Itoa(point.UsersCount),
		})
	}
}

func (h *ExportHandler) writeStatsCSV(w http.ResponseWriter, stats []models.UtilizationStats) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=stats_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"Server", "Feature", "Avg Usage", "Peak Usage",
		"Min Usage", "Total Licenses", "Avg Utilization %",
	})

	// Write data
	for _, stat := range stats {
		avgUtilization := 0.0
		if stat.TotalLicenses > 0 {
			avgUtilization = (stat.AvgUsage / float64(stat.TotalLicenses)) * 100
		}
		writer.Write([]string{
			stat.ServerHostname,
			stat.FeatureName,
			fmt.Sprintf("%.2f", stat.AvgUsage),
			strconv.Itoa(stat.PeakUsage),
			strconv.Itoa(stat.MinUsage),
			strconv.Itoa(stat.TotalLicenses),
			fmt.Sprintf("%.2f", avgUtilization),
		})
	}
}

func (h *ExportHandler) writeReportCSV(w http.ResponseWriter, utilization []models.UtilizationData, stats []models.UtilizationStats) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s.csv", time.Now().Format("20060102_150405")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write report metadata
	writer.Write([]string{"License Utilization Report"})
	writer.Write([]string{"Generated At", time.Now().UTC().Format(time.RFC3339)})
	writer.Write([]string{""})

	// Write utilization summary
	writer.Write([]string{"Current Utilization"})
	writer.Write([]string{
		"Server", "Feature", "Version", "Total", "Used", "Available", "Utilization %",
	})

	for _, util := range utilization {
		writer.Write([]string{
			util.ServerHostname,
			util.FeatureName,
			util.Version,
			strconv.Itoa(util.TotalLicenses),
			strconv.Itoa(util.UsedLicenses),
			strconv.Itoa(util.AvailableLicenses),
			fmt.Sprintf("%.2f", util.UtilizationPct),
		})
	}

	writer.Write([]string{""})

	// Write statistics
	writer.Write([]string{"Usage Statistics"})
	writer.Write([]string{
		"Server", "Feature", "Avg Usage", "Peak Usage", "Min Usage", "Total Licenses",
	})

	for _, stat := range stats {
		writer.Write([]string{
			stat.ServerHostname,
			stat.FeatureName,
			fmt.Sprintf("%.2f", stat.AvgUsage),
			strconv.Itoa(stat.PeakUsage),
			strconv.Itoa(stat.MinUsage),
			strconv.Itoa(stat.TotalLicenses),
		})
	}
}

// sanitizeFilename removes or replaces characters that are not safe for filenames
func sanitizeFilename(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else if c == '@' || c == ':' || c == '/' || c == '\\' || c == ' ' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "export"
	}
	return string(result)
}
