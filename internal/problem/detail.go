package problem

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Detail represents an RFC 9457 Problem Details response.
// It provides a standardized way to carry machine-readable details of errors
// in HTTP response bodies.
type Detail struct {
	// Type is a URI reference that identifies the problem type
	Type string `json:"type,omitempty"`

	// Title is a short, human-readable summary of the problem type
	Title string `json:"title"`

	// Status is the HTTP status code for this occurrence of the problem
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference that identifies the specific occurrence
	Instance string `json:"instance,omitempty"`

	// Extensions contains additional problem-specific data
	Extensions map[string]any `json:"-"`
}

// NewDetail creates a new Detail with the given status code and detail message.
// The title is automatically set based on the HTTP status code.
func NewDetail(statusCode int, detail string) *Detail {
	return &Detail{
		Title:      http.StatusText(statusCode),
		Status:     statusCode,
		Detail:     detail,
		Extensions: make(map[string]any),
	}
}

// MarshalJSON implements custom JSON marshaling to include extensions as top-level fields
func (p *Detail) MarshalJSON() ([]byte, error) {
	result := map[string]any{
		"title":  p.Title,
		"status": p.Status,
	}

	if p.Type != "" {
		result["type"] = p.Type
	}
	if p.Detail != "" {
		result["detail"] = p.Detail
	}
	if p.Instance != "" {
		result["instance"] = p.Instance
	}

	for k, v := range p.Extensions {
		result[k] = v
	}

	return json.Marshal(result)
}

// Set adds an extension field to the problem detail and returns the Detail
// for method chaining. Extension fields are included as top-level JSON properties.
func (p *Detail) Set(key string, value any) *Detail {
	if p.Extensions == nil {
		p.Extensions = make(map[string]any)
	}
	p.Extensions[key] = value
	return p
}

// Render writes the Detail as an HTTP response
func (p *Detail) Render(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	return json.NewEncoder(w).Encode(p)
}

// RenderAuto writes the Detail as an HTTP response, automatically selecting the
// content type based on the Accept header. Returns JSON if the client accepts
// application/json or application/problem+json; otherwise returns plain text.
func (p *Detail) RenderAuto(w http.ResponseWriter, r *http.Request) error {
	if !AcceptsJSON(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(p.Status)
		text := p.Detail
		if text == "" {
			text = p.Title
		}
		if text != "" {
			_, _ = w.Write([]byte(text + "\n"))
		}
		return nil
	}
	return p.Render(w)
}

// ValidationError represents a single validation error with optional field location information
type ValidationError struct {
	// Detail describes what went wrong with the validation
	Detail string `json:"detail"`

	// Pointer is a JSON Pointer (RFC 6901) to the field that failed validation
	Pointer string `json:"pointer,omitempty"`

	// Field is the name of the field that failed validation (alternative to Pointer)
	Field string `json:"field,omitempty"`
}

// NewValidationDetail creates a problem detail for validation errors (HTTP 422).
// It accepts any slice type for errors, allowing custom validation error structures.
// The errors are added as an "errors" extension field.
func NewValidationDetail[T any](detail string, errors []T) *Detail {
	problem := NewDetail(http.StatusUnprocessableEntity, detail)
	problem.Set("errors", errors)
	return problem
}

// AcceptsJSON checks if the client accepts JSON responses based on the Accept header.
// Returns true if the Accept header includes application/json or application/problem+json.
func AcceptsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}
	if strings.Contains(accept, "application/json") ||
		strings.Contains(accept, "application/problem+json") {
		return true
	}
	return strings.Contains(accept, "*/*")
}
