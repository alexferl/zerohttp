package rwutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
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

	zhtest.AssertEqual(t, http.StatusOK, rw.StatusCode())
	zhtest.AssertFalse(t, rw.HeaderWritten())
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	zhtest.AssertEqual(t, http.StatusNotFound, rw.StatusCode())
	zhtest.AssertTrue(t, rw.HeaderWritten())
	zhtest.AssertEqual(t, http.StatusNotFound, rec.Code)
}

func TestResponseWriter_WriteHeader_MultipleCalls(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)
	rw.WriteHeader(http.StatusInternalServerError) // Should be ignored

	zhtest.AssertEqual(t, http.StatusNotFound, rw.StatusCode())
	zhtest.AssertEqual(t, http.StatusNotFound, rec.Code)
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	data := []byte("hello world")
	n, err := rw.Write(data)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, len(data), n)
	zhtest.AssertEqual(t, http.StatusOK, rw.StatusCode())
	zhtest.AssertTrue(t, rw.HeaderWritten())
	zhtest.AssertEqual(t, "hello world", rec.Body.String())
}

func TestResponseWriter_Write_WithHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	n, err := rw.Write([]byte("created"))
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, 7, n)
	zhtest.AssertEqual(t, http.StatusCreated, rw.StatusCode())
}

func TestResponseWriter_Header(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.Header().Set("X-Custom-Header", "value")

	zhtest.AssertEqual(t, "value", rec.Header().Get("X-Custom-Header"))
}

func TestNewFlusherResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	frw := NewFlusherResponseWriter(rec)

	zhtest.AssertEqual(t, http.StatusOK, frw.StatusCode())

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
	zhtest.AssertEqual(t, "data", rec.Body.String())
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

			zhtest.AssertEqual(t, tt.expectFlushCalled, *flushCalled)
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
	zhtest.AssertTrue(t, ok)

	// Write and flush like SSE would
	rw.Header().Set(httpx.HeaderContentType, httpx.MIMETextEventStream)
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("data: hello\n\n"))
	f.Flush()

	zhtest.AssertTrue(t, rec.flushed)
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}
