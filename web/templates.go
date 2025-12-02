package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"time"

	log "github.com/sirupsen/logrus"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var StaticFS embed.FS

// timeSince returns a human-readable duration since the given time
func timeSince(t time.Time) string {
	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// LoadTemplates loads all HTML templates from the embedded filesystem
func LoadTemplates() *template.Template {
	// Create a FuncMap with custom template functions
	funcMap := template.FuncMap{
		"timeSince": timeSince,
		"add": func(a, b int) int {
			return a + b
		},
	}

	// Parse templates with custom functions
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.html")
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
