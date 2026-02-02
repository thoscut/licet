package handlers

import (
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"licet/internal/config"
	"licet/internal/models"
	"licet/internal/services"
	"licet/web"
)

type WebHandler struct {
	query        *services.QueryService
	storage      *services.StorageService
	analytics    *services.AnalyticsService
	alertService *services.AlertService
	cfg          *config.Config
	templates    *template.Template
	version      string
}

func NewWebHandler(query *services.QueryService, storage *services.StorageService, analytics *services.AnalyticsService, alertService *services.AlertService, cfg *config.Config, version string) *WebHandler {
	// Load templates from embedded filesystem via web package
	tmpl := web.LoadTemplates()

	return &WebHandler{
		query:        query,
		storage:      storage,
		analytics:    analytics,
		alertService: alertService,
		cfg:          cfg,
		templates:    tmpl,
		version:      version,
	}
}

// baseData returns common template data used by all handlers
func (h *WebHandler) baseData(title string) map[string]interface{} {
	return map[string]interface{}{
		"Title":              title,
		"UtilizationEnabled": h.cfg.Server.UtilizationEnabled,
		"StatisticsEnabled":  h.cfg.Server.StatisticsEnabled,
		"SettingsEnabled":    h.cfg.Server.SettingsEnabled,
		"Version":            h.version,
	}
}

// render executes a template and handles errors consistently
func (h *WebHandler) render(w http.ResponseWriter, template string, data map[string]interface{}) {
	if err := h.templates.ExecuteTemplate(w, template, data); err != nil {
		log.Errorf("Template error rendering %s: %v", template, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	servers, err := h.query.GetAllServers()
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
		result, err := h.query.QueryServer(server.Hostname, server.Type)
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

	data := h.baseData("License Server Status")
	data["Servers"] = serversWithStatus

	h.render(w, "index.html", data)
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
	result, err := h.query.QueryServer(hostname, serverType)
	if err != nil {
		// Fall back to database features if live query fails
		features, dbErr := h.storage.GetFeatures(r.Context(), hostname)
		if dbErr != nil {
			http.Error(w, "Failed to get server data", http.StatusInternalServerError)
			return
		}

		// Calculate last updated from stored features
		var lastUpdated time.Time
		for _, f := range features {
			if f.LastUpdated.After(lastUpdated) {
				lastUpdated = f.LastUpdated
			}
		}

		data := h.baseData("Server Details")
		data["Hostname"] = hostname
		data["Features"] = features
		data["Users"] = []interface{}{}
		data["Error"] = err.Error()
		data["LastUpdated"] = lastUpdated

		h.render(w, "details.html", data)
		return
	}

	data := h.baseData("Server Details")
	data["Hostname"] = hostname
	data["Features"] = result.Features
	data["Users"] = result.Users
	data["LastUpdated"] = time.Now() // Data was just fetched live

	h.render(w, "details.html", data)
}

func (h *WebHandler) Expiration(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "server")

	// Check if user wants to see inactive/historical licenses
	showInactive := r.URL.Query().Get("show_inactive") == "true"

	var features []models.Feature
	var err error

	if showInactive {
		// Show all features including inactive/replaced ones
		features, err = h.storage.GetAllFeaturesWithExpiration(r.Context(), hostname)
	} else {
		// Show only active features (default)
		features, err = h.storage.GetFeaturesWithExpiration(r.Context(), hostname)
	}

	if err != nil {
		http.Error(w, "Failed to get features", http.StatusInternalServerError)
		return
	}

	data := h.baseData("License Expiration")
	data["Hostname"] = hostname
	data["Features"] = features
	data["ShowInactive"] = showInactive

	h.render(w, "expiration.html", data)
}

func (h *WebHandler) Utilization(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := h.baseData("License Utilization Overview")
	h.render(w, "utilization_overview.html", data)
}

func (h *WebHandler) UtilizationTrends(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := h.baseData("Usage Trends")
	h.render(w, "utilization_trends.html", data)
}

func (h *WebHandler) UtilizationAnalytics(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := h.baseData("Predictive Analytics")
	h.render(w, "utilization_analytics.html", data)
}

func (h *WebHandler) UtilizationStats(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.UtilizationEnabled {
		http.Error(w, "Utilization page is disabled", http.StatusForbidden)
		return
	}

	data := h.baseData("Detailed Statistics")
	h.render(w, "utilization_stats.html", data)
}

func (h *WebHandler) Denials(w http.ResponseWriter, r *http.Request) {
	data := h.baseData("License Denials")
	h.render(w, "denials.html", data)
}

func (h *WebHandler) Alerts(w http.ResponseWriter, r *http.Request) {
	// Get all active alerts from the last 30 days (both sent and unsent)
	alerts, err := h.alertService.GetActiveAlerts(r.Context())
	if err != nil {
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	data := h.baseData("License Alerts")
	data["Alerts"] = alerts

	h.render(w, "alerts.html", data)
}

func (h *WebHandler) Settings(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.SettingsEnabled {
		http.Error(w, "Settings page is disabled", http.StatusForbidden)
		return
	}

	servers, err := h.query.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	// Check utility status
	checker := services.NewUtilityChecker()
	utilities := checker.CheckAll()

	data := h.baseData("Application Settings")
	data["ServerPort"] = h.cfg.Server.Port
	data["DatabaseType"] = h.cfg.Database.Type
	data["TotalServers"] = len(servers)
	data["Servers"] = servers
	data["Utilities"] = utilities
	data["EmailConfig"] = h.cfg.Email
	data["AlertConfig"] = h.cfg.Alerts

	h.render(w, "settings.html", data)
}

func (h *WebHandler) Statistics(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.StatisticsEnabled {
		http.Error(w, "Statistics page is disabled", http.StatusForbidden)
		return
	}

	data := h.baseData("Statistics Dashboard")
	h.render(w, "statistics.html", data)
}

func (h *WebHandler) DatabaseStats(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Server.SettingsEnabled {
		http.Error(w, "Database stats page requires settings to be enabled", http.StatusForbidden)
		return
	}

	data := h.baseData("Database Statistics")
	data["DatabaseType"] = h.cfg.Database.Type
	h.render(w, "database_stats.html", data)
}
