package zerohttp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
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
		zhtest.AssertWith(t, w).
			Header(httpx.HeaderContentType, httpx.MIMETextEventStream).
			Header(httpx.HeaderCacheControl, "no-cache")
	})

	t.Run("returns error if headers already sent", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		// Set Content-Type header to simulate headers already sent
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
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
		zhtest.AssertWith(t, w).BodyContains("data: hello world\n")
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

		zhtest.AssertWith(t, w).
			BodyContains("id: 123\n").
			BodyContains("event: update\n").
			BodyContains("retry: 5000\n").
			BodyContains("data: test data\n")
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

		zhtest.AssertWith(t, w).BodyContains("data: line1\ndata: line2\ndata: line3\n")
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

	t.Run("rejects event ID with CR", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{ID: "abc\rdef", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with CR")
		}
	})

	t.Run("rejects event ID with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{ID: "abc\ndef", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with LF")
		}
	})

	t.Run("rejects event ID with NULL", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{ID: "abc\x00def", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with NULL")
		}
	})

	t.Run("rejects event name with CR", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{Name: "update\rnotify", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for Name with CR")
		}
	})

	t.Run("rejects event name with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{Name: "update\nnotify", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for Name with LF")
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

		zhtest.AssertWith(t, w).BodyContains(": keepalive\n")
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

		zhtest.AssertWith(t, w).BodyContains("retry: 10000\n")
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
			cancel:  func() {},
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
			cancel:  func() {},
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
		zhtest.AssertWith(t, w).Header(httpx.HeaderContentType, httpx.MIMETextEventStream)

		_ = writer
	})

	t.Run("returns error if headers already sent", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlainCharset)
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

		zhtest.AssertWith(t, w).
			BodyContains("id: 456").
			BodyContains("event: message").
			BodyContains("retry: 3000").
			BodyContains("data: hello")
	})

	t.Run("rejects event ID with CRLF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{ID: "abc\r\ndef", Data: []byte("test")}
		err = writer.WriteEvent(event)
		if err == nil {
			t.Error("expected error for ID with CRLF")
		}
	})

	t.Run("rejects event name with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := SSEEvent{Name: "update\nnotify", Data: []byte("test")}
		err = writer.WriteEvent(event)
		if err == nil {
			t.Error("expected error for Name with LF")
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

		zhtest.AssertWith(t, w).BodyContains(": keepalive")
	})

	t.Run("returns error when context cancelled", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

		writer, err := NewSSEWriter(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		cancel()
		time.Sleep(10 * time.Millisecond)

		err = writer.WriteComment("test")
		if err == nil {
			t.Error("expected error when context cancelled")
		}
		if !strings.Contains(err.Error(), "sse:") {
			t.Errorf("expected error to contain 'sse:' prefix, got: %v", err)
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
		r.Header.Set(httpx.HeaderLastEventId, "0")
		replay := NewInMemoryReplayer(100, 0)

		replay.Store(SSEEvent{Data: []byte("event1")})
		replay.Store(SSEEvent{Data: []byte("event2")})

		stream, err := SSEWithReplay(w, r, replay)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = stream.Close() }()

		zhtest.AssertWith(t, w).
			BodyContains("event1").
			BodyContains("event2")
	})

	t.Run("returns error for invalid Last-Event-ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "not-a-number")
		replay := NewInMemoryReplayer(100, 0)

		_, err := SSEWithReplay(w, r, replay)
		if err == nil {
			t.Error("expected error for invalid Last-Event-ID")
		}
	})

	t.Run("returns error when Last-Event-ID present but replayer is nil", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		r.Header.Set(httpx.HeaderLastEventId, "1")

		_, err := SSEWithReplay(w, r, nil)
		if err == nil {
			t.Error("expected error when Last-Event-ID present but replayer is nil")
		}
		if !strings.Contains(err.Error(), "no replayer configured") {
			t.Errorf("expected error to contain 'no replayer configured', got: %v", err)
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

		zhtest.AssertWith(t, w1).BodyContains("hello all")
		zhtest.AssertWith(t, w2).BodyContains("hello all")
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

		zhtest.AssertWith(t, w1).BodyContains("topic1 message")
		zhtest.AssertWith(t, w2).BodyNotContains("topic1 message")
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

	t.Run("auto-unregisters failed connections on broadcast", func(t *testing.T) {
		hub := NewSSEHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := NewSSE(w1, r)
		hub.Register(stream1)

		// Create a stream with an error writer that will fail on send
		header := make(http.Header)
		ew := &errorWriter{header: header}
		fw := &flusherWriter{ResponseWriter: ew, header: header}
		badStream := &SSE{
			w:       fw,
			flusher: fw,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
			done:    make(chan struct{}),
			cancel:  func() {},
		}
		hub.Register(badStream)

		if hub.ConnectionCount() != 2 {
			t.Errorf("expected 2 connections, got %d", hub.ConnectionCount())
		}

		// Broadcast should auto-unregister the failed connection
		hub.Broadcast(SSEEvent{Data: []byte("test")})

		if hub.ConnectionCount() != 1 {
			t.Errorf("expected 1 connection after broadcast, got %d", hub.ConnectionCount())
		}
	})

	t.Run("auto-unregisters failed connections on broadcast to topic", func(t *testing.T) {
		hub := NewSSEHub()

		// Create a working stream
		w1 := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)
		stream1, _ := NewSSE(w1, r)
		hub.Subscribe(stream1, "test-topic")

		// Create a stream with an error writer
		header := make(http.Header)
		ew := &errorWriter{header: header}
		fw := &flusherWriter{ResponseWriter: ew, header: header}
		badStream := &SSE{
			w:       fw,
			flusher: fw,
			ctx:     r.Context(),
			closed:  make(chan struct{}),
			done:    make(chan struct{}),
			cancel:  func() {},
		}
		hub.Subscribe(badStream, "test-topic")

		if hub.TopicCount("test-topic") != 2 {
			t.Errorf("expected 2 subscribers, got %d", hub.TopicCount("test-topic"))
		}

		// Broadcast should auto-unregister the failed connection
		hub.BroadcastTo("test-topic", SSEEvent{Data: []byte("test")})

		if hub.TopicCount("test-topic") != 1 {
			t.Errorf("expected 1 subscriber after broadcast, got %d", hub.TopicCount("test-topic"))
		}
	})
}

