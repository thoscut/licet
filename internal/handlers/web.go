package handlers

import (
	"html/template"
	"net/http"

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
		result, _ := h.licenseService.QueryServer(server.Hostname, server.Type)
		serversWithStatus = append(serversWithStatus, ServerWithStatus{
			Server: server,
			Status: result.Status,
		})
	}

	data := map[string]interface{}{
		"Title":   "License Server Status",
		"Servers": serversWithStatus,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
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
		features, dbErr := h.licenseService.GetFeatures(hostname)
		if dbErr != nil {
			http.Error(w, "Failed to get server data", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"Title":    "Server Details",
			"Hostname": hostname,
			"Features": features,
			"Users":    []interface{}{}, // Empty users if query failed
			"Error":    err.Error(),
		}

		if err := h.templates.ExecuteTemplate(w, "details.html", data); err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	data := map[string]interface{}{
		"Title":    "Server Details",
		"Hostname": hostname,
		"Features": result.Features,
		"Users":    result.Users,
	}

	if err := h.templates.ExecuteTemplate(w, "details.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Expiration(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "server")

	features, err := h.licenseService.GetFeatures(hostname)
	if err != nil {
		http.Error(w, "Failed to get features", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "License Expiration",
		"Hostname": hostname,
		"Features": features,
	}

	if err := h.templates.ExecuteTemplate(w, "expiration.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Utilization(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "License Utilization",
	}

	if err := h.templates.ExecuteTemplate(w, "utilization.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Denials(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "License Denials",
	}

	if err := h.templates.ExecuteTemplate(w, "denials.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Alerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.alertService.GetUnsentAlerts()
	if err != nil {
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":  "License Alerts",
		"Alerts": alerts,
	}

	if err := h.templates.ExecuteTemplate(w, "alerts.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Settings(w http.ResponseWriter, r *http.Request) {
	servers, err := h.licenseService.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	// Check utility status
	checker := services.NewUtilityChecker()
	utilities := checker.CheckAll()

	data := map[string]interface{}{
		"Title":        "Application Settings",
		"ServerPort":   h.cfg.Server.Port,
		"DatabaseType": h.cfg.Database.Type,
		"TotalServers": len(servers),
		"Servers":      servers,
		"Utilities":    utilities,
		"EmailConfig":  h.cfg.Email,
		"AlertConfig":  h.cfg.Alerts,
	}

	if err := h.templates.ExecuteTemplate(w, "settings.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Statistics(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Statistics Dashboard",
	}

	if err := h.templates.ExecuteTemplate(w, "statistics.html", data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
