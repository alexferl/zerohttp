package rwutil

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

func TestResponseBuffer_BasicWrite(t *testing.T) {
	t.Run("buffers data without writing to underlying", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		n, _ := buf.Write([]byte("hello"))
		if n != 5 {
			t.Errorf("expected 5 bytes written, got %d", n)
		}

		// Data should be buffered, not written to response
		if rec.Body.String() != "" {
			t.Error("expected data to be buffered, not written to response")
		}
		if buf.Buf.String() != "hello" {
			t.Errorf("expected buffer to contain 'hello', got %q", buf.Buf.String())
		}
	})

	t.Run("implicit WriteHeader with 200 on Write", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))

		if buf.Status != http.StatusOK {
			t.Errorf("expected status 200, got %d", buf.Status)
		}
		if buf.HasWritten != true {
			t.Error("expected HasWritten to be true")
		}
	})

	t.Run("explicit WriteHeader sets status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.WriteHeader(http.StatusCreated)

		if buf.Status != http.StatusCreated {
			t.Errorf("expected status 201, got %d", buf.Status)
		}
		if buf.HasWritten != true {
			t.Error("expected HasWritten to be true")
		}
	})

	t.Run("WriteHeader is idempotent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.WriteHeader(http.StatusCreated)
		buf.WriteHeader(http.StatusOK) // Should be ignored

		if buf.Status != http.StatusCreated {
			t.Errorf("expected status 201, got %d", buf.Status)
		}
	})
}

func TestResponseBuffer_Commit(t *testing.T) {
	t.Run("Commit writes buffered data", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))
		buf.Commit()

		if rec.Body.String() != "hello" {
			t.Errorf("expected 'hello', got %q", rec.Body.String())
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("Commit with custom status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.WriteHeader(http.StatusCreated)
		_, _ = buf.Write([]byte("created"))
		buf.Commit()

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}
	})

	t.Run("Commit is idempotent", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))
		buf.Commit()
		buf.Commit() // Second commit should be no-op

		if rec.Body.String() != "hello" {
			t.Errorf("expected 'hello', got %q", rec.Body.String())
		}
	})

	t.Run("CommitHeader only writes headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.WriteHeader(http.StatusCreated)
		buf.ResponseWriter.Header().Set("X-Custom", "value")
		buf.CommitHeader()

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}
		if rec.Body.String() != "" {
			t.Error("expected no body written yet")
		}
		if rec.Header().Get("X-Custom") != "value" {
			t.Error("expected header to be written")
		}
	})
}

func TestResponseBuffer_Overflow(t *testing.T) {
	t.Run("overflow switches to pass-through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 10)

		// First write fits in buffer
		_, _ = buf.Write([]byte("hello"))

		// Second write causes overflow
		_, _ = buf.Write([]byte(" world this is long"))

		if buf.Buffering {
			t.Error("expected buffering to be false after overflow")
		}
		if rec.Body.String() != "hello world this is long" {
			t.Errorf("expected full response, got %q", rec.Body.String())
		}
	})

	t.Run("single large write triggers overflow", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 5)

		_, _ = buf.Write([]byte("hello world"))

		if buf.Buffering {
			t.Error("expected buffering to be false after overflow")
		}
		if rec.Body.String() != "hello world" {
			t.Errorf("expected full response, got %q", rec.Body.String())
		}
	})

	t.Run("overflow with custom status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 5)

		buf.WriteHeader(http.StatusCreated)
		_, _ = buf.Write([]byte("hello world"))

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}
	})

	t.Run("zero max size means unlimited", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 0)

		_, _ = buf.Write([]byte("this is a long string that would overflow a small buffer"))

		if !buf.Buffering {
			t.Error("expected buffering to continue with unlimited size")
		}
		if rec.Body.String() != "" {
			t.Error("expected data to still be buffered")
		}
	})
}

func TestResponseBuffer_FlushTo(t *testing.T) {
	t.Run("FlushTo commits buffered data", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))
		buf.FlushTo(rec, nil)

		if rec.Body.String() != "hello" {
			t.Errorf("expected 'hello', got %q", rec.Body.String())
		}
	})

	t.Run("FlushTo calls onFlush callback", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		callbackCalled := false
		_, _ = buf.Write([]byte("hello"))
		buf.FlushTo(rec, func() {
			callbackCalled = true
			buf.ResponseWriter.Header().Set("X-Custom", "value")
		})

		if !callbackCalled {
			t.Error("expected onFlush callback to be called")
		}
		if rec.Header().Get("X-Custom") != "value" {
			t.Error("expected header set in callback to be present")
		}
	})

	t.Run("FlushTo without buffering just flushes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		// Simulate already committed state
		buf.Buffering = false
		buf.HeaderWritten = true

		callbackCalled := false
		buf.FlushTo(rec, func() {
			callbackCalled = true
		})

		if callbackCalled {
			t.Error("expected onFlush not to be called when not buffering")
		}
	})

	t.Run("FlushTo with nil flusher doesn't panic", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))
		buf.FlushTo(nil, nil) // Should not panic

		// Data should still be committed
		if rec.Body.String() != "hello" {
			t.Errorf("expected 'hello', got %q", rec.Body.String())
		}
	})
}

