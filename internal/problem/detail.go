package problem

import (
	"encoding/json"
	"net/http"
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
