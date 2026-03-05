package zerohttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewEventStream(t *testing.T) {
	t.Run("successfully creates SSE", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = stream.Close() }()

		// Check headers
		resp := w.Result()
		if resp.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
		}
		if resp.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("expected Cache-Control no-cache, got %s", resp.Header.Get("Cache-Control"))
		}
	})

	t.Run("returns error if headers already sent", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Set Content-Type header to simulate headers already sent
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := NewSSE(w, r)
		if err == nil {
			t.Error("expected error when headers already sent")
		}
	})
}

func TestEventStream_Send(t *testing.T) {
	t.Run("sends simple event", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{
			Data: []byte("hello world"),
		}

		err = stream.Send(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check body
		body := w.Body.String()
		if !strings.Contains(body, "data: hello world\n") {
			t.Errorf("expected event data in body, got %s", body)
		}
	})

	t.Run("sends event with all fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{
			ID:    "123",
			Name:  "update",
			Data:  []byte("test data"),
			Retry: 5000 * time.Millisecond,
		}

		err = stream.Send(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "id: 123\n") {
			t.Errorf("expected id in body, got %s", body)
		}
		if !strings.Contains(body, "event: update\n") {
			t.Errorf("expected event name in body, got %s", body)
		}
		if !strings.Contains(body, "retry: 5000\n") {
			t.Errorf("expected retry in body, got %s", body)
		}
		if !strings.Contains(body, "data: test data\n") {
			t.Errorf("expected data in body, got %s", body)
		}
	})

	t.Run("handles multi-line data", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{
			Data: []byte("line1\nline2\nline3"),
		}

		err = stream.Send(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		expected := "data: line1\ndata: line2\ndata: line3\n"
		if !strings.Contains(body, expected) {
			t.Errorf("expected multi-line data format, got %s", body)
		}
	})

	t.Run("returns error after close", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_ = stream.Close()

		event := SSEEvent{Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error after close")
		}
	})
}

func TestEventStream_SendComment(t *testing.T) {
	t.Run("sends comment", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = stream.SendComment("keepalive")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, ": keepalive\n") {
			t.Errorf("expected comment in body, got %s", body)
		}
	})
}

func TestEventStream_SetRetry(t *testing.T) {
	t.Run("sets default retry", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = stream.Close() }()

		err = stream.SetRetry(10 * time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Send event without retry - should use default
		event := SSEEvent{Data: []byte("test")}
		err = stream.Send(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "retry: 10000\n") {
			t.Errorf("expected default retry in body, got %s", body)
		}
	})
}

func TestDefaultProvider(t *testing.T) {
	provider := NewDefaultProvider()
	if provider == nil {
		t.Fatal("expected provider to not be nil")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sse", nil)

	conn, err := provider.NewSSE(w, r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer func() { _ = conn.Close() }()

	if conn == nil {
		t.Error("expected connection to not be nil")
	}
}

func TestIsClientDisconnected(t *testing.T) {
	t.Run("returns false for active request", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		if IsClientDisconnected(r) {
			t.Error("expected false for active request")
		}
	})

	t.Run("returns true for cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

		cancel()
		time.Sleep(10 * time.Millisecond) // Let cancellation propagate

		if !IsClientDisconnected(r) {
			t.Error("expected true for cancelled context")
		}
	})
}

func TestEventStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

	stream, err := NewSSE(w, r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond) // Let cancellation propagate

	// Send should fail after context cancellation
	event := SSEEvent{Data: []byte("test")}
	err = stream.Send(event)
	if err == nil {
		t.Error("expected error after context cancellation")
	}
}

// errorWriter is a ResponseWriter that fails on write
type errorWriter struct {
	http.ResponseWriter
	header http.Header
}

func (e *errorWriter) Header() http.Header {
	return e.header
}

func (e *errorWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write error")
}

func (e *errorWriter) WriteHeader(code int) {}

func TestEventStream_Send_WriteError(t *testing.T) {
	t.Run("returns error on write failure", func(t *testing.T) {
		header := make(http.Header)
		hew := &errorWriter{header: header}
		// Create a flusher that works
		f := &flusherWriter{ResponseWriter: hew, header: header}

		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream := &SSE{
			w:       f,
			flusher: f,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
		}

		event := SSEEvent{Data: []byte("test")}
		err := stream.Send(event)
		if err == nil {
			t.Error("expected error on write failure")
		}
	})
}

