package problem

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
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
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
	w.WriteHeader(p.Status)
	return json.NewEncoder(w).Encode(p)
}

// RenderAuto writes the Detail as an HTTP response, automatically selecting the
// content type based on the Accept header. Returns JSON if the client accepts
// application/json or application/problem+json; otherwise returns plain text.
func (p *Detail) RenderAuto(w http.ResponseWriter, r *http.Request) error {
	if !AcceptsJSON(r) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
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
// Returns true if the Accept header includes application/json or application/problem+json
// with higher priority than text/html, or if */* is present without explicit HTML preference.
// Explicit q=0 refusals (e.g., "application/json;q=0, */*") are respected per RFC 7231.
func AcceptsJSON(r *http.Request) bool {
	accept := r.Header.Get(httpx.HeaderAccept)
	if accept == "" {
		return false
	}

	// Parse explicit types only (no wildcard expansion)
	jsonQ, jsonExplicit := parseAcceptQualityExact(accept, httpx.MIMEApplicationJSON, httpx.MIMEApplicationProblemJSON)
	htmlQ, _ := parseAcceptQualityExact(accept, httpx.MIMETextHTML)
	// Parse wildcards separately
	wildcardQ := parseWildcardQuality(accept)

	// Explicit HTML preference wins over explicit JSON
	if htmlQ > jsonQ {
		return false
	}
	// Explicit JSON listed with positive quality and not outranked
	if jsonQ > 0 {
		return true
	}
	// Only use wildcard if JSON was never explicitly mentioned (not even as q=0 refusal)
	return !jsonExplicit && wildcardQ > 0 && htmlQ < wildcardQ
}

// parseAcceptQualityExact parses the Accept header and returns the highest quality value
// for any of the specified media types, using exact matching only (no wildcards).
// Returns the max quality and a boolean indicating if any of the types were explicitly present.
func parseAcceptQualityExact(accept string, mediaTypes ...string) (q float64, found bool) {
	maxQ := 0.0
	wasFound := false

	eachAcceptEntry(accept, func(mediaType string, quality float64) {
		// Skip wildcards - type/* wildcards are silently dropped (intentional limitation)
		if mediaType == "*/*" || strings.HasSuffix(mediaType, "/*") {
			return
		}

		for _, mt := range mediaTypes {
			if mediaType == mt {
				wasFound = true
				if quality > maxQ {
					maxQ = quality
				}
			}
		}
	})

	return maxQ, wasFound
}

// parseWildcardQuality parses the Accept header and returns the highest quality value
// for */* wildcard entries only.
func parseWildcardQuality(accept string) float64 {
	maxQ := 0.0

	eachAcceptEntry(accept, func(mediaType string, quality float64) {
		// Only match */*
		if mediaType == "*/*" && quality > maxQ {
			maxQ = quality
		}
	})

	return maxQ
}

// eachAcceptEntry iterates over each entry in an Accept header, yielding the media type
// and quality value (default 1.0) for each.
func eachAcceptEntry(accept string, fn func(mediaType string, q float64)) {
	for _, r := range strings.Split(accept, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}

		parts := strings.Split(r, ";")
		mediaType := strings.TrimSpace(parts[0])

		q := 1.0
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if prefix, ok := strings.CutPrefix(p, "q="); ok {
				if qv, err := parseQuality(prefix); err == nil {
					q = qv
				}
				break
			}
		}

		fn(mediaType, q)
	}
}

// parseQuality parses a quality value string, returning an error if invalid.
func parseQuality(s string) (float64, error) {
	q, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	if q < 0 || q > 1 {
		return 0, strconv.ErrRange
	}
	return q, nil
}
