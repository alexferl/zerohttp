package sse

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// Connection represents an active Server-Sent Events connection.
// Users can implement this interface with their own SSE library, or use
// the built-in EventStream implementation.
type Connection interface {
	// Send writes an event to the client.
	// Returns error if the connection is closed or write fails.
	Send(event Event) error

	// SendComment sends a comment (heartbeat/keepalive).
	// Comments are ignored by the client but keep connections alive through proxies.
	SendComment(comment string) error

	// Close signals the SSE connection is done.
	// No further events should be sent after Close.
	Close() error

	// SetRetry sets the default reconnection time for this connection.
	// Affects subsequent events without explicit Retry value.
	SetRetry(d time.Duration) error
}

// Event represents a single SSE event
type Event struct {
	ID    string
	Name  string
	Data  []byte
	Retry time.Duration
}

// Ensure SSE implements Connection
var _ Connection = (*SSE)(nil)

// SSE is the built-in SSE implementation using Go's standard library.
type SSE struct {
	w       http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
	cancel  context.CancelFunc
	closed  chan struct{}
	done    chan struct{} // Closed when monitor goroutine exits
	mu      sync.Mutex
	retry   time.Duration
}

// lineEndingReplacer normalizes CR, LF, and CRLF to LF for SSE spec compliance.
// The SSE spec allows lines to be terminated by CRLF, LF, or bare CR.
var lineEndingReplacer = strings.NewReplacer("\r\n", "\n", "\r", "\n")

func normalizeLineEndings(s string) string {
	return lineEndingReplacer.Replace(s)
}

// setupResponse sets up the SSE headers and returns the flusher.
// This is a helper shared between New and NewWriter.
//
// Note: The Content-Type check is a heuristic. It may false-positive if
// middleware sets Content-Type before the SSE handler runs. Consider avoiding
// Content-Type middleware on SSE routes.
func setupResponse(w http.ResponseWriter) (http.Flusher, error) {
	if w.Header().Get(httpx.HeaderContentType) != "" {
		return nil, fmt.Errorf("sse: response headers already sent (Content-Type: %s)", w.Header().Get(httpx.HeaderContentType))
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("sse: streaming not supported (ResponseWriter type: %T)", w)
	}

	w.Header().Set(httpx.HeaderContentType, httpx.MIMETextEventStream)
	w.Header().Set(httpx.HeaderCacheControl, httpx.CacheControlNoCache)
	w.Header().Set(httpx.HeaderConnection, httpx.ConnectionKeepAlive)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return flusher, nil
}

// New creates a new SSE connection using stdlib.
// This sets the appropriate headers and prepares the connection for streaming.
func New(w http.ResponseWriter, r *http.Request) (*SSE, error) {
	flusher, err := setupResponse(w)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(r.Context())

	stream := &SSE{
		w:       w,
		flusher: flusher,
		ctx:     ctx,
		cancel:  cancel,
		closed:  make(chan struct{}),
		done:    make(chan struct{}),
	}

	// Monitor context cancellation
	go func() {
		defer close(stream.done) // Signal goroutine exit
		select {
		case <-ctx.Done():
			_ = stream.Close()
		case <-stream.closed:
		}
	}()

	return stream, nil
}

// Send writes an event to the client.
func (s *SSE) Send(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return fmt.Errorf("sse: connection closed")
	case <-s.ctx.Done():
		return fmt.Errorf("sse: context cancelled")
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

	retry := event.Retry
	if retry == 0 {
		retry = s.retry
	}
	if retry > 0 {
		buf.WriteString("retry: ")
		buf.WriteString(strconv.FormatInt(retry.Milliseconds(), 10))
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

	// Empty line terminates the event
	buf.WriteByte('\n')

	_, err := io.WriteString(s.w, buf.String())
	if err != nil {
		return fmt.Errorf("sse: write error: %w", err)
	}

	s.flusher.Flush()
	return nil
}

// SendComment sends a comment (heartbeat/keepalive).
func (s *SSE) SendComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return fmt.Errorf("sse: connection closed")
	case <-s.ctx.Done():
		return fmt.Errorf("sse: context cancelled")
	default:
	}

	// Comments start with colon
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

// Close signals the SSE connection is done.
func (s *SSE) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
		s.cancel()
		return nil
	}
}

// WaitDone blocks until the monitor goroutine exits.
// This is primarily used for testing to verify goroutine cleanup.
func (s *SSE) WaitDone() {
	<-s.done
}

// SetRetry sets the default reconnection time for this connection.
func (s *SSE) SetRetry(d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retry = d
	return nil
}

// IsClientDisconnected checks if the client has disconnected.
// This checks if the request context is done.
func IsClientDisconnected(r *http.Request) bool {
	select {
	case <-r.Context().Done():
		return true
	default:
		return false
	}
}