func TestEventStream_SendComment_WriteError(t *testing.T) {
	t.Run("returns error on write failure", func(t *testing.T) {
		header := make(http.Header)
		hew := &errorWriter{header: header}
		f := &flusherWriter{ResponseWriter: hew, header: header}

		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream := &SSE{
			w:       f,
			flusher: f,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
		}

		err := stream.SendComment("test")
		if err == nil {
			t.Error("expected error on write failure")
		}
	})
}

func TestEventStream_SendComment_ContextCancelled(t *testing.T) {
	t.Run("returns error when context cancelled", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		cancel()
		time.Sleep(10 * time.Millisecond)

		err = stream.SendComment("test")
		if err == nil {
			t.Error("expected error when context cancelled")
		}
	})
}

// flusherWriter wraps a ResponseWriter and adds Flush
type flusherWriter struct {
	http.ResponseWriter
	header http.Header
}

func (f *flusherWriter) Flush() {}

// SSEWriter tests

func TestNewSSEWriter(t *testing.T) {
	t.Run("successfully creates SSEWriter", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check headers
		resp := w.Result()
		if resp.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
		}

		_ = writer
	})

	t.Run("returns error if headers already sent", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, err := NewSSEWriter(w, r)
		if err == nil {
			t.Error("expected error when headers already sent")
		}
	})
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	t.Run("writes event with all fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{
			ID:    "456",
			Name:  "message",
			Data:  []byte("hello"),
			Retry: 3000 * time.Millisecond,
		}

		err = writer.WriteEvent(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, "id: 456") {
			t.Errorf("expected id in body, got %s", body)
		}
		if !strings.Contains(body, "event: message") {
			t.Errorf("expected event name in body, got %s", body)
		}
		if !strings.Contains(body, "retry: 3000") {
			t.Errorf("expected retry in body, got %s", body)
		}
		if !strings.Contains(body, "data: hello") {
			t.Errorf("expected data in body, got %s", body)
		}
	})
}

func TestSSEWriter_WriteComment(t *testing.T) {
	t.Run("writes comment", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		err = writer.WriteComment("keepalive")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		body := w.Body.String()
		if !strings.Contains(body, ": keepalive") {
			t.Errorf("expected comment in body, got %s", body)
		}
	})
}

func TestSSEWriter_Flush(t *testing.T) {
	t.Run("flushes without error", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Flush should not panic
		writer.Flush()
	})
}

// InMemoryReplayer tests

func TestInMemoryReplayer(t *testing.T) {
	t.Run("stores and replays events", func(t *testing.T) {
		replay := NewInMemoryReplayer(100, 0)

		// Store some events
		replay.Store(SSEEvent{Name: "test1", Data: []byte("data1")})
		replay.Store(SSEEvent{Name: "test2", Data: []byte("data2")})
		replay.Store(SSEEvent{Name: "test3", Data: []byte("data3")})

		// Replay from event 1
		var replayed []SSEEvent
		count, err := replay.Replay("1", func(e SSEEvent) error {
			replayed = append(replayed, e)
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 events replayed, got %d", count)
		}
		if len(replayed) != 2 {
			t.Errorf("expected 2 events in slice, got %d", len(replayed))
		}
	})

	t.Run("respects max events limit", func(t *testing.T) {
		replay := NewInMemoryReplayer(2, 0)

		replay.Store(SSEEvent{Data: []byte("data1")})
		replay.Store(SSEEvent{Data: []byte("data2")})
		replay.Store(SSEEvent{Data: []byte("data3")}) // Should evict data1

		count, _ := replay.Replay("", func(e SSEEvent) error {
			return nil
		})
		if count != 2 {
			t.Errorf("expected 2 events (max), got %d", count)
		}
	})

	t.Run("respects TTL", func(t *testing.T) {
		replay := NewInMemoryReplayer(100, 50*time.Millisecond)

		replay.Store(SSEEvent{Data: []byte("old")})
		time.Sleep(100 * time.Millisecond)
		replay.Store(SSEEvent{Data: []byte("new")})

		count, _ := replay.Replay("", func(e SSEEvent) error {
			return nil
		})
		if count != 1 {
			t.Errorf("expected 1 event (old expired), got %d", count)
		}
	})

	t.Run("auto-assigns IDs", func(t *testing.T) {
		replay := NewInMemoryReplayer(100, 0)

		event := SSEEvent{Data: []byte("test")}
		returnedEvent := replay.Store(event)

		// Check that returned event has ID assigned
		if returnedEvent.ID == "" {
			t.Error("expected returned event to have ID auto-assigned")
		}

		// Check that ID is accessible immediately without replay
		if returnedEvent.ID != "1" {
			t.Errorf("expected ID to be 1, got %s", returnedEvent.ID)
		}

		// Also verify it's stored correctly
		var replayed SSEEvent
		_, _ = replay.Replay("", func(e SSEEvent) error {
			replayed = e
			return nil
		})
		if replayed.ID == "" {
			t.Error("expected ID to be auto-assigned in storage")
		}
		if replayed.ID != returnedEvent.ID {
			t.Errorf("replay ID %s doesn't match returned ID %s", replayed.ID, returnedEvent.ID)
		}
	})
}

func TestSSEWithReplay(t *testing.T) {
	t.Run("creates stream without replay when no header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		replay := NewInMemoryReplayer(100, 0)

		stream, err := SSEWithReplay(w, r, replay)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = stream.Close() }()

		// Should not replay anything
	})

	t.Run("replays events when Last-Event-ID present", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set("Last-Event-ID", "0")
		replay := NewInMemoryReplayer(100, 0)

		replay.Store(SSEEvent{Data: []byte("event1")})
		replay.Store(SSEEvent{Data: []byte("event2")})

		stream, err := SSEWithReplay(w, r, replay)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = stream.Close() }()

		body := w.Body.String()
		if !strings.Contains(body, "event1") || !strings.Contains(body, "event2") {
			t.Errorf("expected replayed events in body, got %s", body)
		}
	})

	t.Run("returns error for invalid Last-Event-ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set("Last-Event-ID", "not-a-number")
		replay := NewInMemoryReplayer(100, 0)

		_, err := SSEWithReplay(w, r, replay)
		if err == nil {
			t.Error("expected error for invalid Last-Event-ID")
		}
	})
}

