package zerohttp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// Re-export SSE types from config package for convenience.

// SSEConnection is an alias for config.SSEConnection.
type SSEConnection = config.SSEConnection

// SSEProvider is an alias for config.SSEProvider.
type SSEProvider = config.SSEProvider

// SSEEvent is an alias for config.SSEEvent.
type SSEEvent = config.SSEEvent

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

// NewSSE creates a new SSE connection using stdlib.
// This sets the appropriate headers and prepares the connection for streaming.
func NewSSE(w http.ResponseWriter, r *http.Request) (*SSE, error) {
	if w.Header().Get("Content-Type") != "" {
		return nil, fmt.Errorf("sse: response headers already sent")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("sse: streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

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
func (s *SSE) Send(event SSEEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return fmt.Errorf("sse: connection closed")
	case <-s.ctx.Done():
		return fmt.Errorf("sse: context cancelled")
	default:
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
		lines := strings.Split(string(event.Data), "\n")
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

	_, err := s.w.Write([]byte(buf.String()))
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
	lines := strings.Split(comment, "\n")
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(": ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	_, err := s.w.Write([]byte(buf.String()))
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
		if s.cancel != nil {
			s.cancel()
		}
		return nil
	}
}

// WaitDone blocks until the monitor goroutine exits.
// This is primarily used for testing to verify goroutine cleanup.
func (s *SSE) WaitDone() {
	<-s.done
}

// isOpen returns true if the connection is still open (not closed).
func (s *SSE) isOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.closed:
		return false
	default:
		return true
	}
}

// SetRetry sets the default reconnection time for this connection.
func (s *SSE) SetRetry(d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retry = d
	return nil
}

// DefaultProvider implements SSEProvider using the stdlib
type DefaultProvider struct{}

// NewDefaultProvider creates a new stdlib-based SSE provider.
func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{}
}

// NewSSE creates a new SSE connection using the stdlib implementation.
func (p *DefaultProvider) NewSSE(w http.ResponseWriter, r *http.Request) (SSEConnection, error) {
	return NewSSE(w, r)
}

// SSEWriter wraps an http.ResponseWriter to provide SSE capabilities.
// This is a lower-level helper for users who want to write SSE directly.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
}

// NewSSEWriter creates a new SSEWriter from an http.ResponseWriter.
// This sets SSE headers and prepares the connection.
func NewSSEWriter(w http.ResponseWriter, r *http.Request) (*SSEWriter, error) {
	if w.Header().Get("Content-Type") != "" {
		return nil, fmt.Errorf("sse: response headers already sent")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("sse: streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	return &SSEWriter{
		w:       w,
		flusher: flusher,
	}, nil
}

// WriteEvent writes an SSE event.
func (s *SSEWriter) WriteEvent(event SSEEvent) error {
	return s.writeEvent(event)
}

// WriteComment writes an SSE comment.
func (s *SSEWriter) WriteComment(comment string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lines := strings.Split(comment, "\n")
	var buf strings.Builder
	for _, line := range lines {
		buf.WriteString(": ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	_, err := s.w.Write([]byte(buf.String()))
	if err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// Flush flushes the underlying writer.
func (s *SSEWriter) Flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flusher.Flush()
}

func (s *SSEWriter) writeEvent(event SSEEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		lines := strings.Split(string(event.Data), "\n")
		for _, line := range lines {
			buf.WriteString("data: ")
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	} else {
		buf.WriteString("data: \n")
	}

	buf.WriteByte('\n')

	_, err := s.w.Write([]byte(buf.String()))
	if err != nil {
		return err
	}
	s.flusher.Flush()
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

// SSEReplayer defines the interface for event replay storage.
// Implementations can use in-memory storage, Redis, databases, etc.
type SSEReplayer interface {
	// Store saves an event to the replay buffer and returns the event with assigned ID.
	Store(event SSEEvent) SSEEvent
	// Replay sends all events after the given ID to the provided send function.
	// Returns the number of events replayed and any error.
	Replay(afterID string, send func(SSEEvent) error) (int, error)
}

// InMemoryReplayer stores events in memory with a circular buffer.
// Events can be limited by max count and/or TTL.
// Ensure InMemoryReplayer implements SSEReplayer
var _ SSEReplayer = (*InMemoryReplayer)(nil)

type InMemoryReplayer struct {
	events    []storedEvent
	maxEvents int
	ttl       time.Duration
	mu        sync.RWMutex
	lastID    int64
}

type storedEvent struct {
	id        int64
	event     SSEEvent
	timestamp time.Time
}

// NewInMemoryReplayer creates a new in-memory event replayer.
// maxEvents is the maximum number of events to keep (0 = unlimited).
// ttl is how long to keep events (0 = no expiration).
func NewInMemoryReplayer(maxEvents int, ttl time.Duration) *InMemoryReplayer {
	if maxEvents < 0 {
		maxEvents = 0
	}
	return &InMemoryReplayer{
		events:    make([]storedEvent, 0),
		maxEvents: maxEvents,
		ttl:       ttl,
	}
}

// Store saves an event to the replay buffer with an auto-generated ID.
// Returns the event with the assigned ID so it can be used for broadcasting.
func (r *InMemoryReplayer) Store(event SSEEvent) SSEEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ttl > 0 {
		now := time.Now()
		valid := make([]storedEvent, 0, len(r.events))
		for _, e := range r.events {
			if now.Sub(e.timestamp) < r.ttl {
				valid = append(valid, e)
			}
		}
		r.events = valid
	}

	r.lastID++
	event.ID = strconv.FormatInt(r.lastID, 10)

	r.events = append(r.events, storedEvent{
		id:        r.lastID,
		event:     event,
		timestamp: time.Now(),
	})

	if r.maxEvents > 0 && len(r.events) > r.maxEvents {
		r.events = r.events[len(r.events)-r.maxEvents:]
	}

	return event
}

// Replay sends all events after the given ID to the provided send function.
func (r *InMemoryReplayer) Replay(afterID string, send func(SSEEvent) error) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	startID := int64(0)
	if afterID != "" {
		var err error
		startID, err = strconv.ParseInt(afterID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("sse: invalid Last-Event-ID: %w", err)
		}
	}

	count := 0
	for _, se := range r.events {
		if se.id > startID {
			if err := send(se.event); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// SSEWithReplay creates a new SSE connection and replays missed events if Last-Event-ID header is present.
// After replay completes, the connection is ready for new events.
func SSEWithReplay(w http.ResponseWriter, r *http.Request, replayer SSEReplayer) (*SSE, error) {
	stream, err := NewSSE(w, r)
	if err != nil {
		return nil, err
	}

	// Check for Last-Event-ID header for replay
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" && replayer != nil {
		_, err := replayer.Replay(lastEventID, func(event SSEEvent) error {
			return stream.Send(event)
		})
		if err != nil {
			_ = stream.Close()
			return nil, fmt.Errorf("sse: replay failed: %w", err)
		}
	}

	return stream, nil
}

// SSEHub manages multiple SSE connections for broadcasting.
type SSEHub struct {
	connections map[*SSE]struct{}
	topics      map[string]map[*SSE]struct{}
	mu          sync.RWMutex
}

// NewSSEHub creates a new SSE broadcast hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		connections: make(map[*SSE]struct{}),
		topics:      make(map[string]map[*SSE]struct{}),
	}
}

// Register adds an SSE connection to the hub.
func (h *SSEHub) Register(s *SSE) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[s] = struct{}{}
}

// Unregister removes an SSE connection from the hub.
func (h *SSEHub) Unregister(s *SSE) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, s)
	for topic, subs := range h.topics {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// Subscribe adds an SSE connection to a topic.
func (h *SSEHub) Subscribe(s *SSE, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*SSE]struct{})
	}
	h.topics[topic][s] = struct{}{}
}

