package zerohttp

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// M is a convenience type for map[string]any, useful for quick JSON responses
type M map[string]any

// Render is the default renderer instance used by the package
var Render Renderer = &defaultRenderer{}

// R is a short alias for Render for convenience
var R = Render

// Renderer handles response rendering for various content types
type Renderer interface {
	// JSON writes a JSON response with the given status code and data
	JSON(w http.ResponseWriter, statusCode int, data any) error

	// Text writes a plain text response with the given status code and data
	Text(w http.ResponseWriter, statusCode int, data string) error

	// HTML writes an HTML response with the given status code and data
	HTML(w http.ResponseWriter, statusCode int, data string) error

	// Blob writes a binary response with the given status code, content type, and data
	Blob(w http.ResponseWriter, statusCode int, contentType string, data []byte) error

	// Stream writes a streaming response with the given status code and content type,
	// copying data from the provided reader to the response writer
	Stream(w http.ResponseWriter, statusCode int, contentType string, reader io.Reader) error

	// File serves a file as the response, automatically setting appropriate headers
	File(w http.ResponseWriter, r *http.Request, filename string) error

	// NoContent writes a 204 No Content response with no body
	NoContent(w http.ResponseWriter) error

	// NotModified writes a 304 Not Modified response for conditional requests
	NotModified(w http.ResponseWriter) error

	// Redirect performs an HTTP redirect with the specified status code and location
	Redirect(w http.ResponseWriter, r *http.Request, url string, code int) error

	// ProblemDetail writes an RFC 9457 Problem Details response
	ProblemDetail(w http.ResponseWriter, problem *ProblemDetail) error
}

// defaultRenderer implements the Renderer interface with standard HTTP response handling
type defaultRenderer struct{}

// JSON writes a JSON response with the given status code and data
func (r *defaultRenderer) JSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set(HeaderContentType, MIMEApplicationJSON)
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// Text writes a plain text response with the given status code and data
func (r *defaultRenderer) Text(w http.ResponseWriter, statusCode int, data string) error {
	w.Header().Set(HeaderContentType, MIMETextPlain)
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(data))
	return err
}

// HTML writes an HTML response with the given status code and data
func (r *defaultRenderer) HTML(w http.ResponseWriter, statusCode int, data string) error {
	w.Header().Set(HeaderContentType, MIMETextHTML)
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(data))
	return err
}

// Blob writes a blob response with the given status code, content type, and data
func (r *defaultRenderer) Blob(w http.ResponseWriter, statusCode int, contentType string, data []byte) error {
	w.Header().Set(HeaderContentType, contentType)
	w.WriteHeader(statusCode)
	_, err := w.Write(data)
	return err
}

// Stream writes a streaming response with the given status code and content type,
// copying data from the provided reader to the response writer
func (r *defaultRenderer) Stream(w http.ResponseWriter, statusCode int, contentType string, reader io.Reader) error {
	w.Header().Set(HeaderContentType, contentType)
	w.WriteHeader(statusCode)
	_, err := io.Copy(w, reader)
	return err
}

// File sends the contents of a file as the response.
// It automatically sets the Content-Type header based on the file extension
// and handles file opening/closing. Also sets ETag and Content-Length headers.
func (r *defaultRenderer) File(w http.ResponseWriter, req *http.Request, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		cErr := file.Close()
		if cErr != nil {
			// If there was no previous error, return the close error
			if err == nil {
				err = cErr
			}
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		contentType = http.DetectContentType(buffer[:n])

		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}

	etag := fmt.Sprintf(`"%x-%x"`, fileInfo.ModTime().Unix(), fileInfo.Size())
	w.Header().Set(HeaderETag, etag)
	w.Header().Set(HeaderContentType, contentType)
	w.Header().Set(HeaderContentLength, strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, req, filepath.Base(filename), fileInfo.ModTime(), file)
	return err
}

// NoContent writes a 204 No Content response with no body
func (r *defaultRenderer) NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// NotModified writes a 304 Not Modified response for conditional requests
func (r *defaultRenderer) NotModified(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNotModified)
	return nil
}

// Redirect performs an HTTP redirect with the specified status code and location
func (r *defaultRenderer) Redirect(w http.ResponseWriter, req *http.Request, url string, code int) error {
	http.Redirect(w, req, url, code)
	return nil
}

// ProblemDetail writes an RFC 9457 Problem Details response
func (r *defaultRenderer) ProblemDetail(w http.ResponseWriter, problem *ProblemDetail) error {
	w.Header().Set(HeaderContentType, MIMEApplicationProblem)
	w.WriteHeader(problem.Status)
	return json.NewEncoder(w).Encode(problem)
}

// ProblemDetail represents an RFC 9457 Problem Details response.
// It provides a standardized way to carry machine-readable details of errors
// in HTTP response bodies.
type ProblemDetail struct {
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

// NewProblemDetail creates a new ProblemDetail with the given status code and detail message.
// The title is automatically set based on the HTTP status code.
func NewProblemDetail(statusCode int, detail string) *ProblemDetail {
	return &ProblemDetail{
		Title:      http.StatusText(statusCode),
		Status:     statusCode,
		Detail:     detail,
		Extensions: make(map[string]any),
	}
}

// MarshalJSON implements custom JSON marshaling to include extensions as top-level fields
func (p *ProblemDetail) MarshalJSON() ([]byte, error) {
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

// Set adds an extension field to the problem detail and returns the ProblemDetail
// for method chaining. Extension fields are included as top-level JSON properties.
func (p *ProblemDetail) Set(key string, value any) *ProblemDetail {
	if p.Extensions == nil {
		p.Extensions = make(map[string]any)
	}
	p.Extensions[key] = value
	return p
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

// NewValidationProblemDetail creates a problem detail for validation errors (HTTP 422).
// It accepts any slice type for errors, allowing custom validation error structures.
// The errors are added as an "errors" extension field.
func NewValidationProblemDetail[T any](detail string, errors []T) *ProblemDetail {
	problem := NewProblemDetail(http.StatusUnprocessableEntity, detail)
	problem.Set("errors", errors)
	return problem
}
