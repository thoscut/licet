package web

import (
	"embed"
	"html/template"

	log "github.com/sirupsen/logrus"
)

//go:embed templates/*.html
var templateFS embed.FS

// LoadTemplates loads all HTML templates from the embedded filesystem
func LoadTemplates() *template.Template {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to load embedded templates: %v", err)
	}

	log.Infof("Successfully loaded %d embedded template(s)", len(tmpl.Templates()))
	return tmpl
}