// mockHijacker is a mock ResponseWriter that supports Hijack
type mockHijacker struct {
	httptest.ResponseRecorder
	hijacked bool
}

func (m *mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

// mockPusher is a mock ResponseWriter that supports Push
type mockPusher struct {
	httptest.ResponseRecorder
	pushed bool
}

func (m *mockPusher) Push(target string, opts *http.PushOptions) error {
	m.pushed = true
	return nil
}

func TestResponseBuffer_Hijack(t *testing.T) {
	t.Run("Hijack delegates to underlying writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		// httptest.ResponseRecorder doesn't support hijacking, so we expect an error
		_, _, err := buf.Hijack()
		if err == nil {
			t.Error("expected error when hijacking non-hijacker")
		}
	})

	t.Run("Hijack succeeds with hijacker", func(t *testing.T) {
		mock := &mockHijacker{}
		buf := NewResponseBuffer(mock, 1024)

		_, _, _ = buf.Hijack()
		if !mock.hijacked {
			t.Error("expected Hijack to be called on underlying writer")
		}
	})
}

func TestResponseBuffer_Push(t *testing.T) {
	t.Run("Push delegates to underlying writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		// httptest.ResponseRecorder doesn't support push
		err := buf.Push("/test", nil)
		if err == nil {
			t.Error("expected error when pushing to non-pusher")
		}
	})

	t.Run("Push succeeds with pusher", func(t *testing.T) {
		mock := &mockPusher{}
		buf := NewResponseBuffer(mock, 1024)

		_ = buf.Push("/test", nil)
		if !mock.pushed {
			t.Error("expected Push to be called on underlying writer")
		}
	})
}

func TestResponseBuffer_Reset(t *testing.T) {
	t.Run("Reset clears state for reuse", func(t *testing.T) {
		rec1 := httptest.NewRecorder()
		buf := NewResponseBuffer(rec1, 1024)

		_, _ = buf.Write([]byte("hello"))
		buf.HasWritten = true
		buf.HeaderWritten = true
		buf.Buffering = false

		rec2 := httptest.NewRecorder()
		buf.Reset(rec2)

		if buf.Buf.Len() != 0 {
			t.Error("expected buffer to be reset")
		}
		if buf.Status != http.StatusOK {
			t.Errorf("expected status reset to 200, got %d", buf.Status)
		}
		if buf.HasWritten {
			t.Error("expected HasWritten to be false")
		}
		if buf.HeaderWritten {
			t.Error("expected HeaderWritten to be false")
		}
		if !buf.Buffering {
			t.Error("expected Buffering to be true")
		}
		if buf.ResponseWriter != rec2 {
			t.Error("expected ResponseWriter to be updated")
		}
	})
}

func TestResponseBuffer_MultipleWrites(t *testing.T) {
	t.Run("multiple small writes accumulate", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		_, _ = buf.Write([]byte("hello"))
		_, _ = buf.Write([]byte(" "))
		_, _ = buf.Write([]byte("world"))
		buf.Commit()

		if rec.Body.String() != "hello world" {
			t.Errorf("expected 'hello world', got %q", rec.Body.String())
		}
	})

	t.Run("mixed buffered and pass-through writes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 10)

		_, _ = buf.Write([]byte("hello"))      // buffered
		_, _ = buf.Write([]byte(" world"))     // triggers overflow
		_, _ = buf.Write([]byte(" more data")) // pass-through

		if rec.Body.String() != "hello world more data" {
			t.Errorf("expected 'hello world more data', got %q", rec.Body.String())
		}
	})
}

func TestResponseBuffer_HeaderManipulation(t *testing.T) {
	t.Run("headers are forwarded to underlying writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.ResponseWriter.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		_, _ = buf.Write([]byte("{}"))
		buf.Commit()

		if rec.Header().Get(httpx.HeaderContentType) != "application/json" {
			t.Error("expected Content-Type header to be set")
		}
	})

	t.Run("headers set before WriteHeader are preserved", func(t *testing.T) {
		rec := httptest.NewRecorder()
		buf := NewResponseBuffer(rec, 1024)

		buf.ResponseWriter.Header().Set("X-Before", "1")
		buf.WriteHeader(http.StatusCreated)
		buf.ResponseWriter.Header().Set("X-After", "2")
		buf.Commit()

		if rec.Header().Get("X-Before") != "1" {
			t.Error("expected X-Before header to be preserved")
		}
		if rec.Header().Get("X-After") != "2" {
			t.Error("expected X-After header to be preserved")
		}
	})
}

func BenchmarkResponseBuffer_Write(b *testing.B) {
	data := bytes.Repeat([]byte("a"), 1024)

	b.Run("buffered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			buf := NewResponseBuffer(rec, 10240)
			_, _ = buf.Write(data)
			buf.Commit()
		}
	})

	b.Run("overflow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			buf := NewResponseBuffer(rec, 512)
			_, _ = buf.Write(data)
		}
	})
}
