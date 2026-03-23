package sse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Writer wraps an http.ResponseWriter to provide SSE capabilities.
// This is a lower-level helper for users who want to write SSE directly.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
	mu      sync.Mutex
}

// NewWriter creates a new Writer from an http.ResponseWriter.
// This sets SSE headers and prepares the connection.
func NewWriter(w http.ResponseWriter, r *http.Request) (*Writer, error) {
	flusher, err := setupResponse(w)
	if err != nil {
		return nil, err
	}

	return &Writer{
		w:       w,
		flusher: flusher,
		ctx:     r.Context(),
	}, nil
}

// WriteEvent writes an SSE event.
func (s *Writer) WriteEvent(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.ctx.Done():
		return fmt.Errorf("sse: %w", s.ctx.Err())
	default:
	}

	// Validate ID and Name per SSE spec: must not contain CR, LF, or NULL
	if strings.ContainsAny(event.ID, "\r\n\x00") {
		return fmt.Errorf("sse: event ID must not contain CR, LF, or NULL")
	}
	if strings.ContainsAny(event.Name, "\r\n") {
		return fmt.Errorf("sse: event name must not contain CR or LF")
	}

	var buf strings.Builder

	if event.ID != "" {
		buf.WriteString("id: ")
		buf.WriteString(event.ID)
		buf.WriteByte('\n')
	}

	if event.Name != "" {
		buf.WriteString("event: ")
		buf.WriteString(event.Name)
		buf.WriteByte('\n')
	}

	if event.Retry > 0 {
		buf.WriteString("retry: ")
		buf.WriteString(strconv.FormatInt(event.Retry.Milliseconds(), 10))
		buf.WriteByte('\n')
	}

	if len(event.Data) > 0 {
		lines := strings.Split(normalizeLineEndings(string(event.Data)), "\n")
		for _, line := range lines {
			buf.WriteString("data: ")
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	} else {
		buf.WriteString("data: \n")
	}

	buf.WriteByte('\n')

	_, err := io.WriteString(s.w, buf.String())
	if err != nil {
		return fmt.Errorf("sse: write error: %w", err)
	}
	s.flusher.Flush()
	return nil
}

// WriteComment writes an SSE comment.
func (s *Writer) WriteComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.ctx.Done():
		return fmt.Errorf("sse: %w", s.ctx.Err())
	default:
	}

	lines := strings.Split(normalizeLineEndings(comment), "\n")
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(": ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	_, err := io.WriteString(s.w, buf.String())
	if err != nil {
		return fmt.Errorf("sse: write error: %w", err)
	}
	s.flusher.Flush()
	return nil
}

// Flush flushes the underlying writer.
func (s *Writer) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flusher.Flush()
}
