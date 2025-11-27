package handlers

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/services"
	"github.com/thoscut/licet/web"
)

type WebHandler struct {
	licenseService *services.LicenseService
	alertService   *services.AlertService
	templates      *template.Template
}

func NewWebHandler(licenseService *services.LicenseService, alertService *services.AlertService) *WebHandler {
	// Load templates from embedded filesystem via web package
	tmpl := web.LoadTemplates()

	return &WebHandler{
		licenseService: licenseService,
		alertService:   alertService,
		templates:      tmpl,
	}
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	servers, err := h.licenseService.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "License Server Status",
		"Servers": servers,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Errorf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Details(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "server")

	features, err := h.licenseService.GetFeatures(hostname)
	if err != nil {
		http.Error(w, "Failed to get features", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":    "Server Details",
		"Hostname": hostname,
		"Features": features,
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
