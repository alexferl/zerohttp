package main

import (
	"net/http"

	"github.com/a-h/templ"
	zh "github.com/alexferl/zerohttp"
)

// TemplTemplateManager wraps Templ components for cleaner usage
type TemplTemplateManager struct {
	components map[string]func(data any) templ.Component
}

func NewTemplTemplateManager() *TemplTemplateManager {
	return &TemplTemplateManager{
		components: make(map[string]func(data any) templ.Component),
	}
}

// RegisterComponent registers a component factory
func (tm *TemplTemplateManager) RegisterComponent(name string, factory func(data any) templ.Component) {
	tm.components[name] = factory
}

// Render handles Content-Type, status code, and component rendering
func (tm *TemplTemplateManager) Render(w http.ResponseWriter, r *http.Request, status int, name string, data any) error {
	factory, exists := tm.components[name]
	if !exists {
		return zh.NewProblemDetail(404, "Template not found: "+name).Render(w)
	}

	component := factory(data)

	w.Header().Set(zh.HeaderContentType, zh.MIMETextHTML)
	w.WriteHeader(status)

	return component.Render(r.Context(), w)
}

// ComponentFactory creates a factory function for components that take message and description
func ComponentFactory(componentFunc func(message, description string) templ.Component) func(data any) templ.Component {
	return func(data any) templ.Component {
		pd := data.(PageData)
		return componentFunc(pd.Message, pd.Description)
	}
}