// Goroutine cleanup tests

func TestSSE_GoroutineCleanup(t *testing.T) {
	t.Run("monitor goroutine exits on close", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Close the stream
		err = stream.Close()
		if err != nil {
			t.Fatalf("expected no error on close, got %v", err)
		}

		// Wait for monitor goroutine to exit
		done := make(chan struct{})
		go func() {
			stream.WaitDone()
			close(done)
		}()

		select {
		case <-done:
			// Goroutine exited successfully
		case <-time.After(time.Second):
			t.Error("monitor goroutine did not exit within timeout")
		}
	})

	t.Run("monitor goroutine exits on context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

		stream, err := NewSSE(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Cancel the context
		cancel()

		// Wait for monitor goroutine to exit
		done := make(chan struct{})
		go func() {
			stream.WaitDone()
			close(done)
		}()

		select {
		case <-done:
			// Goroutine exited successfully
		case <-time.After(time.Second):
			t.Error("monitor goroutine did not exit within timeout")
		}
	})

	t.Run("no goroutine leak on multiple create/close cycles", func(t *testing.T) {
		// Get initial goroutine count
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		// Create and close many SSE connections
		for i := 0; i < 50; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)

			stream, err := NewSSE(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Send an event
			_ = stream.Send(SSEEvent{Data: []byte("test")})

			// Close and wait for cleanup
			_ = stream.Close()
			stream.WaitDone()
		}

		// Give goroutines time to exit
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		final := runtime.NumGoroutine()

		if final != initial {
			t.Errorf("goroutine leak detected: started with %d, ended with %d", initial, final)
		}
	})

	t.Run("no goroutine leak on context cancellation", func(t *testing.T) {
		// Get initial goroutine count
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initial := runtime.NumGoroutine()

		// Create and cancel many contexts
		for i := 0; i < 50; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

			stream, err := NewSSE(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Cancel context and wait for cleanup
			cancel()
			stream.WaitDone()
		}

		// Give goroutines time to exit
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		final := runtime.NumGoroutine()

		if final != initial {
			t.Errorf("goroutine leak detected on cancellation: started with %d, ended with %d", initial, final)
		}
	})
}

