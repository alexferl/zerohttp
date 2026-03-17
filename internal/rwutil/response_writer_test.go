package rwutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// flusherRecorder is a test ResponseWriter that implements http.Flusher
type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestNewResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("expected default status code %d, got %d", http.StatusOK, rw.StatusCode())
	}

	if rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be false initially")
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, rw.StatusCode())
	}

	if !rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be true after WriteHeader")
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected recorder code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResponseWriter_WriteHeader_MultipleCalls(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("expected status code to remain %d, got %d", http.StatusNotFound, rw.StatusCode())
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected recorder code to remain %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	data := []byte("hello world")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("expected status code %d after Write, got %d", http.StatusOK, rw.StatusCode())
	}

	if !rw.HeaderWritten() {
		t.Error("expected HeaderWritten to be true after Write")
	}

	if rec.Body.String() != "hello world" {
		t.Errorf("expected body %q, got %q", "hello world", rec.Body.String())
	}
}

func TestResponseWriter_Write_WithHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	n, err := rw.Write([]byte("created"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != 7 {
		t.Errorf("expected to write 7 bytes, wrote %d", n)
	}

	if rw.StatusCode() != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, rw.StatusCode())
	}
}

func TestResponseWriter_Header(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.Header().Set("X-Custom-Header", "value")

	if rec.Header().Get("X-Custom-Header") != "value" {
		t.Error("expected header to be set on underlying recorder")
	}
}

func TestNewFlusherResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	frw := NewFlusherResponseWriter(rec)

	if frw.StatusCode() != http.StatusOK {
		t.Errorf("expected default status code %d, got %d", http.StatusOK, frw.StatusCode())
	}

	// Should not panic
	frw.Flush()
}

func TestFlusherResponseWriter_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	frw := NewFlusherResponseWriter(rec)

	// Write some data and flush
	_, _ = frw.Write([]byte("data"))
	frw.Flush()

	// The recorder should have the data
	if rec.Body.String() != "data" {
		t.Errorf("expected body %q, got %q", "data", rec.Body.String())
	}
}

func TestResponseWriter_Flush(t *testing.T) {
	tests := []struct {
		name              string
		underlyingFlusher bool
		expectFlushCalled bool
	}{
		{
			name:              "flush passes through to underlying Flusher",
			underlyingFlusher: true,
			expectFlushCalled: true,
		},
		{
			name:              "flush no-op when underlying doesn't implement Flusher",
			underlyingFlusher: false,
			expectFlushCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base http.ResponseWriter
			var flushCalled *bool

			if tt.underlyingFlusher {
				rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
				base = rec
				flushCalled = &rec.flushed
			} else {
				rec := httptest.NewRecorder()
				base = rec
				flushCalled = new(bool)
			}

			// Wrap with ResponseWriter
			rw := NewResponseWriter(base)

			// Call Flush
			rw.Flush()

			if *flushCalled != tt.expectFlushCalled {
				t.Errorf("expected flush called=%v, got=%v", tt.expectFlushCalled, *flushCalled)
			}
		})
	}
}

func TestResponseWriter_Flush_SupportsSSE(t *testing.T) {
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}

	// Wrap with ResponseWriter
	rw := NewResponseWriter(rec)

	// Verify it implements Flusher
	var f http.Flusher
	f, ok := interface{}(rw).(http.Flusher)
	if !ok {
		t.Fatal("expected ResponseWriter to implement http.Flusher")
	}

	// Write and flush like SSE would
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("data: hello\n\n"))
	f.Flush()

	if !rec.flushed {
		t.Error("expected Flush to be called on underlying ResponseWriter")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
