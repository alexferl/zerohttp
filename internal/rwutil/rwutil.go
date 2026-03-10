package rwutil

import "net/http"

// ResponseWriter wraps http.ResponseWriter to capture status code and
// track whether the header has been written. This is a reusable component
// used by multiple middlewares instead of duplicating the same code.
type ResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

// NewResponseWriter creates a new ResponseWriter with default status code 200.
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code and forwards to the underlying ResponseWriter.
// It ensures the header is only written once.
func (rw *ResponseWriter) WriteHeader(code int) {
	if rw.headerWritten {
		return // Prevent multiple WriteHeader calls
	}
	rw.statusCode = code
	rw.headerWritten = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write writes the data to the connection, writing the header first if needed.
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// StatusCode returns the captured status code.
// If WriteHeader was never called, it returns the default 200.
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// HeaderWritten returns true if WriteHeader has been called.
func (rw *ResponseWriter) HeaderWritten() bool {
	return rw.headerWritten
}

// FlusherResponseWriter wraps ResponseWriter and implements http.Flusher.
// Use this when the underlying ResponseWriter may support flushing.
type FlusherResponseWriter struct {
	*ResponseWriter
	flusher http.Flusher
}

// NewFlusherResponseWriter creates a new FlusherResponseWriter.
// It checks if the underlying writer implements http.Flusher.
func NewFlusherResponseWriter(w http.ResponseWriter) *FlusherResponseWriter {
	rw := NewResponseWriter(w)
	var flusher http.Flusher
	if f, ok := w.(http.Flusher); ok {
		flusher = f
	}
	return &FlusherResponseWriter{
		ResponseWriter: rw,
		flusher:        flusher,
	}
}

// Flush implements http.Flusher. If the underlying ResponseWriter does not
// support flushing, this is a no-op.
func (frw *FlusherResponseWriter) Flush() {
	if frw.flusher != nil {
		frw.flusher.Flush()
	}
}

// Ensure interface compliance at compile time.
var (
	_ http.ResponseWriter = (*ResponseWriter)(nil)
	_ http.Flusher        = (*FlusherResponseWriter)(nil)
)