// Unsubscribe removes an SSE connection from a topic.
func (h *SSEHub) Unsubscribe(s *SSE, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.topics[topic]; ok {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// Broadcast sends an event to all registered connections.
// Connections that fail to receive the event are automatically unregistered.
func (h *SSEHub) Broadcast(event SSEEvent) {
	h.mu.RLock()
	connections := make([]*SSE, 0, len(h.connections))
	for conn := range h.connections {
		connections = append(connections, conn)
	}
	h.mu.RUnlock()

	var failed []*SSE
	for _, conn := range connections {
		// Check if connection is still registered and not closed before sending
		if h.isConnectionActive(conn) {
			if err := conn.Send(event); err != nil {
				failed = append(failed, conn)
			}
		} else {
			failed = append(failed, conn)
		}
	}

	// Unregister failed connections
	for _, conn := range failed {
		h.Unregister(conn)
		_ = conn.Close()
	}
}

// isConnectionActive checks if a connection is still open (not closed).
func (h *SSEHub) isConnectionActive(conn *SSE) bool {
	return conn.isOpen()
}

// BroadcastTo sends an event to all connections subscribed to a topic.
// Connections that fail to receive the event are automatically unregistered.
func (h *SSEHub) BroadcastTo(topic string, event SSEEvent) {
	h.mu.RLock()
	var connections []*SSE
	if subs, ok := h.topics[topic]; ok {
		connections = make([]*SSE, 0, len(subs))
		for conn := range subs {
			connections = append(connections, conn)
		}
	}
	h.mu.RUnlock()

	var failed []*SSE
	for _, conn := range connections {
		// Check if connection is still registered and not closed before sending
		if h.isConnectionActive(conn) {
			if err := conn.Send(event); err != nil {
				failed = append(failed, conn)
			}
		} else {
			failed = append(failed, conn)
		}
	}

	// Unregister failed connections
	for _, conn := range failed {
		h.Unregister(conn)
		_ = conn.Close()
	}
}

// ConnectionCount returns the number of registered connections.
func (h *SSEHub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}

// TopicCount returns the number of connections subscribed to a topic.
func (h *SSEHub) TopicCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.topics[topic])
}

var (
	_ SSEConnection = (*SSE)(nil)
	_ SSEProvider   = (*DefaultProvider)(nil)
	_ SSEReplayer   = (*InMemoryReplayer)(nil)
)