// SSEHub tests

func TestSSEHub(t *testing.T) {
	t.Run("registers and unregisters connections", func(t *testing.T) {
		hub := NewSSEHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		hub.Register(stream)

		if hub.ConnectionCount() != 1 {
			t.Errorf("expected 1 connection, got %d", hub.ConnectionCount())
		}

		hub.Unregister(stream)
		if hub.ConnectionCount() != 0 {
			t.Errorf("expected 0 connections, got %d", hub.ConnectionCount())
		}
	})

	t.Run("subscribes to topics", func(t *testing.T) {
		hub := NewSSEHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		hub.Subscribe(stream, "notifications")

		if hub.TopicCount("notifications") != 1 {
			t.Errorf("expected 1 subscriber, got %d", hub.TopicCount("notifications"))
		}

		hub.Unsubscribe(stream, "notifications")
		if hub.TopicCount("notifications") != 0 {
			t.Errorf("expected 0 subscribers, got %d", hub.TopicCount("notifications"))
		}
	})

	t.Run("broadcasts to all connections", func(t *testing.T) {
		hub := NewSSEHub()

		// Create two streams
		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream1, _ := NewSSE(w1, r)
		stream2, _ := NewSSE(w2, r)

		hub.Register(stream1)
		hub.Register(stream2)

		hub.Broadcast(SSEEvent{Data: []byte("hello all")})

		if !strings.Contains(w1.Body.String(), "hello all") {
			t.Error("expected stream1 to receive broadcast")
		}
		if !strings.Contains(w2.Body.String(), "hello all") {
			t.Error("expected stream2 to receive broadcast")
		}
	})

	t.Run("broadcasts to topic subscribers only", func(t *testing.T) {
		hub := NewSSEHub()

		w1 := httptest.NewRecorder()
		w2 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream1, _ := NewSSE(w1, r)
		stream2, _ := NewSSE(w2, r)

		hub.Subscribe(stream1, "topic1")
		hub.Subscribe(stream2, "topic2")

		hub.BroadcastTo("topic1", SSEEvent{Data: []byte("topic1 message")})

		if !strings.Contains(w1.Body.String(), "topic1 message") {
			t.Error("expected stream1 to receive topic1 message")
		}
		if strings.Contains(w2.Body.String(), "topic1 message") {
			t.Error("expected stream2 NOT to receive topic1 message")
		}
	})

	t.Run("unsubscribe removes from all topics", func(t *testing.T) {
		hub := NewSSEHub()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, _ := NewSSE(w, r)
		hub.Subscribe(stream, "topic1")
		hub.Subscribe(stream, "topic2")

		hub.Unregister(stream)

		if hub.TopicCount("topic1") != 0 {
			t.Error("expected stream to be removed from topic1")
		}
		if hub.TopicCount("topic2") != 0 {
			t.Error("expected stream to be removed from topic2")
		}
	})
}