// Test for race condition in Broadcast where connections are closed during broadcast
func TestSSEHub_BroadcastRaceCondition(t *testing.T) {
	t.Run("broadcast with concurrent close", func(t *testing.T) {
		hub := NewSSEHub()

		// Create multiple connections
		var streams []*SSE
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := NewSSE(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Register(stream)
		}

		if hub.ConnectionCount() != 10 {
			t.Errorf("expected 10 connections, got %d", hub.ConnectionCount())
		}

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// Broadcast from one goroutine
			go func() {
				defer wg.Done()
				hub.Broadcast(SSEEvent{Data: []byte("test")})
			}()

			// Close connections from another goroutine
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					if i%2 == 0 {
						_ = streams[i].Close()
					}
				}
			}()

			wg.Wait()
		}

		// Clean up remaining connections
		for _, stream := range streams {
			_ = stream.Close()
		}
	})

	t.Run("broadcastTo with concurrent close", func(t *testing.T) {
		hub := NewSSEHub()

		// Create multiple connections subscribed to a topic
		var streams []*SSE
		for i := 0; i < 10; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := NewSSE(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Subscribe(stream, "test-topic")
		}

		if hub.TopicCount("test-topic") != 10 {
			t.Errorf("expected 10 subscribers, got %d", hub.TopicCount("test-topic"))
		}

		// Run multiple iterations to increase chance of race detection
		for iter := 0; iter < 100; iter++ {
			var wg sync.WaitGroup
			wg.Add(2)

			// BroadcastTo from one goroutine
			go func() {
				defer wg.Done()
				hub.BroadcastTo("test-topic", SSEEvent{Data: []byte("test")})
			}()

			// Close connections from another goroutine
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					if i%2 == 0 {
						_ = streams[i].Close()
					}
				}
			}()

			wg.Wait()
		}

		// Clean up remaining connections
		for _, stream := range streams {
			_ = stream.Close()
		}
	})

	// Stress test with many concurrent operations
	t.Run("concurrent broadcast stress test", func(t *testing.T) {
		hub := NewSSEHub()
		var streams []*SSE

		// Create many connections
		for i := 0; i < 50; i++ {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/sse", nil)
			stream, err := NewSSE(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			streams = append(streams, stream)
			hub.Register(stream)
		}

		var wg sync.WaitGroup
		numWorkers := 10
		iterations := 100

		// Multiple broadcasters
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					hub.Broadcast(SSEEvent{Data: []byte("test"), ID: fmt.Sprintf("worker-%d-iter-%d", id, j)})
				}
			}(i)
		}

		// Multiple closers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					idx := (id*iterations + j) % len(streams)
					_ = streams[idx].Close()
					// Re-register sometimes to keep the pool active
					if j%3 == 0 {
						hub.Register(streams[idx])
					}
				}
			}(i)
		}

		// Multiple registrars
		for i := 0; i < numWorkers/2; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					idx := (id*iterations + j) % len(streams)
					hub.Register(streams[idx])
				}
			}(i)
		}

		wg.Wait()

		// Clean up
		for _, stream := range streams {
			_ = stream.Close()
		}
	})
}
