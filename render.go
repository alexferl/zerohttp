package zerohttp

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alexferl/zerohttp/httpx"
)

// M is a convenience type for map[string]any, useful for quick JSON responses.
//
//	return zh.Render.JSON(w, http.StatusOK, zh.M{
//	    "message": "Hello, World!",
//	    "count":   42,
//	})
type M map[string]any

// Render is the default [Renderer] instance used by the package.
// Use it to write HTTP responses in various formats:
//
//	// JSON response
//	zh.Render.JSON(w, http.StatusOK, user)
//
//	// Plain text
//	zh.Render.Text(w, http.StatusOK, "Hello")
//
//	// File download
//	zh.Render.File(w, r, "/path/to/document.pdf")
//
// For convenience, use the [R] alias.
var Render Renderer = &defaultRenderer{}

// R is a short alias for [Render].
//
//	return zh.R.JSON(w, http.StatusOK, data)
var R = Render

// Renderer handles response rendering for various content types
type Renderer interface {
	// JSON writes a JSON response with the given status code and data
	JSON(w http.ResponseWriter, statusCode int, data any) error

	// Text writes a plain text response with the given status code and data
	Text(w http.ResponseWriter, statusCode int, data string) error

	// HTML writes an HTML response with the given status code and data
	HTML(w http.ResponseWriter, statusCode int, data string) error

	// Template renders an HTML template with proper Content-Type header
	Template(w http.ResponseWriter, code int, tmpl *template.Template, name string, data any) error

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

// Ensure defaultRenderer implements Renderer
var _ Renderer = (*defaultRenderer)(nil)

// defaultRenderer implements the Renderer interface with standard HTTP response handling
type defaultRenderer struct{}

// JSON writes a JSON response with the given status code and data
func (r *defaultRenderer) JSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSONCharset)
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// Text writes a plain text response with the given status code and data
func (r *defaultRenderer) Text(w http.ResponseWriter, statusCode int, data string) error {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(data))
	return err
}

// HTML writes an HTML response with the given status code and data
func (r *defaultRenderer) HTML(w http.ResponseWriter, statusCode int, data string) error {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTMLCharset)
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(data))
	return err
}

// Template writes an HTML response with the given status code, rendered from the specified template and data
func (r *defaultRenderer) Template(w http.ResponseWriter, code int, tmpl *template.Template, name string, data any) error {
	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTMLCharset)
	w.WriteHeader(code)
	return tmpl.ExecuteTemplate(w, name, data)
}

// Blob writes a blob response with the given status code, content type, and data
func (r *defaultRenderer) Blob(w http.ResponseWriter, statusCode int, contentType string, data []byte) error {
	w.Header().Set(httpx.HeaderContentType, contentType)
	w.WriteHeader(statusCode)
	_, err := w.Write(data)
	return err
}

// Stream writes a streaming response with the given status code and content type,
// copying data from the provided reader to the response writer
func (r *defaultRenderer) Stream(w http.ResponseWriter, statusCode int, contentType string, reader io.Reader) error {
	w.Header().Set(httpx.HeaderContentType, contentType)
	w.WriteHeader(statusCode)
	_, err := io.Copy(w, reader)
	return err
}

// File sends the contents of a file as the response.
// It automatically sets the Content-Type header based on the file extension
// and handles file opening/closing. Also sets ETag and Content-Length headers.
func (r *defaultRenderer) File(w http.ResponseWriter, req *http.Request, filename string) (err error) {
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
	// Use charset-aware constants for known types
	if contentType == "application/json" {
		contentType = httpx.MIMEApplicationJSONCharset
	}
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
	w.Header().Set(httpx.HeaderETag, etag)
	w.Header().Set(httpx.HeaderContentType, contentType)
	w.Header().Set(httpx.HeaderContentLength, strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, req, filepath.Base(filename), fileInfo.ModTime(), file)
	return
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
	w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationProblemJSON)
	w.WriteHeader(problem.Status)
	return json.NewEncoder(w).Encode(problem)
}
