package sse

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNewEventStream(t *testing.T) {
	t.Run("successfully creates SSE", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
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

		_, err := New(w, r)
		if err == nil {
			t.Error("expected error when headers already sent")
		}
	})
}

func TestEventStream_Send(t *testing.T) {
	t.Run("sends simple event", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{
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

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{
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

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{
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

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_ = stream.Close()

		event := Event{Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error after close")
		}
	})

	t.Run("rejects event ID with CR", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{ID: "abc\rdef", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with CR")
		}
	})

	t.Run("rejects event ID with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{ID: "abc\ndef", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with LF")
		}
	})

	t.Run("rejects event ID with NULL", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{ID: "abc\x00def", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for ID with NULL")
		}
	})

	t.Run("rejects event name with CR", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{Name: "update\rnotify", Data: []byte("test")}
		err = stream.Send(event)
		if err == nil {
			t.Error("expected error for Name with CR")
		}
	})

	t.Run("rejects event name with LF", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		event := Event{Name: "update\nnotify", Data: []byte("test")}
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

		stream, err := New(w, r)
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

		stream, err := New(w, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() { _ = stream.Close() }()

		err = stream.SetRetry(10 * time.Second)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Send event without retry - should use default
		event := Event{Data: []byte("test")}
		err = stream.Send(event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		zhtest.AssertWith(t, w).BodyContains("retry: 10000\n")
	})
}

func TestEventStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)

	stream, err := New(w, r)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond) // Let cancellation propagate

	// Send should fail after context cancellation
	event := Event{Data: []byte("test")}
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

		event := Event{Data: []byte("test")}
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

		stream, err := New(w, r)
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

func TestSSE_GoroutineCleanup(t *testing.T) {
	t.Run("monitor goroutine exits on close", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/sse", nil)

		stream, err := New(w, r)
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

		stream, err := New(w, r)
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

			stream, err := New(w, r)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Send an event
			_ = stream.Send(Event{Data: []byte("test")})

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

			stream, err := New(w, r)
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
