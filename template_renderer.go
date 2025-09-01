package zerohttp

import (
	"embed"
	"html/template"
	"net/http"
)

// TemplateRenderer defines the interface for rendering HTML templates
type TemplateRenderer interface {
	Render(w http.ResponseWriter, code int, name string, data any) error
}

// TemplateManager implements TemplateRenderer using html/template
type TemplateManager struct {
	templates *template.Template
}

// NewTemplateManager creates a new TemplateManager with parsed templates from the embedded filesystem
func NewTemplateManager(templatesFS embed.FS, pattern string) TemplateRenderer {
	tmpl := template.Must(template.ParseFS(templatesFS, pattern))
	return &TemplateManager{templates: tmpl}
}

// Render renders the specified template with the given data and status code
func (tm *TemplateManager) Render(w http.ResponseWriter, code int, name string, data any) error {
	return R.Template(w, code, tm.templates, name, data)
}
