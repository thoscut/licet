package handlers

import (
	"html/template"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/models"
	"github.com/thoscut/licet/internal/services"
	"github.com/thoscut/licet/web"
)

type WebHandler struct {
	licenseService *services.LicenseService
	alertService   *services.AlertService
	cfg            *config.Config
	templates      *template.Template
}

func NewWebHandler(licenseService *services.LicenseService, alertService *services.AlertService, cfg *config.Config) *WebHandler {
	// Load templates from embedded filesystem via web package
	tmpl := web.LoadTemplates()

	return &WebHandler{
		licenseService: licenseService,
		alertService:   alertService,
		cfg:            cfg,
		templates:      tmpl,
	}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	servers, err := h.licenseService.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	// Query status for each server
	type ServerWithStatus struct {
		Server models.LicenseServer
		Status models.ServerStatus
	}

	serversWithStatus := make([]ServerWithStatus, 0, len(servers))
	for _, server := range servers {
		result, err := h.licenseService.QueryServer(server.Hostname, server.Type)
		if err != nil {
			log.WithError(err).Warnf("Failed to query server %s", server.Hostname)
			// Still add the server with error status
			serversWithStatus = append(serversWithStatus, ServerWithStatus{
				Server: server,
				Status: models.ServerStatus{
					Hostname: server.Hostname,
					Service:  "down",
					Version:  "",
					Message:  err.Error(),
				},
			})
			continue
		}
		serversWithStatus = append(serversWithStatus, ServerWithStatus{
			Server: server,
			Status: result.Status,
		})
	}

	data := map[string]interface{}{
		"Title":              "License Server Status",
		"Servers":            serversWithStatus,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// sortFeaturesByName sorts features alphabetically by name
func sortFeaturesByName(features []models.Feature) {
	sort.Slice(features, func(i, j int) bool {
		return features[i].Name < features[j].Name
	})
}

func (h *WebHandler) Details(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "server")

	// Find the server configuration to get the type
	var serverType string
	for _, srv := range h.cfg.Servers {
		if srv.Hostname == hostname {
			serverType = srv.Type
			break
		}
	}

	if serverType == "" {
		http.Error(w, "Server not found in configuration", http.StatusNotFound)
		return
	}

	// Query the live server to get current features and users
	result, err := h.licenseService.QueryServer(hostname, serverType)
	if err != nil {
		// Fall back to database features if live query fails
		features, dbErr := h.licenseService.GetFeatures(r.Context(), hostname)
		if dbErr != nil {
			http.Error(w, "Failed to get server data", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"Title":              "Server Details",
			"Hostname":           hostname,
			"Features":           features,
			"Users":              []interface{}{}, // Empty users if query failed
			"Error":              err.Error(),
			"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
			"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
			"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
		}

		if err := h.templates.ExecuteTemplate(w, "details.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	data := map[string]interface{}{
		"Title":              "Server Details",
		"Hostname":           hostname,
		"Features":           result.Features,
		"Users":              result.Users,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "details.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Expiration(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "server")

	// Use the specialized method that filters for features with expiration dates
	features, err := h.licenseService.GetFeaturesWithExpiration(r.Context(), hostname)
	if err != nil {
		http.Error(w, "Failed to get features", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":              "License Expiration",
		"Hostname":           hostname,
		"Features":           features,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "expiration.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Utilization(w http.ResponseWriter, r *http.Request) {
	// Check if utilization page is enabled
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":              "License Utilization Overview",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "utilization_overview.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) UtilizationTrends(w http.ResponseWriter, r *http.Request) {
	// Check if utilization page is enabled
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":              "Usage Trends",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "utilization_trends.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) UtilizationAnalytics(w http.ResponseWriter, r *http.Request) {
	// Check if utilization page is enabled
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":              "Predictive Analytics",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "utilization_analytics.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) UtilizationStats(w http.ResponseWriter, r *http.Request) {
	// Check if utilization page is enabled
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":              "Detailed Statistics",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "utilization_stats.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Denials(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":              "License Denials",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "denials.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Alerts(w http.ResponseWriter, r *http.Request) {
	// Get all active alerts from the last 30 days (both sent and unsent)
	alerts, err := h.alertService.GetActiveAlerts(r.Context())
	if err != nil {
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":              "License Alerts",
		"Alerts":             alerts,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "alerts.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Settings(w http.ResponseWriter, r *http.Request) {
	// Check if settings page is enabled
	if !h.cfg.Server.SettingsEnabled {
		http.Error(w, "Settings page is disabled", http.StatusForbidden)
		return
	}

	servers, err := h.licenseService.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	// Check utility status
	checker := services.NewUtilityChecker()
	utilities := checker.CheckAll()

	data := map[string]interface{}{
		"Title":              "Application Settings",
		"ServerPort":         h.cfg.Server.Port,
		"DatabaseType":       h.cfg.Database.Type,
		"TotalServers":       len(servers),
		"Servers":            servers,
		"Utilities":          utilities,
		"EmailConfig":        h.cfg.Email,
		"AlertConfig":        h.cfg.Alerts,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Statistics(w http.ResponseWriter, r *http.Request) {
	// Check if statistics page is enabled
	if !h.cfg.Server.StatisticsEnabled {
		http.Error(w, "Statistics page is disabled", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"Title":              "Statistics Dashboard",
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
	}

	if err := h.templates.ExecuteTemplate(w, "statistics.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
