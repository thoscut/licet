package web

import (
	"embed"
	"html/template"
	"io/fs"

	log "github.com/sirupsen/logrus"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var StaticFS embed.FS

// LoadTemplates loads all HTML templates from the embedded filesystem
func LoadTemplates() *template.Template {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to load embedded templates: %v", err)
	}

	log.Infof("Successfully loaded %d embedded template(s)", len(tmpl.Templates()))
	return tmpl
}

// GetStaticFS returns the embedded static filesystem
func GetStaticFS() fs.FS {
	staticFS, err := fs.Sub(StaticFS, "static")
	if err != nil {
		log.Fatalf("Failed to load embedded static files: %v", err)
	}
	return staticFS
}
